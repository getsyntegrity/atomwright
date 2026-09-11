// Package envcompat resolves Atomwright environment variables across the
// current and the inherited environment prefix.
//
// The prefix changed during the Atomwright rename, but the variables live in
// user dotfiles and CI configuration this project cannot edit. Reading the
// legacy prefix is therefore a permanent compatibility obligation rather than a
// transitional one, and callers must never spell either prefix themselves: they
// pass the suffix and this package decides which prefix answers.
package envcompat

import (
	"fmt"
	"os"

	"github.com/pablogore/atomwright/v2/internal/identity"
)

// Lookup resolves suffix against the current prefix first and the legacy prefix
// second.
//
// found distinguishes "set to the empty string" from "not set", because an
// explicitly empty value is a deliberate choice by the user and must not be
// silently replaced by a default.
//
// deprecated reports that the legacy variable is present, independently of
// which one supplied the value: a user who sets both still needs to be told the
// legacy one no longer decides anything.
func Lookup(suffix string) (value string, found bool, deprecated bool) {
	legacyValue, legacyFound := os.LookupEnv(identity.LegacyEnvPrefix() + suffix)
	if current, currentFound := os.LookupEnv(identity.EnvPrefix() + suffix); currentFound {
		return current, true, legacyFound
	}
	return legacyValue, legacyFound, legacyFound
}

// Warning returns the message to show when the legacy variable is set, and an
// empty string otherwise.
//
// The message names the two variables and never their contents. These variables
// carry tokens and credentials, and the warning is printed to a terminal and
// captured verbatim in CI logs, so interpolating the value would turn a
// migration hint into a credential leak.
func Warning(suffix string) string {
	if _, found := os.LookupEnv(identity.LegacyEnvPrefix() + suffix); !found {
		return ""
	}
	return fmt.Sprintf(
		"%s%s is deprecated; use %s%s instead. The deprecated variable is still read, and %s%s wins when both are set.",
		identity.LegacyEnvPrefix(), suffix,
		identity.EnvPrefix(), suffix,
		identity.EnvPrefix(), suffix,
	)
}

// AliasedSuffixes is the roster of runtime variables that honour both prefixes.
//
// It deliberately excludes names that merely share the legacy prefix but are not
// environment variables at all: the GENTLE_AI_REVIEW_* reviewer prompt markers,
// the {{GENTLE_AI_*}} template placeholders, and the GENTLE_AI_TELEMETRY value
// pinned as an enum in the published telemetry contract. Aliasing any of those
// would change a wire format rather than a user's configuration.
var AliasedSuffixes = []string{
	"CHANNEL",
	"CODEX_REVIEWER_LOOPBACK_BASE_URL",
	"ENGRAM_SETUP_MODE",
	"ENGRAM_SETUP_STRICT",
	"INSTALL_SCOPE",
	"NO_ANIMATION",
	"NO_SELF_UPDATE",
	"OPENCODE_BACKGROUND_SUBAGENTS",
	"PI_BACKGROUND_SUBAGENTS",
	"SDD_STATUS_ENGRAM",
	"SELF_UPDATE_DONE",
	"TELEMETRY",
	"TELEMETRY_ENDPOINT",
	"YES",
}

// Warnings returns one deprecation message per aliased variable currently set
// under the legacy prefix, in AliasedSuffixes order so repeated runs are
// deterministic.
//
// Callers resolve values through Lookup, which cannot warn on its own without
// printing the same notice once per read. Collecting the warnings here lets the
// startup path report each deprecated variable exactly once per run.
func Warnings() []string {
	var warnings []string
	for _, suffix := range AliasedSuffixes {
		if warning := Warning(suffix); warning != "" {
			warnings = append(warnings, warning)
		}
	}
	return warnings
}
