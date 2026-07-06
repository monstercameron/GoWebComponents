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
		delete(detachedRuntimes, parseSelector)
		detachedRuntimeMu.Unlock()
		if parseOK {
			parseRt.Render(nil, parseContainer)
		}
		return nil
	}
	if !parseOK {
		parseRt = NewRuntime(Config{
			DOMAdapter:   parseGlobal.domAdapter,
			EventAdapter: parseGlobal.eventAdapter,
			Scheduler:    parseGlobal.scheduler,
		})
		detachedRuntimes[parseSelector] = parseRt
	}
	detachedRuntimeMu.Unlock()

	parseRt.Render(parseElement, parseContainer)
	return nil
}
