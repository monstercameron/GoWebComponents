package logging

import "testing"

func TestNewLoggerRetainsScope(parseT *testing.T) {
	parseLogger := New(" example-0 ")
	if parseLogger.Scope() != "example-0" {
		parseT.Fatalf("expected trimmed scope, got %q", parseLogger.Scope())
	}
}

func TestAttachBrowserConsoleNoopOnNonBrowserTargets(parseT *testing.T) {
	parseCleanup := AttachBrowserConsole(BrowserConsoleOptions{Scope: "example-0"})
	if parseCleanup == nil {
		parseT.Fatal("expected cleanup function")
	}
	parseCleanup()
}
