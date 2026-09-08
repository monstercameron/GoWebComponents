//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func main() {
	fmt.Println("Go Web Components Blog Landing Page starting...")
	fmt.Println("Loading Blog Landing Page...")

	exampleboot.RenderExampleRoot(ui.CreateElement(BlogLandingPage))

	fmt.Println("Blog Landing Page rendered successfully")
	exampleboot.WaitExampleRuntime()
}
