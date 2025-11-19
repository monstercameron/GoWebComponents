// ./main.go

//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
)

// main is the entry point for the WASM module
func main() {
	fmt.Println("🚀 Go Web Components Blog Landing Page starting...")
	fmt.Println("📊 Loading Blog Landing Page...")

	// Render the blog landing page to the DOM
	render.To(dom.CreateElement(BlogLandingPage, nil), "#app")

	fmt.Println("✅ Blog Landing Page rendered successfully")
	select {}
}
