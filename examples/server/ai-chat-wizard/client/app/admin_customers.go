//go:build js && wasm

package app

import (
	"sort"
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/i18n"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

const adminCustomersPageSize = 10

// renderDashboardCustomersEnhanced renders the Customers admin slice with search,
// sort, pagination, a user detail panel, and disable/restore confirmation modal.
func renderDashboardCustomersEnhanced(parseIntl i18n.Runtime, parseView appViewState, parseCustomers adminCustomersController, parseAdminOperations adminOperationsController) ui.Node {
	_ = parseIntl
	parseData := parseView.AdminDashboardData
	if parseData.IsDenied {
		return Fragment(
			renderDashboardSliceHeader("\U0001f465", "Customers", "User search, billing state, and account controls"),
			renderDashboardDeniedBanner(),
		)
	}
	if parseData.IsLoading {
		return renderDashboardLoadingState("Loading customer data\u2026")
	}
	if parseData.Error != "" {
		return Fragment(
			renderDashboardSliceHeader("\U0001f465", "Customers", "User search, billing state, and account controls"),
			renderDashboardErrorBanner(parseData.Error),
		)
	}

	// Resolve the user list to display.
	parseDisplayUsers := parseCustomers.Data.SearchResults
	if parseDisplayUsers == nil {
		parseDisplayUsers = parseData.RecentUsers
	}
	parseDisplayUsers = parseFilterAndSortAdminUsers(parseDisplayUsers, parseCustomers.Data.Query, parseCustomers.Data.UserAccessStates)

	// Paginate.
	parseTotalUsers := len(parseDisplayUsers)
	parseTotalPages := (parseTotalUsers + adminCustomersPageSize - 1) / adminCustomersPageSize
	if parseTotalPages < 1 {
		parseTotalPages = 1
	}
	parsePage := parseCustomers.Data.CurrentPage
	if parsePage >= parseTotalPages {
		parsePage = parseTotalPages - 1
	}
	if parsePage < 0 {
		parsePage = 0
	}
	parsePageStart := parsePage * adminCustomersPageSize
	parsePageEnd := parsePageStart + adminCustomersPageSize
	if parsePageEnd > parseTotalUsers {
		parsePageEnd = parseTotalUsers
	}
	parsePageUsers := parseDisplayUsers[parsePageStart:parsePageEnd]
	isDetailOpen := parseCustomers.Data.SelectedUserID > 0

	return Fragment(
		renderDashboardSliceHeader("\U0001f465", "Customers", "User search, billing state, and account controls"),
		renderDashboardAlertState(parseView, parseAdminOperations),
		renderAdminCustomersSearchBar(parseCustomers, parseData.Summary.TotalUsers, parseTotalUsers),
		Div(
			ClassNames(
				"flex flex-col gap-5",
				When(isDetailOpen, "lg:flex-row"),
			),
			// User list.
			Div(
				ClassNames(
					"min-w-0",
					When(isDetailOpen, "lg:w-[420px] lg:shrink-0"),
					When(!isDetailOpen, "w-full"),
				),
				renderAdminUserListTable(parsePageUsers, parseCustomers),
				renderAdminCustomersPagination(parsePage, parseTotalPages, parseTotalUsers, parseCustomers),
			),
			// Detail panel.
			If(isDetailOpen,
				renderAdminUserDetailPanel(parseCustomers),
			),
		),
		renderDashboardWorkspacesList(parseAdminOperations),
		// Confirmation modal.
		If(parseCustomers.Data.ConfirmAction != "",
			renderAdminConfirmModal(parseCustomers),
		),
	)
}

// renderAdminCustomersSearchBar renders the search input and result count header.
func renderAdminCustomersSearchBar(parseCustomers adminCustomersController, parseTotalGlobal int64, parseFilteredCount int) ui.Node {
	parseCountLabel := formatDashboardInt(parseFilteredCount) + " shown"
	if parseCustomers.Data.Query.Search == "" && parseCustomers.Data.Query.Filter == "all" {
		parseCountLabel = formatDashboardInt64(parseTotalGlobal) + " total"
	}
	return Div(
		ClassStr("mb-4 grid gap-3 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 md:grid-cols-[1fr_auto_auto_auto]"),
		Div(
			ClassStr("min-w-0"),
			P(ClassStr("mb-1 text-[10px] font-medium uppercase tracking-[0.18em] text-white/35"), Text("Search")),
			Input(
				Type("text"),
				Placeholder("Search users\u2026"),
				Value(parseCustomers.Data.Query.Search),
				OnInput(parseCustomers.HandleSearch),
				ClassStr("w-full rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-white placeholder:text-white/30 outline-none focus:border-white/20 focus:bg-white/[0.06]"),
			),
		),
		renderDashboardRailSelect("Filter", adminScopeUsers, dataAdminFilter, parseCustomers.Data.Query.Filter, []adminSelectOption{
			{ID: "all", Label: "All"},
			{ID: "active", Label: "Active"},
			{ID: "disabled", Label: "Disabled"},
			{ID: "high_spend", Label: "High spend"},
			{ID: "stale", Label: "Stale"},
		}, parseCustomers.HandleFilter),
		renderDashboardRailSelect("Sort", adminScopeUsers, dataAdminSort, parseCustomers.Data.Query.Sort, []adminSelectOption{
			{ID: "last_seen", Label: "Last seen"},
			{ID: "spend", Label: "Spend"},
			{ID: "conversations", Label: "Convs"},
			{ID: "state", Label: "State"},
			{ID: "email", Label: "Email"},
		}, parseCustomers.HandleSort),
		Div(
			ClassStr("min-w-[9rem]"),
			P(ClassStr("mb-1 text-[10px] font-medium uppercase tracking-[0.18em] text-white/35"), Text("Result")),
			Div(ClassStr("rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-white/70"), Text(parseCountLabel)),
		),
		If(parseCustomers.Data.IsSearching,
			Div(ClassStr("h-4 w-4 animate-spin rounded-full border-2 border-white/20 border-t-white/60 md:col-span-4")),
		),
	)
}

// renderAdminUserListTable renders the paginated user list with clickable rows.
func renderAdminUserListTable(parseUsers []adminUserRow, parseCustomers adminCustomersController) ui.Node {
	if len(parseUsers) == 0 {
		return renderDashboardEmptyState("\U0001f465", "No users found", "Try a different search query or check back after users sign up.")
	}
	parseHeaderCells := []ui.Node{
		Th(ClassStr("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("User")),
		Th(ClassStr("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("State")),
		Th(ClassStr("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Convs")),
		Th(ClassStr("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Spend")),
		Th(ClassStr("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Last seen")),
	}
	parseRows := make([]ui.Node, len(parseUsers))
	for parseI, parseUser := range parseUsers {
		parseIsSelected := parseCustomers.Data.SelectedUserID == parseUser.UserID
		parseIDStr := strconv.FormatInt(parseUser.UserID, 10)
		parseRowClass := ClassNames(
			"border-t border-white/[0.06] cursor-pointer transition-colors",
			When(parseIsSelected, "bg-white/[0.07]"),
			When(!parseIsSelected, "hover:bg-white/[0.03]"),
		)
		parseRows[parseI] = Tr(
			ClassStr(parseRowClass),
			Data(dataAdminUserID, parseIDStr),
			OnClick(parseCustomers.HandleSelectUser),
			Td(ClassStr("px-4 py-2.5 text-xs text-white/80 align-top font-medium"),
				Text(parseDashboardUserLabel(parseUser.DisplayName, parseUser.Email)),
			),
			Td(ClassStr("px-4 py-2.5 text-xs text-white/60 align-top"),
				renderAdminUserAccessBadge(parseUserAccessStateForRow(parseUser.UserID, parseCustomers.Data.UserAccessStates)),
			),
			Td(ClassStr("px-4 py-2.5 text-xs text-white/60 align-top"),
				Text(formatDashboardInt64(parseUser.ConversationCount)),
			),
			Td(ClassStr("px-4 py-2.5 text-xs text-white/60 align-top"),
				Text(formatCostUSD(parseUser.TotalCostUSD)),
			),
			Td(ClassStr("px-4 py-2.5 text-xs text-white/40 align-top"),
				Text(parseDashboardShortAt(parseUser.LastSeenAt)),
			),
		)
	}
	return Div(
		ClassStr("overflow-x-auto rounded-[1.4rem] border border-white/10"),
		Table(
			ClassStr("w-full min-w-[360px] border-collapse"),
			Thead(Tr(ClassStr("bg-white/[0.03]"), Fragment(parseHeaderCells))),
			Tbody(Fragment(parseRows)),
		),
	)
}

// renderAdminCustomersPagination renders prev/next page controls.
func renderAdminCustomersPagination(parsePage, parseTotalPages, parseTotalUsers int, parseCustomers adminCustomersController) ui.Node {
	if parseTotalPages <= 1 {
		return nil
	}
	parsePageLabel := "Page " + strconv.Itoa(parsePage+1) + " of " + strconv.Itoa(parseTotalPages)
	parsePrevClass := ClassNames(
		"rounded-lg border px-3 py-1.5 text-xs transition-colors",
		When(parsePage <= 0, "border-white/[0.06] text-white/20 pointer-events-none"),
		When(parsePage > 0, "border-white/10 text-white/60 hover:bg-white/[0.06] hover:text-white/80 cursor-pointer"),
	)
	parseNextClass := ClassNames(
		"rounded-lg border px-3 py-1.5 text-xs transition-colors",
		When(parsePage >= parseTotalPages-1, "border-white/[0.06] text-white/20 pointer-events-none"),
		When(parsePage < parseTotalPages-1, "border-white/10 text-white/60 hover:bg-white/[0.06] hover:text-white/80 cursor-pointer"),
	)
	return Div(
		ClassStr("mt-3 flex items-center justify-between"),
		P(ClassStr("text-xs text-white/35"), Text(parsePageLabel)),
		Div(ClassStr("flex items-center gap-2"),
			Button(ClassStr(parsePrevClass), OnClick(parseCustomers.HandlePrevPage), Text("\u2190 Prev")),
			Button(ClassStr(parseNextClass), OnClick(parseCustomers.HandleNextPage), Text("Next \u2192")),
		),
	)
}

// renderAdminUserDetailPanel renders the expanded detail panel for a selected user.
func renderAdminUserDetailPanel(parseCustomers adminCustomersController) ui.Node {
	parseData := parseCustomers.Data
	if parseData.IsLoadingDetail {
		return Div(
			ClassStr("flex-1 min-w-0 rounded-[1.4rem] border border-white/10 bg-white/[0.02] flex items-center justify-center py-16"),
			Div(ClassStr("flex flex-col items-center gap-3 text-white/40"),
				Div(ClassStr("h-5 w-5 animate-spin rounded-full border-2 border-white/20 border-t-white/60")),
				P(ClassStr("text-xs"), Text("Loading user detail\u2026")),
			),
		)
	}
	parseDetail := parseData.UserDetail
	if !parseDetail.HasData {
		return Div(
			ClassStr("flex-1 min-w-0 rounded-[1.4rem] border border-white/10 bg-white/[0.02] flex items-center justify-center py-16"),
			P(ClassStr("text-xs text-white/30"), Text("Fetching detail\u2026")),
		)
	}

	parseDisplayName := parseDetail.DisplayName
	if parseDisplayName == "" {
		parseDisplayName = parseDetail.Email
	}
	parseAccessState := parseAdminUserAccessStateForDetail(parseDetail, parseData.UserAccessStates)

	return Div(
		ClassStr("flex-1 min-w-0 rounded-[1.4rem] border border-white/10 bg-white/[0.02] p-5"),
		// Header.
		Div(ClassStr("mb-4 flex items-start justify-between gap-3"),
			Div(
				P(ClassStr("text-sm font-semibold text-white"), Text(parseDisplayName)),
				P(ClassStr("mt-0.5 text-xs text-white/40"), Text(parseDetail.Email)),
				Div(ClassStr("mt-2"), renderAdminUserAccessBadge(parseAccessState)),
			),
			// Close button.
			Button(
				ClassStr("rounded-lg border border-white/10 bg-white/[0.04] px-2.5 py-1 text-xs text-white/50 hover:text-white/80 transition-colors"),
				Data(dataAdminUserID, strconv.FormatInt(parseDetail.UserID, 10)),
				OnClick(parseCustomers.HandleSelectUser),
				Text("\u00d7 Close"),
			),
		),

		// Status and actions.
		renderAdminUserActionBand(parseCustomers),

		// Core stats.
		renderDashboardSectionHeader("Account stats"),
		Div(ClassStr("grid grid-cols-3 gap-2 mb-5"),
			renderAdminMiniCard("Chats", formatDashboardInt64(parseDetail.ConvCount)),
			renderAdminMiniCard("Messages", formatDashboardInt64(parseDetail.MessageCount)),
			renderAdminMiniCard("Total spend", formatCostUSD(parseDetail.TotalCostUSD)),
		),

		// Recent sessions.
		If(len(parseDetail.Sessions) > 0,
			Fragment(
				renderDashboardSectionHeader("Recent sessions"),
				renderAdminSessionsTable(parseDetail.Sessions),
			),
		),

		// Recent usage events.
		If(len(parseDetail.UsageEvents) > 0,
			Fragment(
				renderDashboardSectionHeader("Recent usage"),
				renderAdminUsageEventsTable(parseDetail.UsageEvents),
			),
		),

		// Audit log.
		If(len(parseDetail.AuditLogs) > 0,
			Fragment(
				renderDashboardSectionHeader("Audit log"),
				renderAdminAuditTable(parseDetail.AuditLogs),
			),
		),
	)
}

// renderAdminUserActionBand renders the mutation success/error banner and action buttons.
func renderAdminUserActionBand(parseCustomers adminCustomersController) ui.Node {
	parseData := parseCustomers.Data
	parseUserIDStr := strconv.FormatInt(parseData.SelectedUserID, 10)
	_ = parseUserIDStr
	parseAccessState := parseAdminUserAccessStateForDetail(parseData.UserDetail, parseData.UserAccessStates)
	isDisabled := parseAccessState == "disabled"
	return Div(
		ClassStr("mb-4 flex items-center gap-2 flex-wrap"),
		Div(
			ClassStr(ClassNames(
				"w-full rounded-xl border px-4 py-3 text-xs leading-5",
				When(isDisabled, "border-red-500/25 bg-red-500/5 text-red-200/75"),
				When(!isDisabled, "border-green-500/20 bg-green-500/5 text-green-200/70"),
			)),
			Text(parseAdminUserAccessImpactText(parseAccessState)),
		),
		If(parseData.MutationSuccess != "",
			Div(
				ClassStr("w-full rounded-xl border border-green-500/20 bg-green-500/5 px-4 py-3"),
				Div(ClassStr("mb-1 text-[10px] uppercase tracking-[0.18em] text-green-400/55"), Text("Action receipt")),
				P(ClassStr("text-xs text-green-300"), Text(parseData.MutationSuccess)),
				Div(
					ClassStr("mt-2 flex items-center gap-3 text-[10px] text-green-400/45"),
					Span(Text("Recorded just now")),
					Span(ClassStr("text-green-400/20"), Text("·")),
					Span(Text("Verify in user detail below")),
				),
			),
		),
		If(parseData.MutationError != "",
			Div(
				ClassStr("w-full flex flex-col gap-1 rounded-xl border border-red-500/20 bg-red-500/5 px-3 py-2 text-xs text-red-400"),
				Text(parseUserErrorMessage(parseData.MutationError)),
				renderSupportIDChip(parseUserErrorRequestID(parseData.MutationError)),
			),
		),
		If(!isDisabled,
			Button(
				ClassStr("rounded-xl border border-red-500/25 bg-red-500/5 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 transition-colors cursor-pointer"),
				Data(dataAdminAction, "disable"),
				OnClick(parseCustomers.HandleConfirmStart),
				Text("Disable account"),
			),
		),
		If(isDisabled,
			Button(
				ClassStr("rounded-xl border border-green-500/20 bg-green-500/5 px-3 py-1.5 text-xs text-green-400 hover:bg-green-500/10 transition-colors cursor-pointer"),
				Data(dataAdminAction, "restore"),
				OnClick(parseCustomers.HandleConfirmStart),
				Text("Restore account"),
			),
		),
	)
}

// renderAdminMiniCard renders a compact stat card for the user detail panel.
func renderAdminMiniCard(parseLabel, parseValue string) ui.Node {
	return Div(
		ClassStr("rounded-xl border border-white/8 bg-white/[0.03] px-3 py-2"),
		P(ClassStr("text-[10px] font-medium uppercase tracking-wide text-white/30"), Text(parseLabel)),
		P(ClassStr("mt-1 text-sm font-semibold text-white"), Text(parseValue)),
	)
}

// renderAdminSessionsTable renders the recent sessions table inside the user detail panel.
func renderAdminSessionsTable(parseSessions []adminSessionRow) ui.Node {
	parseRows := make([][]string, len(parseSessions))
	for parseI, parseSession := range parseSessions {
		parseAgent := parseSession.UserAgent
		if len(parseAgent) > 32 {
			parseAgent = parseAgent[:32] + "\u2026"
		}
		if parseAgent == "" {
			parseAgent = "\u2014"
		}
		parseRevoked := "\u2014"
		if parseSession.RevokedAt != "" {
			parseRevoked = parseDashboardShortAt(parseSession.RevokedAt)
		}
		parseRows[parseI] = []string{
			parseAgent,
			parseSession.IPAddress,
			parseDashboardShortAt(parseSession.LastSeenAt),
			parseDashboardShortAt(parseSession.ExpiresAt),
			parseRevoked,
		}
	}
	return renderDashboardTable(
		[]string{"Agent", "IP", "Last seen", "Expires", "Revoked"},
		parseRows,
	)
}

// renderAdminUsageEventsTable renders the recent usage events table.
func renderAdminUsageEventsTable(parseEvents []adminUsageEventRow) ui.Node {
	parseRows := make([][]string, len(parseEvents))
	for parseI, parseEvent := range parseEvents {
		parseModel := parseEvent.ModelID
		if parseModel == "" {
			parseModel = parseEvent.ProviderID
		}
		parseRows[parseI] = []string{
			parseModel,
			formatCostUSD(parseEvent.TotalCostUSD),
			parseEvent.Status,
			parseDashboardShortAt(parseEvent.CreatedAt),
		}
	}
	return renderDashboardTable(
		[]string{"Model", "Cost", "Status", "Date"},
		parseRows,
	)
}

// renderAdminAuditTable renders the recent audit log table.
func renderAdminAuditTable(parseAuditLogs []adminAuditRow) ui.Node {
	parseRows := make([][]string, len(parseAuditLogs))
	for parseI, parseLog := range parseAuditLogs {
		parseSummary := parseLog.Summary
		if len(parseSummary) > 60 {
			parseSummary = parseSummary[:60] + "\u2026"
		}
		if parseSummary == "" {
			parseSummary = "\u2014"
		}
		parseRows[parseI] = []string{
			parseLog.EventType,
			parseSummary,
			parseDashboardShortAt(parseLog.CreatedAt),
		}
	}
	return renderDashboardTable(
		[]string{"Event", "Summary", "Date"},
		parseRows,
	)
}

// renderAdminConfirmModal renders the confirmation modal for disable/restore actions.
func renderAdminConfirmModal(parseCustomers adminCustomersController) ui.Node {
	parseData := parseCustomers.Data
	parseAction := parseData.ConfirmAction
	parseDetail := parseData.UserDetail

	parseTitle := "Disable account"
	parseBody := "This will prevent the user from signing in. Their data is preserved and the account can be restored at any time."
	parseButtonLabel := "Confirm disable"
	parseButtonClass := "rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400 hover:bg-red-500/15 transition-colors cursor-pointer"
	if parseAction == "restore" {
		parseTitle = "Restore account"
		parseBody = "This will re-enable the user's ability to sign in and use the workspace."
		parseButtonLabel = "Confirm restore"
		parseButtonClass = "rounded-xl border border-green-500/20 bg-green-500/8 px-4 py-2 text-sm text-green-400 hover:bg-green-500/12 transition-colors cursor-pointer"
	}

	parseTargetLabel := parseDetail.Email
	if parseDetail.DisplayName != "" {
		parseTargetLabel = parseDetail.DisplayName + " \u00b7 " + parseDetail.Email
	}
	if parseTargetLabel == "" {
		parseTargetLabel = "this user"
	}

	return Div(
		ClassStr("fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"),
		Div(
			ClassStr("w-full max-w-md rounded-[1.6rem] border border-white/10 bg-[#13131e] p-6 shadow-2xl"),
			// Modal header.
			Div(ClassStr("mb-4"),
				P(ClassStr("text-base font-semibold text-white"), Text(parseTitle)),
				P(ClassStr("mt-1 text-xs text-white/50"), Text(parseTargetLabel)),
			),

			// Impact copy.
			Div(
				ClassStr("mb-5 rounded-xl border border-white/8 bg-white/[0.03] px-4 py-3 text-xs leading-5 text-white/60"),
				P(Text(parseBody)),
				Ul(ClassStr("mt-2 list-disc space-y-1 pl-4"),
					Li(Text("Authentication and active-session impact is audited.")),
					Li(Text("Related support and billing review surfaces keep the action receipt visible.")),
					Li(Text("The operator reason is required before the action can be submitted.")),
				),
			),

			// Reason input.
			Div(ClassStr("mb-5"),
				P(ClassStr("mb-1.5 text-xs font-medium text-white/50"), Text("Reason")),
				Tag("textarea",
					Placeholder("Why are you taking this action?"),
					Value(parseData.ConfirmReason),
					OnInput(parseCustomers.HandleConfirmReason),
					Rows(3),
					ClassStr("w-full rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-xs text-white placeholder:text-white/25 outline-none resize-none focus:border-white/20 focus:bg-white/[0.06]"),
				),
			),

			// Error banner.
			If(parseData.MutationError != "",
				Div(
					ClassStr("mb-4 flex flex-col gap-1 rounded-xl border border-red-500/20 bg-red-500/5 px-3 py-2 text-xs text-red-400"),
					Text(parseUserErrorMessage(parseData.MutationError)),
					renderSupportIDChip(parseUserErrorRequestID(parseData.MutationError)),
				),
			),

			// Action buttons.
			Div(ClassStr("flex items-center justify-end gap-3"),
				Button(
					ClassStr("rounded-xl border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-white/60 hover:bg-white/[0.07] transition-colors cursor-pointer"),
					OnClick(parseCustomers.HandleConfirmCancel),
					Text("Cancel"),
				),
				Button(
					ClassStr(parseButtonClass),
					Disabled(parseData.IsMutationPending || strings.TrimSpace(parseData.ConfirmReason) == ""),
					OnClick(parseCustomers.HandleConfirmSubmit),
					Text(parseButtonLabel),
				),
			),
		),
	)
}

func parseFilterAndSortAdminUsers(parseUsers []adminUserRow, parseQuery adminListQueryState, parseAccessStates map[int64]string) []adminUserRow {
	parseQuery = parseNormalizeAdminListQuery(parseQuery)
	parseFiltered := make([]adminUserRow, 0, len(parseUsers))
	for _, parseUser := range parseUsers {
		parseState := parseUserAccessStateForRow(parseUser.UserID, parseAccessStates)
		if !parseAdminUserMatchesFilter(parseUser, parseState, parseQuery.Filter) {
			continue
		}
		parseHaystack := parseUser.Email + " " + parseUser.DisplayName + " " + parseState
		if !parseAdminMatchesQuery(parseQuery, parseHaystack) {
			continue
		}
		parseFiltered = append(parseFiltered, parseUser)
	}
	sort.SliceStable(parseFiltered, func(parseI, parseJ int) bool {
		parseLeft := parseFiltered[parseI]
		parseRight := parseFiltered[parseJ]
		switch parseQuery.Sort {
		case "email":
			return strings.ToLower(parseDashboardUserLabel(parseLeft.DisplayName, parseLeft.Email)) < strings.ToLower(parseDashboardUserLabel(parseRight.DisplayName, parseRight.Email))
		case "state":
			return parseUserAccessStateForRow(parseLeft.UserID, parseAccessStates) < parseUserAccessStateForRow(parseRight.UserID, parseAccessStates)
		case "conversations":
			return parseLeft.ConversationCount > parseRight.ConversationCount
		case "spend":
			return parseLeft.TotalCostUSD > parseRight.TotalCostUSD
		default:
			return parseLeft.LastSeenAt > parseRight.LastSeenAt
		}
	})
	return parseFiltered
}

func parseAdminUserMatchesFilter(parseUser adminUserRow, parseState, parseFilter string) bool {
	parseFilter = strings.TrimSpace(strings.ToLower(parseFilter))
	switch parseFilter {
	case "", "all":
		return true
	case "active":
		return parseState != "disabled"
	case "disabled", "blocked":
		return parseState == "disabled"
	case "high_spend":
		return parseUser.TotalCostUSD >= 50
	case "stale":
		return strings.TrimSpace(parseUser.LastSeenAt) == ""
	default:
		return strings.Contains(parseState, parseFilter)
	}
}

func parseUserAccessStateForRow(parseUserID int64, parseAccessStates map[int64]string) string {
	if parseAccessStates != nil {
		if parseState := strings.TrimSpace(strings.ToLower(parseAccessStates[parseUserID])); parseState != "" {
			return parseState
		}
	}
	return "active"
}

func parseAdminUserAccessStateForDetail(parseDetail adminUserDetailSnapshot, parseAccessStates map[int64]string) string {
	for _, parseAudit := range parseDetail.AuditLogs {
		parseEvent := strings.TrimSpace(strings.ToLower(parseAudit.EventType))
		if strings.Contains(parseEvent, "admin.user.restore") || strings.Contains(parseEvent, "user.restore") {
			return "active"
		}
		if strings.Contains(parseEvent, "admin.user.disable") || strings.Contains(parseEvent, "user.disable") {
			return "disabled"
		}
	}
	return parseUserAccessStateForRow(parseDetail.UserID, parseAccessStates)
}

func renderAdminUserAccessBadge(parseState string) ui.Node {
	parseState = strings.TrimSpace(strings.ToLower(parseState))
	if parseState == "disabled" {
		return Span(ClassStr("inline-flex rounded-full border border-red-500/30 bg-red-500/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-red-200"), Text("Blocked"))
	}
	return Span(ClassStr("inline-flex rounded-full border border-green-500/25 bg-green-500/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-green-200"), Text("Active"))
}

func parseAdminUserAccessImpactText(parseState string) string {
	if strings.TrimSpace(strings.ToLower(parseState)) == "disabled" {
		return "Blocked account: sign-in and active access are disabled until an operator restores the user."
	}
	return "Active account: the user can sign in and use available workspace, billing, and support surfaces."
}
