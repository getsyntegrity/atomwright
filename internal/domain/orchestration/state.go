package orchestration

import "fmt"

// State is the lifecycle position of a single run.
//
// It is a closed enum over uint8 rather than a named string type on purpose.
// A string type would let any caller conjure a state out of free-form text --
// a typo, a value read from a config file, a field decoded from an untrusted
// payload -- and that value would look like a legitimate State to the type
// system while meaning nothing to the transition table. With uint8 the only
// way into the set from outside the package is ParseState, which accepts an
// exact spelling and nothing else. Everything else fails closed.
//
// The zero value is StateUnknown and is deliberately invalid, so a State that
// was never initialised -- a struct field nobody set, an element of a
// freshly-made slice -- is rejected rather than silently treated as the first
// real state.
type State uint8

// The declared states, in lifecycle order. StateUnknown occupies iota 0 so
// that the zero value is the invalid one.
//
// Adding a state here is not enough to make it reachable: it also needs at
// least one entry in the transition table, and stateNames needs its spelling.
const (
	// StateUnknown is the zero value and is never a legal run state.
	StateUnknown State = iota
	// StateReceived is the entry point: a request has arrived and nothing
	// has been decided about it yet.
	StateReceived
	// StateAnalyzing means the request is being understood.
	StateAnalyzing
	// StateComplexityCheck means the analysed work is being measured against
	// the complexity budget.
	StateComplexityCheck
	// StateSpecReady means the work fits the budget and has a specification.
	StateSpecReady
	// StateDecomposing means the work exceeded the budget and is being split.
	StateDecomposing
	// StateAwaitingApproval means a human decision is outstanding.
	StateAwaitingApproval
	// StatePlanning means the approved work is being turned into a plan.
	StatePlanning
	// StateExecuting means the plan is being carried out.
	StateExecuting
	// StateVerifying means the executed work is being checked.
	StateVerifying
	// StateAwaitingReview means verification passed and review is outstanding.
	StateAwaitingReview
	// StateReadyToShip means the work is reviewed and waiting to be delivered.
	StateReadyToShip
	// StateCompleted is the terminal state of a run that shipped.
	StateCompleted
	// StateCancelled is the terminal state of a run that was called off.
	StateCancelled
	// StateFailed is the terminal state of a run whose verification rejected
	// the work.
	StateFailed
)

// stateNames holds the canonical spelling of each state, indexed by the state
// itself. Index 0 is the spelling for StateUnknown, which is also the answer
// for any out-of-range value: rendering a State must never panic, because the
// first place a corrupt value tends to surface is a log line or an error
// message, and neither is a good place to lose the process.
var stateNames = [...]string{
	StateUnknown:          "Unknown",
	StateReceived:         "Received",
	StateAnalyzing:        "Analyzing",
	StateComplexityCheck:  "ComplexityCheck",
	StateSpecReady:        "SpecReady",
	StateDecomposing:      "Decomposing",
	StateAwaitingApproval: "AwaitingApproval",
	StatePlanning:         "Planning",
	StateExecuting:        "Executing",
	StateVerifying:        "Verifying",
	StateAwaitingReview:   "AwaitingReview",
	StateReadyToShip:      "ReadyToShip",
	StateCompleted:        "Completed",
	StateCancelled:        "Cancelled",
	StateFailed:           "Failed",
}

// String returns the canonical spelling of the state. StateUnknown and any
// value outside the declared set render as "Unknown" rather than panicking.
func (s State) String() string {
	if int(s) >= len(stateNames) {
		return stateNames[StateUnknown]
	}
	return stateNames[s]
}

// IsValid reports whether the state is one of the declared lifecycle states.
// StateUnknown and any out-of-range value are not.
func (s State) IsValid() bool {
	return s > StateUnknown && int(s) < len(stateNames)
}

// IsTerminal reports whether the run has finished and can move no further.
// Only StateCompleted, StateCancelled and StateFailed are terminal, and a
// terminal state is by construction also valid.
func (s State) IsTerminal() bool {
	switch s {
	case StateCompleted, StateCancelled, StateFailed:
		return true
	default:
		return false
	}
}

// States returns every valid state in declaration order.
//
// The slice is allocated per call: callers routinely range over it, and one
// that writes to it must not be able to corrupt the enum for every other
// caller in the process.
func States() []State {
	all := make([]State, 0, len(stateNames)-1)
	for s := StateReceived; int(s) < len(stateNames); s++ {
		all = append(all, s)
	}
	return all
}

// ParseState converts a canonical spelling back into a State.
//
// This is the boundary constructor: it is how a state crosses into the domain
// from a string, and it is what makes "an unknown state fails closed"
// enforceable. The match is exact -- differing case, surrounding whitespace,
// the empty string, and the spelling "Unknown" itself are all rejected. On
// failure it returns StateUnknown and an error wrapping ErrUnknownState, so
// callers can both branch on errors.Is and safely ignore the value.
func ParseState(s string) (State, error) {
	for candidate := StateReceived; int(candidate) < len(stateNames); candidate++ {
		if stateNames[candidate] == s {
			return candidate, nil
		}
	}
	return StateUnknown, fmt.Errorf("%w: %q", ErrUnknownState, s)
}
