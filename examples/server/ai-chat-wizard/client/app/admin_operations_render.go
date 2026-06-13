//go:build js && wasm

package app

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

const adminOperationsPageSize = 8

func renderDashboardAlertState(parseView appViewState, parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	if parseOps.Error != "" {
		return renderDashboardErrorBanner(parseOps.Error)
	}
	if parseOps.MutationSuccess != "" {
		return Div(
			Class("mb-4 rounded-[1.4rem] border border-green-500/20 bg-green-500/5 px-5 py-4"),
			P(Class("text-sm font-medium text-green-300"), Text(parseOps.MutationSuccess)),
			P(Class("mt-1 text-xs text-green-400/50"), Text("The list state is persistent, so refresh or back/forward keeps the operator context.")),
		)
	}
	if parseOps.IsLoading {
		return Div(
			Class("mb-4 rounded-[1.4rem] border border-white/10 bg-white/[0.03] px-5 py-3 text-xs text-white/45"),
			Text("Refreshing admin slices..."),
		)
	}
	if parseView.IsSuperuser {
		return nil
	}
	return Div(
		Class("mb-4 rounded-[1.4rem] border border-blue-400/15 bg-blue-400/5 px-5 py-4"),
		P(Class("text-sm font-medium text-blue-200"), Text("Workspace-admin scope")),
		P(Class("mt-1 text-xs leading-5 text-blue-100/50"), Text("This view hides platform-wide pricing controls and exposes only workspace members, API keys, webhooks, invitations, billing summary, usage, and audit logs.")),
	)
}

func renderDashboardSharedRail(parseScope string, parseQuery adminListQueryState, parseAdminOperations adminOperationsController) ui.Node {
	return Div(
		Class("mb-5 grid gap-3 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 md:grid-cols-[1fr_auto_auto_auto]"),
		Div(
			Class("min-w-0"),
			P(Class("mb-1 text-[10px] font-medium uppercase tracking-[0.18em] text-white/35"), Text("Search")),
			Input(
				Type("text"),
				Placeholder("Search this admin slice..."),
				Value(parseQuery.Search),
				Data(dataAdminScope, parseScope),
				OnInput(parseAdminOperations.HandleQuerySearch),
				Class("w-full rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-white placeholder:text-white/25 outline-none focus:border-white/20"),
			),
		),
		renderDashboardRailSelect("Filter", parseScope, dataAdminFilter, parseQuery.Filter, []adminSelectOption{
			{ID: "all", Label: "All"},
			{ID: "active", Label: "Active"},
			{ID: "open", Label: "Open"},
			{ID: "suspended", Label: "Suspended"},
			{ID: "failed", Label: "Failed"},
		}, parseAdminOperations.HandleQueryFilter),
		renderDashboardRailSelect("Sort", parseScope, dataAdminSort, parseQuery.Sort, []adminSelectOption{
			{ID: "updated", Label: "Updated"},
			{ID: "priority", Label: "Priority"},
			{ID: "status", Label: "Status"},
			{ID: "plan", Label: "Plan"},
			{ID: "cost", Label: "Cost"},
		}, parseAdminOperations.HandleQuerySort),
		Div(
			Class("min-w-[9rem]"),
			P(Class("mb-1 text-[10px] font-medium uppercase tracking-[0.18em] text-white/35"), Text("Range")),
			Div(Class("rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-white/70"), Text("Last 30 days")),
		),
	)
}

type adminSelectOption struct {
	ID    string
	Label string
}

func renderDashboardRailSelect(parseLabel, parseScope, parseDataKey, parseValue string, parseOptions []adminSelectOption, parseHandler ui.Handler) ui.Node {
	parseNodes := make([]ui.Node, len(parseOptions))
	for parseI, parseOption := range parseOptions {
		parseNodes[parseI] = Option(Value(parseOption.ID), Selected(parseOption.ID == parseValue), Text(parseOption.Label))
	}
	return Div(
		Class("min-w-[9rem]"),
		P(Class("mb-1 text-[10px] font-medium uppercase tracking-[0.18em] text-white/35"), Text(parseLabel)),
		Select(
			Data(dataAdminScope, parseScope),
			Data(parseDataKey, parseValue),
			Value(parseValue),
			OnInput(parseHandler),
			Class("w-full rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-sm text-white outline-none focus:border-white/20"),
			Fragment(parseNodes),
		),
	)
}

func renderDashboardJourneyOverview(parseView appViewState, parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	return Div(
		Class("mb-6 rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4 sm:p-5"),
		P(Class("mb-3 text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Admin journey")),
		Div(Class("grid gap-3 md:grid-cols-4"),
			renderDashboardSummaryCard("Workspaces", formatDashboardInt(len(parseOps.Workspaces))),
			renderDashboardSummaryCard("Support queue", formatDashboardInt(len(parseOps.SupportTickets))),
			renderDashboardSummaryCard("Incidents", formatDashboardInt(len(parseOps.Incidents))),
			renderDashboardSummaryCard("Experiments", formatDashboardInt(len(parseOps.Experiments))),
		),
		Div(Class("mt-4 grid gap-3 lg:grid-cols-2"),
			renderDashboardWorkspaceAdminSurface(parseAdminOperations),
			If(parseView.IsSuperuser, renderDashboardSuperuserSurface(parseView, parseAdminOperations)),
		),
	)
}

func renderDashboardBusinessInterventions(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	return Fragment(
		renderDashboardSectionHeader("Billing interventions"),
		Div(Class("grid gap-3 lg:grid-cols-3"),
			renderAdminControlCard("Failed-payment review", "Open dunning and failed-payment queues before changing access.", formatDashboardInt(len(parseOps.FailedPayments)), "Review queue"),
			renderAdminControlCard("Quota review", "Inspect soft/hard limits, enforcement mode, and usage-based entitlement issues.", formatDashboardInt(len(parseOps.BillingQuotas)), "Review quotas"),
			renderAdminControlCard("Override troubleshooting", "Use customer detail to compare overrides, invoices, and recent usage before applying a fix.", formatDashboardInt(len(parseOps.BillingOverages)), "Open account"),
		),
		renderDashboardSectionHeader("Failed-payment queue"),
		renderAdminFailedPaymentsTable(parseOps.FailedPayments),
		renderDashboardSectionHeader("Dunning queue"),
		renderAdminDunningTable(parseOps.DunningEvents),
		renderAdminBillingTables(parseAdminOperations),
	)
}

func renderAdminFailedPaymentsTable(parseRowsIn []adminBillingEventRow) ui.Node {
	if len(parseRowsIn) == 0 {
		return renderDashboardEmptyState("$", "No failed payments", "Failed-payment events from the business queue will appear here.")
	}
	parseRows := make([][]string, len(parseRowsIn))
	for parseI, parseRow := range parseRowsIn {
		parseRows[parseI] = []string{
			formatDashboardInt64(parseRow.CustomerID),
			formatDashboardInt64(parseRow.InvoiceID),
			parseFallbackText(parseRow.EventType, "-"),
			parseTruncateAdminText(parseRow.EventSummary, 72),
			parseDashboardShortAt(parseRow.CreatedAt),
		}
	}
	return renderDashboardTable([]string{"Customer", "Invoice", "Event", "Summary", "Created"}, parseRows)
}

func renderAdminDunningTable(parseRowsIn []adminDunningEventRow) ui.Node {
	if len(parseRowsIn) == 0 {
		return renderDashboardEmptyState("D", "No dunning events", "Open dunning rows from the business queue will appear here.")
	}
	parseRows := make([][]string, len(parseRowsIn))
	for parseI, parseRow := range parseRowsIn {
		parseRows[parseI] = []string{
			formatDashboardInt64(parseRow.CustomerID),
			formatDashboardInt64(parseRow.InvoiceID),
			parseFallbackText(parseRow.Status, "-"),
			formatDashboardInt64(parseRow.AttemptCount),
			parseTruncateAdminText(parseRow.FailureReason, 72),
			parseDashboardShortAt(parseRow.NextAttemptAt),
		}
	}
	return renderDashboardTable([]string{"Customer", "Invoice", "Status", "Attempts", "Reason", "Next"}, parseRows)
}

func renderAdminBillingTables(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	parseOverageRows := make([][]string, 0, len(parseOps.BillingOverages))
	for _, parseRow := range parseOps.BillingOverages {
		if !parseAdminMatchesQuery(parseOps.BillingQuery, parseRow.PlanCode+" "+parseRow.MeterKey) {
			continue
		}
		parseOverageRows = append(parseOverageRows, []string{
			parseRow.PlanCode,
			parseRow.MeterKey,
			formatDashboardInt64(parseRow.IncludedUnits),
			formatDashboardInt64(parseRow.HardLimitUnits),
			parseCentsLabel(parseRow.OveragePriceCents),
			parseDashboardShortAt(parseRow.UpdatedAt),
		})
	}
	return Fragment(
		renderDashboardSharedRail(adminScopeBilling, parseOps.BillingQuery, parseAdminOperations),
		If(len(parseOverageRows) == 0,
			renderDashboardEmptyState("$", "No billing controls match", "Adjust the billing search or filter to review overages, quotas, and upgrade triggers."),
		),
		If(len(parseOverageRows) > 0,
			renderDashboardTable([]string{"Plan", "Meter", "Included", "Hard limit", "Overage", "Updated"}, parseOverageRows),
		),
	)
}

func renderDashboardWorkspacesList(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	parseRows := make([][]string, 0, len(parseOps.Workspaces))
	for _, parseWorkspace := range parseOps.Workspaces {
		if !parseAdminMatchesQuery(parseOps.WorkspaceQuery, parseWorkspace.Name+" "+parseWorkspace.Slug+" "+parseWorkspace.Status+" "+parseWorkspace.PlanCode) {
			continue
		}
		parseRows = append(parseRows, []string{
			formatDashboardInt64(parseWorkspace.WorkspaceID),
			parseWorkspace.Name,
			parseWorkspace.Status,
			parseWorkspace.PlanCode,
			formatDashboardInt64(parseWorkspace.OwnerUserID),
			parseDashboardShortAt(parseWorkspace.UpdatedAt),
		})
	}
	parseRows, parsePage, parseTotalPages := parsePaginateAdminRows(parseRows, parseOps.WorkspaceQuery.Page)
	return Fragment(
		renderDashboardSectionHeader("Workspaces"),
		renderDashboardSharedRail(adminScopeWorkspaces, parseOps.WorkspaceQuery, parseAdminOperations),
		If(len(parseRows) == 0,
			renderDashboardEmptyState("W", "No workspaces match", "Search by workspace name, slug, plan, or status."),
		),
		If(len(parseRows) > 0,
			renderDashboardTable([]string{"ID", "Workspace", "Status", "Plan", "Owner", "Updated"}, parseRows),
		),
		renderAdminPagination(adminScopeWorkspaces, parsePage, parseTotalPages, parseAdminOperations),
	)
}

func renderDashboardChatsDrilldowns(parseData adminDashboardData, parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	parseFailureRows := make([][]string, 0, len(parseOps.ChatFailedReplies))
	for _, parseUsage := range parseOps.ChatFailedReplies {
		parseFailureRows = append(parseFailureRows, []string{
			parseFallbackText(parseUsage.ProviderID, "-"),
			parseFallbackText(parseUsage.ModelID, "-"),
			parseFallbackText(parseUsage.Status, "-"),
			formatCostUSD(parseUsage.TotalCostUSD),
			parseDashboardShortAt(parseUsage.CreatedAt),
		})
	}
	parseThreadRows := [][]string{}
	for _, parseConv := range parseOps.ChatHighCostThreads {
		parseThreadRows = append(parseThreadRows, []string{
			parseDashboardUserLabel(parseConv.DisplayName, parseConv.Email),
			parseFallbackText(parseConv.PublicID, formatDashboardInt64(parseConv.ConversationID)),
			formatDashboardInt64(parseConv.MessageCount),
			formatCostUSD(parseConv.TotalCostUSD),
			parseDashboardShortAt(parseConv.LastActivityAt),
		})
	}
	if len(parseThreadRows) == 0 {
		for _, parseConv := range parseData.RecentConvs {
			parseThreadRows = append(parseThreadRows, []string{
				parseDashboardUserLabel(parseConv.DisplayName, parseConv.Email),
				parseFallbackText(parseConv.PublicID, formatDashboardInt64(parseConv.ConversationID)),
				formatDashboardInt64(parseConv.MessageCount),
				formatCostUSD(parseConv.TotalCostUSD),
				parseDashboardShortAt(parseConv.LastActivityAt),
			})
		}
	}
	return Fragment(
		renderDashboardSectionHeader("Reply-quality drill-downs"),
		Div(Class("grid gap-3 lg:grid-cols-3"),
			renderAdminControlCard("First-chat conversion", "Open recent onboarding and activation rows from the chats drill-down RPC.", formatDashboardInt64(parseData.Summary.WindowNewConvs), "Inspect funnel"),
			renderAdminControlCard("Reply failures", "Use failed usage rows to inspect provider request IDs, model, trace, and message context.", formatDashboardInt(len(parseOps.ChatFailedReplies)), "Inspect failures"),
			renderAdminControlCard("Thread cost anomalies", "Open high-cost threads before changing prompt defaults or workflow publishing.", formatDashboardInt(len(parseOps.ChatHighCostThreads)), "Inspect threads"),
		),
		If(len(parseFailureRows) > 0,
			renderDashboardTable([]string{"Provider", "Model", "Status", "Cost", "Created"}, parseFailureRows),
		),
		If(len(parseFailureRows) == 0,
			renderDashboardEmptyState("F", "No failed reply rows", "Failed reply anomaly rows appear here when the anomaly RPC returns failures."),
		),
		If(len(parseThreadRows) > 0,
			Fragment(renderDashboardSectionHeader("Thread inspector entry points"), renderDashboardTable([]string{"User", "Thread", "Msgs", "Cost", "Last activity"}, parseThreadRows)),
		),
		renderAdminOperationsConfirmModal(parseAdminOperations),
	)
}

func renderDashboardProviderControls(parseData adminDashboardData, parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	parseRows := make([][]string, 0, len(parseOps.CostGuardrails))
	for _, parseGuardrail := range parseOps.CostGuardrails {
		parseRows = append(parseRows, []string{
			formatDashboardInt64(parseGuardrail.WorkspaceID),
			parseGuardrail.GuardrailKey,
			parseCentsLabel(parseGuardrail.DailyBudgetCents),
			parseCentsLabel(parseGuardrail.MaxCostPerRequestCents),
			parseGuardrail.ActionMode,
			parseDashboardShortAt(parseGuardrail.UpdatedAt),
		})
	}
	parseFallbackRows := make([][]string, 0, len(parseOps.ProviderFallbacks))
	for _, parseEvent := range parseOps.ProviderFallbacks {
		parseFallbackRows = append(parseFallbackRows, []string{
			formatDashboardInt64(parseEvent.WorkspaceID),
			parseFallbackText(parseEvent.EventType, "-"),
			parseTruncateAdminText(parseEvent.Summary, 90),
			parseDashboardShortAt(parseEvent.CreatedAt),
		})
	}
	return Fragment(
		renderDashboardSectionHeader("Provider control entry points"),
		Div(Class("grid gap-3 lg:grid-cols-3"),
			renderAdminControlCard("Fallback routing", "Review provider health before changing default or fallback models.", formatDashboardInt(len(parseOps.ProviderFallbacks)), "Open routing"),
			renderAdminControlCard("Model visibility", "Hide a model only after checking recent failed and slow usage rows.", formatDashboardInt64(parseData.Summary.WindowUsageEvents), "Review models"),
			renderAdminControlCard("Guardrails", "Budget controls show blast-radius context by workspace before enforcing blocks.", formatDashboardInt(len(parseRows)), "Review limits"),
		),
		If(len(parseFallbackRows) > 0,
			renderDashboardTable([]string{"Workspace", "Event", "Summary", "Created"}, parseFallbackRows),
		),
		If(len(parseRows) > 0,
			Fragment(renderDashboardSectionHeader("Cost guardrails"), renderDashboardTable([]string{"Workspace", "Guardrail", "Daily", "Per request", "Action", "Updated"}, parseRows)),
		),
		If(len(parseRows) == 0 && len(parseFallbackRows) == 0,
			renderDashboardEmptyState("G", "No provider controls configured", "Fallback audit events and workspace cost guardrails will appear here when configured."),
		),
	)
}

func renderDashboardSupportTriage(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	parseRows := make([]ui.Node, 0, len(parseOps.SupportTickets))
	for _, parseTicket := range parseOps.SupportTickets {
		if !parseAdminMatchesQuery(parseOps.SupportQuery, parseTicket.Subject+" "+parseTicket.Status+" "+parseTicket.Priority+" "+parseTicket.TicketKey) {
			continue
		}
		parseRows = append(parseRows, renderSupportTicketRow(parseTicket, parseAdminOperations))
	}
	parseRows, parsePage, parseTotalPages := parsePaginateAdminNodes(parseRows, parseOps.SupportQuery.Page)
	return Fragment(
		renderDashboardSectionHeader("Support triage"),
		renderDashboardSharedRail(adminScopeSupport, parseOps.SupportQuery, parseAdminOperations),
		Div(Class("grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(20rem,0.85fr)]"),
			Div(Class("min-w-0 overflow-hidden rounded-[1.4rem] border border-white/10"),
				Table(Class("w-full min-w-[520px] border-collapse"),
					Thead(Tr(Class("bg-white/[0.03]"),
						Th(Class("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Ticket")),
						Th(Class("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Priority")),
						Th(Class("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Status")),
						Th(Class("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Updated")),
					)),
					Tbody(Fragment(parseRows)),
				),
			),
			renderSupportDetailPanel(parseAdminOperations),
		),
		renderAdminPagination(adminScopeSupport, parsePage, parseTotalPages, parseAdminOperations),
	)
}

func renderSupportTicketRow(parseTicket adminSupportTicketRow, parseAdminOperations adminOperationsController) ui.Node {
	parseSelected := parseAdminOperations.Data.SelectedTicketID == parseTicket.TicketID
	return Tr(
		Class(ClassNames("border-t border-white/[0.06] cursor-pointer transition-colors", When(parseSelected, "bg-white/[0.07]"), When(!parseSelected, "hover:bg-white/[0.03]"))),
		Data(dataAdminTicketID, strconv.FormatInt(parseTicket.TicketID, 10)),
		OnClick(parseAdminOperations.HandleSelectTicket),
		Td(Class("px-4 py-2.5 align-top"),
			P(Class("text-xs font-medium text-white/80"), Text(parseFallbackText(parseTicket.Subject, parseTicket.TicketKey))),
			P(Class("mt-0.5 text-[10px] text-white/35"), Text(fmt.Sprintf("workspace %d · user %d", parseTicket.WorkspaceID, parseTicket.UserID))),
		),
		Td(Class("px-4 py-2.5 text-xs text-white/60 align-top"), Text(parseFallbackText(parseTicket.Priority, "-"))),
		Td(Class("px-4 py-2.5 text-xs text-white/60 align-top"), Text(parseFallbackText(parseTicket.Status, "-"))),
		Td(Class("px-4 py-2.5 text-xs text-white/40 align-top"), Text(parseDashboardShortAt(parseTicket.UpdatedAt))),
	)
}

func renderSupportDetailPanel(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	if parseOps.SelectedTicketID <= 0 {
		return Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.02] p-5"),
			P(Class("text-sm font-medium text-white/70"), Text("Ticket detail")),
			P(Class("mt-1 text-xs leading-5 text-white/40"), Text("Select a ticket to review customer context, internal notes, escalation controls, and linked account actions.")),
		)
	}
	if parseOps.IsLoadingDetail {
		return Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.02] p-5"), Text("Loading ticket detail..."))
	}
	parseDetail := parseOps.SupportDetail
	if !parseDetail.HasData {
		return Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.02] p-5"), Text("Ticket detail unavailable."))
	}
	parseMessages := make([]ui.Node, 0, len(parseDetail.Messages))
	for _, parseMessage := range parseDetail.Messages {
		parseMessages = append(parseMessages, Div(Class("rounded-xl border border-white/8 bg-white/[0.03] px-3 py-2"),
			Div(Class("mb-1 flex items-center gap-2 text-[10px] uppercase tracking-wide text-white/35"),
				Span(Text(parseFallbackText(parseMessage.MessageType, "message"))),
				If(parseMessage.IsInternal, Span(Class("rounded-full bg-yellow-400/10 px-1.5 py-0.5 text-yellow-200/60"), Text("internal"))),
				Span(Text(parseDashboardShortAt(parseMessage.CreatedAt))),
			),
			P(Class("text-xs leading-5 text-white/65"), Text(parseTruncateAdminText(parseMessage.Body, 180))),
		))
	}
	return Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.02] p-5"),
		P(Class("text-sm font-semibold text-white"), Text(parseFallbackText(parseDetail.Ticket.Subject, parseDetail.Ticket.TicketKey))),
		P(Class("mt-1 text-xs text-white/40"), Text(fmt.Sprintf("Ticket %s · workspace %d · user %d", parseDetail.Ticket.TicketKey, parseDetail.Ticket.WorkspaceID, parseDetail.Ticket.UserID))),
		Div(Class("mt-4 flex flex-wrap gap-2"),
			Button(Class("rounded-xl border border-blue-400/25 bg-blue-400/8 px-3 py-1.5 text-xs text-blue-200"),
				Data(dataAdminAction, "support-note"),
				Data(dataAdminTargetID, strconv.FormatInt(parseDetail.Ticket.TicketID, 10)),
				Data(dataAdminScope, adminScopeSupport),
				OnClick(parseAdminOperations.HandleConfirmStart),
				Text("Add internal note"),
			),
			Button(Class("rounded-xl border border-yellow-400/25 bg-yellow-400/8 px-3 py-1.5 text-xs text-yellow-200"),
				Data(dataAdminAction, "support-escalate"),
				Data(dataAdminTargetID, strconv.FormatInt(parseDetail.Ticket.TicketID, 10)),
				Data(dataAdminScope, adminScopeSupport),
				OnClick(parseAdminOperations.HandleConfirmStart),
				Text("Escalate"),
			),
		),
		renderDashboardSectionHeader("Messages"),
		If(len(parseMessages) == 0, P(Class("text-xs text-white/35"), Text("No messages on this ticket."))),
		If(len(parseMessages) > 0, Div(Class("space-y-2"), Fragment(parseMessages))),
		If(len(parseDetail.AccountActions) > 0,
			Fragment(renderDashboardSectionHeader("Linked account actions"), renderAdminAuditTable(parseDetail.AccountActions)),
		),
	)
}

func renderDashboardIncidentExperimentControls(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	parseIncidentRows := make([]ui.Node, 0, len(parseOps.Incidents))
	for _, parseIncident := range parseOps.Incidents {
		if !parseAdminMatchesQuery(parseOps.IncidentQuery, parseIncident.Title+" "+parseIncident.Status+" "+parseIncident.Severity+" "+parseIncident.IncidentKey) {
			continue
		}
		parseIncidentRows = append(parseIncidentRows, Tr(Class("border-t border-white/[0.06]"),
			Td(Class("px-4 py-2.5 align-top"), P(Class("text-xs font-medium text-white/80"), Text(parseFallbackText(parseIncident.Title, parseIncident.IncidentKey))), P(Class("mt-0.5 text-[10px] text-white/35"), Text(parseTruncateAdminText(parseIncident.Summary, 90)))),
			Td(Class("px-4 py-2.5 text-xs text-white/60 align-top"), Text(parseFallbackText(parseIncident.Severity, "-"))),
			Td(Class("px-4 py-2.5 text-xs text-white/60 align-top"), Text(parseFallbackText(parseIncident.Status, "-"))),
			Td(Class("px-4 py-2.5 align-top"),
				Button(Class("rounded-lg border border-white/10 bg-white/[0.04] px-2 py-1 text-[10px] text-white/55"),
					Data(dataAdminAction, "incident-update"),
					Data(dataAdminTargetID, strconv.FormatInt(parseIncident.IncidentID, 10)),
					Data(dataAdminScope, adminScopeIncidents),
					OnClick(parseAdminOperations.HandleConfirmStart),
					Text("Move to monitoring"),
				),
			),
		))
	}
	parseIncidentRows, parsePage, parseTotalPages := parsePaginateAdminNodes(parseIncidentRows, parseOps.IncidentQuery.Page)
	return Fragment(
		renderDashboardSectionHeader("Incidents, flags, and experiments"),
		renderDashboardSharedRail(adminScopeIncidents, parseOps.IncidentQuery, parseAdminOperations),
		If(len(parseIncidentRows) > 0,
			Div(Class("overflow-x-auto rounded-[1.4rem] border border-white/10"),
				Table(Class("w-full min-w-[620px] border-collapse"),
					Thead(Tr(Class("bg-white/[0.03]"),
						Th(Class("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Incident")),
						Th(Class("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Severity")),
						Th(Class("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Status")),
						Th(Class("px-4 py-2 text-left text-[10px] font-medium uppercase tracking-wide text-white/40"), Text("Control")),
					)),
					Tbody(Fragment(parseIncidentRows)),
				),
			),
		),
		If(len(parseIncidentRows) == 0,
			renderDashboardEmptyState("I", "No incidents match", "Incident controls appear here with blast-radius copy and explicit state-change actions."),
		),
		renderAdminPagination(adminScopeIncidents, parsePage, parseTotalPages, parseAdminOperations),
		renderFeatureFlagControls(parseAdminOperations),
		renderExperimentControls(parseAdminOperations),
	)
}

func renderFeatureFlagControls(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	parseNodes := make([]ui.Node, 0, len(parseOps.FeatureFlags))
	for _, parseFlag := range parseOps.FeatureFlags {
		parseState := "disabled"
		if parseFlag.IsEnabled {
			parseState = "enabled"
		}
		parseNodes = append(parseNodes, Div(Class("rounded-xl border border-white/8 bg-white/[0.03] p-3"),
			P(Class("text-xs font-semibold text-white/75"), Text(parseFlag.FlagKey)),
			P(Class("mt-1 text-xs leading-5 text-white/40"), Text(parseTruncateAdminText(parseFlag.Description, 110))),
			P(Class("mt-2 text-[10px] uppercase tracking-wide text-white/35"), Text(fmt.Sprintf("%s · %d%% rollout", parseState, parseFlag.RolloutPercent))),
			Button(Class("mt-3 rounded-lg border border-blue-400/25 bg-blue-400/8 px-2.5 py-1 text-[10px] text-blue-200"),
				Data(dataAdminAction, "feature-toggle"),
				Data(dataAdminTargetKey, parseFlag.FlagKey),
				Data(dataAdminScope, adminScopeOps),
				OnClick(parseAdminOperations.HandleConfirmStart),
				Text("Enable to 100%"),
			),
		))
	}
	return Fragment(
		renderDashboardSectionHeader("Feature flags"),
		Div(Class("grid gap-3 md:grid-cols-3"),
			If(len(parseNodes) == 0, renderAdminControlCard("Feature flags", "No feature flags returned for this query.", "-", "No action")),
			If(len(parseNodes) > 0, Fragment(parseNodes)),
		),
	)
}

func renderExperimentControls(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	parseNodes := make([]ui.Node, 0, len(parseOps.Experiments))
	for _, parseExperiment := range parseOps.Experiments {
		parseNodes = append(parseNodes, Div(Class("rounded-xl border border-white/8 bg-white/[0.03] p-3"),
			P(Class("text-xs font-semibold text-white/75"), Text(parseFallbackText(parseExperiment.Name, parseExperiment.ExperimentKey))),
			P(Class("mt-1 text-[10px] uppercase tracking-wide text-white/35"), Text(parseFallbackText(parseExperiment.Status, "unknown"))),
			Button(Class("mt-3 rounded-lg border border-red-400/25 bg-red-400/8 px-2.5 py-1 text-[10px] text-red-200"),
				Data(dataAdminAction, "experiment-rollback"),
				Data(dataAdminTargetKey, parseExperiment.ExperimentKey),
				Data(dataAdminScope, adminScopeOps),
				OnClick(parseAdminOperations.HandleConfirmStart),
				Text("Rollback"),
			),
		))
	}
	return Div(Class("mt-3 grid gap-3 md:grid-cols-3"),
		If(len(parseNodes) == 0, renderAdminControlCard("Experiment controls", "No active experiment rows returned for this query.", "-", "No action")),
		If(len(parseNodes) > 0, Fragment(parseNodes)),
	)
}

func renderDashboardSuperuserSurface(parseView appViewState, parseAdminOperations adminOperationsController) ui.Node {
	if !parseView.IsSuperuser {
		return nil
	}
	parseOps := parseAdminOperations.Data
	return Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4"),
		P(Class("text-sm font-semibold text-white"), Text("Superuser dashboard")),
		P(Class("mt-1 text-xs leading-5 text-white/40"), Text("Pricing controls, workspace state, support tickets, incidents, and cost guardrails are loaded from the superuser slice RPC.")),
		Div(Class("mt-3 grid grid-cols-2 gap-2"),
			renderAdminMiniCard("Pricing rows", formatDashboardInt(len(parseOps.BillingOverages)+len(parseOps.BillingQuotas)+len(parseOps.BillingTriggers))),
			renderAdminMiniCard("Guardrails", formatDashboardInt(len(parseOps.CostGuardrails))),
			renderAdminMiniCard("Workspaces", formatDashboardInt(len(parseOps.Workspaces))),
			renderAdminMiniCard("Support", formatDashboardInt(len(parseOps.SupportTickets))),
		),
	)
}

func renderDashboardWorkspaceAdminSurface(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	return Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4"),
		P(Class("text-sm font-semibold text-white"), Text("Workspace admin")),
		P(Class("mt-1 text-xs leading-5 text-white/40"), Text("API keys, webhook endpoints, invitations, usage, billing summary, and audit logs are visible in workspace scope.")),
		Div(Class("mt-3 grid grid-cols-2 gap-2"),
			renderAdminMiniCard("API keys", formatDashboardInt(len(parseOps.WorkspaceAPIKeys))),
			renderAdminMiniCard("Webhooks", formatDashboardInt(len(parseOps.WorkspaceWebhooks))),
			renderAdminMiniCard("Invites", formatDashboardInt(len(parseOps.WorkspaceInvites))),
			renderAdminMiniCard("Audit rows", formatDashboardInt(len(parseOps.WorkspaceAuditLogs))),
		),
		If(parseOps.WorkspaceBilling.CustomerCount+parseOps.WorkspaceBilling.ActiveSubscriptionCount+parseOps.WorkspaceBilling.OpenInvoiceCount+parseOps.WorkspaceBilling.DunningEventCount > 0,
			Div(Class("mt-3 grid grid-cols-2 gap-2"),
				renderAdminMiniCard("Customers", formatDashboardInt64(parseOps.WorkspaceBilling.CustomerCount)),
				renderAdminMiniCard("Open invoices", formatDashboardInt64(parseOps.WorkspaceBilling.OpenInvoiceCount)),
			),
		),
	)
}

func renderAdminOperationsConfirmModal(parseAdminOperations adminOperationsController) ui.Node {
	parseOps := parseAdminOperations.Data
	if parseOps.ConfirmAction == "" {
		return nil
	}
	parseTitle, parseBody, parseButtonLabel := parseAdminOperationConfirmCopy(parseOps)
	return Div(Class("fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"),
		Div(Class("w-full max-w-md rounded-[1.6rem] border border-white/10 bg-[#13131e] p-6 shadow-2xl"),
			P(Class("text-base font-semibold text-white"), Text(parseTitle)),
			P(Class("mt-2 rounded-xl border border-white/8 bg-white/[0.03] px-4 py-3 text-xs leading-5 text-white/60"), Text(parseBody)),
			If(parseOps.ConfirmAction == "support-note",
				Div(Class("mt-4"),
					P(Class("mb-1.5 text-xs font-medium text-white/50"), Text("Internal note")),
					Tag("textarea", Rows(4), Value(parseOps.SupportNoteBody), OnInput(parseAdminOperations.HandleSupportNote), Placeholder("Add operator-only context..."), Class("w-full resize-none rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-xs text-white placeholder:text-white/25 outline-none")),
				),
			),
			Div(Class("mt-4"),
				P(Class("mb-1.5 text-xs font-medium text-white/50"), Text("Reason")),
				Tag("textarea", Rows(3), Value(parseOps.ConfirmReason), OnInput(parseAdminOperations.HandleConfirmReason), Placeholder("Why is this action needed?"), Class("w-full resize-none rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-xs text-white placeholder:text-white/25 outline-none")),
			),
			If(parseOps.MutationError != "", Div(Class("mt-3 text-xs text-red-300"), Text(parseUserErrorMessage(parseOps.MutationError)), renderSupportIDChip(parseUserErrorRequestID(parseOps.MutationError)))),
			Div(Class("mt-5 flex justify-end gap-3"),
				Button(Class("rounded-xl border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-white/60"), OnClick(parseAdminOperations.HandleConfirmCancel), Text("Cancel")),
				Button(Class("rounded-xl border border-red-400/30 bg-red-400/10 px-4 py-2 text-sm text-red-200 disabled:opacity-50"),
					Disabled(parseOps.IsMutationPending || strings.TrimSpace(parseOps.ConfirmReason) == "" || (parseOps.ConfirmAction == "support-note" && strings.TrimSpace(parseOps.SupportNoteBody) == "")),
					OnClick(parseAdminOperations.HandleConfirmSubmit),
					If(parseOps.IsMutationPending, Text("Working...")),
					If(!parseOps.IsMutationPending, Text(parseButtonLabel)),
				),
			),
		),
	)
}

func parseAdminOperationConfirmCopy(parseOps adminOperationsData) (string, string, string) {
	switch parseOps.ConfirmAction {
	case "support-note":
		return "Add internal note", "This note is operator-only. It links to the selected ticket and becomes part of the support triage history.", "Add note"
	case "support-escalate":
		return "Escalate support ticket", "This changes support priority and status. Confirm the customer impact and include the escalation reason.", "Escalate"
	case "feature-toggle":
		return "Enable feature flag", "This rolls the selected flag to 100 percent. Review audience and blast radius before confirming.", "Enable flag"
	case "experiment-rollback":
		return "Rollback experiment", "This asks the server to rollback the selected experiment and preserve an auditable reason.", "Rollback"
	case "incident-update":
		return "Update incident", "This publishes a monitoring update for the incident. Confirm user impact and recovery state before applying.", "Update incident"
	default:
		return "Confirm admin action", "This admin action records an audit event and may affect user or workspace access.", "Confirm"
	}
}

func renderAdminControlCard(parseTitle, parseBody, parseMetric, parseAction string) ui.Node {
	return Div(Class("rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4"),
		Div(Class("flex items-start justify-between gap-3"),
			Div(Class("min-w-0"),
				P(Class("text-sm font-semibold text-white"), Text(parseTitle)),
				P(Class("mt-1 text-xs leading-5 text-white/40"), Text(parseBody)),
			),
			Span(Class("rounded-full border border-white/10 bg-white/[0.04] px-2 py-1 text-xs text-white/50"), Text(parseMetric)),
		),
		P(Class("mt-3 text-[10px] font-medium uppercase tracking-wide text-white/30"), Text(parseAction)),
	)
}

func renderAdminPagination(parseScope string, parsePage, parseTotalPages int, parseAdminOperations adminOperationsController) ui.Node {
	if parseTotalPages <= 1 {
		return nil
	}
	return Div(Class("mt-3 flex items-center justify-between text-xs text-white/40"),
		Span(Text(fmt.Sprintf("Page %d of %d", parsePage+1, parseTotalPages))),
		Div(Class("flex gap-2"),
			Button(Class("rounded-lg border border-white/10 px-3 py-1.5 disabled:opacity-30"), Data(dataAdminScope, parseScope), Disabled(parsePage <= 0), OnClick(parseAdminOperations.HandlePrevPage), Text("Prev")),
			Button(Class("rounded-lg border border-white/10 px-3 py-1.5 disabled:opacity-30"), Data(dataAdminScope, parseScope), Disabled(parsePage >= parseTotalPages-1), OnClick(parseAdminOperations.HandleNextPage), Text("Next")),
		),
	)
}

func parseAdminMatchesQuery(parseQuery adminListQueryState, parseText string) bool {
	parseNeedle := strings.ToLower(strings.TrimSpace(parseQuery.Search))
	if parseNeedle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(parseText), parseNeedle)
}

func parsePaginateAdminRows(parseRows [][]string, parsePage int) ([][]string, int, int) {
	parseTotalPages := (len(parseRows) + adminOperationsPageSize - 1) / adminOperationsPageSize
	if parseTotalPages < 1 {
		parseTotalPages = 1
	}
	if parsePage < 0 {
		parsePage = 0
	}
	if parsePage >= parseTotalPages {
		parsePage = parseTotalPages - 1
	}
	parseStart := parsePage * adminOperationsPageSize
	parseEnd := parseStart + adminOperationsPageSize
	if parseEnd > len(parseRows) {
		parseEnd = len(parseRows)
	}
	if parseStart > len(parseRows) {
		parseStart = len(parseRows)
	}
	return parseRows[parseStart:parseEnd], parsePage, parseTotalPages
}

func parsePaginateAdminNodes(parseRows []ui.Node, parsePage int) ([]ui.Node, int, int) {
	parseTotalPages := (len(parseRows) + adminOperationsPageSize - 1) / adminOperationsPageSize
	if parseTotalPages < 1 {
		parseTotalPages = 1
	}
	if parsePage < 0 {
		parsePage = 0
	}
	if parsePage >= parseTotalPages {
		parsePage = parseTotalPages - 1
	}
	parseStart := parsePage * adminOperationsPageSize
	parseEnd := parseStart + adminOperationsPageSize
	if parseEnd > len(parseRows) {
		parseEnd = len(parseRows)
	}
	if parseStart > len(parseRows) {
		parseStart = len(parseRows)
	}
	return parseRows[parseStart:parseEnd], parsePage, parseTotalPages
}

func parseFallbackText(parseValue, parseFallback string) string {
	parseValue = strings.TrimSpace(parseValue)
	if parseValue == "" {
		return parseFallback
	}
	return parseValue
}

func parseTruncateAdminText(parseValue string, parseLimit int) string {
	parseValue = strings.TrimSpace(parseValue)
	if parseLimit <= 0 || len(parseValue) <= parseLimit {
		return parseValue
	}
	return parseValue[:parseLimit] + "..."
}

func parseCentsLabel(parseCents int64) string {
	return formatCostUSD(float64(parseCents) / 100)
}
