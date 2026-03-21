//go:build !js || !wasm
// +build !js !wasm

package hooks

import "testing"

// Harness wraps one rendered hook instance.
type Harness[T any] struct{}

// RenderHook requires a js/wasm test environment because interactive hooks only
// use the real runtime on browser-targeted builds.
func RenderHook[T any](tb testing.TB, hook func() T) *Harness[T] {
	tb.Helper()
	tb.Fatalf("testkit/hooks requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
	return nil
}

func (h *Harness[T]) Current() T    { var zero T; return zero }
func (h *Harness[T]) Rerender()     {}
func (h *Harness[T]) Flush()        {}
func (h *Harness[T]) Act(fn func()) {}
func (h *Harness[T]) Cleanup()      {}
