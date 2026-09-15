package orchestration_test

// Reason is the event half of the (from, reason) key the transition table is
// indexed by, so it needs the same closed-set guarantees as State: a zero
// value that fails closed, no meaning for out-of-range values, and a parser
// that accepts only exact spellings.

import (
	"errors"
	"testing"

	"github.com/getsyntegrity/atomwright/internal/domain/orchestration"
)

func TestZeroReasonIsUnknownAndInvalid(t *testing.T) {
	var zero orchestration.Reason

	if zero != orchestration.ReasonUnknown {
		t.Fatalf("zero Reason = %v, want ReasonUnknown", zero)
	}
	if zero.IsValid() {
		t.Error("zero Reason reports IsValid() = true, want false")
	}
	if got := zero.String(); got != "Unknown" {
		t.Errorf("zero Reason String() = %q, want %q", got, "Unknown")
	}
}

func TestReasonStringSpellsEveryDeclaredReason(t *testing.T) {
	cases := []struct {
		reason orchestration.Reason
		want   string
	}{
		{orchestration.ReasonAnalysisStarted, "AnalysisStarted"},
		{orchestration.ReasonAnalysisCompleted, "AnalysisCompleted"},
		{orchestration.ReasonWithinComplexityBudget, "WithinComplexityBudget"},
		{orchestration.ReasonOverComplexityBudget, "OverComplexityBudget"},
		{orchestration.ReasonDecompositionCompleted, "DecompositionCompleted"},
		{orchestration.ReasonApprovalRequested, "ApprovalRequested"},
		{orchestration.ReasonApprovalGranted, "ApprovalGranted"},
		{orchestration.ReasonPlanReady, "PlanReady"},
		{orchestration.ReasonExecutionCompleted, "ExecutionCompleted"},
		{orchestration.ReasonVerificationPassed, "VerificationPassed"},
		{orchestration.ReasonVerificationFailed, "VerificationFailed"},
		{orchestration.ReasonReviewApproved, "ReviewApproved"},
		{orchestration.ReasonShipped, "Shipped"},
		{orchestration.ReasonCancelled, "Cancelled"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.reason.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}

	if len(cases) != len(orchestration.Reasons()) {
		t.Fatalf("this test spells %d reasons, Reasons() reports %d -- a reason was added without a spelling case", len(cases), len(orchestration.Reasons()))
	}
}

func TestOutOfRangeReasonRendersUnknownWithoutPanicking(t *testing.T) {
	out := orchestration.Reason(200)

	if got := out.String(); got != "Unknown" {
		t.Errorf("Reason(200).String() = %q, want %q", got, "Unknown")
	}
	if out.IsValid() {
		t.Error("Reason(200).IsValid() = true, want false")
	}
}

func TestEveryDeclaredReasonIsValid(t *testing.T) {
	for _, reason := range orchestration.Reasons() {
		t.Run(reason.String(), func(t *testing.T) {
			if !reason.IsValid() {
				t.Error("IsValid() = false, want true")
			}
		})
	}
}

func TestParseReasonRoundTripsEveryReasonSpelling(t *testing.T) {
	for _, want := range orchestration.Reasons() {
		t.Run(want.String(), func(t *testing.T) {
			got, err := orchestration.ParseReason(want.String())
			if err != nil {
				t.Fatalf("ParseReason(%q) returned error: %v", want.String(), err)
			}
			if got != want {
				t.Errorf("ParseReason(%q) = %v, want %v", want.String(), got, want)
			}
		})
	}
}

func TestParseReasonRejectsAnythingButAnExactSpelling(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "the unknown sentinel is not parseable", input: "Unknown"},
		{name: "lowercase spelling", input: "shipped"},
		{name: "uppercase spelling", input: "SHIPPED"},
		{name: "surrounding whitespace", input: " Shipped "},
		{name: "snake case spelling", input: "approval_granted"},
		{name: "prefix of a valid spelling", input: "Approval"},
		{name: "unrelated garbage", input: "not-a-reason"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := orchestration.ParseReason(tc.input)
			if err == nil {
				t.Fatalf("ParseReason(%q) = %v, want an error", tc.input, got)
			}
			if !errors.Is(err, orchestration.ErrUnknownReason) {
				t.Errorf("ParseReason(%q) error = %v, want it to wrap ErrUnknownReason", tc.input, err)
			}
			if got != orchestration.ReasonUnknown {
				t.Errorf("ParseReason(%q) = %v, want ReasonUnknown alongside the error", tc.input, got)
			}
		})
	}
}

func TestReasonsReturnsAFreshSliceEachCall(t *testing.T) {
	first := orchestration.Reasons()
	if len(first) == 0 {
		t.Fatal("Reasons() returned nothing")
	}
	first[0] = orchestration.ReasonUnknown

	second := orchestration.Reasons()
	if second[0] == orchestration.ReasonUnknown {
		t.Error("mutating the result of Reasons() changed a later call; the slice is shared package state")
	}
	if second[0] != orchestration.ReasonAnalysisStarted {
		t.Errorf("Reasons()[0] = %v, want ReasonAnalysisStarted (declaration order)", second[0])
	}
}
