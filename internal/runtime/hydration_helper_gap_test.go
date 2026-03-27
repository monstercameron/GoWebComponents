package runtime

import (
	"reflect"
	"strings"
	"testing"
)

// TestHydrationHelperValueBranches covers normalization and equality helpers used during hydration comparisons.
func TestHydrationHelperValueBranches(parseT *testing.T) {
	if parseValue, parseOk := normalizeHydrationInt(7); !parseOk || parseValue != 7 {
		parseT.Fatalf("normalizeHydrationInt(int) = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt(int32(8)); !parseOk || parseValue != 8 {
		parseT.Fatalf("normalizeHydrationInt(int32) = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt(int64(9)); !parseOk || parseValue != 9 {
		parseT.Fatalf("normalizeHydrationInt(int64) = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt(float64(10)); !parseOk || parseValue != 10 {
		parseT.Fatalf("normalizeHydrationInt(float64) = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt(float32(11)); !parseOk || parseValue != 11 {
		parseT.Fatalf("normalizeHydrationInt(float32) = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt("1"); !parseOk || parseValue != 1 {
		parseT.Fatalf("normalizeHydrationInt(\"1\") = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt("element"); !parseOk || parseValue != 1 {
		parseT.Fatalf("normalizeHydrationInt(\"element\") = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt("3"); !parseOk || parseValue != 3 {
		parseT.Fatalf("normalizeHydrationInt(\"3\") = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt("text"); !parseOk || parseValue != 3 {
		parseT.Fatalf("normalizeHydrationInt(\"text\") = %d, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationInt("bad"); parseOk || parseValue != 0 {
		parseT.Fatalf("normalizeHydrationInt(\"bad\") = %d, %t", parseValue, parseOk)
	}

	if parseValue, parseOk := normalizeHydrationString("hero"); !parseOk || parseValue != "hero" {
		parseT.Fatalf("normalizeHydrationString(string) = %q, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationString([]byte("bytes")); !parseOk || parseValue != "bytes" {
		parseT.Fatalf("normalizeHydrationString([]byte) = %q, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationString(42); !parseOk || parseValue != "42" {
		parseT.Fatalf("normalizeHydrationString(int) = %q, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := normalizeHydrationString(nil); parseOk || parseValue != "" {
		parseT.Fatalf("normalizeHydrationString(nil) = %q, %t", parseValue, parseOk)
	}

	if parseValue := stringifyHydrationValue(nil); parseValue != "<nil>" {
		parseT.Fatalf("stringifyHydrationValue(nil) = %q", parseValue)
	}
	if parseValue := stringifyHydrationValue("label"); parseValue != "label" {
		parseT.Fatalf("stringifyHydrationValue(string) = %q", parseValue)
	}
	if parseValue := stringifyHydrationValue(true); parseValue != "true" {
		parseT.Fatalf("stringifyHydrationValue(true) = %q", parseValue)
	}
	if parseValue := stringifyHydrationValue(false); parseValue != "false" {
		parseT.Fatalf("stringifyHydrationValue(false) = %q", parseValue)
	}
	if parseValue := stringifyHydrationValue(12); parseValue != "12" {
		parseT.Fatalf("stringifyHydrationValue(int) = %q", parseValue)
	}

	if !hydrationValuesEqual("checked", true, "true") {
		parseT.Fatal("expected true hydration bool to accept literal true")
	}
	if !hydrationValuesEqual("checked", true, "checked") {
		parseT.Fatal("expected true hydration bool to accept attribute-name form")
	}
	if !hydrationValuesEqual("checked", true, "") {
		parseT.Fatal("expected true hydration bool to accept empty attribute form")
	}
	if !hydrationValuesEqual("checked", false, "false") {
		parseT.Fatal("expected false hydration bool to accept literal false")
	}
	if !hydrationValuesEqual("checked", false, "<nil>") {
		parseT.Fatal("expected false hydration bool to accept nil marker")
	}
	if !hydrationValuesEqual("checked", false, "") {
		parseT.Fatal("expected false hydration bool to accept empty attribute form")
	}
	if !hydrationValuesEqual("data-count", 12, "12") {
		parseT.Fatal("expected numeric hydration comparison to stringify the client value")
	}
	if hydrationValuesEqual("data-count", 12, "9") {
		parseT.Fatal("expected mismatched numeric hydration comparison to fail")
	}
}

// TestHydrationHelperDOMBranches covers DOM-node adapters and fiber labels used by hydration diagnostics.
func TestHydrationHelperDOMBranches(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := &Runtime{domAdapter: parseAdapter}

	parseParent := parseAdapter.CreateElement("div")
	parseWhitespace := parseAdapter.CreateTextNode("   ")
	parseLabel := parseAdapter.CreateElement("label")
	parseText := parseAdapter.CreateTextNode("hello")
	parseAdapter.SetAttribute(parseLabel, "class", "hero")
	parseAdapter.SetAttribute(parseLabel, "for", "email")
	parseAdapter.SetAttribute(parseLabel, "data-id", "7")
	parseAdapter.AppendChild(parseParent, parseWhitespace)
	parseAdapter.AppendChild(parseParent, parseLabel)
	parseAdapter.AppendChild(parseParent, parseText)

	if parseCandidate := parseRt.nextHydrationCandidate(parseWhitespace); parseCandidate != parseLabel {
		parseT.Fatalf("nextHydrationCandidate() = %#v, want %#v", parseCandidate, parseLabel)
	}
	if !parseRt.isIgnorableHydrationNode(parseWhitespace) {
		parseT.Fatal("expected whitespace text node to be ignorable during hydration")
	}
	if parseRt.isIgnorableHydrationNode(parseLabel) {
		parseT.Fatal("expected element node to be non-ignorable during hydration")
	}

	if parseValue, parseOk := parseRt.readHydrationComparableValue(parseLabel, "class"); !parseOk || parseValue != "hero" {
		parseT.Fatalf("readHydrationComparableValue(class) = %q, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := parseRt.readHydrationComparableValue(parseLabel, "htmlFor"); !parseOk || parseValue != "email" {
		parseT.Fatalf("readHydrationComparableValue(htmlFor) = %q, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := parseRt.readHydrationComparableValue(parseLabel, "data-id"); !parseOk || parseValue != "7" {
		parseT.Fatalf("readHydrationComparableValue(data-id) = %q, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := parseRt.readHydrationComparableValue(parseLabel, "missing"); parseOk || parseValue != "" {
		parseT.Fatalf("readHydrationComparableValue(missing) = %q, %t", parseValue, parseOk)
	}
	if parseValue, parseOk := (&Runtime{}).readHydrationComparableValue(parseLabel, "class"); parseOk || parseValue != "" {
		parseT.Fatalf("readHydrationComparableValue(nil adapter) = %q, %t", parseValue, parseOk)
	}

	if parseValue := parseRt.domNodeType(parseText); parseValue != 3 {
		parseT.Fatalf("domNodeType(text) = %d", parseValue)
	}
	if parseValue := parseRt.domNodeType(parseLabel); parseValue != 1 {
		parseT.Fatalf("domNodeType(element) = %d", parseValue)
	}
	if parseValue := parseRt.domNodeTag(parseLabel); parseValue != "LABEL" {
		parseT.Fatalf("domNodeTag() = %q", parseValue)
	}
	if parseValue := parseRt.domNodeText(parseText); parseValue != "hello" {
		parseT.Fatalf("domNodeText() = %q", parseValue)
	}
	if parseValue := parseRt.describeHydrationNode(nil); parseValue != "null" {
		parseT.Fatalf("describeHydrationNode(nil) = %q", parseValue)
	}
	if parseValue := parseRt.describeHydrationNode(parseText); parseValue != `text("hello")` {
		parseT.Fatalf("describeHydrationNode(text) = %q", parseValue)
	}
	if parseValue := parseRt.describeHydrationNode(parseLabel); parseValue != "<label>" {
		parseT.Fatalf("describeHydrationNode(element) = %q", parseValue)
	}

	parseHostFiber := &Fiber{typeOf: "label"}
	parseTextFiber := &Fiber{typeOf: "TEXT_ELEMENT", textContent: "hello"}
	parseFragmentFiber := &Fiber{typeOf: "FRAGMENT"}
	parseReactiveTextFiber := &Fiber{typeOf: ReactiveTextNodeType, textContent: "hello"}
	parseReactiveRegionFiber := &Fiber{typeOf: ReactiveRegionNodeType}
	if !parseRt.matchesHydrationNode(parseHostFiber, parseLabel) {
		parseT.Fatal("expected host fiber to match label node")
	}
	if parseRt.matchesHydrationNode(parseHostFiber, parseText) {
		parseT.Fatal("expected host fiber not to match text node")
	}
	if !parseRt.matchesHydrationNode(parseTextFiber, parseText) {
		parseT.Fatal("expected text fiber to match text node")
	}
	if !parseRt.matchesHydrationNode(parseReactiveTextFiber, parseText) {
		parseT.Fatal("expected reactive text fiber to match text node")
	}
	if parseRt.matchesHydrationNode(parseReactiveRegionFiber, parseLabel) {
		parseT.Fatal("expected reactive region fiber to force fallback matching")
	}
	if parseRt.matchesHydrationNode(parseFragmentFiber, parseLabel) {
		parseT.Fatal("expected fragment fiber not to match a host node")
	}

	if parseName := expectedHydrationFiberName(nil); parseName != "" {
		parseT.Fatalf("expectedHydrationFiberName(nil) = %q", parseName)
	}
	if parseName := expectedHydrationFiberName(parseReactiveRegionFiber); parseName != "reactive region" {
		parseT.Fatalf("expectedHydrationFiberName(reactive region) = %q", parseName)
	}
	if parseName := expectedHydrationFiberName(parseReactiveTextFiber); parseName != "reactive text node" {
		parseT.Fatalf("expectedHydrationFiberName(reactive text) = %q", parseName)
	}
	if parseName := expectedHydrationFiberName(parseTextFiber); parseName != "text node" {
		parseT.Fatalf("expectedHydrationFiberName(text) = %q", parseName)
	}
	if parseName := expectedHydrationFiberName(parseFragmentFiber); parseName != "fragment" {
		parseT.Fatalf("expectedHydrationFiberName(fragment) = %q", parseName)
	}
	if parseName := expectedHydrationFiberName(parseHostFiber); parseName != "<label>" {
		parseT.Fatalf("expectedHydrationFiberName(host) = %q", parseName)
	}
}

// TestRuntimeFlushDeferredHydrationUpdateBranches covers deferred hydration scheduling and queue clearing.
func TestRuntimeFlushDeferredHydrationUpdateBranches(parseT *testing.T) {
	var parseNilRuntime *Runtime
	parseNilRuntime.flushDeferredHydrationUpdates()

	parseScheduler := newTestScheduler()
	parseFiber := &Fiber{typeOf: "section"}
	parseRoot := &Fiber{typeOf: "ROOT", child: parseFiber, dom: newTestDOMAdapter().CreateElement("div")}
	parseFiber.parent = parseRoot
	parseRt := &Runtime{
		scheduler:                parseScheduler,
		currentRoot:              parseRoot,
		deferredHydrationUpdates: map[*Fiber]bool{parseFiber: true},
	}

	parseRt.flushDeferredHydrationUpdates()

	if len(parseRt.deferredHydrationUpdates) != 0 {
		parseT.Fatalf("expected deferred hydration updates to be cleared, got %d", len(parseRt.deferredHydrationUpdates))
	}
	if !parseFiber.dirty || !parseFiber.needsUpdate || parseFiber.updateOrigin != "hydration" {
		parseT.Fatalf("expected deferred hydration fiber to be scheduled, got %+v", parseFiber)
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected hydration flush to schedule one timeout, got %d", len(parseScheduler.timeouts))
	}

	parseEmptyRuntime := &Runtime{}
	parseEmptyRuntime.flushDeferredHydrationUpdates()
}

// TestReconcilerKeyHelperBranches covers comparable-key extraction, fallback keyed matching, and type matching.
func TestReconcilerKeyHelperBranches(parseT *testing.T) {
	if sameFiberType(nil, &Fiber{typeOf: "div"}) {
		parseT.Fatal("expected nil element type comparison to fail")
	}
	if !sameFiberType(&Element{Type: "div"}, &Fiber{typeOf: "div"}) {
		parseT.Fatal("expected matching host types to compare equal")
	}
	if sameFiberType(&Element{Type: "div"}, &Fiber{typeOf: "span"}) {
		parseT.Fatal("expected mismatched host types to compare unequal")
	}

	parseComponent := NewComponentType("example/Widget", "Widget", "example/Widget", nil, nil)
	if !sameFiberType(&Element{Type: parseComponent}, &Fiber{typeOf: NewComponentType("example/Widget", "WidgetRenamed", "example/Widget", nil, nil)}) {
		parseT.Fatal("expected matching component identities to compare equal")
	}
	parseComponentFunc := func() *Element { return nil }
	if !sameFiberType(&Element{Type: parseComponentFunc}, &Fiber{typeOf: parseComponentFunc}) {
		parseT.Fatal("expected matching function component identities to compare equal")
	}

	if parseKey, parseOk := propsComparableKey(map[string]interface{}{"key": "hero"}); !parseOk || parseKey != "hero" {
		parseT.Fatalf("propsComparableKey(string) = %#v, %t", parseKey, parseOk)
	}
	if parseKey, parseOk := propsComparableKey(map[string]interface{}{"key": true}); !parseOk || parseKey != true {
		parseT.Fatalf("propsComparableKey(bool) = %#v, %t", parseKey, parseOk)
	}
	parseElementKey := &Element{Type: "div"}
	if parseKey, parseOk := propsComparableKey(map[string]interface{}{"key": parseElementKey}); !parseOk || parseKey != parseElementKey {
		parseT.Fatalf("propsComparableKey(*Element) = %#v, %t", parseKey, parseOk)
	}
	if parseKey, parseOk := propsComparableKey(map[string]interface{}{"key": []int{1, 2}}); parseOk || parseKey != nil {
		parseT.Fatalf("propsComparableKey([]int) = %#v, %t", parseKey, parseOk)
	}
	if parseKey, parseOk := elementComparableKey(&Element{Props: map[string]interface{}{"key": "card"}}); !parseOk || parseKey != "card" {
		parseT.Fatalf("elementComparableKey() = %#v, %t", parseKey, parseOk)
	}
	if parseKey, parseOk := fiberComparableKey(&Fiber{props: map[string]interface{}{"key": "card"}}); !parseOk || parseKey != "card" {
		parseT.Fatalf("fiberComparableKey() = %#v, %t", parseKey, parseOk)
	}

	parseOldA := &Fiber{props: map[string]interface{}{"key": "a"}}
	parseOldB := &Fiber{props: map[string]interface{}{"key": "b"}}
	parseOldFibers := []*Fiber{parseOldA, parseOldB}
	parseMatched := takeMatchingFallbackKeyed(parseOldFibers, &Element{Props: map[string]interface{}{"key": "b"}})
	if parseMatched != parseOldB {
		parseT.Fatalf("takeMatchingFallbackKeyed() = %#v, want %#v", parseMatched, parseOldB)
	}
	if parseOldFibers[1] != nil {
		parseT.Fatalf("expected matched fallback slot to be cleared, got %#v", parseOldFibers[1])
	}
	if parseMatched2 := takeMatchingFallbackKeyed(parseOldFibers, &Element{Props: map[string]interface{}{"key": "missing"}}); parseMatched2 != nil {
		parseT.Fatalf("expected missing fallback key to return nil, got %#v", parseMatched2)
	}
	if parseMatched3 := takeMatchingFallbackKeyed(nil, &Element{Props: map[string]interface{}{"key": "a"}}); parseMatched3 != nil {
		parseT.Fatalf("expected nil fallback list to return nil, got %#v", parseMatched3)
	}
}

// TestSSRAndHotReloadHelperGapBranches covers internal SSR branches and hot-reload compatibility guards.
func TestSSRAndHotReloadHelperGapBranches(parseT *testing.T) {
	parseBuilder := &strings.Builder{}
	parseProviderElement := &Element{
		Type:     NewContextProviderType(NewContextDescriptor("default")),
		Children: []interface{}{CreateElement("span", nil, "provider")},
	}
	if parseErr := renderElementToString(parseBuilder, parseProviderElement); parseErr != nil {
		parseT.Fatalf("renderElementToString(context provider): %v", parseErr)
	}
	if parseBuilder.String() != "<span>provider</span>" {
		parseT.Fatalf("unexpected provider markup: %q", parseBuilder.String())
	}

	parseBuilder.Reset()
	parsePortalElement := &Element{
		Type:     PortalNodeType,
		Children: []interface{}{CreateElement("span", nil, "portal")},
	}
	if parseErr := renderElementToString(parseBuilder, parsePortalElement); parseErr != nil {
		parseT.Fatalf("renderElementToString(portal): %v", parseErr)
	}
	if parseBuilder.String() != "<span>portal</span>" {
		parseT.Fatalf("unexpected portal markup: %q", parseBuilder.String())
	}

	parseBuilder.Reset()
	parseReactiveTextFallback := &Element{Type: ReactiveTextNodeType, TextContent: "fallback"}
	if parseErr := renderElementToString(parseBuilder, parseReactiveTextFallback); parseErr != nil {
		parseT.Fatalf("renderElementToString(reactive text fallback): %v", parseErr)
	}
	if parseBuilder.String() != "fallback" {
		parseT.Fatalf("unexpected reactive text fallback markup: %q", parseBuilder.String())
	}

	parseBuilder.Reset()
	parseReactiveRegionElement := &Element{
		Type:  ReactiveRegionNodeType,
		Props: map[string]interface{}{reactiveRegionRenderProp: func() *Element { return CreateElement("em", nil, "region") }},
	}
	if parseErr := renderElementToString(parseBuilder, parseReactiveRegionElement); parseErr != nil {
		parseT.Fatalf("renderElementToString(reactive region): %v", parseErr)
	}
	if parseBuilder.String() != "<em>region</em>" {
		parseT.Fatalf("unexpected reactive region markup: %q", parseBuilder.String())
	}

	parseBuilder.Reset()
	if parseErr := renderChildrenToString(parseBuilder, []interface{}{nil, "text", 7}); parseErr != nil {
		parseT.Fatalf("renderChildrenToString(): %v", parseErr)
	}
	if parseBuilder.String() != "text7" {
		parseT.Fatalf("unexpected child markup: %q", parseBuilder.String())
	}

	parseUnsupportedReturn := func() string { return "bad" }
	if _, parseErr := resolveComponentElement(&Element{Type: parseUnsupportedReturn}); parseErr == nil || !strings.Contains(parseErr.Error(), "must return *runtime.Element") {
		parseT.Fatalf("expected unsupported return type error, got %v", parseErr)
	}
	parseUnsupportedArity := func(parseLeft string, parseRight string) *Element { return nil }
	if _, parseErr := resolveComponentElement(&Element{Type: parseUnsupportedArity}); parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported arity") {
		parseT.Fatalf("expected unsupported arity error, got %v", parseErr)
	}
	if _, parseErr := resolveComponentElement(&Element{Type: 42}); parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported element type") {
		parseT.Fatalf("expected unsupported element type error, got %v", parseErr)
	}

	var parseNilRuntime *Runtime
	if parseNilRuntime.HasPendingHotReloadSnapshot() {
		parseT.Fatal("expected nil runtime to report no pending hot reload snapshot")
	}
	parseSelectiveRuntime := &Runtime{pendingHotReloadSelective: true}
	if parseSelectiveRuntime.HasPendingHotReloadSnapshot() {
		parseT.Fatal("expected selective hot reload runtime without paths to report no pending snapshot")
	}
	parseSelectiveRuntime.pendingHotReloadByPath = map[string]HotReloadComponentSnapshot{"ROOT > Widget": {}}
	if !parseSelectiveRuntime.HasPendingHotReloadSnapshot() {
		parseT.Fatal("expected selective hot reload runtime with paths to report pending snapshot")
	}

	parseSnapshot := &HotReloadComponentSnapshot{
		Signature: ComponentSignature{
			Kind:          "component",
			QualifiedName: "example/Widget",
			Key:           "hero",
			HookKinds:     []string{"state"},
		},
	}
	parseFiber := &Fiber{
		typeOf: NewComponentType("example/Widget", "Widget", "example/Widget", nil, nil),
		props:  map[string]interface{}{"key": "hero"},
		hooks:  &Hooks{signature: []string{"state"}},
	}
	if !componentSnapshotCompatible(parseSnapshot, parseFiber) {
		parseT.Fatal("expected matching component snapshot to be compatible")
	}
	if !componentSnapshotFullyCompatible(parseSnapshot, parseFiber) {
		parseT.Fatal("expected matching component snapshot to be fully compatible")
	}
	if componentSnapshotCompatible(nil, parseFiber) {
		parseT.Fatal("expected nil snapshot to be incompatible")
	}
	if componentSnapshotCompatible(parseSnapshot, &Fiber{typeOf: "div"}) {
		parseT.Fatal("expected host fiber to be incompatible with component snapshot")
	}
	parseMismatchedSnapshot := *parseSnapshot
	parseMismatchedSnapshot.Signature.QualifiedName = "example/Other"
	if componentSnapshotCompatible(&parseMismatchedSnapshot, parseFiber) {
		parseT.Fatal("expected identity mismatch to be incompatible")
	}
	parseMismatchedFiber := &Fiber{
		typeOf: NewComponentType("example/Widget", "Widget", "example/Widget", nil, nil),
		props:  map[string]interface{}{"key": "hero"},
		hooks:  &Hooks{signature: []string{"state", "memo"}},
	}
	if componentSnapshotFullyCompatible(parseSnapshot, parseMismatchedFiber) {
		parseT.Fatal("expected hook signature mismatch to fail full compatibility")
	}
	if !reflect.DeepEqual(filterHotReloadSerializableKinds([]string{"state", "effect", "memo"}), []string{"state", "memo"}) {
		parseT.Fatalf("unexpected serializable hook kinds filtering")
	}
}
