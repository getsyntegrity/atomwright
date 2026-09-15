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
// The scope is one file: .github/workflows/check.yml, the workflow this
// repository designates as the gate. It is deliberately not "every
// workflow that runs on a pull request", because that is not what a
// required check is. Whether a workflow blocks a merge lives in branch
// protection, not in the YAML, and plenty of legitimate pull-request
// automation -- previews, labels, comments, informational reports -- runs
// its own commands without ever gating anything. Constraining those would
// forbid the split pipeline this repository wants, so the guard names the
// gate instead of inferring it. If Atomwright ever grows a second gate,
// the way to include it is explicit metadata that classifies a workflow as
// one, not a wider trigger heuristic.
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

	// workflowPath is the gate. It is named, not discovered: see the
	// package comment for why a trigger heuristic is the wrong instrument.
	workflowPath = ".github/workflows/check.yml"

	// checkTarget is the single gate. CI runs this and only this.
	checkTarget = "check"
)

// requiredGates are the gates #66 requires `make check` to run. They are
// listed by Makefile target name, which is also how a contributor runs one
// in isolation.
var requiredGates = []string{"fmt", "vet", "tidy", "build", "test", "arch"}

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

// workflowRunCommands returns every shell command the gate workflow runs.
func workflowRunCommands(t *testing.T, path string) []string {
	t.Helper()

	var commands []string
	for _, line := range strings.Split(read(t, path), "\n") {
		match := runStepLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		command := match[1]
		if command == "|" || command == ">" || strings.HasPrefix(command, "|") || strings.HasPrefix(command, ">") {
			t.Fatalf("%s uses a multi-line run step (%q); keep every CI step a single `make <target>` call so it stays comparable with the Makefile", path, command)
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
			t.Errorf("`make %s` does not run the %q gate (prerequisites: %v); #66 requires fmt, vet, tidy, build, test, and the architecture checks", checkTarget, gate, prerequisites)
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

// The gate workflow runs `make check` and nothing else. A second step
// bolted onto it -- `go test -race ./...`, a coverage upload that fails
// the job -- is a gate that exists in CI and nowhere else, which is the
// drift this package exists to catch. Comparing parsed run commands rather
// than searching the file for a substring is what catches it: `make check`
// is present either way.
//
// Only this workflow is constrained. Any other workflow, whatever it
// triggers on, is free to run whatever it needs.
func TestTheCheckWorkflowRunsNothingButMakeCheck(t *testing.T) {
	want := "make " + checkTarget

	for _, command := range workflowRunCommands(t, workflowPath) {
		if command != want {
			t.Errorf("%s runs %q; every step of the gate must be `%s` so no gate exists in CI that a contributor cannot run locally", workflowPath, command, want)
		}
	}
}

// The mirror of the test above: running nothing but `make check` is also
// satisfied by running nothing, so the workflow that actually gates a PR
// has to invoke it -- on a pull request, which is what makes it a gate.
func TestTheCheckWorkflowRunsTheCheckTargetOnAPullRequest(t *testing.T) {
	want := "make " + checkTarget

	if !pullRequestTrigger(read(t, workflowPath)) {
		t.Fatalf("%s does not run on a pull request; it is the workflow that gates every PR", workflowPath)
	}

	if !slices.Contains(workflowRunCommands(t, workflowPath), want) {
		t.Errorf("%s never runs `%s`; the local gate and the PR gate must be the same target", workflowPath, want)
	}
}
