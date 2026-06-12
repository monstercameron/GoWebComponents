//go:build !js || !wasm

package hooks

import "testing"

// Harness wraps one rendered hook instance.
type Harness[T any] struct{}

// RenderHook requires a js/wasm test environment because interactive hooks only
// use the real runtime on browser-targeted builds.
func RenderHook[T any](parseTb testing.TB, parseHook func() T) *Harness[T] {
	parseTb.Helper()
	parseTb.Fatalf("testkit/hooks requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
	return nil
}

func (parseH *Harness[T]) Current() T         { var parseZero T; return parseZero }
func (parseH *Harness[T]) Rerender()          {}
func (parseH *Harness[T]) Flush()             {}
func (parseH *Harness[T]) Act(parseFn func()) {}
func (parseH *Harness[T]) Cleanup()           {}
