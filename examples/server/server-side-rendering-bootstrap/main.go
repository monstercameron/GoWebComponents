//go:build js && wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func setClientStatus(parseMessage string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseStatus := parseDocument.Call("getElementById", "client-status")
	if parseStatus.Truthy() {
		parseStatus.Set("textContent", parseMessage)
	}
}

func main() {
	utils.DisableAllDebug()
	parsePayload, parseErr := ui.ReadBootstrapScript("")
	if parseErr != nil {
		setClientStatus("Failed to read inline bootstrap payload")
		select {}
	}
	_, parseErr = ui.Hydrate(renderBootstrapView(bootstrapViewFromPayload(parsePayload)), "#app", ui.HydrationOptions{Bootstrap: parsePayload})
	if parseErr != nil {
		setClientStatus("Hydration failed")
		select {}
	}
	setClientStatus("Hydrated from inline bootstrap payload")
	select {}
}
