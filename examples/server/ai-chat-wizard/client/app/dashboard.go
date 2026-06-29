//go:build js && wasm

package app

import (
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// renderDashboardHome is the top-level admin dashboard entry-point.
// It dispatches to the correct slice renderer based on the current path.
// Shared primitives.

// renderDashboardLoadingState renders a centered loading indicator for slice surfaces.
func renderDashboardLoadingState(parseLabel string) ui.Node {
	return Div(
		ClassStr("flex flex-col items-center justify-center py-20 gap-3 text-white/40"),
		Div(ClassStr("h-6 w-6 animate-spin rounded-full border-2 border-white/20 border-t-white/60")),
		P(ClassStr("text-sm"), Text(parseLabel)),
	)
}

// renderDashboardEmptyState renders a centered empty-state placeholder.
func renderDashboardEmptyState(parseIcon, parseTitle, parseSubtitle string) ui.Node {
	return Div(
		ClassStr("flex flex-col items-center justify-center py-20 gap-2 text-center"),
		Div(ClassStr("text-3xl mb-2"), Text(parseIcon)),
		P(ClassStr("text-sm font-medium text-white/60"), Text(parseTitle)),
		P(ClassStr("text-xs text-white/35 max-w-xs leading-5"), Text(parseSubtitle)),
	)
}

// renderDashboardDeniedBanner renders an access-denied message for gated admin surfaces.
func renderDashboardDeniedBanner() ui.Node {
	return Div(
		ClassStr("rounded-[1.4rem] border border-red-500/20 bg-red-500/5 px-5 py-4 text-sm text-red-400"),
		P(ClassStr("font-medium"), Text("Access denied")),
		P(ClassStr("mt-1 text-xs text-red-400/70"), Text("You do not have permission to view this section.")),
	)
}

// renderDashboardErrorBanner renders a general error notice for failed data fetches.
func renderDashboardErrorBanner(parseErr string) ui.Node {
	parseMessage := parseUserErrorMessage(parseErr)
	if parseMessage == "" {
		parseMessage = parseResolveUserFallbackMessage(userErrorScopeDashboard)
	}
	return Div(
		ID("dashboard-error-banner"),
		ClassStr("rounded-[1.4rem] border border-yellow-500/20 bg-yellow-500/5 px-5 py-4 text-sm text-yellow-400"),
		P(ClassStr("font-medium"), Text("Data unavailable")),
		P(ClassStr("mt-1 text-xs text-yellow-400/70"), Text(parseMessage)),
		renderSupportIDChip(parseUserErrorRequestID(parseErr)),
	)
}

// renderDashboardKPICard renders a single KPI metric card with optional trend indicator.
func renderDashboardKPICard(parseLabel, parseValue, parseSub string) ui.Node {
	return Div(
		ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] px-5 py-4"),
		P(ClassStr("text-[10px] font-medium uppercase tracking-[0.18em] text-white/35"), Text(parseLabel)),
		P(ClassStr("mt-2 text-2xl font-bold text-white"), Text(parseValue)),
		If(parseSub != "",
			P(ClassStr("mt-1 text-xs text-white/40"), Text(parseSub)),
		),
	)
}

// renderDashboardSectionHeader renders a section heading within a slice view.
func renderDashboardSectionHeader(parseTitle string) ui.Node {
	return P(ClassStr("mb-3 mt-6 text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text(parseTitle))
}

// renderDashboardTable renders a simple table with a header row + data rows.
// parseHeaders is the list of column labels; parseRows is each row as a []string.
func renderDashboardTable(parseHeaders []string, parseRows [][]string) ui.Node {
	parseHeaderCells := make([]ui.Node, len(parseHeaders))
	for parseI, parseH := range parseHeaders {
		parseHeaderCells[parseI] = Th(ClassStr("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text(parseH))
	}
	parseDataRowNodes := make([]ui.Node, len(parseRows))
	for parseI, parseCells := range parseRows {
		parseTDs := make([]ui.Node, len(parseCells))
		for parseJ, parseCell := range parseCells {
			parseTDs[parseJ] = Td(ClassStr("px-4 py-2.5 text-xs text-white/70 align-top"), Text(parseCell))
		}
		parseDataRowNodes[parseI] = Tr(ClassStr("border-t border-white/[0.06] hover:bg-white/[0.02]"), Fragment(parseTDs))
	}
	return Div(
		ClassStr("overflow-x-auto rounded-[1.4rem] border border-white/10"),
		Table(
			ClassStr("w-full min-w-[560px] border-collapse"),
			Thead(Tr(ClassStr("bg-white/[0.03]"), Fragment(parseHeaderCells))),
			Tbody(Fragment(parseDataRowNodes)),
		),
	)
}

// Slice header helpers.

// renderDashboardSliceHeader renders the breadcrumb-style heading for a slice view.
func renderDashboardSliceHeader(parseIcon, parseTitle, parseSubtitle string) ui.Node {
	return Div(
		ClassStr("mb-6 flex items-center gap-4"),
		Div(ClassStr("flex h-10 w-10 items-center justify-center rounded-2xl border border-white/10 bg-white/[0.04] text-xl"),
			Text(parseIcon),
		),
		Div(
			P(ClassStr("text-base font-semibold text-white"), Text(parseTitle)),
			P(ClassStr("text-xs text-white/40"), Text(parseSubtitle)),
		),
	)
}

// Business slice.

// renderDashboardBusiness renders the Business analytics slice.
func renderDashboardBusiness(parseIntl i18n.Runtime, parseView appViewState, parseAdminWorkspaces adminWorkspacesController, parseAdminOperations adminOperationsController) ui.Node {
	_ = parseIntl
	parseData := parseView.AdminDashboardData
	if parseData.IsDenied {
		return Fragment(renderDashboardSliceHeader("\U0001f4ca", "Business", "Platform KPIs and top accounts"), renderDashboardDeniedBanner())
	}
	if parseData.IsLoading {
		return renderDashboardLoadingState("Loading business metrics\u2026")
	}
	if parseData.Error != "" {
		return Fragment(renderDashboardSliceHeader("\U0001f4ca", "Business", "Platform KPIs and top accounts"), renderDashboardErrorBanner(parseData.Error))
	}
	if !parseData.HasData {
		return Fragment(renderDashboardSliceHeader("\U0001f4ca", "Business", "Platform KPIs and top accounts"), renderDashboardEmptyState("\U0001f4ca", "No data yet", "Business metrics will appear once the platform has usage."))
	}
	parseSummary := parseData.Summary
	return Fragment(
		renderDashboardSliceHeader("\U0001f4ca", "Business", "Platform KPIs and top accounts"),
		renderDashboardAlertState(parseView, parseAdminOperations),
		renderDashboardSharedRail(adminScopeBilling, parseAdminOperations.Data.BillingQuery, parseAdminOperations),
		renderDashboardSectionHeader("Platform totals"),
		Div(ClassStr("grid grid-cols-2 gap-3 sm:grid-cols-3"),
			renderDashboardKPICard("Total users", formatDashboardInt64(parseSummary.TotalUsers), ""),
			renderDashboardKPICard("Conversations", formatDashboardInt64(parseSummary.TotalConversations), ""),
			renderDashboardKPICard("Messages", formatDashboardInt64(parseSummary.TotalMessages), ""),
		),
		renderDashboardSectionHeader("30-day window"),
		Div(ClassStr("grid grid-cols-2 gap-3 sm:grid-cols-4"),
			renderDashboardKPICard("New users", formatDashboardInt64(parseSummary.WindowNewUsers), ""),
			renderDashboardKPICard("Active users", formatDashboardInt64(parseSummary.WindowActiveUsers), ""),
			renderDashboardKPICard("New chats", formatDashboardInt64(parseSummary.WindowNewConvs), ""),
			renderDashboardKPICard("Total spend", formatCostUSD(parseSummary.WindowTotalCostUSD), ""),
		),
		renderDashboardSectionHeader("Top accounts by spend"),
		renderDashboardTopUsersTable(parseData.TopUsers),
		renderDashboardSectionHeader("Daily usage (30 days)"),
		renderDashboardDailyTable(parseData.DailyUsage),
		renderDashboardBusinessInterventions(parseAdminOperations),
		renderDashboardWorkspacesPanel(parseIntl, parseView, parseAdminWorkspaces),
		renderAdminOperationsConfirmModal(parseAdminOperations),
	)
}

// renderDashboardTopUsersTable renders the top-users-by-spend table.
func renderDashboardTopUsersTable(parseUsers []adminUserRow) ui.Node {
	if len(parseUsers) == 0 {
		return renderDashboardEmptyState("\U0001f465", "No usage data", "No users have usage events in this window.")
	}
	parseRows := make([][]string, len(parseUsers))
	for parseI, parseUser := range parseUsers {
		parseRows[parseI] = []string{
			parseDashboardUserLabel(parseUser.DisplayName, parseUser.Email),
			formatDashboardInt64(parseUser.ConversationCount),
			formatDashboardInt64(parseUser.MessageCount),
			formatCostUSD(parseUser.TotalCostUSD),
			parseDashboardShortAt(parseUser.LastSeenAt),
		}
	}
	return renderDashboardTable(
		[]string{"User", "Convs", "Messages", "Spend", "Last seen"},
		parseRows,
	)
}

// renderDashboardDailyTable renders the daily usage trend table.
func renderDashboardDailyTable(parseDays []adminDailyRow) ui.Node {
	if len(parseDays) == 0 {
		return renderDashboardEmptyState("\U0001f4c5", "No daily data", "Daily usage data will populate after the first billing period.")
	}
	parseRows := make([][]string, len(parseDays))
	for parseI, parseDay := range parseDays {
		parseRows[parseI] = []string{
			parseDashboardFormatDay(parseDay.UsageDay),
			formatDashboardInt64(parseDay.EventCount),
			formatDashboardInt64(parseDay.ActiveUsers),
			formatCostUSD(parseDay.TotalCostUSD),
		}
	}
	return renderDashboardTable(
		[]string{"Day", "Events", "Active users", "Cost"},
		parseRows,
	)
}

// Customers slice.

// renderDashboardCustomers renders the Customers analytics slice.
func renderDashboardCustomers(parseIntl i18n.Runtime, parseView appViewState) ui.Node {
	_ = parseIntl
	parseData := parseView.AdminDashboardData
	if parseData.IsDenied {
		return Fragment(renderDashboardSliceHeader("\U0001f465", "Customers", "Recent users and account context"), renderDashboardDeniedBanner())
	}
	if parseData.IsLoading {
		return renderDashboardLoadingState("Loading customer data\u2026")
	}
	if parseData.Error != "" {
		return Fragment(renderDashboardSliceHeader("\U0001f465", "Customers", "Recent users and account context"), renderDashboardErrorBanner(parseData.Error))
	}
	if !parseData.HasData {
		return Fragment(renderDashboardSliceHeader("\U0001f465", "Customers", "Recent users and account context"), renderDashboardEmptyState("\U0001f465", "No customers yet", "User records will appear once accounts are created."))
	}
	return Fragment(
		renderDashboardSliceHeader("\U0001f465", "Customers", "Recent users and account context"),
		renderDashboardSummaryCard2("Total users", formatDashboardInt64(parseData.Summary.TotalUsers)),
		renderDashboardSectionHeader("Recent users"),
		renderDashboardRecentUsersTable(parseData.RecentUsers),
	)
}

// renderDashboardSummaryCard2 renders an inline metric pill for slice headers.
func renderDashboardSummaryCard2(parseLabel, parseValue string) ui.Node {
	return Div(
		ClassStr("mb-5 inline-flex items-center gap-2 rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-2"),
		P(ClassStr("text-[10px] font-medium uppercase tracking-wide text-white/35"), Text(parseLabel)),
		P(ClassStr("text-sm font-bold text-white"), Text(parseValue)),
	)
}

// renderDashboardRecentUsersTable renders the recent-users list table.
func renderDashboardRecentUsersTable(parseUsers []adminUserRow) ui.Node {
	if len(parseUsers) == 0 {
		return renderDashboardEmptyState("\U0001f465", "No recent users", "User records will appear once accounts are created.")
	}
	parseRows := make([][]string, len(parseUsers))
	for parseI, parseUser := range parseUsers {
		parseRows[parseI] = []string{
			parseDashboardUserLabel(parseUser.DisplayName, parseUser.Email),
			formatDashboardInt64(parseUser.ConversationCount),
			formatDashboardInt64(parseUser.MessageCount),
			formatCostUSD(parseUser.TotalCostUSD),
			parseDashboardShortAt(parseUser.CreatedAt),
			parseDashboardShortAt(parseUser.LastSeenAt),
		}
	}
	return renderDashboardTable(
		[]string{"User", "Convs", "Msgs", "Spend", "Joined", "Last seen"},
		parseRows,
	)
}

// Chats slice.

// renderDashboardChats renders the Chats analytics slice.
func renderDashboardChats(parseIntl i18n.Runtime, parseView appViewState, parseAdminOperations adminOperationsController) ui.Node {
	_ = parseIntl
	parseData := parseView.AdminDashboardData
	if parseData.IsDenied {
		return Fragment(renderDashboardSliceHeader("\U0001f4ac", "Chats", "Conversation health and recent threads"), renderDashboardDeniedBanner())
	}
	if parseData.IsLoading {
		return renderDashboardLoadingState("Loading conversation data\u2026")
	}
	if parseData.Error != "" {
		return Fragment(renderDashboardSliceHeader("\U0001f4ac", "Chats", "Conversation health and recent threads"), renderDashboardErrorBanner(parseData.Error))
	}
	if !parseData.HasData {
		return Fragment(renderDashboardSliceHeader("\U0001f4ac", "Chats", "Conversation health and recent threads"), renderDashboardEmptyState("\U0001f4ac", "No conversations yet", "Thread data will appear once users start chatting."))
	}
	return Fragment(
		renderDashboardSliceHeader("\U0001f4ac", "Chats", "Conversation health and recent threads"),
		renderDashboardAlertState(parseView, parseAdminOperations),
		Div(ClassStr("mb-5 grid grid-cols-2 gap-3 sm:grid-cols-3"),
			renderDashboardKPICard("Total convs", formatDashboardInt64(parseData.Summary.TotalConversations), "platform-wide"),
			renderDashboardKPICard("New this period", formatDashboardInt64(parseData.Summary.WindowNewConvs), "30-day window"),
			renderDashboardKPICard("New messages", formatDashboardInt64(parseData.Summary.WindowNewMessages), "30-day window"),
		),
		renderDashboardSectionHeader("Recent conversations"),
		renderDashboardRecentConvsTable(parseData.RecentConvs),
		renderDashboardChatsDrilldowns(parseData, parseAdminOperations),
		renderAdminOperationsConfirmModal(parseAdminOperations),
	)
}

// renderDashboardRecentConvsTable renders the recent-conversations list table.
func renderDashboardRecentConvsTable(parseConvs []adminConvRow) ui.Node {
	if len(parseConvs) == 0 {
		return renderDashboardEmptyState("\U0001f4ac", "No conversations yet", "Thread data will appear once users start chatting.")
	}
	parseRows := make([][]string, len(parseConvs))
	for parseI, parseConv := range parseConvs {
		parsePreview := parseConv.Preview
		if len(parsePreview) > 60 {
			parsePreview = parsePreview[:60] + "\u2026"
		}
		parseRows[parseI] = []string{
			parseDashboardUserLabel(parseConv.DisplayName, parseConv.Email),
			parsePreview,
			formatDashboardInt64(parseConv.MessageCount),
			formatCostUSD(parseConv.TotalCostUSD),
			parseDashboardShortAt(parseConv.LastActivityAt),
		}
	}
	return renderDashboardTable(
		[]string{"User", "Preview", "Msgs", "Cost", "Last activity"},
		parseRows,
	)
}

// Providers slice.

// renderDashboardProviders renders the Providers health slice.
func renderDashboardProviders(parseIntl i18n.Runtime, parseView appViewState, parseAdminOperations adminOperationsController) ui.Node {
	_ = parseIntl
	parseData := parseView.AdminDashboardData
	if parseData.IsDenied {
		return Fragment(renderDashboardSliceHeader("\u26a1", "Providers", "Model routing, health, and cost"), renderDashboardDeniedBanner())
	}
	if parseData.IsLoading {
		return renderDashboardLoadingState("Loading provider data\u2026")
	}
	if parseData.Error != "" {
		return Fragment(renderDashboardSliceHeader("\u26a1", "Providers", "Model routing, health, and cost"), renderDashboardErrorBanner(parseData.Error))
	}
	if !parseData.HasData {
		return Fragment(renderDashboardSliceHeader("\u26a1", "Providers", "Model routing, health, and cost"), renderDashboardEmptyState("\u26a1", "No provider data", "Provider snapshots will appear once a model request is made."))
	}
	return Fragment(
		renderDashboardSliceHeader("\u26a1", "Providers", "Model routing, health, and cost"),
		renderDashboardAlertState(parseView, parseAdminOperations),
		renderDashboardSectionHeader("Provider status"),
		renderDashboardProvidersTable(parseData.ProviderSnaps),
		renderDashboardProviderControls(parseData, parseAdminOperations),
		renderAdminOperationsConfirmModal(parseAdminOperations),
	)
}

// renderDashboardProvidersTable renders the provider health status table.
func renderDashboardProvidersTable(parseSnaps []adminProviderRow) ui.Node {
	if len(parseSnaps) == 0 {
		return renderDashboardEmptyState("\u26a1", "No providers configured", "Configure at least one AI provider in the settings to see data here.")
	}
	parseRows := make([][]string, len(parseSnaps))
	for parseI, parseSnap := range parseSnaps {
		parseLabel := parseSnap.Label
		if parseLabel == "" {
			parseLabel = parseSnap.ProviderID
		}
		parseStatus := "unavailable"
		if parseSnap.IsAvailable {
			parseStatus = "available"
		} else if !parseSnap.IsConfigured {
			parseStatus = "not configured"
		}
		parseErrCol := parseSnap.LastError
		if len(parseErrCol) > 40 {
			parseErrCol = parseErrCol[:40] + "\u2026"
		}
		if parseErrCol == "" {
			parseErrCol = "\u2014"
		}
		parseRows[parseI] = []string{
			parseLabel,
			parseStatus,
			parseDashboardBool(parseSnap.IsConfigured),
			formatDashboardInt64(parseSnap.LastLatencyMs) + " ms",
			formatDashboardInt64(parseSnap.RequestCount),
			parseErrCol,
		}
	}
	return renderDashboardTable(
		[]string{"Provider", "Status", "Auth", "Latency", "Requests", "Last error"},
		parseRows,
	)
}

// Ops slice.

// renderDashboardOps renders the Ops / platform health slice plus the superuser server-tools surface.
func renderDashboardOps(parseIntl i18n.Runtime, parseView appViewState, parseAdminOperations adminOperationsController) ui.Node {
	_ = parseIntl
	parseData := parseView.AdminDashboardData
	if parseData.IsDenied {
		return Fragment(renderDashboardSliceHeader("\U0001f6e0\ufe0f", "Ops", "Platform health, incidents, and experiments"), renderDashboardDeniedBanner())
	}
	if parseData.IsLoading {
		return renderDashboardLoadingState("Loading ops data\u2026")
	}
	if parseData.Error != "" {
		return Fragment(renderDashboardSliceHeader("\U0001f6e0\ufe0f", "Ops", "Platform health, incidents, and experiments"), renderDashboardErrorBanner(parseData.Error))
	}
	if !parseData.HasData {
		return Fragment(renderDashboardSliceHeader("\U0001f6e0\ufe0f", "Ops", "Platform health, incidents, and experiments"), renderDashboardEmptyState("\U0001f6e0\ufe0f", "No ops data", "Platform health data will appear once the service has run."))
	}
	parseSummary := parseData.Summary
	return Fragment(
		renderDashboardSliceHeader("\U0001f6e0\ufe0f", "Ops", "Platform health, incidents, and experiments"),
		renderDashboardAlertState(parseView, parseAdminOperations),
		renderDashboardSectionHeader("Platform health"),
		Div(ClassStr("grid grid-cols-2 gap-3 sm:grid-cols-4"),
			renderDashboardKPICard("Open incidents", formatDashboardInt64(parseSummary.OpenIncidents), ""),
			renderDashboardKPICard("Support tickets", formatDashboardInt64(parseSummary.OpenSupportTickets), ""),
			renderDashboardKPICard("Active experiments", formatDashboardInt64(parseSummary.ActiveExperiments), ""),
			renderDashboardKPICard("Failed events (30d)", formatDashboardInt64(parseSummary.WindowFailedEvents), ""),
		),
		renderDashboardSectionHeader("Daily usage (30 days)"),
		renderDashboardDailyTable(parseData.DailyUsage),
		renderDashboardSupportTriage(parseAdminOperations),
		renderDashboardIncidentExperimentControls(parseAdminOperations),
		renderDashboardSuperuserSurface(parseView, parseAdminOperations),
		renderDashboardWorkspaceAdminSurface(parseAdminOperations),
		renderDashboardServerToolsPanel(parseView),
		renderAdminOperationsConfirmModal(parseAdminOperations),
	)
}

// Formatting helpers.

// formatDashboardInt formats an int as a string for dashboard display, returning an em dash for zero.
func formatDashboardInt(parseN int) string {
	if parseN == 0 {
		return "\u2014"
	}
	return itoa(parseN)
}

// formatDashboardInt64 formats an int64 for dashboard display, returning an em dash for zero.
func formatDashboardInt64(parseN int64) string {
	if parseN == 0 {
		return "\u2014"
	}
	return strconv.FormatInt(parseN, 10)
}

// parseDashboardCostLabel returns a formatted cost string for dashboard metric cards.
func parseDashboardCostLabel(parseHasCost bool, parseCost float64) string {
	if !parseHasCost {
		return "\u2014"
	}
	return formatCostUSD(parseCost)
}

// parseDashboardUserLabel formats a display name + email label for table cells.
func parseDashboardUserLabel(parseDisplayName, parseEmail string) string {
	if parseDisplayName != "" && parseDisplayName != parseEmail {
		return parseDisplayName + " \u00b7 " + parseEmail
	}
	return parseEmail
}

// parseDashboardShortAt trims an ISO timestamp to just the date component.
func parseDashboardShortAt(parseAt string) string {
	if parseAt == "" {
		return "\u2014"
	}
	if parseIdx := strings.IndexByte(parseAt, 'T'); parseIdx > 0 {
		return parseAt[:parseIdx]
	}
	if len(parseAt) > 10 {
		return parseAt[:10]
	}
	return parseAt
}

// parseDashboardFormatDay formats a usage_day field (typically YYYY-MM-DD).
func parseDashboardFormatDay(parseDay string) string {
	if parseDay == "" {
		return "\u2014"
	}
	return parseDay
}

// parseDashboardBool formats a bool for table display.
func parseDashboardBool(parseBool bool) string {
	if parseBool {
		return "yes"
	}
	return "no"
}

// itoa converts an int to its decimal string representation without importing strconv.
func itoa(parseN int) string {
	if parseN == 0 {
		return "0"
	}
	isNeg := parseN < 0
	if isNeg {
		parseN = -parseN
	}
	parseBuf := make([]byte, 0, 20)
	for parseN > 0 {
		parseBuf = append([]byte{byte('0' + parseN%10)}, parseBuf...)
		parseN /= 10
	}
	if isNeg {
		parseBuf = append([]byte{'-'}, parseBuf...)
	}
	return string(parseBuf)
}

// isDashboardRoute returns true when the given path is any admin dashboard route.
func isDashboardRoute(parsePath string) bool {
	switch parsePath {
	case chatRouteDashboardHome, chatRouteDashboardBusiness, chatRouteDashboardCustomers,
		chatRouteDashboardChats, chatRouteDashboardProviders, chatRouteDashboardOps:
		return true
	default:
		return false
	}
}
