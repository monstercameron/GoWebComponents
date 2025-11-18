//go:build js && wasm
// +build js,wasm

package website

import (
	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
)

// HeroSection renders the main hero banner with animated background and key features.
// Features gradient backgrounds, floating animations, and prominent call-to-action buttons
// that guide users to documentation and examples.
func HeroSection(_ Attrs) *Element {
	// Ensure blob animation CSS is loaded once per page
	hooks.UseEffect(func() func() { injectBlobCSS(); return nil }, true)

	return dom.Section(
		Attrs{
			"id":    "home",
			"class": "relative min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-50 via-white to-purple-50 overflow-hidden",
		},

		// Background decoration
		dom.Div(
			Attrs{"class": "absolute inset-0 overflow-hidden"},
			dom.Div(Attrs{"class": "absolute -top-40 -right-40 w-80 h-80 bg-purple-300 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob"}),
			dom.Div(Attrs{"class": "absolute -bottom-40 -left-40 w-80 h-80 bg-yellow-300 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob animation-delay-2000"}),
			dom.Div(Attrs{"class": "absolute top-40 left-40 w-80 h-80 bg-pink-300 rounded-full mix-blend-multiply filter blur-xl opacity-70 animate-blob animation-delay-4000"}),
		),

		// Main content
		dom.Div(
			Attrs{"class": "relative z-10 text-center max-w-5xl mx-auto px-6"},

			// Hero badge
			dom.Div(
				Attrs{"class": "inline-flex items-center px-4 py-2 bg-indigo-100 text-indigo-800 rounded-full text-sm font-medium mb-8 animate-bounce"},
				dom.Span(Attrs{"class": "mr-2"}, "🚀"),
				dom.Span(nil, "Welcome to the Future of Web Development"),
			),

			// Main title
			dom.H1(
				Attrs{"class": "text-6xl md:text-7xl font-bold mb-6 leading-tight"},
				dom.Span(Attrs{"class": "bg-gradient-to-r from-gray-900 via-indigo-600 to-purple-600 bg-clip-text text-transparent"}, "GoWebComponents"),
				dom.Br(nil),
				dom.Span(Attrs{"class": "text-4xl md:text-5xl text-gray-700"}, "Reactive Web Apps in Go"),
			),

			// Subtitle
			dom.P(
				Attrs{"class": "text-xl md:text-2xl text-gray-600 mb-8 max-w-3xl mx-auto leading-relaxed"},
				"Build modern, reactive web applications using Go and WebAssembly. ",
				dom.Strong(nil, "No JavaScript required."),
				" Experience the power of Go's concurrency, type safety, and performance in the browser.",
			),

			// CTA buttons
			dom.Div(
				Attrs{"class": "flex flex-col sm:flex-row gap-4 justify-center items-center mb-12"},
				dom.Button(
					Attrs{
						"class":   "px-8 py-4 bg-gradient-to-r from-indigo-600 to-purple-600 text-white text-lg font-semibold rounded-xl hover:from-indigo-700 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1",
						"onclick": "scrollToSection('getting-started')",
					},
					"Get Started Now",
				),
				dom.A(
					Attrs{
						"href":   "https://github.com/monstercameron/GoWebComponents",
						"target": "_blank",
						"class":  "px-8 py-4 bg-white text-gray-800 text-lg font-semibold rounded-xl border-2 border-gray-200 hover:border-indigo-300 hover:shadow-lg transition-all duration-300 transform hover:-translate-y-1",
					},
					"⚡ View on GitHub",
				),
			),

			// Key features preview
			dom.Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-3 gap-6 max-w-4xl mx-auto"},
				FeaturePreviewCard("⚡", "Lightning Fast", "WebAssembly performance with Go's efficiency"),
				FeaturePreviewCard("🔄", "Hot Reload", "Instant feedback during development"),
				FeaturePreviewCard("🛡️", "Type Safe", "Compile-time error checking and safety"),
			),
		),
	)
}

// FeaturePreviewCard renders a glassmorphism-styled card highlighting key framework features.
// Used in the hero section to provide quick feature overview with icons and descriptions.
func FeaturePreviewCard(icon, title, description string) *Element {
	return dom.Div(
		Attrs{"class": "bg-white/80 backdrop-blur-sm p-6 rounded-xl shadow-lg hover:shadow-xl transition-shadow duration-300 border border-gray-100"},
		dom.Div(Attrs{"class": "text-3xl mb-3"}, icon),
		dom.H3(Attrs{"class": "text-lg font-semibold text-gray-900 mb-2"}, title),
		dom.P(Attrs{"class": "text-gray-600 text-sm"}, description),
	)
}

// FeaturesSection showcases the comprehensive feature set of GoWebComponents.
// Presents features in a responsive grid with detailed descriptions and benefits
// for developers considering the framework.
func FeaturesSection(_ Attrs) *Element {
	return dom.Section(
		Attrs{"id": "features", "class": "py-20 bg-white"},
		dom.Div(
			Attrs{"class": "container mx-auto px-6"},
			dom.Div(
				Attrs{"class": "text-center mb-16"},
				dom.H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-6 bg-gradient-to-r from-indigo-600 to-purple-600 bg-clip-text text-transparent"}, "Powerful Features"),
				dom.P(Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"}, "Everything you need to build modern web applications with Go"),
			),
			dom.Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-8 max-w-6xl mx-auto"},
				FeatureCard("🏗️", "A Truly Unified Stack", "Share data structures, validation logic, and utilities between backend and frontend. Eliminate data-syncing bugs and code duplication."),
				FeatureCard("⚡", "Next-Generation Performance", "Go code compiles to highly optimized WebAssembly. Fiber-based reconciliation minimizes DOM updates. Advanced memory pooling for smooth UI."),
				FeatureCard("🔄", "Superior Concurrency Model", "Run expensive operations in the background with goroutines. No more frozen UIs."),
				FeatureCard("🎯", "The Go Advantage Over JavaScript/TypeScript", "No node_modules, no complex transpilers. Use Go's standard library and compiler for a single, portable binary."),
				FeatureCard("🪝", "A Familiar, Modern API", "React-like hooks: GoUseState, hooks.UseEffect, GoUseMemo for state, effects, and performance."),
				FeatureCard("📦", "Comprehensive Component Library", "80+ pre-built HTML element constructors. Build UIs with Div, Button, Form, and more."),
				FeatureCard("🛡️", "Rock-Solid Reliability", "Go's static type system catches errors at compile time. Write robust, maintainable code."),
				FeatureCard("🌐", "Effortless Data Fetching", "Built-in GoUseFetch hook for declarative data fetching and GoFetch for imperative requests."),
			),
		),
	)
}

// FeatureCard renders an individual feature with icon, title, and detailed description.
// Includes hover animations and gradient styling for enhanced visual appeal.
func FeatureCard(icon, title, description string) *Element {
	return dom.Div(
		Attrs{"class": "bg-gradient-to-br from-gray-50 to-gray-100 p-8 rounded-2xl hover:shadow-lg transition-all duration-300 border border-gray-200 hover:border-indigo-200 group"},
		dom.Div(Attrs{"class": "text-4xl mb-4 group-hover:scale-110 transition-transform duration-300"}, icon),
		dom.H3(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, title),
		dom.P(Attrs{"class": "text-gray-600 leading-relaxed"}, description),
	)
}
