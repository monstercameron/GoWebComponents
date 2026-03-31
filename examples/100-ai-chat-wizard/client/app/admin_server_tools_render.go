//go:build js && wasm

// admin_server_tools_render.go renders the superuser server-tools panel
// inside the Ops dashboard slice.
package app

import (
	"fmt"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderDashboardServerToolsPanel renders the full superuser server-tools surface.
// When the user is not a superuser it renders nothing.
func renderDashboardServerToolsPanel(parseView appViewState) ui.Node {
	if !parseView.IsSuperuser {
		return nil
	}
	parseTools := parseView.AdminServerTools
	return Fragment(
		renderDashboardSectionHeader("Server tools (superuser)"),
		renderDashboardServerToolsBody(parseTools),
	)
}

// renderDashboardServerToolsBody renders loading/error/data states for the server-tools panel.
func renderDashboardServerToolsBody(parseTools adminServerToolsData) ui.Node {
	if parseTools.IsLoading {
		return renderDashboardLoadingState("Loading server tools\u2026")
	}
	if parseTools.IsDenied {
		return renderDashboardDeniedBanner()
	}
	if parseTools.Error != "" {
		return renderDashboardErrorBanner(parseTools.Error)
	}
	if !parseTools.HasData {
		return renderDashboardEmptyState("\U0001f6e0\ufe0f", "Server tools not available", "Fetch the superuser ops diagnostics to review the current tool policy and recent executions.")
	}
	return Fragment(
		renderDashboardServerToolsPolicyCard(parseTools.Policy),
		renderDashboardServerToolsExecutionsTable(parseTools.Executions),
		renderDashboardServerToolsLogTail(parseTools.LogLines),
	)
}

// renderDashboardServerToolsPolicyCard renders the current tool-policy summary.
func renderDashboardServerToolsPolicyCard(parsePolicy adminServerToolsPolicy) ui.Node {
	parseEnabledLabel := "disabled"
	parseEnabledColor := "text-amber-400"
	if parsePolicy.IsEnabled {
		parseEnabledLabel = "enabled"
		parseEnabledColor = "text-emerald-400"
	}
	parseUpdatedAt := parseDashboardShortAt(parsePolicy.UpdatedAt)
	parseSource := strings.TrimSpace(parsePolicy.Source)
	if parseSource == "" {
		parseSource = "\u2014"
	}
	parseToolCount := len(parsePolicy.ApprovedTools)
	return Div(
		Class("mb-4 rounded-[1.2rem] border border-white/10 bg-white/[0.03] p-4"),
		Div(Class("mb-3 flex items-center gap-2"),
			P(Class("text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Tool policy")),
			Span(Class(fmt.Sprintf("text-xs font-semibold %s", parseEnabledColor)), Text(parseEnabledLabel)),
		),
		Div(Class("grid grid-cols-2 gap-3 sm:grid-cols-4 mb-3"),
			renderDashboardKPICard("Approved tools", fmt.Sprintf("%d", parseToolCount), ""),
			renderDashboardKPICard("Max session", fmt.Sprintf("%ds", parsePolicy.MaxSessionSeconds), ""),
			renderDashboardKPICard("Max output", parseFormatBytes(int64(parsePolicy.MaxOutputBytes)), ""),
			renderDashboardKPICard("Source", parseSource, ""),
		),
		If(parseUpdatedAt != "\u2014",
			P(Class("text-[11px] text-white/30"), Text("Updated "+parseUpdatedAt)),
		),
		If(parseToolCount > 0,
			renderDashboardServerToolsRulesTable(parsePolicy.ApprovedTools),
		),
	)
}

// renderDashboardServerToolsRulesTable renders the approved-tool rule list as a data table.
func renderDashboardServerToolsRulesTable(parseRules []adminServerToolsPolicyRule) ui.Node {
	parseRows := make([][]string, len(parseRules))
	for parseI, parseRule := range parseRules {
		parseShell := parseRule.Shell
		if parseShell == "" {
			parseShell = "auto"
		}
		parseEnabledText := "no"
		if parseRule.IsEnabled {
			parseEnabledText = "yes"
		}
		parseRows[parseI] = []string{parseRule.ToolID, parseRule.Description, parseShell, parseEnabledText}
	}
	return renderDashboardTable([]string{"Tool ID", "Description", "Shell", "Enabled"}, parseRows)
}

// renderDashboardServerToolsExecutionsTable renders recent server-tool audit executions.
func renderDashboardServerToolsExecutionsTable(parseExecs []adminServerToolsExecutionRow) ui.Node {
	if len(parseExecs) == 0 {
		return P(Class("mt-2 text-xs text-white/30"), Text("No recent executions recorded."))
	}
	parseRows := make([][]string, len(parseExecs))
	for parseI, parseExec := range parseExecs {
		parseRows[parseI] = []string{parseExec.EventType, parseExec.Summary, parseDashboardShortAt(parseExec.CreatedAt)}
	}
	return Fragment(
		P(Class("mb-2 text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Recent executions")),
		renderDashboardTable([]string{"Event", "Summary", "When"}, parseRows),
	)
}

// renderDashboardServerToolsLogTail renders the most recent log lines from the server.
func renderDashboardServerToolsLogTail(parseLines []string) ui.Node {
	if len(parseLines) == 0 {
		return nil
	}
	parseLogText := strings.Join(parseLines, "\n")
	return Fragment(
		P(Class("mt-4 mb-2 text-xs font-medium uppercase tracking-[0.22em] text-white/35"), Text("Log tail")),
		Div(
			Class("rounded-xl border border-white/10 bg-black/30 p-3 overflow-x-auto"),
			Pre(
				Class("text-[11px] leading-5 text-white/55 font-mono whitespace-pre-wrap break-all"),
				Text(parseLogText),
			),
		),
	)
}

// parseFormatBytes returns a compact human-readable byte count (KB / MB / raw).
func parseFormatBytes(parseN int64) string {
	if parseN <= 0 {
		return "\u2014"
	}
	if parseN >= 1_048_576 {
		return fmt.Sprintf("%.1f MB", float64(parseN)/1_048_576)
	}
	if parseN >= 1024 {
		return fmt.Sprintf("%d KB", parseN/1024)
	}
	return fmt.Sprintf("%d B", parseN)
}

