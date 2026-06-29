package ui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// persistedStringProbe renders the current value of a UsePersistedState[string]
// so RenderToString can assert the initial value on non-browser builds.
func persistedStringProbe(_ struct{}) ui.Node {
	parsePs := ui.UsePersistedState("test-key", "default-value", ui.PersistLocal)
	return html.Div(html.Props{ID: "persisted"},
		html.Text(fmt.Sprintf("value=%s err=%v", parsePs.Get(), parsePs.Err())),
	)
}

// TestPersistedStateNativeUsesInitialValue verifies that on native (non-browser)
// builds, UsePersistedState returns the initial value since storage is
// unavailable, and that Err() is nil (storage errors are swallowed gracefully).
func TestPersistedStateNativeUsesInitialValue(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(persistedStringProbe, struct{}{}))
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "value=default-value") {
		parseT.Fatalf("expected initial value to be rendered, got: %s", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "err=") || (!strings.Contains(parseMarkup, "err=&lt;nil&gt;") && !strings.Contains(parseMarkup, "err=<nil>")) {
		parseT.Fatalf("expected nil error on native build, got: %s", parseMarkup)
	}
}

type persistedItem struct {
	Name  string
	Score int
}

// persistedStructProbe renders a struct value to verify JSON marshalling
// compatibility with UsePersistedState[T].
func persistedStructProbe(_ struct{}) ui.Node {
	parseInitial := persistedItem{Name: "initial", Score: 42}
	parsePs := ui.UsePersistedState("struct-key", parseInitial, ui.PersistSession)
	parseGot := parsePs.Get()
	return html.Div(html.Props{ID: "struct-test"},
		html.Text(fmt.Sprintf("name=%s score=%d", parseGot.Name, parseGot.Score)),
	)
}

// TestPersistedStateNativeWithStruct verifies that UsePersistedState[T] works
// with a struct type on native builds and returns the initial struct value.
func TestPersistedStateNativeWithStruct(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(persistedStructProbe, struct{}{}))
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "name=initial") {
		parseT.Fatalf("expected struct name field, got: %s", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "score=42") {
		parseT.Fatalf("expected struct score field, got: %s", parseMarkup)
	}
}

// persistedSliceProbe renders a slice value to verify generic slice support.
func persistedSliceProbe(_ struct{}) ui.Node {
	parseInitial := []string{"a", "b", "c"}
	parsePs := ui.UsePersistedState("slice-key", parseInitial, ui.PersistLocal)
	parseGot := parsePs.Get()
	return html.Div(html.Props{ID: "slice-test"},
		html.Text(fmt.Sprintf("len=%d first=%s", len(parseGot), parseGot[0])),
	)
}

// TestPersistedStateNativeWithSlice verifies that UsePersistedState[[]string]
// degrades correctly to the in-memory initial value on native builds.
func TestPersistedStateNativeWithSlice(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(persistedSliceProbe, struct{}{}))
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "len=3") {
		parseT.Fatalf("expected slice length 3, got: %s", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "first=a") {
		parseT.Fatalf("expected first element 'a', got: %s", parseMarkup)
	}
}

// TestPersistStorageAreaConstants verifies the public PersistStorageArea values.
func TestPersistStorageAreaConstants(parseT *testing.T) {
	if ui.PersistLocal != "local" {
		parseT.Fatalf("expected PersistLocal='local', got %q", ui.PersistLocal)
	}
	if ui.PersistSession != "session" {
		parseT.Fatalf("expected PersistSession='session', got %q", ui.PersistSession)
	}
}

// persistedSetProbe exercises the Set path on native builds (write-through is
// a no-op but in-memory state should update within the same render).
func persistedSetProbe(_ struct{}) ui.Node {
	parsePs := ui.UsePersistedState("set-key", 0, ui.PersistLocal)
	parsePs.Set(99)
	return html.Div(html.Props{ID: "set-test"},
		html.Text(fmt.Sprintf("value=%d", parsePs.Get())),
	)
}

// TestPersistedStateNativeSetUpdatesInMemory verifies that calling Set on
// native builds still updates the in-memory value (write-through swallowed).
func TestPersistedStateNativeSetUpdatesInMemory(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(persistedSetProbe, struct{}{}))
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "value=99") {
		parseT.Fatalf("expected in-memory Set to produce value=99, got: %s", parseMarkup)
	}
}
