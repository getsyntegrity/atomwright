package identity_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestUpdateRegistryPrimaryToolIsAtomwright asserts the managed-tool registry
// entry for this product carries the new name and release coordinates while
// keeping the PRESERVED module path as its go-install target.
func TestUpdateRegistryPrimaryToolIsAtomwright(t *testing.T) {
	registry := readRepositoryFile(t, "internal/update/registry.go")

	tests := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"Name", regexp.MustCompile(`Name:\s+"atomwright"`)},
		{"Owner", regexp.MustCompile(`Owner:\s+"pablogore"`)},
		{"Repo", regexp.MustCompile(`Repo:\s+"atomwright"`)},
		{"GoImportPath", regexp.MustCompile(`GoImportPath:\s+"github\.com/pablogore/atomwright/v2/cmd/atomwright"`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.pattern.MatchString(registry) {
				t.Errorf("internal/update/registry.go primary tool entry does not match %s field pattern %s", tt.name, tt.pattern)
			}
		})
	}
}

// TestUpdaterNeverTargetsUpstreamRepository asserts no non-test file under
// internal/update reaches the upstream Gentleman-Programming/gentle-ai
// repository. The updater decides where the binary a user runs comes from:
// pointing it at the upstream repository after the fork means self-upgrade
// silently replaces the user's binary with someone else's release.
//
// internal/update/advisory.go currently hardcodes an upstream advisory release
// URL, and internal/update/instructions.go hardcodes an upstream install-script
// URL. Both must be caught here.
//
// Note: entries for genuinely third-party Gentleman-Programming projects
// (engram, gentleman-guardian-angel) stay legitimate; only references that pair
// that owner with this product are forbidden.
func TestUpdaterNeverTargetsUpstreamRepository(t *testing.T) {
	root := identityRepositoryRoot(t)
	updateDir := filepath.Join(root, "internal", "update")

	err := filepath.WalkDir(updateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		for i, line := range strings.Split(string(content), "\n") {
			if strings.Contains(line, "Gentleman-Programming/gentle-ai") {
				t.Errorf("%s:%d targets the upstream repository: %q", rel, i+1, strings.TrimSpace(line))
				continue
			}
			if strings.Contains(line, "Gentleman-Programming") && strings.Contains(line, "gentle-ai") {
				t.Errorf("%s:%d pairs the upstream owner with this product: %q", rel, i+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/update: %v", err)
	}
}
