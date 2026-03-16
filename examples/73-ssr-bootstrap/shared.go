package main

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type bootstrapView struct {
	Path    string
	Message string
}

func bootstrapViewFromPayload(payload ui.SSRBootstrap) bootstrapView {
	view := bootstrapView{Path: payload.Route.Path, Message: "inline bootstrap payload"}
	if view.Path == "" {
		view.Path = "/bootstrap"
	}
	if value, ok := payload.Data["message"].(string); ok && value != "" {
		view.Message = value
	}
	return view
}

func renderBootstrapView(view bootstrapView) ui.Node {
	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-3xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Inline bootstrap")),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("Dedicated SSR bootstrap")),
				html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text("This page was rendered on the server with ui.RenderToString, then paired with an inline bootstrap script emitted by ui.RenderBootstrapScript before the wasm client resumed it.")),
				html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Bootstrap route")),
						html.P(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(view.Path)),
					),
					html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-white/5 p-4"},
						html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Bootstrap message")),
						html.P(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text(view.Message)),
					),
				),
				html.P(html.Props{Class: "mt-6 text-sm leading-7 text-slate-400"}, html.Text("The client reads the inline JSON bootstrap first, then calls ui.Hydrate with the same payload so the already-rendered DOM can be reused.")),
			),
		),
	)
}
