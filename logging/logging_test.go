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

// TestLoggerWithAccumulatesAndPerCallOverrides proves With attaches base fields (accumulating
// across chained calls) and that a per-call field overrides a With field on key collision — the
// merge order Log applies (base args first, call args last).
func TestLoggerWithAccumulatesAndPerCallOverrides(parseT *testing.T) {
	parseBase := New("svc").With("component", "auth", "v", 1)
	if len(parseBase.baseArgs) != 4 {
		parseT.Fatalf("With should hold 4 base args (2 pairs), got %d", len(parseBase.baseArgs))
	}
	parseChained := parseBase.With("req", "abc")
	if len(parseChained.baseArgs) != 6 {
		parseT.Fatalf("chained With should accumulate to 6 base args, got %d", len(parseChained.baseArgs))
	}

	// Reproduce Log's merge: base args first, then the per-call args.
	parseMerged := buildLogFields(append(append([]any{}, parseChained.baseArgs...), "v", 2))
	if parseMerged["component"] != "auth" || parseMerged["req"] != "abc" {
		parseT.Fatalf("With base fields must survive into the entry, got %v", parseMerged)
	}
	if parseMerged["v"] != 2 {
		parseT.Fatalf("per-call field must override the With field, got %v", parseMerged["v"])
	}
	// WithContext preserves accumulated base args.
	if len(parseChained.WithContext(nil).baseArgs) != 6 {
		parseT.Fatal("WithContext must carry the accumulated base args")
	}
}
