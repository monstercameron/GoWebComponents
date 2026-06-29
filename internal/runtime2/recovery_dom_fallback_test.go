package runtime2

import "testing"

// TestHandleDOMCommitFailureMissingParentAnchorEntersFallback verifies missing parent anchors trigger fallback.
func TestHandleDOMCommitFailureMissingParentAnchorEntersFallback(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseFailureErr := parseRecoveryCoordinator.HandleDOMCommitFailure("region-a", DOMCommitFailureKindMissingParentAnchor, 3, 8)
	if parseFailureErr != nil {
		parseTesting.Fatalf("HandleDOMCommitFailure(missing parent anchor) error = %v", parseFailureErr)
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after missing parent anchor")
	}
}

// TestHandleDOMCommitFailureMissingNodeLookupEntersFallback verifies missing required node lookups trigger fallback.
func TestHandleDOMCommitFailureMissingNodeLookupEntersFallback(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseFailureErr := parseRecoveryCoordinator.HandleDOMCommitFailure("region-a", DOMCommitFailureKindMissingNodeLookup, 3, 8)
	if parseFailureErr != nil {
		parseTesting.Fatalf("HandleDOMCommitFailure(missing node lookup) error = %v", parseFailureErr)
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after missing node lookup")
	}
}

// TestHandleDOMCommitFailureInvalidKeyedMoveEntersFallback verifies invalid keyed moves trigger fallback.
func TestHandleDOMCommitFailureInvalidKeyedMoveEntersFallback(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseFailureErr := parseRecoveryCoordinator.HandleDOMCommitFailure("region-a", DOMCommitFailureKindInvalidKeyedMove, 3, 8)
	if parseFailureErr != nil {
		parseTesting.Fatalf("HandleDOMCommitFailure(invalid keyed move) error = %v", parseFailureErr)
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after invalid keyed move")
	}
}
