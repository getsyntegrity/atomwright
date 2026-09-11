package statemigration_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/pablogore/atomwright/v2/internal/statemigration"
)

// --- test helpers ---

// newRoots returns two sibling paths under one temp parent. Neither directory
// exists yet; each test creates exactly the ones its scenario requires, so the
// absent/present state table is expressed honestly on disk.
func newRoots(t *testing.T) (parent, legacyRoot, newRoot string) {
	t.Helper()

	parent = t.TempDir()
	return parent, filepath.Join(parent, ".gentle-ai"), filepath.Join(parent, ".atomwright")
}

// writeFileAt creates path and all its parent directories with the given mode.
func writeFileAt(t *testing.T, path, content string, mode fs.FileMode) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("Chmod(%q, %v): %v", path, mode, err)
	}
}

// readFileAt reads path, failing the test when it does not exist.
func readFileAt(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return string(content)
}

// snapshotTree walks root and returns "<relative path>\t<content>" entries,
// sorted, so two trees can be compared byte for byte. Directories appear as
// "<relative path>/". A missing root returns nil.
func snapshotTree(t *testing.T, root string) []string {
	t.Helper()

	var entries []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			entries = append(entries, rel+"/")
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				return linkErr
			}
			entries = append(entries, rel+"\t-> "+target)
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		entries = append(entries, rel+"\t"+string(content))
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("walk %q: %v", root, err)
	}
	slices.Sort(entries)
	return entries
}

// requireNotExist fails when path exists.
func requireNotExist(t *testing.T, path, why string) {
	t.Helper()

	if _, err := os.Lstat(path); err == nil {
		t.Errorf("%q exists but must not: %s", path, why)
	} else if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("lstat %q: %v", path, err)
	}
}

// seedLegacyState populates a legacy root with the two files the migration owns.
func seedLegacyState(t *testing.T, legacyRoot string) {
	t.Helper()

	writeFileAt(t, filepath.Join(legacyRoot, "state.json"), `{"installed_agents":["claude-code"]}`, 0o600)
	writeFileAt(t, filepath.Join(legacyRoot, "telemetry.json"), `{"enabled":false}`, 0o600)
}

// --- state table ---

// TestPlanOutcomeStateTable covers every combination of legacy and new state
// roots. Plan is the dry-run preview: it classifies the situation and writes
// nothing.
func TestPlanOutcomeStateTable(t *testing.T) {
	tests := []struct {
		name        string
		seedLegacy  bool
		seedNew     func(t *testing.T, newRoot string)
		wantOutcome statemigration.Outcome
	}{
		{
			name:        "neither root exists is a fresh install",
			wantOutcome: statemigration.OutcomeFresh,
		},
		{
			name:        "legacy only is migratable",
			seedLegacy:  true,
			wantOutcome: statemigration.OutcomeMigrate,
		},
		{
			name: "new only is already migrated",
			seedNew: func(t *testing.T, newRoot string) {
				writeFileAt(t, filepath.Join(newRoot, "state.json"), `{"installed_agents":["opencode"]}`, 0o600)
			},
			wantOutcome: statemigration.OutcomeAlreadyMigrated,
		},
		{
			name:       "both roots hold state is a conflict",
			seedLegacy: true,
			seedNew: func(t *testing.T, newRoot string) {
				writeFileAt(t, filepath.Join(newRoot, "state.json"), `{"installed_agents":["opencode"]}`, 0o600)
			},
			wantOutcome: statemigration.OutcomeConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, legacyRoot, newRoot := newRoots(t)
			if tt.seedLegacy {
				seedLegacyState(t, legacyRoot)
			}
			if tt.seedNew != nil {
				tt.seedNew(t, newRoot)
			}

			plan, err := statemigration.Plan(legacyRoot, newRoot)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			if plan.Outcome != tt.wantOutcome {
				t.Errorf("Plan().Outcome = %v, want %v", plan.Outcome, tt.wantOutcome)
			}
		})
	}
}

// TestPlanIsPureDryRun verifies Plan never touches the filesystem. A preview
// that creates the destination directory turns the next run's classification
// from "migrate" into "already migrated" and silently loses the legacy state.
func TestPlanIsPureDryRun(t *testing.T) {
	parent, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)

	before := snapshotTree(t, parent)

	if _, err := statemigration.Plan(legacyRoot, newRoot); err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if after := snapshotTree(t, parent); !reflect.DeepEqual(before, after) {
		t.Errorf("Plan() modified the filesystem:\nbefore = %q\nafter  = %q", before, after)
	}
	requireNotExist(t, newRoot, "Plan() must not create the destination root")
}

// TestPlanIsDeterministic verifies two Plan calls on identical input produce
// equal plans, so a preview shown to the user matches what Apply will do.
func TestPlanIsDeterministic(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)
	writeFileAt(t, filepath.Join(legacyRoot, "logs", "run.log"), "noise\n", 0o644)

	first, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("first Plan() error = %v", err)
	}
	second, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("second Plan() error = %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Errorf("Plan() is not deterministic:\nfirst  = %+v\nsecond = %+v", first, second)
	}
}

// --- apply behaviour ---

// TestApplyMigratesStateAndTelemetry verifies the migrated files land in the
// new root with identical bytes.
func TestApplyMigratesStateAndTelemetry(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Outcome != statemigration.OutcomeMigrate {
		t.Fatalf("Plan().Outcome = %v, want %v", plan.Outcome, statemigration.OutcomeMigrate)
	}
	if _, err := statemigration.Apply(plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	for _, name := range []string{"state.json", "telemetry.json"} {
		want := readFileAt(t, filepath.Join(legacyRoot, name))
		got := readFileAt(t, filepath.Join(newRoot, name))
		if got != want {
			t.Errorf("%s migrated content = %q, want %q", name, got, want)
		}
	}
}

// TestMigrationNeverDeletesLegacyData verifies the legacy directory survives
// the migration untouched. The migration copies; it never moves. A user who
// downgrades, or who runs an older build from another machine, must still find
// their state where it was.
func TestMigrationNeverDeletesLegacyData(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)

	before := snapshotTree(t, legacyRoot)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, err := statemigration.Apply(plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if _, err := os.Stat(legacyRoot); err != nil {
		t.Fatalf("legacy root no longer exists after migration: %v", err)
	}
	if after := snapshotTree(t, legacyRoot); !reflect.DeepEqual(before, after) {
		t.Errorf("legacy tree changed during migration:\nbefore = %q\nafter  = %q", before, after)
	}
}

// TestApplyRefusesConflictAndExplainsBothPaths verifies a conflicting state is
// never resolved silently: nothing is written, and the error names both paths
// and tells the user what to do.
func TestApplyRefusesConflictAndExplainsBothPaths(t *testing.T) {
	parent, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)
	writeFileAt(t, filepath.Join(newRoot, "state.json"), `{"installed_agents":["opencode"]}`, 0o600)

	before := snapshotTree(t, parent)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Outcome != statemigration.OutcomeConflict {
		t.Fatalf("Plan().Outcome = %v, want %v", plan.Outcome, statemigration.OutcomeConflict)
	}

	_, applyErr := statemigration.Apply(plan)
	if applyErr == nil {
		t.Fatal("Apply() on a conflicting plan returned no error; it must refuse")
	}

	msg := applyErr.Error()
	for _, want := range []string{legacyRoot, newRoot} {
		if !strings.Contains(msg, want) {
			t.Errorf("Apply() conflict error = %q, want it to name %q", msg, want)
		}
	}
	if !strings.Contains(strings.ToLower(msg), "remove") && !strings.Contains(strings.ToLower(msg), "move") && !strings.Contains(strings.ToLower(msg), "delete") {
		t.Errorf("Apply() conflict error = %q, want it to tell the user what to do about the two directories", msg)
	}

	if after := snapshotTree(t, parent); !reflect.DeepEqual(before, after) {
		t.Errorf("Apply() modified the filesystem on a refused conflict:\nbefore = %q\nafter  = %q", before, after)
	}
}

// TestMigrationNeverOverwritesExistingAtomwrightState verifies existing state in
// the new root is never clobbered by legacy state.
func TestMigrationNeverOverwritesExistingAtomwrightState(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)

	const existing = `{"installed_agents":["codex"],"strict_tdd":true}`
	newState := filepath.Join(newRoot, "state.json")
	writeFileAt(t, newState, existing, 0o600)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	_, _ = statemigration.Apply(plan)

	if got := readFileAt(t, newState); got != existing {
		t.Errorf("existing atomwright state.json = %q, want it untouched %q", got, existing)
	}
}

// TestMigrationIsIdempotent verifies a second Apply is a no-op: the tree is
// byte-identical and no error is returned. Install, sync and upgrade all run
// the migration, so it runs many times over the life of an installation.
func TestMigrationIsIdempotent(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)

	first, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("first Plan() error = %v", err)
	}
	if _, err := statemigration.Apply(first); err != nil {
		t.Fatalf("first Apply() error = %v", err)
	}
	afterFirst := snapshotTree(t, newRoot)

	second, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("second Plan() error = %v", err)
	}
	if second.Outcome != statemigration.OutcomeAlreadyMigrated {
		t.Errorf("second Plan().Outcome = %v, want %v", second.Outcome, statemigration.OutcomeAlreadyMigrated)
	}
	if _, err := statemigration.Apply(second); err != nil {
		t.Fatalf("second Apply() error = %v", err)
	}

	if afterSecond := snapshotTree(t, newRoot); !reflect.DeepEqual(afterFirst, afterSecond) {
		t.Errorf("second Apply() changed the tree:\nfirst  = %q\nsecond = %q", afterFirst, afterSecond)
	}
}

// TestPreservesFilePermissions verifies restrictive modes survive the copy.
// state.json can carry tokens; widening it to 0644 during migration would leak
// them to every user on a shared machine.
func TestPreservesFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not meaningful on Windows")
	}

	_, legacyRoot, newRoot := newRoots(t)
	writeFileAt(t, filepath.Join(legacyRoot, "state.json"), `{"installed_agents":[]}`, 0o600)
	writeFileAt(t, filepath.Join(legacyRoot, "telemetry.json"), `{"enabled":false}`, 0o600)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, err := statemigration.Apply(plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(newRoot, "state.json"))
	if err != nil {
		t.Fatalf("stat migrated state.json: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("migrated state.json mode = %#o, want %#o", got, 0o600)
	}
}

// TestMigrationDoesNotRelocateBackups verifies backups/ is left behind.
//
// internal/backup/manifest.go validates that a manifest's absolute root_dir is
// contained under the expected backup directory — an anti-tamper check that
// stops a crafted manifest from deleting arbitrary paths. Copying a backup into
// the new root leaves root_dir pointing into the legacy tree, so the manifest
// is rejected at restore time and the user's backup becomes unusable. Backups
// stay where they were created.
func TestMigrationDoesNotRelocateBackups(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)

	backupDir := filepath.Join(legacyRoot, "backups", "20260101120000.000000000")
	manifest, err := json.Marshal(map[string]any{
		"id":         "20260101120000.000000000",
		"created_at": "2026-01-01T12:00:00Z",
		"root_dir":   backupDir,
		"entries":    []any{},
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	writeFileAt(t, filepath.Join(backupDir, "manifest.json"), string(manifest), 0o600)

	plan, planErr := statemigration.Plan(legacyRoot, newRoot)
	if planErr != nil {
		t.Fatalf("Plan() error = %v", planErr)
	}
	if _, err := statemigration.Apply(plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	requireNotExist(t, filepath.Join(newRoot, "backups"), "backups carry absolute root_dir paths that the anti-tamper check rejects once relocated")

	if _, err := os.Stat(filepath.Join(backupDir, "manifest.json")); err != nil {
		t.Errorf("legacy backup manifest was disturbed: %v", err)
	}
}

// TestMigrationDoesNotFollowUnsafeSymlinks verifies a symlink inside the legacy
// root is not dereferenced into the new root, and that nothing outside the two
// roots is read through it or written to.
func TestMigrationDoesNotFollowUnsafeSymlinks(t *testing.T) {
	parent, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)

	outside := t.TempDir()
	secretPath := filepath.Join(outside, "secret.json")
	const secret = `{"token":"super-secret-token"}`
	writeFileAt(t, secretPath, secret, 0o600)

	link := filepath.Join(legacyRoot, "linked.json")
	if err := os.Symlink(secretPath, link); err != nil {
		t.Skipf("symlinks unavailable in this environment: %v", err)
	}

	outsideBefore := snapshotTree(t, outside)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, err := statemigration.Apply(plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	requireNotExist(t, filepath.Join(newRoot, "linked.json"), "a symlink pointing outside both roots must never be followed into the new root")

	for _, entry := range snapshotTree(t, newRoot) {
		if strings.Contains(entry, secret) {
			t.Errorf("migration copied content read through an unsafe symlink: %q", entry)
		}
	}
	if outsideAfter := snapshotTree(t, outside); !reflect.DeepEqual(outsideBefore, outsideAfter) {
		t.Errorf("migration wrote outside both roots:\nbefore = %q\nafter  = %q", outsideBefore, outsideAfter)
	}
	if _, err := os.Stat(filepath.Join(parent, "linked.json")); err == nil {
		t.Errorf("migration created a file in the shared parent directory")
	}
}

// TestMigrationCannotEscapeRoots verifies Apply creates files only inside the
// new root — nothing lands in the parent directory or anywhere else.
func TestMigrationCannotEscapeRoots(t *testing.T) {
	parent, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)
	writeFileAt(t, filepath.Join(legacyRoot, "nested", "deep", "value.json"), `{"a":1}`, 0o600)

	before := snapshotTree(t, parent)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, err := statemigration.Apply(plan); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	newRootRel := filepath.ToSlash(filepath.Base(newRoot))
	for _, entry := range snapshotTree(t, parent) {
		if slices.Contains(before, entry) {
			continue
		}
		path := entry
		if idx := strings.IndexAny(path, "\t"); idx >= 0 {
			path = path[:idx]
		}
		path = strings.TrimSuffix(path, "/")
		if path != newRootRel && !strings.HasPrefix(path, newRootRel+"/") {
			t.Errorf("migration created %q outside the new root %q", path, newRootRel)
		}
	}
}

// TestInterruptedMigrationIsNotRecordedAsComplete verifies a failed Apply never
// leaves the completion marker behind. A marker written before the copy
// finishes makes the next run report "already migrated" and abandon state the
// user still has.
func TestInterruptedMigrationIsNotRecordedAsComplete(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory write permissions are not enforced the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses directory write permissions")
	}

	_, legacyRoot, newRoot := newRoots(t)
	seedLegacyState(t, legacyRoot)

	// A read-only destination root makes the copy fail partway through, which
	// is the observable stand-in for an interrupted migration.
	if err := os.MkdirAll(newRoot, 0o500); err != nil {
		t.Fatalf("MkdirAll(%q): %v", newRoot, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(newRoot, 0o700)
	})

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Outcome != statemigration.OutcomeMigrate {
		t.Fatalf("Plan().Outcome = %v, want %v", plan.Outcome, statemigration.OutcomeMigrate)
	}

	if _, applyErr := statemigration.Apply(plan); applyErr == nil {
		t.Fatal("Apply() into a read-only destination returned no error")
	}

	if err := os.Chmod(newRoot, 0o700); err != nil {
		t.Fatalf("Chmod(%q): %v", newRoot, err)
	}

	marker := filepath.Join(newRoot, statemigration.CompletionMarkerName)
	requireNotExist(t, marker, "an interrupted migration must not be recorded as complete")

	retry, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("retry Plan() error = %v", err)
	}
	if retry.Outcome != statemigration.OutcomeMigrate {
		t.Errorf("retry Plan().Outcome = %v, want %v so the interrupted migration is retried", retry.Outcome, statemigration.OutcomeMigrate)
	}
}

// TestInterruptedMigrationResumes proves an interrupted run is recoverable, not
// merely safe.
//
// A crash between publishing the first file and writing the completion marker
// leaves the new root holding state with no marker. That is distinguishable from
// two independent states: every entry is a file this migration publishes and
// every one already matches its legacy source byte for byte. Resuming then
// cannot lose data, so it must resume rather than refuse.
func TestInterruptedMigrationResumes(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	writeFileAt(t, filepath.Join(legacyRoot, "state.json"), `{"persona":"gentleman"}`, 0o600)
	writeFileAt(t, filepath.Join(legacyRoot, "telemetry.json"), `{"enabled":false}`, 0o600)

	// Simulate the crash: state.json is already published, telemetry.json and
	// the completion marker never made it.
	writeFileAt(t, filepath.Join(newRoot, "state.json"), `{"persona":"gentleman"}`, 0o600)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Outcome != statemigration.OutcomeMigrate {
		t.Fatalf("Plan outcome = %q, want %q: an interrupted migration must resume", plan.Outcome, statemigration.OutcomeMigrate)
	}

	if _, err := statemigration.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got, want := readFileAt(t, filepath.Join(newRoot, "telemetry.json")), `{"enabled":false}`; got != want {
		t.Errorf("telemetry.json = %q, want %q: the resumed run must publish the missing file", got, want)
	}
	if got, want := readFileAt(t, filepath.Join(legacyRoot, "state.json")), `{"persona":"gentleman"}`; got != want {
		t.Errorf("legacy state.json = %q, want %q: resuming must not touch the legacy root", got, want)
	}
}

// TestDivergentNewRootIsAConflictNotAResume guards the boundary of the rule
// above: same file name, different bytes, means the new root has a history of
// its own and only the user can say which is current.
func TestDivergentNewRootIsAConflictNotAResume(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	writeFileAt(t, filepath.Join(legacyRoot, "state.json"), `{"persona":"gentleman"}`, 0o600)
	writeFileAt(t, filepath.Join(newRoot, "state.json"), `{"persona":"neutral"}`, 0o600)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Outcome != statemigration.OutcomeConflict {
		t.Fatalf("Plan outcome = %q, want %q: divergent state must never be silently resumed", plan.Outcome, statemigration.OutcomeConflict)
	}

	if _, err := statemigration.Apply(plan); err == nil {
		t.Fatal("Apply on a conflict returned no error; it must refuse")
	}
	if got, want := readFileAt(t, filepath.Join(newRoot, "state.json")), `{"persona":"neutral"}`; got != want {
		t.Errorf("new state.json = %q, want %q: a refused migration must not overwrite", got, want)
	}
}

// TestUnexpectedEntryInNewRootIsAConflict guards the other boundary: an entry
// this migration never writes means the new root is not an interrupted run.
func TestUnexpectedEntryInNewRootIsAConflict(t *testing.T) {
	_, legacyRoot, newRoot := newRoots(t)
	writeFileAt(t, filepath.Join(legacyRoot, "state.json"), `{"persona":"gentleman"}`, 0o600)
	writeFileAt(t, filepath.Join(newRoot, "state.json"), `{"persona":"gentleman"}`, 0o600)
	writeFileAt(t, filepath.Join(newRoot, "credentials.json"), `{"token":"x"}`, 0o600)

	plan, err := statemigration.Plan(legacyRoot, newRoot)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.Outcome != statemigration.OutcomeConflict {
		t.Fatalf("Plan outcome = %q, want %q: an unexplained entry must block the migration", plan.Outcome, statemigration.OutcomeConflict)
	}
}
