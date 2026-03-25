//go:build !js || !wasm
// +build !js !wasm

package browser

import "testing"

func TestNativeCoordinationHarnessStubMethods(t *testing.T) {
	harness := &CoordinationHarness{}
	harness.SetOffline(true)
	harness.QueueRetry("mut-1", 3)
	harness.FailRetry("mut-1", "offline")
	harness.ResolveRetry("mut-1")
	harness.RecordReconnectFailure("offline")
	harness.RecordReconnectSuccess()

	if harness.Environment() != nil || harness.OpenTab("/workspace", "tab-2") != nil {
		t.Fatalf("expected nil environment and tab from native coordination stub")
	}
	if harness.Worker(0) != nil || len(harness.Workers()) != 0 {
		t.Fatalf("expected no worker handles in native coordination stub")
	}
	if len(harness.CrossTabMessages("sync")) != 0 {
		t.Fatalf("expected no cross-tab messages in native coordination stub")
	}
	if harness.IsOffline() {
		t.Fatalf("expected native coordination stub to report offline=false")
	}
	if len(harness.RetryEntries()) != 0 {
		t.Fatalf("expected native coordination stub to return no retry entries")
	}
	if harness.Reconnect() != (ReconnectSnapshot{}) {
		t.Fatalf("expected native coordination stub reconnect state to be zero value")
	}
}
