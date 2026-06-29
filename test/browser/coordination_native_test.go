//go:build !js || !wasm

package browser

import "testing"

func TestNativeCoordinationHarnessStubMethods(parseT *testing.T) {
	parseHarness := &CoordinationHarness{}
	parseHarness.SetOffline(true)
	parseHarness.QueueRetry("mut-1", 3)
	parseHarness.FailRetry("mut-1", "offline")
	parseHarness.ResolveRetry("mut-1")
	parseHarness.RecordReconnectFailure("offline")
	parseHarness.RecordReconnectSuccess()

	if parseHarness.Environment() != nil || parseHarness.OpenTab("/workspace", "tab-2") != nil {
		parseT.Fatalf("expected nil environment and tab from native coordination stub")
	}
	if parseHarness.Worker(0) != nil || len(parseHarness.Workers()) != 0 {
		parseT.Fatalf("expected no worker handles in native coordination stub")
	}
	if len(parseHarness.CrossTabMessages("sync")) != 0 {
		parseT.Fatalf("expected no cross-tab messages in native coordination stub")
	}
	if parseHarness.IsOffline() {
		parseT.Fatalf("expected native coordination stub to report offline=false")
	}
	if len(parseHarness.RetryEntries()) != 0 {
		parseT.Fatalf("expected native coordination stub to return no retry entries")
	}
	if parseHarness.Reconnect() != (ReconnectSnapshot{}) {
		parseT.Fatalf("expected native coordination stub reconnect state to be zero value")
	}
}
