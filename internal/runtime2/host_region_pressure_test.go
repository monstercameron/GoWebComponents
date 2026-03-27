package runtime2_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHostRegionMultipleRegionsSharingOneRendererIDCoverage verifies multiple regions can mount and update while sharing one renderer ID.
func TestHostRegionMultipleRegionsSharingOneRendererIDCoverage(parseT *testing.T) {
	getRendererID := runtime2.RendererID("dashboard.shared-renderer")
	buildRegionA, parseBuildAErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-a"),
		[]runtime2.SchedulerShardID{"shard-a", "shard-b"},
	)
	if parseBuildAErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter(region-a) returned error: %v", parseBuildAErr)
	}
	buildRegionB, parseBuildBErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-b"),
		[]runtime2.SchedulerShardID{"shard-a", "shard-b"},
	)
	if parseBuildBErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter(region-b) returned error: %v", parseBuildBErr)
	}
	if _, parseMountAErr := buildRegionA.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       getRendererID,
		RegionInstanceID: runtime2.RegionInstanceID("region-a"),
	}, 1); parseMountAErr != nil {
		parseT.Fatalf("HandleHostRegionMount(region-a) returned error: %v", parseMountAErr)
	}
	if _, parseMountBErr := buildRegionB.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       getRendererID,
		RegionInstanceID: runtime2.RegionInstanceID("region-b"),
	}, 1); parseMountBErr != nil {
		parseT.Fatalf("HandleHostRegionMount(region-b) returned error: %v", parseMountBErr)
	}
	if _, parseUpdateAErr := buildRegionA.HandleHostRegionUpdate(2); parseUpdateAErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate(region-a) returned error: %v", parseUpdateAErr)
	}
	if _, parseUpdateBErr := buildRegionB.HandleHostRegionUpdate(2); parseUpdateBErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate(region-b) returned error: %v", parseUpdateBErr)
	}
	getEntryA, hasEntryA := buildRegionA.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-a"))
	if !hasEntryA {
		parseT.Fatal("expected coordinator entry for region-a")
	}
	getEntryB, hasEntryB := buildRegionB.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-b"))
	if !hasEntryB {
		parseT.Fatal("expected coordinator entry for region-b")
	}
	if getEntryA.RendererID != getRendererID || getEntryB.RendererID != getRendererID {
		parseT.Fatalf("expected both regions to retain shared renderer %q, got %q and %q", getRendererID, getEntryA.RendererID, getEntryB.RendererID)
	}
}

// TestHostRegionOneWorkerManyRegionsPressureCoverage verifies one scheduler worker can sustain many mounted regions.
func TestHostRegionOneWorkerManyRegionsPressureCoverage(parseT *testing.T) {
	buildScheduler := runtime2.BuildScheduler([]runtime2.SchedulerShardID{"shard-a"})
	getRegionCount := 64
	for getRegionIndex := 0; getRegionIndex < getRegionCount; getRegionIndex++ {
		getRegionID := fmt.Sprintf("region-%d", getRegionIndex)
		getMountJob, parseMountErr := buildScheduler.HandleSchedulerMount(getRegionID)
		if parseMountErr != nil {
			parseT.Fatalf("HandleSchedulerMount(%s) returned error: %v", getRegionID, parseMountErr)
		}
		if getMountJob.GetSchedulerShardID != runtime2.SchedulerShardID("shard-a") {
			parseT.Fatalf("expected one-worker pressure mount shard-a, got %q", getMountJob.GetSchedulerShardID)
		}
	}
	for getRegionIndex := 0; getRegionIndex < getRegionCount; getRegionIndex++ {
		getRegionID := fmt.Sprintf("region-%d", getRegionIndex)
		getUpdateJob, parseUpdateErr := buildScheduler.HandleSchedulerUpdate(getRegionID)
		if parseUpdateErr != nil {
			parseT.Fatalf("HandleSchedulerUpdate(%s) returned error: %v", getRegionID, parseUpdateErr)
		}
		if getUpdateJob.GetSchedulerShardID != runtime2.SchedulerShardID("shard-a") {
			parseT.Fatalf("expected one-worker pressure update shard-a, got %q", getUpdateJob.GetSchedulerShardID)
		}
	}
	if buildScheduler.GetSchedulerQueueDepth() != getRegionCount*2 {
		parseT.Fatalf("expected queue depth %d after mount+update pressure, got %d", getRegionCount*2, buildScheduler.GetSchedulerQueueDepth())
	}
}

// TestHostRegionManyWorkersFewRegionsSkewCoverage verifies few hot regions over many workers keep stable assignments under skew.
func TestHostRegionManyWorkersFewRegionsSkewCoverage(parseT *testing.T) {
	getShardIDs := []runtime2.SchedulerShardID{"shard-a", "shard-b", "shard-c", "shard-d", "shard-e"}
	buildScheduler := runtime2.BuildScheduler(getShardIDs)
	getMountShardsByRegion := map[string]runtime2.SchedulerShardID{}
	getRegionIDs := []string{"region-a", "region-b"}
	for _, getRegionID := range getRegionIDs {
		getMountJob, parseMountErr := buildScheduler.HandleSchedulerMount(getRegionID)
		if parseMountErr != nil {
			parseT.Fatalf("HandleSchedulerMount(%s) returned error: %v", getRegionID, parseMountErr)
		}
		getMountShardsByRegion[getRegionID] = getMountJob.GetSchedulerShardID
	}
	getUniqueShards := map[runtime2.SchedulerShardID]struct{}{}
	for _, getMountShardID := range getMountShardsByRegion {
		getUniqueShards[getMountShardID] = struct{}{}
	}
	if len(getUniqueShards) >= len(getShardIDs) {
		parseT.Fatalf("expected skew with fewer active shards than workers, workers=%d active=%d", len(getShardIDs), len(getUniqueShards))
	}
	for _, getRegionID := range getRegionIDs {
		getUpdateJob, parseUpdateErr := buildScheduler.HandleSchedulerUpdate(getRegionID)
		if parseUpdateErr != nil {
			parseT.Fatalf("HandleSchedulerUpdate(%s) returned error: %v", getRegionID, parseUpdateErr)
		}
		if getUpdateJob.GetSchedulerShardID != getMountShardsByRegion[getRegionID] {
			parseT.Fatalf(
				"expected stable shard assignment for %s mount=%q update=%q",
				getRegionID,
				getMountShardsByRegion[getRegionID],
				getUpdateJob.GetSchedulerShardID,
			)
		}
	}
}
