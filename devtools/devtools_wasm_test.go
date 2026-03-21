//go:build js && wasm
// +build js,wasm

package devtools

import (
	"testing"
	"time"

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

func TestSnapshotNowIncludesWrappedPanicMetadata(t *testing.T) {
	runtime.ClearLogs()
	runtime.ClearDiagnostics()
	defer runtime.ClearLogs()
	defer runtime.ClearDiagnostics()

	message := runtime.ReportUnhandledPanicContext("runtime", runtime.PanicPhaseStartup, "RenderTo", "#app", []string{"App"}, "startup boom")
	if message == "" {
		t.Fatal("expected wrapped panic message")
	}

	snapshot := SnapshotNow()
	runtimeDiagnostics := runtime.GetDiagnostics()
	if len(runtimeDiagnostics) == 0 {
		t.Fatal("expected runtime panic diagnostic")
	}
	runtimeDiagnostic := runtimeDiagnostics[len(runtimeDiagnostics)-1]
	if len(snapshot.Diagnostics) == 0 {
		t.Fatal("expected panic diagnostic in snapshot")
	}
	diagnostic := snapshot.Diagnostics[len(snapshot.Diagnostics)-1]
	if diagnostic.Code != "GWC-RUNTIME-PANIC-STARTUP" || diagnostic.Path != "#app" {
		t.Fatalf("unexpected diagnostic payload: %+v", diagnostic)
	}
	if diagnostic.TopFrame == "" || diagnostic.Consequence == "" {
		t.Fatalf("expected wrapped panic diagnostic metadata, got %+v", diagnostic)
	}
	if diagnostic.Code != runtimeDiagnostic.Code ||
		diagnostic.Message != runtimeDiagnostic.Message ||
		diagnostic.Path != runtimeDiagnostic.Path ||
		diagnostic.Docs != runtimeDiagnostic.Docs ||
		diagnostic.Remediation != runtimeDiagnostic.Remediation ||
		diagnostic.Recoverable != runtimeDiagnostic.Recoverable ||
		diagnostic.TopFrame != runtimeDiagnostic.TopFrame ||
		diagnostic.Consequence != runtimeDiagnostic.Consequence {
		t.Fatalf("expected snapshot diagnostic to mirror runtime diagnostic, snapshot=%+v runtime=%+v", diagnostic, runtimeDiagnostic)
	}

	runtimeLogs := runtime.GetLogs()
	if len(runtimeLogs) == 0 {
		t.Fatal("expected runtime panic log")
	}
	runtimeLog := runtimeLogs[len(runtimeLogs)-1]
	if len(snapshot.Logs) == 0 {
		t.Fatal("expected panic log in snapshot")
	}
	entry := snapshot.Logs[len(snapshot.Logs)-1]
	if entry.Code != "GWC-RUNTIME-PANIC-STARTUP" {
		t.Fatalf("unexpected log payload: %+v", entry)
	}
	if entry.TopFrame == "" || entry.Consequence == "" {
		t.Fatalf("expected wrapped panic log metadata, got %+v", entry)
	}
	if entry.Fields["path"] != "#app" || entry.Fields["runtime"] == "" || entry.Fields["top_frame"] == "" {
		t.Fatalf("expected panic fields to mirror into log entry, got %+v", entry)
	}
	if entry.Code != runtimeLog.Code ||
		entry.Message != runtimeLog.Message ||
		entry.Docs != runtimeLog.Docs ||
		entry.Remediation != runtimeLog.Remediation ||
		entry.Recoverable != runtimeLog.Recoverable ||
		entry.TopFrame != runtimeLog.TopFrame ||
		entry.Consequence != runtimeLog.Consequence ||
		entry.Fields["path"] != runtimeLog.Fields["path"] ||
		entry.Fields["runtime"] != runtimeLog.Fields["runtime"] ||
		entry.Fields["top_frame"] != runtimeLog.Fields["top_frame"] {
		t.Fatalf("expected snapshot log to mirror runtime log, snapshot=%+v runtime=%+v", entry, runtimeLog)
	}
}

func TestExportSnapshotJSONAndCompareSnapshots(t *testing.T) {
	before := Snapshot{
		Route: Route{Path: "/before"},
		Cache: []CacheEntry{{Key: "item", Ready: true, UpdatedAt: time.Unix(1, 0).UTC()}},
		Stats: Stats{TotalFibers: 1},
		Logs:  []Log{{Domain: "router", Message: "before"}},
	}
	after := Snapshot{
		Route: Route{Path: "/after"},
		Cache: []CacheEntry{{Key: "item", Ready: true, UpdatedAt: time.Unix(1, 0).UTC()}},
		Stats: Stats{TotalFibers: 2},
		Logs:  []Log{{Domain: "router", Message: "after"}},
	}

	payload, err := ExportSnapshotJSON(before)
	if err != nil {
		t.Fatalf("ExportSnapshotJSON failed: %v", err)
	}
	if len(payload) == 0 || payload[0] != '{' {
		t.Fatalf("expected JSON payload, got %q", string(payload))
	}

	comparison, err := CompareSnapshots(before, after)
	if err != nil {
		t.Fatalf("CompareSnapshots failed: %v", err)
	}
	if comparison.Equal {
		t.Fatal("expected snapshots to differ")
	}
	if comparison.PreviousFingerprint == "" || comparison.CurrentFingerprint == "" {
		t.Fatal("expected fingerprints to be populated")
	}
	if comparison.PreviousSize == 0 || comparison.CurrentSize == 0 {
		t.Fatal("expected serialized sizes to be populated")
	}
	if len(comparison.ChangedSections) == 0 {
		t.Fatal("expected changed sections to be reported")
	}
	want := map[string]bool{"route": true, "stats": true, "logs": true}
	for _, section := range comparison.ChangedSections {
		delete(want, section)
	}
	if len(want) != 0 {
		t.Fatalf("expected route, stats, and logs to change, missing %v", want)
	}
}

func TestMapInspectionFineGrainedMetadata(t *testing.T) {
	node := mapNode(&runtime.FiberSnapshot{
		Name:           "ReactiveText",
		Kind:           "text",
		FineGrained:    true,
		ReactiveSource: "count",
		UpdateOrigin:   "fine-grained",
	})
	if node == nil {
		t.Fatal("expected mapped node")
	}
	if !node.FineGrained {
		t.Fatal("expected fine-grained flag to map")
	}
	if node.ReactiveSource != "count" {
		t.Fatalf("expected reactive source count, got %q", node.ReactiveSource)
	}
	if node.UpdateOrigin != "fine-grained" {
		t.Fatalf("expected update origin fine-grained, got %q", node.UpdateOrigin)
	}

	stats := mapStats(runtime.InspectionStats{FineGrainedFibers: 1})
	if stats.FineGrainedFibers != 1 {
		t.Fatalf("expected fine-grained fiber count to map, got %d", stats.FineGrainedFibers)
	}

	profiling := mapProfiling(runtime.ProfilingSnapshot{ScheduledGranularMarks: 4, FineGrainedCommits: 3, FineGrainedDescendantHostCommits: 6, FineGrainedDescendantTextCommits: 2})
	if profiling.ScheduledGranularMarks != 4 {
		t.Fatalf("expected granular marks to map, got %d", profiling.ScheduledGranularMarks)
	}
	if profiling.FineGrainedCommits != 3 {
		t.Fatalf("expected fine-grained commits to map, got %d", profiling.FineGrainedCommits)
	}
	if profiling.FineGrainedDescendantHostCommits != 6 {
		t.Fatalf("expected descendant host commits to map, got %d", profiling.FineGrainedDescendantHostCommits)
	}
	if profiling.FineGrainedDescendantTextCommits != 2 {
		t.Fatalf("expected descendant text commits to map, got %d", profiling.FineGrainedDescendantTextCommits)
	}
}
