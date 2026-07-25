//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/agentui"
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// buildRegistry registers exactly the components an agent is allowed to compose. Every node an agent
// emits must name one of these types; anything else is rejected before it can render.
func buildRegistry() *agentui.Registry {
	parseReg := agentui.NewRegistry()
	parseReg.Register(agentui.ComponentSpec{
		Name:         "card",
		AllowedProps: []string{"title"},
		Render: func(parseProps map[string]string, parseChildren []ui.Node) ui.Node {
			return html.Div(html.Props{Class: "rounded-xl border border-slate-600 bg-slate-900 p-4"},
				html.H3(html.Props{Class: "text-lg font-semibold text-slate-100"}, html.Text(parseProps["title"])),
				html.Div(html.Props{Class: "mt-2 space-y-2 text-slate-300"}, parseChildren...),
			)
		},
	})
	parseReg.Register(agentui.ComponentSpec{
		Name:         "paragraph",
		AllowedProps: nil,
		Render: func(parseProps map[string]string, parseChildren []ui.Node) ui.Node {
			return html.P(html.Props{}, parseChildren...)
		},
	})
	return parseReg
}

// agentuiExample demonstrates agentui: an agent emits a typed JSON schema, which is validated against
// a component allow-list and rendered to safe UI. A node naming a non-allow-listed type is rejected.
func agentuiExample() ui.Node {
	parseReg := buildRegistry()

	parseAgentJSON := `{"type":"card","props":{"title":"Composed by an agent"},"children":[
		{"type":"paragraph","text":"This subtree was emitted as JSON, validated against the allow-list, then rendered — it carries no code, no handlers, and no raw HTML."}
	]}`
	parseRendered, parseRenderErr := parseReg.RenderJSON([]byte(parseAgentJSON))
	if parseRenderErr != nil {
		parseRendered = html.P(html.Props{Class: "text-rose-300"}, html.Text("render error: "+parseRenderErr.Error()))
	}

	// A hostile node naming a non-allow-listed type is rejected structurally.
	_, parseDeniedErr := parseReg.RenderJSON([]byte(`{"type":"script","text":"alert(document.cookie)"}`))
	parseDenied := "unexpectedly allowed"
	if parseDeniedErr != nil {
		parseDenied = parseDeniedErr.Error()
	}

	parseCatalog := make([]string, 0)
	for _, parseInfo := range parseReg.Catalog() {
		parseCatalog = append(parseCatalog, parseInfo.Name)
	}

	return shared.ExamplePage(
		"agentui.Registry",
		"Agent-native generative UI, safe by construction",
		"An agent emits a typed Node schema; Validate rejects any non-allow-listed type or prop, and Render turns the rest into real UI. Catalog() exposes the allow-list so an agent (or an MCP tool) knows up front what it may emit.",
		shared.ExamplePanel("Rendered from agent JSON", parseRendered),
		shared.ExamplePanel("Allow-list (Registry.Catalog)",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Permitted component types: "+strings.Join(parseCatalog, ", "))),
		),
		shared.ExamplePanel("Hostile node rejected",
			html.P(html.Props{Class: "mt-3 text-amber-300"}, html.Text(parseDenied)),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(agentuiExample))
	exampleboot.WaitExampleRuntime()
}
