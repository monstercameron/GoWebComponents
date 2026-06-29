//go:build js && wasm

package runtime

import "syscall/js"

// Pointer/mouse coordinate and modifier-key accessors for GoEvent. Before these, a handler that
// needed a cursor position or a Shift/Ctrl chord had to escape-hatch through GetTarget().(DOMNode)
// and raw js; now the common MouseEvent/PointerEvent/KeyboardEvent fields are first-class.

// eventNumber reads a numeric property off the event, returning 0 when the event or property is
// absent (mirrors the nil-guard style of GetKeyCode).
func (parseE GoEvent) eventNumber(parseProp string) float64 {
	if parseE.jsValue.IsUndefined() || parseE.jsValue.IsNull() {
		return 0
	}
	parseValue := parseE.jsValue.Get(parseProp)
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return 0
	}
	if parseValue.Type() != js.TypeNumber {
		return 0
	}
	return parseValue.Float()
}

// eventFlag reads a boolean modifier property off the event, returning false when absent.
func (parseE GoEvent) eventFlag(parseProp string) bool {
	if parseE.jsValue.IsUndefined() || parseE.jsValue.IsNull() {
		return false
	}
	parseValue := parseE.jsValue.Get(parseProp)
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return false
	}
	return parseValue.Truthy()
}

// GetClientX / GetClientY return the pointer position relative to the viewport (event.clientX/Y).
func (parseE GoEvent) GetClientX() float64 { return parseE.eventNumber("clientX") }
func (parseE GoEvent) GetClientY() float64 { return parseE.eventNumber("clientY") }

// GetPageX / GetPageY return the pointer position relative to the whole document (event.pageX/Y).
func (parseE GoEvent) GetPageX() float64 { return parseE.eventNumber("pageX") }
func (parseE GoEvent) GetPageY() float64 { return parseE.eventNumber("pageY") }

// GetOffsetX / GetOffsetY return the pointer position relative to the target's padding edge
// (event.offsetX/Y).
func (parseE GoEvent) GetOffsetX() float64 { return parseE.eventNumber("offsetX") }
func (parseE GoEvent) GetOffsetY() float64 { return parseE.eventNumber("offsetY") }

// GetButton returns which mouse button changed state (0 main/left, 1 auxiliary/middle, 2
// secondary/right); -1 when no button is associated with the event.
func (parseE GoEvent) GetButton() int {
	if parseE.jsValue.IsUndefined() || parseE.jsValue.IsNull() {
		return -1
	}
	parseValue := parseE.jsValue.Get("button")
	if parseValue.IsUndefined() || parseValue.IsNull() || parseValue.Type() != js.TypeNumber {
		return -1
	}
	return parseValue.Int()
}

// GetButtons returns the bitmask of buttons currently held (event.buttons); 0 when none.
func (parseE GoEvent) GetButtons() int { return int(parseE.eventNumber("buttons")) }

// GetShiftKey / GetCtrlKey / GetAltKey / GetMetaKey report modifier-key state at event time,
// for both keyboard and pointer events.
func (parseE GoEvent) GetShiftKey() bool { return parseE.eventFlag("shiftKey") }
func (parseE GoEvent) GetCtrlKey() bool  { return parseE.eventFlag("ctrlKey") }
func (parseE GoEvent) GetAltKey() bool   { return parseE.eventFlag("altKey") }
func (parseE GoEvent) GetMetaKey() bool  { return parseE.eventFlag("metaKey") }
