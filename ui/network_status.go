package ui

// readOnlineStatus is the platform seam returning the current navigator.onLine
// value. It is a package var so tests can substitute it.
var readOnlineStatus = defaultReadOnlineStatus

// UseNetworkStatus reports whether the browser is currently online and
// re-renders the component when connectivity changes (G32). It seeds from
// navigator.onLine and stays live via the window online/offline events, with the
// listeners released on unmount. On native/SSR builds it reports true (assume
// connectivity) and never changes.
//
//	online := ui.UseNetworkStatus()
func UseNetworkStatus() bool {
	parseOnline := UseState(readOnlineStatus())
	UseWindowEvent("online", func(Event) { parseOnline.Set(true) }, networkStatusSentinel)
	UseWindowEvent("offline", func(Event) { parseOnline.Set(false) }, networkStatusSentinel)
	return parseOnline.Get()
}

const networkStatusSentinel = "gwc-network-status"
