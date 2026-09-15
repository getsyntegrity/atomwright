package bootstrap_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/getsyntegrity/atomwright/internal/bootstrap"
)

func TestNewRejectsMissingWriters(t *testing.T) {
	tests := map[string]bootstrap.Config{
		"no stdout": {Stderr: io.Discard},
		"no stderr": {Stdout: io.Discard},
		"neither":   {},
	}

	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			app, err := bootstrap.New(cfg)
			if err == nil {
				t.Fatalf("New(%+v) succeeded; want an error", cfg)
			}
			if app != nil {
				t.Errorf("New returned a non-nil app alongside an error: %+v", app)
			}
		})
	}
}

func TestRunReportsTheWiredStateOnStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app, err := bootstrap.New(bootstrap.Config{Stdout: &stdout, Stderr: &stderr, LogLevel: slog.LevelInfo})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := app.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := stdout.String(); !strings.Contains(got, "atomwright") {
		t.Errorf("Run wrote no recognisable status line to stdout\ngot: %q", got)
	}
}

// Diagnostics belong on stderr so stdout stays usable as a data stream for
// the CLI surfaces the functional epics will add.
func TestRunKeepsDiagnosticsOffStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app, err := bootstrap.New(bootstrap.Config{Stdout: &stdout, Stderr: &stderr, LogLevel: slog.LevelDebug})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := app.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if stderr.Len() == 0 {
		t.Error("Run logged nothing at debug level; the composition root should be observable")
	}
	if got := stdout.String(); strings.Contains(got, "level=") {
		t.Errorf("log records leaked onto stdout\ngot: %q", got)
	}
}

func TestRunStopsOnACancelledContext(t *testing.T) {
	var stdout, stderr bytes.Buffer

	app, err := bootstrap.New(bootstrap.Config{Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := app.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run(cancelled ctx) = %v; want context.Canceled", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("Run wrote to stdout after cancellation\ngot: %q", stdout.String())
	}
}
