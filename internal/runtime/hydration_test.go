package runtime

import (
	"fmt"
	"strings"
	"testing"
)

func runHydrationWork(parseT *testing.T, parseScheduler *testScheduler) {
	parseT.Helper()
	if len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected scheduled hydration work")
	}
	parseTimeout := parseScheduler.timeouts[0]
	parseScheduler.timeouts = parseScheduler.timeouts[1:]
	parseTimeout()
}

func TestHydrateReusesExistingDOMForSimpleTree(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("section")
	parseAdapter.SetAttribute(parseServerNode, "id", "hero")
	parseServerText := parseAdapter.CreateTextNode("Hello")
	parseAdapter.AppendChild(parseServerNode, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseRt.Hydrate(CreateElement("section", map[string]any{"id": "hero"}, "Hello"), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 {
		parseT.Fatalf("expected one hydrated child, got %d", len(parseChildren))
	}
	if !parseChildren[0].Equals(parseServerNode) {
		parseT.Fatal("expected hydration to reuse existing host node")
	}
	parseTextChildren := parseAdapter.GetChildren(parseChildren[0])
	if len(parseTextChildren) != 1 || !parseTextChildren[0].Equals(parseServerText) {
		parseT.Fatal("expected hydration to reuse existing text node")
	}
}

func TestHydrateFallsBackForTagMismatch(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("span")
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseRt.Hydrate(CreateElement("div", map[string]any{"id": "client"}), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 {
		parseT.Fatalf("expected one client-rendered child after fallback, got %d", len(parseChildren))
	}
	if parseChildren[0].Equals(parseServerNode) {
		parseT.Fatal("expected mismatched server node to be discarded")
	}
	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 || !strings.Contains(parseDiagnostics[len(parseDiagnostics)-1].Message, "fell back to client rendering") {
		parseT.Fatalf("expected hydration fallback diagnostic, got %+v", parseDiagnostics)
	}
}

func TestHydrateReportsTextMismatchAndUpdatesNode(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("p")
	parseServerText := parseAdapter.CreateTextNode("Server")
	parseAdapter.AppendChild(parseServerNode, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseRt.Hydrate(CreateElement("p", nil, "Client"), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	if parseGot := parseServerText.(*testDOMNode).text; parseGot != "Client" {
		parseT.Fatalf("expected hydrated text node to be updated, got %q", parseGot)
	}
	parseDiagnostics := GetDiagnostics()
	isParseFound := false
	for _, parseDiagnostic := range parseDiagnostics {
		if strings.Contains(parseDiagnostic.Message, "hydration text mismatch") {
			isParseFound = true
			break
		}
	}
	if !isParseFound {
		parseT.Fatalf("expected text mismatch diagnostic, got %+v", parseDiagnostics)
	}
}

func TestHydrateMismatchDiagnosticsIncludePathAndComponentStack(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("p")
	parseServerText := parseAdapter.CreateTextNode("Server")
	parseAdapter.AppendChild(parseServerNode, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseApp := func() *Element {
		return CreateElement("p", nil, "Client")
	}

	parseRt.Hydrate(CreateElement(parseApp, nil), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	parseDiagnostics := GetDiagnostics()
	for _, parseDiagnostic := range parseDiagnostics {
		if !strings.Contains(parseDiagnostic.Message, "hydration text mismatch") {
			continue
		}
		if parseDiagnostic.Code != "GWC-HYDRATION-TEXT-MISMATCH" {
			parseT.Fatalf("expected hydration text mismatch code, got %+v", parseDiagnostic)
		}
		if parseDiagnostic.Docs == "" || parseDiagnostic.Remediation == "" || !parseDiagnostic.Recoverable {
			parseT.Fatalf("expected hydration mismatch guidance, got %+v", parseDiagnostic)
		}
		if parseDiagnostic.Path == "" {
			parseT.Fatalf("expected hydration diagnostic path, got %+v", parseDiagnostic)
		}
		if len(parseDiagnostic.ComponentStack) == 0 {
			parseT.Fatalf("expected hydration component stack, got %+v", parseDiagnostic)
		}
		if parseDiagnostic.ComponentStack[len(parseDiagnostic.ComponentStack)-1] != "p" {
			parseT.Fatalf("expected hydration stack to end at host node, got %+v", parseDiagnostic.ComponentStack)
		}
		return
	}
	parseT.Fatalf("expected hydration text mismatch diagnostic with context, got %+v", parseDiagnostics)
}

func TestHydrateDiscardsTrailingUnexpectedNodes(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseFirst := parseAdapter.CreateElement("li")
	parseSecond := parseAdapter.CreateElement("li")
	parseAdapter.AppendChild(parseContainer, parseFirst)
	parseAdapter.AppendChild(parseContainer, parseSecond)

	parseRt.Hydrate(CreateElement("li", map[string]any{"id": "only"}), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 {
		parseT.Fatalf("expected trailing server node to be removed, got %d children", len(parseChildren))
	}
	if !parseChildren[0].Equals(parseFirst) {
		parseT.Fatal("expected first matching node to be preserved")
	}
}

func TestHydrateStrictModePanicsOnTagMismatch(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, ShowRawPanicOutput: true})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("span")
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseRt.SetNextHydrationStrict(true)
	parseRt.Hydrate(CreateElement("div", map[string]any{"id": "client"}), parseContainer)
	expectPanic(parseT, func() {
		runHydrationWork(parseT, parseScheduler)
	})

	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 || !parseChildren[0].Equals(parseServerNode) {
		parseT.Fatalf("expected strict hydration to leave original DOM intact, got %+v", parseChildren)
	}

	parseDiagnostics := GetDiagnostics()
	for _, parseDiagnostic := range parseDiagnostics {
		if strings.Contains(parseDiagnostic.Message, "fell back to client rendering") {
			if parseDiagnostic.Severity != DiagnosticError {
				parseT.Fatalf("expected strict hydration mismatch to be an error, got %+v", parseDiagnostic)
			}
			return
		}
	}
	parseT.Fatalf("expected strict hydration fallback diagnostic, got %+v", parseDiagnostics)
}

func TestHydrateStrictModePanicsOnTextMismatch(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, ShowRawPanicOutput: true})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("p")
	parseServerText := parseAdapter.CreateTextNode("Server")
	parseAdapter.AppendChild(parseServerNode, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseRt.SetNextHydrationStrict(true)
	parseRt.Hydrate(CreateElement("p", nil, "Client"), parseContainer)
	expectPanic(parseT, func() {
		runHydrationWork(parseT, parseScheduler)
	})

	if parseGot := parseServerText.(*testDOMNode).text; parseGot != "Server" {
		parseT.Fatalf("expected strict hydration to avoid rewriting mismatched text, got %q", parseGot)
	}

	parseDiagnostics := GetDiagnostics()
	for _, parseDiagnostic := range parseDiagnostics {
		if strings.Contains(parseDiagnostic.Message, "hydration text mismatch") {
			if parseDiagnostic.Severity != DiagnosticError {
				parseT.Fatalf("expected strict hydration text mismatch to be an error, got %+v", parseDiagnostic)
			}
			return
		}
	}
	parseT.Fatalf("expected strict hydration text mismatch diagnostic, got %+v", parseDiagnostics)
}

func TestHydrateSupportsComponentUpdatesAfterResume(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerButton := parseAdapter.CreateElement("button")
	parseAdapter.SetAttribute(parseServerButton, "id", "counter")
	parseServerText := parseAdapter.CreateTextNode("count:0")
	parseAdapter.AppendChild(parseServerButton, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerButton)

	var setCount func(any)
	parseCounter := func() *Element {
		parseCount, set := GoUseState(parseRt, 0)
		setCount = set
		return CreateElement("button", map[string]any{"id": "counter"}, fmt.Sprintf("count:%d", parseCount()))
	}

	parseRt.Hydrate(CreateElement(parseCounter, nil), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	if setCount == nil {
		parseT.Fatal("expected hydrated component to expose state setter")
	}
	setCount(1)
	if len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected state update to schedule follow-up render")
	}
	runHydrationWork(parseT, parseScheduler)
	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 || !parseChildren[0].Equals(parseServerButton) {
		parseT.Fatal("expected hydrated update to keep existing host node")
	}
	parseTextChildren := parseAdapter.GetChildren(parseServerButton)
	if len(parseTextChildren) != 1 {
		parseT.Fatalf("expected one button text child, got %d", len(parseTextChildren))
	}
	if parseGot := parseTextChildren[0].(*testDOMNode).text; parseGot != "count:1" {
		parseT.Fatalf("expected hydrated update to change text to count:1, got %q", parseGot)
	}
}

func TestHydrateReportsObservabilityMetrics(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("span")
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	var parseObserved HydrationMetrics
	parseRt.SetNextHydrationObserver("req-42", func(parseMetrics HydrationMetrics) {
		parseObserved = parseMetrics
	})

	parseRt.Hydrate(CreateElement("div", map[string]any{"id": "client"}), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	if parseObserved.CorrelationID != "req-42" {
		parseT.Fatalf("expected correlation id to be preserved, got %+v", parseObserved)
	}
	if parseObserved.FallbackCount != 1 {
		parseT.Fatalf("expected one hydration fallback, got %+v", parseObserved)
	}
	if parseObserved.ExistingDOMNodeCount != 1 {
		parseT.Fatalf("expected existing DOM count to be recorded, got %+v", parseObserved)
	}
	if parseObserved.DiscardedNodeCount != 1 {
		parseT.Fatalf("expected discarded node count to be recorded, got %+v", parseObserved)
	}
	if parseObserved.DurationNs < 0 {
		parseT.Fatalf("expected non-negative hydration duration, got %+v", parseObserved)
	}
	if parseObserved.Failed {
		parseT.Fatalf("expected non-strict hydration to finish without failure, got %+v", parseObserved)
	}
}

func TestHydrateUpdatesClosureComponentChildrenAfterResume(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("p")
	parseAdapter.SetAttribute(parseServerNode, "id", "value")
	parseServerText := parseAdapter.CreateTextNode("value:0")
	parseAdapter.AppendChild(parseServerNode, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	var setCount func(any)
	parseComponent := func() *Element {
		parseCount, set := GoUseState(parseRt, 0)
		setCount = set
		parseCurrent := parseCount()
		return CreateElement(func() *Element {
			return CreateElement("p", map[string]any{"id": "value"}, fmt.Sprintf("value:%d", parseCurrent))
		}, nil)
	}

	parseRt.Hydrate(CreateElement(parseComponent, nil), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	if setCount == nil {
		parseT.Fatal("expected hydrated component to expose state setter")
	}
	setCount(1)
	if len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected state update to schedule follow-up render")
	}
	runHydrationWork(parseT, parseScheduler)

	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 {
		parseT.Fatalf("expected one host child after hydrated closure update, got %d", len(parseChildren))
	}
	parseUpdatedNode := parseChildren[0]
	parseUpdatedElement, parseOk := parseUpdatedNode.(*testDOMNode)
	if !parseOk {
		parseT.Fatal("expected updated hydrated node to be a test DOM node")
	}
	if parseGot := parseUpdatedElement.attributes["id"]; parseGot != "value" {
		parseT.Fatalf("expected hydrated closure update to keep the rendered id, got %q", parseGot)
	}
	parseTextChildren := parseAdapter.GetChildren(parseUpdatedNode)
	if len(parseTextChildren) != 1 {
		parseT.Fatalf("expected one text child after hydrated closure update, got %d", len(parseTextChildren))
	}
	if parseGot2 := parseTextChildren[0].(*testDOMNode).text; parseGot2 != "value:1" {
		parseT.Fatalf("expected hydrated closure update to change text to value:1, got %q", parseGot2)
	}
}

func TestHydrateDefersAtomSubscriptionsUntilCommitCompletes(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("p")
	parseServerText := parseAdapter.CreateTextNode("light")
	parseAdapter.AppendChild(parseServerNode, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseReader := func() *Element {
		parseTheme, _ := GoUseAtom(parseRt, "theme", "light")
		return CreateElement("p", nil, parseTheme())
	}

	parseRt.Hydrate(CreateElement(parseReader, nil), parseContainer)
	if parseCount := parseRt.atomRegistry.GetSubscriberCount("theme"); parseCount != 0 {
		parseT.Fatalf("expected no atom subscribers before hydration commit, got %d", parseCount)
	}
	runHydrationWork(parseT, parseScheduler)
	if parseCount2 := parseRt.atomRegistry.GetSubscriberCount("theme"); parseCount2 != 1 {
		parseT.Fatalf("expected atom subscription after hydration commit, got %d", parseCount2)
	}
}

func TestHydrateEffectStateUpdatesScheduleAfterCommit(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("p")
	parseServerText := parseAdapter.CreateTextNode("count:0")
	parseAdapter.AppendChild(parseServerNode, parseServerText)
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseComponent := func() *Element {
		parseCount, setCount := GoUseState(parseRt, 0)
		GoUseEffect(func() func() {
			if parseCount() == 0 {
				setCount(1)
			}
			return nil
		}, parseCount())
		return CreateElement("p", nil, fmt.Sprintf("count:%d", parseCount()))
	}

	parseRt.Hydrate(CreateElement(parseComponent, nil), parseContainer)
	runHydrationWork(parseT, parseScheduler)
	if len(parseScheduler.timeouts) == 0 {
		parseT.Fatal("expected effect-triggered state update to schedule follow-up work after hydration commit")
	}
	runHydrationWork(parseT, parseScheduler)

	parseChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseChildren) != 1 {
		parseT.Fatalf("expected one child after post-hydration effect update, got %d", len(parseChildren))
	}
	parseTextChildren := parseAdapter.GetChildren(parseChildren[0])
	if len(parseTextChildren) != 1 {
		parseT.Fatalf("expected one text child after post-hydration effect update, got %d", len(parseTextChildren))
	}
	if parseGot := parseTextChildren[0].(*testDOMNode).text; parseGot != "count:1" {
		parseT.Fatalf("expected effect-driven post-hydration update to change text to count:1, got %q", parseGot)
	}
}

func TestHydrateAttachesEventHandlersBeforeEffectsRun(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerButton := parseAdapter.CreateElement("button")
	parseAdapter.SetAttribute(parseServerButton, "id", "action")
	parseAdapter.AppendChild(parseServerButton, parseAdapter.CreateTextNode("Run"))
	parseAdapter.AppendChild(parseContainer, parseServerButton)

	isParseHandlerVisibleDuringEffect := false
	parseComponent := func() *Element {
		GoUseEffect(func() func() {
			isParseHandlerVisibleDuringEffect = parseServerButton.(*testDOMNode).properties["onclick"] != nil
			return nil
		})
		return CreateElement("button", map[string]any{
			"id":      "action",
			"onclick": func() {},
		}, "Run")
	}

	parseRt.Hydrate(CreateElement(parseComponent, nil), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	if !isParseHandlerVisibleDuringEffect {
		parseT.Fatal("expected hydration to attach event handlers before effects run")
	}
}

func TestHydratePreservesLiveInputValueUntilPostHydrationUpdate(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	parseServerInput := parseAdapter.CreateElement("input")
	parseAdapter.SetAttribute(parseServerInput, "id", "name")
	parseAdapter.SetAttribute(parseServerInput, "value", "server")
	parseAdapter.SetProperty(parseServerInput, "value", "draft")
	parseAdapter.AppendChild(parseContainer, parseServerInput)

	var setValue func(any)
	parseComponent := func() *Element {
		parseValue, set := GoUseState(parseRt, "server")
		setValue = set
		return CreateElement("input", map[string]any{
			"id":    "name",
			"value": parseValue(),
		})
	}

	parseRt.Hydrate(CreateElement(parseComponent, nil), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	if setValue == nil {
		parseT.Fatal("expected hydrated input component to expose state setter")
	}
	if parseGot := parseServerInput.(*testDOMNode).properties["value"]; parseGot != "draft" {
		parseT.Fatalf("expected hydration to preserve live input value draft, got %#v", parseGot)
	}

	setValue("client")
	runHydrationWork(parseT, parseScheduler)

	if parseGot2 := parseServerInput.(*testDOMNode).properties["value"]; parseGot2 != "client" {
		parseT.Fatalf("expected post-hydration update to apply controlled value, got %#v", parseGot2)
	}
}
