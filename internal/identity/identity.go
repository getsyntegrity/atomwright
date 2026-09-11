// Package identity is the single place that names Atomwright's public
// identifiers.
//
// Each identifier is an independent concept with its own lifetime. They are
// exposed as separate accessors, and deliberately not derived from one another,
// because collapsing them into a single branding constant is what makes a
// rename break installs:
//
//   - The executable is the token a user types and the name of the file placed
//     on PATH. It also names the member inside a release archive.
//   - The state directory is an on-disk location that existing installs already
//     wrote to, so it carries a migration obligation the executable does not.
//   - The environment prefix appears in user dotfiles and CI configuration that
//     this project cannot edit, so the legacy prefix stays readable.
//   - The release coordinates say where artifacts are published. They move when
//     ownership moves.
//   - The source module path is a permanent identifier of the code itself. It
//     is resolved by the Go module proxy and is NOT a branding surface; renaming
//     it would break every existing import and source install.
//
// The module path and the release coordinates are the pair most often conflated.
// They are asserted to be independent in the package tests.
package identity

const (
	executable         = "atomwright"
	stateDirName       = ".atomwright"
	envPrefix          = "ATOMWRIGHT_"
	legacyEnvPrefix    = "GENTLE_AI_"
	legacyStateDirName = ".gentle-ai"
	releaseOwner       = "pablogore"
	releaseRepo        = "atomwright"
	sourceModulePath   = "github.com/gentleman-programming/gentle-ai/v2"
)

// Executable is the public command name: the token a user types, the file
// installed on PATH, and the member name inside a release archive.
func Executable() string { return executable }

// StateDirName is the home-relative directory holding Atomwright's own state.
func StateDirName() string { return stateDirName }

// LegacyStateDirName is the inherited state directory. It is read for migration
// and never written to.
func LegacyStateDirName() string { return legacyStateDirName }

// EnvPrefix is the prefix for Atomwright's environment variables.
func EnvPrefix() string { return envPrefix }

// LegacyEnvPrefix is the inherited environment prefix, still accepted as a
// deprecated alias so existing dotfiles and CI configuration keep working.
func LegacyEnvPrefix() string { return legacyEnvPrefix }

// ReleaseOwner is the GitHub owner that publishes Atomwright releases.
func ReleaseOwner() string { return releaseOwner }

// ReleaseRepo is the GitHub repository that publishes Atomwright releases.
func ReleaseRepo() string { return releaseRepo }

// SourceModulePath is the Go module path of this codebase. It is inherited and
// intentionally unchanged: it is resolved by the module proxy, not a brand.
func SourceModulePath() string { return sourceModulePath }

// GoInstallPackage is the package a source install targets. It is the one place
// the preserved module path and the new executable name legitimately meet.
func GoInstallPackage() string { return sourceModulePath + "/cmd/" + executable }
