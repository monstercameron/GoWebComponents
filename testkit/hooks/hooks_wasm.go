//go:build js && wasm

package hooks

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/testkit/render"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// Harness wraps one rendered hook instance.
type Harness[T any] struct {
	tb      testing.TB
	fixture *render.Fixture
	hook    func() T
	current T
	tick    int
	cleaned bool
}

type hostProps struct {
	Tick int
}

// RenderHook mounts a lightweight host component that evaluates one hook function.
func RenderHook[T any](parseTb testing.TB, parseHook func() T) *Harness[T] {
	parseTb.Helper()
	parseHarness := &Harness[T]{
		tb:   parseTb,
		hook: parseHook,
	}
	parseHarness.fixture = render.New(parseTb)
	parseHarness.render()
	parseTb.Cleanup(func() {
		parseHarness.Cleanup()
	})
	return parseHarness
}

// Current returns the latest committed hook result.
func (parseH *Harness[T]) Current() T {
	parseH.requireActive()
	return parseH.current
}

// Rerender re-invokes the hook through the host component and settles the fixture.
func (parseH *Harness[T]) Rerender() {
	parseH.requireActive()
	parseH.render()
}

// Flush settles pending scheduled work for the hook host.
func (parseH *Harness[T]) Flush() {
	parseH.requireActive()
	parseH.fixture.Stabilize()
}

// Act runs one mutation and then settles the hook host.
func (parseH *Harness[T]) Act(parseFn func()) {
	parseH.tb.Helper()
	parseH.requireActive()
	if parseFn != nil {
		parseFn()
	}
	parseH.Flush()
}

// Cleanup releases the underlying render fixture.
func (parseH *Harness[T]) Cleanup() {
	if parseH == nil || parseH.cleaned {
		return
	}
	parseH.cleaned = true
	if parseH.fixture != nil {
		parseH.fixture.Cleanup()
		parseH.fixture = nil
	}
}

func (parseH *Harness[T]) render() {
	parseH.tick++
	parseH.fixture.Render(ui.CreateElement(func(_ hostProps) ui.Node {
		parseH.current = parseH.hook()
		return html.Div(html.Props{ID: "hook-host"})
	}, hostProps{Tick: parseH.tick}))
}

func (parseH *Harness[T]) requireActive() {
	if parseH == nil || parseH.cleaned || parseH.fixture == nil {
		parseH.tb.Fatal("hook harness is no longer active")
	}
}
