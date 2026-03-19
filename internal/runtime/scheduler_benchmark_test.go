package runtime

import "testing"

func BenchmarkScheduleUpdate(b *testing.B) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  map[string]interface{}{"children": []interface{}{}},
			dom:    newTestDOMAdapter().CreateElement("div"),
		},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt.updateScheduled = false
		rt.wipRoot = nil
		scheduler.timeouts = scheduler.timeouts[:0]
		rt.ScheduleUpdate()
	}
}

func BenchmarkScheduleUpdateSteadyState(b *testing.B) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  map[string]interface{}{"children": []interface{}{}},
			dom:    newTestDOMAdapter().CreateElement("div"),
			alternate: &Fiber{
				typeOf: "ROOT",
			},
		},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt.updateScheduled = false
		rt.wipRoot = nil
		scheduler.timeouts = scheduler.timeouts[:0]
		rt.ScheduleUpdate()
	}
}

func BenchmarkScheduleUpdateForFiberDepth32(b *testing.B) {
	scheduler := newTestScheduler()
	rt := &Runtime{
		scheduler: scheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  map[string]interface{}{"children": []interface{}{}},
		},
	}

	root := &Fiber{typeOf: "root"}
	current := root
	for i := 0; i < 31; i++ {
		next := &Fiber{typeOf: "node", parent: current}
		current = next
	}
	leaf := current

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for f := leaf; f != nil; f = f.parent {
			f.dirty = false
			f.needsUpdate = false
		}
		rt.updateScheduled = false
		rt.wipRoot = nil
		scheduler.timeouts = scheduler.timeouts[:0]
		rt.ScheduleUpdateForFiber(leaf)
	}
}

func BenchmarkRender(b *testing.B) {
	scheduler := newTestScheduler()
	adapter := newTestDOMAdapter()
	container := adapter.CreateElement("div")
	currentRoot := &Fiber{
		typeOf: "ROOT",
		dom:    container,
		props:  map[string]interface{}{"children": []interface{}{}},
	}
	rt := &Runtime{
		scheduler:   scheduler,
		currentRoot: currentRoot,
		deletions:   make([]*Fiber, 0, 8),
	}
	element := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt.currentRoot = currentRoot
		currentRoot.alternate = nil
		scheduler.timeouts = scheduler.timeouts[:0]
		rt.Render(element, container)
	}
}

func BenchmarkRenderSteadyState(b *testing.B) {
	scheduler := newTestScheduler()
	adapter := newTestDOMAdapter()
	container := adapter.CreateElement("div")
	currentRoot := &Fiber{
		typeOf: "ROOT",
		dom:    container,
		props:  map[string]interface{}{"children": []interface{}{}},
		alternate: &Fiber{
			typeOf: "ROOT",
		},
	}
	rt := &Runtime{
		scheduler:   scheduler,
		currentRoot: currentRoot,
		deletions:   make([]*Fiber, 0, 8),
	}
	element := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt.currentRoot = currentRoot
		scheduler.timeouts = scheduler.timeouts[:0]
		rt.Render(element, container)
	}
}

func BenchmarkEnqueueUI(b *testing.B) {
	ProcessUIQueue()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		EnqueueUI(func() {})
		ProcessUIQueue()
	}
}

func BenchmarkProcessUIQueueBatch64(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ProcessUIQueue()
		for j := 0; j < 64; j++ {
			EnqueueUI(func() {})
		}
		ProcessUIQueue()
	}
}

func BenchmarkTransitionListRefresh250(b *testing.B) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")

	var refresh func()
	component := func() *Element {
		items, setItems := GoUseState(rt, []string{"seed"})
		refresh = func() {
			rt.StartTransition(func() {
				next := make([]string, 0, 250)
				for i := 0; i < 250; i++ {
					next = append(next, "item")
				}
				setItems(next)
			})
		}

		children := make([]interface{}, 0, len(items()))
		for index, item := range items() {
			children = append(children, CreateElement("li", map[string]interface{}{"key": index}, item))
		}
		return CreateElement("ul", nil, children...)
	}

	rt.Render(CreateElement(component, nil), container)
	for len(scheduler.timeouts) > 0 {
		callback := scheduler.timeouts[0]
		scheduler.timeouts = scheduler.timeouts[1:]
		callback()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		refresh()
		for len(scheduler.timeouts) > 0 {
			callback := scheduler.timeouts[0]
			scheduler.timeouts = scheduler.timeouts[1:]
			callback()
		}
	}
}
