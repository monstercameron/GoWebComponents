package runtime

// Crash containment: every framework-owned goroutine and host callback runs
// behind these guards so an application panic produces an agent-readable
// console report instead of exiting the Go program. In wasm a single escaped
// panic kills the whole page; containment keeps the committed UI alive.

import "github.com/monstercameron/GoWebComponents/v5/interop"

// init wires interop host callbacks into the structured crash reporting; the
// interop package cannot import this package directly (import cycle).
func init() {
	interop.SetContainedPanicHandler(func(parseSubject string, parseRecovered any) {
		ContainPanic("interop", PanicPhaseAsync, parseSubject, parseRecovered)
	})
}

// ContainPanic reports a recovered panic through the structured panic-report
// pipeline and never re-panics. Call it from a deferred recover at a
// containment boundary (goroutine top, host callback, work loop).
func ContainPanic(parseSource string, parsePhase PanicPhase, parseSubject string, parseRecovered any) {
	if parseRecovered == nil {
		return
	}
	defer func() { _ = recover() }()
	if parseOriginal, isParseReported := unwrapReportedPanic(parseRecovered); isParseReported {
		// Already reported at an inner boundary; swallow without duplicating.
		_ = parseOriginal
		return
	}
	emitWrappedPanicReport(buildUnhandledPanicReport(parseSource, parsePhase, parseSubject, "", nil, parseRecovered))
}

// RecoverContainedPanic is the deferred form of ContainPanic: place
// `defer runtime.RecoverContainedPanic(source, subject)` at the top of any
// goroutine or host callback that must never crash the program.
func RecoverContainedPanic(parseSource string, parseSubject string) {
	if parseRecovered := recover(); parseRecovered != nil {
		ContainPanic(parseSource, PanicPhaseAsync, parseSubject, parseRecovered)
	}
}

// containAsyncPanic is the package-internal alias of RecoverContainedPanic.
func containAsyncPanic(parseSource string, parseSubject string) {
	if parseRecovered := recover(); parseRecovered != nil {
		ContainPanic(parseSource, PanicPhaseAsync, parseSubject, parseRecovered)
	}
}

// SafeGo starts fn on a new goroutine with panic containment. A panic inside
// fn is reported to the console (agent-readable, with stack buckets) and the
// goroutine ends; the program and the committed UI keep running.
func SafeGo(parseSource string, parseSubject string, parseFn func()) {
	if parseFn == nil {
		return
	}
	go func() {
		defer containAsyncPanic(parseSource, parseSubject)
		parseFn()
	}()
}

// GuardCallback wraps fn with panic containment for host-callback boundaries
// (js.FuncOf bodies, timer callbacks, observer callbacks). The returned
// function never lets a panic escape into the host bridge.
func GuardCallback(parseSource string, parseSubject string, parseFn func()) func() {
	if parseFn == nil {
		return func() {}
	}
	return func() {
		defer containAsyncPanic(parseSource, parseSubject)
		parseFn()
	}
}

// recoverWorkLoopState abandons an in-flight render after a contained work
// loop panic so the runtime can accept future updates. The last committed
// tree (currentRoot) is left untouched: the page stays partially alive
// instead of wedged on corrupt work-in-progress state.
//
// "Left untouched" was not true of its LINKS. The bailout path shares fiber
// objects between the committed and work-in-progress trees and repoints their
// .parent at the WIP fiber, so dropping wipRoot here left the committed tree's
// descendants anchored to a discarded root — and every later update targeting
// them resolved to nil and vanished without a diagnostic. Re-anchoring makes the
// promise in the paragraph above actually hold. See restoreCommittedTreeLinks.
func (parseRt *Runtime) recoverWorkLoopState() {
	if parseRt == nil {
		return
	}
	restoreCommittedTreeLinks(parseRt.currentRoot)
	parseRt.nextUnitOfWork = nil
	parseRt.wipRoot = nil
	if parseRt.deletions != nil {
		clear(parseRt.deletions)
		parseRt.deletions = parseRt.deletions[:0]
	}
	parseRt.pendingEffectFibers = nil
	parseRt.tracksPendingEffects = false
	parseRt.updateScheduled = false
	parseRt.pendingBoundaryRecovery = false
	parseRt.hydrating = false
}
