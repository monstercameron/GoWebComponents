//go:build js && wasm

package hooks_test

import (
	"testing"

	hooks "github.com/monstercameron/GoWebComponents/v5/test/hooks"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type counterHookValue struct {
	Count     int
	SelectOne func()
}

func useCounterHook() counterHookValue {
	parseCount := ui.UseState(0)
	return counterHookValue{
		Count: parseCount.Get(),
		SelectOne: func() {
			parseCount.Set(1)
		},
	}
}

func TestConsumerHookPattern_RenderHook(parseT *testing.T) {
	parseHarness := hooks.RenderHook(parseT, useCounterHook)

	parseHarness.Act(parseHarness.Current().SelectOne)

	if parseGot := parseHarness.Current().Count; parseGot != 1 {
		parseT.Fatalf("expected hook state to update, got %d", parseGot)
	}
}
