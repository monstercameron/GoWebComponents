package runtime2

import "testing"

// TestGetEntryIsolatesSourceIDsSlice pins that GetEntry returns an entry whose
// SourceIDs slice is a COPY, not the coordinator's live backing array. A shared
// slice would let a caller reading/mutating the returned SourceIDs race a
// concurrent SetRegionSourceIDs (the coordinator serves the multithreaded runtime).
func TestGetEntryIsolatesSourceIDsSlice(parseT *testing.T) {
	parseCoordinator := BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(CoordinatorEntry{
		RegionInstanceID: "region-1",
		RendererID:       "panel",
		SourceIDs:        []string{"a", "b", "c"},
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion: %v", parseErr)
	}

	parseEntry, parseOk := parseCoordinator.GetEntry("region-1")
	if !parseOk || len(parseEntry.SourceIDs) != 3 {
		parseT.Fatalf("unexpected entry: ok=%t sourceIDs=%+v", parseOk, parseEntry.SourceIDs)
	}

	// Mutating the returned slice must NOT affect the coordinator's stored entry.
	parseEntry.SourceIDs[0] = "TAMPERED"
	parseEntry.SourceIDs = append(parseEntry.SourceIDs, "extra")

	parseFresh, _ := parseCoordinator.GetEntry("region-1")
	if len(parseFresh.SourceIDs) != 3 || parseFresh.SourceIDs[0] != "a" {
		parseT.Fatalf("caller mutation leaked into coordinator state: %+v", parseFresh.SourceIDs)
	}

	// Conversely, a concurrent-style write after GetEntry must not mutate the
	// already-returned slice.
	if parseErr := parseCoordinator.SetRegionSourceIDs("region-1", []string{"x", "y"}); parseErr != nil {
		parseT.Fatalf("SetRegionSourceIDs: %v", parseErr)
	}
	if parseFresh.SourceIDs[0] != "a" {
		parseT.Fatalf("later write mutated a previously-returned entry slice: %+v", parseFresh.SourceIDs)
	}
}
