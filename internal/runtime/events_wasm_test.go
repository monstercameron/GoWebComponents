//go:build js && wasm
// +build js,wasm

package runtime

import (
	"syscall/js"
	"testing"
)

func TestGoEventImplementsEvent(parseT *testing.T) {
	var _ Event = GoEvent{}
}

func TestGoEventAccessorsReturnExpectedValues(parseT *testing.T) {
	parseTarget := js.Global().Get("Object").New()
	parseTarget.Set("value", "hello")
	parseTarget.Set("checked", true)

	parseEventValue := js.Global().Get("Object").New()
	parseEventValue.Set("target", parseTarget)
	parseEventValue.Set("key", "Enter")
	parseEventValue.Set("keyCode", 13)

	parseEvent := NewGoEvent(parseEventValue)

	if parseGot := parseEvent.GetValue(); parseGot != "hello" {
		parseT.Fatalf("expected target value, got %q", parseGot)
	}
	if !parseEvent.IsChecked() {
		parseT.Fatalf("expected checked target state")
	}
	if parseGot2 := parseEvent.GetKey(); parseGot2 != "Enter" {
		parseT.Fatalf("expected key, got %q", parseGot2)
	}
	if parseGot3 := parseEvent.GetKeyCode(); parseGot3 != 13 {
		parseT.Fatalf("expected keyCode 13, got %d", parseGot3)
	}

	parseTargetNode, parseOk := parseEvent.GetTarget().(*jsEventTargetNode)
	if !parseOk {
		parseT.Fatalf("expected jsEventTargetNode target wrapper, got %T", parseEvent.GetTarget())
	}
	if parseTargetNode.IsNull() {
		parseT.Fatalf("expected non-null target wrapper")
	}
	if !parseTargetNode.value.Equal(parseTarget) {
		parseT.Fatalf("expected target wrapper to hold original js target")
	}
}

func TestGoEventMissingPropertiesReturnZeroValues(parseT *testing.T) {
	parseEvent := NewGoEvent(js.Global().Get("Object").New())

	if parseGot := parseEvent.GetValue(); parseGot != "" {
		parseT.Fatalf("expected empty value, got %q", parseGot)
	}
	if parseEvent.IsChecked() {
		parseT.Fatalf("expected unchecked state when target.checked is missing")
	}
	if parseGot2 := parseEvent.GetKey(); parseGot2 != "" {
		parseT.Fatalf("expected empty key, got %q", parseGot2)
	}
	if parseGot3 := parseEvent.GetKeyCode(); parseGot3 != 0 {
		parseT.Fatalf("expected zero keyCode, got %d", parseGot3)
	}
	if parseTarget := parseEvent.GetTarget(); parseTarget != nil {
		parseT.Fatalf("expected nil target when target is missing, got %T", parseTarget)
	}
}

func TestGoEventPreventDefaultAndStopPropagationCallUnderlyingMethods(parseT *testing.T) {
	parsePreventCalled := 0
	parseStopCalled := 0

	parsePreventFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parsePreventCalled++
		return nil
	})
	defer parsePreventFn.Release()

	parseStopFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseStopCalled++
		return nil
	})
	defer parseStopFn.Release()

	parseEventValue := js.Global().Get("Object").New()
	parseEventValue.Set("preventDefault", parsePreventFn)
	parseEventValue.Set("stopPropagation", parseStopFn)

	parseEvent := NewGoEvent(parseEventValue)
	parseEvent.PreventDefault()
	parseEvent.StopPropagation()

	if parsePreventCalled != 1 {
		parseT.Fatalf("expected preventDefault to be invoked once, got %d", parsePreventCalled)
	}
	if parseStopCalled != 1 {
		parseT.Fatalf("expected stopPropagation to be invoked once, got %d", parseStopCalled)
	}
}

func TestGoEventSafeOnUndefinedOrMissingMethods(parseT *testing.T) {
	NewGoEvent(js.Undefined()).PreventDefault()
	NewGoEvent(js.Undefined()).StopPropagation()

	parseEventValue := js.Global().Get("Object").New()
	parseEvent := NewGoEvent(parseEventValue)
	parseEvent.PreventDefault()
	parseEvent.StopPropagation()
}

func TestJSEventTargetNodeEquals(parseT *testing.T) {
	parseValue := js.Global().Get("Object").New()
	parseSame := &jsEventTargetNode{value: parseValue}
	parseAlsoSame := &jsEventTargetNode{value: parseValue}
	parseDifferent := &jsEventTargetNode{value: js.Global().Get("Object").New()}

	if !parseSame.Equals(parseAlsoSame) {
		parseT.Fatalf("expected equal wrappers for same underlying js value")
	}
	if parseSame.Equals(parseDifferent) {
		parseT.Fatalf("expected different wrappers to compare false")
	}
	if parseSame.Equals(nil) {
		parseT.Fatalf("expected non-nil node to not equal nil")
	}
}

func TestJSEventTargetNodeNullHelpers(parseT *testing.T) {
	var parseTypedNil DOMNode = (*jsEventTargetNode)(nil)
	parseNullWrapper := DOMNode(&jsEventTargetNode{value: js.Null()})

	if !IsDOMNodeNull(parseTypedNil) {
		parseT.Fatal("expected typed nil event target node to report null")
	}
	if !IsDOMNodeNull(parseNullWrapper) {
		parseT.Fatal("expected null js.Value wrapper to report null")
	}
	if !IsSameDOMNode(parseTypedNil, parseNullWrapper) {
		parseT.Fatal("expected typed nil and null-wrapper event target nodes to compare equal")
	}
}
