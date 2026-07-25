//go:build !js || !wasm

package render

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// bailoutRowProps parameterizes one keyed row's inner component. The inner is
// a single top-level component (stable identity) instead of a per-row closure:
// loop-created closures share one component handle whose implementation is the
// last closure registered (the G1 gotcha MapKeyedComponent exists to avoid).
type bailoutRowProps struct {
	id    int
	bumps map[int]func(any)
	rt    *runtime.Runtime
}

func bailoutRowInner(parseProps bailoutRowProps) ui.Node {
	getCount, parseSetCount := runtime.GoUseState(parseProps.rt, 0)
	parseProps.bumps[parseProps.id] = parseSetCount
	return html.Span(html.Props{Text: strconv.Itoa(getCount())})
}

// TestKeyedFastLaneRowsSurviveBailoutCloneThenReorder pins that fiber.key
// survives cloneChildFibers (the subtree-dirty pass-through clone). Fast-lane
// keyed rows carry their key ONLY on the fiber key field (props is nil): a
// deep update inside one row clones the whole keyed row chain, and a clone
// that drops the key makes the next reorder fall into the unkeyed positional
// path — rows lose DOM identity.
func TestKeyedFastLaneRowsSurviveBailoutCloneThenReorder(t *testing.T) {
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)

	storeBumps := map[int]func(any){}
	var storeSetOrder func(any)
	getApp := func() ui.Node {
		getOrder, parseSetOrder := runtime.GoUseState(getRuntime, []int{0, 1, 2, 3, 4})
		storeSetOrder = parseSetOrder
		getRows := make([]ui.Node, 0, len(getOrder()))
		for _, parseID := range getOrder() {
			getRows = append(getRows, html.Li(html.Props{
				Key:      "row-" + strconv.Itoa(parseID),
				DataAttr: html.DataAttribute{Name: "row-id", Value: strconv.Itoa(parseID)},
			}, ui.CreateElement(bailoutRowInner, bailoutRowProps{id: parseID, bumps: storeBumps, rt: getRuntime})))
		}
		return html.Ul(html.Props{}, getRows...)
	}

	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getApp)); parseErr != nil {
		t.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()

	getHost := getRoot.Children[0]
	if len(getHost.Children) != 5 {
		t.Fatalf("mounted rows = %d, want 5", len(getHost.Children))
	}
	if len(storeBumps) != 5 || storeBumps[2] == nil {
		t.Fatalf("inner row components did not all render; registered=%d", len(storeBumps))
	}
	getNodeIDByRow := map[string]int{}
	for _, getRow := range getHost.Children {
		getNodeIDByRow[getRow.Attrs["data-row-id"]] = getRow.ID
	}

	// Deep update inside row 2 only: every ancestor (including the keyed <ul>)
	// takes the subtree-dirty clone path without re-rendering the rows.
	storeBumps[2](1)
	getScheduler.FlushAll()

	// Now reorder. The keyed reconciler matches the new elements against the
	// fiber generation produced by the bailout clone above.
	storeSetOrder([]int{4, 3, 2, 1, 0})
	getScheduler.FlushAll()

	if len(getHost.Children) != 5 {
		t.Fatalf("rows after reorder = %d, want 5", len(getHost.Children))
	}
	for parseIndex, parseWantID := range []string{"4", "3", "2", "1", "0"} {
		getRow := getHost.Children[parseIndex]
		if getRow.Attrs["data-row-id"] != parseWantID {
			t.Fatalf("row %d has data-row-id %q, want %q", parseIndex, getRow.Attrs["data-row-id"], parseWantID)
		}
		if getRow.ID != getNodeIDByRow[parseWantID] {
			t.Fatalf("row %q lost DOM identity after bailout clone + reorder (node %d, want %d)",
				parseWantID, getRow.ID, getNodeIDByRow[parseWantID])
		}
	}
}
