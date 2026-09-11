package identity_test

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v2/internal/identity"
)

// TestIdentityValues pins every identity concept to an exact value. The six
// concepts are deliberately separate accessors rather than derivations of a
// single branding constant: the executable name, the on-disk state directory,
// the environment prefix, the release coordinates and the Go module path
// changed independently during the Atomwright rename, and collapsing them into
// one constant is what makes a rename silently break installs.
func TestIdentityValues(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"Executable", identity.Executable(), "atomwright"},
		{"StateDirName", identity.StateDirName(), ".atomwright"},
		{"EnvPrefix", identity.EnvPrefix(), "ATOMWRIGHT_"},
		{"LegacyEnvPrefix", identity.LegacyEnvPrefix(), "GENTLE_AI_"},
		{"LegacyStateDirName", identity.LegacyStateDirName(), ".gentle-ai"},
		{"ReleaseOwner", identity.ReleaseOwner(), "pablogore"},
		{"ReleaseRepo", identity.ReleaseRepo(), "atomwright"},
		{"SourceModulePath", identity.SourceModulePath(), "github.com/gentleman-programming/gentle-ai/v2"},
		{"GoInstallPackage", identity.GoInstallPackage(), "github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s() = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestIdentityConceptsAreDistinctValues verifies that the identity concepts are
// not aliases of one another. A single branding constant reused for all of them
// would make every assertion above pass in lockstep and hide the real coupling.
func TestIdentityConceptsAreDistinctValues(t *testing.T) {
	if identity.Executable() == identity.StateDirName() {
		t.Errorf("Executable() and StateDirName() are the same value %q; the state directory is dot-prefixed and must be its own concept", identity.Executable())
	}
	if identity.EnvPrefix() == identity.LegacyEnvPrefix() {
		t.Errorf("EnvPrefix() and LegacyEnvPrefix() are the same value %q; the deprecated alias must remain distinguishable", identity.EnvPrefix())
	}
	if identity.StateDirName() == identity.LegacyStateDirName() {
		t.Errorf("StateDirName() and LegacyStateDirName() are the same value %q; migration cannot detect legacy state otherwise", identity.StateDirName())
	}
}

// TestSourceModulePathIsDecoupledFromReleaseCoordinates proves the module path
// is NOT derived from the release owner/repo.
//
// This is the documented landmine in scripts/install.sh: the installer composed
// its `go install` target as github.com/${GITHUB_OWNER}/${GITHUB_REPO}/v2/cmd/...
// Once the release coordinates moved to pablogore/atomwright, that composition
// resolves to a module that does not exist, and every source install breaks.
// The Go module path is a permanent identifier of the code and deliberately
// stays github.com/gentleman-programming/gentle-ai/v2; the release coordinates
// are where artifacts are published. They must never be derived from each other.
func TestSourceModulePathIsDecoupledFromReleaseCoordinates(t *testing.T) {
	module := identity.SourceModulePath()
	owner := identity.ReleaseOwner()
	repo := identity.ReleaseRepo()

	if !strings.Contains(module, "gentleman-programming/gentle-ai") {
		t.Fatalf("SourceModulePath() = %q, want it to preserve the original module path substring %q", module, "gentleman-programming/gentle-ai")
	}
	if strings.Contains(module, owner) {
		t.Errorf("SourceModulePath() = %q contains ReleaseOwner() %q; the module path must not be derived from the release coordinates", module, owner)
	}
	if strings.Contains(module, repo) {
		t.Errorf("SourceModulePath() = %q contains ReleaseRepo() %q; the module path must not be derived from the release coordinates", module, repo)
	}
	if composed := "github.com/" + owner + "/" + repo + "/v2"; module == composed {
		t.Errorf("SourceModulePath() = %q equals the composed release path %q; composing the module path from release coordinates breaks `go install`", module, composed)
	}
}

// TestGoInstallPackageIsModulePathPlusExecutable verifies the install target is
// the preserved module path joined with the NEW executable command directory —
// the one place where the two otherwise-independent concepts legitimately meet.
func TestGoInstallPackageIsModulePathPlusExecutable(t *testing.T) {
	want := identity.SourceModulePath() + "/cmd/" + identity.Executable()
	if got := identity.GoInstallPackage(); got != want {
		t.Errorf("GoInstallPackage() = %q, want %q", got, want)
	}
}
