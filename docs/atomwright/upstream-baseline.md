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
| Product CLI binary | `cmd/atomwright/main.go` → `internal/app.Run()` (was `cmd/gentle-ai/main.go` at the baseline) |
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
| `NOTICE.md` | **Added** — independent-fork statement, trademark ownership, inherited-identifier disclosure |
| `AGENTS.md`, `CONTRIBUTING.md` | Document titles, intro prose, contributor-facing repository links |

`TRADEMARKS.md` is **not** modified by this change. It is preserved byte-for-byte from the base
commit, so that the upstream policy travels with the code it governs unchanged. `git diff
a7502587..HEAD -- TRADEMARKS.md` produces no output. The fork statement lives in `NOTICE.md`
instead; nothing in this repository reinterprets, amends, or narrows the upstream policy.
| `package.json` | `description`, `repository`, `bugs`, `homepage` metadata |

## Resolved since the baseline (public CLI rename)

These identifiers were listed as deferred in the first change and have since been renamed. They are
recorded here so the register stays a true account of what is and is not still inherited.

| Identifier | Resolution | Compatibility |
| --- | --- | --- |
| CLI invocation token `gentle-ai` → `atomwright` | RESOLVED. `cmd/gentle-ai/` is now `cmd/atomwright/`; the binary, the archive member, and the token a user types are all `atomwright`. | **None.** No `gentle-ai` executable is built, shipped, or installed. There is no wrapper, shim, or alias. |
| Config directory `~/.gentle-ai/` → `~/.atomwright/` | RESOLVED. `internal/identity` owns both names; `internal/statemigration` copies `state.json` and `telemetry.json` on every install, sync, and upgrade. | Legacy state is **read and copied**, never moved or deleted. `backups/` deliberately stays in the legacy root because each manifest embeds an absolute `root_dir` that restore validates as an anti-tamper containment check. Both roots populated is a reported conflict, never a merge. |
| `GENTLE_AI_*` environment prefix → `ATOMWRIGHT_*` | RESOLVED. `internal/envcompat` resolves the current prefix first and the legacy prefix second. | The 14 suffixes in `envcompat.AliasedSuffixes` still honour `GENTLE_AI_*`. `ATOMWRIGHT_*` wins when both are set, and a startup warning names both variables and never their values. This alias is permanent, not transitional: the variables live in dotfiles and CI this project cannot edit. |
| Update registry owner/repo | RESOLVED. Self-update targets `pablogore/atomwright`. | — |
| Installer constants and banners | RESOLVED. `scripts/install.sh` and `scripts/install.ps1` fetch and install `atomwright` from `pablogore/atomwright`. | — |
| Go module path `github.com/gentleman-programming/gentle-ai/v2` → `github.com/pablogore/atomwright/v2` | RESOLVED. `go.mod`, `bench/go.mod`, every import in 646 Go files, both installer scripts, the `.goreleaser.yaml` ldflags symbol and its byte-exact pinned copy in `internal/releasepolicy/policy.go` now use the Atomwright path. | **This was not a cosmetic rename.** The inherited path resolves through the Go module proxy to the *upstream* repository, which ships `cmd/gentle-ai` and which this project cannot publish to: `go install github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright@latest` fails with `module ...@latest found (v2.7.0), but does not contain package .../cmd/atomwright`. Consequences before the move: Windows had **no install path at all** (binary distribution is on Authenticode hold, so `go install` is its only route) and `--channel beta` was broken on every platform. Renaming the module path is what makes `go install` able to install Atomwright. Previously published `github.com/gentleman-programming/gentle-ai/v2` versions are unaffected — they remain upstream's and were never ours to serve. |

**Homebrew support was removed, not renamed.** The release path previously published a formula to
an upstream tap. No Atomwright-owned tap exists, so continuing to publish there would have shipped a
formula under someone else's namespace. Homebrew publication was therefore removed from
`.goreleaser.yaml` and from both installers entirely, and every `brew install` / `brew tap` /
`brew upgrade` instruction was removed from the documentation. The supported install paths are now
exactly two: the curl install script and `go install`. Homebrew remains supported only as a *system
package manager* for prerequisites such as `git`, `curl`, and `node`. Restoring a Homebrew
distribution channel under an Atomwright-owned tap is a separate future change.

**The only compatibility provided is inbound reading.** Atomwright reads legacy configuration in
`~/.gentle-ai/` and legacy `GENTLE_AI_*` environment variables. It does not ship, install, alias, or
support a `gentle-ai` executable in any form.

## Deliberately unchanged (deferred identifiers)

Renaming any of these is a migration, not a rename. Each is recorded here as future atomic work.

| Identifier | Location | Why deferred |
| --- | --- | --- |
| Advisory URL | `internal/update/advisory.go` | Fetched at launch; depends on a real upstream release tag |
| Pinned release policy config | `internal/releasepolicy/policy.go` | Validated against the real `.goreleaser.yaml`; release-critical |
| Injected block markers `<!-- gentle-ai:... -->` | `internal/components/sdd/inject.go` | Written into users' `CLAUDE.md`, `AGENTS.md`, and other agent config files; renaming the marker orphans every block already installed on a user's machine, so sync would append a second copy instead of updating the first |
| Protocol identifiers `gentle-ai.<name>/v<N>` (211 recorded; 208 distinct spellings match outside `docs/` today) | `contracts/**`, `internal/telemetry/telemetry.go`, `internal/reviewtransaction/**` | Negotiated contract names; a rename is a breaking protocol change. One of them is not merely a label: `internal/reviewtransaction/authority_repair.go` feeds the literal `gentle-ai.review-repository-binding/v1` into `sha256.Sum256`, so changing that string changes a persisted binding digest and invalidates existing review authority |
| `GENTLE_AI_REVIEW_*` reviewer prompt markers | `internal/reviewerprovider/**`, review prompt assembly | Wire format, not configuration. `GENTLE_AI_REVIEW_BINDING ` is the literal first bytes of every reviewer prompt and `GENTLE_AI_FROZEN_CANDIDATE_CONTEXT` names a prohibited transport; neither is an environment variable, so aliasing them would change the protocol |
| `GENTLE_AI_TELEMETRY` pinned contract enum | `contracts/telemetry/v1/schemas/`, `internal/telemetry/killswitch.go` | The published telemetry contract pins this exact string as the `source` enum value. It is a wire value, deliberately distinct from the `ATOMWRIGHT_TELEMETRY` environment variable that shares its spelling |
| Review authority store path component `<git-common-dir>/gentle-ai/` | `internal/reviewtransaction/rar_path_safety.go`, `rdd_mode.go`, `store_reset.go` | A security invariant, not a location: `rar_path_safety.go` requires the exact `gentle-ai` path segment when it validates ownership and permissions of the RDD authority tree. Renaming the segment weakens or bypasses that containment check, and orphans every existing repository's authority store |
| Skill IDs and directories `gentle-ai-*` | `internal/model/types.go`, `skills/` | Installed verbatim into users' `.claude/skills/`; a rename orphans installed skills |
| Agent-side installed filenames | `~/.kiro/steering/gentle-ai.md`, `~/.cursor/rules/gentle-ai.mdc`, `Code/User/prompts/gentle-ai.instructions.md`, `~/.pi/gentle-ai/`, `~/.config/gentle-ai/`, `.gentle-ai-telemetry-runtime.json` | Already written to users' machines by previous installs; renaming leaves the old files behind and unmanaged |
| OpenCode agent key `gentle-orchestrator` | `internal/tui/screens/model_picker.go:204` | Persisted as a literal JSON key in users' `opencode.json` |
| Embedded agent assets and golden files | `internal/assets/**`, `testdata/golden/*.golden` | Operational instructions, not brand material; changing them changes runtime output and every golden comparison |
| Telemetry endpoint and schema ids | `internal/telemetry/telemetry.go`, `contracts/telemetry/**`, `deploy/telemetry/` | Live infrastructure and published schema `$id` URLs; requires a server-side migration |
| Telemetry collector binary `gentle-telemetry` | `cmd/gentle-telemetry/` | Deployed server-side unit, not the product CLI; renaming it is a deployment migration |
| Sibling projects `engram`, `gentle-pi`, `gentle-engram`, `gga` | `internal/update/registry.go:67`, `internal/agents/pi/adapter.go` | External packages and repositories this project does not own |
| Private Go identifiers (`GentleAIUpgradeVersion`, …) | `internal/tui/model.go` | No external risk, but out of scope for a branding-only change |
| Historical `PRD.md`, `PRD-AGENT-BUILDER.md` | repository root | Use the older "Gentleman AI" name; historical artifacts |

## Known limitations

- The shipped CLI is still the inherited Gentle AI runtime. Atomwright's atomic-delivery workflow is
  not implemented.
- The CLI-identifier clause of `TRADEMARKS.md` is now satisfied: the invocation token is
  `atomwright` and no `gentle-ai` executable is produced. The remaining inherited identifiers listed
  above are protocol, on-disk, and asset identifiers rather than project or CLI identifiers.
- Inherited Gentle AI product prose remains throughout `docs/**`. The installation, invocation,
  state-path, and environment-variable surfaces have been brought up to date; the surrounding
  product naming has not, and is tracked as separate work.
- Atomwright has published no releases of its own yet. `go install ...@latest` and the install
  scripts resolve against the coordinates recorded above; a version pin is the reproducible form
  until an Atomwright release exists.
