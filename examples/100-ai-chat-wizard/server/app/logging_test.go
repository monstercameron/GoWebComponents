package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestOTELSeverityNumberMapping(t *testing.T) {
	if got := otelSeverityNumber(-4); got != 5 {
		t.Fatalf("otelSeverityNumber(debug) = %d, want 5", got)
	}
	if got := otelSeverityNumber(0); got != 9 {
		t.Fatalf("otelSeverityNumber(info) = %d, want 9", got)
	}
	if got := otelSeverityNumber(4); got != 13 {
		t.Fatalf("otelSeverityNumber(warn) = %d, want 13", got)
	}
	if got := otelSeverityNumber(8); got != 17 {
		t.Fatalf("otelSeverityNumber(error) = %d, want 17", got)
	}
}

func TestErrorBoundaryFromMessage(t *testing.T) {
	if got := errorBoundaryFromMessage("rpc.Send: provider stream error"); got != "rpc.Send" {
		t.Fatalf("errorBoundaryFromMessage() = %q, want rpc.Send", got)
	}
	if got := errorBoundaryFromMessage("single-boundary"); got != "single-boundary" {
		t.Fatalf("errorBoundaryFromMessage() = %q, want single-boundary", got)
	}
}

func TestOTELLoggerAddsBoundaryFieldsForErrors(t *testing.T) {
	var output bytes.Buffer
	logger := newOTELLogger(&output, serverServiceName)
	logger.Error("rpc.Send: provider stream error", slog.String("error", "provider timeout"))

	logLine := strings.TrimSpace(output.String())
	if logLine == "" {
		t.Fatal("expected one log line")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(logLine), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if payload["severity_text"] != "ERROR" {
		t.Fatalf("severity_text = %v, want ERROR", payload["severity_text"])
	}
	if payload["severity_number"] != float64(17) {
		t.Fatalf("severity_number = %v, want 17", payload["severity_number"])
	}
	if payload["error.boundary"] != "rpc.Send" {
		t.Fatalf("error.boundary = %v, want rpc.Send", payload["error.boundary"])
	}
	if payload["error.message"] != "provider timeout" {
		t.Fatalf("error.message = %v, want provider timeout", payload["error.message"])
	}
	if payload["service.name"] != serverServiceName {
		t.Fatalf("service.name = %v, want %s", payload["service.name"], serverServiceName)
	}
}

func TestOTELLoggerAddsErrorTypeForErrorValues(t *testing.T) {
	var output bytes.Buffer
	logger := newOTELLogger(&output, serverServiceName)
	logger.Error("db: failed to open", slog.Any("error", errors.New("permission denied")))

	logLine := strings.TrimSpace(output.String())
	if logLine == "" {
		t.Fatal("expected one log line")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(logLine), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if payload["error.message"] != "permission denied" {
		t.Fatalf("error.message = %v, want permission denied", payload["error.message"])
	}
	if payload["error.type"] == "" {
		t.Fatalf("expected error.type to be populated, got %v", payload["error.type"])
	}
}
