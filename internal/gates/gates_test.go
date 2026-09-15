// The gates package is test-only. It makes one rule from #66
// ATOM-BOOT-006 executable: every gate a contributor runs locally is the
// same gate that blocks a PR.
//
// The rule has a specific failure mode worth guarding -- drift. Someone
// adds a step to the CI workflow and not to the Makefile (so the gate
// cannot be reproduced locally), or adds a Makefile gate and not to CI (so
// nothing blocks a PR that breaks it). Both are silent: the build stays
// green either way. These tests are what makes them loud.
//
// Nothing here ships in a binary: every file in this package is a
// _test.go file. Like internal/architecture it belongs to no ADR-0001
// layer, so no governed package may import it -- moduleDependencyRule
// enforces that.
package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const (
	makefilePath = "Makefile"
	workflowPath = ".github/workflows/check.yml"

	// checkTarget is the single gate. CI runs this and only this.
	checkTarget = "check"
)

// requiredGates are the gates #66 requires `make check` to run. They are
// listed by Makefile target name, which is also how a contributor runs one
// in isolation.
var requiredGates = []string{"fmt", "vet", "tidy", "test", "arch"}

var (
	targetLine  = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*):(.*)$`)
	runStepLine = regexp.MustCompile(`^\s*-?\s*run:\s*(.+?)\s*$`)
)

// repoRoot is this package's directory, two levels below the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving the repository root: %v", err)
	}
	return root
}

func read(t *testing.T, relPath string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repoRoot(t), relPath))
	if err != nil {
		t.Fatalf("reading %s: %v", relPath, err)
	}
	return string(content)
}

// makefileTargets returns every target in the Makefile mapped to its
// prerequisites.
func makefileTargets(t *testing.T) map[string][]string {
	t.Helper()

	targets := make(map[string][]string)
	for _, line := range strings.Split(read(t, makefilePath), "\n") {
		match := targetLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		targets[match[1]] = strings.Fields(match[2])
	}
	return targets
}

// workflowRunCommands returns every shell command the CI workflow runs.
func workflowRunCommands(t *testing.T) []string {
	t.Helper()

	var commands []string
	for _, line := range strings.Split(read(t, workflowPath), "\n") {
		match := runStepLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		command := match[1]
		if command == "|" || command == ">" || strings.HasPrefix(command, "|") || strings.HasPrefix(command, ">") {
			t.Fatalf("%s uses a multi-line run step (%q); keep every CI step a single `make <target>` call so it stays comparable with the Makefile", workflowPath, command)
		}
		commands = append(commands, command)
	}
	return commands
}

func TestCheckRunsEveryRequiredGate(t *testing.T) {
	targets := makefileTargets(t)

	prerequisites, ok := targets[checkTarget]
	if !ok {
		t.Fatalf("%s defines no %q target; it is the entry point for every gate", makefilePath, checkTarget)
	}

	for _, gate := range requiredGates {
		if !slices.Contains(prerequisites, gate) {
			t.Errorf("`make %s` does not run the %q gate (prerequisites: %v); #66 requires lint, vet, tidy, test, and the architecture checks", checkTarget, gate, prerequisites)
		}
	}
}

func TestEveryCheckPrerequisiteIsADefinedTarget(t *testing.T) {
	targets := makefileTargets(t)

	for _, prerequisite := range targets[checkTarget] {
		if _, ok := targets[prerequisite]; !ok {
			t.Errorf("`make %s` depends on %q, which is not a target in %s", checkTarget, prerequisite, makefilePath)
		}
	}
}

func TestCIRunsNothingButMakeCheck(t *testing.T) {
	commands := workflowRunCommands(t)

	if len(commands) == 0 {
		t.Fatalf("%s runs no command at all; CI must run `make %s`", workflowPath, checkTarget)
	}

	want := "make " + checkTarget
	for _, command := range commands {
		if command == want {
			continue
		}
		t.Errorf("%s runs %q; every CI step must be `%s` so no gate exists in CI that a contributor cannot run locally", workflowPath, command, want)
	}
}

func TestCIRunsTheCheckTarget(t *testing.T) {
	if !slices.Contains(workflowRunCommands(t), "make "+checkTarget) {
		t.Errorf("%s never runs `make %s`; the local gate and the PR gate must be the same target", workflowPath, checkTarget)
	}
}

// A gate that runs in CI but not locally is the drift this package exists
// to catch, and the cheapest way for it to appear is a second workflow
// nobody remembers to mirror in the Makefile. Every workflow under
// .github/workflows must therefore go through `make check` too.
func TestNoWorkflowBypassesMakeCheck(t *testing.T) {
	workflowDir := filepath.Join(repoRoot(t), ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Fatalf("reading %s: %v", workflowDir, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || (!strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
			continue
		}
		content := read(t, filepath.Join(".github", "workflows", name))
		if !strings.Contains(content, "make "+checkTarget) {
			t.Errorf(".github/workflows/%s never runs `make %s`; a workflow that gates PRs on anything else is a CI-only gate", name, checkTarget)
		}
	}
}
