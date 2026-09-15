package orchestration

import (
	"errors"
	"fmt"
	"strings"
)

// The package's sentinel errors.
//
// They are sentinels rather than distinct error types because callers need to
// classify a failure, not inspect it: "was this refused because the run is
// finished, or because the move was never legal?" is answered with errors.Is
// against one of these, and the concrete error carrying it can keep changing
// shape without breaking that check.
var (
	// ErrUnknownState reports a State that is not one of the declared
	// lifecycle states -- the zero value, or a value conjured by conversion.
	ErrUnknownState = errors.New("orchestration: unknown run state")
	// ErrUnknownReason reports a Reason that is not one of the declared
	// events.
	ErrUnknownReason = errors.New("orchestration: unknown transition reason")
	// ErrTerminalState reports an attempt to move a run that has already
	// finished. It is distinct from ErrTransitionNotAllowed because the two
	// mean different things to a caller: a terminal run will never accept any
	// event, whereas a disallowed move may simply be the wrong event for a
	// state that is still live.
	ErrTerminalState = errors.New("orchestration: run is in a terminal state")
	// ErrTransitionNotAllowed reports a move the transition table does not
	// draw from a state that is otherwise live.
	ErrTransitionNotAllowed = errors.New("orchestration: transition not allowed")
	// ErrEmptyRunID reports a run identifier that carries no identity at all.
	ErrEmptyRunID = errors.New("orchestration: run id is empty")
	// ErrInvalidRunID reports an identifier that has identity but is not in
	// its canonical form. Accepting " run-1" and "run-1" as two different
	// runs is a bug waiting to happen, so the padded form is rejected at the
	// boundary instead of being silently trimmed -- trimming would hide a
	// caller's formatting mistake rather than surface it.
	ErrInvalidRunID = errors.New("orchestration: run id has surrounding whitespace")
)

// TransitionError is the typed error every rejected transition returns. It
// names the move that was refused -- a caller logging the failure should not
// have to reconstruct which state and which event were involved -- and wraps
// the sentinel that says why.
type TransitionError struct {
	// From is the state the run was in when the move was attempted.
	From State
	// Reason is the event that was offered.
	Reason Reason
	// Err is the sentinel explaining the refusal.
	Err error
}

// Error renders the refused move and its cause. The sentinel's own text
// already carries the package prefix, so only its explanatory tail is appended
// here; the result reads as one sentence rather than two stuttering ones.
func (e *TransitionError) Error() string {
	return fmt.Sprintf("orchestration: cannot transition from %s on %s: %s", e.From, e.Reason, transitionCause(e.Err))
}

// Unwrap exposes the sentinel so errors.Is works against ErrUnknownState,
// ErrUnknownReason, ErrTerminalState and ErrTransitionNotAllowed.
func (e *TransitionError) Unwrap() error { return e.Err }

// transitionCause strips the shared "orchestration: " prefix from a sentinel
// so that Error does not repeat it. An error from outside this package is
// rendered verbatim.
func transitionCause(err error) string {
	const prefix = "orchestration: "

	if err == nil {
		return "unspecified"
	}
	return strings.TrimPrefix(err.Error(), prefix)
}
