package markdownrender

import (
	"strings"
	"testing"
)

func TestNormalizeFenceLanguageAliases(t *testing.T) {
	input := "```golang title=demo\nfmt.Println(\"hi\")\n```\n~~~c#\nConsole.WriteLine(\"hi\");\n~~~\n"
	normalized := normalizeFenceLanguageAliases(input)
	if !strings.Contains(normalized, "``` go title=demo") {
		t.Fatalf("expected golang alias normalization, got %q", normalized)
	}
	if !strings.Contains(normalized, "~~~ csharp") {
		t.Fatalf("expected c# alias normalization, got %q", normalized)
	}
}

func TestNormalizeFenceHelpers(t *testing.T) {
	if got := leadingFenceLength("```go"); got != 3 {
		t.Fatalf("leadingFenceLength() = %d, want 3", got)
	}
	if got := leadingFenceLength(""); got != 0 {
		t.Fatalf("leadingFenceLength(empty) = %d, want 0", got)
	}
	if got := normalizeFenceInfoString("js linenos"); got != "javascript linenos" {
		t.Fatalf("normalizeFenceInfoString() = %q, want javascript linenos", got)
	}
	if got := normalizeFenceInfoString("   "); got != "   " {
		t.Fatalf("normalizeFenceInfoString(blank) = %q, want original spacing", got)
	}
}

func TestRenderConvertsMarkdownAndNormalizesAliases(t *testing.T) {
	markup, err := Render("# Heading\n\n```sh\necho hi\n```\n")
	if err != nil {
		t.Fatalf("Render(): %v", err)
	}
	for _, expected := range []string{"<h1>Heading</h1>", "<pre", "echo hi"} {
		if !strings.Contains(markup, expected) {
			t.Fatalf("expected rendered markup to contain %q, got %s", expected, markup)
		}
	}

	plain := "no fences here"
	if got := normalizeFenceLanguageAliases(plain); got != plain {
		t.Fatalf("expected plain input to remain unchanged, got %q", got)
	}
}
