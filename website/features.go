//go:build js && wasm
// +build js,wasm

package website

import . "github.com/monstercameron/GoWebComponents/fiber"

// FeaturesSection lists all major features of GoWebComponents with in-page anchors
func FeaturesSection(props Attrs) *Element {
	return Section(
		Attrs{"id": "features", "class": "py-12 bg-white"},
		Div(
			Attrs{"class": "container mx-auto px-4"},
			H2(Attrs{"class": "text-3xl font-bold mb-6 text-indigo-700"}, "Features"),
			Ul(Attrs{"class": "space-y-4"},
				Li(Attrs{"id": "feature-unified-stack"},
					Strong(Attrs{"class": "text-gray-900"}, "A Truly Unified Stack: "),
					Span(Attrs{"class": "text-gray-800"}, "Share data structures, validation logic, and utilities between backend and frontend. Eliminate data-syncing bugs and code duplication."),
				),
				Li(Attrs{"id": "feature-performance"},
					Strong(Attrs{"class": "text-gray-900"}, "Next-Generation Performance: "),
					Span(Attrs{"class": "text-gray-800"}, "Go code compiles to highly optimized WebAssembly. Fiber-based reconciliation minimizes DOM updates. Advanced memory pooling for smooth UI."),
				),
				Li(Attrs{"id": "feature-concurrency"},
					Strong(Attrs{"class": "text-gray-900"}, "Superior Concurrency Model: "),
					Span(Attrs{"class": "text-gray-800"}, "Run expensive operations in the background with goroutines. No more frozen UIs."),
				),
				Li(Attrs{"id": "feature-go-advantage"},
					Strong(Attrs{"class": "text-gray-900"}, "The Go Advantage Over JavaScript/TypeScript: "),
					Span(Attrs{"class": "text-gray-800"}, "No node_modules, no complex transpilers. Use Go's standard library and compiler for a single, portable binary."),
				),
				Li(Attrs{"id": "feature-modern-api"},
					Strong(Attrs{"class": "text-gray-900"}, "A Familiar, Modern API: "),
					Span(Attrs{"class": "text-gray-800"}, "React-like hooks: GoUseState, GoUseEffect, GoUseMemo for state, effects, and performance."),
				),
				Li(Attrs{"id": "feature-component-library"},
					Strong(Attrs{"class": "text-gray-900"}, "Comprehensive Component Library: "),
					Span(Attrs{"class": "text-gray-800"}, "80+ pre-built HTML element constructors. Build UIs with Div, Button, Form, and more."),
				),
				Li(Attrs{"id": "feature-reliability"},
					Strong(Attrs{"class": "text-gray-900"}, "Rock-Solid Reliability: "),
					Span(Attrs{"class": "text-gray-800"}, "Go's static type system catches errors at compile time. Write robust, maintainable code."),
				),
				Li(Attrs{"id": "feature-data-fetching"},
					Strong(Attrs{"class": "text-gray-900"}, "Effortless Data Fetching: "),
					Span(Attrs{"class": "text-gray-800"}, "Built-in GoUseFetch hook for declarative data fetching and GoFetch for imperative requests."),
				),
			),
		),
	)
}
