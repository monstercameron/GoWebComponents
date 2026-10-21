package interop

import (
	"encoding/hex"
	"errors"
	"strings"
)

// TraceContext is one parsed W3C traceparent value.
type TraceContext struct {
	TraceID string
	SpanID  string
	Sampled bool
}

// NewTraceID returns a random 32-character lowercase hex W3C trace id that is never all zero.
// It returns a CodeUnavailable error when no secure random source exists.
func NewTraceID() (string, error) {
	return buildRandomHexID("NewTraceID", 16)
}

// NewSpanID returns a random 16-character lowercase hex W3C span id that is never all zero.
// It returns a CodeUnavailable error when no secure random source exists.
func NewSpanID() (string, error) {
	return buildRandomHexID("NewSpanID", 8)
}

// FormatTraceparent formats a version 00 W3C traceparent header value. It returns an empty
// string when either id is malformed so callers never emit an invalid header.
func FormatTraceparent(parseTraceID string, parseSpanID string, isSampled bool) string {
	if !IsValidTraceID(parseTraceID) || !IsValidSpanID(parseSpanID) {
		return ""
	}
	parseFlags := "00"
	if isSampled {
		parseFlags = "01"
	}
	return "00-" + parseTraceID + "-" + parseSpanID + "-" + parseFlags
}

// ParseTraceparent parses a W3C traceparent header value and rejects malformed, all-zero and
// version ff values.
func ParseTraceparent(parseValue string) (TraceContext, error) {
	parseParts := strings.Split(strings.TrimSpace(parseValue), "-")
	if len(parseParts) < 4 {
		return TraceContext{}, wrapError("ParseTraceparent", "traceparent", CodeInvalid, errors.New("expected version-traceid-spanid-flags"))
	}
	parseVersion := parseParts[0]
	if len(parseVersion) != 2 || !isLowerHex(parseVersion) || parseVersion == "ff" {
		return TraceContext{}, wrapError("ParseTraceparent", "traceparent", CodeInvalid, errors.New("invalid version"))
	}
	if parseVersion == "00" && len(parseParts) != 4 {
		return TraceContext{}, wrapError("ParseTraceparent", "traceparent", CodeInvalid, errors.New("version 00 takes exactly four fields"))
	}
	if !IsValidTraceID(parseParts[1]) {
		return TraceContext{}, wrapError("ParseTraceparent", "traceparent", CodeInvalid, errors.New("invalid trace id"))
	}
	if !IsValidSpanID(parseParts[2]) {
		return TraceContext{}, wrapError("ParseTraceparent", "traceparent", CodeInvalid, errors.New("invalid span id"))
	}
	if len(parseParts[3]) != 2 || !isLowerHex(parseParts[3]) {
		return TraceContext{}, wrapError("ParseTraceparent", "traceparent", CodeInvalid, errors.New("invalid flags"))
	}
	parseFlags, _ := hex.DecodeString(parseParts[3])
	return TraceContext{TraceID: parseParts[1], SpanID: parseParts[2], Sampled: parseFlags[0]&1 == 1}, nil
}

// IsValidTraceID reports whether parseValue is 32 lowercase hex characters and not all zero.
func IsValidTraceID(parseValue string) bool {
	return len(parseValue) == 32 && isLowerHex(parseValue) && !isAllZero(parseValue)
}

// IsValidSpanID reports whether parseValue is 16 lowercase hex characters and not all zero.
func IsValidSpanID(parseValue string) bool {
	return len(parseValue) == 16 && isLowerHex(parseValue) && !isAllZero(parseValue)
}

// buildRandomHexID draws parseBytes random bytes, retrying so the id is never all zero.
func buildRandomHexID(parseOp string, parseBytes int) (string, error) {
	parseBuf := make([]byte, parseBytes)
	for parseAttempt := 0; parseAttempt < 4; parseAttempt++ {
		if parseErr := fillRandomBytes(parseOp, parseBuf); parseErr != nil {
			return "", parseErr
		}
		parseID := hex.EncodeToString(parseBuf)
		if !isAllZero(parseID) {
			return parseID, nil
		}
	}
	return "", wrapError(parseOp, "crypto.getRandomValues", CodeUnavailable, errors.New("random source returned only zero bytes"))
}

func isLowerHex(parseValue string) bool {
	for parseIndex := 0; parseIndex < len(parseValue); parseIndex++ {
		parseC := parseValue[parseIndex]
		if (parseC < '0' || parseC > '9') && (parseC < 'a' || parseC > 'f') {
			return false
		}
	}
	return true
}

func isAllZero(parseValue string) bool {
	return strings.Trim(parseValue, "0") == ""
}
