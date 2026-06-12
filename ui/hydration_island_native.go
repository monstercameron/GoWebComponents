//go:build !js || !wasm

package ui

// ConfigureHydrationIslandBudget is a browser-only runtime setting. Native SSR
// can still validate plans with InspectHydrationIslandBudget.
func ConfigureHydrationIslandBudget(parseBudget HydrationIslandBudget) {
	_ = parseBudget
}

// HydrateIsland is browser-only. Server builds can render HydrationIsland
// wrappers and validate their budget plan before emitting HTML.
func HydrateIsland(parseRoot Node, parseOptions HydrationIslandOptions) (func(), error) {
	_ = parseRoot
	_ = parseOptions
	return nil, UnsupportedOnServer("HydrateIsland")
}
