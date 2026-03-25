package runtime

import (
	"strconv"
	"testing"
)

func benchmarkBuildHostChain(parseAdapter DOMAdapter, parseCount int) (*Fiber, []interface{}) {
	var parseFirst *Fiber
	var parsePrev *Fiber
	parseElements := make([]interface{}, parseCount)

	for parseI := 0; parseI < parseCount; parseI++ {
		parseProps := map[string]interface{}{"id": strconv.Itoa(parseI)}
		parseElem := &Element{Type: "div", Props: parseProps}
		parseElements[parseI] = parseElem

		parseFiber := &Fiber{
			typeOf: "div",
			props:  parseProps,
			dom:    parseAdapter.CreateElement("div"),
		}
		if parseFirst == nil {
			parseFirst = parseFiber
		} else {
			parsePrev.sibling = parseFiber
		}
		parsePrev = parseFiber
	}

	return parseFirst, parseElements
}

func benchmarkBuildKeyedHostChain(parseAdapter DOMAdapter, parseCount int) (*Fiber, []interface{}) {
	var parseFirst *Fiber
	var parsePrev *Fiber
	parseElements := make([]interface{}, parseCount)

	for parseI := 0; parseI < parseCount; parseI++ {
		parseProps := map[string]interface{}{"id": strconv.Itoa(parseI), "key": strconv.Itoa(parseI)}
		parseElem := &Element{Type: "div", Props: parseProps}
		parseElements[parseI] = parseElem

		parseFiber := &Fiber{
			typeOf: "div",
			props:  parseProps,
			dom:    parseAdapter.CreateElement("div"),
		}
		if parseFirst == nil {
			parseFirst = parseFiber
		} else {
			parsePrev.sibling = parseFiber
		}
		parsePrev = parseFiber
	}

	return parseFirst, parseElements
}

func BenchmarkCreateElementHostWithTextChildren(parseB *testing.B) {
	parseProps := map[string]interface{}{"id": "root", "className": "card", "role": "button"}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = CreateElement("button", parseProps, "alpha", "beta", "gamma")
	}
}

func BenchmarkPropsEqualChildrenSamePointer(parseB *testing.B) {
	parseChildren := []interface{}{&Element{Type: "span"}}
	parseProps := map[string]interface{}{
		"id":       "same",
		"children": parseChildren,
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if !propsEqual(parseProps, parseProps) {
			parseB.Fatal("expected props to compare equal")
		}
	}
}

func BenchmarkPropsEqualChildrenDifferentPointer(parseB *testing.B) {
	parsePropsA := map[string]interface{}{
		"id":       "same",
		"children": []interface{}{&Element{Type: "span"}},
	}
	parsePropsB := map[string]interface{}{
		"id":       "same",
		"children": []interface{}{&Element{Type: "span"}},
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if propsEqual(parsePropsA, parsePropsB) {
			parseB.Fatal("expected props to compare different")
		}
	}
}

func BenchmarkReconcileChildrenStableList16(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseOldFirst, parseElements := benchmarkBuildHostChain(parseAdapter, 16)
	parseParent := &Fiber{alternate: &Fiber{child: parseOldFirst}}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseParent.child = nil
		parseRt.deletions = parseRt.deletions[:0]
		parseRt.reconcileChildren(parseParent, parseElements)
	}
}

func BenchmarkReconcileChildrenWithFragments16(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseOldFirst, parseFlat := benchmarkBuildHostChain(parseAdapter, 16)
	parseParent := &Fiber{alternate: &Fiber{child: parseOldFirst}}
	parseElements := []interface{}{
		&Element{
			Type: "FRAGMENT",
			Props: map[string]interface{}{
				"children": parseFlat[:8],
			},
		},
		&Element{
			Type: "FRAGMENT",
			Props: map[string]interface{}{
				"children": parseFlat[8:],
			},
		},
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseParent.child = nil
		parseRt.deletions = parseRt.deletions[:0]
		parseRt.reconcileChildren(parseParent, parseElements)
	}
}

func BenchmarkReconcileChildrenKeyedStableList16(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseOldFirst, parseElements := benchmarkBuildKeyedHostChain(parseAdapter, 16)
	parseParent := &Fiber{alternate: &Fiber{child: parseOldFirst}}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseParent.child = nil
		parseRt.deletions = parseRt.deletions[:0]
		parseRt.reconcileChildren(parseParent, parseElements)
	}
}

func BenchmarkPerformUnitOfWorkFunctionComponentLeaf(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseProps := map[string]interface{}{"id": "leaf"}
	parseRendered := &Element{Type: "div", Props: parseProps}
	parseComponent := func(map[string]interface{}) *Element { return parseRendered }
	parseOldChild := &Fiber{typeOf: "div", props: parseProps, dom: parseAdapter.CreateElement("div")}
	parseFiber := &Fiber{
		typeOf:    parseComponent,
		props:     map[string]interface{}{},
		dirty:     true,
		alternate: &Fiber{child: parseOldChild},
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseFiber.dirty = true
		parseFiber.child = nil
		parseRt.deletions = parseRt.deletions[:0]
		_ = parseRt.performUnitOfWork(parseFiber)
	}
}

func BenchmarkPerformUnitOfWorkFunctionComponentNoPropsLeaf(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseProps := map[string]interface{}{"id": "leaf"}
	parseRendered := &Element{Type: "div", Props: parseProps}
	parseComponent := func() *Element { return parseRendered }
	parseOldChild := &Fiber{typeOf: "div", props: parseProps, dom: parseAdapter.CreateElement("div")}
	parseFiber := &Fiber{
		typeOf:    parseComponent,
		props:     map[string]interface{}{},
		dirty:     true,
		alternate: &Fiber{child: parseOldChild},
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseFiber.dirty = true
		parseFiber.child = nil
		parseRt.deletions = parseRt.deletions[:0]
		_ = parseRt.performUnitOfWork(parseFiber)
	}
}

func BenchmarkUpdateDomPropertiesInitialRender(parseB *testing.B) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseNewProps := map[string]interface{}{
		"id":        "field",
		"className": "large",
		"value":     "abc",
		"checked":   true,
		"style": map[string]string{
			"color":      "red",
			"background": "white",
		},
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseDom := parseRt.domAdapter.CreateElement("input")
		parseRt.updateDomProperties(parseDom, nil, parseNewProps)
	}
}

func BenchmarkUpdateDomPropertiesSteadyState(parseB *testing.B) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("input")
	parseOldProps := map[string]interface{}{
		"id":        "field",
		"className": "large",
		"value":     "abc",
		"checked":   false,
		"style": map[string]string{
			"color":      "red",
			"background": "white",
		},
	}
	parseNewProps := map[string]interface{}{
		"id":        "field-next",
		"className": "large active",
		"value":     "abcd",
		"checked":   true,
		"style": map[string]string{
			"color":      "blue",
			"background": "white",
		},
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.updateDomProperties(parseDom, parseOldProps, parseNewProps)
	}
}

func BenchmarkCommitDeletionDomlessSubtree32(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseParentDOM := parseAdapter.CreateElement("div")
		parseRoot := &Fiber{typeOf: func(map[string]interface{}) *Element { return nil }}
		var parsePrev *Fiber
		for parseJ := 0; parseJ < 32; parseJ++ {
			parseChild := &Fiber{
				typeOf: "div",
				dom:    parseAdapter.CreateElement("div"),
				parent: parseRoot,
			}
			parseAdapter.AppendChild(parseParentDOM, parseChild.dom)
			if parseRoot.child == nil {
				parseRoot.child = parseChild
			} else {
				parsePrev.sibling = parseChild
			}
			parsePrev = parseChild
		}
		parseRt.commitDeletion(parseRoot, parseParentDOM)
	}
}

func BenchmarkCommitWorkPlacementChain16(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseParentDOM := parseAdapter.CreateElement("div")
	parseParent := &Fiber{typeOf: "root", dom: parseParentDOM}

	var parseFirst *Fiber
	var parsePrev *Fiber
	for parseI := 0; parseI < 16; parseI++ {
		parseFiber := &Fiber{
			typeOf:    "div",
			dom:       parseAdapter.CreateElement("div"),
			parent:    parseParent,
			effectTag: "PLACEMENT",
		}
		if parseFirst == nil {
			parseFirst = parseFiber
		} else {
			parsePrev.sibling = parseFiber
		}
		parsePrev = parseFiber
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI2 := 0; parseI2 < parseB.N; parseI2++ {
		parseParentDOM.(*testDOMNode).children = parseParentDOM.(*testDOMNode).children[:0]
		parseRt.commitWork(parseFirst, parseParentDOM)
	}
}

func BenchmarkRunEffectsChain16(parseB *testing.B) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRoot := &Fiber{}
	parseCurrent := parseRoot
	for parseI := 0; parseI < 16; parseI++ {
		parseChild := &Fiber{
			hooks: &Hooks{cleanups: make([]func(), 1)},
			effects: []Effect{{
				Fn: func() func() { return nil },
			}},
		}
		parseCurrent.child = parseChild
		parseCurrent = parseChild
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI2 := 0; parseI2 < parseB.N; parseI2++ {
		parseRt.runEffects(parseRoot)
	}
}
