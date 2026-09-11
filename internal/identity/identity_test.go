package identity_test

import (
	"strings"
	"testing"

	"github.com/pablogore/atomwright/v2/internal/identity"
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
		{"SourceModulePath", identity.SourceModulePath(), "github.com/pablogore/atomwright/v2"},
		{"GoInstallPackage", identity.GoInstallPackage(), "github.com/pablogore/atomwright/v2/cmd/atomwright"},
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

// TestSourceModulePathMatchesGoMod pins identity.SourceModulePath() to the
// module directive in the repository's own go.mod.
//
// The failure this prevents is the identity package silently drifting from the
// real module path: go.mod is what the Go module proxy and every import path
// actually follow, so a stale constant here makes `go install` and the
// installer scripts target a package that does not exist, while every unit
// test that compares the constant to itself still passes.
//
// This replaces an earlier textual assertion that the module path did not
// contain the release owner or repository. After the module moved to
// github.com/pablogore/atomwright/v2 it legitimately contains both, so that
// wording no longer expressed the invariant. The real invariant — that the
// module path is declared literally and never COMPOSED from the branding
// variables — is enforced where the composition actually happened, in
// TestPosixInstallerGoInstallUsesLiteralModulePath over scripts/install.sh.
func TestSourceModulePathMatchesGoMod(t *testing.T) {
	gomod := readRepositoryFile(t, "go.mod")

	var declared string
	for _, line := range strings.Split(gomod, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			declared = strings.TrimSpace(rest)
			break
		}
	}
	if declared == "" {
		t.Fatal("go.mod declares no module directive")
	}

	if got := identity.SourceModulePath(); got != declared {
		t.Errorf("SourceModulePath() = %q, want the go.mod module directive %q; the identity package must never drift from the real module path", got, declared)
	}
}

// TestGoInstallPackageIsModulePathPlusExecutable verifies the install target is
// the module path joined with the executable command directory — the one place
// where the two otherwise-independent concepts legitimately meet.
func TestGoInstallPackageIsModulePathPlusExecutable(t *testing.T) {
	want := identity.SourceModulePath() + "/cmd/" + identity.Executable()
	if got := identity.GoInstallPackage(); got != want {
		t.Errorf("GoInstallPackage() = %q, want %q", got, want)
	}
}
