package app

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseNormalizeAdminCustomerTimelineSource normalizes one timeline source key into the supported source enum.
func parseNormalizeAdminCustomerTimelineSource(parseSource string) parseAdminCustomerTimelineSource {
	switch strings.TrimSpace(strings.ToLower(parseSource)) {
	case string(parseAdminCustomerTimelineSourceChatContent):
		return parseAdminCustomerTimelineSourceChatContent
	case string(parseAdminCustomerTimelineSourceBilling):
		return parseAdminCustomerTimelineSourceBilling
	case string(parseAdminCustomerTimelineSourceSupport):
		return parseAdminCustomerTimelineSourceSupport
	case string(parseAdminCustomerTimelineSourceAuthSession):
		return parseAdminCustomerTimelineSourceAuthSession
	case string(parseAdminCustomerTimelineSourceAuditEvent):
		return parseAdminCustomerTimelineSourceAuditEvent
	default:
		return parseAdminCustomerTimelineSource(strings.TrimSpace(strings.ToLower(parseSource)))
	}
}

// parseFilterAdminCustomerTimelineRowsByWorkspaceScope filters workspace-scoped timeline rows while preserving user-scoped global rows.
func parseFilterAdminCustomerTimelineRowsByWorkspaceScope(parseRows []parseAdminCustomerTimelineEventRow, parseWorkspaceIDs map[int64]struct{}) []parseAdminCustomerTimelineEventRow {
	parseFilteredRows := make([]parseAdminCustomerTimelineEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID > 0 && !parseHasAdminWorkspaceScope(parseWorkspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseBuildAdminCustomerTimelineEvent maps one unified store timeline row into protobuf form with optional workspace-scope redaction.
func parseBuildAdminCustomerTimelineEvent(parseRow parseAdminCustomerTimelineEventRow, isParseRedacted bool) *chatpb.AdminCustomerTimelineEvent {
	parseTimelineID := strings.TrimSpace(parseRow.TimelineID)
	parseSource := parseNormalizeAdminCustomerTimelineSource(parseRow.Source)
	parseSummary := strings.TrimSpace(parseRow.Summary)
	if parseSummary == "" {
		parseSummary = strings.TrimSpace(parseRow.EventType)
	}
	parseDetailJSON := strings.TrimSpace(parseRow.DetailJSON)
	if parseDetailJSON == "" {
		parseDetailJSON = "{}"
	}
	if isParseRedacted {
		switch parseSource {
		case parseAdminCustomerTimelineSourceChatContent, parseAdminCustomerTimelineSourceSupport:
			if parseSummary != "" {
				parseSummary = parseWorkspaceScopeRedactionText
			}
		}
		parseDetailJSON = "{}"
	}
	if parseTimelineID == "" {
		parseTimelineID = string(parseSource) + ":" + strings.TrimSpace(parseRow.CreatedAt)
	}
	return &chatpb.AdminCustomerTimelineEvent{
		TimelineId:        parseTimelineID,
		Source:            string(parseSource),
		EventType:         strings.TrimSpace(parseRow.EventType),
		UserId:            parseRow.UserID,
		WorkspaceId:       parseRow.WorkspaceID,
		ConversationId:    parseRow.ConversationID,
		SupportTicketId:   parseRow.SupportTicketID,
		BillingCustomerId: parseRow.BillingCustomerID,
		AuthSessionId:     parseRow.AuthSessionID,
		AuditLogId:        parseRow.AuditLogID,
		Summary:           parseSummary,
		DetailJson:        parseDetailJSON,
		CreatedAt:         strings.TrimSpace(parseRow.CreatedAt),
	}
}

// GetAdminCustomerAccountTimeline returns one typed unified customer timeline for chats, support, billing, auth sessions, and audit events.
func (parseS *chatServer) GetAdminCustomerAccountTimeline(parseCtx context.Context, parseReq *chatpb.GetAdminCustomerAccountTimelineRequest) (*chatpb.GetAdminCustomerAccountTimelineResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminCustomerAccountTimeline"))
	parseFetchStart := time.Now()

	var parseUserID int64
	var parseLookbackDays int32
	var parseLimit int32
	if parseReq != nil {
		parseUserID = parseReq.GetUserId()
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
	}
	if parseUserID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)

	parseScope, isParseRedacted, parseErr := parseS.parseAuthorizeAdminCustomerTimelineScope(parseCtx, parseUserID, parseAdminCustomerTimelineSourceChatContent)
	if parseErr != nil {
		return nil, parseErr
	}
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.GetAdminCustomerAccountTimeline: drill-down fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.Int64("target_user_id", parseUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"user",
		strconv.FormatInt(parseUserID, 10),
		"Admin customer timeline viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)

	parseResponse := &chatpb.GetAdminCustomerAccountTimelineResponse{
		Events: make([]*chatpb.AdminCustomerTimelineEvent, 0),
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminCustomerAccountTimeline: store unavailable",
			slog.String("next_action", "restore store availability before retrying customer timeline"),
		)
		return parseResponse, nil
	}

	parseUserRow, hasParseUserRow, parseErr := parseS.store.parseGetAdminUserSummaryByUserID(parseUserID)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminCustomerAccountTimeline: user summary query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get admin user summary: %v", parseErr)
	}
	if !hasParseUserRow {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	parseResponse.User = parseBuildAdminUserSummary(parseUserRow)

	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseTimelineRows, parseErr := parseS.store.parseListAdminCustomerAccountTimelineEvents(parseUserID, parseSince, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminCustomerAccountTimeline: timeline query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list customer account timeline events: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseTimelineRows = parseFilterAdminCustomerTimelineRowsByWorkspaceScope(parseTimelineRows, parseScope.workspaceIDs)
		parseTimelineRows = parseApplyAdminSliceWindow(parseTimelineRows, 0, parseLimit)
	}
	for _, parseTimelineRow := range parseTimelineRows {
		parseSource := parseNormalizeAdminCustomerTimelineSource(parseTimelineRow.Source)
		if !parseIsAdminCustomerTimelineSourceSupported(parseSource) {
			continue
		}
		parseResponse.Events = append(parseResponse.Events, parseBuildAdminCustomerTimelineEvent(parseTimelineRow, isParseRedacted))
	}

	parseFetchDuration := time.Since(parseFetchStart)
	parseLogAdminFetchOutcome(parseLogger, "rpc.GetAdminCustomerAccountTimeline", "drilldown", parseScopeType, parseFetchDuration, len(parseResponse.Events))
	parseLogger.Info(
		"rpc.GetAdminCustomerAccountTimeline: complete",
		slog.String("scope", parseScopeType),
		slog.Int64("target_user_id", parseUserID),
		slog.Int("event_count", len(parseResponse.Events)),
		slog.Duration("duration", parseFetchDuration),
	)
	return parseResponse, nil
}
