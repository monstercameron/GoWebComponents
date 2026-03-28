package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const backgroundJobTypeWeeklySummary = "weekly-summary"
const backgroundJobTypeDunningRetry = "dunning-retry"
const backgroundJobTypeRetentionPurge = "retention-purge"
const backgroundJobTypeHealthScoreRefresh = "health-score-refresh"

type parseBackgroundJobHandlers struct {
	HandleWeeklySummary      func(context.Context, parseBackgroundJobRow) error
	HandleDunningRetry       func(context.Context, parseBackgroundJobRow) error
	HandleRetentionPurge     func(context.Context, parseBackgroundJobRow) error
	HandleHealthScoreRefresh func(context.Context, parseBackgroundJobRow) error
}

// parseStoreWeeklySummaryJob queues one weekly-summary background job.
func parseStoreWeeklySummaryJob(parseCtx context.Context, parseStore *Store, parseJobKey, parsePayloadJSON, parseRunAfter string) error {
	return parseStoreBackgroundJob(parseCtx, parseStore, parseJobKey, backgroundJobTypeWeeklySummary, parsePayloadJSON, parseRunAfter)
}

// parseStoreDunningRetryJob queues one dunning-retry background job.
func parseStoreDunningRetryJob(parseCtx context.Context, parseStore *Store, parseJobKey, parsePayloadJSON, parseRunAfter string) error {
	return parseStoreBackgroundJob(parseCtx, parseStore, parseJobKey, backgroundJobTypeDunningRetry, parsePayloadJSON, parseRunAfter)
}

// parseStoreRetentionPurgeJob queues one retention-purge background job.
func parseStoreRetentionPurgeJob(parseCtx context.Context, parseStore *Store, parseJobKey, parsePayloadJSON, parseRunAfter string) error {
	return parseStoreBackgroundJob(parseCtx, parseStore, parseJobKey, backgroundJobTypeRetentionPurge, parsePayloadJSON, parseRunAfter)
}

// parseStoreHealthScoreRefreshJob queues one health-score-refresh background job.
func parseStoreHealthScoreRefreshJob(parseCtx context.Context, parseStore *Store, parseJobKey, parsePayloadJSON, parseRunAfter string) error {
	return parseStoreBackgroundJob(parseCtx, parseStore, parseJobKey, backgroundJobTypeHealthScoreRefresh, parsePayloadJSON, parseRunAfter)
}

// parseHandleBackgroundJobs dispatches due pending system jobs and persists completion or retry state.
func parseHandleBackgroundJobs(parseCtx context.Context, parseStore *Store, parseNow string, parseLimit int64, parseHandlers parseBackgroundJobHandlers) (int, error) {
	if parseStore == nil {
		parseErr := errors.New("handle background jobs: store is required")
		slog.Default().Error("background job dispatch failed", slog.String("error", parseErr.Error()))
		return 0, parseErr
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	parseNow = strings.TrimSpace(parseNow)
	if parseNow == "" {
		parseErr := errors.New("handle background jobs: now timestamp is required")
		slog.Default().Error("background job dispatch failed", slog.String("error", parseErr.Error()))
		return 0, parseErr
	}
	if _, parseErr := time.Parse(time.RFC3339, parseNow); parseErr != nil {
		slog.Default().Error("background job dispatch failed", slog.String("error", parseErr.Error()))
		return 0, fmt.Errorf("handle background jobs: invalid now timestamp: %w", parseErr)
	}
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseStore.parseListBackgroundJobs(parseLimit)
	if parseErr != nil {
		slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{ParseAction: "background_job_dispatch"})...).Error("background job list failed", slog.String("error", parseErr.Error()))
		return 0, parseErr
	}

	parseProcessedCount := 0
	for _, parseRow := range parseRows {
		if strings.TrimSpace(parseRow.Status) != "pending" || !parseHasBackgroundJobTypeSupported(parseRow.JobType) || !parseHasBackgroundJobReady(parseRow.RunAfter, parseNow) {
			continue
		}
		if parseOperationalErr := parseRequireBackgroundJobOperational(parseStore, parseRow.QueueKey, parseRow.PayloadJSON); parseOperationalErr != nil {
			slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
				ParseAction:      "background_job_dispatch",
				ParseTargetID:    parseRow.JobKey,
				ParseTargetScope: parseRow.JobType,
				ParseRoute:       parseRow.QueueKey,
			})...).Warn("background job blocked", slog.String("error", parseOperationalErr.Error()))
			if parseErr2 := parseStore.parseUpsertBackgroundJob(parseBackgroundJobWrite{
				JobKey:       parseRow.JobKey,
				JobType:      parseRow.JobType,
				QueueKey:     parseRow.QueueKey,
				Status:       "failed",
				AttemptCount: parseRow.AttemptCount,
				MaxAttempts:  parseResolveBackgroundJobMaxAttempts(parseRow.MaxAttempts),
				PayloadJSON:  parseRow.PayloadJSON,
				RunAfter:     parseNow,
				StartedAt:    parseNow,
				FinishedAt:   parseNow,
				ErrorMessage: parseOperationalErr.Error(),
			}); parseErr2 != nil {
				slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
					ParseAction:      "background_job_dispatch",
					ParseTargetID:    parseRow.JobKey,
					ParseTargetScope: parseRow.JobType,
					ParseRoute:       parseRow.QueueKey,
				})...).Error("background job failure update failed", slog.String("error", parseErr2.Error()))
				return parseProcessedCount, parseErr2
			}
			parseProcessedCount++
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
			slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
				ParseAction:      "background_job_dispatch",
				ParseTargetID:    parseRow.JobKey,
				ParseTargetScope: parseRow.JobType,
				ParseRoute:       parseRow.QueueKey,
			})...).Error("background job running state update failed", slog.String("error", parseErr2.Error()))
			return parseProcessedCount, parseErr2
		}

		parseHandleErr := parseHandleBackgroundJobByType(parseCtx, parseRow, parseHandlers)
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
				slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
					ParseAction:      "background_job_dispatch",
					ParseTargetID:    parseRow.JobKey,
					ParseTargetScope: parseRow.JobType,
					ParseRoute:       parseRow.QueueKey,
				})...).Error("background job completion update failed", slog.String("error", parseErr2.Error()))
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
		slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
			ParseAction:      "background_job_dispatch",
			ParseTargetID:    parseRow.JobKey,
			ParseTargetScope: parseRow.JobType,
			ParseRoute:       parseRow.QueueKey,
		})...).Error("background job handler failed", slog.String("error", parseHandleErr.Error()))
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
			slog.Default().With(parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
				ParseAction:      "background_job_dispatch",
				ParseTargetID:    parseRow.JobKey,
				ParseTargetScope: parseRow.JobType,
				ParseRoute:       parseRow.QueueKey,
			})...).Error("background job retry update failed", slog.String("error", parseErr2.Error()))
			return parseProcessedCount, parseErr2
		}
		parseProcessedCount++
	}
	return parseProcessedCount, nil
}

// parseResolveWorkspaceIDFromBackgroundQueueKey parses one queue key formatted as `workspace:<id>`.
func parseResolveWorkspaceIDFromBackgroundQueueKey(parseQueueKey string) int64 {
	parseQueueKey = strings.TrimSpace(strings.ToLower(parseQueueKey))
	if !strings.HasPrefix(parseQueueKey, "workspace:") {
		return 0
	}
	parseWorkspaceID, parseErr := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(parseQueueKey, "workspace:")), 10, 64)
	if parseErr != nil || parseWorkspaceID <= 0 {
		return 0
	}
	return parseWorkspaceID
}

// parseStoreBackgroundJob writes one pending system background job row.
func parseStoreBackgroundJob(parseCtx context.Context, parseStore *Store, parseJobKey, parseJobType, parsePayloadJSON, parseRunAfter string) error {
	if parseStore == nil {
		return errors.New("store background job: store is required")
	}
	if strings.TrimSpace(parseJobKey) == "" {
		return errors.New("store background job: job key is required")
	}
	if !parseHasBackgroundJobTypeSupported(parseJobType) {
		return fmt.Errorf("store background job: unsupported job type %q", parseJobType)
	}
	parsePayloadJSON = parseMergeJSONObjectStrings(parseNormalizeBillingJSON(parsePayloadJSON), parseBuildTraceabilityJSON(parseCtx))
	if parseErr := parseRequireBackgroundJobOperational(parseStore, "system", parsePayloadJSON); parseErr != nil {
		return parseErr
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
		PayloadJSON:  parsePayloadJSON,
		RunAfter:     strings.TrimSpace(parseRunAfter),
		StartedAt:    "",
		FinishedAt:   "",
		ErrorMessage: "",
	})
}

// parseRequireBackgroundJobOperational enforces user-disable and workspace-suspend policy for queued job scope.
func parseRequireBackgroundJobOperational(parseStore *Store, parseQueueKey string, parsePayloadJSON string) error {
	if parseStore == nil {
		return errors.New("require background job operational: store is required")
	}
	parseWorkspaceID := parseResolveBackgroundJobWorkspaceID(parseQueueKey, parsePayloadJSON)
	if parseWorkspaceID > 0 {
		if parseErr := parseStore.parseRequireWorkspaceOperational(parseWorkspaceID); parseErr != nil {
			return parseErr
		}
	}
	parseUserID := parseResolveBackgroundJobUserID(parsePayloadJSON)
	if parseUserID > 0 {
		if parseErr := parseStore.parseRequireUserOperational(parseUserID); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

// parseResolveBackgroundJobWorkspaceID resolves one workspace id from queue key first, then payload JSON scope.
func parseResolveBackgroundJobWorkspaceID(parseQueueKey string, parsePayloadJSON string) int64 {
	if parseWorkspaceID := parseResolveWorkspaceIDFromBackgroundQueueKey(parseQueueKey); parseWorkspaceID > 0 {
		return parseWorkspaceID
	}
	parsePayloadMap := parseResolveBackgroundJobPayloadMap(parsePayloadJSON)
	return parseResolveBackgroundJobScopeID(parsePayloadMap, "workspace_id", "workspaceId")
}

// parseResolveBackgroundJobUserID resolves one user id from job payload JSON scope.
func parseResolveBackgroundJobUserID(parsePayloadJSON string) int64 {
	parsePayloadMap := parseResolveBackgroundJobPayloadMap(parsePayloadJSON)
	return parseResolveBackgroundJobScopeID(parsePayloadMap, "user_id", "userId")
}

// parseResolveBackgroundJobPayloadMap parses one payload JSON object into a key-value map for scope extraction.
func parseResolveBackgroundJobPayloadMap(parsePayloadJSON string) map[string]any {
	parsePayloadJSON = strings.TrimSpace(parsePayloadJSON)
	if parsePayloadJSON == "" {
		return nil
	}
	parsePayloadMap := map[string]any{}
	if parseErr := json.Unmarshal([]byte(parsePayloadJSON), &parsePayloadMap); parseErr != nil {
		return nil
	}
	return parsePayloadMap
}

// parseResolveBackgroundJobScopeID resolves one numeric scope id by key variants from one parsed payload map.
func parseResolveBackgroundJobScopeID(parsePayloadMap map[string]any, parseKeys ...string) int64 {
	if len(parsePayloadMap) == 0 {
		return 0
	}
	for _, parseKey := range parseKeys {
		parseValue, hasParseValue := parsePayloadMap[strings.TrimSpace(parseKey)]
		if !hasParseValue {
			continue
		}
		parseScopeID, isParseFound := parseResolveBackgroundJobScopeValue(parseValue)
		if !isParseFound {
			continue
		}
		return parseScopeID
	}
	return 0
}

// parseResolveBackgroundJobScopeValue normalizes one payload scope value into a positive int64 id.
func parseResolveBackgroundJobScopeValue(parseValue any) (int64, bool) {
	switch parseTyped := parseValue.(type) {
	case float64:
		parseScopeID := int64(parseTyped)
		return parseScopeID, parseScopeID > 0 && float64(parseScopeID) == parseTyped
	case string:
		parseScopeID, parseErr := strconv.ParseInt(strings.TrimSpace(parseTyped), 10, 64)
		if parseErr != nil || parseScopeID <= 0 {
			return 0, false
		}
		return parseScopeID, true
	case json.Number:
		parseScopeID, parseErr := parseTyped.Int64()
		if parseErr != nil || parseScopeID <= 0 {
			return 0, false
		}
		return parseScopeID, true
	default:
		return 0, false
	}
}

// parseHandleBackgroundJobByType dispatches one job row to its typed handler callback.
func parseHandleBackgroundJobByType(parseCtx context.Context, parseRow parseBackgroundJobRow, parseHandlers parseBackgroundJobHandlers) error {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	switch strings.TrimSpace(parseRow.JobType) {
	case backgroundJobTypeWeeklySummary:
		if parseHandlers.HandleWeeklySummary == nil {
			return nil
		}
		return parseHandlers.HandleWeeklySummary(parseCtx, parseRow)
	case backgroundJobTypeDunningRetry:
		if parseHandlers.HandleDunningRetry == nil {
			return nil
		}
		return parseHandlers.HandleDunningRetry(parseCtx, parseRow)
	case backgroundJobTypeRetentionPurge:
		if parseHandlers.HandleRetentionPurge == nil {
			return nil
		}
		return parseHandlers.HandleRetentionPurge(parseCtx, parseRow)
	case backgroundJobTypeHealthScoreRefresh:
		if parseHandlers.HandleHealthScoreRefresh == nil {
			return nil
		}
		return parseHandlers.HandleHealthScoreRefresh(parseCtx, parseRow)
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
