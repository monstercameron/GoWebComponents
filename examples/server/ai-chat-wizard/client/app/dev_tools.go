//go:build js && wasm

package app

import (
	"net/url"
	"strconv"
	"strings"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// ─── Dev-tool query param ────────────────────────────────────────────────────
//
// Add ?gwc-dev=tour to any route to open the framework tour overlay.
// Add ?gwc-dev=panel to any route to open the runtime state debug panel.
// Both are dismissible and do not affect normal product use.

const devToolQueryKey = "gwc-dev"
const devToolTourMode = "tour"
const devToolPanelMode = "panel"
const devToolDemoHelperMode = "demo-helper"

// parseGetDevToolMode reads the ?gwc-dev= query param from window.location.search.
// Returns empty string when not present or when running outside a browser context.
func parseGetDevToolMode() string {
	parseWindow := js.Global()
	if !parseWindow.Truthy() {
		return ""
	}
	return parseDevToolModeFromSearch(parseWindow.Get("location").Get("search").String())
}

func parseNormalizeDevToolMode(parseRaw string) string {
	switch strings.ToLower(strings.TrimSpace(parseRaw)) {
	case devToolTourMode, "framework-tour":
		return devToolTourMode
	case devToolPanelMode, devToolDemoHelperMode, "helper", "demo", "runtime":
		return devToolPanelMode
	default:
		return ""
	}
}

func parseDevToolModeFromSearch(parseSearch string) string {
	parseValues, parseErr := url.ParseQuery(strings.TrimPrefix(strings.TrimSpace(parseSearch), "?"))
	if parseErr != nil {
		return ""
	}
	return parseNormalizeDevToolMode(parseValues.Get(devToolQueryKey))
}

func parseBuildURLWithoutDevTool(parsePathname, parseSearch, parseHash string) string {
	parseValues, parseErr := url.ParseQuery(strings.TrimPrefix(strings.TrimSpace(parseSearch), "?"))
	if parseErr != nil {
		return parsePathname + parseHash
	}
	parseValues.Del(devToolQueryKey)
	parseEncoded := parseValues.Encode()
	if parseEncoded != "" {
		return parsePathname + "?" + parseEncoded + parseHash
	}
	return parsePathname + parseHash
}

func parseDismissDevToolURL() string {
	parseWindow := js.Global()
	if !parseWindow.Truthy() {
		return ""
	}
	parseLoc := parseWindow.Get("location")
	return parseBuildURLWithoutDevTool(
		parseLoc.Get("pathname").String(),
		parseLoc.Get("search").String(),
		parseLoc.Get("hash").String(),
	)
}

// ─── Dev tools overlay ───────────────────────────────────────────────────────

// renderDevToolsOverlay renders the active dev tool overlay (tour or panel) as a fixed layer
// on top of whatever shell surface is currently rendered. Returns nil when no dev mode is active.
func renderDevToolsOverlay(parseView appViewState) ui.Node {
	parseMode := parseGetDevToolMode()
	switch parseMode {
	case devToolTourMode:
		return renderDevTourOverlay(parseView)
	case devToolPanelMode:
		return renderDevPanelOverlay(parseView)
	}
	return nil
}

// ─── Framework tour overlay ──────────────────────────────────────────────────

type parseTourStop struct {
	parseTitle   string
	parsePattern string
	parseFile    string
	parseDesc    string
}

func parseBuildTourStops() []parseTourStop {
	return []parseTourStop{
		{
			parseTitle:   "App shell",
			parsePattern: "Single-shell routed SPA",
			parseFile:    "client/app/app.go",
			parseDesc:    "ParseApp owns the single render path from route change to correct surface. Every public, auth, and workspace view forks from one function.",
		},
		{
			parseTitle:   "Route transition",
			parsePattern: "Typed routes + history router",
			parseFile:    "client/app/routes.go",
			parseDesc:    "The client defines route constants. ParseRun registers each with the history router. URL changes drive the WASM render cycle without a page load.",
		},
		{
			parseTitle:   "Model picker state",
			parsePattern: "Cross-tab preference sync",
			parseFile:    "client/app/model_preferences.go",
			parseDesc:    "Model selection, tone, and thinking state persist across reloads and sync across browser tabs via the state package atom broadcast.",
		},
		{
			parseTitle:   "Composer runtime2 region",
			parsePattern: "Display-only ui.ParallelRegion",
			parseFile:    "client/app/composer_runtime2.go",
			parseDesc:    "Example 100 currently uses runtime2 in the composer cost summary. The renderer is registered once and mounted through a display-only parallel region on the public UI path.",
		},
		{
			parseTitle:   "Streamed thread",
			parsePattern: "Streaming progressive render",
			parseFile:    "client/app/stream.go",
			parseDesc:    "ChatChunk deltas arrive over the gRPC tunnel and are applied incrementally to the thread view without buffering the full reply first.",
		},
		{
			parseTitle:   "Dashboard slice",
			parsePattern: "Route-scoped async state",
			parseFile:    "client/app/dashboard_shell.go",
			parseDesc:    "Each dashboard tile represents one admin slice. Data loads on route activation via typed RPCs and re-renders on update without refreshing other slices.",
		},
		{
			parseTitle:   "Settings panel",
			parsePattern: "Route-scoped panels + persisted preferences",
			parseFile:    "client/app/settings_route.go",
			parseDesc:    "Query param ?panel= selects the active settings section. Each section loads its own data and submits changes via typed gRPC. Preferences persist server-side.",
		},
	}
}

func parsePathOnly(parsePath string) string {
	parsePath = strings.TrimSpace(parsePath)
	if parseIdx := strings.Index(parsePath, "#"); parseIdx >= 0 {
		parsePath = parsePath[:parseIdx]
	}
	if parseIdx := strings.Index(parsePath, "?"); parseIdx >= 0 {
		parsePath = parsePath[:parseIdx]
	}
	if parsePath == "" {
		return authLandingRoute
	}
	return parsePath
}

func parseResolveDashboardSliceID(parsePath string) string {
	switch parsePathOnly(parsePath) {
	case chatRouteDashboardBusiness:
		return "business"
	case chatRouteDashboardCustomers:
		return "customers"
	case chatRouteDashboardChats:
		return "chats"
	case chatRouteDashboardProviders:
		return "providers"
	case chatRouteDashboardOps:
		return "ops"
	case chatRouteDashboardHome:
		return "home"
	default:
		return ""
	}
}

func parseResolveDevRouteID(parsePath string) string {
	parseCleanPath := parsePathOnly(parsePath)
	if isLandingRoute(parseCleanPath) {
		return "public." + parseLandingPageForPath(parseCleanPath)
	}
	switch {
	case parseCleanPath == authLoginRoute:
		return "auth.login"
	case parseCleanPath == settingsRoutePath:
		return "workspace.settings"
	case parseResolveDashboardSliceID(parseCleanPath) != "":
		return "workspace.dashboard." + parseResolveDashboardSliceID(parseCleanPath)
	case strings.Contains(parseCleanPath, "/canvas/"):
		return "workspace.thread.canvas"
	case parseThreadRoutePublicIDFromPath(parseCleanPath) != "":
		return "workspace.thread"
	case parseCleanPath == chatRouteRoot:
		return "workspace.root"
	case strings.HasPrefix(parseCleanPath, chatRouteRoot+"/"):
		return "workspace.route"
	default:
		return "unknown"
	}
}

func parseResolveDevShellSection(parseView appViewState) string {
	parseCleanPath := parsePathOnly(parseView.CurrentPath)
	if isLandingRoute(parseCleanPath) {
		return "public landing: " + parseLandingPageForPath(parseCleanPath)
	}
	if !parseView.AuthResolved {
		return "auth bootstrap"
	}
	if !parseView.Authenticated {
		return "auth shell"
	}
	if parseView.CanvasOnlyRoute {
		return "canvas workspace"
	}
	if parseCleanPath == settingsRoutePath {
		parseSection := parseNormalizeSettingsSectionID(parseView.ActiveSettingsSection)
		if parseSection == "" {
			parseSection = defaultSettingsSectionID
		}
		return "settings: " + parseSection
	}
	if parseSlice := parseResolveDashboardSliceID(parseCleanPath); parseSlice != "" {
		return "dashboard: " + parseSlice
	}
	if parseThreadRoutePublicIDFromPath(parseCleanPath) != "" {
		return "thread workspace"
	}
	if parseCleanPath == chatRouteRoot {
		return "workspace home"
	}
	return "workspace"
}

func parseBoolReady(parseReady bool, parseReadyLabel, parseWaitingLabel string) string {
	if parseReady {
		return parseReadyLabel
	}
	return parseWaitingLabel
}

func parseResolveDashboardAsyncState(parseView appViewState) string {
	if !strings.HasPrefix(parsePathOnly(parseView.CurrentPath), chatRouteDashboardHome) {
		return "dashboard: idle"
	}
	if parseView.AdminDashboardData.IsLoading {
		return "dashboard: loading"
	}
	if strings.TrimSpace(parseView.AdminDashboardData.Error) != "" {
		return "dashboard: error"
	}
	if parseView.AdminDashboardData.HasData {
		return "dashboard: ready"
	}
	if parseView.AdminDashboardData.IsDenied {
		return "dashboard: denied"
	}
	return "dashboard: waiting"
}

func parseResolveDevAsyncResources(parseView appViewState) []string {
	parseAuthState := "auth: pending"
	if parseView.AuthResolved && parseView.Authenticated {
		parseAuthState = "auth: session"
	} else if parseView.AuthResolved {
		parseAuthState = "auth: guest"
	}
	parseWorkerState := "worker: wasm"
	if parseView.MarkdownWorkerFallback {
		parseWorkerState = "worker: fallback"
	}
	return []string{
		parseAuthState,
		parseBoolReady(parseView.GRPCReady, "tunnel: ready", "tunnel: waiting"),
		parseBoolReady(parseView.CatalogServerSynced, "catalog: synced", "catalog: local"),
		parseWorkerState,
		parseResolveDashboardAsyncState(parseView),
	}
}

func parseResolveDevRuntimeStates(parseView appViewState) []string {
	parseCanvasState := "canvas: off"
	if parseView.CanvasSession.Active {
		parseCanvasState = "canvas: " + parseView.CanvasSession.LayoutMode
	}
	parseAdminState := "role: user"
	if parseView.IsSuperuser {
		parseAdminState = "role: superuser"
	} else if parseView.CanAccessAdmin {
		parseAdminState = "role: admin"
	}
	parseModel := strings.TrimSpace(parseView.SelectedModel)
	if parseModel == "" {
		parseModel = "(default)"
	}
	return []string{
		parseBoolReady(parseView.IsStreaming, "streaming: active", "streaming: idle"),
		"model: " + parseModel,
		parseAdminState,
		parseCanvasState,
		"messages: " + strconv.Itoa(len(parseView.Messages)),
	}
}

func parseBuildDevPanelRows(parseView appViewState) [][2]string {
	parseUserName := strings.TrimSpace(parseView.UserName)
	if parseUserName == "" {
		parseUserName = "guest"
	}
	parseRows := [][2]string{
		{"Route ID", parseResolveDevRouteID(parseView.CurrentPath)},
		{"Shell section", parseResolveDevShellSection(parseView)},
		{"Async resources", strings.Join(parseResolveDevAsyncResources(parseView), " | ")},
		{"Runtime states", strings.Join(parseResolveDevRuntimeStates(parseView), " | ")},
		{"Route", parseView.CurrentPath},
		{"Active conv ID", strconv.FormatInt(parseView.ActiveConversationID, 10)},
		{"Workspace", parseUserName},
	}
	if parseDashboardSlice := parseResolveDashboardSliceID(parseView.CurrentPath); parseDashboardSlice != "" {
		parseRows = append(parseRows, [2]string{"Dashboard slice", parseDashboardSlice})
	}
	if parseView.ActiveSettingsSection != "" {
		parseRows = append(parseRows, [2]string{"Settings section", parseView.ActiveSettingsSection})
	}
	return parseRows
}

// renderDevTourOverlay renders a dismissible framework-tour modal listing the major GWC pattern stops.
func renderDevTourOverlay(parseView appViewState) ui.Node {
	parseDismiss := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		parseWindow := js.Global()
		if !parseWindow.Truthy() {
			return
		}
		parseWindow.Get("history").Call("replaceState", js.Null(), "", parseDismissDevToolURL())
	})
	parseStops := parseBuildTourStops()
	return Div(
		ClassStr("fixed inset-0 z-[100] flex items-center justify-center bg-black/70 backdrop-blur-md"),
		OnClick(parseDismiss),
		Div(
			ClassStr("relative mx-4 w-full max-w-2xl rounded-2xl border border-white/[0.10] bg-[#0e0e16] p-6 shadow-2xl"),
			OnClick(ui.UseEvent(func(parseE ui.Event) { parseE.StopPropagation() })),
			// Header
			Div(
				ClassStr("mb-5 flex items-start justify-between gap-4"),
				Div(
					Div(ClassStr("text-[10px] uppercase tracking-[0.18em] text-[#8e7bff]/60"), Text("GoWebComponents framework tour")),
					H2(ClassStr("mt-1 text-lg font-semibold text-white"), Text("Example 100 — Pattern tour")),
					P(ClassStr("mt-1 text-xs text-white/40"), Text("Each stop maps a visible UI surface to the GWC pattern it demonstrates. Append ?gwc-dev=panel to inspect live runtime state.")),
				),
				Button(
					ClassStr("shrink-0 rounded-xl border border-white/10 bg-white/5 p-1.5 text-white/40 hover:text-white transition-colors"),
					OnClick(parseDismiss),
					Text("✕"),
				),
			),
			// Tour stops grid
			Div(
				ClassStr("grid grid-cols-1 gap-3 sm:grid-cols-2"),
				Map(parseStops, func(parseStop parseTourStop) ui.Node {
					return Div(
						ClassStr("rounded-xl border border-white/[0.07] bg-white/[0.03] p-4"),
						Div(ClassStr("mb-0.5 text-[10px] uppercase tracking-[0.14em] text-[#8e7bff]/60"), Text(parseStop.parsePattern)),
						Div(ClassStr("text-sm font-semibold text-white"), Text(parseStop.parseTitle)),
						P(ClassStr("mt-1 text-[11px] leading-4 text-white/40"), Text(parseStop.parseDesc)),
						Div(ClassStr("mt-2 font-mono text-[10px] text-white/20"), Text(parseStop.parseFile)),
					)
				}),
			),
			// Footer
			P(ClassStr("mt-4 text-[10px] text-white/20"), Text("Route: "+parseView.CurrentPath+" · Click outside or ✕ to dismiss")),
		),
	)
}

// ─── Runtime state debug panel ───────────────────────────────────────────────

// renderDevPanelOverlay renders a fixed sidebar panel showing key runtime states for local development.
func renderDevPanelOverlay(parseView appViewState) ui.Node {
	// panel dismisses the same way as tour
	parseDismiss := ui.UseEvent(func(parseE ui.Event) {
		_ = parseE
		parseWindow := js.Global()
		if !parseWindow.Truthy() {
			return
		}
		parseWindow.Get("history").Call("replaceState", js.Null(), "", parseDismissDevToolURL())
	})

	parseRows := parseBuildDevPanelRows(parseView)

	return Div(
		ClassStr("fixed bottom-4 right-4 z-[100] w-[min(28rem,calc(100vw-2rem))]"),
		Div(
			ClassStr("rounded-2xl border border-white/[0.10] bg-[#0e0e16] shadow-2xl"),
			// Panel header
			Div(
				ClassStr("flex items-center justify-between border-b border-white/[0.07] px-4 py-2.5"),
				Div(
					ClassStr("text-[10px] uppercase tracking-[0.18em] text-[#8e7bff]/60"),
					Text("GWC demo helper"),
				),
				Button(
					ClassStr("rounded-lg border border-white/10 bg-white/5 p-1 text-white/30 hover:text-white transition-colors"),
					OnClick(parseDismiss),
					Text("✕"),
				),
			),
			// State rows
			Div(
				ClassStr("px-4 py-3 space-y-1.5"),
				Map(parseRows, func(parseRow [2]string) ui.Node {
					return Div(
						ClassStr("rounded-xl border border-white/[0.05] bg-white/[0.02] px-3 py-2"),
						Span(ClassStr("block text-[10px] text-white/30"), Text(parseRow[0])),
						Span(ClassStr("mt-1 block break-words font-mono text-[10px] leading-4 text-white/60"), Text(parseRow[1])),
					)
				}),
			),
			// Footer hint
			P(ClassStr("border-t border-white/[0.06] px-4 py-2 text-[10px] text-white/20"), Text("?gwc-dev=tour for pattern tour, ?gwc-dev=demo-helper for this panel")),
		),
	)
}
