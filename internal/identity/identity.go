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
//   - The source module path is Atomwright's own Go module path. It is a
//     separate concept from the release coordinates because it is what the Go
//     module proxy resolves and what every import statement in this repository
//     spells out — not because it is spelled differently.
//
// The module path and the release coordinates are the pair most often conflated.
// They currently share the same owner and repository segments, which makes the
// conflation cheap to introduce and invisible to a textual check: composing the
// module path from the release coordinates would produce the right string today
// and the wrong one the day either changes. They are therefore declared
// independently, and the package tests assert the declaration against the real
// go.mod rather than against a composed value.
package identity

const (
	executable         = "atomwright"
	stateDirName       = ".atomwright"
	envPrefix          = "ATOMWRIGHT_"
	legacyEnvPrefix    = "GENTLE_AI_"
	legacyStateDirName = ".gentle-ai"
	releaseOwner       = "pablogore"
	releaseRepo        = "atomwright"
	sourceModulePath   = "github.com/pablogore/atomwright/v2"
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

// SourceModulePath is the Go module path of this codebase, declared here as a
// literal and never composed from the release coordinates. It must stay equal
// to the module directive in go.mod: it is what the Go module proxy resolves
// and what every import path in this repository begins with.
func SourceModulePath() string { return sourceModulePath }

// GoInstallPackage is the package a source install targets. It is the one place
// the module path and the executable name legitimately meet.
func GoInstallPackage() string { return sourceModulePath + "/cmd/" + executable }
