//go:build js && wasm && admin_workspaces

package app

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/i18n"
	"github.com/monstercameron/GoWebComponents/ui"
)

const adminWorkspacesPageSize = 10

// renderDashboardWorkspacesPanel renders the Workspaces admin panel with a
// filter bar, paginated workspace list sourced from the admin dashboard data,
// an expandable detail panel, and a suspend/restore confirmation modal.
func renderDashboardWorkspacesPanel(parseIntl i18n.Runtime, parseView appViewState, parseWS adminWorkspacesController) ui.Node {
	_ = parseIntl
	parseData := parseView.AdminDashboardData
	if parseData.IsDenied {
		return renderDashboardDeniedBanner()
	}
	if parseData.IsLoading {
		return renderDashboardLoadingState("Loading workspace data\u2026")
	}
	if parseData.Error != "" {
		return renderDashboardErrorBanner(parseData.Error)
	}

	// Build the workspace list from the dashboard recent conversations by collecting
	// unique workspace contexts — the dashboard data carries WorkspaceEntry slices
	// embedded in the summary. We synthesise a light list from adminDashboardData.
	// Since GetAdminDashboard does not return a dedicated workspace list, we show
	// a filter-able static label and drive detail-fetch on row click.
	parseAllWS := parseView.AdminWorkspacesPreview
	parseQuery := strings.ToLower(strings.TrimSpace(parseWS.Data.FilterQuery))
	parseFiltered := parseAllWS
	if parseQuery != "" {
		parseFiltered = make([]adminWorkspacePreviewRow, 0, len(parseAllWS))
		for _, parseRow := range parseAllWS {
			if strings.Contains(strings.ToLower(parseRow.Name), parseQuery) ||
				strings.Contains(strings.ToLower(parseRow.Slug), parseQuery) ||
				strings.Contains(strings.ToLower(parseRow.Status), parseQuery) {
				parseFiltered = append(parseFiltered, parseRow)
			}
		}
	}

	// Paginate.
	parseTotalWS := len(parseFiltered)
	parseTotalPages := (parseTotalWS + adminWorkspacesPageSize - 1) / adminWorkspacesPageSize
	if parseTotalPages < 1 {
		parseTotalPages = 1
	}
	parsePage := parseWS.Data.CurrentPage
	if parsePage >= parseTotalPages {
		parsePage = parseTotalPages - 1
	}
	if parsePage < 0 {
		parsePage = 0
	}
	parsePageStart := parsePage * adminWorkspacesPageSize
	parsePageEnd := parsePageStart + adminWorkspacesPageSize
	if parsePageEnd > parseTotalWS {
		parsePageEnd = parseTotalWS
	}
	parsePageWS := parseFiltered[parsePageStart:parsePageEnd]
	isDetailOpen := parseWS.Data.SelectedWorkspaceID > 0

	return Fragment(
		renderDashboardSectionHeader("Workspaces"),
		renderAdminWSFilterBar(parseWS, parseTotalWS),
		Div(
			ClassNames(
				"flex flex-col gap-5",
				When(isDetailOpen, "lg:flex-row"),
			),
			Div(
				ClassNames(
					"min-w-0",
					When(isDetailOpen, "lg:w-[420px] lg:shrink-0"),
					When(!isDetailOpen, "w-full"),
				),
				renderAdminWSListTable(parsePageWS, parseWS),
				renderAdminWSPagination(parsePage, parseTotalPages, parseTotalWS, parseWS),
			),
			If(isDetailOpen,
				renderAdminWSDetailPanel(parseWS),
			),
		),
		If(parseWS.Data.ConfirmAction != "",
			renderAdminWSConfirmModal(parseWS),
		),
	)
}

// renderAdminWSFilterBar renders the workspace filter input with result count.
func renderAdminWSFilterBar(parseWS adminWorkspacesController, parseTotalFiltered int) ui.Node {
	return Div(
		Class("mb-4 flex items-center gap-3"),
		Div(
			Class("relative flex-1"),
			Input(
				Class("w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 pr-8 text-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"),
				Type("text"),
				Placeholder("Filter workspaces by name, slug, or status\u2026"),
				Value(parseWS.Data.FilterQuery),
				OnInput(parseWS.HandleFilter),
			),
		),
		Span(
			Class("shrink-0 text-xs text-gray-500 dark:text-gray-400"),
			Text(fmt.Sprintf("%d workspace(s)", parseTotalFiltered)),
		),
	)
}

// renderAdminWSListTable renders the paginated workspace list.
func renderAdminWSListTable(parseRows []adminWorkspacePreviewRow, parseWS adminWorkspacesController) ui.Node {
	if len(parseRows) == 0 {
		return renderDashboardEmptyState("\U0001f3e2", "No workspaces found", "No workspaces match the current filter.")
	}
	parseHeaderCells := []ui.Node{
		Th(Class("px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400"), Text("Name")),
		Th(Class("px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400"), Text("Slug")),
		Th(Class("px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400"), Text("Plan")),
		Th(Class("px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400"), Text("Status")),
	}
	parseBodyRows := make([]ui.Node, len(parseRows))
	for parseI, parseRow := range parseRows {
		isSelected := parseWS.Data.SelectedWorkspaceID == parseRow.WorkspaceID
		parseBodyRows[parseI] = Tr(
			ClassNames(
				"cursor-pointer transition-colors",
				When(isSelected, "bg-blue-50 dark:bg-blue-900/20 font-medium"),
				When(!isSelected, "hover:bg-gray-50 dark:hover:bg-gray-800/40"),
			),
			Data(dataAdminWorkspaceID, strconv.FormatInt(parseRow.WorkspaceID, 10)),
			OnClick(parseWS.HandleSelectWS),
			Td(Class("px-3 py-2 text-sm"), Text(parseRow.Name)),
			Td(Class("px-3 py-2 text-sm font-mono text-gray-600 dark:text-gray-400"), Text(parseRow.Slug)),
			Td(Class("px-3 py-2 text-sm"), Text(parseRow.PlanCode)),
			Td(Class("px-3 py-2 text-sm"),
				renderAdminWSStatusBadge(parseRow.Status),
			),
		)
	}
	return Div(
		Class("overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700"),
		Tag("table",
			Class("w-full border-collapse text-left"),
			Tag("thead",
				Tr(Class("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
					parseHeaderCells...,
				),
			),
			Tag("tbody",
				Class("divide-y divide-gray-100 dark:divide-gray-800"),
				parseBodyRows...,
			),
		),
	)
}

// renderAdminWSStatusBadge renders a coloured plan/status badge.
func renderAdminWSStatusBadge(parseStatus string) ui.Node {
	parseLower := strings.ToLower(parseStatus)
	parseCls := "inline-flex items-center rounded px-2 py-0.5 text-xs font-medium "
	switch parseLower {
	case "active":
		parseCls += "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300"
	case "suspended":
		parseCls += "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300"
	case "pending":
		parseCls += "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300"
	default:
		parseCls += "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
	}
	return Span(Class(parseCls), Text(parseStatus))
}

// renderAdminWSPagination renders prev/next pagination controls.
func renderAdminWSPagination(parsePage, parseTotalPages, parseTotalWS int, parseWS adminWorkspacesController) ui.Node {
	if parseTotalPages <= 1 {
		return nil
	}
	return Div(
		Class("mt-3 flex items-center justify-between gap-3"),
		Button(
			ClassNames(
				"rounded px-3 py-1 text-xs",
				When(parsePage <= 0, "cursor-not-allowed opacity-40 bg-gray-100 dark:bg-gray-800 text-gray-400"),
				When(parsePage > 0, "bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700"),
			),
			Disabled(parsePage <= 0),
			OnClick(parseWS.HandlePrevPage),
			Text("Previous"),
		),
		Span(
			Class("text-xs text-gray-500 dark:text-gray-400"),
			Text(fmt.Sprintf("Page %d of %d \u00b7 %d total", parsePage+1, parseTotalPages, parseTotalWS)),
		),
		Button(
			ClassNames(
				"rounded px-3 py-1 text-xs",
				When(parsePage >= parseTotalPages-1, "cursor-not-allowed opacity-40 bg-gray-100 dark:bg-gray-800 text-gray-400"),
				When(parsePage < parseTotalPages-1, "bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700"),
			),
			Disabled(parsePage >= parseTotalPages-1),
			OnClick(parseWS.HandleNextPage),
			Text("Next"),
		),
	)
}

// renderAdminWSDetailPanel renders the right-side detail panel for one workspace.
func renderAdminWSDetailPanel(parseWS adminWorkspacesController) ui.Node {
	parseSnap := parseWS.Data.WorkspaceDetail
	return Div(
		Class("flex-1 min-w-0 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-5 relative"),
		// Close button.
		Button(
			Class("absolute right-4 top-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-lg leading-none"),
			Data(dataAdminWorkspaceID, "0"),
			OnClick(parseWS.HandleSelectWS),
			Text("\u00d7"),
		),
		If(parseWS.Data.IsLoadingDetail,
			P(Class("text-sm text-gray-500 dark:text-gray-400"), Text("Loading workspace detail\u2026")),
		),
		If(!parseWS.Data.IsLoadingDetail && parseSnap.HasData,
			Fragment(
				renderAdminWSDetailHeader(parseSnap),
				renderAdminWSActionBand(parseWS, parseSnap),
				renderAdminWSMembersTable(parseSnap.Members),
				renderAdminWSAPIKeysTable(parseSnap.APIKeys),
				renderAdminWSWebhooksTable(parseSnap.Webhooks),
				renderAdminWSAuditTable(parseSnap.AuditLogs),
			),
		),
		If(!parseWS.Data.IsLoadingDetail && !parseSnap.HasData,
			P(Class("text-sm text-gray-500 dark:text-gray-400"), Text("Select a workspace to view detail.")),
		),
	)
}

// renderAdminWSDetailHeader renders the workspace name, slug, plan, and status.
func renderAdminWSDetailHeader(parseSnap adminWorkspaceDetailSnapshot) ui.Node {
	return Div(
		Class("mb-4 pr-8"),
		H3(Class("text-base font-semibold text-gray-900 dark:text-gray-100"), Text(parseSnap.Name)),
		P(
			Class("text-xs text-gray-500 dark:text-gray-400 font-mono mt-0.5"),
			Text(parseSnap.Slug),
		),
		Div(
			Class("mt-2 flex flex-wrap gap-2"),
			renderAdminWSStatusBadge(parseSnap.Status),
			Span(
				Class("inline-flex items-center rounded px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300"),
				Text(parseSnap.PlanCode),
			),
		),
		P(
			Class("mt-2 text-xs text-gray-400 dark:text-gray-500"),
			Text(fmt.Sprintf("Created %s \u00b7 Owner ID %d", parseDashboardShortAt(parseSnap.CreatedAt), parseSnap.OwnerUserID)),
		),
	)
}

// renderAdminWSActionBand renders suspend/restore action buttons plus banners.
func renderAdminWSActionBand(parseWS adminWorkspacesController, parseSnap adminWorkspaceDetailSnapshot) ui.Node {
	isSuspended := strings.ToLower(parseSnap.Status) == "suspended"
	return Div(
		Class("mb-5 space-y-2"),
		If(parseWS.Data.MutationSuccess != "",
			Div(
				Class("rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 px-3 py-2 text-sm text-green-800 dark:text-green-300"),
				Text(parseWS.Data.MutationSuccess),
			),
		),
		If(parseWS.Data.MutationError != "",
			Div(
				Class("rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 px-3 py-2 text-sm text-red-800 dark:text-red-300"),
				Text(parseWS.Data.MutationError),
			),
		),
		Div(
			Class("flex flex-wrap gap-2"),
			If(!isSuspended,
				Button(
					Class("rounded-lg border border-red-300 dark:border-red-700 bg-red-50 dark:bg-red-900/20 px-3 py-1.5 text-xs font-medium text-red-800 dark:text-red-300 hover:bg-red-100 dark:hover:bg-red-900/40 transition-colors"),
					Data(dataAdminAction, "suspend"),
					OnClick(parseWS.HandleConfirmStart),
					Text("Suspend workspace"),
				),
			),
			If(isSuspended,
				Button(
					Class("rounded-lg border border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/20 px-3 py-1.5 text-xs font-medium text-green-800 dark:text-green-300 hover:bg-green-100 dark:hover:bg-green-900/40 transition-colors"),
					Data(dataAdminAction, "restore"),
					OnClick(parseWS.HandleConfirmStart),
					Text("Restore workspace"),
				),
			),
		),
	)
}

// renderAdminWSMembersTable renders the workspace members sub-table.
func renderAdminWSMembersTable(parseMembers []adminWorkspaceMemberRow) ui.Node {
	return Div(
		Class("mb-4"),
		P(Class("mb-1 text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"), Text("Members")),
		If(len(parseMembers) == 0,
			P(Class("text-xs text-gray-400 dark:text-gray-500 italic"), Text("No members.")),
		),
		If(len(parseMembers) > 0,
			Div(
				Class("overflow-x-auto rounded border border-gray-200 dark:border-gray-700"),
				Tag("table",
					Class("w-full border-collapse text-xs"),
					Tag("thead",
						Tr(Class("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("User ID")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Role")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Status")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Joined")),
						),
					),
					Tag("tbody",
						Class("divide-y divide-gray-100 dark:divide-gray-800"),
						func() []ui.Node {
							parseMemberRows := make([]ui.Node, len(parseMembers))
							for parseI, parseM := range parseMembers {
								parseMemberRows[parseI] = Tr(
									Td(Class("px-2 py-1 font-mono"), Text(strconv.FormatInt(parseM.UserID, 10))),
									Td(Class("px-2 py-1"), Text(parseM.RoleKey)),
									Td(Class("px-2 py-1"), Text(parseM.Status)),
									Td(Class("px-2 py-1 text-gray-400 dark:text-gray-500"), Text(parseDashboardShortAt(parseM.CreatedAt))),
								)
							}
							return parseMemberRows
						}()...,
					),
				),
			),
		),
	)
}

// renderAdminWSAPIKeysTable renders the workspace API keys sub-table.
func renderAdminWSAPIKeysTable(parseKeys []adminWorkspaceAPIKeyRow) ui.Node {
	return Div(
		Class("mb-4"),
		P(Class("mb-1 text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"), Text("API Keys")),
		If(len(parseKeys) == 0,
			P(Class("text-xs text-gray-400 dark:text-gray-500 italic"), Text("No API keys.")),
		),
		If(len(parseKeys) > 0,
			Div(
				Class("overflow-x-auto rounded border border-gray-200 dark:border-gray-700"),
				Tag("table",
					Class("w-full border-collapse text-xs"),
					Tag("thead",
						Tr(Class("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Label")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Prefix")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Status")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Created")),
						),
					),
					Tag("tbody",
						Class("divide-y divide-gray-100 dark:divide-gray-800"),
						func() []ui.Node {
							parseKeyRows := make([]ui.Node, len(parseKeys))
							for parseI, parseK := range parseKeys {
								isRevoked := parseK.RevokedAt != ""
								parseKeyRows[parseI] = Tr(
									Td(Class("px-2 py-1"), Text(parseK.Label)),
									Td(Class("px-2 py-1 font-mono text-gray-400"), Text(parseK.KeyPrefix+"…")),
									Td(Class("px-2 py-1"),
										If(isRevoked,
											Span(Class("text-red-500 dark:text-red-400"), Text("Revoked")),
										),
										If(!isRevoked,
											Span(Class("text-green-600 dark:text-green-400"), Text("Active")),
										),
									),
									Td(Class("px-2 py-1 text-gray-400"), Text(parseDashboardShortAt(parseK.CreatedAt))),
								)
							}
							return parseKeyRows
						}()...,
					),
				),
			),
		),
	)
}

// renderAdminWSWebhooksTable renders the workspace webhooks sub-table.
func renderAdminWSWebhooksTable(parseWebhooks []adminWorkspaceWebhookRow) ui.Node {
	return Div(
		Class("mb-4"),
		P(Class("mb-1 text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"), Text("Webhooks")),
		If(len(parseWebhooks) == 0,
			P(Class("text-xs text-gray-400 dark:text-gray-500 italic"), Text("No webhook endpoints.")),
		),
		If(len(parseWebhooks) > 0,
			Div(
				Class("overflow-x-auto rounded border border-gray-200 dark:border-gray-700"),
				Tag("table",
					Class("w-full border-collapse text-xs"),
					Tag("thead",
						Tr(Class("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Label")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("URL")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Enabled")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Failures")),
						),
					),
					Tag("tbody",
						Class("divide-y divide-gray-100 dark:divide-gray-800"),
						func() []ui.Node {
							parseWhRows := make([]ui.Node, len(parseWebhooks))
							for parseI, parseW := range parseWebhooks {
								parseEnabledText := "No"
								parseEnabledCls := "text-red-500 dark:text-red-400"
								if parseW.IsEnabled {
									parseEnabledText = "Yes"
									parseEnabledCls = "text-green-600 dark:text-green-400"
								}
								parseWhRows[parseI] = Tr(
									Td(Class("px-2 py-1"), Text(parseW.Label)),
									Td(Class("px-2 py-1 font-mono text-gray-400 max-w-[180px] truncate"), Text(parseW.TargetURL)),
									Td(Class("px-2 py-1 "+parseEnabledCls), Text(parseEnabledText)),
									Td(Class("px-2 py-1"), Text(strconv.FormatInt(parseW.FailureCount, 10))),
								)
							}
							return parseWhRows
						}()...,
					),
				),
			),
		),
	)
}

// renderAdminWSAuditTable renders the workspace audit log sub-table.
func renderAdminWSAuditTable(parseAudit []adminAuditRow) ui.Node {
	return Div(
		Class("mb-2"),
		P(Class("mb-1 text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"), Text("Recent audit log")),
		If(len(parseAudit) == 0,
			P(Class("text-xs text-gray-400 dark:text-gray-500 italic"), Text("No recent audit events.")),
		),
		If(len(parseAudit) > 0,
			Div(
				Class("overflow-x-auto rounded border border-gray-200 dark:border-gray-700"),
				Tag("table",
					Class("w-full border-collapse text-xs"),
					Tag("thead",
						Tr(Class("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Event")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Summary")),
							Th(Class("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("At")),
						),
					),
					Tag("tbody",
						Class("divide-y divide-gray-100 dark:divide-gray-800"),
						func() []ui.Node {
							parseAuditRows := make([]ui.Node, len(parseAudit))
							for parseI, parseA := range parseAudit {
								parseAuditRows[parseI] = Tr(
									Td(Class("px-2 py-1 font-mono text-xs text-gray-700 dark:text-gray-300"), Text(parseA.EventType)),
									Td(Class("px-2 py-1 text-gray-600 dark:text-gray-400"), Text(parseA.Summary)),
									Td(Class("px-2 py-1 text-gray-400"), Text(parseDashboardShortAt(parseA.CreatedAt))),
								)
							}
							return parseAuditRows
						}()...,
					),
				),
			),
		),
	)
}

// renderAdminWSConfirmModal renders the full-screen confirmation overlay for
// workspace suspend or restore actions.
func renderAdminWSConfirmModal(parseWS adminWorkspacesController) ui.Node {
	parseAction := parseWS.Data.ConfirmAction
	parseTitle := "Suspend workspace"
	parseBody := "This will immediately halt all API calls, background jobs, and billing for this workspace. Members will lose access."
	parseBtnLabel := "Suspend"
	parseBtnCls := "rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 transition-colors"
	if parseAction == "restore" {
		parseTitle = "Restore workspace"
		parseBody = "API keys, webhook endpoints, and background jobs will be re-enabled. Billing resumes from this point forward."
		parseBtnLabel = "Restore"
		parseBtnCls = "rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white hover:bg-green-700 transition-colors"
	}
	return Div(
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"),
		Div(
			Class("w-full max-w-md rounded-2xl bg-white dark:bg-gray-900 p-6 shadow-2xl border border-gray-200 dark:border-gray-700 mx-4"),
			H3(Class("text-base font-semibold text-gray-900 dark:text-gray-100 mb-2"), Text(parseTitle)),
			P(Class("text-sm text-gray-600 dark:text-gray-400 mb-4"), Text(parseBody)),
			Div(
				Class("mb-4"),
				Label(Class("block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1"), Text("Reason (required)")),
				Tag("textarea",
					Class("w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"),
					Rows(3),
					Placeholder("Describe the reason for this action\u2026"),
					Value(parseWS.Data.ConfirmReason),
					OnInput(parseWS.HandleConfirmReason),
				),
			),
			If(parseWS.Data.MutationError != "",
				Div(
					Class("mb-3 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 px-3 py-2 text-sm text-red-800 dark:text-red-300"),
					Text(parseWS.Data.MutationError),
				),
			),
			Div(
				Class("flex justify-end gap-3"),
				Button(
					Class("rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"),
					Disabled(parseWS.Data.IsMutationPending),
					OnClick(parseWS.HandleConfirmCancel),
					Text("Cancel"),
				),
				Button(
					ClassNames(
						parseBtnCls,
						When(parseWS.Data.IsMutationPending, "opacity-60 cursor-not-allowed"),
					),
					Disabled(parseWS.Data.IsMutationPending || parseWS.Data.ConfirmReason == ""),
					OnClick(parseWS.HandleConfirmSubmit),
					If(parseWS.Data.IsMutationPending, Text("Working\u2026")),
					If(!parseWS.Data.IsMutationPending, Text(parseBtnLabel)),
				),
			),
		),
	)
}
