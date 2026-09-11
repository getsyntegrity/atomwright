# Backup & Rollback Guide

The backup system automatically snapshots your configuration files before every install, sync, and upgrade. Backups are compressed, deduplicated, and automatically pruned to keep disk usage under control.

## How it works

Every time you run `atomwright install`, `sync`, or `upgrade`, the system:

1. **Computes a checksum** of all files that will be backed up
2. **Skips the backup** if it would be identical to the most recent one (dedup)
3. **Creates a compressed snapshot** (`snapshot.tar.gz`) with all your config files
4. **Prunes old backups** — keeps the 5 most recent, deletes the rest

## Snapshot contents

- `manifest.json` — metadata (source, timestamp, file count, checksum, pin status)
- `snapshot.tar.gz` — compressed archive of all backed-up files
- For paths that did not exist before the operation, the manifest tracks `existed=false`

> **Backup scope**: pre-upgrade and pre-sync snapshots cover only the agents listed in `state.InstalledAgents` (`~/.atomwright/state.json`). Config directories for agents you installed outside of atomwright are not included in the snapshot.

Legacy (pre-v1.16) backups use a `files/` directory with plain copies instead of a tar.gz archive. Both formats are fully supported for restore.

> **Backups are not moved by the `~/.gentle-ai/` → `~/.atomwright/` migration.**
> Each `manifest.json` embeds an absolute `root_dir`, and restore validates that
> every entry stays contained under it as an anti-tamper check. A manifest
> copied to a different root would be rejected at restore time, so snapshots
> created before the rename stay in `~/.gentle-ai/backups/` and remain readable
> and restorable there. New snapshots are written to `~/.atomwright/backups/`.
> See [Migrating from `~/.gentle-ai/`](quickstart.md#migrating-from-gentle-ai).

## Retention policy

| Setting | Default | Behavior |
|---------|---------|----------|
| Keep count | 5 | The 5 most recent unpinned backups are kept |
| Pinned backups | Never deleted | Survive pruning regardless of count |
| Duplicates | Skipped | If config hasn't changed, no new backup is created |
| Compression | Always | New backups use tar.gz (~75% smaller) |

## Pinning backups

You can mark any backup as "pinned" in the TUI to protect it from automatic pruning:

1. Run `atomwright` and navigate to the **Backups** screen
2. Use `j`/`k` to select a backup
3. Press **`p`** to toggle pin/unpin
4. Pinned backups show a `[pinned]` indicator

Pinned backups are never automatically deleted, even when the retention limit is exceeded.

## Managing backups (TUI)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | Restore selected backup |
| `p` | Pin/unpin (protect from pruning) |
| `r` | Rename (add a description) |
| `d` | Delete |
| `Esc` | Back |

## Restore behavior

- If `existed=true`: restores the file from the snapshot to its original path
- If `existed=false`: removes the file (reverting files created during install)
- Restore is atomic per file write — no partial restores
- Works with both compressed (tar.gz) and legacy (plain file) backups

## If verification fails

1. Review failed checks in verification report
2. Restore from latest snapshot via the TUI or `atomwright restore latest`
3. Re-run install with `--dry-run` to validate plan
4. Re-run install after fixing external dependencies

## What rollback does NOT cover

- Prerequisite packages installed via `brew install`, `apt-get install`, or `pacman -S` are not uninstalled during rollback. The snapshot system handles configuration files only.
- If you need to undo a package install, use your platform's package manager directly (e.g., `brew uninstall`, `sudo apt-get remove`, `sudo pacman -R`).
- The `atomwright` binary itself is not rolled back. Reinstall a specific version with `go install github.com/pablogore/atomwright/v2/cmd/atomwright@vX.Y.Z`.
