package cachecore

import "testing"

// TestResolveCachePolicyDefaults verifies default policy values for each class.
func TestResolveCachePolicyDefaults(parseT *testing.T) {
	parseClasses := []PolicyClass{
		PolicyClassStatic,
		PolicyClassSession,
		PolicyClassVolatile,
		PolicyClassQueued,
	}
	for _, parseClass := range parseClasses {
		parsePolicy := ResolveCachePolicyDefaults(parseClass)
		if parsePolicy.Class != parseClass {
			parseT.Fatalf("default policy class mismatch for %q: got %q", parseClass, parsePolicy.Class)
		}
		if parsePolicy.ExpiresAfter < parsePolicy.StaleAfter {
			parseT.Fatalf("invalid default policy duration bounds for %q: stale=%s expires=%s", parseClass, parsePolicy.StaleAfter, parsePolicy.ExpiresAfter)
		}
	}
}

// TestResolvePolicyReadDecisionDisplayOnlyWhenStale verifies stale-display semantics for expired records.
func TestResolvePolicyReadDecisionDisplayOnlyWhenStale(parseT *testing.T) {
	parsePolicy := BuildCachePolicy(PolicyClassStatic, 0, 0, true, true, false)
	parseDecision := ResolvePolicyReadDecision(parsePolicy, FreshnessExpired)
	if !parseDecision.CanReturnCached || !parseDecision.IsStale {
		parseT.Fatalf("expected stale display allowed for expired records, decision=%+v", parseDecision)
	}
	if !parseDecision.ShouldRefreshInBackground {
		parseT.Fatalf("expected background refresh when stale display is allowed, decision=%+v", parseDecision)
	}
}

// TestResolvePolicyReadDecisionMustRefetchBeforeMutation verifies mutation gating semantics.
func TestResolvePolicyReadDecisionMustRefetchBeforeMutation(parseT *testing.T) {
	parsePolicy := BuildCachePolicy(PolicyClassVolatile, 0, 0, true, false, true)
	parseDecision := ResolvePolicyReadDecision(parsePolicy, FreshnessFresh)
	if !parseDecision.CanReturnCached {
		parseT.Fatalf("expected cached read allowed while fresh, decision=%+v", parseDecision)
	}
	if !parseDecision.ShouldRefetchBeforeMutation {
		parseT.Fatalf("expected mutation refetch gate enabled, decision=%+v", parseDecision)
	}
}
