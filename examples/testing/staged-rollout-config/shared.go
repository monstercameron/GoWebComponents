package main

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

const (
	rolloutConfigPayloadKey = "public-config"
	rolloutFlagsPayloadKey  = "rollout-flags"
	rolloutControlPath      = "/control"
	rolloutBetaPath         = "/beta"
)

type rolloutPublicConfig struct {
	Environment      string `json:"environment"`
	EnvironmentLabel string `json:"environmentLabel"`
	APIBaseURL       string `json:"apiBaseURL"`
}

type rolloutFlags struct {
	BetaRouteEnabled bool   `json:"betaRouteEnabled"`
	Cohort           string `json:"cohort"`
	EvaluationSource string `json:"evaluationSource"`
	RolloutStage     string `json:"rolloutStage"`
}

type rolloutBootstrapView struct {
	RoutePath    string
	Config       rolloutPublicConfig
	Flags        rolloutFlags
	FlagRevision string
}

func defaultRolloutView() rolloutBootstrapView {
	return rolloutBootstrapView{
		RoutePath: rolloutBetaPath,
		Config: rolloutPublicConfig{
			Environment:      "staging-us",
			EnvironmentLabel: "Staging US",
			APIBaseURL:       "https://staging-api.atlas.local/v1",
		},
		Flags: rolloutFlags{
			BetaRouteEnabled: true,
			Cohort:           "design-partners",
			EvaluationSource: "request bootstrap",
			RolloutStage:     "10% canary",
		},
		FlagRevision: "ring-03",
	}
}

func rolloutViewFromBootstrap(parsePayload ui.SSRBootstrap) rolloutBootstrapView {
	parseView := defaultRolloutView()
	if parsePath := strings.TrimSpace(parsePayload.Route.Path); parsePath != "" {
		parseView.RoutePath = parsePath
	}
	if parseConfigValue, parseOk, parseErr := ui.ReadBootstrapPayload[rolloutPublicConfig](parsePayload, rolloutConfigPayloadKey); parseErr == nil && parseOk {
		parseView.Config = parseConfigValue.Value
	}
	if parseFlagsValue, parseOk2, parseErr2 := ui.ReadBootstrapPayload[rolloutFlags](parsePayload, rolloutFlagsPayloadKey); parseErr2 == nil && parseOk2 {
		parseView.Flags = parseFlagsValue.Value
		if parseRevision := strings.TrimSpace(parseFlagsValue.Revision); parseRevision != "" {
			parseView.FlagRevision = parseRevision
		}
	}
	if strings.TrimSpace(parseView.RoutePath) == "" {
		parseView.RoutePath = rolloutControlPath
		if parseView.Flags.BetaRouteEnabled {
			parseView.RoutePath = rolloutBetaPath
		}
	}
	if parseView.RoutePath == rolloutBetaPath && !parseView.Flags.BetaRouteEnabled {
		parseView.RoutePath = rolloutControlPath
	}
	return parseView
}

func rolloutStat(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-lg font-black text-white md:text-xl"}, html.Text(parseValue)),
	)
}

func rolloutRoutePanel(parseView rolloutBootstrapView, parseRoutePath string) ui.Node {
	switch parseRoutePath {
	case rolloutBetaPath:
		return html.Div(html.Props{Class: "mt-6 rounded-[1.5rem] border border-cyan-400/20 bg-cyan-950/20 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Beta workspace route")),
			html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Feature-gated route is live")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-300"}, html.Text("The beta workspace route is only registered because the server-evaluated snapshot set betaRouteEnabled=true. The same evaluated decision also produced this server HTML and is read again before hydration, so the route remains available after startup instead of flickering closed.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				rolloutStat("Evaluation source", parseView.Flags.EvaluationSource),
				rolloutStat("Cohort", parseView.Flags.Cohort),
				rolloutStat("Route gate", "enabled"),
			),
		)
	case rolloutControlPath:
		return html.Div(html.Props{Class: "mt-6 rounded-[1.5rem] border border-emerald-400/20 bg-emerald-950/20 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-emerald-300"}, html.Text("Control route")),
			html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Always-on route reads the same snapshot")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-300"}, html.Text("This route stays available regardless of rollout state, but it still reads the same public config and flag snapshot that the server used for first paint. That keeps the environment label, API endpoint, and rollout status aligned after hydration.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				rolloutStat("Current gate", enabledLabel(parseView.Flags.BetaRouteEnabled)),
				rolloutStat("Public API", parseView.Config.APIBaseURL),
				rolloutStat("Revision", parseView.FlagRevision),
			),
		)
	default:
		return html.Div(html.Props{Class: "mt-6 rounded-[1.5rem] border border-amber-400/20 bg-amber-950/20 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-amber-300"}, html.Text("Unavailable route")),
			html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Route is outside the active rollout")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-300"}, html.Text("This path is not part of the current staged rollout snapshot. In a real app the server and client would both keep it unavailable until the next evaluated flag revision opens it.")),
		)
	}
}

func enabledLabel(isEnabled bool) string {
	if isEnabled {
		return "enabled"
	}
	return "disabled"
}

func renderRolloutPage(parseView rolloutBootstrapView, parseRoutePath string) ui.Node {
	parseLinkClass := "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200"
	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-5xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Staged rollout bootstrap")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Environment-aware staged rollout")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("This page restores a browser-safe config snapshot and evaluated rollout flags from ui.SSRBootstrap, then uses that same snapshot to keep the initial route and hydrated route table in sync.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					html.A(html.Props{Href: "#" + rolloutControlPath, Class: parseLinkClass}, html.Text("Control route")),
					html.A(html.Props{Href: "#" + rolloutBetaPath, Class: parseLinkClass}, html.Text("Beta workspace")),
				),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-5"},
					rolloutStat("Environment", parseView.Config.EnvironmentLabel),
					rolloutStat("Public API", parseView.Config.APIBaseURL),
					rolloutStat("Rollout stage", parseView.Flags.RolloutStage),
					rolloutStat("Flag revision", parseView.FlagRevision),
					rolloutStat("Current route", parseRoutePath),
				),
				rolloutRoutePanel(parseView, parseRoutePath),
				html.Div(html.Props{Class: "mt-6 rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("SSR consistency contract")),
					html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-300"}, html.Text("The bootstrap payload carries only browser-safe data: environment label, public API base URL, rollout stage, and the evaluated boolean gate. Secret targeting inputs stay on the server; the browser only sees the public decision it needs to render and navigate consistently.")),
				),
			),
		),
	)
}
