package runtime2

import (
	"reflect"
	"strings"
	"testing"
)

// TestBuildWorkerRenderInputWithSourceOrderReusesCachedSourceIDs verifies worker render input adaptation reuses cached source-ID order when snapshot keys are unchanged.
func TestBuildWorkerRenderInputWithSourceOrderReusesCachedSourceIDs(parseT *testing.T) {
	parseSnapshot, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "Orders"},
		[]string{"status", "count"},
		map[string]any{"count": 5, "status": "healthy"},
		map[string]uint64{"count": 9, "status": 9},
	)
	if parseSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	parseCachedSourceIDs := []string{"count", "status"}
	getRenderInput, getSourceIDs, parseRenderInputErr := buildWorkerRenderInputWithSourceOrder(parseSnapshot, parseCachedSourceIDs)
	if parseRenderInputErr != nil {
		parseT.Fatalf("buildWorkerRenderInputWithSourceOrder returned error: %v", parseRenderInputErr)
	}
	if !reflect.DeepEqual(getSourceIDs, parseCachedSourceIDs) {
		parseT.Fatalf("expected cached source IDs %+v, got %+v", parseCachedSourceIDs, getSourceIDs)
	}
	if len(getRenderInput.GetSourceEntries) != 2 {
		parseT.Fatalf("expected two source entries, got %d", len(getRenderInput.GetSourceEntries))
	}
	if getRenderInput.GetSourceEntries[0].GetSourceID != "count" || getRenderInput.GetSourceEntries[1].GetSourceID != "status" {
		parseT.Fatalf("expected source entry order [count status], got %+v", getRenderInput.GetSourceEntries)
	}
}

// TestBuildWorkerRenderInputWithSourceOrderRebuildsOnSourceSetChange verifies worker render input adaptation rebuilds normalized source-ID order when snapshot keys change.
func TestBuildWorkerRenderInputWithSourceOrderRebuildsOnSourceSetChange(parseT *testing.T) {
	parseSnapshot, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "Orders"},
		[]string{"status", "count", "alpha"},
		map[string]any{"count": 5, "status": "healthy", "alpha": true},
		map[string]uint64{"count": 9, "status": 9, "alpha": 9},
	)
	if parseSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	getRenderInput, getSourceIDs, parseRenderInputErr := buildWorkerRenderInputWithSourceOrder(
		parseSnapshot,
		[]string{"count", "status"},
	)
	if parseRenderInputErr != nil {
		parseT.Fatalf("buildWorkerRenderInputWithSourceOrder returned error: %v", parseRenderInputErr)
	}
	getExpectedSourceIDs := []string{"alpha", "count", "status"}
	if !reflect.DeepEqual(getSourceIDs, getExpectedSourceIDs) {
		parseT.Fatalf("expected rebuilt source IDs %+v, got %+v", getExpectedSourceIDs, getSourceIDs)
	}
	if len(getRenderInput.GetSourceEntries) != 3 {
		parseT.Fatalf("expected three source entries, got %d", len(getRenderInput.GetSourceEntries))
	}
	for parseSourceIndex, getExpectedSourceID := range getExpectedSourceIDs {
		if getRenderInput.GetSourceEntries[parseSourceIndex].GetSourceID != getExpectedSourceID {
			parseT.Fatalf("expected source entry %d to be %q, got %q", parseSourceIndex, getExpectedSourceID, getRenderInput.GetSourceEntries[parseSourceIndex].GetSourceID)
		}
	}
}

// TestBuildWorkerRenderInputRejectsInvalidSourceID verifies render-input adaptation rejects unsupported source key formats.
func TestBuildWorkerRenderInputRejectsInvalidSourceID(parseT *testing.T) {
	_, parseRenderInputErr := BuildWorkerRenderInput(SnapshotEnvelope{
		RegionInstanceID: "region-invalid-source",
		Epoch:            1,
		InputVersion:     1,
		Props:            map[string]any{"title": "Orders"},
		Sources: map[string]any{
			"bad source": "invalid",
		},
	})
	if parseRenderInputErr == nil {
		parseT.Fatal("expected BuildWorkerRenderInput to reject invalid source IDs")
	}
	if !strings.Contains(parseRenderInputErr.Error(), "source ID") {
		parseT.Fatalf("expected source ID validation error, got: %v", parseRenderInputErr)
	}
}
