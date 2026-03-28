package cachecore

import "testing"

// TestResolveOwnershipDecision verifies main-thread vs worker ownership boundaries.
func TestResolveOwnershipDecision(parseT *testing.T) {
	parseMainReadDecision := ResolveOwnershipDecision(OwnershipTaskHotRead)
	if parseMainReadDecision.Actor != OwnershipActorMainThread {
		parseT.Fatalf("expected hot read ownership on main thread, got %+v", parseMainReadDecision)
	}
	parseOptimisticDecision := ResolveOwnershipDecision(OwnershipTaskOptimisticState)
	if parseOptimisticDecision.Actor != OwnershipActorMainThread {
		parseT.Fatalf("expected optimistic-state ownership on main thread, got %+v", parseOptimisticDecision)
	}
	parseWorkerDecision := ResolveOwnershipDecision(OwnershipTaskRetryScheduling)
	if parseWorkerDecision.Actor != OwnershipActorWorker {
		parseT.Fatalf("expected retry scheduling ownership on worker, got %+v", parseWorkerDecision)
	}
}
