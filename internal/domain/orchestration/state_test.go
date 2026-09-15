package orchestration_test

// These tests pin the State enum's closed-set behaviour. The point of the
// type is that a State can only ever be one of the declared constants, so the
// interesting cases are the ones at the edges of that set: the zero value, a
// value past the last constant, and every string that is nearly but not
// exactly a state name.

import (
	"errors"
	"testing"

	"github.com/getsyntegrity/atomwright/internal/domain/orchestration"
)

func TestZeroStateIsUnknownAndInvalid(t *testing.T) {
	// An uninitialised State must fail closed rather than silently mean
	// "Received". This is the whole reason StateUnknown occupies iota 0.
	var zero orchestration.State

	if zero != orchestration.StateUnknown {
		t.Fatalf("zero State = %v, want StateUnknown", zero)
	}
	if zero.IsValid() {
		t.Error("zero State reports IsValid() = true, want false")
	}
	if zero.IsTerminal() {
		t.Error("zero State reports IsTerminal() = true, want false")
	}
	if got := zero.String(); got != "Unknown" {
		t.Errorf("zero State String() = %q, want %q", got, "Unknown")
	}
}

func TestStateStringSpellsEveryDeclaredState(t *testing.T) {
	cases := []struct {
		state orchestration.State
		want  string
	}{
		{orchestration.StateReceived, "Received"},
		{orchestration.StateAnalyzing, "Analyzing"},
		{orchestration.StateComplexityCheck, "ComplexityCheck"},
		{orchestration.StateSpecReady, "SpecReady"},
		{orchestration.StateDecomposing, "Decomposing"},
		{orchestration.StateAwaitingApproval, "AwaitingApproval"},
		{orchestration.StatePlanning, "Planning"},
		{orchestration.StateExecuting, "Executing"},
		{orchestration.StateVerifying, "Verifying"},
		{orchestration.StateAwaitingReview, "AwaitingReview"},
		{orchestration.StateReadyToShip, "ReadyToShip"},
		{orchestration.StateCompleted, "Completed"},
		{orchestration.StateCancelled, "Cancelled"},
		{orchestration.StateFailed, "Failed"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.state.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}

	if len(cases) != len(orchestration.States()) {
		t.Fatalf("this test spells %d states, States() reports %d -- a state was added without a spelling case", len(cases), len(orchestration.States()))
	}
}

func TestOutOfRangeStateRendersUnknownWithoutPanicking(t *testing.T) {
	// A State can be conjured by conversion from any uint8. Rendering must
	// degrade to "Unknown" rather than panic or index out of bounds, because
	// a log line is not a good place to discover a corrupt value.
	out := orchestration.State(200)

	if got := out.String(); got != "Unknown" {
		t.Errorf("State(200).String() = %q, want %q", got, "Unknown")
	}
	if out.IsValid() {
		t.Error("State(200).IsValid() = true, want false")
	}
	if out.IsTerminal() {
		t.Error("State(200).IsTerminal() = true, want false")
	}
}

func TestEveryDeclaredStateIsValid(t *testing.T) {
	for _, state := range orchestration.States() {
		t.Run(state.String(), func(t *testing.T) {
			if !state.IsValid() {
				t.Errorf("IsValid() = false, want true")
			}
		})
	}
}

func TestTerminalSetIsExactlyCompletedCancelledAndFailed(t *testing.T) {
	// The terminal set is a contract, not an implementation detail: Next
	// refuses to move out of it. Asserting the exact membership catches a
	// state being quietly promoted or demoted.
	terminal := map[orchestration.State]bool{
		orchestration.StateCompleted: true,
		orchestration.StateCancelled: true,
		orchestration.StateFailed:    true,
	}

	for _, state := range orchestration.States() {
		t.Run(state.String(), func(t *testing.T) {
			if got := state.IsTerminal(); got != terminal[state] {
				t.Errorf("IsTerminal() = %t, want %t", got, terminal[state])
			}
			if state.IsTerminal() && !state.IsValid() {
				t.Error("terminal state reports IsValid() = false; a terminal state must also be valid")
			}
		})
	}
}

func TestParseStateRoundTripsEveryStateSpelling(t *testing.T) {
	for _, want := range orchestration.States() {
		t.Run(want.String(), func(t *testing.T) {
			got, err := orchestration.ParseState(want.String())
			if err != nil {
				t.Fatalf("ParseState(%q) returned error: %v", want.String(), err)
			}
			if got != want {
				t.Errorf("ParseState(%q) = %v, want %v", want.String(), got, want)
			}
		})
	}
}

func TestParseStateRejectsAnythingButAnExactSpelling(t *testing.T) {
	// ParseState is the boundary constructor. Everything that is not a
	// character-for-character match of a declared spelling must fail closed,
	// including the spelling of the invalid state itself.
	cases := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "the unknown sentinel is not parseable", input: "Unknown"},
		{name: "lowercase spelling", input: "received"},
		{name: "uppercase spelling", input: "RECEIVED"},
		{name: "surrounding whitespace", input: " Received "},
		{name: "snake case spelling", input: "complexity_check"},
		{name: "prefix of a valid spelling", input: "Complexity"},
		{name: "unrelated garbage", input: "not-a-state"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := orchestration.ParseState(tc.input)
			if err == nil {
				t.Fatalf("ParseState(%q) = %v, want an error", tc.input, got)
			}
			if !errors.Is(err, orchestration.ErrUnknownState) {
				t.Errorf("ParseState(%q) error = %v, want it to wrap ErrUnknownState", tc.input, err)
			}
			if got != orchestration.StateUnknown {
				t.Errorf("ParseState(%q) = %v, want StateUnknown alongside the error", tc.input, got)
			}
		})
	}
}

func TestStatesReturnsAFreshSliceEachCall(t *testing.T) {
	// A caller that mutates the returned slice must not be able to corrupt
	// the enum for everyone else in the process.
	first := orchestration.States()
	if len(first) == 0 {
		t.Fatal("States() returned nothing")
	}
	first[0] = orchestration.StateUnknown

	second := orchestration.States()
	if second[0] == orchestration.StateUnknown {
		t.Error("mutating the result of States() changed a later call; the slice is shared package state")
	}
	if second[0] != orchestration.StateReceived {
		t.Errorf("States()[0] = %v, want StateReceived (declaration order)", second[0])
	}
}
