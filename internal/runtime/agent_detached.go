package runtime

import (
	"fmt"
	"sync"
)

// detachedRuntimeMu guards the per-selector detached runtime registry.
var detachedRuntimeMu sync.Mutex

// detachedRuntimes holds one isolated runtime per mount selector, reused across
// renders into the same container.
var detachedRuntimes = map[string]*Runtime{}

// detachedRuntimeOrder lists live selectors least-recently-rendered first, so
// the registry can evict without scanning. A slice rather than anything
// cleverer because it is bounded by maxDetachedRuntimes.
var detachedRuntimeOrder []string

// maxDetachedRuntimes bounds the registry.
//
// Entries are keyed by an AGENT-SUPPLIED selector and were only ever removed by
// an explicit nil render, so an agent that mounted into #panel-1, #panel-2, ...
// and never cleared them accumulated one full Runtime — fiber tree, atom
// registry, slabs and all — per selector, for the session.
//
// Thirty-two concurrent detached mounts is far past any real agent surface; the
// bound exists to make unbounded growth impossible, not to be reached.
const maxDetachedRuntimes = 32

// touchDetachedRuntimeLocked marks one selector as most-recently-rendered.
func touchDetachedRuntimeLocked(parseSelector string) {
	for parseIndex, parseExisting := range detachedRuntimeOrder {
		if parseExisting == parseSelector {
			detachedRuntimeOrder = append(detachedRuntimeOrder[:parseIndex], detachedRuntimeOrder[parseIndex+1:]...)
			break
		}
	}
	detachedRuntimeOrder = append(detachedRuntimeOrder, parseSelector)
}

// forgetDetachedRuntimeLocked drops one selector from the registry and its
// recency list.
func forgetDetachedRuntimeLocked(parseSelector string) {
	delete(detachedRuntimes, parseSelector)
	for parseIndex, parseExisting := range detachedRuntimeOrder {
		if parseExisting == parseSelector {
			// Release the vacated slot. touchDetachedRuntimeLocked does not need
			// this because it re-appends immediately, overwriting the slot; this
			// path only removes.
			copy(detachedRuntimeOrder[parseIndex:], detachedRuntimeOrder[parseIndex+1:])
			detachedRuntimeOrder[len(detachedRuntimeOrder)-1] = ""
			detachedRuntimeOrder = detachedRuntimeOrder[:len(detachedRuntimeOrder)-1]
			return
		}
	}
}

// evictOldestDetachedRuntimeLocked removes the least-recently-rendered entry
// and returns it so the caller can tear its tree down outside the lock.
//
// The recency list is treated as a hint over the map, not a second source of
// truth: it drops any selector the map no longer has (tests swap the map
// wholesale, and a nil render removes an entry directly), and falls back to an
// arbitrary key if the hint is exhausted while the map is still over its bound.
// The map decides what exists; the list only orders it.
func evictOldestDetachedRuntimeLocked() (string, *Runtime, bool) {
	for len(detachedRuntimeOrder) > 0 {
		parseSelector := detachedRuntimeOrder[0]
		detachedRuntimeOrder = detachedRuntimeOrder[1:]
		if parseRt, parseOK := detachedRuntimes[parseSelector]; parseOK {
			delete(detachedRuntimes, parseSelector)
			return parseSelector, parseRt, true
		}
	}
	for parseSelector, parseRt := range detachedRuntimes {
		delete(detachedRuntimes, parseSelector)
		return parseSelector, parseRt, true
	}
	return "", nil, false
}

// RenderDetached renders parseElement into the DOM container at parseSelector
// using a runtime INDEPENDENT of the global app runtime. Agent-injected or
// agent-mounted content therefore renders into its own root and can never
// clobber the app's single currentRoot (which a plain RenderTo into a second
// container would). Each selector gets its own isolated runtime, reused across
// calls, sharing the global runtime's DOM/event adapters and scheduler so the
// detached tree renders into the same live browser. A nil element clears the
// container.
func RenderDetached(parseSelector string, parseElement *Element) error {
	parseGlobal := GetGlobalRuntime()
	if parseGlobal == nil || parseGlobal.domAdapter == nil {
		return fmt.Errorf("RenderDetached: global runtime has no DOM adapter")
	}
	parseContainer := parseGlobal.queryContainer(parseSelector)
	if IsDOMNodeNull(parseContainer) {
		return fmt.Errorf("RenderDetached: selector %q did not resolve to a container", parseSelector)
	}

	detachedRuntimeMu.Lock()
	parseRt, parseOK := detachedRuntimes[parseSelector]
	if parseElement == nil {
		// A nil element clears the container AND evicts the registry entry:
		// the registry is keyed by agent-supplied selectors, so keeping one
		// full Runtime alive per selector ever unmounted grows without bound.
		forgetDetachedRuntimeLocked(parseSelector)
		detachedRuntimeMu.Unlock()
		if parseOK {
			parseRt.Render(nil, parseContainer)
		}
		return nil
	}
	parseEvictedSelector := ""
	var parseEvictedRt *Runtime
	hasEviction := false
	if !parseOK {
		if len(detachedRuntimes) >= maxDetachedRuntimes {
			parseEvictedSelector, parseEvictedRt, hasEviction = evictOldestDetachedRuntimeLocked()
		}
		parseRt = NewRuntime(Config{
			DOMAdapter:   parseGlobal.domAdapter,
			EventAdapter: parseGlobal.eventAdapter,
			Scheduler:    parseGlobal.scheduler,
		})
		detachedRuntimes[parseSelector] = parseRt
	}
	touchDetachedRuntimeLocked(parseSelector)
	detachedRuntimeMu.Unlock()

	// Tear the evicted tree down rather than only dropping the reference.
	// Abandoning a mounted tree would leave its DOM in the container with no
	// runtime owning it, and a later render into that selector would build a
	// fresh runtime that appends alongside the orphaned nodes instead of
	// reconciling against them. Done outside the lock: Render is reentrant work.
	//
	// A selector that no longer resolves has already left the document, so
	// there is nothing to tear down and dropping the reference is the whole job.
	if hasEviction && parseEvictedRt != nil {
		if parseEvictedContainer := parseGlobal.queryContainer(parseEvictedSelector); !IsDOMNodeNull(parseEvictedContainer) {
			parseEvictedRt.Render(nil, parseEvictedContainer)
		}
		ReportDiagnostic("runtime", DiagnosticInfo,
			"detached runtime registry evicted its least-recently-rendered mount to stay within its bound; "+
				"rendering into that selector again will remount it from scratch")
	}

	parseRt.Render(parseElement, parseContainer)
	return nil
}
