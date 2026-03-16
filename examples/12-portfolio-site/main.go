//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/utils"
)

func main() {
	fmt.Println("🚀 Portfolio Site starting...")

	// Configure debug logging namespaces for development visibility
	utils.SetDebugNamespacesExclusive(map[string]bool{
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
	utils.EnableHotReload(true)

	// Initialize and mount the global router
	r := router.GetRouter()
	r.GoRegisterRoute(portfolioHomeRoute, DocsWebsite)
	r.GoRegisterRoute(portfolioDocsRoute, DocsPage)
	r.GoRegisterRoute(portfolioCatchAllRoute, NotFoundPage)

	r.Mount("#app")

	fmt.Println("✅ Portfolio Site rendered")
	select {}
}
