// Package logging is the cross-cutting platform logger defined by
// ADR-0001. It was scaffolded empty by ATOM-BOOT-003 and is owned by #66
// ATOM-BOOT-006, the issue that filled it in when the composition root
// needed a logger to wire.
//
// Like every platform package it must never import internal/domain, and it
// is restricted to the standard library. It wraps log/slog rather than
// replacing it: New returns a *slog.Logger, so callers keep the standard
// logging vocabulary and nothing in the tree depends on a bespoke logging
// interface.
package logging
