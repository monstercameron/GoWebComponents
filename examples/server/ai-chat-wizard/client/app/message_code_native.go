//go:build !js || !wasm

package app

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderMessageCodeBlock renders the same initial markup without browser clipboard access.
func renderMessageCodeBlock(parseProps messageCodeProps) ui.Node {
	return html.Div(html.Props{Class: "prose-pre-wrap", Raw: map[string]any{"data-gwc-code": "true"}},
		parseProps.parseElement,
		html.Button(html.Props{Type: "button", Class: "prose-copy-btn"}, html.Text("Copy")),
	)
}
