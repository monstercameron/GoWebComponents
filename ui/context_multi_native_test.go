//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// Deepens the ReactNewContext port: multiple independent contexts coexist; a
// single consumer can read several; updating one context updates only the
// consumers of that context.
func TestMultipleIndependentContexts(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	parseDesc1 := runtime.NewContextDescriptor("D1")
	parseDesc2 := runtime.NewContextDescriptor("D2")
	parseProvider1 := runtime.NewContextProviderType(parseDesc1)
	parseProvider2 := runtime.NewContextProviderType(parseDesc2)

	// A consumer that reads both contexts.
	parseBothConsumer := func() *runtime.Element {
		return runtime.CreateElement("span", map[string]any{"data-c": "both"},
			fmt.Sprintf("%v|%v", runtime.GoUseContextValue(parseDesc1), runtime.GoUseContextValue(parseDesc2)))
	}
	// A consumer that reads only context 2.
	parseSecondConsumer := func() *runtime.Element {
		return runtime.CreateElement("span", map[string]any{"data-c": "second"},
			fmt.Sprint(runtime.GoUseContextValue(parseDesc2)))
	}

	textByMarker := func(parseMarker string) string {
		var parseWalk func(parseN *mockdom.MockDOMNode) string
		parseWalk = func(parseN *mockdom.MockDOMNode) string {
			if parseN == nil {
				return ""
			}
			if parseN.Attrs["data-c"] == parseMarker {
				return parseN.TextContent
			}
			for _, parseC := range parseN.Children {
				if parseR := parseWalk(parseC); parseR != "" {
					return parseR
				}
			}
			return ""
		}
		for _, parseK := range adapter.GetChildren(root) {
			if parseN, parseOk := parseK.(*mockdom.MockDOMNode); parseOk {
				if parseR := parseWalk(parseN); parseR != "" {
					return parseR
				}
			}
		}
		return ""
	}

	var parseSet1 func(any)
	parseApp := func() *runtime.Element {
		parseGet1, parseSetter1 := runtime.GoUseState(rt, "A")
		parseSet1 = parseSetter1
		return runtime.CreateElementOwned(parseProvider1, map[string]any{"value": parseGet1()},
			runtime.CreateElementOwned(parseProvider2, map[string]any{"value": "B"},
				runtime.CreateElement("div", map[string]any{},
					runtime.CreateElement(parseBothConsumer, map[string]any{}),
					runtime.CreateElement(parseSecondConsumer, map[string]any{}))))
	}

	rt.RenderInto(root, runtime.CreateElement(parseApp, map[string]any{}))
	if parseGot := textByMarker("both"); parseGot != "A|B" {
		t.Fatalf("both-consumer = %q, want A|B", parseGot)
	}
	if parseGot := textByMarker("second"); parseGot != "B" {
		t.Fatalf("second-consumer = %q, want B", parseGot)
	}

	// Updating context 1 updates the both-consumer; the second-consumer (context 2
	// only) is unaffected by the value change.
	parseSet1("A2")
	if parseGot := textByMarker("both"); parseGot != "A2|B" {
		t.Errorf("after update both-consumer = %q, want A2|B", parseGot)
	}
	if parseGot := textByMarker("second"); parseGot != "B" {
		t.Errorf("after update second-consumer = %q, want B (unchanged)", parseGot)
	}
}

// Same context used at two nesting levels: the inner provider value shadows the
// outer one for the inner consumer, while a consumer between the two providers
// still sees the outer value.
func TestSameContextTwoLevels(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	parseDesc := runtime.NewContextDescriptor("DEFAULT")
	parseProvider := runtime.NewContextProviderType(parseDesc)
	parseConsumer := func() *runtime.Element {
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(runtime.GoUseContextValue(parseDesc)))
	}

	rt.RenderInto(root, runtime.CreateElementOwned(parseProvider, map[string]any{"value": "OUTER"},
		runtime.CreateElement("div", map[string]any{},
			runtime.CreateElement(parseConsumer, map[string]any{}), // sees OUTER
			runtime.CreateElementOwned(parseProvider, map[string]any{"value": "INNER"},
				runtime.CreateElement(parseConsumer, map[string]any{}))))) // sees INNER

	parseOut, _ := concatTreeText(t, adapter, root)
	if parseOut != "OUTERINNER" {
		t.Errorf("two-level same context = %q, want OUTERINNER", parseOut)
	}
}

// concatTreeText concatenates all span text under root in document order.
func concatTreeText(t *testing.T, adapter *mockdom.MockDOMAdapter, root runtime.DOMNode) (string, error) {
	t.Helper()
	var parseOut string
	var parseWalk func(parseN *mockdom.MockDOMNode)
	parseWalk = func(parseN *mockdom.MockDOMNode) {
		if parseN == nil {
			return
		}
		parseOut += parseN.TextContent
		for _, parseC := range parseN.Children {
			parseWalk(parseC)
		}
	}
	for _, parseK := range adapter.GetChildren(root) {
		if parseN, parseOk := parseK.(*mockdom.MockDOMNode); parseOk {
			parseWalk(parseN)
		}
	}
	return parseOut, nil
}
