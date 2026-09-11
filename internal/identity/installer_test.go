package identity_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestPosixInstallerUsesAtomwrightIdentity asserts the POSIX installer publishes
// and downloads under the new release coordinates.
func TestPosixInstallerUsesAtomwrightIdentity(t *testing.T) {
	script := readRepositoryFile(t, "scripts/install.sh")

	for _, want := range []string{
		`BINARY_NAME="atomwright"`,
		`GITHUB_OWNER="pablogore"`,
		`GITHUB_REPO="atomwright"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("scripts/install.sh is missing %q", want)
		}
	}
}

// TestPosixInstallerHasNoHomebrewPath asserts Homebrew is removed from the
// installer entirely — no tap, no formula reference, no brew install/upgrade
// branch. Homebrew is no longer part of the release path.
func TestPosixInstallerHasNoHomebrewPath(t *testing.T) {
	script := readRepositoryFile(t, "scripts/install.sh")

	for _, forbidden := range []string{
		"BREW_TAP",
		"BREW_FORMULA_REF",
		"brew install",
		"brew upgrade",
		"brew tap",
		"homebrew-tap",
	} {
		if strings.Contains(script, forbidden) {
			t.Errorf("scripts/install.sh still references %q; Homebrew is removed from the release path", forbidden)
		}
	}
}

// TestPosixInstallerGoInstallUsesLiteralModulePath asserts the `go install`
// target is the LITERAL module path.
//
// The landmine: the installer used to compose its module path from the release
// coordinates, e.g. github.com/${owner_lc}/${GITHUB_REPO}/v2/cmd/${BINARY_NAME}.
// Now that the module path is github.com/pablogore/atomwright/v2, that
// composition happens to produce the same string — which makes this guard MORE
// important, not less. A composed path would look correct today and silently
// name a module that does not exist the day either coordinate changes, taking
// every source install with it. This is also the guard that keeps the module
// path from being derived from the branding variables at all.
func TestPosixInstallerGoInstallUsesLiteralModulePath(t *testing.T) {
	script := readRepositoryFile(t, "scripts/install.sh")

	const want = "github.com/pablogore/atomwright/v2/cmd/atomwright@"
	if !strings.Contains(script, want) {
		t.Errorf("scripts/install.sh go install target is missing the literal module path %q", want)
	}

	// Only a MODULE path composed from the release coordinates is a defect. The
	// release download and documentation URLs legitimately interpolate the same
	// variables, because published artifacts really do live at the release
	// coordinates. A Go module path is distinguished by its /v2 major-version
	// suffix, so that is what this asserts on.
	composed := regexp.MustCompile(`(?i)github\.com/\$\{?[a-z0-9_]*owner[a-z0-9_]*\}?/\$\{?github_repo\}?/v[0-9]`)
	for i, line := range strings.Split(script, "\n") {
		if composed.MatchString(line) {
			t.Errorf("scripts/install.sh:%d composes a Go module path from the release coordinates: %q", i+1, strings.TrimSpace(line))
		}
	}
}

// TestWindowsInstallerUsesAtomwrightIdentity asserts the PowerShell installer
// installs the new executable from the literal module path.
func TestWindowsInstallerUsesAtomwrightIdentity(t *testing.T) {
	script := readRepositoryFile(t, "scripts/install.ps1")

	for _, want := range []string{
		`$BINARY_NAME = "atomwright"`,
		`$GITHUB_REPO = "atomwright"`,
		"github.com/pablogore/atomwright/v2/cmd/atomwright",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("scripts/install.ps1 is missing %q", want)
		}
	}

	if strings.Contains(script, `cmd/gentle-ai`) {
		t.Errorf("scripts/install.ps1 still installs the cmd/gentle-ai package")
	}
}

// TestInstallersAcceptAtomwrightChannelWithDeprecatedAlias asserts both
// installers read ATOMWRIGHT_CHANNEL and still honour GENTLE_AI_CHANNEL as a
// deprecated alias, so an existing pinned-beta user is not silently moved back
// to stable by the rename.
func TestInstallersAcceptAtomwrightChannelWithDeprecatedAlias(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"posix", "scripts/install.sh"},
		{"windows", "scripts/install.ps1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := readRepositoryFile(t, tt.file)

			if !strings.Contains(script, "ATOMWRIGHT_CHANNEL") {
				t.Errorf("%s does not accept ATOMWRIGHT_CHANNEL", tt.file)
			}
			if !strings.Contains(script, "GENTLE_AI_CHANNEL") {
				t.Errorf("%s no longer reads the deprecated GENTLE_AI_CHANNEL alias", tt.file)
			}
		})
	}
}
