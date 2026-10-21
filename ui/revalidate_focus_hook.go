package ui

import (
	"sync"

	"github.com/monstercameron/GoWebComponents/v7/query"
)

// queryRevalidators holds, per cache, the refetch callbacks of every mounted UseQuery. Marking
// entries stale alone refetches nothing (UseQuery's effect is keyed on [key, deps] and never
// re-runs), so the focus signal must also call these. Guarded by queryRevalidatorsMu.
var (
	queryRevalidatorsMu     sync.Mutex
	queryRevalidatorsNextID uint64
	queryRevalidators       = map[*query.Cache]map[uint64]func(){}
)

// storeQueryRevalidator registers a refetch callback for parseCache and returns its unregister
// func. Callbacks join any in-flight fetch through query.SWR, so many subscribers never storm.
func storeQueryRevalidator(parseCache *query.Cache, parseRefetch func()) func() {
	queryRevalidatorsMu.Lock()
	defer queryRevalidatorsMu.Unlock()
	queryRevalidatorsNextID++
	parseID := queryRevalidatorsNextID
	if queryRevalidators[parseCache] == nil {
		queryRevalidators[parseCache] = map[uint64]func(){}
	}
	queryRevalidators[parseCache][parseID] = parseRefetch
	return func() {
		queryRevalidatorsMu.Lock()
		defer queryRevalidatorsMu.Unlock()
		delete(queryRevalidators[parseCache], parseID)
		if len(queryRevalidators[parseCache]) == 0 {
			delete(queryRevalidators, parseCache)
		}
	}
}

// handleQueryRevalidate runs every mounted UseQuery refetch callback for parseCache. Callbacks
// are copied out first so a callback may mount or unmount queries without deadlocking.
func handleQueryRevalidate(parseCache *query.Cache) {
	queryRevalidatorsMu.Lock()
	parseCallbacks := make([]func(), 0, len(queryRevalidators[parseCache]))
	for _, parseCallback := range queryRevalidators[parseCache] {
		parseCallbacks = append(parseCallbacks, parseCallback)
	}
	queryRevalidatorsMu.Unlock()
	for _, parseCallback := range parseCallbacks {
		parseCallback()
	}
}

// revalidateOnSignal is the action behind UseRevalidateOnFocus: it marks every cached query
// stale and triggers a refetch in every mounted UseQuery on that cache (deduped by query.SWR).
// Split out so the behavior is testable without the window-event lifecycle.
func revalidateOnSignal(parseCache *query.Cache) {
	if parseCache != nil {
		parseCache.InvalidateAll()
		handleQueryRevalidate(parseCache)
	}
}

// UseRevalidateOnFocus wires the standard SWR freshness triggers: when the tab regains focus
// or the browser comes back online, every cached query is marked stale and the component
// re-renders, so its UseQuery hooks revalidate against the server. Mount it once near the app
// root. It is a no-op on native/SSR (no window).
//
//	func App(props AppProps) ui.Node {
//	    ui.UseRevalidateOnFocus(appCache) // refetch stale data on focus / reconnect
//	    ...
//	}
func UseRevalidateOnFocus(parseCache *query.Cache) {
	parseRerender := UseForceUpdate()
	parseHandler := func(Event) {
		revalidateOnSignal(parseCache)
		parseRerender()
	}
	UseWindowEvent("focus", parseHandler)
	UseWindowEvent("online", parseHandler)
}
