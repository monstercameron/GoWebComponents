package runtime2

import "testing"

// TestHandlePatchIdempotencyDuplicatePatchStreamSameIdentityIsIgnored verifies duplicate patch identities are ignored.
func TestHandlePatchIdempotencyDuplicatePatchStreamSameIdentityIsIgnored(parseTesting *testing.T) {
	buildTracker := BuildPatchIdempotencyTracker()
	parseApplyFirst, parseFirstErr := buildTracker.HandlePatchIdempotency("region-1", 1, 3, "patch-a")
	if parseFirstErr != nil {
		parseTesting.Fatalf("HandlePatchIdempotency(first) error = %v", parseFirstErr)
	}
	if !parseApplyFirst {
		parseTesting.Fatal("HandlePatchIdempotency(first) apply = false, want true")
	}
	parseApplySecond, parseSecondErr := buildTracker.HandlePatchIdempotency("region-1", 1, 3, "patch-a")
	if parseSecondErr != nil {
		parseTesting.Fatalf("HandlePatchIdempotency(duplicate) error = %v", parseSecondErr)
	}
	if parseApplySecond {
		parseTesting.Fatal("HandlePatchIdempotency(duplicate) apply = true, want false")
	}
}

// TestHandlePatchIdempotencyDifferentPatchStreamSameVersionFails verifies conflicting identities are rejected.
func TestHandlePatchIdempotencyDifferentPatchStreamSameVersionFails(parseTesting *testing.T) {
	buildTracker := BuildPatchIdempotencyTracker()
	_, parseFirstErr := buildTracker.HandlePatchIdempotency("region-1", 1, 3, "patch-a")
	if parseFirstErr != nil {
		parseTesting.Fatalf("HandlePatchIdempotency(first) error = %v", parseFirstErr)
	}
	_, parseSecondErr := buildTracker.HandlePatchIdempotency("region-1", 1, 3, "patch-b")
	if parseSecondErr == nil {
		parseTesting.Fatal("HandlePatchIdempotency(conflicting identity) error = nil, want error")
	}
}

// TestHandlePatchIdempotencyStateResetsOnEpochChange verifies epoch changes reset idempotency tracking.
func TestHandlePatchIdempotencyStateResetsOnEpochChange(parseTesting *testing.T) {
	buildTracker := BuildPatchIdempotencyTracker()
	_, parseFirstErr := buildTracker.HandlePatchIdempotency("region-1", 1, 3, "patch-a")
	if parseFirstErr != nil {
		parseTesting.Fatalf("HandlePatchIdempotency(epoch1) error = %v", parseFirstErr)
	}
	parseApplySecond, parseSecondErr := buildTracker.HandlePatchIdempotency("region-1", 2, 3, "patch-b")
	if parseSecondErr != nil {
		parseTesting.Fatalf("HandlePatchIdempotency(epoch2) error = %v", parseSecondErr)
	}
	if !parseApplySecond {
		parseTesting.Fatal("HandlePatchIdempotency(epoch2) apply = false, want true")
	}
}
