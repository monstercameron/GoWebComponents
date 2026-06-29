//go:build js && wasm

package interop

import "context"

// Await resolves a JavaScript Promise wrapped in a [Value] into a Go value,
// blocking the calling goroutine until the promise settles or the context is
// canceled. It is the public Promise→Go bridge: it collapses the manual
// js.FuncOf `then`/`catch` chain (with hand `Release()`) that every async Web
// API otherwise forces an app to re-author.
//
//	result, err := promise.Await(ctx)
//
// Behavior:
//   - If the value is not a thenable, it is returned unchanged (so Await is safe
//     to call on a synchronous result).
//   - If the promise rejects, Await returns a CodePromiseRejected error carrying
//     the rejection summary.
//   - If the context is canceled before the promise settles, Await returns the
//     context error and stops waiting. (The underlying js.Func callbacks are
//     released when the promise eventually settles; Await never leaks on the
//     happy path or on rejection, and the context path simply stops blocking.)
//
// Call Await from a goroutine (for example via ui.SafeGo), never directly inside
// a render or event callback, because it blocks until the promise settles.
func (parseV Value) Await(parseCtx context.Context) (Value, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseRaw, parseOk := parseV.rawValue()
	if !parseOk {
		return Value{}, unavailable("Value.Await", "")
	}
	parseResolved, parseErr := awaitValue(parseCtx, "Value.Await", "", parseRaw)
	if parseErr != nil {
		return Value{}, parseErr
	}
	return Value{raw: parseResolved}, nil
}

// AwaitCall invokes a method on the value and awaits the returned promise in one
// step — the common shape for promise-returning Web APIs
// (`navigator.storage.estimate()`, `caches.open(...)`, etc.).
//
//	estimate, err := navigatorStorage.AwaitCall(ctx, "estimate")
func (parseV Value) AwaitCall(parseCtx context.Context, parseMethod string, parseArgs ...any) (Value, error) {
	parsePromise, parseErr := parseV.Call(parseMethod, parseArgs...)
	if parseErr != nil {
		return Value{}, parseErr
	}
	return parsePromise.Await(parseCtx)
}
