//go:build js && wasm

package app

import (
	"context"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	adminScopeBilling    = "billing"
	adminScopeWorkspaces = "workspaces"
	adminScopeSupport    = "support"
	adminScopeIncidents  = "incidents"
	adminScopeProviders  = "providers"
	adminScopeChats      = "chats"
	adminScopeOps        = "ops"
)

type adminListQueryState struct {
	Search string
	Filter string
	Sort   string
	Page   int
}

type adminOperationsData struct {
	IsLoading           bool
	Error               string
	HasSuperuserData    bool
	HasWorkspaceData    bool
	WorkspaceQuery      adminListQueryState
	SupportQuery        adminListQueryState
	BillingQuery        adminListQueryState
	IncidentQuery       adminListQueryState
	Workspaces          []adminWorkspaceListRow
	SupportTickets      []adminSupportTicketRow
	BillingOverages     []adminBillingOverageRow
	BillingQuotas       []adminBillingQuotaRow
	BillingTriggers     []adminBillingTriggerRow
	FailedPayments      []adminBillingEventRow
	DunningEvents       []adminDunningEventRow
	BusinessTopAccounts []adminUserRow
	ChatFailedReplies   []adminUsageEventRow
	ChatSlowReplies     []adminUsageEventRow
	ChatHighCostThreads []adminConvRow
	ProviderFallbacks   []adminProviderFallbackRow
	FeatureFlags        []adminFeatureFlagRow
	Incidents           []adminIncidentRow
	Experiments         []adminExperimentRow
	CostGuardrails      []adminCostGuardrailRow
	WorkspaceMembers    []adminWorkspaceMemberRow
	WorkspaceAPIKeys    []adminWorkspaceAPIKeyRow
	WorkspaceWebhooks   []adminWorkspaceWebhookRow
	WorkspaceInvites    []adminWorkspaceInviteRow
	WorkspaceAuditLogs  []adminAuditRow
	WorkspaceUsage      []adminUsageEventRow
	WorkspaceBilling    adminWorkspaceBillingRow
	SupportDetail       adminSupportDetailSnapshot
	SelectedTicketID    int64
	IsLoadingDetail     bool
	ConfirmAction       string
	ConfirmTargetID     int64
	ConfirmTargetKey    string
	ConfirmScope        string
	ConfirmReason       string
	SupportNoteBody     string
	IsMutationPending   bool
	MutationError       string
	MutationSuccess     string
}

type adminOperationsController struct {
	Data                  adminOperationsData
	HandleQuerySearch     ui.Handler
	HandleQueryFilter     ui.Handler
	HandleQuerySort       ui.Handler
	HandleNextPage        ui.Handler
	HandlePrevPage        ui.Handler
	HandleSelectWorkspace ui.Handler
	HandleSelectTicket    ui.Handler
	HandleConfirmStart    ui.Handler
	HandleConfirmReason   ui.Handler
	HandleSupportNote     ui.Handler
	HandleConfirmSubmit   ui.Handler
	HandleConfirmCancel   ui.Handler
}

type adminWorkspaceListRow struct {
	WorkspaceID int64
	Name        string
	Slug        string
	PlanCode    string
	Status      string
	OwnerUserID int64
	UpdatedAt   string
}

type adminSupportTicketRow struct {
	TicketID       int64
	TicketKey      string
	WorkspaceID    int64
	UserID         int64
	Status         string
	Priority       string
	Subject        string
	AssigneeUserID int64
	UpdatedAt      string
}

type adminSupportMessageRow struct {
	AuthorUserID int64
	MessageType  string
	Body         string
	IsInternal   bool
	CreatedAt    string
}

type adminSupportDetailSnapshot struct {
	HasData        bool
	Ticket         adminSupportTicketRow
	Messages       []adminSupportMessageRow
	AccountActions []adminAuditRow
}

type adminBillingOverageRow struct {
	PlanCode          string
	MeterKey          string
	IncludedUnits     int64
	SoftLimitUnits    int64
	HardLimitUnits    int64
	OveragePriceCents int64
	UpdatedAt         string
}

type adminBillingQuotaRow struct {
	PlanCode        string
	QuotaKey        string
	SoftLimitValue  int64
	HardLimitValue  int64
	EnforcementMode string
	UpdatedAt       string
}

type adminBillingTriggerRow struct {
	PlanCode        string
	TriggerKey      string
	ThresholdPct    int64
	UpgradePlanCode string
	IsEnabled       bool
	UpdatedAt       string
}

type adminBillingEventRow struct {
	EventID      int64
	CustomerID   int64
	InvoiceID    int64
	EventType    string
	EventSummary string
	CreatedAt    string
}

type adminDunningEventRow struct {
	EventID       int64
	CustomerID    int64
	InvoiceID     int64
	Status        string
	AttemptCount  int64
	FailureReason string
	NextAttemptAt string
	ResolvedAt    string
}

type adminIncidentRow struct {
	IncidentID  int64
	IncidentKey string
	Severity    string
	Status      string
	Title       string
	Summary     string
	UpdatedAt   string
}

type adminFeatureFlagRow struct {
	FlagKey        string
	Description    string
	IsEnabled      bool
	RolloutPercent int64
	UpdatedAt      string
}

type adminProviderFallbackRow struct {
	WorkspaceID int64
	EventType   string
	Summary     string
	CreatedAt   string
}

type adminExperimentRow struct {
	ExperimentKey string
	Name          string
	Status        string
	UpdatedAt     string
}

type adminCostGuardrailRow struct {
	WorkspaceID            int64
	GuardrailKey           string
	DailyBudgetCents       int64
	MonthlyBudgetCents     int64
	MaxCostPerRequestCents int64
	ActionMode             string
	UpdatedAt              string
}

type adminWorkspaceInviteRow struct {
	Email     string
	RoleKey   string
	Status    string
	ExpiresAt string
	CreatedAt string
}

type adminWorkspaceBillingRow struct {
	CustomerCount           int64
	ActiveSubscriptionCount int64
	OpenInvoiceCount        int64
	DunningEventCount       int64
	RecentUsageCostUSD      float64
}

func parseUseAdminOperations(
	parseCurrentState appState,
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	handleAuthFailure func(error) bool,
) adminOperationsController {
	parseWorkspaceQueryState := ui.UsePersistedState[adminListQueryState]("chatwizard.admin.query.workspaces", adminListQueryState{Sort: "updated"}, ui.PersistLocal)
	parseSupportQueryState := ui.UsePersistedState[adminListQueryState]("chatwizard.admin.query.support", adminListQueryState{Filter: "open", Sort: "priority"}, ui.PersistLocal)
	parseBillingQueryState := ui.UsePersistedState[adminListQueryState]("chatwizard.admin.query.billing", adminListQueryState{Filter: "active", Sort: "plan"}, ui.PersistLocal)
	parseIncidentQueryState := ui.UsePersistedState[adminListQueryState]("chatwizard.admin.query.incidents", adminListQueryState{Filter: "open", Sort: "updated"}, ui.PersistLocal)
	parseDataState := ui.UseState(adminOperationsData{})
	parseSelectedTicketID := ui.UseState(int64(0))
	parseSupportDetail := ui.UseState(adminSupportDetailSnapshot{})
	parseIsLoadingDetail := ui.UseState(false)
	parseConfirmAction := ui.UseState("")
	parseConfirmTargetID := ui.UseState(int64(0))
	parseConfirmTargetKey := ui.UseState("")
	parseConfirmScope := ui.UseState("")
	parseConfirmReason := ui.UseState("")
	parseSupportNoteBody := ui.UseState("")
	parseIsMutationPending := ui.UseState(false)
	parseMutationError := ui.UseState("")
	parseMutationSuccess := ui.UseState("")
	parseRequestSeq := ui.UseRef(uint64(0))
	parseDetailSeq := ui.UseRef(uint64(0))

	parseWorkspaceQuery := parseNormalizeAdminListQuery(parseWorkspaceQueryState.Get())
	parseSupportQuery := parseNormalizeAdminListQuery(parseSupportQueryState.Get())
	parseBillingQuery := parseNormalizeAdminListQuery(parseBillingQueryState.Get())
	parseIncidentQuery := parseNormalizeAdminListQuery(parseIncidentQueryState.Get())

	ui.UseEffect(func() func() {
		if !parseCurrentState.Authenticated || !parseCurrentState.CanAccessAdmin {
			parseDataState.Set(adminOperationsData{})
			return nil
		}
		if !parseCurrentState.GRPCReady {
			parseDataState.Set(adminOperationsData{Error: parseBuildUserErrorText(userErrorScopeDashboard, nil)})
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return nil
		}
		parseNextSeq := parseRequestSeq.Get() + 1
		parseRequestSeq.Set(parseNextSeq)
		parseDataState.Set(adminOperationsData{IsLoading: true})
		go func(parseSeq uint64, isSuperuser bool, parseWQ, parseSQ, parseBQ, parseIQ adminListQueryState) {
			parseNext := adminOperationsData{IsLoading: false}
			if isSuperuser {
				parseSuperResp, parseSuperErr := parseClient.GetSuperuserSlices(context.Background(), &chatpb.GetSuperuserSlicesRequest{
					LookbackDays:       30,
					Limit:              40,
					WorkspaceListQuery: parseBuildAdminListQuery(parseWQ, 40),
					WorkspaceStatus:    parseStatusFilterValue(parseWQ.Filter),
					SupportListQuery:   parseBuildAdminListQuery(parseSQ, 40),
					SupportStatus:      parseStatusFilterValue(parseSQ.Filter),
					IncidentListQuery:  parseBuildAdminListQuery(parseIQ, 40),
					IncidentStatus:     parseStatusFilterValue(parseIQ.Filter),
				})
				if parseRequestSeq.Get() != parseSeq {
					return
				}
				if parseSuperErr != nil {
					if handleAuthFailure != nil && handleAuthFailure(parseSuperErr) {
						return
					}
					parseNext.Error = parseBuildUserErrorText(userErrorScopeDashboard, parseSuperErr)
					chatLog.Warn("admin operations superuser slices failed", logging.Fields{"error": parseSuperErr.Error()})
				} else {
					parseApplySuperuserSlices(&parseNext, parseSuperResp)
				}
				parseBusinessResp, parseBusinessErr := parseClient.GetAdminBusinessQueue(context.Background(), &chatpb.GetAdminBusinessQueueRequest{
					LookbackDays: 30,
					Limit:        20,
				})
				if parseBusinessErr != nil {
					chatLog.Warn("admin business queue failed", logging.Fields{"error": parseBusinessErr.Error()})
				} else {
					parseApplyBusinessQueue(&parseNext, parseBusinessResp)
				}
				parseChatResp, parseChatErr := parseClient.GetAdminChatsAnomalies(context.Background(), &chatpb.GetAdminChatsAnomaliesRequest{
					LookbackDays: 30,
					Limit:        20,
				})
				if parseChatErr != nil {
					chatLog.Warn("admin chats anomalies failed", logging.Fields{"error": parseChatErr.Error()})
				} else {
					parseApplyChatsAnomalies(&parseNext, parseChatResp)
				}
				parseProviderResp, parseProviderErr := parseClient.GetAdminProviderHealthTrends(context.Background(), &chatpb.GetAdminProviderHealthTrendsRequest{
					LookbackDays: 30,
					Limit:        20,
				})
				if parseProviderErr != nil {
					chatLog.Warn("admin provider health trends failed", logging.Fields{"error": parseProviderErr.Error()})
				} else {
					parseApplyProviderHealthTrends(&parseNext, parseProviderResp)
				}
				parseOpsResp, parseOpsErr := parseClient.GetAdminOpsDrilldown(context.Background(), &chatpb.GetAdminOpsDrilldownRequest{
					LookbackDays: 30,
					Limit:        20,
				})
				if parseOpsErr != nil {
					chatLog.Warn("admin ops drilldown failed", logging.Fields{"error": parseOpsErr.Error()})
				} else {
					parseApplyOpsDrilldown(&parseNext, parseOpsResp)
				}
			}
			parseWorkspaceResp, parseWorkspaceErr := parseClient.GetWorkspaceAdminSlices(context.Background(), &chatpb.GetWorkspaceAdminSlicesRequest{
				LookbackDays: 30,
				Limit:        40,
			})
			if parseRequestSeq.Get() != parseSeq {
				return
			}
			if parseWorkspaceErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseWorkspaceErr) {
					return
				}
				if parseNext.Error == "" {
					parseNext.Error = parseBuildUserErrorText(userErrorScopeDashboard, parseWorkspaceErr)
				}
				chatLog.Warn("workspace admin slices failed", logging.Fields{"error": parseWorkspaceErr.Error()})
			} else {
				parseApplyWorkspaceAdminSlices(&parseNext, parseWorkspaceResp)
			}
			parseNext.WorkspaceQuery = parseWQ
			parseNext.SupportQuery = parseSQ
			parseNext.BillingQuery = parseBQ
			parseNext.IncidentQuery = parseIQ
			parseDataState.Set(parseNext)
		}(parseNextSeq, parseCurrentState.IsSuperuser, parseWorkspaceQuery, parseSupportQuery, parseBillingQuery, parseIncidentQuery)
		return nil
	}, parseCurrentState.Authenticated, parseCurrentState.CanAccessAdmin, parseCurrentState.IsSuperuser, parseCurrentState.GRPCReady, parseWorkspaceQuery.Search, parseWorkspaceQuery.Filter, parseWorkspaceQuery.Sort, parseSupportQuery.Search, parseSupportQuery.Filter, parseSupportQuery.Sort, parseBillingQuery.Search, parseBillingQuery.Filter, parseBillingQuery.Sort, parseIncidentQuery.Search, parseIncidentQuery.Filter, parseIncidentQuery.Sort)

	ui.UseEffect(func() func() {
		parseTicketID := parseSelectedTicketID.Get()
		if parseTicketID <= 0 || !parseCurrentState.Authenticated || !parseCurrentState.CanAccessAdmin || !parseCurrentState.GRPCReady {
			if parseTicketID <= 0 {
				parseSupportDetail.Set(adminSupportDetailSnapshot{})
			}
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return nil
		}
		parseNextSeq := parseDetailSeq.Get() + 1
		parseDetailSeq.Set(parseNextSeq)
		parseIsLoadingDetail.Set(true)
		go func(parseSeq uint64, parseTicketID2 int64) {
			parseResp, parseErr := parseClient.GetAdminSupportTicketDetail(context.Background(), &chatpb.GetAdminSupportTicketDetailRequest{
				TicketId: parseTicketID2,
				Limit:    20,
			})
			if parseDetailSeq.Get() != parseSeq {
				return
			}
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					return
				}
				parseIsLoadingDetail.Set(false)
				chatLog.Warn("admin support detail failed", logging.Fields{"ticket_id": parseTicketID2, "error": parseErr.Error()})
				return
			}
			parseSupportDetail.Set(parseMarshalAdminSupportDetail(parseResp))
			parseIsLoadingDetail.Set(false)
		}(parseNextSeq, parseTicketID)
		return nil
	}, parseSelectedTicketID.Get(), parseCurrentState.Authenticated, parseCurrentState.CanAccessAdmin, parseCurrentState.GRPCReady)

	parseSetQuery := func(parseScope string, parseNext adminListQueryState) {
		parseNext = parseNormalizeAdminListQuery(parseNext)
		switch parseScope {
		case adminScopeWorkspaces:
			parseWorkspaceQueryState.Set(parseNext)
		case adminScopeSupport:
			parseSupportQueryState.Set(parseNext)
		case adminScopeBilling:
			parseBillingQueryState.Set(parseNext)
		case adminScopeIncidents:
			parseIncidentQueryState.Set(parseNext)
		}
	}
	parseGetQuery := func(parseScope string) adminListQueryState {
		switch parseScope {
		case adminScopeWorkspaces:
			return parseWorkspaceQuery
		case adminScopeSupport:
			return parseSupportQuery
		case adminScopeBilling:
			return parseBillingQuery
		case adminScopeIncidents:
			return parseIncidentQuery
		default:
			return adminListQueryState{}
		}
	}

	handleQuerySearch := ui.UseEvent(func(parseE ui.Event) {
		parseScope := parseEventDatasetValue(parseE, dataAdminScope)
		parseNext := parseGetQuery(parseScope)
		parseNext.Search = parseE.GetValue()
		parseNext.Page = 0
		parseSetQuery(parseScope, parseNext)
	})
	handleQueryFilter := ui.UseEvent(func(parseE ui.Event) {
		parseScope := parseEventDatasetValue(parseE, dataAdminScope)
		parseNext := parseGetQuery(parseScope)
		parseNext.Filter = parseEventValueOrDataset(parseE, dataAdminFilter)
		parseNext.Page = 0
		parseSetQuery(parseScope, parseNext)
	})
	handleQuerySort := ui.UseEvent(func(parseE ui.Event) {
		parseScope := parseEventDatasetValue(parseE, dataAdminScope)
		parseNext := parseGetQuery(parseScope)
		parseNext.Sort = parseEventValueOrDataset(parseE, dataAdminSort)
		parseNext.Page = 0
		parseSetQuery(parseScope, parseNext)
	})
	handleNextPage := ui.UseEvent(func(parseE ui.Event) {
		parseScope := parseEventDatasetValue(parseE, dataAdminScope)
		parseNext := parseGetQuery(parseScope)
		parseNext.Page++
		parseSetQuery(parseScope, parseNext)
	})
	handlePrevPage := ui.UseEvent(func(parseE ui.Event) {
		parseScope := parseEventDatasetValue(parseE, dataAdminScope)
		parseNext := parseGetQuery(parseScope)
		if parseNext.Page > 0 {
			parseNext.Page--
		}
		parseSetQuery(parseScope, parseNext)
	})
	handleSelectWorkspace := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
	})
	handleSelectTicket := ui.UseEvent(func(parseE ui.Event) {
		parseTicketID, parseOk := parseEventDatasetInt64(parseE, dataAdminTicketID)
		if !parseOk {
			return
		}
		if parseSelectedTicketID.Get() == parseTicketID {
			parseSelectedTicketID.Set(0)
			return
		}
		parseSelectedTicketID.Set(parseTicketID)
		parseMutationError.Set("")
		parseMutationSuccess.Set("")
	})
	handleConfirmStart := ui.UseEvent(func(parseE ui.Event) {
		parseConfirmAction.Set(parseEventDatasetValue(parseE, dataAdminAction))
		parseConfirmTargetID.Set(parseEventDatasetInt64OrZero(parseE, dataAdminTargetID))
		parseConfirmTargetKey.Set(parseEventDatasetValue(parseE, dataAdminTargetKey))
		parseConfirmScope.Set(parseEventDatasetValue(parseE, dataAdminScope))
		parseConfirmReason.Set("")
		parseSupportNoteBody.Set("")
		parseMutationError.Set("")
	})
	handleConfirmReason := ui.UseEvent(func(parseE ui.Event) {
		parseConfirmReason.Set(parseE.GetValue())
	})
	handleSupportNote := ui.UseEvent(func(parseE ui.Event) {
		parseSupportNoteBody.Set(parseE.GetValue())
	})
	handleConfirmCancel := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		parseConfirmAction.Set("")
		parseConfirmTargetID.Set(0)
		parseConfirmTargetKey.Set("")
		parseConfirmScope.Set("")
		parseConfirmReason.Set("")
		parseSupportNoteBody.Set("")
		parseMutationError.Set("")
	})
	handleConfirmSubmit := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		if parseIsMutationPending.Get() {
			return
		}
		parseAction := parseConfirmAction.Get()
		parseClient := parseChatClientRef.Get()
		if parseAction == "" || parseClient == nil {
			return
		}
		parseIsMutationPending.Set(true)
		parseMutationError.Set("")
		go func(parseAct string, parseTargetID int64, parseTargetKey string, parseReason string, parseNoteBody string) {
			parseReason = strings.TrimSpace(parseReason)
			var parseErr error
			parseSuccess := "Action applied."
			switch parseAct {
			case "support-note":
				_, parseErr = parseClient.AddAdminSupportInternalNote(context.Background(), &chatpb.AddAdminSupportInternalNoteRequest{
					TicketId: parseTargetID,
					Body:     strings.TrimSpace(parseNoteBody),
					Confirm:  true,
					Reason:   parseReason,
				})
				parseSuccess = "Internal note added."
			case "support-escalate":
				_, parseErr = parseClient.EscalateAdminSupportTicket(context.Background(), &chatpb.EscalateAdminSupportTicketRequest{
					TicketId:       parseTargetID,
					Priority:       "urgent",
					Status:         "escalated",
					ResolutionNote: "Escalated from admin dashboard.",
					Confirm:        true,
					Reason:         parseReason,
				})
				parseSuccess = "Support ticket escalated."
			case "feature-toggle":
				_, parseErr = parseClient.SetAdminFeatureFlag(context.Background(), &chatpb.SetAdminFeatureFlagRequest{
					FlagKey:        parseTargetKey,
					IsEnabled:      true,
					RolloutPercent: 100,
					Confirm:        true,
					Reason:         parseReason,
				})
				parseSuccess = "Feature flag enabled."
			case "experiment-rollback":
				_, parseErr = parseClient.RollbackAdminExperiment(context.Background(), &chatpb.RollbackAdminExperimentRequest{
					ExperimentKey: parseTargetKey,
					Confirm:       true,
					Reason:        parseReason,
				})
				parseSuccess = "Experiment rollback requested."
			case "incident-update":
				_, parseErr = parseClient.UpdateAdminIncident(context.Background(), &chatpb.UpdateAdminIncidentRequest{
					IncidentId: parseTargetID,
					Status:     "monitoring",
					Message:    "Moved to monitoring from dashboard control.",
					IsPublic:   true,
					Confirm:    true,
					Reason:     parseReason,
				})
				parseSuccess = "Incident moved to monitoring."
			}
			if parseErr != nil {
				if handleAuthFailure != nil && handleAuthFailure(parseErr) {
					parseIsMutationPending.Set(false)
					return
				}
				parseMutationError.Set(parseBuildUserErrorText(userErrorScopeDashboard, parseErr))
				parseIsMutationPending.Set(false)
				chatLog.Warn("admin operation mutation failed", logging.Fields{"action": parseAct, "target_id": parseTargetID, "target_key": parseTargetKey, "error": parseErr.Error()})
				return
			}
			parseMutationSuccess.Set(parseSuccess)
			parseConfirmAction.Set("")
			parseConfirmTargetID.Set(0)
			parseConfirmTargetKey.Set("")
			parseConfirmScope.Set("")
			parseConfirmReason.Set("")
			parseSupportNoteBody.Set("")
			parseIsMutationPending.Set(false)
			parseRequestSeq.Set(parseRequestSeq.Get() + 1)
		}(parseAction, parseConfirmTargetID.Get(), parseConfirmTargetKey.Get(), parseConfirmReason.Get(), parseSupportNoteBody.Get())
	})

	parseData := parseDataState.Get()
	parseData.WorkspaceQuery = parseWorkspaceQuery
	parseData.SupportQuery = parseSupportQuery
	parseData.BillingQuery = parseBillingQuery
	parseData.IncidentQuery = parseIncidentQuery
	parseData.SelectedTicketID = parseSelectedTicketID.Get()
	parseData.SupportDetail = parseSupportDetail.Get()
	parseData.IsLoadingDetail = parseIsLoadingDetail.Get()
	parseData.ConfirmAction = parseConfirmAction.Get()
	parseData.ConfirmTargetID = parseConfirmTargetID.Get()
	parseData.ConfirmTargetKey = parseConfirmTargetKey.Get()
	parseData.ConfirmScope = parseConfirmScope.Get()
	parseData.ConfirmReason = parseConfirmReason.Get()
	parseData.SupportNoteBody = parseSupportNoteBody.Get()
	parseData.IsMutationPending = parseIsMutationPending.Get()
	parseData.MutationError = parseMutationError.Get()
	parseData.MutationSuccess = parseMutationSuccess.Get()

	return adminOperationsController{
		Data:                  parseData,
		HandleQuerySearch:     handleQuerySearch,
		HandleQueryFilter:     handleQueryFilter,
		HandleQuerySort:       handleQuerySort,
		HandleNextPage:        handleNextPage,
		HandlePrevPage:        handlePrevPage,
		HandleSelectWorkspace: handleSelectWorkspace,
		HandleSelectTicket:    handleSelectTicket,
		HandleConfirmStart:    handleConfirmStart,
		HandleConfirmReason:   handleConfirmReason,
		HandleSupportNote:     handleSupportNote,
		HandleConfirmSubmit:   handleConfirmSubmit,
		HandleConfirmCancel:   handleConfirmCancel,
	}
}

func parseNormalizeAdminListQuery(parseQuery adminListQueryState) adminListQueryState {
	parseQuery.Search = strings.TrimSpace(parseQuery.Search)
	parseQuery.Filter = strings.TrimSpace(parseQuery.Filter)
	parseQuery.Sort = strings.TrimSpace(parseQuery.Sort)
	if parseQuery.Page < 0 {
		parseQuery.Page = 0
	}
	return parseQuery
}

func parseBuildAdminListQuery(parseQuery adminListQueryState, parseLimit int32) *chatpb.AdminListQuery {
	parseOffset := int32(parseQuery.Page) * parseLimit
	if parseOffset < 0 {
		parseOffset = 0
	}
	return &chatpb.AdminListQuery{
		Search:        parseQuery.Search,
		SortBy:        parseQuery.Sort,
		SortDirection: "desc",
		Limit:         parseLimit,
		Offset:        parseOffset,
	}
}

func parseStatusFilterValue(parseFilter string) string {
	parseFilter = strings.TrimSpace(strings.ToLower(parseFilter))
	switch parseFilter {
	case "", "all", "active":
		return ""
	default:
		return parseFilter
	}
}

func parseEventDatasetInt64OrZero(parseE ui.Event, parseKey string) int64 {
	parseID, parseOk := parseEventDatasetInt64(parseE, parseKey)
	if !parseOk {
		return 0
	}
	return parseID
}

func parseApplySuperuserSlices(parseNext *adminOperationsData, parseResp *chatpb.GetSuperuserSlicesResponse) {
	if parseNext == nil || parseResp == nil {
		return
	}
	parseNext.HasSuperuserData = true
	for _, parseWorkspace := range parseResp.GetWorkspaces() {
		parseNext.Workspaces = append(parseNext.Workspaces, adminWorkspaceListRow{
			WorkspaceID: parseWorkspace.GetId(),
			Name:        parseWorkspace.GetName(),
			Slug:        parseWorkspace.GetSlug(),
			PlanCode:    parseWorkspace.GetPlanCode(),
			Status:      parseWorkspace.GetStatus(),
			OwnerUserID: parseWorkspace.GetOwnerUserId(),
			UpdatedAt:   parseWorkspace.GetUpdatedAt(),
		})
	}
	for _, parseTicket := range parseResp.GetSupportTickets() {
		parseNext.SupportTickets = append(parseNext.SupportTickets, parseMarshalSupportTicketRow(parseTicket))
	}
	for _, parseOverage := range parseResp.GetBillingPlanOverages() {
		parseNext.BillingOverages = append(parseNext.BillingOverages, adminBillingOverageRow{
			PlanCode:          parseOverage.GetPlanCode(),
			MeterKey:          parseOverage.GetMeterKey(),
			IncludedUnits:     parseOverage.GetIncludedUnits(),
			SoftLimitUnits:    parseOverage.GetSoftLimitUnits(),
			HardLimitUnits:    parseOverage.GetHardLimitUnits(),
			OveragePriceCents: parseOverage.GetOveragePriceCents(),
			UpdatedAt:         parseOverage.GetUpdatedAt(),
		})
	}
	for _, parseQuota := range parseResp.GetBillingQuotaPolicies() {
		parseNext.BillingQuotas = append(parseNext.BillingQuotas, adminBillingQuotaRow{
			PlanCode:        parseQuota.GetPlanCode(),
			QuotaKey:        parseQuota.GetQuotaKey(),
			SoftLimitValue:  parseQuota.GetSoftLimitValue(),
			HardLimitValue:  parseQuota.GetHardLimitValue(),
			EnforcementMode: parseQuota.GetEnforcementMode(),
			UpdatedAt:       parseQuota.GetUpdatedAt(),
		})
	}
	for _, parseTrigger := range parseResp.GetBillingUpgradeTriggers() {
		parseNext.BillingTriggers = append(parseNext.BillingTriggers, adminBillingTriggerRow{
			PlanCode:        parseTrigger.GetPlanCode(),
			TriggerKey:      parseTrigger.GetTriggerKey(),
			ThresholdPct:    parseTrigger.GetThresholdPercent(),
			UpgradePlanCode: parseTrigger.GetUpgradePlanCode(),
			IsEnabled:       parseTrigger.GetIsEnabled(),
			UpdatedAt:       parseTrigger.GetUpdatedAt(),
		})
	}
	for _, parseIncident := range parseResp.GetIncidents() {
		parseNext.Incidents = append(parseNext.Incidents, parseMarshalIncidentRow(parseIncident))
	}
	for _, parseExperiment := range parseResp.GetExperiments() {
		parseNext.Experiments = append(parseNext.Experiments, adminExperimentRow{
			ExperimentKey: parseExperiment.GetExperimentKey(),
			Name:          parseExperiment.GetName(),
			Status:        parseExperiment.GetStatus(),
			UpdatedAt:     parseExperiment.GetUpdatedAt(),
		})
	}
	for _, parseGuardrail := range parseResp.GetWorkspaceCostGuardrails() {
		parseNext.CostGuardrails = append(parseNext.CostGuardrails, adminCostGuardrailRow{
			WorkspaceID:            parseGuardrail.GetWorkspaceId(),
			GuardrailKey:           parseGuardrail.GetGuardrailKey(),
			DailyBudgetCents:       parseGuardrail.GetDailyBudgetCents(),
			MonthlyBudgetCents:     parseGuardrail.GetMonthlyBudgetCents(),
			MaxCostPerRequestCents: parseGuardrail.GetMaxCostPerRequestCents(),
			ActionMode:             parseGuardrail.GetActionMode(),
			UpdatedAt:              parseGuardrail.GetUpdatedAt(),
		})
	}
}

func parseApplyBusinessQueue(parseNext *adminOperationsData, parseResp *chatpb.GetAdminBusinessQueueResponse) {
	if parseNext == nil || parseResp == nil {
		return
	}
	for _, parseEvent := range parseResp.GetFailedPaymentEvents() {
		parseNext.FailedPayments = append(parseNext.FailedPayments, adminBillingEventRow{
			EventID:      parseEvent.GetId(),
			CustomerID:   parseEvent.GetCustomerId(),
			InvoiceID:    parseEvent.GetInvoiceId(),
			EventType:    parseEvent.GetEventType(),
			EventSummary: parseEvent.GetEventSummary(),
			CreatedAt:    parseEvent.GetCreatedAt(),
		})
	}
	for _, parseEvent := range parseResp.GetDunningEvents() {
		parseNext.DunningEvents = append(parseNext.DunningEvents, adminDunningEventRow{
			EventID:       parseEvent.GetId(),
			CustomerID:    parseEvent.GetCustomerId(),
			InvoiceID:     parseEvent.GetInvoiceId(),
			Status:        parseEvent.GetStatus(),
			AttemptCount:  parseEvent.GetAttemptCount(),
			FailureReason: parseEvent.GetFailureReason(),
			NextAttemptAt: parseEvent.GetNextAttemptAt(),
			ResolvedAt:    parseEvent.GetResolvedAt(),
		})
	}
	for _, parseAccount := range parseResp.GetTopAccounts() {
		parseNext.BusinessTopAccounts = append(parseNext.BusinessTopAccounts, adminUserRow{
			UserID:            parseAccount.GetUserId(),
			Email:             parseAccount.GetEmail(),
			DisplayName:       parseAccount.GetDisplayName(),
			ConversationCount: parseAccount.GetConversationCount(),
			MessageCount:      parseAccount.GetMessageCount(),
			TotalCostUSD:      parseAccount.GetTotalCostUsd(),
			LastSeenAt:        parseAccount.GetLastSeenAt(),
		})
	}
}

func parseApplyChatsAnomalies(parseNext *adminOperationsData, parseResp *chatpb.GetAdminChatsAnomaliesResponse) {
	if parseNext == nil || parseResp == nil {
		return
	}
	for _, parseUsage := range parseResp.GetFailedReplies() {
		parseNext.ChatFailedReplies = append(parseNext.ChatFailedReplies, adminUsageEventRow{
			ProviderID:   parseUsage.GetProviderId(),
			ModelID:      parseUsage.GetModelId(),
			TotalCostUSD: parseUsage.GetTotalCostUsd(),
			Status:       parseUsage.GetStatus(),
			CreatedAt:    parseUsage.GetCreatedAt(),
		})
	}
	for _, parseUsage := range parseResp.GetSlowReplies() {
		parseNext.ChatSlowReplies = append(parseNext.ChatSlowReplies, adminUsageEventRow{
			ProviderID:   parseUsage.GetProviderId(),
			ModelID:      parseUsage.GetModelId(),
			TotalCostUSD: parseUsage.GetTotalCostUsd(),
			Status:       parseUsage.GetStatus(),
			CreatedAt:    parseUsage.GetCreatedAt(),
		})
	}
	for _, parseThread := range parseResp.GetHighCostThreads() {
		parseNext.ChatHighCostThreads = append(parseNext.ChatHighCostThreads, adminConvRow{
			ConversationID: parseThread.GetConversationId(),
			PublicID:       parseThread.GetPublicId(),
			UserID:         parseThread.GetUserId(),
			Email:          parseThread.GetEmail(),
			DisplayName:    parseThread.GetDisplayName(),
			Preview:        parseThread.GetPreview(),
			MessageCount:   parseThread.GetMessageCount(),
			TotalCostUSD:   parseThread.GetTotalCostUsd(),
			LastActivityAt: parseThread.GetLastActivityAt(),
			StartedAt:      parseThread.GetStartedAt(),
		})
	}
}

func parseApplyProviderHealthTrends(parseNext *adminOperationsData, parseResp *chatpb.GetAdminProviderHealthTrendsResponse) {
	if parseNext == nil || parseResp == nil {
		return
	}
	for _, parseEvent := range parseResp.GetFallbackEvents() {
		parseNext.ProviderFallbacks = append(parseNext.ProviderFallbacks, adminProviderFallbackRow{
			WorkspaceID: parseEvent.GetWorkspaceId(),
			EventType:   parseEvent.GetEventType(),
			Summary:     parseEvent.GetSummary(),
			CreatedAt:   parseEvent.GetCreatedAt(),
		})
	}
}

func parseApplyOpsDrilldown(parseNext *adminOperationsData, parseResp *chatpb.GetAdminOpsDrilldownResponse) {
	if parseNext == nil || parseResp == nil {
		return
	}
	for _, parseFlag := range parseResp.GetFeatureFlags() {
		parseNext.FeatureFlags = append(parseNext.FeatureFlags, adminFeatureFlagRow{
			FlagKey:        parseFlag.GetFlagKey(),
			Description:    parseFlag.GetDescription(),
			IsEnabled:      parseFlag.GetIsEnabled(),
			RolloutPercent: parseFlag.GetRolloutPercent(),
			UpdatedAt:      parseFlag.GetUpdatedAt(),
		})
	}
}

func parseApplyWorkspaceAdminSlices(parseNext *adminOperationsData, parseResp *chatpb.GetWorkspaceAdminSlicesResponse) {
	if parseNext == nil || parseResp == nil {
		return
	}
	parseNext.HasWorkspaceData = true
	for _, parseMember := range parseResp.GetMemberships() {
		parseNext.WorkspaceMembers = append(parseNext.WorkspaceMembers, adminWorkspaceMemberRow{
			UserID:    parseMember.GetUserId(),
			RoleKey:   parseMember.GetRoleKey(),
			Status:    parseMember.GetStatus(),
			CreatedAt: parseMember.GetCreatedAt(),
		})
	}
	for _, parseKey := range parseResp.GetApiKeys() {
		parseNext.WorkspaceAPIKeys = append(parseNext.WorkspaceAPIKeys, adminWorkspaceAPIKeyRow{
			KeyID:     parseKey.GetKeyId(),
			Label:     parseKey.GetLabel(),
			KeyPrefix: parseKey.GetKeyPrefix(),
			RevokedAt: parseKey.GetRevokedAt(),
			CreatedAt: parseKey.GetCreatedAt(),
		})
	}
	for _, parseWebhook := range parseResp.GetWebhookEndpoints() {
		parseNext.WorkspaceWebhooks = append(parseNext.WorkspaceWebhooks, adminWorkspaceWebhookRow{
			Label:          parseWebhook.GetLabel(),
			TargetURL:      parseWebhook.GetTargetUrl(),
			IsEnabled:      parseWebhook.GetIsEnabled(),
			FailureCount:   parseWebhook.GetFailureCount(),
			LastDeliveryAt: parseWebhook.GetLastDeliveryAt(),
			CreatedAt:      parseWebhook.GetCreatedAt(),
		})
	}
	for _, parseInvite := range parseResp.GetInvitations() {
		parseNext.WorkspaceInvites = append(parseNext.WorkspaceInvites, adminWorkspaceInviteRow{
			Email:     parseInvite.GetEmail(),
			RoleKey:   parseInvite.GetRoleKey(),
			Status:    parseInvite.GetStatus(),
			ExpiresAt: parseInvite.GetExpiresAt(),
			CreatedAt: parseInvite.GetCreatedAt(),
		})
	}
	for _, parseAudit := range parseResp.GetAuditLogs() {
		parseNext.WorkspaceAuditLogs = append(parseNext.WorkspaceAuditLogs, adminAuditRow{
			EventType: parseAudit.GetEventType(),
			Summary:   parseAudit.GetSummary(),
			CreatedAt: parseAudit.GetCreatedAt(),
		})
	}
	for _, parseUsage := range parseResp.GetUsageEvents() {
		parseNext.WorkspaceUsage = append(parseNext.WorkspaceUsage, adminUsageEventRow{
			ProviderID:   parseUsage.GetProviderId(),
			ModelID:      parseUsage.GetModelId(),
			TotalCostUSD: parseUsage.GetTotalCostUsd(),
			Status:       parseUsage.GetStatus(),
			CreatedAt:    parseUsage.GetCreatedAt(),
		})
	}
	if parseBilling := parseResp.GetBillingSummary(); parseBilling != nil {
		parseNext.WorkspaceBilling = adminWorkspaceBillingRow{
			CustomerCount:           parseBilling.GetCustomerCount(),
			ActiveSubscriptionCount: parseBilling.GetActiveSubscriptionCount(),
			OpenInvoiceCount:        parseBilling.GetOpenInvoiceCount(),
			DunningEventCount:       parseBilling.GetDunningEventCount(),
			RecentUsageCostUSD:      parseBilling.GetRecentUsageCostUsd(),
		}
	}
}

func parseMarshalSupportTicketRow(parseTicket *chatpb.SupportTicketEntry) adminSupportTicketRow {
	if parseTicket == nil {
		return adminSupportTicketRow{}
	}
	return adminSupportTicketRow{
		TicketID:       parseTicket.GetId(),
		TicketKey:      parseTicket.GetTicketKey(),
		WorkspaceID:    parseTicket.GetWorkspaceId(),
		UserID:         parseTicket.GetUserId(),
		Status:         parseTicket.GetStatus(),
		Priority:       parseTicket.GetPriority(),
		Subject:        parseTicket.GetSubject(),
		AssigneeUserID: parseTicket.GetAssigneeUserId(),
		UpdatedAt:      parseTicket.GetUpdatedAt(),
	}
}

func parseMarshalIncidentRow(parseIncident *chatpb.IncidentEntry) adminIncidentRow {
	if parseIncident == nil {
		return adminIncidentRow{}
	}
	return adminIncidentRow{
		IncidentID:  parseIncident.GetId(),
		IncidentKey: parseIncident.GetIncidentKey(),
		Severity:    parseIncident.GetSeverity(),
		Status:      parseIncident.GetStatus(),
		Title:       parseIncident.GetTitle(),
		Summary:     parseIncident.GetSummary(),
		UpdatedAt:   parseIncident.GetUpdatedAt(),
	}
}

func parseMarshalAdminSupportDetail(parseResp *chatpb.GetAdminSupportTicketDetailResponse) adminSupportDetailSnapshot {
	if parseResp == nil || parseResp.GetDetail() == nil {
		return adminSupportDetailSnapshot{}
	}
	parseDetail := parseResp.GetDetail()
	parseSnap := adminSupportDetailSnapshot{
		HasData: true,
		Ticket:  parseMarshalSupportTicketRow(parseDetail.GetTicket()),
	}
	for _, parseMessage := range parseDetail.GetMessages() {
		parseSnap.Messages = append(parseSnap.Messages, adminSupportMessageRow{
			AuthorUserID: parseMessage.GetAuthorUserId(),
			MessageType:  parseMessage.GetMessageType(),
			Body:         parseMessage.GetBody(),
			IsInternal:   parseMessage.GetIsInternal(),
			CreatedAt:    parseMessage.GetCreatedAt(),
		})
	}
	for _, parseAudit := range parseDetail.GetAccountActionHistory() {
		parseSnap.AccountActions = append(parseSnap.AccountActions, adminAuditRow{
			EventType: parseAudit.GetEventType(),
			Summary:   parseAudit.GetSummary(),
			CreatedAt: parseAudit.GetCreatedAt(),
		})
	}
	return parseSnap
}

func parseAdminListQueryKey(parseQuery adminListQueryState) string {
	return strings.Join([]string{
		strings.TrimSpace(parseQuery.Search),
		strings.TrimSpace(parseQuery.Filter),
		strings.TrimSpace(parseQuery.Sort),
		formatDashboardInt(parseQuery.Page),
	}, "|")
}

func parseAdminNowLabel() string {
	return time.Now().Format("15:04")
}
