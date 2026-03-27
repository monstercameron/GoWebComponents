package runtime2

import "testing"

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
