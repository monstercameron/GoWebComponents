package runtime

import (
	"testing"
)

func TestHydrateClearsExistingContainerChildrenBeforeFallbackRender(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	firstSSRChild := adapter.CreateElement("span")
	secondSSRChild := adapter.CreateElement("p")
	adapter.AppendChild(container, firstSSRChild)
	adapter.AppendChild(container, secondSSRChild)

	rt.Hydrate(&Element{Type: "div", Props: map[string]interface{}{"id": "client"}}, container)

	children := adapter.GetChildren(container)
	if len(children) != 0 {
		t.Fatalf("expected hydration fallback to clear existing children before fresh render scheduling, got %d", len(children))
	}
	if rt.wipRoot == nil || rt.wipRoot.dom != container {
		t.Fatal("expected hydration fallback to prepare a work-in-progress root for the container")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected hydration fallback to schedule one timeout, got %d", len(scheduler.timeouts))
	}

	containerNode := container.(*testDOMNode)
	if len(containerNode.children) != 0 {
		t.Fatalf("expected hydration fallback to clear the container children before re-rendering, got %d", len(containerNode.children))
	}
}
