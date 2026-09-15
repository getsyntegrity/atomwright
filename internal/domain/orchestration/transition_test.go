package orchestration_test

// The edge list below is transcribed independently from the state diagram in
// issue #20 (ATOM-CORE-001). It is deliberately NOT read from the
// implementation's table: a test that asks the table what the table says
// proves nothing. Reviewing this change means diffing these sixteen lines
// against the diagram in the issue, and diffing the implementation's table
// against the same diagram. If the two transcriptions disagree, these tests
// fail.

import (
	"errors"
	"strings"
	"testing"

	"github.com/getsyntegrity/atomwright/internal/domain/orchestration"
)

// diagramEdge is one arrow in the issue's state diagram: a source state, the
// event that fires, and the resulting state.
type diagramEdge struct {
	from   orchestration.State
	reason orchestration.Reason
	to     orchestration.State
}

// diagramEdges is the complete set of arrows. Every pair of (state, reason)
// that does not appear here must be rejected -- that is the fail-closed
// contract, and TestNextRejectsEveryEdgeTheDiagramDoesNotDraw checks it over
// the full cross product.
var diagramEdges = []diagramEdge{
	{orchestration.StateReceived, orchestration.ReasonAnalysisStarted, orchestration.StateAnalyzing},
	{orchestration.StateAnalyzing, orchestration.ReasonAnalysisCompleted, orchestration.StateComplexityCheck},
	{orchestration.StateComplexityCheck, orchestration.ReasonWithinComplexityBudget, orchestration.StateSpecReady},
	{orchestration.StateComplexityCheck, orchestration.ReasonOverComplexityBudget, orchestration.StateDecomposing},
	{orchestration.StateDecomposing, orchestration.ReasonDecompositionCompleted, orchestration.StateComplexityCheck},
	{orchestration.StateSpecReady, orchestration.ReasonApprovalRequested, orchestration.StateAwaitingApproval},
	{orchestration.StateAwaitingApproval, orchestration.ReasonApprovalGranted, orchestration.StatePlanning},
	{orchestration.StatePlanning, orchestration.ReasonPlanReady, orchestration.StateExecuting},
	{orchestration.StateExecuting, orchestration.ReasonExecutionCompleted, orchestration.StateVerifying},
	{orchestration.StateVerifying, orchestration.ReasonVerificationPassed, orchestration.StateAwaitingReview},
	{orchestration.StateVerifying, orchestration.ReasonVerificationFailed, orchestration.StateFailed},
	{orchestration.StateAwaitingReview, orchestration.ReasonReviewApproved, orchestration.StateReadyToShip},
	{orchestration.StateReadyToShip, orchestration.ReasonShipped, orchestration.StateCompleted},
	{orchestration.StateReceived, orchestration.ReasonCancelled, orchestration.StateCancelled},
	{orchestration.StateAwaitingApproval, orchestration.ReasonCancelled, orchestration.StateCancelled},
	{orchestration.StateExecuting, orchestration.ReasonCancelled, orchestration.StateCancelled},
}

// lookupDiagramEdge answers whether the diagram draws an arrow for this pair,
// and where it leads.
func lookupDiagramEdge(from orchestration.State, reason orchestration.Reason) (orchestration.State, bool) {
	for _, edge := range diagramEdges {
		if edge.from == from && edge.reason == reason {
			return edge.to, true
		}
	}
	return orchestration.StateUnknown, false
}

func TestNextFollowsEveryEdgeTheDiagramDraws(t *testing.T) {
	for _, edge := range diagramEdges {
		t.Run(edge.from.String()+" on "+edge.reason.String()+" reaches "+edge.to.String(), func(t *testing.T) {
			got, err := orchestration.Next(edge.from, edge.reason)
			if err != nil {
				t.Fatalf("Next(%v, %v) returned error: %v", edge.from, edge.reason, err)
			}
			if got != edge.to {
				t.Errorf("Next(%v, %v) = %v, want %v", edge.from, edge.reason, got, edge.to)
			}
		})
	}

	if len(diagramEdges) != 16 {
		t.Fatalf("the diagram has 16 edges, this transcription has %d", len(diagramEdges))
	}
}

func TestNextRejectsEveryEdgeTheDiagramDoesNotDraw(t *testing.T) {
	// The exhaustive half of acceptance criterion 4. Walking the full cross
	// product is what makes "no undrawn transition is reachable" a proof
	// rather than a claim: an accidental extra entry in the implementation's
	// table shows up here even if nobody thought to write a case for it.
	for _, from := range orchestration.States() {
		for _, reason := range orchestration.Reasons() {
			name := from.String() + " on " + reason.String()
			t.Run(name, func(t *testing.T) {
				want, drawn := lookupDiagramEdge(from, reason)

				got, err := orchestration.Next(from, reason)
				if drawn {
					if err != nil {
						t.Fatalf("Next(%v, %v) returned error: %v", from, reason, err)
					}
					if got != want {
						t.Errorf("Next(%v, %v) = %v, want %v", from, reason, got, want)
					}
					return
				}

				if err == nil {
					t.Fatalf("Next(%v, %v) = %v with no error, but the diagram draws no such edge", from, reason, got)
				}
				if got != orchestration.StateUnknown {
					t.Errorf("Next(%v, %v) = %v alongside an error, want StateUnknown", from, reason, got)
				}

				wantSentinel := orchestration.ErrTransitionNotAllowed
				if from.IsTerminal() {
					wantSentinel = orchestration.ErrTerminalState
				}
				if !errors.Is(err, wantSentinel) {
					t.Errorf("Next(%v, %v) error = %v, want it to wrap %v", from, reason, err, wantSentinel)
				}
			})
		}
	}
}

func TestNextReportsAnInvalidSourceState(t *testing.T) {
	cases := []struct {
		name string
		from orchestration.State
	}{
		{name: "the zero value", from: orchestration.StateUnknown},
		{name: "a value past the last constant", from: orchestration.State(200)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := orchestration.Next(tc.from, orchestration.ReasonAnalysisStarted)
			if err == nil {
				t.Fatalf("Next(%v, ReasonAnalysisStarted) = %v, want an error", tc.from, got)
			}
			if !errors.Is(err, orchestration.ErrUnknownState) {
				t.Errorf("error = %v, want it to wrap ErrUnknownState", err)
			}
			if got != orchestration.StateUnknown {
				t.Errorf("state = %v, want StateUnknown", got)
			}
			assertTransitionErrorCarries(t, err, tc.from, orchestration.ReasonAnalysisStarted)
		})
	}
}

func TestNextReportsAnInvalidReason(t *testing.T) {
	cases := []struct {
		name   string
		reason orchestration.Reason
	}{
		{name: "the zero value", reason: orchestration.ReasonUnknown},
		{name: "a value past the last constant", reason: orchestration.Reason(200)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := orchestration.Next(orchestration.StateReceived, tc.reason)
			if err == nil {
				t.Fatalf("Next(StateReceived, %v) = %v, want an error", tc.reason, got)
			}
			if !errors.Is(err, orchestration.ErrUnknownReason) {
				t.Errorf("error = %v, want it to wrap ErrUnknownReason", err)
			}
			if got != orchestration.StateUnknown {
				t.Errorf("state = %v, want StateUnknown", got)
			}
			assertTransitionErrorCarries(t, err, orchestration.StateReceived, tc.reason)
		})
	}
}

func TestNextChecksItsPreconditionsInTheDocumentedOrder(t *testing.T) {
	// When more than one precondition is broken, the caller should be told
	// about the most fundamental one. An unusable state value is a worse
	// problem than an unusable reason, and both are worse than a run that is
	// merely finished.
	cases := []struct {
		name   string
		from   orchestration.State
		reason orchestration.Reason
		want   error
	}{
		{
			name:   "an invalid state outranks an invalid reason",
			from:   orchestration.StateUnknown,
			reason: orchestration.ReasonUnknown,
			want:   orchestration.ErrUnknownState,
		},
		{
			name:   "an out-of-range state outranks a reason that would otherwise be accepted",
			from:   orchestration.State(200),
			reason: orchestration.ReasonShipped,
			want:   orchestration.ErrUnknownState,
		},
		{
			name:   "an invalid reason outranks a terminal state",
			from:   orchestration.StateCompleted,
			reason: orchestration.ReasonUnknown,
			want:   orchestration.ErrUnknownReason,
		},
		{
			name:   "a terminal state outranks a merely undrawn edge",
			from:   orchestration.StateCompleted,
			reason: orchestration.ReasonAnalysisStarted,
			want:   orchestration.ErrTerminalState,
		},
		{
			name:   "an undrawn edge from a live state is the last resort",
			from:   orchestration.StateReceived,
			reason: orchestration.ReasonShipped,
			want:   orchestration.ErrTransitionNotAllowed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := orchestration.Next(tc.from, tc.reason)
			if err == nil {
				t.Fatal("Next returned no error, want one")
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("error = %v, want it to wrap %v", err, tc.want)
			}
		})
	}
}

func TestTransitionErrorCarriesTheAttemptedMove(t *testing.T) {
	// The error has to name the move that was refused; otherwise a caller
	// logging it learns only that something was rejected.
	_, err := orchestration.Next(orchestration.StateExecuting, orchestration.ReasonReviewApproved)
	if err == nil {
		t.Fatal("Next returned no error, want one")
	}

	assertTransitionErrorCarries(t, err, orchestration.StateExecuting, orchestration.ReasonReviewApproved)

	message := err.Error()
	for _, fragment := range []string{"Executing", "ReviewApproved", "orchestration"} {
		if !strings.Contains(message, fragment) {
			t.Errorf("error message %q does not mention %q", message, fragment)
		}
	}
}

func TestAllowedReasonsMatchesTheDiagramForEveryState(t *testing.T) {
	for _, from := range orchestration.States() {
		t.Run(from.String(), func(t *testing.T) {
			var want []orchestration.Reason
			for _, reason := range orchestration.Reasons() {
				if _, drawn := lookupDiagramEdge(from, reason); drawn {
					want = append(want, reason)
				}
			}

			got := orchestration.AllowedReasons(from)
			if len(got) != len(want) {
				t.Fatalf("AllowedReasons(%v) = %v, want %v", from, got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("AllowedReasons(%v) = %v, want %v (Reason declaration order)", from, got, want)
				}
			}
		})
	}
}

func TestTerminalStatesOfferNoWayOut(t *testing.T) {
	// Acceptance criterion 3: the diagram draws no recovery edge, so a
	// terminal run has nothing left to offer. A recovery path would be a new
	// table entry with its own reason, added deliberately.
	for _, from := range orchestration.States() {
		if !from.IsTerminal() {
			continue
		}
		t.Run(from.String(), func(t *testing.T) {
			if got := orchestration.AllowedReasons(from); len(got) != 0 {
				t.Errorf("AllowedReasons(%v) = %v, want nothing", from, got)
			}
		})
	}
}

func TestAllowedReasonsOffersNothingForAnInvalidState(t *testing.T) {
	cases := []struct {
		name string
		from orchestration.State
	}{
		{name: "the zero value", from: orchestration.StateUnknown},
		{name: "a value past the last constant", from: orchestration.State(200)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := orchestration.AllowedReasons(tc.from); len(got) != 0 {
				t.Errorf("AllowedReasons(%v) = %v, want nothing", tc.from, got)
			}
		})
	}
}

func TestAllowedReasonsReturnsAFreshSliceEachCall(t *testing.T) {
	// ComplexityCheck is the branch point, so it has more than one reason and
	// makes a good subject for a mutation attempt.
	first := orchestration.AllowedReasons(orchestration.StateComplexityCheck)
	if len(first) == 0 {
		t.Fatal("AllowedReasons(StateComplexityCheck) returned nothing")
	}
	first[0] = orchestration.ReasonUnknown

	second := orchestration.AllowedReasons(orchestration.StateComplexityCheck)
	if second[0] == orchestration.ReasonUnknown {
		t.Error("mutating the result of AllowedReasons changed a later call; the slice is shared package state")
	}
}

func TestEveryStateIsWiredIntoTheDiagram(t *testing.T) {
	// A typo in the table is most likely to show up as an orphan: a state
	// nothing leads to, or a live state nothing leads out of. Neither is
	// something the diagram models, so both are bugs.
	reached := map[orchestration.State]bool{}
	departs := map[orchestration.State]bool{}
	for _, edge := range diagramEdges {
		reached[edge.to] = true
		departs[edge.from] = true
	}

	for _, state := range orchestration.States() {
		t.Run(state.String(), func(t *testing.T) {
			if state != orchestration.StateReceived && !reached[state] {
				t.Errorf("%v is the target of no edge; it is unreachable", state)
			}
			if !state.IsTerminal() && !departs[state] {
				t.Errorf("%v is not terminal but has no outgoing edge; a run would be stuck there", state)
			}
			if state.IsTerminal() && departs[state] {
				t.Errorf("%v is terminal but has an outgoing edge", state)
			}
		})
	}

	if !departs[orchestration.StateReceived] {
		t.Error("StateReceived has no outgoing edge; a run could never start")
	}
}

// assertTransitionErrorCarries checks that the error is the package's typed
// transition error and that it names the move the caller attempted.
func assertTransitionErrorCarries(t *testing.T, err error, from orchestration.State, reason orchestration.Reason) {
	t.Helper()

	var transitionErr *orchestration.TransitionError
	if !errors.As(err, &transitionErr) {
		t.Fatalf("error %v is not a *TransitionError", err)
	}
	if transitionErr.From != from {
		t.Errorf("TransitionError.From = %v, want %v", transitionErr.From, from)
	}
	if transitionErr.Reason != reason {
		t.Errorf("TransitionError.Reason = %v, want %v", transitionErr.Reason, reason)
	}
	if transitionErr.Unwrap() == nil {
		t.Error("TransitionError.Unwrap() = nil, want the wrapped sentinel")
	}
}
