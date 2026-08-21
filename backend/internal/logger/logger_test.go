package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	if ParseLevel("DEBUG") != Debug {
		t.Fatal("debug")
	}
	if ParseLevel("warn") != Warn {
		t.Fatal("warn")
	}
	if ParseLevel("error") != Error {
		t.Fatal("error")
	}
	if ParseLevel("nope") != Info {
		t.Fatal("default")
	}
}

func TestLoggerFiltersDebug(t *testing.T) {
	var buf bytes.Buffer
	log := New(Info, &buf)
	log.Debug("hidden")
	log.Info("visible", "k", "v")
	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "visible") || !strings.Contains(out, "k=v") {
		t.Fatal(out)
	}
}
