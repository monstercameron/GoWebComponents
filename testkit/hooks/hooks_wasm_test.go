//go:build js && wasm

package hooks

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

type counterHookState struct {
	Count     int
	Increment func()
}

func useCounterHook() counterHookState {
	parseCount := ui.UseState(0)
	return counterHookState{
		Count: parseCount.Get(),
		Increment: func() {
			parseCount.Update(func(parsePrevious int) int {
				return parsePrevious + 1
			})
		},
	}
}

func TestRenderHookTracksStateTransitions(parseT *testing.T) {
	parseHarness := RenderHook(parseT, useCounterHook)
	if parseGot := parseHarness.Current().Count; parseGot != 0 {
		parseT.Fatalf("expected initial hook count 0, got %d", parseGot)
	}

	parseHarness.Act(func() {
		parseHarness.Current().Increment()
	})

	if parseGot2 := parseHarness.Current().Count; parseGot2 != 1 {
		parseT.Fatalf("expected incremented hook count 1, got %d", parseGot2)
	}
}

func TestRenderHookSupportsExplicitRerender(parseT *testing.T) {
	parseLabel := "first"
	parseHarness := RenderHook(parseT, func() string {
		return parseLabel
	})
	if parseGot := parseHarness.Current(); parseGot != "first" {
		parseT.Fatalf("expected initial hook value, got %q", parseGot)
	}

	parseLabel = "second"
	parseHarness.Rerender()

	if parseGot2 := parseHarness.Current(); parseGot2 != "second" {
		parseT.Fatalf("expected rerendered hook value, got %q", parseGot2)
	}
}
