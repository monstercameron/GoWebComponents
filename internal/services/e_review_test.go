package services

import "testing"

// TestHeartbeatTimeoutNeverImprovesDeadWorker verifies a worker already marked
// dead stays dead after a heartbeat timeout, so it can still be replaced.
func TestHeartbeatTimeoutNeverImprovesDeadWorker(parseT *testing.T) {
	parseDispatcher := NewDispatcher([]WorkerID{"w1", "w2"})
	if parseErr := parseDispatcher.SetWorkerHealth("w1", HealthDead); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseHealth, parseErr := parseDispatcher.RecordHeartbeatTimeout("w1")
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseHealth != HealthDead || parseDispatcher.WorkerHealthOf("w1") != HealthDead {
		parseT.Fatalf("timeout resurrected a dead worker to %s", parseHealth)
	}
	if _, parseReplaceErr := parseDispatcher.ReplaceWorker("w1", "w3"); parseReplaceErr != nil {
		parseT.Fatalf("dead worker could no longer be replaced: %v", parseReplaceErr)
	}
}

// TestRestoreRejectsEpochEqualToFloorAfterDeathFallback verifies the epoch of a
// worker that just died cannot be reused to restore the unit.
func TestRestoreRejectsEpochEqualToFloorAfterDeathFallback(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Restore("u", 5); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if _, parseErr := parseArbiter.HandleRemoteDeath("u", false, 0, 1); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseErr := parseArbiter.Restore("u", 5); parseErr == nil {
		parseT.Fatal("restore at the already-used epoch 5 must be refused")
	}
	if parseErr := parseArbiter.Restore("u", 6); parseErr != nil {
		parseT.Fatalf("restore at a fresh epoch must succeed: %v", parseErr)
	}
}
