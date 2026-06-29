//go:build js && wasm

// Command global-events is the e2e fixture for G9 (ui.UseGlobalKey): a child
// component owns a document-level keydown listener and increments a counter that
// lives in the parent (so it survives the child's unmount/remount). This lets the
// e2e prove both that the listener fires and that it is cleaned up on unmount —
// if the js.Func leaked, a single keypress after remount would double-count.
package main

import (
	"strconv"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type keyListenerProps struct {
	onKey func()
}

func keyListener(parseProps keyListenerProps) ui.Node {
	ui.UseGlobalKey(func(parseEvent ui.KeyboardEvent) {
		parseProps.onKey()
	})
	return Span(FromProps(Props{ID: "listener-marker"}), "listening")
}

func App() ui.Node {
	parseCount := ui.UseState(0)
	parseListening := ui.UseState(true)

	parseArgs := []any{
		FromProps(Props{ID: "app-root"}),
		Button(
			FromProps(Props{ID: "toggle", Type: "button", OnClick: ui.UseEvent(func() { parseListening.Set(!parseListening.Get()) })}),
			"toggle",
		),
		Span(FromProps(Props{ID: "count"}), strconv.Itoa(parseCount.Get())),
	}
	if parseListening.Get() {
		parseArgs = append(parseArgs, ui.CreateElement(keyListener, keyListenerProps{
			onKey: func() { parseCount.Set(parseCount.Get() + 1) },
		}))
	}
	return Div(parseArgs...)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
