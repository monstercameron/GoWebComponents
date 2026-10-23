package runtime

import "testing"

// TestGetHookRuntimeUsesCurrentFiberOwner verifies an owned fiber bypasses the global runtime fallback.
func TestGetHookRuntimeUsesCurrentFiberOwner(t *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()
	parseWasMultiRootActive := multiRootActive.Load()
	multiRootActive.Store(false)
	defer multiRootActive.Store(parseWasMultiRootActive)
	defer SetCurrentFiber(nil)

	parseGlobal := GetGlobalRuntime()
	parseOwner := &Runtime{}
	SetCurrentFiber(&Fiber{ownerRuntime: parseOwner})
	if getHookRuntime() != parseOwner {
		t.Fatal("hook runtime did not use the current fiber's owner")
	}
	if getHookRuntime() == parseGlobal {
		t.Fatal("owned fiber fell through to the global runtime")
	}
}

// TestGetHookRuntimeFindsOwnedAncestorForIsolatedFiber preserves isolated-root ancestor resolution.
func TestGetHookRuntimeFindsOwnedAncestorForIsolatedFiber(t *testing.T) {
	parseWasMultiRootActive := multiRootActive.Load()
	multiRootActive.Store(true)
	defer multiRootActive.Store(parseWasMultiRootActive)
	defer SetCurrentFiber(nil)

	parseOwner := &Runtime{}
	parseRoot := &Fiber{ownerRuntime: parseOwner}
	parseChild := &Fiber{parent: parseRoot}
	SetCurrentFiber(parseChild)
	if getHookRuntime() != parseOwner {
		t.Fatal("hook runtime did not walk to the isolated root owner")
	}
}

// TestGetHookRuntimeFallsBackToGlobalOutsideOwnedRender pins the no-fiber fallback.
func TestGetHookRuntimeFallsBackToGlobalOutsideOwnedRender(t *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()
	parseWasMultiRootActive := multiRootActive.Load()
	multiRootActive.Store(false)
	defer multiRootActive.Store(parseWasMultiRootActive)
	SetCurrentFiber(nil)

	parseGlobal := GetGlobalRuntime()
	if getHookRuntime() != parseGlobal {
		t.Fatal("hook runtime outside an owned render must use the global runtime")
	}
}
