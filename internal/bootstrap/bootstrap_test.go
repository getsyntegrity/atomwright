// Package bootstrap_test exercises the composition root from outside the
// package, which is the only way to prove the exported surface alone is enough
// to wire an application and run it.
//
// The suite is written with go-specs. internal/bootstrap is the composition
// root, so its imports are unconstrained by ADR-0001 -- and even in a
// standard-library-only layer this import would be permitted, because it
// appears only in a _test.go file (ADR-0003).
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
	"github.com/pablogore/go-specs/specs"
)

// miswired is one Config the composition root must refuse, because a missing
// writer means output would be silently discarded.
type miswired struct {
	behaviour string
	cfg       bootstrap.Config
}

func miswiredConfigs() []miswired {
	return []miswired{
		{"refuses to wire an application with nowhere to write program output", bootstrap.Config{Stderr: io.Discard}},
		{"refuses to wire an application with nowhere to write diagnostics", bootstrap.Config{Stdout: io.Discard}},
		{"refuses to wire an application with no writers at all", bootstrap.Config{}},
	}
}

// wireAndRun wires the composition root onto fresh buffers and runs it against
// runCtx, so a spec can state what reached each stream and how the run ended.
// Wiring is a precondition here, not the behaviour under test, so a failure to
// wire stops the spec rather than being asserted on.
func wireAndRun(ctx *specs.Context, level slog.Level, runCtx context.Context) (stdout, stderr *bytes.Buffer, err error) {
	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}

	app, wireErr := bootstrap.New(bootstrap.Config{Stdout: stdout, Stderr: stderr, LogLevel: level})
	if wireErr != nil {
		ctx.T.Fatalf("wiring the composition root: %v", wireErr)
	}

	return stdout, stderr, app.Run(runCtx)
}

// cancelledContext is a context that is already done before the run begins.
func cancelledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func TestBootstrap(t *testing.T) {
	specs.Describe(t, "the composition root", func(s *specs.Spec) {
		s.When("a writer the application needs is missing", func(s *specs.Spec) {
			for _, c := range miswiredConfigs() {
				s.It(c.behaviour, func(ctx *specs.Context) {
					app, err := bootstrap.New(c.cfg)

					// It fails at construction, and hands back nothing a
					// caller could mistake for a wired application.
					ctx.Expect(err != nil).To(specs.BeTrue())
					ctx.Expect(app == nil).To(specs.BeTrue())
				})
			}
		})

		s.When("every writer is supplied and the run completes", func(s *specs.Spec) {
			s.It("reports the wired state on stdout", func(ctx *specs.Context) {
				stdout, _, err := wireAndRun(ctx, slog.LevelInfo, ctx.T.Context())

				ctx.Expect(err == nil).To(specs.BeTrue())
				ctx.Expect(strings.Contains(stdout.String(), "atomwright")).To(specs.BeTrue())
			})
		})

		// Diagnostics belong on stderr so stdout stays usable as a data stream
		// for the CLI surfaces the functional epics will add.
		s.When("the logger is admitting debug records", func(s *specs.Spec) {
			s.It("makes the composition root observable on stderr", func(ctx *specs.Context) {
				_, stderr, err := wireAndRun(ctx, slog.LevelDebug, ctx.T.Context())

				ctx.Expect(err == nil).To(specs.BeTrue())
				ctx.Expect(stderr.Len() > 0).To(specs.BeTrue())
			})

			s.It("leaks no log record onto stdout", func(ctx *specs.Context) {
				stdout, _, err := wireAndRun(ctx, slog.LevelDebug, ctx.T.Context())

				ctx.Expect(err == nil).To(specs.BeTrue())
				ctx.Expect(strings.Contains(stdout.String(), "level=")).To(specs.BeFalse())
			})
		})

		s.When("the context is already cancelled before the run begins", func(s *specs.Spec) {
			s.It("stops with the cancellation the caller asked for", func(ctx *specs.Context) {
				_, _, err := wireAndRun(ctx, 0, cancelledContext(ctx.T))

				ctx.Expect(errors.Is(err, context.Canceled)).To(specs.BeTrue())
			})

			s.It("writes nothing to stdout", func(ctx *specs.Context) {
				stdout, _, _ := wireAndRun(ctx, 0, cancelledContext(ctx.T))

				specs.EqualTo(ctx, stdout.Len(), 0)
			})
		})
	})
}
