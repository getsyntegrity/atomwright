package specification

import (
	"errors"
	"strings"
)

// The reasons a field can be rejected. The set is deliberately small: a
// sentinel exists only when an author would act differently on it.
var (
	// ErrMissing: a required field was never supplied.
	ErrMissing = errors.New("required field is missing")
	// ErrBlank: supplied, but with no content or padded with whitespace --
	// a different authoring mistake from ErrMissing, so a different sentinel.
	ErrBlank = errors.New("field is blank or surrounded by whitespace")
	// ErrDuplicate: a repeated entry in a set-like field.
	ErrDuplicate = errors.New("entry is duplicated")
	// ErrSelfReference: a spec names itself -- a cycle of length one.
	ErrSelfReference = errors.New("spec refers to itself")
	// ErrNotSelfContained: a criterion only meaningful relative to another.
	ErrNotSelfContained = errors.New("acceptance criterion is not self-contained")
)

// ValidationError is one rejected field. Field is the path to the offending
// value ("dependsOn[1]", "diagrams[0].title"), so a reviewer goes straight to it.
type ValidationError struct {
	Field  string
	Reason error
}

func (e ValidationError) Error() string { return e.Field + ": " + e.Reason.Error() }

// Unwrap exposes the sentinel, so errors.Is works without knowing this shape.
func (e ValidationError) Unwrap() error { return e.Reason }

// ValidationErrors is every problem found in one pass. Validation never stops
// at the first failure: an author fixing a spec one rejection at a time learns
// the contract slowly and painfully.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	messages := make([]string, len(e))
	for i, problem := range e {
		messages[i] = problem.Error()
	}
	return "invalid specification: " + strings.Join(messages, "; ")
}

// Unwrap returns the problems as a multi-error, so errors.Is reaches every
// sentinel in the aggregate and not only the first.
func (e ValidationErrors) Unwrap() []error {
	unwrapped := make([]error, len(e))
	for i, problem := range e {
		unwrapped[i] = problem
	}
	return unwrapped
}
