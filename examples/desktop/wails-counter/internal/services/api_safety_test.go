package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"example.com/gwc-wails-counter/contracts"
)

// TestAPIResultBoundsPreserveUnicode verifies truncation cannot corrupt Unicode paths or reports.
func TestAPIResultBoundsPreserveUnicode(parseTest *testing.T) {
	parseResult := boundResult(contracts.APIResult{ID: strings.Repeat("雪", 60), Detail: strings.Repeat("雪", 1000), Paths: []string{strings.Repeat("雪", 2000)}})
	if !utf8.ValidString(parseResult.ID) || !utf8.ValidString(parseResult.Detail) || !utf8.ValidString(parseResult.Paths[0]) {
		parseTest.Fatal("report truncation split a UTF-8 character")
	}
}

// TestAPIPickerCancellationCompatibility recognizes only the pinned Windows leaf error.
func TestAPIPickerCancellationCompatibility(parseTest *testing.T) {
	if runtime.GOOS != "windows" {
		parseTest.Skip("Windows-only pinned CFD adapter")
	}
	parseCancel := errors.New("cancelled by user")
	for _, parseErr := range []error{parseCancel, fmt.Errorf("picker: %w", parseCancel)} {
		if !isAPIPickerCancelled(parseErr) {
			parseTest.Fatal("explicit cancellation not recognized")
		}
		parseResult, parseReturned := (&APIService{}).finishPickerError(contracts.APIResult{ID: "open-file"}, parseErr)
		if parseReturned != nil || parseResult.Outcome != "cancelled" {
			parseTest.Fatalf("cancel outcome: %+v %v", parseResult, parseReturned)
		}
	}
	for _, parseErr := range []error{nil, context.Canceled, errors.New("access denied"), errors.New("not cancelled by user"), errors.Join(parseCancel, errors.New("I/O failed"))} {
		if isAPIPickerCancelled(parseErr) {
			parseTest.Fatalf("real error masked: %v", parseErr)
		}
	}
}

// TestAPIReportExportSafety verifies durable paths, cancellation and no-overwrite behavior.
func TestAPIReportExportSafety(parseTest *testing.T) {
	parseFixture := parseTest.TempDir()
	parseOutput := parseTest.TempDir()
	parsePath := filepath.Join(parseOutput, "report.json")
	parseContext, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if parseErr := writeAPIReport(parseContext, parseFixture, parsePath, []byte("cancelled")); !errors.Is(parseErr, context.Canceled) {
		parseTest.Fatal(parseErr)
	}
	if _, parseErr := os.Stat(parsePath); !errors.Is(parseErr, os.ErrNotExist) {
		parseTest.Fatal("cancelled export created a file")
	}
	if parseErr := writeAPIReport(context.Background(), parseFixture, parsePath, []byte("kept")); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseErr := writeAPIReport(context.Background(), parseFixture, parsePath, []byte("overwritten")); !errors.Is(parseErr, os.ErrExist) {
		parseTest.Fatalf("existing path: %v", parseErr)
	}
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil || string(parseData) != "kept" {
		parseTest.Fatalf("existing export changed: %v", parseErr)
	}
	for _, parseInvalid := range []string{filepath.Join(parseFixture, "report.json"), filepath.Join(parseFixture, "..", filepath.Base(parseFixture), "report.json"), filepath.Join(parseOutput, "report.json."), filepath.Join(parseOutput, "report:alternate.json"), filepath.Join(parseOutput, "report.txt")} {
		if _, parseErr := validateAPIReportPath(parseFixture, parseInvalid); parseErr == nil {
			parseTest.Fatalf("accepted unsafe export path %s", parseInvalid)
		}
	}
	if runtime.GOOS == "windows" {
		if _, parseErr := validateAPIReportPath(parseFixture, filepath.Join(strings.ToUpper(parseFixture), "report.json")); parseErr == nil {
			parseTest.Fatal("case alias bypassed fixture boundary")
		}
		if _, parseErr := validateAPIReportPath(parseFixture, filepath.Join(parseOutput, "NUL.json")); parseErr == nil {
			parseTest.Fatal("Windows device filename accepted")
		}
	}
}

// TestAPIReportExportRejectsFixtureLink verifies resolved parent paths, when link creation is permitted.
func TestAPIReportExportRejectsFixtureLink(parseTest *testing.T) {
	parseFixture := parseTest.TempDir()
	parseAlias := filepath.Join(parseTest.TempDir(), "fixture-link")
	if parseErr := os.Symlink(parseFixture, parseAlias); parseErr != nil {
		parseTest.Skipf("symlink privilege unavailable: %v", parseErr)
	}
	if _, parseErr := validateAPIReportPath(parseFixture, filepath.Join(parseAlias, "report.json")); parseErr == nil {
		parseTest.Fatal("fixture symlink alias accepted")
	}
}

// TestAPIResultReturnDoesNotAliasReport verifies mutation of a returned result cannot rewrite evidence.
func TestAPIResultReturnDoesNotAliasReport(parseTest *testing.T) {
	parseService := &APIService{}
	parseResult := parseService.appendAPIResult(contracts.APIResult{ID: "copy", Paths: []string{"kept"}})
	parseResult.Paths[0] = "changed"
	parseReport, parseErr := parseService.GetReport(context.Background())
	if parseErr != nil || parseReport.Results[0].Paths[0] != "kept" {
		parseTest.Fatalf("aliased result: %+v %v", parseReport, parseErr)
	}
}
