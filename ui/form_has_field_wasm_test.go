//go:build js && wasm

package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

type hasFieldModel struct {
	Name  string
	Count int
}

// TestFormHasFieldWasm exercises the wasm build of Form.HasField (the only field-introspection
// method whose value access differs from native — it reads through the state hook). It runs inside
// a real render so the hook context is live, asserting an existing field is found and a typo / empty
// name are rejected. (MustSetField is byte-identical to the native impl, covered by TestFormMustSetField.)
func TestFormHasFieldWasm(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePrev := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePrev })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	var parseHasName, parseHasCount, parseHasTypo, parseHasEmpty, parseMustOK bool
	parseProbe := func() Node {
		parseForm := UseForm(hasFieldModel{Name: "Atlas"})
		parseHasName = parseForm.HasField("Name")
		parseHasCount = parseForm.HasField("Count")
		parseHasTypo = parseForm.HasField("Nmae")
		parseHasEmpty = parseForm.HasField("")
		// MustSetField happy path on the wasm SetField code path (state.Update) must not panic.
		parseForm.MustSetField("Name", "Delta")
		parseMustOK = true
		return runtime.CreateElement("span", map[string]any{}, "")
	}

	if parseErr := RenderInto(CreateElement(parseProbe), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto returned error: %v", parseErr)
	}
	parseScheduler.Flush()

	if !parseHasName || !parseHasCount {
		parseT.Fatalf("wasm HasField must find existing fields (Name=%v Count=%v)", parseHasName, parseHasCount)
	}
	if parseHasTypo || parseHasEmpty {
		parseT.Fatalf("wasm HasField must reject a typo/empty name (Nmae=%v ''=%v)", parseHasTypo, parseHasEmpty)
	}
	if !parseMustOK {
		parseT.Fatal("wasm MustSetField happy path must complete without panicking")
	}
}
