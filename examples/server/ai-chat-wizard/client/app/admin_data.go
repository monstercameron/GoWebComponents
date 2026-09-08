//go:build js && wasm

package app

import (
	"context"
	"strconv"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v6/logging"
	"github.com/monstercameron/GoWebComponents/v6/ui"
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
	TotalUsers         int64
	TotalConversations int64
	TotalMessages      int64
	WindowNewUsers     int64
	WindowActiveUsers  int64
	WindowNewConvs     int64
	WindowNewMessages  int64
	WindowUsageEvents  int64
	WindowTotalCostUSD float64
	WindowFailedEvents int64
	OpenIncidents      int64
	OpenSupportTickets int64
	ActiveExperiments  int64
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
	ProviderID    string
	Label         string
	IsAvailable   bool
	IsConfigured  bool
	Status        string
	LastError     string
	LastLatencyMs int64
	RequestCount  int64
}

// adminDailyRow holds one day of usage for trend rendering.
type adminDailyRow struct {
	UsageDay     string
	EventCount   int64
	TotalCostUSD float64
	ActiveUsers  int64
}

// parseUseAdminDashboard returns an adminDashboardData snapshot refreshed
// whenever the user gains admin access and the gRPC connection is ready.
func parseUseAdminDashboard(
	parseCurrentState appState,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) adminDashboardData {
	_ = handleAuthFailure
	parseDataState := ui.UseState(adminDashboardData{IsLoading: false})
	parseRequestSeq := ui.UseRef(uint64(0))

	ui.UseEffect(func() func() {
		if !parseCurrentState.Authenticated || !parseCurrentState.CanAccessAdmin {
			parseDataState.Set(adminDashboardData{IsLoading: false})
			return nil
		}
		if !parseCurrentState.GRPCReady {
			parseDataState.Set(adminDashboardData{
				IsLoading: false,
				Error:     parseBuildUserErrorText(userErrorScopeDashboard, nil),
			})
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
			parseDashboardResp, parseDashboardErr := parseClient.GetAdminDashboard(context.Background(), &chatpb.GetAdminDashboardRequest{
				LookbackDays: 30,
				TopLimit:     20,
				RecentLimit:  20,
			})
			if parseRequestSeq.Get() != parseSeq {
				return
			}
			if parseDashboardErr != nil {
				parseErrMsg := parseDashboardErr.Error()
				isDenied := strings.Contains(parseErrMsg, "permission denied") ||
					strings.Contains(parseErrMsg, "unauthenticated") ||
					strings.Contains(parseErrMsg, "PermissionDenied")
				chatLog.Warn("admin dashboard fetch failed", logging.Fields{"error": parseErrMsg, "denied": isDenied})
				parseDataState.Set(adminDashboardData{Error: parseBuildUserErrorText(userErrorScopeDashboard, parseDashboardErr), IsDenied: isDenied})
				return
			}
			parseDataState.Set(parseMarshalAdminDashboardResp(parseDashboardResp))
		}(parseNextSeq)
		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.GRPCReady, parseCurrentState.CanAccessAdmin)

	return parseDataState.Get()
}

// parseMarshalAdminDashboardResp converts a proto response into the flat render snapshot.
func parseMarshalAdminDashboardResp(parseDashboardResp *chatpb.GetAdminDashboardResponse) adminDashboardData {
	if parseDashboardResp == nil {
		return adminDashboardData{Error: parseBuildUserErrorText(userErrorScopeDashboard, nil)}
	}
	parseDashboardData := adminDashboardData{HasData: true}
	if parseSummary := parseDashboardResp.GetSummary(); parseSummary != nil {
		parseDashboardData.Summary = adminSummarySnapshot{
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
	for _, parseTopUser := range parseDashboardResp.GetTopUsers() {
		parseDashboardData.TopUsers = append(parseDashboardData.TopUsers, adminUserRow{
			UserID:            parseTopUser.GetUserId(),
			Email:             parseTopUser.GetEmail(),
			DisplayName:       parseTopUser.GetDisplayName(),
			ConversationCount: parseTopUser.GetConversationCount(),
			MessageCount:      parseTopUser.GetMessageCount(),
			TotalCostUSD:      parseTopUser.GetTotalCostUsd(),
			LastSeenAt:        parseTopUser.GetLastSeenAt(),
		})
	}
	for _, parseRecentUser := range parseDashboardResp.GetRecentUsers() {
		parseDashboardData.RecentUsers = append(parseDashboardData.RecentUsers, adminUserRow{
			UserID:            parseRecentUser.GetUserId(),
			Email:             parseRecentUser.GetEmail(),
			DisplayName:       parseRecentUser.GetDisplayName(),
			ConversationCount: parseRecentUser.GetConversationCount(),
			MessageCount:      parseRecentUser.GetMessageCount(),
			TotalCostUSD:      parseRecentUser.GetTotalCostUsd(),
			LastSeenAt:        parseRecentUser.GetLastSeenAt(),
			CreatedAt:         parseRecentUser.GetCreatedAt(),
		})
	}
	for _, parseConversation := range parseDashboardResp.GetRecentConversations() {
		parseDashboardData.RecentConvs = append(parseDashboardData.RecentConvs, adminConvRow{
			ConversationID: parseConversation.GetConversationId(),
			PublicID:       parseConversation.GetPublicId(),
			UserID:         parseConversation.GetUserId(),
			Email:          parseConversation.GetEmail(),
			DisplayName:    parseConversation.GetDisplayName(),
			Preview:        parseConversation.GetPreview(),
			MessageCount:   parseConversation.GetMessageCount(),
			TotalCostUSD:   parseConversation.GetTotalCostUsd(),
			LastActivityAt: parseConversation.GetLastActivityAt(),
			StartedAt:      parseConversation.GetStartedAt(),
		})
	}
	for _, parseProviderSnapshot := range parseDashboardResp.GetProviderSnapshots() {
		parseDashboardData.ProviderSnaps = append(parseDashboardData.ProviderSnaps, adminProviderRow{
			ProviderID:    parseProviderSnapshot.GetProviderId(),
			Label:         parseProviderSnapshot.GetLabel(),
			IsAvailable:   parseProviderSnapshot.GetAvailable(),
			IsConfigured:  parseProviderSnapshot.GetAuthConfigured(),
			Status:        parseProviderSnapshot.GetStatus(),
			LastError:     parseProviderSnapshot.GetLastError(),
			LastLatencyMs: parseProviderSnapshot.GetLastLatencyMs(),
			RequestCount:  parseProviderSnapshot.GetRequestCount(),
		})
	}
	for _, parseUsageDay := range parseDashboardResp.GetDailyUsage() {
		parseDashboardData.DailyUsage = append(parseDashboardData.DailyUsage, adminDailyRow{
			UsageDay:     parseUsageDay.GetUsageDay(),
			EventCount:   parseUsageDay.GetUsageEventCount(),
			TotalCostUSD: parseUsageDay.GetTotalCostUsd(),
			ActiveUsers:  parseUsageDay.GetActiveUsers(),
		})
	}
	return parseDashboardData
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
	Query             adminListQueryState
	SearchResults     []adminUserRow
	IsSearching       bool
	CurrentPage       int
	SelectedUserID    int64
	UserAccessStates  map[int64]string
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
	HandleFilter        ui.Handler
	HandleSort          ui.Handler
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
	parseUserQueryState := ui.UsePersistedState[adminListQueryState]("chatwizard.admin.query.users", adminListQueryState{Filter: "all", Sort: "last_seen"}, ui.PersistLocal)
	parseSearchResults := ui.UseState([]adminUserRow(nil))
	parseIsSearching := ui.UseState(false)
	parseCurrentPage := ui.UseState(0)
	parseSelectedUserID := ui.UseState(int64(0))
	parseUserAccessStates := ui.UsePersistedState[map[int64]string]("chatwizard.admin.user.access_state", map[int64]string{}, ui.PersistLocal)
	parseUserDetail := ui.UseState(adminUserDetailSnapshot{})
	parseIsLoadingDetail := ui.UseState(false)
	parseConfirmAction := ui.UseState("")
	parseConfirmReason := ui.UseState("")
	parseIsMutationPending := ui.UseState(false)
	parseMutationError := ui.UseState("")
	parseMutationSuccess := ui.UseState("")

	parseSearchSeq := ui.UseRef(uint64(0))
	parseDetailSeq := ui.UseRef(uint64(0))
	parseUserQuery := parseNormalizeAdminListQuery(parseUserQueryState.Get())

	// Keep SearchResults in sync with the dashboard's RecentUsers when search is empty.
	ui.UseEffect(func() func() {
		if parseUserQuery.Search == "" {
			parseSearchResults.Set(parseAdminDashboard.RecentUsers)
		}
		return nil
	}, parseUserQuery.Search, parseAdminDashboard.HasData, len(parseAdminDashboard.RecentUsers))

	// Trigger server-side search when query is non-empty.
	ui.UseEffect(func() func() {
		parseQuery := parseUserQuery.Search
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
			parseSearchResp, parseSearchErr := parseClient.SearchAdminUsers(context.Background(), &chatpb.SearchAdminUsersRequest{
				Query: parseQ,
				Limit: 50,
			})
			if parseSearchSeq.Get() != parseSeq {
				return
			}
			if parseSearchErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseSearchErr) {
					return
				}
				parseIsSearching.Set(false)
				chatLog.Warn("admin user search failed", logging.Fields{"error": parseSearchErr.Error()})
				return
			}
			parseUserRows := make([]adminUserRow, 0, len(parseSearchResp.GetUsers()))
			for _, parseUser := range parseSearchResp.GetUsers() {
				parseUserRows = append(parseUserRows, adminUserRow{
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
			parseSearchResults.Set(parseUserRows)
			parseIsSearching.Set(false)
		}(parseNextSeq, parseQuery)
		return nil
	}, parseUserQuery.Search, parseCurrentState.Authenticated, parseCurrentState.GRPCReady)

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
			parseDetailResp, parseDetailErr := parseClient.GetAdminUserDetail(context.Background(), &chatpb.GetAdminUserDetailRequest{
				UserId:       parseUID,
				LookbackDays: 30,
				Limit:        10,
			})
			if parseDetailSeq.Get() != parseSeq {
				return
			}
			if parseDetailErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseDetailErr) {
					return
				}
				parseIsLoadingDetail.Set(false)
				chatLog.Warn("admin user detail fetch failed", logging.Fields{"user_id": parseUID, "error": parseDetailErr.Error()})
				return
			}
			parseUserDetail.Set(parseMarshalAdminUserDetail(parseDetailResp))
			parseIsLoadingDetail.Set(false)
		}(parseNextSeq, parseUserID)
		return nil
	}, parseSelectedUserID.Get(), parseCurrentState.Authenticated, parseCurrentState.GRPCReady)

	handleSearch := ui.UseEvent(func(parseE ui.Event) {
		parseNext := parseUserQueryState.Get()
		parseNext.Search = parseE.GetValue()
		parseNext.Page = 0
		parseUserQueryState.Set(parseNormalizeAdminListQuery(parseNext))
		parseCurrentPage.Set(0)
	})

	handleFilter := ui.UseEvent(func(parseE ui.Event) {
		parseNext := parseUserQueryState.Get()
		parseNext.Filter = parseEventValueOrDataset(parseE, dataAdminFilter)
		parseNext.Page = 0
		parseUserQueryState.Set(parseNormalizeAdminListQuery(parseNext))
		parseCurrentPage.Set(0)
	})

	handleSort := ui.UseEvent(func(parseE ui.Event) {
		parseNext := parseUserQueryState.Get()
		parseNext.Sort = parseEventValueOrDataset(parseE, dataAdminSort)
		parseNext.Page = 0
		parseUserQueryState.Set(parseNormalizeAdminListQuery(parseNext))
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
			var parseMutationResp *chatpb.AdminUserMutationResponse
			if parseAct == "disable" {
				parseMutationResp, parseErr = parseClient.DisableAdminUser(context.Background(), parseReq)
			} else {
				parseMutationResp, parseErr = parseClient.RestoreAdminUser(context.Background(), parseReq)
			}
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					parseIsMutationPending.Set(false)
					return
				}
				parseMutationError.Set(parseBuildUserErrorText(userErrorScopeDashboard, parseErr))
				parseIsMutationPending.Set(false)
				chatLog.Warn("admin user mutation failed", logging.Fields{"action": parseAct, "user_id": parseID, "error": parseErr.Error()})
				return
			}
			parseStatus := "done"
			if parseMutationResp != nil && parseMutationResp.GetStatus() != "" {
				parseStatus = parseMutationResp.GetStatus()
			}
			parseMutationSuccess.Set("User " + parseStatus + ".")
			parseNextAccessStates := cloneAdminUserAccessStates(parseUserAccessStates.Get())
			if parseAct == "disable" {
				parseNextAccessStates[parseID] = "disabled"
			} else {
				parseNextAccessStates[parseID] = "active"
			}
			parseUserAccessStates.Set(parseNextAccessStates)
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
				parseRefreshDetailResp, parseRefreshDetailErr := parseDetailClient.GetAdminUserDetail(context.Background(), &chatpb.GetAdminUserDetailRequest{
					UserId:       parseID,
					LookbackDays: 30,
					Limit:        10,
				})
				if parseDetailSeq.Get() != parseSeq {
					return
				}
				if parseRefreshDetailErr != nil {
					return
				}
				parseUserDetail.Set(parseMarshalAdminUserDetail(parseRefreshDetailResp))
			}(parseNextSeq)
		}(parseAction, parseUID, parseConfirmReason.Get())
	})

	return adminCustomersController{
		Data: adminCustomersData{
			Query:             parseUserQuery,
			SearchResults:     parseSearchResults.Get(),
			IsSearching:       parseIsSearching.Get(),
			CurrentPage:       parseCurrentPage.Get(),
			SelectedUserID:    parseSelectedUserID.Get(),
			UserAccessStates:  parseUserAccessStates.Get(),
			UserDetail:        parseUserDetail.Get(),
			IsLoadingDetail:   parseIsLoadingDetail.Get(),
			ConfirmAction:     parseConfirmAction.Get(),
			ConfirmReason:     parseConfirmReason.Get(),
			IsMutationPending: parseIsMutationPending.Get(),
			MutationError:     parseMutationError.Get(),
			MutationSuccess:   parseMutationSuccess.Get(),
		},
		HandleSearch:        handleSearch,
		HandleFilter:        handleFilter,
		HandleSort:          handleSort,
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
func parseMarshalAdminUserDetail(parseDetailResp *chatpb.GetAdminUserDetailResponse) adminUserDetailSnapshot {
	if parseDetailResp == nil {
		return adminUserDetailSnapshot{}
	}
	parseDetail := parseDetailResp.GetDetail()
	if parseDetail == nil {
		return adminUserDetailSnapshot{}
	}
	parseUser := parseDetail.GetUser()
	parseUserDetail := adminUserDetailSnapshot{
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
		parseUserDetail.Sessions = append(parseUserDetail.Sessions, adminSessionRow{
			UserAgent:  parseSession.GetUserAgent(),
			IPAddress:  parseSession.GetIpAddress(),
			LastSeenAt: parseSession.GetLastSeenAt(),
			ExpiresAt:  parseSession.GetExpiresAt(),
			RevokedAt:  parseSession.GetRevokedAt(),
		})
	}
	for _, parseUsageEv := range parseDetail.GetRecentUsageEvents() {
		parseUserDetail.UsageEvents = append(parseUserDetail.UsageEvents, adminUsageEventRow{
			ProviderID:   parseUsageEv.GetProviderId(),
			ModelID:      parseUsageEv.GetModelId(),
			TotalCostUSD: parseUsageEv.GetTotalCostUsd(),
			Status:       parseUsageEv.GetStatus(),
			CreatedAt:    parseUsageEv.GetCreatedAt(),
		})
	}
	for _, parseAudit := range parseDetail.GetRecentAuditLogs() {
		parseUserDetail.AuditLogs = append(parseUserDetail.AuditLogs, adminAuditRow{
			EventType: parseAudit.GetEventType(),
			Summary:   parseAudit.GetSummary(),
			CreatedAt: parseAudit.GetCreatedAt(),
		})
	}
	return parseUserDetail
}

func cloneAdminUserAccessStates(parseStates map[int64]string) map[int64]string {
	parseNext := make(map[int64]string, len(parseStates)+1)
	for parseID, parseState := range parseStates {
		parseNext[parseID] = parseState
	}
	return parseNext
}

// ─── Workspace admin types ────────────────────────────────────────────────────

// adminWorkspaceMemberRow holds a flattened workspace membership entry.
type adminWorkspaceMemberRow struct {
	UserID    int64
	RoleKey   string
	Status    string
	CreatedAt string
}

// adminWorkspaceAPIKeyRow holds a flattened API key entry.
type adminWorkspaceAPIKeyRow struct {
	KeyID     string
	Label     string
	KeyPrefix string
	RevokedAt string
	CreatedAt string
}

// adminWorkspaceWebhookRow holds a flattened webhook endpoint entry.
type adminWorkspaceWebhookRow struct {
	Label          string
	TargetURL      string
	IsEnabled      bool
	FailureCount   int64
	LastDeliveryAt string
	CreatedAt      string
}

// adminWorkspaceDetailSnapshot is the render-only snapshot for one selected workspace.
type adminWorkspaceDetailSnapshot struct {
	HasData     bool
	WorkspaceID int64
	Name        string
	Slug        string
	PlanCode    string
	Status      string
	OwnerUserID int64
	CreatedAt   string
	UpdatedAt   string
	Members     []adminWorkspaceMemberRow
	APIKeys     []adminWorkspaceAPIKeyRow
	Webhooks    []adminWorkspaceWebhookRow
	AuditLogs   []adminAuditRow
}

// adminWorkspacesData is the render-only snapshot of Workspaces slice UI state.
type adminWorkspacesData struct {
	LookupInput         string
	LookupError         string
	SelectedWorkspaceID int64
	WorkspaceDetail     adminWorkspaceDetailSnapshot
	IsLoadingDetail     bool
	ConfirmAction       string
	ConfirmReason       string
	IsMutationPending   bool
	MutationError       string
	MutationSuccess     string
}

// adminWorkspacesController bundles Workspaces slice render state and event handlers.
type adminWorkspacesController struct {
	Data                adminWorkspacesData
	HandleLookupInput   ui.Handler
	HandleLookupSubmit  ui.Handler
	HandleDismiss       ui.Handler
	HandleConfirmStart  ui.Handler
	HandleConfirmReason ui.Handler
	HandleConfirmSubmit ui.Handler
	HandleConfirmCancel ui.Handler
}

// parseUseAdminWorkspaces manages local state for the Workspaces admin slice.
func parseUseAdminWorkspaces(
	parseCurrentState appState,
	parseAdminDashboard adminDashboardData,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) adminWorkspacesController {
	parseLookupInput := ui.UseState("")
	parseLookupError := ui.UseState("")
	parseSelectedWSID := ui.UseState(int64(0))
	parseWorkspaceDetailState := ui.UseState(adminWorkspaceDetailSnapshot{})
	parseIsLoadingDetail := ui.UseState(false)
	parseConfirmAction := ui.UseState("")
	parseConfirmReason := ui.UseState("")
	parseIsMutationPending := ui.UseState(false)
	parseMutationError := ui.UseState("")
	parseMutationSuccess := ui.UseState("")

	parseDetailSeq := ui.UseRef(uint64(0))

	// Fetch workspace detail when selection changes.
	ui.UseEffect(func() func() {
		parseWID := parseSelectedWSID.Get()
		if parseWID <= 0 {
			parseWorkspaceDetailState.Set(adminWorkspaceDetailSnapshot{})
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
		go func(parseSeq uint64, parseWsID int64) {
			parseWorkspaceDetailResp, parseWorkspaceDetailErr := parseClient.GetAdminWorkspaceDetail(context.Background(), &chatpb.GetAdminWorkspaceDetailRequest{
				WorkspaceId:  parseWsID,
				LookbackDays: 30,
				Limit:        10,
			})
			if parseDetailSeq.Get() != parseSeq {
				return
			}
			if parseWorkspaceDetailErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseWorkspaceDetailErr) {
					return
				}
				parseIsLoadingDetail.Set(false)
				chatLog.Warn("admin workspace detail fetch failed", logging.Fields{"workspace_id": parseWsID, "error": parseWorkspaceDetailErr.Error()})
				return
			}
			parseWorkspaceDetailState.Set(parseMarshalAdminWorkspaceDetail(parseWorkspaceDetailResp))
			parseIsLoadingDetail.Set(false)
		}(parseNextSeq, parseWID)
		return nil
	}, parseSelectedWSID.Get(), parseCurrentState.Authenticated, parseCurrentState.GRPCReady)

	handleLookupInput := ui.UseEvent(func(parseE ui.Event) {
		parseLookupInput.Set(parseE.GetValue())
		parseLookupError.Set("")
	})

	handleLookupSubmit := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		parseRaw := strings.TrimSpace(parseLookupInput.Get())
		if parseRaw == "" {
			parseLookupError.Set("Enter a workspace ID.")
			return
		}
		parseWID, parseParseErr := strconv.ParseInt(parseRaw, 10, 64)
		if parseParseErr != nil || parseWID <= 0 {
			parseLookupError.Set("Workspace ID must be a positive integer.")
			return
		}
		parseLookupError.Set("")
		parseSelectedWSID.Set(parseWID)
		parseConfirmAction.Set("")
		parseConfirmReason.Set("")
		parseMutationError.Set("")
		parseMutationSuccess.Set("")
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
		parseWID := parseSelectedWSID.Get()
		if parseAction == "" || parseWID <= 0 {
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
			parseReq := &chatpb.AdminWorkspaceMutationRequest{
				WorkspaceId: parseID,
				Confirm:     true,
				Reason:      strings.TrimSpace(parseReason),
			}
			var parseErr error
			var parseWorkspaceStatus string
			if parseAct == "suspend" {
				parseWorkspaceResp, parseWorkspaceRespErr := parseClient.SuspendAdminWorkspace(context.Background(), parseReq)
				parseErr = parseWorkspaceRespErr
				if parseWorkspaceResp != nil {
					parseWorkspaceStatus = parseWorkspaceResp.GetStatus()
				}
			} else {
				parseReq.RestoreApiKeys = true
				parseReq.RestoreWebhookEndpoints = true
				parseReq.RestoreBackgroundJobs = true
				parseWorkspaceResp, parseWorkspaceRespErr := parseClient.RestoreAdminWorkspace(context.Background(), parseReq)
				parseErr = parseWorkspaceRespErr
				if parseWorkspaceResp != nil {
					parseWorkspaceStatus = parseWorkspaceResp.GetStatus()
				}
			}
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					parseIsMutationPending.Set(false)
					return
				}
				parseMutationError.Set(parseBuildUserErrorText(userErrorScopeDashboard, parseErr))
				parseIsMutationPending.Set(false)
				chatLog.Warn("admin workspace mutation failed", logging.Fields{"action": parseAct, "workspace_id": parseID, "error": parseErr.Error()})
				return
			}
			if parseWorkspaceStatus == "" {
				parseWorkspaceStatus = "done"
			}
			parseMutationSuccess.Set("Workspace " + parseWorkspaceStatus + ".")
			parseConfirmAction.Set("")
			parseConfirmReason.Set("")
			parseIsMutationPending.Set(false)
			// Re-fetch workspace detail after a successful mutation.
			parseDetailClient := parseChatClientRef.Get()
			if parseDetailClient == nil {
				return
			}
			parseNextSeq := parseDetailSeq.Get() + 1
			parseDetailSeq.Set(parseNextSeq)
			go func(parseSeq uint64) {
				parseRefreshWorkspaceDetailResp, parseRefreshWorkspaceDetailErr := parseDetailClient.GetAdminWorkspaceDetail(context.Background(), &chatpb.GetAdminWorkspaceDetailRequest{
					WorkspaceId:  parseID,
					LookbackDays: 30,
					Limit:        10,
				})
				if parseDetailSeq.Get() != parseSeq {
					return
				}
				if parseRefreshWorkspaceDetailErr != nil {
					return
				}
				parseWorkspaceDetailState.Set(parseMarshalAdminWorkspaceDetail(parseRefreshWorkspaceDetailResp))
			}(parseNextSeq)
		}(parseAction, parseWID, parseConfirmReason.Get())
	})

	_ = parseAdminDashboard

	return adminWorkspacesController{
		Data: adminWorkspacesData{
			LookupInput:         parseLookupInput.Get(),
			LookupError:         parseLookupError.Get(),
			SelectedWorkspaceID: parseSelectedWSID.Get(),
			WorkspaceDetail:     parseWorkspaceDetailState.Get(),
			IsLoadingDetail:     parseIsLoadingDetail.Get(),
			ConfirmAction:       parseConfirmAction.Get(),
			ConfirmReason:       parseConfirmReason.Get(),
			IsMutationPending:   parseIsMutationPending.Get(),
			MutationError:       parseMutationError.Get(),
			MutationSuccess:     parseMutationSuccess.Get(),
		},
		HandleLookupInput:   handleLookupInput,
		HandleLookupSubmit:  handleLookupSubmit,
		HandleDismiss:       handleConfirmCancel,
		HandleConfirmStart:  handleConfirmStart,
		HandleConfirmReason: handleConfirmReason,
		HandleConfirmSubmit: handleConfirmSubmit,
		HandleConfirmCancel: handleConfirmCancel,
	}
}

// parseMarshalAdminWorkspaceDetail converts a GetAdminWorkspaceDetailResponse into a flat snapshot.
func parseMarshalAdminWorkspaceDetail(parseWorkspaceDetailResp *chatpb.GetAdminWorkspaceDetailResponse) adminWorkspaceDetailSnapshot {
	if parseWorkspaceDetailResp == nil {
		return adminWorkspaceDetailSnapshot{}
	}
	parseDetail := parseWorkspaceDetailResp.GetDetail()
	if parseDetail == nil {
		return adminWorkspaceDetailSnapshot{}
	}
	parseWorkspace := parseDetail.GetWorkspace()
	parseWorkspaceDetail := adminWorkspaceDetailSnapshot{
		HasData:     true,
		WorkspaceID: parseWorkspace.GetId(),
		Name:        parseWorkspace.GetName(),
		Slug:        parseWorkspace.GetSlug(),
		PlanCode:    parseWorkspace.GetPlanCode(),
		Status:      parseWorkspace.GetStatus(),
		OwnerUserID: parseWorkspace.GetOwnerUserId(),
		CreatedAt:   parseWorkspace.GetCreatedAt(),
		UpdatedAt:   parseWorkspace.GetUpdatedAt(),
	}
	for _, parseMembership := range parseDetail.GetMemberships() {
		parseWorkspaceDetail.Members = append(parseWorkspaceDetail.Members, adminWorkspaceMemberRow{
			UserID:    parseMembership.GetUserId(),
			RoleKey:   parseMembership.GetRoleKey(),
			Status:    parseMembership.GetStatus(),
			CreatedAt: parseMembership.GetCreatedAt(),
		})
	}
	for _, parseAPIKey := range parseDetail.GetApiKeys() {
		parseWorkspaceDetail.APIKeys = append(parseWorkspaceDetail.APIKeys, adminWorkspaceAPIKeyRow{
			KeyID:     parseAPIKey.GetKeyId(),
			Label:     parseAPIKey.GetLabel(),
			KeyPrefix: parseAPIKey.GetKeyPrefix(),
			RevokedAt: parseAPIKey.GetRevokedAt(),
			CreatedAt: parseAPIKey.GetCreatedAt(),
		})
	}
	for _, parseWebhook := range parseDetail.GetWebhookEndpoints() {
		parseWorkspaceDetail.Webhooks = append(parseWorkspaceDetail.Webhooks, adminWorkspaceWebhookRow{
			Label:          parseWebhook.GetLabel(),
			TargetURL:      parseWebhook.GetTargetUrl(),
			IsEnabled:      parseWebhook.GetIsEnabled(),
			FailureCount:   parseWebhook.GetFailureCount(),
			LastDeliveryAt: parseWebhook.GetLastDeliveryAt(),
			CreatedAt:      parseWebhook.GetCreatedAt(),
		})
	}
	for _, parseAuditLog := range parseDetail.GetRecentAuditLogs() {
		parseWorkspaceDetail.AuditLogs = append(parseWorkspaceDetail.AuditLogs, adminAuditRow{
			EventType: parseAuditLog.GetEventType(),
			Summary:   parseAuditLog.GetSummary(),
			CreatedAt: parseAuditLog.GetCreatedAt(),
		})
	}
	return parseWorkspaceDetail
}
