package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"
)

// TestHandleBackgroundJobsLifecycle verifies typed system-job flow scheduling, retry, and completion state transitions.
func TestHandleBackgroundJobsLifecycle(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseNow := "2026-03-27T22:30:00Z"

	if parseErr := parseStoreWeeklySummaryJob(context.Background(), parseStore, "job-weekly", `{}`, parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreWeeklySummaryJob: %v", parseErr)
	}
	if parseErr := parseStoreDunningRetryJob(context.Background(), parseStore, "job-dunning", `{}`, parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreDunningRetryJob: %v", parseErr)
	}
	if parseErr := parseStoreRetentionPurgeJob(context.Background(), parseStore, "job-retention", `{}`, parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreRetentionPurgeJob: %v", parseErr)
	}
	if parseErr := parseStoreHealthScoreRefreshJob(context.Background(), parseStore, "job-health", `{}`, "2026-03-27T23:30:00Z"); parseErr != nil {
		parseT.Fatalf("parseStoreHealthScoreRefreshJob: %v", parseErr)
	}

	parseProcessed, parseErr := parseHandleBackgroundJobs(context.Background(), parseStore, parseNow, 20, parseBackgroundJobHandlers{
		HandleDunningRetry: func(context.Context, parseBackgroundJobRow) error {
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

	parseProcessed, parseErr = parseHandleBackgroundJobs(context.Background(), parseStore, "2026-03-27T22:31:30Z", 20, parseBackgroundJobHandlers{})
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

// TestHandleBackgroundJobsLogsHandlerFailures verifies failed job handlers emit one boundary log entry.
func TestHandleBackgroundJobsLogsHandlerFailures(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseNow := "2026-03-27T22:30:00Z"
	if parseErr := parseStoreWeeklySummaryJob(context.Background(), parseStore, "job-weekly-log", `{}`, parseNow); parseErr != nil {
		parseT.Fatalf("parseStoreWeeklySummaryJob: %v", parseErr)
	}
	var parseBuffer bytes.Buffer
	parseOriginalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&parseBuffer, nil)))
	parseT.Cleanup(func() {
		slog.SetDefault(parseOriginalLogger)
	})

	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-job-log-123",
		correlationIDMetadataKey, "corr-job-log-456",
	))
	parseProcessed, parseErr := parseHandleBackgroundJobs(parseCtx, parseStore, parseNow, 10, parseBackgroundJobHandlers{
		HandleWeeklySummary: func(context.Context, parseBackgroundJobRow) error {
			return errors.New("billing provider timeout")
		},
	})
	if parseErr != nil {
		parseT.Fatalf("parseHandleBackgroundJobs: %v", parseErr)
	}
	if parseProcessed != 1 {
		parseT.Fatalf("expected 1 processed row, got %d", parseProcessed)
	}
	parseLogOutput := parseBuffer.String()
	if !strings.Contains(parseLogOutput, "background job handler failed") || !strings.Contains(parseLogOutput, "job-weekly-log") {
		parseT.Fatalf("expected job failure log, got %q", parseLogOutput)
	}
}
