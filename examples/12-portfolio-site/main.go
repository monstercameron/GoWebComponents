//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/utils"
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
	parseR.Register(portfolioDocsRoute, DocsPage)
	parseR.Register(portfolioCatchAllRoute, NotFoundPage)

	parseR.Mount("#app")

	fmt.Println("✅ Portfolio Site rendered")
	select {}
}
