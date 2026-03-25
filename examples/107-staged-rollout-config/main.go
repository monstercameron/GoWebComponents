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

func setRolloutStatus(message string) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	status := document.Call("getElementById", "client-status")
	if status.Truthy() {
		status.Set("textContent", message)
	}
}

func ensureRolloutHash(path string) {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return
	}
	location := window.Get("location")
	if location.Get("hash").String() == "" {
		location.Set("hash", "#"+path)
	}
}

func rolloutPage(view rolloutBootstrapView, routePath string) *router.Element {
	return ui.CreateElement(func() ui.Node {
		return renderRolloutPage(view, routePath)
	})
}

func main() {
	utils.DisableAllDebug()

	bootstrap, err := ui.ReadBootstrapScript("")
	if err != nil {
		setRolloutStatus("Failed to read staged rollout bootstrap")
		select {}
	}

	view := rolloutViewFromBootstrap(bootstrap)
	ensureRolloutHash(view.RoutePath)

	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: rolloutControlPath})
	r.Register(rolloutControlPath, func(router.Attrs) *router.Element {
		return rolloutPage(view, rolloutControlPath)
	})
	if view.Flags.BetaRouteEnabled {
		r.Register(rolloutBetaPath, func(router.Attrs) *router.Element {
			return rolloutPage(view, rolloutBetaPath)
		})
	}
	r.Register("*", func(router.Attrs) *router.Element {
		return rolloutPage(view, "*")
	})

	root := ui.CreateElement(func() ui.Node { return r.Current() })
	if _, err := ui.Hydrate(root, "#app", ui.HydrationOptions{Bootstrap: bootstrap}); err != nil {
		setRolloutStatus("Hydration failed")
		select {}
	}
	r.HydrateMount("#app")
	setRolloutStatus("Hydrated with the same public config and flag snapshot used for SSR")
	select {}
}
