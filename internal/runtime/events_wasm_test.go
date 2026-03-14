//go:build js && wasm
// +build js,wasm

package runtime

import (
	"syscall/js"
	"testing"
)

func TestGoEventImplementsEvent(t *testing.T) {
	var _ Event = GoEvent{}
}

func TestGoEventAccessorsReturnExpectedValues(t *testing.T) {
	target := js.Global().Get("Object").New()
	target.Set("value", "hello")
	target.Set("checked", true)

	eventValue := js.Global().Get("Object").New()
	eventValue.Set("target", target)
	eventValue.Set("key", "Enter")
	eventValue.Set("keyCode", 13)

	event := NewGoEvent(eventValue)

	if got := event.GetValue(); got != "hello" {
		t.Fatalf("expected target value, got %q", got)
	}
	if !event.IsChecked() {
		t.Fatalf("expected checked target state")
	}
	if got := event.GetKey(); got != "Enter" {
		t.Fatalf("expected key, got %q", got)
	}
	if got := event.GetKeyCode(); got != 13 {
		t.Fatalf("expected keyCode 13, got %d", got)
	}

	targetNode, ok := event.GetTarget().(*jsEventTargetNode)
	if !ok {
		t.Fatalf("expected jsEventTargetNode target wrapper, got %T", event.GetTarget())
	}
	if targetNode.IsNull() {
		t.Fatalf("expected non-null target wrapper")
	}
	if !targetNode.value.Equal(target) {
		t.Fatalf("expected target wrapper to hold original js target")
	}
}

func TestGoEventMissingPropertiesReturnZeroValues(t *testing.T) {
	event := NewGoEvent(js.Global().Get("Object").New())

	if got := event.GetValue(); got != "" {
		t.Fatalf("expected empty value, got %q", got)
	}
	if event.IsChecked() {
		t.Fatalf("expected unchecked state when target.checked is missing")
	}
	if got := event.GetKey(); got != "" {
		t.Fatalf("expected empty key, got %q", got)
	}
	if got := event.GetKeyCode(); got != 0 {
		t.Fatalf("expected zero keyCode, got %d", got)
	}
	if target := event.GetTarget(); target != nil {
		t.Fatalf("expected nil target when target is missing, got %T", target)
	}
}

func TestGoEventPreventDefaultAndStopPropagationCallUnderlyingMethods(t *testing.T) {
	preventCalled := 0
	stopCalled := 0

	preventFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		preventCalled++
		return nil
	})
	defer preventFn.Release()

	stopFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		stopCalled++
		return nil
	})
	defer stopFn.Release()

	eventValue := js.Global().Get("Object").New()
	eventValue.Set("preventDefault", preventFn)
	eventValue.Set("stopPropagation", stopFn)

	event := NewGoEvent(eventValue)
	event.PreventDefault()
	event.StopPropagation()

	if preventCalled != 1 {
		t.Fatalf("expected preventDefault to be invoked once, got %d", preventCalled)
	}
	if stopCalled != 1 {
		t.Fatalf("expected stopPropagation to be invoked once, got %d", stopCalled)
	}
}

func TestGoEventSafeOnUndefinedOrMissingMethods(t *testing.T) {
	NewGoEvent(js.Undefined()).PreventDefault()
	NewGoEvent(js.Undefined()).StopPropagation()

	eventValue := js.Global().Get("Object").New()
	event := NewGoEvent(eventValue)
	event.PreventDefault()
	event.StopPropagation()
}

func TestJSEventTargetNodeEquals(t *testing.T) {
	value := js.Global().Get("Object").New()
	same := &jsEventTargetNode{value: value}
	alsoSame := &jsEventTargetNode{value: value}
	different := &jsEventTargetNode{value: js.Global().Get("Object").New()}

	if !same.Equals(alsoSame) {
		t.Fatalf("expected equal wrappers for same underlying js value")
	}
	if same.Equals(different) {
		t.Fatalf("expected different wrappers to compare false")
	}
	if same.Equals(nil) {
		t.Fatalf("expected non-nil node to not equal nil")
	}
}
