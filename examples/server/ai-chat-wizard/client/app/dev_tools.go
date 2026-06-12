//go:build js && wasm

package app

import (
	"strconv"
	"strings"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// ─── Dev-tool query param ────────────────────────────────────────────────────
//
// Add ?gwc-dev=tour to any route to open the framework tour overlay.
// Add ?gwc-dev=panel to any route to open the runtime state debug panel.
// Both are dismissible and do not affect normal product use.

const devToolQueryKey = "gwc-dev"
const devToolTourMode = "tour"
const devToolPanelMode = "panel"

// parseGetDevToolMode reads the ?gwc-dev= query param from window.location.search.
// Returns empty string when not present or when running outside a browser context.
func parseGetDevToolMode() string {
	parseWindow := js.Global()
	if !parseWindow.Truthy() {
		return ""
	}
	parseSearch := parseWindow.Get("location").Get("search").String()
	for _, parsePart := range strings.Split(strings.TrimPrefix(parseSearch, "?"), "&") {
		parseKV := strings.SplitN(parsePart, "=", 2)
		if len(parseKV) == 2 && parseKV[0] == devToolQueryKey {
			return parseKV[1]
		}
	}
	return ""
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

// renderDevTourOverlay renders a dismissible framework-tour modal listing the major GWC pattern stops.
func renderDevTourOverlay(parseView appViewState) ui.Node {
	parseDismiss := ui.UseEvent(func(parseE ui.Event) {
		parseWindow := js.Global()
		if !parseWindow.Truthy() {
			return
		}
		// Remove the gwc-dev param from the URL without triggering a navigation.
		parseLoc := parseWindow.Get("location")
		parseSearch := parseLoc.Get("search").String()
		parseNewSearch := ""
		parseParts := strings.Split(strings.TrimPrefix(parseSearch, "?"), "&")
		parseKept := []string{}
		for _, parsePart := range parseParts {
			if !strings.HasPrefix(parsePart, devToolQueryKey+"=") && parsePart != "" {
				parseKept = append(parseKept, parsePart)
			}
		}
		if len(parseKept) > 0 {
			parseNewSearch = "?" + strings.Join(parseKept, "&")
		}
		parseWindow.Get("history").Call("replaceState", js.Null(), "", parseLoc.Get("pathname").String()+parseNewSearch)
	})
	parseStops := parseBuildTourStops()
	return Div(
		Class("fixed inset-0 z-[100] flex items-center justify-center bg-black/70 backdrop-blur-md"),
		OnClick(parseDismiss),
		Div(
			Class("relative mx-4 w-full max-w-2xl rounded-2xl border border-white/[0.10] bg-[#0e0e16] p-6 shadow-2xl"),
			OnClick(ui.UseEvent(func(parseE ui.Event) { parseE.StopPropagation() })),
			// Header
			Div(
				Class("mb-5 flex items-start justify-between gap-4"),
				Div(
					Div(Class("text-[10px] uppercase tracking-[0.18em] text-[#8e7bff]/60"), Text("GoWebComponents framework tour")),
					H2(Class("mt-1 text-lg font-semibold text-white"), Text("Example 100 — Pattern tour")),
					P(Class("mt-1 text-xs text-white/40"), Text("Each stop maps a visible UI surface to the GWC pattern it demonstrates. Append ?gwc-dev=panel to inspect live runtime state.")),
				),
				Button(
					Class("shrink-0 rounded-xl border border-white/10 bg-white/5 p-1.5 text-white/40 hover:text-white transition-colors"),
					OnClick(parseDismiss),
					Text("✕"),
				),
			),
			// Tour stops grid
			Div(
				Class("grid grid-cols-1 gap-3 sm:grid-cols-2"),
				Map(parseStops, func(parseStop parseTourStop) ui.Node {
					return Div(
						Class("rounded-xl border border-white/[0.07] bg-white/[0.03] p-4"),
						Div(Class("mb-0.5 text-[10px] uppercase tracking-[0.14em] text-[#8e7bff]/60"), Text(parseStop.parsePattern)),
						Div(Class("text-sm font-semibold text-white"), Text(parseStop.parseTitle)),
						P(Class("mt-1 text-[11px] leading-4 text-white/40"), Text(parseStop.parseDesc)),
						Div(Class("mt-2 font-mono text-[10px] text-white/20"), Text(parseStop.parseFile)),
					)
				}),
			),
			// Footer
			P(Class("mt-4 text-[10px] text-white/20"), Text("Route: "+parseView.CurrentPath+" · Click outside or ✕ to dismiss")),
		),
	)
}

// ─── Runtime state debug panel ───────────────────────────────────────────────

// renderDevPanelOverlay renders a fixed sidebar panel showing key runtime states for local development.
func renderDevPanelOverlay(parseView appViewState) ui.Node {
	// panel dismisses the same way as tour
	parseDismiss := ui.UseEvent(func(parseE ui.Event) {
		parseWindow := js.Global()
		if !parseWindow.Truthy() {
			return
		}
		parseLoc := parseWindow.Get("location")
		parseSearch := parseLoc.Get("search").String()
		parseKept := []string{}
		for _, parsePart := range strings.Split(strings.TrimPrefix(parseSearch, "?"), "&") {
			if !strings.HasPrefix(parsePart, devToolQueryKey+"=") && parsePart != "" {
				parseKept = append(parseKept, parsePart)
			}
		}
		parseNewSearch := ""
		if len(parseKept) > 0 {
			parseNewSearch = "?" + strings.Join(parseKept, "&")
		}
		parseWindow.Get("history").Call("replaceState", js.Null(), "", parseLoc.Get("pathname").String()+parseNewSearch)
	})

	parseBool := func(parseB bool) string {
		if parseB {
			return "yes"
		}
		return "no"
	}
	parseRows := [][2]string{
		{"Route", parseView.CurrentPath},
		{"Auth resolved", parseBool(parseView.AuthResolved)},
		{"Authenticated", parseBool(parseView.Authenticated)},
		{"GRPC ready", parseBool(parseView.GRPCReady)},
		{"Catalog synced", parseBool(parseView.CatalogServerSynced)},
		{"Worker fallback", parseBool(parseView.MarkdownWorkerFallback)},
		{"Streaming", parseBool(parseView.IsStreaming)},
		{"Selected model", parseView.SelectedModel},
		{"Workspace", parseView.UserName + "'s workspace"},
		{"Can access admin", parseBool(parseView.CanAccessAdmin)},
		{"Is superuser", parseBool(parseView.IsSuperuser)},
		{"Canvas active", parseBool(parseView.CanvasSession.Active)},
		{"Active conv ID", strconv.FormatInt(parseView.ActiveConversationID, 10)},
	}

	return Div(
		Class("fixed bottom-4 right-4 z-[100] w-72"),
		Div(
			Class("rounded-2xl border border-white/[0.10] bg-[#0e0e16] shadow-2xl"),
			// Panel header
			Div(
				Class("flex items-center justify-between border-b border-white/[0.07] px-4 py-2.5"),
				Div(
					Class("text-[10px] uppercase tracking-[0.18em] text-[#8e7bff]/60"),
					Text("GWC dev panel"),
				),
				Button(
					Class("rounded-lg border border-white/10 bg-white/5 p-1 text-white/30 hover:text-white transition-colors"),
					OnClick(parseDismiss),
					Text("✕"),
				),
			),
			// State rows
			Div(
				Class("px-4 py-3 space-y-1.5"),
				Map(parseRows, func(parseRow [2]string) ui.Node {
					return Div(
						Class("flex items-baseline justify-between gap-2"),
						Span(Class("text-[10px] text-white/30"), Text(parseRow[0])),
						Span(Class("max-w-[55%] truncate text-right font-mono text-[10px] text-white/60"), Text(parseRow[1])),
					)
				}),
			),
			// Footer hint
			P(Class("border-t border-white/[0.06] px-4 py-2 text-[10px] text-white/20"), Text("?gwc-dev=tour for pattern tour")),
		),
	)
}
