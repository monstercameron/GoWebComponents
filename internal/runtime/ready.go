package runtime

import "sync"

// First-commit ("app ready") signal (G16).
//
// There is no built-in way for a host to know the initial render has painted, so
// apps hand-manage splash overlays and special-case "#app already has children".
// OnFirstCommit fires once, right after the first commit completes (the initial
// tree is on screen and its effects have run), giving a deterministic ready
// signal. ui.OnReady wraps this for Go callers; on wasm the ui layer also
// dispatches a "gwc:ready" DOM event so the host page can drop its splash.
//
// v5 P2.3: this state is PER RUNTIME. It used to be three package globals, so
// "first commit" meant first commit in the process — a second Runtime's initial
// render would find firstCommitDone already true and never fire its own ready
// hooks. That is a semantic singleton leak rather than a data race: every
// global involved was already mutex-guarded, so -race could never have found
// it, which is why P2.3's acceptance criterion is a behavioral conformance
// suite instead.

// readyState holds one runtime's first-commit signal.
type readyState struct {
	mu    sync.Mutex
	hooks []func()
	done  bool
}

// OnFirstCommit registers fn to run once, immediately after this runtime's
// first commit. If that commit already happened, fn runs synchronously now. A
// nil fn is ignored.
func (parseRt *Runtime) OnFirstCommit(parseFn func()) {
	if parseRt == nil || parseFn == nil {
		return
	}
	parseRt.ready.mu.Lock()
	if parseRt.ready.done {
		parseRt.ready.mu.Unlock()
		parseFn()
		return
	}
	parseRt.ready.hooks = append(parseRt.ready.hooks, parseFn)
	parseRt.ready.mu.Unlock()
}

// fireFirstCommitHooks runs this runtime's hooks exactly once, on its first
// completed commit. Subsequent commits are no-ops. Hooks run outside the lock so
// a hook may itself call OnFirstCommit (it then runs immediately) without
// deadlocking, and a re-entrant commit is ignored because the done flag is
// already set.
func (parseRt *Runtime) fireFirstCommitHooks() {
	if parseRt == nil {
		return
	}
	parseRt.ready.mu.Lock()
	if parseRt.ready.done {
		parseRt.ready.mu.Unlock()
		return
	}
	parseRt.ready.done = true
	parseHooks := parseRt.ready.hooks
	parseRt.ready.hooks = nil
	parseRt.ready.mu.Unlock()

	for _, parseHook := range parseHooks {
		parseHook()
	}
}

// HasCommitted reports whether this runtime has completed its first commit.
func (parseRt *Runtime) HasCommitted() bool {
	if parseRt == nil {
		return false
	}
	parseRt.ready.mu.Lock()
	defer parseRt.ready.mu.Unlock()
	return parseRt.ready.done
}

// OnFirstCommit registers fn against the global runtime.
//
// Retained so existing callers (ui.OnReady and the wasm "gwc:ready" bridge)
// keep working unchanged; apps with one runtime — which is every app today —
// see no difference.
func OnFirstCommit(parseFn func()) {
	GetGlobalRuntime().OnFirstCommit(parseFn)
}

// ResetFirstCommitHooksForTest restores the global runtime's pre-commit state.
// Test-only.
func ResetFirstCommitHooksForTest() {
	GetGlobalRuntime().ResetFirstCommitHooksForTest()
}

// ResetFirstCommitHooksForTest restores this runtime's pre-commit state.
// Test-only.
func (parseRt *Runtime) ResetFirstCommitHooksForTest() {
	if parseRt == nil {
		return
	}
	parseRt.ready.mu.Lock()
	parseRt.ready.done = false
	parseRt.ready.hooks = nil
	parseRt.ready.mu.Unlock()
}
