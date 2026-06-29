package deprecation

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/diagnostics"
)

// captureEmit replaces the package-level emit var with a function that appends
// every emitted report to a slice, then restores the original on cleanup.
func captureEmit(t *testing.T) *[]diagnostics.Report {
	t.Helper()
	parseCapture := &[]diagnostics.Report{}
	parseOrig := emit
	emit = func(parseReport diagnostics.Report) {
		*parseCapture = append(*parseCapture, parseReport)
	}
	t.Cleanup(func() { emit = parseOrig })
	return parseCapture
}

// captureNewReport replaces the package-level newReport var so tests can
// inspect the Options used to build the report without relying on Report fields
// that may be transformed by diagnostics internals.
func captureNewReport(t *testing.T) *[]diagnostics.Options {
	t.Helper()
	parseCaptured := &[]diagnostics.Options{}
	parseOrig := newReport
	newReport = func(parseOpts diagnostics.Options) diagnostics.Report {
		*parseCaptured = append(*parseCaptured, parseOpts)
		return diagnostics.Report{}
	}
	t.Cleanup(func() { newReport = parseOrig })
	return parseCaptured
}

func TestWarnEmitsExactlyOnce(t *testing.T) {
	resetForTest()
	parseReports := captureEmit(t)
	_ = captureNewReport(t)

	Warn("old.Foo", "new.Foo")
	Warn("old.Foo", "new.Foo")

	if len(*parseReports) != 1 {
		t.Fatalf("want 1 emit call, got %d", len(*parseReports))
	}
}

func TestWarnEmittedTrueAfterFirstCall(t *testing.T) {
	resetForTest()
	_ = captureEmit(t)
	_ = captureNewReport(t)

	if WarnEmitted("old.Bar") {
		t.Fatal("expected WarnEmitted false before Warn is called")
	}
	Warn("old.Bar", "new.Bar")
	if !WarnEmitted("old.Bar") {
		t.Fatal("expected WarnEmitted true after Warn is called")
	}
}

func TestWarnReportCarriesExpectedFields(t *testing.T) {
	resetForTest()
	_ = captureEmit(t)
	parseOpts := captureNewReport(t)

	Warn("old.Baz", "new.Baz")

	if len(*parseOpts) != 1 {
		t.Fatalf("want 1 options capture, got %d", len(*parseOpts))
	}
	parseOpt := (*parseOpts)[0]
	if parseOpt.Code != "GWC-DEPRECATION" {
		t.Errorf("Code: want GWC-DEPRECATION, got %q", parseOpt.Code)
	}
	if parseOpt.Path != "old.Baz" {
		t.Errorf("Path: want old.Baz, got %q", parseOpt.Path)
	}
	if parseOpt.Next != "use new.Baz instead" {
		t.Errorf("Next: want %q, got %q", "use new.Baz instead", parseOpt.Next)
	}
}

func TestWarnOmitsNextWhenReplacementEmpty(t *testing.T) {
	resetForTest()
	_ = captureEmit(t)
	parseOpts := captureNewReport(t)

	Warn("old.NoReplacement", "")

	if len(*parseOpts) != 1 {
		t.Fatalf("want 1 options capture, got %d", len(*parseOpts))
	}
	if (*parseOpts)[0].Next != "" {
		t.Errorf("Next: want empty string, got %q", (*parseOpts)[0].Next)
	}
}

func TestWarnIndependentAPIsWarnSeparately(t *testing.T) {
	resetForTest()
	parseReports := captureEmit(t)
	_ = captureNewReport(t)

	Warn("api.Alpha", "api.AlphaV2")
	Warn("api.Beta", "api.BetaV2")
	Warn("api.Alpha", "api.AlphaV2") // duplicate, should be suppressed

	if len(*parseReports) != 2 {
		t.Fatalf("want 2 emit calls for two distinct APIs, got %d", len(*parseReports))
	}
	if !WarnEmitted("api.Alpha") {
		t.Error("expected WarnEmitted true for api.Alpha")
	}
	if !WarnEmitted("api.Beta") {
		t.Error("expected WarnEmitted true for api.Beta")
	}
}
