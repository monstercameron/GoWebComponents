package app

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TestSanitizePublicErrorHidesStorageInternals verifies SQL/storage details are removed from public internal errors.
func TestSanitizePublicErrorHidesStorageInternals(parseT *testing.T) {
	parseErr := parseSanitizePublicError(status.Error(codes.Internal, `set user name: sqlite: no such table user_profile`))
	if status.Code(parseErr) != codes.Internal {
		parseT.Fatalf("status code = %v, want Internal", status.Code(parseErr))
	}
	if parseErr == nil || status.Convert(parseErr).Message() != parsePublicErrorStorageMessage {
		parseT.Fatalf("sanitized message = %q, want %q", status.Convert(parseErr).Message(), parsePublicErrorStorageMessage)
	}
}

// TestResolveSanitizedPublicMessageHidesSensitiveValidationPayload verifies sensitive validation text is collapsed.
func TestResolveSanitizedPublicMessageHidesSensitiveValidationPayload(parseT *testing.T) {
	parseMessage := parseResolveSanitizedPublicMessage(codes.InvalidArgument, `api_key=sk-live-123 is invalid`)
	if parseMessage != "request validation failed" {
		parseT.Fatalf("sanitized message = %q, want request validation failed", parseMessage)
	}
}

// TestSanitizeProviderErrorMessageRedactsBearerTokens verifies provider errors never echo raw bearer values.
func TestSanitizeProviderErrorMessageRedactsBearerTokens(parseT *testing.T) {
	parseMessage := parseSanitizeProviderErrorMessage("openai", "upstream failed with Authorization: Bearer abc.def.ghi")
	if parseMessage != parsePublicErrorProviderMessage {
		parseT.Fatalf("provider sanitized message = %q, want %q", parseMessage, parsePublicErrorProviderMessage)
	}
	if strings.Contains(parseMessage, "abc.def.ghi") {
		parseT.Fatalf("expected bearer token redaction, got %q", parseMessage)
	}
}

// TestBuildSanitizedInternalStatusWrapsOperation verifies internal status wrappers stay sanitized.
func TestBuildSanitizedInternalStatusWrapsOperation(parseT *testing.T) {
	parseErr := parseBuildSanitizedInternalStatus(context.Background(), "set custom system prompt", status.Error(codes.Internal, `open C:\secrets\config.json: access denied`))
	if status.Code(parseErr) != codes.Internal {
		parseT.Fatalf("status code = %v, want Internal", status.Code(parseErr))
	}
	if parseMessage := status.Convert(parseErr).Message(); parseMessage != parsePublicErrorGenericMessage {
		parseT.Fatalf("sanitized message = %q, want %q", parseMessage, parsePublicErrorGenericMessage)
	}
}

// TestBuildErrorEnvelopeCarriesTraceability verifies the shared envelope keeps stable public and operator fields together.
func TestBuildErrorEnvelopeCarriesTraceability(parseT *testing.T) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-envelope-123",
		correlationIDMetadataKey, "corr-envelope-456",
		traceParentMetadataKey, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		traceStateMetadataKey, "vendor=relay",
	))
	parseEnvelope := parseBuildErrorEnvelope(
		parseCtx,
		codes.Internal,
		"open billing ledger",
		status.Error(codes.Internal, `open C:\secrets\ledger.db: access denied`),
		slog.LevelError,
	)

	if parseEnvelope.ParseCode != codes.Internal {
		parseT.Fatalf("ParseCode = %v, want Internal", parseEnvelope.ParseCode)
	}
	if parseEnvelope.ParsePublicMessage != parsePublicErrorGenericMessage {
		parseT.Fatalf("ParsePublicMessage = %q, want %q", parseEnvelope.ParsePublicMessage, parsePublicErrorGenericMessage)
	}
	if parseEnvelope.ParseMessageKey != "request_failed" {
		parseT.Fatalf("ParseMessageKey = %q, want request_failed", parseEnvelope.ParseMessageKey)
	}
	if parseEnvelope.ParseSeverity != slog.LevelError.String() {
		parseT.Fatalf("ParseSeverity = %q, want %q", parseEnvelope.ParseSeverity, slog.LevelError.String())
	}
	if parseEnvelope.ParseRequestID != "req-envelope-123" {
		parseT.Fatalf("ParseRequestID = %q, want req-envelope-123", parseEnvelope.ParseRequestID)
	}
	if parseEnvelope.ParseCorrelationID != "corr-envelope-456" {
		parseT.Fatalf("ParseCorrelationID = %q, want corr-envelope-456", parseEnvelope.ParseCorrelationID)
	}
	if parseEnvelope.ParseTraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		parseT.Fatalf("ParseTraceID = %q, want parsed trace id", parseEnvelope.ParseTraceID)
	}
	if parseEnvelope.ParseSpanID != "00f067aa0ba902b7" {
		parseT.Fatalf("ParseSpanID = %q, want parsed span id", parseEnvelope.ParseSpanID)
	}
	if parseEnvelope.ParseSupportID != parseBuildCustomerSupportID("corr-envelope-456") {
		parseT.Fatalf("ParseSupportID = %q, want derived support id", parseEnvelope.ParseSupportID)
	}

	parseAttrs := parseEnvelope.parseBuildErrorEnvelopeAttrs()
	parseAttrValues := map[string]string{}
	for _, parseAttr := range parseAttrs {
		if parseTypedAttr, isParseAttr := parseAttr.(slog.Attr); isParseAttr {
			parseAttrValues[parseTypedAttr.Key] = parseTypedAttr.Value.Resolve().String()
		}
	}
	if parseAttrValues["request.id"] != "req-envelope-123" {
		parseT.Fatalf("request.id = %q, want req-envelope-123", parseAttrValues["request.id"])
	}
	if parseAttrValues["correlation.id"] != "corr-envelope-456" {
		parseT.Fatalf("correlation.id = %q, want corr-envelope-456", parseAttrValues["correlation.id"])
	}
	if parseAttrValues["trace.id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		parseT.Fatalf("trace.id = %q, want parsed trace id", parseAttrValues["trace.id"])
	}
	if parseAttrValues["span.id"] != "00f067aa0ba902b7" {
		parseT.Fatalf("span.id = %q, want parsed span id", parseAttrValues["span.id"])
	}
	if parseAttrValues["support.id"] != parseBuildCustomerSupportID("corr-envelope-456") {
		parseT.Fatalf("support.id = %q, want derived support id", parseAttrValues["support.id"])
	}
	if parseAttrValues["error.message_key"] != "request_failed" {
		parseT.Fatalf("error.message_key = %q, want request_failed", parseAttrValues["error.message_key"])
	}
	if parseAttrValues["severity_text"] != slog.LevelError.String() {
		parseT.Fatalf("severity_text = %q, want %q", parseAttrValues["severity_text"], slog.LevelError.String())
	}
	if parseAttrValues["error.code"] != codes.Internal.String() {
		parseT.Fatalf("error.code = %q, want %q", parseAttrValues["error.code"], codes.Internal.String())
	}

	parseStatusErr := parseEnvelope.parseBuildErrorEnvelopeStatus()
	if status.Code(parseStatusErr) != codes.Internal {
		parseT.Fatalf("status code = %v, want Internal", status.Code(parseStatusErr))
	}
	if parseStatusMessage := status.Convert(parseStatusErr).Message(); parseStatusMessage != parsePublicErrorGenericMessage {
		parseT.Fatalf("status message = %q, want %q", parseStatusMessage, parsePublicErrorGenericMessage)
	}
	if parseEnvelope.ParseTraceID == "" || parseEnvelope.ParseSpanID == "" {
		parseT.Fatalf("expected trace ids from context, got trace=%q span=%q", parseEnvelope.ParseTraceID, parseEnvelope.ParseSpanID)
	}
}
