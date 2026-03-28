package app

import (
	"errors"
	"testing"
)

// TestHandleBackgroundJobsLifecycle verifies typed system-job flow scheduling, retry, and completion state transitions.
func TestHandleBackgroundJobsLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseNow := "2026-03-27T22:30:00Z"

	if parseErr := parseStoreWeeklySummaryJob(parseStore, "job-weekly", `{}`, parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreWeeklySummaryJob: %v", parseErr)
	}
	if parseErr := parseStoreDunningRetryJob(parseStore, "job-dunning", `{}`, parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreDunningRetryJob: %v", parseErr)
	}
	if parseErr := parseStoreRetentionPurgeJob(parseStore, "job-retention", `{}`, parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreRetentionPurgeJob: %v", parseErr)
	}
	if parseErr := parseStoreHealthScoreRefreshJob(parseStore, "job-health", `{}`, "2026-03-27T23:30:00Z"); parseErr != nil {
		parseT.Fatalf("parseStoreHealthScoreRefreshJob: %v", parseErr)
	}

	parseProcessed, parseErr := parseHandleBackgroundJobs(parseStore, parseNow, 20, parseBackgroundJobHandlers{
		HandleDunningRetry: func(parseRow parseBackgroundJobRow) error {
			return errors.New("billing provider timeout")
		},
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleBackgroundJobs first pass: %v", parseErr)
	}
	if parseProcessed != 3 {
		parseT.Fatalf("expected 3 processed rows, got %d", parseProcessed)
	}

	parseRows, parseErr := parseStore.parseListBackgroundJobs(20)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs first pass: %v", parseErr)
	}
	parseRowsByKey := make(map[string]parseBackgroundJobRow, len(parseRows))
	for _, parseRow := range parseRows {
		parseRowsByKey[parseRow.JobKey] = parseRow
	}
	if parseRow, hasParseRow := parseRowsByKey["job-weekly"]; !hasParseRow || parseRow.Status != "completed" {
		parseT.Fatalf("unexpected weekly row after first pass: found=%v row=%+v", hasParseRow, parseRow)
	}
	if parseRow, hasParseRow := parseRowsByKey["job-retention"]; !hasParseRow || parseRow.Status != "completed" {
		parseT.Fatalf("unexpected retention row after first pass: found=%v row=%+v", hasParseRow, parseRow)
	}
	if parseRow, hasParseRow := parseRowsByKey["job-dunning"]; !hasParseRow || parseRow.Status != "pending" || parseRow.AttemptCount != 1 {
		parseT.Fatalf("unexpected dunning row after first pass: found=%v row=%+v", hasParseRow, parseRow)
	}
	if parseRow, hasParseRow := parseRowsByKey["job-health"]; !hasParseRow || parseRow.Status != "pending" || parseRow.AttemptCount != 0 {
		parseT.Fatalf("unexpected health row after first pass: found=%v row=%+v", hasParseRow, parseRow)
	}

	parseProcessed, parseErr = parseHandleBackgroundJobs(parseStore, "2026-03-27T22:31:30Z", 20, parseBackgroundJobHandlers{})
	if parseErr != nil {
		parseT.Fatalf("parseHandleBackgroundJobs second pass: %v", parseErr)
	}
	if parseProcessed != 1 {
		parseT.Fatalf("expected 1 processed row on second pass, got %d", parseProcessed)
	}

	parseRows, parseErr = parseStore.parseListBackgroundJobs(20)
	if parseErr != nil {
		parseT.Fatalf("parseListBackgroundJobs second pass: %v", parseErr)
	}
	parseRowsByKey = make(map[string]parseBackgroundJobRow, len(parseRows))
	for _, parseRow := range parseRows {
		parseRowsByKey[parseRow.JobKey] = parseRow
	}
	if parseRow, hasParseRow := parseRowsByKey["job-dunning"]; !hasParseRow || parseRow.Status != "completed" || parseRow.AttemptCount != 2 {
		parseT.Fatalf("unexpected dunning row after second pass: found=%v row=%+v", hasParseRow, parseRow)
	}
	if parseRow, hasParseRow := parseRowsByKey["job-health"]; !hasParseRow || parseRow.Status != "pending" {
		parseT.Fatalf("unexpected health row after second pass: found=%v row=%+v", hasParseRow, parseRow)
	}
}

// BenchmarkResolveBackgroundJobScopeIDs measures payload scope extraction cost for blocked-job policy checks.
func BenchmarkResolveBackgroundJobScopeIDs(parseB *testing.B) {
	parsePayloadJSON := `{"workspace_id":42,"user_id":"108","nested":{"ignored":true}}`
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseWorkspaceID := parseResolveBackgroundJobWorkspaceID("system", parsePayloadJSON)
		parseUserID := parseResolveBackgroundJobUserID(parsePayloadJSON)
		if parseWorkspaceID != 42 || parseUserID != 108 {
			parseB.Fatalf("unexpected scope ids workspace=%d user=%d", parseWorkspaceID, parseUserID)
		}
	}
}
