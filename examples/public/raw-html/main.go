//go:build js && wasm

// Command raw-html is the e2e fixture for the G3 markup nodes (html.RawHTML /
// RawHTMLUnsafe): it renders sanitized rich text and a trusted inline SVG, each
// parsed into real nodes (never innerHTML).
package main

import (
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func App() ui.Node {
	parseArgs := []any{FromProps(Props{ID: "app-root"})}
	// Sanitized: the <b> survives as a real element, the <script> is stripped.
	for _, parseNode := range RawHTML(`<p id="rich">hello <b id="bold">world</b><script>window.__pwned=1</script></p>`) {
		parseArgs = append(parseArgs, parseNode)
	}
	// Trusted: inline SVG kept as real nodes, committed in the SVG namespace.
	for _, parseNode := range RawHTMLUnsafe(`<svg id="chart" width="20" height="20"><circle id="dot" cx="10" cy="10" r="5"></circle></svg>`) {
		parseArgs = append(parseArgs, parseNode)
	}
	return Div(parseArgs...)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
