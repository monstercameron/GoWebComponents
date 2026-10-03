package html_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	_ "github.com/monstercameron/GoWebComponents/v6/html/markdown"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestMarkdownRendersThroughShorthand verifies Markdown delegates to
// RenderMarkdown and produces semantic HTML once the engine is imported.
func TestMarkdownRendersThroughShorthand(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(html.Div(html.Props{}, html.Markdown("# Title\n\nBody text.")...))
	if parseErr != nil {
		parseT.Fatalf("expected Markdown render, got %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "<h1>Title</h1>") || !strings.Contains(parseMarkup, "Body text.") {
		parseT.Fatalf("expected markdown HTML, got %q", parseMarkup)
	}
}
