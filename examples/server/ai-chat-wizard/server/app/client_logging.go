package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetClientIdentity returns a stable server-issued client identity for metadata and logging correlation.
func (parseS *chatServer) GetClientIdentity(parseCtx context.Context, _ *emptypb.Empty) (*chatpb.GetClientIdentityResponse, error) {
	parseClientID := parseResolveClientIdentityFromContext(parseCtx)
	isParseIssued := false
	if parseClientID == "" {
		parseClientID = uuid.NewString()
		isParseIssued = true
	}

	parseLogger := parseS.parseResolveClientLogger().With(slog.String("rpc", "GetClientIdentity"))
	parseAttrs := []any{
		slog.String("client.id", parseClientID),
		slog.Bool("issued", isParseIssued),
	}
	if parseUser, parseOk := parseS.parseAuthenticatedUserFromContext(parseCtx); parseOk && parseUser.ID > 0 {
		parseAttrs = append(parseAttrs,
			slog.Int64("user.id", parseUser.ID),
			slog.String("user.email", parseUser.Email),
		)
	}
	parseLogger.Info("rpc.GetClientIdentity: complete", parseAttrs...)
	return &chatpb.GetClientIdentityResponse{ClientId: parseClientID}, nil
}

// ReportClientLog stores one structured client log record in the dedicated client log stream.
func (parseS *chatServer) ReportClientLog(parseCtx context.Context, parseReq *chatpb.ReportClientLogRequest) (*emptypb.Empty, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "client log payload is required")
	}

	parseScope := strings.TrimSpace(parseReq.GetScope())
	if parseScope == "" {
		parseScope = "chat-wizard"
	}
	parseLevel := parseResolveClientLogLevel(parseReq.GetLevel())
	parseClientID := parseResolveClientIdentityFromRequest(parseCtx, parseReq.GetClientId())
	if parseClientID == "" {
		parseClientID = uuid.NewString()
	}
	parseMessage := strings.TrimSpace(parseReq.GetMessage())
	if parseMessage == "" {
		parseMessage = parseFormatFallbackClientLogMessage(parseScope, parseLevel, parseClientID)
	}

	parseObservedAt := time.Now().UTC()
	if parseTimestamp := parseReq.GetObservedAt(); parseTimestamp != nil {
		parseObservedAt = parseTimestamp.AsTime().UTC()
	}

	parseTraceID, parseSpanID, parseTraceState := parseExtractTraceContextFromContext(parseCtx)
	parseRequestID, parseCorrelationID := parseExtractCorrelationFromContext(parseCtx)
	parseLogger := parseS.parseResolveClientLogger().With(
		slog.String("rpc.method", "ChatService/ReportClientLog"),
		slog.String("log.scope", parseScope),
		slog.String("client.id", parseClientID),
	)
	parseAttrs := []any{
		slog.Time("event.observed", parseObservedAt),
	}
	if parseLevel >= slog.LevelError {
		parseAttrs = append(parseAttrs, slog.String("error.boundary", parseResolveClientErrorBoundary(parseScope)))
	}
	if parseTraceID != "" {
		parseAttrs = append(parseAttrs, slog.String("trace.id", parseTraceID))
	}
	if parseSpanID != "" {
		parseAttrs = append(parseAttrs, slog.String("span.id", parseSpanID))
	}
	if parseTraceState != "" {
		parseAttrs = append(parseAttrs, slog.String("trace.state", parseTraceState))
	}
	if parseRequestID != "" {
		parseAttrs = append(parseAttrs, slog.String("request.id", parseRequestID))
	}
	if parseCorrelationID != "" {
		parseAttrs = append(parseAttrs, slog.String("correlation.id", parseCorrelationID))
	}
	if parseUser, parseOk := parseS.parseAuthenticatedUserFromContext(parseCtx); parseOk && parseUser.ID > 0 {
		parseAttrs = append(parseAttrs,
			slog.Int64("user.id", parseUser.ID),
			slog.String("user.email", parseUser.Email),
		)
	}
	parseAttrs = parseAppendStructuredClientLogFields(parseAttrs, parseReq.GetFields())

	parseLogger.Log(parseCtx, parseLevel, parseMessage, parseAttrs...)
	return &emptypb.Empty{}, nil
}

// parseResolveClientLogger selects the dedicated client logger and falls back safely.
func (parseS *chatServer) parseResolveClientLogger() *slog.Logger {
	if parseS != nil && parseS.clientLogger != nil {
		return parseS.clientLogger
	}
	if parseS != nil && parseS.logger != nil {
		return parseS.logger.With(
			slog.String("service.name", clientLogServiceName),
			slog.String("log.source", "client"),
			slog.String("event.dataset", "chat-wizard.client"),
		)
	}
	return slog.Default().With(
		slog.String("service.name", clientLogServiceName),
		slog.String("log.source", "client"),
		slog.String("event.dataset", "chat-wizard.client"),
	)
}

// parseResolveClientIdentityFromRequest resolves client identity with metadata precedence.
func parseResolveClientIdentityFromRequest(parseCtx context.Context, parseRequestClientID string) string {
	parseMetadataClientID := parseResolveClientIdentityFromContext(parseCtx)
	if parseMetadataClientID != "" {
		return parseMetadataClientID
	}
	return parseNormalizeClientIdentity(parseRequestClientID)
}

// parseResolveClientIdentityFromContext extracts a normalized client identity from gRPC metadata.
func parseResolveClientIdentityFromContext(parseCtx context.Context) string {
	parseIncomingMD, parseOk := metadata.FromIncomingContext(parseCtx)
	if !parseOk {
		return ""
	}
	for _, parseValue := range parseIncomingMD.Get(clientMetadataKey) {
		if parseClientID := parseNormalizeClientIdentity(parseValue); parseClientID != "" {
			return parseClientID
		}
	}
	return ""
}

// parseNormalizeClientIdentity validates and normalizes one candidate client identity value.
func parseNormalizeClientIdentity(parseRawClientID string) string {
	parseRawClientID = strings.TrimSpace(parseRawClientID)
	if parseRawClientID == "" {
		return ""
	}
	if _, parseErr := uuid.Parse(parseRawClientID); parseErr != nil {
		return ""
	}
	return parseRawClientID
}

// parseResolveClientLogLevel maps textual client levels to slog levels.
func parseResolveClientLogLevel(parseRawLevel string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(parseRawLevel)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// parseResolveClientErrorBoundary resolves one stable boundary for client-origin error events.
func parseResolveClientErrorBoundary(parseScope string) string {
	parseScope = strings.TrimSpace(parseScope)
	if parseScope != "" {
		return parseScope
	}
	return "ChatService/ReportClientLog"
}

// parseFormatFallbackClientLogMessage builds one context-rich fallback message for blank client entries.
func parseFormatFallbackClientLogMessage(parseScope string, parseLevel slog.Level, parseClientID string) string {
	parseScope = strings.TrimSpace(parseScope)
	if parseScope == "" {
		parseScope = "chat-wizard"
	}
	parseClientID = strings.TrimSpace(parseClientID)
	if parseClientID == "" {
		parseClientID = "unknown"
	}
	parseLevelLabel := strings.ToLower(strings.TrimSpace(parseLevel.String()))
	if parseLevelLabel == "" {
		parseLevelLabel = "info"
	}
	return fmt.Sprintf("client log event: level=%s scope=%s client.id=%s", parseLevelLabel, parseScope, parseClientID)
}

// parseAppendStructuredClientLogFields appends one standard attributes object for client structured fields.
func parseAppendStructuredClientLogFields(parseAttrs []any, parseFields *structpb.Struct) []any {
	if parseFields == nil {
		return parseAttrs
	}
	parseFieldMap := parseFields.AsMap()
	if len(parseFieldMap) == 0 {
		return parseAttrs
	}
	return append(parseAttrs, slog.Any("attributes", parseFieldMap))
}

// parseExtractTraceContextFromContext resolves OpenTelemetry trace correlation from incoming metadata.
func parseExtractTraceContextFromContext(parseCtx context.Context) (string, string, string) {
	parseCarrier := parseNewTraceMetadataCarrier(parseCtx)
	parseTraceCtx := propagation.TraceContext{}
	parseExtractedCtx := parseTraceCtx.Extract(context.Background(), parseCarrier)
	parseSpanCtx := trace.SpanContextFromContext(parseExtractedCtx)
	if !parseSpanCtx.IsValid() {
		parseSpanCtx = trace.SpanContextFromContext(parseCtx)
	}
	if !parseSpanCtx.IsValid() {
		return "", "", strings.TrimSpace(parseCarrier.Get(traceStateMetadataKey))
	}
	return parseSpanCtx.TraceID().String(), parseSpanCtx.SpanID().String(), parseSpanCtx.TraceState().String()
}

// parseExtractCorrelationFromContext resolves request and correlation identifiers from gRPC metadata.
func parseExtractCorrelationFromContext(parseCtx context.Context) (string, string) {
	parseIncomingMD, parseOk := metadata.FromIncomingContext(parseCtx)
	if !parseOk {
		return "", ""
	}
	parseRequestID := parseResolveMetadataValue(parseIncomingMD, requestIDMetadataKey, "request-id")
	parseCorrelationID := parseResolveMetadataValue(parseIncomingMD, correlationIDMetadataKey, "correlation-id")
	if parseCorrelationID == "" {
		parseCorrelationID = parseRequestID
	}
	if parseRequestID == "" {
		parseRequestID = parseCorrelationID
	}
	return parseRequestID, parseCorrelationID
}

// parseResolveMetadataValue resolves one metadata value by trying keys in order.
func parseResolveMetadataValue(parseMetadata metadata.MD, parseKeys ...string) string {
	for _, parseKey := range parseKeys {
		parseKey = strings.ToLower(strings.TrimSpace(parseKey))
		if parseKey == "" {
			continue
		}
		parseValues := parseMetadata.Get(parseKey)
		for _, parseValue := range parseValues {
			parseValue = strings.TrimSpace(parseValue)
			if parseValue != "" {
				return parseValue
			}
		}
	}
	return ""
}

// parseNewTraceMetadataCarrier builds a tracecarrier from incoming gRPC metadata.
func parseNewTraceMetadataCarrier(parseCtx context.Context) parseTraceMetadataCarrier {
	parseIncomingMD, parseOk := metadata.FromIncomingContext(parseCtx)
	if !parseOk {
		return parseTraceMetadataCarrier{parseMetadata: metadata.MD{}}
	}
	return parseTraceMetadataCarrier{parseMetadata: parseIncomingMD}
}

type parseTraceMetadataCarrier struct {
	parseMetadata metadata.MD
}

// Get returns one metadata value for the requested key.
func (parseCarrier parseTraceMetadataCarrier) Get(parseKey string) string {
	parseValues := parseCarrier.parseMetadata.Get(strings.ToLower(strings.TrimSpace(parseKey)))
	if len(parseValues) == 0 {
		return ""
	}
	return parseValues[0]
}

// Set stores one metadata value for the requested key.
func (parseCarrier parseTraceMetadataCarrier) Set(parseKey string, parseValue string) {
	parseKey = strings.ToLower(strings.TrimSpace(parseKey))
	if parseKey == "" {
		return
	}
	parseCarrier.parseMetadata.Set(parseKey, parseValue)
}

// Keys returns all available metadata keys for propagation.
func (parseCarrier parseTraceMetadataCarrier) Keys() []string {
	parseKeys := make([]string, 0, len(parseCarrier.parseMetadata))
	for parseKey := range parseCarrier.parseMetadata {
		parseKeys = append(parseKeys, parseKey)
	}
	return parseKeys
}
