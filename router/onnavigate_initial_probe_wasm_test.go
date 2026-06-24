//go:build js && wasm

package router

import "testing"

// TestOnNavigateInitialMountFires documents whether OnNavigate fires on the very
// first render (initial mount), before any explicit Navigate.
func TestOnNavigateInitialMountFires(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	navigationCallbacks = map[int]func(Location){}
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)

	var parsePaths []string
	OnNavigate(func(parseLoc Location) { parsePaths = append(parsePaths, parseLoc.Path) })

	// Initial render with no prior Navigate.
	globalRouter.renderCurrentRoute(false)
	parseT.Logf("after initial render, fires = %v", parsePaths)
	parseInitialCount := len(parsePaths)

	// A subsequent real navigation must fire exactly once more.
	Navigate("/beta")
	globalRouter.renderCurrentRoute(false)
	parseT.Logf("after Navigate(/beta), fires = %v", parsePaths)
	if len(parsePaths) != parseInitialCount+1 {
		parseT.Fatalf("expected exactly one more fire after navigation, got %v", parsePaths)
	}
	if parsePaths[len(parsePaths)-1] != "/beta" {
		parseT.Fatalf("last fire path = %q, want /beta", parsePaths[len(parsePaths)-1])
	}
}
