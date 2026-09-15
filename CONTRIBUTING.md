# Contributing to Atomwright

Atomwright is an independent fork of Gentle AI; see [`NOTICE.md`](NOTICE.md).

> **Repository status.** The inherited implementation was removed. What remains is
> the ADR-0001 layers, most of them still holding only a `doc.go`. #66 ATOM-BOOT-006
> added the `cmd/atomwright` composition root and the `make check` gate; the functional
> epics are still unimplemented. See
> [ADR-0002](docs/adr/0002-atomwright-identity-and-greenfield-baseline.md).

## Table of Contents

- [Issue-first workflow](#issue-first-workflow)
- [AI-assisted contributions](#ai-assisted-contributions)
- [Label system](#label-system)
- [Development setup](#development-setup)
- [Architecture rules](#architecture-rules)
- [Commit convention](#commit-convention)
- [Branch naming](#branch-naming)
- [Pull request rules](#pull-request-rules)
- [Code of conduct](#code-of-conduct)

## Issue-first workflow

**No PR without an issue. No exceptions.**

1. Open an issue using the [bug report or feature request template](https://github.com/getsyntegrity/atomwright/issues/new/choose).
2. Wait for `status:approved`. Work may begin only once the issue carries it.
3. Comment on the issue so others know you have taken it.
4. Open a PR referencing the approved issue.

An issue without `status:approved` is usually waiting on information
(`status:needs-info`) or on an architectural decision (`status:needs-design`).
Implementing before that decision lands means the work gets thrown away.

## AI-assisted contributions

AI assistance is allowed, but you must understand and own the complete submission.
Before opening a PR:

- [ ] Confirm the change matches the approved issue scope.
- [ ] Inspect every changed line.
- [ ] Remove invented, unverifiable, or unrelated output.
- [ ] Identify the responsible cause or invariant; confirm the fix resolves it
      rather than masking the symptom.
- [ ] Remove duplicate authority and unrelated complexity; keep the fix proportionate.
- [ ] Run the checks below and report the actual outcomes.
- [ ] Be ready to explain the design and its tradeoffs.
- [ ] Disclose material AI assistance in the PR.

See the [AI-Assisted Contribution Policy](AI_POLICY.md) for disclosure boundaries and
attribution rules.

## Label system

**Type** (on PRs): `type:bug`, `type:feature`, `type:docs`, `type:refactor`,
`type:chore`, `type:breaking-change`.

**Size** (on PRs): `size:exception` — maintainer-approved exception to the
400 changed-line review budget.

**Status** (on issues): `status:needs-review`, `status:approved`,
`status:in-progress`, `status:blocked`, `status:wont-fix`.

**Priority** (on issues): `priority:critical`, `priority:high`,
`priority:medium`, `priority:low`.

## Development setup

Requires Go 1.25 or newer.

```sh
git clone git@github.com:getsyntegrity/atomwright.git
cd atomwright
make binary
./atomwright
```

`cmd/atomwright` is the only composition root and the only binary. It builds and
runs today, but no functional command is wired to it yet — the epics that add one
are #9–#19.

### Verifying a change

One gate, run the same way in both places:

```sh
make check
```

That runs `gofmt`, `go vet`, `go mod tidy -diff`, `go build`, `go test`, and the
ADR-0001 architecture checks. CI runs the identical `make check` target on every
pull request, so there is no CI-only or local-only step in either direction — and
`internal/gates` fails the build if the Makefile and the workflow ever drift apart.
Report the real output in your PR. `go build` alone is not verification — it
compiles without running tests.

`make help` lists the individual gates (`fmt`, `vet`, `tidy`, `build`, `test`,
`arch`) for when you want to run one in isolation while fixing it.

## Architecture rules

[ADR-0001](docs/adr/0001-modular-monolith-skeleton.md) is binding, as amended by
[ADR-0002](docs/adr/0002-atomwright-identity-and-greenfield-baseline.md) and
[ADR-0003](docs/adr/0003-test-only-third-party-imports.md). Nothing
that contradicts an accepted ADR merges without a superseding ADR.

```
cmd/*       ──▶  internal/bootstrap  ──▶  (every layer, wiring only)
adapters/*  ──▶  internal/application  ──▶  internal/domain/*
adapters/*  ──▶  internal/domain/*      (ports and contractual types only)
platform/*  ──▶  stdlib, platform/* internals, third-party in test files only
internal/domain/*  ──▶  stdlib, other internal/domain/* packages,
                        third-party in test files only
```

- `platform/*` is cross-cutting infrastructure. It depends on nothing above it:
  not `internal/application`, and never `internal/domain/*`.
- `adapters/*` may import `internal/domain/*` only to implement a port contract
  declared there, never to call domain logic.
- `internal/domain/*` must never import application, platform, or adapters —
  from a `_test.go` file either. ADR-0003 relaxes only third-party reach, never
  a layer edge.
- `internal/domain/*` and `platform/*` may import a third-party package from a
  `_test.go` file. Their production imports stay standard library plus their own
  layer ([ADR-0003](docs/adr/0003-test-only-third-party-imports.md)).
- `cmd/*` sees `internal/bootstrap` and the standard library only.
- Enforcement is automated: `internal/architecture` runs the ADR-0001 table as
  tests on every `go test ./...`. The allowlist is default-deny, so a new
  cross-boundary import fails until it is added to the ADR-0001 table and to
  `allowedLayerEdges`, in the same change, with the justification in the PR.

### Module layout

One Go module, `github.com/getsyntegrity/atomwright`, rooted at the repository
root. `cmd/atomwright` is the only composition root and links every bounded
context together, so no bounded context has an independent build, test, or
release lifecycle that would justify a second module.

There is no second entry point. ADR-0001 planned to keep `cmd/gentle-ai` as a
temporary legacy alias during migration;
[ADR-0002](docs/adr/0002-atomwright-identity-and-greenfield-baseline.md) removed
it outright instead, so no retirement plan is outstanding. A second `cmd/*`
package is allowed only as another thin process shell over `internal/bootstrap`
— never as a second composition root, which is a direct ADR-0001 violation.

- Never add a `go.mod` under `internal/`, `platform/`, or `adapters/`. The
  commands in [Verifying a change](#verifying-a-change) run from the repository
  root and already cover the whole tree. The one exception is a `testdata/`
  directory: the Go toolchain ignores `testdata/` entirely, so a module there
  is never built, tested, or released with the real one. The architecture
  fixtures in `internal/architecture/testdata/` use this to give `go list` a
  module boundary for deliberately-wrong import graphs.
- There is no `go.work`. A workspace file only has an effect across two or more
  modules; with one module it is a no-op that every contributor still has to
  read and reason about.
- Splitting a bounded context into its own module needs an ADR superseding
  ADR-0001, not a `go.mod` or `go.work` tweak.

## Commit convention

Conventional commits: `type(scope): subject`.

Allowed types: `feat`, `fix`, `docs`, `refactor`, `chore`, `style`, `perf`,
`test`, `build`, `ci`, `revert`.

Breaking changes append `!` after the type or scope and carry a
`BREAKING CHANGE:` footer.

Write commit messages, code, comments, and documentation in English. Do not add
AI attribution beyond an explicit `Co-Authored-By` line.

## Branch naming

```
^(feat|fix|chore|docs|style|refactor|perf|test|build|ci|revert)\/[a-z0-9._-]+$
```

Lowercase only; hyphens, dots, or underscores as separators. Examples:
`feat/user-login`, `fix/crash-on-startup`, `docs/api-reference`.

## Pull request rules

**Size budget.** Keep PRs at or below **400 changed lines**. This is a deliberate
cognitive-load limit: a PR should be reviewable in roughly 60 minutes. If your
change cannot fit, split it into chained PRs. Use `size:exception` only when a
maintainer agrees the large diff is unavoidable.

**Work-unit commits.** Structure commits by deliverable unit, not by file type. A
good commit carries the code, tests, and docs needed to understand and verify one
behaviour. Reverting one commit should not remove unrelated work.

**Documentation size.** 200 lines is a warning, 250 recommends a split, 300 is a
hard limit. Never grow a document indefinitely to absorb new scope.

**Before opening a PR.** Run the verification commands above, confirm the change
matches the approved issue scope, and link the issue with `Closes #NNN`.

**Review comments.** Warm, direct, and useful quickly. Lead with the actionable
point; explain why when it helps.

## Code of conduct

Be respectful and constructive. Report unacceptable behaviour to the maintainers
through a GitHub issue or direct contact.
