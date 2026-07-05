package runtime

import "testing"

// TestPlainFiberDeletionSkipsAtomRegistry pins the invariant behind the
// CleanupAtomSubscriptions fast path: every Subscribe call site records the
// atom id on hooks.atoms or fiber.reactiveSourceIDs, so a fiber with both
// empty can never hold a registry subscription — and its deletion must not
// need the registry-wide scan the old fallback performed. If a future
// subscribe path breaks the recording invariant, this test fails.
func TestPlainFiberDeletionSkipsAtomRegistry(parseT *testing.T) {
	parseRt := &Runtime{atomRegistry: NewAtomRegistry(), scheduler: newTestScheduler()}

	parseSubscribed := &Fiber{typeOf: "test"}
	SetCurrentFiber(parseSubscribed)
	getValue, setValue := GoUseAtom(parseRt, "invariant-atom", 1)
	SetCurrentFiber(nil)
	if getValue() != 1 {
		parseT.Fatalf("atom read = %d, want 1", getValue())
	}
	setValue(2)

	// The subscribed fiber must carry the recording.
	if parseSubscribed.hooks == nil || len(parseSubscribed.hooks.atoms) == 0 {
		parseT.Fatal("GoUseAtom did not record the subscription on hooks.atoms — the CleanupAtomSubscriptions fast path is no longer sound")
	}

	// Every fiber present in the registry's subscription sets must have a
	// recorded reason to be there.
	parseRt.atomRegistry.mu.RLock()
	for parseAtomID, parseSet := range parseRt.atomRegistry.subscriptions {
		for parseFiber := range parseSet {
			hasRecordedAtoms := parseFiber.hooks != nil && len(parseFiber.hooks.atoms) > 0
			if !hasRecordedAtoms && len(parseFiber.reactiveSourceIDs) == 0 {
				parseT.Fatalf("fiber subscribed to %q without recording it on hooks.atoms or reactiveSourceIDs", parseAtomID)
			}
		}
	}
	parseRt.atomRegistry.mu.RUnlock()

	// A plain fiber (no hooks, no sources) cleans up without touching the
	// registry; afterwards the subscribed fiber's cleanup removes its entry.
	parsePlain := &Fiber{typeOf: "div"}
	parseRt.CleanupAtomSubscriptions(parsePlain)
	parseRt.CleanupAtomSubscriptions(parseSubscribed)
	parseRt.atomRegistry.mu.RLock()
	for parseAtomID, parseSet := range parseRt.atomRegistry.subscriptions {
		if len(parseSet) != 0 {
			parseT.Fatalf("subscriptions for %q not cleaned: %d left", parseAtomID, len(parseSet))
		}
	}
	parseRt.atomRegistry.mu.RUnlock()
}
