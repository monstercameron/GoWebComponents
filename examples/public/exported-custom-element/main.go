//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

type exportedTileProps struct {
	Title  string
	Tone   string
	Detail string
}

func exportedStatusTile(parseProps exportedTileProps) ui.Node {
	parseAccent := "#67e8f9"
	parseBorder := "rgba(103,232,249,0.22)"
	parseBackground := "linear-gradient(180deg, rgba(8,15,29,0.96), rgba(15,23,42,0.82))"
	if parseProps.Tone == "warning" {
		parseAccent = "#f59e0b"
		parseBorder = "rgba(245,158,11,0.3)"
		parseBackground = "linear-gradient(180deg, rgba(69,26,3,0.92), rgba(120,53,15,0.78))"
	}
	if parseProps.Tone == "critical" {
		parseAccent = "#f87171"
		parseBorder = "rgba(248,113,113,0.32)"
		parseBackground = "linear-gradient(180deg, rgba(69,10,10,0.94), rgba(127,29,29,0.8))"
	}

	return html.Section(html.Props{},
		html.Tag("style", html.Props{}, html.Text(fmt.Sprintf(`
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
		`, parseBorder, parseBackground, parseAccent, parseAccent))),
		html.Div(html.Props{Class: "tile"},
			html.P(html.Props{Class: "eyebrow"}, html.Text("Exported GoWebComponents widget")),
			html.H2(html.Props{}, html.Text(parseProps.Title)),
			html.P(html.Props{}, html.Text(parseProps.Detail)),
			html.Div(html.Props{Class: "pulse"}),
		),
	)
}

func exportedCustomElementExample() ui.Node {
	parseStatus := ui.UseState("Waiting for plain HTML hosts to request mounts.")
	parseMountCount := ui.UseState(0)

	ui.UseEffect(func() func() {
		parseMountFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			if len(parseArgs) == 0 || parseArgs[0].IsUndefined() || parseArgs[0].IsNull() {
				parseStatus.Set("Mount request was missing a target root.")
				return nil
			}
			parseProps := exportedTileProps{
				Title:  "Untitled tile",
				Tone:   "neutral",
				Detail: "No detail provided.",
			}
			if len(parseArgs) > 1 && !parseArgs[1].IsUndefined() && !parseArgs[1].IsNull() {
				parseAttrs := parseArgs[1]
				if parseValue := parseAttrs.Get("title"); parseValue.Type() == js.TypeString {
					parseProps.Title = parseValue.String()
				}
				if parseValue2 := parseAttrs.Get("tone"); parseValue2.Type() == js.TypeString {
					parseProps.Tone = parseValue2.String()
				}
				if parseValue3 := parseAttrs.Get("detail"); parseValue3.Type() == js.TypeString {
					parseProps.Detail = parseValue3.String()
				}
			}
			if parseErr := ui.RenderInto(ui.CreateElement(exportedStatusTile, parseProps), parseArgs[0]); parseErr != nil {
				parseStatus.Set("Mount request failed.")
				return nil
			}
			parseMountCount.Update(func(parsePrevious int) int { return parsePrevious + 1 })
			parseStatus.Set("Mounted exported widget for plain HTML host.")
			return nil
		})
		parseUnmountFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if len(parseArgs2) > 0 && !parseArgs2[0].IsUndefined() && !parseArgs2[0].IsNull() {
				var parseEmpty ui.Node
				_ = ui.RenderInto(parseEmpty, parseArgs2[0])
			}
			parseStatus.Set("Unmounted exported widget host.")
			return nil
		})

		js.Global().Set("__gwcMountStatusTile", parseMountFn)
		js.Global().Set("__gwcUnmountStatusTile", parseUnmountFn)
		if parseCtor := js.Global().Get("CustomEvent"); parseCtor.Type() == js.TypeFunction {
			js.Global().Call("dispatchEvent", parseCtor.New("gwc-export-ready"))
		}

		return func() {
			js.Global().Set("__gwcMountStatusTile", js.Undefined())
			js.Global().Set("__gwcUnmountStatusTile", js.Undefined())
			parseMountFn.Release()
			parseUnmountFn.Release()
		}
	}, true)

	return shared.ExamplePage(
		"Exported Custom Element Prototype",
		"ui.RenderInto and experimental custom-element export wrapper",
		"Prototype mounting a GoWebComponents subtree into a standards-based custom-element host so plain HTML can consume a shadow-root widget.",
		shared.ExamplePanel("Bridge status",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Mount calls", fmt.Sprintf("%d", parseMountCount.Get())),
				shared.ExampleStat("Last bridge state", parseStatus.Get()),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(exportedCustomElementExample))
	exampleboot.WaitExampleRuntime()
}
