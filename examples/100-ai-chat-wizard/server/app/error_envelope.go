package app

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseErrorEnvelope struct {
	ParseCode          codes.Code
	ParseMessageKey    string
	ParsePublicMessage string
	ParseSeverity      string
	ParseRequestID     string
	ParseCorrelationID string
	ParseTraceID       string
	ParseSpanID        string
	ParseSupportID     string
}

// parseBuildErrorEnvelope builds one shared error envelope for customer-safe logs, gRPC responses, and operator traces.
func parseBuildErrorEnvelope(parseCtx context.Context, parseCode codes.Code, parseOperation string, parseErr error, parseSeverity slog.Level) parseErrorEnvelope {
	parseOperation = strings.TrimSpace(parseOperation)
	if parseOperation == "" {
		parseOperation = "operation failed"
	}
	parseEnvelope := parseErrorEnvelope{
		ParseCode:      parseCode,
		ParseSeverity:  parseSeverity.String(),
		ParseRequestID: "",
	}
	if parseErr == nil {
		parseEnvelope.ParsePublicMessage = parsePublicErrorGenericMessage
	} else {
		parseStatusErr := parseSanitizePublicError(statusErrorWithOperation(parseCode, parseOperation, parseErr))
		parseEnvelope.ParseCode = codes.Code(statusCodeFromError(parseStatusErr))
		parseEnvelope.ParsePublicMessage = statusMessageFromError(parseStatusErr)
	}
	parseEnvelope.ParseMessageKey = parseBuildErrorMessageKey(parseEnvelope.ParseCode, parseEnvelope.ParsePublicMessage)
	parseEnvelope.ParseRequestID, parseEnvelope.ParseCorrelationID = parseExtractCorrelationFromContext(parseCtx)
	parseEnvelope.ParseTraceID, parseEnvelope.ParseSpanID, _ = parseExtractTraceContextFromContext(parseCtx)
	parseSupportExposure := parseResolveSupportIDExposure(parseCtx, "", "")
	parseEnvelope.ParseSupportID = strings.TrimSpace(parseSupportExposure.ParseCustomerSupportID)
	return parseEnvelope
}

// parseBuildErrorEnvelopeAttrs builds one log field list from one shared error envelope.
func (parseEnvelope parseErrorEnvelope) parseBuildErrorEnvelopeAttrs() []any {
	parseAttrs := make([]any, 0, 8)
	if parseEnvelope.ParseRequestID != "" {
		parseAttrs = append(parseAttrs, slog.String("request.id", parseEnvelope.ParseRequestID))
	}
	if parseEnvelope.ParseCorrelationID != "" {
		parseAttrs = append(parseAttrs, slog.String("correlation.id", parseEnvelope.ParseCorrelationID))
	}
	if parseEnvelope.ParseTraceID != "" {
		parseAttrs = append(parseAttrs, slog.String("trace.id", parseEnvelope.ParseTraceID))
	}
	if parseEnvelope.ParseSpanID != "" {
		parseAttrs = append(parseAttrs, slog.String("span.id", parseEnvelope.ParseSpanID))
	}
	if parseEnvelope.ParseSupportID != "" {
		parseAttrs = append(parseAttrs, slog.String("support.id", parseEnvelope.ParseSupportID))
	}
	if parseEnvelope.ParseMessageKey != "" {
		parseAttrs = append(parseAttrs, slog.String("error.message_key", parseEnvelope.ParseMessageKey))
	}
	if parseEnvelope.ParseSeverity != "" {
		parseAttrs = append(parseAttrs, slog.String("severity_text", parseEnvelope.ParseSeverity))
	}
	if parseEnvelope.ParseCode != codes.OK {
		parseAttrs = append(parseAttrs, slog.String("error.code", parseEnvelope.ParseCode.String()))
	}
	return parseAttrs
}

// parseBuildErrorEnvelopeStatus returns one gRPC status error from one shared envelope.
func (parseEnvelope parseErrorEnvelope) parseBuildErrorEnvelopeStatus() error {
	if parseEnvelope.ParsePublicMessage == "" {
		parseEnvelope.ParsePublicMessage = parsePublicErrorGenericMessage
	}
	return status.Error(parseEnvelope.ParseCode, parseEnvelope.ParsePublicMessage)
}

// parseBuildErrorMessageKey resolves one stable message key from one public error code and message.
func parseBuildErrorMessageKey(parseCode codes.Code, parseMessage string) string {
	parseMessage = strings.TrimSpace(parseMessage)
	switch parseMessage {
	case parsePublicErrorStorageMessage:
		return "storage_unavailable"
	case parsePublicErrorProviderMessage:
		return "provider_unavailable"
	case parsePublicErrorGenericMessage:
		return "request_failed"
	case "service unavailable; retry later":
		return "service_unavailable"
	case "request validation failed":
		return "request_validation_failed"
	case "permission denied":
		return "permission_denied"
	case "authentication required":
		return "authentication_required"
	case "request failed precondition":
		return "request_failed_precondition"
	}
	switch parseCode {
	case codes.Internal, codes.Unknown, codes.DataLoss:
		return "request_failed"
	case codes.Unavailable, codes.DeadlineExceeded:
		return "service_unavailable"
	case codes.InvalidArgument:
		return "request_validation_failed"
	case codes.PermissionDenied:
		return "permission_denied"
	case codes.Unauthenticated:
		return "authentication_required"
	case codes.FailedPrecondition:
		return "request_failed_precondition"
	default:
		return "request_failed"
	}
}

func statusErrorWithOperation(parseCode codes.Code, parseOperation string, parseErr error) error {
	return status.Errorf(parseCode, "%s: %v", parseOperation, parseErr)
}

func statusCodeFromError(parseErr error) codes.Code {
	if parseErr == nil {
		return codes.Internal
	}
	return status.Code(parseErr)
}

func statusMessageFromError(parseErr error) string {
	if parseErr == nil {
		return ""
	}
	return status.Convert(parseErr).Message()
}
