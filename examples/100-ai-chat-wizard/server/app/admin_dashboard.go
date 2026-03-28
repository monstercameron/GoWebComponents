package app

import (
	"context"
	"log/slog"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
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
		UserId:            parseRow.UserID,
		Email:             parseRow.Email,
		DisplayName:       parseRow.DisplayName,
		CreatedAt:         parseRow.CreatedAt,
		ConversationCount: parseRow.ConversationCount,
		MessageCount:      parseRow.MessageCount,
		UsageEventCount:   parseRow.UsageEventCount,
		TotalCostUsd:      parseRow.TotalCostUSD,
		LastSeenAt:        parseRow.LastSeenAt,
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

// GetAdminDashboard returns one global dashboard snapshot for authenticated superusers.
func (parseS *chatServer) GetAdminDashboard(parseCtx context.Context, parseReq *chatpb.GetAdminDashboardRequest) (*chatpb.GetAdminDashboardResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminDashboard"))
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseCtx); parseErr != nil {
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
	parseResponse := &chatpb.GetAdminDashboardResponse{
		ProviderSnapshots: parseBuildAdminProviderSnapshots(parseS.providerRegistry),
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.GetAdminDashboard: store unavailable, returning provider snapshots only")
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
		TotalUsers:                parseSummary.TotalUsers,
		TotalConversations:        parseSummary.TotalConversations,
		TotalMessages:             parseSummary.TotalMessages,
		WindowNewUsers:            parseSummary.WindowNewUsers,
		WindowNewConversations:    parseSummary.WindowNewConversations,
		WindowNewMessages:         parseSummary.WindowNewMessages,
		WindowUsageEvents:         parseSummary.WindowUsageEvents,
		WindowTotalCostUsd:        parseSummary.WindowTotalCostUSD,
		WindowPromptTokens:        parseSummary.WindowPromptTokens,
		WindowCompletionTokens:    parseSummary.WindowCompletionTokens,
		WindowActiveUsers:         parseSummary.WindowActiveUsers,
		WindowActiveConversations: parseSummary.WindowActiveConversations,
		WindowActiveClients:       parseSummary.WindowActiveClients,
		WindowCompletedEvents:     parseSummary.WindowCompletedEvents,
		WindowFailedEvents:        parseSummary.WindowFailedEvents,
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

	parseLogger.Info(
		"rpc.GetAdminDashboard: complete",
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("top_limit", int(parseTopLimit)),
		slog.Int("recent_limit", int(parseRecentLimit)),
		slog.Int("daily_rows", len(parseResponse.DailyUsage)),
		slog.Int("provider_rows", len(parseResponse.Providers)),
		slog.Int("model_rows", len(parseResponse.Models)),
	)
	return parseResponse, nil
}

// ListAdminUsers returns recent user rows for authenticated superusers.
func (parseS *chatServer) ListAdminUsers(parseCtx context.Context, parseReq *chatpb.ListAdminUsersRequest) (*chatpb.ListAdminUsersResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminUsers"))
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseCtx); parseErr != nil {
		return nil, parseErr
	}
	var parseLimit int32
	if parseReq != nil {
		parseLimit = parseReq.GetLimit()
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	if parseS.store == nil {
		parseLogger.Warn("rpc.ListAdminUsers: store unavailable")
		return &chatpb.ListAdminUsersResponse{}, nil
	}
	parseRows, parseErr := parseS.store.parseListAdminUsers(int64(parseLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminUsers: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin users: %v", parseErr)
	}
	parseUsers := make([]*chatpb.AdminUserSummary, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseUsers = append(parseUsers, parseBuildAdminUserSummary(parseRow))
	}
	parseLogger.Info("rpc.ListAdminUsers: complete", slog.Int("limit", int(parseLimit)), slog.Int("count", len(parseUsers)))
	return &chatpb.ListAdminUsersResponse{Users: parseUsers}, nil
}

// ListAdminUsageEvents returns recent global usage rows for authenticated superusers.
func (parseS *chatServer) ListAdminUsageEvents(parseCtx context.Context, parseReq *chatpb.ListAdminUsageEventsRequest) (*chatpb.ListAdminUsageEventsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminUsageEvents"))
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseCtx); parseErr != nil {
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
	if parseS.store == nil {
		parseLogger.Warn("rpc.ListAdminUsageEvents: store unavailable")
		return &chatpb.ListAdminUsageEventsResponse{}, nil
	}
	parseRows, parseErr := parseS.store.parseListAdminUsageEvents(parseBuildAdminSinceTimestamp(parseLookbackDays), int64(parseLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminUsageEvents: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin usage events: %v", parseErr)
	}
	parseEvents := make([]*chatpb.AdminUsageEvent, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseEvents = append(parseEvents, parseBuildAdminUsageEvent(parseRow))
	}
	parseLogger.Info(
		"rpc.ListAdminUsageEvents: complete",
		slog.Int("lookback_days", int(parseLookbackDays)),
		slog.Int("limit", int(parseLimit)),
		slog.Int("count", len(parseEvents)),
	)
	return &chatpb.ListAdminUsageEventsResponse{Events: parseEvents}, nil
}

// ListAdminConversations returns recent conversation rows for authenticated superusers.
func (parseS *chatServer) ListAdminConversations(parseCtx context.Context, parseReq *chatpb.ListAdminConversationsRequest) (*chatpb.ListAdminConversationsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminConversations"))
	if _, parseErr := parseS.parseRequireSuperuserUserID(parseCtx); parseErr != nil {
		return nil, parseErr
	}
	var parseLimit int32
	if parseReq != nil {
		parseLimit = parseReq.GetLimit()
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	if parseS.store == nil {
		parseLogger.Warn("rpc.ListAdminConversations: store unavailable")
		return &chatpb.ListAdminConversationsResponse{}, nil
	}
	parseRows, parseErr := parseS.store.parseListAdminConversations(int64(parseLimit))
	if parseErr != nil {
		parseLogger.Error("rpc.ListAdminConversations: query failed", slog.String("error", parseErr.Error()))
		return nil, status.Errorf(codes.Internal, "list admin conversations: %v", parseErr)
	}
	parseConversations := make([]*chatpb.AdminConversationSummary, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseConversations = append(parseConversations, parseBuildAdminConversationSummary(parseRow))
	}
	parseLogger.Info("rpc.ListAdminConversations: complete", slog.Int("limit", int(parseLimit)), slog.Int("count", len(parseConversations)))
	return &chatpb.ListAdminConversationsResponse{Conversations: parseConversations}, nil
}
