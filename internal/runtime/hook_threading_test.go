//go:build !production

package runtime

import (
	"strings"
	"testing"
)

func TestHookThreadingGuardRejectsCrossGoroutineHookUse(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	SetCurrentFiber(nil)
	parseT.Cleanup(func() {
		ClearDiagnostics()
		ClearLogs()
		SetCurrentFiber(nil)
	})

	parseFiber := &Fiber{typeOf: "test", props: map[string]any{}}
	SetCurrentFiber(parseFiber)

	parseRecovered := make(chan any, 1)
	go func() {
		defer func() {
			parseRecovered <- recover()
		}()
		_, _ = GoUseState[int](nil, 0)
	}()

	parsePanic := <-parseRecovered
	if parsePanic == nil {
		parseT.Fatal("expected hook call from another goroutine to panic")
	}
	parseMessage, parseOk := parsePanic.(string)
	if !parseOk || !strings.Contains(parseMessage, "GWC-RUNTIME-HOOK-THREADING") || !strings.Contains(parseMessage, "GoUseState called from a goroutine") {
		parseT.Fatalf("expected actionable threading panic, got %#v", parsePanic)
	}

	parseFoundDiagnostic := false
	for _, parseDiagnostic := range GetDiagnostics() {
		if parseDiagnostic.Code == "GWC-RUNTIME-HOOK-THREADING" &&
			strings.Contains(parseDiagnostic.Message, "GoUseState called from a goroutine") {
			parseFoundDiagnostic = true
			break
		}
	}
	if !parseFoundDiagnostic {
		parseT.Fatalf("expected GWC-RUNTIME-HOOK-THREADING diagnostic, got %#v", GetDiagnostics())
	}
}

func TestRenderFunctionComponentClearsCurrentFiber(parseT *testing.T) {
	SetCurrentFiber(nil)
	parseT.Cleanup(func() {
		SetCurrentFiber(nil)
	})

	parseRuntime := NewRuntime(Config{Scheduler: newTestScheduler()})
	parseFiber := &Fiber{
		typeOf: func() *Element {
			_, _ = GoUseState[int](parseRuntime, 1)
			return CreateElement("div", nil)
		},
		props: map[string]any{},
	}

	parseElement, isHandled, parseNext := parseRuntime.renderFunctionComponent(parseFiber)
	if isHandled || parseNext != nil || parseElement == nil {
		parseT.Fatalf("expected successful component render, element=%#v handled=%v next=%#v", parseElement, isHandled, parseNext)
	}
	if parseCurrent := GetCurrentFiber(); parseCurrent != nil {
		parseT.Fatalf("expected current fiber to be cleared after render, got %#v", parseCurrent)
	}

	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected hook after render to panic outside component context")
		}
		parseMessage, parseOk := parseRecovered.(string)
		if !parseOk || !strings.Contains(parseMessage, "GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT") {
			parseT.Fatalf("expected outside-component hook panic, got %#v", parseRecovered)
		}
	}()
	_ = GoUseRef("after-render")
}
