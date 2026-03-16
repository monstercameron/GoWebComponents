//go:build js && wasm
// +build js,wasm

package main

// AboutSection explains the value proposition of GoWebComponents over traditional development.
// Features side-by-side comparison cards and key statistics to highlight framework benefits.
func AboutSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "about",
			"class": "py-20 bg-[#0a0a0a]",
		},
		Div(
			Attrs{"class": "container mx-auto px-6"},
			Div(
				Attrs{"class": "max-w-4xl mx-auto text-center mb-16"},
				H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-8 text-white"}, "Why GoWebComponents?"),
				P(
					Attrs{"class": "text-xl text-gray-400 leading-relaxed mb-8"},
					"Born from the need to build complex, performant web applications without the JavaScript ecosystem's complexity. ",
					"GoWebComponents brings Go's elegance, safety, and performance to the frontend.",
				),
			),

			// Comparison cards
			Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-2 gap-12 max-w-6xl mx-auto"},

				// Traditional approach
				Div(
					Attrs{"class": "bg-red-900/10 border border-red-500/20 rounded-2xl p-8 backdrop-blur-sm"},
					H3(Attrs{"class": "text-2xl font-bold text-red-400 mb-6 flex items-center"},
						Span(Attrs{"class": "mr-3"}, "❌"),
						"Traditional Web Development"),
					Ul(Attrs{"class": "space-y-4 text-red-300"},
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Complex build pipelines and toolchains"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Runtime errors and type coercion issues"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Separate backend/frontend codebases"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Heavy node_modules dependencies"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"State management complexity"),
					),
				),

				// GoWebComponents approach
				Div(
					Attrs{"class": "bg-green-900/10 border border-green-500/20 rounded-2xl p-8 backdrop-blur-sm"},
					H3(Attrs{"class": "text-2xl font-bold text-green-400 mb-6 flex items-center"},
						Span(Attrs{"class": "mr-3"}, "✅"),
						"GoWebComponents Approach"),
					Ul(Attrs{"class": "space-y-4 text-green-300"},
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Single Go codebase for everything"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Compile-time error checking"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Shared types and logic"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Zero external dependencies"),
						Li(Attrs{"class": "flex items-start"},
							Span(Attrs{"class": "mr-3 mt-1"}, "•"),
							"Built-in state management"),
					),
				),
			),

			// Stats section
			Div(
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
	return Div(
		Attrs{"class": "text-center bg-white/5 rounded-xl p-6 shadow-lg border border-white/10 hover:border-white/20 transition-all duration-300 backdrop-blur-sm"},
		Div(Attrs{"class": "text-4xl font-bold text-indigo-400 mb-2"}, stat),
		H4(Attrs{"class": "text-xl font-bold text-white mb-2"}, title),
		P(Attrs{"class": "text-gray-400"}, description),
	)
}

// GettingStartedSection provides step-by-step instructions for new developers.
// Includes installation commands, running examples, and a simple component example
// to demonstrate the framework's ease of use.
func GettingStartedSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "getting-started",
			"class": "py-20 bg-white/5",
		},
		Div(
			Attrs{"class": "container mx-auto px-6"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-6 text-white"}, "Get Started"),
				P(Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"}, "Start building with GoWebComponents in minutes, not hours"),
			),

			// Installation steps
			Div(
				Attrs{"class": "max-w-4xl mx-auto space-y-12"},

				InstallationStep("1", "Clone the Repository",
					Code(Attrs{"class": "bg-black/50 text-green-400 p-4 rounded-lg block border border-white/10"},
						"git clone https://github.com/monstercameron/GoWebComponents.git")),

				InstallationStep("2", "Run the Examples",
					Code(Attrs{"class": "bg-black/50 text-green-400 p-4 rounded-lg block border border-white/10"},
						"cd GoWebComponents && go run main.go")),

				InstallationStep("3", "Start Building",
					P(Attrs{"class": "text-gray-400"},
						"Create your first component using familiar Go syntax. Check out the examples folder for inspiration!")),
			),

			// Quick example
			Div(
				Attrs{"class": "mt-16 bg-white/5 rounded-2xl p-8 border border-white/10"},
				H3(Attrs{"class": "text-2xl font-bold text-white mb-6 text-center"}, "Your First Component"),
				Pre(Attrs{"class": "bg-black/50 text-gray-300 p-6 rounded-lg overflow-x-auto text-sm border border-white/10"},
					Code(nil, `func HelloWorld(props Attrs) *Element {
    return Div(
        Attrs{"class": "text-center p-8"},
        H1(nil, "Hello, GoWebComponents!"),
        P(nil, "Building reactive UIs with Go"),
        Button(
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
	return Div(
		Attrs{"class": "flex items-start space-x-6"},
		Div(
			Attrs{"class": "flex-shrink-0 w-12 h-12 bg-indigo-600 text-white rounded-full flex items-center justify-center text-xl font-bold shadow-lg shadow-indigo-500/20"},
			number,
		),
		Div(
			Attrs{"class": "flex-1"},
			H3(Attrs{"class": "text-2xl font-bold text-white mb-4"}, title),
			content,
		),
	)
}

// ExamplesSection displays a gallery of interactive examples with live demos.
// Each example card includes a working demo and links to detailed explanations
// and source code viewing functionality.
func ExamplesSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "examples",
			"class": "py-20 bg-[#0a0a0a]",
		},
		Div(
			Attrs{"class": "container mx-auto px-6"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-6 bg-gradient-to-r from-purple-400 to-indigo-400 bg-clip-text text-transparent"}, "Examples"),
				P(Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"}, "See GoWebComponents in action with these real-world examples"),
			),

			// Example categories
			Div(
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
			Div(
				Attrs{"class": "mt-20 bg-white/5 rounded-2xl shadow-xl p-8 max-w-4xl mx-auto border border-white/10 backdrop-blur-sm"},
				H3(Attrs{"class": "text-3xl font-bold text-white mb-6 text-center"}, "Try It Live"),
				P(Attrs{"class": "text-lg text-gray-400 text-center mb-8"}, "Experience the power of GoWebComponents right in your browser"),

				// Interactive demo buttons
				Div(
					Attrs{"class": "flex flex-wrap justify-center gap-4 mb-8"},
					DemoButton("Counter", "loadCounterDemo()"),
					DemoButton("Todo List", "loadTodoDemo()"),
					DemoButton("Dashboard", "loadDashboardDemo()"),
				),

				// Demo container
				Div(
					Attrs{
						"id":    "demo-container",
						"class": "bg-black/20 rounded-lg p-6 min-h-64 flex items-center justify-center border-2 border-dashed border-white/10",
					},
					P(Attrs{"class": "text-gray-500 text-lg"}, "Select an example above to see it in action"),
				),
			),
		),
	)
}

// ExampleCard creates a card for showcasing an example
func ExampleCard(icon, title, description, code, id string) *Element {
	return Div(
		Attrs{"class": "bg-white/5 rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 p-6 border border-white/10 group hover:scale-105 hover:bg-white/10 backdrop-blur-sm"},
		Div(
			Attrs{"class": "text-center mb-6"},
			Div(Attrs{"class": "text-4xl mb-4 group-hover:scale-110 transition-transform duration-300"}, icon),
			H3(Attrs{"class": "text-xl font-bold text-white mb-2"}, title),
			P(Attrs{"class": "text-gray-400 text-sm leading-relaxed"}, description),
		),

		Div(
			Attrs{"class": "space-y-4"},
			// Code snippet
			Pre(Attrs{"class": "bg-black/50 text-green-400 p-3 rounded-lg text-xs overflow-x-auto border border-white/5"},
				Code(nil, code)),

			// Action buttons
			Div(
				Attrs{"class": "flex space-x-3"},
				Button(
					Attrs{
						"class":   "flex-1 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors duration-200 text-sm font-medium shadow-lg shadow-indigo-500/20",
						"onclick": "runExample('" + id + "')",
					},
					"▶ Run",
				),
				Button(
					Attrs{
						"class":   "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors duration-200 text-sm font-medium border border-white/10",
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
	return Button(
		Attrs{
			"class":   "px-6 py-3 bg-gradient-to-r from-purple-600 to-indigo-600 text-white rounded-lg hover:from-purple-700 hover:to-indigo-700 transition-all duration-200 font-medium shadow-md hover:shadow-lg transform hover:-translate-y-0.5",
			"onclick": onclick,
		},
		label,
	)
}

// ApiDocumentationSection provides comprehensive API documentation
func ApiDocumentationSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "api",
			"class": "py-20 bg-white/5",
		},
		Div(
			Attrs{"class": "container mx-auto px-6"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(Attrs{"class": "text-4xl md:text-5xl font-bold mb-6 text-white"}, "API Documentation"),
				P(Attrs{"class": "text-xl text-gray-400 max-w-3xl mx-auto"}, "Complete reference for all GoWebComponents APIs and hooks"),
			),

			// API sections
			Div(
				Attrs{"class": "max-w-6xl mx-auto space-y-16"},

				// Core Components
				ApiSection(
					"Core Components",
					"Essential building blocks for your applications",
					[]ApiItem{
						{"Div", "Container element with props and children", "Div(Attrs{\"class\": \"container\"}, children...)"},
						{"Button", "Interactive button element", "Button(Attrs{\"onclick\": ui.UseEvent(func(event ui.MouseEvent) { /* ... */ })}, \"Click Me\")"},
						{"Input", "Form input element", "Input(Attrs{\"type\": \"text\", \"placeholder\": \"Enter text\"})"},
						{"Form", "Form container element", "Form(Attrs{\"onsubmit\": ui.UseEvent(func(event ui.FormEvent) { event.PreventDefault() })}, children...)"},
					},
				),

				// Hooks
				ApiSection(
					"Hooks",
					"State management and lifecycle hooks",
					[]ApiItem{
						{"UseState", "Manage component state", "state := ui.UseState(initialValue); current := state.Get(); state.Set(nextValue)"},
						{"UseEffect", "Handle side effects and lifecycle", "UseEffect(func() func() { /* effect */ return nil }, dependencies...)"},
						{"UseMemo", "Memoize expensive calculations", "memoizedValue := ui.UseMemo(func() Result { return calc() }, deps...)"},
						{"UseFetch", "Raw fetch state for URL-driven requests", "resource := fetch.UseFetch(url); state := resource.Get()"},
						{"UseResource", "Typed async loading with cancellation", "resource := fetch.UseResource(loader, deps...)"},
					},
				),

				// Utilities
				ApiSection(
					"Utilities",
					"Helper functions and utilities",
					[]ApiItem{
						{"Render", "Mount a component tree into the DOM", "ui.Render(ui.CreateElement(App), \"#app\")"},
						{"Hydrate", "Resume a server-rendered tree on the client", "ui.Hydrate(ui.CreateElement(App), \"#app\")"},
						{"Fetch", "Imperative HTTP requests", "resultChan := fetch.Fetch(url, fetch.Options{Method: \"GET\"})"},
						{"SetDebugNamespacesExclusive", "Focus debug output on selected subsystems", "utils.SetDebugNamespacesExclusive(map[string]bool{\"FETCH\": true})"},
						{"EnableHotReload", "Enable hot reload for development", "utils.EnableHotReload(true)"},
					},
				),
			),

			// Quick reference
			Div(
				Attrs{"class": "mt-20 bg-white/5 rounded-2xl p-8 border border-white/10 backdrop-blur-sm"},
				H3(Attrs{"class": "text-3xl font-bold text-white mb-6 text-center"}, "Quick Reference"),
				Div(
					Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-8"},

					// Component pattern
					Div(
						Attrs{"class": "bg-black/20 rounded-xl p-6 shadow-lg border border-white/10"},
						H4(Attrs{"class": "text-xl font-bold text-white mb-4"}, "Component Pattern"),
						Pre(Attrs{"class": "bg-black/50 text-gray-300 p-4 rounded-lg text-sm overflow-x-auto border border-white/5"},
							Code(nil, `func MyComponent(props Attrs) *Element {
    return Div(
        Attrs{"class": "component"},
        H1(nil, "Hello World"),
        P(nil, "Component content"),
    )
}`)),
					),

					// State management
					Div(
						Attrs{"class": "bg-black/20 rounded-xl p-6 shadow-lg border border-white/10"},
						H4(Attrs{"class": "text-xl font-bold text-white mb-4"}, "State Management"),
						Pre(Attrs{"class": "bg-black/50 text-gray-300 p-4 rounded-lg text-sm overflow-x-auto border border-white/5"},
							Code(nil, `func Counter(props Attrs) *Element {
    count, setCount := UseState(0)
    
    return Div(nil,
        P(nil, fmt.Sprintf("Count: %d", count)),
        Button(Attrs{
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
	return Div(
		Attrs{"class": ""},
		H3(Attrs{"class": "text-3xl font-bold text-white mb-4"}, title),
		P(Attrs{"class": "text-lg text-gray-400 mb-8"}, description),
		Div(
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
	return Div(
		Attrs{"class": "bg-white/5 rounded-xl p-6 border border-white/10 hover:border-indigo-500/50 hover:shadow-lg transition-all duration-300 backdrop-blur-sm"},
		H4(Attrs{"class": "text-lg font-bold text-white mb-2"}, item.Name),
		P(Attrs{"class": "text-gray-400 mb-4"}, item.Description),
		Pre(Attrs{"class": "bg-black/50 text-green-400 p-3 rounded-lg text-sm overflow-x-auto border border-white/5"},
			Code(nil, item.Example)),
	)
}

// FooterSection renders comprehensive site footer with organized links and information.
// Includes project links, documentation navigation, social media, and professional
// contact information in a responsive multi-column layout.
func FooterSection(props Attrs) *Element {
	return Footer(
		Attrs{"class": "bg-[#0a0a0a] text-white py-16 border-t border-white/10"},
		Div(
			Attrs{"class": "container mx-auto px-6"},

			// Main footer content
			Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-4 gap-8 mb-12"},

				// Brand section
				Div(
					Attrs{"class": "col-span-1 md:col-span-2"},
					Div(
						Attrs{"class": "flex items-center space-x-3 mb-6"},
						Img(Attrs{
							"src":   "/static/images/hero.jpg",
							"alt":   "GoWebComponents Logo",
							"class": "h-10 w-10 rounded-full border-2 border-indigo-400",
						}),
						H3(Attrs{"class": "text-2xl font-bold bg-gradient-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent"}, "GoWebComponents"),
					),
					P(Attrs{"class": "text-gray-400 leading-relaxed mb-6 max-w-md"},
						"Build modern, reactive web applications using Go and WebAssembly. Created by Earl Cameron for developers who value simplicity, performance, and type safety."),

					// Social links
					Div(
						Attrs{"class": "flex space-x-4"},
						SocialLink("https://github.com/monstercameron/GoWebComponents", "⚡ GitHub"),
						SocialLink("https://earlcameron.com", "🌐 Website"),
						SocialLink("mailto:earl@earlcameron.com", "📧 Contact"),
					),
				),

				// Quick links
				Div(
					Attrs{"class": ""},
					H4(Attrs{"class": "text-lg font-bold mb-4 text-white"}, "Quick Links"),
					Ul(Attrs{"class": "space-y-2"},
						FooterLink("Getting Started", "#gwc-showcase"),
						FooterLink("Examples", "#examples"),
						FooterLink("API Docs", "#api"),
						FooterLink("Features", "#features"),
					),
				),

				// Resources
				Div(
					Attrs{"class": ""},
					H4(Attrs{"class": "text-lg font-bold mb-4 text-white"}, "Resources"),
					Ul(Attrs{"class": "space-y-2"},
						FooterLink("Hot Reload Guide", "#gwc-showcase"),
						FooterLink("Performance Tips", "#features"),
						FooterLink("Best Practices", "#api"),
						FooterLink("Troubleshooting", "#contact"),
					),
				),
			),

			// Bottom bar
			Div(
				Attrs{"class": "border-t border-white/10 pt-8 flex flex-col md:flex-row justify-between items-center"},
				P(Attrs{"class": "text-gray-400 text-sm mb-4 md:mb-0"},
					"© 2024 Earl Cameron. Built with GoWebComponents. All rights reserved."),

				Div(
					Attrs{"class": "flex items-center space-x-4 text-sm text-gray-400"},
					Span(nil, "Made with ❤️ and Go"),
					Span(nil, "•"),
					Span(nil, "WebAssembly Powered"),
					Span(nil, "•"),
					Span(nil, "Type Safe"),
				),
			),
		),
	)
}

// SocialLink creates a social media link
func SocialLink(href, text string) *Element {
	return A(
		Attrs{
			"href":   href,
			"target": "_blank",
			"class":  "text-gray-400 hover:text-white transition-colors duration-200 text-sm",
		},
		text,
	)
}

// FooterLink creates a footer navigation link
func FooterLink(text, href string) *Element {
	return Li(
		Attrs{"class": ""},
		A(
			Attrs{
				"href":    href,
				"class":   "text-gray-400 hover:text-white transition-colors duration-200 text-sm",
				"onclick": "scrollToSection('" + href[1:] + "')",
			},
			text,
		),
	)
}
