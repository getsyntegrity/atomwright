package orchestration

import "fmt"

// Reason is the event that triggers a transition.
//
// The transition table is keyed by the pair (from, reason), so a state plus a
// reason determines the next state uniquely and deterministically: there is no
// ambiguity to resolve at the call site and no policy hiding behind a
// condition. Where the lifecycle branches -- the complexity check -- the
// branch is expressed as two different events, not as a decision the state
// machine makes.
//
// Reason is a closed uint8 enum for exactly the reasons State is: an event
// arriving from outside cannot be invented from free text, only parsed with
// ParseReason, and the zero value is invalid so an unset Reason fails closed.
type Reason uint8

// The declared reasons. ReasonUnknown occupies iota 0 so that the zero value
// is the invalid one.
const (
	// ReasonUnknown is the zero value and never triggers a transition.
	ReasonUnknown Reason = iota
	// ReasonAnalysisStarted reports that work on understanding the request
	// has begun.
	ReasonAnalysisStarted
	// ReasonAnalysisCompleted reports that the request is understood.
	ReasonAnalysisCompleted
	// ReasonWithinComplexityBudget reports that the work is small enough to
	// specify directly.
	ReasonWithinComplexityBudget
	// ReasonOverComplexityBudget reports that the work must be split first.
	ReasonOverComplexityBudget
	// ReasonDecompositionCompleted reports that the split is done and the
	// result needs measuring again.
	ReasonDecompositionCompleted
	// ReasonApprovalRequested reports that a human decision has been asked
	// for.
	ReasonApprovalRequested
	// ReasonApprovalGranted reports that the human said yes.
	ReasonApprovalGranted
	// ReasonPlanReady reports that the approved work has an executable plan.
	ReasonPlanReady
	// ReasonExecutionCompleted reports that the plan has been carried out.
	ReasonExecutionCompleted
	// ReasonVerificationPassed reports that the executed work checks out.
	ReasonVerificationPassed
	// ReasonVerificationFailed reports that the executed work does not.
	ReasonVerificationFailed
	// ReasonReviewApproved reports that review accepted the work.
	ReasonReviewApproved
	// ReasonShipped reports that the work was delivered.
	ReasonShipped
	// ReasonCancelled reports that the run was called off.
	ReasonCancelled
)

// reasonNames holds the canonical spelling of each reason, indexed by the
// reason itself. Index 0 doubles as the answer for out-of-range values so
// that rendering never panics.
var reasonNames = [...]string{
	ReasonUnknown:                "Unknown",
	ReasonAnalysisStarted:        "AnalysisStarted",
	ReasonAnalysisCompleted:      "AnalysisCompleted",
	ReasonWithinComplexityBudget: "WithinComplexityBudget",
	ReasonOverComplexityBudget:   "OverComplexityBudget",
	ReasonDecompositionCompleted: "DecompositionCompleted",
	ReasonApprovalRequested:      "ApprovalRequested",
	ReasonApprovalGranted:        "ApprovalGranted",
	ReasonPlanReady:              "PlanReady",
	ReasonExecutionCompleted:     "ExecutionCompleted",
	ReasonVerificationPassed:     "VerificationPassed",
	ReasonVerificationFailed:     "VerificationFailed",
	ReasonReviewApproved:         "ReviewApproved",
	ReasonShipped:                "Shipped",
	ReasonCancelled:              "Cancelled",
}

// String returns the canonical spelling of the reason. ReasonUnknown and any
// value outside the declared set render as "Unknown" rather than panicking.
func (r Reason) String() string {
	if int(r) >= len(reasonNames) {
		return reasonNames[ReasonUnknown]
	}
	return reasonNames[r]
}

// IsValid reports whether the reason is one of the declared events.
// ReasonUnknown and any out-of-range value are not.
func (r Reason) IsValid() bool {
	return r > ReasonUnknown && int(r) < len(reasonNames)
}

// Reasons returns every valid reason in declaration order.
//
// The slice is allocated per call so that a caller mutating it cannot corrupt
// the enum for anyone else.
func Reasons() []Reason {
	all := make([]Reason, 0, len(reasonNames)-1)
	for r := ReasonAnalysisStarted; int(r) < len(reasonNames); r++ {
		all = append(all, r)
	}
	return all
}

// ParseReason converts a canonical spelling back into a Reason.
//
// Like ParseState, the match is exact and everything else -- including the
// empty string and the spelling "Unknown" -- returns ReasonUnknown and an
// error wrapping ErrUnknownReason.
func ParseReason(s string) (Reason, error) {
	for candidate := ReasonAnalysisStarted; int(candidate) < len(reasonNames); candidate++ {
		if reasonNames[candidate] == s {
			return candidate, nil
		}
	}
	return ReasonUnknown, fmt.Errorf("%w: %q", ErrUnknownReason, s)
}
