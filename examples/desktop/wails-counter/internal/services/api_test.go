package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/gwc-wails-counter/contracts"
	"github.com/monstercameron/GoWebComponents/v6/desktop"
)

// TestAPIServiceFixtureAndActionValidation verifies fixture files and action bounds.
func TestAPIServiceFixtureAndActionValidation(parseTest *testing.T) {
	parseService, parseErr := NewAPIService()
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	defer closeAPITestService(parseTest, parseService)
	for _, parseName := range []string{"sample-a.txt", "sample-b.txt", "sample-unicode-雪.txt"} {
		if _, parseErr = os.Stat(filepath.Join(parseService.parseFixture, parseName)); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
	}
	parseResult, parseErr := parseService.Run(context.Background(), "not-an-action")
	if parseErr == nil || parseResult.Outcome != "error" {
		parseTest.Fatalf("unknown action: %#v %v", parseResult, parseErr)
	}
	parseReport, parseErr := parseService.GetReport(context.Background())
	if parseErr != nil || parseReport.FixtureDir == "" {
		parseTest.Fatalf("report: %#v %v", parseReport, parseErr)
	}
}

// TestAPIServiceCancelledAndBoundedObservations verifies cancellation, validation, and bounds.
func TestAPIServiceCancelledAndBoundedObservations(parseTest *testing.T) {
	parseService, parseErr := NewAPIService()
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	defer closeAPITestService(parseTest, parseService)
	parseContext, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	parseResult, parseErr := parseService.Run(parseContext, "window-info")
	if parseErr == nil || parseResult.Outcome != "cancelled" {
		parseTest.Fatalf("cancelled run: %#v %v", parseResult, parseErr)
	}
	if _, parseErr = parseService.RecordObservation(context.Background(), "id", "invalid", "detail"); parseErr == nil {
		parseTest.Fatal("expected invalid outcome")
	}
	if _, parseErr = parseService.RecordObservation(context.Background(), "id", "observed-pass", strings.Repeat("x", 2049)); parseErr == nil {
		parseTest.Fatal("expected detail bound")
	}
	for parseIndex := 0; parseIndex < apiMaxResults+8; parseIndex++ {
		if _, parseErr = parseService.RecordObservation(context.Background(), "id", "not-tested", "ok"); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
	}
	parseReport, parseErr := parseService.GetReport(context.Background())
	if parseErr != nil || len(parseReport.Results) != apiMaxResults {
		parseTest.Fatalf("bounded report: %d %v", len(parseReport.Results), parseErr)
	}
}

// TestAPIServiceAppendResultBounds verifies result and path bounds.
func TestAPIServiceAppendResultBounds(parseTest *testing.T) {
	parseService, parseErr := NewAPIService()
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	defer closeAPITestService(parseTest, parseService)
	parseResult := parseService.appendAPIResult(contracts.APIResult{ID: strings.Repeat("x", 200), Detail: strings.Repeat("x", 3000), Paths: []string{"", strings.Repeat("p", apiMaxPathLen+1)}, Outcome: "completed"})
	if len(parseResult.ID) != 128 || len(parseResult.Detail) != 2048 || len(parseResult.Paths) != 1 || len(parseResult.Paths[0]) != apiMaxPathLen || parseResult.At == "" {
		parseTest.Fatalf("bounded result: %#v", parseResult)
	}
}

// TestAPIServiceNewestRing verifies that the newest observation displaces the oldest.
func TestAPIServiceNewestRing(parseTest *testing.T) {
	parseService, parseErr := NewAPIService()
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	defer closeAPITestService(parseTest, parseService)
	for parseIndex := 0; parseIndex <= apiMaxResults; parseIndex++ {
		parseService.appendAPIResult(contracts.APIResult{ID: "id-" + string(rune(parseIndex))})
	}
	parseReport, parseErr := parseService.GetReport(context.Background())
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if len(parseReport.Results) != apiMaxResults || parseReport.Results[0].ID != "id-\x01" || parseReport.Results[apiMaxResults-1].ID != "id-Ā" {
		parseTest.Fatalf("ring: first=%q last=%q", parseReport.Results[0].ID, parseReport.Results[apiMaxResults-1].ID)
	}
}

// TestAPIServiceSnapshotCopy verifies paths are copied out of the service.
func TestAPIServiceSnapshotCopy(parseTest *testing.T) {
	parseService, parseErr := NewAPIService()
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	defer closeAPITestService(parseTest, parseService)
	parseService.appendAPIResult(contracts.APIResult{ID: "copy", Paths: []string{"a"}})
	parseReport, parseErr := parseService.GetReport(context.Background())
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseReport.Results[0].Paths[0] = "mutated"
	parseAgain, _ := parseService.GetReport(context.Background())
	if parseAgain.Results[0].Paths[0] != "a" {
		parseTest.Fatal("snapshot was not deep copied")
	}
}

// TestAPIServiceMissingContextAndClose verifies nil contexts and idempotent shutdown.
func TestAPIServiceMissingContextAndClose(parseTest *testing.T) {
	parseService, parseErr := NewAPIService()
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if _, parseErr = parseService.GetReport(nil); parseErr == nil { //nolint:staticcheck // Deliberate invalid-context contract test.
		parseTest.Fatal("expected nil context error")
	}
	if parseErr = parseService.ServiceShutdown(); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseErr = parseService.ServiceShutdown(); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if _, parseErr = parseService.GetReport(context.Background()); parseErr == nil {
		parseTest.Fatal("expected closed service error")
	}
}

// closeAPITestService surfaces fixture cleanup failures in native tests.
func closeAPITestService(parseTest *testing.T, parseService *APIService) {
	parseTest.Helper()
	if parseErr := parseService.ServiceShutdown(); parseErr != nil {
		parseTest.Error(parseErr)
	}
}

// TestAPIServiceSelectionOutcome verifies empty picker selections are cancellations.
func TestAPIServiceSelectionOutcome(parseTest *testing.T) {
	if selectionOutcome(nil) != "cancelled" || selectionOutcome([]string{}) != "cancelled" || selectionOutcome([]string{"x"}) != "completed" {
		parseTest.Fatal("selection outcome mismatch")
	}
}

// TestAPISavePickerCancellationNeverClaimsSelection keeps manual cancellation evidence truthful.
func TestAPISavePickerCancellationNeverClaimsSelection(parseTest *testing.T) {
	parseResult := contracts.APIResult{Detail: "path selected; no file written", Paths: []string{"stale.json"}}
	if parseOutcome := applyPickerSelection(&parseResult, "save-path", desktop.FileSelection{Cancelled: true}); parseOutcome != "cancelled" {
		parseTest.Fatalf("outcome=%q", parseOutcome)
	}
	if parseResult.Detail != "" || len(parseResult.Paths) != 0 {
		parseTest.Fatalf("cancelled result retained selection evidence: %#v", parseResult)
	}
	if parseOutcome := applyPickerSelection(&parseResult, "save-path", desktop.FileSelection{Paths: []string{"report.json"}}); parseOutcome != "completed" || parseResult.Detail != "path selected; no file written" {
		parseTest.Fatalf("completed result=%#v outcome=%q", parseResult, parseOutcome)
	}
}

// TestAPIServiceExclusiveExport verifies an existing path is never overwritten.
func TestAPIServiceExclusiveExport(parseTest *testing.T) {
	parsePath := filepath.Join(parseTest.TempDir(), "report.json")
	if parseErr := os.WriteFile(parsePath, []byte("old"), 0o600); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseFile, parseErr := createExclusive(parsePath); parseErr == nil {
		_ = parseFile.Close()
		parseTest.Fatal("expected exclusive create failure")
	}
	parseData, _ := os.ReadFile(parsePath)
	if string(parseData) != "old" {
		parseTest.Fatal("existing export changed")
	}
}
