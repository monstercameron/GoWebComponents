package templatelowering

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type LandingProps struct {
	Eyebrow      string
	Headline     string
	Summary      string
	PrimaryTag   string
	SecondaryTag string
}

func RenderLanding(props LandingProps) ui.Node {
	return html.Section(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/85 p-8 shadow-2xl"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"},
			html.Text(props.Eyebrow)),
		html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"},
			html.Text(props.Headline)),
		html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"},
			html.Text(props.Summary)),
		html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
			html.Span(html.Props{Class: "rounded-full border border-cyan-400/20 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100"},
				html.Text(props.PrimaryTag)),
			html.Span(html.Props{Class: "rounded-full border border-emerald-400/20 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-100"},
				html.Text(props.SecondaryTag))))
}
