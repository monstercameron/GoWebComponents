//go:build js && wasm

package hooks_test

import (
	"testing"

	hooks "github.com/monstercameron/GoWebComponents/v6/test/hooks"
	base "github.com/monstercameron/GoWebComponents/v6/testkit/hooks"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type parityHookValue struct {
	Count     int
	Increment func()
}

func useParityCounterHook() parityHookValue {
	parseCount := ui.UseState(0)
	return parityHookValue{
		Count: parseCount.Get(),
		Increment: func() {
			parseCount.Update(func(parsePrevious int) int {
				return parsePrevious + 1
			})
		},
	}
}

func TestPreferredHookWrappersMatchCompatibilityAliasBehavior(parseT *testing.T) {
	parsePreferred := hooks.RenderHook(parseT, useParityCounterHook)
	parsePreferred.Act(parsePreferred.Current().Increment)
	parsePreferredCount := parsePreferred.Current().Count
	parsePreferred.Cleanup()

	parseCompat := base.RenderHook(parseT, useParityCounterHook)
	parseCompat.Act(parseCompat.Current().Increment)
	parseCompatCount := parseCompat.Current().Count
	parseCompat.Cleanup()

	if parsePreferredCount != parseCompatCount {
		parseT.Fatalf("expected preferred wrapper and compatibility alias to match, got preferred=%d compat=%d", parsePreferredCount, parseCompatCount)
	}
}
