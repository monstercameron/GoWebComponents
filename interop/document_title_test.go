package interop

import (
	"strings"
	"testing"
)

// TestDocumentTitleContract pins the Document.Title/SetTitle wrapper contract:
// wired functions are delegated to, and the zero value (native/SSR builds)
// returns the structured unavailable error instead of panicking.
func TestDocumentTitleContract(parseT *testing.T) {
	parseStored := "RelayDesk"
	parseDocument := Document{
		title:    func() (string, error) { return parseStored, nil },
		setTitle: func(parseNext string) error { parseStored = parseNext; return nil },
	}
	if parseErr := parseDocument.SetTitle("Thread — RelayDesk"); parseErr != nil {
		parseT.Fatalf("SetTitle: %v", parseErr)
	}
	if parseTitle, parseErr := parseDocument.Title(); parseErr != nil || parseTitle != "Thread — RelayDesk" {
		parseT.Fatalf("Title() = %q, %v; want round-tripped title", parseTitle, parseErr)
	}

	parseZero := Document{}
	if _, parseErr := parseZero.Title(); parseErr == nil || !strings.Contains(parseErr.Error(), "Document.Title") {
		parseT.Fatalf("expected unavailable error from zero-value Title, got %v", parseErr)
	}
	if parseErr := parseZero.SetTitle("x"); parseErr == nil || !strings.Contains(parseErr.Error(), "Document.SetTitle") {
		parseT.Fatalf("expected unavailable error from zero-value SetTitle, got %v", parseErr)
	}
}
