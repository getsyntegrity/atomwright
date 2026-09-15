package gates

import (
	"regexp"
	"strings"
	"testing"
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

// #66 ATOM-BOOT-006 is about the gate a contributor runs locally being the
// gate that blocks a PR, so the gate workflow has to actually run on a
// pull request. These cases cover the YAML shapes GitHub accepts for that
// declaration, and the near misses that must not be mistaken for it.
func TestPullRequestTrigger(t *testing.T) {
	tests := []struct {
		name            string
		workflow        string
		wantPullRequest bool
	}{
		{
			name:            "block form naming pull_request",
			workflow:        "name: check\n\non:\n  pull_request:\n  push:\n    branches: [main]\n",
			wantPullRequest: true,
		},
		{
			name:            "flow sequence naming pull_request",
			workflow:        "on: [push, pull_request]\n",
			wantPullRequest: true,
		},
		{
			name:            "scalar naming pull_request",
			workflow:        "on: pull_request\n",
			wantPullRequest: true,
		},
		{
			name:            "quoted key",
			workflow:        "\"on\":\n  pull_request:\n",
			wantPullRequest: true,
		},
		{
			name:            "pull_request_target also runs on a pull request",
			workflow:        "on:\n  pull_request_target:\n    types: [opened]\n",
			wantPullRequest: true,
		},
		{
			name:            "pull_request_review is a different event",
			workflow:        "on:\n  pull_request_review:\n    types: [submitted]\n",
			wantPullRequest: false,
		},
		{
			name:            "a release workflow does not run on a pull request",
			workflow:        "name: release\n\non:\n  push:\n    tags: ['v*']\n  workflow_dispatch:\n\njobs:\n  publish:\n    steps:\n      - run: make binary\n",
			wantPullRequest: false,
		},
		{
			name:            "a job mentioning pull_request does not widen the trigger block",
			workflow:        "on:\n  workflow_dispatch:\n\njobs:\n  publish:\n    if: github.event_name == 'pull_request'\n    steps:\n      - run: make binary\n",
			wantPullRequest: false,
		},
		{
			name:            "a comment after the key does not hide the block",
			workflow:        "on: # when this runs\n  pull_request:\n",
			wantPullRequest: true,
		},
		{
			name:            "a workflow with no trigger block runs on nothing",
			workflow:        "name: broken\n\njobs:\n  check:\n    steps:\n      - run: make check\n",
			wantPullRequest: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := pullRequestTrigger(test.workflow); got != test.wantPullRequest {
				t.Errorf("triggers on a pull request = %v, want %v", got, test.wantPullRequest)
			}
		})
	}
}
