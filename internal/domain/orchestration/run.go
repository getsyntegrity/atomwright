package orchestration

import (
	"fmt"
	"strings"
)

// RunID identifies one run.
//
// It is a domain type rather than a bare string so that identity cannot be
// confused with any of the other strings a run is surrounded by -- a title, a
// branch name, a provider's own opaque handle. The compiler refuses the mix-up
// that a `string` parameter would accept silently, and having a named type
// gives validation a place to live.
//
// It does not make that validation unbypassable, and nothing here pretends
// otherwise: RunID is a defined string type, so RunID("   ") is a legal
// conversion and a decoder can produce one too. NewRunID is where an
// identifier is checked, not a gate the type system enforces, which is why
// every constructor that accepts a RunID checks it again rather than assuming
// it arrived through NewRunID. Making the type opaque -- a struct with an
// unexported field -- would close that hole, at the cost of a type that no
// longer prints, compares or maps like the identifier it is; that trade is
// worth revisiting if a caller ever does smuggle a bad value past a
// constructor, but it is not worth paying for speculatively here.
type RunID string

// NewRunID validates an identifier and returns it as a RunID.
//
// An identifier that is empty, or that consists only of whitespace, carries no
// identity and is rejected with ErrEmptyRunID. An identifier that has identity
// but is padded with whitespace is rejected with ErrInvalidRunID rather than
// trimmed: silently trimming would let " run-1" and "run-1" name the same run
// while looking different in every log, and would hide the caller's mistake
// instead of reporting it. Whitespace inside the identifier is the caller's
// business and is left alone.
func NewRunID(id string) (RunID, error) {
	trimmed := strings.TrimSpace(id)
	switch {
	case trimmed == "":
		return "", ErrEmptyRunID
	case trimmed != id:
		return "", fmt.Errorf("%w: %q", ErrInvalidRunID, id)
	}
	return RunID(id), nil
}

// String returns the underlying identifier.
func (r RunID) String() string { return string(r) }

// Run is a single unit of orchestrated work and the aggregate that owns its
// lifecycle position.
//
// Both fields are unexported on purpose: a Run obtained from NewRun is always
// in a legal state, and the only way to move it is Transition, which consults
// the transition table. A caller outside this package therefore cannot put a
// run into a state the diagram does not allow -- not by assignment, not by
// constructing a literal, and not by ignoring an error.
//
// Run is a value type. Copying one snapshots its state; advancing the copy
// leaves the original where it was.
type Run struct {
	id    RunID
	state State
}

// NewRun starts a run at StateReceived, the diagram's entry point.
//
// It revalidates the identifier through NewRunID rather than trusting it.
// RunID is a defined string type, so a caller can produce one by conversion,
// or decode one, without ever going through NewRunID -- and the zero value is
// what they hold if they ignored its error. Checking only for "" here would
// let RunID("   ") and RunID(" run-1") open runs whose identifiers NewRunID
// explicitly refuses, which is exactly the ambiguity the validation exists to
// prevent. Revalidating costs a string comparison and keeps one definition of
// a usable identifier instead of two that can drift.
func NewRun(id RunID) (Run, error) {
	validated, err := NewRunID(string(id))
	if err != nil {
		return Run{}, err
	}
	return Run{id: validated, state: StateReceived}, nil
}

// ID returns the run's identifier.
func (r Run) ID() RunID { return r.id }

// State returns the run's current lifecycle position.
func (r Run) State() State { return r.state }

// IsTerminal reports whether the run has finished and will accept no further
// transition.
func (r Run) IsTerminal() bool { return r.state.IsTerminal() }

// Transition advances the run by applying an event.
//
// The decision belongs to Next, so there is exactly one definition of what is
// legal. On failure the run is left exactly as it was -- the new state is
// computed before anything is assigned, so a refused move cannot half-apply
// and leave the caller holding a run whose state nobody chose. The returned
// error is a *TransitionError naming the attempted move.
func (r *Run) Transition(reason Reason) error {
	next, err := Next(r.state, reason)
	if err != nil {
		return err
	}
	r.state = next
	return nil
}
