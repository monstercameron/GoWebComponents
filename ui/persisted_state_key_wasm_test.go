//go:build js && wasm

package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
)

// TestUsePersistedStateReReadsStoredValueWhenKeyChanges proves the stored value for the NEW key
// is what the hook returns after the key prop changes, instead of the previous key's value.
func TestUsePersistedStateReReadsStoredValueWhenKeyChanges(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePrev := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePrev })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	withFakePersistStorage(parseT, &fakePersistStorage{parseValues: map[string]string{
		"user-a": `"alice"`,
		"user-b": `"bob"`,
	}}, nil)
	withFakePersistWindowEvents(parseT, &fakePersistEventTarget{}, nil)

	type probeProps struct{ Key string }
	var parseSeen string
	parseProbe := func(parseProps probeProps) Node {
		parseSeen = UsePersistedState(parseProps.Key, "fallback", PersistLocal).Get()
		return runtime.CreateElement("span", map[string]any{}, parseSeen)
	}

	if parseErr := RenderInto(CreateElement(parseProbe, probeProps{Key: "user-a"}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	parseScheduler.Flush()
	if parseSeen != "alice" {
		parseT.Fatalf("precondition: key user-a rendered %q, want alice", parseSeen)
	}

	if parseErr := RenderInto(CreateElement(parseProbe, probeProps{Key: "user-b"}), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto after key change: %v", parseErr)
	}
	parseScheduler.Flush()
	if parseSeen != "bob" {
		parseT.Fatalf("after key change to user-b the hook returned %q, want the stored value bob", parseSeen)
	}
}
