//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/ui"
)

func main() {
	fmt.Println("Go Web Components Blog Landing Page starting...")
	fmt.Println("Loading Blog Landing Page...")

	ui.Render(ui.CreateElement(BlogLandingPage), "#app")

	fmt.Println("Blog Landing Page rendered successfully")
	select {}
}
