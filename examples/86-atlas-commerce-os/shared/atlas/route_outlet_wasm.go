//go:build js && wasm

package atlas

import (
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func routeOutletNode() ui.Node {
	return router.Outlet()
}
