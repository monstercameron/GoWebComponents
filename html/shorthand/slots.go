package shorthand

import "github.com/monstercameron/GoWebComponents/v4/ui"

// NamedSlot is a group of children tagged with a slot name, produced by Slot and collected
// into a Slots set.
type NamedSlot struct {
	Name     string
	Children []ui.Node
}

// Slots is a named collection of child node groups a component renders into labeled regions
// — explicit, typed named slots (Vue's named slots / React render-children-by-name) without
// magic. A component takes a Slots in its props and places each region with Render/Or.
type Slots map[string][]ui.Node

// Slot tags children with a slot name for placement into a Slots set.
//
//	card(NewSlots(
//	    Slot("header", html.Text("Title")),
//	    Slot("body", content...),
//	))
func Slot(parseName string, parseChildren ...ui.Node) NamedSlot {
	return NamedSlot{Name: parseName, Children: parseChildren}
}

// NewSlots collects named slots into a Slots set. A later entry with the same name replaces
// an earlier one, so callers get last-wins override semantics.
func NewSlots(parseSlots ...NamedSlot) Slots {
	parseSet := make(Slots, len(parseSlots))
	for _, parseSlot := range parseSlots {
		parseSet[parseSlot.Name] = parseSlot.Children
	}
	return parseSet
}

// Has reports whether a non-empty slot was provided under name.
func (parseS Slots) Has(parseName string) bool {
	parseChildren, parseOk := parseS[parseName]
	return parseOk && len(parseChildren) > 0
}

// Render returns the children for a named slot (nil when the slot is absent or empty) —
// what a component calls to place a slot in its layout.
func (parseS Slots) Render(parseName string) []ui.Node {
	if !parseS.Has(parseName) {
		return nil
	}
	return parseS[parseName]
}

// Or returns the named slot's children, or the fallback when the slot was not provided —
// the default-content pattern (a component supplies a sensible default a parent can
// override).
func (parseS Slots) Or(parseName string, parseFallback ...ui.Node) []ui.Node {
	if parseS.Has(parseName) {
		return parseS[parseName]
	}
	return parseFallback
}
