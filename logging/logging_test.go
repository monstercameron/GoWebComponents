package logging

import "testing"

func TestNewLoggerRetainsScope(t *testing.T) {
	logger := New(" example-0 ")
	if logger.Scope() != "example-0" {
		t.Fatalf("expected trimmed scope, got %q", logger.Scope())
	}
}

func TestAttachBrowserConsoleNoopOnNonBrowserTargets(t *testing.T) {
	cleanup := AttachBrowserConsole(BrowserConsoleOptions{Scope: "example-0"})
	if cleanup == nil {
		t.Fatal("expected cleanup function")
	}
	cleanup()
}
