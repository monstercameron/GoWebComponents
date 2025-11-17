//go:build js && wasm
// +build js,wasm

package website

import . "github.com/monstercameron/GoWebComponents/fiber"

// FeaturesSection now acts as a structured Table of Contents with grouped sections and in-page links
func FeaturesSection(props Attrs) *Element {
	return Section(
		Attrs{"id": "features", "class": "py-12 bg-white"},
		Div(
			Attrs{"class": "container mx-auto px-4"},
			H2(Attrs{"class": "text-3xl font-bold mb-6 text-indigo-700"}, "Table of Contents"),
			Ul(Attrs{"class": "space-y-4 list-disc list-inside"},
				// Section: Core Philosophy
				Li(nil,
					Strong(Attrs{"class": "text-lg text-gray-900"}, "Core Philosophy"),
					Ul(Attrs{"class": "ml-6 list-disc list-inside space-y-1"},
						Li(nil, A(Attrs{"href": "#feature-unified-stack", "class": "text-indigo-700 hover:underline"}, "A Truly Unified Stack")),
						Li(nil, A(Attrs{"href": "#feature-go-advantage", "class": "text-indigo-700 hover:underline"}, "The Go Advantage Over JavaScript/TypeScript")),
						Li(nil, A(Attrs{"href": "#feature-reliability", "class": "text-indigo-700 hover:underline"}, "Rock-Solid Reliability")),
					),
				),
				// Section: Performance & Concurrency
				Li(nil,
					Strong(Attrs{"class": "text-lg text-gray-900"}, "Performance & Concurrency"),
					Ul(Attrs{"class": "ml-6 list-disc list-inside space-y-1"},
						Li(nil, A(Attrs{"href": "#feature-performance", "class": "text-indigo-700 hover:underline"}, "Next-Generation Performance")),
						Li(nil, A(Attrs{"href": "#feature-concurrency", "class": "text-indigo-700 hover:underline"}, "Superior Concurrency Model")),
					),
				),
				// Section: Developer Experience
				Li(nil,
					Strong(Attrs{"class": "text-lg text-gray-900"}, "Developer Experience"),
					Ul(Attrs{"class": "ml-6 list-disc list-inside space-y-1"},
						Li(nil, A(Attrs{"href": "#feature-modern-api", "class": "text-indigo-700 hover:underline"}, "A Familiar, Modern API")),
						Li(nil, A(Attrs{"href": "#feature-component-library", "class": "text-indigo-700 hover:underline"}, "Comprehensive Component Library")),
						Li(nil, A(Attrs{"href": "#feature-data-fetching", "class": "text-indigo-700 hover:underline"}, "Effortless Data Fetching")),
					),
				),
			),
		),
	)
}
