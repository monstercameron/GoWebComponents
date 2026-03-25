//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type ratingChange struct {
	Score   int    `json:"score"`
	Source  string `json:"source"`
	Palette string `json:"palette"`
}

func webComponentsExample() ui.Node {
	palette := ui.UseState("ocean")
	score := ui.UseState(3)
	lastEvent := ui.UseState(ratingChange{Score: 3, Source: "initial", Palette: "ocean"})
	status := ui.UseState("Waiting for the widget to emit rating-change.")

	setOcean := ui.UseEvent(func() {
		palette.Set("ocean")
		status.Set("Palette updated from Go.")
	})
	setSunset := ui.UseEvent(func() {
		palette.Set("sunset")
		status.Set("Palette updated from Go.")
	})
	resetScore := ui.UseEvent(func() {
		score.Set(3)
		status.Set("Score reset from Go.")
	})

	ui.UseEffect(func() func() {
		document, err := interop.GetDocument()
		if err != nil {
			status.Set("Browser interop is unavailable in this build.")
			return nil
		}
		host, ok, err := document.ElementByID("rating-card")
		if err != nil {
			status.Set("Failed to resolve the custom-element host.")
			return nil
		}
		if !ok {
			status.Set("The custom-element host has not mounted yet.")
			return nil
		}
		events, err := host.Events()
		if err != nil {
			status.Set("Failed to acquire host events.")
			return nil
		}
		subscription, err := interop.SubscribeDecoded(events, "rating-change", func(event interop.DecodedCustomEvent[ratingChange], err error) {
			if err != nil {
				status.Set("Failed to decode rating-change detail.")
				return
			}
			score.Set(event.Detail.Score)
			lastEvent.Set(event.Detail)
			status.Set(fmt.Sprintf("Received rating-change from %s.", event.Detail.Source))
		})
		if err != nil {
			status.Set("Failed to subscribe to rating-change.")
			return nil
		}
		return func() {
			subscription.Cancel()
		}
	}, "rating-card")

	card := html.CustomElement("demo-rating-card", html.CustomElementProps{
		Props: html.Props{
			ID:    "rating-card",
			Class: "mt-6 block",
		},
		Attributes: map[string]string{
			"palette": palette.Get(),
		},
		Presence: map[string]bool{
			"interactive": true,
		},
		Properties: map[string]interface{}{
			"score": score.Get(),
			"config": map[string]interface{}{
				"headline": "Warehouse confidence",
				"caption":  "Custom element owned by browser code, driven from Go state.",
			},
		},
	},
		html.Div(html.Props{Slot: "summary", Class: "rounded-2xl border border-cyan-300/15 bg-slate-950/60 px-4 py-3 text-sm text-slate-200"},
			html.Text("The summary slot is regular GoWebComponents light DOM."),
		),
		html.Button(html.Props{
			Slot:    "actions",
			OnClick: resetScore,
			Class:   "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80",
		}, html.Text("Reset From Go")),
	)

	return shared.ExamplePage(
		"Web Components",
		"html.CustomElement and interop.SubscribeDecoded",
		"Consume a browser-defined custom element with reflected attributes, property-only config, named slots, and typed CustomEvent payloads.",
		shared.ExamplePanel("Widget host",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The browser-defined <demo-rating-card> element reacts to palette as an attribute, score and config as client-only properties, and emits typed rating-change events back into Go.")),
			html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-3"},
				shared.ExampleButton("Ocean palette", setOcean),
				shared.ExampleButton("Sunset palette", setSunset),
			),
			card,
		),
		shared.ExamplePanel("Live state",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Score", fmt.Sprintf("%d", score.Get())),
				shared.ExampleStat("Palette", palette.Get()),
				shared.ExampleStat("Event source", lastEvent.Get().Source),
			),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(status.Get())),
		),
		shared.ExamplePanel("Integration shape",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("This is the supported browser-interop path for third-party web components: render the host declaratively, subscribe through interop during an effect, and cancel that subscription during cleanup.")),
			shared.ExampleCode(
				`html.CustomElement("demo-rating-card", html.CustomElementProps{`,
				`    Attributes: map[string]string{"palette": palette},`,
				`    Properties: map[string]interface{}{"score": score, "config": config},`,
				`}, html.Div(html.Props{Slot: "summary"}, html.Text("Light DOM slot")))`,
				`interop.SubscribeDecoded[ratingChange](events, "rating-change", handler)`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(webComponentsExample), "#app")
	select {}
}
