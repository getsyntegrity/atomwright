package identity_test

import (
	"strings"
	"testing"
)

// TestHelpUsageInvokesAtomwright asserts the USAGE block prints the command the
// user actually types. This is a characterization test on the real bytes of
// internal/app/help.go: help text is the first thing a user copies, so a stale
// invocation token there is a user-visible failure, not a cosmetic one.
func TestHelpUsageInvokesAtomwright(t *testing.T) {
	help := readRepositoryFile(t, "internal/app/help.go")

	usageIdx := strings.Index(help, "USAGE")
	if usageIdx < 0 {
		t.Fatalf("internal/app/help.go has no USAGE block")
	}
	commandsIdx := strings.Index(help[usageIdx:], "COMMANDS")
	if commandsIdx < 0 {
		t.Fatalf("internal/app/help.go has no COMMANDS block after USAGE")
	}
	usage := help[usageIdx : usageIdx+commandsIdx]

	if !strings.Contains(usage, "atomwright <command> [flags]") {
		t.Errorf("internal/app/help.go USAGE block does not show %q", "atomwright <command> [flags]")
	}
	if !strings.Contains(usage, "Launch interactive TUI") || !strings.Contains(usage, "atomwright ") {
		t.Errorf("internal/app/help.go USAGE block does not invoke atomwright:\n%s", usage)
	}
	if strings.Contains(help, "gentle-ai") {
		t.Errorf("internal/app/help.go still shows the retired gentle-ai invocation token")
	}
}

// TestUninstallManualActionNamesAtomwrightExecutable asserts the manual cleanup
// hint tells the user to remove the binary they actually have on PATH. A stale
// hint here makes the user run a command that removes nothing and leaves the
// executable behind.
func TestUninstallManualActionNamesAtomwrightExecutable(t *testing.T) {
	service := readRepositoryFile(t, "internal/components/uninstall/service.go")

	const want = "rm -f $(which atomwright)"
	if !strings.Contains(service, want) {
		t.Errorf("internal/components/uninstall/service.go manual action does not tell the user %q", want)
	}
	if strings.Contains(service, "rm -f $(which gentle-ai)") {
		t.Errorf("internal/components/uninstall/service.go still tells the user to remove the retired gentle-ai executable")
	}
}
