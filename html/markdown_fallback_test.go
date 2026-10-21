package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/ui"
)

// TestMarkdownWithoutEngineRendersEscapedPlainText pins the unregistered path:
// the source is escaped inside the unavailable wrapper and never becomes markup.
func TestMarkdownWithoutEngineRendersEscapedPlainText(parseT *testing.T) {
	parsePrevious := getMarkdownEngine()
	RegisterMarkdownEngine(nil)
	defer RegisterMarkdownEngine(parsePrevious)

	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, Markdown("# hi <script>alert(1)</script>")...))
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "data-gwc-markdown-unavailable") {
		parseT.Fatalf("expected unavailable wrapper, got %q", parseMarkup)
	}
	if strings.Contains(parseMarkup, "<script>") || strings.Contains(parseMarkup, "<h1>") {
		parseT.Fatalf("fallback must not emit markup, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "&lt;script&gt;") {
		parseT.Fatalf("expected escaped source, got %q", parseMarkup)
	}
}

// TestRegisterMarkdownEngineLastWins pins that a later registration replaces an earlier one.
func TestRegisterMarkdownEngineLastWins(parseT *testing.T) {
	parsePrevious := getMarkdownEngine()
	defer RegisterMarkdownEngine(parsePrevious)

	RegisterMarkdownEngine(func(string, MarkdownRenderOptions) []ui.Node { return []ui.Node{Text("first")} })
	RegisterMarkdownEngine(func(string, MarkdownRenderOptions) []ui.Node { return []ui.Node{Text("second")} })
	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, Markdown("x")...))
	if parseErr != nil || !strings.Contains(parseMarkup, "second") || strings.Contains(parseMarkup, "first") {
		parseT.Fatalf("expected last registration to win, got %q (%v)", parseMarkup, parseErr)
	}
}
