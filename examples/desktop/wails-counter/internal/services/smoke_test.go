package services

import (
	"testing"

	"example.com/gwc-wails-counter/contracts"
)

var parseSmokeChecks = []string{"dom", "local-counter", "native-call", "native-error", "native-event", "routing", "route-remount", "wasm-mime", "csp", "backend-cancellation", "unmount-cancellation", "adapter-cleanup", "native-handshake", "origin-guard", "durable-native-state", "two-window-invalidation", "window-local-state", "api-tester-ui", "api-window-info", "api-session-report", "desktop-file-contract", "native-sdk-contract"}

// parseCompleteSmokeReport builds the required success evidence for validation tests.
func parseCompleteSmokeReport() contracts.SmokeReport {
	return contracts.SmokeReport{OK: true, Checks: append([]string(nil), parseSmokeChecks...)}
}

// TestSmokeServiceRejectsIncompleteSuccess verifies success requires every
// native WebView check and leaves the reporter available for a later report.
func TestSmokeServiceRejectsIncompleteSuccess(parseT *testing.T) {
	parseResults := make(chan contracts.SmokeReport, 1)
	parseService := NewSmokeService(parseResults)
	parseReport := parseCompleteSmokeReport()
	parseReport.Checks = parseReport.Checks[:len(parseReport.Checks)-1]
	if parseErr := parseService.ReportResult(parseReport); parseErr == nil {
		parseT.Fatal("incomplete success report was accepted")
	}
	if parseErr := parseService.ReportResult(parseCompleteSmokeReport()); parseErr != nil {
		parseT.Fatalf("complete report after rejected incomplete report: %v", parseErr)
	}
}

// TestSmokeServiceAcceptsFailureReport verifies an explicit failure is valid
// evidence and is delivered exactly once.
func TestSmokeServiceAcceptsFailureReport(parseT *testing.T) {
	parseResults := make(chan contracts.SmokeReport, 1)
	parseService := NewSmokeService(parseResults)
	parseReport := contracts.SmokeReport{Error: "native call failed"}
	if parseErr := parseService.ReportResult(parseReport); parseErr != nil {
		parseT.Fatalf("failure report rejected: %v", parseErr)
	}
	if parseReceived := <-parseResults; parseReceived.Error != parseReport.Error || parseReceived.OK {
		parseT.Fatalf("received failure report = %#v, want %#v", parseReceived, parseReport)
	}
}

// TestSmokeServiceAcceptsCompleteReportOnce verifies duplicate reports are
// rejected after the first report reaches the receiver.
func TestSmokeServiceAcceptsCompleteReportOnce(parseT *testing.T) {
	parseResults := make(chan contracts.SmokeReport, 2)
	parseService := NewSmokeService(parseResults)
	parseReport := parseCompleteSmokeReport()
	if parseErr := parseService.ReportResult(parseReport); parseErr != nil {
		parseT.Fatalf("complete report rejected: %v", parseErr)
	}
	if parseErr := parseService.ReportResult(parseReport); parseErr == nil {
		parseT.Fatal("duplicate report was accepted")
	}
	if parseCount := len(parseResults); parseCount != 1 {
		parseT.Fatalf("result channel contains %d reports, want 1", parseCount)
	}
}

// TestSmokeServiceAcceptsExplicitDisabledFeatureEvidence verifies a restricted
// host cannot claim persistence or native-window coverage while still proving
// the rendered app, report service, and file policy are alive.
func TestSmokeServiceAcceptsExplicitDisabledFeatureEvidence(parseT *testing.T) {
	parseResults := make(chan contracts.SmokeReport, 1)
	parseService := NewSmokeService(parseResults)
	parseReport := parseCompleteSmokeReport()
	parseReport.Checks = []string{}
	for _, parseCheck := range parseSmokeChecks {
		if parseCheck != "durable-native-state" && parseCheck != "two-window-invalidation" && parseCheck != "window-local-state" && parseCheck != "api-window-info" {
			parseReport.Checks = append(parseReport.Checks, parseCheck)
		}
	}
	parseReport.Checks = append(parseReport.Checks, "persistent-storage-disabled", "native-window-disabled")
	if parseErr := parseService.ReportResult(parseReport); parseErr != nil {
		parseT.Fatalf("restricted report rejected: %v", parseErr)
	}
}

// TestSmokeServiceDoesNotAcceptUnavailableOrFullReceiver verifies delivery
// failure never marks a report as accepted.
func TestSmokeServiceDoesNotAcceptUnavailableOrFullReceiver(parseT *testing.T) {
	parseT.Run("unavailable", func(parseT *testing.T) {
		parseService := NewSmokeService(nil)
		if parseErr := parseService.ReportResult(parseCompleteSmokeReport()); parseErr == nil {
			parseT.Fatal("report accepted with unavailable receiver")
		}
	})
	parseT.Run("full", func(parseT *testing.T) {
		parseResults := make(chan contracts.SmokeReport, 1)
		parseResults <- contracts.SmokeReport{Error: "occupied"}
		parseService := NewSmokeService(parseResults)
		if parseErr := parseService.ReportResult(parseCompleteSmokeReport()); parseErr == nil {
			parseT.Fatal("report accepted with full receiver")
		}
		<-parseResults
		if parseErr := parseService.ReportResult(parseCompleteSmokeReport()); parseErr != nil {
			parseT.Fatalf("report not retryable after receiver became available: %v", parseErr)
		}
	})
}
