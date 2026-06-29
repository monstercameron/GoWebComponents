//go:build js && wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func instructionPage() ui.Node {
	return html.Div(html.Props{Class: "min-h-screen bg-[#08111d] px-6 py-16 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-3xl rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Server example")),
			html.H1(html.Props{Class: "mt-4 text-5xl font-black tracking-tight text-white"}, html.Text("ui.RenderToString")),
			html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text("This example is meant to be run as a small Go HTTP server so the server-side HTML render is the main teaching surface.")),
			html.Div(html.Props{Class: "mt-6 rounded-2xl border border-white/10 bg-white/5 p-4 text-sm leading-7 text-slate-300"},
				html.P(html.Props{}, html.Text("Run go run ./examples/server/render-to-string and open http://127.0.0.1:8084/.")),
				html.P(html.Props{Class: "mt-3"}, html.Text("The server page shows the exact HTML string returned by ui.RenderToString beside the rendered preview.")),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(instructionPage), "#app")
	select {}
}
