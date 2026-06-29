//go:build js && wasm

package app

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/i18n"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// renderDashboardWorkspacesPanel renders the Workspaces admin panel with an
// ID-based lookup bar, workspace detail panel, and suspend/restore confirmation modal.
func renderDashboardWorkspacesPanel(parseIntl i18n.Runtime, parseView appViewState, parseWS adminWorkspacesController) ui.Node {
	_ = parseIntl
	_ = parseView
	isDetailOpen := parseWS.Data.SelectedWorkspaceID > 0

	return Fragment(
		renderDashboardSectionHeader("Workspaces"),
		renderAdminWSLookupBar(parseWS),
		If(isDetailOpen,
			renderAdminWSDetailPanel(parseWS),
		),
		If(parseWS.Data.ConfirmAction != "",
			renderAdminWSConfirmModal(parseWS),
		),
	)
}

// renderAdminWSLookupBar renders the workspace ID input + look-up button.
func renderAdminWSLookupBar(parseWS adminWorkspacesController) ui.Node {
	return Div(
		ClassStr("mb-5"),
		P(ClassStr("mb-2 text-sm text-gray-600 dark:text-gray-400"),
			Text("Enter a workspace ID to inspect its detail, members, API keys, webhooks, and audit log."),
		),
		Div(
			ClassStr("flex items-center gap-2"),
			Input(
				ClassStr("w-48 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-3 py-2 text-sm font-mono placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"),
				Type("text"),
				Placeholder("Workspace ID\u2026"),
				Value(parseWS.Data.LookupInput),
				OnInput(parseWS.HandleLookupInput),
			),
			Button(
				ClassStr("rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 transition-colors"),
				OnClick(parseWS.HandleLookupSubmit),
				Text("Look up"),
			),
		),
		If(parseWS.Data.LookupError != "",
			P(ClassStr("mt-1 text-xs text-red-600 dark:text-red-400"), Text(parseWS.Data.LookupError)),
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
	return Span(ClassStr(parseCls), Text(parseStatus))
}

// renderAdminWSDetailPanel renders the right-side detail panel for one workspace.
func renderAdminWSDetailPanel(parseWS adminWorkspacesController) ui.Node {
	parseSnap := parseWS.Data.WorkspaceDetail
	return Div(
		ClassStr("flex-1 min-w-0 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-5 relative"),
		// Close button.
		Button(
			ClassStr("absolute right-4 top-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-lg leading-none"),
			OnClick(parseWS.HandleDismiss),
			Text("\u00d7"),
		),
		If(parseWS.Data.IsLoadingDetail,
			P(ClassStr("text-sm text-gray-500 dark:text-gray-400"), Text("Loading workspace detail\u2026")),
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
			P(ClassStr("text-sm text-gray-500 dark:text-gray-400"), Text("Select a workspace to view detail.")),
		),
	)
}

// renderAdminWSDetailHeader renders the workspace name, slug, plan, and status.
func renderAdminWSDetailHeader(parseSnap adminWorkspaceDetailSnapshot) ui.Node {
	return Div(
		ClassStr("mb-4 pr-8"),
		H3(ClassStr("text-base font-semibold text-gray-900 dark:text-gray-100"), Text(parseSnap.Name)),
		P(
			ClassStr("text-xs text-gray-500 dark:text-gray-400 font-mono mt-0.5"),
			Text(parseSnap.Slug),
		),
		Div(
			ClassStr("mt-2 flex flex-wrap gap-2"),
			renderAdminWSStatusBadge(parseSnap.Status),
			Span(
				ClassStr("inline-flex items-center rounded px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300"),
				Text(parseSnap.PlanCode),
			),
		),
		P(
			ClassStr("mt-2 text-xs text-gray-400 dark:text-gray-500"),
			Text(fmt.Sprintf("Created %s \u00b7 Owner ID %d", parseDashboardShortAt(parseSnap.CreatedAt), parseSnap.OwnerUserID)),
		),
	)
}

// renderAdminWSActionBand renders suspend/restore action buttons plus banners.
func renderAdminWSActionBand(parseWS adminWorkspacesController, parseSnap adminWorkspaceDetailSnapshot) ui.Node {
	isSuspended := strings.ToLower(parseSnap.Status) == "suspended"
	return Div(
		ClassStr("mb-5 space-y-2"),
		If(parseWS.Data.MutationSuccess != "",
			Div(
				ClassStr("rounded-xl border border-green-500/20 bg-green-500/5 px-4 py-3"),
				Div(ClassStr("mb-1 text-[10px] uppercase tracking-[0.18em] text-green-400/55"), Text("Action receipt")),
				P(ClassStr("text-xs text-green-300"), Text(parseWS.Data.MutationSuccess)),
				Div(
					ClassStr("mt-2 flex items-center gap-3 text-[10px] text-green-400/45"),
					Span(Text("Recorded just now")),
					Span(ClassStr("text-green-400/20"), Text("·")),
					Span(Text("Verify in workspace detail below")),
				),
			),
		),
		If(parseWS.Data.MutationError != "",
			Div(
				ClassStr("flex flex-col gap-1 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 px-3 py-2 text-sm text-red-800 dark:text-red-300"),
				Text(parseUserErrorMessage(parseWS.Data.MutationError)),
				renderSupportIDChip(parseUserErrorRequestID(parseWS.Data.MutationError)),
			),
		),
		Div(
			ClassStr("flex flex-wrap gap-2"),
			If(!isSuspended,
				Button(
					ClassStr("rounded-lg border border-red-300 dark:border-red-700 bg-red-50 dark:bg-red-900/20 px-3 py-1.5 text-xs font-medium text-red-800 dark:text-red-300 hover:bg-red-100 dark:hover:bg-red-900/40 transition-colors"),
					Data(dataAdminAction, "suspend"),
					OnClick(parseWS.HandleConfirmStart),
					Text("Suspend workspace"),
				),
			),
			If(isSuspended,
				Button(
					ClassStr("rounded-lg border border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/20 px-3 py-1.5 text-xs font-medium text-green-800 dark:text-green-300 hover:bg-green-100 dark:hover:bg-green-900/40 transition-colors"),
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
		ClassStr("mb-4"),
		P(ClassStr("mb-1 text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"), Text("Members")),
		If(len(parseMembers) == 0,
			P(ClassStr("text-xs text-gray-400 dark:text-gray-500 italic"), Text("No members.")),
		),
		If(len(parseMembers) > 0,
			Div(
				ClassStr("overflow-x-auto rounded border border-gray-200 dark:border-gray-700"),
				Tag("table",
					ClassStr("w-full border-collapse text-xs"),
					Tag("thead",
						Tr(ClassStr("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("User ID")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Role")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Status")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Joined")),
						),
					),
					Tbody(
						ClassStr("divide-y divide-gray-100 dark:divide-gray-800"),
						func() ui.Node {
							parseMemberRows := make([]ui.Node, len(parseMembers))
							for parseI, parseM := range parseMembers {
								parseMemberRows[parseI] = Tr(
									Td(ClassStr("px-2 py-1 font-mono"), Text(strconv.FormatInt(parseM.UserID, 10))),
									Td(ClassStr("px-2 py-1"), Text(parseM.RoleKey)),
									Td(ClassStr("px-2 py-1"), Text(parseM.Status)),
									Td(ClassStr("px-2 py-1 text-gray-400 dark:text-gray-500"), Text(parseDashboardShortAt(parseM.CreatedAt))),
								)
							}
							return Fragment(parseMemberRows)
						}(),
					),
				),
			),
		),
	)
}

// renderAdminWSAPIKeysTable renders the workspace API keys sub-table.
func renderAdminWSAPIKeysTable(parseKeys []adminWorkspaceAPIKeyRow) ui.Node {
	return Div(
		ClassStr("mb-4"),
		P(ClassStr("mb-1 text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"), Text("API Keys")),
		If(len(parseKeys) == 0,
			P(ClassStr("text-xs text-gray-400 dark:text-gray-500 italic"), Text("No API keys.")),
		),
		If(len(parseKeys) > 0,
			Div(
				ClassStr("overflow-x-auto rounded border border-gray-200 dark:border-gray-700"),
				Tag("table",
					ClassStr("w-full border-collapse text-xs"),
					Tag("thead",
						Tr(ClassStr("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Label")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Prefix")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Status")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Created")),
						),
					),
					Tbody(
						ClassStr("divide-y divide-gray-100 dark:divide-gray-800"),
						func() ui.Node {
							parseKeyRows := make([]ui.Node, len(parseKeys))
							for parseI, parseK := range parseKeys {
								isRevoked := parseK.RevokedAt != ""
								parseKeyRows[parseI] = Tr(
									Td(ClassStr("px-2 py-1"), Text(parseK.Label)),
									Td(ClassStr("px-2 py-1 font-mono text-gray-400"), Text(parseK.KeyPrefix+"…")),
									Td(ClassStr("px-2 py-1"),
										If(isRevoked,
											Span(ClassStr("text-red-500 dark:text-red-400"), Text("Revoked")),
										),
										If(!isRevoked,
											Span(ClassStr("text-green-600 dark:text-green-400"), Text("Active")),
										),
									),
									Td(ClassStr("px-2 py-1 text-gray-400"), Text(parseDashboardShortAt(parseK.CreatedAt))),
								)
							}
							return Fragment(parseKeyRows)
						}(),
					),
				),
			),
		),
	)
}

// renderAdminWSWebhooksTable renders the workspace webhooks sub-table.
func renderAdminWSWebhooksTable(parseWebhooks []adminWorkspaceWebhookRow) ui.Node {
	return Div(
		ClassStr("mb-4"),
		P(ClassStr("mb-1 text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"), Text("Webhooks")),
		If(len(parseWebhooks) == 0,
			P(ClassStr("text-xs text-gray-400 dark:text-gray-500 italic"), Text("No webhook endpoints.")),
		),
		If(len(parseWebhooks) > 0,
			Div(
				ClassStr("overflow-x-auto rounded border border-gray-200 dark:border-gray-700"),
				Tag("table",
					ClassStr("w-full border-collapse text-xs"),
					Tag("thead",
						Tr(ClassStr("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Label")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("URL")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Enabled")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Failures")),
						),
					),
					Tbody(
						ClassStr("divide-y divide-gray-100 dark:divide-gray-800"),
						func() ui.Node {
							parseWhRows := make([]ui.Node, len(parseWebhooks))
							for parseI, parseW := range parseWebhooks {
								parseEnabledText := "No"
								parseEnabledCls := "text-red-500 dark:text-red-400"
								if parseW.IsEnabled {
									parseEnabledText = "Yes"
									parseEnabledCls = "text-green-600 dark:text-green-400"
								}
								parseWhRows[parseI] = Tr(
									Td(ClassStr("px-2 py-1"), Text(parseW.Label)),
									Td(ClassStr("px-2 py-1 font-mono text-gray-400 max-w-[180px] truncate"), Text(parseW.TargetURL)),
									Td(ClassStr("px-2 py-1 "+parseEnabledCls), Text(parseEnabledText)),
									Td(ClassStr("px-2 py-1"), Text(strconv.FormatInt(parseW.FailureCount, 10))),
								)
							}
							return Fragment(parseWhRows)
						}(),
					),
				),
			),
		),
	)
}

// renderAdminWSAuditTable renders the workspace audit log sub-table.
func renderAdminWSAuditTable(parseAudit []adminAuditRow) ui.Node {
	return Div(
		ClassStr("mb-2"),
		P(ClassStr("mb-1 text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"), Text("Recent audit log")),
		If(len(parseAudit) == 0,
			P(ClassStr("text-xs text-gray-400 dark:text-gray-500 italic"), Text("No recent audit events.")),
		),
		If(len(parseAudit) > 0,
			Div(
				ClassStr("overflow-x-auto rounded border border-gray-200 dark:border-gray-700"),
				Tag("table",
					ClassStr("w-full border-collapse text-xs"),
					Tag("thead",
						Tr(ClassStr("border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/60"),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Event")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("Summary")),
							Th(ClassStr("px-2 py-1 text-left text-gray-500 dark:text-gray-400"), Text("At")),
						),
					),
					Tbody(
						ClassStr("divide-y divide-gray-100 dark:divide-gray-800"),
						func() ui.Node {
							parseAuditRows := make([]ui.Node, len(parseAudit))
							for parseI, parseA := range parseAudit {
								parseAuditRows[parseI] = Tr(
									Td(ClassStr("px-2 py-1 font-mono text-xs text-gray-700 dark:text-gray-300"), Text(parseA.EventType)),
									Td(ClassStr("px-2 py-1 text-gray-600 dark:text-gray-400"), Text(parseA.Summary)),
									Td(ClassStr("px-2 py-1 text-gray-400"), Text(parseDashboardShortAt(parseA.CreatedAt))),
								)
							}
							return Fragment(parseAuditRows)
						}(),
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
		ClassStr("fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"),
		Div(
			ClassStr("w-full max-w-md rounded-2xl bg-white dark:bg-gray-900 p-6 shadow-2xl border border-gray-200 dark:border-gray-700 mx-4"),
			H3(ClassStr("text-base font-semibold text-gray-900 dark:text-gray-100 mb-2"), Text(parseTitle)),
			P(ClassStr("text-sm text-gray-600 dark:text-gray-400 mb-4"), Text(parseBody)),
			Div(
				ClassStr("mb-4"),
				Label(ClassStr("block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1"), Text("Reason (required)")),
				Tag("textarea",
					ClassStr("w-full rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"),
					Rows(3),
					Placeholder("Describe the reason for this action\u2026"),
					Value(parseWS.Data.ConfirmReason),
					OnInput(parseWS.HandleConfirmReason),
				),
			),
			If(parseWS.Data.MutationError != "",
				Div(
					ClassStr("mb-3 flex flex-col gap-1 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 px-3 py-2 text-sm text-red-800 dark:text-red-300"),
					Text(parseUserErrorMessage(parseWS.Data.MutationError)),
					renderSupportIDChip(parseUserErrorRequestID(parseWS.Data.MutationError)),
				),
			),
			Div(
				ClassStr("flex justify-end gap-3"),
				Button(
					ClassStr("rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"),
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
