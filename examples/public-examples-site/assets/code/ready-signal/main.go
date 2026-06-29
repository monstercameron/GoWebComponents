//go:build js && wasm

// Command ready-signal is the e2e fixture for G16 (ui.OnReady + the gwc:ready
// DOM event): it records the Go ready callback firing, and the host page listens
// for the dispatched gwc:ready event — both after the first commit.
package main

import (
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

func App() ui.Node {
	return Div(FromProps(Props{ID: "app-root"}), Text("ready-demo"))
}

func main() {
	ui.OnReady(func() {
		js.Global().Set("__goReady", true)
	})
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
