package extract

import "testing"

// TestExtractUnquotesEscapedStringLiterals pins that keys written with Go escape sequences are
// extracted as their runtime value, so they match the catalog key the program actually looks up.
func TestExtractUnquotesEscapedStringLiterals(parseT *testing.T) {
	parseSource := "package p\nfunc f(r R) {\n\tr.T(\"ns\", \"say \\\"hi\\\"\")\n\tr.T(\"ns\", \"tab\\tkey\")\n\tr.T(\"ns\", `raw\\n`)\n}\n"
	parseResult, parseErr := ExtractFromSource("p.go", parseSource)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseWant := []string{"say \"hi\"", "tab\tkey", "raw\\n"}
	if len(parseResult.Messages) != len(parseWant) {
		parseT.Fatalf("messages = %+v", parseResult.Messages)
	}
	for parseIndex, parseKey := range parseWant {
		if parseResult.Messages[parseIndex].Key != parseKey {
			parseT.Errorf("message %d key = %q, want %q", parseIndex, parseResult.Messages[parseIndex].Key, parseKey)
		}
	}
}
