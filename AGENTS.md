# Atomwright — Agent Instructions

> Atomwright is an independent fork of Gentle AI; see [`NOTICE.md`](NOTICE.md). The upstream
> trademark policy is reproduced unchanged in [`TRADEMARKS.md`](TRADEMARKS.md).

## What this repository currently is

A greenfield skeleton. The inherited implementation was removed; see
[ADR-0002](docs/adr/0002-atomwright-identity-and-greenfield-baseline.md).

- Module path: `github.com/getsyntegrity/atomwright`
- Seventeen packages, each holding only a `doc.go`
- No binary and no CI until #66 ATOM-BOOT-006 lands

## Read before writing code

1. [ADR-0001](docs/adr/0001-modular-monolith-skeleton.md) — the layer boundaries and
   dependency direction. Binding.
2. [ADR-0002](docs/adr/0002-atomwright-identity-and-greenfield-baseline.md) — what
   ADR-0001 no longer covers.
3. The `doc.go` of the package you are touching. It names the issue and epic that own it.

## Dependency direction

```
adapters/*  ──▶  internal/application  ──▶  internal/domain/*
platform/*  ──▶  internal/application
```

- `adapters/*` and `platform/*` must never reach into `internal/domain/*`, except for an
  adapter implementing a port contract declared there.
- `internal/domain/*` must never import application, platform, or adapters.
- Automated enforcement arrives with #65 ATOM-BOOT-005. Until then, review is the gate.

## Verifying a change

There is no CI. Run these yourself and report the real output:

```sh
go build ./...
go vet ./...
gofmt -l .
go test ./...
```

`go build` alone is not verification — it compiles without running tests.

## Conventions

- Conventional commits. No AI attribution lines beyond an explicit `Co-Authored-By`.
- Code, comments, documentation, and commit messages are written in English.
- Docs: 200 lines is a warning, 250 recommends a split, 300 is a hard limit.
- Do not change an issue's functional scope to make work fit.
