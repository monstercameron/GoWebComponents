//go:build !js || !wasm

package render

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestSerializedSubtreeMountFiresAndBinds pins the serialized-mount strategy:
// a mount of an all-fast-lane subtree must go through CreateHTMLSubtree (one
// parse) instead of per-node SetTextContent calls, and the binding walk must
// leave every fiber attached to a live node so later targeted updates land.
func TestSerializedSubtreeMountFiresAndBinds(t *testing.T) {
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)

	var storeSetLabel func(any)
	getComponent := func() ui.Node {
		getLabel, parseSetLabel := runtime.GoUseState(getRuntime, "alpha")
		storeSetLabel = parseSetLabel
		getItems := make([]ui.Node, 0, 4)
		for parseIndex := range 4 {
			getItems = append(getItems, html.Div(
				html.Props{
					Key:   strconv.Itoa(parseIndex),
					Class: "serialized-row",
					Data:  map[string]string{"row-id": strconv.Itoa(parseIndex)},
				},
				html.Text("Row "+strconv.Itoa(parseIndex)+" "+getLabel()),
			))
		}
		return html.Div(html.Props{ID: "serialized-host", Class: "grid"}, getItems...)
	}
	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getComponent)); parseErr != nil {
		t.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()

	// Serialized-mount signature: the mock parse produces createTextNode ops
	// for direct text; the per-node fallback uses setTextContent instead.
	// If this flips, the serialized strategy silently stopped firing.
	sawCreateTextNode := false
	for _, getOp := range getAdapter.GetOperations() {
		if getOp.Type == "setTextContent" {
			t.Fatalf("mount used per-node setTextContent — serialized subtree mount did not fire")
		}
		if getOp.Type == "createTextNode" {
			sawCreateTextNode = true
		}
	}
	if !sawCreateTextNode {
		t.Fatalf("no createTextNode ops recorded — serialized mount did not parse direct text")
	}

	// Structure: host div with 4 rows, attrs and text intact after the parse.
	if len(getRoot.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(getRoot.Children))
	}
	getHost := getRoot.Children[0]
	if getHost.Attrs["id"] != "serialized-host" || getHost.Attrs["class"] != "grid" {
		t.Fatalf("host attrs wrong: %v", getHost.Attrs)
	}
	if len(getHost.Children) != 4 {
		t.Fatalf("host children = %d, want 4", len(getHost.Children))
	}
	for parseIndex, getRow := range getHost.Children {
		if getRow.Tag != "div" || getRow.Attrs["class"] != "serialized-row" {
			t.Fatalf("row %d wrong shape: tag=%q attrs=%v", parseIndex, getRow.Tag, getRow.Attrs)
		}
		if getRow.Attrs["data-row-id"] != strconv.Itoa(parseIndex) {
			t.Fatalf("row %d data-row-id = %q", parseIndex, getRow.Attrs["data-row-id"])
		}
		wantText := "Row " + strconv.Itoa(parseIndex) + " alpha"
		if rowText(getRow) != wantText {
			t.Fatalf("row %d text = %q, want %q", parseIndex, rowText(getRow), wantText)
		}
	}

	// Binding: a state update must land as targeted text updates on the
	// parsed nodes, proving bindSerializedSubtree attached the right DOM.
	getAdapter.ClearOperations()
	storeSetLabel("beta")
	getScheduler.FlushAll()
	for parseIndex, getRow := range getHost.Children {
		wantText := "Row " + strconv.Itoa(parseIndex) + " beta"
		if rowText(getRow) != wantText {
			t.Fatalf("after update, row %d text = %q, want %q", parseIndex, rowText(getRow), wantText)
		}
	}
	for _, getOp := range getAdapter.GetOperations() {
		if getOp.Type == "createElement" {
			t.Fatalf("text-only update recreated elements — serialized binding is wrong")
		}
	}
}

// TestSerializedSiblingRunMountFires pins the sibling-run strategy: a flat
// list whose rows are ONE-host subtrees (below the single-subtree threshold)
// must still mount from a combined fragment parse — signature: direct text
// arrives as parsed text nodes, never per-row setTextContent.
func TestSerializedSiblingRunMountFires(t *testing.T) {
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)

	var storeSetLabel func(any)
	getComponent := func() ui.Node {
		getLabel, parseSetLabel := runtime.GoUseState(getRuntime, "alpha")
		storeSetLabel = parseSetLabel
		getItems := make([]ui.Node, 0, 6)
		for parseIndex := range 6 {
			getItems = append(getItems, html.Div(
				html.Props{Key: strconv.Itoa(parseIndex), Class: "run-row"},
				html.Text("Row "+strconv.Itoa(parseIndex)+" "+getLabel()),
			))
		}
		return html.Div(html.Props{ID: "run-host"}, getItems...)
	}
	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getComponent)); parseErr != nil {
		t.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()

	for _, getOp := range getAdapter.GetOperations() {
		if getOp.Type == "setTextContent" {
			t.Fatalf("flat-list mount used per-node setTextContent — sibling-run serialization did not fire")
		}
	}
	if len(getRoot.Children) != 1 || len(getRoot.Children[0].Children) != 6 {
		t.Fatalf("host shape wrong: %+v", getRoot.Children)
	}
	for parseIndex, getRow := range getRoot.Children[0].Children {
		wantText := "Row " + strconv.Itoa(parseIndex) + " alpha"
		if rowText(getRow) != wantText {
			t.Fatalf("row %d text = %q, want %q", parseIndex, rowText(getRow), wantText)
		}
	}

	// Binding correctness: a text-only update must land on the parsed nodes.
	storeSetLabel("beta")
	getScheduler.FlushAll()
	for parseIndex, getRow := range getRoot.Children[0].Children {
		wantText := "Row " + strconv.Itoa(parseIndex) + " beta"
		if rowText(getRow) != wantText {
			t.Fatalf("after update, row %d text = %q, want %q", parseIndex, rowText(getRow), wantText)
		}
	}
}

// TestSerializedMountHandlesMixedTextChildren pins that deep chains whose
// layers mix a text label with a nested element (the deep-tree benchmark
// shape) serialize as ONE subtree: the SerializedMountRoots counter must
// advance, and content plus later updates must be intact.
func TestSerializedMountHandlesMixedTextChildren(t *testing.T) {
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)

	buildChain := func(parseDepth int, parseVersion string) ui.Node {
		getNode := html.Div(html.Props{Class: "leaf"}, html.Text("Leaf "+parseVersion))
		for parseIndex := 1; parseIndex <= parseDepth; parseIndex++ {
			getNode = html.Div(
				html.Props{Class: "layer"},
				html.Text("Layer "+strconv.Itoa(parseIndex)+" "+parseVersion),
				getNode,
			)
		}
		return getNode
	}

	var storeSetVersion func(any)
	getComponent := func() ui.Node {
		getVersion, parseSetVersion := runtime.GoUseState(getRuntime, "v1")
		storeSetVersion = parseSetVersion
		return html.Div(html.Props{ID: "deep-host"}, buildChain(5, getVersion()))
	}
	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getComponent)); parseErr != nil {
		t.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()

	if getRuntime.SerializedMountRoots() == 0 {
		t.Fatal("mixed text+element chain did not take the serialized mount path")
	}

	getCursor := getRoot.Children[0]
	if getCursor.Attrs["id"] != "deep-host" {
		t.Fatalf("host attrs wrong: %v", getCursor.Attrs)
	}
	getLayer := getCursor.Children[0]
	for parseIndex := 5; parseIndex >= 1; parseIndex-- {
		if len(getLayer.Children) != 2 {
			t.Fatalf("layer %d children = %d, want 2 (text + nested)", parseIndex, len(getLayer.Children))
		}
		wantLabel := "Layer " + strconv.Itoa(parseIndex) + " v1"
		if getLayer.Children[0].TextContent != wantLabel {
			t.Fatalf("layer %d label = %q, want %q", parseIndex, getLayer.Children[0].TextContent, wantLabel)
		}
		getLayer = getLayer.Children[1]
	}
	if rowText(getLayer) != "Leaf v1" {
		t.Fatalf("leaf text = %q", rowText(getLayer))
	}

	// Update correctness through the bound text-node fibers.
	storeSetVersion("v2")
	getScheduler.FlushAll()
	getLayer = getRoot.Children[0].Children[0]
	if getLayer.Children[0].TextContent != "Layer 5 v2" {
		t.Fatalf("after update, outermost label = %q, want %q", getLayer.Children[0].TextContent, "Layer 5 v2")
	}
}

// rowText reads a row's effective text whether it lives in the TextContent
// field (per-node SetTextContent) or in a parsed child text node.
func rowText(parseRow *mockdom.MockDOMNode) string {
	if parseRow.TextContent != "" {
		return parseRow.TextContent
	}
	getText := ""
	for _, getChild := range parseRow.Children {
		getText += getChild.TextContent
	}
	return getText
}
