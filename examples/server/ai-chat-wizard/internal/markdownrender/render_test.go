package markdownrender

import (
	"strings"
	"testing"
)

func TestNormalizeFenceLanguageAliases(parseT *testing.T) {
	parseInput := "```golang title=demo\nfmt.Println(\"hi\")\n```\n~~~c#\nConsole.WriteLine(\"hi\");\n~~~\n"
	parseNormalized := parseNormalizeFenceLanguageAliases(parseInput)
	if !strings.Contains(parseNormalized, "``` go title=demo") {
		parseT.Fatalf("expected golang alias normalization, got %q", parseNormalized)
	}
	if !strings.Contains(parseNormalized, "~~~ csharp") {
		parseT.Fatalf("expected c# alias normalization, got %q", parseNormalized)
	}
}

func TestNormalizeFenceHelpers(parseT *testing.T) {
	if parseGot := parseLeadingFenceLength("```go"); parseGot != 3 {
		parseT.Fatalf("leadingFenceLength() = %d, want 3", parseGot)
	}
	if parseGot2 := parseLeadingFenceLength(""); parseGot2 != 0 {
		parseT.Fatalf("leadingFenceLength(empty) = %d, want 0", parseGot2)
	}
	if parseGot3 := parseNormalizeFenceInfoString("js linenos"); parseGot3 != "javascript linenos" {
		parseT.Fatalf("normalizeFenceInfoString() = %q, want javascript linenos", parseGot3)
	}
	if parseGot4 := parseNormalizeFenceInfoString("   "); parseGot4 != "   " {
		parseT.Fatalf("normalizeFenceInfoString(blank) = %q, want original spacing", parseGot4)
	}
}

func TestRenderConvertsMarkdownAndNormalizesAliases(parseT *testing.T) {
	parseMarkup, parseErr := Render("# Heading\n\n```sh\necho hi\n```\n")
	if parseErr != nil {
		parseT.Fatalf("Render(): %v", parseErr)
	}
	for _, parseExpected := range []string{"<h1>Heading</h1>", "<pre", "echo hi"} {
		if !strings.Contains(parseMarkup, parseExpected) {
			parseT.Fatalf("expected rendered markup to contain %q, got %s", parseExpected, parseMarkup)
		}
	}

	parsePlain := "no fences here"
	if parseGot := parseNormalizeFenceLanguageAliases(parsePlain); parseGot != parsePlain {
		parseT.Fatalf("expected plain input to remain unchanged, got %q", parseGot)
	}
}
