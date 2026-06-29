//go:build !js || !wasm

package ui

import (
	"strings"
	"testing"
)

// TestInspectMessageFormatsChanges proves the pure formatter: an init line on mount, an
// old->new line on change, and no line when unchanged.
func TestInspectMessageFormatsChanges(parseT *testing.T) {
	if parseMsg, parseOk := inspectMessage("count", 0, 5, true); !parseOk || !strings.Contains(parseMsg, "count = 5 (init)") {
		parseT.Fatalf("init: got %q ok=%v", parseMsg, parseOk)
	}
	if parseMsg, parseOk := inspectMessage("count", 5, 7, false); !parseOk || !strings.Contains(parseMsg, "count: 5 -> 7") {
		parseT.Fatalf("change: got %q ok=%v", parseMsg, parseOk)
	}
	if parseMsg, parseOk := inspectMessage("count", 7, 7, false); parseOk || parseMsg != "" {
		parseT.Fatalf("unchanged should emit nothing, got %q ok=%v", parseMsg, parseOk)
	}
}

// TestInspectMessageStructuralEquality proves change detection uses structural (deep)
// equality, so equal composite values do not log.
func TestInspectMessageStructuralEquality(parseT *testing.T) {
	parsePrev := []string{"a", "b"}
	parseSame := []string{"a", "b"}
	parseDiff := []string{"a", "c"}

	if _, parseOk := inspectMessage("list", parsePrev, parseSame, false); parseOk {
		parseT.Fatal("structurally equal slices should not log a change")
	}
	if _, parseOk := inspectMessage("list", parsePrev, parseDiff, false); !parseOk {
		parseT.Fatal("differing slices should log a change")
	}
}

// TestSetInspectSinkRoutesAndRestores proves the sink can be redirected (e.g. to a test
// buffer or devtools) and restored.
func TestSetInspectSinkRoutesAndRestores(parseT *testing.T) {
	var parseCaptured []string
	parseRestore := SetInspectSink(func(parseMessage string) { parseCaptured = append(parseCaptured, parseMessage) })

	emitInspect("hello")
	if len(parseCaptured) != 1 || parseCaptured[0] != "hello" {
		parseT.Fatalf("expected the redirected sink to capture the message, got %v", parseCaptured)
	}

	parseRestore()
	emitInspect("after-restore") // must not reach the captured buffer
	if len(parseCaptured) != 1 {
		parseT.Fatalf("expected the sink to be restored, got %v", parseCaptured)
	}
}
