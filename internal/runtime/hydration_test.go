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

func TestHydrateMismatchDiagnosticsIncludePathAndComponentStack(t *testing.T) {
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

	App := func() *Element {
		return CreateElement("p", nil, "Client")
	}

	rt.Hydrate(CreateElement(App, nil), container)
	runHydrationWork(t, scheduler)

	diagnostics := GetDiagnostics()
	for _, diagnostic := range diagnostics {
		if !strings.Contains(diagnostic.Message, "hydration text mismatch") {
			continue
		}
		if diagnostic.Path == "" {
			t.Fatalf("expected hydration diagnostic path, got %+v", diagnostic)
		}
		if len(diagnostic.ComponentStack) == 0 {
			t.Fatalf("expected hydration component stack, got %+v", diagnostic)
		}
		if diagnostic.ComponentStack[len(diagnostic.ComponentStack)-1] != "p" {
			t.Fatalf("expected hydration stack to end at host node, got %+v", diagnostic.ComponentStack)
		}
		return
	}
	t.Fatalf("expected hydration text mismatch diagnostic with context, got %+v", diagnostics)
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

func TestHydrateStrictModePanicsOnTagMismatch(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("span")
	adapter.AppendChild(container, serverNode)

	rt.SetNextHydrationStrict(true)
	rt.Hydrate(CreateElement("div", map[string]interface{}{"id": "client"}), container)
	expectPanic(t, func() {
		runHydrationWork(t, scheduler)
	})

	children := adapter.GetChildren(container)
	if len(children) != 1 || !children[0].Equals(serverNode) {
		t.Fatalf("expected strict hydration to leave original DOM intact, got %+v", children)
	}

	diagnostics := GetDiagnostics()
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "fell back to client rendering") {
			if diagnostic.Severity != DiagnosticError {
				t.Fatalf("expected strict hydration mismatch to be an error, got %+v", diagnostic)
			}
			return
		}
	}
	t.Fatalf("expected strict hydration fallback diagnostic, got %+v", diagnostics)
}

func TestHydrateStrictModePanicsOnTextMismatch(t *testing.T) {
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

	rt.SetNextHydrationStrict(true)
	rt.Hydrate(CreateElement("p", nil, "Client"), container)
	expectPanic(t, func() {
		runHydrationWork(t, scheduler)
	})

	if got := serverText.(*testDOMNode).text; got != "Server" {
		t.Fatalf("expected strict hydration to avoid rewriting mismatched text, got %q", got)
	}

	diagnostics := GetDiagnostics()
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Message, "hydration text mismatch") {
			if diagnostic.Severity != DiagnosticError {
				t.Fatalf("expected strict hydration text mismatch to be an error, got %+v", diagnostic)
			}
			return
		}
	}
	t.Fatalf("expected strict hydration text mismatch diagnostic, got %+v", diagnostics)
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

func TestHydrateReportsObservabilityMetrics(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("span")
	adapter.AppendChild(container, serverNode)

	var observed HydrationMetrics
	rt.SetNextHydrationObserver("req-42", func(metrics HydrationMetrics) {
		observed = metrics
	})

	rt.Hydrate(CreateElement("div", map[string]interface{}{"id": "client"}), container)
	runHydrationWork(t, scheduler)

	if observed.CorrelationID != "req-42" {
		t.Fatalf("expected correlation id to be preserved, got %+v", observed)
	}
	if observed.FallbackCount != 1 {
		t.Fatalf("expected one hydration fallback, got %+v", observed)
	}
	if observed.ExistingDOMNodeCount != 1 {
		t.Fatalf("expected existing DOM count to be recorded, got %+v", observed)
	}
	if observed.DiscardedNodeCount != 1 {
		t.Fatalf("expected discarded node count to be recorded, got %+v", observed)
	}
	if observed.DurationNs < 0 {
		t.Fatalf("expected non-negative hydration duration, got %+v", observed)
	}
	if observed.Failed {
		t.Fatalf("expected non-strict hydration to finish without failure, got %+v", observed)
	}
}

func TestHydrateUpdatesClosureComponentChildrenAfterResume(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("p")
	adapter.SetAttribute(serverNode, "id", "value")
	serverText := adapter.CreateTextNode("value:0")
	adapter.AppendChild(serverNode, serverText)
	adapter.AppendChild(container, serverNode)

	var setCount func(interface{})
	component := func() *Element {
		count, set := GoUseState(rt, 0)
		setCount = set
		current := count()
		return CreateElement(func() *Element {
			return CreateElement("p", map[string]interface{}{"id": "value"}, fmt.Sprintf("value:%d", current))
		}, nil)
	}

	rt.Hydrate(CreateElement(component, nil), container)
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
	if len(children) != 1 {
		t.Fatalf("expected one host child after hydrated closure update, got %d", len(children))
	}
	updatedNode := children[0]
	updatedElement, ok := updatedNode.(*testDOMNode)
	if !ok {
		t.Fatal("expected updated hydrated node to be a test DOM node")
	}
	if got := updatedElement.attributes["id"]; got != "value" {
		t.Fatalf("expected hydrated closure update to keep the rendered id, got %q", got)
	}
	textChildren := adapter.GetChildren(updatedNode)
	if len(textChildren) != 1 {
		t.Fatalf("expected one text child after hydrated closure update, got %d", len(textChildren))
	}
	if got := textChildren[0].(*testDOMNode).text; got != "value:1" {
		t.Fatalf("expected hydrated closure update to change text to value:1, got %q", got)
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

func TestHydrateAttachesEventHandlersBeforeEffectsRun(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverButton := adapter.CreateElement("button")
	adapter.SetAttribute(serverButton, "id", "action")
	adapter.AppendChild(serverButton, adapter.CreateTextNode("Run"))
	adapter.AppendChild(container, serverButton)

	handlerVisibleDuringEffect := false
	component := func() *Element {
		GoUseEffect(func() func() {
			handlerVisibleDuringEffect = serverButton.(*testDOMNode).properties["onclick"] != nil
			return nil
		})
		return CreateElement("button", map[string]interface{}{
			"id":      "action",
			"onclick": func() {},
		}, "Run")
	}

	rt.Hydrate(CreateElement(component, nil), container)
	runHydrationWork(t, scheduler)

	if !handlerVisibleDuringEffect {
		t.Fatal("expected hydration to attach event handlers before effects run")
	}
}

func TestHydratePreservesLiveInputValueUntilPostHydrationUpdate(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	container := adapter.CreateElement("div")
	serverInput := adapter.CreateElement("input")
	adapter.SetAttribute(serverInput, "id", "name")
	adapter.SetAttribute(serverInput, "value", "server")
	adapter.SetProperty(serverInput, "value", "draft")
	adapter.AppendChild(container, serverInput)

	var setValue func(interface{})
	component := func() *Element {
		value, set := GoUseState(rt, "server")
		setValue = set
		return CreateElement("input", map[string]interface{}{
			"id":    "name",
			"value": value(),
		})
	}

	rt.Hydrate(CreateElement(component, nil), container)
	runHydrationWork(t, scheduler)

	if setValue == nil {
		t.Fatal("expected hydrated input component to expose state setter")
	}
	if got := serverInput.(*testDOMNode).properties["value"]; got != "draft" {
		t.Fatalf("expected hydration to preserve live input value draft, got %#v", got)
	}

	setValue("client")
	runHydrationWork(t, scheduler)

	if got := serverInput.(*testDOMNode).properties["value"]; got != "client" {
		t.Fatalf("expected post-hydration update to apply controlled value, got %#v", got)
	}
}
