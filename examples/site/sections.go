//go:build js && wasm
// +build js,wasm

package website
import (
"github.com/monstercameron/GoWebComponents/dom"
)

// AboutSection explains the value proposition of GoWebComponents over traditional development.
// Features side-by-side comparison cards and key statistics to highlight framework benefits.
func AboutSection(props Attrs) *Element {
	return dom.Section(
		Attrs{
			"id":    "about",
			"class": "py-20 bg-gradient-to-br from-gray-50 to-indigo-50",
		},
		dom.Div(
			Attrs{"class": "container mx-auto px-6"},
			dom.Div(
				Attrs{"class": "max-w-4xl mx-auto text-center mb-16"},
				dom.H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-8 text-gray-900"}, "Why GoWebComponents?"),
				dom.P(
					Attrs{"class": "text-xl text-gray-600 leading-relaxed mb-8"},
					"Born from the need to build complex, performant web applications without the JavaScript ecosystem's complexity. ",
					"GoWebComponents brings Go's elegance, safety, and performance to the frontend.",
				),
			),

			// Comparison cards
			dom.Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-12 max-w-6xl mx-auto"},

				// Traditional approach
				dom.Div(
					Attrs{"class": "bg-red-50 border border-red-200 rounded-2xl p-8"},
					dom.H3(Attrs{"class": "text-2xl font-bold text-red-800 mb-6 flex items-center"},
						dom.Span(Attrs{"class": "mr-3"}, "❌"),
						"Traditional Web Development"),
					dom.Ul(Attrs{"class": "space-y-4 text-red-700"},
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Complex build pipelines and toolchains"),
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Runtime errors and type coercion issues"),
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Separate backend/frontend codebases"),
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Heavy node_modules dependencies"),
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"State management complexity"),
					),
				),

				// GoWebComponents approach
				dom.Div(
					Attrs{"class": "bg-green-50 border border-green-200 rounded-2xl p-8"},
					dom.H3(Attrs{"class": "text-2xl font-bold text-green-800 mb-6 flex items-center"},
						dom.Span(Attrs{"class": "mr-3"}, "✅"),
						"GoWebComponents Approach"),
					dom.Ul(Attrs{"class": "space-y-4 text-green-700"},
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Single Go codebase for everything"),
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Compile-time error checking"),
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Shared types and logic"),
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Zero external dependencies"),
						dom.Li(Attrs{"class": "flex items-start"},
							dom.Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Built-in state management"),
					),
				),
			),

			// Stats section
			dom.Div(
				Attrs{"class": "mt-16 grid grid-cols-1 md:grid-cols-3 gap-8 max-w-4xl mx-auto"},
				StatCard("10x", "Faster Development", "No build tools, instant feedback"),
				StatCard("100%", "Type Safe", "Go's compiler catches all errors"),
				StatCard("0", "Dependencies", "Pure Go, no node_modules"),
			),
		),
	)
}

// StatCard displays a key metric with large number, title, and description.
// Used to highlight quantifiable benefits of choosing GoWebComponents.
func StatCard(stat, title, description string) *Element {
	return dom.Div(
		Attrs{"class": "text-center bg-white rounded-xl p-6 shadow-lg hover:shadow-xl transition-shadow duration-300"},
		dom.Div(Attrs{"class": "text-4xl font-bold text-indigo-600 mb-2"}, stat),
		dom.H4(Attrs{"class": "text-xl font-bold text-gray-900 mb-2"}, title),
		dom.P(Attrs{"class": "text-gray-600"}, description),
	)
}

// GettingStartedSection provides step-by-step instructions for new developers.
// Includes installation commands, running examples, and a simple component example
// to demonstrate the framework's ease of use.
func GettingStartedSection(props Attrs) *Element {
	return dom.Section(
		Attrs{
			"id":    "getting-started",
			"class": "py-20 bg-white",
		},
		dom.Div(
			Attrs{"class": "container mx-auto px-6"},
			dom.Div(
				Attrs{"class": "text-center mb-16"},
				dom.H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-6 text-gray-900"}, "Get Started"),
				dom.P(Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"}, "Start building with GoWebComponents in minutes, not hours"),
			),

			// Installation steps
			dom.Div(
				Attrs{"class": "max-w-4xl mx-auto space-y-12"},

				InstallationStep("1", "Clone the Repository",
					dom.Code(Attrs{"class": "bg-gray-900 text-green-400 p-4 rounded-lg block"},
						"git clone https://github.com/monstercameron/GoWebComponents.git")),

				InstallationStep("2", "Run the Examples",
					dom.Code(Attrs{"class": "bg-gray-900 text-green-400 p-4 rounded-lg block"},
						"cd GoWebComponents && go run main.go")),

				InstallationStep("3", "Start Building",
					dom.P(Attrs{"class": "text-gray-600"},
						"Create your first component using familiar Go syntax. Check out the examples folder for inspiration!")),
			),

			// Quick example
			dom.Div(
				Attrs{"class": "mt-16 bg-gray-50 rounded-2xl p-8"},
				dom.H3(Attrs{"class": "text-2xl font-bold text-gray-900 mb-6 text-center"}, "Your First Component"),
				dom.Pre(Attrs{"class": "bg-gray-900 text-gray-100 p-6 rounded-lg overflow-x-auto text-sm"},
					dom.Code(nil, `func HelloWorld(props Attrs) *Element {
    return dom.Div(
        Attrs{"class": "text-center p-8"},
        H1(nil, "Hello, GoWebComponents!"),
        dom.P(nil, "Building reactive UIs with Go"),
        dom.Button(
            Attrs{
                "onclick": "alert('Hello from Go!')",
                "class": "px-4 py-2 bg-blue-500 text-white rounded",
            },
            "Click Me!",
        ),
    )
}`)),
			),
		),
	)
}

// InstallationStep renders a numbered instruction with icon and content.
// Provides consistent styling for multi-step processes and tutorials.
func InstallationStep(number, title string, content *Element) *Element {
	return dom.Div(
		Attrs{"class": "flex items-start space-x-6"},
		dom.Div(
			Attrs{"class": "flex-shrink-0 w-12 h-12 bg-indigo-600 text-white rounded-full flex items-center justify-center text-xl font-bold"},
			number,
		),
		dom.Div(
			Attrs{"class": "flex-1"},
			dom.H3(Attrs{"class": "text-2xl font-bold text-gray-900 mb-4"}, title),
			content,
		),
	)
}

// ExamplesSection displays a gallery of interactive examples with live demos.
// Each example card includes a working demo and links to detailed explanations
// and source code viewing functionality.
func ExamplesSection(props Attrs) *Element {
	return dom.Section(
		Attrs{
			"id":    "examples",
			"class": "py-20 bg-gradient-to-br from-purple-50 to-indigo-50",
		},
		dom.Div(
			Attrs{"class": "container mx-auto px-6"},
			dom.Div(
				Attrs{"class": "text-center mb-16"},
				dom.H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-6 bg-gradient-to-r from-purple-600 to-indigo-600 bg-clip-text text-transparent"}, "Examples"),
				dom.P(Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"}, "See GoWebComponents in action with these real-world examples"),
			),

			// Example categories
			dom.Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 max-w-6xl mx-auto"},

				ExampleCard(
					"🔢",
					"Click Counter",
					"A simple interactive counter demonstrating state management and event handling.",
					"examples.ClickCounterExample()",
					"click-counter",
				),

				ExampleCard(
					"📋",
					"Todo App",
					"Full-featured todo application with add, delete, and mark complete functionality.",
					"examples.TodoAppExample()",
					"todo-app",
				),

				ExampleCard(
					"📊",
					"Dashboard",
					"Real-time dashboard with concurrent data updates and beautiful visualizations.",
					"examples.ConcurrentDashboardExample()",
					"dashboard",
				),

				ExampleCard(
					"🌐",
					"Network Monitor",
					"Live network monitoring dashboard showing real-time connection status.",
					"examples.NetworkMonitoringDashboardExample()",
					"network-monitor",
				),

				ExampleCard(
					"🎨",
					"Blog Landing",
					"Beautiful blog landing page with responsive design and smooth animations.",
					"examples.BlogLandingPageExample()",
					"blog-landing",
				),

				ExampleCard(
					"🔥",
					"Hot Reload Demo",
					"Demonstrates hot reload capabilities for rapid development iteration.",
					"examples.HotReloadExample()",
					"hot-reload",
				),
			),

			// Live demo section
			dom.Div(
				Attrs{"class": "mt-20 bg-white rounded-2xl shadow-xl p-8 max-w-4xl mx-auto"},
				dom.H3(Attrs{"class": "text-3xl font-bold text-gray-900 mb-6 text-center"}, "Try It Live"),
				dom.P(Attrs{"class": "text-lg text-gray-600 text-center mb-8"}, "Experience the power of GoWebComponents right in your browser"),

				// Interactive demo buttons
				dom.Div(
					Attrs{"class": "flex flex-wrap justify-center gap-4 mb-8"},
					DemoButton("Counter", "loadCounterDemo()"),
					DemoButton("Todo List", "loadTodoDemo()"),
					DemoButton("Dashboard", "loadDashboardDemo()"),
				),

				// Demo container
				dom.Div(
					Attrs{
						"id":    "demo-container",
						"class": "bg-gray-50 rounded-lg p-6 min-h-64 flex items-center justify-center border-2 border-dashed border-gray-300",
					},
					dom.P(Attrs{"class": "text-gray-500 text-lg"}, "Select an example above to see it in action"),
				),
			),
		),
	)
}

// ExampleCard creates a card for showcasing an example
func ExampleCard(icon, title, description, code, id string) *Element {
	return dom.Div(
		Attrs{"class": "bg-white dark:bg-gray-800 dark:text-gray-100 rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 p-6 border border-gray-100 dark:border-gray-700 group hover:scale-105"},
		dom.Div(
			Attrs{"class": "text-center mb-6"},
			dom.Div(Attrs{"class": "text-4xl mb-4 group-hover:scale-110 transition-transform duration-300"}, icon),
			dom.H3(Attrs{"class": "text-xl font-bold text-gray-900 dark:text-gray-100 mb-2"}, title),
			dom.P(Attrs{"class": "text-gray-600 dark:text-gray-300 text-sm leading-relaxed"}, description),
		),

		dom.Div(
			Attrs{"class": "space-y-4"},
			// Code snippet
			dom.Pre(Attrs{"class": "bg-gray-900 text-green-400 p-3 rounded-lg text-xs overflow-x-auto"},
				dom.Code(nil, code)),

			// Action buttons
			dom.Div(
				Attrs{"class": "flex space-x-3"},
				dom.Button(
					Attrs{
						"class":   "flex-1 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors duration-200 text-sm font-medium",
						"onclick": "runExample('" + id + "')",
					},
					"▶ Run",
				),
				dom.Button(
					Attrs{
						"class":   "px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-100 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors duration-200 text-sm font-medium",
						"onclick": "viewSource('" + id + "')",
					},
					"{ }",
				),
			),
		),
	)
}

// DemoButton creates a button for the live demo section
func DemoButton(label, onclick string) *Element {
	return dom.Button(
		Attrs{
			"class":   "px-6 py-3 bg-gradient-to-r from-purple-600 to-indigo-600 text-white rounded-lg hover:from-purple-700 hover:to-indigo-700 transition-all duration-200 font-medium shadow-md hover:shadow-lg transform hover:-translate-y-0.5",
			"onclick": onclick,
		},
		label,
	)
}

// ApiDocumentationSection provides comprehensive API documentation
func ApiDocumentationSection(props Attrs) *Element {
	return dom.Section(
		Attrs{
			"id":    "api",
			"class": "py-20 bg-white",
		},
		dom.Div(
			Attrs{"class": "container mx-auto px-6"},
			dom.Div(
				Attrs{"class": "text-center mb-16"},
				dom.H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-6 text-gray-900"}, "API Documentation"),
				dom.P(Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"}, "Complete reference for all GoWebComponents APIs and hooks"),
			),

			// API sections
			dom.Div(
				Attrs{"class": "max-w-6xl mx-auto space-y-16"},

				// Core Components
				ApiSection(
					"Core Components",
					"Essential building blocks for your applications",
					[]ApiItem{
						{"Div", "Container element with props and children", "dom.Div(Attrs{\"class\": \"container\"}, children...)"},
						{"Button", "Interactive button element", "dom.Button(Attrs{\"onclick\": \"handleClick()\"}, \"Click Me\")"},
						{"Input", "Form input element", "Input(Attrs{\"type\": \"text\", \"placeholder\": \"Enter text\"})"},
						{"Form", "Form container element", "Form(Attrs{\"onsubmit\": \"handleSubmit()\"}, children...)"},
					},
				),

				// Hooks
				ApiSection(
					"Hooks",
					"State management and lifecycle hooks",
					[]ApiItem{
						{"hooks.UseState", "Manage component state", "[value, setValue] := hooks.UseState(initialValue)"},
						{"hooks.UseEffect", "Handle side effects and lifecycle", "hooks.UseEffect(func() func() { /* effect */ return nil }, dependencies...)"},
						{"GoUseMemo", "Memoize expensive calculations", "memoizedValue := fiber.GoUseMemo(func() interface{} { return calc() }, deps)"},
						{"GoUseFetch", "Declarative data fetching", "data, loading, err := fiber.GoUseFetch(url, options)"},
					},
				),

				// Utilities
				ApiSection(
					"Utilities",
					"Helper functions and utilities",
					[]ApiItem{
						{"RenderTo", "Render component to DOM element", "fiber.RenderTo(\"#app\", component)"},
						{"GoFetch", "Imperative HTTP requests", "response, err := fiber.GoFetch(url, options)"},
						{"SetDebugMode", "Enable/disable debug logging", "fiber.SetDebugMode(true)"},
						{"EnableHotReload", "Enable hot reload for development", "fiber.EnableHotReload(true)"},
					},
				),
			),

			// Quick reference
			dom.Div(
				Attrs{"class": "mt-20 bg-gradient-to-br from-gray-50 to-indigo-50 rounded-2xl p-8"},
				dom.H3(Attrs{"class": "text-3xl font-bold text-gray-900 mb-6 text-center"}, "Quick Reference"),
				dom.Div(
					Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-8"},

					// Component pattern
					dom.Div(
						Attrs{"class": "bg-white rounded-xl p-6 shadow-lg"},
						dom.H4(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, "Component Pattern"),
						dom.Pre(Attrs{"class": "bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto"},
							dom.Code(nil, `func MyComponent(props Attrs) *Element {
    return dom.Div(
        Attrs{"class": "component"},
        H1(nil, "Hello World"),
        dom.P(nil, "Component content"),
    )
}`)),
					),

					// State management
					dom.Div(
						Attrs{"class": "bg-white rounded-xl p-6 shadow-lg"},
						dom.H4(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, "State Management"),
						dom.Pre(Attrs{"class": "bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto"},
							dom.Code(nil, `func Counter(props Attrs) *Element {
    count, setCount := hooks.UseState(0)
    
    return dom.Div(nil,
        dom.P(nil, fmt.Sprintf("Count: %d", count)),
        dom.Button(Attrs{
            "onclick": func() { setCount(count + 1) },
        }, "Increment"),
    )
}`)),
					),
				),
			),
		),
	)
}

// ApiSection creates a section for API documentation
func ApiSection(title, description string, items []ApiItem) *Element {
	return dom.Div(
		Attrs{"class": ""},
		dom.H3(Attrs{"class": "text-3xl font-bold text-gray-900 mb-4"}, title),
		dom.P(Attrs{"class": "text-lg text-gray-600 mb-8"}, description),
		dom.Div(
			Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-6"},
			func() []interface{} {
				var elements []interface{}
				for _, item := range items {
					elements = append(elements, ApiItemCard(item))
				}
				return elements
			}()...,
		),
	)
}

// ApiItem represents an API documentation item
type ApiItem struct {
	Name        string
	Description string
	Example     string
}

// ApiItemCard creates a card for an API item
func ApiItemCard(item ApiItem) *Element {
	return dom.Div(
		Attrs{"class": "bg-gray-50 rounded-xl p-6 border border-gray-200 hover:border-indigo-200 hover:shadow-lg transition-all duration-300"},
		dom.H4(Attrs{"class": "text-lg font-bold text-gray-900 mb-2"}, item.Name),
		dom.P(Attrs{"class": "text-gray-600 mb-4"}, item.Description),
		dom.Pre(Attrs{"class": "bg-gray-900 text-green-400 p-3 rounded-lg text-sm overflow-x-auto"},
			dom.Code(nil, item.Example)),
	)
}

// FooterSection renders comprehensive site footer with organized links and information.
// Includes project links, documentation navigation, social media, and professional
// contact information in a responsive multi-column layout.
func FooterSection(props Attrs) *Element {
	return dom.Footer(
		Attrs{"class": "bg-gray-900 text-white py-16"},
		dom.Div(
			Attrs{"class": "container mx-auto px-6"},

			// Main footer content
			dom.Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-4 gap-8 mb-12"},

				// Brand section
				dom.Div(
					Attrs{"class": "col-span-1 md:col-span-2"},
					dom.Div(
						Attrs{"class": "flex items-center space-x-3 mb-6"},
						dom.Img(Attrs{
							"src":   "/static/images/hero.jpg",
							"alt":   "GoWebComponents Logo",
							"class": "h-10 w-10 rounded-full border-2 border-indigo-400",
						}),
						dom.H3(Attrs{"class": "text-2xl font-bold bg-gradient-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent"}, "GoWebComponents"),
					),
					dom.P(Attrs{"class": "text-gray-300 leading-relaxed mb-6 max-w-md"},
						"Build modern, reactive web applications using Go and WebAssembly. Created by Earl Cameron for developers who value simplicity, performance, and type safety."),

					// Social links
					dom.Div(
						Attrs{"class": "flex space-x-4"},
						SocialLink("https://github.com/monstercameron/GoWebComponents", "⚡ GitHub"),
						SocialLink("https://earlcameron.com", "🌐 Website"),
						SocialLink("mailto:earl@earlcameron.com", "📧 Contact"),
					),
				),

				// Quick links
				dom.Div(
					Attrs{"class": ""},
					dom.H4(Attrs{"class": "text-lg font-bold mb-4"}, "Quick Links"),
					dom.Ul(Attrs{"class": "space-y-2"},
						FooterLink("Getting Started", "#getting-started"),
						FooterLink("Examples", "#examples"),
						FooterLink("API Docs", "#api"),
						FooterLink("Features", "#features"),
					),
				),

				// Resources
				dom.Div(
					Attrs{"class": ""},
					dom.H4(Attrs{"class": "text-lg font-bold mb-4"}, "Resources"),
					dom.Ul(Attrs{"class": "space-y-2"},
						FooterLink("Hot Reload Guide", "#live-reload"),
						FooterLink("Performance Tips", "#performance"),
						FooterLink("Best Practices", "#best-practices"),
						FooterLink("Troubleshooting", "#troubleshooting"),
					),
				),
			),

			// Bottom bar
			dom.Div(
				Attrs{"class": "border-t border-gray-700 pt-8 flex flex-col md:flex-row justify-between items-center"},
				dom.P(Attrs{"class": "text-gray-400 text-sm mb-4 md:mb-0"},
					"© 2024 Earl Cameron. Built with GoWebComponents. All rights reserved."),

				dom.Div(
					Attrs{"class": "flex items-center space-x-4 text-sm text-gray-400"},
					dom.Span(nil, "Made with ❤️ and Go"),
					dom.Span(nil, "•"),
					dom.Span(nil, "WebAssembly Powered"),
					dom.Span(nil, "•"),
					dom.Span(nil, "Type Safe"),
				),
			),
		),
	)
}

// SocialLink creates a social media link
func SocialLink(href, text string) *Element {
	return dom.A(
		Attrs{
			"href":   href,
			"target": "_blank",
			"class":  "text-gray-300 hover:text-white transition-colors duration-200 text-sm",
		},
		text,
	)
}

// FooterLink creates a footer navigation link
func FooterLink(text, href string) *Element {
	return dom.Li(
		Attrs{"class": ""},
		dom.A(
			Attrs{
				"href":    href,
				"class":   "text-gray-300 hover:text-white transition-colors duration-200 text-sm",
				"onclick": "scrollToSection('" + href[1:] + "')",
			},
			text,
		),
	)
}





