package runtime

import (
	"errors"
	"strings"
	"testing"
)

func flushScheduledWork(scheduler *testScheduler) {
	for len(scheduler.timeouts) > 0 {
		callbacks := append([]func(){}, scheduler.timeouts...)
		scheduler.timeouts = scheduler.timeouts[:0]
		for _, callback := range callbacks {
			callback()
		}
	}
}

func textFromNode(node DOMNode) string {
	if node == nil {
		return ""
	}
	typed, ok := node.(*testDOMNode)
	if !ok {
		return ""
	}
	if typed.nodeType == "text" {
		return typed.text
	}
	var builder strings.Builder
	for _, child := range typed.children {
		builder.WriteString(textFromNode(child))
	}
	return builder.String()
}

func TestRenderToStringErrorBoundaryFallback(t *testing.T) {
	boundary := NewErrorBoundaryType()
	boom := func() *Element {
		panic("server boom")
	}
	element := CreateElement(boundary, map[string]interface{}{
		"errorFallback": func(err error, reset func()) *Element {
			if err == nil || err.Error() != "server boom" {
				t.Fatalf("unexpected boundary error: %v", err)
			}
			return CreateElement("p", nil, "caught server boom")
		},
	}, CreateElement(boom, nil))

	markup, err := RenderToString(element)
	if err != nil {
		t.Fatalf("unexpected error-boundary render error: %v", err)
	}
	if markup != `<p>caught server boom</p>` {
		t.Fatalf("unexpected boundary fallback markup: %q", markup)
	}
}

func TestErrorBoundaryRecoversRenderPanic(t *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	boundary := NewErrorBoundaryType()
	boom := func() *Element {
		panic("render boom")
	}

	rt.Render(CreateElement(boundary, map[string]interface{}{
		"errorFallback": func(err error, reset func()) *Element {
			return CreateElement("p", nil, "render fallback: "+err.Error())
		},
	}, CreateElement(boom, nil)), container)
	flushScheduledWork(scheduler)

	root := container.(*testDOMNode)
	if len(root.children) != 1 {
		t.Fatalf("expected one fallback node, got %d", len(root.children))
	}
	if got := textFromNode(root.children[0]); got != "render fallback: render boom" {
		t.Fatalf("unexpected render fallback text: %q", got)
	}

	diagnostics := GetDiagnostics()
	if len(diagnostics) == 0 {
		t.Fatal("expected recovered render panic to produce a diagnostic")
	}
	last := diagnostics[len(diagnostics)-1]
	if !strings.Contains(last.Message, "error boundary caught render failure") {
		t.Fatalf("expected render recovery diagnostic, got %+v", last)
	}
	if last.Path == "" || len(last.ComponentStack) == 0 {
		t.Fatalf("expected boundary recovery diagnostic to include path and stack context, got %+v", last)
	}
	foundBoundary := false
	for _, entry := range last.ComponentStack {
		if entry == "ErrorBoundary" {
			foundBoundary = true
			break
		}
	}
	if !foundBoundary {
		t.Fatalf("expected component stack to include ErrorBoundary, got %+v", last.ComponentStack)
	}
}

func TestErrorBoundaryRecoversEffectPanic(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	boundary := NewErrorBoundaryType()
	effectComp := func() *Element {
		GoUseEffect(func() func() {
			panic("effect boom")
		})
		return CreateElement("span", nil, "content")
	}

	rt.Render(CreateElement(boundary, map[string]interface{}{
		"errorFallback": func(err error, reset func()) *Element {
			return CreateElement("p", nil, "effect fallback: "+err.Error())
		},
	}, CreateElement(effectComp, nil)), container)
	flushScheduledWork(scheduler)

	root := container.(*testDOMNode)
	if len(root.children) != 1 {
		t.Fatalf("expected one fallback node after effect failure, got %d", len(root.children))
	}
	if got := textFromNode(root.children[0]); got != "effect fallback: effect boom" {
		t.Fatalf("unexpected effect fallback text: %q", got)
	}
}

func TestErrorBoundaryRecoversEventPanicAndResets(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	rt := GetGlobalRuntime()
	container := adapter.CreateElement("div")
	boundary := NewErrorBoundaryType()
	shouldPanic := true
	eventComp := func() *Element {
		handler := GoUseFunc(func() {
			if shouldPanic {
				panic(errors.New("event boom"))
			}
		})
		return CreateElement("button", map[string]interface{}{"onclick": handler}, "click")
	}

	rootElement := CreateElement(boundary, map[string]interface{}{
		"errorFallback": func(err error, reset func()) *Element {
			return CreateElement("button", map[string]interface{}{
				"onclick": func() {
					shouldPanic = false
					reset()
				},
			}, "reset: "+err.Error())
		},
	}, CreateElement(eventComp, nil))

	rt.Render(rootElement, container)
	flushScheduledWork(scheduler)

	root := container.(*testDOMNode)
	if len(root.children) == 0 {
		t.Fatal("expected initial child before firing event")
	}
	button := root.children[0].(*testDOMNode)
	handler, ok := button.properties["onclick"].(func())
	if !ok {
		t.Fatal("expected click handler on rendered button")
	}
	handler()
	flushScheduledWork(scheduler)
	if len(root.children) == 0 {
		t.Fatalf("expected fallback child after event recovery, got none; boundary error=%v", rt.currentRoot.child.boundaryError)
	}

	if got := textFromNode(root.children[0]); got != "reset: event boom" {
		t.Fatalf("unexpected event fallback text: %q", got)
	}

	resetHandler, ok := root.children[0].(*testDOMNode).properties["onclick"].(func())
	if !ok {
		t.Fatal("expected reset handler on fallback button")
	}
	resetHandler()
	flushScheduledWork(scheduler)
	if len(root.children) == 0 {
		t.Fatal("expected restored child after boundary reset, got none")
	}

	if got := textFromNode(root.children[0]); got != "click" {
		t.Fatalf("expected boundary reset to restore original child, got %q", got)
	}
}
