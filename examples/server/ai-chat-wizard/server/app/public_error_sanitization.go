package app

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parsePublicErrorGenericMessage = "request failed; retry later"
const parsePublicErrorStorageMessage = "storage unavailable; retry later"
const parsePublicErrorProviderMessage = "provider unavailable; retry later"

var parseSanitizeSQLPattern = regexp.MustCompile(`(?i)(sqlite|sql|query|database|constraint|no such table|syntax error)`)
var parseSanitizeProviderPattern = regexp.MustCompile(`(?i)(provider|provider payload|response body|upstream|openai|anthropic|cerebras|api[_-]?key|authorization|bearer|cookie|token=|x-api-key)`)
var parseSanitizePathPattern = regexp.MustCompile(`([A-Za-z]:\\|/[^\\s]+|\\.\\/|\\.\\\\)`)
var parseSanitizeEnvSecretPattern = regexp.MustCompile(`(?i)\b[A-Z][A-Z0-9_]{2,}(_KEY|_SECRET|_TOKEN)\b`)

// parseSanitizePublicError rewrites one error into one customer-safe status error without leaking sensitive internals.
func parseSanitizePublicError(parseErr error) error {
	if parseErr == nil {
		return nil
	}
	parseStatus, isParseStatus := status.FromError(parseErr)
	if !isParseStatus {
		return status.Error(codes.Internal, parsePublicErrorGenericMessage)
	}
	parseSanitizedMessage := parseResolveSanitizedPublicMessage(parseStatus.Code(), parseStatus.Message())
	if parseSanitizedMessage == "" {
		parseSanitizedMessage = parsePublicErrorGenericMessage
	}
	return status.Error(parseStatus.Code(), parseSanitizedMessage)
}

// parseBuildSanitizedInternalStatus builds one internal error envelope and sanitizes it for public response safety.
func parseBuildSanitizedInternalStatus(parseCtx context.Context, parseOperation string, parseErr error) error {
	parseOperation = strings.TrimSpace(parseOperation)
	if parseOperation == "" {
		parseOperation = "operation failed"
	}
	if parseErr == nil {
		return status.Error(codes.Internal, parsePublicErrorGenericMessage)
	}
	parseEnvelope := parseBuildErrorEnvelope(parseCtx, codes.Internal, parseOperation, parseErr, slog.LevelError)
	return parseEnvelope.parseBuildErrorEnvelopeStatus()
}

// parseResolveSanitizedPublicMessage maps one raw status code/message pair to one customer-safe response message.
func parseResolveSanitizedPublicMessage(parseCode codes.Code, parseMessage string) string {
	parseRawMessage := strings.TrimSpace(parseMessage)
	parseMessage = parseScrubSecretString(parseMessage)
	parseMessage = strings.ReplaceAll(parseMessage, parseLogRedactionText, "redacted")
	parseMessage = strings.TrimSpace(parseMessage)
	if parseMessage == "" {
		parseMessage = parsePublicErrorGenericMessage
	}

	isParseSensitiveMessage := parseSanitizeSQLPattern.MatchString(parseRawMessage) ||
		parseSanitizeProviderPattern.MatchString(parseRawMessage) ||
		parseSanitizePathPattern.MatchString(parseRawMessage) ||
		parseSanitizeEnvSecretPattern.MatchString(parseRawMessage)

	switch parseCode {
	case codes.Internal, codes.Unknown, codes.DataLoss:
		if parseSanitizeSQLPattern.MatchString(parseRawMessage) {
			return parsePublicErrorStorageMessage
		}
		if parseSanitizeProviderPattern.MatchString(parseRawMessage) {
			return parsePublicErrorProviderMessage
		}
		return parsePublicErrorGenericMessage
	case codes.Unavailable, codes.DeadlineExceeded:
		if parseSanitizeProviderPattern.MatchString(parseRawMessage) {
			return parsePublicErrorProviderMessage
		}
		if parseSanitizeSQLPattern.MatchString(parseRawMessage) {
			return parsePublicErrorStorageMessage
		}
		return "service unavailable; retry later"
	case codes.InvalidArgument:
		if isParseSensitiveMessage {
			return "request validation failed"
		}
		return parseMessage
	case codes.PermissionDenied:
		if isParseSensitiveMessage {
			return "permission denied"
		}
		return parseMessage
	case codes.Unauthenticated:
		if isParseSensitiveMessage {
			return "authentication required"
		}
		return parseMessage
	case codes.FailedPrecondition:
		if isParseSensitiveMessage {
			return "request failed precondition"
		}
		return parseMessage
	default:
		if isParseSensitiveMessage {
			return parsePublicErrorGenericMessage
		}
		return parseMessage
	}
}

// parseSanitizeProviderErrorMessage resolves one upstream-provider failure into one customer-safe message.
func parseSanitizeProviderErrorMessage(parseProviderID string, parseMessage string) string {
	parseProviderID = strings.TrimSpace(parseProviderID)
	parseMessage = strings.TrimSpace(parseMessage)
	if parseMessage == "" {
		if parseProviderID == "" {
			return "model provider stream failed"
		}
		return parseProviderID + " stream failed"
	}
	parseSanitizedMessage := parseResolveSanitizedPublicMessage(codes.Unavailable, parseMessage)
	if parseSanitizedMessage == "" {
		parseSanitizedMessage = parsePublicErrorProviderMessage
	}
	return parseSanitizedMessage
}
