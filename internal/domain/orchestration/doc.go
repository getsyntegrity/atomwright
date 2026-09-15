// Package orchestration is the internal/domain bounded context for #9
// ATOM-CORE. It owns the run state machine: the lifecycle a unit of
// orchestrated work moves through, and the rules about what may move it.
//
// The contract has three parts. State is the closed set of lifecycle
// positions a run can occupy, from Received through to one of the three
// terminal states Completed, Cancelled and Failed. Reason is the closed set
// of events that can occur. Next is the transition table that maps a pair of
// them -- a current state and an event -- onto exactly one next state, so the
// lifecycle is defined in one place instead of being spread across the
// callers that advance it.
//
// Everything about it fails closed. Both enums are uint8 with an invalid zero
// value, so an uninitialised state or event is rejected rather than mistaken
// for the first real one, and neither can be conjured from free-form text
// except through ParseState and ParseReason, which accept exact spellings
// only. A transition the table does not draw is refused; so is any transition
// out of a terminal state, because the lifecycle models no recovery. Run
// keeps its identity and state unexported and moves only through Transition,
// which leaves the run untouched when the move is refused.
//
// The package is deliberately narrow: it is the domain contract and nothing
// else. Persisting a run, scheduling it, timing it, recording its history and
// notifying anyone about it all belong to the layers above.
//
// It was scaffolded empty by ATOM-BOOT-002 (ADR-0001) and implemented by
// ATOM-CORE-001 (#20).
package orchestration
