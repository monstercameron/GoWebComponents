//go:build !js || !wasm

package runtime

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestNativeAgentHelpersNilAndErrorContracts(parseT *testing.T) {
	var parseNil *Runtime
	if parseNil.SelectorResolves("#app") {
		parseT.Fatal("nil runtime should not resolve selectors")
	}
	if parseNil.AgentStateVersion() != 0 || parseNil.AdvanceAgentStateVersion() != 0 {
		parseT.Fatal("nil runtime state version methods should return zero")
	}

	parseErr := writeInvokeHandlerPlatform(func() {}, "onclick", "ref-1")
	if !errors.Is(parseErr, ErrAgentNoHandler) {
		parseT.Fatalf("native handler error should wrap ErrAgentNoHandler, got %v", parseErr)
	}
	if !strings.Contains(parseErr.Error(), `handler "onclick" on ref "ref-1"`) {
		parseT.Fatalf("native handler error lost context: %v", parseErr)
	}
}

func TestAgentStateVersionAdvancesMonotonically(parseT *testing.T) {
	parseRt := NewRuntime(Config{})
	if parseRt.AgentStateVersion() != 0 {
		parseT.Fatalf("initial version = %d, want 0", parseRt.AgentStateVersion())
	}
	if parseGot := parseRt.AdvanceAgentStateVersion(); parseGot != 1 {
		parseT.Fatalf("first advance = %d, want 1", parseGot)
	}
	if parseGot := parseRt.AdvanceAgentStateVersion(); parseGot != 2 || parseRt.AgentStateVersion() != 2 {
		parseT.Fatalf("second advance/current = %d/%d, want 2/2", parseGot, parseRt.AgentStateVersion())
	}
}

func TestRenderDetachedRequiresGlobalDOMAdapter(parseT *testing.T) {
	globalRuntimeMu.Lock()
	parsePrev := globalRuntime
	globalRuntime = &Runtime{}
	globalRuntimeMu.Unlock()
	parseT.Cleanup(func() {
		globalRuntimeMu.Lock()
		globalRuntime = parsePrev
		globalRuntimeMu.Unlock()
	})

	parseErr := RenderDetached("#agent", CreateElement("TEXT_ELEMENT", map[string]any{"nodeValue": "x"}))
	if parseErr == nil || !strings.Contains(parseErr.Error(), "global runtime has no DOM adapter") {
		parseT.Fatalf("expected missing DOM adapter error, got %v", parseErr)
	}
}

func TestRenderDetachedResolvesSelectorAndReusesRuntime(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseOtherContainer := parseAdapter.CreateElement("section")
	parseAdapter.selectorResults["#agent"] = parseContainer
	parseAdapter.selectorResults["#other"] = parseOtherContainer
	parseScheduler := newTestScheduler()

	globalRuntimeMu.Lock()
	parsePrevGlobal := globalRuntime
	globalRuntime = NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	globalRuntimeMu.Unlock()
	detachedRuntimeMu.Lock()
	parsePrevDetached := detachedRuntimes
	detachedRuntimes = map[string]*Runtime{}
	detachedRuntimeMu.Unlock()
	parseT.Cleanup(func() {
		globalRuntimeMu.Lock()
		globalRuntime = parsePrevGlobal
		globalRuntimeMu.Unlock()
		detachedRuntimeMu.Lock()
		detachedRuntimes = parsePrevDetached
		detachedRuntimeMu.Unlock()
	})

	if parseErr := RenderDetached("#agent", CreateElement("div", map[string]any{"id": "overlay"})); parseErr != nil {
		parseT.Fatalf("RenderDetached first call error = %v", parseErr)
	}
	detachedRuntimeMu.Lock()
	parseFirst := detachedRuntimes["#agent"]
	parseCountAfterFirst := len(detachedRuntimes)
	detachedRuntimeMu.Unlock()
	if parseFirst == nil || parseCountAfterFirst != 1 {
		parseT.Fatalf("detached registry after first render = %#v", detachedRuntimes)
	}
	if parseFirst.domAdapter != parseAdapter || parseFirst.scheduler != parseScheduler {
		parseT.Fatal("detached runtime should share global adapters")
	}

	if parseErr := RenderDetached("#agent", nil); parseErr != nil {
		parseT.Fatalf("RenderDetached second call error = %v", parseErr)
	}
	detachedRuntimeMu.Lock()
	parseSecond := detachedRuntimes["#agent"]
	detachedRuntimeMu.Unlock()
	if parseSecond != parseFirst {
		parseT.Fatal("same selector should reuse detached runtime")
	}

	if parseErr := RenderDetached("#other", CreateElement("span", nil)); parseErr != nil {
		parseT.Fatalf("RenderDetached other selector error = %v", parseErr)
	}
	detachedRuntimeMu.Lock()
	parseOther := detachedRuntimes["#other"]
	parseCountAfterOther := len(detachedRuntimes)
	detachedRuntimeMu.Unlock()
	if parseOther == nil || parseOther == parseFirst || parseCountAfterOther != 2 {
		parseT.Fatalf("different selectors should get isolated runtimes: first=%p other=%p count=%d", parseFirst, parseOther, parseCountAfterOther)
	}
}

func TestRenderDetachedReportsMissingSelector(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	globalRuntimeMu.Lock()
	parsePrevGlobal := globalRuntime
	globalRuntime = NewRuntime(Config{DOMAdapter: parseAdapter})
	globalRuntimeMu.Unlock()
	parseT.Cleanup(func() {
		globalRuntimeMu.Lock()
		globalRuntime = parsePrevGlobal
		globalRuntimeMu.Unlock()
	})

	parseErr := RenderDetached("#missing", CreateElement("div", nil))
	if parseErr == nil || !strings.Contains(parseErr.Error(), `selector "#missing" did not resolve`) {
		parseT.Fatalf("expected missing selector error, got %v", parseErr)
	}
}

func TestSelectorResolvesWithNativeMockDOM(parseT *testing.T) {
	parseAdapter := newQueryTestDOMAdapter()
	parseNode := parseAdapter.CreateElement("section")
	parseAdapter.selectorResults["#agent"] = parseNode
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})

	if !parseRt.SelectorResolves("#agent") {
		parseT.Fatal("expected #agent selector to resolve")
	}
	if parseRt.SelectorResolves("#missing") {
		parseT.Fatal("expected missing selector not to resolve")
	}
}

func TestEffectTagStringValues(parseT *testing.T) {
	parseCases := map[effectTagKind]string{
		effectTagNone:      "",
		effectTagPlacement: "PLACEMENT",
		effectTagUpdate:    "UPDATE",
		effectTagDeletion:  "DELETION",
		effectTagHydrate:   "HYDRATE",
		effectTagKind(99):  "",
	}
	for parseTag, parseWant := range parseCases {
		if parseGot := parseTag.String(); parseGot != parseWant {
			parseT.Fatalf("effectTag %d String() = %q, want %q", parseTag, parseGot, parseWant)
		}
	}
}

func TestNativeGlobalHookShimsDelegateToRuntime(parseT *testing.T) {
	globalRuntimeMu.Lock()
	parsePrevGlobal := globalRuntime
	globalRuntime = NewRuntime(Config{Scheduler: newTestScheduler()})
	globalRuntimeMu.Unlock()
	parseFiber := &Fiber{typeOf: "test", props: map[string]any{}}
	SetCurrentFiber(parseFiber)
	parseT.Cleanup(func() {
		SetCurrentFiber(nil)
		globalRuntimeMu.Lock()
		globalRuntime = parsePrevGlobal
		globalRuntimeMu.Unlock()
	})

	parseGet, parseSet := GoUseStateGlobal(1)
	if parseGet() != 1 {
		parseT.Fatalf("GoUseStateGlobal initial = %d", parseGet())
	}
	parseSet(2)
	if parseGet() != 2 {
		parseT.Fatalf("GoUseStateGlobal set = %d", parseGet())
	}

	parseEffectCalled := false
	GoUseEffectGlobal(func() func() {
		parseEffectCalled = true
		return nil
	}, "dep")
	if parseEffectCalled {
		parseT.Fatal("GoUseEffectGlobal should register effect without running it synchronously")
	}

	if parseMemo := GoUseMemoGlobal(func() any { return "memo" }, "dep"); parseMemo != "memo" {
		parseT.Fatalf("GoUseMemoGlobal = %#v", parseMemo)
	}
	if parseMemoTyped := GoUseMemoGlobalTyped(func() any { return 7 }, reflect.TypeFor[int](), "dep"); parseMemoTyped != 7 {
		parseT.Fatalf("GoUseMemoGlobalTyped = %#v", parseMemoTyped)
	}
	if parseID := GoUseIdGlobal(); parseID == "" {
		parseT.Fatal("GoUseIdGlobal returned empty id")
	}
	if parseWrapped, parseOK := BuildDOMWrappedFunctionIfReadyGlobal(func() {}); parseWrapped != nil || parseOK {
		parseT.Fatalf("native DOM wrapper = (%#v, %v), want nil/false", parseWrapped, parseOK)
	}

	parseAtomGet, parseAtomSet := GoUseAtomGlobal("native-shim-atom", "initial")
	if parseAtomGet() != "initial" {
		parseT.Fatalf("GoUseAtomGlobal initial = %q", parseAtomGet())
	}
	parseAtomSet("updated")
	if parseAtomGet() != "updated" {
		parseT.Fatalf("GoUseAtomGlobal updated = %q", parseAtomGet())
	}
}
