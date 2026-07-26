package runtime

// Runtime resolution for the package-level async entry points (v5 P2.1).
//
// Untagged on purpose. The exported wrappers live in shim.go, which is
// js/wasm-only, so the rule below could only ever be exercised in a browser —
// and it is a rule about goroutines and ownership, which is precisely what the
// native test suite and the race detector are for.

// postAsyncGlobal queues work on the GLOBAL runtime's async inbox.
//
// This used to resolve with ResolveRuntime, "so a post issued while a second
// runtime is rendering lands on that runtime rather than on whichever happened
// to be global". That reasoning inverts the API's own premise.
//
// ResolveRuntime walks up from currentFiber, which answers "which runtime is
// rendering RIGHT NOW". PostAsync exists for callers that are NOT on the frame
// loop — a worker reply, a gRPC callback, a goroutine — and for such a caller
// that answer is not merely unhelpful, it is arbitrary: the post lands on
// whichever runtime happened to be mid-render at that instant, or on the global
// one if none was. The same call site therefore targets different runtimes
// depending on timing, which is the one property a queue must not have.
//
// It is also a data race. currentFiber is package state the reconciler writes on
// every component render and treats as render-goroutine-owned; reading it from a
// producer goroutine is exactly the unsynchronized access the race detector
// exists to find (.github/workflows/race.yml).
//
// ResolveRuntime stays correct where it is actually used — P2.6's runtime-scoped
// atoms — because hooks run DURING render, on the render goroutine, so
// currentFiber is both meaningful and safely read. The distinction that matters
// is on-loop versus off-loop, not global versus scoped.
//
// An owner of a second runtime posts to it directly with rt.PostAsync, which is
// exact and needs no inference. §7 of the v5 plan already scopes the public
// surface to a single global runtime.
func postAsyncGlobal(parseWork func()) {
	GetGlobalRuntime().PostAsync(parseWork)
}

// asyncIngressEnabledGlobal reports whether the global runtime routes off-loop
// state writes through the inbox.
//
// Same resolution as postAsyncGlobal and for the same reason: a caller asking
// "will my off-loop write be queued?" is off-loop, so resolving through
// currentFiber would answer for a runtime other than the one it is about to
// post to.
func asyncIngressEnabledGlobal() bool {
	return GetGlobalRuntime().AsyncIngressEnabled()
}
