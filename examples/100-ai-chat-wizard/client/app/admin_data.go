//go:build js && wasm

package app

import (
	"context"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
)

// adminDashboardData is the render-only snapshot of admin analytics data
// fetched from the server via GetAdminDashboard.
type adminDashboardData struct {
	IsLoading     bool
	IsDenied      bool
	Error         string
	HasData       bool
	Summary       adminSummarySnapshot
	TopUsers      []adminUserRow
	RecentUsers   []adminUserRow
	RecentConvs   []adminConvRow
	ProviderSnaps []adminProviderRow
	DailyUsage    []adminDailyRow
}

// adminSummarySnapshot holds the key KPI scalars from AdminDashboardSummary.
type adminSummarySnapshot struct {
	TotalUsers          int64
	TotalConversations  int64
	TotalMessages       int64
	WindowNewUsers      int64
	WindowActiveUsers   int64
	WindowNewConvs      int64
	WindowNewMessages   int64
	WindowUsageEvents   int64
	WindowTotalCostUSD  float64
	WindowFailedEvents  int64
	OpenIncidents       int64
	OpenSupportTickets  int64
	ActiveExperiments   int64
}

// adminUserRow holds a flattened user summary for list/table rendering.
type adminUserRow struct {
	UserID            int64
	Email             string
	DisplayName       string
	ConversationCount int64
	MessageCount      int64
	TotalCostUSD      float64
	LastSeenAt        string
	CreatedAt         string
}

// adminConvRow holds a flattened conversation summary for list rendering.
type adminConvRow struct {
	ConversationID int64
	PublicID       string
	UserID         int64
	Email          string
	DisplayName    string
	Preview        string
	MessageCount   int64
	TotalCostUSD   float64
	LastActivityAt string
	StartedAt      string
}

// adminProviderRow holds a flattened provider health snapshot.
type adminProviderRow struct {
	ProviderID      string
	Label           string
	IsAvailable     bool
	IsConfigured    bool
	Status          string
	LastError       string
	LastLatencyMs   int64
	RequestCount    int64
}

// adminDailyRow holds one day of usage for trend rendering.
type adminDailyRow struct {
	UsageDay      string
	EventCount    int64
	TotalCostUSD  float64
	ActiveUsers   int64
}

// parseUseAdminDashboard returns an adminDashboardData snapshot refreshed
// whenever the user gains admin access and the gRPC connection is ready.
func parseUseAdminDashboard(
	parseCurrentState appState,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) adminDashboardData {
	parseDataState := ui.UseState(adminDashboardData{IsLoading: false})
	parseRequestSeq := ui.UseRef(uint64(0))

	ui.UseEffect(func() func() {
		if !parseCurrentState.Authenticated || !parseCurrentState.GRPCReady || !parseCurrentState.CanAccessAdmin {
			parseDataState.Set(adminDashboardData{IsLoading: false})
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return nil
		}
		parseNextSeq := parseRequestSeq.Get() + 1
		parseRequestSeq.Set(parseNextSeq)
		parseDataState.Set(adminDashboardData{IsLoading: true})

		go func(parseSeq uint64) {
			parseResp, parseErr := parseClient.GetAdminDashboard(context.Background(), &chatpb.GetAdminDashboardRequest{
				LookbackDays: 30,
				TopLimit:     20,
				RecentLimit:  20,
			})
			if parseRequestSeq.Get() != parseSeq {
				return
			}
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				parseErrMsg := parseErr.Error()
				isDenied := strings.Contains(parseErrMsg, "permission denied") ||
					strings.Contains(parseErrMsg, "unauthenticated") ||
					strings.Contains(parseErrMsg, "PermissionDenied")
				chatLog.Warn("admin dashboard fetch failed", logging.Fields{"error": parseErrMsg, "denied": isDenied})
				parseDataState.Set(adminDashboardData{Error: parseErrMsg, IsDenied: isDenied})
				return
			}
			parseDataState.Set(parseMarshalAdminDashboardResp(parseResp))
		}(parseNextSeq)
		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.GRPCReady, parseCurrentState.CanAccessAdmin)

	return parseDataState.Get()
}

// parseMarshalAdminDashboardResp converts a proto response into the flat render snapshot.
func parseMarshalAdminDashboardResp(parseResp *chatpb.GetAdminDashboardResponse) adminDashboardData {
	if parseResp == nil {
		return adminDashboardData{Error: "empty response"}
	}
	parseData := adminDashboardData{HasData: true}
	if parseSummary := parseResp.GetSummary(); parseSummary != nil {
		parseData.Summary = adminSummarySnapshot{
			TotalUsers:         parseSummary.GetTotalUsers(),
			TotalConversations: parseSummary.GetTotalConversations(),
			TotalMessages:      parseSummary.GetTotalMessages(),
			WindowNewUsers:     parseSummary.GetWindowNewUsers(),
			WindowActiveUsers:  parseSummary.GetWindowActiveUsers(),
			WindowNewConvs:     parseSummary.GetWindowNewConversations(),
			WindowNewMessages:  parseSummary.GetWindowNewMessages(),
			WindowUsageEvents:  parseSummary.GetWindowUsageEvents(),
			WindowTotalCostUSD: parseSummary.GetWindowTotalCostUsd(),
			WindowFailedEvents: parseSummary.GetWindowFailedEvents(),
			OpenIncidents:      parseSummary.GetOpenIncidents(),
			OpenSupportTickets: parseSummary.GetOpenSupportTickets(),
			ActiveExperiments:  parseSummary.GetActiveExperiments(),
		}
	}
	for _, parseUser := range parseResp.GetTopUsers() {
		parseData.TopUsers = append(parseData.TopUsers, adminUserRow{
			UserID:            parseUser.GetUserId(),
			Email:             parseUser.GetEmail(),
			DisplayName:       parseUser.GetDisplayName(),
			ConversationCount: parseUser.GetConversationCount(),
			MessageCount:      parseUser.GetMessageCount(),
			TotalCostUSD:      parseUser.GetTotalCostUsd(),
			LastSeenAt:        parseUser.GetLastSeenAt(),
		})
	}
	for _, parseUser := range parseResp.GetRecentUsers() {
		parseData.RecentUsers = append(parseData.RecentUsers, adminUserRow{
			UserID:            parseUser.GetUserId(),
			Email:             parseUser.GetEmail(),
			DisplayName:       parseUser.GetDisplayName(),
			ConversationCount: parseUser.GetConversationCount(),
			MessageCount:      parseUser.GetMessageCount(),
			TotalCostUSD:      parseUser.GetTotalCostUsd(),
			LastSeenAt:        parseUser.GetLastSeenAt(),
			CreatedAt:         parseUser.GetCreatedAt(),
		})
	}
	for _, parseConv := range parseResp.GetRecentConversations() {
		parseData.RecentConvs = append(parseData.RecentConvs, adminConvRow{
			ConversationID: parseConv.GetConversationId(),
			PublicID:       parseConv.GetPublicId(),
			UserID:         parseConv.GetUserId(),
			Email:          parseConv.GetEmail(),
			DisplayName:    parseConv.GetDisplayName(),
			Preview:        parseConv.GetPreview(),
			MessageCount:   parseConv.GetMessageCount(),
			TotalCostUSD:   parseConv.GetTotalCostUsd(),
			LastActivityAt: parseConv.GetLastActivityAt(),
			StartedAt:      parseConv.GetStartedAt(),
		})
	}
	for _, parseSnap := range parseResp.GetProviderSnapshots() {
		parseData.ProviderSnaps = append(parseData.ProviderSnaps, adminProviderRow{
			ProviderID:    parseSnap.GetProviderId(),
			Label:         parseSnap.GetLabel(),
			IsAvailable:   parseSnap.GetAvailable(),
			IsConfigured:  parseSnap.GetAuthConfigured(),
			Status:        parseSnap.GetStatus(),
			LastError:     parseSnap.GetLastError(),
			LastLatencyMs: parseSnap.GetLastLatencyMs(),
			RequestCount:  parseSnap.GetRequestCount(),
		})
	}
	for _, parseDay := range parseResp.GetDailyUsage() {
		parseData.DailyUsage = append(parseData.DailyUsage, adminDailyRow{
			UsageDay:     parseDay.GetUsageDay(),
			EventCount:   parseDay.GetUsageEventCount(),
			TotalCostUSD: parseDay.GetTotalCostUsd(),
			ActiveUsers:  parseDay.GetActiveUsers(),
		})
	}
	return parseData
}
