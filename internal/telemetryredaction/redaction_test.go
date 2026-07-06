package telemetryredaction

import (
	"errors"
	"reflect"
	"testing"
)

func resetPolicy(parseT *testing.T) {
	parseT.Helper()
	Configure(Policy{})
	parseT.Cleanup(func() {
		Configure(Policy{})
	})
}

// TestValueGuardsAgainstNonJSONValues pins the #50 contract guard: a non-JSON-like
// value (a struct or pointer) reaching Value() cannot be walked, so it is flagged
// via the dev guard (a future caller passing a raw struct with secret fields is
// caught rather than leaking silently), while ordinary JSON scalars never flag.
func TestValueGuardsAgainstNonJSONValues(parseT *testing.T) {
	resetPolicy(parseT)

	type secretBearing struct {
		APIKey string
	}

	parseOldReport := reportNonJSONTelemetryValue
	parseFlagged := map[string]int{}
	reportNonJSONTelemetryValue = func(parsePath string, parseValue any) {
		parseFlagged[parsePath]++
	}
	parseT.Cleanup(func() { reportNonJSONTelemetryValue = parseOldReport })

	// A struct nested inside a JSON map hits the un-walkable default branch.
	parseInput := map[string]any{
		"scalar": "safe",
		"count":  42,
		"creds":  secretBearing{APIKey: "sk-live-123"},
	}
	Value("telemetry", parseInput)

	if parseFlagged["telemetry.creds"] != 1 {
		parseT.Fatalf("struct value must be flagged once, got %d for telemetry.creds (all: %v)", parseFlagged["telemetry.creds"], parseFlagged)
	}
	// Scalars must NOT be flagged.
	if parseFlagged["telemetry.scalar"] != 0 || parseFlagged["telemetry.count"] != 0 {
		parseT.Fatalf("scalars must not be flagged: %v", parseFlagged)
	}

	// A pointer is likewise flagged.
	parseSecret := &secretBearing{APIKey: "sk-live-456"}
	Value("ptr", map[string]any{"creds": parseSecret})
	if parseFlagged["ptr.creds"] != 1 {
		parseT.Fatalf("pointer value must be flagged, got %v", parseFlagged)
	}
}

// TestNonJSONReportDedupesPerType pins that the default (diagnostics-emitting) guard
// reports each distinct offending type at most once, so it cannot spam the sink.
func TestNonJSONReportDedupesPerType(parseT *testing.T) {
	resetPolicy(parseT)

	type widget struct{ Token string }

	parseOldSeen := nonJSONReported
	nonJSONMu.Lock()
	nonJSONReported = map[string]struct{}{}
	nonJSONMu.Unlock()
	parseT.Cleanup(func() {
		nonJSONMu.Lock()
		nonJSONReported = parseOldSeen
		nonJSONMu.Unlock()
	})

	parseEmits := 0
	parseOldReport := reportNonJSONTelemetryValue
	reportNonJSONTelemetryValue = func(parsePath string, parseValue any) {
		// Exercise the real dedup set, counting only first-sightings.
		parseTypeName := fmtSprintType(parseValue)
		nonJSONMu.Lock()
		_, parseSeen := nonJSONReported[parseTypeName]
		if !parseSeen {
			nonJSONReported[parseTypeName] = struct{}{}
			parseEmits++
		}
		nonJSONMu.Unlock()
	}
	parseT.Cleanup(func() { reportNonJSONTelemetryValue = parseOldReport })

	for parseI := 0; parseI < 5; parseI++ {
		Value("t", map[string]any{"w": widget{Token: "x"}})
	}
	if parseEmits != 1 {
		parseT.Fatalf("same type must emit once, got %d", parseEmits)
	}
}

func fmtSprintType(parseValue any) string { return reflect.TypeOf(parseValue).String() }

func TestFieldMatchesNormalizedNamesAndLastPathSegment(parseT *testing.T) {
	resetPolicy(parseT)
	Configure(Policy{Fields: []string{"api-key", "user token"}})

	if parseGot, parseKeep := Field("headers.api_key", "", "secret"); !parseKeep || parseGot != RedactedValue {
		parseT.Fatalf("Field path-segment redaction = (%v, %v), want (%q, true)", parseGot, parseKeep, RedactedValue)
	}
	if parseGot, parseKeep := Field("headers", "User.Token", "secret"); !parseKeep || parseGot != RedactedValue {
		parseT.Fatalf("Field normalized-name redaction = (%v, %v), want (%q, true)", parseGot, parseKeep, RedactedValue)
	}
	if parseGot, parseKeep := Field("headers.request_id", "", "safe"); !parseKeep || parseGot != "safe" {
		parseT.Fatalf("Field unmatched = (%v, %v), want original safe value", parseGot, parseKeep)
	}
}

func TestValueRecursivelyRedactsAndDropsFailedFields(parseT *testing.T) {
	resetPolicy(parseT)
	Configure(Policy{
		Fields: []string{"password", "token"},
		Redact: func(parseCtx Context) (any, error) {
			if parseCtx.Field == "token" {
				return nil, errors.New("redactor unavailable")
			}
			return "masked:" + parseCtx.Path, nil
		},
	})

	parseInput := map[string]any{
		"user": map[string]any{
			"password": "secret",
			"token":    "drop-me",
			"name":     "Ada",
		},
		"events": []any{
			map[string]any{"password": "nested"},
		},
	}
	parseGot := Value("root", parseInput)
	parseWant := map[string]any{
		"user": map[string]any{
			"password": "masked:root.user.password",
			"name":     "Ada",
		},
		"events": []any{
			map[string]any{"password": "masked:root.events[0].password"},
		},
	}
	if !reflect.DeepEqual(parseGot, parseWant) {
		parseT.Fatalf("Value redaction mismatch:\n got: %#v\nwant: %#v", parseGot, parseWant)
	}
	if _, parseExists := parseInput["user"].(map[string]any)["token"]; !parseExists {
		parseT.Fatal("Value mutated the source map while dropping redacted output field")
	}
}

func TestCurrentAndValueReturnDefensiveCopies(parseT *testing.T) {
	resetPolicy(parseT)
	Configure(Policy{Fields: []string{"secret"}})

	parseCurrent := Current()
	parseCurrent.Fields[0] = "mutated"
	if parseGot, parseKeep := Field("secret", "secret", "value"); !parseKeep || parseGot != RedactedValue {
		parseT.Fatalf("Current exposed mutable policy fields: got (%v, %v)", parseGot, parseKeep)
	}

	parseNested := map[string]any{"child": "value"}
	parseSlice := []any{parseNested}
	parseGot := Value("", map[string]any{
		"nested": parseNested,
		"slice":  parseSlice,
	}).(map[string]any)
	parseGot["nested"].(map[string]any)["child"] = "changed"
	parseGot["slice"].([]any)[0] = "changed"

	if parseNested["child"] != "value" {
		parseT.Fatalf("Value returned map alias to source: %#v", parseNested)
	}
	if !reflect.DeepEqual(parseSlice, []any{map[string]any{"child": "value"}}) {
		parseT.Fatalf("Value returned slice alias to source: %#v", parseSlice)
	}
}

func TestStringMapNilEmptyAndFailureSemantics(parseT *testing.T) {
	resetPolicy(parseT)
	if parseGot := StringMap("root", nil); parseGot != nil {
		parseT.Fatalf("nil StringMap = %#v, want nil", parseGot)
	}
	parseEmpty := map[string]string{}
	if parseGot := StringMap("root", parseEmpty); len(parseGot) != 0 {
		parseT.Fatalf("empty StringMap = %#v, want empty", parseGot)
	}

	Configure(Policy{
		Fields: []string{"secret"},
		Redact: func(Context) (any, error) {
			return nil, errors.New("drop")
		},
	})
	parseGot := StringMap("root", map[string]string{"secret": "x", "safe": "y"})
	parseWant := map[string]string{"safe": "y"}
	if !reflect.DeepEqual(parseGot, parseWant) {
		parseT.Fatalf("StringMap fail-closed mismatch: got %#v want %#v", parseGot, parseWant)
	}
}

func TestJoinPathTrimsAndHandlesEmptySegments(parseT *testing.T) {
	if parseGot := JoinPath(" root ", " child "); parseGot != "root.child" {
		parseT.Fatalf("JoinPath trims segments = %q", parseGot)
	}
	if parseGot := JoinPath("", " child "); parseGot != "child" {
		parseT.Fatalf("JoinPath empty base = %q", parseGot)
	}
	if parseGot := JoinPath("root", " "); parseGot != "root" {
		parseT.Fatalf("JoinPath empty field = %q", parseGot)
	}
}
