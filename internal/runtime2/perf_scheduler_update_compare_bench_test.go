package runtime2

import (
	"strconv"
	"testing"
)

// buildSchedulerUpdateLegacyState stores the previous scheduler update queue state used by compare benchmarks.
type buildSchedulerUpdateLegacyState struct {
	getScheduler                   *Scheduler
	storeSchedulerQueue            []SchedulerJob
	storeSchedulerUpdateQueueByKey map[schedulerQueueCoalesceKey]int
}

// buildSchedulerUpdateBench builds one scheduler with mounted regions for scheduler update compare benchmarks.
func buildSchedulerUpdateBench(parseB *testing.B, parseRegionCount int) (*Scheduler, []string) {
	parseB.Helper()
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2", "shard-3", "shard-4"})
	getRegionIDs := make([]string, 0, parseRegionCount)
	for parseIndex := 0; parseIndex < parseRegionCount; parseIndex++ {
		getRegionID := buildSchedulerUpdateBenchRegionID(parseIndex)
		if _, parseMountErr := getScheduler.HandleSchedulerMount(getRegionID); parseMountErr != nil {
			parseB.Fatalf("HandleSchedulerMount(%s) returned error: %v", getRegionID, parseMountErr)
		}
		getRegionIDs = append(getRegionIDs, getRegionID)
	}
	return getScheduler, getRegionIDs
}

// buildSchedulerUpdateLegacyBench builds one legacy scheduler update state backed by the previous coalesce-index layout.
func buildSchedulerUpdateLegacyBench(parseB *testing.B, parseRegionCount int) (*buildSchedulerUpdateLegacyState, []string) {
	parseB.Helper()
	getScheduler, getRegionIDs := buildSchedulerUpdateBench(parseB, parseRegionCount)
	return &buildSchedulerUpdateLegacyState{
		getScheduler:                   getScheduler,
		storeSchedulerQueue:            append([]SchedulerJob(nil), getScheduler.storeSchedulerQueue...),
		storeSchedulerUpdateQueueByKey: make(map[schedulerQueueCoalesceKey]int),
	}, getRegionIDs
}

// buildSchedulerUpdateBenchRegionID builds one stable benchmark region ID.
func buildSchedulerUpdateBenchRegionID(parseIndex int) string {
	return "region-bench-" + strconv.Itoa(parseIndex)
}

// handleSchedulerQueueAppendLegacy preserves the previous append helper behavior with update-key revalidation and replacement.
func handleSchedulerQueueAppendLegacy(parseState *buildSchedulerUpdateLegacyState, parseSchedulerJob SchedulerJob) error {
	if parseState == nil || parseState.getScheduler == nil {
		return nil
	}
	if parseSchedulerJob.GetSchedulerJobKind == SchedulerJobKindUpdate {
		getQueueCoalesceKey := buildSchedulerQueueCoalesceKey(
			parseSchedulerJob.GetSchedulerRegionID,
			parseSchedulerJob.GetSchedulerCancelVersion,
		)
		getQueueIndex, hasQueueIndex := parseState.storeSchedulerUpdateQueueByKey[getQueueCoalesceKey]
		if hasQueueIndex {
			if getQueueIndex >= 0 && getQueueIndex < len(parseState.storeSchedulerQueue) {
				getQueuedJob := parseState.storeSchedulerQueue[getQueueIndex]
				if getQueuedJob.GetSchedulerJobKind == SchedulerJobKindUpdate &&
					getQueuedJob.GetSchedulerRegionID == parseSchedulerJob.GetSchedulerRegionID &&
					getQueuedJob.GetSchedulerCancelVersion == parseSchedulerJob.GetSchedulerCancelVersion {
					parseState.storeSchedulerQueue[getQueueIndex] = parseSchedulerJob
					return nil
				}
			}
			delete(parseState.storeSchedulerUpdateQueueByKey, getQueueCoalesceKey)
		}
	}
	if parseState.getScheduler.storeSchedulerQueueLimit > 0 && len(parseState.storeSchedulerQueue) >= parseState.getScheduler.storeSchedulerQueueLimit {
		return nil
	}
	parseState.storeSchedulerQueue = append(parseState.storeSchedulerQueue, parseSchedulerJob)
	if parseSchedulerJob.GetSchedulerJobKind == SchedulerJobKindUpdate {
		getQueueCoalesceKey := buildSchedulerQueueCoalesceKey(
			parseSchedulerJob.GetSchedulerRegionID,
			parseSchedulerJob.GetSchedulerCancelVersion,
		)
		parseState.storeSchedulerUpdateQueueByKey[getQueueCoalesceKey] = len(parseState.storeSchedulerQueue) - 1
	}
	return nil
}

// handleSchedulerUpdateLegacy preserves the previous update path that rechecked the queued update key inside the append helper.
func handleSchedulerUpdateLegacy(parseState *buildSchedulerUpdateLegacyState, parseRegionID string) (SchedulerJob, error) {
	if parseState == nil || parseState.getScheduler == nil {
		return SchedulerJob{}, nil
	}
	parseScheduler := parseState.getScheduler
	getSchedulerShardID, hasSchedulerRegionShardID := parseScheduler.getSchedulerShardModel.GetSchedulerRegionAssignedShardID(parseRegionID)
	if !hasSchedulerRegionShardID {
		return SchedulerJob{}, nil
	}
	getSchedulerCancelVersion := parseScheduler.getSchedulerCancelVersion(parseRegionID)
	if hasSchedulerEquivalentQueuedUpdateLegacy(parseState, parseRegionID, getSchedulerShardID, getSchedulerCancelVersion) {
		return SchedulerJob{
			GetSchedulerJobKind:       SchedulerJobKindUpdate,
			GetSchedulerRegionID:      parseRegionID,
			GetSchedulerShardID:       getSchedulerShardID,
			GetSchedulerCancelVersion: getSchedulerCancelVersion,
		}, nil
	}
	switch parseScheduler.getSchedulerWorkerHealth(getSchedulerShardID) {
	case SchedulerWorkerHealthReady:
	case SchedulerWorkerHealthDegraded:
		return SchedulerJob{}, nil
	case SchedulerWorkerHealthDead, SchedulerWorkerHealthRestarting:
		getSchedulerShardRepair, getSchedulerShardRepairErr := parseScheduler.getSchedulerRepairShardID(parseRegionID)
		if getSchedulerShardRepairErr != nil {
			return SchedulerJob{}, getSchedulerShardRepairErr
		}
		getSchedulerShardID = getSchedulerShardRepair
	default:
		return SchedulerJob{}, nil
	}
	buildSchedulerJob := SchedulerJob{
		GetSchedulerJobKind:       SchedulerJobKindUpdate,
		GetSchedulerRegionID:      parseRegionID,
		GetSchedulerShardID:       getSchedulerShardID,
		GetSchedulerCancelVersion: getSchedulerCancelVersion,
	}
	if parseQueueErr := handleSchedulerQueueAppendLegacy(parseState, buildSchedulerJob); parseQueueErr != nil {
		return SchedulerJob{}, parseQueueErr
	}
	return buildSchedulerJob, nil
}

// hasSchedulerEquivalentQueuedUpdateLegacy preserves the previous equivalent-update lookup path used before queue append.
func hasSchedulerEquivalentQueuedUpdateLegacy(
	parseState *buildSchedulerUpdateLegacyState,
	parseRegionID string,
	parseSchedulerShardID SchedulerShardID,
	parseSchedulerCancelVersion uint64,
) bool {
	if parseState == nil {
		return false
	}
	getQueueCoalesceKey := buildSchedulerQueueCoalesceKey(parseRegionID, parseSchedulerCancelVersion)
	getQueueIndex, hasQueueIndex := parseState.storeSchedulerUpdateQueueByKey[getQueueCoalesceKey]
	if !hasQueueIndex {
		return false
	}
	if getQueueIndex < 0 || getQueueIndex >= len(parseState.storeSchedulerQueue) {
		delete(parseState.storeSchedulerUpdateQueueByKey, getQueueCoalesceKey)
		return false
	}
	getQueuedJob := parseState.storeSchedulerQueue[getQueueIndex]
	if getQueuedJob.GetSchedulerJobKind != SchedulerJobKindUpdate ||
		getQueuedJob.GetSchedulerRegionID != parseRegionID ||
		getQueuedJob.GetSchedulerCancelVersion != parseSchedulerCancelVersion {
		delete(parseState.storeSchedulerUpdateQueueByKey, getQueueCoalesceKey)
		return false
	}
	return getQueuedJob.GetSchedulerShardID == parseSchedulerShardID
}

// BenchmarkHandleSchedulerUpdateCurrentVsLegacy compares the scheduler update hot path against the previous double-lookup queue path.
func BenchmarkHandleSchedulerUpdateCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("append_distinct_updates_legacy", func(parseB *testing.B) {
		const getRegionCount = 64
		getScheduler, getRegionIDs := buildSchedulerUpdateLegacyBench(parseB, getRegionCount)
		getRegionIndex := 0
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if getRegionIndex == len(getRegionIDs) {
				parseB.StopTimer()
				getScheduler, getRegionIDs = buildSchedulerUpdateLegacyBench(parseB, getRegionCount)
				getRegionIndex = 0
				parseB.StartTimer()
			}
			if _, parseUpdateErr := handleSchedulerUpdateLegacy(getScheduler, getRegionIDs[getRegionIndex]); parseUpdateErr != nil {
				parseB.Fatalf("handleSchedulerUpdateLegacy returned error: %v", parseUpdateErr)
			}
			getRegionIndex++
		}
	})

	parseB.Run("append_distinct_updates_current", func(parseB *testing.B) {
		const getRegionCount = 64
		getScheduler, getRegionIDs := buildSchedulerUpdateBench(parseB, getRegionCount)
		getRegionIndex := 0
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if getRegionIndex == len(getRegionIDs) {
				parseB.StopTimer()
				getScheduler, getRegionIDs = buildSchedulerUpdateBench(parseB, getRegionCount)
				getRegionIndex = 0
				parseB.StartTimer()
			}
			if _, parseUpdateErr := getScheduler.HandleSchedulerUpdate(getRegionIDs[getRegionIndex]); parseUpdateErr != nil {
				parseB.Fatalf("HandleSchedulerUpdate returned error: %v", parseUpdateErr)
			}
			getRegionIndex++
		}
	})

	parseB.Run("equivalent_queued_updates_legacy", func(parseB *testing.B) {
		getScheduler, _ := buildSchedulerUpdateLegacyBench(parseB, 1)
		if _, parseUpdateErr := handleSchedulerUpdateLegacy(getScheduler, "region-bench-0"); parseUpdateErr != nil {
			parseB.Fatalf("handleSchedulerUpdateLegacy warm-up returned error: %v", parseUpdateErr)
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseUpdateErr := handleSchedulerUpdateLegacy(getScheduler, "region-bench-0"); parseUpdateErr != nil {
				parseB.Fatalf("handleSchedulerUpdateLegacy returned error: %v", parseUpdateErr)
			}
		}
	})

	parseB.Run("equivalent_queued_updates_current", func(parseB *testing.B) {
		getScheduler, _ := buildSchedulerUpdateBench(parseB, 1)
		if _, parseUpdateErr := getScheduler.HandleSchedulerUpdate("region-bench-0"); parseUpdateErr != nil {
			parseB.Fatalf("HandleSchedulerUpdate warm-up returned error: %v", parseUpdateErr)
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseUpdateErr := getScheduler.HandleSchedulerUpdate("region-bench-0"); parseUpdateErr != nil {
				parseB.Fatalf("HandleSchedulerUpdate returned error: %v", parseUpdateErr)
			}
		}
	})
}
