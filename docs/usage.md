# Usage

← [Back to README](../README.md)

---

## Persona Modes

| Persona   | ID          | Description                                                                       |
| --------- | ----------- | --------------------------------------------------------------------------------- |
| Gentleman | `gentleman` | Teaching-oriented mentor persona — pushes back on bad practices, explains the why |
| Neutral   | `neutral`   | Same teacher, same philosophy, no regional language — warm and professional       |
| Custom    | `custom`    | Keep your existing persona/config unmanaged — atomwright does not inject a persona |

`custom` is a compatibility/ownership choice, not a persona editor. Use it when you already have your own persona instructions and want atomwright to leave them alone.

---

## Interactive TUI

Just run it — the Bubbletea TUI guides you through agent selection, components, skills, presets, and managed uninstall flows:

```bash
atomwright
```

The uninstall flow is also available from the TUI menu. It lets you:

- select one or more configured agents
- select which managed components to remove (for example `sdd`, `persona`, or `context7`)
- confirm the exact uninstall scope before applying changes

Before any managed file is modified, `atomwright` creates a backup snapshot so the configuration can be restored later if needed.

### Receipt-Driven Development during installation

Before the final installation confirmation, the customizable installer explains Receipt-Driven Development (RDD) and asks you to choose **RDD ON** or **RDD OFF**. RDD records bounded, independent review evidence for a frozen change candidate and supports a bounded correction process. It can add review time and model cost.

The choice is optional and defaults to OFF when no global preference exists. You can return from the confirmation screen to revise it. Gentle AI saves the selected global setting only after installation succeeds; an interrupted or failed installation leaves it unchanged. Existing clone-local overrides remain unchanged, so a clone's effective mode can differ from the global setting. RDD evidence does not authorize commits, pushes, pull requests, or releases; ordinary repository policy still governs delivery.

### Disable TUI spinner animation

Set `ATOMWRIGHT_NO_ANIMATION=1` to keep TUI spinner frames static:

```bash
ATOMWRIGHT_NO_ANIMATION=1 atomwright
```

This disables only spinner animation; install, update, sync, and uninstall operations continue normally. Unset the variable, or use any value other than `1`, to keep the default animation behavior.

---

## CLI Commands

### install

First-time setup — detects your tools, configures agents, injects all components. When installing a single agent with `--agent X`, atomwright **merges** the new agent into the existing `installed_agents` list in `state.json` and **preserves** any existing `model_assignments` — it does not overwrite the full state.

```bash
# Full ecosystem for multiple agents
atomwright install \
  --agent claude-code,opencode,gemini-cli \
  --preset full-gentleman

# Minimal setup for Cursor
atomwright install \
  --agent cursor \
  --preset minimal

# OpenClaw setup after installing OpenClaw manually
atomwright install \
  --agent openclaw \
  --preset full-gentleman

# Pick specific components and skills
atomwright install \
  --agent claude-code \
  --component engram,sdd,skills,context7,persona,permissions \
  --skill go-testing,skill-creator,branch-pr,issue-creation \
  --persona gentleman

# Dry-run first (preview plan without applying changes)
atomwright install --dry-run \
  --agent claude-code,opencode \
  --preset full-gentleman
```

### skill-registry refresh

Refresh the project-local skill registry used by orchestrators before they delegate work:

```bash
atomwright skill-registry refresh
atomwright skill-registry refresh --force
atomwright skill-registry refresh --cwd /path/to/project --quiet
```

The command scans project skills first (`skills/`, `.opencode/skills/`, `.claude/skills/`, `.github/skills/`, and other supported workspace skill roots), then global agent skill directories. Project-local skills win over same-name global skills.

The command writes `.atl/skill-registry.md` and `.atl/.skill-registry.cache.json`. The cache fingerprint includes schema version plus each discovered `SKILL.md` file path, mtime, and size, so normal startup is a cheap cache-hit when skills have not changed.

Codex, Claude Code, and OpenCode installs wire this command into startup/plugin hooks. Pi gets the equivalent behavior from `gentle-pi`; keep those hook/plugin scan roots in sync when changing these discovery rules.

See [Skill Registry](skill-registry.md) for the full index-first flow and diagrams.

### sync

Refresh managed assets to the current version. Run it after replacing or upgrading the `atomwright` binary, including with `atomwright upgrade` or `go install`. It does NOT reinstall binaries (engram, GGA) — only updates prompt content, skills, MCP configs, and SDD orchestrators.

Managed reviewer and runtime assets are version-bound to the binary. Until sync succeeds, review lifecycle operations fail closed when managed writer provenance is missing or mismatched.

> **Important:** `atomwright sync` updates the agents recorded as installed by Gentle AI™, not every AI agent config directory on your machine.
>
> Gentle AI stores your selected install targets in `~/.atomwright/state.json`. Future `sync` runs use that stored selection so Gentle AI does not accidentally write into tools you did not choose to manage. If you rerun install and select only one agent, that new selection becomes the default sync scope.
>
> Before syncing, you can preview the active scope with `atomwright sync --dry-run`. If you want to sync agents outside the stored selection, pass them explicitly with `--agent`.

```bash
# Preview which agents sync will update
atomwright sync --dry-run

# Sync the agents currently registered in ~/.atomwright/state.json
atomwright sync

# Sync specific agents only
atomwright sync --agent claude-code --agent opencode

# Refresh OpenClaw workspace instructions and MCP config
atomwright sync --agent openclaw
```

Sync is safe and idempotent — running it twice produces no changes the second time. When files change, the summary reports the changed file count and lists the changed file paths.

`sync` refreshes the managed component set for the selected agents. It does not support `--component`; use `--include-permissions` or `--include-theme` for the opt-in components that are excluded from the default sync scope.

After upgrading the binary, `atomwright sync --dry-run` previews the selected targets; `atomwright sync` refreshes their primary remote-authorization guidance. To select a specific managed client, use e.g. `atomwright sync --agent opencode`. This behavioral section is delivered with unconditional routing guidance, without requiring persona, SDD, or `--include-permissions`. It requires explicit destination, operation, and credential/session authorization before remote work or ambient access discovery/reuse.

This update covers the 15 non-Pi primary instruction carriers only. Executor roles, named profiles, and Pi's package-owned instructions require separate behavioral coverage. Existing automation modes and remembered approvals may suppress runtime prompts. The guidance is not a sandbox or a fresh-human-per-execution guarantee. Shared settings merging and profile cleanup preserve existing permission-rule order.

For OpenCode native remote-command asks, opt in separately: `atomwright sync --agent opencode --include-permissions` (or `--agent kilocode` for the shared generated configuration). Defaults ask for direct `ssh`, `scp`, `sftp`, and `rsync`, bare or with arguments; local-only rsync also asks conservatively. Existing restrictions and explicit custom allows remain authoritative, so custom configurations may still allow remote commands. Defaults do not rewrite those personal allows. Agent overrides and remembered approvals may also bypass a prompt.

Matcher fixtures follow OpenCode [v1.2.27 wildcard matching](https://github.com/anomalyco/opencode/blob/v1.2.27/packages/opencode/src/util/wildcard.ts) and its last-matching permission evaluation. They do not prove interception of absolute executable paths, env wrappers, interpreters, or arbitrary compound shell syntax; the runtime extracts command nodes separately. Kilocode runtime equivalence is not verified. Issue #4324 remains open for all-client/all-role completion.

For OpenClaw, sync reads the active workspace from `~/.openclaw/openclaw.json` (`agents.defaults.workspace`). It writes `AGENTS.md` / `SOUL.md` into that workspace, while MCP servers stay in the global OpenClaw config under `mcp.servers`.

For Hermes, atomwright is detect-only: it cannot install Hermes. Install Hermes manually first. Detection is driven by the `~/.hermes` config directory (the binary being on `PATH` is reported separately). Once Hermes is detected, `atomwright install --agent hermes` injects context7 and Engram™ MCP blocks into `~/.hermes/config.yaml`, writes the SDD orchestrator and persona into `~/.hermes/SOUL.md`, and copies skills to `~/.hermes/skills/`. Use `atomwright sync --agent hermes` to update the managed configuration after upgrades.

### uninstall

Remove only the `atomwright` managed configuration from one or more agents. This does not uninstall external packages or binaries — it removes managed prompt sections, MCP entries, skills/config fragments, and other managed files, then updates `state.json` accordingly.

Before any change is applied, `atomwright` creates a backup snapshot of the affected files.

```bash
# Partial uninstall for specific agents
atomwright uninstall \
  --agent claude-code \
  --agent opencode

# Partial uninstall for specific components only
atomwright uninstall \
  --agent claude-code \
  --component sdd,persona,context7

# Complete uninstall of managed config from all supported agents
atomwright uninstall --all

# Skip confirmation prompt
atomwright uninstall --agent cursor --component skills --yes
```

If no `--component` flag is provided for a partial uninstall, `atomwright` removes all managed uninstallable components for the selected agent set.

### update / upgrade

Check for and install new versions of `atomwright` itself. The pre-upgrade backup snapshot covers only the agents recorded in `state.InstalledAgents` (`~/.atomwright/state.json`) — not every agent config directory that exists on your machine.

```bash
# Check if a newer version is available
atomwright update

# Upgrade to the latest release (downloads new binary, replaces current)
atomwright upgrade
```

After any upgrade or manual binary replacement, run `atomwright sync` to refresh all managed assets to the new version's content.

If GitHub rate-limits update checks, export `GITHUB_TOKEN` or `GH_TOKEN` before running `atomwright update`/`upgrade`.

Atomwright is not published through Homebrew. There is no formula, cask, or
tap, so nothing here needs `brew trust`. Upgrade with `atomwright upgrade`, or
reinstall with the install script or `go install` — see
[Quickstart](quickstart.md#install).

**Self-update prompt behavior** (changed in v1.x slice 5 — `GENTLE_AI_CONFIRM_UPDATE` removed):

| Situation | Behavior |
|-----------|----------|
| Interactive terminal (TTY) | Always prompts `Apply now? [Y/n]`. Empty Enter accepts. |
| Non-TTY (CI, pipe, script) | Auto-declines — never hangs. |
| `ATOMWRIGHT_YES=1` | Auto-accepts without prompting (for scripted upgrades). This variable is inherited by subprocesses, so scope it to a single invocation when needed (e.g. `ATOMWRIGHT_YES=1 atomwright …`). |
| `ATOMWRIGHT_NO_SELF_UPDATE=1` | Skips the self-update check entirely. |

`GENTLE_AI_CONFIRM_UPDATE` was removed in slice 5. It is now ignored if set.

`ATOMWRIGHT_SELF_UPDATE_DONE` is an internal loop guard and should not be set manually.

### model assignment

The TUI **Configure Models** screen can assign different models to SDD phases, `sdd-onboard`, and Judgment Day agents (`jd-judge-a`, `jd-judge-b`, `jd-fix-agent`) when the selected agent supports those slots. This lets you keep review or apply phases on stronger models while routing cheaper phases to faster models.

### doctor

Read-only ecosystem health diagnostics — no changes made to your configuration:

```bash
atomwright doctor
```

Checks performed:

| Check | What it verifies |
|-------|-----------------|
| Tool binaries | Required tools present on `PATH`; shadow detection (wrong binary resolves first) |
| `state.json` validity | Parses `~/.atomwright/state.json` and reports any schema/corruption issues |
| Engram MCP reachability | Confirms the Engram MCP server responds |
| Disk space | Warns when available space is critically low |

Each check reports **pass**, **warn**, or **fail** with an optional remedy hint. Run `doctor` first when troubleshooting an unexpected install or sync result.

### version

```bash
atomwright version
atomwright --version
atomwright -v
```

---

## CLI Flags (install)

| Flag                          | Description                                                                                                       |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `--agent`, `--agents`         | Agents to configure (comma-separated)                                                                             |
| `--component`, `--components` | Components to install (comma-separated)                                                                           |
| `--skill`, `--skills`         | Skills to install (comma-separated)                                                                               |
| `--persona`                   | Persona mode: `gentleman`, `neutral`, `custom` (`custom` keeps your existing persona unmanaged)                   |
| `--preset`                    | Preset: `full-gentleman`, `ecosystem-only`, `minimal`, `custom` (`custom` means manual component/skill selection) |
| `--sdd-mode`                  | SDD orchestrator mode: `single` or `multi`                                                                        |
| `--scope`                     | Install scope for agent-scoped files: `global` (default, writes to each selected agent's global config directory) or `workspace` (writes to the current project root). Also settable via `ATOMWRIGHT_INSTALL_SCOPE` env var for CI/non-interactive use. |
| `--dry-run`                   | Preview the install plan without applying changes                                                                 |

## CLI Flags (sync)

| Flag                     | Description                                                                                          |
| ------------------------ | ---------------------------------------------------------------------------------------------------- |
| `--agent`, `--agents`    | Agents to sync (defaults to all installed agents)                                                    |
| `--skill`, `--skills`    | Skills to sync (comma-separated; defaults to selected preset skills)                                  |
| `--sdd-mode`             | SDD orchestrator mode: `single` or `multi`                                                           |
| `--strict-tdd`           | Enable Strict TDD Mode for SDD agents                                                                |
| `--profile`              | Create or update an SDD profile: `name:provider/model` (sets the default model for all phases)       |
| `--profile-phase`        | Override a specific phase in a profile: `name:phase:provider/model`                                  |
| `--sdd-profile-strategy` | OpenCode profile sync strategy: `generated-multi` or `external-single-active`                        |
| `--include-permissions`  | Include permissions sync (opt-in)                                                                    |
| `--include-theme`        | Include theme sync (opt-in)                                                                          |
| `--dry-run`              | Preview the sync plan without applying changes                                                       |

**Profile examples:**

```bash
# Create a "cheap" profile using a free model for all phases
atomwright sync --profile cheap:openrouter/qwen/qwen3-30b-a3b:free

# Override the design phase to use a stronger model
atomwright sync --profile-phase cheap:sdd-design:anthropic/claude-sonnet-4-20250514

# Create multiple profiles in one command
atomwright sync \
  --profile cheap:openrouter/qwen/qwen3-30b-a3b:free \
  --profile premium:anthropic/claude-sonnet-4-20250514

# Use compatibility mode with an external OpenCode profile manager
atomwright sync --agent opencode --sdd-profile-strategy external-single-active
```

See [OpenCode SDD Profiles](opencode-profiles.md) for the full guide.

## CLI Flags (uninstall)

| Flag                          | Description                                                             |
| ----------------------------- | ----------------------------------------------------------------------- |
| `--agent`, `--agents`         | Agents to uninstall managed config from (required unless using `--all`) |
| `--component`, `--components` | Managed components to remove only from the selected agents              |
| `--all`                       | Remove managed configuration from all supported agents                  |
| `--yes`, `-y`                 | Skip the confirmation prompt                                            |

---

## Environment Variables

Atomwright's environment variables use the `ATOMWRIGHT_` prefix. The inherited
`GENTLE_AI_` prefix is still read as a deprecated alias, because these variables
live in user dotfiles and CI configuration this project cannot edit.

**Precedence:** when both are set, `ATOMWRIGHT_*` wins. The `GENTLE_AI_*`
variable is still read when the `ATOMWRIGHT_*` one is unset — including when it
is set to the empty string, which is treated as a deliberate choice rather than
"unset". Whenever a `GENTLE_AI_*` alias is present, Atomwright prints one
deprecation warning per variable at startup. The warning names both variable
names and **never** their values: these variables can carry tokens and are
captured verbatim in CI logs.

| Current name | Deprecated alias | Controls |
| --- | --- | --- |
| `ATOMWRIGHT_CHANNEL` | `GENTLE_AI_CHANNEL` | Release channel for install/upgrade: `stable` (default), `beta`, or `nightly` (an alias for beta). |
| `ATOMWRIGHT_CODEX_REVIEWER_LOOPBACK_BASE_URL` | `GENTLE_AI_CODEX_REVIEWER_LOOPBACK_BASE_URL` | Base URL of a loopback model provider for the Codex reviewer adapter. Unset means the adapter uses Codex's own provider configuration. |
| `ATOMWRIGHT_ENGRAM_SETUP_MODE` | `GENTLE_AI_ENGRAM_SETUP_MODE` | Which agents Engram™ MCP setup runs for: `supported` (default), `opencode`, or `off`. |
| `ATOMWRIGHT_ENGRAM_SETUP_STRICT` | `GENTLE_AI_ENGRAM_SETUP_STRICT` | Fail the install when Engram™ setup fails instead of degrading. Truthy values: `1`, `true`, `yes`, `on`. |
| `ATOMWRIGHT_INSTALL_SCOPE` | `GENTLE_AI_INSTALL_SCOPE` | Install scope for agent-scoped files: `global` (default) or `workspace`. The non-interactive equivalent of `--scope`. |
| `ATOMWRIGHT_NO_ANIMATION` | `GENTLE_AI_NO_ANIMATION` | Set to `1` to keep TUI spinner frames static. |
| `ATOMWRIGHT_NO_SELF_UPDATE` | `GENTLE_AI_NO_SELF_UPDATE` | Set to `1` to skip the self-update check entirely. |
| `ATOMWRIGHT_OPENCODE_BACKGROUND_SUBAGENTS` | `GENTLE_AI_OPENCODE_BACKGROUND_SUBAGENTS` | Managed OpenCode background-subagent policy: `auto`, `on`, or `off`. The non-interactive equivalent of `--opencode-background-subagents`. |
| `ATOMWRIGHT_PI_BACKGROUND_SUBAGENTS` | `GENTLE_AI_PI_BACKGROUND_SUBAGENTS` | Managed Pi background-subagent policy: `auto`, `on`, or `off`. The non-interactive equivalent of `--pi-background-subagents`. |
| `ATOMWRIGHT_SDD_STATUS_ENGRAM` | `GENTLE_AI_SDD_STATUS_ENGRAM` | Set to any non-empty value to let SDD status consult Engram™ in a workspace that has no `.engram` directory. An explicit `openspec/config.yaml` declaration still wins. |
| `ATOMWRIGHT_SELF_UPDATE_DONE` | `GENTLE_AI_SELF_UPDATE_DONE` | Internal self-update loop guard (`1`). Do not set it manually. |
| `ATOMWRIGHT_TELEMETRY` | `GENTLE_AI_TELEMETRY` | Set to `0` to opt out of telemetry for the run. See [Telemetry](telemetry.md). |
| `ATOMWRIGHT_TELEMETRY_ENDPOINT` | `GENTLE_AI_TELEMETRY_ENDPOINT` | Override the telemetry collector URL (self-hosting or local testing). |
| `ATOMWRIGHT_YES` | `GENTLE_AI_YES` | Set to `1` to auto-accept the self-update prompt in scripted upgrades. Inherited by subprocesses, so scope it to a single invocation. |

That table is the complete alias roster. Names that merely share the
`GENTLE_AI_` spelling but are **not** environment variables are deliberately not
aliased, because changing them would change a wire format rather than a user's
configuration:

- `GENTLE_AI_REVIEW_*` reviewer prompt markers (for example the
  `GENTLE_AI_REVIEW_BINDING ` prefix),
- `{{GENTLE_AI_*}}` template placeholders,
- the `GENTLE_AI_TELEMETRY` **value** pinned as an enum in the published
  telemetry contract — distinct from the `ATOMWRIGHT_TELEMETRY` variable above.

`GENTLE_AI_CONFIRM_UPDATE` is not in the table either: it was removed in v1.x
slice 5 and is ignored if set. There is no `ATOMWRIGHT_CONFIRM_UPDATE`.

---

## Typical Workflow

```bash
# First time: install the binary, then install everything
curl -sL https://raw.githubusercontent.com/pablogore/atomwright/main/scripts/install.sh | bash
atomwright install --agent claude-code,cursor --preset full-gentleman

# After a new release: upgrade + sync
atomwright upgrade
atomwright sync

# Remove only managed SDD + persona config from one agent
atomwright uninstall --agent claude-code --component sdd,persona

# Adding a new agent later
atomwright install --agent windsurf --preset full-gentleman
```

---

## Dependency Management

`atomwright` auto-detects prerequisites before installation and provides platform-specific guidance:

- **Detected tools**: git, curl, node, npm, brew, go
- **Version checks**: validates minimum versions where applicable
- **Platform-aware hints**: suggests `brew install`, `apt install`, `pacman -S`, `dnf install`, or `winget install` depending on your OS
- **Node LTS alignment**: on apt/dnf systems, Node.js hints use NodeSource LTS bootstrap before package install
- **Dependency-first approach**: detects what's installed, calculates what's needed, shows the full dependency tree before installing anything, then verifies each dependency after installation
