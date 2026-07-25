//go:build !js

package shorthand

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// These regression tests for #77 construct event-handler PropOptions outside a
// component render, which is only legal on the native build (the WASM build's
// handler options register hooks and require component context).  The merge
// logic under test (mergeProps) is platform-independent.

// TestSplitArgsPropsMergesOntoAccumulatedState is a regression guard for #77:
// a bare Props{} argument must not wipe PropOption state accumulated earlier in
// the same call (e.g. an OnClick handler must survive when Props{Class:"x"} follows).
func TestSplitArgsPropsMergesOntoAccumulatedState(parseT *testing.T) {
	parseCalled := false
	parseHandler := func() { parseCalled = true }

	parseNode := Div(OnClick(parseHandler), Props{Class: "x"})
	if parseNode == nil {
		parseT.Fatal("expected non-nil node")
	}
	if parseNode.Props["class"] != "x" {
		parseT.Fatalf("expected class=x, got %#v", parseNode.Props["class"])
	}
	if parseNode.Props["onclick"] == nil {
		parseT.Fatalf("expected onclick to be preserved after Props{} merge, got nil; Props: %#v", parseNode.Props)
	}
	_ = parseCalled
}

// TestSplitArgsPropsDoesNotWipeHandler verifies that Props appearing before
// a PropOption also does not wipe later-applied handler state.
func TestSplitArgsPropsDoesNotWipeHandler(parseT *testing.T) {
	parseCalled := false
	parseHandler := func() { parseCalled = true }

	parseNode := Div(Props{Class: "base"}, OnClick(parseHandler))
	if parseNode.Props["class"] != "base" {
		parseT.Fatalf("expected class=base, got %#v", parseNode.Props["class"])
	}
	if parseNode.Props["onclick"] == nil {
		parseT.Fatalf("expected onclick to survive Props-then-PropOption ordering, got nil")
	}
	_ = parseCalled
}

// TestMergePropsMapAndScalarSemantics pins down mergeProps behavior: scalar
// fields use last-write-wins for non-zero values, map fields merge key-wise.
func TestMergePropsMapAndScalarSemantics(parseT *testing.T) {
	parseBase := Props{
		Class: "base",
		ID:    "keep-me",
		Data:  map[string]string{"a": "1", "b": "2"},
	}
	parseIncoming := Props{
		Class: "override",
		Data:  map[string]string{"b": "3", "c": "4"},
	}
	parseMerged := mergeProps(parseBase, parseIncoming)
	if parseMerged.Class != "override" {
		parseT.Fatalf("expected later Class to win, got %q", parseMerged.Class)
	}
	if parseMerged.ID != "keep-me" {
		parseT.Fatalf("expected unset incoming field to preserve base, got %q", parseMerged.ID)
	}
	if parseMerged.Data["a"] != "1" || parseMerged.Data["b"] != "3" || parseMerged.Data["c"] != "4" {
		parseT.Fatalf("expected key-wise map merge, got %#v", parseMerged.Data)
	}
	if parseBase.Data["b"] != "2" {
		parseT.Fatal("mergeProps must not mutate the base map")
	}
}

// TestHelperReexportsEventHandlerOptions covers the event-handler PropOption
// re-exports.  Native-only: on the WASM build these options register hooks and
// therefore must run inside a component render.
func TestHelperReexportsEventHandlerOptions(parseT *testing.T) {
	parseProps := PropsOf(
		ID("demo"),
		OnClick(func() {}),
		OnInput(func(ui.InputEvent) {}),
		OnChange(func(ui.ChangeEvent) {}),
		OnSubmit(func(ui.FormEvent) {}),
		OnKeyDown(func(ui.KeyboardEvent) {}),
		OnKeyUp(func(ui.KeyboardEvent) {}),
		OnMouseUp(func(ui.MouseEvent) {}),
		OnFocus(func(ui.FocusEvent) {}),
		OnBlur(func(ui.FocusEvent) {}),
	)
	parseElem := Button(FromProps(parseProps), "save")
	if parseElem.Props["onclick"] == nil || parseElem.Props["oninput"] == nil || parseElem.Props["onchange"] == nil || parseElem.Props["onsubmit"] == nil || parseElem.Props["onkeydown"] == nil || parseElem.Props["onkeyup"] == nil || parseElem.Props["onmouseup"] == nil || parseElem.Props["onfocus"] == nil || parseElem.Props["onblur"] == nil {
		parseT.Fatalf("expected event props, got %#v", parseElem.Props)
	}
}
