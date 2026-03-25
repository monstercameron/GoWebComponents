//go:build js && wasm
// +build js,wasm

package browser

import (
	"syscall/js"
	"testing"
)

func TestCoordinationHarnessTracksOfflineRetryReconnectAndChannels(t *testing.T) {
	harness := NewCoordinationHarness(t)
	if harness == nil || harness.Environment() == nil {
		t.Fatalf("expected coordination harness environment")
	}
	if harness.IsOffline() {
		t.Fatalf("expected harness to start online")
	}

	harness.SetOffline(true)
	if !harness.IsOffline() {
		t.Fatalf("expected harness offline toggle to persist")
	}

	harness.QueueRetry("mut-1", 2)
	harness.FailRetry("mut-1", "offline")
	entries := harness.RetryEntries()
	if len(entries) != 1 || entries[0].Attempts != 1 || !entries[0].Pending {
		t.Fatalf("expected first retry failure to remain pending, got %+v", entries)
	}
	harness.FailRetry("mut-1", "still-offline")
	entries = harness.RetryEntries()
	if len(entries) != 1 || entries[0].Attempts != 2 || entries[0].Pending {
		t.Fatalf("expected retry to stop pending after max attempts, got %+v", entries)
	}
	harness.ResolveRetry("mut-1")
	entries = harness.RetryEntries()
	if len(entries) != 1 || entries[0].Pending || entries[0].LastError != "" {
		t.Fatalf("expected resolve retry to clear pending state, got %+v", entries)
	}

	harness.RecordReconnectFailure("offline")
	harness.RecordReconnectFailure("timeout")
	reconnect := harness.Reconnect()
	if reconnect.IsConnected || reconnect.Attempts != 2 || reconnect.LastError != "timeout" {
		t.Fatalf("expected reconnect failures to increment attempts, got %+v", reconnect)
	}
	harness.RecordReconnectSuccess()
	reconnect = harness.Reconnect()
	if !reconnect.IsConnected || reconnect.LastError != "" {
		t.Fatalf("expected reconnect success to clear error state, got %+v", reconnect)
	}

	workerInit := js.Global().Get("Object").New()
	workerInit.Set("name", "sync-worker")
	harness.Environment().Window().Get("Worker").New("/workers/sync.mjs", workerInit)
	if workers := harness.Workers(); len(workers) != 1 || workers[0].URL() != "/workers/sync.mjs" {
		t.Fatalf("expected worker tracking to record created worker, got %+v", workers)
	}

	channel := harness.Environment().Window().Get("BroadcastChannel").New("sync")
	channel.Call("postMessage", "refresh")
	if messages := harness.CrossTabMessages("sync"); len(messages) != 1 {
		t.Fatalf("expected one filtered cross-tab message, got %+v", messages)
	}

	tab := harness.OpenTab("/workspace", "tab-2")
	if tab == nil {
		t.Fatalf("expected OpenTab to return one mock window")
	}
	if calls := harness.Environment().OpenCalls(); len(calls) != 1 || calls[0].URL != "/workspace" || calls[0].Name != "tab-2" {
		t.Fatalf("expected OpenTab to record one window.open call, got %+v", calls)
	}
}
