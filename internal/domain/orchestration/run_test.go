package orchestration_test

// Run is the aggregate that owns a state and refuses to leave it illegally.
// These tests walk the paths the issue's diagram describes end to end, so a
// broken edge shows up as a broken journey rather than as an abstract table
// mismatch.

import (
	"errors"
	"testing"

	"github.com/getsyntegrity/atomwright/internal/domain/orchestration"
)

func TestNewRunIDAcceptsAMeaningfulIdentifier(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "an opaque token", input: "run-01J8Z9"},
		{name: "inner spaces are the caller's business", input: "run 42"},
		{name: "a single character", input: "r"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := orchestration.NewRunID(tc.input)
			if err != nil {
				t.Fatalf("NewRunID(%q) returned error: %v", tc.input, err)
			}
			if id.String() != tc.input {
				t.Errorf("id.String() = %q, want %q", id.String(), tc.input)
			}
		})
	}
}

func TestNewRunIDRejectsAnIdentifierThatCarriesNoIdentity(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  error
	}{
		{name: "empty", input: "", want: orchestration.ErrEmptyRunID},
		{name: "only spaces", input: "   ", want: orchestration.ErrEmptyRunID},
		{name: "only a tab", input: "\t", want: orchestration.ErrEmptyRunID},
		{name: "only a newline", input: "\n", want: orchestration.ErrEmptyRunID},
		{name: "leading whitespace", input: " run-1", want: orchestration.ErrInvalidRunID},
		{name: "trailing whitespace", input: "run-1 ", want: orchestration.ErrInvalidRunID},
		{name: "a trailing newline", input: "run-1\n", want: orchestration.ErrInvalidRunID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := orchestration.NewRunID(tc.input)
			if err == nil {
				t.Fatalf("NewRunID(%q) = %q, want an error", tc.input, id)
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("NewRunID(%q) error = %v, want it to wrap %v", tc.input, err, tc.want)
			}
			if id != "" {
				t.Errorf("NewRunID(%q) = %q, want the zero RunID alongside the error", tc.input, id)
			}
		})
	}
}

func TestNewRunStartsInReceived(t *testing.T) {
	// Received is the diagram's entry point, and a fresh Run must not be
	// mistakable for one already in flight.
	run := newRunForTest(t, "run-1")

	if run.State() != orchestration.StateReceived {
		t.Errorf("State() = %v, want StateReceived", run.State())
	}
	if run.ID().String() != "run-1" {
		t.Errorf("ID() = %q, want %q", run.ID(), "run-1")
	}
	if run.IsTerminal() {
		t.Error("a fresh run reports IsTerminal() = true, want false")
	}
}

func TestNewRunRejectsARunWithoutIdentity(t *testing.T) {
	// The zero RunID is what a caller gets when they ignore NewRunID's error,
	// so NewRun refuses it rather than creating an untraceable run.
	run, err := orchestration.NewRun("")
	if err == nil {
		t.Fatalf("NewRun(\"\") = %+v, want an error", run)
	}
	if !errors.Is(err, orchestration.ErrEmptyRunID) {
		t.Errorf("error = %v, want it to wrap ErrEmptyRunID", err)
	}
}

func TestRunWalksTheHappyPathToCompleted(t *testing.T) {
	// The straight-through journey: a request arrives, proves small enough to
	// spec directly, is approved, planned, executed, verified, reviewed, and
	// shipped.
	run := newRunForTest(t, "run-happy")

	walk(t, &run, []step{
		{orchestration.ReasonAnalysisStarted, orchestration.StateAnalyzing},
		{orchestration.ReasonAnalysisCompleted, orchestration.StateComplexityCheck},
		{orchestration.ReasonWithinComplexityBudget, orchestration.StateSpecReady},
		{orchestration.ReasonApprovalRequested, orchestration.StateAwaitingApproval},
		{orchestration.ReasonApprovalGranted, orchestration.StatePlanning},
		{orchestration.ReasonPlanReady, orchestration.StateExecuting},
		{orchestration.ReasonExecutionCompleted, orchestration.StateVerifying},
		{orchestration.ReasonVerificationPassed, orchestration.StateAwaitingReview},
		{orchestration.ReasonReviewApproved, orchestration.StateReadyToShip},
		{orchestration.ReasonShipped, orchestration.StateCompleted},
	})

	if !run.IsTerminal() {
		t.Error("a completed run reports IsTerminal() = false, want true")
	}
}

func TestRunLoopsThroughDecompositionUntilItFitsTheBudget(t *testing.T) {
	// Decomposing feeds back into ComplexityCheck, so a run that is too large
	// is re-measured after being split rather than being pushed forward
	// regardless.
	run := newRunForTest(t, "run-decompose")

	walk(t, &run, []step{
		{orchestration.ReasonAnalysisStarted, orchestration.StateAnalyzing},
		{orchestration.ReasonAnalysisCompleted, orchestration.StateComplexityCheck},
		{orchestration.ReasonOverComplexityBudget, orchestration.StateDecomposing},
		{orchestration.ReasonDecompositionCompleted, orchestration.StateComplexityCheck},
		{orchestration.ReasonWithinComplexityBudget, orchestration.StateSpecReady},
	})
}

func TestRunReachesFailedWhenVerificationRejectsTheWork(t *testing.T) {
	run := newRunForTest(t, "run-failed")

	walk(t, &run, []step{
		{orchestration.ReasonAnalysisStarted, orchestration.StateAnalyzing},
		{orchestration.ReasonAnalysisCompleted, orchestration.StateComplexityCheck},
		{orchestration.ReasonWithinComplexityBudget, orchestration.StateSpecReady},
		{orchestration.ReasonApprovalRequested, orchestration.StateAwaitingApproval},
		{orchestration.ReasonApprovalGranted, orchestration.StatePlanning},
		{orchestration.ReasonPlanReady, orchestration.StateExecuting},
		{orchestration.ReasonExecutionCompleted, orchestration.StateVerifying},
		{orchestration.ReasonVerificationFailed, orchestration.StateFailed},
	})

	if !run.IsTerminal() {
		t.Error("a failed run reports IsTerminal() = false, want true")
	}
}

func TestRunCanBeCancelledAtEveryPointTheDiagramAllows(t *testing.T) {
	// Cancellation is not universal: the diagram permits it only where a
	// human or an agent is plausibly still holding the work.
	cases := []struct {
		name  string
		setup []step
	}{
		{
			name:  "before analysis has started",
			setup: nil,
		},
		{
			name: "while a human is deciding",
			setup: []step{
				{orchestration.ReasonAnalysisStarted, orchestration.StateAnalyzing},
				{orchestration.ReasonAnalysisCompleted, orchestration.StateComplexityCheck},
				{orchestration.ReasonWithinComplexityBudget, orchestration.StateSpecReady},
				{orchestration.ReasonApprovalRequested, orchestration.StateAwaitingApproval},
			},
		},
		{
			name: "while work is in flight",
			setup: []step{
				{orchestration.ReasonAnalysisStarted, orchestration.StateAnalyzing},
				{orchestration.ReasonAnalysisCompleted, orchestration.StateComplexityCheck},
				{orchestration.ReasonWithinComplexityBudget, orchestration.StateSpecReady},
				{orchestration.ReasonApprovalRequested, orchestration.StateAwaitingApproval},
				{orchestration.ReasonApprovalGranted, orchestration.StatePlanning},
				{orchestration.ReasonPlanReady, orchestration.StateExecuting},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := newRunForTest(t, "run-cancel")
			walk(t, &run, tc.setup)

			walk(t, &run, []step{{orchestration.ReasonCancelled, orchestration.StateCancelled}})

			if !run.IsTerminal() {
				t.Error("a cancelled run reports IsTerminal() = false, want true")
			}
		})
	}
}

func TestARejectedTransitionLeavesTheRunUntouched(t *testing.T) {
	// Acceptance criterion 2. A refused move must not be half-applied: the
	// caller retains a run it can still reason about, and retrying with a
	// legal reason still works.
	run := newRunForTest(t, "run-rejected")
	walk(t, &run, []step{
		{orchestration.ReasonAnalysisStarted, orchestration.StateAnalyzing},
		{orchestration.ReasonAnalysisCompleted, orchestration.StateComplexityCheck},
	})

	rejected := []struct {
		name   string
		reason orchestration.Reason
		want   error
	}{
		{name: "an edge the diagram does not draw", reason: orchestration.ReasonShipped, want: orchestration.ErrTransitionNotAllowed},
		{name: "the zero reason", reason: orchestration.ReasonUnknown, want: orchestration.ErrUnknownReason},
		{name: "a reason past the last constant", reason: orchestration.Reason(200), want: orchestration.ErrUnknownReason},
	}

	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			err := run.Transition(tc.reason)
			if err == nil {
				t.Fatalf("Transition(%v) returned no error, want one", tc.reason)
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("error = %v, want it to wrap %v", err, tc.want)
			}

			var transitionErr *orchestration.TransitionError
			if !errors.As(err, &transitionErr) {
				t.Fatalf("error %v is not a *TransitionError", err)
			}
			if transitionErr.From != orchestration.StateComplexityCheck {
				t.Errorf("TransitionError.From = %v, want StateComplexityCheck", transitionErr.From)
			}

			if run.State() != orchestration.StateComplexityCheck {
				t.Fatalf("State() = %v after a rejected transition, want StateComplexityCheck (unchanged)", run.State())
			}
		})
	}

	// The run is still usable after every rejection above.
	walk(t, &run, []step{{orchestration.ReasonWithinComplexityBudget, orchestration.StateSpecReady}})
}

func TestATerminalRunRefusesEveryFurtherTransition(t *testing.T) {
	// There is no recovery edge out of a terminal state, so the only honest
	// answer is a typed refusal that names the terminal condition.
	run := newRunForTest(t, "run-terminal")
	walk(t, &run, []step{{orchestration.ReasonCancelled, orchestration.StateCancelled}})

	for _, reason := range orchestration.Reasons() {
		t.Run(reason.String(), func(t *testing.T) {
			err := run.Transition(reason)
			if err == nil {
				t.Fatalf("Transition(%v) on a cancelled run returned no error, want one", reason)
			}
			if !errors.Is(err, orchestration.ErrTerminalState) {
				t.Errorf("error = %v, want it to wrap ErrTerminalState", err)
			}
			if run.State() != orchestration.StateCancelled {
				t.Errorf("State() = %v, want StateCancelled (unchanged)", run.State())
			}
		})
	}
}

func TestTheZeroRunFailsClosed(t *testing.T) {
	// A Run that never went through NewRun -- a struct field nobody set, an
	// element of a freshly-made slice -- holds StateUnknown. It must be
	// refused rather than treated as a run sitting in Received, because
	// mistaking one for the other would silently start work nobody asked for.
	var run orchestration.Run

	if run.State() != orchestration.StateUnknown {
		t.Errorf("the zero Run's State() = %v, want StateUnknown", run.State())
	}
	if run.IsTerminal() {
		t.Error("the zero Run reports IsTerminal() = true, want false")
	}

	err := run.Transition(orchestration.ReasonAnalysisStarted)
	if err == nil {
		t.Fatal("Transition on the zero Run returned no error, want one")
	}
	if !errors.Is(err, orchestration.ErrUnknownState) {
		t.Errorf("error = %v, want it to wrap ErrUnknownState", err)
	}
	if run.State() != orchestration.StateUnknown {
		t.Errorf("State() = %v after the refusal, want StateUnknown (unchanged)", run.State())
	}
}

func TestARunCopyDoesNotShareStateWithTheOriginal(t *testing.T) {
	// Run is a value type, so copying one and advancing the copy must not
	// move the original. This is what keeps a caller from mutating a run it
	// only received for reading.
	run := newRunForTest(t, "run-copy")
	snapshot := run

	if err := run.Transition(orchestration.ReasonAnalysisStarted); err != nil {
		t.Fatalf("Transition(ReasonAnalysisStarted) returned error: %v", err)
	}

	if snapshot.State() != orchestration.StateReceived {
		t.Errorf("the copy's State() = %v, want StateReceived", snapshot.State())
	}
	if run.State() != orchestration.StateAnalyzing {
		t.Errorf("State() = %v, want StateAnalyzing", run.State())
	}
}

// step is one move in a journey through the diagram: the event, and the state
// the run must be in afterwards.
type step struct {
	reason orchestration.Reason
	want   orchestration.State
}

// walk applies each step in order, failing at the first one that does not land
// where the diagram says it should.
func walk(t *testing.T, run *orchestration.Run, steps []step) {
	t.Helper()

	for i, s := range steps {
		if err := run.Transition(s.reason); err != nil {
			t.Fatalf("step %d: Transition(%v) returned error: %v", i, s.reason, err)
		}
		if got := run.State(); got != s.want {
			t.Fatalf("step %d: State() = %v after %v, want %v", i, got, s.reason, s.want)
		}
	}
}

// newRunForTest builds a Run from a known-good identifier, failing the test if
// the constructors reject it.
func newRunForTest(t *testing.T, id string) orchestration.Run {
	t.Helper()

	runID, err := orchestration.NewRunID(id)
	if err != nil {
		t.Fatalf("NewRunID(%q) returned error: %v", id, err)
	}
	run, err := orchestration.NewRun(runID)
	if err != nil {
		t.Fatalf("NewRun(%q) returned error: %v", runID, err)
	}
	return run
}
