package runtime2

import (
	"fmt"
	"testing"
)

var parseFormatRenderStyleScalarBenchmarkSink string

// TestFormatRenderStyleValueEquivalentStylesNormalizeConsistently verifies logically equivalent styles normalize identically.
func TestFormatRenderStyleValueEquivalentStylesNormalizeConsistently(parseTesting *testing.T) {
	parseStyleA := map[string]any{
		"color":  "red",
		"margin": 0,
	}
	parseStyleB := map[string]any{
		"margin": 0,
		"color":  "red",
	}
	formatStyleA, formatStyleAErr := FormatRenderStyleValue(parseStyleA)
	if formatStyleAErr != nil {
		parseTesting.Fatalf("FormatRenderStyleValue(styleA) error = %v", formatStyleAErr)
	}
	formatStyleB, formatStyleBErr := FormatRenderStyleValue(parseStyleB)
	if formatStyleBErr != nil {
		parseTesting.Fatalf("FormatRenderStyleValue(styleB) error = %v", formatStyleBErr)
	}
	if formatStyleA != formatStyleB {
		parseTesting.Fatalf("FormatRenderStyleValue outputs differ, styleA=%q styleB=%q", formatStyleA, formatStyleB)
	}
}

// TestFormatRenderStyleValueInvalidPayloadFails verifies invalid style payloads are rejected.
func TestFormatRenderStyleValueInvalidPayloadFails(parseTesting *testing.T) {
	_, formatStyleErr := FormatRenderStyleValue(42)
	if formatStyleErr == nil {
		parseTesting.Fatal("FormatRenderStyleValue(invalid payload) error = nil, want error")
	}
}

// TestFormatRenderStyleValueUnsupportedNestedShapeFails verifies nested style structures are rejected.
func TestFormatRenderStyleValueUnsupportedNestedShapeFails(parseTesting *testing.T) {
	parseNestedStyle := map[string]any{
		"color": map[string]string{
			"base": "red",
		},
	}
	_, formatStyleErr := FormatRenderStyleValue(parseNestedStyle)
	if formatStyleErr == nil {
		parseTesting.Fatal("FormatRenderStyleValue(nested payload) error = nil, want error")
	}
}

// TestFormatRenderStyleValueWhitespaceNormalizedAcrossInputs verifies string and map style payloads normalize whitespace consistently.
func TestFormatRenderStyleValueWhitespaceNormalizedAcrossInputs(parseTesting *testing.T) {
	parseStringStyle, parseStringStyleErr := FormatRenderStyleValue(" margin : 0 ; color : red ; ")
	if parseStringStyleErr != nil {
		parseTesting.Fatalf("FormatRenderStyleValue(string style) error = %v", parseStringStyleErr)
	}
	if parseStringStyle != "color:red;margin:0" {
		parseTesting.Fatalf("FormatRenderStyleValue(string style) = %q, want canonical ordering", parseStringStyle)
	}

	parseStringMapStyle, parseStringMapStyleErr := FormatRenderStyleValue(map[string]string{
		" margin ": " 0 ",
		" color ":  " red ",
	})
	if parseStringMapStyleErr != nil {
		parseTesting.Fatalf("FormatRenderStyleValue(string map style) error = %v", parseStringMapStyleErr)
	}
	if parseStringMapStyle != parseStringStyle {
		parseTesting.Fatalf("FormatRenderStyleValue(string map style) = %q, want %q", parseStringMapStyle, parseStringStyle)
	}

	parseAnyMapStyle, parseAnyMapStyleErr := FormatRenderStyleValue(map[string]any{
		" margin ": " 0 ",
		" color ":  " red ",
	})
	if parseAnyMapStyleErr != nil {
		parseTesting.Fatalf("FormatRenderStyleValue(any map style) error = %v", parseAnyMapStyleErr)
	}
	if parseAnyMapStyle != parseStringStyle {
		parseTesting.Fatalf("FormatRenderStyleValue(any map style) = %q, want %q", parseAnyMapStyle, parseStringStyle)
	}
}

// TestFormatRenderStyleValueRejectsInvalidStringAndCollisionShapes verifies malformed style segments and normalization collisions fail.
func TestFormatRenderStyleValueRejectsInvalidStringAndCollisionShapes(parseTesting *testing.T) {
	if _, parseErr := FormatRenderStyleValue("color"); parseErr == nil {
		parseTesting.Fatal("expected invalid style segment to fail")
	}
	if _, parseErr := FormatRenderStyleValue(":red"); parseErr == nil {
		parseTesting.Fatal("expected empty style key to fail")
	}
	if _, parseErr := FormatRenderStyleValue(map[string]any{
		"color":   "red",
		" color ": "blue",
	}); parseErr == nil {
		parseTesting.Fatal("expected colliding normalized style keys to fail")
	}
}

// TestFormatRenderStyleScalarCoversSupportedPrimitiveBranches verifies supported scalar style payloads normalize without relying on benchmark-only coverage.
func TestFormatRenderStyleScalarCoversSupportedPrimitiveBranches(parseTesting *testing.T) {
	parseCases := []struct {
		parseName  string
		parseValue any
		parseWant  string
	}{
		{parseName: "string", parseValue: "red", parseWant: "red"},
		{parseName: "true", parseValue: true, parseWant: "true"},
		{parseName: "false", parseValue: false, parseWant: "false"},
		{parseName: "int", parseValue: int(-42), parseWant: "-42"},
		{parseName: "int8", parseValue: int8(-8), parseWant: "-8"},
		{parseName: "int16", parseValue: int16(-16), parseWant: "-16"},
		{parseName: "int32", parseValue: int32(-32), parseWant: "-32"},
		{parseName: "int64", parseValue: int64(-64), parseWant: "-64"},
		{parseName: "uint", parseValue: uint(7), parseWant: "7"},
		{parseName: "uint8", parseValue: uint8(8), parseWant: "8"},
		{parseName: "uint16", parseValue: uint16(16), parseWant: "16"},
		{parseName: "uint32", parseValue: uint32(32), parseWant: "32"},
		{parseName: "uint64", parseValue: uint64(64), parseWant: "64"},
		{parseName: "float32", parseValue: float32(12.5), parseWant: "12.5"},
		{parseName: "float64", parseValue: float64(0.125), parseWant: "0.125"},
	}

	for _, parseCase := range parseCases {
		parseGot, parseErr := formatRenderStyleScalar(parseCase.parseValue)
		if parseErr != nil {
			parseTesting.Fatalf("formatRenderStyleScalar(%s) error = %v", parseCase.parseName, parseErr)
		}
		if parseGot != parseCase.parseWant {
			parseTesting.Fatalf("formatRenderStyleScalar(%s) = %q, want %q", parseCase.parseName, parseGot, parseCase.parseWant)
		}
	}

	if _, parseErr := formatRenderStyleScalar(nil); parseErr == nil {
		parseTesting.Fatal("expected nil style scalar to fail")
	}
}

// BenchmarkFormatRenderStyleScalarCurrentVsLegacy compares the current scalar formatter against the previous fmt-based numeric normalization path.
func BenchmarkFormatRenderStyleScalarCurrentVsLegacy(parseBenchmark *testing.B) {
	parseValues := []any{
		"red",
		true,
		int(-42),
		int64(123456789),
		uint64(99),
		float32(12.5),
		float64(0.000123),
	}
	parseBenchmark.Run("legacy", func(parseLegacyBenchmark *testing.B) {
		parseLegacyBenchmark.ReportAllocs()
		parseLegacyBenchmark.ResetTimer()
		for parseIndex := 0; parseLegacyBenchmark.Loop(); parseIndex++ {
			parseValue := parseValues[parseIndex%len(parseValues)]
			parseFormatValue, _ := parseFormatRenderStyleScalarLegacy(parseValue)
			parseFormatRenderStyleScalarBenchmarkSink = parseFormatValue
		}
	})
	parseBenchmark.Run("current", func(parseCurrentBenchmark *testing.B) {
		parseCurrentBenchmark.ReportAllocs()
		parseCurrentBenchmark.ResetTimer()
		for parseIndex := 0; parseCurrentBenchmark.Loop(); parseIndex++ {
			parseValue := parseValues[parseIndex%len(parseValues)]
			parseFormatValue, _ := formatRenderStyleScalar(parseValue)
			parseFormatRenderStyleScalarBenchmarkSink = parseFormatValue
		}
	})
}

// parseFormatRenderStyleScalarLegacy preserves the previous fmt-based numeric formatting for benchmark comparison.
func parseFormatRenderStyleScalarLegacy(parseRaw any) (string, error) {
	switch parseValue := parseRaw.(type) {
	case string:
		return parseValue, nil
	case bool:
		if parseValue {
			return "true", nil
		}
		return "false", nil
	case int:
		return fmt.Sprintf("%d", parseValue), nil
	case int8:
		return fmt.Sprintf("%d", parseValue), nil
	case int16:
		return fmt.Sprintf("%d", parseValue), nil
	case int32:
		return fmt.Sprintf("%d", parseValue), nil
	case int64:
		return fmt.Sprintf("%d", parseValue), nil
	case uint:
		return fmt.Sprintf("%d", parseValue), nil
	case uint8:
		return fmt.Sprintf("%d", parseValue), nil
	case uint16:
		return fmt.Sprintf("%d", parseValue), nil
	case uint32:
		return fmt.Sprintf("%d", parseValue), nil
	case uint64:
		return fmt.Sprintf("%d", parseValue), nil
	case float32:
		return fmt.Sprintf("%g", parseValue), nil
	case float64:
		return fmt.Sprintf("%g", parseValue), nil
	case nil:
		return "", fmt.Errorf("style value is nil")
	default:
		return "", fmt.Errorf("unsupported nested style shape %T", parseRaw)
	}
}

// TestFormatRenderStyleMapRejectsInjection pins that a map-form style value
// cannot smuggle extra CSS declarations via ';' '{' '}' or control bytes.
func TestFormatRenderStyleMapRejectsInjection(parseT *testing.T) {
	parseCases := []map[string]any{
		{"color": "red; position:fixed; top:0"},
		{"color": "red} .evil{color:blue"},
		{"color": "red\n; opacity:0"},
	}
	for parseIndex, parseCase := range parseCases {
		if _, parseErr := FormatRenderStyleValue(parseCase); parseErr == nil {
			parseT.Fatalf("case %d: expected injection rejection, got nil error", parseIndex)
		}
	}
	// A clean value still formats.
	parseOut, parseErr := FormatRenderStyleValue(map[string]any{"color": "red", "opacity": "0.5"})
	if parseErr != nil {
		parseT.Fatalf("clean style rejected: %v", parseErr)
	}
	if parseOut != "color:red;opacity:0.5" {
		parseT.Fatalf("unexpected clean style output %q", parseOut)
	}
}
