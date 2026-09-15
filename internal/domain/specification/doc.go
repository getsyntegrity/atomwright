// Package specification is the internal/domain bounded context for #10
// ATOM-SPEC. It defines the atomic specification contract (#23 ATOM-SPEC-001):
// identity and relationships, intent, scope and out-of-scope, independently
// verifiable acceptance criteria, mandatory failure semantics, optional
// Mermaid diagrams, and a complexity budget. Validation reports every problem
// in one pass, and only a validated Spec becomes a ReadySpec -- so an invalid
// spec cannot transition to ready-for-implementation.
package specification
