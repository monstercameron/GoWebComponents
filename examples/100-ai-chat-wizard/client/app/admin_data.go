//go:build js && wasm

package app

import (
	"context"
	"strings"
	"time"

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

// ─── Customers slice types ────────────────────────────────────────────────────

// adminSessionRow holds a flattened auth session for table rendering.
type adminSessionRow struct {
	UserAgent  string
	IPAddress  string
	LastSeenAt string
	ExpiresAt  string
	RevokedAt  string
}

// adminUsageEventRow holds a flattened usage event for table rendering.
type adminUsageEventRow struct {
	ProviderID   string
	ModelID      string
	TotalCostUSD float64
	Status       string
	CreatedAt    string
}

// adminAuditRow holds a flattened audit log entry for table rendering.
type adminAuditRow struct {
	EventType string
	Summary   string
	CreatedAt string
}

// adminUserDetailSnapshot is the render-only snapshot for one selected user.
type adminUserDetailSnapshot struct {
	HasData      bool
	UserID       int64
	Email        string
	DisplayName  string
	CreatedAt    string
	LastSeenAt   string
	ConvCount    int64
	MessageCount int64
	TotalCostUSD float64
	Sessions     []adminSessionRow
	UsageEvents  []adminUsageEventRow
	AuditLogs    []adminAuditRow
}

// adminCustomersData is the render-only snapshot of Customers slice UI state.
type adminCustomersData struct {
	SearchQuery       string
	SearchResults     []adminUserRow
	IsSearching       bool
	CurrentPage       int
	SelectedUserID    int64
	UserDetail        adminUserDetailSnapshot
	IsLoadingDetail   bool
	ConfirmAction     string
	ConfirmReason     string
	IsMutationPending bool
	MutationError     string
	MutationSuccess   string
}

// adminCustomersController bundles Customers slice render state and event handlers.
type adminCustomersController struct {
	Data                adminCustomersData
	HandleSearch        ui.Handler
	HandleSelectUser    ui.Handler
	HandleNextPage      ui.Handler
	HandlePrevPage      ui.Handler
	HandleConfirmStart  ui.Handler
	HandleConfirmReason ui.Handler
	HandleConfirmSubmit ui.Handler
	HandleConfirmCancel ui.Handler
}

// parseUseAdminCustomers manages local state for the Customers admin slice.
func parseUseAdminCustomers(
	parseCurrentState appState,
	parseAdminDashboard adminDashboardData,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) adminCustomersController {
	parseSearchQuery := ui.UseState("")
	parseSearchResults := ui.UseState([]adminUserRow(nil))
	parseIsSearching := ui.UseState(false)
	parseCurrentPage := ui.UseState(0)
	parseSelectedUserID := ui.UseState(int64(0))
	parseUserDetail := ui.UseState(adminUserDetailSnapshot{})
	parseIsLoadingDetail := ui.UseState(false)
	parseConfirmAction := ui.UseState("")
	parseConfirmReason := ui.UseState("")
	parseIsMutationPending := ui.UseState(false)
	parseMutationError := ui.UseState("")
	parseMutationSuccess := ui.UseState("")

	parseSearchSeq := ui.UseRef(uint64(0))
	parseDetailSeq := ui.UseRef(uint64(0))

	// Keep SearchResults in sync with the dashboard's RecentUsers when search is empty.
	ui.UseEffect(func() func() {
		if parseSearchQuery.Get() == "" {
			parseSearchResults.Set(parseAdminDashboard.RecentUsers)
		}
		return nil
	}, parseSearchQuery.Get(), parseAdminDashboard.HasData, len(parseAdminDashboard.RecentUsers))

	// Trigger server-side search when query is non-empty.
	ui.UseEffect(func() func() {
		parseQuery := parseSearchQuery.Get()
		if parseQuery == "" {
			parseIsSearching.Set(false)
			return nil
		}
		if !parseCurrentState.Authenticated || !parseCurrentState.GRPCReady {
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return nil
		}
		parseNextSeq := parseSearchSeq.Get() + 1
		parseSearchSeq.Set(parseNextSeq)
		parseIsSearching.Set(true)
		parseCurrentPage.Set(0)
		go func(parseSeq uint64, parseQ string) {
			time.Sleep(300 * time.Millisecond)
			if parseSearchSeq.Get() != parseSeq {
				return
			}
			parseResp, parseErr := parseClient.SearchAdminUsers(context.Background(), &chatpb.SearchAdminUsersRequest{
				Query: parseQ,
				Limit: 50,
			})
			if parseSearchSeq.Get() != parseSeq {
				return
			}
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				parseIsSearching.Set(false)
				chatLog.Warn("admin user search failed", logging.Fields{"error": parseErr.Error()})
				return
			}
			parseRows := make([]adminUserRow, 0, len(parseResp.GetUsers()))
			for _, parseU := range parseResp.GetUsers() {
				parseRows = append(parseRows, adminUserRow{
					UserID:            parseU.GetUserId(),
					Email:             parseU.GetEmail(),
					DisplayName:       parseU.GetDisplayName(),
					ConversationCount: parseU.GetConversationCount(),
					MessageCount:      parseU.GetMessageCount(),
					TotalCostUSD:      parseU.GetTotalCostUsd(),
					LastSeenAt:        parseU.GetLastSeenAt(),
					CreatedAt:         parseU.GetCreatedAt(),
				})
			}
			parseSearchResults.Set(parseRows)
			parseIsSearching.Set(false)
		}(parseNextSeq, parseQuery)
		return nil
	}, parseSearchQuery.Get(), parseCurrentState.Authenticated, parseCurrentState.GRPCReady)

	// Fetch user detail when selection changes.
	ui.UseEffect(func() func() {
		parseUserID := parseSelectedUserID.Get()
		if parseUserID <= 0 {
			parseUserDetail.Set(adminUserDetailSnapshot{})
			parseIsLoadingDetail.Set(false)
			parseMutationError.Set("")
			parseMutationSuccess.Set("")
			return nil
		}
		if !parseCurrentState.Authenticated || !parseCurrentState.GRPCReady {
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return nil
		}
		parseNextSeq := parseDetailSeq.Get() + 1
		parseDetailSeq.Set(parseNextSeq)
		parseIsLoadingDetail.Set(true)
		parseMutationError.Set("")
		parseMutationSuccess.Set("")
		go func(parseSeq uint64, parseUID int64) {
			parseResp, parseErr := parseClient.GetAdminUserDetail(context.Background(), &chatpb.GetAdminUserDetailRequest{
				UserId:       parseUID,
				LookbackDays: 30,
				Limit:        10,
			})
			if parseDetailSeq.Get() != parseSeq {
				return
			}
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				parseIsLoadingDetail.Set(false)
				chatLog.Warn("admin user detail fetch failed", logging.Fields{"user_id": parseUID, "error": parseErr.Error()})
				return
			}
			parseUserDetail.Set(parseMarshalAdminUserDetail(parseResp))
			parseIsLoadingDetail.Set(false)
		}(parseNextSeq, parseUserID)
		return nil
	}, parseSelectedUserID.Get(), parseCurrentState.Authenticated, parseCurrentState.GRPCReady)

	handleSearch := ui.UseEvent(func(parseE ui.Event) {
		parseSearchQuery.Set(parseE.GetValue())
		parseCurrentPage.Set(0)
	})

	handleSelectUser := ui.UseEvent(func(parseE ui.Event) {
		parseUID, parseOk := parseEventDatasetInt64(parseE, dataAdminUserID)
		if !parseOk {
			return
		}
		if parseSelectedUserID.Get() == parseUID {
			parseSelectedUserID.Set(0)
			return
		}
		parseSelectedUserID.Set(parseUID)
		parseConfirmAction.Set("")
		parseConfirmReason.Set("")
		parseMutationError.Set("")
		parseMutationSuccess.Set("")
	})

	handleNextPage := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		parseCurrentPage.Set(parseCurrentPage.Get() + 1)
	})

	handlePrevPage := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		parsePrev := parseCurrentPage.Get() - 1
		if parsePrev < 0 {
			parsePrev = 0
		}
		parseCurrentPage.Set(parsePrev)
	})

	handleConfirmStart := ui.UseEvent(func(parseE ui.Event) {
		parseAction := parseEventDatasetValue(parseE, dataAdminAction)
		parseConfirmAction.Set(parseAction)
		parseConfirmReason.Set("")
		parseMutationError.Set("")
		parseMutationSuccess.Set("")
	})

	handleConfirmReason := ui.UseEvent(func(parseE ui.Event) {
		parseConfirmReason.Set(parseE.GetValue())
	})

	handleConfirmCancel := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		parseConfirmAction.Set("")
		parseConfirmReason.Set("")
		parseMutationError.Set("")
	})

	handleConfirmSubmit := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		if parseIsMutationPending.Get() {
			return
		}
		parseAction := parseConfirmAction.Get()
		parseUID := parseSelectedUserID.Get()
		if parseAction == "" || parseUID <= 0 {
			return
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			parseMutationError.Set("Connection not ready.")
			return
		}
		parseIsMutationPending.Set(true)
		parseMutationError.Set("")
		go func(parseAct string, parseID int64, parseReason string) {
			parseReq := &chatpb.AdminUserMutationRequest{
				UserId:  parseID,
				Confirm: true,
				Reason:  strings.TrimSpace(parseReason),
			}
			var parseErr error
			var parseResp *chatpb.AdminUserMutationResponse
			if parseAct == "disable" {
				parseResp, parseErr = parseClient.DisableAdminUser(context.Background(), parseReq)
			} else {
				parseResp, parseErr = parseClient.RestoreAdminUser(context.Background(), parseReq)
			}
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					parseIsMutationPending.Set(false)
					return
				}
				parseMutationError.Set(parseErr.Error())
				parseIsMutationPending.Set(false)
				chatLog.Warn("admin user mutation failed", logging.Fields{"action": parseAct, "user_id": parseID, "error": parseErr.Error()})
				return
			}
			parseStatus := "done"
			if parseResp != nil && parseResp.GetStatus() != "" {
				parseStatus = parseResp.GetStatus()
			}
			parseMutationSuccess.Set("User " + parseStatus + ".")
			parseConfirmAction.Set("")
			parseConfirmReason.Set("")
			parseIsMutationPending.Set(false)
			// Refresh the user detail after a successful mutation.
			parseDetailClient := parseChatClientRef.Get()
			if parseDetailClient == nil {
				return
			}
			parseNextSeq := parseDetailSeq.Get() + 1
			parseDetailSeq.Set(parseNextSeq)
			go func(parseSeq uint64) {
				parseDetailResp, parseDetailErr := parseDetailClient.GetAdminUserDetail(context.Background(), &chatpb.GetAdminUserDetailRequest{
					UserId:       parseID,
					LookbackDays: 30,
					Limit:        10,
				})
				if parseDetailSeq.Get() != parseSeq {
					return
				}
				if parseDetailErr != nil {
					return
				}
				parseUserDetail.Set(parseMarshalAdminUserDetail(parseDetailResp))
			}(parseNextSeq)
		}(parseAction, parseUID, parseConfirmReason.Get())
	})

	return adminCustomersController{
		Data: adminCustomersData{
			SearchQuery:       parseSearchQuery.Get(),
			SearchResults:     parseSearchResults.Get(),
			IsSearching:       parseIsSearching.Get(),
			CurrentPage:       parseCurrentPage.Get(),
			SelectedUserID:    parseSelectedUserID.Get(),
			UserDetail:        parseUserDetail.Get(),
			IsLoadingDetail:   parseIsLoadingDetail.Get(),
			ConfirmAction:     parseConfirmAction.Get(),
			ConfirmReason:     parseConfirmReason.Get(),
			IsMutationPending: parseIsMutationPending.Get(),
			MutationError:     parseMutationError.Get(),
			MutationSuccess:   parseMutationSuccess.Get(),
		},
		HandleSearch:        handleSearch,
		HandleSelectUser:    handleSelectUser,
		HandleNextPage:      handleNextPage,
		HandlePrevPage:      handlePrevPage,
		HandleConfirmStart:  handleConfirmStart,
		HandleConfirmReason: handleConfirmReason,
		HandleConfirmSubmit: handleConfirmSubmit,
		HandleConfirmCancel: handleConfirmCancel,
	}
}

// parseMarshalAdminUserDetail converts a GetAdminUserDetailResponse into a flat snapshot.
func parseMarshalAdminUserDetail(parseResp *chatpb.GetAdminUserDetailResponse) adminUserDetailSnapshot {
	if parseResp == nil {
		return adminUserDetailSnapshot{}
	}
	parseDetail := parseResp.GetDetail()
	if parseDetail == nil {
		return adminUserDetailSnapshot{}
	}
	parseUser := parseDetail.GetUser()
	parseSnap := adminUserDetailSnapshot{
		HasData:      true,
		UserID:       parseUser.GetUserId(),
		Email:        parseUser.GetEmail(),
		DisplayName:  parseUser.GetDisplayName(),
		CreatedAt:    parseUser.GetCreatedAt(),
		LastSeenAt:   parseUser.GetLastSeenAt(),
		ConvCount:    parseUser.GetConversationCount(),
		MessageCount: parseUser.GetMessageCount(),
		TotalCostUSD: parseUser.GetTotalCostUsd(),
	}
	for _, parseSession := range parseDetail.GetRecentSessions() {
		parseSnap.Sessions = append(parseSnap.Sessions, adminSessionRow{
			UserAgent:  parseSession.GetUserAgent(),
			IPAddress:  parseSession.GetIpAddress(),
			LastSeenAt: parseSession.GetLastSeenAt(),
			ExpiresAt:  parseSession.GetExpiresAt(),
			RevokedAt:  parseSession.GetRevokedAt(),
		})
	}
	for _, parseUsageEv := range parseDetail.GetRecentUsageEvents() {
		parseSnap.UsageEvents = append(parseSnap.UsageEvents, adminUsageEventRow{
			ProviderID:   parseUsageEv.GetProviderId(),
			ModelID:      parseUsageEv.GetModelId(),
			TotalCostUSD: parseUsageEv.GetTotalCostUsd(),
			Status:       parseUsageEv.GetStatus(),
			CreatedAt:    parseUsageEv.GetCreatedAt(),
		})
	}
	for _, parseAudit := range parseDetail.GetRecentAuditLogs() {
		parseSnap.AuditLogs = append(parseSnap.AuditLogs, adminAuditRow{
			EventType: parseAudit.GetEventType(),
			Summary:   parseAudit.GetSummary(),
			CreatedAt: parseAudit.GetCreatedAt(),
		})
	}
	return parseSnap
}
