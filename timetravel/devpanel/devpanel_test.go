package devpanel_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/timetravel"
	"github.com/monstercameron/GoWebComponents/v5/timetravel/devpanel"
)

// TestPanelRendersTimelineAndMarksCurrent renders the panel headlessly through the real
// runtime and asserts the snapshot labels appear and the current one is marked — the engine
// dogfooded as a GWC component.
func TestPanelRendersTimelineAndMarksCurrent(parseT *testing.T) {
	parseHistory := timetravel.New(0, "a")
	parseHistory.Record("edit-b", "b")
	parseHistory.Record("edit-c", "c")
	parseHistory.Undo() // cursor on index 1 → label "edit-b"

	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, devpanel.Panel(devpanel.Props{Model: parseHistory})); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}

	parseText := collectText(parseAdapter, parseRoot)
	for _, parseWant := range []string{"initial", "edit-b", "edit-c", "Back", "Forward"} {
		if !strings.Contains(parseText, parseWant) {
			parseT.Fatalf("panel DOM missing %q; got %q", parseWant, parseText)
		}
	}

	parseCurrent := findByClass(parseAdapter, parseRoot, "gwc-timetravel-current")
	if len(parseCurrent) != 1 {
		parseT.Fatalf("expected exactly one current item, got %d", len(parseCurrent))
	}
	if parseLabel := collectText(parseAdapter, parseCurrent[0]); parseLabel != "edit-b" {
		parseT.Fatalf("expected the current marker on edit-b, got %q", parseLabel)
	}
}

// TestPanelHandlesNilModel proves a nil model renders an empty, panic-free panel.
func TestPanelHandlesNilModel(parseT *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, devpanel.Panel(devpanel.Props{})); parseErr != nil {
		parseT.Fatalf("nil-model panel should still render: %v", parseErr)
	}
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

func findByClass(parseAdapter *mockdom.MockDOMAdapter, parseNode runtime.DOMNode, parseWant string) []*mockdom.MockDOMNode {
	var parseMatches []*mockdom.MockDOMNode
	parseMock, parseOk := parseNode.(*mockdom.MockDOMNode)
	if !parseOk {
		return parseMatches
	}
	if strings.Contains(parseMock.Attrs["class"], parseWant) {
		parseMatches = append(parseMatches, parseMock)
	}
	for _, parseChild := range parseAdapter.GetChildren(parseNode) {
		parseMatches = append(parseMatches, findByClass(parseAdapter, parseChild, parseWant)...)
	}
	return parseMatches
}
