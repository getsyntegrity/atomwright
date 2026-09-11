package telemetry

import (
	"strings"

	"github.com/pablogore/atomwright/v2/internal/identity"
)

// Source names which input decided whether telemetry is enabled, in the
// exact precedence order the issue specifies.
type Source string

const (
	SourceDoNotTrack   Source = "DO_NOT_TRACK"
	SourceEnvOptOut    Source = "GENTLE_AI_TELEMETRY"
	SourceCI           Source = "CI"
	SourceStateDisable Source = "state"
	SourceDefault      Source = "default"
)

// telemetryEnvSuffix is the unprefixed opt-out variable. SourceEnvOptOut above
// keeps its inherited spelling on purpose: it is a pinned wire enum value in
// contracts/telemetry/v1, not a variable name.
const telemetryEnvSuffix = "TELEMETRY"

// Decision reports whether sending is allowed and which source decided it.
type Decision struct {
	Enabled bool
	Source  Source
}

// Getenv is the shape telemetry needs from the environment; production
// callers pass os.Getenv, tests pass a map lookup.
type Getenv func(key string) string

// Decide evaluates the kill switches in their documented precedence:
// DO_NOT_TRACK set to anything but empty, "0", or "false", then
// the TELEMETRY opt-out set to 0, then CI or GITHUB_ACTIONS set to anything but
// empty, "0", or "false", then the persisted state's enabled
// flag. The first one that opts out wins; with none present, telemetry is
// enabled by default.
func Decide(getenv Getenv, persisted State) Decision {
	if doNotTrack(getenv("DO_NOT_TRACK")) {
		return Decision{Enabled: false, Source: SourceDoNotTrack}
	}
	if lookup(getenv, telemetryEnvSuffix) == "0" {
		return Decision{Enabled: false, Source: SourceEnvOptOut}
	}
	if truthy(getenv("CI")) || truthy(getenv("GITHUB_ACTIONS")) {
		return Decision{Enabled: false, Source: SourceCI}
	}
	if !persisted.Enabled {
		return Decision{Enabled: false, Source: SourceStateDisable}
	}
	return Decision{Enabled: true, Source: SourceDefault}
}

// doNotTrack follows the console DO_NOT_TRACK convention: opted out for any
// value other than empty, "0", or "false" (case-insensitive, trimmed).
func doNotTrack(v string) bool { return truthy(v) }

// truthy reads a CI-style flag: set to anything but empty, "0", or "false".
// CI systems disagree on the value (GitHub Actions and most others export
// CI=true, some export CI=1), so equality with "true" is not enough.
func truthy(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v != "" && v != "0" && v != "false"
}

// Endpoint resolves the collector URL: the endpoint override when set and
// non-empty, otherwise DefaultEndpoint.
func Endpoint(getenv Getenv) string {
	if v := strings.TrimSpace(lookup(getenv, EndpointEnvSuffix)); v != "" {
		return v
	}
	return DefaultEndpoint
}

// lookup resolves suffix against the current prefix first and the inherited one
// second, mirroring envcompat.
//
// envcompat itself reads the real process environment, and every telemetry
// decision is taken through an injected Getenv so tests can describe an
// environment without mutating the process. Reusing it here would quietly
// bypass that injection, so the same precedence is applied to the caller's
// environment instead.
func lookup(getenv Getenv, suffix string) string {
	if v := getenv(identity.EnvPrefix() + suffix); v != "" {
		return v
	}
	return getenv(identity.LegacyEnvPrefix() + suffix)
}
