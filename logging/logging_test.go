package logging

import (
	"errors"
	"log/slog"
	"testing"
)

func TestNewLoggerRetainsScope(parseT *testing.T) {
	parseLogger := New(" public-examples-site ")
	if parseLogger.Scope() != "public-examples-site" {
		parseT.Fatalf("expected trimmed scope, got %q", parseLogger.Scope())
	}
}

func TestAttachBrowserConsoleNoopOnNonBrowserTargets(parseT *testing.T) {
	parseCleanup := AttachBrowserConsole(BrowserConsoleOptions{Scope: "public-examples-site"})
	if parseCleanup == nil {
		parseT.Fatal("expected cleanup function")
	}
	parseCleanup()
}

// TestBuildLogFieldsAcceptsKeyValueAndSlogAttrs verifies the logging surface accepts low-ceremony key/value and slog attributes together.
func TestBuildLogFieldsAcceptsKeyValueAndSlogAttrs(parseT *testing.T) {
	parseFields := buildLogFields([]any{
		"route", "/admin",
		slog.String("reason", "missing_role"),
		map[string]string{"status": "blocked"},
		errors.New("permission denied"),
	})

	if parseFields["route"] != "/admin" || parseFields["reason"] != "missing_role" || parseFields["status"] != "blocked" || parseFields["error"] != "permission denied" {
		parseT.Fatalf("unexpected normalized fields: %#v", parseFields)
	}
}
