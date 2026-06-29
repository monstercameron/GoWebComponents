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

func TestOTELLoggerRedactsSensitiveLogFields(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, serverServiceName)
	parseLogger.Error(
		"auth failed: Bearer abc.def.ghi",
		slog.String("password", "super-secret-password"),
		slog.String("authorization", "Bearer xyz.123"),
		slog.String("provider_payload", `{"api_key":"sk-test"}`),
		slog.Int("token_version", 3),
	)

	parseLogLine := strings.TrimSpace(parseOutput.String())
	if parseLogLine == "" {
		parseT.Fatal("expected one log line")
	}

	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parseLogLine), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	if parsePayload["password"] != parseLogRedactionText {
		parseT.Fatalf("password = %v, want %s", parsePayload["password"], parseLogRedactionText)
	}
	if parsePayload["authorization"] != parseLogRedactionText {
		parseT.Fatalf("authorization = %v, want %s", parsePayload["authorization"], parseLogRedactionText)
	}
	if parsePayload["provider_payload"] != parseLogRedactionText {
		parseT.Fatalf("provider_payload = %v, want %s", parsePayload["provider_payload"], parseLogRedactionText)
	}
	if parsePayload["token_version"] != float64(3) {
		parseT.Fatalf("token_version = %v, want 3", parsePayload["token_version"])
	}
	parseMessage, _ := parsePayload["message"].(string)
	if strings.Contains(parseMessage, "abc.def.ghi") {
		parseT.Fatalf("expected bearer token redaction in message, got %q", parseMessage)
	}
	if !strings.Contains(parseMessage, parseLogRedactionText) {
		parseT.Fatalf("expected redaction marker in message, got %q", parseMessage)
	}
}

func TestOTELLoggerRedactsSensitiveStructuredAnyPayloads(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, serverServiceName)
	parseLogger.Info("client payload", slog.Any("attributes", map[string]any{
		"api_key":       "sk-live-123",
		"internal_note": "private escalation details",
		"safe":          "visible",
	}))

	parseLogLine := strings.TrimSpace(parseOutput.String())
	if parseLogLine == "" {
		parseT.Fatal("expected one log line")
	}

	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parseLogLine), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	parseAttributesValue, parseOk := parsePayload["attributes"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("attributes = %#v, want map", parsePayload["attributes"])
	}
	if parseAttributesValue["api_key"] != parseLogRedactionText {
		parseT.Fatalf("attributes.api_key = %v, want %s", parseAttributesValue["api_key"], parseLogRedactionText)
	}
	if parseAttributesValue["internal_note"] != parseLogRedactionText {
		parseT.Fatalf("attributes.internal_note = %v, want %s", parseAttributesValue["internal_note"], parseLogRedactionText)
	}
	if parseAttributesValue["safe"] != "visible" {
		parseT.Fatalf("attributes.safe = %v, want visible", parseAttributesValue["safe"])
	}
}

// TestScrubSecretStringRedactsCommonSecrets verifies the shared scrubber removes bearer tokens, API keys, cookies, and file paths.
func TestScrubSecretStringRedactsCommonSecrets(parseT *testing.T) {
	parseInput := `open C:\secrets\config.json with OPENAI_API_KEY=sk-live-123; Authorization: Bearer abc.def.ghi; cookie=session=abc`
	parseOutput := parseScrubSecretString(parseInput)
	if strings.Contains(parseOutput, "C:\\secrets\\config.json") {
		parseT.Fatalf("expected file path redaction, got %q", parseOutput)
	}
	if strings.Contains(parseOutput, "OPENAI_API_KEY") || strings.Contains(parseOutput, "sk-live-123") {
		parseT.Fatalf("expected env secret redaction, got %q", parseOutput)
	}
	if strings.Contains(parseOutput, "abc.def.ghi") || strings.Contains(parseOutput, "session=abc") {
		parseT.Fatalf("expected token redaction, got %q", parseOutput)
	}
	if strings.Count(parseOutput, parseLogRedactionText) < 3 {
		parseT.Fatalf("expected multiple redaction markers, got %q", parseOutput)
	}
}

// BenchmarkScrubSecretString measures the shared secret-scrubbing helper overhead.
func BenchmarkScrubSecretString(parseB *testing.B) {
	parseInput := `open C:\secrets\config.json with OPENAI_API_KEY=sk-live-123; Authorization: Bearer abc.def.ghi; cookie=session=abc`
	parseB.ResetTimer()
	for parseN := 0; parseN < parseB.N; parseN++ {
		_ = parseScrubSecretString(parseInput)
	}
}
