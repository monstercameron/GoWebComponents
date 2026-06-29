package runtime2

import "testing"

// TestSchedulerGuardAndCommitPaths exercises nil guards, health setters, and fallback ownership transitions.
func TestSchedulerGuardAndCommitPaths(t *testing.T) {
	var getNilScheduler *Scheduler
	if err := getNilScheduler.HandleSchedulerKeepalivePong("shard-1", 1); err == nil {
		t.Fatal("expected nil scheduler pong to fail")
	}
	if _, err := getNilScheduler.HandleSchedulerKeepaliveTimeout("shard-1"); err == nil {
		t.Fatal("expected nil scheduler timeout to fail")
	}
	if _, err := getNilScheduler.HandleSchedulerMount("region-a"); err == nil {
		t.Fatal("expected nil scheduler mount to fail")
	}
	if _, err := getNilScheduler.HandleSchedulerUpdate("region-a"); err == nil {
		t.Fatal("expected nil scheduler update to fail")
	}
	if getNilScheduler.HandleSchedulerDispose("region-a") {
		t.Fatal("expected nil scheduler dispose to return false")
	}
	if getNilScheduler.HandleSchedulerFallback("region-a") {
		t.Fatal("expected nil scheduler fallback to return false")
	}
	if getNilScheduler.HandleSchedulerCancel("region-a") {
		t.Fatal("expected nil scheduler cancel to return false")
	}
	if !getNilScheduler.HasSchedulerJobStale(SchedulerJob{GetSchedulerRegionID: "region-a", GetSchedulerCancelVersion: 1}) {
		t.Fatal("expected nil scheduler stale check to be conservative")
	}
	if getNilScheduler.HasSchedulerCommitAllowed("region-a", 1) {
		t.Fatal("expected nil scheduler commit check to fail")
	}
	if getNilScheduler.GetSchedulerIsDegraded() {
		t.Fatal("expected nil scheduler to report non-degraded")
	}
	if getNilScheduler.GetSchedulerQueueDepth() != 0 {
		t.Fatal("expected nil scheduler queue depth to be zero")
	}
	if getNilScheduler.HasSchedulerFallbackOwnership("region-a") {
		t.Fatal("expected nil scheduler fallback ownership to be false")
	}
	if getNilScheduler.ClearSchedulerFallbackOwnership("region-a") {
		t.Fatal("expected nil scheduler clear fallback to return false")
	}
	if err := getNilScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthReady); err == nil {
		t.Fatal("expected nil scheduler health update to fail")
	}

	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if getScheduler.HandleSchedulerFallback("missing") {
		t.Fatal("expected fallback on unknown region to fail")
	}
	if getScheduler.HasSchedulerCommitAllowed("missing", 0) {
		t.Fatal("expected commit check on unknown region to fail")
	}
	if err := getScheduler.SetSchedulerWorkerHealth("missing", SchedulerWorkerHealthReady); err == nil {
		t.Fatal("expected health update for unknown shard to fail")
	}
	if err := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealth(255)); err == nil {
		t.Fatal("expected unsupported worker health to fail")
	}
	if err := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDegraded); err != nil {
		t.Fatalf("SetSchedulerWorkerHealth(shard-1, degraded) returned error: %v", err)
	}
	if !getScheduler.GetSchedulerIsDegraded() {
		t.Fatal("expected degraded scheduler to report degraded")
	}
	if err := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthReady); err != nil {
		t.Fatalf("SetSchedulerWorkerHealth(shard-1, ready) returned error: %v", err)
	}
	if getScheduler.GetSchedulerIsDegraded() {
		t.Fatal("expected ready scheduler to report non-degraded")
	}
	getMountJob, err := getScheduler.HandleSchedulerMount("region-a")
	if err != nil {
		t.Fatalf("HandleSchedulerMount(region-a) returned error: %v", err)
	}
	if getScheduler.GetSchedulerQueueDepth() != 1 {
		t.Fatalf("expected one queued job after mount, got %d", getScheduler.GetSchedulerQueueDepth())
	}
	if getScheduler.HasSchedulerJobStale(getMountJob) {
		t.Fatal("expected mounted job to be current before cancel")
	}
	if !getScheduler.HasSchedulerCommitAllowed("region-a", getMountJob.GetSchedulerCancelVersion) {
		t.Fatal("expected current commit version to be allowed")
	}
	if !getScheduler.HandleSchedulerFallback("region-a") {
		t.Fatal("expected fallback to succeed for mounted region")
	}
	if getScheduler.HasSchedulerCommitAllowed("region-a", getMountJob.GetSchedulerCancelVersion) {
		t.Fatal("expected fallback region commit to be blocked")
	}
	if !getScheduler.ClearSchedulerFallbackOwnership("region-a") {
		t.Fatal("expected fallback ownership clear to succeed")
	}
	if getScheduler.ClearSchedulerFallbackOwnership("region-a") {
		t.Fatal("expected repeated fallback clear to report false")
	}
	if !getScheduler.HasSchedulerCommitAllowed("region-a", getMountJob.GetSchedulerCancelVersion) {
		t.Fatal("expected commit to be allowed again after clearing fallback")
	}
	if !getScheduler.HandleSchedulerCancel("region-a") {
		t.Fatal("expected cancel to succeed for mounted region")
	}
	if !getScheduler.HasSchedulerJobStale(getMountJob) {
		t.Fatal("expected mounted job to become stale after cancel")
	}
	if !getScheduler.HandleSchedulerDispose("region-a") {
		t.Fatal("expected dispose to clear assigned region")
	}
}

// TestSchedulerKeepaliveAndRepairPaths exercises keepalive transitions and mount repair fallback.
func TestSchedulerKeepaliveAndRepairPaths(t *testing.T) {
	var getNilScheduler *Scheduler
	if err := getNilScheduler.HandleSchedulerKeepalivePong("shard-1", 1); err == nil {
		t.Fatal("expected nil scheduler pong to fail")
	}
	if _, err := getNilScheduler.HandleSchedulerKeepaliveTimeout("shard-1"); err == nil {
		t.Fatal("expected nil scheduler timeout to fail")
	}

	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if err := getScheduler.HandleSchedulerKeepalivePong("shard-1", 0); err == nil {
		t.Fatal("expected zero pong sequence to fail")
	}
	if err := getScheduler.HandleSchedulerKeepalivePong("missing", 1); err == nil {
		t.Fatal("expected pong for unknown shard to fail")
	}
	if err := getScheduler.HandleSchedulerKeepalivePong("shard-1", 2); err != nil {
		t.Fatalf("HandleSchedulerKeepalivePong(shard-1, 2) returned error: %v", err)
	}
	if err := getScheduler.HandleSchedulerKeepalivePong("shard-1", 1); err == nil {
		t.Fatal("expected stale pong sequence to fail")
	}
	getHealth, err := getScheduler.HandleSchedulerKeepaliveTimeout("shard-1")
	if err != nil {
		t.Fatalf("HandleSchedulerKeepaliveTimeout(shard-1) returned error: %v", err)
	}
	if getHealth != SchedulerWorkerHealthDegraded {
		t.Fatalf("expected first timeout to degrade worker, got %d", getHealth)
	}
	getHealth, err = getScheduler.HandleSchedulerKeepaliveTimeout("shard-1")
	if err != nil {
		t.Fatalf("HandleSchedulerKeepaliveTimeout(shard-1) returned error: %v", err)
	}
	if getHealth != SchedulerWorkerHealthDead {
		t.Fatalf("expected second timeout to mark worker dead, got %d", getHealth)
	}

	getDegradedScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if err := getDegradedScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDegraded); err != nil {
		t.Fatalf("SetSchedulerWorkerHealth(shard-1, degraded) returned error: %v", err)
	}
	if _, err := getDegradedScheduler.HandleSchedulerMount("region-a"); err == nil {
		t.Fatal("expected mount to reject degraded worker")
	}

	getRepairScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if err := getRepairScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthRestarting); err != nil {
		t.Fatalf("SetSchedulerWorkerHealth(shard-1, restarting) returned error: %v", err)
	}
	if _, err := getRepairScheduler.HandleSchedulerMount("region-b"); err == nil {
		t.Fatal("expected mount repair to fail when no ready shard is available")
	}
	if !getRepairScheduler.HasSchedulerFallbackOwnership("region-b") {
		t.Fatal("expected mount repair failure to mark region fallback-owned")
	}
	if !getRepairScheduler.ClearSchedulerFallbackOwnership("region-b") {
		t.Fatal("expected fallback ownership clear after repair failure")
	}
}

// TestSchedulerQueueLookupAndReplacementEdgeCases exercises queue lookup helpers and replacement guards.
func TestSchedulerQueueLookupAndReplacementEdgeCases(t *testing.T) {
	if getKey := buildSchedulerQueueCoalesceKey("region-a", 7); getKey.getSchedulerRegionID != "region-a" || getKey.getSchedulerCancelVersion != 7 {
		t.Fatalf("unexpected coalesce key: %+v", getKey)
	}

	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	getScheduler.storeSchedulerUpdateQueueIndexByRegionID["region-stale"] = schedulerUpdateQueueIndex{
		getSchedulerQueueIndex:    0,
		getSchedulerCancelVersion: 1,
	}
	if _, _, hasQueuedUpdate := getScheduler.getSchedulerQueuedUpdateByKey("region-stale", 2); hasQueuedUpdate {
		t.Fatal("expected stale queued update lookup to fail")
	}

	getScheduler.storeSchedulerQueue = []SchedulerJob{{
		GetSchedulerJobKind:       SchedulerJobKindMount,
		GetSchedulerRegionID:      "region-kind",
		GetSchedulerCancelVersion: 1,
	}}
	getScheduler.storeSchedulerUpdateQueueIndexByRegionID["region-kind"] = schedulerUpdateQueueIndex{
		getSchedulerQueueIndex:    0,
		getSchedulerCancelVersion: 1,
	}
	if _, _, hasQueuedUpdate := getScheduler.getSchedulerQueuedUpdateByKey("region-kind", 1); hasQueuedUpdate {
		t.Fatal("expected wrong-kind queued update lookup to fail")
	}

	getScheduler.storeSchedulerQueue = []SchedulerJob{{
		GetSchedulerJobKind:       SchedulerJobKindUpdate,
		GetSchedulerRegionID:      "region-index",
		GetSchedulerCancelVersion: 1,
	}}
	getScheduler.storeSchedulerUpdateQueueIndexByRegionID["region-index"] = schedulerUpdateQueueIndex{
		getSchedulerQueueIndex:    1,
		getSchedulerCancelVersion: 1,
	}
	if _, _, hasQueuedUpdate := getScheduler.getSchedulerQueuedUpdateByKey("region-index", 1); hasQueuedUpdate {
		t.Fatal("expected out-of-range queued update lookup to fail")
	}

	if err := getScheduler.HandleSchedulerReplaceWorker("missing", "shard-2"); err == nil {
		t.Fatal("expected replacement to reject unknown dead shard")
	}
	if err := getScheduler.HandleSchedulerReplaceWorker("shard-1", "shard-2"); err == nil {
		t.Fatal("expected replacement to reject non-dead shard")
	}
}

// TestSchedulerUpdateAndWorkerHealthBranches exercises the update health checks and private health readers.
func TestSchedulerUpdateAndWorkerHealthBranches(t *testing.T) {
	var getNilScheduler *Scheduler
	if got := getNilScheduler.getSchedulerWorkerHealth("shard-1"); got != schedulerWorkerHealthInvalid {
		t.Fatalf("expected nil scheduler health read to return invalid, got %d", got)
	}
	if got, has := getNilScheduler.getSchedulerWorkerHealthState("shard-1"); got != schedulerWorkerHealthInvalid || has {
		t.Fatalf("expected nil scheduler health-state read to return invalid,false, got %d,%t", got, has)
	}

	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	if got := getScheduler.getSchedulerWorkerHealth("shard-1"); got != SchedulerWorkerHealthReady {
		t.Fatalf("expected ready shard health read, got %d", got)
	}
	if got, has := getScheduler.getSchedulerWorkerHealthState("shard-1"); got != SchedulerWorkerHealthReady || !has {
		t.Fatalf("expected ready shard health-state read, got %d,%t", got, has)
	}
	if got := getScheduler.getSchedulerWorkerHealth("missing"); got != schedulerWorkerHealthInvalid {
		t.Fatalf("expected missing shard health read to return invalid, got %d", got)
	}
	if got, has := getScheduler.getSchedulerWorkerHealthState("missing"); got != schedulerWorkerHealthInvalid || has {
		t.Fatalf("expected missing shard health-state read to return invalid,false, got %d,%t", got, has)
	}

	getMountJob, err := getScheduler.HandleSchedulerMount("region-a")
	if err != nil {
		t.Fatalf("HandleSchedulerMount(region-a) returned error: %v", err)
	}
	if !getScheduler.clearSchedulerQueueByRegionID("region-a") {
		t.Fatal("expected queue clear to remove mounted job before update branch checks")
	}
	if err := getScheduler.SetSchedulerWorkerHealth(getMountJob.GetSchedulerShardID, SchedulerWorkerHealthDegraded); err != nil {
		t.Fatalf("SetSchedulerWorkerHealth(%q, degraded) returned error: %v", getMountJob.GetSchedulerShardID, err)
	}
	if _, err := getScheduler.HandleSchedulerUpdate("region-a"); err == nil {
		t.Fatal("expected update against degraded shard to fail")
	}
	if err := getScheduler.SetSchedulerWorkerHealth(getMountJob.GetSchedulerShardID, SchedulerWorkerHealthReady); err != nil {
		t.Fatalf("SetSchedulerWorkerHealth(%q, ready) returned error: %v", getMountJob.GetSchedulerShardID, err)
	}
	delete(getScheduler.storeSchedulerWorkerHealthByShardID, getMountJob.GetSchedulerShardID)
	if _, err := getScheduler.HandleSchedulerUpdate("region-a"); err == nil {
		t.Fatal("expected update against shard with missing health state to fail")
	}
}
