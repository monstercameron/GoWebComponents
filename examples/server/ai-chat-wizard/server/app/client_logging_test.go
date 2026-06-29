package app

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/metadata"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestGetClientIdentityReturnsMetadataValue verifies metadata-provided client identity passthrough.
func TestGetClientIdentityReturnsMetadataValue(parseT *testing.T) {
	parseExpectedClientID := uuid.NewString()
	parseServer := &chatServer{
		logger:       parseNewTestLogger(),
		clientLogger: parseNewTestLogger(),
	}
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(clientMetadataKey, parseExpectedClientID))
	parseResp, parseErr := parseServer.GetClientIdentity(parseCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetClientIdentity: %v", parseErr)
	}
	if parseResp.GetClientId() != parseExpectedClientID {
		parseT.Fatalf("client identity = %q, want %q", parseResp.GetClientId(), parseExpectedClientID)
	}
}

// TestReportClientLogWritesStructuredRecord verifies structured client log forwarding into OTEL JSON output.
func TestReportClientLogWritesStructuredRecord(parseT *testing.T) {
	var parseLogOutput bytes.Buffer
	parseServer := &chatServer{
		logger: parseNewTestLogger(),
		clientLogger: parseNewOTELLogger(&parseLogOutput, clientLogServiceName).With(
			slog.String("log.source", "client"),
			slog.String("event.dataset", "chat-wizard.client"),
		),
	}

	parseObservedAt := time.Date(2026, time.March, 27, 14, 10, 2, 0, time.UTC)
	parseClientID := uuid.NewString()
	parseFields, parseErr := structpb.NewStruct(map[string]any{
		"route":   "/app",
		"attempt": float64(3),
		"nested": map[string]any{
			"reason": "bridge reconnect",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("structpb.NewStruct: %v", parseErr)
	}
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		clientMetadataKey, parseClientID,
		traceParentMetadataKey, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		traceStateMetadataKey, "vendor=relay",
		requestIDMetadataKey, "req-bridge-123",
		correlationIDMetadataKey, "corr-bridge-456",
	))
	_, parseErr2 := parseServer.ReportClientLog(parseCtx, &chatpb.ReportClientLogRequest{
		ClientId:   parseClientID,
		Level:      "warn",
		Scope:      "chat-wizard",
		Message:    "bridge state changed",
		Fields:     parseFields,
		ObservedAt: timestamppb.New(parseObservedAt),
	})
	if parseErr2 != nil {
		parseT.Fatalf("ReportClientLog: %v", parseErr2)
	}

	var parsePayload map[string]any
	if parseErr3 := json.Unmarshal(bytes.TrimSpace(parseLogOutput.Bytes()), &parsePayload); parseErr3 != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr3)
	}
	if parsePayload["service.name"] != clientLogServiceName {
		parseT.Fatalf("service.name = %v, want %q", parsePayload["service.name"], clientLogServiceName)
	}
	if parsePayload["severity_text"] != "WARN" {
		parseT.Fatalf("severity_text = %v, want WARN", parsePayload["severity_text"])
	}
	if parsePayload["message"] != "bridge state changed" {
		parseT.Fatalf("message = %v, want bridge state changed", parsePayload["message"])
	}
	if parsePayload["log.source"] != "client" {
		parseT.Fatalf("log.source = %v, want client", parsePayload["log.source"])
	}
	if parsePayload["event.dataset"] != "chat-wizard.client" {
		parseT.Fatalf("event.dataset = %v, want chat-wizard.client", parsePayload["event.dataset"])
	}
	if parsePayload["client.id"] != parseClientID {
		parseT.Fatalf("client.id = %v, want %q", parsePayload["client.id"], parseClientID)
	}
	if parsePayload["trace.id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		parseT.Fatalf("trace.id = %v, want parsed trace id", parsePayload["trace.id"])
	}
	if parsePayload["span.id"] != "00f067aa0ba902b7" {
		parseT.Fatalf("span.id = %v, want parsed span id", parsePayload["span.id"])
	}
	if parsePayload["trace.state"] != "vendor=relay" {
		parseT.Fatalf("trace.state = %v, want vendor=relay", parsePayload["trace.state"])
	}
	if parsePayload["request.id"] != "req-bridge-123" {
		parseT.Fatalf("request.id = %v, want req-bridge-123", parsePayload["request.id"])
	}
	if parsePayload["correlation.id"] != "corr-bridge-456" {
		parseT.Fatalf("correlation.id = %v, want corr-bridge-456", parsePayload["correlation.id"])
	}
	parseAttributes, parseOk := parsePayload["attributes"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("attributes = %T, want map[string]any", parsePayload["attributes"])
	}
	if parseAttributes["route"] != "/app" {
		parseT.Fatalf("attributes.route = %v, want /app", parseAttributes["route"])
	}
}

// TestResolveClientLogLevelMapping verifies textual level mapping into slog levels.
func TestResolveClientLogLevelMapping(parseT *testing.T) {
	parseCases := []struct {
		parseName  string
		parseInput string
		parseWant  slog.Level
	}{
		{parseName: "debug", parseInput: "debug", parseWant: slog.LevelDebug},
		{parseName: "warn", parseInput: "warn", parseWant: slog.LevelWarn},
		{parseName: "warning", parseInput: "warning", parseWant: slog.LevelWarn},
		{parseName: "error", parseInput: "error", parseWant: slog.LevelError},
		{parseName: "default", parseInput: "info-ish", parseWant: slog.LevelInfo},
		{parseName: "trimmed", parseInput: "  ERROR ", parseWant: slog.LevelError},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.parseName, func(parseT2 *testing.T) {
			if parseGot := parseResolveClientLogLevel(parseCase.parseInput); parseGot != parseCase.parseWant {
				parseT2.Fatalf("parseResolveClientLogLevel(%q) = %v, want %v", parseCase.parseInput, parseGot, parseCase.parseWant)
			}
		})
	}
}

// TestResolveClientIdentityFromRequestPrefersMetadata verifies metadata precedence over request payload identity.
func TestResolveClientIdentityFromRequestPrefersMetadata(parseT *testing.T) {
	parseMetadataClientID := uuid.NewString()
	parseRequestClientID := uuid.NewString()
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(clientMetadataKey, parseMetadataClientID))
	parseResolvedClientID := parseResolveClientIdentityFromRequest(parseCtx, parseRequestClientID)
	if parseResolvedClientID != parseMetadataClientID {
		parseT.Fatalf("parseResolveClientIdentityFromRequest() = %q, want metadata %q", parseResolvedClientID, parseMetadataClientID)
	}
}

// TestNormalizeClientIdentityValidatesUUID verifies UUID normalization and validation.
func TestNormalizeClientIdentityValidatesUUID(parseT *testing.T) {
	parseValidClientID := uuid.NewString()
	if parseGot := parseNormalizeClientIdentity("  " + parseValidClientID + "  "); parseGot != parseValidClientID {
		parseT.Fatalf("parseNormalizeClientIdentity(valid) = %q, want %q", parseGot, parseValidClientID)
	}
	if parseGot2 := parseNormalizeClientIdentity("not-a-uuid"); parseGot2 != "" {
		parseT.Fatalf("parseNormalizeClientIdentity(invalid) = %q, want blank", parseGot2)
	}
	if parseGot3 := parseNormalizeClientIdentity("   "); parseGot3 != "" {
		parseT.Fatalf("parseNormalizeClientIdentity(blank) = %q, want blank", parseGot3)
	}
}

// TestAppendStructuredClientLogFieldsAddsAttributesObject verifies standardized structured attributes payload.
func TestAppendStructuredClientLogFieldsAddsAttributesObject(parseT *testing.T) {
	parseFields, parseErr := structpb.NewStruct(map[string]any{
		"zeta":  float64(1),
		"alpha": "x",
	})
	if parseErr != nil {
		parseT.Fatalf("structpb.NewStruct: %v", parseErr)
	}
	parseAttrs := parseAppendStructuredClientLogFields(nil, parseFields)
	if len(parseAttrs) != 1 {
		parseT.Fatalf("structured attrs length = %d, want 1", len(parseAttrs))
	}
	parseAttr, parseOk2 := parseAttrs[0].(slog.Attr)
	if !parseOk2 {
		parseT.Fatalf("structured attr type = %T, want slog.Attr", parseAttrs[0])
	}
	if parseAttr.Key != "attributes" {
		parseT.Fatalf("structured attr key = %q, want attributes", parseAttr.Key)
	}
}

// TestReportClientLogErrorAddsBoundaryFromScope verifies error-level client logs always include a stable boundary.
func TestReportClientLogErrorAddsBoundaryFromScope(parseT *testing.T) {
	var parseLogOutput bytes.Buffer
	parseServer := &chatServer{
		logger: parseNewTestLogger(),
		clientLogger: parseNewOTELLogger(&parseLogOutput, clientLogServiceName).With(
			slog.String("log.source", "client"),
			slog.String("event.dataset", "chat-wizard.client"),
		),
	}

	parseClientID := uuid.NewString()
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(clientMetadataKey, parseClientID))
	_, parseErr := parseServer.ReportClientLog(parseCtx, &chatpb.ReportClientLogRequest{
		ClientId: parseClientID,
		Level:    "error",
		Scope:    "chat-wizard",
		Message:  "send failed",
	})
	if parseErr != nil {
		parseT.Fatalf("ReportClientLog: %v", parseErr)
	}

	var parsePayload map[string]any
	if parseErr2 := json.Unmarshal(bytes.TrimSpace(parseLogOutput.Bytes()), &parsePayload); parseErr2 != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr2)
	}
	if parsePayload["error.boundary"] != "chat-wizard" {
		parseT.Fatalf("error.boundary = %v, want chat-wizard", parsePayload["error.boundary"])
	}
}

// TestFormatFallbackClientLogMessageIncludesContext verifies fallback log messages include key filter dimensions.
func TestFormatFallbackClientLogMessageIncludesContext(parseT *testing.T) {
	parseMessage := parseFormatFallbackClientLogMessage("chat-wizard", slog.LevelWarn, "abc-123")
	if !strings.Contains(parseMessage, "level=warn") {
		parseT.Fatalf("fallback message = %q, want level=warn", parseMessage)
	}
	if !strings.Contains(parseMessage, "scope=chat-wizard") {
		parseT.Fatalf("fallback message = %q, want scope=chat-wizard", parseMessage)
	}
	if !strings.Contains(parseMessage, "client.id=abc-123") {
		parseT.Fatalf("fallback message = %q, want client.id=abc-123", parseMessage)
	}
}
