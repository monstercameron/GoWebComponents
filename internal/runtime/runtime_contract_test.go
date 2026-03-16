package runtime

import (
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

func (a *queryTestDOMAdapter) QuerySelector(selector string) interface{} {
	return a.selectorResults[selector]
}

func resetGlobalRuntimeForTest() {
	globalRuntime = nil
}

func TestInitGlobalRuntime_CanUpgradeLazyGlobalRuntime(t *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	lazy := GetGlobalRuntime()
	if lazy == nil {
		t.Fatal("expected lazy global runtime to be created")
	}
	if lazy.atomRegistry == nil {
		t.Fatal("expected lazy global runtime to have atom registry")
	}

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	rt := GetGlobalRuntime()
	if rt.domAdapter != adapter {
		t.Fatal("expected InitGlobalRuntime to install DOM adapter on lazy-created global runtime")
	}
	if rt.scheduler != scheduler {
		t.Fatal("expected InitGlobalRuntime to install scheduler on lazy-created global runtime")
	}
}

func TestRenderTo_UsesQuerySelectorAndSchedulesRender(t *testing.T) {
	adapter := newQueryTestDOMAdapter()
	scheduler := newTestScheduler()
	container := adapter.CreateElement("div")
	adapter.selectorResults["#app"] = container
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	element := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	rt.RenderTo("#app", element)

	if rt.wipRoot == nil {
		t.Fatal("expected RenderTo to delegate to Render and create wip root")
	}
	if rt.wipRoot.dom != container {
		t.Fatal("expected RenderTo to use queried container")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected RenderTo to schedule one timeout, got %d", len(scheduler.timeouts))
	}
}

func TestRenderTo_PanicsWhenSelectorMissing(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newQueryTestDOMAdapter(), Scheduler: newTestScheduler()})

	defer func() {
		if recover() == nil {
			t.Fatal("expected RenderTo to panic when selector is missing")
		}
	}()

	rt.RenderTo("#missing", &Element{Type: "div", Props: map[string]interface{}{}})
}

func TestHydrateTo_UsesQuerySelectorAndSchedulesHydration(t *testing.T) {
	adapter := newQueryTestDOMAdapter()
	scheduler := newTestScheduler()
	container := adapter.CreateElement("div")
	adapter.selectorResults["#app"] = container
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	element := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	rt.HydrateTo("#app", element)

	if rt.wipRoot == nil {
		t.Fatal("expected HydrateTo to delegate to Hydrate and create wip root")
	}
	if rt.wipRoot.dom != container {
		t.Fatal("expected HydrateTo to use queried container")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected HydrateTo to schedule one timeout, got %d", len(scheduler.timeouts))
	}
}

func TestHydrateTo_PanicsWhenSelectorMissing(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newQueryTestDOMAdapter(), Scheduler: newTestScheduler()})

	defer func() {
		if recover() == nil {
			t.Fatal("expected HydrateTo to panic when selector is missing")
		}
	}()

	rt.HydrateTo("#missing", &Element{Type: "div", Props: map[string]interface{}{}})
}
