//go:build !js || !wasm

package servercomponents

import "github.com/monstercameron/GoWebComponents/v6/ui"

// ServerOnly renders its server implementation on non-wasm targets.
func ServerOnly(parseProps Props) ui.Node {
	if parseProps.Render == nil {
		return parseProps.Placeholder
	}
	return parseProps.Render()
}
