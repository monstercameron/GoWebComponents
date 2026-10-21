package runtime

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v7/internal/errcode"
)

// hookOrderTracker keeps the previous render's hook-kind sequence so a render
// can be compared against it. It exists only while hookSignatureRecordingEnabled
// is true; production wasm never allocates one.
type hookOrderTracker struct {
	// prev is the hook-kind sequence of the last completed render. Its backing
	// array is swapped with Hooks.signature every render, so steady-state
	// renders allocate nothing.
	prev []string
	// hasPrev is true when prev came from a render that ran to completion.
	hasPrev bool
	// isCurComplete is true once the in-flight (or latest) render finished.
	isCurComplete bool
	// isReported latches after the first mismatch so a component that keeps
	// breaking the rule produces exactly one diagnostic.
	isReported bool
	// reportedDeps latches one unstable-dependency diagnostic per hook slot.
	reportedDeps map[string]struct{}
	// isListReported latches the one handler-in-list diagnostic per component.
	isListReported bool
}

// hookKindLabels maps recorded hook kinds to the public hook names shown in
// hook-order diagnostics.
var hookKindLabels = map[string]string{
	"state":    "UseState",
	"effect":   "UseEffect",
	"memo":     "UseMemo",
	"callback": "UseCallback",
	"ref":      "UseRef",
	"id":       "UseId",
	"context":  "UseContext",
	"fetch":    "UseFetch",
	"atom":     "UseAtom",
	"func":     "UseEvent",
}

// getHookKindLabel returns the public hook name for one recorded hook kind.
func getHookKindLabel(parseKind string) string {
	if parseLabel, parseOk := hookKindLabels[parseKind]; parseOk {
		return parseLabel
	}
	return parseKind
}

// beginHookOrderRender rotates the recorded hook sequence at the start of a
// re-render so the finished render becomes the comparison baseline.
func beginHookOrderRender(parseHooks *Hooks) {
	if !hookSignatureRecordingEnabled || parseHooks == nil {
		return
	}
	if parseHooks.order == nil {
		// A completed hookless first render keeps a non-nil empty signature
		// instead of allocating a tracker. Carry that baseline into this render
		// so adding a hook still reports a zero-to-one length change.
		parseWasEmptyComplete := parseHooks.signature != nil && len(parseHooks.signature) == 0
		parseHooks.order = &hookOrderTracker{isCurComplete: parseWasEmptyComplete}
	}
	parseOrder := parseHooks.order
	parseOrder.hasPrev = parseOrder.isCurComplete
	parseOrder.isCurComplete = false
	parseOrder.prev, parseHooks.signature = parseHooks.signature, parseOrder.prev[:0]
}

// checkHookOrderSlot reports a kind swap when the hook about to be recorded
// differs from the hook the previous render called at the same slot.
func checkHookOrderSlot(parseHooks *Hooks, parseKind string) {
	if !hookSignatureRecordingEnabled {
		return
	}
	parseOrder := parseHooks.order
	if parseOrder == nil || !parseOrder.hasPrev || parseOrder.isReported {
		return
	}
	parseSlot := len(parseHooks.signature)
	if parseSlot >= len(parseOrder.prev) || parseOrder.prev[parseSlot] == parseKind {
		return
	}
	reportHookOrderMismatch(parseHooks, fmt.Sprintf("slot %d: was %s, now %s",
		parseSlot, getHookKindLabel(parseOrder.prev[parseSlot]), getHookKindLabel(parseKind)))
}

// finishHookOrderRender reports a length change once a re-render completes and
// marks the render as a valid baseline for the next one.
func finishHookOrderRender(parseHooks *Hooks) {
	if !hookSignatureRecordingEnabled || parseHooks == nil {
		return
	}
	// A hookless first render needs only a zero-length baseline. Defer the
	// tracker allocation until a second render actually needs comparison.
	if parseHooks.order == nil && len(parseHooks.signature) == 0 {
		parseHooks.signature = []string{}
		return
	}
	if parseHooks.order == nil {
		parseHooks.order = &hookOrderTracker{}
	}
	parseOrder := parseHooks.order
	if parseOrder.hasPrev && !parseOrder.isReported && len(parseHooks.signature) != len(parseOrder.prev) {
		reportHookOrderMismatch(parseHooks, fmt.Sprintf("hook count changed: %d hooks last render, %d now",
			len(parseOrder.prev), len(parseHooks.signature)))
	}
	parseOrder.isCurComplete = true
}

// reportHookOrderMismatch emits the one-time GWC-RUNTIME-HOOK-ORDER diagnostic
// for a fiber and latches the tracker.
func reportHookOrderMismatch(parseHooks *Hooks, parseDetail string) {
	parseHooks.order.isReported = true
	parseFiber := parseHooks.owner
	ReportDiagnosticWithContext(
		"runtime",
		DiagnosticWarning,
		errcode.New(errcode.RuntimeHookOrder, parseDetail+"; hooks must be called in the same order every render; do not call hooks in conditions or loops").Error(),
		diagnosticPathWithSource(parseFiber),
		diagnosticComponentStackWithSource(parseFiber),
	)
}
