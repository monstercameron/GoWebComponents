package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestResolveMarkdownHrefResolvesRelativeDestinations(t *testing.T) {
	resolved := ResolveMarkdownHref("assets/docs/start-here.md", "troubleshooting.md#hydration")
	if resolved != "assets/docs/troubleshooting.md#hydration" {
		t.Fatalf("expected relative href resolution, got %q", resolved)
	}

	absolute := ResolveMarkdownHref("assets/docs/start-here.md", "https://example.test/docs")
	if absolute != "https://example.test/docs" {
		t.Fatalf("expected absolute href passthrough, got %q", absolute)
	}
}

func TestRenderMarkdownRendersSemanticHTML(t *testing.T) {
	nodes := RenderMarkdown("# Start Here\n\nParagraph with [guide](troubleshooting.md) and `code`.\n\n- First\n- Second\n\n```go\nfmt.Println(\"hi\")\n```", MarkdownRenderOptions{SourcePath: "assets/docs/start-here.md"})
	markup, err := ui.RenderToString(Div(Props{}, nodes...))
	if err != nil {
		t.Fatalf("expected markdown render to stringify, got %v", err)
	}
	for _, snippet := range []string{"<h1>Start Here</h1>", "<a href=\"assets/docs/troubleshooting.md\">guide</a>", "<code>code</code>", "<ul>", "First", "fmt.Println(&#34;hi&#34;)"} {
		if !strings.Contains(markup, snippet) {
			t.Fatalf("expected markdown markup to contain %q, got %s", snippet, markup)
		}
	}
}

func TestRenderMarkdownAppliesOptions(t *testing.T) {
	nodes := RenderMarkdown("## Heading\n\n[Docs](guide.md)\n\n```txt\nhello\n```", MarkdownRenderOptions{
		SourcePath:     "assets/docs/start-here.md",
		CodeBlockLabel: "Snippet",
		LinkTarget:     "_blank",
		LinkRel:        "noreferrer",
		Classes: MarkdownClasses{
			Heading2:           "heading-two",
			Link:               "markdown-link",
			CodeBlockContainer: "code-shell",
			CodeBlockLabel:     "code-label",
			CodeBlockPre:       "code-pre",
		},
	})
	markup, err := ui.RenderToString(Div(Props{}, nodes...))
	if err != nil {
		t.Fatalf("expected optioned markdown render to stringify, got %v", err)
	}
	for _, snippet := range []string{"class=\"heading-two\"", "class=\"markdown-link\"", "target=\"_blank\"", "rel=\"noreferrer\"", "class=\"code-shell\"", "Snippet", "class=\"code-pre\""} {
		if !strings.Contains(markup, snippet) {
			t.Fatalf("expected markdown markup to contain %q, got %s", snippet, markup)
		}
	}
}
