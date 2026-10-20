//go:build !js || !wasm

package main

import "github.com/monstercameron/GoWebComponents/v7/ui"

func renderDemoShellStreamShell(parseView demoShellView) ui.Node {
	return renderDemoShellWithDeferredMode(parseView, true)
}
