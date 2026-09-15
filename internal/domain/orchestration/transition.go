package orchestration

// The transition table: the single place the run lifecycle is defined.
//
// It is transcribed from the state diagram in issue #20 (ATOM-CORE-001),
// reproduced here so a reviewer can diff the code against the issue without
// leaving the file:
//
//	stateDiagram-v2
//	    [*]              --> Received
//	    Received         --> Analyzing        : AnalysisStarted
//	    Analyzing        --> ComplexityCheck  : AnalysisCompleted
//	    ComplexityCheck  --> SpecReady        : WithinComplexityBudget
//	    ComplexityCheck  --> Decomposing      : OverComplexityBudget
//	    Decomposing      --> ComplexityCheck  : DecompositionCompleted
//	    SpecReady        --> AwaitingApproval : ApprovalRequested
//	    AwaitingApproval --> Planning         : ApprovalGranted
//	    Planning         --> Executing        : PlanReady
//	    Executing        --> Verifying        : ExecutionCompleted
//	    Verifying        --> AwaitingReview   : VerificationPassed
//	    Verifying        --> Failed           : VerificationFailed
//	    AwaitingReview   --> ReadyToShip      : ReviewApproved
//	    ReadyToShip      --> Completed        : Shipped
//	    Received         --> Cancelled        : Cancelled
//	    AwaitingApproval --> Cancelled        : Cancelled
//	    Executing        --> Cancelled        : Cancelled
//	    Completed        --> [*]
//	    Cancelled        --> [*]
//	    Failed           --> [*]
//
// Sixteen edges, and nothing else is legal. A move the diagram does not draw
// is refused rather than tolerated, which is what makes the table -- not the
// call sites -- the authority on what a run may do next.
//
// Note what is deliberately absent: there is no edge out of Completed,
// Cancelled or Failed. The diagram models no recovery, and recovery is not
// something to improvise at a call site. A recovery path would be an explicit
// new entry in this table plus its own Reason constant, added by the change
// that actually needs it, with review -- not a retry loop that happens to
// mutate a finished run.

// transitions maps a source state to the events it accepts and where each one
// leads. It is an unexported package variable built from a literal, so the
// lifecycle cannot be rewritten from outside this package; the only public way
// to consult it is Next and AllowedReasons, neither of which hands out the
// underlying maps.
var transitions = map[State]map[Reason]State{
	StateReceived: {
		ReasonAnalysisStarted: StateAnalyzing,
		ReasonCancelled:       StateCancelled,
	},
	StateAnalyzing: {
		ReasonAnalysisCompleted: StateComplexityCheck,
	},
	StateComplexityCheck: {
		ReasonWithinComplexityBudget: StateSpecReady,
		ReasonOverComplexityBudget:   StateDecomposing,
	},
	StateSpecReady: {
		ReasonApprovalRequested: StateAwaitingApproval,
	},
	StateDecomposing: {
		ReasonDecompositionCompleted: StateComplexityCheck,
	},
	StateAwaitingApproval: {
		ReasonApprovalGranted: StatePlanning,
		ReasonCancelled:       StateCancelled,
	},
	StatePlanning: {
		ReasonPlanReady: StateExecuting,
	},
	StateExecuting: {
		ReasonExecutionCompleted: StateVerifying,
		ReasonCancelled:          StateCancelled,
	},
	StateVerifying: {
		ReasonVerificationPassed: StateAwaitingReview,
		ReasonVerificationFailed: StateFailed,
	},
	StateAwaitingReview: {
		ReasonReviewApproved: StateReadyToShip,
	},
	StateReadyToShip: {
		ReasonShipped: StateCompleted,
	},
}

// Next reports the state a run reaches from `from` when `reason` fires.
//
// It is fail-closed and checks its preconditions in a fixed order, most
// fundamental first, so that a caller with more than one problem is told about
// the one worth fixing first:
//
//  1. an unusable source state    -> ErrUnknownState
//  2. an unusable reason          -> ErrUnknownReason
//  3. a run that already finished -> ErrTerminalState
//  4. an edge the table does not draw -> ErrTransitionNotAllowed
//
// Every failure is a *TransitionError naming the attempted move and wrapping
// the sentinel, and every failure returns StateUnknown, so a caller that
// ignores the error still cannot end up holding a plausible-looking state.
func Next(from State, reason Reason) (State, error) {
	switch {
	case !from.IsValid():
		return StateUnknown, &TransitionError{From: from, Reason: reason, Err: ErrUnknownState}
	case !reason.IsValid():
		return StateUnknown, &TransitionError{From: from, Reason: reason, Err: ErrUnknownReason}
	case from.IsTerminal():
		return StateUnknown, &TransitionError{From: from, Reason: reason, Err: ErrTerminalState}
	}

	to, ok := transitions[from][reason]
	if !ok {
		return StateUnknown, &TransitionError{From: from, Reason: reason, Err: ErrTransitionNotAllowed}
	}
	return to, nil
}

// AllowedReasons returns the events Next accepts from the given state, in
// Reason declaration order.
//
// It returns nil for terminal and invalid states: neither has anything to
// offer, and callers that use the result to build a menu of next moves get an
// empty menu rather than a misleading one. The slice is allocated per call so
// that a caller mutating it cannot reach the table behind it.
func AllowedReasons(from State) []Reason {
	if !from.IsValid() || from.IsTerminal() {
		return nil
	}

	edges := transitions[from]
	if len(edges) == 0 {
		return nil
	}

	// Ranging over Reasons() rather than over the map keeps the result
	// deterministic; Go's map iteration order is not.
	allowed := make([]Reason, 0, len(edges))
	for _, reason := range Reasons() {
		if _, ok := edges[reason]; ok {
			allowed = append(allowed, reason)
		}
	}
	return allowed
}
