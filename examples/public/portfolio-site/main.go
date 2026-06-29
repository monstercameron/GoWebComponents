//go:build js && wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/v4/hotreload"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func main() {
	fmt.Println("🚀 Portfolio Site starting...")

	// Configure debug logging namespaces for development visibility
	utils.ConfigureDebugNamespacesExclusive(map[string]bool{
		"HOOKS":  false,
		"RENDER": false,
		"MEMORY": false,
		"DOM":    false,
		"FETCH":  false,
		"EVENTS": false,
		"COMMIT": false,
		"FIBER":  false,
	})

	// Enable hot reload for instant development feedback
	hotreload.Enable()

	// Initialize and mount the global router
	parseR := router.GetRouter()
	parseR.Register(portfolioHomeRoute, DocsWebsite)
	parseR.Register(portfolioDocsRoute, renderDocsPageCompact)
	parseR.Register(portfolioCatchAllRoute, renderNotFoundPageCompact)

	exampleboot.RenderExampleRouter(parseR)

	fmt.Println("✅ Portfolio Site rendered")
	exampleboot.WaitExampleRuntime()
}
