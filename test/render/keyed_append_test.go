//go:build !js || !wasm

package render

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestKeyedTrailingAppendKeepsPrefixIdentity pins the in-order append fast
// path: growing a keyed list must reuse every existing row's DOM node
// (no remount), append the tail in order, and remove nothing.
func TestKeyedTrailingAppendKeepsPrefixIdentity(t *testing.T) {
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)

	var storeSetRows func(any)
	getComponent := func() ui.Node {
		getRows, parseSetRows := runtime.GoUseState(getRuntime, []mirrorCoreRow{})
		storeSetRows = parseSetRows
		return renderMirrorCoreList(getRows(), 0)
	}
	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getComponent)); parseErr != nil {
		t.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()

	storeSetRows(buildMirrorCoreRows(5, ""))
	getScheduler.FlushAll()
	getHost := getRoot.Children[0]
	if len(getHost.Children) != 5 {
		t.Fatalf("mounted rows = %d, want 5", len(getHost.Children))
	}
	getBeforeIDs := make([]int, 5)
	for parseIndex, getRow := range getHost.Children {
		getBeforeIDs[parseIndex] = getRow.ID
	}

	getAdapter.ClearOperations()
	storeSetRows(buildMirrorCoreRows(8, ""))
	getScheduler.FlushAll()

	if len(getHost.Children) != 8 {
		t.Fatalf("rows after append = %d, want 8", len(getHost.Children))
	}
	for parseIndex := range 5 {
		if getHost.Children[parseIndex].ID != getBeforeIDs[parseIndex] {
			t.Fatalf("row %d was remounted (id %d -> %d)", parseIndex, getBeforeIDs[parseIndex], getHost.Children[parseIndex].ID)
		}
	}
	for parseIndex := 5; parseIndex < 8; parseIndex++ {
		if getHost.Children[parseIndex].Attrs["data-row-id"] != strconv.Itoa(parseIndex) {
			t.Fatalf("appended row %d has data-row-id %q", parseIndex, getHost.Children[parseIndex].Attrs["data-row-id"])
		}
	}
	for _, getOp := range getAdapter.GetOperations() {
		if getOp.Type == "removeChild" || getOp.Type == "replaceChildren" {
			t.Fatalf("append pass issued %s", getOp.Type)
		}
	}
}
