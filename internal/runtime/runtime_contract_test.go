package runtime

import (
	"strings"
	"testing"
)

type queryTestDOMAdapter struct {
	*testDOMAdapter
	selectorResults map[string]interface{}
}

func newQueryTestDOMAdapter() *queryTestDOMAdapter {
	return &queryTestDOMAdapter{
		testDOMAdapter:  newTestDOMAdapter(),
		selectorResults: make(map[string]interface{}),
	}
}

func (parseA *queryTestDOMAdapter) QuerySelector(parseSelector string) interface{} {
	return parseA.selectorResults[parseSelector]
}

func (parseA *queryTestDOMAdapter) ResolveNode(parseValue interface{}) DOMNode {
	if parseNode, parseOk := parseValue.(DOMNode); parseOk {
		return parseNode
	}
	return nil
}

func resetGlobalRuntimeForTest() {
	globalRuntime = nil
}

func TestInitGlobalRuntime_CanUpgradeLazyGlobalRuntime(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseLazy := GetGlobalRuntime()
	if parseLazy == nil {
		parseT.Fatal("expected lazy global runtime to be created")
	}
	if parseLazy.atomRegistry == nil {
		parseT.Fatal("expected lazy global runtime to have atom registry")
	}

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseRt := GetGlobalRuntime()
	if parseRt.domAdapter != parseAdapter {
		parseT.Fatal("expected InitGlobalRuntime to install DOM adapter on lazy-created global runtime")
	}
	if parseRt.scheduler != parseScheduler {
		parseT.Fatal("expected InitGlobalRuntime to install scheduler on lazy-created global runtime")
	}
}

func TestInitGlobalRuntime_PreservesExistingAdaptersOnPartialConfig(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	InitGlobalRuntime(Config{HideRawPanicOutput: true})

	parseRt := GetGlobalRuntime()
	if parseRt.domAdapter != parseAdapter {
		parseT.Fatal("expected partial InitGlobalRuntime to preserve DOM adapter")
	}
	if parseRt.scheduler != parseScheduler {
		parseT.Fatal("expected partial InitGlobalRuntime to preserve scheduler")
	}
}

func TestRender_PanicsWithoutDOMAdapter(parseT *testing.T) {
	parseRt := NewRuntime(Config{})

	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected Render to panic when DOM adapter is missing")
		}
		parseMessage := parseRecovered.(string)
		if !strings.Contains(parseMessage, "Render requires a DOM adapter") {
			parseT.Fatalf("expected actionable DOM adapter panic, got %q", parseMessage)
		}
	}()

	parseRt.Render(&Element{Type: "div", Props: map[string]interface{}{}}, nil)
}

func TestRender_WithoutSchedulerRunsImmediately(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})

	parseRt.Render(&Element{Type: "div", Props: map[string]interface{}{"id": "app"}}, parseContainer)

	if parseRt.currentRoot == nil {
		parseT.Fatal("expected render without scheduler to commit immediately")
	}
	if parseRt.currentRoot.dom != parseContainer {
		parseT.Fatal("expected immediate render to keep the target container as root dom")
	}
	if parseRt.updateScheduled {
		parseT.Fatal("expected immediate render to settle updateScheduled")
	}
}

func TestRenderTo_UsesQuerySelectorAndSchedulesRender(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#app"] = parseContainer
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseElement := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	parseRt.RenderTo("#app", parseElement)

	if parseRt.wipRoot == nil {
		parseT.Fatal("expected RenderTo to delegate to Render and create wip root")
	}
	if parseRt.wipRoot.dom != parseContainer {
		parseT.Fatal("expected RenderTo to use queried container")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected RenderTo to schedule one timeout, got %d", len(parseScheduler.timeouts))
	}
}

func TestRenderTo_PanicsWhenSelectorMissing(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newQueryTestDOMAdapter(), Scheduler: newTestScheduler(), ShowRawPanicOutput: true})

	defer func() {
		if recover() == nil {
			parseT.Fatal("expected RenderTo to panic when selector is missing")
		}
	}()

	parseRt.RenderTo("#missing", &Element{Type: "div", Props: map[string]interface{}{}})
}

func TestHydrateTo_UsesQuerySelectorAndSchedulesHydration(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#app"] = parseContainer
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseElement := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	parseRt.HydrateTo("#app", parseElement)

	if parseRt.wipRoot == nil {
		parseT.Fatal("expected HydrateTo to delegate to Hydrate and create wip root")
	}
	if parseRt.wipRoot.dom != parseContainer {
		parseT.Fatal("expected HydrateTo to use queried container")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected HydrateTo to schedule one timeout, got %d", len(parseScheduler.timeouts))
	}
}

func TestHydrateTo_PanicsWhenSelectorMissing(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newQueryTestDOMAdapter(), Scheduler: newTestScheduler(), ShowRawPanicOutput: true})

	defer func() {
		if recover() == nil {
			parseT.Fatal("expected HydrateTo to panic when selector is missing")
		}
	}()

	parseRt.HydrateTo("#missing", &Element{Type: "div", Props: map[string]interface{}{}})
}

func TestRenderInto_UsesResolvedNodeAndSchedulesRender(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseContainer := parseAdapter.CreateElement("section")
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseElement := &Element{Type: "div", Props: map[string]interface{}{"id": "widget"}}

	if parseErr := parseRt.RenderInto(parseContainer, parseElement); parseErr != nil {
		parseT.Fatalf("expected RenderInto to succeed, got %v", parseErr)
	}
	if parseRt.wipRoot == nil {
		parseT.Fatal("expected RenderInto to create wip root")
	}
	if parseRt.wipRoot.dom != parseContainer {
		parseT.Fatal("expected RenderInto to use resolved container")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected RenderInto to schedule one timeout, got %d", len(parseScheduler.timeouts))
	}
}

func TestHydrateInto_UsesResolvedNodeAndSchedulesHydration(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseContainer := parseAdapter.CreateElement("section")
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseElement := &Element{Type: "div", Props: map[string]interface{}{"id": "widget"}}

	if parseErr := parseRt.HydrateInto(parseContainer, parseElement); parseErr != nil {
		parseT.Fatalf("expected HydrateInto to succeed, got %v", parseErr)
	}
	if parseRt.wipRoot == nil {
		parseT.Fatal("expected HydrateInto to create wip root")
	}
	if parseRt.wipRoot.dom != parseContainer {
		parseT.Fatal("expected HydrateInto to use resolved container")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected HydrateInto to schedule one timeout, got %d", len(parseScheduler.timeouts))
	}
}
