package runtime

import (
	"strings"
	"testing"
)

var hookOrderSkipsMemo bool
var hookOrderExtraHook bool

func hookOrderConditionalComponent() *Element {
	GoUseState(nil, 1)
	if hookOrderSkipsMemo {
		GoUseId()
	} else {
		GoUseMemo(func() any { return 1 })
	}
	return CreateElement("div", nil, "ok")
}

func hookOrderLengthComponent() *Element {
	GoUseState(nil, 1)
	if hookOrderExtraHook {
		GoUseId()
	}
	return CreateElement("div", nil, "ok")
}

func hookOrderStableComponent() *Element {
	GoUseState(nil, 1)
	GoUseId()
	return CreateElement("div", nil, "ok")
}

// renderHookOrderPasses renders one component through renderFunctionComponent
// the way the reconciler does: each pass is a new fiber whose alternate is the
// previous pass, so hooks (and the hook-order baseline) carry over.
func renderHookOrderPasses(parseT *testing.T, parseComponent func() *Element, parsePasses int, parseBeforePass func(parsePass int)) {
	parseT.Helper()
	parseRt := &Runtime{}
	parseRoot := &Fiber{typeOf: "ROOT"}
	var parsePrevious *Fiber
	for parsePass := 0; parsePass < parsePasses; parsePass++ {
		if parseBeforePass != nil {
			parseBeforePass(parsePass)
		}
		parseFiber := &Fiber{typeOf: parseComponent, parent: parseRoot, alternate: parsePrevious}
		if _, parseHandled, _ := parseRt.renderFunctionComponent(parseFiber); parseHandled {
			parseT.Fatalf("pass %d: render was handled as a panic", parsePass)
		}
		parsePrevious = parseFiber
	}
}

func getHookOrderDiagnostics() []Diagnostic {
	var parseOut []Diagnostic
	for _, parseDiagnostic := range GetDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, "GWC-RUNTIME-HOOK-ORDER") {
			parseOut = append(parseOut, parseDiagnostic)
		}
	}
	return parseOut
}

func TestHookOrderKindSwapReportsExactlyOnce(parseT *testing.T) {
	if !hookSignatureRecordingEnabled {
		parseT.Skip("hook signature recording is compiled out")
	}
	ClearDiagnostics()
	defer ClearDiagnostics()
	defer func() { hookOrderSkipsMemo = false }()

	renderHookOrderPasses(parseT, hookOrderConditionalComponent, 5, func(parsePass int) {
		hookOrderSkipsMemo = parsePass >= 2
	})

	parseDiagnostics := getHookOrderDiagnostics()
	if len(parseDiagnostics) != 1 {
		parseT.Fatalf("expected exactly one hook-order diagnostic, got %d: %+v", len(parseDiagnostics), parseDiagnostics)
	}
	parseMessage := parseDiagnostics[0].Message
	if !strings.Contains(parseMessage, "slot 1: was UseMemo, now UseId") {
		parseT.Fatalf("expected slot/kind detail, got %q", parseMessage)
	}
	if parseDiagnostics[0].Code != "GWC-RUNTIME-HOOK-ORDER" {
		parseT.Fatalf("expected registered code, got %+v", parseDiagnostics[0])
	}
}

func TestHookOrderLengthChangeReportsExactlyOnce(parseT *testing.T) {
	if !hookSignatureRecordingEnabled {
		parseT.Skip("hook signature recording is compiled out")
	}
	ClearDiagnostics()
	defer ClearDiagnostics()
	defer func() { hookOrderExtraHook = false }()

	renderHookOrderPasses(parseT, hookOrderLengthComponent, 4, func(parsePass int) {
		hookOrderExtraHook = parsePass >= 1
	})

	parseDiagnostics := getHookOrderDiagnostics()
	if len(parseDiagnostics) != 1 {
		parseT.Fatalf("expected exactly one hook-order diagnostic, got %d: %+v", len(parseDiagnostics), parseDiagnostics)
	}
	if !strings.Contains(parseDiagnostics[0].Message, "hook count changed: 1 hooks last render, 2 now") {
		parseT.Fatalf("expected length detail, got %q", parseDiagnostics[0].Message)
	}
}

func TestHookOrderStableComponentReportsNothing(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	renderHookOrderPasses(parseT, hookOrderStableComponent, 6, nil)

	if parseDiagnostics := getHookOrderDiagnostics(); len(parseDiagnostics) != 0 {
		parseT.Fatalf("stable component must not trigger hook-order diagnostics, got %+v", parseDiagnostics)
	}
}

func TestHookOrderSteadyStateRendersDoNotAllocateSignatureBuffers(parseT *testing.T) {
	if !hookSignatureRecordingEnabled {
		parseT.Skip("hook signature recording is compiled out")
	}
	parseHooks := &Hooks{}
	recordHookSignature(parseHooks, "state")
	recordHookSignature(parseHooks, "memo")
	finishHookOrderRender(parseHooks)
	beginHookOrderRender(parseHooks)
	recordHookSignature(parseHooks, "state")
	recordHookSignature(parseHooks, "memo")
	finishHookOrderRender(parseHooks)

	parseAllocs := testing.AllocsPerRun(50, func() {
		beginHookOrderRender(parseHooks)
		recordHookSignature(parseHooks, "state")
		recordHookSignature(parseHooks, "memo")
		finishHookOrderRender(parseHooks)
	})
	if parseAllocs != 0 {
		parseT.Fatalf("steady-state hook-order tracking allocated %.1f times per render", parseAllocs)
	}
}

func TestHookOrderDiagnosticCarriesComponentSourceLocation(parseT *testing.T) {
	if !hookSignatureRecordingEnabled {
		parseT.Skip("hook signature recording is compiled out")
	}
	ClearDiagnostics()
	defer ClearDiagnostics()
	defer func() { hookOrderSkipsMemo = false }()

	renderHookOrderPasses(parseT, hookOrderConditionalComponent, 3, func(parsePass int) {
		hookOrderSkipsMemo = parsePass >= 1
	})

	parseDiagnostics := getHookOrderDiagnostics()
	if len(parseDiagnostics) != 1 || len(parseDiagnostics[0].ComponentStack) != 1 {
		parseT.Fatalf("expected one diagnostic with one stack entry, got %+v", parseDiagnostics)
	}
	if !strings.Contains(parseDiagnostics[0].ComponentStack[0], "hook_order_test.go:") {
		parseT.Fatalf("expected file:line in component stack, got %q", parseDiagnostics[0].ComponentStack[0])
	}
}

func TestDescribeCallableSourceReportsFileAndLine(parseT *testing.T) {
	if !hookSignatureRecordingEnabled {
		parseT.Skip("hook signature recording is compiled out")
	}
	parseFile, parseLine := describeCallableSource(hookOrderStableComponent)
	if !strings.HasSuffix(parseFile, "hook_order_test.go") || parseLine <= 0 {
		parseT.Fatalf("unexpected source %q:%d", parseFile, parseLine)
	}
	parseFiber := &Fiber{typeOf: hookOrderStableComponent, parent: &Fiber{typeOf: "ROOT"}}
	if parsePath := diagnosticPathWithSource(parseFiber); !strings.Contains(parsePath, "hookOrderStableComponent (") || !strings.Contains(parsePath, "hook_order_test.go:") {
		parseT.Fatalf("expected source in path, got %q", parsePath)
	}
	if parsePlain := diagnosticPathForFiber(parseFiber); parsePlain != "hookOrderStableComponent" {
		parseT.Fatalf("plain path must stay name-only (hot reload keys on it), got %q", parsePlain)
	}
	if parseFile, parseLine := describeCallableSource(nil); parseFile != "" || parseLine != 0 {
		parseT.Fatalf("nil must yield no source, got %q:%d", parseFile, parseLine)
	}
}
