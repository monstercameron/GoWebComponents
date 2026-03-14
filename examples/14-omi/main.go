//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/router"
)

func main() {
	fmt.Println("OMI example starting...")

	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	r.Register("/", OverviewPage)
	r.Register("/data", DataPage)
	r.Register("/playground", PlaygroundPage)
	r.Register("*", NotFoundPage)
	r.Mount("#app")

	fmt.Println("OMI example mounted")
	select {}
}
