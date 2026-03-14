//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/ui"
)

func main() {
	fmt.Println("Calculator example starting...")
	ui.Render(ui.CreateElement(App), "#app")
	fmt.Println("Calculator example mounted")
	select {}
}
