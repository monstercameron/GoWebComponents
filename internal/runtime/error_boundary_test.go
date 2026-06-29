package runtime

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func flushScheduledWork(parseScheduler *testScheduler) {
	for len(parseScheduler.timeouts) > 0 {
		parseCallbacks := append([]func(){}, parseScheduler.timeouts...)
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		for _, parseCallback := range parseCallbacks {
			parseCallback()
		}
	}
}

func textFromNode(parseNode DOMNode) string {
	if parseNode == nil {
		return ""
	}
	parseTyped, parseOk := parseNode.(*testDOMNode)
	if !parseOk {
		return ""
	}
	if parseTyped.nodeType == "text" {
		return parseTyped.text
	}
	var parseBuilder strings.Builder
	for _, parseChild := range parseTyped.children {
		parseBuilder.WriteString(textFromNode(parseChild))
	}
	return parseBuilder.String()
}

func TestRenderToStringErrorBoundaryFallback(parseT *testing.T) {
	parseBoundary := NewErrorBoundaryType()
	parseBoom := func() *Element {
		panic("server boom")
	}
	parseElement := CreateElement(parseBoundary, map[string]any{
		"errorFallback": func(parseErr2 error, reset func()) *Element {
			if parseErr2 == nil || parseErr2.Error() != "server boom" {
				parseT.Fatalf("unexpected boundary error: %v", parseErr2)
			}
			return CreateElement("p", nil, "caught server boom")
		},
	}, CreateElement(parseBoom, nil))

	parseMarkup, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected error-boundary render error: %v", parseErr)
	}
	if parseMarkup != `<p>caught server boom</p>` {
		parseT.Fatalf("unexpected boundary fallback markup: %q", parseMarkup)
	}
}

func TestErrorBoundaryRecoversRenderPanic(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseBoundary := NewErrorBoundaryType()
	parseBoom := func() *Element {
		panic("render boom")
	}

	parseRt.Render(CreateElement(parseBoundary, map[string]any{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("p", nil, "render fallback: "+parseErr.Error())
		},
	}, CreateElement(parseBoom, nil)), parseContainer)
	flushScheduledWork(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	if len(parseRoot.children) != 1 {
		parseT.Fatalf("expected one fallback node, got %d", len(parseRoot.children))
	}
	if parseGot := textFromNode(parseRoot.children[0]); parseGot != "render fallback: render boom" {
		parseT.Fatalf("unexpected render fallback text: %q", parseGot)
	}

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 {
		parseT.Fatal("expected recovered render panic to produce a diagnostic")
	}
	parseLast := parseDiagnostics[len(parseDiagnostics)-1]
	if !strings.Contains(parseLast.Message, "error boundary caught render failure") {
		parseT.Fatalf("expected render recovery diagnostic, got %+v", parseLast)
	}
	if parseLast.Path == "" || len(parseLast.ComponentStack) == 0 {
		parseT.Fatalf("expected boundary recovery diagnostic to include path and stack context, got %+v", parseLast)
	}
	isParseFoundBoundary := slices.Contains(parseLast.ComponentStack, "ErrorBoundary")
	if !isParseFoundBoundary {
		parseT.Fatalf("expected component stack to include ErrorBoundary, got %+v", parseLast.ComponentStack)
	}
}

func TestErrorBoundaryRecoversEffectPanic(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseBoundary := NewErrorBoundaryType()
	parseEffectComp := func() *Element {
		GoUseEffect(func() func() {
			panic("effect boom")
		})
		return CreateElement("span", nil, "content")
	}

	parseRt.Render(CreateElement(parseBoundary, map[string]any{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("p", nil, "effect fallback: "+parseErr.Error())
		},
	}, CreateElement(parseEffectComp, nil)), parseContainer)
	flushScheduledWork(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	if len(parseRoot.children) != 1 {
		parseT.Fatalf("expected one fallback node after effect failure, got %d", len(parseRoot.children))
	}
	if parseGot := textFromNode(parseRoot.children[0]); parseGot != "effect fallback: effect boom" {
		parseT.Fatalf("unexpected effect fallback text: %q", parseGot)
	}
}

func TestErrorBoundaryRecoversEventPanicAndResets(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseRt := GetGlobalRuntime()
	parseContainer := parseAdapter.CreateElement("div")
	parseBoundary := NewErrorBoundaryType()
	shouldPanic := true
	parseEventComp := func() *Element {
		parseHandler := GoUseFunc(func() {
			if shouldPanic {
				panic(errors.New("event boom"))
			}
		})
		return CreateElement("button", map[string]any{"onclick": parseHandler}, "click")
	}

	parseRootElement := CreateElement(parseBoundary, map[string]any{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("button", map[string]any{
				"onclick": func() {
					shouldPanic = false
					reset()
				},
			}, "reset: "+parseErr.Error())
		},
	}, CreateElement(parseEventComp, nil))

	parseRt.Render(parseRootElement, parseContainer)
	flushScheduledWork(parseScheduler)

	parseRoot := parseContainer.(*testDOMNode)
	if len(parseRoot.children) == 0 {
		parseT.Fatal("expected initial child before firing event")
	}
	parseButton := parseRoot.children[0].(*testDOMNode)
	parseHandler2, parseOk := parseButton.properties["onclick"].(func())
	if !parseOk {
		parseT.Fatal("expected click handler on rendered button")
	}
	parseHandler2()
	flushScheduledWork(parseScheduler)
	if len(parseRoot.children) == 0 {
		parseT.Fatalf("expected fallback child after event recovery, got none; boundary error=%v", parseRt.currentRoot.child.boundaryError)
	}

	if parseGot := textFromNode(parseRoot.children[0]); parseGot != "reset: event boom" {
		parseT.Fatalf("unexpected event fallback text: %q", parseGot)
	}

	resetHandler, parseOk := parseRoot.children[0].(*testDOMNode).properties["onclick"].(func())
	if !parseOk {
		parseT.Fatal("expected reset handler on fallback button")
	}
	resetHandler()
	flushScheduledWork(parseScheduler)
	if len(parseRoot.children) == 0 {
		parseT.Fatal("expected restored child after boundary reset, got none")
	}

	if parseGot2 := textFromNode(parseRoot.children[0]); parseGot2 != "click" {
		parseT.Fatalf("expected boundary reset to restore original child, got %q", parseGot2)
	}
}
