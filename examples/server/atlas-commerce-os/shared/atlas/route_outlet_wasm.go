//go:build js && wasm

package atlas

import (
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func routeOutletNode() ui.Node {
	return router.GetOutlet()
}
