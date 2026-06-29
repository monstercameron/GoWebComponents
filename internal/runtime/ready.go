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

var (
	firstCommitMu    sync.Mutex
	firstCommitHooks []func()
	firstCommitDone  bool
)

// OnFirstCommit registers fn to run once, immediately after the first commit. If
// the first commit has already happened, fn runs synchronously now. A nil fn is
// ignored.
func OnFirstCommit(parseFn func()) {
	if parseFn == nil {
		return
	}
	firstCommitMu.Lock()
	if firstCommitDone {
		firstCommitMu.Unlock()
		parseFn()
		return
	}
	firstCommitHooks = append(firstCommitHooks, parseFn)
	firstCommitMu.Unlock()
}

// fireFirstCommitHooks runs the registered hooks exactly once, on the first
// completed commit. Subsequent commits are no-ops. Hooks run outside the lock so
// a hook may itself call OnFirstCommit (it then runs immediately) without
// deadlocking, and a re-entrant commit is ignored because the done flag is
// already set.
func fireFirstCommitHooks() {
	firstCommitMu.Lock()
	if firstCommitDone {
		firstCommitMu.Unlock()
		return
	}
	firstCommitDone = true
	parseHooks := firstCommitHooks
	firstCommitHooks = nil
	firstCommitMu.Unlock()

	for _, parseHook := range parseHooks {
		parseHook()
	}
}

// ResetFirstCommitHooksForTest restores the pre-commit state. Test-only.
func ResetFirstCommitHooksForTest() {
	firstCommitMu.Lock()
	firstCommitDone = false
	firstCommitHooks = nil
	firstCommitMu.Unlock()
}
