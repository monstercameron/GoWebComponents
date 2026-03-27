package runtime

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// resolveContainerTestDOMAdapter adds a string-based ResolveNode branch for container resolution tests.
type resolveContainerTestDOMAdapter struct {
	*queryTestDOMAdapter
	resolveByKey map[string]DOMNode
}

// buildResolveContainerTestDOMAdapter constructs a DOM adapter that can resolve string keys to test nodes.
func buildResolveContainerTestDOMAdapter() *resolveContainerTestDOMAdapter {
	return &resolveContainerTestDOMAdapter{
		queryTestDOMAdapter: newQueryTestDOMAdapter(),
		resolveByKey:        make(map[string]DOMNode),
	}
}

// ResolveNode resolves string keys to DOM nodes before falling back to the base adapter behavior.
func (parseA *resolveContainerTestDOMAdapter) ResolveNode(parseValue interface{}) DOMNode {
	if parseKey, parseOk := parseValue.(string); parseOk {
		return parseA.resolveByKey[parseKey]
	}
	return parseA.queryTestDOMAdapter.ResolveNode(parseValue)
}

// TestRuntimeComponentTypeBranchHelpers covers nil and fallback branches for stable component handles.
func TestRuntimeComponentTypeBranchHelpers(parseT *testing.T) {
	var parseNilComponent *ComponentType
	parseNilComponent.SetImplementation("ignored")
	if parseNilComponent.Render(nil) != nil {
		parseT.Fatal("expected nil component handle render to return nil")
	}
	if parseNilComponent.IdentityKey() != "" {
		parseT.Fatalf("expected nil component handle identity to be empty, got %q", parseNilComponent.IdentityKey())
	}

	parseHandle := NewComponentType("", "", "", nil, nil)
	if parseHandle.Render(nil) != nil {
		parseT.Fatal("expected missing render implementation to return nil")
	}

	parseHandle.QualifiedName = "example.com/Widget"
	if parseHandle.IdentityKey() != "example.com/Widget" {
		parseT.Fatalf("expected qualified-name identity fallback, got %q", parseHandle.IdentityKey())
	}

	parseHandle.QualifiedName = ""
	parseHandle.Name = "Widget"
	if parseHandle.IdentityKey() != "Widget" {
		parseT.Fatalf("expected name identity fallback, got %q", parseHandle.IdentityKey())
	}
}

// TestRuntimeWrapEventHandlerBranchHelpers covers native event-wrapper branches without triggering panic recovery.
func TestRuntimeWrapEventHandlerBranchHelpers(parseT *testing.T) {
	parseRt := &Runtime{}
	parseRt.profiling.startupStartedAt = time.Now().Add(-1 * time.Millisecond)

	isParseCalled := false
	parseWrappedFunc, parseOk := parseRt.wrapEventHandler(nil, func() {
		isParseCalled = true
	}).(func())
	if !parseOk {
		parseT.Fatal("expected func() to be wrapped as func()")
	}
	parseWrappedFunc()
	if !isParseCalled {
		parseT.Fatal("expected wrapped func() handler to run")
	}
	if !parseRt.profiling.firstInteractionCaptured || parseRt.profiling.firstInteractionEvent != "event" {
		parseT.Fatalf("expected first interaction capture from wrapped event, got %+v", parseRt.profiling)
	}

	parseRt.profiling.firstInteractionCaptured = false
	parseRt.profiling.startupStartedAt = time.Now().Add(-1 * time.Millisecond)
	parseReceivedValue := ""
	parseWrappedString, parseOk := parseRt.wrapEventHandler(nil, func(parseEventValue string) {
		parseReceivedValue = parseEventValue
	}).(func(string))
	if !parseOk {
		parseT.Fatal("expected func(string) to be wrapped as func(string)")
	}
	parseWrappedString("payload")
	if parseReceivedValue != "payload" {
		parseT.Fatalf("expected wrapped string event payload, got %q", parseReceivedValue)
	}

	parseRt.profiling.firstInteractionCaptured = false
	parseRt.profiling.startupStartedAt = time.Now().Add(-1 * time.Millisecond)
	parseWrappedError, parseOk := parseRt.wrapEventHandler(nil, func() error {
		return errors.New("boom")
	}).(func() error)
	if !parseOk {
		parseT.Fatal("expected func() error to be wrapped as func() error")
	}
	if parseErr := parseWrappedError(); parseErr == nil || parseErr.Error() != "boom" {
		parseT.Fatalf("expected wrapped error handler to preserve return error, got %v", parseErr)
	}

	if parseDefault := parseRt.wrapEventHandler(nil, 42); parseDefault != 42 {
		parseT.Fatalf("expected unsupported handler to pass through unchanged, got %#v", parseDefault)
	}
}

// TestRuntimeStrictDiagnosticsBranchHelpers covers strict-diagnostics filter and match branches.
func TestRuntimeStrictDiagnosticsBranchHelpers(parseT *testing.T) {
	parsePreviousOptions := CurrentStrictDiagnosticsOptions()
	parseT.Cleanup(func() {
		ConfigureStrictDiagnostics(parsePreviousOptions)
	})

	parseDetails := diagnosticDetails{Code: "MATCH", Recoverable: true}
	if shouldEscalateDiagnosticStrictly("runtime", DiagnosticWarning, DiagnosticRecovered, parseDetails) {
		parseT.Fatal("expected disabled strict diagnostics to skip escalation")
	}

	ConfigureStrictDiagnostics(StrictDiagnosticsOptions{
		Enabled:         true,
		Codes:           []string{"MATCH"},
		Sources:         []string{"runtime"},
		Classifications: []DiagnosticClassification{DiagnosticRecovered},
	})

	if shouldEscalateDiagnosticStrictly("runtime", DiagnosticWarning, DiagnosticRecovered, diagnosticDetails{Code: "MATCH"}) {
		parseT.Fatal("expected unrecoverable diagnostic to be skipped in strict mode")
	}
	if shouldEscalateDiagnosticStrictly("router", DiagnosticWarning, DiagnosticRecovered, parseDetails) {
		parseT.Fatal("expected source mismatch to skip escalation")
	}
	if shouldEscalateDiagnosticStrictly("runtime", DiagnosticWarning, DiagnosticRecovered, diagnosticDetails{Code: "OTHER", Recoverable: true}) {
		parseT.Fatal("expected code mismatch to skip escalation")
	}
	if shouldEscalateDiagnosticStrictly("runtime", DiagnosticWarning, DiagnosticCorrectness, parseDetails) {
		parseT.Fatal("expected classification mismatch to skip escalation")
	}
	if !shouldEscalateDiagnosticStrictly("runtime", DiagnosticWarning, DiagnosticRecovered, parseDetails) {
		parseT.Fatal("expected matching strict diagnostic to escalate")
	}
}

// TestRuntimeHotReloadHelperBranches covers label and fallback reasoning helpers used for remount diagnostics.
func TestRuntimeHotReloadHelperBranches(parseT *testing.T) {
	if parseGot := hotReloadHookKindsSummary(nil); parseGot != "no hooks" {
		parseT.Fatalf("expected empty hook summary, got %q", parseGot)
	}
	if parseGot2 := hotReloadHookKindsSummary([]string{"state", "memo"}); parseGot2 != "state > memo" {
		parseT.Fatalf("expected joined hook summary, got %q", parseGot2)
	}

	parseCurrentFiber := &Fiber{
		typeOf: NewComponentType("stable-id", "CurrentWidget", "stable-id", nil, nil),
		props:  map[string]interface{}{"key": "stable-key"},
		hooks:  &Hooks{signature: []string{"state"}},
	}
	parseSnapshot := &HotReloadComponentSnapshot{
		Signature: ComponentSignature{
			Kind:          "component",
			Name:          "SnapshotWidget",
			QualifiedName: "stable-id",
			Key:           "stable-key",
			HookKinds:     []string{"state"},
		},
	}

	if parseLabel := hotReloadComponentLabel(parseSnapshot, parseCurrentFiber); parseLabel != "CurrentWidget" {
		parseT.Fatalf("expected current component name label, got %q", parseLabel)
	}
	if parseLabel2 := hotReloadComponentLabel(&HotReloadComponentSnapshot{
		Signature: ComponentSignature{QualifiedName: "example/Snapshot"},
	}, &Fiber{typeOf: "div"}); parseLabel2 != "example/Snapshot" {
		parseT.Fatalf("expected snapshot qualified-name label fallback, got %q", parseLabel2)
	}
	if parseLabel3 := hotReloadComponentLabel(&HotReloadComponentSnapshot{}, &Fiber{typeOf: "div"}); parseLabel3 != "component" {
		parseT.Fatalf("expected default component label, got %q", parseLabel3)
	}

	if parseReason := hotReloadFallbackReason(nil, parseCurrentFiber); parseReason != "saved state could not be matched" {
		parseT.Fatalf("expected missing snapshot fallback reason, got %q", parseReason)
	}
	if parseReason2 := hotReloadFallbackReason(parseSnapshot, &Fiber{typeOf: "div"}); parseReason2 != "current component identity could not be inspected" {
		parseT.Fatalf("expected non-component identity failure, got %q", parseReason2)
	}

	parseKindSnapshot := *parseSnapshot
	parseKindSnapshot.Signature.Kind = "host"
	if parseReason3 := hotReloadFallbackReason(&parseKindSnapshot, parseCurrentFiber); !strings.Contains(parseReason3, "component kind changed") {
		parseT.Fatalf("expected kind-change fallback reason, got %q", parseReason3)
	}

	parseIdentitySnapshot := *parseSnapshot
	parseIdentitySnapshot.Signature.QualifiedName = "other-id"
	if parseReason4 := hotReloadFallbackReason(&parseIdentitySnapshot, parseCurrentFiber); !strings.Contains(parseReason4, "component identity changed") {
		parseT.Fatalf("expected identity-change fallback reason, got %q", parseReason4)
	}

	parseKeySnapshot := *parseSnapshot
	parseKeySnapshot.Signature.Key = "other-key"
	if parseReason5 := hotReloadFallbackReason(&parseKeySnapshot, parseCurrentFiber); !strings.Contains(parseReason5, "component key changed") {
		parseT.Fatalf("expected key-change fallback reason, got %q", parseReason5)
	}

	parseHookCountSnapshot := *parseSnapshot
	parseCurrentFiber.hooks.signature = []string{"state", "memo"}
	if parseReason6 := hotReloadFallbackReason(&parseHookCountSnapshot, parseCurrentFiber); !strings.Contains(parseReason6, "hook count changed from 1 to 2") {
		parseT.Fatalf("expected hook-count fallback reason, got %q", parseReason6)
	}

	parseHookOrderSnapshot := *parseSnapshot
	parseHookOrderSnapshot.Signature.HookKinds = []string{"state", "memo"}
	parseCurrentFiber.hooks.signature = []string{"memo", "state"}
	if parseReason7 := hotReloadFallbackReason(&parseHookOrderSnapshot, parseCurrentFiber); !strings.Contains(parseReason7, "hook order changed") {
		parseT.Fatalf("expected hook-order fallback reason, got %q", parseReason7)
	}

	parseCurrentFiber.typeOf = NewComponentType("stable-id", "CurrentWidgetRenamed", "stable-id", nil, nil)
	parseCurrentFiber.hooks.signature = []string{"state"}
	if parseReason8 := hotReloadFallbackReason(parseSnapshot, parseCurrentFiber); !strings.Contains(parseReason8, "component signature changed") {
		parseT.Fatalf("expected summary fallback reason, got %q", parseReason8)
	}
}

// TestRuntimeSSRHelperBranches covers error-boundary SSR fallbacks, prop conversion, and typed memo restoration.
func TestRuntimeSSRHelperBranches(parseT *testing.T) {
	if parseErr := renderErrorBoundaryToString(&strings.Builder{}, nil); parseErr != nil {
		parseT.Fatalf("expected nil boundary render to succeed, got %v", parseErr)
	}

	parseBoundaryBuilder := &strings.Builder{}
	parseBoundaryError := ""
	parseBoundaryElement := &Element{
		Type: NewErrorBoundaryType(),
		Props: map[string]interface{}{
			"onError": func(parseErr error) {
				parseBoundaryError = parseErr.Error()
			},
			"errorFallback": func(parseErr error, parseReset func()) *Element {
				return CreateElement("span", nil, "handled:"+parseErr.Error())
			},
		},
		Children: []interface{}{
			CreateElement(func() *Element {
				panic("boom")
			}, nil),
		},
	}
	if parseErr := renderErrorBoundaryToString(parseBoundaryBuilder, parseBoundaryElement); parseErr != nil {
		parseT.Fatalf("expected error-boundary fallback render to recover, got %v", parseErr)
	}
	if parseBoundaryError != "boom" {
		parseT.Fatalf("expected onError callback to receive panic, got %q", parseBoundaryError)
	}
	if parseBoundaryBuilder.String() != "<span>handled:boom</span>" {
		parseT.Fatalf("unexpected fallback markup: %q", parseBoundaryBuilder.String())
	}

	parseElementFallbackBuilder := &strings.Builder{}
	parseElementFallbackCalled := false
	parseElementFallbackBoundary := &Element{
		Type: NewErrorBoundaryType(),
		Props: map[string]interface{}{
			"onError": func(parseErr error) {
				parseElementFallbackCalled = true
				panic("ignore onError panic")
			},
			"fallback": CreateElement("strong", nil, "static fallback"),
		},
		Children: []interface{}{
			CreateElement(func() *Element {
				panic("fallback boom")
			}, nil),
		},
	}
	if parseErr := renderErrorBoundaryToString(parseElementFallbackBuilder, parseElementFallbackBoundary); parseErr != nil {
		parseT.Fatalf("expected fallback element render to recover, got %v", parseErr)
	}
	if !parseElementFallbackCalled {
		parseT.Fatal("expected onError callback to run before element fallback render")
	}
	if parseElementFallbackBuilder.String() != "<strong>static fallback</strong>" {
		parseT.Fatalf("unexpected element fallback markup: %q", parseElementFallbackBuilder.String())
	}

	parseNoFallbackBuilder := &strings.Builder{}
	isParseOnErrorCalled := false
	parseNoFallbackBoundary := &Element{
		Type: NewErrorBoundaryType(),
		Props: map[string]interface{}{
			"onError": func(parseErr error) {
				isParseOnErrorCalled = true
			},
		},
		Children: []interface{}{
			CreateElement(func() *Element {
				panic("plain boom")
			}, nil),
		},
	}
	if parseErr := renderErrorBoundaryToString(parseNoFallbackBuilder, parseNoFallbackBoundary); parseErr != nil {
		parseT.Fatalf("expected missing fallback branch to swallow panic, got %v", parseErr)
	}
	if !isParseOnErrorCalled {
		parseT.Fatal("expected onError callback to run for missing-fallback branch")
	}
	if parseNoFallbackBuilder.String() != "" {
		parseT.Fatalf("expected missing-fallback boundary to render nothing, got %q", parseNoFallbackBuilder.String())
	}

	type parseNamedProps map[string]interface{}

	parseZeroArg, parseErr := buildComponentArg(reflect.TypeOf(struct{ ID string }{}), nil)
	if parseErr != nil {
		parseT.Fatalf("expected nil props to produce zero value, got %v", parseErr)
	}
	if !parseZeroArg.IsZero() {
		parseT.Fatalf("expected nil props zero value, got %#v", parseZeroArg.Interface())
	}

	parseAttrsArg, parseErr := buildComponentArg(reflect.TypeOf(Attrs{}), map[string]interface{}{"id": "hero"})
	if parseErr != nil {
		parseT.Fatalf("expected Attrs prop conversion to succeed, got %v", parseErr)
	}
	if parseAttrsArg.Interface().(Attrs)["id"] != "hero" {
		parseT.Fatalf("expected Attrs prop passthrough, got %#v", parseAttrsArg.Interface())
	}

	parseAssignableArg, parseErr := buildComponentArg(reflect.TypeOf(map[string]interface{}{}), map[string]interface{}{"id": "hero"})
	if parseErr != nil {
		parseT.Fatalf("expected assignable map conversion to succeed, got %v", parseErr)
	}
	if parseAssignableArg.Interface().(Attrs)["id"] != "hero" {
		parseT.Fatalf("expected assignable prop value to preserve map contents, got %#v", parseAssignableArg.Interface())
	}

	parseConvertibleArg, parseErr := buildComponentArg(reflect.TypeOf(parseNamedProps{}), map[string]interface{}{"id": "hero"})
	if parseErr != nil {
		parseT.Fatalf("expected convertible map conversion to succeed, got %v", parseErr)
	}
	if parseConvertibleArg.Interface().(parseNamedProps)["id"] != "hero" {
		parseT.Fatalf("expected convertible prop value to preserve map contents, got %#v", parseConvertibleArg.Interface())
	}

	if _, parseErr := buildComponentArg(reflect.TypeOf(struct{ ID string }{}), map[string]interface{}{"id": "hero"}); parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported component prop type") {
		parseT.Fatalf("expected unsupported prop type error, got %v", parseErr)
	}

	parseMemoFiber := &Fiber{typeOf: "memo"}
	parseMemoFiber.hooks = &Hooks{
		owner: parseMemoFiber,
		hotReloadRestore: &HotReloadComponentSnapshot{
			Memos: []HotReloadMemoSnapshot{{
				Value: float64(9),
				Deps:  []interface{}{"same"},
			}},
		},
	}
	SetCurrentFiber(parseMemoFiber)
	defer SetCurrentFiber(nil)

	parseMemoComputeCount := 0
	parseMemoValue := GoUseMemoTyped(func() interface{} {
		parseMemoComputeCount++
		return 12
	}, reflect.TypeOf(int(0)), "same")
	if parseMemoComputeCount != 0 {
		parseT.Fatalf("expected restored typed memo to skip recompute, got %d calls", parseMemoComputeCount)
	}
	if parseMemoInt, parseOk := parseMemoValue.(int); !parseOk || parseMemoInt != 9 {
		parseT.Fatalf("expected restored typed memo coercion to int, got %#v", parseMemoValue)
	}
}

// TestRuntimeResolveContainerTransitionAndAtomBranches covers direct container resolution, transition guards, and atom notification edge cases.
func TestRuntimeResolveContainerTransitionAndAtomBranches(parseT *testing.T) {
	var parseNilRuntime *Runtime
	if parseNilRuntime.resolveContainer("app") != nil {
		parseT.Fatal("expected nil runtime to resolve no container")
	}

	parseAdapter := buildResolveContainerTestDOMAdapter()
	parseResolvedNode := parseAdapter.CreateElement("section")
	parseAdapter.resolveByKey["#app"] = parseResolvedNode
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	if parseGot := parseRt.resolveContainer(nil); parseGot != nil {
		parseT.Fatalf("expected nil target resolution to return nil, got %#v", parseGot)
	}
	if parseGot2 := parseRt.resolveContainer(parseResolvedNode); parseGot2 != parseResolvedNode {
		parseT.Fatal("expected direct DOM node resolution to pass through unchanged")
	}
	if parseGot3 := parseRt.resolveContainer("#app"); parseGot3 != parseResolvedNode {
		parseT.Fatal("expected adapter ResolveNode branch to resolve configured string target")
	}
	if parseGot4 := parseRt.resolveContainer("#missing"); parseGot4 != nil {
		parseT.Fatalf("expected unresolved target to return nil, got %#v", parseGot4)
	}

	isParseTransitionCalled := false
	parseNilRuntime.StartTransition(func() {
		isParseTransitionCalled = true
	})
	if !isParseTransitionCalled {
		parseT.Fatal("expected nil runtime StartTransition to run callback immediately")
	}

	parseRt.StartTransition(nil)
	parseRt.atomRegistry = nil
	parseRt.setTransitionPending(true)

	parseRt.atomRegistry = NewAtomRegistry()
	parseTransitionFiber := &Fiber{typeOf: "transition-subscriber"}
	parseRt.atomRegistry.Subscribe(transitionPendingAtomID, parseTransitionFiber)
	parseRt.setTransitionPending(true)
	if parseValue, parseOk := parseRt.GetAtomValue(transitionPendingAtomID); !parseOk || parseValue != true {
		parseT.Fatalf("expected transition pending atom update, got value=%#v ok=%t", parseValue, parseOk)
	}

	parseRegistry := NewAtomRegistry()
	parseRegistry.setAtomAndNotify("count", 3, nil)
	if parseValue2, parseOk2 := parseRegistry.GetAtom("count"); !parseOk2 || parseValue2 != 3 {
		parseT.Fatalf("expected nil-notify setAtomAndNotify to delegate to SetAtom, got value=%#v ok=%t", parseValue2, parseOk2)
	}

	parseRegistry.InitAtom("count", 1)
	parseSharedFiber := &Fiber{typeOf: "shared"}
	parseRegistry.Subscribe("count", parseSharedFiber)
	parseRegistry.Subscribe("derived", parseSharedFiber)
	if parseErr := parseRegistry.RegisterDerivedAtom("derived", []string{"count"}, func() interface{} {
		parseValue3, _ := parseRegistry.GetAtom("count")
		return parseValue3.(int) + 1
	}); parseErr != nil {
		parseT.Fatalf("expected derived atom registration to succeed, got %v", parseErr)
	}

	parseNotifications := make([]string, 0, 1)
	parseRegistry.setAtomAndNotify("count", 2, func(parseFiber *Fiber) {
		parseNotifications = append(parseNotifications, parseFiber.typeOf.(string))
	})
	if len(parseNotifications) != 1 || parseNotifications[0] != "shared" {
		parseT.Fatalf("expected deduplicated notifications across direct and derived subscribers, got %#v", parseNotifications)
	}
	if parseDerivedValue, parseOk3 := parseRegistry.GetAtom("derived"); !parseOk3 || parseDerivedValue != 3 {
		parseT.Fatalf("expected derived atom recompute during notification, got value=%#v ok=%t", parseDerivedValue, parseOk3)
	}

	ClearDiagnostics()
	defer ClearDiagnostics()
	parseCycleRegistry := NewAtomRegistry()
	parseCycleRegistry.InitAtom("count", 1)
	parseCycleRegistry.derived["loop"] = derivedAtom{
		deps:    []string{"count"},
		compute: func() interface{} { return 2 },
		active:  true,
	}
	parseCycleRegistry.dependents["count"] = map[string]bool{"loop": true}
	parseCycleRegistry.dependents["loop"] = map[string]bool{"loop": true}
	parseCycleRegistry.setAtomAndNotify("count", 5, func(parseFiber *Fiber) {})
	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 || !strings.Contains(parseDiagnostics[len(parseDiagnostics)-1].Message, "derived atom cycle detected involving loop") {
		parseT.Fatalf("expected cycle diagnostic from setAtomAndNotify, got %+v", parseDiagnostics)
	}
}
