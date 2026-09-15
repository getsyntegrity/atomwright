package specification

import (
	"fmt"
	"strings"
)

// Markers that make an acceptance criterion depend on its neighbours. This is
// a mechanical proxy for a human judgement, not a proof: whether a statement is
// independently verifiable cannot be decided by a machine, only whether the
// sentence points elsewhere instead of saying something. A criterion that
// passes this check still needs a reviewer.
var (
	backReferencePhrases = []string{"as above", "same as the previous", "see criterion"}
	// Matched as whole words, so "idempotent" is not mistaken for "idem".
	backReferenceWords = []string{"idem", "ditto"}
)

// Validate reports every problem in one pass as a ValidationErrors, or nil when
// the spec is sound. Fields are checked in a fixed order so the message is stable.
func (s Spec) Validate() error {
	var problems ValidationErrors
	add := func(field string, reason error) {
		problems = append(problems, ValidationError{Field: field, Reason: reason})
	}
	require := func(field, value string) {
		if reason := required(value); reason != nil {
			add(field, reason)
		}
	}

	require("id", string(s.ID))
	if !s.Parent.IsZero() {
		if reason := required(string(s.Parent)); reason != nil {
			add("parent", reason)
		} else if s.Parent == s.ID {
			add("parent", ErrSelfReference)
		}
	}
	s.validateDependencies(add)

	require("intent", s.Intent)
	validateEntries("scope", s.Scope, add)
	validateEntries("outOfScope", s.OutOfScope, add)
	s.validateAcceptanceCriteria(add)

	// The zero FailureSemantics is invalid: a spec that never declared how its
	// behaviour fails must not be able to reach implementation.
	require("failure.statement", s.Failure.Statement)

	// Diagrams are optional -- an empty slice is valid, because a diagram that
	// adds no information is noise. One that is present must be complete.
	for i, diagram := range s.Diagrams {
		require(fmt.Sprintf("diagrams[%d].title", i), diagram.Title)
		require(fmt.Sprintf("diagrams[%d].source", i), diagram.Source)
	}

	require("complexityBudget", s.ComplexityBudget)

	if len(problems) == 0 {
		return nil
	}
	return problems
}

// ReadyForImplementation gates the spec into ReadySpec, the only representation
// of approved work. An invalid spec cannot make the transition: it gets the
// problems and a zero ReadySpec.
func (s Spec) ReadyForImplementation() (ReadySpec, error) {
	if err := s.Validate(); err != nil {
		return ReadySpec{}, err
	}
	return ReadySpec{spec: s}, nil
}

// validateDependencies checks that every dependency is a usable identity, is
// not the spec itself, and appears only once.
func (s Spec) validateDependencies(add func(string, error)) {
	seen := make(map[ID]bool, len(s.DependsOn))
	for i, id := range s.DependsOn {
		at := fmt.Sprintf("dependsOn[%d]", i)
		reason := required(string(id))
		switch {
		case reason != nil:
			add(at, reason)
		case !s.ID.IsZero() && id == s.ID:
			add(at, ErrSelfReference)
		case seen[id]:
			add(at, ErrDuplicate)
		default:
			seen[id] = true
		}
	}
}

func (s Spec) validateAcceptanceCriteria(add func(string, error)) {
	if len(s.AcceptanceCriteria) == 0 {
		add("acceptanceCriteria", ErrMissing)
		return
	}
	seen := make(map[string]bool, len(s.AcceptanceCriteria))
	for i, criterion := range s.AcceptanceCriteria {
		at := fmt.Sprintf("acceptanceCriteria[%d]", i)
		text := strings.TrimSpace(string(criterion))
		switch {
		case text == "":
			add(at, ErrBlank)
		case seen[text]:
			add(at, ErrDuplicate)
		case isBackReference(text):
			add(at, ErrNotSelfContained)
		default:
			seen[text] = true
		}
	}
}

func isBackReference(criterion string) bool {
	lowered := strings.ToLower(criterion)
	for _, phrase := range backReferencePhrases {
		if strings.Contains(lowered, phrase) {
			return true
		}
	}
	for _, word := range strings.Fields(lowered) {
		word = strings.Trim(word, `.,;:!?()"'`)
		for _, marker := range backReferenceWords {
			if word == marker {
				return true
			}
		}
	}
	return false
}

// validateEntries checks a list that must carry at least one usable entry. An
// absent list is ErrMissing on the list itself; an empty entry is ErrBlank on
// that entry, because the author did write something there.
func validateEntries(field string, entries []string, add func(string, error)) {
	if len(entries) == 0 {
		add(field, ErrMissing)
		return
	}
	for i, entry := range entries {
		if entry == "" || strings.TrimSpace(entry) != entry {
			add(fmt.Sprintf("%s[%d]", field, i), ErrBlank)
		}
	}
}

// required separates "nothing was supplied" from "something meaningless was
// supplied". Surrounding whitespace counts: identities are compared for
// equality, so " ATOM-1" would silently become a second spec.
func required(value string) error {
	switch {
	case value == "":
		return ErrMissing
	case strings.TrimSpace(value) != value:
		return ErrBlank
	default:
		return nil
	}
}
