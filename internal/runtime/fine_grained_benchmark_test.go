package runtime

import (
	"strconv"
	"testing"
)

type benchmarkDashboardModel struct {
	Hot    int
	Panels []string
}

func drainBenchmarkScheduler(scheduler *testScheduler) {
	for len(scheduler.timeouts) > 0 {
		callback := scheduler.timeouts[0]
		scheduler.timeouts = scheduler.timeouts[1:]
		callback()
	}
}

func benchmarkKeyedDashboardComponentUpdate(panelCount int) (*Runtime, *testScheduler) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	hotIndex := panelCount / 2
	hotAtomID := "dashboard-hot-count"
	rt.atomRegistry.InitAtom(hotAtomID, -1)

	app := func() *Element {
		count, _ := GoUseAtom(rt, hotAtomID, -1)
		children := make([]interface{}, 0, panelCount)
		for i := 0; i < panelCount; i++ {
			label := "panel-" + strconv.Itoa(i)
			if i == hotIndex {
				label = strconv.Itoa(count())
			}
			children = append(children, CreateElement("li", map[string]interface{}{"key": i}, label))
		}
		return CreateElement("ul", nil, children...)
	}

	rt.Render(CreateElement(app, nil), container)
	drainBenchmarkScheduler(scheduler)
	return rt, scheduler
}

func benchmarkKeyedDashboardReactiveTextUpdate(panelCount int) (*Runtime, *testScheduler) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	hotIndex := panelCount / 2
	hotAtomID := "dashboard-hot-count"
	rt.atomRegistry.InitAtom(hotAtomID, -1)

	app := func() *Element {
		children := make([]interface{}, 0, panelCount)
		for i := 0; i < panelCount; i++ {
			if i == hotIndex {
				children = append(children, CreateElement("li", map[string]interface{}{"key": i},
					CreateElement(ReactiveTextNodeType, map[string]interface{}{
						reactiveTextAtomIDProp: hotAtomID,
						reactiveTextGetterProp: func() string {
							value, _ := rt.atomRegistry.GetAtom(hotAtomID)
							if count, ok := value.(int); ok {
								return strconv.Itoa(count)
							}
							return "?"
						},
					}),
				))
				continue
			}
			children = append(children, CreateElement("li", map[string]interface{}{"key": i}, "panel-"+strconv.Itoa(i)))
		}
		return CreateElement("ul", nil, children...)
	}

	rt.Render(CreateElement(app, nil), container)
	drainBenchmarkScheduler(scheduler)
	return rt, scheduler
}

func BenchmarkFineGrainedKeyedDashboardComponentUpdate16(b *testing.B) {
	rt, scheduler := benchmarkKeyedDashboardComponentUpdate(16)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := rt.SetAtomValue("dashboard-hot-count", i); err != nil {
			b.Fatalf("unexpected atom update error: %v", err)
		}
		drainBenchmarkScheduler(scheduler)
	}
}

func BenchmarkFineGrainedKeyedDashboardReactiveTextUpdate16(b *testing.B) {
	rt, scheduler := benchmarkKeyedDashboardReactiveTextUpdate(16)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := rt.SetAtomValue("dashboard-hot-count", i); err != nil {
			b.Fatalf("unexpected atom update error: %v", err)
		}
		drainBenchmarkScheduler(scheduler)
	}
}

func benchmarkSelectorDashboardComponentUpdate(panelCount int) (*Runtime, *testScheduler) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	panels := make([]string, panelCount)
	for i := 0; i < panelCount; i++ {
		panels[i] = "panel-" + strconv.Itoa(i)
	}
	model := benchmarkDashboardModel{Hot: -1, Panels: panels}
	rt.atomRegistry.InitAtom("dashboard-model", model)

	app := func() *Element {
		currentModel, _ := GoUseAtom(rt, "dashboard-model", model)
		value := currentModel()
		children := make([]interface{}, 0, len(value.Panels))
		for i, panel := range value.Panels {
			label := panel
			if i == len(value.Panels)/2 {
				label = strconv.Itoa(value.Hot)
			}
			children = append(children, CreateElement("li", map[string]interface{}{"key": i}, label))
		}
		return CreateElement("ul", nil, children...)
	}

	rt.Render(CreateElement(app, nil), container)
	drainBenchmarkScheduler(scheduler)
	return rt, scheduler
}

func benchmarkSelectorDashboardReactiveTextUpdate(panelCount int) (*Runtime, *testScheduler) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	panels := make([]string, panelCount)
	for i := 0; i < panelCount; i++ {
		panels[i] = "panel-" + strconv.Itoa(i)
	}
	model := benchmarkDashboardModel{Hot: -1, Panels: panels}
	rt.atomRegistry.InitAtom("dashboard-model", model)
	if err := rt.RegisterDerivedAtom("dashboard-model-hot", []string{"dashboard-model"}, func() interface{} {
		value, _ := rt.GetAtomValue("dashboard-model")
		return value.(benchmarkDashboardModel).Hot
	}); err != nil {
		panic(err)
	}

	app := func() *Element {
		children := make([]interface{}, 0, len(model.Panels))
		for i, panel := range model.Panels {
			if i == len(model.Panels)/2 {
				children = append(children, CreateElement("li", map[string]interface{}{"key": i},
					CreateElement(ReactiveTextNodeType, map[string]interface{}{
						reactiveTextAtomIDProp: "dashboard-model-hot",
						reactiveTextGetterProp: func() string {
							value, _ := rt.GetAtomValue("dashboard-model-hot")
							if hot, ok := value.(int); ok {
								return strconv.Itoa(hot)
							}
							return "?"
						},
					}),
				))
				continue
			}
			children = append(children, CreateElement("li", map[string]interface{}{"key": i}, panel))
		}
		return CreateElement("ul", nil, children...)
	}

	rt.Render(CreateElement(app, nil), container)
	drainBenchmarkScheduler(scheduler)
	return rt, scheduler
}

func BenchmarkFineGrainedSelectorDashboardComponentUpdate16(b *testing.B) {
	rt, scheduler := benchmarkSelectorDashboardComponentUpdate(16)
	panelsValue, _ := rt.GetAtomValue("dashboard-model")
	base := panelsValue.(benchmarkDashboardModel)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		next := base
		next.Hot = i
		if err := rt.SetAtomValue("dashboard-model", next); err != nil {
			b.Fatalf("unexpected model atom update error: %v", err)
		}
		drainBenchmarkScheduler(scheduler)
	}
}

func BenchmarkFineGrainedSelectorDashboardReactiveTextUpdate16(b *testing.B) {
	rt, scheduler := benchmarkSelectorDashboardReactiveTextUpdate(16)
	panelsValue, _ := rt.GetAtomValue("dashboard-model")
	base := panelsValue.(benchmarkDashboardModel)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		next := base
		next.Hot = i
		if err := rt.SetAtomValue("dashboard-model", next); err != nil {
			b.Fatalf("unexpected model atom update error: %v", err)
		}
		drainBenchmarkScheduler(scheduler)
	}
}

func benchmarkAncestorRerenderStaticLeaves(regionCount int) (func(int), *testScheduler) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")

	leaves := make([]interface{}, 0, regionCount)
	for i := 0; i < regionCount; i++ {
		leaves = append(leaves, CreateElement("div", map[string]interface{}{"key": i},
			CreateElement("span", map[string]interface{}{"data-panel": strconv.Itoa(i)}, "stable"),
		))
	}

	var setTick func(interface{})
	app := func() *Element {
		tick, set := GoUseState(rt, 0)
		setTick = set
		children := make([]interface{}, 0, regionCount+1)
		children = append(children, CreateElement("h1", nil, strconv.Itoa(tick())))
		children = append(children, leaves...)
		return CreateElement("section", nil, children...)
	}

	rt.Render(CreateElement(app, nil), container)
	drainBenchmarkScheduler(scheduler)

	return func(next int) {
		setTick(next)
	}, scheduler
}

func benchmarkAncestorRerenderReactiveRegions(regionCount int) (func(int), *testScheduler) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")

	regions := make([]interface{}, 0, regionCount)
	for i := 0; i < regionCount; i++ {
		atomID := "ancestor-region-" + strconv.Itoa(i)
		regionIndex := i
		rt.atomRegistry.InitAtom(atomID, regionIndex)
		regions = append(regions, CreateElement("div", map[string]interface{}{"key": i},
			CreateElement(ReactiveRegionNodeType, map[string]interface{}{
				reactiveRegionSourceIDsProp: []string{atomID},
				reactiveRegionRenderProp: func() *Element {
					value, _ := rt.GetAtomValue(atomID)
					current, _ := value.(int)
					return CreateElement("input", map[string]interface{}{
						"data-panel": strconv.Itoa(regionIndex),
						"value":      strconv.Itoa(current),
					})
				},
			}),
		))
	}

	var setTick func(interface{})
	app := func() *Element {
		tick, set := GoUseState(rt, 0)
		setTick = set
		children := make([]interface{}, 0, regionCount+1)
		children = append(children, CreateElement("h1", nil, strconv.Itoa(tick())))
		children = append(children, regions...)
		return CreateElement("section", nil, children...)
	}

	rt.Render(CreateElement(app, nil), container)
	drainBenchmarkScheduler(scheduler)

	return func(next int) {
		setTick(next)
	}, scheduler
}

func BenchmarkFineGrainedAncestorRerenderStaticLeaves64(b *testing.B) {
	setTick, scheduler := benchmarkAncestorRerenderStaticLeaves(64)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		setTick(i)
		drainBenchmarkScheduler(scheduler)
	}
}

func BenchmarkFineGrainedAncestorRerenderReactiveRegions64(b *testing.B) {
	setTick, scheduler := benchmarkAncestorRerenderReactiveRegions(64)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		setTick(i)
		drainBenchmarkScheduler(scheduler)
	}
}
