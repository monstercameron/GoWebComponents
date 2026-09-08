//go:build js && wasm

package main

import "github.com/monstercameron/GoWebComponents/v6/examples/shared"

// renderDocsPageCompact renders the documentation route inside the shared example shell.
func renderDocsPageCompact(_ Attrs) *Element {
	return Div(
		Attrs{"class": "min-h-screen bg-[#0a0a0a] text-white"},
		shared.ExamplePage(
			"Portfolio Docs",
			"routed reference surface",
			"Keep the API reference on its own route, but present the core sections in a tighter example shell.",
			shared.ExamplePanel("Contents", TableOfContents(nil)),
			shared.ExamplePanel("Core API",
				Div(
					Attrs{"class": "space-y-10"},
					CoreTypesSection(nil),
					ComponentsHooksSection(nil),
				),
			),
			shared.ExamplePanel("Reference",
				Div(
					Attrs{"class": "space-y-10"},
					HTMLElementsSection(nil),
					EventHandlingSection(nil),
					StateManagementSection(nil),
					MemoryManagementSection(nil),
					UtilitiesSection(nil),
				),
			),
		),
	)
}
