package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// multiRootActive turns on after the first isolated (detached or per-island)
// runtime is created. Hook claims use the current fiber's direct owner when
// available, then walk ancestors only when an isolated root is active.
var multiRootActive atomic.Bool

// getHookRuntime returns the runtime a hook claimed during render should
// schedule its updates on. A state slot bound to the global runtime while its
// fiber lives in an isolated root would mark the wrong tree dirty. Rendered
// fibers carry their owner directly; older ownerless fibers retain the
// multi-root ancestor walk and global fallback.
//
//nolint:unused // called by shim.go and hooks, which are js/wasm-only.
func getHookRuntime() *Runtime {
	if parseFiber := GetCurrentFiber(); parseFiber != nil && parseFiber.ownerRuntime != nil {
		return parseFiber.ownerRuntime
	}
	if multiRootActive.Load() {
		return ResolveRuntime()
	}
	return GetGlobalRuntime()
}

// hydrationRootMu guards hydrationRoots.
var hydrationRootMu sync.Mutex

// hydrationRoots records one isolated runtime per additional hydrated container.
//
// A slice scanned with IsSameDOMNode rather than a map: DOMNode wrappers are not
// comparable by identity (a browser adapter hands out a fresh wrapper per
// lookup), so equality has to go through the node's own Equals. The list is as
// long as the number of independently hydrated islands on a page.
var hydrationRoots []hydrationRootEntry

// hydrationRootEntry pairs one hydrated container with the runtime that owns it.
type hydrationRootEntry struct {
	container DOMNode
	runtime   *Runtime
}

// rootContainerLocked reports the container this runtime's root renders into,
// or nil when it has no root yet. schedulerMu must be held.
func (parseRt *Runtime) rootContainerLocked() DOMNode {
	if parseRt.wipRoot != nil && !IsDOMNodeNull(parseRt.wipRoot.dom) {
		return parseRt.wipRoot.dom
	}
	if parseRt.currentRoot != nil && !IsDOMNodeNull(parseRt.currentRoot.dom) {
		return parseRt.currentRoot.dom
	}
	return nil
}

// ResolveHydrationRuntime picks the runtime that should hydrate parseTarget so
// that independent containers (islands) never share one root.
//
// The first container hydrated on a page keeps using the global runtime,
// unchanged. Hydrating the same container again keeps using whichever runtime
// already owns it. A different container gets its own isolated runtime that
// shares the global runtime's DOM, event and scheduler adapters, the same way
// RenderDetached isolates agent mounts. Without this, Runtime.Hydrate would
// build a second ROOT fiber against the single currentRoot and reconcile the
// first island away.
func ResolveHydrationRuntime(parseTarget any) (*Runtime, error) {
	parseGlobal := GetGlobalRuntime()
	if parseGlobal.domAdapter == nil {
		return nil, fmt.Errorf("ResolveHydrationRuntime: global runtime has no DOM adapter")
	}
	parseContainer := parseGlobal.resolveContainer(parseTarget)
	if IsDOMNodeNull(parseContainer) {
		return nil, fmt.Errorf("ResolveHydrationRuntime: target node could not be resolved")
	}

	schedulerMu.Lock()
	parseGlobalContainer := parseGlobal.rootContainerLocked()
	schedulerMu.Unlock()
	if IsDOMNodeNull(parseGlobalContainer) || IsSameDOMNode(parseGlobalContainer, parseContainer) {
		return parseGlobal, nil
	}

	hydrationRootMu.Lock()
	defer hydrationRootMu.Unlock()
	// Entries built against a previous global runtime (tests and hot restarts
	// re-initialise it) point at a DOM the new one no longer drives; drop them.
	parseLive := hydrationRoots[:0]
	for _, parseEntry := range hydrationRoots {
		if parseEntry.runtime.domAdapter == parseGlobal.domAdapter {
			parseLive = append(parseLive, parseEntry)
		}
	}
	clear(hydrationRoots[len(parseLive):])
	hydrationRoots = parseLive
	for _, parseEntry := range hydrationRoots {
		if IsSameDOMNode(parseEntry.container, parseContainer) {
			return parseEntry.runtime, nil
		}
	}
	multiRootActive.Store(true)
	parseRt := NewRuntime(Config{
		DOMAdapter:   parseGlobal.domAdapter,
		EventAdapter: parseGlobal.eventAdapter,
		Scheduler:    parseGlobal.scheduler,
	})
	hydrationRoots = append(hydrationRoots, hydrationRootEntry{container: parseContainer, runtime: parseRt})
	return parseRt, nil
}
