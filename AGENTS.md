# Atomwright — Agent Instructions

> Atomwright is an independent fork of Gentle AI; see [`NOTICE.md`](NOTICE.md). The upstream
> trademark policy is reproduced unchanged in [`TRADEMARKS.md`](TRADEMARKS.md).

## What this repository currently is

A greenfield skeleton. The inherited implementation was removed; see
[ADR-0002](docs/adr/0002-atomwright-identity-and-greenfield-baseline.md).

- Module path: `github.com/getsyntegrity/atomwright`
- The ADR-0001 layers, most packages still holding only a `doc.go`
- One binary, `cmd/atomwright`, wired through `internal/bootstrap` (#66 ATOM-BOOT-006).
  It builds and runs; no functional command is attached to it yet
- One gate, `make check`, run identically by CI

## Read before writing code

1. [ADR-0001](docs/adr/0001-modular-monolith-skeleton.md) — the layer boundaries and
   dependency direction. Binding.
2. [ADR-0002](docs/adr/0002-atomwright-identity-and-greenfield-baseline.md) — what
   ADR-0001 no longer covers.
3. The `doc.go` of the package you are touching. It names the issue and epic that own it.

## Dependency direction

```
cmd/*       ──▶  internal/bootstrap  ──▶  (every layer, wiring only)
adapters/*  ──▶  internal/application  ──▶  internal/domain/*
platform/*  ──▶  stdlib, platform/* internals
```

- `cmd/*` sees `internal/bootstrap` and the standard library only. `internal/bootstrap`
  is the single composition root; it is the one package that may see every layer at once.
- `adapters/*` may reach into `internal/domain/*` only to implement a port contract
  declared there. `platform/*` must never reach into it at all.
- `internal/domain/*` must never import application, platform, or adapters — including from a
  `_test.go` file.
- `internal/domain/*` and `platform/*` may import a third-party package from a `_test.go` file
  only; their production imports stay standard library plus their own layer
  ([ADR-0003](docs/adr/0003-test-only-third-party-imports.md)).
- Enforcement is automated (#65 ATOM-BOOT-005): `internal/architecture` runs the ADR-0001
  table as tests, default-deny, on every `make check`.

## Verifying a change

Run the gate yourself and report the real output:

```sh
make check
```

It runs `gofmt`, `go vet`, `go mod tidy -diff`, `go build`, `go test`, and the
ADR-0001 architecture checks. CI runs the identical target, so a green `make check`
locally is the same gate that blocks the PR. Never add a gate to one side only —
`internal/gates` fails the build when the Makefile and the workflow drift apart.

`go build` alone is not verification — it compiles without running tests.

## Conventions

- Conventional commits. No AI attribution lines beyond an explicit `Co-Authored-By`.
- Code, comments, documentation, and commit messages are written in English.
- Docs: 200 lines is a warning, 250 recommends a split, 300 is a hard limit.
- Do not change an issue's functional scope to make work fit.
