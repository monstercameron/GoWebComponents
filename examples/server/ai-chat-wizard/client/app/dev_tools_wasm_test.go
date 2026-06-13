//go:build js && wasm

package app

import (
	"strings"
	"testing"
)

func TestFrameworkCalloutsIncludeRequiredLinkedPatterns(parseT *testing.T) {
	parseCallouts := parseBuildFrameworkCallouts()
	parseWantLabels := []string{
		"SSR bootstrap",
		"Typed routes",
		"Streaming chat",
		"Worker tasks",
		"Cross-tab sync",
		"Server functions",
	}
	if len(parseCallouts) != len(parseWantLabels) {
		parseT.Fatalf("callout count = %d, want %d", len(parseCallouts), len(parseWantLabels))
	}
	parseSeen := map[string]bool{}
	for _, parseCallout := range parseCallouts {
		if strings.TrimSpace(parseCallout.Href) == "" {
			parseT.Fatalf("callout %q has empty href", parseCallout.Label)
		}
		if !strings.Contains(parseCallout.Href, "github.com/monstercameron/GoWebComponents") {
			parseT.Fatalf("callout %q href = %q, want repo docs/source link", parseCallout.Label, parseCallout.Href)
		}
		if strings.TrimSpace(parseCallout.LinkLabel) == "" {
			parseT.Fatalf("callout %q has empty link label", parseCallout.Label)
		}
		parseSeen[parseCallout.Label] = true
	}
	for _, parseLabel := range parseWantLabels {
		if !parseSeen[parseLabel] {
			parseT.Fatalf("missing framework callout %q", parseLabel)
		}
	}
}

func TestDevToolModeQueryAliases(parseT *testing.T) {
	parseTests := []struct {
		search string
		want   string
	}{
		{search: "?gwc-dev=tour", want: devToolTourMode},
		{search: "?panel=settings-billing&gwc-dev=panel", want: devToolPanelMode},
		{search: "?gwc-dev=demo-helper", want: devToolPanelMode},
		{search: "?gwc-dev=helper", want: devToolPanelMode},
		{search: "?gwc-dev=unknown", want: ""},
		{search: "?other=1", want: ""},
	}
	for _, parseTest := range parseTests {
		if parseGot := parseDevToolModeFromSearch(parseTest.search); parseGot != parseTest.want {
			parseT.Fatalf("parseDevToolModeFromSearch(%q) = %q, want %q", parseTest.search, parseGot, parseTest.want)
		}
	}
}

func TestBuildURLWithoutDevToolPreservesOtherState(parseT *testing.T) {
	parseGot := parseBuildURLWithoutDevTool("/app/settings", "?panel=settings-billing&gwc-dev=demo-helper&tab=usage", "#settings-billing")
	if parseGot != "/app/settings?panel=settings-billing&tab=usage#settings-billing" {
		parseT.Fatalf("parseBuildURLWithoutDevTool returned %q", parseGot)
	}
}

func TestDevPanelRouteAndShellClassification(parseT *testing.T) {
	parseTests := []struct {
		name      string
		view      appViewState
		wantRoute string
		wantShell string
	}{
		{
			name:      "public pricing route",
			view:      appViewState{CurrentPath: marketingPricingRoute},
			wantRoute: "public.pricing",
			wantShell: "public landing: pricing",
		},
		{
			name: "dashboard provider slice",
			view: appViewState{
				CurrentPath:   chatRouteDashboardProviders,
				AuthResolved:  true,
				Authenticated: true,
			},
			wantRoute: "workspace.dashboard.providers",
			wantShell: "dashboard: providers",
		},
		{
			name: "settings section",
			view: appViewState{
				CurrentPath:           settingsRoutePath,
				AuthResolved:          true,
				Authenticated:         true,
				ActiveSettingsSection: settingsSectionBilling,
			},
			wantRoute: "workspace.settings",
			wantShell: "settings: settings-billing",
		},
		{
			name: "thread canvas route",
			view: appViewState{
				CurrentPath:     "/app/thread/thread-a/canvas/canvas-1",
				AuthResolved:    true,
				Authenticated:   true,
				CanvasOnlyRoute: true,
			},
			wantRoute: "workspace.thread.canvas",
			wantShell: "canvas workspace",
		},
	}
	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := parseResolveDevRouteID(parseTest.view.CurrentPath); parseGot != parseTest.wantRoute {
				parseT2.Fatalf("parseResolveDevRouteID(%q) = %q, want %q", parseTest.view.CurrentPath, parseGot, parseTest.wantRoute)
			}
			if parseGot := parseResolveDevShellSection(parseTest.view); parseGot != parseTest.wantShell {
				parseT2.Fatalf("parseResolveDevShellSection(...) = %q, want %q", parseGot, parseTest.wantShell)
			}
		})
	}
}

func TestDevPanelRowsIncludeAsyncAndRuntimeSummaries(parseT *testing.T) {
	parseRows := parseBuildDevPanelRows(appViewState{
		CurrentPath:            chatRouteDashboardOps,
		AuthResolved:           true,
		Authenticated:          true,
		GRPCReady:              true,
		CatalogServerSynced:    true,
		MarkdownWorkerFallback: true,
		IsStreaming:            true,
		SelectedModel:          "openai/gpt-4.1",
		CanAccessAdmin:         true,
		UserName:               "Admin",
		ActiveConversationID:   42,
		AdminDashboardData:     adminDashboardData{IsLoading: true},
	})
	parseJoined := ""
	for _, parseRow := range parseRows {
		parseJoined += parseRow[0] + "=" + parseRow[1] + "\n"
	}
	for _, parseWant := range []string{
		"Route ID=workspace.dashboard.ops",
		"Shell section=dashboard: ops",
		"auth: session",
		"tunnel: ready",
		"catalog: synced",
		"worker: fallback",
		"dashboard: loading",
		"streaming: active",
		"model: openai/gpt-4.1",
		"role: admin",
		"Active conv ID=42",
	} {
		if !strings.Contains(parseJoined, parseWant) {
			parseT.Fatalf("dev panel rows missing %q in:\n%s", parseWant, parseJoined)
		}
	}
}
