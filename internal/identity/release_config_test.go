package identity_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// identityRepositoryRoot walks up from the test working directory until it
// finds go.mod, so the assertions below run against the real repository files
// regardless of where `go test` was invoked from.
func identityRepositoryRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	for {
		info, err := os.Stat(filepath.Join(dir, "go.mod"))
		switch {
		case err == nil && !info.IsDir():
			return dir
		case err == nil:
			t.Fatalf("go.mod under %q is not a file", dir)
		case !errors.Is(err, fs.ErrNotExist):
			t.Fatalf("stat go.mod under %q: %v", dir, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("locate repository root from test working directory: go.mod not found")
		}
		dir = parent
	}
}

// readRepositoryFile reads a repository-relative file as raw bytes.
func readRepositoryFile(t *testing.T, rel string) string {
	t.Helper()

	path := filepath.Join(identityRepositoryRoot(t), filepath.FromSlash(rel))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", rel, err)
	}
	return string(content)
}

// TestGoreleaserPublishesAtomwright asserts the release configuration names the
// new executable everywhere a user-visible artifact identity is declared.
func TestGoreleaserPublishesAtomwright(t *testing.T) {
	config := readRepositoryFile(t, ".goreleaser.yaml")

	for _, want := range []string{
		"project_name: atomwright",
		"binary: atomwright",
		"main: ./cmd/atomwright",
	} {
		if !strings.Contains(config, want) {
			t.Errorf(".goreleaser.yaml is missing %q", want)
		}
	}
}

// TestGoreleaserSignatureBindsAtomwrightReleaseIdentity asserts the minisign
// trusted comment binds the NEW release repository. The updater rejects a valid
// signature whose trusted comment names a different repository, so a stale
// trusted comment silently breaks every self-upgrade.
func TestGoreleaserSignatureBindsAtomwrightReleaseIdentity(t *testing.T) {
	config := readRepositoryFile(t, ".goreleaser.yaml")

	const wantTrusted = "repo=pablogore/atomwright;tag={{ .Tag }}"
	if !strings.Contains(config, wantTrusted) {
		t.Errorf(".goreleaser.yaml minisign trusted comment is missing %q", wantTrusted)
	}
	if strings.Contains(config, "repo=Gentleman-Programming/gentle-ai;") {
		t.Errorf(".goreleaser.yaml minisign trusted comment still binds the upstream repository Gentleman-Programming/gentle-ai")
	}
	if strings.Contains(config, "gentle-ai release") {
		t.Errorf(".goreleaser.yaml minisign untrusted comment still says %q", "gentle-ai release")
	}
}

// TestGoreleaserHasNoHomebrewPublishing asserts Homebrew is gone from the
// release path entirely: no formula is published and no tap token is consumed.
func TestGoreleaserHasNoHomebrewPublishing(t *testing.T) {
	config := readRepositoryFile(t, ".goreleaser.yaml")

	for _, line := range strings.Split(config, "\n") {
		if strings.HasPrefix(strings.TrimRight(line, " \t\r"), "brews:") {
			t.Errorf(".goreleaser.yaml still declares a brews: block: %q", line)
		}
	}
	if strings.Contains(config, "HOMEBREW_TAP_TOKEN") {
		t.Errorf(".goreleaser.yaml still references HOMEBREW_TAP_TOKEN")
	}
}

// TestGoreleaserLdflagsPreserveModulePath asserts the ldflags injection target
// still uses the PRESERVED Go module path. Renaming it here would silently stop
// injecting the release signing keys, leaving upgrades unverifiable.
func TestGoreleaserLdflagsPreserveModulePath(t *testing.T) {
	config := readRepositoryFile(t, ".goreleaser.yaml")

	const want = "github.com/gentleman-programming/gentle-ai/v2/internal/update/upgrade.releaseMinisignPublicKeys"
	if !strings.Contains(config, want) {
		t.Errorf(".goreleaser.yaml ldflags no longer reference the preserved module path symbol %q", want)
	}
}

// TestNoPublicReleaseArtifactNamedGentleAI is the hard safety net: after the
// rename, nothing in the release path may declare an artifact, binary, archive
// or asset named gentle-ai.
//
// Two substrings are explicitly allowed:
//   - "gentleman-programming/gentle-ai" — the PRESERVED Go module path.
//   - "gentle-ai-review-provider-contract-" — the frozen provider-contract
//     bundle name, which is a published contract identity and not the CLI.
//
// Every other occurrence of gentle-ai in these files is a failure.
func TestNoPublicReleaseArtifactNamedGentleAI(t *testing.T) {
	files := []string{
		".goreleaser.yaml",
		".github/workflows/release.yml",
		".github/workflows/promote-stable-rc.yml",
		"scripts/verify-release-assets.sh",
		"scripts/promote-stable-preflight.sh",
	}

	for _, rel := range files {
		t.Run(rel, func(t *testing.T) {
			content := readRepositoryFile(t, rel)
			lower := strings.ToLower(content)

			for offset := 0; ; {
				idx := strings.Index(lower[offset:], "gentle-ai")
				if idx < 0 {
					break
				}
				at := offset + idx
				offset = at + len("gentle-ai")

				if strings.HasSuffix(lower[:at], "gentleman-programming/") {
					continue
				}
				if strings.HasPrefix(lower[offset:], "-review-provider-contract-") {
					continue
				}

				line := 1 + strings.Count(content[:at], "\n")
				lineStart := strings.LastIndex(content[:at], "\n") + 1
				lineEnd := at + strings.IndexByte(content[at:]+"\n", '\n')
				t.Errorf("%s:%d declares a release artifact named gentle-ai: %q", rel, line, strings.TrimSpace(content[lineStart:lineEnd]))
			}
		})
	}
}

// TestCommandDirectoryIsAtomwright asserts the main package moved to
// cmd/atomwright and the old command directory no longer exists.
func TestCommandDirectoryIsAtomwright(t *testing.T) {
	root := identityRepositoryRoot(t)

	mainPath := filepath.Join(root, "cmd", "atomwright", "main.go")
	info, err := os.Stat(mainPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		t.Errorf("cmd/atomwright/main.go does not exist")
	case err != nil:
		t.Errorf("stat cmd/atomwright/main.go: %v", err)
	case info.IsDir():
		t.Errorf("cmd/atomwright/main.go is a directory, want a file")
	}

	legacy := filepath.Join(root, "cmd", "gentle-ai")
	if _, err := os.Lstat(legacy); err == nil {
		t.Errorf("cmd/gentle-ai/ still exists; the command directory must be renamed, not duplicated")
	} else if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("lstat cmd/gentle-ai: %v", err)
	}
}

// TestGoModulePathIsPreserved guards the single most dangerous accidental
// change: renaming the module itself would break every `go install` of any
// previously published version.
func TestGoModulePathIsPreserved(t *testing.T) {
	gomod := readRepositoryFile(t, "go.mod")

	const want = "module github.com/gentleman-programming/gentle-ai/v2"
	if !bytes.Contains([]byte(gomod), []byte(want)) {
		t.Fatalf("go.mod no longer declares %q; the module path must be preserved across the rename", want)
	}
}
