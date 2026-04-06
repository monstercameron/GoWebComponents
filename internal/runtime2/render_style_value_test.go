package runtime2

import (
	"fmt"
	"testing"
)

var parseFormatRenderStyleScalarBenchmarkSink string

// TestFormatRenderStyleValueEquivalentStylesNormalizeConsistently verifies logically equivalent styles normalize identically.
func TestFormatRenderStyleValueEquivalentStylesNormalizeConsistently(parseTesting *testing.T) {
	parseStyleA := map[string]interface{}{
		"color":  "red",
		"margin": 0,
	}
	parseStyleB := map[string]interface{}{
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
	parseNestedStyle := map[string]interface{}{
		"color": map[string]string{
			"base": "red",
		},
	}
	_, formatStyleErr := FormatRenderStyleValue(parseNestedStyle)
	if formatStyleErr == nil {
		parseTesting.Fatal("FormatRenderStyleValue(nested payload) error = nil, want error")
	}
}

// BenchmarkFormatRenderStyleScalarCurrentVsLegacy compares the current scalar formatter against the previous fmt-based numeric normalization path.
func BenchmarkFormatRenderStyleScalarCurrentVsLegacy(parseBenchmark *testing.B) {
	parseValues := []interface{}{
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
func parseFormatRenderStyleScalarLegacy(parseRaw interface{}) (string, error) {
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
