//go:build js && wasm

package state

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// TestSignalTextTracksValueFineGrained proves the fine-grained property: a
// Signal.Text node holds a live getter that observes the latest signal value
// after Set — i.e. the bound DOM text updates from the signal directly, without
// re-rendering the component that produced the node.
func TestSignalTextTracksValueFineGrained(parseT *testing.T) {
	installStateHookContext(parseT)

	parseCount := NewKeyedSignal("signal-test-text", 2)
	parseNode := parseCount.Text(func(parseValue int) string {
		return fmt.Sprintf("count:%d", parseValue)
	})
	if parseNode == nil || parseNode.Type != runtime.ReactiveTextNodeType {
		parseT.Fatalf("expected a reactive text node, got %#v", parseNode)
	}
	if parseGot, _ := parseNode.Props["__gwc_reactive_text_atom_id"].(string); parseGot != "signal-test-text" {
		parseT.Fatalf("expected reactive text bound to the signal id, got %q", parseGot)
	}
	parseGetter, _ := parseNode.Props["__gwc_reactive_text_getter"].(func() string)
	if parseGetter == nil {
		parseT.Fatal("expected a reactive text getter")
	}
	if parseGot := parseGetter(); parseGot != "count:2" {
		parseT.Fatalf("expected initial fine-grained text count:2, got %q", parseGot)
	}
	// The fine-grained update: changing the signal is observed by the bound getter
	// without re-running any component render.
	parseCount.Set(4)
	if parseGot := parseGetter(); parseGot != "count:4" {
		parseT.Fatalf("expected fine-grained text to observe count:4 after Set, got %q", parseGot)
	}
}

// TestComputedTextTracksDependency proves a ComputedSignal.Text getter recomputes
// from its live sources when a dependency changes.
func TestComputedTextTracksDependency(parseT *testing.T) {
	installStateHookContext(parseT)

	parsePrice := NewKeyedSignal("signal-test-computed-price", 100)
	parseLabel := NewComputed(func() string {
		return fmt.Sprintf("$%d", parsePrice.Get()*110/100)
	}, parsePrice)

	parseNode := parseLabel.Text(func(parseValue string) string { return parseValue })
	parseGetter, _ := parseNode.Props["__gwc_reactive_text_getter"].(func() string)
	if parseGetter == nil {
		parseT.Fatal("expected a computed reactive text getter")
	}
	if parseGot := parseGetter(); parseGot != "$110" {
		parseT.Fatalf("expected initial computed text $110, got %q", parseGot)
	}
	parsePrice.Set(200)
	if parseGot := parseGetter(); parseGot != "$220" {
		parseT.Fatalf("expected computed text to track dependency to $220, got %q", parseGot)
	}
}
