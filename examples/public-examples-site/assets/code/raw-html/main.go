//go:build js && wasm

// Command raw-html is the e2e fixture for the G3 markup nodes (html.RawHTML /
// RawHTMLUnsafe): it renders sanitized rich text and a trusted inline SVG, each
// parsed into real nodes (never innerHTML).
package main

import (
	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// App demonstrates safe rich-text nodes and replacement through GWC state.
func App() ui.Node {
	parseUpdated := ui.UseState(false)
	parseToggle := ui.UseEvent(func() { parseUpdated.Update(func(parsePrevious bool) bool { return !parsePrevious }) })
	parseMarkup := `<p id="rich">hello <b id="bold">world</b><script>window.__pwned=1</script></p>`
	if parseUpdated.Get() {
		parseMarkup = `<div id="replacement"><h2>Updated message</h2><pre><code>one &lt; two</code></pre><a href="javascript:alert(1)">safe link</a></div>`
	}
	parseArgs := []any{FromProps(Props{ID: "app-root"})}
	// Sanitized: the <b> survives as a real element, the <script> is stripped.
	for _, parseNode := range RawHTML(parseMarkup) {
		parseArgs = append(parseArgs, parseNode)
	}
	// Trusted: inline SVG kept as real nodes, committed in the SVG namespace.
	for _, parseNode := range RawHTMLUnsafe(`<svg id="chart" width="20" height="20"><circle id="dot" cx="10" cy="10" r="5"></circle></svg>`) {
		parseArgs = append(parseArgs, parseNode)
	}
	parseArgs = append(parseArgs, Button(ID("replace-markup"), OnClick(parseToggle), Text("Replace message")))
	return Div(parseArgs...)
}

// main mounts the GWC example and keeps its event loop alive.
func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
