//go:build js && wasm
// +build js,wasm

// Package main is a minimal GWC wasm fixture that exercises the UseSpring hook.
// It mounts a component that animates a value from 0 toward 300 when the user
// (or test) clicks #spring-go. The current rounded position is rendered into
// #spring-value, and #spring-done shows "settled" once the value reaches its
// target.
package main

import (
	"fmt"
	"math"

	"github.com/monstercameron/GoWebComponents/v5/anim"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

// parseSpringProps holds the (empty) props for the spring demo component.
type parseSpringProps struct{}

const parseAnimTarget = 300.0

// parseSpringComponent animates a value from 0 toward parseAnimTarget via
// UseSpring. It renders:
//
//   - #spring-value: the current rounded position as text
//   - #spring-go:    a button that sets the target and starts the animation
//   - #spring-done:  shows "settled" once the value is within 1 unit of the target
func parseSpringComponent(_ parseSpringProps) ui.Node {
	// parseActive tracks whether the animation has been started.
	parseActive := ui.UseState(false)

	// When active, animate toward the target; otherwise stay at 0.
	var parseTarget float64
	if parseActive.Get() {
		parseTarget = parseAnimTarget
	}

	parsePos := ui.UseSpring(parseTarget, anim.WobblySpring())

	parseRounded := math.Round(parsePos)
	parseDoneText := ""
	if parseActive.Get() && math.Abs(parsePos-parseAnimTarget) < 1.0 {
		parseDoneText = "settled"
	}

	return html.Div(html.Props{ID: "spring-root"},
		html.Div(html.Props{ID: "spring-value"},
			html.Text(fmt.Sprintf("%d", int(parseRounded))),
		),
		html.Button(html.Props{
			ID: "spring-go",
			OnClick: ui.WrapHandler(func() {
				parseActive.Set(true)
			}),
		},
			html.Text("Go"),
		),
		html.Div(html.Props{ID: "spring-done"},
			html.Text(parseDoneText),
		),
	)
}

func main() {
	ui.Render(ui.CreateElement(parseSpringComponent, parseSpringProps{}), "#app")
	utils.WaitForever()
}
