# ADR-0003: Test-Only Third-Party Imports in Standard-Library-Only Layers

## Status

Accepted. This record amends [ADR-0001](0001-modular-monolith-skeleton.md)'s dependency table on
exactly one point: the `stdlib` rows for `platform/*` and `internal/domain/*`. Every other
decision in ADR-0001, and every decision in
[ADR-0002](0002-atomwright-identity-and-greenfield-baseline.md), stands unchanged.

## Date

2026-09-15

## Deciders

Atomwright maintainers, via #60 ATOM-BOOT / #111 ATOM-BOOT-007.

## Context

ADR-0001 states the rule for two layers as `stdlib`:

```
platform/*         -> stdlib, platform/* internals
internal/domain/*  -> stdlib, other internal/domain/* packages
```

#65 ATOM-BOOT-005 made that table executable in `internal/architecture`, where the two layers
appear in `externalImportsAllowed` as `false`. The import graph those rules evaluate is built by
`pkg.allImports()`, which merges all three `go list` import sets into one:

```go
merged := slices.Concat(p.Imports, p.TestImports, p.XTestImports)
```

With the sets merged, the rule cannot tell a production dependency from a test-only one. Adding
the project's testing framework to a domain test therefore fails the suite:

```
--- FAIL: TestDependencyDirection
    internal/domain/specification imports github.com/pablogore/go-specs/specs
        rule:   moduleDependencyRule
        reason: ADR-0001 restricts internal/domain/* to the standard library and its own
                layer; a third-party import is not allowed there
```

That is over-enforcement, not the decision ADR-0001 recorded. The reason domain and platform code
is standard-library-only is that a shipped binary must not carry infrastructure or vendor coupling
through its centre — the `stdlib` rows were written about **production** dependencies. The Go
toolchain excludes `_test.go` files from a non-test build, so a test-only import is linked into no
binary `go build ./...` produces. The rule as written forbids something the decision never intended
to forbid, and the cost is real: either the domain is tested with a different framework from the
rest of the codebase, or it is tested less.

## Decision

`internal/domain/*` and `platform/*` may take third-party imports through `TestImports` and
`XTestImports` — that is, from `_test.go` files — only.

Their production import set is unchanged: the standard library plus their own layer. An import that
appears in `Imports` is a production dependency and is still denied, including one that also
appears in a test set.

### The amended table

ADR-0001's canonical allowlist, restated here with the amendment applied. Only the last two rows of
the first block differ from ADR-0001; nothing else changes.

```
cmd/*             -> internal/bootstrap
internal/bootstrap -> internal/application, internal/domain/*, adapters/*, platform/*
adapters/*         -> internal/application
adapters/*         -> internal/domain/*      [ports and contractual types only]
internal/application -> internal/domain/*
platform/*         -> stdlib, platform/* internals, third-party in test files only
internal/domain/*  -> stdlib, other internal/domain/* packages (explicitly shared concepts only),
                      third-party in test files only

internal/domain/*    -X-> internal/application
internal/domain/*    -X-> adapters/*
internal/domain/*    -X-> platform/*
internal/application -X-> adapters/* (concrete)
cmd/*                -X-> adapters/* (concrete)
cmd/*                -X-> internal/application, internal/domain/*, platform/* (directly)
```

### What does not change

- **Every anti-edge still holds in test code.** `internal/domain/*` must never import
  `internal/application`, `adapters/*`, or `platform/*`, from a `_test.go` file either. The
  amendment is about third-party modules, not about layer direction, and `dependencyRule`,
  `restrictedImportRule`, and `compositionRootRule` ignore an import's origin entirely.
- **`externalImportsAllowed` stays default-deny for production.** The permission is a second,
  separate map (`testOnlyExternalImportsAllowed`), and every governed layer states a value in it
  explicitly, so a missing entry cannot silently read as allow or deny.
- **Unknown import sets fail closed.** An import that cannot be attributed to a known `go list` set
  is treated as a production import and denied. `originProduction` is the denying side, so only
  imports positively proven to come from `TestImports`/`XTestImports` alone gain the permission.
- **No testing framework is chosen here.** This record removes a mechanical blocker; adopting a
  framework is a separate decision.

## Decision classification

### Accepted now

- The two amended rows above, and their enforcement in `internal/architecture` through an import
  origin carried from `go list` to `moduleDependencyRule`.
- The fail-closed attribution rule for unknown or ambiguous import sets.

### Deferred

- Whether `adapters/*`, `internal/application`, `internal/bootstrap`, or `cmd/*` should ever have
  their production third-party access narrowed. They are unconstrained today, and this record does
  not revisit that.

### Open hypotheses

- That `go list`'s three import sets remain a sufficient proxy for "is this linked into the shipped
  binary". A build-tag-gated production file that imports a third-party package would still be
  denied, correctly, because it lands in `Imports`.

## Consequences

**Benefits:** the domain and platform layers can be tested with the same framework as the rest of
the codebase, so the layers with the strictest invariants stop being the hardest to test; the rule
now says what ADR-0001 meant instead of a stricter thing that happened to be easier to compute.

**Costs:** the allowlist grows a second map, so a reader has to hold two policies rather than one;
the distinction depends on `go list`'s import sets, which is a coarser signal than reading the
build; and a third-party package reached from a test is a real dependency of the module's `go.mod`
even though it is not in the binary, so the dependency count still grows.

## Rejected / deferred alternatives

- **Leave the rule as-is and test the domain with a different framework from the rest of the
  codebase** — rejected. It makes the centre of the system the least conveniently tested part of
  it, which is the opposite of the pressure the layering is meant to create, and it buys no
  protection: the framework was never going to be in the binary.
- **Allow third-party imports in domain production code** — rejected. That is the coupling the
  `stdlib` row exists to prevent, and it would be a real change to ADR-0001's decision rather than
  a correction to its enforcement.
- **An AST or `go/types` analyzer that inspects what crosses each edge** — rejected, for the same
  reason #65 rejected it: disproportionate to the question. Which import set a path came from is
  already a fact `go list` reports; nothing here needs to know what the code does with it.
- **Drop the merge and check `Imports` only** — rejected. It would make test files a loophole for
  every layer anti-edge, which is exactly what `allImports()` was written to close.

## Related work

- [ADR-0001](0001-modular-monolith-skeleton.md) — the dependency table this record amends.
- #65 ATOM-BOOT-005 — the default-deny allowlist and the merged import graph.
- #111 ATOM-BOOT-007 — the issue this record closes.
- #23 ATOM-SPEC-001 and every future story adding tests to `internal/domain/*` or `platform/*`.
