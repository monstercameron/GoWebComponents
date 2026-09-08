package kvstate

import "testing"

// TestInvalidateNotifiesAllKeysWithoutRebroadcast verifies coalesced native commits
// refresh every binding, and released subscriptions receive no notifications.
func TestInvalidateNotifiesAllKeysWithoutRebroadcast(parseT *testing.T) {
	parseName := "desktop-invalidation-test"
	parseFirst, parseSecond := 0, 0
	parseStopFirst := subscribeCrossTab(parseName, "one", func() { parseFirst++ })
	parseStopSecond := subscribeCrossTab(parseName, "two", func() { parseSecond++ })
	defer parseStopFirst()
	defer parseStopSecond()
	Invalidate("other-database")
	if parseFirst != 0 || parseSecond != 0 {
		parseT.Fatal("cross-database invalidation")
	}
	Invalidate(parseName)
	if parseFirst != 1 || parseSecond != 1 {
		parseT.Fatal("invalidation did not refresh both keys")
	}
	parseStopFirst()
	Invalidate(parseName)
	if parseFirst != 1 || parseSecond != 2 {
		parseT.Fatal("released subscription called")
	}
}

// TestExternalInvalidationDoesNotOpenBrowserTransport verifies desktop bindings
// cannot accidentally use BroadcastChannel instead of native commit delivery.
func TestExternalInvalidationDoesNotOpenBrowserTransport(parseT *testing.T) {
	parseName := "desktop-external-only-test"
	parseCalls := 0
	parseStop := subscribeBinding(Options{Name: parseName, ExternalInvalidation: true}, "one", func() { parseCalls++ })
	defer parseStop()
	hubsMu.Lock()
	parseBrowserHub := hubs[parseName]
	parseExternalHub := externalHubs[parseName]
	hubsMu.Unlock()
	if parseBrowserHub != nil || parseExternalHub == nil || parseExternalHub.available {
		parseT.Fatal("external binding opened browser transport")
	}
	Invalidate(parseName)
	if parseCalls != 1 {
		parseT.Fatal("external commit did not notify binding")
	}
	parseStop()
	Invalidate(parseName)
	if parseCalls != 1 {
		parseT.Fatal("external watcher survived cleanup")
	}
	hubsMu.Lock()
	parseRetained := externalHubs[parseName]
	hubsMu.Unlock()
	if parseRetained != nil {
		parseT.Fatal("empty external hub retained")
	}
}
