//go:build !js || !wasm

package atlas

import "github.com/monstercameron/GoWebComponents/v7/ui"

func routeOutletNode() ui.Node {
	return nil
}
