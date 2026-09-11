package identity_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// networkFailureMarkers are the substrings that identify a module-fetch failure
// rather than a real assertion failure. The skip below is deliberately narrow:
// it fires only when the deadcode tool itself cannot be fetched, never when the
// script behaves incorrectly.
var networkFailureMarkers = []string{
	"dial tcp",
	"no such host",
	"connection refused",
	"i/o timeout",
	"proxy.golang.org",
	"module lookup disabled",
	"cannot find module providing",
	"unrecognized import path",
	"Get \"https://",
	"TLS handshake timeout",
}

func looksLikeToolFetchFailure(output string) bool {
	for _, marker := range networkFailureMarkers {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

// TestDeadcodeRatchetFailsLoudlyOnMissingTarget asserts the ratchet script
// refuses to pass when its analysis target does not exist.
//
// The script pipes the analyzer's stderr to /dev/null. When DEADCODE_TARGET
// names a package that does not exist, the analyzer fails, the current set
// comes back empty, every baselined entry looks "now reachable or gone", and
// the script reports success. A guard that silently degrades into a passing
// no-op is worse than no guard: the rename moves cmd/gentle-ai to
// cmd/atomwright, which is exactly the edit that would leave DEADCODE_TARGET
// pointing at nothing.
func TestDeadcodeRatchetFailsLoudlyOnMissingTarget(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash unavailable: %v", err)
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go toolchain unavailable: %v", err)
	}

	root := identityRepositoryRoot(t)

	// Narrow, explicit skip: probe whether the pinned analyzer can be fetched at
	// all. Only a tool-fetch/network failure skips this test; an assertion
	// failure below never does.
	probe := exec.Command("go", "run", "golang.org/x/tools/cmd/deadcode@v0.30.0", "--help")
	probe.Dir = root
	probeOut, _ := probe.CombinedOutput()
	if looksLikeToolFetchFailure(string(probeOut)) {
		t.Skipf("deadcode analyzer cannot be fetched in this environment; skipping:\n%s", probeOut)
	}

	const missingTarget = "./cmd/this-package-does-not-exist"

	cmd := exec.Command("bash", "scripts/deadcode-ratchet.sh")
	cmd.Dir = root
	cmd.Env = append(cmd.Environ(), "DEADCODE_TARGET="+missingTarget)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	combined := stdout.String() + stderr.String()
	if runErr == nil && looksLikeToolFetchFailure(combined) {
		t.Skipf("deadcode tool could not be fetched offline; skipping:\n%s", combined)
	}

	if runErr == nil {
		t.Fatalf("scripts/deadcode-ratchet.sh exited 0 with DEADCODE_TARGET=%q; a missing analysis target must fail loudly, not degrade into a passing no-op.\noutput:\n%s", missingTarget, combined)
	}

	if !strings.Contains(combined, missingTarget) {
		t.Errorf("scripts/deadcode-ratchet.sh failed without naming the bad target %q in its diagnostic.\noutput:\n%s", missingTarget, combined)
	}
}
