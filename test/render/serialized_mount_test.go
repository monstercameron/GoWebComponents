//go:build !js || !wasm

package render

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
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
