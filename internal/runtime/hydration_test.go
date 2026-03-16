package runtime

import (
	"fmt"
	"strings"
	"testing"
)

func runHydrationWork(t *testing.T, scheduler *testScheduler) {
	t.Helper()
	if len(scheduler.timeouts) == 0 {
		t.Fatal("expected scheduled hydration work")
	}
	timeout := scheduler.timeouts[0]
	scheduler.timeouts = scheduler.timeouts[1:]
	timeout()
}

func TestHydrateReusesExistingDOMForSimpleTree(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("section")
	adapter.SetAttribute(serverNode, "id", "hero")
	serverText := adapter.CreateTextNode("Hello")
	adapter.AppendChild(serverNode, serverText)
	adapter.AppendChild(container, serverNode)

	rt.Hydrate(CreateElement("section", map[string]interface{}{"id": "hero"}, "Hello"), container)
	runHydrationWork(t, scheduler)

	children := adapter.GetChildren(container)
	if len(children) != 1 {
		t.Fatalf("expected one hydrated child, got %d", len(children))
	}
	if !children[0].Equals(serverNode) {
		t.Fatal("expected hydration to reuse existing host node")
	}
	textChildren := adapter.GetChildren(children[0])
	if len(textChildren) != 1 || !textChildren[0].Equals(serverText) {
		t.Fatal("expected hydration to reuse existing text node")
	}
}

func TestHydrateFallsBackForTagMismatch(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("span")
	adapter.AppendChild(container, serverNode)

	rt.Hydrate(CreateElement("div", map[string]interface{}{"id": "client"}), container)
	runHydrationWork(t, scheduler)

	children := adapter.GetChildren(container)
	if len(children) != 1 {
		t.Fatalf("expected one client-rendered child after fallback, got %d", len(children))
	}
	if children[0].Equals(serverNode) {
		t.Fatal("expected mismatched server node to be discarded")
	}
	diagnostics := GetDiagnostics()
	if len(diagnostics) == 0 || !strings.Contains(diagnostics[len(diagnostics)-1].Message, "fell back to client rendering") {
		t.Fatalf("expected hydration fallback diagnostic, got %+v", diagnostics)
	}
}

func TestHydrateReportsTextMismatchAndUpdatesNode(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("p")
	serverText := adapter.CreateTextNode("Server")
	adapter.AppendChild(serverNode, serverText)
	adapter.AppendChild(container, serverNode)

	rt.Hydrate(CreateElement("p", nil, "Client"), container)
	runHydrationWork(t, scheduler)

	if got := serverText.(*testDOMNode).text; got != "Client" {
		t.Fatalf("expected hydrated text node to be updated, got %q", got)
	}
	diagnostics := GetDiagnostics()
	found := false
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "hydration text mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected text mismatch diagnostic, got %+v", diagnostics)
	}
}

func TestHydrateDiscardsTrailingUnexpectedNodes(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	first := adapter.CreateElement("li")
	second := adapter.CreateElement("li")
	adapter.AppendChild(container, first)
	adapter.AppendChild(container, second)

	rt.Hydrate(CreateElement("li", map[string]interface{}{"id": "only"}), container)
	runHydrationWork(t, scheduler)

	children := adapter.GetChildren(container)
	if len(children) != 1 {
		t.Fatalf("expected trailing server node to be removed, got %d children", len(children))
	}
	if !children[0].Equals(first) {
		t.Fatal("expected first matching node to be preserved")
	}
}

func TestHydrateSupportsComponentUpdatesAfterResume(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverButton := adapter.CreateElement("button")
	adapter.SetAttribute(serverButton, "id", "counter")
	serverText := adapter.CreateTextNode("count:0")
	adapter.AppendChild(serverButton, serverText)
	adapter.AppendChild(container, serverButton)

	var setCount func(interface{})
	counter := func() *Element {
		count, set := GoUseState(rt, 0)
		setCount = set
		return CreateElement("button", map[string]interface{}{"id": "counter"}, fmt.Sprintf("count:%d", count()))
	}

	rt.Hydrate(CreateElement(counter, nil), container)
	runHydrationWork(t, scheduler)

	if setCount == nil {
		t.Fatal("expected hydrated component to expose state setter")
	}
	setCount(1)
	if len(scheduler.timeouts) == 0 {
		t.Fatal("expected state update to schedule follow-up render")
	}
	runHydrationWork(t, scheduler)

	children := adapter.GetChildren(container)
	if len(children) != 1 || !children[0].Equals(serverButton) {
		t.Fatal("expected hydrated update to keep existing host node")
	}
	textChildren := adapter.GetChildren(serverButton)
	if len(textChildren) != 1 {
		t.Fatalf("expected one button text child, got %d", len(textChildren))
	}
	if got := textChildren[0].(*testDOMNode).text; got != "count:1" {
		t.Fatalf("expected hydrated update to change text to count:1, got %q", got)
	}
}

func TestHydrateDefersAtomSubscriptionsUntilCommitCompletes(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("p")
	serverText := adapter.CreateTextNode("light")
	adapter.AppendChild(serverNode, serverText)
	adapter.AppendChild(container, serverNode)

	reader := func() *Element {
		theme, _ := GoUseAtom(rt, "theme", "light")
		return CreateElement("p", nil, theme())
	}

	rt.Hydrate(CreateElement(reader, nil), container)
	if count := rt.atomRegistry.GetSubscriberCount("theme"); count != 0 {
		t.Fatalf("expected no atom subscribers before hydration commit, got %d", count)
	}
	runHydrationWork(t, scheduler)
	if count := rt.atomRegistry.GetSubscriberCount("theme"); count != 1 {
		t.Fatalf("expected atom subscription after hydration commit, got %d", count)
	}
}

func TestHydrateEffectStateUpdatesScheduleAfterCommit(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("p")
	serverText := adapter.CreateTextNode("count:0")
	adapter.AppendChild(serverNode, serverText)
	adapter.AppendChild(container, serverNode)

	component := func() *Element {
		count, setCount := GoUseState(rt, 0)
		GoUseEffect(func() func() {
			if count() == 0 {
				setCount(1)
			}
			return nil
		}, count())
		return CreateElement("p", nil, fmt.Sprintf("count:%d", count()))
	}

	rt.Hydrate(CreateElement(component, nil), container)
	runHydrationWork(t, scheduler)
	if len(scheduler.timeouts) == 0 {
		t.Fatal("expected effect-triggered state update to schedule follow-up work after hydration commit")
	}
	runHydrationWork(t, scheduler)

	children := adapter.GetChildren(container)
	if len(children) != 1 {
		t.Fatalf("expected one child after post-hydration effect update, got %d", len(children))
	}
	textChildren := adapter.GetChildren(children[0])
	if len(textChildren) != 1 {
		t.Fatalf("expected one text child after post-hydration effect update, got %d", len(textChildren))
	}
	if got := textChildren[0].(*testDOMNode).text; got != "count:1" {
		t.Fatalf("expected effect-driven post-hydration update to change text to count:1, got %q", got)
	}
}
