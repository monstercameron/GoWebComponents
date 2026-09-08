package shorthand_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestDeferDoesNotBuildContentUntilShown proves the core deferral guarantee: the content
// thunk (the expensive subtree) is not invoked while the view is deferred, and renders the
// placeholder instead.
func TestDeferDoesNotBuildContentUntilShown(parseT *testing.T) {
	parseBuilt := 0
	parseContent := func() ui.Node {
		parseBuilt++
		return html.Text("heavy")
	}

	parseDeferred := shorthand.Defer(false, html.Text("loading"), parseContent)
	if parseBuilt != 0 {
		parseT.Fatalf("content must not be built while deferred, built=%d", parseBuilt)
	}
	if parseText := render(parseT, parseDeferred); !strings.Contains(parseText, "loading") {
		parseT.Fatalf("deferred view should show the placeholder, got %q", parseText)
	}

	parseShown := shorthand.Defer(true, html.Text("loading"), parseContent)
	if parseBuilt != 1 {
		parseT.Fatalf("content should build exactly once when shown, built=%d", parseBuilt)
	}
	if parseText := render(parseT, parseShown); !strings.Contains(parseText, "heavy") {
		parseT.Fatalf("shown view should render the content, got %q", parseText)
	}
}

func render(parseT *testing.T, parseNode *runtime.Element) string {
	parseT.Helper()
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, parseNode); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	return collectText(parseAdapter, parseRoot)
}

func collectText(parseAdapter *mockdom.MockDOMAdapter, parseNode runtime.DOMNode) string {
	parseMock, parseOk := parseNode.(*mockdom.MockDOMNode)
	if !parseOk {
		return ""
	}
	parseText := parseMock.TextContent
	for _, parseChild := range parseAdapter.GetChildren(parseNode) {
		parseText += collectText(parseAdapter, parseChild)
	}
	return parseText
}
