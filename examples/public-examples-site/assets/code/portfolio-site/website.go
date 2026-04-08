//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// DocsWebsite composes Earl Cameron's personal website and GoWebComponents showcase.
// Combines personal branding, project portfolio, technical documentation, and
// interactive examples into a cohesive single-page application experience.
func DocsWebsite(parseProps Attrs) *Element {
	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] text-white selection:bg-blue-500/30"},
		shared.ExamplePage(
			"Portfolio Site",
			"routed multi-section app",
			"Show a personal landing page, project gallery, and framework examples without turning the demo into a long-form marketing page.",
			shared.ExamplePanel("Hero", ui.CreateElement(PersonalHeroSection)),
			shared.ExamplePanel("Projects", ui.CreateElement(PortfolioProjectsSection)),
			shared.ExamplePanel("Examples", ui.CreateElement(GWCExamplesSection)),
			shared.ExamplePanel("Contact", ui.CreateElement(ContactSection)),
		),
	)
}
