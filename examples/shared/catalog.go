package shared

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func ExamplePage(title, feature, summary string, content ...ui.Node) ui.Node {
	children := []ui.Node{
		html.Div(html.Props{Class: "mx-auto max-w-5xl px-6 py-12"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-[radial-gradient(circle_at_top,_rgba(59,130,246,0.22),_transparent_45%),rgba(15,23,42,0.92)] p-8 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text(feature)),
				html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text(title)),
				html.P(html.Props{Class: "mt-4 max-w-3xl text-lg leading-8 text-slate-300"}, html.Text(summary)),
			),
			html.Div(html.Props{Class: "mt-8 grid gap-6"}, content...),
		),
	}

	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] text-slate-100"}, children...)
}

func ExamplePanel(title string, body ...ui.Node) ui.Node {
	children := append([]ui.Node{
		html.H2(html.Props{Class: "text-xl font-bold text-white"}, html.Text(title)),
	}, body...)
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-white/5 p-6"}, children...)
}

func ExampleButton(label string, handler ui.Handler) ui.Node {
	return html.Button(
		html.Props{
			OnClick: handler,
			Class:   "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80",
		},
		html.Text(label),
	)
}

func ExampleStat(label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 p-4"},
		html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text(value)),
	)
}

func ExampleCode(lines ...string) ui.Node {
	children := make([]ui.Node, 0, len(lines))
	for _, line := range lines {
		children = append(children, html.Text(line+"\n"))
	}
	return html.Pre(html.Props{Class: "overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300"},
		html.Code(html.Props{}, children...),
	)
}
