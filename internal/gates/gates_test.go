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

// workflowFiles returns the repository-relative path of every workflow
// under .github/workflows.
func workflowFiles(t *testing.T) []string {
	t.Helper()

	workflowDir := filepath.Join(".github", "workflows")
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), workflowDir))
	if err != nil {
		t.Fatalf("reading %s: %v", workflowDir, err)
	}

	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || (!strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
			continue
		}
		paths = append(paths, filepath.ToSlash(filepath.Join(workflowDir, name)))
	}
	return paths
}

// workflowRunCommands returns every shell command one workflow runs.
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

// Every workflow, not just check.yml. A gate that runs in CI but not
// locally is the drift this package exists to catch, and the cheapest way
// for it to appear is a second workflow nobody remembers to mirror in the
// Makefile -- or an extra step bolted onto an existing one. Both are
// caught by comparing parsed run commands rather than searching the file
// for a substring, which a workflow running `make check` plus `go test
// -race ./...` would satisfy while still being a CI-only gate.
func TestCIRunsNothingButMakeCheck(t *testing.T) {
	want := "make " + checkTarget

	for _, path := range workflowFiles(t) {
		commands := workflowRunCommands(t, path)

		if len(commands) == 0 {
			t.Errorf("%s runs no command at all; every workflow must run `%s`", path, want)
			continue
		}

		for _, command := range commands {
			if command != want {
				t.Errorf("%s runs %q; every CI step must be `%s` so no gate exists in CI that a contributor cannot run locally", path, command, want)
			}
		}
	}
}

// The mirror of the test above: running nothing but `make check` is also
// satisfied by running nothing, so each workflow has to actually invoke it.
func TestEveryWorkflowRunsTheCheckTarget(t *testing.T) {
	want := "make " + checkTarget

	paths := workflowFiles(t)
	if !slices.Contains(paths, workflowPath) {
		t.Fatalf("%s is missing; it is the workflow that gates every PR", workflowPath)
	}

	for _, path := range paths {
		if !slices.Contains(workflowRunCommands(t, path), want) {
			t.Errorf("%s never runs `%s`; the local gate and the PR gate must be the same target", path, want)
		}
	}
}
