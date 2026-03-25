package main

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
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

func rolloutViewFromBootstrap(payload ui.SSRBootstrap) rolloutBootstrapView {
	view := defaultRolloutView()
	if path := strings.TrimSpace(payload.Route.Path); path != "" {
		view.RoutePath = path
	}
	if configValue, ok, err := ui.ReadBootstrapPayload[rolloutPublicConfig](payload, rolloutConfigPayloadKey); err == nil && ok {
		view.Config = configValue.Value
	}
	if flagsValue, ok, err := ui.ReadBootstrapPayload[rolloutFlags](payload, rolloutFlagsPayloadKey); err == nil && ok {
		view.Flags = flagsValue.Value
		if revision := strings.TrimSpace(flagsValue.Revision); revision != "" {
			view.FlagRevision = revision
		}
	}
	if strings.TrimSpace(view.RoutePath) == "" {
		view.RoutePath = rolloutControlPath
		if view.Flags.BetaRouteEnabled {
			view.RoutePath = rolloutBetaPath
		}
	}
	if view.RoutePath == rolloutBetaPath && !view.Flags.BetaRouteEnabled {
		view.RoutePath = rolloutControlPath
	}
	return view
}

func rolloutStat(label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{Class: "mt-3 text-lg font-black text-white md:text-xl"}, html.Text(value)),
	)
}

func rolloutRoutePanel(view rolloutBootstrapView, routePath string) ui.Node {
	switch routePath {
	case rolloutBetaPath:
		return html.Div(html.Props{Class: "mt-6 rounded-[1.5rem] border border-cyan-400/20 bg-cyan-950/20 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Beta workspace route")),
			html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Feature-gated route is live")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-300"}, html.Text("The beta workspace route is only registered because the server-evaluated snapshot set betaRouteEnabled=true. The same evaluated decision also produced this server HTML and is read again before hydration, so the route remains available after startup instead of flickering closed.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				rolloutStat("Evaluation source", view.Flags.EvaluationSource),
				rolloutStat("Cohort", view.Flags.Cohort),
				rolloutStat("Route gate", "enabled"),
			),
		)
	case rolloutControlPath:
		return html.Div(html.Props{Class: "mt-6 rounded-[1.5rem] border border-emerald-400/20 bg-emerald-950/20 p-6"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-emerald-300"}, html.Text("Control route")),
			html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Always-on route reads the same snapshot")),
			html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-300"}, html.Text("This route stays available regardless of rollout state, but it still reads the same public config and flag snapshot that the server used for first paint. That keeps the environment label, API endpoint, and rollout status aligned after hydration.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				rolloutStat("Current gate", enabledLabel(view.Flags.BetaRouteEnabled)),
				rolloutStat("Public API", view.Config.APIBaseURL),
				rolloutStat("Revision", view.FlagRevision),
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

func enabledLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func renderRolloutPage(view rolloutBootstrapView, routePath string) ui.Node {
	linkClass := "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200"
	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-5xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Staged rollout bootstrap")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Environment-aware staged rollout")),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text("This page restores a browser-safe config snapshot and evaluated rollout flags from ui.SSRBootstrap, then uses that same snapshot to keep the initial route and hydrated route table in sync.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					html.A(html.Props{Href: "#" + rolloutControlPath, Class: linkClass}, html.Text("Control route")),
					html.A(html.Props{Href: "#" + rolloutBetaPath, Class: linkClass}, html.Text("Beta workspace")),
				),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-5"},
					rolloutStat("Environment", view.Config.EnvironmentLabel),
					rolloutStat("Public API", view.Config.APIBaseURL),
					rolloutStat("Rollout stage", view.Flags.RolloutStage),
					rolloutStat("Flag revision", view.FlagRevision),
					rolloutStat("Current route", routePath),
				),
				rolloutRoutePanel(view, routePath),
				html.Div(html.Props{Class: "mt-6 rounded-[1.5rem] border border-white/10 bg-white/5 p-6"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("SSR consistency contract")),
					html.P(html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-300"}, html.Text("The bootstrap payload carries only browser-safe data: environment label, public API base URL, rollout stage, and the evaluated boolean gate. Secret targeting inputs stay on the server; the browser only sees the public decision it needs to render and navigate consistently.")),
				),
			),
		),
	)
}
