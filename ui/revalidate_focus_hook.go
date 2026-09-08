package ui

import "github.com/monstercameron/GoWebComponents/v6/query"

// revalidateOnSignal is the pure action behind UseRevalidateOnFocus: it marks every cached
// query stale so the next render's UseQuery components revalidate. Split out so the
// invalidation behavior is unit-testable without the window-event lifecycle.
func revalidateOnSignal(parseCache *query.Cache) {
	if parseCache != nil {
		parseCache.InvalidateAll()
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
