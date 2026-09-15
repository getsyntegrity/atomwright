package architecture

import (
	"strings"

	"github.com/pablogore/go-specs/mock"
)

// fakeToolchain is a hand-written stand-in for the Go toolchain, built against
// goRunner -- the interface the code under test declares -- and spying on
// every invocation so a spec can assert what was asked of the boundary and in
// what order.
//
// This is the only double in the package, and it stands at the only real
// collaborator boundary it has: process execution. The rules themselves are
// pure functions over a graph and get none, and the testdata/ fixtures run the
// real `go list` on purpose -- a hand-built graph would not be the graph
// production sees. What the real toolchain cannot demonstrate is its own
// failure modes, which is what this exists for.
type fakeToolchain struct {
	spy       *mock.Spy
	responses map[string]string
	failOn    string
	err       error
}

// run satisfies goRunner.
//
// The go arguments are recorded joined into one string so a spec can name the
// subcommand the way a reader would say it out loud -- "list -json ./..."
// rather than a three-element slice.
func (f *fakeToolchain) run(dir string, extraEnv []string, args ...string) (string, error) {
	command := strings.Join(args, " ")
	f.spy.Call(dir, strings.Join(extraEnv, " "), command)

	if f.err != nil && command == f.failOn {
		return "", f.err
	}
	return f.responses[command], nil
}

// workingToolchainResponses are the answers a healthy toolchain gives for the
// three subcommands buildGraph issues.
func workingToolchainResponses() map[string]string {
	return map[string]string{
		"list -m":          "example.test/mod\n",
		"list -json ./...": `{"ImportPath":"example.test/mod/internal/domain/execution"}`,
		"list std":         "errors\nfmt\nstrings\n",
	}
}

// recordingRunner answers each go subcommand with a canned response and never
// fails, so a spec can inspect what the boundary was asked for. Any subcommand
// the caller does not override is answered the way a healthy toolchain would,
// so a spec states only the answer it is actually about.
func recordingRunner(responses map[string]string) (goRunner, *mock.Spy) {
	answers := workingToolchainResponses()
	for command, response := range responses {
		answers[command] = response
	}
	f := &fakeToolchain{spy: mock.New().Spy("go"), responses: answers}
	return f.run, f.spy
}

// failingRunner answers like a healthy toolchain until the named subcommand,
// which fails with err.
func failingRunner(failOn string, err error) (goRunner, *mock.Spy) {
	f := &fakeToolchain{
		spy:       mock.New().Spy("go"),
		responses: workingToolchainResponses(),
		failOn:    failOn,
		err:       err,
	}
	return f.run, f.spy
}
