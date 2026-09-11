package styles

import (
	"strings"
	"testing"
)

// TestTaglinePresentsAtomwrightIdentity pins the welcome-screen product
// identity of this fork to Atomwright.
func TestTaglinePresentsAtomwrightIdentity(t *testing.T) {
	tagline := Tagline("v1.0.0-test")

	if !strings.Contains(tagline, "Atomwright") {
		t.Errorf("tagline does not present the Atomwright product identity: %q", tagline)
	}
	if !strings.Contains(tagline, "v1.0.0-test") {
		t.Errorf("tagline dropped the version it was given: %q", tagline)
	}
}
