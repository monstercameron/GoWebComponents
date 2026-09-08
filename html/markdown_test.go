package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestSanitizeMarkdownHrefDropsDangerousSchemes(parseT *testing.T) {
	parseDropped := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"  javascript:alert(1)",
		"java\tscript:alert(1)",
		"java\nscript:alert(1)",
		"jAvAsCrIpT:alert(1)",
		"vbscript:msgbox(1)",
		"data:text/html,<script>alert(1)</script>",
		"data:image/png;base64,AAAA",
		"\x01javascript:alert(1)",
	}
	for _, parseCase := range parseDropped {
		if parseGot := SanitizeMarkdownHref(parseCase, nil); parseGot != "" {
			parseT.Fatalf("expected %q to be dropped, got %q", parseCase, parseGot)
		}
	}

	parseKept := []string{
		"https://example.test/docs",
		"http://example.test",
		"mailto:dev@example.test",
		"guide.md#hydration",
		"./relative/page",
		"/absolute/path",
		"#section",
		"//cdn.example.test/asset.js",
	}
	for _, parseCase := range parseKept {
		if parseGot := SanitizeMarkdownHref(parseCase, nil); parseGot != parseCase {
			parseT.Fatalf("expected %q to survive unchanged, got %q", parseCase, parseGot)
		}
	}

	// A configurable allowlist round-trips: opting data in keeps inline images.
	if parseGot := SanitizeMarkdownHref("data:image/png;base64,AAAA", []string{"http", "https", "data"}); parseGot == "" {
		parseT.Fatal("expected data: to survive when explicitly allowed")
	}
	// And a custom allowlist still drops what it does not list.
	if parseGot := SanitizeMarkdownHref("https://example.test", []string{"mailto"}); parseGot != "" {
		parseT.Fatal("expected https to be dropped when only mailto is allowed")
	}
}

func TestRenderMarkdownNeutralizesXSSAcrossSinks(parseT *testing.T) {
	parseMarkdown := strings.Join([]string{
		"[click](javascript:alert(1))",
		"",
		"![logo](data:text/html,<script>alert(1)</script>)",
		"",
		"<https://example.test/safe>",
		"",
		"[doc](guide.md) and [home](#top)",
	}, "\n")
	parseNodes := RenderMarkdown(parseMarkdown, MarkdownRenderOptions{SourcePath: "docs/start.md"})
	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("expected markdown render to stringify, got %v", parseErr)
	}
	for _, parseForbidden := range []string{"javascript:", "data:text/html", "href=\"javascript", "src=\"data:"} {
		if strings.Contains(parseMarkup, parseForbidden) {
			parseT.Fatalf("rendered markup leaked dangerous content %q:\n%s", parseForbidden, parseMarkup)
		}
	}
	for _, parseExpected := range []string{"https://example.test/safe", "href=\"docs/guide.md\"", "href=\"#top\""} {
		if !strings.Contains(parseMarkup, parseExpected) {
			parseT.Fatalf("expected legitimate link %q to survive:\n%s", parseExpected, parseMarkup)
		}
	}
}

func TestResolveMarkdownHrefResolvesRelativeDestinations(parseT *testing.T) {
	parseResolved := ResolveMarkdownHref("assets/docs/start-here.md", "troubleshooting.md#hydration")
	if parseResolved != "assets/docs/troubleshooting.md#hydration" {
		parseT.Fatalf("expected relative href resolution, got %q", parseResolved)
	}

	parseAbsolute := ResolveMarkdownHref("assets/docs/start-here.md", "https://example.test/docs")
	if parseAbsolute != "https://example.test/docs" {
		parseT.Fatalf("expected absolute href passthrough, got %q", parseAbsolute)
	}
}

func TestRenderMarkdownRendersSemanticHTML(parseT *testing.T) {
	parseNodes := RenderMarkdown("# Start Here\n\nParagraph with [guide](troubleshooting.md) and `code`.\n\n- First\n- Second\n\n```go\nfmt.Println(\"hi\")\n```", MarkdownRenderOptions{SourcePath: "assets/docs/start-here.md"})
	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("expected markdown render to stringify, got %v", parseErr)
	}
	for _, parseSnippet := range []string{"<h1>Start Here</h1>", "<a href=\"assets/docs/troubleshooting.md\">guide</a>", "<code>code</code>", "<ul>", "First", "fmt.Println(&#34;hi&#34;)"} {
		if !strings.Contains(parseMarkup, parseSnippet) {
			parseT.Fatalf("expected markdown markup to contain %q, got %s", parseSnippet, parseMarkup)
		}
	}
}

func TestRenderMarkdownEscapesRawHTMLAndEnablesGFM(parseT *testing.T) {
	parseSource := strings.Join([]string{
		"<section><strong>raw</strong></section>",
		"",
		"Paragraph with <span>inline</span> and ~~old~~.",
		"",
		"https://example.test/gfm",
		"",
		"- [x] Done",
		"- [ ] Todo",
		"",
		"| A | B |",
		"| - | - |",
		"| 1 | 2 |",
	}, "\n")

	parseNodes := RenderMarkdown(parseSource)
	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("expected markdown render to stringify, got %v", parseErr)
	}

	for _, parseSnippet := range []string{
		`&lt;section&gt;&lt;strong&gt;raw&lt;/strong&gt;&lt;/section&gt;`,
		`Paragraph with &lt;span&gt;inline&lt;/span&gt; and <del>old</del>.`,
		`<a href="https://example.test/gfm">https://example.test/gfm</a>`,
		`<input checked disabled type="checkbox">`,
		`<input disabled type="checkbox">`,
		`<table><thead><tr><th>A</th><th>B</th></tr></thead><tbody><tr><td>1</td><td>2</td></tr></tbody></table>`,
	} {
		if !strings.Contains(parseMarkup, parseSnippet) {
			parseT.Fatalf("expected GFM/raw-HTML markdown markup to contain %q, got %s", parseSnippet, parseMarkup)
		}
	}
}

func TestRenderMarkdownAppliesOptions(parseT *testing.T) {
	parseNodes := RenderMarkdown("## Heading\n\n[Docs](guide.md)\n\n```txt\nhello\n```", MarkdownRenderOptions{
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
	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("expected optioned markdown render to stringify, got %v", parseErr)
	}
	for _, parseSnippet := range []string{"class=\"heading-two\"", "class=\"markdown-link\"", "target=\"_blank\"", "rel=\"noreferrer\"", "class=\"code-shell\"", "Snippet", "class=\"code-pre\""} {
		if !strings.Contains(parseMarkup, parseSnippet) {
			parseT.Fatalf("expected markdown markup to contain %q, got %s", parseSnippet, parseMarkup)
		}
	}
}

func TestResolveMarkdownHrefEdgeCases(parseT *testing.T) {
	if parseGot := ResolveMarkdownHref("docs/start.md", ""); parseGot != "" {
		parseT.Fatalf("expected empty destination to resolve empty, got %q", parseGot)
	}
	if parseGot2 := ResolveMarkdownHref("docs/start.md", "%zz"); parseGot2 != "%zz" {
		parseT.Fatalf("expected invalid URL destination passthrough, got %q", parseGot2)
	}
	if parseGot3 := ResolveMarkdownHref("docs/start.md", "#section"); parseGot3 != "#section" {
		parseT.Fatalf("expected fragment destination passthrough, got %q", parseGot3)
	}
	if parseGot4 := ResolveMarkdownHref("docs/start.md", "/guide"); parseGot4 != "/guide" {
		parseT.Fatalf("expected absolute path destination passthrough, got %q", parseGot4)
	}
	if parseGot5 := ResolveMarkdownHref("", "guide.md"); parseGot5 != "guide.md" {
		parseT.Fatalf("expected blank source path passthrough, got %q", parseGot5)
	}
	if parseGot6 := ResolveMarkdownHref("https://example.test/docs/start.md", "guide.md?q=1"); parseGot6 != "https://example.test/docs/guide.md?q=1" {
		parseT.Fatalf("expected absolute source path resolution, got %q", parseGot6)
	}
}

func TestRenderMarkdownCoversAdditionalBlockAndInlineBranches(parseT *testing.T) {
	parseSource := strings.Join([]string{
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

	parseNodes := RenderMarkdown(parseSource, MarkdownRenderOptions{
		SourcePath: "docs/start.md",
		ResolveHref: func(parseSourcePath, parseDestination string) string {
			return "/resolved/" + strings.TrimSpace(parseDestination)
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

	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("expected markdown render to stringify, got %v", parseErr)
	}
	for _, parseSnippet := range []string{
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
		if !strings.Contains(parseMarkup, parseSnippet) {
			parseT.Fatalf("expected markdown markup to contain %q, got %s", parseSnippet, parseMarkup)
		}
	}
}

func TestMarkdownHelperBranchesAndFallbackNodes(parseT *testing.T) {
	if parseGot := markdownLinesText(nil, nil); parseGot != "" {
		parseT.Fatalf("expected nil markdown lines text to be empty, got %q", parseGot)
	}
	if parseGot2 := markdownPlainText(nil, []byte("ignored")); parseGot2 != "" {
		parseT.Fatalf("expected nil markdown node plain text to be empty, got %q", parseGot2)
	}

	parseSource := []byte("alpha")
	parseTextNode := ast.NewTextSegment(text.NewSegment(0, 5))
	parseRendered, parseOk := renderMarkdownBlock(parseTextNode, parseSource, MarkdownRenderOptions{}, 0)
	if !parseOk || parseRendered == nil || parseRendered.Type != "p" {
		parseT.Fatalf("expected text-node markdown block fallback to paragraph, got ok=%t rendered=%#v", parseOk, parseRendered)
	}

	parseEmptyNode := ast.NewTextSegment(text.NewSegment(0, 0))
	parseRendered, parseOk = renderMarkdownBlock(parseEmptyNode, parseSource, MarkdownRenderOptions{}, 0)
	if parseOk || parseRendered != nil {
		parseT.Fatalf("expected empty text-node markdown block fallback to skip render, got ok=%t rendered=%#v", parseOk, parseRendered)
	}

	parseInlineContainer := ast.NewParagraph()
	parseInlineContainer.AppendChild(parseInlineContainer, ast.NewTextSegment(text.NewSegment(0, 5)))
	parseInline := renderMarkdownInline(parseInlineContainer, parseSource, MarkdownRenderOptions{}, 0)
	if len(parseInline) != 1 || parseInline[0] == nil || parseInline[0].TextContent != "alpha" {
		parseT.Fatalf("expected default inline fallback to text node, got %#v", parseInline)
	}

	parseEmptyInline := renderMarkdownInline(ast.NewParagraph(), parseSource, MarkdownRenderOptions{}, 0)
	if len(parseEmptyInline) != 0 {
		parseT.Fatalf("expected empty inline fallback to produce no nodes, got %#v", parseEmptyInline)
	}
}
