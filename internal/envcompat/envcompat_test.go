package envcompat_test

import (
	"os"
	"strings"
	"testing"

	"github.com/pablogore/atomwright/v2/internal/envcompat"
)

// TestLookupResolvesPrefixedVariables covers the full state table of the
// ATOMWRIGHT_/GENTLE_AI_ environment prefix pair. Lookup takes the SUFFIX only
// (e.g. "CHANNEL") and resolves it against both prefixes so callers never
// hardcode a prefix and never have to remember the deprecated one.
func TestLookupResolvesPrefixedVariables(t *testing.T) {
	tests := []struct {
		name           string
		atomwright     string
		setAtomwright  bool
		legacy         string
		setLegacy      bool
		wantValue      string
		wantFound      bool
		wantDeprecated bool
	}{
		{
			name:          "only current prefix set",
			atomwright:    "beta",
			setAtomwright: true,
			wantValue:     "beta",
			wantFound:     true,
		},
		{
			name:           "only legacy prefix set is found and reported deprecated",
			legacy:         "beta",
			setLegacy:      true,
			wantValue:      "beta",
			wantFound:      true,
			wantDeprecated: true,
		},
		{
			name:           "both set with different values resolves to the current prefix",
			atomwright:     "stable",
			setAtomwright:  true,
			legacy:         "beta",
			setLegacy:      true,
			wantValue:      "stable",
			wantFound:      true,
			wantDeprecated: true,
		},
		{
			name:      "neither set is not found",
			wantValue: "",
			wantFound: false,
		},
		{
			name:          "current prefix set to empty string still counts as set",
			atomwright:    "",
			setAtomwright: true,
			wantValue:     "",
			wantFound:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const suffix = "CHANNEL"

			if tt.setAtomwright {
				t.Setenv("ATOMWRIGHT_"+suffix, tt.atomwright)
			} else {
				unsetForTest(t, "ATOMWRIGHT_"+suffix)
			}
			if tt.setLegacy {
				t.Setenv("GENTLE_AI_"+suffix, tt.legacy)
			} else {
				unsetForTest(t, "GENTLE_AI_"+suffix)
			}

			value, found, deprecated := envcompat.Lookup(suffix)
			if value != tt.wantValue {
				t.Errorf("Lookup(%q) value = %q, want %q", suffix, value, tt.wantValue)
			}
			if found != tt.wantFound {
				t.Errorf("Lookup(%q) found = %v, want %v", suffix, found, tt.wantFound)
			}
			if deprecated != tt.wantDeprecated {
				t.Errorf("Lookup(%q) deprecated = %v, want %v", suffix, deprecated, tt.wantDeprecated)
			}
		})
	}
}

// TestLookupIsDeterministic verifies repeated calls with identical environment
// return identical results. Resolution must not depend on os.Environ ordering
// or on any map iteration order.
func TestLookupIsDeterministic(t *testing.T) {
	t.Setenv("ATOMWRIGHT_CHANNEL", "stable")
	t.Setenv("GENTLE_AI_CHANNEL", "beta")

	wantValue, wantFound, wantDeprecated := envcompat.Lookup("CHANNEL")
	for i := range 32 {
		value, found, deprecated := envcompat.Lookup("CHANNEL")
		if value != wantValue || found != wantFound || deprecated != wantDeprecated {
			t.Fatalf("Lookup(%q) call %d = (%q, %v, %v), want (%q, %v, %v)", "CHANNEL", i, value, found, deprecated, wantValue, wantFound, wantDeprecated)
		}
	}
}

// TestDeprecationWarningNamesBothVariables verifies the warning tells the user
// exactly which variable is deprecated and exactly which one to use instead.
func TestDeprecationWarningNamesBothVariables(t *testing.T) {
	t.Setenv("GENTLE_AI_CHANNEL", "beta")
	unsetForTest(t, "ATOMWRIGHT_CHANNEL")

	warning := envcompat.Warning("CHANNEL")
	if warning == "" {
		t.Fatal("Warning(\"CHANNEL\") returned an empty message for a deprecated variable")
	}
	for _, want := range []string{"GENTLE_AI_CHANNEL", "ATOMWRIGHT_CHANNEL"} {
		if !strings.Contains(warning, want) {
			t.Errorf("Warning(%q) = %q, want it to name %q", "CHANNEL", warning, want)
		}
	}
}

// TestDeprecationWarningNeverLeaksValues is a secret-safety property: these
// variables carry tokens and credentials, and a deprecation warning is printed
// to the terminal and captured in CI logs. The warning names variables, never
// their contents.
func TestDeprecationWarningNeverLeaksValues(t *testing.T) {
	const secret = "super-secret-token"

	tests := []struct {
		name          string
		setAtomwright bool
		setLegacy     bool
	}{
		{name: "legacy only", setLegacy: true},
		{name: "current only", setAtomwright: true},
		{name: "both set", setAtomwright: true, setLegacy: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const suffix = "GITHUB_TOKEN"

			if tt.setAtomwright {
				t.Setenv("ATOMWRIGHT_"+suffix, secret)
			} else {
				unsetForTest(t, "ATOMWRIGHT_"+suffix)
			}
			if tt.setLegacy {
				t.Setenv("GENTLE_AI_"+suffix, secret)
			} else {
				unsetForTest(t, "GENTLE_AI_"+suffix)
			}

			if warning := envcompat.Warning(suffix); strings.Contains(warning, secret) {
				t.Errorf("Warning(%q) leaked the variable value: %q", suffix, warning)
			}
		})
	}
}

// TestWarningIsEmptyWhenNothingIsDeprecated verifies no warning is emitted when
// the user is already on the current prefix, or when nothing is set at all.
func TestWarningIsEmptyWhenNothingIsDeprecated(t *testing.T) {
	tests := []struct {
		name          string
		setAtomwright bool
	}{
		{name: "current prefix only", setAtomwright: true},
		{name: "nothing set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const suffix = "CHANNEL"

			if tt.setAtomwright {
				t.Setenv("ATOMWRIGHT_"+suffix, "stable")
			} else {
				unsetForTest(t, "ATOMWRIGHT_"+suffix)
			}
			unsetForTest(t, "GENTLE_AI_"+suffix)

			if warning := envcompat.Warning(suffix); warning != "" {
				t.Errorf("Warning(%q) = %q, want no warning", suffix, warning)
			}
		})
	}
}

// unsetForTest removes a variable for the duration of the test. t.Setenv
// registers the cleanup that restores the original process environment.
func unsetForTest(t *testing.T, name string) {
	t.Helper()

	t.Setenv(name, "")
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("unset %s: %v", name, err)
	}
}

// TestWarningsReportsEveryDeprecatedVariableOnce proves the startup path has a
// deterministic, complete list to report. A warning nothing calls is the same as
// no warning at all, which is what the deadcode ratchet caught.
func TestWarningsReportsEveryDeprecatedVariableOnce(t *testing.T) {
	t.Setenv("GENTLE_AI_CHANNEL", "beta")
	t.Setenv("GENTLE_AI_YES", "1")

	warnings := envcompat.Warnings()
	if len(warnings) != 2 {
		t.Fatalf("Warnings() returned %d messages, want 2: %q", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "GENTLE_AI_CHANNEL") || !strings.Contains(warnings[1], "GENTLE_AI_YES") {
		t.Errorf("Warnings() = %q, want CHANNEL before YES in AliasedSuffixes order", warnings)
	}
	for _, warning := range warnings {
		if strings.Contains(warning, "beta") {
			t.Errorf("Warnings() leaked a variable value: %q", warning)
		}
	}
}

// TestWarningsIsEmptyWithoutLegacyVariables keeps the startup path quiet for the
// users who never had the legacy prefix set.
func TestWarningsIsEmptyWithoutLegacyVariables(t *testing.T) {
	t.Setenv("ATOMWRIGHT_CHANNEL", "stable")

	if warnings := envcompat.Warnings(); len(warnings) != 0 {
		t.Errorf("Warnings() = %q, want none when no legacy variable is set", warnings)
	}
}
