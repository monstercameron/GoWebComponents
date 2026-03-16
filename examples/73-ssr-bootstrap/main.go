//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func setClientStatus(message string) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	status := document.Call("getElementById", "client-status")
	if status.Truthy() {
		status.Set("textContent", message)
	}
}

func main() {
	utils.DisableAllDebug()
	payload, err := ui.ReadBootstrapScript("")
	if err != nil {
		setClientStatus("Failed to read inline bootstrap payload")
		select {}
	}
	_, err = ui.Hydrate(renderBootstrapView(bootstrapViewFromPayload(payload)), "#app", ui.HydrationOptions{Bootstrap: payload})
	if err != nil {
		setClientStatus("Hydration failed")
		select {}
	}
	setClientStatus("Hydrated from inline bootstrap payload")
	select {}
}