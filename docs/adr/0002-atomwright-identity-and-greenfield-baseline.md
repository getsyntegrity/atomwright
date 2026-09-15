# ADR-0002: Atomwright Identity and Greenfield Baseline

## Status

Accepted

## Date

2026-09-15

## Deciders

Atomwright maintainers, via #60 ATOM-BOOT.

## Context

ADR-0001 assumed the inherited Gentle AI implementation would stay in the
repository and be migrated away incrementally. It therefore recorded two
non-goals: the Go module path would not be renamed, and `cmd/gentle-ai`
would survive as a temporary legacy alias until the functional epics had
moved their entry points.

That assumption no longer holds. The inherited implementation was removed
outright rather than migrated: every `internal/*` package except `domain`
and `application`, both command binaries, and the supporting upstream
trees. What remains is seventeen packages that declare no imports and
contain only a `doc.go`.

The incremental migration ADR-0001 designed around has no subject left.
Continuing to defer the rename would mean carrying a module path naming a
different project through every import written from here on, with the cost
of the rename growing on each one.

## Decision

Supersede two decisions from ADR-0001 and leave the rest intact.

1. **The module path is `github.com/getsyntegrity/atomwright`.** ADR-0001
   listed renaming it as a non-goal; that non-goal is withdrawn. The rename
   was performed while all surviving packages declared zero imports, so no
   import path required rewriting.

2. **`cmd/gentle-ai` is removed, not deprecated.** ADR-0001 kept it as a
   thin legacy alias importing only `internal/bootstrap`. Neither the
   binary nor `internal/bootstrap` exists any more. `cmd/atomwright`
   becomes the sole composition root when ATOM-BOOT-006 (#66) creates it,
   and there is no migration window to manage.

Everything else in ADR-0001 stands unchanged: the layer boundaries, the
dependency direction, `cmd/atomwright` plus `internal/bootstrap` as the
composition root, and the rule that adapters may import `internal/domain`
only to satisfy a port contract.

Names inherited from Gentle AI are removed from project documentation and
identifiers. They are retained wherever attribution requires it: `LICENSE`
(MIT, upstream copyright notice), `NOTICE.md`, `TRADEMARKS.md`, and
`CONTRIBUTORS.md`.

## Decision classification

### Accepted now

- The module path rename, already applied.
- Removal of `cmd/gentle-ai` as a supported surface.
- Retention of upstream attribution in the legal and identity files.

### Deferred

- `cmd/atomwright` and `internal/bootstrap` themselves, owned by #66.
- Rewriting the runtime behaviour the removed packages used to provide.

### Open hypotheses

- Whether any inherited package is worth reintroducing deliberately, as new
  code written against these boundaries rather than restored wholesale.

## Consequences

The repository has no binary and no CI until #66 lands, so there is no
automated gate in that window; `go build`, `go vet` and `go test` are run
by hand.

Import paths, install instructions, and any external reference to the old
module path break immediately. Nothing external depends on it yet, which is
why this is the cheapest moment to absorb that break.

ADR-0001 must be read together with this record: its Non-goals section and
its `cmd/gentle-ai` coexistence plan are no longer in force.

## Rejected / deferred alternatives

- **Keeping the `gentle-ai/v2` module path.** Rejected: it names a
  different project, and the rename cost rises with every import added.

- **Keeping `cmd/gentle-ai` as a deprecated alias, per ADR-0001.**
  Rejected: the packages it depended on were removed, so the alias would
  have to be rebuilt before it could be deprecated.

- **Editing ADR-0001 in place.** Rejected: accepted decisions are
  superseded by a new record, never rewritten.

## Errata to ADR-0001

ADR-0001's *Accepted now* list states:

> No per-bounded-context `go.mod` (see **Modules**, with the explicit
> criterion for revisiting it).

ADR-0001 has no **Modules** section. The criterion it points to is real and
lives in that record's *Rejected / deferred alternatives*:

> Deferred until a bounded context has a genuine, independent
> lifecycle/build/versioning/distribution need — not for aesthetic
> separation.

This is a broken cross-reference, not a missing decision. Read that bullet
as pointing at *Rejected / deferred alternatives*.

Two points worth restating plainly, because the missing section is where a
reader would look for them:

- **The single module is a V1 build-layout choice, not the definition of the
  architecture.** What makes Atomwright a modular monolith is one deployed
  process with enforced package boundaries; a modular monolith can span
  several Go modules and remain one process. So splitting a bounded context
  into its own module does not end the architecture — it is a build-layout
  change, which is why it is gated on the criterion above rather than
  forbidden outright, and why ADR-0001 scopes the single-module decision to
  V1 and leaves it revisitable.
- **Out-of-process plugins are decided, not omitted.** ADR-0001 rejects
  `go-plugin`, gRPC and out-of-process providers for V1 and defers them to
  #58 ATOM-PLUG, which is itself gated on #18 ATOM-PROV having stable
  in-process contracts. That is the change that would end the single-process
  property, and it is deferred, not denied. Nothing here changes it.

## Related work

- ADR-0001 — Modular-Monolith Skeleton for Atomwright, superseded in part.
- #63 ATOM-BOOT-003 — scaffolded the surviving package skeletons.
- #66 ATOM-BOOT-006 — introduces `cmd/atomwright` and the CI baseline.
- #58 ATOM-PLUG — where out-of-process providers are evaluated, once
  #18 ATOM-PROV's in-process contracts are stable.
