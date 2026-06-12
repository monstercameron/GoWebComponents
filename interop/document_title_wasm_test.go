//go:build js && wasm

package interop

import (
	"syscall/js"
	"testing"
)

// TestDocumentTitleRoundTripWASM verifies Title/SetTitle read and write the
// document.title property through the wasm wiring (a shimmed document object,
// matching the package's node-runner test convention).
func TestDocumentTitleRoundTripWASM(parseT *testing.T) {
	parseDocument := js.Global().Get("Object").New()
	parseDocument.Set("title", "RelayDesk – AI Chat Workspace")
	parseRestoreDocument := setGlobalValue("document", parseDocument)
	defer parseRestoreDocument()

	parseWrapped, parseErr := GetDocument()
	if parseErr != nil {
		parseT.Fatalf("GetDocument: %v", parseErr)
	}
	if parseTitle, parseErr2 := parseWrapped.Title(); parseErr2 != nil || parseTitle != "RelayDesk – AI Chat Workspace" {
		parseT.Fatalf("Title() = %q, %v; want initial shim title", parseTitle, parseErr2)
	}
	if parseErr3 := parseWrapped.SetTitle("Thread preview — RelayDesk"); parseErr3 != nil {
		parseT.Fatalf("SetTitle: %v", parseErr3)
	}
	if parseRaw := parseDocument.Get("title").String(); parseRaw != "Thread preview — RelayDesk" {
		parseT.Fatalf("document.title = %q; want written value", parseRaw)
	}
	if parseTitle, parseErr4 := parseWrapped.Title(); parseErr4 != nil || parseTitle != "Thread preview — RelayDesk" {
		parseT.Fatalf("Title() after set = %q, %v", parseTitle, parseErr4)
	}
}
