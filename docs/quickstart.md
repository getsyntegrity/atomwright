# Quickstart

## Prerequisites

### macOS

- `git` available.
- `curl` available (pre-installed), for the install script.
- Go 1.25.10+ if you prefer to install from source.

### Ubuntu/Debian (and derivatives like Linux Mint, Pop!\_OS)

- `apt-get` available (standard on these distros).
- `sudo` access for package installs.
- `git` available.
- If Node.js is missing, `atomwright install` prints this install hint: NodeSource LTS setup + `apt-get install -y nodejs` (npm comes bundled).

### Arch Linux (and derivatives like Manjaro, EndeavourOS)

- `pacman` available (standard on these distros).
- `sudo` access for package installs.
- `git` available.
- If Node.js is missing, `atomwright install` prints this install hint: `pacman -S --noconfirm nodejs npm`.

### Fedora / RHEL family (Fedora, CentOS Stream, Rocky Linux, AlmaLinux)

- `dnf` available (standard on these distros).
- `sudo` access for package installs.
- `git` available.
- If Node.js is missing, `atomwright install` prints this install hint: NodeSource LTS setup + `dnf install -y nodejs` (npm comes bundled).

### All platforms

- Git 2.38+.
- Go 1.25.10+ (for building from source).
- Node.js 18+ and npm: `atomwright install` checks these as required prerequisites on every platform and prints a warning with a distro-specific install hint (see above) if either is missing — regardless of which agents/components you select. It does not install them for you, and it does not install agent runtimes either: if a selected agent isn't detected, `atomwright install` refuses and prints the exact `npm install -g` (or equivalent) command for you to run yourself. Node.js/npm are strictly required if you select the CodeGraph community tool, which atomwright does install via `npm install -g`.
- Pi installed and available as `pi` on `PATH` if you select the Pi agent.

### Windows

- Go 1.25.10+, because Windows installs and upgrades through `go install`.
  Official Windows binaries and the Scoop bucket are temporarily unavailable
  while publicly trusted Authenticode signing is provisioned, so nothing
  unsigned is ever fetched. With Go on `PATH`, `atomwright upgrade` updates
  itself automatically by running `go install …/cmd/atomwright@vX.Y.Z` pinned to
  the release tag and verified against the Go checksum database; without Go it
  fails closed and just prints that command. See [platforms.md](platforms.md)
  and the
  [restoration gate](release-signing.md#windows-distribution-restoration-gate).

```powershell
# Stable channel (`@latest`, currently v2.6.0)
go install github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright@latest
```

This command uses the `/v2` module path. Go requires that suffix for major
version 2 and above.

## Install

Atomwright has exactly two supported install paths: the curl installer and
`go install`. There is no Homebrew formula, cask, or tap — Homebrew publication
was removed from the release path because no Atomwright-owned tap exists.
Homebrew may still be present on your machine as the *package manager* the
installer uses to satisfy prerequisites such as `git`, `curl`, or `node`; that
is unrelated to how Atomwright itself is distributed.

### Install script (macOS / Linux)

```bash
curl -sL https://raw.githubusercontent.com/pablogore/atomwright/main/scripts/install.sh | bash
atomwright version
```

### Install script (Windows, PowerShell)

```powershell
irm https://raw.githubusercontent.com/pablogore/atomwright/main/scripts/install.ps1 | iex
atomwright version
```

### From source

```bash
go install github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright@latest
atomwright version
```

> **The Go module path was not renamed.** The command is `atomwright`, the
> release repository is `pablogore/atomwright`, and the state directory is
> `~/.atomwright/` — but the module path is still
> `github.com/gentleman-programming/gentle-ai/v2`. Renaming a module path
> breaks every existing import and source install, so it is a separate future
> migration. Type the package path above literally: only the final path element
> (`cmd/atomwright`) carries the new name.

## Migrating from `~/.gentle-ai/`

Atomwright reads its own state from `~/.atomwright/`. Installs that predate the
rename wrote to `~/.gentle-ai/`. Every `install`, `sync`, and `upgrade` run
inspects both directories and migrates automatically when it is safe to do so.

> ### ⚠️ ACTION REQUIRED: update your shell profile
>
> **If `~/.gentle-ai/bin` is on your `PATH`, you must repoint it at
> `~/.atomwright/bin` yourself.** Atomwright never edits shell profiles, so the
> `PATH` entry your old install told you to add still points at the old
> directory — but the OpenCode managed launchers are now written under
> `~/.atomwright/bin`. Until you edit `~/.zshrc`, `~/.bashrc`, or your
> PowerShell profile, **OpenCode will not launch**.
>
> Atomwright prints this warning on every run while `~/.gentle-ai/bin` still
> exists, because the breakage lasts until you make the edit. Nothing else in
> the migration requires manual work.

### What happens on startup

| `~/.gentle-ai/` | `~/.atomwright/` | Outcome |
| --- | --- | --- |
| absent | absent | **Fresh install.** Nothing to migrate. |
| present | absent | **Migration.** The portable files are copied across and the result is reported. |
| absent | present | **Normal startup.** The new directory is already authoritative. |
| present | present | **Conflict.** Nothing is copied, merged, or removed. |

A conflict is never resolved by guessing. Atomwright prints both paths and stops
at the migration step: keep the directory you want, move or remove the other,
then run the command again. State is never merged across the two roots.

### What migrates

Exactly two files are relocated:

- `state.json` — your persisted install selections (agents, components, skills,
  persona, model assignments, RDD mode).
- `telemetry.json` — the local telemetry state and install identity.

Both were verified to contain no absolute paths, which is what makes copying
them to a different root safe. A file that already exists in `~/.atomwright/` is
never replaced by an older copy from the legacy root.

### What does not migrate

**`backups/` stays in `~/.gentle-ai/`.** Each `backups/<timestamp>/manifest.json`
embeds an absolute `root_dir`, and the backup code validates that every entry in
a manifest stays contained under that root. That check is an anti-tamper
containment guard against a crafted manifest deleting arbitrary paths — it is
not incidental. A manifest copied to a new root still points into the legacy
tree, so the copy would be rejected at restore time, leaving you with backups
that appear to exist but cannot be restored. Leaving them where they were
created keeps them readable and restorable from their original location.

Everything else in the legacy directory is regenerated on demand rather than
copied:

- caches (for example the model-variants cache),
- lock files,
- the OpenCode managed launchers under `bin/`,
- `pi-codegraph.json`.

Run `atomwright sync` after the migration and these are rewritten under
`~/.atomwright/`.

Per-repository review authority is unaffected: it lives inside each
repository's Git common directory, not in your home directory.

### Safety properties

- **The legacy directory is never deleted.** The migration is a copy, never a
  move, so an older build run from another machine against the same home
  directory still finds its state where it left it. Removing `~/.gentle-ai/` is
  always your decision.
- **It is idempotent.** Install, sync, and upgrade all run it, so it runs many
  times over the life of an installation. After the first successful run it is
  a no-op.
- **It is safe to interrupt.** Every file is published by an atomic
  write-then-rename, and the completion marker is written last, after the copies
  are durable. An interrupted run never records itself as complete, and the next
  run resumes it — as long as what is in the new root is only byte-identical
  copies of what the migration itself writes. Anything else means the new root
  has a history of its own, and that is reported as a conflict instead.

## Version Policy

Receipt-Driven Development (RDD) began in `v1.47.0` on 2026-07-10, and `v2.2.0` made it the supported stable path. Those are historical milestones. The negotiated public review contract was published in `v2.1.6`.

The current stable release is [`v2.6.0`](https://github.com/Gentleman-Programming/gentle-ai/releases/tag/v2.6.0). `@latest` explicitly tracks this stable channel. No prerelease is ahead of stable. `@main` installs unreleased development changes.

### Install the stable channel

```bash
go install github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright@latest
atomwright version
```

### Install unreleased development changes

Only use `main` when testing changes that are not part of a release yet:

```bash
# macOS / Linux
go install github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright@main
atomwright version

# Windows (PowerShell)
$env:ATOMWRIGHT_CHANNEL="beta"; go install github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright@main
atomwright version
```

To update a beta installation later, preserve the beta channel:

```bash
# macOS / Linux
ATOMWRIGHT_CHANNEL=beta atomwright upgrade

# Windows (PowerShell)
$env:ATOMWRIGHT_CHANNEL="beta"; atomwright upgrade
```

`atomwright upgrade` advances the `atomwright` binary from `main` and refreshes managed tools on macOS, Linux, and Windows with Go on `PATH`.

If you re-run an installer, pass beta explicitly because both installers default to stable:

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/pablogore/atomwright/main/scripts/install.sh | bash -s -- --channel beta

# Windows (PowerShell)
$env:ATOMWRIGHT_CHANNEL="beta"; irm https://raw.githubusercontent.com/pablogore/atomwright/main/scripts/install.ps1 | iex
```

> **Go module proxy cache**: `proxy.golang.org` can lag behind new commits on `main` for up to several hours. If manual `go install ...@main` does not update to the newest commit, bypass the cache with `GOPROXY=direct go install github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright@main` (PowerShell: `$env:GOPROXY="direct"; go install github.com/gentleman-programming/gentle-ai/v2/cmd/atomwright@main`).

The managed install scripts select the latest version for their chosen channel and do not accept arbitrary release pins. Use `go install` with an exact tag when you need a reproducible prerelease or stable version.

## Run

```bash
go run ./cmd/atomwright install --dry-run
```

Use `--dry-run` first to validate selections and execution plan without applying changes. The dry-run output includes a `Platform decision` line showing the detected OS, distro, package manager, and support status.

## First real install

```bash
go run ./cmd/atomwright install
```

The installer detects your platform automatically — no flags needed to select macOS vs Linux. Prerequisite install commands are resolved through the appropriate system package manager (brew, apt, pacman, or dnf) based on detection. Atomwright itself is never installed through a package manager.

After completion, verify that agent configs and selected components were installed to their expected paths.

The agents you select during install become the default scope for future `atomwright sync` runs. Gentle AI records that selection in `~/.atomwright/state.json` and does not automatically sync every agent config directory that exists on your machine. To check what will be updated after an upgrade, run:

```bash
atomwright sync --dry-run
```

To update a different set explicitly, pass every target agent:

```bash
atomwright sync --agent claude-code --agent opencode
```

## Verification outcome

When checks pass, installer reports:

`You're ready. Run 'claude' or 'opencode' and start building.`

If something looks wrong after install, run `atomwright doctor` for a read-only health check. It verifies tool binaries, `state.json` validity, Engram™ MCP reachability, and disk space — each check reports pass/warn/fail with a remedy hint.

For a Pi-only install, the plan shows the Pi package stack instead of Gentle AI components. It installs `gentle-pi`, `gentle-engram`, and `pi-mcp-adapter`, runs `pi-engram init` through the pinned `gentle-engram` package, then installs `@juicesharp/rpiv-ask-user-question`, `pi-web-access`, and `pi-btw`.

## Hardening recommendations for users

Gentle AI pins versions and disables postinstall scripts on every npm install it generates. When you install the `permissions` component, a sensitive-paths deny list is applied to Claude Code and OpenCode blocking access to `~/.ssh/*`, `**/*.pem`, `**/*.key`, `**/.env*`, `~/.aws/credentials`, and other credential paths. See [Components](../docs/components.md) for the full list.

For broader protection across npm packages you install yourself, set these once on your machine:

- `npm config set ignore-scripts true` — blocks postinstall scripts globally; the primary supply-chain attack vector.
- `npm config set min-release-age 3` — skip packages published in the last 3 days; catches malicious typosquats before you install them.
- `npm config set allow-git none` — block git: dependencies, which can be moving targets.

Optional wrapper tools for extra defense:

- [`npq`](https://github.com/lirantal/npq) — audits a package against several heuristics before it installs.
- [`sfw`](https://socket.dev/) (Socket Firewall) — runtime guard that intercepts suspicious behavior at install/run time.

## Unsupported platforms

If you run the installer on an unsupported OS or Linux distro, it exits immediately with an error:

- `unsupported operating system: only macOS, Linux, and Windows are supported (detected <os>)`
- `unsupported linux distro: Linux support is limited to Ubuntu/Debian, Arch, and Fedora/RHEL family (detected <distro>)`
