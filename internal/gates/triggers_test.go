package gates

import (
	"regexp"
	"strings"
	"testing"

	"github.com/pablogore/go-specs/specs"
)

var (
	// onKeyLine matches the top-level trigger key. GitHub only reads it at
	// column zero, and YAML lets it be quoted.
	onKeyLine = regexp.MustCompile(`^(?:on|"on"|'on'):\s*(.*)$`)

	// indentedLine matches a line that still belongs to the block opened by
	// the preceding top-level key.
	indentedLine = regexp.MustCompile(`^\s`)

	// pullRequestEvent matches the two events GitHub fires for a pull
	// request, as whole tokens so `pull_request_review` does not count.
	pullRequestEvent = regexp.MustCompile(`(^|[^0-9A-Za-z_])pull_request(_target)?([^0-9A-Za-z_]|$)`)
)

// pullRequestTrigger reports whether a workflow's `on:` block names a
// pull-request event. Only the trigger block is read: a `pull_request`
// mentioned in a job condition describes what a step does, not what the
// workflow runs on.
//
// This answers one question about one named workflow -- does the gate
// still run on pull requests -- and is not a way to discover which
// workflows are gates. A workflow with no readable `on:` block runs on
// nothing a pull request fires, so it reports false, and for the gate that
// is a failure.
func pullRequestTrigger(workflow string) bool {
	lines := strings.Split(workflow, "\n")

	for index, line := range lines {
		match := onKeyLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		// `on: [push, pull_request]` and `on: pull_request` carry the whole
		// trigger list on the key's own line.
		if inline := strings.TrimSpace(strings.SplitN(match[1], "#", 2)[0]); inline != "" {
			return pullRequestEvent.MatchString(inline)
		}

		// Otherwise the triggers are the indented block below the key.
		for _, blockLine := range lines[index+1:] {
			trimmed := strings.TrimSpace(blockLine)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if !indentedLine.MatchString(blockLine) {
				break
			}
			if pullRequestEvent.MatchString(trimmed) {
				return true
			}
		}
		return false
	}

	return false
}

// declaration is one workflow the check must read a verdict from, paired
// with the behaviour that reading it proves.
type declaration struct {
	behaviour string
	workflow  string
}

// recognised are the YAML shapes GitHub accepts for declaring a
// pull-request trigger. #66 ATOM-BOOT-006 is about the gate a contributor
// runs locally being the gate that blocks a PR, so the check has to see a
// pull-request trigger in every shape the gate workflow might use.
func recognisedTriggers() []declaration {
	return []declaration{
		{"recognises a pull-request trigger declared as an indented block",
			"name: check\n\non:\n  pull_request:\n  push:\n    branches: [main]\n"},
		{"recognises a pull-request trigger listed in a flow sequence",
			"on: [push, pull_request]\n"},
		{"recognises a pull-request trigger written as a bare scalar",
			"on: pull_request\n"},
		{"recognises a pull-request trigger under a quoted on key",
			"\"on\":\n  pull_request:\n"},
		{"treats pull_request_target as running on a pull request too",
			"on:\n  pull_request_target:\n    types: [opened]\n"},
		{"reads past a comment following the on key to the block below it",
			"on: # when this runs\n  pull_request:\n"},
	}
}

// nearMisses are the shapes that must not be mistaken for a pull-request
// trigger. Without them the check could report true for everything and
// still satisfy every case above.
func nearMissTriggers() []declaration {
	return []declaration{
		{"does not mistake pull_request_review for a pull-request trigger",
			"on:\n  pull_request_review:\n    types: [submitted]\n"},
		{"reports no pull-request trigger for a workflow that runs on tags and manual dispatch",
			"name: release\n\non:\n  push:\n    tags: ['v*']\n  workflow_dispatch:\n\njobs:\n  publish:\n    steps:\n      - run: make binary\n"},
		{"does not let a pull_request named in a job condition widen the trigger block",
			"on:\n  workflow_dispatch:\n\njobs:\n  publish:\n    if: github.event_name == 'pull_request'\n    steps:\n      - run: make binary\n"},
		{"reports no pull-request trigger for a workflow with no trigger block at all",
			"name: broken\n\njobs:\n  check:\n    steps:\n      - run: make check\n"},
	}
}

func TestPullRequestTrigger(t *testing.T) {
	specs.Describe(t, "the pull-request trigger check", func(s *specs.Spec) {
		s.When("a workflow declares a pull-request trigger", func(s *specs.Spec) {
			for _, c := range recognisedTriggers() {
				s.It(c.behaviour, func(ctx *specs.Context) {
					ctx.Expect(pullRequestTrigger(c.workflow)).To(specs.BeTrue())
				})
			}
		})

		s.When("a workflow declares something that only looks like one", func(s *specs.Spec) {
			for _, c := range nearMissTriggers() {
				s.It(c.behaviour, func(ctx *specs.Context) {
					ctx.Expect(pullRequestTrigger(c.workflow)).To(specs.BeFalse())
				})
			}
		})
	})
}
