// Package logging_test exercises the logger from outside the package, which is
// the only way to prove the exported surface alone is enough to construct one
// and control what it emits.
//
// The suite is written with go-specs. platform/* is a standard-library-only
// layer, and this third-party import is permitted there because it appears only
// in a _test.go file -- see ADR-0003.
package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/getsyntegrity/atomwright/platform/logging"
	"github.com/pablogore/go-specs/specs"
)

// fragment is one part of a written record a reader must be able to find in
// the output.
type fragment struct {
	behaviour string
	want      string
}

func recordFragments() []fragment {
	return []fragment{
		{"writes the level of the record", "level=INFO"},
		{"writes the message of the record", `msg="composition root started"`},
		{"writes the attributes the caller supplied", "component=bootstrap"},
	}
}

// loggedRecord writes one info record through a logger built on a fresh buffer
// and returns everything that reached the writer.
func loggedRecord() string {
	var buf bytes.Buffer

	logging.New(&buf, slog.LevelInfo).Info("composition root started", "component", "bootstrap")

	return buf.String()
}

// recordsAroundTheThreshold writes one record below the minimum level and one
// at it, so a spec can state which of the two survived.
func recordsAroundTheThreshold() string {
	var buf bytes.Buffer

	logger := logging.New(&buf, slog.LevelWarn)
	logger.Info("below the threshold")
	logger.Warn("at the threshold")

	return buf.String()
}

func TestLogger(t *testing.T) {
	specs.Describe(t, "the platform logger", func(s *specs.Spec) {
		s.When("a record is logged at or above the minimum level", func(s *specs.Spec) {
			for _, f := range recordFragments() {
				s.It(f.behaviour+" to the writer it was given", func(ctx *specs.Context) {
					ctx.Expect(strings.Contains(loggedRecord(), f.want)).To(specs.BeTrue())
				})
			}
		})

		s.When("the logger is built with a minimum level", func(s *specs.Spec) {
			s.It("drops a record below that level", func(ctx *specs.Context) {
				ctx.Expect(strings.Contains(recordsAroundTheThreshold(), "below the threshold")).To(specs.BeFalse())
			})

			s.It("writes a record at that level", func(ctx *specs.Context) {
				ctx.Expect(strings.Contains(recordsAroundTheThreshold(), "at the threshold")).To(specs.BeTrue())
			})
		})
	})
}
