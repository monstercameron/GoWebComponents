package kvstate

// Invalidate reloads mounted bindings for a logical database after an external
// durable commit. It never publishes a BroadcastChannel event or writes storage,
// so native commit notifications cannot create an echo loop. It affects existing
// subscriptions only; newly mounted bindings hydrate normally from their backend.
// Call it from an asynchronous task, not a synchronous JS event callback: reloads
// may await a remote backend. Multiple commit notifications may safely coalesce.
func Invalidate(parseName string) {
	if parseName == "" {
		parseName = "gwc"
	}
	hubsMu.Lock()
	parseHub := hubs[parseName]
	parseExternal := externalHubs[parseName]
	hubsMu.Unlock()
	invalidateHub(parseHub)
	invalidateHub(parseExternal)
}

// invalidateHub snapshots callbacks so reloads and unsubscribe never hold hub locks.
func invalidateHub(parseHub *crossTabHub) {
	if parseHub == nil {
		return
	}
	parseHub.mu.Lock()
	parseCallbacks := make([]func(), 0)
	for _, parseSubscribers := range parseHub.subs {
		for _, parseCallback := range parseSubscribers {
			parseCallbacks = append(parseCallbacks, parseCallback)
		}
	}
	parseHub.mu.Unlock()
	for _, parseCallback := range parseCallbacks {
		parseCallback()
	}
}
