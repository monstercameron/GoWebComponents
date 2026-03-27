package runtime2

import "testing"

// TestHandleSchedulerKeepaliveTimeoutTransitionsHealthyToDegradedToDead verifies missed pong delivery transitions shard health from ready to degraded and then dead.
func TestHandleSchedulerKeepaliveTimeoutTransitionsHealthyToDegradedToDead(parseT *testing.T) {
	parseScheduler := BuildScheduler([]SchedulerShardID{"shard-a"})
	if parseScheduler.getSchedulerWorkerHealth("shard-a") != SchedulerWorkerHealthReady {
		parseT.Fatal("expected shard health to start in ready state before keepalive misses")
	}
	parseHealthAfterFirstMiss, parseFirstMissErr := parseScheduler.HandleSchedulerKeepaliveTimeout("shard-a")
	if parseFirstMissErr != nil {
		parseT.Fatalf("HandleSchedulerKeepaliveTimeout(first miss) returned error: %v", parseFirstMissErr)
	}
	if parseHealthAfterFirstMiss != SchedulerWorkerHealthDegraded {
		parseT.Fatalf("expected first missed pong to degrade health, got %d", parseHealthAfterFirstMiss)
	}
	parseHealthAfterSecondMiss, parseSecondMissErr := parseScheduler.HandleSchedulerKeepaliveTimeout("shard-a")
	if parseSecondMissErr != nil {
		parseT.Fatalf("HandleSchedulerKeepaliveTimeout(second miss) returned error: %v", parseSecondMissErr)
	}
	if parseHealthAfterSecondMiss != SchedulerWorkerHealthDead {
		parseT.Fatalf("expected second missed pong to mark shard dead, got %d", parseHealthAfterSecondMiss)
	}
}

// TestHandleSchedulerKeepaliveTimeoutTransitionsToDegradedThenDead verifies missed pong windows transition shard health from ready to degraded to dead.
func TestHandleSchedulerKeepaliveTimeoutTransitionsToDegradedThenDead(parseT *testing.T) {
	parseScheduler := BuildScheduler([]SchedulerShardID{"shard-a"})
	parseHealthAfterFirstMiss, parseFirstMissErr := parseScheduler.HandleSchedulerKeepaliveTimeout("shard-a")
	if parseFirstMissErr != nil {
		parseT.Fatalf("HandleSchedulerKeepaliveTimeout(first miss) returned error: %v", parseFirstMissErr)
	}
	if parseHealthAfterFirstMiss != SchedulerWorkerHealthDegraded {
		parseT.Fatalf("expected first miss to degrade health, got %d", parseHealthAfterFirstMiss)
	}
	parseHealthAfterSecondMiss, parseSecondMissErr := parseScheduler.HandleSchedulerKeepaliveTimeout("shard-a")
	if parseSecondMissErr != nil {
		parseT.Fatalf("HandleSchedulerKeepaliveTimeout(second miss) returned error: %v", parseSecondMissErr)
	}
	if parseHealthAfterSecondMiss != SchedulerWorkerHealthDead {
		parseT.Fatalf("expected second miss to mark shard dead, got %d", parseHealthAfterSecondMiss)
	}
}

// TestHandleSchedulerKeepalivePongRestoresReady verifies pong delivery restores ready health and resets missed-pong counters.
func TestHandleSchedulerKeepalivePongRestoresReady(parseT *testing.T) {
	parseScheduler := BuildScheduler([]SchedulerShardID{"shard-a"})
	if _, parseTimeoutErr := parseScheduler.HandleSchedulerKeepaliveTimeout("shard-a"); parseTimeoutErr != nil {
		parseT.Fatalf("HandleSchedulerKeepaliveTimeout returned error: %v", parseTimeoutErr)
	}
	if parsePongErr := parseScheduler.HandleSchedulerKeepalivePong("shard-a", 1); parsePongErr != nil {
		parseT.Fatalf("HandleSchedulerKeepalivePong returned error: %v", parsePongErr)
	}
	if parseScheduler.getSchedulerWorkerHealth("shard-a") != SchedulerWorkerHealthReady {
		parseT.Fatal("expected pong to restore ready worker health")
	}
	parseHealthAfterTimeout, parseTimeoutErr := parseScheduler.HandleSchedulerKeepaliveTimeout("shard-a")
	if parseTimeoutErr != nil {
		parseT.Fatalf("HandleSchedulerKeepaliveTimeout(post-pong miss) returned error: %v", parseTimeoutErr)
	}
	if parseHealthAfterTimeout != SchedulerWorkerHealthDegraded {
		parseT.Fatalf("expected first miss after pong reset to degrade health, got %d", parseHealthAfterTimeout)
	}
}

// TestHandleSchedulerKeepalivePongRejectsStaleSequence verifies stale pong sequence values are rejected.
func TestHandleSchedulerKeepalivePongRejectsStaleSequence(parseT *testing.T) {
	parseScheduler := BuildScheduler([]SchedulerShardID{"shard-a"})
	if parsePongErr := parseScheduler.HandleSchedulerKeepalivePong("shard-a", 2); parsePongErr != nil {
		parseT.Fatalf("HandleSchedulerKeepalivePong(sequence=2) returned error: %v", parsePongErr)
	}
	if parsePongErr := parseScheduler.HandleSchedulerKeepalivePong("shard-a", 1); parsePongErr == nil {
		parseT.Fatal("expected stale pong sequence to fail")
	}
}

// TestBuildSchedulerCanonicalizesShardIDsForLookup verifies scheduler shard membership checks work for unsorted duplicate shard lists.
func TestBuildSchedulerCanonicalizesShardIDsForLookup(parseT *testing.T) {
	parseScheduler := BuildScheduler([]SchedulerShardID{"shard-b", "shard-a", "shard-b"})
	if parsePongErr := parseScheduler.HandleSchedulerKeepalivePong("shard-a", 1); parsePongErr != nil {
		parseT.Fatalf("HandleSchedulerKeepalivePong(shard-a) returned error: %v", parsePongErr)
	}
	if parsePongErr := parseScheduler.HandleSchedulerKeepalivePong("shard-b", 1); parsePongErr != nil {
		parseT.Fatalf("HandleSchedulerKeepalivePong(shard-b) returned error: %v", parsePongErr)
	}
}
