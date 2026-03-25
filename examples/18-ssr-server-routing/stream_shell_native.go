//go:build !js || !wasm
// +build !js !wasm

package main

import "github.com/monstercameron/GoWebComponents/ui"

func renderDemoShellStreamShell(view demoShellView) ui.Node {
	return renderDemoShellWithDeferredMode(view, true)
}
