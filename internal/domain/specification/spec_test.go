// Package specification_test exercises the contract from outside the package,
// which is the only way to prove the exported surface alone is enough to build
// a spec, validate it, and gate it to ready-for-implementation.
//
// The suite is written with go-specs. Its third-party import is permitted here
// because it appears only in a _test.go file -- see ADR-0003.
package specification_test

import (
	"errors"
	"strings"
	"testing"

	spec "github.com/getsyntegrity/atomwright/internal/domain/specification"
	"github.com/pablogore/go-specs/specs"
)

// valid is the one baseline every spec mutates, so each case differs from a
// passing spec in exactly one way and a failure names only the rule under test.
func valid() spec.Spec {
	return spec.Spec{
		ID: "ATOM-SPEC-001", Parent: "ATOM-SPEC",
		DependsOn:          []spec.ID{"ATOM-BOOT-002"},
		Intent:             "Define the atomic specification contract.",
		Scope:              []string{"The required fields of an atomic specification."},
		OutOfScope:         []string{"OpenSpec file formats.", "Complexity scoring."},
		AcceptanceCriteria: []spec.AcceptanceCriterion{"A spec missing a required field is rejected."},
		Failure:            spec.FailureSemantics{CanFail: true, Statement: "An invalid spec cannot become ready."},
		ComplexityBudget:   "The contract fits a single human-readable page.",
	}
}

// accepted is one shape the contract must allow.
type accepted struct {
	behaviour string
	mutate    func(*spec.Spec)
}

// rejected is one shape the contract must refuse, naming the field it blames
// and the sentinel reason, so a case asserts both.
type rejected struct {
	behaviour string
	mutate    func(*spec.Spec)
	field     string
	reason    error
}

func acceptedSpecs() []accepted {
	return []accepted{
		{"accepts a spec with no diagrams at all", func(s *spec.Spec) {}},
		{"accepts a spec carrying a diagram", func(s *spec.Spec) {
			s.Diagrams = []spec.Diagram{{Title: "Validation flow", Source: "flowchart TD; A-->B;"}}
		}},
		{"accepts a root spec with no parent", func(s *spec.Spec) { s.Parent = "" }},
		{"accepts a leaf spec with no dependencies", func(s *spec.Spec) { s.DependsOn = nil }},
		{"accepts a behaviour that cannot fail when it states why", func(s *spec.Spec) {
			s.Failure = spec.FailureSemantics{Statement: "These are types; nothing executes."}
		}},
		{"accepts a criterion containing a look-alike word", func(s *spec.Spec) {
			s.AcceptanceCriteria = []spec.AcceptanceCriterion{"Re-running the identifier check is idempotent."}
		}},
	}
}

func rejectedSpecs() []rejected {
	return []rejected{
		{"rejects a spec with no identity", func(s *spec.Spec) { s.ID = "" }, "id", spec.ErrMissing},
		{"rejects an identity padded with whitespace", func(s *spec.Spec) { s.ID = " ATOM-SPEC-001 " }, "id", spec.ErrBlank},
		{"rejects a spec that is its own parent", func(s *spec.Spec) { s.Parent = s.ID }, "parent", spec.ErrSelfReference},
		{"rejects a spec that depends on itself", func(s *spec.Spec) { s.DependsOn = []spec.ID{s.ID} }, "dependsOn[0]", spec.ErrSelfReference},
		{"rejects a dependency listed twice", func(s *spec.Spec) { s.DependsOn = []spec.ID{"A", "A"} }, "dependsOn[1]", spec.ErrDuplicate},
		{"rejects a blank dependency identity", func(s *spec.Spec) { s.DependsOn = []spec.ID{"  "} }, "dependsOn[0]", spec.ErrBlank},
		{"rejects a spec with no intent", func(s *spec.Spec) { s.Intent = "" }, "intent", spec.ErrMissing},
		{"rejects a blank intent", func(s *spec.Spec) { s.Intent = "   " }, "intent", spec.ErrBlank},
		{"rejects a spec with no scope", func(s *spec.Spec) { s.Scope = nil }, "scope", spec.ErrMissing},
		{"rejects a blank scope entry", func(s *spec.Spec) { s.Scope = []string{" "} }, "scope[0]", spec.ErrBlank},
		{"rejects a spec that never says what it is not", func(s *spec.Spec) { s.OutOfScope = nil }, "outOfScope", spec.ErrMissing},
		{"rejects a blank out-of-scope entry", func(s *spec.Spec) { s.OutOfScope = []string{""} }, "outOfScope[0]", spec.ErrBlank},
		{"rejects a spec with no acceptance criteria", func(s *spec.Spec) { s.AcceptanceCriteria = nil }, "acceptanceCriteria", spec.ErrMissing},
		{"rejects a blank acceptance criterion", func(s *spec.Spec) {
			s.AcceptanceCriteria = []spec.AcceptanceCriterion{" "}
		}, "acceptanceCriteria[0]", spec.ErrBlank},
		{"rejects an acceptance criterion repeated verbatim", func(s *spec.Spec) {
			s.AcceptanceCriteria = []spec.AcceptanceCriterion{"It validates.", "It validates."}
		}, "acceptanceCriteria[1]", spec.ErrDuplicate},
		{"rejects an acceptance criterion that back-references another", func(s *spec.Spec) {
			s.AcceptanceCriteria = []spec.AcceptanceCriterion{"Same as the previous one."}
		}, "acceptanceCriteria[0]", spec.ErrNotSelfContained},
		{"rejects an acceptance criterion that only says ditto", func(s *spec.Spec) {
			s.AcceptanceCriteria = []spec.AcceptanceCriterion{"Ditto."}
		}, "acceptanceCriteria[0]", spec.ErrNotSelfContained},
		{"rejects failure semantics that were never declared", func(s *spec.Spec) {
			s.Failure = spec.FailureSemantics{}
		}, "failure.statement", spec.ErrMissing},
		{"rejects a behaviour that cannot fail without saying why", func(s *spec.Spec) {
			s.Failure = spec.FailureSemantics{Statement: "  "}
		}, "failure.statement", spec.ErrBlank},
		{"rejects a diagram with no title", func(s *spec.Spec) {
			s.Diagrams = []spec.Diagram{{Title: " ", Source: "flowchart TD;"}}
		}, "diagrams[0].title", spec.ErrBlank},
		{"rejects a diagram with no source", func(s *spec.Spec) {
			s.Diagrams = []spec.Diagram{{Title: "Flow"}}
		}, "diagrams[0].source", spec.ErrMissing},
		{"rejects a spec with no complexity budget", func(s *spec.Spec) { s.ComplexityBudget = "" }, "complexityBudget", spec.ErrMissing},
	}
}

func TestSpecification(t *testing.T) {
	specs.Describe(t, "the atomic specification contract", func(s *specs.Spec) {
		s.When("the spec satisfies the contract", func(s *specs.Spec) {
			for _, c := range acceptedSpecs() {
				s.It(c.behaviour, func(ctx *specs.Context) {
					subject := valid()
					c.mutate(&subject)

					ctx.Expect(subject.Validate() == nil).To(specs.BeTrue())

					ready, err := subject.ReadyForImplementation()
					ctx.Expect(err == nil).To(specs.BeTrue())
					specs.ExpectT(ctx, ready.Spec().ID).ToEqual(subject.ID)
				})
			}
		})

		s.When("the spec breaks the contract", func(s *specs.Spec) {
			for _, c := range rejectedSpecs() {
				s.It(c.behaviour, func(ctx *specs.Context) {
					subject := valid()
					c.mutate(&subject)

					ready, err := subject.ReadyForImplementation()

					// It cannot become ready, and it hands back nothing that
					// could pass as approved work.
					ctx.Expect(err != nil).To(specs.BeTrue())
					specs.ExpectT(ctx, ready.Spec().ID).ToEqual(spec.ID(""))
					specs.ExpectT(ctx, ready.Spec().Intent).ToEqual("")

					// The sentinel is reachable through the aggregate, and the
					// blamed field carries that same reason.
					ctx.Expect(errors.Is(err, c.reason)).To(specs.BeTrue())
					blamed, found := problemFor(err, c.field)
					ctx.Expect(found).To(specs.BeTrue())
					ctx.Expect(errors.Is(blamed, c.reason)).To(specs.BeTrue())
				})
			}
		})

		s.It("reports every problem in one pass instead of one at a time", func(ctx *specs.Context) {
			subject := valid()
			subject.ID, subject.Intent, subject.Scope, subject.AcceptanceCriteria = "", "", nil, nil

			err := subject.Validate()
			ctx.Expect(err != nil).To(specs.BeTrue())

			var problems spec.ValidationErrors
			ctx.Expect(errors.As(err, &problems)).To(specs.BeTrue())

			want := []string{"id", "intent", "scope", "acceptanceCriteria"}
			specs.EqualTo(ctx, len(problems), len(want))
			for i, field := range want {
				specs.EqualTo(ctx, problems[i].Field, field)
				ctx.Expect(strings.Contains(err.Error(), field)).To(specs.BeTrue())
			}
		})
	})
}

// problemFor finds the reported problem for a field, so a spec can assert both
// which field was blamed and why.
func problemFor(err error, field string) (spec.ValidationError, bool) {
	var problems spec.ValidationErrors
	if !errors.As(err, &problems) {
		return spec.ValidationError{}, false
	}
	for _, problem := range problems {
		if problem.Field == field {
			return problem, true
		}
	}
	return spec.ValidationError{}, false
}
