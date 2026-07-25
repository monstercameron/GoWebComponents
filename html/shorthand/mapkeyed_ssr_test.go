package shorthand_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	sh "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestMapKeyedComponentRendersThroughSSR proves the keyed per-item components
// render server-side via RenderToString (SSR-compatible, so they hydrate).
func TestMapKeyedComponentRendersThroughSSR(parseT *testing.T) {
	parseRows := []string{"alpha", "beta", "gamma"}
	parseNode := html.Div(html.Props{ID: "list"},
		sh.MapKeyedComponent(parseRows,
			func(parseR string) any { return parseR },
			func(parseR string) ui.Node {
				return html.P(html.Props{ID: "row-" + parseR}, html.Text(parseR))
			})...,
	)

	parseOut, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	for _, parseR := range parseRows {
		if !strings.Contains(parseOut, ">"+parseR+"<") {
			parseT.Fatalf("SSR output missing row %q:\n%s", parseR, parseOut)
		}
	}
	if parseCount := strings.Count(parseOut, "<p"); parseCount != 3 {
		parseT.Fatalf("expected 3 <p> rows, got %d:\n%s", parseCount, parseOut)
	}
}
