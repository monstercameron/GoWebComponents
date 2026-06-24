//go:build !(js && wasm)

package interop

import "context"

// Await is unavailable on non-browser builds. It returns a structured
// CodeUnavailable error so native unit tests of code paths that call Await fail
// loudly rather than silently.
func (parseV Value) Await(parseCtx context.Context) (Value, error) {
	_ = parseCtx
	return Value{}, unavailable("Value.Await", "")
}

// AwaitCall is unavailable on non-browser builds.
func (parseV Value) AwaitCall(parseCtx context.Context, parseMethod string, parseArgs ...any) (Value, error) {
	_ = parseCtx
	_ = parseMethod
	_ = parseArgs
	return Value{}, unavailable("Value.AwaitCall", "")
}
