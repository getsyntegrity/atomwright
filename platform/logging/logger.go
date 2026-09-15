package logging

import (
	"io"
	"log/slog"
)

// New returns the process logger: a text-handler *slog.Logger that writes
// records at or above level to w.
//
// It is deliberately thin. ADR-0001 places logging in platform/* as
// cross-cutting infrastructure, and the cheapest way to keep it
// cross-cutting is to hand back the standard library's own logger type
// instead of an Atomwright-specific interface every layer would then have
// to learn. Anything richer -- a handler per environment, routing,
// sampling -- lands with the functional epic that needs it, not before.
func New(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}
