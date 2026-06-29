package app

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseAdminChatSlowReplyTokenThreshold int64 = 1200
const parseAdminChatHighCostThreadThresholdUSD float64 = 5

// parseFilterAdminChatFeatureUsageRowsByScope filters feature-usage rows to one workspace-admin scope.
func parseFilterAdminChatFeatureUsageRowsByScope(parseRows []parseAdminChatFeatureUsageRow, parseWorkspaceIDs map[int64]struct{}) []parseAdminChatFeatureUsageRow {
	parseFilteredRows := make([]parseAdminChatFeatureUsageRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseHasAdminWorkspaceScope(parseWorkspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseBuildAdminChatFeatureUsageSlices aggregates per-workspace feature rows into one typed feature-usage slice list.
func parseBuildAdminChatFeatureUsageSlices(parseRows []parseAdminChatFeatureUsageRow, parseLimit int32) []*chatpb.AdminChatFeatureUsageSlice {
	type parseFeatureAggregate struct {
		parseEventCount     int64
		parseUserCount      int64
		parseWorkspaceCount int64
		parseLastSeenAt     string
		parseWorkspaceSet   map[int64]struct{}
	}
	parseFeatureAggregates := make(map[string]*parseFeatureAggregate, len(parseRows))
	for _, parseRow := range parseRows {
		parseFeatureKey := strings.TrimSpace(parseRow.FeatureKey)
		if parseFeatureKey == "" {
			parseFeatureKey = "unknown_feature"
		}
		parseAggregate, hasParseAggregate := parseFeatureAggregates[parseFeatureKey]
		if !hasParseAggregate {
			parseAggregate = &parseFeatureAggregate{
				parseWorkspaceSet: make(map[int64]struct{}),
			}
			parseFeatureAggregates[parseFeatureKey] = parseAggregate
		}
		parseAggregate.parseEventCount += parseRow.EventCount
		parseAggregate.parseUserCount += parseRow.UserCount
		if parseRow.WorkspaceID > 0 {
			if _, hasParseWorkspace := parseAggregate.parseWorkspaceSet[parseRow.WorkspaceID]; !hasParseWorkspace {
				parseAggregate.parseWorkspaceSet[parseRow.WorkspaceID] = struct{}{}
				parseAggregate.parseWorkspaceCount++
			}
		}
		if strings.TrimSpace(parseRow.LastSeenAt) > strings.TrimSpace(parseAggregate.parseLastSeenAt) {
			parseAggregate.parseLastSeenAt = strings.TrimSpace(parseRow.LastSeenAt)
		}
	}
	parseSlices := make([]*chatpb.AdminChatFeatureUsageSlice, 0, len(parseFeatureAggregates))
	for parseFeatureKey, parseAggregate := range parseFeatureAggregates {
		parseSlices = append(parseSlices, &chatpb.AdminChatFeatureUsageSlice{
			FeatureKey:     parseFeatureKey,
			EventCount:     parseAggregate.parseEventCount,
			UserCount:      parseAggregate.parseUserCount,
			WorkspaceCount: parseAggregate.parseWorkspaceCount,
			LastSeenAt:     parseAggregate.parseLastSeenAt,
		})
	}
	sort.SliceStable(parseSlices, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftSlice := parseSlices[parseLeftIndex]
		parseRightSlice := parseSlices[parseRightIndex]
		if parseLeftSlice.GetEventCount() == parseRightSlice.GetEventCount() {
			if strings.TrimSpace(parseLeftSlice.GetLastSeenAt()) == strings.TrimSpace(parseRightSlice.GetLastSeenAt()) {
				return strings.TrimSpace(parseLeftSlice.GetFeatureKey()) < strings.TrimSpace(parseRightSlice.GetFeatureKey())
			}
			return strings.TrimSpace(parseLeftSlice.GetLastSeenAt()) > strings.TrimSpace(parseRightSlice.GetLastSeenAt())
		}
		return parseLeftSlice.GetEventCount() > parseRightSlice.GetEventCount()
	})
	return parseApplyAdminSliceWindow(parseSlices, 0, parseLimit)
}

// parseBuildAdminChatsAnomaliesSummary derives one typed chats-anomaly summary from anomaly row slices.
func parseBuildAdminChatsAnomaliesSummary(
	parseFailedRows []parseAdminUsageEventRow,
	parseSlowRows []parseAdminUsageEventRow,
	parseHighCostRows []parseAdminConversationRow,
	parseFeatureSlices []*chatpb.AdminChatFeatureUsageSlice,
) *chatpb.AdminChatsAnomaliesSummary {
	parseSummary := &chatpb.AdminChatsAnomaliesSummary{
		FailedReplyCount:       int64(len(parseFailedRows)),
		SlowReplyCount:         int64(len(parseSlowRows)),
		HighCostThreadCount:    int64(len(parseHighCostRows)),
		FeatureUsageSliceCount: int64(len(parseFeatureSlices)),
	}
	for _, parseConversationRow := range parseHighCostRows {
		parseSummary.HighCostTotalUsd += parseConversationRow.TotalCostUSD
	}
	return parseSummary
}

// GetAdminChatsAnomalies returns typed failed/slow/high-cost/feature-usage anomaly slices for dashboard chats surfaces.
func (parseS *chatServer) GetAdminChatsAnomalies(parseCtx context.Context, parseReq *chatpb.GetAdminChatsAnomaliesRequest) (*chatpb.GetAdminChatsAnomaliesResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminChatsAnomalies"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.chats.anomalies")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLookbackDays int32
	var parseLimit int32
	if parseReq != nil {
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"chats-anomalies",
		"Admin chats anomalies slice viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	parseResponse := &chatpb.GetAdminChatsAnomaliesResponse{
		Summary:            &chatpb.AdminChatsAnomaliesSummary{},
		FailedReplies:      make([]*chatpb.AdminUsageEvent, 0),
		SlowReplies:        make([]*chatpb.AdminUsageEvent, 0),
		HighCostThreads:    make([]*chatpb.AdminConversationSummary, 0),
		FeatureUsageSlices: make([]*chatpb.AdminChatFeatureUsageSlice, 0),
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminChatsAnomalies: store unavailable",
			slog.String("next_action", "restore store availability before retrying chats anomaly slices"),
		)
		return parseResponse, nil
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseFailedRows, parseErr := parseS.store.parseListAdminChatFailedReplies(parseSince, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list chat failed replies: %v", parseErr)
	}
	parseSlowRows, parseErr := parseS.store.parseListAdminChatSlowReplies(parseSince, parseAdminChatSlowReplyTokenThreshold, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list chat slow replies: %v", parseErr)
	}
	parseHighCostRows, parseErr := parseS.store.parseListAdminChatHighCostThreads(parseSince, parseAdminChatHighCostThreadThresholdUSD, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list chat high-cost threads: %v", parseErr)
	}
	parseFeatureRows, parseErr := parseS.store.parseListAdminChatFeatureUsageSlices(parseSince, parseQueryLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list chat feature-usage slices: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseFailedRows = parseFilterAdminUsageRowsByScope(parseFailedRows, parseScope.userIDs)
		parseSlowRows = parseFilterAdminUsageRowsByScope(parseSlowRows, parseScope.userIDs)
		parseHighCostRows = parseFilterAdminConversationRowsByScope(parseHighCostRows, parseScope.userIDs)
		parseFeatureRows = parseFilterAdminChatFeatureUsageRowsByScope(parseFeatureRows, parseScope.workspaceIDs)
	}
	parseFeatureSlices := parseBuildAdminChatFeatureUsageSlices(parseFeatureRows, parseLimit)
	parseResponse.Summary = parseBuildAdminChatsAnomaliesSummary(parseFailedRows, parseSlowRows, parseHighCostRows, parseFeatureSlices)
	parseFailedRows = parseApplyAdminSliceWindow(parseFailedRows, 0, parseLimit)
	parseSlowRows = parseApplyAdminSliceWindow(parseSlowRows, 0, parseLimit)
	parseHighCostRows = parseApplyAdminSliceWindow(parseHighCostRows, 0, parseLimit)

	for _, parseFailedRow := range parseFailedRows {
		parseResponse.FailedReplies = append(parseResponse.FailedReplies, parseRedactAdminUsageEventByScope(parseScope, parseBuildAdminUsageEvent(parseFailedRow)))
	}
	for _, parseSlowRow := range parseSlowRows {
		parseResponse.SlowReplies = append(parseResponse.SlowReplies, parseRedactAdminUsageEventByScope(parseScope, parseBuildAdminUsageEvent(parseSlowRow)))
	}
	for _, parseHighCostRow := range parseHighCostRows {
		parseResponse.HighCostThreads = append(parseResponse.HighCostThreads, parseRedactAdminConversationSummaryByScope(parseScope, parseBuildAdminConversationSummary(parseHighCostRow)))
	}
	parseResponse.FeatureUsageSlices = append(parseResponse.FeatureUsageSlices, parseFeatureSlices...)
	parseLogger.Info(
		"rpc.GetAdminChatsAnomalies: complete",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("failed_replies", len(parseResponse.FailedReplies)),
		slog.Int("slow_replies", len(parseResponse.SlowReplies)),
		slog.Int("high_cost_threads", len(parseResponse.HighCostThreads)),
		slog.Int("feature_usage_slices", len(parseResponse.FeatureUsageSlices)),
	)
	return parseResponse, nil
}
