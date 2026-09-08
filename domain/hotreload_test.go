package domain_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/delta"
	"github.com/monstercameron/GoWebComponents/v6/domain"
)

// v5 P3.14 — domain hot-reload.
//
// Criterion: editing a domain.Handle body reloads without dropping published
// projections.
//
// A reload is a worker restart with new code, so everything in memory is gone.
// What must NOT be gone, from the render thread's point of view, is the
// projection it is already displaying. Two distinct things would break it, and
// they break differently:
//
//   - The delta engine's index. Lost, the reloaded worker's first publish emits
//     an insert for every row — a full re-send of the projection across the
//     boundary on a code edit. The UI does not go blank, but the "hot" in hot
//     reload does, and at 20,000 rows it is a multi-megabyte stall.
//   - The command ledger. Lost, a retry in flight across the reload is treated
//     as new work, so a command that succeeded just before the edit runs again
//     just after it. This one is silent and corrupting.
//
// These tests are written as an actual reload: export, discard, restore into
// fresh objects, and check what the render thread would observe.

type projectionRow struct {
	Name  string `json:"name"`
	Total int    `json:"total"`
}

func encodeProjectionRow(parseRow projectionRow) []byte {
	parseBytes, _ := json.Marshal(parseRow)
	return parseBytes
}

func buildPublishedRows(parseCount int) []delta.Row {
	parseRows := make([]delta.Row, 0, parseCount)
	for parseIndex := range parseCount {
		parseRows = append(parseRows, delta.Row{
			Key:     delta.Key(fmt.Sprintf("row-%05d", parseIndex)),
			Version: 1,
			Payload: encodeProjectionRow(projectionRow{Name: fmt.Sprintf("item-%d", parseIndex), Total: parseIndex}),
		})
	}
	return parseRows
}

// ----------------------------------------------- the projection guarantee

// TestReloadDoesNotRepublishTheProjection is P3.14's criterion.
func TestReloadDoesNotRepublishTheProjection(parseT *testing.T) {
	const parseRowCount = 20000

	parseEngine := delta.New()
	parseRows := buildPublishedRows(parseRowCount)
	parseInitialOps, parseErr := parseEngine.Publish(parseRows)
	if parseErr != nil {
		parseT.Fatalf("initial publish: %v", parseErr)
	}
	if len(parseInitialOps) != parseRowCount {
		parseT.Fatalf("initial publish emitted %d ops, want %d inserts", len(parseInitialOps), parseRowCount)
	}

	// --- the reload: export, discard the worker, restore into a fresh engine.
	parseState := parseEngine.ExportState()
	parseEngine = nil

	parseReloaded, parseRestoreErr := delta.Restore(parseState)
	if parseRestoreErr != nil {
		parseT.Fatalf("Restore: %v", parseRestoreErr)
	}

	// An edit that changed code but no data must publish NOTHING.
	parseAfterReload, parseErr := parseReloaded.Publish(parseRows)
	if parseErr != nil {
		parseT.Fatalf("publish after reload: %v", parseErr)
	}
	if len(parseAfterReload) != 0 {
		parseT.Errorf("publishing unchanged rows after a reload emitted %d ops, want 0 — the projection was re-sent",
			len(parseAfterReload))
	}

	// And a real change still publishes exactly its own delta.
	parseRows[7500].Version = 2
	parseRows[7500].Payload = encodeProjectionRow(projectionRow{Name: "edited", Total: 1})

	parseChangeOps, parseErr := parseReloaded.Publish(parseRows)
	if parseErr != nil {
		parseT.Fatalf("publish after edit: %v", parseErr)
	}
	if len(parseChangeOps) != 1 {
		parseT.Fatalf("a one-row change after a reload emitted %d ops, want 1", len(parseChangeOps))
	}
	if parseChangeOps[0].Key != "row-07500" || parseChangeOps[0].Kind != delta.OpUpdate {
		parseT.Errorf("op = %+v, want an update of row-07500", parseChangeOps[0])
	}
}

// TestReloadPreservesOrderNotJustMembership: a restored engine that kept the
// right keys in the wrong order would emit a storm of moves on the next publish
// — technically not a re-send, and just as bad to watch.
func TestReloadPreservesOrderNotJustMembership(parseT *testing.T) {
	parseEngine := delta.New()
	parseRows := buildPublishedRows(500)
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}

	parseReloaded, parseErr := delta.Restore(parseEngine.ExportState())
	if parseErr != nil {
		parseT.Fatalf("Restore: %v", parseErr)
	}

	parseOriginalKeys := parseEngine.Keys()
	parseRestoredKeys := parseReloaded.Keys()
	if len(parseRestoredKeys) != len(parseOriginalKeys) {
		parseT.Fatalf("restored %d keys, want %d", len(parseRestoredKeys), len(parseOriginalKeys))
	}
	for parseIndex := range parseOriginalKeys {
		if parseRestoredKeys[parseIndex] != parseOriginalKeys[parseIndex] {
			parseT.Fatalf("restored key %d = %q, want %q — the order was not preserved",
				parseIndex, parseRestoredKeys[parseIndex], parseOriginalKeys[parseIndex])
		}
	}
}

// TestReloadStateSurvivesJSON: the state crosses a worker boundary, so it is
// serialized in practice.
func TestReloadStateSurvivesJSON(parseT *testing.T) {
	parseEngine := delta.New()
	if _, parseErr := parseEngine.Publish(buildPublishedRows(100)); parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}

	parseEncoded, parseErr := json.Marshal(parseEngine.ExportState())
	if parseErr != nil {
		parseT.Fatalf("marshal: %v", parseErr)
	}
	var parseDecoded delta.State
	if parseErr := json.Unmarshal(parseEncoded, &parseDecoded); parseErr != nil {
		parseT.Fatalf("unmarshal: %v", parseErr)
	}

	parseReloaded, parseRestoreErr := delta.Restore(parseDecoded)
	if parseRestoreErr != nil {
		parseT.Fatalf("Restore: %v", parseRestoreErr)
	}
	parseOps, parseErr := parseReloaded.Publish(buildPublishedRows(100))
	if parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}
	if len(parseOps) != 0 {
		parseT.Errorf("a JSON round-tripped state republished %d ops, want 0", len(parseOps))
	}
}

// TestExportedStateCarriesNoPayloads is what makes carrying it across a reload
// cheap enough to be worth doing at all.
func TestExportedStateCarriesNoPayloads(parseT *testing.T) {
	parseEngine := delta.New()
	parseRows := make([]delta.Row, 0, 200)
	parseBigPayload := make([]byte, 4096)
	for parseIndex := range 200 {
		parseRows = append(parseRows, delta.Row{
			Key: delta.Key(fmt.Sprintf("k%03d", parseIndex)), Version: 1, Payload: parseBigPayload,
		})
	}
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("publish: %v", parseErr)
	}

	parseEncoded, _ := json.Marshal(parseEngine.ExportState())
	parsePayloadBytes := 200 * len(parseBigPayload)
	if len(parseEncoded) >= parsePayloadBytes {
		parseT.Errorf("exported state is %d B against %d B of payload; it is carrying the data it should not have",
			len(parseEncoded), parsePayloadBytes)
	}
	parseT.Logf("200 rows x 4096 B payload = %d B; exported index = %d B", parsePayloadBytes, len(parseEncoded))
}

// -------------------------------------------------- the replay guarantee

// TestReloadPreservesReplayProtection covers the silent failure.
//
// A reloaded worker with a fresh ledger treats an in-flight retry as new work,
// so a command that succeeded just before the edit runs again just after it.
// Nothing reports it; the effects simply happen twice.
func TestReloadPreservesReplayProtection(parseT *testing.T) {
	parseStore := domain.NewMemoryCheckpointStore()
	parseRuntime := domain.NewRuntime(parseStore)

	parseEffects := 0
	parseEffect := func() error {
		parseEffects++
		return nil
	}

	for _, parseCommandID := range []domain.CommandID{"cmd-a", "cmd-b", "cmd-c"} {
		if _, parseErr := parseRuntime.Execute(parseCommandID, parseEffect); parseErr != nil {
			parseT.Fatalf("Execute(%s): %v", parseCommandID, parseErr)
		}
	}
	if parseErr := parseRuntime.Cancel("cmd-cancelled"); parseErr != nil {
		parseT.Fatalf("Cancel: %v", parseErr)
	}

	// --- the reload.
	parseState := parseRuntime.ExportState()
	parseReloaded, parseErr := domain.RestoreRuntime(parseStore, parseState)
	if parseErr != nil {
		parseT.Fatalf("RestoreRuntime: %v", parseErr)
	}

	// A retry that crossed the reload must still be a replay.
	parseOutcome, parseRetryErr := parseReloaded.Execute("cmd-b", parseEffect)
	if parseRetryErr != nil {
		parseT.Fatalf("retry after reload: %v", parseRetryErr)
	}
	if parseOutcome != domain.OutcomeReplayed {
		parseT.Errorf("outcome = %s, want replayed — the reload lost replay protection and re-ran an applied command",
			parseOutcome)
	}
	if parseEffects != 3 {
		parseT.Errorf("effects ran %d times, want 3 — a command ran twice across the reload", parseEffects)
	}

	// A cancellation must stay a cancellation.
	parseCancelledOutcome, _ := parseReloaded.Execute("cmd-cancelled", parseEffect)
	if parseCancelledOutcome != domain.OutcomeCancelled {
		parseT.Errorf("outcome = %s, want cancelled — the reload resurrected a cancelled command", parseCancelledOutcome)
	}
}

// TestReloadResumesBulkProgress: checkpoints are durable and were deliberately
// left out of the export, so a bulk command must resume from the store rather
// than from anything carried across.
func TestReloadResumesBulkProgress(parseT *testing.T) {
	parseStore := domain.NewMemoryCheckpointStore()
	parseRuntime := domain.NewRuntime(parseStore)
	parseCommand := domain.BulkCommand{ID: "import", Total: 1000, CheckpointEvery: 100}

	parseRuns := make([]int, 1000)
	if _, parseErr := parseRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		if parseIndex == 400 {
			return fmt.Errorf("edit landed mid-import")
		}
		parseRuns[parseIndex]++
		return nil
	}); parseErr == nil {
		parseT.Fatal("the first run was supposed to stop")
	}

	parseReloaded, parseErr := domain.RestoreRuntime(parseStore, parseRuntime.ExportState())
	if parseErr != nil {
		parseT.Fatalf("RestoreRuntime: %v", parseErr)
	}

	parseResult, parseResumeErr := parseReloaded.ExecuteBulk(parseCommand, func(parseIndex int) error {
		parseRuns[parseIndex]++
		return nil
	})
	if parseResumeErr != nil {
		parseT.Fatalf("resume after reload: %v", parseResumeErr)
	}
	if !parseResult.Completed || parseResult.ResumedFrom != 400 {
		parseT.Errorf("result = %+v, want completed resuming from 400", parseResult)
	}
	for parseIndex, parseCount := range parseRuns {
		if parseCount != 1 {
			parseT.Fatalf("row %d ran %d times across the reload, want exactly 1", parseIndex, parseCount)
		}
	}
}

func TestRestoredRuntimeStateSurvivesJSON(parseT *testing.T) {
	parseRuntime := domain.NewRuntime(nil)
	if _, parseErr := parseRuntime.Execute("cmd-x", func() error { return nil }); parseErr != nil {
		parseT.Fatalf("Execute: %v", parseErr)
	}

	parseEncoded, parseErr := json.Marshal(parseRuntime.ExportState())
	if parseErr != nil {
		parseT.Fatalf("marshal: %v", parseErr)
	}
	var parseDecoded domain.RuntimeState
	if parseErr := json.Unmarshal(parseEncoded, &parseDecoded); parseErr != nil {
		parseT.Fatalf("unmarshal: %v", parseErr)
	}

	parseReloaded, parseRestoreErr := domain.RestoreRuntime(nil, parseDecoded)
	if parseRestoreErr != nil {
		parseT.Fatalf("RestoreRuntime: %v", parseRestoreErr)
	}
	parseEffects := 0
	parseOutcome, _ := parseReloaded.Execute("cmd-x", func() error {
		parseEffects++
		return nil
	})
	if parseOutcome != domain.OutcomeReplayed || parseEffects != 0 {
		parseT.Errorf("outcome = %s, effects = %d — replay protection did not survive JSON", parseOutcome, parseEffects)
	}
}

// ------------------------------------------------------------------ guards

func TestRestoreRejectsMalformedState(parseT *testing.T) {
	if _, parseErr := delta.Restore(delta.State{
		Order:    []delta.Key{"a", "b"},
		Versions: []uint64{1},
	}); parseErr == nil {
		parseT.Error("mismatched parallel arrays must be rejected")
	}
	if _, parseErr := delta.Restore(delta.State{
		Order:    []delta.Key{"a", "a"},
		Versions: []uint64{1, 2},
	}); parseErr == nil {
		parseT.Error("a repeated key makes the restored order ambiguous and must be rejected")
	}
	if _, parseErr := delta.Restore(delta.State{
		Order:    []delta.Key{""},
		Versions: []uint64{1},
	}); parseErr == nil {
		parseT.Error("an empty key must be rejected")
	}

	if _, parseErr := domain.RestoreRuntime(nil, domain.RuntimeState{Applied: []domain.CommandID{""}}); parseErr == nil {
		parseT.Error("an empty command id must be rejected")
	}
	if _, parseErr := domain.RestoreRuntime(nil, domain.RuntimeState{Cancelled: []domain.CommandID{""}}); parseErr == nil {
		parseT.Error("an empty cancelled command id must be rejected")
	}
}

func TestRestoringAnEmptyStateIsAFreshStart(parseT *testing.T) {
	parseEngine, parseErr := delta.Restore(delta.State{})
	if parseErr != nil {
		parseT.Fatalf("Restore(empty): %v", parseErr)
	}
	if parseEngine.Len() != 0 {
		parseT.Errorf("Len = %d, want 0", parseEngine.Len())
	}

	parseRuntime, parseRuntimeErr := domain.RestoreRuntime(nil, domain.RuntimeState{})
	if parseRuntimeErr != nil {
		parseT.Fatalf("RestoreRuntime(empty): %v", parseRuntimeErr)
	}
	if parseRuntime.TrackedCommands() != 0 {
		parseT.Errorf("tracked = %d, want 0", parseRuntime.TrackedCommands())
	}
}

func TestNilExportsAreSafe(parseT *testing.T) {
	var parseEngine *delta.Engine
	if parseState := parseEngine.ExportState(); len(parseState.Order) != 0 {
		parseT.Error("a nil engine exports nothing")
	}
	var parseRuntime *domain.Runtime
	if parseState := parseRuntime.ExportState(); len(parseState.Applied) != 0 {
		parseT.Error("a nil runtime exports nothing")
	}
}
