//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func main() {
	fmt.Println("Calculator example starting...")
	exampleboot.RenderExampleRoot(ui.CreateElement(App))
	fmt.Println("Calculator example mounted")
	exampleboot.WaitExampleRuntime()
}
