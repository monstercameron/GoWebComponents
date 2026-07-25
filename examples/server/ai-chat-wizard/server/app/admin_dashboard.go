package app

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultAdminLookbackDays = 30
const maxAdminLookbackDays = 365
const defaultAdminTopLimit = 8
const maxAdminTopLimit = 25
const defaultAdminRecentLimit = 10
const maxAdminRecentLimit = 50
const defaultAdminListLimit = 25
const maxAdminListLimit = 100
const parseAdminScopedScanLimit int64 = 5000
const parseAdminFetchSlowThreshold = 1500 * time.Millisecond

// parseLogAdminFetchOutcome emits one actionable warning for empty or slow dashboard fetch outcomes.
func parseLogAdminFetchOutcome(parseLogger *slog.Logger, parseRPC string, parseFetchType string, parseScopeType string, parseDuration time.Duration, parseCount int) {
	if parseLogger == nil {
		return
	}
	if parseCount == 0 {
		parseLogger.Warn(
			parseRPC+": empty fetch",
			slog.String("fetch_type", parseFetchType),
			slog.String("scope", parseScopeType),
			slog.Int("count", parseCount),
			slog.Duration("duration", parseDuration),
			slog.String("next_action", "verify filters, lookback window, and upstream ingestion before retrying"),
		)
		return
	}
	if parseDuration >= parseAdminFetchSlowThreshold {
		parseLogger.Warn(
			parseRPC+": slow fetch",
			slog.String("fetch_type", parseFetchType),
			slog.String("scope", parseScopeType),
			slog.Int("count", parseCount),
			slog.Duration("duration", parseDuration),
			slog.String("next_action", "reduce query scope or check database latency before retrying"),
		)
	}
}

// parseClampAdminLookbackDays clamps one dashboard lookback window into a safe range.
func parseClampAdminLookbackDays(parseDays int32) int32 {
	if parseDays <= 0 {
		return defaultAdminLookbackDays
	}
	if parseDays > maxAdminLookbackDays {
		return maxAdminLookbackDays
	}
	return parseDays
}

// parseClampAdminTopLimit clamps one dashboard leaderboard size into a safe range.
func parseClampAdminTopLimit(parseLimit int32) int32 {
	if parseLimit <= 0 {
		return defaultAdminTopLimit
	}
	if parseLimit > maxAdminTopLimit {
		return maxAdminTopLimit
	}
	return parseLimit
}

// parseClampAdminRecentLimit clamps one dashboard recent-activity list size into a safe range.
func parseClampAdminRecentLimit(parseLimit int32) int32 {
	if parseLimit <= 0 {
		return defaultAdminRecentLimit
	}
	if parseLimit > maxAdminRecentLimit {
		return maxAdminRecentLimit
	}
	return parseLimit
}

// parseClampAdminListLimit clamps one admin list RPC size into a safe range.
func parseClampAdminListLimit(parseLimit int32) int32 {
	if parseLimit <= 0 {
		return defaultAdminListLimit
	}
	if parseLimit > maxAdminListLimit {
		return maxAdminListLimit
	}
	return parseLimit
}

// parseBuildAdminSinceTimestamp formats one UTC lookback cutoff for text-stored timestamps.
func parseBuildAdminSinceTimestamp(parseLookbackDays int32) string {
	parseSince := time.Now().UTC().Add(-time.Duration(parseClampAdminLookbackDays(parseLookbackDays)) * 24 * time.Hour)
	return parseSince.Format(time.RFC3339)
}

// parseFormatAdminTimestamp formats one optional time for dashboard payloads.
func parseFormatAdminTimestamp(parseValue time.Time) string {
	if parseValue.IsZero() {
		return ""
	}
	return parseValue.UTC().Format(time.RFC3339)
}

// parseBuildAdminRateLimitBucket maps one provider rate-limit bucket into protobuf form.
func parseBuildAdminRateLimitBucket(parseBucket provider.RateLimitBucket) *chatpb.AdminRateLimitBucket {
	return &chatpb.AdminRateLimitBucket{
		Limit:        parseBucket.Limit,
		Remaining:    parseBucket.Remaining,
		ResetSeconds: int64(parseBucket.ResetAfter / time.Second),
	}
}

// parseBuildAdminRateLimitSnapshot maps one provider rate-limit snapshot into protobuf form.
func parseBuildAdminRateLimitSnapshot(parseSnapshot provider.RateLimitSnapshot) *chatpb.AdminRateLimitSnapshot {
	return &chatpb.AdminRateLimitSnapshot{
		RequestsPerMinute: parseBuildAdminRateLimitBucket(parseSnapshot.RequestsPerMinute),
		RequestsPerHour:   parseBuildAdminRateLimitBucket(parseSnapshot.RequestsPerHour),
		RequestsPerDay:    parseBuildAdminRateLimitBucket(parseSnapshot.RequestsPerDay),
		TokensPerMinute:   parseBuildAdminRateLimitBucket(parseSnapshot.TokensPerMinute),
		TokensPerHour:     parseBuildAdminRateLimitBucket(parseSnapshot.TokensPerHour),
		TokensPerDay:      parseBuildAdminRateLimitBucket(parseSnapshot.TokensPerDay),
		LastUpdated:       parseFormatAdminTimestamp(parseSnapshot.LastUpdated),
		Source:            parseSnapshot.Source,
		HasData:           !parseSnapshot.ParseEmpty(),
	}
}

// parseBuildAdminProviderSnapshots maps configured providers and live health into protobuf form.
func parseBuildAdminProviderSnapshots(parseRegistry *provider.Registry) []*chatpb.AdminProviderSnapshot {
	if parseRegistry == nil {
		return nil
	}
	parseInfos := parseRegistry.ParseProviderInfos()
	parseHealthRows := parseRegistry.ParseHealthSnapshots()
	parseHealthByProviderID := make(map[string]provider.ProviderHealth, len(parseHealthRows))
	for _, parseHealthRow := range parseHealthRows {
		parseHealthByProviderID[parseHealthRow.ProviderID] = parseHealthRow
	}

	parseSnapshots := make([]*chatpb.AdminProviderSnapshot, 0, len(parseInfos))
	for _, parseInfo := range parseInfos {
		parseHealthRow := parseHealthByProviderID[parseInfo.ID]
		parseSnapshots = append(parseSnapshots, &chatpb.AdminProviderSnapshot{
			ProviderId:         parseInfo.ID,
			Label:              parseInfo.Label,
			AuthConfigured:     parseInfo.AuthConfigured,
			Available:          parseInfo.Available,
			StreamingSupported: parseInfo.StreamingSupported,
			ReasoningSupported: parseInfo.ReasoningSupported,
			ToolUseSupported:   parseInfo.ToolUseSupported,
			Notes:              append([]string(nil), parseInfo.Notes...),
			Status:             string(parseHealthRow.Status),
			LastSuccess:        parseFormatAdminTimestamp(parseHealthRow.LastSuccess),
			LastFailure:        parseFormatAdminTimestamp(parseHealthRow.LastFailure),
			LastError:          parseHealthRow.LastError,
			LastLatencyMs:      int64(parseHealthRow.LastLatency / time.Millisecond),
			RequestCount:       parseHealthRow.RequestCount,
			TokenCount:         parseHealthRow.TokenCount,
			RateLimits:         parseBuildAdminRateLimitSnapshot(parseHealthRow.RateLimits),
		})
	}
	return parseSnapshots
}

// parseBuildAdminUsageEvent maps one admin usage row into protobuf form.
func parseBuildAdminUsageEvent(parseRow parseAdminUsageEventRow) *chatpb.AdminUsageEvent {
	return &chatpb.AdminUsageEvent{
		EventId:                 parseRow.EventID,
		UserId:                  parseRow.UserID,
		Email:                   parseRow.Email,
		DisplayName:             parseRow.DisplayName,
		ConversationId:          parseRow.ConversationID,
		ConversationPublicId:    parseRow.ConversationPublicID,
		ConversationTitle:       parseRow.ConversationTitle,
		ProviderId:              parseRow.ProviderID,
		ModelId:                 parseRow.ModelID,
		PromptTokens:            parseRow.PromptTokens,
		CompletionTokens:        parseRow.CompletionTokens,
		UsageSource:             parseRow.UsageSource,
		ProviderRequestId:       parseRow.ProviderRequestID,
		InputCostPerMillionUsd:  parseRow.InputCostPerMillionUSD,
		OutputCostPerMillionUsd: parseRow.OutputCostPerMillionUSD,
		PricingCurrency:         parseRow.PricingCurrency,
		InputCostUsd:            parseRow.InputCostUSD,
		OutputCostUsd:           parseRow.OutputCostUSD,
		TotalCostUsd:            parseRow.TotalCostUSD,
		ClientId:                parseRow.ClientID,
		TraceId:                 parseRow.TraceID,
		SpanId:                  parseRow.SpanID,
		TraceState:              parseRow.TraceState,
		Status:                  parseRow.Status,
		ErrorMessage:            parseRow.ErrorMessage,
		CreatedAt:               parseRow.CreatedAt,
	}
}

// parseBuildAdminUserSummary maps one admin user row into protobuf form.
func parseBuildAdminUserSummary(parseRow parseAdminUserRow) *chatpb.AdminUserSummary {
	return &chatpb.AdminUserSummary{
		UserId:                  parseRow.UserID,
		Email:                   parseRow.Email,
		DisplayName:             parseRow.DisplayName,
		CreatedAt:               parseRow.CreatedAt,
		ConversationCount:       parseRow.ConversationCount,
		MessageCount:            parseRow.MessageCount,
		UsageEventCount:         parseRow.UsageEventCount,
		TotalCostUsd:            parseRow.TotalCostUSD,
		LastSeenAt:              parseRow.LastSeenAt,
		WorkspaceCount:          parseRow.WorkspaceCount,
		MemoryCount:             parseRow.MemoryCount,
		OpenSupportTicketCount:  parseRow.OpenSupportCount,
		ActiveSubscriptionCount: parseRow.ActiveSubCount,
		TokenVersion:            parseRow.TokenVersion,
		SupportMessageCount:     parseRow.SupportMsgCount,
		ActiveSessionCount:      parseRow.ActiveSessionCount,
	}
}

// parseBuildAdminConversationSummary maps one admin conversation row into protobuf form.
func parseBuildAdminConversationSummary(parseRow parseAdminConversationRow) *chatpb.AdminConversationSummary {
	return &chatpb.AdminConversationSummary{
		ConversationId:  parseRow.ConversationID,
		PublicId:        parseRow.PublicID,
		UserId:          parseRow.UserID,
		Email:           parseRow.Email,
		DisplayName:     parseRow.DisplayName,
		StartedAt:       parseRow.StartedAt,
		Preview:         parseRow.Preview,
		MessageCount:    parseRow.MessageCount,
		UsageEventCount: parseRow.UsageEventCount,
		TotalCostUsd:    parseRow.TotalCostUSD,
		LastActivityAt:  parseRow.LastActivityAt,
	}
}

// parseBuildAdminAuthSessionEntry maps one auth-session row into protobuf form.
func parseBuildAdminAuthSessionEntry(parseRow parseAuthSessionRow) *chatpb.AuthSessionEntry {
	return &chatpb.AuthSessionEntry{
		Id:           parseRow.ID,
		UserId:       parseRow.UserID,
		SessionId:    parseRow.SessionID,
		TokenVersion: parseRow.TokenVersion,
		UserAgent:    parseRow.UserAgent,
		IpAddress:    parseRow.IPAddress,
		LastSeenAt:   parseRow.LastSeenAt,
		ExpiresAt:    parseRow.ExpiresAt,
		RevokedAt:    parseRow.RevokedAt,
		CreatedAt:    parseRow.CreatedAt,
		UpdatedAt:    parseRow.UpdatedAt,
	}
}

// parseBuildAdminUserMemoryEntry maps one user-memory row into protobuf form.
func parseBuildAdminUserMemoryEntry(parseRow userMemoryRow) *chatpb.UserMemory {
	return &chatpb.UserMemory{
		Key:             parseRow.Key,
		Category:        parseRow.Category,
		Summary:         parseRow.Summary,
		Detail:          parseRow.Detail,
		SourceMessage:   parseRow.SourceMessage,
		UsefulnessScore: int32(parseRow.UsefulnessScore),
		ConfidenceScore: parseRow.ConfidenceScore,
		RubricReason:    parseRow.RubricReason,
		UpdatedAt:       parseRow.UpdatedAt,
	}
}

// parseBuildAdminAuditLogEntry maps one audit-log row into protobuf form.
func parseBuildAdminAuditLogEntry(parseRow parseAuditLogRow) *chatpb.AuditLogEntry {
	return &chatpb.AuditLogEntry{
		Id:          parseRow.ID,
		ActorUserId: parseRow.ActorUserID,
		WorkspaceId: parseRow.WorkspaceID,
		EventType:   parseRow.EventType,
		TargetType:  parseRow.TargetType,
		TargetId:    parseRow.TargetID,
		Summary:     parseRow.Summary,
		PayloadJson: parseRow.PayloadJSON,
		CreatedAt:   parseRow.CreatedAt,
	}
}

// parseFilterAdminAuditRowsByWorkspaceScope filters audit rows to one workspace-admin scope.
func parseFilterAdminAuditRowsByWorkspaceScope(parseRows []parseAuditLogRow, parseWorkspaceIDs map[int64]struct{}) []parseAuditLogRow {
	parseFilteredRows := make([]parseAuditLogRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if !parseHasAdminWorkspaceScope(parseWorkspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseLimitAdminAuditRows truncates audit rows to one RPC-safe limit.
func parseLimitAdminAuditRows(parseRows []parseAuditLogRow, parseLimit int32) []parseAuditLogRow {
	if parseLimit <= 0 || len(parseRows) <= int(parseLimit) {
		return parseRows
	}
	return parseRows[:parseLimit]
}

// parseFilterAdminUserRowsBySearch filters admin-user rows by one optional search term.
func parseFilterAdminUserRowsBySearch(parseRows []parseAdminUserRow, parseSearch string) []parseAdminUserRow {
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	if parseSearch == "" {
		return parseRows
	}
	parseFilteredRows := make([]parseAdminUserRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if strings.Contains(strconv.FormatInt(parseRow.UserID, 10), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Email)), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.DisplayName)), parseSearch) {
			parseFilteredRows = append(parseFilteredRows, parseRow)
		}
	}
	return parseFilteredRows
}

// parseSortAdminUserRows sorts admin-user rows by one typed sort key and direction when provided.
func parseSortAdminUserRows(parseRows []parseAdminUserRow, parseSortBy string, isParseSortAscending bool) {
	parseSortBy = strings.TrimSpace(strings.ToLower(parseSortBy))
	if parseSortBy == "" {
		return
	}
	sort.SliceStable(parseRows, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftRow := parseRows[parseLeftIndex]
		parseRightRow := parseRows[parseRightIndex]
		switch parseSortBy {
		case "user_id":
			return parseCompareAdminInt64(parseLeftRow.UserID, parseRightRow.UserID, isParseSortAscending)
		case "email":
			return parseCompareAdminString(parseLeftRow.Email, parseRightRow.Email, isParseSortAscending)
		case "display_name":
			return parseCompareAdminString(parseLeftRow.DisplayName, parseRightRow.DisplayName, isParseSortAscending)
		case "created_at":
			return parseCompareAdminString(parseLeftRow.CreatedAt, parseRightRow.CreatedAt, isParseSortAscending)
		case "last_seen_at":
			return parseCompareAdminString(parseLeftRow.LastSeenAt, parseRightRow.LastSeenAt, isParseSortAscending)
		case "conversation_count":
			return parseCompareAdminInt64(parseLeftRow.ConversationCount, parseRightRow.ConversationCount, isParseSortAscending)
		case "message_count":
			return parseCompareAdminInt64(parseLeftRow.MessageCount, parseRightRow.MessageCount, isParseSortAscending)
		case "usage_event_count":
			return parseCompareAdminInt64(parseLeftRow.UsageEventCount, parseRightRow.UsageEventCount, isParseSortAscending)
		case "total_cost_usd":
			return parseCompareAdminFloat64(parseLeftRow.TotalCostUSD, parseRightRow.TotalCostUSD, isParseSortAscending)
		case "workspace_count":
			return parseCompareAdminInt64(parseLeftRow.WorkspaceCount, parseRightRow.WorkspaceCount, isParseSortAscending)
		case "memory_count":
			return parseCompareAdminInt64(parseLeftRow.MemoryCount, parseRightRow.MemoryCount, isParseSortAscending)
		case "open_support_ticket_count":
			return parseCompareAdminInt64(parseLeftRow.OpenSupportCount, parseRightRow.OpenSupportCount, isParseSortAscending)
		case "active_subscription_count":
			return parseCompareAdminInt64(parseLeftRow.ActiveSubCount, parseRightRow.ActiveSubCount, isParseSortAscending)
		case "token_version":
			return parseCompareAdminInt64(parseLeftRow.TokenVersion, parseRightRow.TokenVersion, isParseSortAscending)
		case "support_message_count":
			return parseCompareAdminInt64(parseLeftRow.SupportMsgCount, parseRightRow.SupportMsgCount, isParseSortAscending)
		case "active_session_count":
			return parseCompareAdminInt64(parseLeftRow.ActiveSessionCount, parseRightRow.ActiveSessionCount, isParseSortAscending)
		default:
			return false
		}
	})
}

// parseFilterAdminUsageRowsByListQuery filters admin-usage rows by one optional search term and typed status/provider filters.
func parseFilterAdminUsageRowsByListQuery(parseRows []parseAdminUsageEventRow, parseSearch string, parseStatusFilter string, parseProviderFilter string) []parseAdminUsageEventRow {
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	parseStatusFilter = strings.TrimSpace(strings.ToLower(parseStatusFilter))
	parseProviderFilter = strings.TrimSpace(strings.ToLower(parseProviderFilter))
	parseFilteredRows := make([]parseAdminUsageEventRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseStatusFilter != "" && strings.TrimSpace(strings.ToLower(parseRow.Status)) != parseStatusFilter {
			continue
		}
		if parseProviderFilter != "" && strings.TrimSpace(strings.ToLower(parseRow.ProviderID)) != parseProviderFilter {
			continue
		}
		if parseSearch != "" {
			if !strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.EventID)), parseSearch) &&
				!strings.Contains(strconv.FormatInt(parseRow.UserID, 10), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Email)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.ProviderID)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.ModelID)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Status)), parseSearch) &&
				!strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.ClientID)), parseSearch) {
				continue
			}
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseSortAdminUsageRows sorts admin-usage rows by one typed sort key and direction when provided.
func parseSortAdminUsageRows(parseRows []parseAdminUsageEventRow, parseSortBy string, isParseSortAscending bool) {
	parseSortBy = strings.TrimSpace(strings.ToLower(parseSortBy))
	if parseSortBy == "" {
		return
	}
	sort.SliceStable(parseRows, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftRow := parseRows[parseLeftIndex]
		parseRightRow := parseRows[parseRightIndex]
		switch parseSortBy {
		case "created_at":
			return parseCompareAdminString(parseLeftRow.CreatedAt, parseRightRow.CreatedAt, isParseSortAscending)
		case "total_cost_usd":
			return parseCompareAdminFloat64(parseLeftRow.TotalCostUSD, parseRightRow.TotalCostUSD, isParseSortAscending)
		case "prompt_tokens":
			return parseCompareAdminInt64(parseLeftRow.PromptTokens, parseRightRow.PromptTokens, isParseSortAscending)
		case "completion_tokens":
			return parseCompareAdminInt64(parseLeftRow.CompletionTokens, parseRightRow.CompletionTokens, isParseSortAscending)
		case "user_id":
			return parseCompareAdminInt64(parseLeftRow.UserID, parseRightRow.UserID, isParseSortAscending)
		case "provider_id":
			return parseCompareAdminString(parseLeftRow.ProviderID, parseRightRow.ProviderID, isParseSortAscending)
		case "model_id":
			return parseCompareAdminString(parseLeftRow.ModelID, parseRightRow.ModelID, isParseSortAscending)
		default:
			return false
		}
	})
}

// parseFilterAdminConversationRowsBySearch filters admin-conversation rows by one optional search term.
func parseFilterAdminConversationRowsBySearch(parseRows []parseAdminConversationRow, parseSearch string) []parseAdminConversationRow {
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	if parseSearch == "" {
		return parseRows
	}
	parseFilteredRows := make([]parseAdminConversationRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if strings.Contains(strconv.FormatInt(parseRow.ConversationID, 10), parseSearch) ||
			strings.Contains(strconv.FormatInt(parseRow.UserID, 10), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.PublicID)), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Email)), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.DisplayName)), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Preview)), parseSearch) {
			parseFilteredRows = append(parseFilteredRows, parseRow)
		}
	}
	return parseFilteredRows
}

// parseSortAdminConversationRows sorts admin-conversation rows by one typed sort key and direction when provided.
func parseSortAdminConversationRows(parseRows []parseAdminConversationRow, parseSortBy string, isParseSortAscending bool) {
	parseSortBy = strings.TrimSpace(strings.ToLower(parseSortBy))
	if parseSortBy == "" {
		return
	}
	sort.SliceStable(parseRows, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftRow := parseRows[parseLeftIndex]
		parseRightRow := parseRows[parseRightIndex]
		switch parseSortBy {
		case "conversation_id":
			return parseCompareAdminInt64(parseLeftRow.ConversationID, parseRightRow.ConversationID, isParseSortAscending)
		case "user_id":
			return parseCompareAdminInt64(parseLeftRow.UserID, parseRightRow.UserID, isParseSortAscending)
		case "public_id":
			return parseCompareAdminString(parseLeftRow.PublicID, parseRightRow.PublicID, isParseSortAscending)
		case "started_at":
			return parseCompareAdminString(parseLeftRow.StartedAt, parseRightRow.StartedAt, isParseSortAscending)
		case "last_activity_at":
			return parseCompareAdminString(parseLeftRow.LastActivityAt, parseRightRow.LastActivityAt, isParseSortAscending)
		case "message_count":
			return parseCompareAdminInt64(parseLeftRow.MessageCount, parseRightRow.MessageCount, isParseSortAscending)
		case "usage_event_count":
			return parseCompareAdminInt64(parseLeftRow.UsageEventCount, parseRightRow.UsageEventCount, isParseSortAscending)
		case "total_cost_usd":
			return parseCompareAdminFloat64(parseLeftRow.TotalCostUSD, parseRightRow.TotalCostUSD, isParseSortAscending)
		default:
			return false
		}
	})
}

// parseCompareAdminString compares one string pair using one requested direction.
func parseCompareAdminString(parseLeft string, parseRight string, isParseSortAscending bool) bool {
	if isParseSortAscending {
		return strings.Compare(parseLeft, parseRight) < 0
	}
	return strings.Compare(parseLeft, parseRight) > 0
}

// parseCompareAdminInt64 compares one int64 pair using one requested direction.
func parseCompareAdminInt64(parseLeft int64, parseRight int64, isParseSortAscending bool) bool {
	if isParseSortAscending {
		return parseLeft < parseRight
	}
	return parseLeft > parseRight
}

// parseCompareAdminFloat64 compares one float64 pair using one requested direction.
func parseCompareAdminFloat64(parseLeft float64, parseRight float64, isParseSortAscending bool) bool {
	if isParseSortAscending {
		return parseLeft < parseRight
	}
	return parseLeft > parseRight
}

// GetAdminDashboard returns one global dashboard snapshot for authenticated superusers.
func (parseS *chatServer) GetAdminDashboard(parseCtx context.Context, parseReq *chatpb.GetAdminDashboardRequest) (*chatpb.GetAdminDashboardResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminDashboard"))
	parseFetchStart := time.Now()
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.home")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLookbackDays int32
	var parseTopLimit int32
	var parseRecentLimit int32
	if parseReq != nil {
		parseLookbackDays = parseReq.GetLookbackDays()
		parseTopLimit = parseReq.GetTopLimit()
		parseRecentLimit = parseReq.GetRecentLimit()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseTopLimit = parseClampAdminTopLimit(parseTopLimit)
	parseRecentLimit = parseClampAdminRecentLimit(parseRecentLimit)
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.GetAdminDashboard: bootstrap",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("top_limit", int(parseTopLimit)),
		slog.Int("recent_limit", int(parseRecentLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.entry",
		"dashboard",
		"home",
		"Admin dashboard home viewed",
		"{}",
		0,
	)
	if !parseScope.isPlatformScope {
		return parseS.parseGetWorkspaceScopedAdminDashboard(parseLogger, parseScope, parseLookbackDays, parseTopLimit, parseRecentLimit)
	}
	parseResponse := &chatpb.GetAdminDashboardResponse{
		ProviderSnapshots: parseBuildAdminProviderSnapshots(parseS.providerRegistry),
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminDashboard: store unavailable, returning provider snapshots only",
			slog.String("next_action", "restore store availability before retrying dashboard bootstrap"),
		)
		return parseResponse, nil
	}

	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseSummary, parseErr := parseS.store.parseGetAdminDashboardSummary(parseSince)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: summary query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get admin dashboard summary: %v", parseErr)
	}
	parseDailyRows, parseErr := parseS.store.parseListAdminDashboardDailyUsage(parseSince)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: daily usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin dashboard daily usage: %v", parseErr)
	}
	parseProviderRows, parseErr := parseS.store.parseListAdminDashboardProviderUsage(parseSince, int64(parseTopLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: provider usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin dashboard provider usage: %v", parseErr)
	}
	parseModelRows, parseErr := parseS.store.parseListAdminDashboardModelUsage(parseSince, int64(parseTopLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: model usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin dashboard model usage: %v", parseErr)
	}
	parseUserRows, parseErr := parseS.store.parseListAdminDashboardUserUsage(parseSince, int64(parseTopLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: top users query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin dashboard user usage: %v", parseErr)
	}
	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEvents(parseSince, int64(parseRecentLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: usage events query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin usage events: %v", parseErr)
	}
	parseRecentUsers, parseErr := parseS.store.parseListAdminUsers(int64(parseRecentLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: users query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin users: %v", parseErr)
	}
	parseRecentConversations, parseErr := parseS.store.parseListAdminConversations(int64(parseRecentLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: conversations query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin conversations: %v", parseErr)
	}

	parseResponse.Summary = &chatpb.AdminDashboardSummary{
		TotalUsers:                  parseSummary.TotalUsers,
		TotalConversations:          parseSummary.TotalConversations,
		TotalMessages:               parseSummary.TotalMessages,
		WindowNewUsers:              parseSummary.WindowNewUsers,
		WindowNewConversations:      parseSummary.WindowNewConversations,
		WindowNewMessages:           parseSummary.WindowNewMessages,
		WindowUsageEvents:           parseSummary.WindowUsageEvents,
		WindowTotalCostUsd:          parseSummary.WindowTotalCostUSD,
		WindowPromptTokens:          parseSummary.WindowPromptTokens,
		WindowCompletionTokens:      parseSummary.WindowCompletionTokens,
		WindowActiveUsers:           parseSummary.WindowActiveUsers,
		WindowActiveConversations:   parseSummary.WindowActiveConversations,
		WindowActiveClients:         parseSummary.WindowActiveClients,
		WindowCompletedEvents:       parseSummary.WindowCompletedEvents,
		WindowFailedEvents:          parseSummary.WindowFailedEvents,
		WindowBillingEvents:         parseSummary.WindowBillingEvents,
		WindowOpenInvoices:          parseSummary.WindowOpenInvoices,
		WindowOpenDunningEvents:     parseSummary.WindowOpenDunningEvents,
		OpenIncidents:               parseSummary.OpenIncidents,
		OpenSupportTickets:          parseSummary.OpenSupportTickets,
		ActiveExperiments:           parseSummary.ActiveExperiments,
		UnhealthyExperiments:        parseSummary.UnhealthyExperiments,
		WindowExperimentAssignments: parseSummary.WindowExperimentAssignments,
	}

	parseResponse.DailyUsage = make([]*chatpb.AdminDashboardDailyUsage, 0, len(parseDailyRows))
	for _, parseRow := range parseDailyRows {
		parseResponse.DailyUsage = append(parseResponse.DailyUsage, &chatpb.AdminDashboardDailyUsage{
			UsageDay:            parseRow.UsageDay,
			UsageEventCount:     parseRow.UsageEventCount,
			TotalCostUsd:        parseRow.TotalCostUSD,
			PromptTokens:        parseRow.PromptTokens,
			CompletionTokens:    parseRow.CompletionTokens,
			ActiveUsers:         parseRow.ActiveUsers,
			ActiveConversations: parseRow.ActiveConversations,
			ActiveClients:       parseRow.ActiveClients,
		})
	}
	parseResponse.Providers = make([]*chatpb.AdminProviderUsage, 0, len(parseProviderRows))
	for _, parseRow := range parseProviderRows {
		parseResponse.Providers = append(parseResponse.Providers, &chatpb.AdminProviderUsage{
			ProviderId:          parseRow.ProviderID,
			UsageEventCount:     parseRow.UsageEventCount,
			TotalCostUsd:        parseRow.TotalCostUSD,
			PromptTokens:        parseRow.PromptTokens,
			CompletionTokens:    parseRow.CompletionTokens,
			ActiveUsers:         parseRow.ActiveUsers,
			CompletedEventCount: parseRow.CompletedEventCount,
			FailedEventCount:    parseRow.FailedEventCount,
		})
	}
	parseResponse.Models = make([]*chatpb.AdminModelUsage, 0, len(parseModelRows))
	for _, parseRow := range parseModelRows {
		parseResponse.Models = append(parseResponse.Models, &chatpb.AdminModelUsage{
			ProviderId:          parseRow.ProviderID,
			ModelId:             parseRow.ModelID,
			UsageEventCount:     parseRow.UsageEventCount,
			TotalCostUsd:        parseRow.TotalCostUSD,
			PromptTokens:        parseRow.PromptTokens,
			CompletionTokens:    parseRow.CompletionTokens,
			ActiveUsers:         parseRow.ActiveUsers,
			CompletedEventCount: parseRow.CompletedEventCount,
			FailedEventCount:    parseRow.FailedEventCount,
		})
	}
	parseResponse.TopUsers = make([]*chatpb.AdminUserUsage, 0, len(parseUserRows))
	for _, parseRow := range parseUserRows {
		parseResponse.TopUsers = append(parseResponse.TopUsers, &chatpb.AdminUserUsage{
			UserId:            parseRow.UserID,
			Email:             parseRow.Email,
			DisplayName:       parseRow.DisplayName,
			ConversationCount: parseRow.ConversationCount,
			MessageCount:      parseRow.MessageCount,
			UsageEventCount:   parseRow.UsageEventCount,
			TotalCostUsd:      parseRow.TotalCostUSD,
			PromptTokens:      parseRow.PromptTokens,
			CompletionTokens:  parseRow.CompletionTokens,
			LastSeenAt:        parseRow.LastSeenAt,
		})
	}
	parseResponse.RecentUsageEvents = make([]*chatpb.AdminUsageEvent, 0, len(parseUsageRows))
	for _, parseRow := range parseUsageRows {
		parseResponse.RecentUsageEvents = append(parseResponse.RecentUsageEvents, parseBuildAdminUsageEvent(parseRow))
	}
	parseResponse.RecentUsers = make([]*chatpb.AdminUserSummary, 0, len(parseRecentUsers))
	for _, parseRow := range parseRecentUsers {
		parseResponse.RecentUsers = append(parseResponse.RecentUsers, parseBuildAdminUserSummary(parseRow))
	}
	parseResponse.RecentConversations = make([]*chatpb.AdminConversationSummary, 0, len(parseRecentConversations))
	for _, parseRow := range parseRecentConversations {
		parseResponse.RecentConversations = append(parseResponse.RecentConversations, parseBuildAdminConversationSummary(parseRow))
	}
	parseFetchDuration := time.Since(parseFetchStart)
	if len(parseResponse.DailyUsage) == 0 || len(parseResponse.Providers) == 0 || len(parseResponse.Models) == 0 || len(parseResponse.TopUsers) == 0 {
		parseLogger.Warn(
			"rpc.GetAdminDashboard: partial summary payload",
			slog.String("scope", parseScopeType),
			slog.Int("daily_rows", len(parseResponse.DailyUsage)),
			slog.Int("provider_rows", len(parseResponse.Providers)),
			slog.Int("model_rows", len(parseResponse.Models)),
			slog.Int("top_user_rows", len(parseResponse.TopUsers)),
			slog.Duration("duration", parseFetchDuration),
			slog.String("next_action", "check summary aggregation queries and dashboard lookback scope before retrying"),
		)
	}
	parseAggregateCount := len(parseResponse.DailyUsage) + len(parseResponse.Providers) + len(parseResponse.Models) + len(parseResponse.TopUsers) + len(parseResponse.RecentUsageEvents) + len(parseResponse.RecentUsers) + len(parseResponse.RecentConversations)
	parseLogAdminFetchOutcome(parseLogger, "rpc.GetAdminDashboard", "summary", parseScopeType, parseFetchDuration, parseAggregateCount)

	parseLogger.Info(
		"rpc.GetAdminDashboard: complete",
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("top_limit", int(parseTopLimit)),
		slog.Int("recent_limit", int(parseRecentLimit)),
		slog.Int("daily_rows", len(parseResponse.DailyUsage)),
		slog.Int("provider_rows", len(parseResponse.Providers)),
		slog.Int("model_rows", len(parseResponse.Models)),
		slog.Duration("duration", parseFetchDuration),
	)
	return parseResponse, nil
}

// parseGetWorkspaceScopedAdminDashboard builds one workspace-admin scoped dashboard snapshot.
func (parseS *chatServer) parseGetWorkspaceScopedAdminDashboard(parseLogger *slog.Logger, parseScope parseAdminAccessScope, parseLookbackDays int32, parseTopLimit int32, parseRecentLimit int32) (*chatpb.GetAdminDashboardResponse, error) {
	parseFetchStart := time.Now()
	parseLogger.Info(
		"rpc.GetAdminDashboard: bootstrap workspace scope",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.Int("workspace_count", len(parseScope.workspaceIDs)),
		slog.Int("scoped_user_count", len(parseScope.userIDs)),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("top_limit", int(parseTopLimit)),
		slog.Int("recent_limit", int(parseRecentLimit)),
	)
	parseResponse := &chatpb.GetAdminDashboardResponse{
		ProviderSnapshots: parseBuildAdminProviderSnapshots(parseS.providerRegistry),
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminDashboard: store unavailable for workspace scope",
			slog.String("next_action", "restore store availability before retrying workspace dashboard bootstrap"),
		)
		return parseResponse, nil
	}

	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEvents(parseSince, parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: scoped usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list scoped admin usage events: %v", parseErr)
	}
	parseUserRows, parseErr := parseS.store.parseListAdminUsers(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: scoped users query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list scoped admin users: %v", parseErr)
	}
	parseConversationRows, parseErr := parseS.store.parseListAdminConversations(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminDashboard: scoped conversations query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list scoped admin conversations: %v", parseErr)
	}

	parseScopedUsageRows := parseFilterAdminUsageRowsByScope(parseUsageRows, parseScope.userIDs)
	parseScopedUserRows := parseFilterAdminUserRowsByScope(parseUserRows, parseScope.userIDs)
	parseScopedConversationRows := parseFilterAdminConversationRowsByScope(parseConversationRows, parseScope.userIDs)

	parseRecentUsageRows := parseLimitAdminUsageRows(parseScopedUsageRows, parseRecentLimit)
	parseRecentUserRows := parseLimitAdminUserRows(parseScopedUserRows, parseRecentLimit)
	parseRecentConversationRows := parseLimitAdminConversationRows(parseScopedConversationRows, parseRecentLimit)
	parseTopUserRows := parseLimitAdminUserRows(parseScopedUserRows, parseTopLimit)

	parseResponse.Summary = parseBuildWorkspaceScopedAdminSummary(parseScopedUsageRows, parseScopedUserRows, parseScopedConversationRows)
	parseResponse.TopUsers = make([]*chatpb.AdminUserUsage, 0, len(parseTopUserRows))
	for _, parseRow := range parseTopUserRows {
		parseResponse.TopUsers = append(parseResponse.TopUsers, &chatpb.AdminUserUsage{
			UserId:            parseRow.UserID,
			Email:             parseRow.Email,
			DisplayName:       parseRow.DisplayName,
			ConversationCount: parseRow.ConversationCount,
			MessageCount:      parseRow.MessageCount,
			UsageEventCount:   parseRow.UsageEventCount,
			TotalCostUsd:      parseRow.TotalCostUSD,
			LastSeenAt:        parseRow.LastSeenAt,
		})
	}
	parseResponse.RecentUsageEvents = make([]*chatpb.AdminUsageEvent, 0, len(parseRecentUsageRows))
	for _, parseRow := range parseRecentUsageRows {
		parseResponse.RecentUsageEvents = append(parseResponse.RecentUsageEvents, parseBuildAdminUsageEvent(parseRow))
	}
	parseResponse.RecentUsers = make([]*chatpb.AdminUserSummary, 0, len(parseRecentUserRows))
	for _, parseRow := range parseRecentUserRows {
		parseResponse.RecentUsers = append(parseResponse.RecentUsers, parseBuildAdminUserSummary(parseRow))
	}
	parseResponse.RecentConversations = make([]*chatpb.AdminConversationSummary, 0, len(parseRecentConversationRows))
	for _, parseRow := range parseRecentConversationRows {
		parseResponse.RecentConversations = append(parseResponse.RecentConversations, parseBuildAdminConversationSummary(parseRow))
	}
	parseFetchDuration := time.Since(parseFetchStart)
	parseAggregateCount := len(parseResponse.TopUsers) + len(parseResponse.RecentUsageEvents) + len(parseResponse.RecentUsers) + len(parseResponse.RecentConversations)
	parseLogAdminFetchOutcome(parseLogger, "rpc.GetAdminDashboard", "summary", "workspace", parseFetchDuration, parseAggregateCount)

	parseLogger.Info(
		"rpc.GetAdminDashboard: complete workspace scope",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.Int("workspace_count", len(parseScope.workspaceIDs)),
		slog.Int("scoped_user_count", len(parseScope.userIDs)),
		slog.Int("recent_usage_count", len(parseResponse.RecentUsageEvents)),
		slog.Int("recent_user_count", len(parseResponse.RecentUsers)),
		slog.Int("recent_conversation_count", len(parseResponse.RecentConversations)),
		slog.Duration("duration", parseFetchDuration),
	)
	return parseResponse, nil
}

// parseBuildWorkspaceScopedAdminSummary derives one safe dashboard summary from scoped rows only.
func parseBuildWorkspaceScopedAdminSummary(parseUsageRows []parseAdminUsageEventRow, parseUserRows []parseAdminUserRow, parseConversationRows []parseAdminConversationRow) *chatpb.AdminDashboardSummary {
	parseSummary := &chatpb.AdminDashboardSummary{
		TotalUsers:             int64(len(parseUserRows)),
		TotalConversations:     int64(len(parseConversationRows)),
		WindowNewUsers:         int64(len(parseUserRows)),
		WindowNewConversations: int64(len(parseConversationRows)),
		WindowUsageEvents:      int64(len(parseUsageRows)),
	}
	parseActiveUsers := make(map[int64]struct{})
	parseActiveConversations := make(map[int64]struct{})
	parseActiveClients := make(map[string]struct{})
	for _, parseConversationRow := range parseConversationRows {
		parseSummary.TotalMessages += parseConversationRow.MessageCount
		parseSummary.WindowNewMessages += parseConversationRow.MessageCount
	}
	for _, parseUsageRow := range parseUsageRows {
		parseSummary.WindowTotalCostUsd += parseUsageRow.TotalCostUSD
		parseSummary.WindowPromptTokens += parseUsageRow.PromptTokens
		parseSummary.WindowCompletionTokens += parseUsageRow.CompletionTokens
		parseActiveUsers[parseUsageRow.UserID] = struct{}{}
		parseActiveConversations[parseUsageRow.ConversationID] = struct{}{}
		if parseClientID := parseUsageRow.ClientID; parseClientID != "" {
			parseActiveClients[parseClientID] = struct{}{}
		}
		switch strings.ToLower(parseUsageRow.Status) {
		case "completed":
			parseSummary.WindowCompletedEvents++
		case "failed":
			parseSummary.WindowFailedEvents++
		}
	}
	parseSummary.WindowActiveUsers = int64(len(parseActiveUsers))
	parseSummary.WindowActiveConversations = int64(len(parseActiveConversations))
	parseSummary.WindowActiveClients = int64(len(parseActiveClients))
	return parseSummary
}

// ListAdminUsers returns recent user rows for authenticated superusers.
func (parseS *chatServer) ListAdminUsers(parseCtx context.Context, parseReq *chatpb.ListAdminUsersRequest) (*chatpb.ListAdminUsersResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminUsers"))
	parseFetchStart := time.Now()
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.users")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLimit int32
	var parseListQuery *chatpb.AdminListQuery
	if parseReq != nil {
		parseLimit = parseReq.GetLimit()
		parseListQuery = parseReq.GetListQuery()
	}
	parseListQueryShape := parseBuildAdminListQueryShape(parseLimit, parseListQuery)
	parseLimit = parseListQueryShape.parseLimit
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.ListAdminUsers: slice fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("limit", int(parseLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"users",
		"Admin users slice viewed",
		"{}",
		0,
	)
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.ListAdminUsers: store unavailable",
			slog.String("next_action", "restore store availability before retrying users slice fetch"),
		)
		return &chatpb.ListAdminUsersResponse{}, nil
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope || parseListQueryShape.parseSearch != "" || parseListQueryShape.parseSortBy != "" || parseListQueryShape.parseOffset > 0 {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseRows, parseErr := parseS.store.parseListAdminUsers(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminUsers: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin users: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseRows = parseFilterAdminUserRowsByScope(parseRows, parseScope.userIDs)
	}
	parseRows = parseFilterAdminUserRowsBySearch(parseRows, parseListQueryShape.parseSearch)
	parseSortAdminUserRows(parseRows, parseListQueryShape.parseSortBy, parseListQueryShape.isParseSortAscending)
	parseRows = parseApplyAdminSliceWindow(parseRows, parseListQueryShape.parseOffset, parseListQueryShape.parseLimit)
	parseUsers := make([]*chatpb.AdminUserSummary, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseUsers = append(parseUsers, parseBuildAdminUserSummary(parseRow))
	}
	parseFetchDuration := time.Since(parseFetchStart)
	parseLogAdminFetchOutcome(parseLogger, "rpc.ListAdminUsers", "table", parseScopeType, parseFetchDuration, len(parseUsers))
	parseLogger.Info(
		"rpc.ListAdminUsers: complete",
		slog.String("scope", parseScopeType),
		slog.Int("limit", int(parseLimit)),
		slog.Int("count", len(parseUsers)),
		slog.Duration("duration", parseFetchDuration),
	)
	return &chatpb.ListAdminUsersResponse{Users: parseUsers}, nil
}

// ListAdminUsageEvents returns recent global usage rows for authenticated superusers.
func (parseS *chatServer) ListAdminUsageEvents(parseCtx context.Context, parseReq *chatpb.ListAdminUsageEventsRequest) (*chatpb.ListAdminUsageEventsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminUsageEvents"))
	parseFetchStart := time.Now()
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.usage")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLookbackDays int32
	var parseLimit int32
	var parseStatusFilter string
	var parseProviderFilter string
	var parseListQuery *chatpb.AdminListQuery
	if parseReq != nil {
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
		parseStatusFilter = strings.TrimSpace(parseReq.GetStatus())
		parseProviderFilter = strings.TrimSpace(parseReq.GetProviderId())
		parseListQuery = parseReq.GetListQuery()
	}
	parseLookbackDays = parseClampAdminLookbackDays(parseLookbackDays)
	parseListQueryShape := parseBuildAdminListQueryShape(parseLimit, parseListQuery)
	parseLimit = parseListQueryShape.parseLimit
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.ListAdminUsageEvents: slice fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"usage",
		"Admin usage slice viewed",
		"{}",
		0,
	)
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.ListAdminUsageEvents: store unavailable",
			slog.String("next_action", "restore store availability before retrying usage slice fetch"),
		)
		return &chatpb.ListAdminUsageEventsResponse{}, nil
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope || parseListQueryShape.parseSearch != "" || parseListQueryShape.parseSortBy != "" || parseListQueryShape.parseOffset > 0 || parseStatusFilter != "" || parseProviderFilter != "" {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseRows, parseErr := parseS.store.parseListAdminUsageEvents(parseBuildAdminSinceTimestamp(parseLookbackDays), parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminUsageEvents: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin usage events: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseRows = parseFilterAdminUsageRowsByScope(parseRows, parseScope.userIDs)
	}
	parseRows = parseFilterAdminUsageRowsByListQuery(parseRows, parseListQueryShape.parseSearch, parseStatusFilter, parseProviderFilter)
	parseSortAdminUsageRows(parseRows, parseListQueryShape.parseSortBy, parseListQueryShape.isParseSortAscending)
	parseRows = parseApplyAdminSliceWindow(parseRows, parseListQueryShape.parseOffset, parseListQueryShape.parseLimit)
	parseEvents := make([]*chatpb.AdminUsageEvent, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseEvents = append(parseEvents, parseRedactAdminUsageEventByScope(parseScope, parseBuildAdminUsageEvent(parseRow)))
	}
	parseFetchDuration := time.Since(parseFetchStart)
	parseLogAdminFetchOutcome(parseLogger, "rpc.ListAdminUsageEvents", "table", parseScopeType, parseFetchDuration, len(parseEvents))
	parseLogger.Info(
		"rpc.ListAdminUsageEvents: complete",
		slog.String("scope", parseScopeType),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
		slog.Int("count", len(parseEvents)),
		slog.Duration("duration", parseFetchDuration),
	)
	return &chatpb.ListAdminUsageEventsResponse{Events: parseEvents}, nil
}

// ListAdminConversations returns recent conversation rows for authenticated superusers.
func (parseS *chatServer) ListAdminConversations(parseCtx context.Context, parseReq *chatpb.ListAdminConversationsRequest) (*chatpb.ListAdminConversationsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminConversations"))
	parseFetchStart := time.Now()
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.conversations")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLimit int32
	var parseListQuery *chatpb.AdminListQuery
	if parseReq != nil {
		parseLimit = parseReq.GetLimit()
		parseListQuery = parseReq.GetListQuery()
	}
	parseListQueryShape := parseBuildAdminListQueryShape(parseLimit, parseListQuery)
	parseLimit = parseListQueryShape.parseLimit
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.ListAdminConversations: slice fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("limit", int(parseLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"conversations",
		"Admin conversations slice viewed",
		"{}",
		0,
	)
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.ListAdminConversations: store unavailable",
			slog.String("next_action", "restore store availability before retrying conversations slice fetch"),
		)
		return &chatpb.ListAdminConversationsResponse{}, nil
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope || parseListQueryShape.parseSearch != "" || parseListQueryShape.parseSortBy != "" || parseListQueryShape.parseOffset > 0 {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseRows, parseErr := parseS.store.parseListAdminConversations(parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminConversations: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin conversations: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseRows = parseFilterAdminConversationRowsByScope(parseRows, parseScope.userIDs)
	}
	parseRows = parseFilterAdminConversationRowsBySearch(parseRows, parseListQueryShape.parseSearch)
	parseSortAdminConversationRows(parseRows, parseListQueryShape.parseSortBy, parseListQueryShape.isParseSortAscending)
	parseRows = parseApplyAdminSliceWindow(parseRows, parseListQueryShape.parseOffset, parseListQueryShape.parseLimit)
	parseConversations := make([]*chatpb.AdminConversationSummary, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseConversations = append(parseConversations, parseRedactAdminConversationSummaryByScope(parseScope, parseBuildAdminConversationSummary(parseRow)))
	}
	parseFetchDuration := time.Since(parseFetchStart)
	parseLogAdminFetchOutcome(parseLogger, "rpc.ListAdminConversations", "table", parseScopeType, parseFetchDuration, len(parseConversations))
	parseLogger.Info(
		"rpc.ListAdminConversations: complete",
		slog.String("scope", parseScopeType),
		slog.Int("limit", int(parseLimit)),
		slog.Int("count", len(parseConversations)),
		slog.Duration("duration", parseFetchDuration),
	)
	return &chatpb.ListAdminConversationsResponse{Conversations: parseConversations}, nil
}

// SearchAdminUsers returns typed user rows filtered by one query for authenticated admin callers.
func (parseS *chatServer) SearchAdminUsers(parseCtx context.Context, parseReq *chatpb.SearchAdminUsersRequest) (*chatpb.SearchAdminUsersResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "SearchAdminUsers"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.users.search")
	if parseErr != nil {
		return nil, parseErr
	}
	var parseLimit int32
	var parseQuery string
	if parseReq != nil {
		parseLimit = parseReq.GetLimit()
		parseQuery = parseReq.GetQuery()
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	parseQuery = strings.TrimSpace(parseQuery)
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.SearchAdminUsers: slice fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("limit", int(parseLimit)),
		slog.Int("query_len", len(parseQuery)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"users-search",
		"Admin user search viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.SearchAdminUsers: store unavailable",
			slog.String("next_action", "restore store availability before retrying admin user search"),
		)
		return &chatpb.SearchAdminUsersResponse{}, nil
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseRows, parseErr := parseS.store.parseSearchAdminUsers(parseQuery, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.SearchAdminUsers: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "search admin users: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseRows = parseFilterAdminUserRowsByScope(parseRows, parseScope.userIDs)
		parseRows = parseLimitAdminUserRows(parseRows, parseLimit)
	}
	parseUsers := make([]*chatpb.AdminUserSummary, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseUsers = append(parseUsers, parseBuildAdminUserSummary(parseRow))
	}
	parseLogger.Info(
		"rpc.SearchAdminUsers: complete",
		slog.String("scope", parseScopeType),
		slog.Int("limit", int(parseLimit)),
		slog.Int("count", len(parseUsers)),
	)
	return &chatpb.SearchAdminUsersResponse{Users: parseUsers}, nil
}

// GetAdminUserDetail returns one typed customer detail payload with user, session, usage, audit, workspace, memory, support, and billing slices.
func (parseS *chatServer) GetAdminUserDetail(parseCtx context.Context, parseReq *chatpb.GetAdminUserDetailRequest) (*chatpb.GetAdminUserDetailResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminUserDetail"))
	parseFetchStart := time.Now()
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.user.detail")
	if parseErr != nil {
		return nil, parseErr
	}
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
	if !parseScope.isPlatformScope {
		if _, hasParseUser := parseScope.userIDs[parseUserID]; !hasParseUser {
			return nil, status.Error(codes.PermissionDenied, "target user outside workspace-admin scope")
		}
	}
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseLogger.Info(
		"rpc.GetAdminUserDetail: drill-down fetch",
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
		"Admin user detail drill-down viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetAdminUserDetail: store unavailable",
			slog.String("next_action", "restore store availability before retrying admin user detail"),
		)
		return &chatpb.GetAdminUserDetailResponse{}, nil
	}
	parseUserRow, hasParseUser, parseErr := parseS.store.parseGetAdminUserSummaryByUserID(parseUserID)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: user summary query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get admin user summary: %v", parseErr)
	}
	if !hasParseUser {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseSince := parseBuildAdminSinceTimestamp(parseLookbackDays)
	parseSessionRows, parseErr := parseS.store.parseListAdminAuthSessionsByUser(parseUserID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: session query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user sessions: %v", parseErr)
	}
	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEventsByUser(parseUserID, parseSince, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user usage events: %v", parseErr)
	}
	parseAuditRows, parseErr := parseS.store.parseListAdminAuditLogsByUser(parseUserID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: audit query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user audit logs: %v", parseErr)
	}
	parseMembershipRows, parseErr := parseS.store.parseListAdminWorkspaceMembershipsByUser(parseUserID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: workspace membership query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user workspace memberships: %v", parseErr)
	}
	parseWorkspaceRows, parseErr := parseS.store.parseListAdminWorkspacesByUser(parseUserID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: workspace query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user workspaces: %v", parseErr)
	}
	parseMemoryRows, parseErr := parseS.store.parseListAdminUserMemoriesByUser(parseUserID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: memory query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user memories: %v", parseErr)
	}
	parseSupportTicketRows, parseErr := parseS.store.parseListAdminSupportTicketsByUser(parseUserID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: support ticket query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user support tickets: %v", parseErr)
	}
	parseSupportMessageRows, parseErr := parseS.store.parseListAdminSupportTicketMessagesByUser(parseUserID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: support message query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user support ticket messages: %v", parseErr)
	}
	parseSubscriptionRows, parseErr := parseS.store.parseListAdminBillingSubscriptionsByUser(parseUserID, parseQueryLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: billing subscription query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin user billing subscriptions: %v", parseErr)
	}
	parseTokenVersion, parseErr := parseS.store.parseGetAdminAuthTokenVersionByUser(parseUserID)
	if parseErr != nil {
		parseLogger.Error("rpc.GetAdminUserDetail: auth token-version query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "get admin user auth token version: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseScopedMembershipRows := make([]parseWorkspaceMembershipRow, 0, len(parseMembershipRows))
		for _, parseMembershipRow := range parseMembershipRows {
			if !parseHasAdminWorkspaceScope(parseScope.workspaceIDs, parseMembershipRow.WorkspaceID) {
				continue
			}
			parseScopedMembershipRows = append(parseScopedMembershipRows, parseMembershipRow)
		}
		parseMembershipRows = parseScopedMembershipRows

		parseScopedWorkspaceRows := make([]parseWorkspaceRow, 0, len(parseWorkspaceRows))
		for _, parseWorkspaceRow := range parseWorkspaceRows {
			if !parseHasAdminWorkspaceScope(parseScope.workspaceIDs, parseWorkspaceRow.ID) {
				continue
			}
			parseScopedWorkspaceRows = append(parseScopedWorkspaceRows, parseWorkspaceRow)
		}
		parseWorkspaceRows = parseScopedWorkspaceRows

		parseSupportTicketRows = parseFilterSupportTicketRowsByWorkspaceScope(parseSupportTicketRows, parseScope.workspaceIDs)
		parseScopedSupportTicketIDSet := make(map[int64]struct{}, len(parseSupportTicketRows))
		for _, parseSupportTicketRow := range parseSupportTicketRows {
			parseScopedSupportTicketIDSet[parseSupportTicketRow.ID] = struct{}{}
		}
		parseScopedSupportMessageRows := make([]parseSupportTicketMessageRow, 0, len(parseSupportMessageRows))
		for _, parseSupportMessageRow := range parseSupportMessageRows {
			if _, hasParseScopedTicket := parseScopedSupportTicketIDSet[parseSupportMessageRow.TicketID]; !hasParseScopedTicket {
				continue
			}
			parseScopedSupportMessageRows = append(parseScopedSupportMessageRows, parseSupportMessageRow)
		}
		parseSupportMessageRows = parseScopedSupportMessageRows

		parseAuditRows = parseFilterAdminAuditRowsByWorkspaceScope(parseAuditRows, parseScope.workspaceIDs)
		parseAuditRows = parseLimitAdminAuditRows(parseAuditRows, parseLimit)
	}
	parseDetail := &chatpb.AdminUserDetail{
		User:                  parseBuildAdminUserSummary(parseUserRow),
		RecentSessions:        make([]*chatpb.AuthSessionEntry, 0, len(parseSessionRows)),
		RecentUsageEvents:     make([]*chatpb.AdminUsageEvent, 0, len(parseUsageRows)),
		RecentAuditLogs:       make([]*chatpb.AuditLogEntry, 0, len(parseAuditRows)),
		Memberships:           make([]*chatpb.WorkspaceMembershipEntry, 0, len(parseMembershipRows)),
		Workspaces:            make([]*chatpb.WorkspaceEntry, 0, len(parseWorkspaceRows)),
		Memories:              make([]*chatpb.UserMemory, 0, len(parseMemoryRows)),
		SupportTickets:        make([]*chatpb.SupportTicketEntry, 0, len(parseSupportTicketRows)),
		SupportTicketMessages: make([]*chatpb.SupportTicketMessageEntry, 0, len(parseSupportMessageRows)),
		BillingSubscriptions:  make([]*chatpb.BillingSubscriptionEntry, 0, len(parseSubscriptionRows)),
		TokenVersion:          parseTokenVersion,
	}
	for _, parseSessionRow := range parseSessionRows {
		parseDetail.RecentSessions = append(parseDetail.RecentSessions, parseRedactAdminAuthSessionEntryByScope(parseScope, parseBuildAdminAuthSessionEntry(parseSessionRow)))
	}
	for _, parseUsageRow := range parseUsageRows {
		parseDetail.RecentUsageEvents = append(parseDetail.RecentUsageEvents, parseRedactAdminUsageEventByScope(parseScope, parseBuildAdminUsageEvent(parseUsageRow)))
	}
	for _, parseAuditRow := range parseAuditRows {
		parseDetail.RecentAuditLogs = append(parseDetail.RecentAuditLogs, parseRedactAdminAuditLogEntryByScope(parseScope, parseBuildAdminAuditLogEntry(parseAuditRow)))
	}
	for _, parseMembershipRow := range parseMembershipRows {
		parseDetail.Memberships = append(parseDetail.Memberships, parseBuildAdminWorkspaceMembershipEntry(parseMembershipRow))
	}
	for _, parseWorkspaceRow := range parseWorkspaceRows {
		parseDetail.Workspaces = append(parseDetail.Workspaces, parseBuildAdminWorkspaceEntry(parseWorkspaceRow))
	}
	for _, parseMemoryRow := range parseMemoryRows {
		parseDetail.Memories = append(parseDetail.Memories, parseBuildAdminUserMemoryEntry(parseMemoryRow))
	}
	for _, parseSupportTicketRow := range parseSupportTicketRows {
		parseDetail.SupportTickets = append(parseDetail.SupportTickets, parseBuildAdminSupportTicketEntry(parseSupportTicketRow))
	}
	for _, parseSupportMessageRow := range parseSupportMessageRows {
		parseDetail.SupportTicketMessages = append(parseDetail.SupportTicketMessages, parseBuildAdminSupportTicketMessageEntry(parseSupportMessageRow))
	}
	for _, parseSubscriptionRow := range parseSubscriptionRows {
		parseDetail.BillingSubscriptions = append(parseDetail.BillingSubscriptions, parseBuildBillingSubscriptionEntry(parseSubscriptionRow))
	}
	parseFetchDuration := time.Since(parseFetchStart)
	parseAggregateCount := len(parseDetail.RecentSessions) +
		len(parseDetail.RecentUsageEvents) +
		len(parseDetail.RecentAuditLogs) +
		len(parseDetail.Memberships) +
		len(parseDetail.Workspaces) +
		len(parseDetail.Memories) +
		len(parseDetail.SupportTickets) +
		len(parseDetail.SupportTicketMessages) +
		len(parseDetail.BillingSubscriptions)
	parseLogAdminFetchOutcome(parseLogger, "rpc.GetAdminUserDetail", "drilldown", parseScopeType, parseFetchDuration, parseAggregateCount)
	parseLogger.Info(
		"rpc.GetAdminUserDetail: complete",
		slog.String("scope", parseScopeType),
		slog.Int64("target_user_id", parseUserID),
		slog.Int("sessions", len(parseDetail.RecentSessions)),
		slog.Int("usage_events", len(parseDetail.RecentUsageEvents)),
		slog.Int("audit_logs", len(parseDetail.RecentAuditLogs)),
		slog.Int("memberships", len(parseDetail.Memberships)),
		slog.Int("workspaces", len(parseDetail.Workspaces)),
		slog.Int("memories", len(parseDetail.Memories)),
		slog.Int("support_tickets", len(parseDetail.SupportTickets)),
		slog.Int("support_messages", len(parseDetail.SupportTicketMessages)),
		slog.Int("billing_subscriptions", len(parseDetail.BillingSubscriptions)),
		slog.Int64("token_version", parseDetail.TokenVersion),
		slog.Duration("duration", parseFetchDuration),
	)
	return &chatpb.GetAdminUserDetailResponse{Detail: parseDetail}, nil
}

// parseBuildAdminUserMutationResponse builds one typed mutation response for one target user.
func (parseS *chatServer) parseBuildAdminUserMutationResponse(parseUserID int64, parseFallbackStatus string) (*chatpb.AdminUserMutationResponse, error) {
	parseStatusValue := strings.TrimSpace(parseFallbackStatus)
	if parseS == nil || parseS.store == nil || parseUserID <= 0 {
		if parseStatusValue == "" {
			parseStatusValue = "active"
		}
		return &chatpb.AdminUserMutationResponse{
			UserId: parseUserID,
			Status: parseStatusValue,
		}, nil
	}
	parseState, hasParseState, parseErr := parseS.store.parseGetUserAccessState(parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	if hasParseState && strings.TrimSpace(parseState.Status) != "" {
		parseStatusValue = strings.TrimSpace(parseState.Status)
	}
	if parseStatusValue == "" {
		parseStatusValue = "active"
	}
	return &chatpb.AdminUserMutationResponse{
		UserId: parseUserID,
		Status: parseStatusValue,
	}, nil
}

// DisableAdminUser applies one typed admin user-disable mutation.
func (parseS *chatServer) DisableAdminUser(parseCtx context.Context, parseReq *chatpb.AdminUserMutationRequest) (*chatpb.AdminUserMutationResponse, error) {
	var parseTarget parseAdminMutationTarget
	if parseReq != nil {
		parseTarget.userID = parseReq.GetUserId()
		parseTarget.isConfirmed = parseReq.GetConfirm()
		parseTarget.reason = parseReq.GetReason()
	}
	if parseTarget.userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	if parseErr := parseS.parseExecuteAdminMutationAction(parseCtx, parseAdminMutationDisableUser, parseTarget); parseErr != nil {
		return nil, parseErr
	}
	parseResp, parseErr := parseS.parseBuildAdminUserMutationResponse(parseTarget.userID, "disabled")
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "build disable user response: %v", parseErr)
	}
	return parseResp, nil
}

// RestoreAdminUser applies one typed admin user-restore mutation.
func (parseS *chatServer) RestoreAdminUser(parseCtx context.Context, parseReq *chatpb.AdminUserMutationRequest) (*chatpb.AdminUserMutationResponse, error) {
	var parseTarget parseAdminMutationTarget
	if parseReq != nil {
		parseTarget.userID = parseReq.GetUserId()
		parseTarget.isConfirmed = parseReq.GetConfirm()
		parseTarget.reason = parseReq.GetReason()
	}
	if parseTarget.userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}
	if parseErr := parseS.parseExecuteAdminMutationAction(parseCtx, parseAdminMutationRestoreUser, parseTarget); parseErr != nil {
		return nil, parseErr
	}
	parseResp, parseErr := parseS.parseBuildAdminUserMutationResponse(parseTarget.userID, "active")
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "build restore user response: %v", parseErr)
	}
	return parseResp, nil
}

// GetWorkspaceAdminSlices returns workspace-scoped control-plane slices for authenticated workspace admins.
func (parseS *chatServer) GetWorkspaceAdminSlices(parseCtx context.Context, parseReq *chatpb.GetWorkspaceAdminSlicesRequest) (*chatpb.GetWorkspaceAdminSlicesResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetWorkspaceAdminSlices"))
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.workspace_slices")
	if parseErr != nil {
		return nil, parseErr
	}
	if parseScope.isPlatformScope {
		return nil, status.Error(codes.PermissionDenied, "workspace admin scope required")
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
	parseLogger.Info(
		"rpc.GetWorkspaceAdminSlices: slice fetch",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.Int("workspace_count", len(parseScope.workspaceIDs)),
		slog.Int("scoped_user_count", len(parseScope.userIDs)),
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"workspace-admin",
		"Workspace admin slices viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"workspace",
		"workspace-admin",
		"Workspace admin drill-down access",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	parseResponse := &chatpb.GetWorkspaceAdminSlicesResponse{
		BillingSummary: &chatpb.WorkspaceAdminBillingSummary{},
	}
	if parseS.store == nil {
		parseLogger.Warn(
			"rpc.GetWorkspaceAdminSlices: store unavailable",
			slog.String("next_action", "restore store availability before retrying workspace slices"),
		)
		return parseResponse, nil
	}

	parseMembershipRows := make([]parseWorkspaceMembershipRow, 0, int(parseLimit))
	for parseWorkspaceID := range parseScope.workspaceIDs {
		parseRows, parseErr := parseS.store.parseListWorkspaceMembershipsByWorkspace(parseWorkspaceID)
		if parseErr != nil {
			parseLogger.Error("rpc.GetWorkspaceAdminSlices: workspace membership query failed", slog.String("error", parseErr.Error()))
			return nil, status.Errorf(codes.Internal, "list workspace memberships by workspace: %v", parseErr)
		}
		parseMembershipRows = append(parseMembershipRows, parseRows...)
	}
	parseResponse.Memberships = make([]*chatpb.WorkspaceMembershipEntry, 0, int(parseLimit))
	for _, parseRow := range parseMembershipRows {
		if len(parseResponse.Memberships) >= int(parseLimit) {
			break
		}
		parseResponse.Memberships = append(parseResponse.Memberships, &chatpb.WorkspaceMembershipEntry{
			Id:              parseRow.ID,
			WorkspaceId:     parseRow.WorkspaceID,
			UserId:          parseRow.UserID,
			RoleKey:         parseRow.RoleKey,
			Status:          parseRow.Status,
			InvitedByUserId: parseRow.InvitedByUserID,
			CreatedAt:       parseRow.CreatedAt,
			UpdatedAt:       parseRow.UpdatedAt,
		})
	}

	parseAPIKeyRows, parseErr := parseS.store.parseListAPIKeys(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetWorkspaceAdminSlices: api key query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list scoped api keys: %v", parseErr)
	}
	parseResponse.ApiKeys = make([]*chatpb.APIKeyEntry, 0, int(parseLimit))
	for _, parseRow := range parseAPIKeyRows {
		if !parseHasAdminWorkspaceScope(parseScope.workspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseResponse.ApiKeys = append(parseResponse.ApiKeys, &chatpb.APIKeyEntry{
			Id:          parseRow.ID,
			KeyId:       parseRow.KeyID,
			WorkspaceId: parseRow.WorkspaceID,
			UserId:      parseRow.UserID,
			Label:       parseRow.Label,
			KeyPrefix:   parseRow.KeyPrefix,
			ScopesJson:  parseRow.ScopesJSON,
			LastUsedAt:  parseRow.LastUsedAt,
			RevokedAt:   parseRow.RevokedAt,
			CreatedAt:   parseRow.CreatedAt,
		})
		if len(parseResponse.ApiKeys) >= int(parseLimit) {
			break
		}
	}

	parseWebhookRows, parseErr := parseS.store.parseListWebhookEndpoints(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetWorkspaceAdminSlices: webhook query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list scoped webhooks: %v", parseErr)
	}
	parseResponse.WebhookEndpoints = make([]*chatpb.WebhookEndpointEntry, 0, int(parseLimit))
	for _, parseRow := range parseWebhookRows {
		if !parseHasAdminWorkspaceScope(parseScope.workspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseResponse.WebhookEndpoints = append(parseResponse.WebhookEndpoints, &chatpb.WebhookEndpointEntry{
			Id:             parseRow.ID,
			WorkspaceId:    parseRow.WorkspaceID,
			Label:          parseRow.Label,
			TargetUrl:      parseRow.TargetURL,
			EventsJson:     parseRow.EventsJSON,
			IsEnabled:      parseRow.IsEnabled,
			LastDeliveryAt: parseRow.LastDeliveryAt,
			FailureCount:   parseRow.FailureCount,
			CreatedAt:      parseRow.CreatedAt,
			UpdatedAt:      parseRow.UpdatedAt,
		})
		if len(parseResponse.WebhookEndpoints) >= int(parseLimit) {
			break
		}
	}

	parseAuditRows, parseErr := parseS.store.parseListAuditLogs(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetWorkspaceAdminSlices: audit query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list scoped audit logs: %v", parseErr)
	}
	parseResponse.AuditLogs = make([]*chatpb.AuditLogEntry, 0, int(parseLimit))
	for _, parseRow := range parseAuditRows {
		if !parseHasAdminWorkspaceScope(parseScope.workspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseResponse.AuditLogs = append(parseResponse.AuditLogs, &chatpb.AuditLogEntry{
			Id:          parseRow.ID,
			ActorUserId: parseRow.ActorUserID,
			WorkspaceId: parseRow.WorkspaceID,
			EventType:   parseRow.EventType,
			TargetType:  parseRow.TargetType,
			TargetId:    parseRow.TargetID,
			Summary:     parseRow.Summary,
			PayloadJson: parseRow.PayloadJSON,
			CreatedAt:   parseRow.CreatedAt,
		})
		if len(parseResponse.AuditLogs) >= int(parseLimit) {
			break
		}
	}

	parseInvitationRows, parseErr := parseS.store.parseListWorkspaceInvitations(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetWorkspaceAdminSlices: invitation query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list scoped workspace invitations: %v", parseErr)
	}
	parseResponse.Invitations = make([]*chatpb.WorkspaceInvitationEntry, 0, int(parseLimit))
	for _, parseRow := range parseInvitationRows {
		if !parseHasAdminWorkspaceScope(parseScope.workspaceIDs, parseRow.WorkspaceID) {
			continue
		}
		parseResponse.Invitations = append(parseResponse.Invitations, &chatpb.WorkspaceInvitationEntry{
			Id:                  parseRow.ID,
			WorkspaceId:         parseRow.WorkspaceID,
			Email:               parseRow.Email,
			RoleKey:             parseRow.RoleKey,
			InvitationTokenHash: parseRow.InvitationTokenHash,
			InvitedByUserId:     parseRow.InvitedByUserID,
			Status:              parseRow.Status,
			ExpiresAt:           parseRow.ExpiresAt,
			AcceptedAt:          parseRow.AcceptedAt,
			CreatedAt:           parseRow.CreatedAt,
			UpdatedAt:           parseRow.UpdatedAt,
		})
		if len(parseResponse.Invitations) >= int(parseLimit) {
			break
		}
	}

	parseUsageRows, parseErr := parseS.store.parseListAdminUsageEvents(parseSince, parseAdminScopedScanLimit)
	if parseErr != nil {
		parseLogger.Error("rpc.GetWorkspaceAdminSlices: usage query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list scoped usage events: %v", parseErr)
	}
	parseScopedUsageRows := parseFilterAdminUsageRowsByScope(parseUsageRows, parseScope.userIDs)
	parseResponse.UsageEvents = make([]*chatpb.AdminUsageEvent, 0, int(parseLimit))
	for _, parseRow := range parseScopedUsageRows {
		parseResponse.UsageEvents = append(parseResponse.UsageEvents, parseBuildAdminUsageEvent(parseRow))
		if len(parseResponse.UsageEvents) >= int(parseLimit) {
			break
		}
	}

	parseBillingSummary, parseErr := parseS.parseBuildWorkspaceAdminBillingSummary(parseScope, parseScopedUsageRows, parseSince)
	if parseErr != nil {
		parseLogger.Error("rpc.GetWorkspaceAdminSlices: billing summary failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "build scoped billing summary: %v", parseErr)
	}
	parseResponse.BillingSummary = parseBillingSummary

	parseLogger.Info(
		"rpc.GetWorkspaceAdminSlices: complete",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.Int("workspace_count", len(parseScope.workspaceIDs)),
		slog.Int("memberships", len(parseResponse.Memberships)),
		slog.Int("api_keys", len(parseResponse.ApiKeys)),
		slog.Int("webhooks", len(parseResponse.WebhookEndpoints)),
		slog.Int("audit_logs", len(parseResponse.AuditLogs)),
		slog.Int("invitations", len(parseResponse.Invitations)),
		slog.Int("usage_events", len(parseResponse.UsageEvents)),
	)
	return parseResponse, nil
}

// parseBuildWorkspaceAdminBillingSummary derives one workspace-admin billing summary from scoped users and usage rows.
func (parseS *chatServer) parseBuildWorkspaceAdminBillingSummary(parseScope parseAdminAccessScope, parseUsageRows []parseAdminUsageEventRow, parseSince string) (*chatpb.WorkspaceAdminBillingSummary, error) {
	parseSummary := &chatpb.WorkspaceAdminBillingSummary{}
	if parseS == nil || parseS.store == nil {
		return parseSummary, nil
	}
	parseCustomerIDs := make(map[int64]struct{})
	for parseUserID := range parseScope.userIDs {
		parseCustomer, hasParseCustomer, parseErr := parseS.store.parseGetBillingCustomerByUser(parseUserID)
		if parseErr != nil {
			return nil, parseErr
		}
		if !hasParseCustomer || parseCustomer.ID <= 0 {
			continue
		}
		parseCustomerIDs[parseCustomer.ID] = struct{}{}
	}
	parseSummary.CustomerCount = int64(len(parseCustomerIDs))
	for parseCustomerID := range parseCustomerIDs {
		parseSubscriptions, parseErr := parseS.store.parseListBillingSubscriptionsByCustomer(parseCustomerID, parseAdminScopedScanLimit)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, parseSubscription := range parseSubscriptions {
			if parseHasBillingSubscriptionActive(parseSubscription.Status) {
				parseSummary.ActiveSubscriptionCount++
			}
		}

		parseInvoices, parseErr := parseS.store.parseListBillingInvoicesByCustomer(parseCustomerID, parseAdminScopedScanLimit)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, parseInvoice := range parseInvoices {
			if parseHasBillingInvoiceOpen(parseInvoice.Status) {
				parseSummary.OpenInvoiceCount++
			}
		}

		parseBillingEvents, parseErr := parseS.store.parseListBillingEventsByCustomer(parseCustomerID, parseAdminScopedScanLimit)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, parseBillingEvent := range parseBillingEvents {
			if !parseHasAdminTimestampAtOrAfter(parseBillingEvent.CreatedAt, parseSince) {
				continue
			}
			if parseHasBillingDunningEvent(parseBillingEvent.EventType) {
				parseSummary.DunningEventCount++
			}
		}
	}
	for _, parseUsageRow := range parseUsageRows {
		parseSummary.RecentUsageCostUsd += parseUsageRow.TotalCostUSD
	}
	return parseSummary, nil
}

// parseHasAdminWorkspaceScope reports whether one workspace id belongs to the caller admin scope.
func parseHasAdminWorkspaceScope(parseWorkspaceIDs map[int64]struct{}, parseWorkspaceID int64) bool {
	if parseWorkspaceID <= 0 {
		return false
	}
	_, hasParseWorkspace := parseWorkspaceIDs[parseWorkspaceID]
	return hasParseWorkspace
}

// parseHasBillingSubscriptionActive reports whether one billing subscription status is currently active.
func parseHasBillingSubscriptionActive(parseStatus string) bool {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "", "active", "trialing", "past_due":
		return true
	default:
		return false
	}
}

// parseHasBillingInvoiceOpen reports whether one billing invoice status should be treated as still open.
func parseHasBillingInvoiceOpen(parseStatus string) bool {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "paid", "void", "uncollectible":
		return false
	default:
		return true
	}
}

// parseHasBillingDunningEvent reports whether one billing event type should count toward dunning pressure.
func parseHasBillingDunningEvent(parseEventType string) bool {
	parseEventType = strings.TrimSpace(strings.ToLower(parseEventType))
	return strings.Contains(parseEventType, "dunning") || strings.Contains(parseEventType, "payment_failed") || strings.Contains(parseEventType, "payment.failed")
}

// parseHasAdminTimestampAtOrAfter reports whether one RFC3339 timestamp is at or after one RFC3339 cutoff.
func parseHasAdminTimestampAtOrAfter(parseTimestamp string, parseSince string) bool {
	parseTimestamp = strings.TrimSpace(parseTimestamp)
	parseSince = strings.TrimSpace(parseSince)
	if parseTimestamp == "" || parseSince == "" {
		return false
	}
	parseAtTime, parseErr := time.Parse(time.RFC3339, parseTimestamp)
	if parseErr != nil {
		return false
	}
	parseSinceTime, parseErr := time.Parse(time.RFC3339, parseSince)
	if parseErr != nil {
		return false
	}
	return !parseAtTime.Before(parseSinceTime)
}
