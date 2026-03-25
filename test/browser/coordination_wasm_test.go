//go:build js && wasm
// +build js,wasm

package browser

import (
	"syscall/js"
	"testing"
)

func TestCoordinationHarnessTracksOfflineRetryReconnectAndChannels(parseT *testing.T) {
	parseHarness := NewCoordinationHarness(parseT)
	if parseHarness == nil || parseHarness.Environment() == nil {
		parseT.Fatalf("expected coordination harness environment")
	}
	if parseHarness.IsOffline() {
		parseT.Fatalf("expected harness to start online")
	}

	parseHarness.SetOffline(true)
	if !parseHarness.IsOffline() {
		parseT.Fatalf("expected harness offline toggle to persist")
	}

	parseHarness.QueueRetry("mut-1", 2)
	parseHarness.FailRetry("mut-1", "offline")
	parseEntries := parseHarness.RetryEntries()
	if len(parseEntries) != 1 || parseEntries[0].Attempts != 1 || !parseEntries[0].Pending {
		parseT.Fatalf("expected first retry failure to remain pending, got %+v", parseEntries)
	}
	parseHarness.FailRetry("mut-1", "still-offline")
	parseEntries = parseHarness.RetryEntries()
	if len(parseEntries) != 1 || parseEntries[0].Attempts != 2 || parseEntries[0].Pending {
		parseT.Fatalf("expected retry to stop pending after max attempts, got %+v", parseEntries)
	}
	parseHarness.ResolveRetry("mut-1")
	parseEntries = parseHarness.RetryEntries()
	if len(parseEntries) != 1 || parseEntries[0].Pending || parseEntries[0].LastError != "" {
		parseT.Fatalf("expected resolve retry to clear pending state, got %+v", parseEntries)
	}

	parseHarness.RecordReconnectFailure("offline")
	parseHarness.RecordReconnectFailure("timeout")
	parseReconnect := parseHarness.Reconnect()
	if parseReconnect.IsConnected || parseReconnect.Attempts != 2 || parseReconnect.LastError != "timeout" {
		parseT.Fatalf("expected reconnect failures to increment attempts, got %+v", parseReconnect)
	}
	parseHarness.RecordReconnectSuccess()
	parseReconnect = parseHarness.Reconnect()
	if !parseReconnect.IsConnected || parseReconnect.LastError != "" {
		parseT.Fatalf("expected reconnect success to clear error state, got %+v", parseReconnect)
	}

	parseWorkerInit := js.Global().Get("Object").New()
	parseWorkerInit.Set("name", "sync-worker")
	parseHarness.Environment().Window().Get("Worker").New("/workers/sync.mjs", parseWorkerInit)
	if parseWorkers := parseHarness.Workers(); len(parseWorkers) != 1 || parseWorkers[0].URL() != "/workers/sync.mjs" {
		parseT.Fatalf("expected worker tracking to record created worker, got %+v", parseWorkers)
	}

	parseChannel := parseHarness.Environment().Window().Get("BroadcastChannel").New("sync")
	parseChannel.Call("postMessage", "refresh")
	if parseMessages := parseHarness.CrossTabMessages("sync"); len(parseMessages) != 1 {
		parseT.Fatalf("expected one filtered cross-tab message, got %+v", parseMessages)
	}

	parseTab := parseHarness.OpenTab("/workspace", "tab-2")
	if parseTab == nil {
		parseT.Fatalf("expected OpenTab to return one mock window")
	}
	if parseCalls := parseHarness.Environment().OpenCalls(); len(parseCalls) != 1 || parseCalls[0].URL != "/workspace" || parseCalls[0].Name != "tab-2" {
		parseT.Fatalf("expected OpenTab to record one window.open call, got %+v", parseCalls)
	}
}
