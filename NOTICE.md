# Notice

## Independent fork

Atomwright is an independent fork of [Gentle AI](https://github.com/Gentleman-Programming/gentle-ai),
an open-source project by Gentleman Programming.

Atomwright is **not** official, sponsored, endorsed, certified, partnered with, or affiliated with
Gentle AI, Gentleman Programming, or the maintainers of any integration referenced in this
repository.

## Trademarks

Gentle AI, Gentle-AI, gentle-ai, Engram, and their associated logos belong to their respective owner,
as described in [`TRADEMARKS.md`](TRADEMARKS.md).

That policy is reproduced in this repository **unchanged and byte-for-byte**, so that it travels with
the code it governs. Nothing in this notice modifies, reinterprets, amends, or narrows it. Where this
notice and `TRADEMARKS.md` appear to differ, `TRADEMARKS.md` governs.

References to Gentle AI in this repository are nominative: they state truthfully what Atomwright is
derived from, and are not a claim of endorsement or affiliation.

## Inherited technical identifiers

Some technical identifiers inherited from Gentle AI remain in this repository **temporarily, for
compatibility**. They include, among others:

- injected configuration markers of the form `<!-- gentle-ai:... -->`, which are already written
  into users' files and cannot be renamed without orphaning them
- `gentle-ai.<name>/v<N>` protocol identifiers, which are negotiated wire contracts
- the `gentle-ai` path component of the review authority store inside a repository's Git directory
- the `gentle-ai-review-provider-contract-<semver>` release bundle name, which third-party review
  providers resolve by name
- `GENTLE_AI_*` spellings of runtime environment variables, still read as deprecated aliases
- `~/.gentle-ai/`, read only to migrate compatible state and never written to
- `gentle-ai-*` skill identifiers and agent-side filenames installed on disk under those names

Retaining them is a compatibility measure, not a claim of any right to them, and it is not a
statement about how the upstream trademark policy applies to them. `TRADEMARKS.md` states that
policy; this notice does not.

The public command name, the Go module path, the primary configuration directory and the primary
environment prefix are no longer inherited: they are `atomwright`, `github.com/pablogore/atomwright/v2`,
`~/.atomwright/` and `ATOMWRIGHT_`. **No executable or release asset named `gentle-ai` is
distributed.**

**The identifiers listed above must still be migrated before they stop being compatibility
surfaces.** [`docs/atomwright/upstream-baseline.md`](docs/atomwright/upstream-baseline.md) records
each one, with the reason it is retained.

## License

Atomwright is distributed under the [MIT License](LICENSE), inherited from Gentle AI. The upstream
copyright notice is retained in full, as that license requires, alongside the Atomwright copyright
for subsequent modifications.

The MIT license grants copyright permissions only. It does not grant permission to use upstream
project names or logos.
