package markdown

import (
	"strings"
	"testing"

	. "github.com/monstercameron/GoWebComponents/v7/html"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// renderMarkdownMarkup renders markdown nodes to markup for assertions.
func renderMarkdownMarkup(parseT *testing.T, parseNodes []ui.Node) string {
	parseT.Helper()
	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	return parseMarkup
}

func TestMarkdownURLSchemeOptions(parseT *testing.T) {
	const parseSource = "[a](https://a.example) [b](mailto:x@y.z) [c](ftp://c.example) [d](/rel)"
	parseDefault := renderMarkdownMarkup(parseT, Markdown(parseSource))
	if !strings.Contains(parseDefault, "https://a.example") || !strings.Contains(parseDefault, "mailto:x@y.z") || strings.Contains(parseDefault, "ftp://") {
		parseT.Errorf("default allowlist wrong: %s", parseDefault)
	}
	parseFTP := renderMarkdownMarkup(parseT, Markdown(parseSource, WithURLSchemes("ftp")))
	if !strings.Contains(parseFTP, "ftp://c.example") || strings.Contains(parseFTP, "https://a.example") {
		parseT.Errorf("explicit allowlist wrong: %s", parseFTP)
	}
	parseNone := renderMarkdownMarkup(parseT, Markdown(parseSource, WithURLSchemes()))
	if strings.Contains(parseNone, "https://") || strings.Contains(parseNone, "mailto:") || strings.Contains(parseNone, "ftp://") || !strings.Contains(parseNone, "/rel") {
		parseT.Errorf("empty allowlist must allow no schemes but keep relative links: %s", parseNone)
	}
}

func TestMarkdownAppliesEveryOptionAndLegacyStruct(parseT *testing.T) {
	const parseSource = "[c](ftp://c.example)"
	parseMarkup := renderMarkdownMarkup(parseT, Markdown(parseSource, MarkdownRenderOptions{LinkRel: "nofollow"}, WithURLSchemes("ftp")))
	if !strings.Contains(parseMarkup, "ftp://c.example") || !strings.Contains(parseMarkup, "nofollow") {
		parseT.Errorf("struct option and WithURLSchemes must both apply: %s", parseMarkup)
	}
	parseLegacy := renderMarkdownMarkup(parseT, Markdown(parseSource, MarkdownRenderOptions{AllowedURLSchemes: []string{"ftp"}}))
	if !strings.Contains(parseLegacy, "ftp://c.example") {
		parseT.Errorf("legacy struct form must keep working: %s", parseLegacy)
	}
}
