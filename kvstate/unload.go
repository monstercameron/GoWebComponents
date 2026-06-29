package kvstate

import "github.com/monstercameron/GoWebComponents/v4/interop"

// registerUnloadFlush invokes parseFlush when the page is about to be hidden, so
// the OnUnload strategy can persist before navigation/close. On native the
// window event target is unavailable and this is a no-op.
func registerUnloadFlush(parseFlush func()) {
	parseEvents, parseErr := interop.GetWindowEvents()
	if parseErr != nil {
		return
	}
	// "pagehide" is the reliable lifecycle signal for bfcache + close + navigation.
	parseEvents.Listen("pagehide", func(parseEvent interop.BrowserEvent) {
		parseFlush()
	})
}
