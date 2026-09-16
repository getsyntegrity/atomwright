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
//
// The suite is written with go-specs, like every other suite in the
// module. These specs live in the package rather than in an external one
// for the same reason internal/architecture's do: every file here is a
// _test.go file, so there is no exported surface an external test package
// could reach.
package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pablogore/go-specs/specs"
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

// checkCommand is the one command the gate workflow is allowed to run, and
// the one a contributor runs locally.
const checkCommand = "make " + checkTarget

func TestGates(t *testing.T) {
	specs.Describe(t, "the single gate every contributor and every pull request runs", func(s *specs.Spec) {
		s.Describe("the Makefile", func(s *specs.Spec) {
			s.When("the check target is the entry point for every gate", func(s *specs.Spec) {
				s.It("defines the check target at all", func(ctx *specs.Context) {
					_, ok := makefileTargets(ctx.T)[checkTarget]

					ctx.Expect(ok).To(specs.BeTrue())
				})

				// #66 requires fmt, vet, tidy, build, test, and the
				// architecture checks. One spec per gate names which one
				// went missing instead of reporting that something did.
				for _, gate := range requiredGates {
					s.It("runs the "+gate+" gate", func(ctx *specs.Context) {
						prerequisites, ok := makefileTargets(ctx.T)[checkTarget]
						if !ok {
							ctx.T.Fatalf("%s defines no %q target; it is the entry point for every gate", makefilePath, checkTarget)
						}

						ctx.Expect(slices.Contains(prerequisites, gate)).To(specs.BeTrue())
					})
				}

				s.It("depends on nothing the Makefile does not define as a target", func(ctx *specs.Context) {
					targets := makefileTargets(ctx.T)

					for _, prerequisite := range targets[checkTarget] {
						_, defined := targets[prerequisite]
						if !defined {
							ctx.T.Errorf("`make %s` depends on %q, which is not a target in %s", checkTarget, prerequisite, makefilePath)
						}
						ctx.Expect(defined).To(specs.BeTrue())
					}
				})
			})
		})

		// Only this workflow is constrained. Any other workflow, whatever
		// it triggers on, is free to run whatever it needs.
		s.Describe("the gate workflow", func(s *specs.Spec) {
			s.When("its run steps are compared with the Makefile", func(s *specs.Spec) {
				// The gate workflow runs `make check` and nothing else. A
				// second step bolted onto it -- `go test -race ./...`, a
				// coverage upload that fails the job -- is a gate that
				// exists in CI and nowhere else, which is the drift this
				// package exists to catch. Comparing parsed run commands
				// rather than searching the file for a substring is what
				// catches it: `make check` is present either way.
				s.It("runs no step other than the check target", func(ctx *specs.Context) {
					for _, command := range workflowRunCommands(ctx.T, workflowPath) {
						if command != checkCommand {
							ctx.T.Errorf("%s runs %q; every step of the gate must be `%s` so no gate exists in CI that a contributor cannot run locally", workflowPath, command, checkCommand)
						}
						specs.EqualTo(ctx, command, checkCommand)
					}
				})

				// The mirror of the spec above: running nothing but `make
				// check` is also satisfied by running nothing, so the
				// workflow that actually gates a PR has to invoke it.
				s.It("does run the check target, so the local gate and the PR gate are the same target", func(ctx *specs.Context) {
					ctx.Expect(slices.Contains(workflowRunCommands(ctx.T, workflowPath), checkCommand)).To(specs.BeTrue())
				})

				// ...and on a pull request, which is what makes it a gate.
				s.It("is triggered by a pull request", func(ctx *specs.Context) {
					ctx.Expect(pullRequestTrigger(read(ctx.T, workflowPath))).To(specs.BeTrue())
				})
			})
		})
	})
}
