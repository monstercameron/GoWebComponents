//go:build !js || !wasm

package browser

import "testing"

func TestNewCoordinationHarnessNativeReportsUnavailable(t *testing.T) {
	parseCalls := 0
	parsePrev := nativeCoordinationHarnessFatal
	nativeCoordinationHarnessFatal = func(parseTb testing.TB) {
		parseTb.Helper()
		parseCalls++
	}
	t.Cleanup(func() {
		nativeCoordinationHarnessFatal = parsePrev
	})

	parseHarness := NewCoordinationHarness(t, Options{Path: "/demo"})
	if parseHarness != nil {
		t.Fatalf("NewCoordinationHarness() = %#v, want nil", parseHarness)
	}
	if parseCalls != 1 {
		t.Fatalf("native fatal calls = %d, want 1", parseCalls)
	}
}

func TestCoordinationHarnessNativeNoOpMethods(t *testing.T) {
	parseHarness := &CoordinationHarness{}
	if parseHarness.Environment() != nil {
		t.Fatal("Environment() returned non-nil native environment")
	}
	if parseHarness.OpenTab("/path", "tab") != nil {
		t.Fatal("OpenTab() returned non-nil native tab")
	}
	if parseHarness.Workers() != nil || parseHarness.Worker(0) != nil {
		t.Fatal("native worker accessors should return nil")
	}
	if parseHarness.CrossTabMessages("updates") != nil {
		t.Fatal("CrossTabMessages() returned non-nil native messages")
	}
	parseHarness.SetOffline(true)
	if parseHarness.IsOffline() {
		t.Fatal("native IsOffline() = true, want false")
	}
	parseHarness.QueueRetry("job", 3)
	parseHarness.FailRetry("job", "failed")
	parseHarness.ResolveRetry("job")
	if parseHarness.RetryEntries() != nil {
		t.Fatal("RetryEntries() returned non-nil native entries")
	}
	parseHarness.RecordReconnectFailure("failed")
	parseHarness.RecordReconnectSuccess()
	if parseSnapshot := parseHarness.Reconnect(); parseSnapshot != (ReconnectSnapshot{}) {
		t.Fatalf("Reconnect() = %+v, want zero snapshot", parseSnapshot)
	}
}
