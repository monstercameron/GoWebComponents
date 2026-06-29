//go:build js && wasm

// Command agent-demo is a tiny app whose entire visible state lives in atoms,
// so an AI agent can change the title, counter, message, and colors live over
// the agent bridge while you watch in the browser.
package main

import (
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// demoShapeStyle returns the inline style for the shape-box, selected by the
// demo.shape atom. Changing the atom changes the rendered shape — all from the
// single app root, so it stays rock-solid under live agent drives.
func demoShapeStyle(parseShape string, parseBox string) map[string]string {
	parseStyle := map[string]string{
		"background":      parseBox,
		"color":           "#f8fafc",
		"display":         "flex",
		"align-items":     "center",
		"justify-content": "center",
		"box-shadow":      "0 22px 64px rgba(0,0,0,0.45)",
		"transition":      "all 450ms ease",
		"margin":          "8px auto 0",
	}
	switch parseShape {
	case "pill":
		parseStyle["border-radius"] = "999px"
		parseStyle["padding"] = "44px 96px"
	case "circle":
		parseStyle["border-radius"] = "50%"
		parseStyle["width"] = "260px"
		parseStyle["height"] = "260px"
	case "square":
		parseStyle["border-radius"] = "8px"
		parseStyle["width"] = "260px"
		parseStyle["height"] = "260px"
	default: // rounded
		parseStyle["border-radius"] = "28px"
		parseStyle["padding"] = "52px 72px"
	}
	return parseStyle
}

// DemoApp renders state driven entirely by agent-settable atoms.
func DemoApp() ui.Node {
	parseTitle := state.UseAtom[string]("demo.title", "Live Agent Bridge Demo")
	parseCount := state.UseAtom[float64]("demo.counter", 0)
	parseMessage := state.UseAtom[string]("demo.message", "Open this page, then watch an AI agent change it live.")
	parseColor := state.UseAtom[string]("demo.bg", "#0b1020")
	parseAccent := state.UseAtom[string]("demo.accent", "#38bdf8")
	parseBox := state.UseAtom[string]("demo.box", "#1e293b")
	parseShape := state.UseAtom[string]("demo.shape", "rounded")

	parseBump := ui.UseEvent(func() {
		parseCount.Set(parseCount.Get() + 1)
	})

	return Div(Style(map[string]string{
		"min-height":      "100vh",
		"margin":          "0",
		"background":      parseColor.Get(),
		"color":           "#e5e7eb",
		"font-family":     "ui-sans-serif, system-ui, -apple-system, sans-serif",
		"display":         "flex",
		"align-items":     "center",
		"justify-content": "center",
		"transition":      "background 400ms ease",
	}),
		Div(Style(map[string]string{"text-align": "center", "max-width": "680px", "padding": "32px"}),
			H1(Style(map[string]string{"font-size": "46px", "margin": "0 0 12px", "color": parseAccent.Get(), "transition": "color 400ms ease"}),
				Text(parseTitle.Get())),
			Div(Style(demoShapeStyle(parseShape.Get(), parseBox.Get())),
				Div(Style(map[string]string{"font-size": "104px", "font-weight": "800", "line-height": "1", "font-variant-numeric": "tabular-nums"}),
					Textf("%d", int(parseCount.Get()))),
			),
			P(Style(map[string]string{"font-size": "18px", "opacity": "0.85", "margin": "22px 0 24px", "line-height": "1.6"}),
				Text(parseMessage.Get())),
			Button(OnClick(parseBump), Style(map[string]string{
				"padding":       "10px 22px",
				"font-size":     "15px",
				"border-radius": "12px",
				"border":        "1px solid rgba(255,255,255,0.22)",
				"background":    "rgba(255,255,255,0.08)",
				"color":         "#ffffff",
				"cursor":        "pointer",
			}), Text("+1 (you)")),
			P(Style(map[string]string{"font-size": "12px", "opacity": "0.5", "margin-top": "26px", "letter-spacing": "0.06em"}),
				Text("atoms: demo.title · demo.counter · demo.message · demo.bg · demo.box · demo.accent · demo.shape")),
		),
	)
}

func main() {
	enableDemoAgentBridge()
	ui.Render(ui.CreateElement(DemoApp), "#app")
	select {}
}
