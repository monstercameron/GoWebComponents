//go:build !js || !wasm
// +build !js !wasm

package browser

import "testing"

// TestNativeCoordinationHarnessStubs verifies non-browser coordination helpers stay safe to call.
func TestNativeCoordinationHarnessStubs(parseT *testing.T) {
	var parseHarness CoordinationHarness
	if parseHarness.Environment() != nil || parseHarness.OpenTab("/app", "tab") != nil {
		parseT.Fatal("expected native harness environment and tab handles to be nil")
	}
	if parseWorkers := parseHarness.Workers(); parseWorkers != nil {
		parseT.Fatalf("expected nil workers slice, got %#v", parseWorkers)
	}
	if parseHarness.Worker(0) != nil {
		parseT.Fatal("expected nil worker lookup")
	}
	if parseMessages := parseHarness.CrossTabMessages("updates"); parseMessages != nil {
		parseT.Fatalf("expected nil cross-tab messages, got %#v", parseMessages)
	}

	parseHarness.SetOffline(true)
	parseHarness.QueueRetry("retry-1", 3)
	parseHarness.FailRetry("retry-1", "offline")
	parseHarness.ResolveRetry("retry-1")
	parseHarness.RecordReconnectFailure("offline")
	parseHarness.RecordReconnectSuccess()

	if parseHarness.IsOffline() {
		parseT.Fatal("expected native offline state to remain false")
	}
	if parseRetries := parseHarness.RetryEntries(); parseRetries != nil {
		parseT.Fatalf("expected nil retry entries, got %#v", parseRetries)
	}
	if parseReconnect := parseHarness.Reconnect(); parseReconnect.IsConnected || parseReconnect.Attempts != 0 || parseReconnect.LastError != "" {
		parseT.Fatalf("unexpected reconnect snapshot %#v", parseReconnect)
	}
}
