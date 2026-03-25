//go:build !js || !wasm
// +build !js !wasm

package ui

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

type nativeProps struct {
	Label string
}

func nativeComponent(props nativeProps) Node {
	return Text("hello " + props.Label)
}

func nativeZeroArgComponent() Node {
	return Text("zero")
}

func TestNativeStateAndHookHelpers(t *testing.T) {
	var zeroState State[int]
	if zeroState.Get() != 0 {
		t.Fatalf("zero state Get() = %d, want 0", zeroState.Get())
	}
	state := UseState(1)
	state.Set(3)
	state.Update(func(current int) int { return current + 2 })
	if state.Get() != 5 {
		t.Fatalf("state.Get() = %d, want 5", state.Get())
	}

	var zeroRef Ref[string]
	if zeroRef.Get() != "" {
		t.Fatalf("zero ref Get() = %q, want empty", zeroRef.Get())
	}
	ref := UseRef("start")
	ref.Set("updated")
	if ref.Get() != "updated" {
		t.Fatalf("ref.Get() = %q, want updated", ref.Get())
	}

	var zeroReducer Reducer[int, int]
	if zeroReducer.Get() != 0 {
		t.Fatalf("zero reducer Get() = %d, want 0", zeroReducer.Get())
	}
	reducer := UseReducer(func(state int, action int) int { return state + action }, 2)
	reducer.Dispatch(4)
	if reducer.Get() != 6 {
		t.Fatalf("reducer.Get() = %d, want 6", reducer.Get())
	}

	var zeroPrevious Previous[string]
	if zeroPrevious.Ok() || zeroPrevious.Get() != "" {
		t.Fatalf("zero previous = (%q, %t), want empty/false", zeroPrevious.Get(), zeroPrevious.Ok())
	}
	previous := UsePrevious("value")
	if previous.Ok() || previous.Get() != "" {
		t.Fatalf("native previous should be empty, got (%q, %t)", previous.Get(), previous.Ok())
	}

	var zeroDebounced Debounced[string]
	if zeroDebounced.Get() != "" || zeroDebounced.Pending() {
		t.Fatalf("zero debounced = (%q, %t), want empty/false", zeroDebounced.Get(), zeroDebounced.Pending())
	}
	debounced := UseDebounced("ready", time.Second)
	if debounced.Get() != "ready" || debounced.Pending() {
		t.Fatalf("debounced = (%q, %t), want ready/false", debounced.Get(), debounced.Pending())
	}

	var zeroThrottled Throttled[string]
	if zeroThrottled.Get() != "" || zeroThrottled.Pending() {
		t.Fatalf("zero throttled = (%q, %t), want empty/false", zeroThrottled.Get(), zeroThrottled.Pending())
	}
	throttled := UseThrottled("steady", time.Second)
	if throttled.Get() != "steady" || throttled.Pending() {
		t.Fatalf("throttled = (%q, %t), want steady/false", throttled.Get(), throttled.Pending())
	}

	if got := UseDeferredValue(42); got != 42 {
		t.Fatalf("UseDeferredValue() = %d, want 42", got)
	}
	if got := UseMemo(func() int { return 7 }); got != 7 {
		t.Fatalf("UseMemo() = %d, want 7", got)
	}
	if got := UseMemo[int](nil); got != 0 {
		t.Fatalf("UseMemo(nil) = %d, want 0", got)
	}
	if got := UseCallback("handler"); got != "handler" {
		t.Fatalf("UseCallback() = %q, want handler", got)
	}

	ran := false
	transition := UseTransition()
	if transition.Pending() {
		t.Fatal("native transition should not be pending")
	}
	transition.Start(func() { ran = true })
	if !ran {
		t.Fatal("transition.Start() did not run callback")
	}
	StartTransition(nil)

	if UseId() == UseId() {
		t.Fatal("UseId() should return distinct IDs")
	}

	if got := UseEvent("wrapped").Value(); got != "wrapped" {
		t.Fatalf("UseEvent().Value() = %#v, want wrapped", got)
	}
	if got := WrapHandler(123).Value(); got != 123 {
		t.Fatalf("WrapHandler().Value() = %#v, want 123", got)
	}
}

func TestNativeServerOnlyHelpersAndRenderComponent(t *testing.T) {
	rendered := renderComponent(nativeComponent, map[string]interface{}{propsKey: nativeProps{Label: "gwc"}})
	markup, err := RenderToString(rendered)
	if err != nil {
		t.Fatalf("RenderToString(renderComponent) error = %v", err)
	}
	if markup != "hello gwc" {
		t.Fatalf("rendered markup = %q, want hello gwc", markup)
	}

	markup, err = RenderToString(CreateElement(nativeZeroArgComponent))
	if err != nil {
		t.Fatalf("RenderToString(CreateElement) error = %v", err)
	}
	if markup != "zero" {
		t.Fatalf("CreateElement zero-arg markup = %q, want zero", markup)
	}

	node := Text("passthrough")
	if CreateElement(node) != node {
		t.Fatal("CreateElement(node) should return the same node")
	}
	if renderComponent(nil, nil) != nil {
		t.Fatal("renderComponent(nil) should return nil")
	}

	assertPanics := func(name string, fn func(), contains string) {
		t.Helper()
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatalf("%s: expected panic", name)
			}
			if contains != "" && !strings.Contains(recovered.(string), contains) {
				t.Fatalf("%s: panic = %q, want substring %q", name, recovered, contains)
			}
		}()
		fn()
	}

	assertPanics("invalid component", func() {
		renderComponent(123, nil)
	}, "GWC-UI-CREATE-ELEMENT-TYPE")
	assertPanics("too many args", func() {
		getComponentMeta(reflect.TypeOf(func(string, string) Node { return nil }))
	}, "components may accept at most one props argument")
	assertPanics("missing return", func() {
		getComponentMeta(reflect.TypeOf(func() {}))
	}, "components must return ui.Node")

	if len(toInterfaces(nil)) != 0 {
		t.Fatal("toInterfaces(nil) should return nil")
	}
	values := toInterfaces([]Node{Text("a"), nil})
	if len(values) != 2 {
		t.Fatalf("toInterfaces() len = %d, want 2", len(values))
	}

	if got := resolveHydrationOptions(nil); got.ScriptID != "" {
		t.Fatalf("resolveHydrationOptions(nil) = %+v, want zero value", got)
	}
	if got := resolveHydrationOptions([]HydrationOptions{{ScriptID: "boot"}}); got.ScriptID != "boot" {
		t.Fatalf("resolveHydrationOptions() = %+v, want ScriptID=boot", got)
	}

	if err := RenderInto(Text("render"), nil); err == nil || !strings.Contains(err.Error(), "RenderInto") {
		t.Fatalf("RenderInto() error = %v, want RenderInto unsupported error", err)
	}
	if _, err := Hydrate(Text("hydrate"), "#app"); err == nil || !strings.Contains(err.Error(), "Hydrate") {
		t.Fatalf("Hydrate() error = %v, want Hydrate unsupported error", err)
	}
	if _, err := HydrateInto(Text("hydrate"), nil); err == nil || !strings.Contains(err.Error(), "HydrateInto") {
		t.Fatalf("HydrateInto() error = %v, want HydrateInto unsupported error", err)
	}
	if _, err := ReadBootstrapScript("boot"); err == nil || !strings.Contains(err.Error(), "ReadBootstrapScript") {
		t.Fatalf("ReadBootstrapScript() error = %v, want unsupported error", err)
	}
	if _, err := ReadBootstrapReferenceScript("boot"); err == nil || !strings.Contains(err.Error(), "ReadBootstrapReferenceScript") {
		t.Fatalf("ReadBootstrapReferenceScript() error = %v, want unsupported error", err)
	}
	if _, err := ReadBootstrapReference(SSRBootstrapReference{URL: "/bootstrap.json"}); err == nil || !strings.Contains(err.Error(), "ReadBootstrapReference") {
		t.Fatalf("ReadBootstrapReference() error = %v, want unsupported error", err)
	}

	lazy := UseLazyNode(nil)
	lazyState := lazy.Get()
	if lazyState.Error == nil || lazyState.Ready {
		t.Fatalf("UseLazyNode(nil) = %+v, want unsupported error and Ready=false", lazyState)
	}
	lazy.Reload()
	lazy.Cancel()

	content := Lazy(LazyProps{
		Loader: func(context.Context) (Node, error) { return Text("resolved"), nil },
	})
	markup, err = RenderToString(content)
	if err != nil {
		t.Fatalf("RenderToString(Lazy) error = %v", err)
	}
	if markup != "resolved" {
		t.Fatalf("Lazy markup = %q, want resolved", markup)
	}

	if err := UnsupportedOnServer("Test"); err == nil || !strings.Contains(err.Error(), "Test") {
		t.Fatalf("UnsupportedOnServer() error = %v, want helper name", err)
	}
}

func TestNativeAccessibilityOverlayAndFileStubs(t *testing.T) {
	manager := UseFocusManager()
	if manager.FocusFirstError(FieldErrors{"email": "required"}, map[string]string{"email": "email"}) {
		t.Fatal("FocusFirstError() should be false on native builds")
	}
	if manager.RememberActive() || manager.FocusSelector("#email") || manager.FocusByID("email") || manager.FocusFirst("#form") || manager.Restore() {
		t.Fatal("native focus manager methods should return false")
	}

	nav := UseCompositeNavigation([]CompositeItem{{ID: "first"}}, CompositeNavigationOptions{InitialIndex: 0})
	if nav.ActiveIndex() != -1 || nav.ActiveID() != "" || nav.ActiveDescendant() != "" || nav.IsActive(0) || nav.TabIndex(0) != -1 {
		t.Fatalf("unexpected native composite navigation state: index=%d id=%q desc=%q active=%t tab=%d", nav.ActiveIndex(), nav.ActiveID(), nav.ActiveDescendant(), nav.IsActive(0), nav.TabIndex(0))
	}
	nav.SetActive(0)
	nav.MoveNext()
	nav.MovePrevious()
	nav.MoveHome()
	nav.MoveEnd()
	nav.OnKeyDown(nil)

	announcer := UseAnnouncer()
	announcer.Announce(AnnouncementPolite, "hello")
	announcer.Polite("hello")
	announcer.Assertive("hello")
	announcer.Clear()
	if announcer.PoliteID() != "" || announcer.AssertiveID() != "" || announcer.Region() != nil {
		t.Fatalf("unexpected native announcer IDs/region: polite=%q assertive=%q region=%v", announcer.PoliteID(), announcer.AssertiveID(), announcer.Region())
	}

	globalOverlayStackManager = newOverlayStackManager()
	stack := UseOverlayStack(OverlayStackOptions{
		Open:                true,
		Kind:                OverlayKindDialog,
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
		TrapFocus:           true,
	})
	if stack.ID != "overlay-layer" || !stack.IsTop || stack.LayerCount != 1 || !stack.HandlesEscape || !stack.HandlesOutsideClick || !stack.TrapFocusActive {
		t.Fatalf("unexpected overlay stack snapshot: %+v", stack)
	}

	overlayMarkup, err := RenderToString(Overlay(OverlayProps{
		Child:    Text("primary"),
		Children: []Node{Text("secondary")},
	}))
	if err != nil {
		t.Fatalf("RenderToString(Overlay) error = %v", err)
	}
	if overlayMarkup != "primarysecondary" {
		t.Fatalf("Overlay markup = %q, want primarysecondary", overlayMarkup)
	}

	accessibleMarkup, err := RenderToString(AccessibleOverlay(AccessibleOverlayProps{
		Open:     true,
		Child:    Text("one"),
		Children: []Node{Text("two")},
	}))
	if err != nil {
		t.Fatalf("RenderToString(AccessibleOverlay) error = %v", err)
	}
	if accessibleMarkup != "onetwo" {
		t.Fatalf("AccessibleOverlay markup = %q, want onetwo", accessibleMarkup)
	}
	UseFocusTrap(FocusTrapOptions{Active: true})

	file := File{}
	if file.Name() != "" || file.Type() != "" || file.Size() != 0 || file.LastModified() != 0 {
		t.Fatalf("unexpected native file metadata: name=%q type=%q size=%d modified=%d", file.Name(), file.Type(), file.Size(), file.LastModified())
	}
	if files := GetFiles(Event{}); files != nil {
		t.Fatalf("GetFiles() = %#v, want nil", files)
	}

	var event Event
	event.PreventDefault()
	event.StopPropagation()
	if event.GetValue() != "" || event.IsChecked() || event.GetKeyCode() != 0 || event.GetKey() != "" {
		t.Fatalf("unexpected event accessors: value=%q checked=%t keyCode=%d key=%q", event.GetValue(), event.IsChecked(), event.GetKeyCode(), event.GetKey())
	}
}
