//go:build js && wasm
// +build js,wasm

package hooks

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

type counterHookState struct {
	Count     int
	Increment func()
}

func useCounterHook() counterHookState {
	count := ui.UseState(0)
	return counterHookState{
		Count: count.Get(),
		Increment: func() {
			count.Update(func(previous int) int {
				return previous + 1
			})
		},
	}
}

func TestRenderHookTracksStateTransitions(t *testing.T) {
	harness := RenderHook(t, useCounterHook)
	if got := harness.Current().Count; got != 0 {
		t.Fatalf("expected initial hook count 0, got %d", got)
	}

	harness.Act(func() {
		harness.Current().Increment()
	})

	if got := harness.Current().Count; got != 1 {
		t.Fatalf("expected incremented hook count 1, got %d", got)
	}
}

func TestRenderHookSupportsExplicitRerender(t *testing.T) {
	label := "first"
	harness := RenderHook(t, func() string {
		return label
	})
	if got := harness.Current(); got != "first" {
		t.Fatalf("expected initial hook value, got %q", got)
	}

	label = "second"
	harness.Rerender()

	if got := harness.Current(); got != "second" {
		t.Fatalf("expected rerendered hook value, got %q", got)
	}
}
