package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseAdminReadOnlyReportKeySummary = "dashboard_summary"
const parseAdminReadOnlyReportKeyUsers = "recent_users"
const parseAdminReadOnlyReportKeyUsage = "recent_usage_events"
const parseAdminReadOnlyReportKeyConversations = "recent_conversations"

// parseNormalizeAdminReadOnlyReportKey normalizes one read-only report key for allowlisted admin report dispatch.
func parseNormalizeAdminReadOnlyReportKey(parseReportKey string) string {
	return strings.TrimSpace(strings.ToLower(parseReportKey))
}

// parseBuildAdminReadOnlyReportScopeLabel maps one admin access scope into one stable scope label.
func parseBuildAdminReadOnlyReportScopeLabel(parseScope parseAdminAccessScope) string {
	if parseScope.isPlatformScope {
		return "platform"
	}
	return "workspace"
}

// parseResolveAdminReadOnlyReportSliceKey maps one allowlisted report key onto the dashboard slice used for scope gating.
func parseResolveAdminReadOnlyReportSliceKey(parseReportKey string) string {
	switch parseNormalizeAdminReadOnlyReportKey(parseReportKey) {
	case parseAdminReadOnlyReportKeyUsers:
		return "dashboard.users.read_only_report"
	case parseAdminReadOnlyReportKeyUsage:
		return "dashboard.usage.read_only_report"
	case parseAdminReadOnlyReportKeyConversations:
		return "dashboard.conversations.read_only_report"
	default:
		return "dashboard.home.read_only_report"
	}
}

// GetAdminReadOnlyReport returns one allowlisted read-only admin report slice without exposing raw SQL passthrough.
func (parseS *chatServer) GetAdminReadOnlyReport(parseCtx context.Context, parseReq *chatpb.GetAdminReadOnlyReportRequest) (*chatpb.GetAdminReadOnlyReportResponse, error) {
	parseReportKey := parseAdminReadOnlyReportKeySummary
	var parseLookbackDays int32
	var parseLimit int32
	if parseReq != nil {
		parseReportKey = parseNormalizeAdminReadOnlyReportKey(parseReq.GetReportKey())
		parseLookbackDays = parseReq.GetLookbackDays()
		parseLimit = parseReq.GetLimit()
	}
	if parseReportKey == "" {
		parseReportKey = parseAdminReadOnlyReportKeySummary
	}
	switch parseReportKey {
	case parseAdminReadOnlyReportKeySummary, parseAdminReadOnlyReportKeyUsers, parseAdminReadOnlyReportKeyUsage, parseAdminReadOnlyReportKeyConversations:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported report key: %s", parseReportKey)
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, parseResolveAdminReadOnlyReportSliceKey(parseReportKey))
	if parseErr != nil {
		return nil, parseErr
	}
	parseResponse := &chatpb.GetAdminReadOnlyReportResponse{
		ReportKey: parseReportKey,
		Scope:     parseBuildAdminReadOnlyReportScopeLabel(parseScope),
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.report.view",
		"report",
		parseReportKey,
		"Admin read-only report viewed",
		"{}",
		0,
	)
	switch parseReportKey {
	case parseAdminReadOnlyReportKeySummary:
		parseDashboardResponse, parseErr := parseS.GetAdminDashboard(parseCtx, &chatpb.GetAdminDashboardRequest{
			LookbackDays: parseLookbackDays,
			TopLimit:     1,
			RecentLimit:  1,
		})
		if parseErr != nil {
			return nil, parseErr
		}
		parseResponse.Summary = parseDashboardResponse.GetSummary()
		if parseResponse.Summary != nil {
			parseResponse.RowCount = 1
		}
	case parseAdminReadOnlyReportKeyUsers:
		parseLimit = parseClampAdminListLimit(parseLimit)
		parseUsersResponse, parseErr := parseS.ListAdminUsers(parseCtx, &chatpb.ListAdminUsersRequest{
			Limit: parseLimit,
			ListQuery: &chatpb.AdminListQuery{
				Limit: parseLimit,
			},
		})
		if parseErr != nil {
			return nil, parseErr
		}
		parseResponse.Users = parseUsersResponse.GetUsers()
		parseResponse.RowCount = int32(len(parseResponse.Users))
	case parseAdminReadOnlyReportKeyUsage:
		parseLimit = parseClampAdminListLimit(parseLimit)
		parseUsageResponse, parseErr := parseS.ListAdminUsageEvents(parseCtx, &chatpb.ListAdminUsageEventsRequest{
			LookbackDays: parseLookbackDays,
			Limit:        parseLimit,
			ListQuery: &chatpb.AdminListQuery{
				Limit: parseLimit,
			},
		})
		if parseErr != nil {
			return nil, parseErr
		}
		parseResponse.UsageEvents = parseUsageResponse.GetEvents()
		parseResponse.RowCount = int32(len(parseResponse.UsageEvents))
	case parseAdminReadOnlyReportKeyConversations:
		parseLimit = parseClampAdminListLimit(parseLimit)
		parseConversationsResponse, parseErr := parseS.ListAdminConversations(parseCtx, &chatpb.ListAdminConversationsRequest{
			Limit: parseLimit,
			ListQuery: &chatpb.AdminListQuery{
				Limit: parseLimit,
			},
		})
		if parseErr != nil {
			return nil, parseErr
		}
		parseResponse.Conversations = parseConversationsResponse.GetConversations()
		parseResponse.RowCount = int32(len(parseResponse.Conversations))
	}
	return parseResponse, nil
}
