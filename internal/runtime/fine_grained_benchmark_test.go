package runtime

import (
	"strconv"
	"testing"
)

type benchmarkDashboardModel struct {
	Hot    int
	Panels []string
}

func drainBenchmarkScheduler(parseScheduler *testScheduler) {
	for len(parseScheduler.timeouts) > 0 {
		parseCallback := parseScheduler.timeouts[0]
		parseScheduler.timeouts = parseScheduler.timeouts[1:]
		parseCallback()
	}
}

func benchmarkKeyedDashboardComponentUpdate(parsePanelCount int) (*Runtime, *testScheduler) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseHotIndex := parsePanelCount / 2
	parseHotAtomID := "dashboard-hot-count"
	parseRt.atomRegistry.InitAtom(parseHotAtomID, -1)

	parseApp := func() *Element {
		parseCount, _ := GoUseAtom(parseRt, parseHotAtomID, -1)
		parseChildren := make([]interface{}, 0, parsePanelCount)
		for parseI := 0; parseI < parsePanelCount; parseI++ {
			parseLabel := "panel-" + strconv.Itoa(parseI)
			if parseI == parseHotIndex {
				parseLabel = strconv.Itoa(parseCount())
			}
			parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{"key": parseI}, parseLabel))
		}
		return CreateElement("ul", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainBenchmarkScheduler(parseScheduler)
	return parseRt, parseScheduler
}

func benchmarkKeyedDashboardReactiveTextUpdate(parsePanelCount int) (*Runtime, *testScheduler) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseHotIndex := parsePanelCount / 2
	parseHotAtomID := "dashboard-hot-count"
	parseRt.atomRegistry.InitAtom(parseHotAtomID, -1)

	parseApp := func() *Element {
		parseChildren := make([]interface{}, 0, parsePanelCount)
		for parseI := 0; parseI < parsePanelCount; parseI++ {
			if parseI == parseHotIndex {
				parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{"key": parseI},
					CreateElement(ReactiveTextNodeType, map[string]interface{}{
						reactiveTextAtomIDProp: parseHotAtomID,
						reactiveTextGetterProp: func() string {
							parseValue, _ := parseRt.atomRegistry.GetAtom(parseHotAtomID)
							if parseCount, parseOk := parseValue.(int); parseOk {
								return strconv.Itoa(parseCount)
							}
							return "?"
						},
					}),
				))
				continue
			}
			parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{"key": parseI}, "panel-"+strconv.Itoa(parseI)))
		}
		return CreateElement("ul", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainBenchmarkScheduler(parseScheduler)
	return parseRt, parseScheduler
}

func BenchmarkFineGrainedKeyedDashboardComponentUpdate16(parseB *testing.B) {
	parseRt, parseScheduler := benchmarkKeyedDashboardComponentUpdate(16)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if parseErr := parseRt.SetAtomValue("dashboard-hot-count", parseI); parseErr != nil {
			parseB.Fatalf("unexpected atom update error: %v", parseErr)
		}
		drainBenchmarkScheduler(parseScheduler)
	}
}

func BenchmarkFineGrainedKeyedDashboardReactiveTextUpdate16(parseB *testing.B) {
	parseRt, parseScheduler := benchmarkKeyedDashboardReactiveTextUpdate(16)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if parseErr := parseRt.SetAtomValue("dashboard-hot-count", parseI); parseErr != nil {
			parseB.Fatalf("unexpected atom update error: %v", parseErr)
		}
		drainBenchmarkScheduler(parseScheduler)
	}
}

func benchmarkSelectorDashboardComponentUpdate(parsePanelCount int) (*Runtime, *testScheduler) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parsePanels := make([]string, parsePanelCount)
	for parseI := 0; parseI < parsePanelCount; parseI++ {
		parsePanels[parseI] = "panel-" + strconv.Itoa(parseI)
	}
	parseModel := benchmarkDashboardModel{Hot: -1, Panels: parsePanels}
	parseRt.atomRegistry.InitAtom("dashboard-model", parseModel)

	parseApp := func() *Element {
		parseCurrentModel, _ := GoUseAtom(parseRt, "dashboard-model", parseModel)
		parseValue := parseCurrentModel()
		parseChildren := make([]interface{}, 0, len(parseValue.Panels))
		for parseI2, parsePanel := range parseValue.Panels {
			parseLabel := parsePanel
			if parseI2 == len(parseValue.Panels)/2 {
				parseLabel = strconv.Itoa(parseValue.Hot)
			}
			parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{"key": parseI2}, parseLabel))
		}
		return CreateElement("ul", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainBenchmarkScheduler(parseScheduler)
	return parseRt, parseScheduler
}

func benchmarkSelectorDashboardReactiveTextUpdate(parsePanelCount int) (*Runtime, *testScheduler) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parsePanels := make([]string, parsePanelCount)
	for parseI := 0; parseI < parsePanelCount; parseI++ {
		parsePanels[parseI] = "panel-" + strconv.Itoa(parseI)
	}
	parseModel := benchmarkDashboardModel{Hot: -1, Panels: parsePanels}
	parseRt.atomRegistry.InitAtom("dashboard-model", parseModel)
	if parseErr := parseRt.RegisterDerivedAtom("dashboard-model-hot", []string{"dashboard-model"}, func() interface{} {
		parseValue, _ := parseRt.GetAtomValue("dashboard-model")
		return parseValue.(benchmarkDashboardModel).Hot
	}); parseErr != nil {
		panic(parseErr)
	}

	parseApp := func() *Element {
		parseChildren := make([]interface{}, 0, len(parseModel.Panels))
		for parseI2, parsePanel := range parseModel.Panels {
			if parseI2 == len(parseModel.Panels)/2 {
				parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{"key": parseI2},
					CreateElement(ReactiveTextNodeType, map[string]interface{}{
						reactiveTextAtomIDProp: "dashboard-model-hot",
						reactiveTextGetterProp: func() string {
							parseValue2, _ := parseRt.GetAtomValue("dashboard-model-hot")
							if parseHot, parseOk := parseValue2.(int); parseOk {
								return strconv.Itoa(parseHot)
							}
							return "?"
						},
					}),
				))
				continue
			}
			parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{"key": parseI2}, parsePanel))
		}
		return CreateElement("ul", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainBenchmarkScheduler(parseScheduler)
	return parseRt, parseScheduler
}

func benchmarkSignalStyleDashboardReactiveRegions(parsePanelCount int) (*Runtime, *testScheduler) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseHotIndex := parsePanelCount / 2

	parsePanelAtomIDs := make([]string, parsePanelCount)
	for parseI := 0; parseI < parsePanelCount; parseI++ {
		parseAtomID := "signal-style-panel-" + strconv.Itoa(parseI)
		parsePanelAtomIDs[parseI] = parseAtomID
		parseInitial := "panel-" + strconv.Itoa(parseI)
		if parseI == parseHotIndex {
			parseInitial = strconv.Itoa(-1)
		}
		parseRt.atomRegistry.InitAtom(parseAtomID, parseInitial)
	}

	parseApp := func() *Element {
		parseChildren := make([]interface{}, 0, parsePanelCount)
		for parseI2, parseAtomID2 := range parsePanelAtomIDs {
			parsePanelIndex := parseI2
			parsePanelAtomID := parseAtomID2
			parseChildren = append(parseChildren, CreateElement("li", map[string]interface{}{"key": parseI2},
				CreateElement(ReactiveRegionNodeType, map[string]interface{}{
					reactiveRegionSourceIDsProp: []string{parsePanelAtomID},
					reactiveRegionRenderProp: func() *Element {
						parseValue, _ := parseRt.GetAtomValue(parsePanelAtomID)
						parseLabel, _ := parseValue.(string)
						return CreateElement("span", map[string]interface{}{"data-panel": strconv.Itoa(parsePanelIndex)}, parseLabel)
					},
				}),
			))
		}
		return CreateElement("ul", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainBenchmarkScheduler(parseScheduler)
	return parseRt, parseScheduler
}

func BenchmarkFineGrainedSelectorDashboardComponentUpdate16(parseB *testing.B) {
	parseRt, parseScheduler := benchmarkSelectorDashboardComponentUpdate(16)
	parsePanelsValue, _ := parseRt.GetAtomValue("dashboard-model")
	parseBase := parsePanelsValue.(benchmarkDashboardModel)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseNext := parseBase
		parseNext.Hot = parseI
		if parseErr := parseRt.SetAtomValue("dashboard-model", parseNext); parseErr != nil {
			parseB.Fatalf("unexpected model atom update error: %v", parseErr)
		}
		drainBenchmarkScheduler(parseScheduler)
	}
}

func BenchmarkFineGrainedSelectorDashboardReactiveTextUpdate16(parseB *testing.B) {
	parseRt, parseScheduler := benchmarkSelectorDashboardReactiveTextUpdate(16)
	parsePanelsValue, _ := parseRt.GetAtomValue("dashboard-model")
	parseBase := parsePanelsValue.(benchmarkDashboardModel)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseNext := parseBase
		parseNext.Hot = parseI
		if parseErr := parseRt.SetAtomValue("dashboard-model", parseNext); parseErr != nil {
			parseB.Fatalf("unexpected model atom update error: %v", parseErr)
		}
		drainBenchmarkScheduler(parseScheduler)
	}
}

func BenchmarkFineGrainedSignalStyleDashboardReactiveRegions16(parseB *testing.B) {
	parseRt, parseScheduler := benchmarkSignalStyleDashboardReactiveRegions(16)
	parseHotAtomID := "signal-style-panel-" + strconv.Itoa(16/2)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if parseErr := parseRt.SetAtomValue(parseHotAtomID, strconv.Itoa(parseI)); parseErr != nil {
			parseB.Fatalf("unexpected signal-style atom update error: %v", parseErr)
		}
		drainBenchmarkScheduler(parseScheduler)
	}
}

func benchmarkAncestorRerenderStaticLeaves(parseRegionCount int) (func(int), *testScheduler) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")

	parseLeaves := make([]interface{}, 0, parseRegionCount)
	for parseI := 0; parseI < parseRegionCount; parseI++ {
		parseLeaves = append(parseLeaves, CreateElement("div", map[string]interface{}{"key": parseI},
			CreateElement("span", map[string]interface{}{"data-panel": strconv.Itoa(parseI)}, "stable"),
		))
	}

	var setTick func(interface{})
	parseApp := func() *Element {
		parseTick, set := GoUseState(parseRt, 0)
		setTick = set
		parseChildren := make([]interface{}, 0, parseRegionCount+1)
		parseChildren = append(parseChildren, CreateElement("h1", nil, strconv.Itoa(parseTick())))
		parseChildren = append(parseChildren, parseLeaves...)
		return CreateElement("section", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainBenchmarkScheduler(parseScheduler)

	return func(parseNext int) {
		setTick(parseNext)
	}, parseScheduler
}

func benchmarkAncestorRerenderReactiveRegions(parseRegionCount int) (func(int), *testScheduler) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")

	parseRegions := make([]interface{}, 0, parseRegionCount)
	for parseI := 0; parseI < parseRegionCount; parseI++ {
		parseAtomID := "ancestor-region-" + strconv.Itoa(parseI)
		parseRegionIndex := parseI
		parseRt.atomRegistry.InitAtom(parseAtomID, parseRegionIndex)
		parseRegions = append(parseRegions, CreateElement("div", map[string]interface{}{"key": parseI},
			CreateElement(ReactiveRegionNodeType, map[string]interface{}{
				reactiveRegionSourceIDsProp: []string{parseAtomID},
				reactiveRegionRenderProp: func() *Element {
					parseValue, _ := parseRt.GetAtomValue(parseAtomID)
					parseCurrent, _ := parseValue.(int)
					return CreateElement("input", map[string]interface{}{
						"data-panel": strconv.Itoa(parseRegionIndex),
						"value":      strconv.Itoa(parseCurrent),
					})
				},
			}),
		))
	}

	var setTick func(interface{})
	parseApp := func() *Element {
		parseTick, set := GoUseState(parseRt, 0)
		setTick = set
		parseChildren := make([]interface{}, 0, parseRegionCount+1)
		parseChildren = append(parseChildren, CreateElement("h1", nil, strconv.Itoa(parseTick())))
		parseChildren = append(parseChildren, parseRegions...)
		return CreateElement("section", nil, parseChildren...)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	drainBenchmarkScheduler(parseScheduler)

	return func(parseNext int) {
		setTick(parseNext)
	}, parseScheduler
}

func BenchmarkFineGrainedAncestorRerenderStaticLeaves64(parseB *testing.B) {
	setTick, parseScheduler := benchmarkAncestorRerenderStaticLeaves(64)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		setTick(parseI)
		drainBenchmarkScheduler(parseScheduler)
	}
}

func BenchmarkFineGrainedAncestorRerenderReactiveRegions64(parseB *testing.B) {
	setTick, parseScheduler := benchmarkAncestorRerenderReactiveRegions(64)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		setTick(parseI)
		drainBenchmarkScheduler(parseScheduler)
	}
}
