//go:build js && wasm

package runtime

import (
	"syscall/js"
	"testing"
)

// TestGoEventPointerAndModifierAccessors proves the new coordinate/modifier accessors read the
// event's own fields (clientX/Y, pageX/Y, offsetX/Y, button, buttons, shift/ctrl/alt/metaKey).
func TestGoEventPointerAndModifierAccessors(parseT *testing.T) {
	parseEventValue := js.Global().Get("Object").New()
	parseEventValue.Set("clientX", 12.5)
	parseEventValue.Set("clientY", 34)
	parseEventValue.Set("pageX", 100)
	parseEventValue.Set("pageY", 200)
	parseEventValue.Set("offsetX", 5)
	parseEventValue.Set("offsetY", 6)
	parseEventValue.Set("button", 2)
	parseEventValue.Set("buttons", 3)
	parseEventValue.Set("shiftKey", true)
	parseEventValue.Set("ctrlKey", false)
	parseEventValue.Set("altKey", true)
	parseEventValue.Set("metaKey", true)

	parseEvent := NewGoEvent(parseEventValue)

	if parseEvent.GetClientX() != 12.5 || parseEvent.GetClientY() != 34 {
		parseT.Fatalf("clientX/Y = %v,%v want 12.5,34", parseEvent.GetClientX(), parseEvent.GetClientY())
	}
	if parseEvent.GetPageX() != 100 || parseEvent.GetPageY() != 200 {
		parseT.Fatalf("pageX/Y = %v,%v want 100,200", parseEvent.GetPageX(), parseEvent.GetPageY())
	}
	if parseEvent.GetOffsetX() != 5 || parseEvent.GetOffsetY() != 6 {
		parseT.Fatalf("offsetX/Y = %v,%v want 5,6", parseEvent.GetOffsetX(), parseEvent.GetOffsetY())
	}
	if parseEvent.GetButton() != 2 {
		parseT.Fatalf("button = %d want 2", parseEvent.GetButton())
	}
	if parseEvent.GetButtons() != 3 {
		parseT.Fatalf("buttons = %d want 3", parseEvent.GetButtons())
	}
	if !parseEvent.GetShiftKey() || parseEvent.GetCtrlKey() || !parseEvent.GetAltKey() || !parseEvent.GetMetaKey() {
		parseT.Fatalf("modifiers shift/ctrl/alt/meta = %v/%v/%v/%v want true/false/true/true",
			parseEvent.GetShiftKey(), parseEvent.GetCtrlKey(), parseEvent.GetAltKey(), parseEvent.GetMetaKey())
	}
}

// TestGoEventPointerAccessorsAbsentAreSafe proves the accessors degrade to safe zero/false on a
// nil event or a missing property — no panic, no surprise (button absent reports -1).
func TestGoEventPointerAccessorsAbsentAreSafe(parseT *testing.T) {
	parseNil := NewGoEvent(js.Undefined())
	if parseNil.GetClientX() != 0 || parseNil.GetButtons() != 0 || parseNil.GetShiftKey() {
		parseT.Fatal("nil event accessors must return zero/false")
	}
	if parseNil.GetButton() != -1 {
		parseT.Fatalf("nil event GetButton = %d want -1", parseNil.GetButton())
	}

	parseEmpty := NewGoEvent(js.Global().Get("Object").New())
	if parseEmpty.GetClientY() != 0 || parseEmpty.GetMetaKey() {
		parseT.Fatal("missing properties must return zero/false")
	}
	if parseEmpty.GetButton() != -1 {
		parseT.Fatalf("missing button = %d want -1", parseEmpty.GetButton())
	}
}
