//go:build js && wasm
// +build js,wasm

package exampleboot

import "github.com/monstercameron/GoWebComponents/v6/router"

// RenderExampleRouter mounts the router into the current example host selector.
func RenderExampleRouter(parseRouter *router.Router) {
	parseRouter.Mount(GetExampleMountSelector())
}

// ApplyExampleHydratedRouter binds a hydrated router to the current example host selector.
func ApplyExampleHydratedRouter(parseRouter *router.Router) {
	parseRouter.HydrateMount(GetExampleMountSelector())
}
