package app

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const backgroundJobTypeWeeklySummary = "weekly-summary"
const backgroundJobTypeDunningRetry = "dunning-retry"
const backgroundJobTypeRetentionPurge = "retention-purge"
const backgroundJobTypeHealthScoreRefresh = "health-score-refresh"

type parseBackgroundJobHandlers struct {
	HandleWeeklySummary      func(parseBackgroundJobRow) error
	HandleDunningRetry       func(parseBackgroundJobRow) error
	HandleRetentionPurge     func(parseBackgroundJobRow) error
	HandleHealthScoreRefresh func(parseBackgroundJobRow) error
}

// parseStoreWeeklySummaryJob queues one weekly-summary background job.
func parseStoreWeeklySummaryJob(parseStore *Store, parseJobKey, parsePayloadJSON, parseRunAfter string) error {
	return parseStoreBackgroundJob(parseStore, parseJobKey, backgroundJobTypeWeeklySummary, parsePayloadJSON, parseRunAfter)
}

// parseStoreDunningRetryJob queues one dunning-retry background job.
func parseStoreDunningRetryJob(parseStore *Store, parseJobKey, parsePayloadJSON, parseRunAfter string) error {
	return parseStoreBackgroundJob(parseStore, parseJobKey, backgroundJobTypeDunningRetry, parsePayloadJSON, parseRunAfter)
}

// parseStoreRetentionPurgeJob queues one retention-purge background job.
func parseStoreRetentionPurgeJob(parseStore *Store, parseJobKey, parsePayloadJSON, parseRunAfter string) error {
	return parseStoreBackgroundJob(parseStore, parseJobKey, backgroundJobTypeRetentionPurge, parsePayloadJSON, parseRunAfter)
}

// parseStoreHealthScoreRefreshJob queues one health-score-refresh background job.
func parseStoreHealthScoreRefreshJob(parseStore *Store, parseJobKey, parsePayloadJSON, parseRunAfter string) error {
	return parseStoreBackgroundJob(parseStore, parseJobKey, backgroundJobTypeHealthScoreRefresh, parsePayloadJSON, parseRunAfter)
}

// parseHandleBackgroundJobs dispatches due pending system jobs and persists completion or retry state.
func parseHandleBackgroundJobs(parseStore *Store, parseNow string, parseLimit int64, parseHandlers parseBackgroundJobHandlers) (int, error) {
	if parseStore == nil {
		return 0, errors.New("handle background jobs: store is required")
	}
	parseNow = strings.TrimSpace(parseNow)
	if parseNow == "" {
		return 0, errors.New("handle background jobs: now timestamp is required")
	}
	if _, parseErr := time.Parse(time.RFC3339, parseNow); parseErr != nil {
		return 0, fmt.Errorf("handle background jobs: invalid now timestamp: %w", parseErr)
	}
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseStore.parseListBackgroundJobs(parseLimit)
	if parseErr != nil {
		return 0, parseErr
	}

	parseProcessedCount := 0
	for _, parseRow := range parseRows {
		if strings.TrimSpace(parseRow.Status) != "pending" || !parseHasBackgroundJobTypeSupported(parseRow.JobType) || !parseHasBackgroundJobReady(parseRow.RunAfter, parseNow) {
			continue
		}
		parseAttemptCount := parseRow.AttemptCount + 1
		if parseErr2 := parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
			JobKey:       parseRow.JobKey,
			JobType:      parseRow.JobType,
			QueueKey:     parseRow.QueueKey,
			Status:       "running",
			AttemptCount: parseAttemptCount,
			MaxAttempts:  parseResolveBackgroundJobMaxAttempts(parseRow.MaxAttempts),
			PayloadJSON:  parseRow.PayloadJSON,
			RunAfter:     parseRow.RunAfter,
			StartedAt:    parseNow,
			FinishedAt:   "",
			ErrorMessage: "",
		}); parseErr2 != nil {
			return parseProcessedCount, parseErr2
		}

		parseHandleErr := parseHandleBackgroundJobByType(parseRow, parseHandlers)
		if parseHandleErr == nil {
			if parseErr2 := parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
				JobKey:       parseRow.JobKey,
				JobType:      parseRow.JobType,
				QueueKey:     parseRow.QueueKey,
				Status:       "completed",
				AttemptCount: parseAttemptCount,
				MaxAttempts:  parseResolveBackgroundJobMaxAttempts(parseRow.MaxAttempts),
				PayloadJSON:  parseRow.PayloadJSON,
				RunAfter:     parseNow,
				StartedAt:    parseNow,
				FinishedAt:   parseNow,
				ErrorMessage: "",
			}); parseErr2 != nil {
				return parseProcessedCount, parseErr2
			}
			parseProcessedCount++
			continue
		}

		parseRetryStatus := "failed"
		parseRetryRunAfter := parseNow
		if parseAttemptCount < parseResolveBackgroundJobMaxAttempts(parseRow.MaxAttempts) {
			parseRetryStatus = "pending"
			parseRetryRunAfter = parseBuildBackgroundJobRetryAt(parseNow, time.Minute)
		}
		if parseErr2 := parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
			JobKey:       parseRow.JobKey,
			JobType:      parseRow.JobType,
			QueueKey:     parseRow.QueueKey,
			Status:       parseRetryStatus,
			AttemptCount: parseAttemptCount,
			MaxAttempts:  parseResolveBackgroundJobMaxAttempts(parseRow.MaxAttempts),
			PayloadJSON:  parseRow.PayloadJSON,
			RunAfter:     parseRetryRunAfter,
			StartedAt:    parseNow,
			FinishedAt:   parseNow,
			ErrorMessage: parseHandleErr.Error(),
		}); parseErr2 != nil {
			return parseProcessedCount, parseErr2
		}
		parseProcessedCount++
	}
	return parseProcessedCount, nil
}

// parseStoreBackgroundJob writes one pending system background job row.
func parseStoreBackgroundJob(parseStore *Store, parseJobKey, parseJobType, parsePayloadJSON, parseRunAfter string) error {
	if parseStore == nil {
		return errors.New("store background job: store is required")
	}
	if strings.TrimSpace(parseJobKey) == "" {
		return errors.New("store background job: job key is required")
	}
	if !parseHasBackgroundJobTypeSupported(parseJobType) {
		return fmt.Errorf("store background job: unsupported job type %q", parseJobType)
	}
	if strings.TrimSpace(parseRunAfter) == "" {
		parseRunAfter = time.Now().UTC().Format(time.RFC3339)
	}
	return parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
		JobKey:       parseJobKey,
		JobType:      parseJobType,
		QueueKey:     "system",
		Status:       "pending",
		AttemptCount: 0,
		MaxAttempts:  3,
		PayloadJSON:  parseNormalizeBillingJSON(parsePayloadJSON),
		RunAfter:     strings.TrimSpace(parseRunAfter),
		StartedAt:    "",
		FinishedAt:   "",
		ErrorMessage: "",
	})
}

// parseHandleBackgroundJobByType dispatches one job row to its typed handler callback.
func parseHandleBackgroundJobByType(parseRow parseBackgroundJobRow, parseHandlers parseBackgroundJobHandlers) error {
	switch strings.TrimSpace(parseRow.JobType) {
	case backgroundJobTypeWeeklySummary:
		if parseHandlers.HandleWeeklySummary == nil {
			return nil
		}
		return parseHandlers.HandleWeeklySummary(parseRow)
	case backgroundJobTypeDunningRetry:
		if parseHandlers.HandleDunningRetry == nil {
			return nil
		}
		return parseHandlers.HandleDunningRetry(parseRow)
	case backgroundJobTypeRetentionPurge:
		if parseHandlers.HandleRetentionPurge == nil {
			return nil
		}
		return parseHandlers.HandleRetentionPurge(parseRow)
	case backgroundJobTypeHealthScoreRefresh:
		if parseHandlers.HandleHealthScoreRefresh == nil {
			return nil
		}
		return parseHandlers.HandleHealthScoreRefresh(parseRow)
	default:
		return fmt.Errorf("handle background job by type: unsupported type %q", parseRow.JobType)
	}
}

// parseHasBackgroundJobTypeSupported reports whether one job type is supported by system job flow dispatch.
func parseHasBackgroundJobTypeSupported(parseJobType string) bool {
	switch strings.TrimSpace(parseJobType) {
	case backgroundJobTypeWeeklySummary, backgroundJobTypeDunningRetry, backgroundJobTypeRetentionPurge, backgroundJobTypeHealthScoreRefresh:
		return true
	default:
		return false
	}
}

// parseHasBackgroundJobReady reports whether one pending job is ready for execution at one RFC3339 timestamp.
func parseHasBackgroundJobReady(parseRunAfter string, parseNow string) bool {
	parseRunAfter = strings.TrimSpace(parseRunAfter)
	if parseRunAfter == "" {
		return true
	}
	parseRunAfterTime, parseErr := time.Parse(time.RFC3339, parseRunAfter)
	if parseErr != nil {
		return false
	}
	parseNowTime, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(parseNow))
	if parseErr != nil {
		return false
	}
	return !parseRunAfterTime.After(parseNowTime)
}

// parseResolveBackgroundJobMaxAttempts returns one safe max-attempt value.
func parseResolveBackgroundJobMaxAttempts(parseMaxAttempts int64) int64 {
	if parseMaxAttempts <= 0 {
		return 1
	}
	return parseMaxAttempts
}

// parseBuildBackgroundJobRetryAt returns one retry timestamp advanced by one duration.
func parseBuildBackgroundJobRetryAt(parseNow string, parseDelay time.Duration) string {
	parseNowTime, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(parseNow))
	if parseErr != nil {
		return parseNow
	}
	return parseNowTime.Add(parseDelay).Format(time.RFC3339)
}
