package specification

// ID identifies one spec. It stays a plain string: the contract needs
// identities it can compare, not a format grammar that would freeze an
// issue-tracker convention into the domain.
type ID string

// IsZero reports whether no identity was supplied. A zero Parent legitimately
// means "this is a root spec"; a zero Spec.ID never does.
func (id ID) IsZero() bool { return id == "" }

// AcceptanceCriterion is one independently verifiable statement about the
// finished work. A string rather than a struct: extra fields would invite
// metadata that substitutes for writing a clear sentence.
type AcceptanceCriterion string

// FailureSemantics records what happens when the behaviour fails. Statement is
// required in both directions, because claiming a behaviour cannot fail is
// itself a claim, and unreviewed claims are how silent failure modes get in.
// The zero value is therefore invalid, which makes the default fail-closed.
type FailureSemantics struct {
	CanFail bool
	// Statement describes what happens when the behaviour fails, or -- when
	// CanFail is false -- why this behaviour cannot fail.
	Statement string
}

// Diagram is an optional Mermaid illustration. Source is never parsed:
// rendering is a presentation concern, and a domain package that understood
// diagram syntax would own a second, competing definition of it.
type Diagram struct {
	Title  string
	Source string
}

// Spec is the atomic specification contract (#23 ATOM-SPEC-001).
//
// Relationships are expressed by the child naming its Parent and by a spec
// naming what it depends on. Neither inverse is stored: two places recording
// the same edge can disagree, and nothing here could say which one is right.
//
// The slices are caller-owned and nothing is copied, so ReadySpec proves a spec
// was valid when checked, not that its slices are frozen forever. Deep-copying
// costs more code than that stronger guarantee is worth here.
type Spec struct {
	ID        ID
	Parent    ID // zero value means this is a root spec
	DependsOn []ID

	Intent     string
	Scope      []string
	OutOfScope []string // an atomic spec must also state what it is not

	AcceptanceCriteria []AcceptanceCriterion
	Failure            FailureSemantics
	Diagrams           []Diagram // optional: a diagram that adds nothing is not required
	ComplexityBudget   string    // prose only -- no score, no number, no threshold
}

// ReadySpec is a Spec that has passed validation, so "ready for implementation"
// is a type rather than a boolean somebody can set. It has no exported fields
// and no constructor but Spec.ReadyForImplementation, so an invalid spec cannot
// be represented as ready. Returning it by value is enough: the only forgeable
// value is the zero ReadySpec, whose Spec() is the zero Spec -- itself invalid,
// so a `valid bool` flag would add nothing.
type ReadySpec struct{ spec Spec }

// Spec returns the validated specification, or -- on a zero ReadySpec -- the
// zero Spec, which is invalid by construction.
func (r ReadySpec) Spec() Spec { return r.spec }
