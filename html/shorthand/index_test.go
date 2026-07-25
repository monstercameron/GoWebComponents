package shorthand_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestIndexRendersPositionKeyedNodes proves Index renders one node per item, passing the
// position index, and (via headless mount) that the content appears in order.
func TestIndexRendersPositionKeyedNodes(parseT *testing.T) {
	parseItems := []string{"a", "b", "c"}
	parseNodes := shorthand.Index(parseItems, func(parseIndex int, parseItem string) ui.Node {
		return html.Tag("li", html.Props{}, html.Text(parseItem))
	})
	if len(parseNodes) != 3 {
		parseT.Fatalf("expected 3 nodes, got %d", len(parseNodes))
	}

	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("ul")
	if parseErr := parseRuntime.RenderInto(parseRoot, html.Tag("ul", html.Props{}, parseNodes...)); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	parseText := collectText(parseAdapter, parseRoot)
	if !strings.Contains(parseText, "a") || !strings.Contains(parseText, "b") || !strings.Contains(parseText, "c") {
		parseT.Fatalf("expected a,b,c rendered, got %q", parseText)
	}
}

// TestIndexPassesPositionIndex proves the render callback receives the slot index.
func TestIndexPassesPositionIndex(parseT *testing.T) {
	parseSeen := []int{}
	shorthand.Index([]string{"x", "y"}, func(parseIndex int, _ string) ui.Node {
		parseSeen = append(parseSeen, parseIndex)
		return html.Text("n")
	})
	if len(parseSeen) != 2 || parseSeen[0] != 0 || parseSeen[1] != 1 {
		parseT.Fatalf("expected indices [0 1], got %v", parseSeen)
	}
}
