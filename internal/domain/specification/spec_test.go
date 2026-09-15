// Package specification_test exercises the contract from outside the package,
// which is the only way to prove the exported surface alone is enough to build
// a spec, validate it, and gate it to ready-for-implementation.
package specification_test

import (
	"errors"
	"strings"
	"testing"

	spec "github.com/getsyntegrity/atomwright/internal/domain/specification"
)

// valid is the one baseline every case mutates, so each case differs from a
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

func TestValidSpecReachesReadyForImplementation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*spec.Spec)
	}{
		{"a spec with no diagrams at all is valid", func(s *spec.Spec) {}},
		{"a spec carrying a diagram is valid", func(s *spec.Spec) {
			s.Diagrams = []spec.Diagram{{Title: "Validation flow", Source: "flowchart TD; A-->B;"}}
		}},
		{"a root spec without a parent is valid", func(s *spec.Spec) { s.Parent = "" }},
		{"a leaf spec without dependencies is valid", func(s *spec.Spec) { s.DependsOn = nil }},
		{"a behaviour that cannot fail may state why instead", func(s *spec.Spec) {
			s.Failure = spec.FailureSemantics{Statement: "These are types; nothing executes."}
		}},
		{"a criterion containing a look-alike word is still self-contained", func(s *spec.Spec) {
			s.AcceptanceCriteria = []spec.AcceptanceCriterion{"Re-running the identifier check is idempotent."}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := valid()
			c.mutate(&s)
			if err := s.Validate(); err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
			ready, err := s.ReadyForImplementation()
			if err != nil {
				t.Fatalf("ReadyForImplementation() error = %v, want nil", err)
			}
			if ready.Spec().ID != s.ID {
				t.Errorf("ready.Spec().ID = %q, want %q", ready.Spec().ID, s.ID)
			}
		})
	}
}

func TestInvalidSpecIsRejectedAndCannotBecomeReady(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*spec.Spec)
		field   string
		wantErr error
	}{
		{"a spec without an identity is rejected", func(s *spec.Spec) { s.ID = "" }, "id", spec.ErrMissing},
		{"an identity padded with whitespace is rejected", func(s *spec.Spec) { s.ID = " ATOM-SPEC-001 " }, "id", spec.ErrBlank},
		{"a spec that is its own parent is rejected", func(s *spec.Spec) { s.Parent = s.ID }, "parent", spec.ErrSelfReference},
		{"a spec that depends on itself is rejected", func(s *spec.Spec) { s.DependsOn = []spec.ID{s.ID} }, "dependsOn[0]", spec.ErrSelfReference},
		{"a dependency listed twice is rejected", func(s *spec.Spec) { s.DependsOn = []spec.ID{"A", "A"} }, "dependsOn[1]", spec.ErrDuplicate},
		{"a blank dependency identity is rejected", func(s *spec.Spec) { s.DependsOn = []spec.ID{"  "} }, "dependsOn[0]", spec.ErrBlank},
		{"a spec without intent is rejected", func(s *spec.Spec) { s.Intent = "" }, "intent", spec.ErrMissing},
		{"a blank intent is rejected", func(s *spec.Spec) { s.Intent = "   " }, "intent", spec.ErrBlank},
		{"a spec without scope is rejected", func(s *spec.Spec) { s.Scope = nil }, "scope", spec.ErrMissing},
		{"a blank scope entry is rejected", func(s *spec.Spec) { s.Scope = []string{" "} }, "scope[0]", spec.ErrBlank},
		{"a spec that never says what it is not is rejected", func(s *spec.Spec) { s.OutOfScope = nil }, "outOfScope", spec.ErrMissing},
		{"a blank out-of-scope entry is rejected", func(s *spec.Spec) { s.OutOfScope = []string{""} }, "outOfScope[0]", spec.ErrBlank},
		{"a spec without acceptance criteria is rejected", func(s *spec.Spec) { s.AcceptanceCriteria = nil }, "acceptanceCriteria", spec.ErrMissing},
		{"a blank acceptance criterion is rejected", func(s *spec.Spec) { s.AcceptanceCriteria = []spec.AcceptanceCriterion{" "} }, "acceptanceCriteria[0]", spec.ErrBlank},
		{"an acceptance criterion repeated verbatim is rejected", func(s *spec.Spec) {
			s.AcceptanceCriteria = []spec.AcceptanceCriterion{"It validates.", "It validates."}
		}, "acceptanceCriteria[1]", spec.ErrDuplicate},
		{"an acceptance criterion that back-references another is rejected", func(s *spec.Spec) { s.AcceptanceCriteria = []spec.AcceptanceCriterion{"Same as the previous one."} }, "acceptanceCriteria[0]", spec.ErrNotSelfContained},
		{"an acceptance criterion that only says ditto is rejected", func(s *spec.Spec) { s.AcceptanceCriteria = []spec.AcceptanceCriterion{"Ditto."} }, "acceptanceCriteria[0]", spec.ErrNotSelfContained},
		{"failure semantics that were never declared are rejected", func(s *spec.Spec) { s.Failure = spec.FailureSemantics{} }, "failure.statement", spec.ErrMissing},
		{"a behaviour that cannot fail without saying why is rejected", func(s *spec.Spec) { s.Failure = spec.FailureSemantics{Statement: "  "} }, "failure.statement", spec.ErrBlank},
		{"a diagram without a title is rejected", func(s *spec.Spec) { s.Diagrams = []spec.Diagram{{Title: " ", Source: "flowchart TD;"}} }, "diagrams[0].title", spec.ErrBlank},
		{"a diagram without source is rejected", func(s *spec.Spec) { s.Diagrams = []spec.Diagram{{Title: "Flow"}} }, "diagrams[0].source", spec.ErrMissing},
		{"a spec without a complexity budget is rejected", func(s *spec.Spec) { s.ComplexityBudget = "" }, "complexityBudget", spec.ErrMissing},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := valid()
			c.mutate(&s)

			ready, err := s.ReadyForImplementation()
			if err == nil {
				t.Fatalf("ReadyForImplementation() error = nil, want a problem on %q", c.field)
			}
			// An invalid spec must not reach ready-for-implementation, and
			// must hand back nothing that could pass as approved work.
			if got := ready.Spec(); got.ID != "" || got.Intent != "" {
				t.Errorf("ReadySpec.Spec() = %+v, want the zero Spec", got)
			}
			if !errors.Is(err, c.wantErr) {
				t.Errorf("errors.Is(aggregate, %v) = false; got %v", c.wantErr, err)
			}
			if got := problem(t, err, c.field); !errors.Is(got, c.wantErr) {
				t.Errorf("problem on %q = %v, want errors.Is(..., %v)", c.field, got, c.wantErr)
			}
		})
	}
}

// problem finds the reported problem for a field, so a case asserts both which
// field was blamed and why.
func problem(t *testing.T, err error, field string) spec.ValidationError {
	t.Helper()
	var problems spec.ValidationErrors
	if !errors.As(err, &problems) {
		t.Fatalf("error is not a specification.ValidationErrors: %v", err)
	}
	for _, p := range problems {
		if p.Field == field {
			return p
		}
	}
	t.Fatalf("no problem reported for field %q; got %v", field, problems)
	return spec.ValidationError{}
}

func TestAllProblemsAreReportedTogether(t *testing.T) {
	s := valid()
	s.ID, s.Intent, s.Scope, s.AcceptanceCriteria = "", "", nil, nil

	err := s.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want several problems")
	}
	var problems spec.ValidationErrors
	if !errors.As(err, &problems) {
		t.Fatalf("error is not a specification.ValidationErrors: %v", err)
	}

	want := []string{"id", "intent", "scope", "acceptanceCriteria"}
	if len(problems) != len(want) {
		t.Fatalf("len(problems) = %d, want %d: %v", len(problems), len(want), problems)
	}
	for i, field := range want {
		if problems[i].Field != field {
			t.Errorf("problems[%d].Field = %q, want %q", i, problems[i].Field, field)
		}
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Error() = %q, want it to mention %q", err.Error(), field)
		}
	}
}
