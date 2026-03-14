package runtime

import (
	"strconv"
	"testing"
)

func benchmarkBuildHostChain(adapter DOMAdapter, count int) (*Fiber, []interface{}) {
	var first *Fiber
	var prev *Fiber
	elements := make([]interface{}, count)

	for i := 0; i < count; i++ {
		props := map[string]interface{}{"id": strconv.Itoa(i)}
		elem := &Element{Type: "div", Props: props}
		elements[i] = elem

		fiber := &Fiber{
			typeOf: "div",
			props:  props,
			dom:    adapter.CreateElement("div"),
		}
		if first == nil {
			first = fiber
		} else {
			prev.sibling = fiber
		}
		prev = fiber
	}

	return first, elements
}

func BenchmarkCreateElementHostWithTextChildren(b *testing.B) {
	props := map[string]interface{}{"id": "root", "className": "card", "role": "button"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CreateElement("button", props, "alpha", "beta", "gamma")
	}
}

func BenchmarkPropsEqualChildrenSamePointer(b *testing.B) {
	children := []interface{}{&Element{Type: "span"}}
	props := map[string]interface{}{
		"id":       "same",
		"children": children,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !propsEqual(props, props) {
			b.Fatal("expected props to compare equal")
		}
	}
}

func BenchmarkPropsEqualChildrenDifferentPointer(b *testing.B) {
	propsA := map[string]interface{}{
		"id":       "same",
		"children": []interface{}{&Element{Type: "span"}},
	}
	propsB := map[string]interface{}{
		"id":       "same",
		"children": []interface{}{&Element{Type: "span"}},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if propsEqual(propsA, propsB) {
			b.Fatal("expected props to compare different")
		}
	}
}

func BenchmarkReconcileChildrenStableList16(b *testing.B) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})
	oldFirst, elements := benchmarkBuildHostChain(adapter, 16)
	parent := &Fiber{alternate: &Fiber{child: oldFirst}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parent.child = nil
		rt.deletions = rt.deletions[:0]
		rt.reconcileChildren(parent, elements)
	}
}

func BenchmarkReconcileChildrenWithFragments16(b *testing.B) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})
	oldFirst, flat := benchmarkBuildHostChain(adapter, 16)
	parent := &Fiber{alternate: &Fiber{child: oldFirst}}
	elements := []interface{}{
		&Element{
			Type: "FRAGMENT",
			Props: map[string]interface{}{
				"children": flat[:8],
			},
		},
		&Element{
			Type: "FRAGMENT",
			Props: map[string]interface{}{
				"children": flat[8:],
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parent.child = nil
		rt.deletions = rt.deletions[:0]
		rt.reconcileChildren(parent, elements)
	}
}

func BenchmarkPerformUnitOfWorkFunctionComponentLeaf(b *testing.B) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})
	props := map[string]interface{}{"id": "leaf"}
	rendered := &Element{Type: "div", Props: props}
	component := func(map[string]interface{}) *Element { return rendered }
	oldChild := &Fiber{typeOf: "div", props: props, dom: adapter.CreateElement("div")}
	fiber := &Fiber{
		typeOf:    component,
		props:     map[string]interface{}{},
		dirty:     true,
		alternate: &Fiber{child: oldChild},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fiber.dirty = true
		fiber.child = nil
		rt.deletions = rt.deletions[:0]
		_ = rt.performUnitOfWork(fiber)
	}
}

func BenchmarkUpdateDomPropertiesInitialRender(b *testing.B) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	newProps := map[string]interface{}{
		"id":        "field",
		"className": "large",
		"value":     "abc",
		"checked":   true,
		"style": map[string]string{
			"color":      "red",
			"background": "white",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dom := rt.domAdapter.CreateElement("input")
		rt.updateDomProperties(dom, nil, newProps)
	}
}

func BenchmarkCommitDeletionDomlessSubtree32(b *testing.B) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parentDOM := adapter.CreateElement("div")
		root := &Fiber{typeOf: func(map[string]interface{}) *Element { return nil }}
		var prev *Fiber
		for j := 0; j < 32; j++ {
			child := &Fiber{
				typeOf: "div",
				dom:    adapter.CreateElement("div"),
				parent: root,
			}
			adapter.AppendChild(parentDOM, child.dom)
			if root.child == nil {
				root.child = child
			} else {
				prev.sibling = child
			}
			prev = child
		}
		rt.commitDeletion(root, parentDOM)
	}
}
