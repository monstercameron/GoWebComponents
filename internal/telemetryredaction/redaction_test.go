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
