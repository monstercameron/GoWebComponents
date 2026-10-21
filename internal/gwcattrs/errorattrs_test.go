package gwcattrs

import (
	"strings"
	"testing"
)

// TestParseErrorCode covers the rendered errcode forms and non-matches.
func TestParseErrorCode(parseT *testing.T) {
	for _, parseCase := range []struct{ in, want string }{
		{"[GWC-FRAMEWORK-USAGE] misuse: x", "GWC-FRAMEWORK-USAGE"},
		{"  [GWC-A1] y", "GWC-A1"},
		{"plain failure", ""},
		{"[OTHER-1] z", ""},
		{"[GWC-bad code] z", ""},
		{"[GWC-OPEN", ""},
		{"", ""},
	} {
		if parseGot := ParseErrorCode(parseCase.in); parseGot != parseCase.want {
			parseT.Errorf("ParseErrorCode(%q) = %q, want %q", parseCase.in, parseGot, parseCase.want)
		}
	}
}

// TestNormalizeErrorAttrs covers code derivation, precedence, stack trimming and capping.
func TestNormalizeErrorAttrs(parseT *testing.T) {
	parseIn := map[string]string{KeyComponentStack: "  App > Page \n"}
	parseOut := NormalizeErrorAttrs(parseIn, "[GWC-X] boom")
	if parseOut[KeyErrorCode] != "GWC-X" || parseOut[KeyComponentStack] != "App > Page" {
		parseT.Fatalf("out = %v", parseOut)
	}
	if parseIn[KeyComponentStack] != "  App > Page \n" || len(parseIn) != 1 {
		parseT.Fatalf("input mutated: %v", parseIn)
	}
	if parseGot := NormalizeErrorAttrs(map[string]string{KeyErrorCode: "GWC-OWN"}, "[GWC-X] boom")[KeyErrorCode]; parseGot != "GWC-OWN" {
		parseT.Fatalf("explicit code overridden: %q", parseGot)
	}
	if parseGot := NormalizeErrorAttrs(map[string]string{KeyErrorCode: " "}, "[GWC-X] boom")[KeyErrorCode]; parseGot != "GWC-X" {
		parseT.Fatalf("blank code not replaced: %q", parseGot)
	}
	if _, parseHas := NormalizeErrorAttrs(map[string]string{KeyComponentStack: "  "}, "")[KeyComponentStack]; parseHas {
		parseT.Fatal("blank stack kept")
	}
	if parseGot := NormalizeErrorAttrs(map[string]string{KeyComponentStack: strings.Repeat("a", 5000)}, "")[KeyComponentStack]; len(parseGot) != MaxComponentStackBytes {
		parseT.Fatalf("stack len = %d", len(parseGot))
	}
	if len(NormalizeErrorAttrs(nil, "plain")) != 0 {
		parseT.Fatal("nil input produced attrs")
	}
}