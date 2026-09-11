// Package statemigration copies an install's state from the legacy state
// directory into the Atomwright one.
//
// The migration is a copy, never a move: a user who downgrades, or who runs an
// older build from another machine against the same home directory, must still
// find their state where it was. Nothing in this package deletes or rewrites
// the legacy root.
//
// Only the portable files are migrated. Everything else in the legacy root is
// either regenerated on demand or is not relocatable at all — see
// portableFiles and the backups note on it.
package statemigration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v2/internal/components/filemerge"
)

// CompletionMarkerName is the file that records a finished migration. It is
// written last, so its presence means every migrated file is already durably in
// place.
const CompletionMarkerName = ".migration-complete.json"

// portableFiles are the only files this migration relocates.
//
// Both were verified to hold no absolute paths, which is what makes copying
// them to a different root safe.
//
// backups/ is deliberately absent. Each backups/<ts>/manifest.json embeds an
// absolute "root_dir", and internal/backup/manifest.go validates that the
// manifest's entries stay contained under it as an anti-tamper check against a
// crafted manifest deleting arbitrary paths. A copied manifest keeps pointing
// into the legacy tree, so the copy is rejected at restore time: the user would
// see backups that exist but cannot restore, which is worse than not having
// them in the new root at all. Backups stay where they were created.
var portableFiles = []string{"state.json", "telemetry.json"}

// Outcome classifies a legacy/new root pair. It is a string so it reads plainly
// in logs and test failures.
type Outcome string

const (
	// OutcomeFresh means neither root holds state: a first install.
	OutcomeFresh Outcome = "fresh"
	// OutcomeMigrate means only the legacy root holds state.
	OutcomeMigrate Outcome = "migrate"
	// OutcomeAlreadyMigrated means the new root is authoritative already.
	OutcomeAlreadyMigrated Outcome = "already-migrated"
	// OutcomeConflict means both roots hold state and only the user can say
	// which one is current.
	OutcomeConflict Outcome = "conflict"
)

// Migration is the dry-run preview produced by Plan. Producing one writes
// nothing, so the preview shown to a user is exactly what Apply will act on.
type Migration struct {
	Outcome    Outcome
	LegacyRoot string
	NewRoot    string
	// Files are the portable files present in the legacy root, sorted, as
	// root-relative names.
	Files []string
}

// Result reports what Apply did.
type Result struct {
	Outcome Outcome
	// Migrated are the root-relative names actually copied, sorted.
	Migrated []string
}

// Plan classifies the two roots and lists what a migration would copy.
//
// It is pure: it reads the filesystem and never modifies it. Creating the
// destination root here would turn the next run's classification from "migrate"
// into "already migrated" and strand the legacy state permanently.
func Plan(legacyRoot, newRoot string) (Migration, error) {
	legacyRoot = filepath.Clean(legacyRoot)
	newRoot = filepath.Clean(newRoot)

	plan := Migration{LegacyRoot: legacyRoot, NewRoot: newRoot}

	files, err := migratableFiles(legacyRoot)
	if err != nil {
		return Migration{}, err
	}
	plan.Files = files

	complete, err := migrationCompleted(newRoot)
	if err != nil {
		return Migration{}, err
	}
	populated, err := hasEntries(newRoot)
	if err != nil {
		return Migration{}, err
	}

	switch {
	case complete:
		plan.Outcome = OutcomeAlreadyMigrated
	case populated:
		if len(plan.Files) == 0 {
			plan.Outcome = OutcomeAlreadyMigrated
			break
		}
		// The new root holds state but carries no completion marker, so this is
		// either an interrupted run of this migration or two independent states.
		//
		// Those are distinguishable. An interrupted run can only have left files
		// this migration writes, each byte-identical to its legacy source,
		// because that is all it ever copies. When that holds, resuming is a
		// no-op for everything already published and cannot lose data, so the
		// migration is recoverable rather than blocked.
		//
		// Anything else — an unexpected file, or the same name with different
		// bytes — means the new root has a history of its own. Then only the
		// user can say which is current, and refusing is the safe direction:
		// the alternative is guessing and overwriting the other.
		resumable, err := isResumablePartialMigration(plan)
		if err != nil {
			return Migration{}, err
		}
		if resumable {
			plan.Outcome = OutcomeMigrate
		} else {
			plan.Outcome = OutcomeConflict
		}
	case len(plan.Files) > 0:
		plan.Outcome = OutcomeMigrate
	default:
		plan.Outcome = OutcomeFresh
	}

	return plan, nil
}

// Apply executes plan.
//
// It is idempotent: re-running it on an already migrated or fresh install is a
// no-op. Install, sync and upgrade all run the migration, so it runs many times
// over the life of an installation.
//
// It is crash-safe: every file is published by an atomic write-then-rename and
// the completion marker is written last, after the copies are durable. An
// interrupted run therefore never records itself as complete and stays
// re-runnable.
func Apply(plan Migration) (Result, error) {
	result := Result{Outcome: plan.Outcome}

	switch plan.Outcome {
	case OutcomeFresh, OutcomeAlreadyMigrated:
		return result, nil
	case OutcomeConflict:
		return result, fmt.Errorf(
			"state exists in both %q and %q, so the migration cannot tell which one is current: "+
				"keep the one you want, then move or remove the other before running this again",
			plan.LegacyRoot, plan.NewRoot)
	case OutcomeMigrate:
	default:
		return result, fmt.Errorf("unknown migration outcome %q", plan.Outcome)
	}

	if err := os.MkdirAll(plan.NewRoot, 0o700); err != nil {
		return result, fmt.Errorf("create state directory %q: %w", plan.NewRoot, err)
	}

	for _, name := range plan.Files {
		source, err := containedPath(plan.LegacyRoot, name)
		if err != nil {
			return result, err
		}
		target, err := containedPath(plan.NewRoot, name)
		if err != nil {
			return result, err
		}
		if _, err := os.Lstat(target); err == nil {
			// Existing state in the new root is authoritative and is never
			// replaced by an older copy.
			continue
		} else if !os.IsNotExist(err) {
			return result, fmt.Errorf("inspect %q: %w", target, err)
		}
		if err := copyFileDurably(source, target); err != nil {
			return result, err
		}
		result.Migrated = append(result.Migrated, name)
	}

	if err := writeCompletionMarker(plan); err != nil {
		return result, err
	}

	slices.Sort(result.Migrated)
	return result, nil
}

// migratableFiles lists the portable files present in root, skipping anything
// that is not a regular file. A symlink is never dereferenced: following one
// would read bytes from outside the legacy root and copy them in under a name
// the user never put there.
func migratableFiles(root string) ([]string, error) {
	var files []string
	for _, name := range portableFiles {
		path, err := containedPath(root, name)
		if err != nil {
			return nil, err
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %q: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		files = append(files, name)
	}
	slices.Sort(files)
	return files, nil
}

// migrationCompleted reports whether root carries the completion marker.
func migrationCompleted(root string) (bool, error) {
	marker, err := containedPath(root, CompletionMarkerName)
	if err != nil {
		return false, err
	}
	info, err := os.Lstat(marker)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect %q: %w", marker, err)
	}
	return info.Mode().IsRegular(), nil
}

// hasEntries reports whether root exists and holds anything. An existing but
// empty directory is not state: it is what an interrupted run leaves behind,
// and the retry must still classify as a migration.
func hasEntries(root string) (bool, error) {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read directory %q: %w", root, err)
	}
	return len(entries) > 0, nil
}

// isResumablePartialMigration reports whether the new root's contents are
// explainable as an interrupted run of this same migration: every entry is a
// file this migration publishes, and every one of them already matches its
// legacy source byte for byte.
//
// It is deliberately strict. A single unexplained entry, or one differing byte,
// makes the new root independent state and the answer is no.
func isResumablePartialMigration(plan Migration) (bool, error) {
	entries, err := os.ReadDir(plan.NewRoot)
	if err != nil {
		return false, fmt.Errorf("read directory %q: %w", plan.NewRoot, err)
	}

	planned := make(map[string]struct{}, len(plan.Files))
	for _, name := range plan.Files {
		planned[name] = struct{}{}
	}

	for _, entry := range entries {
		if _, ok := planned[entry.Name()]; !ok || entry.IsDir() {
			return false, nil
		}
		same, err := sameFileContent(plan.LegacyRoot, plan.NewRoot, entry.Name())
		if err != nil {
			return false, err
		}
		if !same {
			return false, nil
		}
	}
	return true, nil
}

// sameFileContent reports whether name holds identical bytes under both roots.
func sameFileContent(legacyRoot, newRoot, name string) (bool, error) {
	legacyPath, err := containedPath(legacyRoot, name)
	if err != nil {
		return false, err
	}
	newPath, err := containedPath(newRoot, name)
	if err != nil {
		return false, err
	}

	legacyContent, err := os.ReadFile(legacyPath)
	if err != nil {
		return false, fmt.Errorf("read %q: %w", legacyPath, err)
	}
	newContent, err := os.ReadFile(newPath)
	if err != nil {
		return false, fmt.Errorf("read %q: %w", newPath, err)
	}
	return bytes.Equal(legacyContent, newContent), nil
}

// containedPath joins name onto root and proves the result stays inside root,
// so no crafted name can make the migration read or write outside the two roots
// it was given.
func containedPath(root, name string) (string, error) {
	path := filepath.Clean(filepath.Join(root, name))
	if path != root && !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to use %q: it escapes %q", path, root)
	}
	return path, nil
}

// copyFileDurably copies src to dst, preserving src's permission bits.
//
// state.json can carry tokens, so widening 0600 to a default mode during the
// copy would expose them to every user on a shared machine.
func copyFileDurably(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("inspect %q: %w", src, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to migrate %q: it is not a regular file", src)
	}

	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %q: %w", src, err)
	}
	return writeFileDurably(dst, content, info.Mode().Perm())
}

// writeFileDurably publishes content at path with an atomic write-then-rename
// and syncs the parent directory so the published name survives recovery.
//
// filemerge.WriteFileAtomic is deliberately not reused here: it relaxes a
// read-only parent directory's permissions to complete the write. That is right
// for the files it manages and wrong here, where the destination is the user's
// state directory — a migration must not widen its permissions to make itself
// succeed. filemerge.SyncDir, which carries the subtle Windows behaviour, is
// reused as-is.
func writeFileDurably(path string, content []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".migration-*.tmp")
	if err != nil {
		return fmt.Errorf("stage replacement for %q: %w", path, err)
	}
	tmpPath := tmp.Name()
	published := false
	defer func() {
		if !published {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write staged replacement for %q: %w", path, err)
	}
	// Chmod precedes Sync so a recovered rename cannot expose the file with the
	// wrong mode.
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set permissions on staged replacement for %q: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync staged replacement for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close staged replacement for %q: %w", path, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publish %q: %w", path, err)
	}
	published = true

	if err := filemerge.SyncDir(dir); err != nil {
		return fmt.Errorf("sync directory %q: %w", dir, err)
	}
	return nil
}

// writeCompletionMarker records the finished migration. It runs only after
// every copy is durable, because a marker that lands early makes the next run
// report "already migrated" and abandon state the user still has.
func writeCompletionMarker(plan Migration) error {
	marker, err := containedPath(plan.NewRoot, CompletionMarkerName)
	if err != nil {
		return err
	}
	content, err := json.Marshal(struct {
		MigratedFrom string   `json:"migrated_from"`
		Files        []string `json:"files"`
	}{MigratedFrom: plan.LegacyRoot, Files: plan.Files})
	if err != nil {
		return fmt.Errorf("encode migration marker: %w", err)
	}
	return writeFileDurably(marker, content, 0o600)
}
