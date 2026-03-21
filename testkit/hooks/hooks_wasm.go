//go:build js && wasm
// +build js,wasm

package hooks

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/testkit/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Harness wraps one rendered hook instance.
type Harness[T any] struct {
	tb      testing.TB
	fixture *render.Fixture
	hook    func() T
	current T
	cleaned bool
}

// RenderHook mounts a lightweight host component that evaluates one hook function.
func RenderHook[T any](tb testing.TB, hook func() T) *Harness[T] {
	tb.Helper()
	harness := &Harness[T]{
		tb:   tb,
		hook: hook,
	}
	harness.fixture = render.New(tb)
	harness.render()
	tb.Cleanup(func() {
		harness.Cleanup()
	})
	return harness
}

// Current returns the latest committed hook result.
func (h *Harness[T]) Current() T {
	h.requireActive()
	return h.current
}

// Rerender re-invokes the hook through the host component and settles the fixture.
func (h *Harness[T]) Rerender() {
	h.requireActive()
	h.render()
}

// Flush settles pending scheduled work for the hook host.
func (h *Harness[T]) Flush() {
	h.requireActive()
	h.fixture.Stabilize()
}

// Act runs one mutation and then settles the hook host.
func (h *Harness[T]) Act(fn func()) {
	h.tb.Helper()
	h.requireActive()
	if fn != nil {
		fn()
	}
	h.Flush()
}

// Cleanup releases the underlying render fixture.
func (h *Harness[T]) Cleanup() {
	if h == nil || h.cleaned {
		return
	}
	h.cleaned = true
	if h.fixture != nil {
		h.fixture.Cleanup()
		h.fixture = nil
	}
}

func (h *Harness[T]) render() {
	h.fixture.Render(ui.CreateElement(func() ui.Node {
		h.current = h.hook()
		return html.Div(html.Props{ID: "hook-host"})
	}))
}

func (h *Harness[T]) requireActive() {
	if h == nil || h.cleaned || h.fixture == nil {
		h.tb.Fatal("hook harness is no longer active")
	}
}
