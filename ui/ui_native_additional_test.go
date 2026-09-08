//go:build !js || !wasm

package ui

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

type nativeProps struct {
	Label string
}

func nativeComponent(parseProps nativeProps) Node {
	return Text("hello " + parseProps.Label)
}

func nativeZeroArgComponent() Node {
	return Text("zero")
}

func nativeMapPropsComponent(parseProps map[string]any) Node {
	parseLabel, _ := parseProps["label"].(string)
	return Text("map " + parseLabel)
}

func nativeAttrsPropsComponent(parseProps runtime.Attrs) Node {
	parseLabel, _ := parseProps["label"].(string)
	return Text("attrs " + parseLabel)
}

func TestNativeStateAndHookHelpers(parseT *testing.T) {
	var parseZeroState State[int]
	if parseZeroState.Get() != 0 {
		parseT.Fatalf("zero state Get() = %d, want 0", parseZeroState.Get())
	}
	parseState := UseState(1)
	parseState.Set(3)
	parseState.Update(func(parseCurrent int) int { return parseCurrent + 2 })
	if parseState.Get() != 5 {
		parseT.Fatalf("state.Get() = %d, want 5", parseState.Get())
	}

	var parseZeroRef Ref[string]
	if parseZeroRef.Get() != "" {
		parseT.Fatalf("zero ref Get() = %q, want empty", parseZeroRef.Get())
	}
	parseRef := UseRef("start")
	parseRef.Set("updated")
	if parseRef.Get() != "updated" {
		parseT.Fatalf("ref.Get() = %q, want updated", parseRef.Get())
	}

	var parseZeroReducer Reducer[int, int]
	if parseZeroReducer.Get() != 0 {
		parseT.Fatalf("zero reducer Get() = %d, want 0", parseZeroReducer.Get())
	}
	parseReducer := UseReducer(func(parseState2 int, parseAction int) int { return parseState2 + parseAction }, 2)
	parseReducer.Dispatch(4)
	if parseReducer.Get() != 6 {
		parseT.Fatalf("reducer.Get() = %d, want 6", parseReducer.Get())
	}

	var parseZeroPrevious Previous[string]
	if parseZeroPrevious.Ok() || parseZeroPrevious.Get() != "" {
		parseT.Fatalf("zero previous = (%q, %t), want empty/false", parseZeroPrevious.Get(), parseZeroPrevious.Ok())
	}
	parsePrevious := UsePrevious("value")
	if parsePrevious.Ok() || parsePrevious.Get() != "" {
		parseT.Fatalf("native previous should be empty, got (%q, %t)", parsePrevious.Get(), parsePrevious.Ok())
	}

	var parseZeroDebounced Debounced[string]
	if parseZeroDebounced.Get() != "" || parseZeroDebounced.Pending() {
		parseT.Fatalf("zero debounced = (%q, %t), want empty/false", parseZeroDebounced.Get(), parseZeroDebounced.Pending())
	}
	parseDebounced := UseDebounced("ready", time.Second)
	if parseDebounced.Get() != "ready" || parseDebounced.Pending() {
		parseT.Fatalf("debounced = (%q, %t), want ready/false", parseDebounced.Get(), parseDebounced.Pending())
	}

	var parseZeroThrottled Throttled[string]
	if parseZeroThrottled.Get() != "" || parseZeroThrottled.Pending() {
		parseT.Fatalf("zero throttled = (%q, %t), want empty/false", parseZeroThrottled.Get(), parseZeroThrottled.Pending())
	}
	parseThrottled := UseThrottled("steady", time.Second)
	if parseThrottled.Get() != "steady" || parseThrottled.Pending() {
		parseT.Fatalf("throttled = (%q, %t), want steady/false", parseThrottled.Get(), parseThrottled.Pending())
	}

	if parseGot := UseDeferredValue(42); parseGot != 42 {
		parseT.Fatalf("UseDeferredValue() = %d, want 42", parseGot)
	}
	if parseGot2 := UseMemo(func() int { return 7 }); parseGot2 != 7 {
		parseT.Fatalf("UseMemo() = %d, want 7", parseGot2)
	}
	if parseGot3 := UseMemo[int](nil); parseGot3 != 0 {
		parseT.Fatalf("UseMemo(nil) = %d, want 0", parseGot3)
	}
	if parseGot4 := UseCallback("handler"); parseGot4 != "handler" {
		parseT.Fatalf("UseCallback() = %q, want handler", parseGot4)
	}

	isParseRan := false
	parseTransition := UseTransition()
	if parseTransition.Pending() {
		parseT.Fatal("native transition should not be pending")
	}
	parseTransition.Start(func() { isParseRan = true })
	if !isParseRan {
		parseT.Fatal("transition.Start() did not run callback")
	}
	StartTransition(nil)

	parseFirstID := UseId()
	parseSecondID := UseId()
	if parseFirstID == parseSecondID {
		parseT.Fatal("UseId() should return distinct IDs")
	}

	if parseGot5 := UseEvent("wrapped").Value(); parseGot5 != "wrapped" {
		parseT.Fatalf("UseEvent().Value() = %#v, want wrapped", parseGot5)
	}
	if parseGot6 := WrapHandler(123).Value(); parseGot6 != 123 {
		parseT.Fatalf("WrapHandler().Value() = %#v, want 123", parseGot6)
	}
}

func TestNativeServerOnlyHelpersAndRenderComponent(parseT *testing.T) {
	parseRendered := renderComponent(nativeComponent, map[string]any{propsKey: nativeProps{Label: "gwc"}})
	parseMarkup, parseErr := RenderToString(parseRendered)
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderComponent) error = %v", parseErr)
	}
	if parseMarkup != "hello gwc" {
		parseT.Fatalf("rendered markup = %q, want hello gwc", parseMarkup)
	}

	parseMarkup, parseErr = RenderToString(CreateElement(nativeZeroArgComponent))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(CreateElement) error = %v", parseErr)
	}
	if parseMarkup != "zero" {
		parseT.Fatalf("CreateElement zero-arg markup = %q, want zero", parseMarkup)
	}

	parseMarkup, parseErr = RenderToString(renderComponent(nativeMapPropsComponent, map[string]any{propsKey: runtime.Attrs{"label": "fast"}}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderComponent map props) error = %v", parseErr)
	}
	if parseMarkup != "map fast" {
		parseT.Fatalf("map props markup = %q, want map fast", parseMarkup)
	}

	parseMarkup, parseErr = RenderToString(renderComponent(nativeAttrsPropsComponent, map[string]any{propsKey: map[string]any{"label": "path"}}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderComponent attrs props) error = %v", parseErr)
	}
	if parseMarkup != "attrs path" {
		parseT.Fatalf("attrs props markup = %q, want attrs path", parseMarkup)
	}

	parseNode := Text("passthrough")
	if CreateElement(parseNode) != parseNode {
		parseT.Fatal("CreateElement(node) should return the same node")
	}
	if renderComponent(nil, nil) != nil {
		parseT.Fatal("renderComponent(nil) should return nil")
	}

	parseAssertPanics := func(parseName string, parseFn func(), parseContains string) {
		parseT.Helper()
		defer func() {
			parseRecovered := recover()
			if parseRecovered == nil {
				parseT.Fatalf("%s: expected panic", parseName)
			}
			if parseContains != "" && !strings.Contains(parseRecovered.(string), parseContains) {
				parseT.Fatalf("%s: panic = %q, want substring %q", parseName, parseRecovered, parseContains)
			}
		}()
		parseFn()
	}

	parseAssertPanics("invalid component", func() {
		renderComponent(123, nil)
	}, "GWC-UI-CREATE-ELEMENT-TYPE")
	parseAssertPanics("too many args", func() {
		getComponentMeta(reflect.TypeFor[func(string, string) Node]())
	}, "components may accept at most one props argument")
	parseAssertPanics("missing return", func() {
		getComponentMeta(reflect.TypeFor[func()]())
	}, "components must return ui.Node")

	if len(toInterfaces(nil)) != 0 {
		parseT.Fatal("toInterfaces(nil) should return nil")
	}
	parseValues := toInterfaces([]Node{Text("a"), nil})
	if len(parseValues) != 2 {
		parseT.Fatalf("toInterfaces() len = %d, want 2", len(parseValues))
	}

	if parseGot := resolveHydrationOptions(nil); parseGot.ScriptID != "" {
		parseT.Fatalf("resolveHydrationOptions(nil) = %+v, want zero value", parseGot)
	}
	if parseGot2 := resolveHydrationOptions([]HydrationOptions{{ScriptID: "boot"}}); parseGot2.ScriptID != "boot" {
		parseT.Fatalf("resolveHydrationOptions() = %+v, want ScriptID=boot", parseGot2)
	}

	if parseErr2 := RenderInto(Text("render"), nil); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "RenderInto") {
		parseT.Fatalf("RenderInto() error = %v, want RenderInto unsupported error", parseErr2)
	}
	if _, parseErr3 := Hydrate(Text("hydrate"), "#app"); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "Hydrate") {
		parseT.Fatalf("Hydrate() error = %v, want Hydrate unsupported error", parseErr3)
	}
	if _, parseErr4 := HydrateInto(Text("hydrate"), nil); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "HydrateInto") {
		parseT.Fatalf("HydrateInto() error = %v, want HydrateInto unsupported error", parseErr4)
	}
	if _, parseErr5 := ReadBootstrapScript("boot"); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "ReadBootstrapScript") {
		parseT.Fatalf("ReadBootstrapScript() error = %v, want unsupported error", parseErr5)
	}
	if _, parseErr6 := ReadBootstrapReferenceScript("boot"); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "ReadBootstrapReferenceScript") {
		parseT.Fatalf("ReadBootstrapReferenceScript() error = %v, want unsupported error", parseErr6)
	}
	if _, parseErr7 := ReadBootstrapReference(SSRBootstrapReference{URL: "/bootstrap.json"}); parseErr7 == nil || !strings.Contains(parseErr7.Error(), "ReadBootstrapReference") {
		parseT.Fatalf("ReadBootstrapReference() error = %v, want unsupported error", parseErr7)
	}

	parseLazy := UseLazyNode(nil)
	parseLazyState := parseLazy.Get()
	if parseLazyState.Error == nil || parseLazyState.Ready {
		parseT.Fatalf("UseLazyNode(nil) = %+v, want unsupported error and Ready=false", parseLazyState)
	}
	parseLazy.Reload()
	parseLazy.Cancel()

	parseContent := Lazy(LazyProps{
		Loader: func(context.Context) (Node, error) { return Text("resolved"), nil },
	})
	parseMarkup, parseErr = RenderToString(parseContent)
	if parseErr != nil {
		parseT.Fatalf("RenderToString(Lazy) error = %v", parseErr)
	}
	if parseMarkup != "resolved" {
		parseT.Fatalf("Lazy markup = %q, want resolved", parseMarkup)
	}

	if parseErr8 := UnsupportedOnServer("Test"); parseErr8 == nil || !strings.Contains(parseErr8.Error(), "Test") {
		parseT.Fatalf("UnsupportedOnServer() error = %v, want helper name", parseErr8)
	}
}

func TestNativeAccessibilityOverlayAndFileStubs(parseT *testing.T) {
	parseManager := UseFocusManager()
	if parseManager.FocusFirstError(FieldErrors{"email": "required"}, map[string]string{"email": "email"}) {
		parseT.Fatal("FocusFirstError() should be false on native builds")
	}
	if parseManager.RememberActive() || parseManager.FocusSelector("#email") || parseManager.FocusByID("email") || parseManager.FocusFirst("#form") || parseManager.Restore() {
		parseT.Fatal("native focus manager methods should return false")
	}

	parseNav := UseCompositeNavigation([]CompositeItem{{ID: "first"}}, CompositeNavigationOptions{InitialIndex: 0})
	if parseNav.ActiveIndex() != -1 || parseNav.ActiveID() != "" || parseNav.ActiveDescendant() != "" || parseNav.IsActive(0) || parseNav.TabIndex(0) != -1 {
		parseT.Fatalf("unexpected native composite navigation state: index=%d id=%q desc=%q active=%t tab=%d", parseNav.ActiveIndex(), parseNav.ActiveID(), parseNav.ActiveDescendant(), parseNav.IsActive(0), parseNav.TabIndex(0))
	}
	parseNav.SetActive(0)
	parseNav.MoveNext()
	parseNav.MovePrevious()
	parseNav.MoveHome()
	parseNav.MoveEnd()
	parseNav.OnKeyDown(nil)

	parseAnnouncer := UseAnnouncer()
	parseAnnouncer.Announce(AnnouncementPolite, "hello")
	parseAnnouncer.Polite("hello")
	parseAnnouncer.Assertive("hello")
	parseAnnouncer.Clear()
	if parseAnnouncer.PoliteID() != "" || parseAnnouncer.AssertiveID() != "" || parseAnnouncer.Region() != nil {
		parseT.Fatalf("unexpected native announcer IDs/region: polite=%q assertive=%q region=%v", parseAnnouncer.PoliteID(), parseAnnouncer.AssertiveID(), parseAnnouncer.Region())
	}

	globalOverlayStackManager = newOverlayStackManager()
	parseStack := UseOverlayStack(OverlayStackOptions{
		Open:                true,
		Kind:                OverlayKindDialog,
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
		TrapFocus:           true,
	})
	if parseStack.ID != "overlay-layer" || !parseStack.IsTop || parseStack.LayerCount != 1 || !parseStack.HandlesEscape || !parseStack.HandlesOutsideClick || !parseStack.TrapFocusActive {
		parseT.Fatalf("unexpected overlay stack snapshot: %+v", parseStack)
	}

	parseOverlayMarkup, parseErr := RenderToString(Overlay(OverlayProps{
		Child:    Text("primary"),
		Children: []Node{Text("secondary")},
	}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(Overlay) error = %v", parseErr)
	}
	if parseOverlayMarkup != "primarysecondary" {
		parseT.Fatalf("Overlay markup = %q, want primarysecondary", parseOverlayMarkup)
	}

	parseAccessibleMarkup, parseErr := RenderToString(AccessibleOverlay(AccessibleOverlayProps{
		Open:     true,
		Child:    Text("one"),
		Children: []Node{Text("two")},
	}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(AccessibleOverlay) error = %v", parseErr)
	}
	if parseAccessibleMarkup != "onetwo" {
		parseT.Fatalf("AccessibleOverlay markup = %q, want onetwo", parseAccessibleMarkup)
	}
	UseFocusTrap(FocusTrapOptions{Active: true})

	parseFile := File{}
	if parseFile.Name() != "" || parseFile.Type() != "" || parseFile.Size() != 0 || parseFile.LastModified() != 0 {
		parseT.Fatalf("unexpected native file metadata: name=%q type=%q size=%d modified=%d", parseFile.Name(), parseFile.Type(), parseFile.Size(), parseFile.LastModified())
	}
	if parseFiles := GetFiles(Event{}); parseFiles != nil {
		parseT.Fatalf("GetFiles() = %#v, want nil", parseFiles)
	}

	var parseEvent Event
	parseEvent.PreventDefault()
	parseEvent.StopPropagation()
	if parseEvent.GetValue() != "" || parseEvent.IsChecked() || parseEvent.GetKeyCode() != 0 || parseEvent.GetKey() != "" {
		parseT.Fatalf("unexpected event accessors: value=%q checked=%t keyCode=%d key=%q", parseEvent.GetValue(), parseEvent.IsChecked(), parseEvent.GetKeyCode(), parseEvent.GetKey())
	}
}
