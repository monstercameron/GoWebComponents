//go:build js && wasm
// +build js,wasm

package website

import (
	"strconv"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// GWCShowcaseSection highlights GoWebComponents features
func GWCShowcaseSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "gwc-showcase",
			"class": "py-20 bg-gradient-to-br from-gray-900 to-purple-900 text-white",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold mb-4"},
					"Why GoWebComponents?",
				),
				P(
					Attrs{"class": "text-xl text-gray-300 max-w-3xl mx-auto"},
					"Revolutionizing frontend development with the power of Go",
				),
			),

			Div(
				Attrs{"class": "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8"},
				GWCFeatureCard("⚡", "Lightning Fast", "WebAssembly performance with Go's efficiency and memory safety"),
				GWCFeatureCard("🛡️", "Type Safety", "Compile-time error checking eliminates runtime surprises"),
				GWCFeatureCard("🎯", "Developer Experience", "Hot reload, debugging tools, and familiar Go syntax"),
				GWCFeatureCard("🏗️", "Component Architecture", "Reusable, composable components with clear data flow"),
				GWCFeatureCard("🎨", "Modern UI", "Beautiful interfaces with Tailwind CSS integration"),
				GWCFeatureCard("🚀", "Production Ready", "Battle-tested framework with real-world applications"),
			),
		),
	)
}

// GWCFeatureCard creates a feature highlight card
func GWCFeatureCard(icon, title, description string) *Element {
	return Div(
		Attrs{"class": "bg-white/10 backdrop-blur-sm p-6 rounded-xl hover:bg-white/20 transition-all duration-300 border border-white/20"},
		Div(Attrs{"class": "text-3xl mb-4"}, icon),
		H3(Attrs{"class": "text-xl font-semibold mb-3"}, title),
		P(Attrs{"class": "text-gray-300"}, description),
	)
}

// GWCExamplesSection showcases interactive examples
func GWCExamplesSection(props Attrs) *Element {
	return Section(
		Attrs{
			"id":    "examples",
			"class": "py-20 bg-gray-50",
		},
		Div(
			Attrs{"class": "max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"},
			Div(
				Attrs{"class": "text-center mb-16"},
				H2(
					Attrs{"class": "text-4xl font-bold text-gray-900 mb-4"},
					"Live Examples",
				),
				P(
					Attrs{"class": "text-xl text-gray-600 max-w-3xl mx-auto"},
					"See GoWebComponents in action with these interactive demonstrations",
				),
			),

			Div(
				Attrs{"class": "grid grid-cols-1 lg:grid-cols-3 gap-8"},

				// Example cards
				GWCExampleCard("🖱️ Click Counter", "Simple state management demonstration", GWCClickCounter),
				GWCExampleCard("📝 Todo App", "Complete CRUD operations with local state", GWCTodoApp),
				GWCExampleCard("📊 Dashboard", "Real-time data updates and charts", GWCDashboard),
			),
		),
	)
}

// GWCExampleCard creates an example showcase card
func GWCExampleCard(title, description string, component func(Attrs) *Element) *Element {
	return Div(
		Attrs{"class": "bg-white rounded-xl shadow-lg overflow-hidden hover:shadow-xl transition-shadow duration-300"},

		// Header
		Div(
			Attrs{"class": "p-6 border-b border-gray-200"},
			H3(Attrs{"class": "text-lg font-semibold text-gray-900 mb-2"}, title),
			P(Attrs{"class": "text-gray-600 text-sm"}, description),
		),

		// Example component
		Div(
			Attrs{"class": "p-6"},
			component(nil),
		),
	)
}

// GWCClickCounter creates a simple click counter example
func GWCClickCounter(props Attrs) *Element {
	counter, setCounter := GoUseState(0)
	count := counter()

	handleIncrement := GoUseFunc(func(event GoEvent) {
		setCounter(count + 1)
	})

	handleReset := GoUseFunc(func(event GoEvent) {
		setCounter(0)
	})

	return Div(
		Attrs{"class": "text-center"},
		H3(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, "Click Counter"),
		Div(
			Attrs{"class": "mb-4"},
			P(Attrs{"class": "text-3xl font-bold text-indigo-600"}, Text(strconv.Itoa(count))),
			P(Attrs{"class": "text-gray-600"}, "clicks"),
		),
		Div(
			Attrs{"class": "space-x-2"},
			Button(
				Attrs{
					"class":   "px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700",
					"onclick": handleIncrement,
				},
				"Increment",
			),
			Button(
				Attrs{
					"class":   "px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700",
					"onclick": handleReset,
				},
				"Reset",
			),
		),
	)
}

// GWCTodoApp creates a todo app example
func GWCTodoApp(props Attrs) *Element {
	todos, _ := GoUseState([]map[string]interface{}{
		{"id": 1, "text": "Learn GoWebComponents", "done": false},
		{"id": 2, "text": "Build something cool", "done": false},
	})

	todoList := todos()
	remaining := 0
	for _, todo := range todoList {
		if !todo["done"].(bool) {
			remaining++
		}
	}

	return Div(
		Attrs{"class": "text-center"},
		H3(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, "Todo App"),
		P(Attrs{"class": "text-gray-600 mb-4"}, Text(strconv.Itoa(remaining)), " items remaining"),
		Div(
			Attrs{"class": "space-y-2"},
			// Simplified todo list display
			P(Attrs{"class": "text-sm text-gray-500"}, "Interactive todos coming soon!"),
		),
	)
}

// GWCDashboard creates a dashboard example
func GWCDashboard(props Attrs) *Element {
	users, setUsers := GoUseState(1234)
	revenue, setRevenue := GoUseState(45678)

	handleRefresh := GoUseFunc(func(event GoEvent) {
		currentUsers := users()
		currentRevenue := revenue()
		setUsers(1200 + (currentUsers % 100))
		setRevenue(40000 + (currentRevenue % 10000))
	})

	return Div(
		Attrs{"class": "text-center"},
		H3(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, "Dashboard"),
		Div(
			Attrs{"class": "grid grid-cols-2 gap-4 mb-4"},
			Div(
				Attrs{"class": "bg-blue-50 p-3 rounded"},
				P(Attrs{"class": "text-sm text-gray-600"}, "Users"),
				P(Attrs{"class": "text-lg font-bold"}, Text(strconv.Itoa(users()))),
			),
			Div(
				Attrs{"class": "bg-green-50 p-3 rounded"},
				P(Attrs{"class": "text-sm text-gray-600"}, "Revenue"),
				P(Attrs{"class": "text-lg font-bold"}, "$", Text(strconv.Itoa(revenue()))),
			),
		),
		Button(
			Attrs{
				"class":   "px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700",
				"onclick": handleRefresh,
			},
			"Refresh",
		),
	)
}
