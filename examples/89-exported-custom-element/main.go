//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type exportedTileProps struct {
	Title  string
	Tone   string
	Detail string
}

func exportedStatusTile(props exportedTileProps) ui.Node {
	accent := "#67e8f9"
	border := "rgba(103,232,249,0.22)"
	background := "linear-gradient(180deg, rgba(8,15,29,0.96), rgba(15,23,42,0.82))"
	if props.Tone == "warning" {
		accent = "#f59e0b"
		border = "rgba(245,158,11,0.3)"
		background = "linear-gradient(180deg, rgba(69,26,3,0.92), rgba(120,53,15,0.78))"
	}
	if props.Tone == "critical" {
		accent = "#f87171"
		border = "rgba(248,113,113,0.32)"
		background = "linear-gradient(180deg, rgba(69,10,10,0.94), rgba(127,29,29,0.8))"
	}

	return html.Section(html.Props{},
		html.Style(html.Props{}, html.Text(fmt.Sprintf(`
			:host {
				display: block;
				font-family: "Segoe UI Variable", Aptos, "Trebuchet MS", sans-serif;
			}
			.tile {
				border-radius: 26px;
				border: 1px solid %s;
				background: %s;
				box-shadow: 0 20px 40px rgba(2, 6, 23, 0.28);
				padding: 22px;
				color: #e2e8f0;
			}
			.eyebrow {
				font-size: 11px;
				letter-spacing: 0.22em;
				text-transform: uppercase;
				color: %s;
			}
			h2 {
				margin: 12px 0 0;
				font-size: 1.6rem;
				line-height: 1.1;
				color: white;
			}
			p {
				margin: 12px 0 0;
				line-height: 1.7;
				color: #cbd5e1;
			}
			.pulse {
				margin-top: 18px;
				height: 10px;
				border-radius: 999px;
				background: linear-gradient(90deg, %s, rgba(255,255,255,0.18));
			}
		`, border, background, accent, accent))),
		html.Div(html.Props{Class: "tile"},
			html.P(html.Props{Class: "eyebrow"}, html.Text("Exported GoWebComponents widget")),
			html.H2(html.Props{}, html.Text(props.Title)),
			html.P(html.Props{}, html.Text(props.Detail)),
			html.Div(html.Props{Class: "pulse"}),
		),
	)
}

func exportedCustomElementExample() ui.Node {
	status := ui.UseState("Waiting for plain HTML hosts to request mounts.")
	mountCount := ui.UseState(0)

	ui.UseEffect(func() func() {
		mountFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) == 0 || args[0].IsUndefined() || args[0].IsNull() {
				status.Set("Mount request was missing a target root.")
				return nil
			}
			props := exportedTileProps{
				Title:  "Untitled tile",
				Tone:   "neutral",
				Detail: "No detail provided.",
			}
			if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
				attrs := args[1]
				if value := attrs.Get("title"); value.Type() == js.TypeString {
					props.Title = value.String()
				}
				if value := attrs.Get("tone"); value.Type() == js.TypeString {
					props.Tone = value.String()
				}
				if value := attrs.Get("detail"); value.Type() == js.TypeString {
					props.Detail = value.String()
				}
			}
			if err := ui.RenderInto(ui.CreateElement(exportedStatusTile, props), args[0]); err != nil {
				status.Set("Mount request failed.")
				return nil
			}
			mountCount.Update(func(previous int) int { return previous + 1 })
			status.Set("Mounted exported widget for plain HTML host.")
			return nil
		})
		unmountFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 && !args[0].IsUndefined() && !args[0].IsNull() {
				var empty ui.Node
				_ = ui.RenderInto(empty, args[0])
			}
			status.Set("Unmounted exported widget host.")
			return nil
		})

		js.Global().Set("__gwcMountStatusTile", mountFn)
		js.Global().Set("__gwcUnmountStatusTile", unmountFn)
		if ctor := js.Global().Get("CustomEvent"); ctor.Type() == js.TypeFunction {
			js.Global().Call("dispatchEvent", ctor.New("gwc-export-ready"))
		}

		return func() {
			js.Global().Set("__gwcMountStatusTile", js.Undefined())
			js.Global().Set("__gwcUnmountStatusTile", js.Undefined())
			mountFn.Release()
			unmountFn.Release()
		}
	}, true)

	return shared.ExamplePage(
		"Exported Custom Element Prototype",
		"ui.RenderInto and experimental custom-element export wrapper",
		"Prototype mounting a GoWebComponents subtree into a standards-based custom-element host so plain HTML can consume a shadow-root widget.",
		shared.ExamplePanel("Bridge status",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Mount calls", fmt.Sprintf("%d", mountCount.Get())),
				shared.ExampleStat("Last bridge state", status.Get()),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("The raw HTML hosts live below this app shell. Their wrapper class asks Go to render into a shadow root through ui.RenderInto, and replays that mount when observed attributes change.")),
		),
		shared.ExamplePanel("Why this is still experimental",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The prototype proves the runtime can target explicit DOM nodes cleanly, but the custom-element class, attribute observation rules, and shadow-root ownership are still example-level code rather than a committed framework API.")),
			shared.ExampleCode(
				`window.__gwcMountStatusTile(shadowRoot, { title, tone, detail })`,
				`ui.RenderInto(ui.CreateElement(exportedStatusTile, props), shadowRoot)`,
				`window.__gwcUnmountStatusTile(shadowRoot)`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(exportedCustomElementExample), "#app")
	select {}
}
