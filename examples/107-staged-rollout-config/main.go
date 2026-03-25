//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func setRolloutStatus(parseMessage string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseStatus := parseDocument.Call("getElementById", "client-status")
	if parseStatus.Truthy() {
		parseStatus.Set("textContent", parseMessage)
	}
}

func ensureRolloutHash(parsePath string) {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.Get("hash").String() == "" {
		parseLocation.Set("hash", "#"+parsePath)
	}
}

func rolloutPage(parseView rolloutBootstrapView, parseRoutePath string) *router.Element {
	return ui.CreateElement(func() ui.Node {
		return renderRolloutPage(parseView, parseRoutePath)
	})
}

func main() {
	utils.DisableAllDebug()

	parseBootstrap, parseErr := ui.ReadBootstrapScript("")
	if parseErr != nil {
		setRolloutStatus("Failed to read staged rollout bootstrap")
		select {}
	}

	parseView := rolloutViewFromBootstrap(parseBootstrap)
	ensureRolloutHash(parseView.RoutePath)

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: rolloutControlPath})
	parseR.Register(rolloutControlPath, func(router.Attrs) *router.Element {
		return rolloutPage(parseView, rolloutControlPath)
	})
	if parseView.Flags.BetaRouteEnabled {
		parseR.Register(rolloutBetaPath, func(router.Attrs) *router.Element {
			return rolloutPage(parseView, rolloutBetaPath)
		})
	}
	parseR.Register("*", func(router.Attrs) *router.Element {
		return rolloutPage(parseView, "*")
	})

	parseRoot := ui.CreateElement(func() ui.Node { return parseR.Current() })
	if _, parseErr2 := ui.Hydrate(parseRoot, "#app", ui.HydrationOptions{Bootstrap: parseBootstrap}); parseErr2 != nil {
		setRolloutStatus("Hydration failed")
		select {}
	}
	parseR.HydrateMount("#app")
	setRolloutStatus("Hydrated with the same public config and flag snapshot used for SSR")
	select {}
}
