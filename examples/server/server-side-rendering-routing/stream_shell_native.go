//go:build !js || !wasm

package main

import "github.com/monstercameron/GoWebComponents/ui"

func renderDemoShellStreamShell(parseView demoShellView) ui.Node {
	return renderDemoShellWithDeferredMode(parseView, true)
}
