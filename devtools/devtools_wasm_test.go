//go:build js && wasm
// +build js,wasm

package devtools

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func TestSnapshotNowIncludesBufferedLogs(t *testing.T) {
	runtime.ClearLogs()
	runtime.ClearDiagnostics()
	defer runtime.ClearLogs()
	defer runtime.ClearDiagnostics()

	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "navigation started", "nav-1", map[string]string{
		"target": "/dashboard",
	})

	snapshot := SnapshotNow()
	if len(snapshot.Logs) != 1 {
		t.Fatalf("expected one buffered log, got %+v", snapshot.Logs)
	}
	if snapshot.Logs[0].Domain != "router" || snapshot.Logs[0].Fields["target"] != "/dashboard" {
		t.Fatalf("unexpected buffered log payload: %+v", snapshot.Logs[0])
	}
}
