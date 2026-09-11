# Atomwright upstream baseline

Atomwright is an independent fork of [Gentle AI](https://github.com/Gentleman-Programming/gentle-ai).
This document records the exact point the fork starts from, how that baseline was verified, and which
inherited identifiers are intentionally left unchanged.

## Baseline

| Field | Value |
| --- | --- |
| Upstream repository | `https://github.com/Gentleman-Programming/gentle-ai` |
| Upstream remote | `upstream` (fetch only; push URL disabled in this clone) |
| Base commit | `a7502587b0643472a550a3624b1c60dd00ee6312` |
| Base commit subject | `fix(windows): prevent consoles for background Git processes (#4476)` |
| Base commit date | `2026-09-11 14:24:42 +0000` |
| `git describe` | `v2.7.0-114-ga7502587` |
| Atomwright repository | `https://github.com/pablogore/atomwright` (`origin`) |
| Atomwright default branch | `main` |

`origin/main` was set to this exact upstream commit before any Atomwright change was made, so the
fork history is the upstream history. The three pre-fork seed commits that previously occupied
`origin/main` are preserved under the tag `atomwright-seed` (`db9322ec`).

## Entry points

| Surface | Location |
| --- | --- |
| Product CLI binary | `cmd/gentle-ai/main.go` → `internal/app.Run()` |
| Telemetry collector binary | `cmd/gentle-telemetry/main.go` |
| Command dispatcher | `internal/app/app.go` (hand-written `switch`; no cobra) |
| Help banner | `internal/app/help.go` |
| TUI welcome tagline | `internal/tui/styles/styles.go` |
| Go module | `github.com/gentleman-programming/gentle-ai/v2` (Go 1.25.10) |
| Packages | 82 |

## Baseline verification

Commands run against the base commit, before any modification. All were executed; none are inferred.

| Command | Result |
| --- | --- |
| `go build ./...` | exit 0, no output |
| `go vet ./...` | exit 0, no output |
| `go test ./...` | exit 0 — 78 packages ok, 0 failed, 4 with no test files |

The repository's own CI (`.github/workflows/ci.yml`) additionally runs
`go run ./internal/gofmtcheck` and `./scripts/deadcode-ratchet.sh`. There is no `Makefile`,
`Taskfile`, or `golangci-lint` configuration; `go vet` plus `gofmtcheck` is the entire lint surface.
The `package.json` `test` script is a placeholder stub and is not the build system.

**Shipwright is not configured in this repository** (zero references at the base commit). Adopting it
as the default verification backend is future work.

## Renamed in this change (user-facing prose only)

| Surface | Change |
| --- | --- |
| `internal/app/help.go` | Help banner product name, `uninstall` description, documentation link, fork disclaimer |
| `internal/tui/styles/styles.go` | Welcome-screen tagline |
| `README.md` | Replaced with the Atomwright vision, origin, and license attribution |
| `LICENSE` | Atomwright copyright **added**; upstream notice retained verbatim |
| `TRADEMARKS.md` | Upstream policy retained verbatim; fork notice prepended |
| `AGENTS.md`, `CONTRIBUTING.md` | Document titles and intro prose |
| `package.json` | `description`, `repository`, `bugs`, `homepage` metadata |

## Deliberately unchanged (deferred identifiers)

Renaming any of these is a migration, not a rename. Each is recorded here as future atomic work.

| Identifier | Location | Why deferred |
| --- | --- | --- |
| Go module path `github.com/gentleman-programming/gentle-ai/v2` | `go.mod:1` | Import path in 628 Go files; breaks `go install` and every CI cache |
| CLI invocation token `gentle-ai` | `.goreleaser.yaml:14`, `internal/app/help.go` | Renaming changes installer and release behavior |
| Config directory `~/.gentle-ai/` | `internal/state/state.go:15` | Orphans existing installs' state and backups without a migration step |
| `GENTLE_AI_*` environment variables (44 distinct) | `internal/cli/channel.go:15` and others | Documented and scripted; silent breakage in user dotfiles and CI |
| Update registry owner/repo | `internal/update/registry.go:19-21` | Drives live self-update against the GitHub Releases API |
| Advisory URL | `internal/update/advisory.go:27` | Fetched at launch; depends on a real upstream release tag |
| Installer constants and banners | `scripts/install.sh:17-19,534`, `scripts/install.ps1:28-29,54` | Out of scope: this change must not alter installer behavior |
| Pinned release policy config | `internal/releasepolicy/policy.go:671` | Validated against the real `.goreleaser.yaml`; release-critical |
| Injected block markers `<!-- gentle-ai:... -->` | `internal/components/sdd/inject.go:1229` | Written into users' config files; renaming orphans installed blocks |
| Protocol identifiers `gentle-ai.<name>/v<N>` | `contracts/**`, `internal/telemetry/telemetry.go:15` | Negotiated contract names; a rename is a breaking protocol change |
| OpenCode agent key `gentle-orchestrator` | `internal/tui/screens/model_picker.go:204` | Persisted as a literal JSON key in users' `opencode.json` |
| Skill IDs and directories `gentle-ai-*` | `internal/model/types.go:145`, `skills/` | Installed verbatim into users' `.claude/skills/` |
| Embedded agent assets and golden files | `internal/assets/**`, `testdata/golden/*.golden` | Operational instructions, not brand material; changing them changes runtime output |
| Telemetry endpoint and systemd units | `internal/telemetry/telemetry.go:28`, `deploy/telemetry/` | Live infrastructure; requires a server-side migration |
| Sibling projects `engram`, `gentle-pi`, `gentle-engram`, `gga` | `internal/update/registry.go:67`, `internal/agents/pi/adapter.go` | External packages and repositories this project does not own |
| Private Go identifiers (`GentleAIUpgradeVersion`, …) | `internal/tui/model.go` | No external risk, but out of scope for a branding-only change |
| Historical `PRD.md`, `PRD-AGENT-BUILDER.md` | repository root | Use the older "Gentleman AI" name; historical artifacts |

## Known limitations

- The shipped CLI is still the inherited Gentle AI runtime. Atomwright's atomic-delivery workflow is
  not implemented.
- Because the invocation token is still `gentle-ai`, this fork does not yet fully satisfy the
  "Forks and modified distributions" clause of `TRADEMARKS.md` at the CLI-identifier level. That must
  be resolved before any public distribution or release.
- `docs/**` still contains extensive inherited Gentle AI prose. It was left untouched to keep this
  change atomic and reviewable.
