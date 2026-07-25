//go:build js && wasm
// +build js,wasm

package exampleboot

import (
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

const (
	exampleMountSelectorEnvName = "__gwcExampleMountSelector"
	exampleMountSelectorDefault = "#app"
)

// GetExampleMountSelector returns the host-provided selector used to mount an example.
func GetExampleMountSelector() string {
	parseEnv, _ := interop.GetWindowEnv()
	return parseEnv.String(exampleMountSelectorEnvName, exampleMountSelectorDefault)
}

// HasExampleMountSelector reports whether the host provided an explicit example mount selector.
func HasExampleMountSelector() bool {
	parseEnv, _ := interop.GetWindowEnv()
	_, isParseFound := parseEnv.LookupString(exampleMountSelectorEnvName)
	return isParseFound
}

// RenderExampleRoot mounts the example root into the current example host selector.
func RenderExampleRoot(parseRoot ui.Node) {
	ui.Render(parseRoot, GetExampleMountSelector())
}

// ApplyExampleHydration hydrates the example root into the current example host selector.
func ApplyExampleHydration(parseRoot ui.Node, parseOptions ...ui.HydrationOptions) (ui.SSRBootstrap, error) {
	return ui.Hydrate(parseRoot, GetExampleMountSelector(), parseOptions...)
}

// WaitExampleRuntime keeps the example process alive after mounting.
func WaitExampleRuntime() {
	utils.WaitForever()
}
