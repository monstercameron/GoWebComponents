//go:build js && wasm
// +build js,wasm

package website

import (
	"strings"
	"syscall/js"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// Route represents a single route configuration
type Route struct {
	Path      string
	Component func(Attrs) *Element
	Title     string
}

// Router component handles hash-based routing
func Router(props Attrs) *Element {
	// State for current route
	currentRoute, setCurrentRoute := GoUseState("app")

	// Define the two main routes
	routes := map[string]Route{
		"app": {
			Path:      "",
			Component: AppPage,
			Title:     "Personal Website 2025 - Earl Cameron",
		},
		"docs": {
			Path:      "docs",
			Component: DocsPage,
			Title:     "Documentation - GoWebComponents",
		},
	}

	// Effect to handle hash changes and initial route
	GoUseEffect(func() {
		// Function to update route based on current hash
		updateRoute := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			hash := js.Global().Get("location").Get("hash").String()

			// Remove the '#' prefix
			if strings.HasPrefix(hash, "#") {
				hash = hash[1:]
			}

			// Route logic - only handle main routes, ignore section hashes
			var routeKey string
			switch {
			case hash == "docs" || hash == "/docs":
				routeKey = "docs"
			default:
				// All other hashes (empty, sections like #about, #contact, etc.) go to app
				routeKey = "app"
			}

			// Update route if it exists
			if route, exists := routes[routeKey]; exists {
				setCurrentRoute(routeKey)
				// Update page title
				js.Global().Get("document").Set("title", route.Title)
			}

			return nil
		})

		// Listen for hash changes
		js.Global().Get("window").Call("addEventListener", "hashchange", updateRoute)

		// Handle initial route on page load
		updateRoute.Invoke()

		// Note: Cleanup would be handled by the framework
		return
	})

	// Get current route component
	route := routes[currentRoute()]
	if route.Component == nil {
		// Fallback to app if component is missing
		route = routes["app"]
	}

	return route.Component(nil)
}

// DocsNavBar creates a simplified navigation bar for the documentation page
func DocsNavBar(props Attrs) *Element {
	return Nav(
		Attrs{
			"class": "fixed top-0 left-0 right-0 z-50 bg-white/95 backdrop-blur-xl border-b border-gray-200/50 shadow-lg",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "flex items-center justify-between h-16 md:h-20"},

				// Back Home Link
				Button(
					Attrs{
						"class":   "group flex items-center space-x-3 px-4 py-2 text-gray-700 hover:text-indigo-600 transition-all duration-300 font-medium rounded-xl hover:bg-gradient-to-r hover:from-indigo-50 hover:to-purple-50 cursor-pointer",
						"onclick": GoUseFunc(func(event GoEvent) { Navigate("app") }),
					},
					Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:scale-110"}, "←"),
					Span(Attrs{"class": "font-semibold transition-transform duration-300 group-hover:translate-x-0.5"}, "Back to App"),
				),

				// Title
				Div(
					Attrs{"class": "flex items-center"},
					H1(
						Attrs{"class": "text-xl md:text-2xl font-bold text-gray-900"},
						"📚 Documentation",
					),
				),

				// GitHub button
				A(
					Attrs{
						"href":   "https://github.com/monstercameron/GoWebComponents",
						"target": "_blank",
						"class":  "group relative overflow-hidden px-5 py-2.5 bg-gray-900 text-white rounded-xl hover:bg-gray-800 transition-all duration-300 shadow-lg hover:shadow-xl transform hover:-translate-y-1 cursor-pointer",
					},
					Div(
						Attrs{"class": "absolute inset-0 bg-gradient-to-r from-gray-800 to-gray-900 opacity-0 group-hover:opacity-100 transition-opacity duration-300"},
					),
					Div(
						Attrs{"class": "relative flex items-center space-x-2"},
						Span(Attrs{"class": "text-lg transition-transform duration-300 group-hover:rotate-12"}, "🐙"),
						Span(Attrs{"class": "font-semibold text-sm"}, "GitHub"),
					),
				),
			),
		),
	)
}

// Navigate function for programmatic navigation
func Navigate(path string) {
	// Only handle main routes, let section links work naturally
	if path == "docs" {
		js.Global().Get("location").Set("hash", "#docs")
	} else if path == "app" {
		js.Global().Get("location").Set("hash", "#")
	} else {
		// For section links, just set the hash and let the app handle scrolling
		js.Global().Get("location").Set("hash", "#"+path)
	}
}

// AppPage contains the current application (all sections)
func AppPage(props Attrs) *Element {
	return Div(
		Attrs{"class": "min-h-screen bg-gradient-to-br from-gray-50 to-blue-50"},

		// Navigation bar
		NavBar(nil),

		// All app sections
		Div(nil,
			// Personal Hero Section
			PersonalHeroSection(nil),

			// About Me Section
			PersonalAboutSection(nil),

			// Skills & Technologies Section
			PersonalSkillsSection(nil),

			// YouTube Channel Section
			PersonalYouTubeSection(nil),

			// Featured Projects Section
			PortfolioProjectsSection(nil),

			// GoWebComponents Showcase
			GWCShowcaseSection(nil),

			// Interactive Examples Section
			GWCExamplesSection(nil),

			// Why GoWebComponents Section
			WhyGoWebComponentsSection(nil),

			// Contact Section
			ContactSection(nil),
		),

		// Footer
		FooterSection(nil),

		// Scroll to top button
		ScrollToTopButton(nil),
	)
}

// DocsPage will contain documentation (to be developed)
func DocsPage(props Attrs) *Element {
	return Div(
		Attrs{"class": "min-h-screen bg-gradient-to-br from-gray-50 to-blue-50"},

		// Documentation-specific navigation bar
		DocsNavBar(nil),

		// Documentation content
		Div(
			Attrs{"class": "pt-20 pb-20"},
			Div(
				Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
				Div(
					Attrs{"class": "text-center"},
					H1(
						Attrs{"class": "text-4xl font-bold text-gray-900 mb-8"},
						"📚 Documentation",
					),
					P(
						Attrs{"class": "text-xl text-gray-600 mb-8"},
						"Comprehensive documentation for GoWebComponents is coming soon!",
					),
					Div(
						Attrs{"class": "bg-white rounded-2xl p-8 shadow-lg border border-gray-200 max-w-2xl mx-auto"},
						H2(
							Attrs{"class": "text-2xl font-semibold text-gray-900 mb-4"},
							"🚧 Under Development",
						),
						P(
							Attrs{"class": "text-gray-600 mb-6"},
							"We're working hard to create comprehensive documentation that will include:",
						),
						Ul(
							Attrs{"class": "text-left space-y-3 text-gray-700"},
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-green-600"}, "📖"),
								"API Reference"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-blue-600"}, "🎯"),
								"Getting Started Guide"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-purple-600"}, "💡"),
								"Best Practices"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-orange-600"}, "🔧"),
								"Advanced Usage"),
							Li(Attrs{"class": "flex items-center"},
								Span(Attrs{"class": "mr-3 text-red-600"}, "🧪"),
								"Testing Guidelines"),
						),
						Div(
							Attrs{"class": "mt-8"},
							Button(
								Attrs{
									"class":   "px-6 py-3 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors duration-200 font-medium",
									"onclick": GoUseFunc(func(event GoEvent) { Navigate("app") }),
								},
								"← Back to App",
							),
						),
					),
				),
			),
		),

		// Footer
		FooterSection(nil),

		// Scroll to top button
		ScrollToTopButton(nil),
	)
}
