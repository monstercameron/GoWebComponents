//go:build js && wasm

package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
)

// TestFormSetFieldReportsValidFieldWhenUpdateIsDeferred proves SetField/MustSetField judge the
// field name and value type synchronously: inside a transition the state update is queued, so a
// result computed inside the queued updater would read false and make MustSetField panic on a
// perfectly valid field. An invalid field must still report false.
func TestFormSetFieldReportsValidFieldWhenUpdateIsDeferred(parseT *testing.T) {
	parseAdapter := newQueryHydrationDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	parseScheduler := &queuedScheduler{}

	parsePrev := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() { runtimeInitialized = parsePrev })
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	var parseValid, parseTypo, parseMismatch bool
	var parseMustPanicked bool
	var parseValue string
	parseProbe := func() Node {
		parseForm := UseForm(hasFieldModel{Name: "Atlas"})
		parseValue = parseForm.Get().Name
		if parseValue == "Atlas" {
			StartTransition(func() {
				parseValid = parseForm.SetField("Name", "Delta")
				parseTypo = parseForm.SetField("Nmae", "x")
				parseMismatch = parseForm.SetField("Count", "not-an-int")
				func() {
					defer func() { parseMustPanicked = recover() != nil }()
					parseForm.MustSetField("Name", "Delta")
				}()
			})
		}
		return runtime.CreateElement("span", map[string]any{}, parseValue)
	}

	if parseErr := RenderInto(CreateElement(parseProbe), parseContainer); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	parseScheduler.Flush()

	if !parseValid {
		parseT.Fatal("SetField on a valid field reported false while the update was deferred")
	}
	if parseMustPanicked {
		parseT.Fatal("MustSetField panicked on a valid field while the update was deferred")
	}
	if parseTypo || parseMismatch {
		parseT.Fatalf("invalid SetField reported success (typo=%v mismatch=%v)", parseTypo, parseMismatch)
	}
}
