//go:build js && wasm

// dashboard_shell.go owns the dashboard entry shell, role banner, and summary tiles.
package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

func renderDashboardHome(parseIntl i18n.Runtime, parseView appViewState, parseOpenAdminDashboard ui.Handler, parseAdminCustomers adminCustomersController, parseAdminWorkspaces adminWorkspacesController, parseAdminOperations adminOperationsController) ui.Node {
	_ = parseOpenAdminDashboard
	return Div(
		ClassStr("flex h-full min-h-0 flex-1 flex-col overflow-hidden"),
		renderDashboardTopBar(parseIntl, parseView),
		renderDashboardBody(parseIntl, parseView, parseAdminCustomers, parseAdminWorkspaces, parseAdminOperations),
	)
}

// renderDashboardBody dispatches to the correct slice renderer.
func renderDashboardBody(parseIntl i18n.Runtime, parseView appViewState, parseAdminCustomers adminCustomersController, parseAdminWorkspaces adminWorkspacesController, parseAdminOperations adminOperationsController) ui.Node {
	// Pick the slice that matches the current route so each dashboard page can stay focused on one job.
	switch parseView.CurrentPath {
	case chatRouteDashboardBusiness:
		return renderDashboardSliceWrap(renderDashboardBusiness(parseIntl, parseView, parseAdminWorkspaces, parseAdminOperations))
	case chatRouteDashboardCustomers:
		return renderDashboardSliceWrap(renderDashboardCustomersEnhanced(parseIntl, parseView, parseAdminCustomers, parseAdminOperations))
	case chatRouteDashboardChats:
		return renderDashboardSliceWrap(renderDashboardChats(parseIntl, parseView, parseAdminOperations))
	case chatRouteDashboardProviders:
		return renderDashboardSliceWrap(renderDashboardProviders(parseIntl, parseView, parseAdminOperations))
	case chatRouteDashboardOps:
		return renderDashboardSliceWrap(renderDashboardOps(parseIntl, parseView, parseAdminOperations))
	default:
		return renderDashboardSliceWrap(renderDashboardHomeBody(parseIntl, parseView, parseAdminOperations))
	}
}

// renderDashboardSliceWrap is a shared scroll container for all slice views.
func renderDashboardSliceWrap(parseContent ui.Node) ui.Node {
	return Div(
		ClassStr("chat-scrollbar flex-1 overflow-y-auto"),
		Div(
			ClassStr("mx-auto w-full max-w-5xl px-5 py-6"),
			parseContent,
		),
	)
}

// renderDashboardTopBar renders the top navigation bar for the admin dashboard.
func renderDashboardTopBar(parseIntl i18n.Runtime, parseView appViewState) ui.Node {
	_ = parseIntl
	parseIsSlice := parseView.CurrentPath != chatRouteDashboardHome
	return Div(
		ClassStr("flex items-center gap-3 border-b border-white/10 bg-[#13131e] px-5 py-3"),
		If(parseIsSlice,
			A(
				Href(chatRouteDashboardHome),
				OnClick(parseLandingNavigateHandler(chatRouteDashboardHome)),
				ClassStr("flex items-center gap-2 rounded-xl border border-white/10 bg-white/[0.04] px-3 py-1.5 text-xs text-white/60 no-underline transition-colors hover:bg-white/8 hover:text-white/90"),
				Span(Text("\u2190")),
				Span(Text("Dashboard")),
			),
		),
		If(!parseIsSlice,
			A(
				Href(chatRouteRoot),
				OnClick(parseLandingNavigateHandler(chatRouteRoot)),
				ClassStr("flex items-center gap-2 rounded-xl border border-white/10 bg-white/[0.04] px-3 py-1.5 text-xs text-white/60 no-underline transition-colors hover:bg-white/8 hover:text-white/90"),
				Span(Text("\u2190")),
				Span(Text("Back to workspace")),
			),
		),
		Div(ClassStr("mx-2 h-4 w-px bg-white/10")),
		P(ClassStr("text-sm font-semibold text-white"), Text("Admin Dashboard")),
		Div(ClassStr("ml-auto flex items-center gap-2"),
			Div(ClassStr("rounded-full border border-white/10 bg-white/[0.04] px-2.5 py-1 text-[10px] font-medium uppercase tracking-wide text-white/40"),
				Text("RelayDesk"),
			),
		),
	)
}

// â”€â”€â”€ Home body â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// renderDashboardHomeBody keeps the home tiles, role banner, and account summary together so the landing view reads as one overview surface.
func renderDashboardHomeBody(parseIntl i18n.Runtime, parseView appViewState, parseAdminOperations adminOperationsController) ui.Node {
	return Fragment(
		renderDashboardRoleBanner(parseIntl, parseView),
		renderDashboardSliceTiles(parseIntl, parseView),
		renderDashboardJourneyOverview(parseView, parseAdminOperations),
		renderDashboardAccountSummary(parseIntl, parseView),
	)
}

// renderDashboardRoleBanner renders the role context banner at the top of the dashboard.
func renderDashboardRoleBanner(parseIntl i18n.Runtime, parseView appViewState) ui.Node {
	_ = parseIntl
	return Div(
		ClassStr("mb-6 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
		Div(ClassStr("flex items-center gap-3"),
			Div(ClassStr("flex h-9 w-9 items-center justify-center rounded-xl border border-white/10 bg-white/6 text-base"),
				Text("\U0001f6e1\ufe0f"),
			),
			Div(ClassStr("min-w-0"),
				P(ClassStr("text-sm font-semibold text-white"),
					Text(parseDashboardRoleLabel(parseView)),
				),
				P(ClassStr("mt-0.5 text-xs text-white/45"),
					Text(parseDashboardRoleDescription(parseView)),
				),
			),
		),
	)
}

// parseDashboardRoleLabel returns the role heading for the dashboard banner.
func parseDashboardRoleLabel(parseView appViewState) string {
	if parseView.CanAccessAdmin {
		return "Admin view"
	}
	return "Workspace view"
}

// parseDashboardRoleDescription returns the role sub-line for the dashboard banner.
func parseDashboardRoleDescription(parseView appViewState) string {
	if parseView.CanAccessAdmin {
		return "Check Business for failed payments and churn, Customers for billing state, Chats for reply quality, Providers for cost and health, and Ops for incidents."
	}
	return "You can view workspace usage and manage your own account settings."
}

// dashboardSlice holds display metadata for a single admin dashboard slice tile.
type dashboardSlice struct {
	Label        string
	Icon         string
	Route        string
	Subtitle     string
	IsReady      bool
	Availability string // explicit state label shown when the tile is not ready; e.g. "Limited", "Superuser only", "Admin only"
}

// parseDashboardSlices returns the ordered list of admin dashboard tiles.
func parseDashboardSlices() []dashboardSlice {
	return []dashboardSlice{
		{
			Label:    "Business",
			Icon:     "\U0001f4ca",
			Route:    chatRouteDashboardBusiness,
			Subtitle: "Active subscriptions, failed payments, conversion, churn, and top revenue accounts",
			IsReady:  true,
		},
		{
			Label:    "Customers",
			Icon:     "\U0001f465",
			Route:    chatRouteDashboardCustomers,
			Subtitle: "User and workspace search, billing state, recent activity, and disable or suspend controls",
			IsReady:  true,
		},
		{
			Label:    "Chats",
			Icon:     "\U0001f4ac",
			Route:    chatRouteDashboardChats,
			Subtitle: "First-chat conversion, reply quality, latency, failures, and onboarding template state",
			IsReady:  true,
		},
		{
			Label:    "Providers",
			Icon:     "\u26a1",
			Route:    chatRouteDashboardProviders,
			Subtitle: "Provider and model health, cost tables, routing and fallback state, and guardrail controls",
			IsReady:  true,
		},
		{
			Label:    "Ops",
			Icon:     "\U0001f6e0\ufe0f",
			Route:    chatRouteDashboardOps,
			Subtitle: "Server and gRPC health, background jobs, incident log, audit feed, and SLO tracking",
			IsReady:  true,
		},
	}
}

// renderDashboardSliceTiles renders the grid of navigation tiles for the five dashboard slices.
func renderDashboardSliceTiles(parseIntl i18n.Runtime, parseView appViewState) ui.Node {
	_ = parseIntl
	parseSlices := parseDashboardSlices()
	return Div(
		ClassStr("mb-6"),
		P(ClassStr("mb-3 text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Admin surfaces")),
		Div(
			ClassStr("grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3"),
			Fragment(Map(parseSlices, func(parseSlice dashboardSlice) ui.Node {
				return renderDashboardSliceTile(parseSlice, parseView)
			})),
		),
	)
}

// renderDashboardSliceTile renders a single admin dashboard slice navigation tile.
func renderDashboardSliceTile(parseSlice dashboardSlice, parseView appViewState) ui.Node {
	_ = parseView
	parseTileClass := ClassNames(
		"block rounded-[1.4rem] border p-4 transition-colors no-underline",
		When(parseSlice.IsReady, "border-white/10 bg-white/[0.03] hover:bg-white/[0.06] cursor-pointer"),
		When(!parseSlice.IsReady, "border-white/[0.06] bg-white/[0.02] opacity-60 pointer-events-none"),
	)
	parseInner := Fragment(
		Div(ClassStr("mb-3 text-2xl"), Text(parseSlice.Icon)),
		P(ClassStr("text-sm font-semibold text-white"), Text(parseSlice.Label)),
		P(ClassStr("mt-1 text-xs leading-5 text-white/45"), Text(parseSlice.Subtitle)),
		If(!parseSlice.IsReady,
			renderDashboardAvailabilityBadge(parseSlice.Availability),
		),
	)
	if parseSlice.IsReady {
		return A(
			Href(parseSlice.Route),
			OnClick(parseLandingNavigateHandler(parseSlice.Route)),
			ClassStr(ClassNames(parseTileClass, "block no-underline")),
			parseInner,
		)
	}
	return Div(ClassStr(parseTileClass), parseInner)
}

// renderDashboardAvailabilityBadge renders a small availability-state label for dashboard slice tiles that are not yet ready.
func renderDashboardAvailabilityBadge(parseAvailability string) ui.Node {
	parseLabel := parseAvailability
	if parseLabel == "" {
		parseLabel = "Not enabled"
	}
	return Div(
		ClassStr("mt-2 inline-flex items-center rounded-full border border-white/10 bg-white/[0.04] px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-white/30"),
		Text(parseLabel),
	)
}

// renderDashboardAccountSummary renders a compact account cost and session summary block.
func renderDashboardAccountSummary(parseIntl i18n.Runtime, parseView appViewState) ui.Node {
	_ = parseIntl
	parseHasCost := parseView.AccountCostSummary.HasAnyExactCosts && parseView.AccountCostSummary.TotalCost > 0
	parseFreshnessLabel := "Store unavailable — showing defaults"
	if parseHasCost {
		parseFreshnessLabel = "Live provider snapshot"
	}
	return Div(
		ClassStr("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
		Div(
			ClassStr("mb-3 flex items-baseline justify-between gap-2"),
			P(ClassStr("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Your account this period")),
			Span(ClassStr("text-[10px] text-white/25"), Text(parseFreshnessLabel)),
		),
		Div(ClassStr("grid grid-cols-2 gap-3 sm:grid-cols-4"),
			renderDashboardSummaryCard("Chats", formatDashboardInt(parseView.AccountCostSummary.ThreadCount)),
			renderDashboardSummaryCard("Total spend", parseDashboardCostLabel(parseHasCost, parseView.AccountCostSummary.TotalCost)),
			renderDashboardSummaryCard("Model cost", parseDashboardCostLabel(parseHasCost, parseView.AccountCostSummary.UsageCost)),
			renderDashboardSummaryCard("Service premium", parseDashboardCostLabel(parseHasCost, parseView.AccountCostSummary.PremiumCost)),
		),
	)
}

// renderDashboardSummaryCard renders a single labeled metric card for the account summary grid.
func renderDashboardSummaryCard(parseLabel, parseValue string) ui.Node {
	return Div(
		ClassStr("rounded-xl border border-white/8 bg-white/[0.03] px-4 py-3"),
		P(ClassStr("text-[10px] font-medium uppercase tracking-wide text-white/35"), Text(parseLabel)),
		P(ClassStr("mt-1.5 text-base font-semibold text-white"), Text(parseValue)),
	)
}
