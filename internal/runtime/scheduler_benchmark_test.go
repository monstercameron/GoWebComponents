package runtime

import "testing"

func BenchmarkScheduleUpdate(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  map[string]interface{}{"children": []interface{}{}},
			dom:    newTestDOMAdapter().CreateElement("div"),
		},
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.updateScheduled = false
		parseRt.wipRoot = nil
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		parseRt.ScheduleUpdate()
	}
}

func BenchmarkScheduleUpdateSteadyState(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  map[string]interface{}{"children": []interface{}{}},
			dom:    newTestDOMAdapter().CreateElement("div"),
			alternate: &Fiber{
				typeOf: "ROOT",
			},
		},
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.updateScheduled = false
		parseRt.wipRoot = nil
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		parseRt.ScheduleUpdate()
	}
}

func BenchmarkScheduleUpdateForFiberDepth32(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{
		scheduler: parseScheduler,
		currentRoot: &Fiber{
			typeOf: "ROOT",
			props:  map[string]interface{}{"children": []interface{}{}},
		},
	}

	parseRoot := &Fiber{typeOf: "root"}
	parseCurrent := parseRoot
	for parseI := 0; parseI < 31; parseI++ {
		parseNext := &Fiber{typeOf: "node", parent: parseCurrent}
		parseCurrent = parseNext
	}
	parseLeaf := parseCurrent

	parseB.ReportAllocs()
	for parseI2 := 0; parseI2 < parseB.N; parseI2++ {
		for parseF := parseLeaf; parseF != nil; parseF = parseF.parent {
			parseF.dirty = false
			parseF.needsUpdate = false
		}
		parseRt.updateScheduled = false
		parseRt.wipRoot = nil
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		parseRt.ScheduleUpdateForFiber(parseLeaf)
	}
}

func BenchmarkRender(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseAdapter := newTestDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseCurrentRoot := &Fiber{
		typeOf: "ROOT",
		dom:    parseContainer,
		props:  map[string]interface{}{"children": []interface{}{}},
	}
	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseCurrentRoot,
		deletions:   make([]*Fiber, 0, 8),
	}
	parseElement := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.currentRoot = parseCurrentRoot
		parseCurrentRoot.alternate = nil
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		parseRt.Render(parseElement, parseContainer)
	}
}

func BenchmarkRenderSteadyState(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseAdapter := newTestDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseCurrentRoot := &Fiber{
		typeOf: "ROOT",
		dom:    parseContainer,
		props:  map[string]interface{}{"children": []interface{}{}},
		alternate: &Fiber{
			typeOf: "ROOT",
		},
	}
	parseRt := &Runtime{
		scheduler:   parseScheduler,
		currentRoot: parseCurrentRoot,
		deletions:   make([]*Fiber, 0, 8),
	}
	parseElement := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.currentRoot = parseCurrentRoot
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		parseRt.Render(parseElement, parseContainer)
	}
}

func BenchmarkEnqueueUI(parseB *testing.B) {
	ProcessUIQueue()

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		EnqueueUI(func() {})
		ProcessUIQueue()
	}
}

func BenchmarkProcessUIQueueBatch64(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		ProcessUIQueue()
		for parseJ := 0; parseJ < 64; parseJ++ {
			EnqueueUI(func() {})
		}
		ProcessUIQueue()
	}
}

func BenchmarkTransitionListRefresh250(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")

	var parseRefresh func()
	parseComponent := func() *Element {
		parseItems, setItems := GoUseState(parseRt, []string{"seed"})
		parseRefresh = func() {
			parseRt.StartTransition(func() {
				parseNext := make([]string, 0, 250)
				for parseI := 0; parseI < 250; parseI++ {
					parseNext = append(parseNext, "item")
				}
				setItems(parseNext)
			})
		}

		parseChildren := make([]interface{}, 0, len(parseItems()))
		for parseIndex, parseItem := range parseItems() {
			parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{"key": parseIndex}, parseItem))
		}
		return CreateElement("ul", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseComponent, nil), parseContainer)
	for len(parseScheduler.timeouts) > 0 {
		parseCallback := parseScheduler.timeouts[0]
		parseScheduler.timeouts = parseScheduler.timeouts[1:]
		parseCallback()
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI2 := 0; parseI2 < parseB.N; parseI2++ {
		parseRefresh()
		for len(parseScheduler.timeouts) > 0 {
			parseCallback2 := parseScheduler.timeouts[0]
			parseScheduler.timeouts = parseScheduler.timeouts[1:]
			parseCallback2()
		}
	}
}
