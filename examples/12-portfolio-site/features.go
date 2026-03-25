//go:build js && wasm
// +build js,wasm

package main

// HeroSection renders the main hero banner with animated background and key features.
// Features gradient backgrounds, floating animations, and prominent call-to-action buttons
// that guide users to documentation and examples.
func HeroSection(_ Attrs) *Element {
	// Ensure blob animation CSS is loaded once per page
	UseEffect(func() func() { injectBlobCSS(); return nil }, true)

	return Section(
		Attrs{
			"id":    "home",
			"class": "relative min-h-screen flex items-center justify-center bg-[#0a0a0a] overflow-hidden",
		},

		// Background decoration
		Div(
			Attrs{"class": "absolute inset-0 overflow-hidden"},
			Div(Attrs{"class": "absolute -top-40 -right-40 w-80 h-80 bg-purple-600 rounded-full mix-blend-screen filter blur-[100px] opacity-20 animate-blob"}),
			Div(Attrs{"class": "absolute -bottom-40 -left-40 w-80 h-80 bg-blue-600 rounded-full mix-blend-screen filter blur-[100px] opacity-20 animate-blob animation-delay-2000"}),
			Div(Attrs{"class": "absolute top-40 left-40 w-80 h-80 bg-pink-600 rounded-full mix-blend-screen filter blur-[100px] opacity-20 animate-blob animation-delay-4000"}),
		),

		// Main content
		Div(
			Attrs{"class": "relative z-10 text-center max-w-5xl mx-auto px-6"},

			// Hero badge
			Div(
				Attrs{"class": "inline-flex items-center px-4 py-2 bg-indigo-900/30 text-indigo-300 border border-indigo-500/30 rounded-full text-sm font-medium mb-8 animate-bounce backdrop-blur-sm"},
				Span(Attrs{"class": "mr-2"}, "🚀"),
				Span(nil, "Welcome to the Future of Web Development"),
			),

			// Main title
			H1(
				Attrs{"class": "text-6xl md:text-7xl font-bold mb-6 leading-tight"},
				Span(Attrs{"class": "bg-gradient-to-r from-white via-indigo-400 to-purple-400 bg-clip-text text-transparent"}, "GoWebComponents"),
				Br(nil),
				Span(Attrs{"class": "text-4xl md:text-5xl text-gray-300"}, "Reactive Web Apps in Go"),
			),

			// Subtitle
			P(
				Attrs{"class": "text-xl md:text-2xl text-gray-400 mb-8 max-w-3xl mx-auto leading-relaxed"},
				"Build modern, reactive web applications using Go and WebAssembly. ",
				Strong(Attrs{"class": "text-white"}, "No JavaScript required."),
				" Experience the power of Go's concurrency, type safety, and performance in the browser.",
			),

			// CTA buttons
			Div(
				Attrs{"class": "flex flex-col sm:flex-row gap-4 justify-center items-center mb-12"},
				Button(
					Attrs{
						"class":   "px-8 py-4 bg-gradient-to-r from-indigo-600 to-purple-600 text-white text-lg font-semibold rounded-xl hover:from-indigo-700 hover:to-purple-700 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 cursor-pointer",
						"onclick": "scrollToSection('getting-started')",
					},
					"Get Started Now",
				),
				A(
					Attrs{
						"href":   "https://github.com/monstercameron/GoWebComponents",
						"target": "_blank",
						"class":  "px-8 py-4 bg-white/5 text-white text-lg font-semibold rounded-xl border-2 border-white/10 hover:border-indigo-500/50 hover:bg-white/10 hover:shadow-lg transition-all duration-300 transform hover:-translate-y-1 cursor-pointer",
					},
					"⚡ View on GitHub",
				),
			),

			// Key features preview
			Div(
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
func FeaturePreviewCard(parseIcon, parseTitle, parseDescription string) *Element {
	return Div(
		Attrs{"class": "bg-white/5 backdrop-blur-sm p-6 rounded-xl shadow-lg hover:shadow-xl transition-shadow duration-300 border border-white/10 hover:border-white/20"},
		Div(Attrs{"class": "text-3xl mb-3"}, parseIcon),
		H3(Attrs{"class": "text-lg font-semibold text-white mb-2"}, parseTitle),
		P(Attrs{"class": "text-gray-400 text-sm"}, parseDescription),
	)
}

// FeaturesSection showcases the comprehensive feature set of GoWebComponents.
// Presents features in a responsive grid with detailed descriptions and benefits
// for developers considering the framework.
func FeaturesSection(_ Attrs) *Element {
	return Section(
		Attrs{"id": "features", "class": "py-20 bg-[#0a0a0a]"},
		Div(
			Attrs{"class": "container mx-auto px-6"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-6 bg-gradient-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent"}, "Powerful Features"),
				P(Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"}, "Everything you need to build modern web applications with Go"),
			),
			Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-8 max-w-6xl mx-auto"},
				FeatureCard("🏗️", "A Truly Unified Stack", "Share data structures, validation logic, and utilities between backend and frontend. Eliminate data-syncing bugs and code duplication."),
				FeatureCard("⚡", "Next-Generation Performance", "Go code compiles to highly optimized WebAssembly. Fiber-based reconciliation minimizes DOM updates. Advanced memory pooling for smooth UI."),
				FeatureCard("🔄", "Superior Concurrency Model", "Run expensive operations in the background with goroutines. No more frozen UIs."),
				FeatureCard("🎯", "The Go Advantage Over JavaScript/TypeScript", "No node_modules, no complex transpilers. Use Go's standard library and compiler for a single, portable binary."),
				FeatureCard("🪝", "A Familiar, Modern API", "React-like hooks: UseState, UseEffect, and UseMemo for state, effects, and performance."),
				FeatureCard("📦", "Comprehensive Component Library", "80+ pre-built HTML element constructors. Build UIs with Div, Button, Form, and more."),
				FeatureCard("🛡️", "Rock-Solid Reliability", "Go's static type system catches errors at compile time. Write robust, maintainable code."),
				FeatureCard("🌐", "Effortless Data Fetching", "Built-in UseFetch for raw fetch state, UseResource for typed async loading, and Fetch for imperative requests."),
			),
		),
	)
}

// FeatureCard renders an individual feature with icon, title, and detailed description.
// Includes hover animations and gradient styling for enhanced visual appeal.
func FeatureCard(parseIcon, parseTitle, parseDescription string) *Element {
	return Div(
		Attrs{"class": "bg-white/5 p-8 rounded-2xl hover:shadow-lg transition-all duration-300 border border-white/10 hover:border-indigo-500/50 group backdrop-blur-sm"},
		Div(Attrs{"class": "text-4xl mb-4 group-hover:scale-110 transition-transform duration-300"}, parseIcon),
		H3(Attrs{"class": "text-xl font-bold text-white mb-4"}, parseTitle),
		P(Attrs{"class": "text-gray-400 leading-relaxed"}, parseDescription),
	)
}
