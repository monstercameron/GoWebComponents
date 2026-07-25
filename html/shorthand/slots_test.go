package shorthand_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/html/shorthand"
)

// TestSlotsRenderAndHas proves provided slots are placed and Has reflects presence.
func TestSlotsRenderAndHas(parseT *testing.T) {
	parseSlots := shorthand.NewSlots(
		shorthand.Slot("header", html.Text("Title")),
		shorthand.Slot("body", html.Text("one"), html.Text("two")),
	)

	if !parseSlots.Has("header") || !parseSlots.Has("body") {
		parseT.Fatal("expected header and body slots to be present")
	}
	if parseSlots.Has("footer") {
		parseT.Fatal("footer was never provided")
	}
	if len(parseSlots.Render("body")) != 2 {
		parseT.Fatalf("expected 2 body children, got %d", len(parseSlots.Render("body")))
	}
	if parseSlots.Render("footer") != nil {
		parseT.Fatal("an absent slot should render nil")
	}
}

// TestSlotsOrFallback proves the default-content pattern: a fallback is used only when the
// slot is absent.
func TestSlotsOrFallback(parseT *testing.T) {
	parseSlots := shorthand.NewSlots(shorthand.Slot("header", html.Text("Provided")))

	if len(parseSlots.Or("header", html.Text("Default"))) != 1 {
		parseT.Fatal("a provided slot should use its own children")
	}
	parseFallback := parseSlots.Or("footer", html.Text("Default footer"))
	if len(parseFallback) != 1 {
		parseT.Fatalf("an absent slot should fall back, got %d nodes", len(parseFallback))
	}
}

// TestNewSlotsLastWins proves a later slot with the same name overrides an earlier one.
func TestNewSlotsLastWins(parseT *testing.T) {
	parseSlots := shorthand.NewSlots(
		shorthand.Slot("x", html.Text("first")),
		shorthand.Slot("x", html.Text("second"), html.Text("third")),
	)
	if len(parseSlots.Render("x")) != 2 {
		parseT.Fatalf("expected last-wins override (2 children), got %d", len(parseSlots.Render("x")))
	}
}

// TestEmptySlotIsAbsent proves a slot declared with no children counts as not provided, so
// Or still falls back.
func TestEmptySlotIsAbsent(parseT *testing.T) {
	parseSlots := shorthand.NewSlots(shorthand.Slot("header"))
	if parseSlots.Has("header") {
		parseT.Fatal("an empty slot should not count as present")
	}
	if len(parseSlots.Or("header", html.Text("fallback"))) != 1 {
		parseT.Fatal("an empty slot should fall back")
	}
}
