package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestOTELSeverityNumberMapping(parseT *testing.T) {
	if parseGot := parseOtelSeverityNumber(-4); parseGot != 5 {
		parseT.Fatalf("otelSeverityNumber(debug) = %d, want 5", parseGot)
	}
	if parseGot2 := parseOtelSeverityNumber(0); parseGot2 != 9 {
		parseT.Fatalf("otelSeverityNumber(info) = %d, want 9", parseGot2)
	}
	if parseGot3 := parseOtelSeverityNumber(4); parseGot3 != 13 {
		parseT.Fatalf("otelSeverityNumber(warn) = %d, want 13", parseGot3)
	}
	if parseGot4 := parseOtelSeverityNumber(8); parseGot4 != 17 {
		parseT.Fatalf("otelSeverityNumber(error) = %d, want 17", parseGot4)
	}
}

func TestErrorBoundaryFromMessage(parseT *testing.T) {
	if parseGot := parseErrorBoundaryFromMessage("rpc.Send: provider stream error"); parseGot != "rpc.Send" {
		parseT.Fatalf("errorBoundaryFromMessage() = %q, want rpc.Send", parseGot)
	}
	if parseGot2 := parseErrorBoundaryFromMessage("single-boundary"); parseGot2 != "single-boundary" {
		parseT.Fatalf("errorBoundaryFromMessage() = %q, want single-boundary", parseGot2)
	}
	if parseGot3 := parseErrorBoundaryFromMessage("send failed"); parseGot3 != "" {
		parseT.Fatalf("errorBoundaryFromMessage() = %q, want empty for free-form phrase", parseGot3)
	}
}

func TestOTELLoggerAddsBoundaryFieldsForErrors(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, serverServiceName)
	parseLogger.Error("rpc.Send: provider stream error", slog.String("error", "provider timeout"))

	parseLogLine := strings.TrimSpace(parseOutput.String())
	if parseLogLine == "" {
		parseT.Fatal("expected one log line")
	}

	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parseLogLine), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	if parsePayload["severity_text"] != "ERROR" {
		parseT.Fatalf("severity_text = %v, want ERROR", parsePayload["severity_text"])
	}
	if parsePayload["severity_number"] != float64(17) {
		parseT.Fatalf("severity_number = %v, want 17", parsePayload["severity_number"])
	}
	if parsePayload["error.boundary"] != "rpc.Send" {
		parseT.Fatalf("error.boundary = %v, want rpc.Send", parsePayload["error.boundary"])
	}
	if parsePayload["error.message"] != "provider timeout" {
		parseT.Fatalf("error.message = %v, want provider timeout", parsePayload["error.message"])
	}
	if parsePayload["service.name"] != serverServiceName {
		parseT.Fatalf("service.name = %v, want %s", parsePayload["service.name"], serverServiceName)
	}
}

func TestOTELLoggerAddsErrorTypeForErrorValues(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, serverServiceName)
	parseLogger.Error("db: failed to open", slog.Any("error", errors.New("permission denied")))

	parseLogLine := strings.TrimSpace(parseOutput.String())
	if parseLogLine == "" {
		parseT.Fatal("expected one log line")
	}

	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parseLogLine), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	if parsePayload["error.message"] != "permission denied" {
		parseT.Fatalf("error.message = %v, want permission denied", parsePayload["error.message"])
	}
	if parsePayload["error.type"] == "" {
		parseT.Fatalf("expected error.type to be populated, got %v", parsePayload["error.type"])
	}
}

// TestOTELLoggerPreservesExplicitErrorFields verifies explicit error fields are not overwritten by enrichment.
func TestOTELLoggerPreservesExplicitErrorFields(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, serverServiceName)
	parseLogger.Error("rpc.Send: provider stream error",
		slog.String("error.boundary", "relay.custom"),
		slog.String("error.message", "already normalized"),
		slog.String("error", "provider timeout"),
	)

	parseLogLine := strings.TrimSpace(parseOutput.String())
	if parseLogLine == "" {
		parseT.Fatal("expected one log line")
	}

	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parseLogLine), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	if parsePayload["error.boundary"] != "relay.custom" {
		parseT.Fatalf("error.boundary = %v, want relay.custom", parsePayload["error.boundary"])
	}
	if parsePayload["error.message"] != "already normalized" {
		parseT.Fatalf("error.message = %v, want already normalized", parsePayload["error.message"])
	}
}

// TestOTELLoggerDerivesBoundaryFromScope verifies scope fallback for error boundaries when message lacks one.
func TestOTELLoggerDerivesBoundaryFromScope(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, serverServiceName)
	parseLogger.Error("send failed",
		slog.String("log.scope", "chat-wizard"),
		slog.String("error", "stream recv failed"),
	)

	parseLogLine := strings.TrimSpace(parseOutput.String())
	if parseLogLine == "" {
		parseT.Fatal("expected one log line")
	}

	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parseLogLine), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	if parsePayload["error.boundary"] != "chat-wizard" {
		parseT.Fatalf("error.boundary = %v, want chat-wizard", parsePayload["error.boundary"])
	}
}
