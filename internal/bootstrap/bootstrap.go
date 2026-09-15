package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/getsyntegrity/atomwright/platform/logging"
)

// statusLine is what a run of the binary prints today. The tree is a
// skeleton, so the honest thing for the composition root to report is that
// it wired successfully and that no functional command exists yet.
const statusLine = "atomwright: composition root is wired; no functional command is available yet (see docs/adr/0001-modular-monolith-skeleton.md)"

// Config is the process-level input the composition root needs. Every
// field is supplied by the caller rather than read from the environment
// here, which is what makes App constructible in a test without touching
// the real process.
type Config struct {
	// Stdout carries program output.
	Stdout io.Writer
	// Stderr carries diagnostics, including every log record.
	Stderr io.Writer
	// LogLevel is the minimum level the logger emits. The zero value is
	// slog.LevelInfo.
	LogLevel slog.Level
}

// App is the wired application. It owns the constructed collaborators and
// exposes the single entry point every cmd/* surface calls.
type App struct {
	logger *slog.Logger
	stdout io.Writer
}

// New wires the application from cfg using plain constructor calls -- no
// reflection-based container, per ADR-0001. It returns an error rather
// than defaulting a missing writer, so a miswired caller fails at
// construction instead of silently discarding output.
func New(cfg Config) (*App, error) {
	if cfg.Stdout == nil {
		return nil, errors.New("bootstrap: Config.Stdout is required")
	}
	if cfg.Stderr == nil {
		return nil, errors.New("bootstrap: Config.Stderr is required")
	}

	return &App{
		logger: logging.New(cfg.Stderr, cfg.LogLevel),
		stdout: cfg.Stdout,
	}, nil
}

// Run executes the application until it completes or ctx is cancelled.
func (a *App) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	a.logger.DebugContext(ctx, "composition root started")

	if _, err := fmt.Fprintln(a.stdout, statusLine); err != nil {
		return fmt.Errorf("bootstrap: writing status line: %w", err)
	}

	a.logger.DebugContext(ctx, "composition root finished")
	return nil
}
