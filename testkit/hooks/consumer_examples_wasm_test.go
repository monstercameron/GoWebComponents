//go:build js && wasm
// +build js,wasm

package hooks_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/testkit/hooks"
	"github.com/monstercameron/GoWebComponents/ui"
)

type counterHookValue struct {
	Count     int
	SelectOne func()
}

func useCounterHook() counterHookValue {
	count := ui.UseState(0)
	return counterHookValue{
		Count: count.Get(),
		SelectOne: func() {
			count.Set(1)
		},
	}
}

func TestConsumerHookPattern_RenderHook(t *testing.T) {
	harness := hooks.RenderHook(t, useCounterHook)

	harness.Act(harness.Current().SelectOne)

	if got := harness.Current().Count; got != 1 {
		t.Fatalf("expected hook state to update, got %d", got)
	}
}
