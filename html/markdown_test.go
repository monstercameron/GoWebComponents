package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
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

func TestResolveMarkdownHrefEdgeCases(t *testing.T) {
	if got := ResolveMarkdownHref("docs/start.md", ""); got != "" {
		t.Fatalf("expected empty destination to resolve empty, got %q", got)
	}
	if got := ResolveMarkdownHref("docs/start.md", "%zz"); got != "%zz" {
		t.Fatalf("expected invalid URL destination passthrough, got %q", got)
	}
	if got := ResolveMarkdownHref("docs/start.md", "#section"); got != "#section" {
		t.Fatalf("expected fragment destination passthrough, got %q", got)
	}
	if got := ResolveMarkdownHref("docs/start.md", "/guide"); got != "/guide" {
		t.Fatalf("expected absolute path destination passthrough, got %q", got)
	}
	if got := ResolveMarkdownHref("", "guide.md"); got != "guide.md" {
		t.Fatalf("expected blank source path passthrough, got %q", got)
	}
	if got := ResolveMarkdownHref("https://example.test/docs/start.md", "guide.md?q=1"); got != "https://example.test/docs/guide.md?q=1" {
		t.Fatalf("expected absolute source path resolution, got %q", got)
	}
}

func TestRenderMarkdownCoversAdditionalBlockAndInlineBranches(t *testing.T) {
	source := strings.Join([]string{
		"###### Heading Six",
		"",
		"> Quoted context",
		"",
		"3. Third",
		"4. Fourth",
		"",
		"Paragraph soft",
		"break and hard  ",
		"break with **strong** and *em* and [custom](guide.md) and <https://example.test/docs>.",
		"",
		"---",
	}, "\n")

	nodes := RenderMarkdown(source, MarkdownRenderOptions{
		SourcePath: "docs/start.md",
		ResolveHref: func(sourcePath, destination string) string {
			return "/resolved/" + strings.TrimSpace(destination)
		},
		Classes: MarkdownClasses{
			Heading6:       "h6",
			Blockquote:     "quote",
			List:           "list",
			OrderedList:    "ordered",
			Strong:         "strong",
			Emphasis:       "em",
			Link:           "link",
			HorizontalRule: "rule",
		},
	})

	markup, err := ui.RenderToString(Div(Props{}, nodes...))
	if err != nil {
		t.Fatalf("expected markdown render to stringify, got %v", err)
	}
	for _, snippet := range []string{
		`<h6 class="h6">Heading Six</h6>`,
		`<blockquote class="quote"><p>Quoted context</p></blockquote>`,
		`<ol class="list ordered" start="3">`,
		`<strong class="strong">strong</strong>`,
		`<em class="em">em</em>`,
		`<a class="link" href="/resolved/guide.md">custom</a>`,
		`<a class="link" href="https://example.test/docs">https://example.test/docs</a>`,
		`<br>`,
		`<hr class="rule">`,
	} {
		if !strings.Contains(markup, snippet) {
			t.Fatalf("expected markdown markup to contain %q, got %s", snippet, markup)
		}
	}
}

func TestMarkdownHelperBranchesAndFallbackNodes(t *testing.T) {
	if got := markdownLinesText(nil, nil); got != "" {
		t.Fatalf("expected nil markdown lines text to be empty, got %q", got)
	}
	if got := markdownPlainText(nil, []byte("ignored")); got != "" {
		t.Fatalf("expected nil markdown node plain text to be empty, got %q", got)
	}

	source := []byte("alpha")
	textNode := ast.NewTextSegment(text.NewSegment(0, 5))
	rendered, ok := renderMarkdownBlock(textNode, source, MarkdownRenderOptions{})
	if !ok || rendered == nil || rendered.Type != "p" {
		t.Fatalf("expected text-node markdown block fallback to paragraph, got ok=%t rendered=%#v", ok, rendered)
	}

	emptyNode := ast.NewTextSegment(text.NewSegment(0, 0))
	rendered, ok = renderMarkdownBlock(emptyNode, source, MarkdownRenderOptions{})
	if ok || rendered != nil {
		t.Fatalf("expected empty text-node markdown block fallback to skip render, got ok=%t rendered=%#v", ok, rendered)
	}

	inlineContainer := ast.NewParagraph()
	inlineContainer.AppendChild(inlineContainer, ast.NewTextSegment(text.NewSegment(0, 5)))
	inline := renderMarkdownInline(inlineContainer, source, MarkdownRenderOptions{})
	if len(inline) != 1 || inline[0] == nil || inline[0].TextContent != "alpha" {
		t.Fatalf("expected default inline fallback to text node, got %#v", inline)
	}

	emptyInline := renderMarkdownInline(ast.NewParagraph(), source, MarkdownRenderOptions{})
	if len(emptyInline) != 0 {
		t.Fatalf("expected empty inline fallback to produce no nodes, got %#v", emptyInline)
	}
}
