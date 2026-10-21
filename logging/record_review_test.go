package logging

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// reviewNilErr is an error whose method dereferences its receiver, like most real error structs.
type reviewNilErr struct{ msg string }

func (parseE *reviewNilErr) Error() string { return parseE.msg }

// reviewNilStringer is the fmt.Stringer equivalent.
type reviewNilStringer struct{ text string }

func (parseS *reviewNilStringer) String() string { return parseS.text }

// TestNormalizeLogValueTypedNilDoesNotPanic pins that a typed-nil error/Stringer (the classic
// "var err *MyErr; log.Error(..., err)") is logged as null instead of panicking inside the logger.
// fmt guards this case; the hand-rolled error/Stringer branches did not.
func TestNormalizeLogValueTypedNilDoesNotPanic(parseT *testing.T) {
	var parseErr *reviewNilErr
	var parseStringer *reviewNilStringer

	if parseGot := normalizeLogValue(parseErr); parseGot != nil {
		parseT.Errorf("typed-nil error = %#v, want nil", parseGot)
	}
	if parseGot := normalizeLogValue(parseStringer); parseGot != nil {
		parseT.Errorf("typed-nil Stringer = %#v, want nil", parseGot)
	}
	parseFields := buildLogFields([]any{parseErr, "who", parseStringer})
	if _, parseOk := parseFields["error"]; !parseOk {
		parseT.Errorf("bare error argument produced no error field: %v", parseFields)
	}
	// And the full record path.
	_ = buildLogRecord(context.Background(), "error", "scope", "msg", parseFields)
}

// TestNormalizeLogValueDepthLimitDoesNotLeakSecrets pins that a value nested past the depth
// limit is not stringified wholesale: fmt.Sprint of the remaining subtree hands every nested
// secret to the log sink as an opaque string the key-based redaction policy cannot see.
func TestNormalizeLogValueDepthLimitDoesNotLeakSecrets(parseT *testing.T) {
	ConfigureTelemetryRedaction(RedactionPolicy{Fields: []string{"password"}})
	parseT.Cleanup(func() { ConfigureTelemetryRedaction(RedactionPolicy{}) })

	parseDeep := map[string]any{"password": "hunter2"}
	for range maxLogValueDepth + 4 {
		parseDeep = map[string]any{"next": parseDeep}
	}
	parseRecord := buildLogRecord(context.Background(), "info", "scope", "msg", map[string]any{"payload": parseDeep})
	if parseDump := strings.TrimSpace(fmt.Sprint(parseRecord)); strings.Contains(parseDump, "hunter2") {
		parseT.Fatalf("secret leaked through the depth limit: %s", parseDump)
	}
}
