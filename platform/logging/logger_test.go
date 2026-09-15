package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/getsyntegrity/atomwright/platform/logging"
)

func TestNewWritesRecordsToTheGivenWriter(t *testing.T) {
	var buf bytes.Buffer

	logging.New(&buf, slog.LevelInfo).Info("composition root started", "component", "bootstrap")

	got := buf.String()
	for _, want := range []string{"level=INFO", `msg="composition root started"`, "component=bootstrap"} {
		if !strings.Contains(got, want) {
			t.Errorf("log output is missing %q\ngot: %s", want, got)
		}
	}
}

func TestNewHonoursTheMinimumLevel(t *testing.T) {
	var buf bytes.Buffer

	logger := logging.New(&buf, slog.LevelWarn)
	logger.Info("below the threshold")
	logger.Warn("at the threshold")

	got := buf.String()
	if strings.Contains(got, "below the threshold") {
		t.Errorf("a record below the minimum level was written\ngot: %s", got)
	}
	if !strings.Contains(got, "at the threshold") {
		t.Errorf("a record at the minimum level was dropped\ngot: %s", got)
	}
}
