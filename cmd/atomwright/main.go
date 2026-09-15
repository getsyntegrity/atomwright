// Command atomwright is Atomwright's single composition-root binary.
//
// ADR-0001 keeps this file thin on purpose: it handles process-level
// concerns only -- signals, streams, exit codes -- and hands everything
// else to internal/bootstrap. It imports internal/bootstrap and the
// standard library, and nothing else from the module; compositionRootRule
// in internal/architecture fails the build if that ever stops being true.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/getsyntegrity/atomwright/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "atomwright: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	app, err := bootstrap.New(bootstrap.Config{
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		LogLevel: slog.LevelInfo,
	})
	if err != nil {
		return err
	}

	return app.Run(ctx)
}
