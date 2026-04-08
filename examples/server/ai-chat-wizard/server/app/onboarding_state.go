package app

import (
	"encoding/json"
	"log/slog"
	"time"
)

const parseStarterMilestoneFirstRunDetected = "starter.first_run_detected"
const parseStarterMilestoneReturningUserDetected = "starter.returning_user_detected"
const parseStarterMilestoneFirstThreadCreated = "starter.first_thread_created"
const parseStarterMilestoneFirstReplyCompleted = "starter.first_reply_completed"

type parseStarterStateSummary struct {
	isBrandNewUser      bool
	hasPriorThreads     bool
	isFirstReplyTracked bool
}

// parseResolveStarterStateForConversationCount resolves one user starter-state summary from conversation count and persisted milestones.
func (parseS *chatServer) parseResolveStarterStateForConversationCount(parseUserID int64, parseConversationCount int) parseStarterStateSummary {
	parseSummary := parseStarterStateSummary{
		isBrandNewUser:  parseConversationCount <= 0,
		hasPriorThreads: parseConversationCount > 0,
	}
	if parseS == nil || parseS.store == nil || parseUserID <= 0 {
		return parseSummary
	}
	parseMilestoneRows, parseErr := parseS.store.parseListUserActivationMilestones(500)
	if parseErr != nil {
		return parseSummary
	}
	for _, parseMilestoneRow := range parseMilestoneRows {
		if parseMilestoneRow.UserID != parseUserID {
			continue
		}
		if parseMilestoneRow.MilestoneKey == parseStarterMilestoneFirstReplyCompleted && parseMilestoneRow.Status == "completed" {
			parseSummary.isFirstReplyTracked = true
			break
		}
	}
	if parseSummary.hasPriorThreads || parseSummary.isFirstReplyTracked {
		parseSummary.isBrandNewUser = false
	}
	return parseSummary
}

// parseSyncStarterStateMilestones ensures first-run vs returning-user starter milestones are persisted for one user.
func (parseS *chatServer) parseSyncStarterStateMilestones(parseUserID int64, parseSummary parseStarterStateSummary) {
	if parseS == nil || parseS.store == nil || parseUserID <= 0 {
		return
	}
	if parseSummary.isBrandNewUser {
		parseS.parseUpsertStarterMilestone(parseUserID, parseStarterMilestoneFirstRunDetected, map[string]any{
			"state": "brand_new_user",
		})
	}
	if parseSummary.hasPriorThreads {
		parseS.parseUpsertStarterMilestone(parseUserID, parseStarterMilestoneReturningUserDetected, map[string]any{
			"state": "returning_user",
		})
	}
}

// parseUpsertStarterMilestone persists one starter-state milestone row with best-effort logging on failure.
func (parseS *chatServer) parseUpsertStarterMilestone(parseUserID int64, parseMilestoneKey string, parseMetadata map[string]any) {
	if parseS == nil || parseS.store == nil || parseUserID <= 0 {
		return
	}
	parseMetadataJSON := parseBuildStarterMilestoneMetadataJSON(parseMetadata)
	parseNow := time.Now().UTC().Format(time.RFC3339)
	if parseErr := parseS.store.parseUpsertUserActivationMilestone(parseUserActivationMilestoneWrite{
		UserID:       parseUserID,
		MilestoneKey: parseMilestoneKey,
		Status:       "completed",
		AchievedAt:   parseNow,
		MetadataJSON: parseMetadataJSON,
	}); parseErr != nil {
		parseLogger := parseS.logger
		if parseLogger == nil {
			parseLogger = slog.Default()
		}
		parseLogger.Warn("starter milestone upsert failed",
			slog.Int64("user_id", parseUserID),
			slog.String("milestone_key", parseMilestoneKey),
			slog.String("error", parseErr.Error()),
		)
	}
}

// parseBuildStarterMilestoneMetadataJSON encodes one starter-state milestone metadata payload.
func parseBuildStarterMilestoneMetadataJSON(parseMetadata map[string]any) string {
	if len(parseMetadata) == 0 {
		return "{}"
	}
	parsePayload, parseErr := json.Marshal(parseMetadata)
	if parseErr != nil {
		return "{}"
	}
	return string(parsePayload)
}
