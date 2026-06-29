package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBuildBinarySourceValueRoundTripsScalars verifies bool, number, string, and nil source values round-trip.
func TestBuildBinarySourceValueRoundTripsScalars(parseT *testing.T) {
	parseCases := []any{true, float64(3.5), "ok", nil}
	for _, parseCase := range parseCases {
		parsePayload, parseErr := runtime2.BuildBinarySourceValue(parseCase)
		if parseErr != nil {
			parseT.Fatalf("BuildBinarySourceValue(%#v) returned error: %v", parseCase, parseErr)
		}
		parseDecoded, parseErr := runtime2.ParseBinarySourceValue(parsePayload)
		if parseErr != nil {
			parseT.Fatalf("ParseBinarySourceValue(%#v) returned error: %v", parseCase, parseErr)
		}
		switch parseWant := parseCase.(type) {
		case bool:
			parseGot, parseOk := parseDecoded.(bool)
			if !parseOk || parseGot != parseWant {
				parseT.Fatalf("expected bool %#v, got %#v", parseWant, parseDecoded)
			}
		case float64:
			parseGot, parseOk := parseDecoded.(float64)
			if !parseOk || parseGot != parseWant {
				parseT.Fatalf("expected number %#v, got %#v", parseWant, parseDecoded)
			}
		case string:
			parseGot, parseOk := parseDecoded.(string)
			if !parseOk || parseGot != parseWant {
				parseT.Fatalf("expected string %#v, got %#v", parseWant, parseDecoded)
			}
		case nil:
			if parseDecoded != nil {
				parseT.Fatalf("expected nil, got %#v", parseDecoded)
			}
		}
	}
}

// TestBuildBinarySourceValueRoundTripsSmallList verifies supported small source lists round-trip.
func TestBuildBinarySourceValueRoundTripsSmallList(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildBinarySourceValue([]any{true, float64(2), "x"})
	if parseErr != nil {
		parseT.Fatalf("BuildBinarySourceValue returned error: %v", parseErr)
	}
	parseDecoded, parseErr := runtime2.ParseBinarySourceValue(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinarySourceValue returned error: %v", parseErr)
	}
	parseList, parseOk := parseDecoded.([]any)
	if !parseOk {
		parseT.Fatalf("expected decoded list, got %#v", parseDecoded)
	}
	if len(parseList) != 3 {
		parseT.Fatalf("expected list length 3, got %d", len(parseList))
	}
}

// TestBuildBinarySourceValueRoundTripsNestedMap verifies canonical nested maps round-trip.
func TestBuildBinarySourceValueRoundTripsNestedMap(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildBinarySourceValue(map[string]any{
		"meta": map[string]any{
			"count":  float64(3),
			"status": "ok",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinarySourceValue returned error: %v", parseErr)
	}
	parseDecoded, parseErr := runtime2.ParseBinarySourceValue(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinarySourceValue returned error: %v", parseErr)
	}
	parseMap, parseOk := parseDecoded.(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected decoded map, got %#v", parseDecoded)
	}
	parseNestedMap, parseOk := parseMap["meta"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected nested map, got %#v", parseMap["meta"])
	}
	if parseNestedMap["status"] != "ok" {
		parseT.Fatalf("expected nested status ok, got %#v", parseNestedMap["status"])
	}
}

// TestBuildBinarySourceValueRejectsUnsupportedKind verifies unsupported kinds fail explicitly.
func TestBuildBinarySourceValueRejectsUnsupportedKind(parseT *testing.T) {
	parseValue := complex(1, 2)
	if _, parseErr := runtime2.BuildBinarySourceValue(parseValue); parseErr == nil {
		parseT.Fatal("expected unsupported complex source value to fail")
	}
}
