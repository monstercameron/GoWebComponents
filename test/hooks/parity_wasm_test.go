//go:build js && wasm
// +build js,wasm

package hooks_test

import (
	"testing"

	hooks "github.com/monstercameron/GoWebComponents/test/hooks"
	base "github.com/monstercameron/GoWebComponents/testkit/hooks"
	"github.com/monstercameron/GoWebComponents/ui"
)

type parityHookValue struct {
	Count     int
	Increment func()
}

func useParityCounterHook() parityHookValue {
	count := ui.UseState(0)
	return parityHookValue{
		Count: count.Get(),
		Increment: func() {
			count.Update(func(previous int) int {
				return previous + 1
			})
		},
	}
}

func TestPreferredHookWrappersMatchCompatibilityAliasBehavior(t *testing.T) {
	preferred := hooks.RenderHook(t, useParityCounterHook)
	preferred.Act(preferred.Current().Increment)
	preferredCount := preferred.Current().Count
	preferred.Cleanup()

	compat := base.RenderHook(t, useParityCounterHook)
	compat.Act(compat.Current().Increment)
	compatCount := compat.Current().Count
	compat.Cleanup()

	if preferredCount != compatCount {
		t.Fatalf("expected preferred wrapper and compatibility alias to match, got preferred=%d compat=%d", preferredCount, compatCount)
	}
}
