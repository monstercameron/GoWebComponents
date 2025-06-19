//go:build js && wasm
// +build js,wasm

package website

import . "github.com/monstercameron/GoWebComponents/fiber"

// ExampleModal creates a modal container for displaying examples
func ExampleModal(props Attrs) *Element {
	return Div(
		Attrs{
			"id":    "example-modal",
			"class": "fixed inset-0 bg-black/50 dark:bg-black/70 backdrop-blur-sm z-50 hidden opacity-0 transition-all duration-300",
		},
		Div(
			Attrs{
				"class":   "flex items-center justify-center min-h-screen p-4",
				"onclick": "closeExampleModal(event)",
			},
			Div(
				Attrs{
					"class":   "bg-white dark:bg-gray-800 dark:text-gray-100 rounded-2xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden",
					"onclick": "event.stopPropagation()",
				},

				// Modal header
				Div(
					Attrs{"class": "flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700 bg-gradient-to-r from-indigo-50 to-purple-50 dark:from-gray-800 dark:to-gray-700 dark:bg-gradient-to-r"},
					H3(
						Attrs{"id": "modal-title", "class": "text-2xl font-bold text-gray-900 dark:text-gray-100"},
						"Example",
					),
					Button(
						Attrs{
							"class":   "p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-full transition-colors duration-200",
							"onclick": "closeExampleModal()",
						},
						Span(Attrs{"class": "text-2xl text-gray-500"}, "×"),
					),
				),

				// Modal content
				Div(
					Attrs{
						"id":    "modal-content",
						"class": "p-6 overflow-y-auto max-h-[70vh] dark:bg-gray-900 dark:text-gray-100",
					},
					P(Attrs{"class": "text-gray-500 dark:text-gray-400"}, "Loading example..."),
				),

				// Modal footer
				Div(
					Attrs{"class": "flex justify-between items-center p-6 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800"},
					Div(
						Attrs{"class": "flex space-x-3"},
						Button(
							Attrs{
								"id":      "view-source-btn",
								"class":   "px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-100 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors duration-200 font-medium",
								"onclick": "toggleSourceView()",
							},
							"View Source",
						),
						Button(
							Attrs{
								"id":      "reset-example-btn",
								"class":   "px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors duration-200 font-medium",
								"onclick": "resetExample()",
							},
							"Reset",
						),
					),
					Button(
						Attrs{
							"class":   "px-6 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors duration-200 font-medium",
							"onclick": "closeExampleModal()",
						},
						"Close",
					),
				),
			),
		),
	)
}

// ClickCounter creates a working click counter example
func ClickCounter(props Attrs) *Element {
	// This would use GoUseState in a real implementation
	return Div(
		Attrs{"class": "text-center p-8 bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-900 dark:to-indigo-900 dark:text-gray-100 rounded-xl"},
		H2(Attrs{"class": "text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6"}, "Click Counter"),
		Div(
			Attrs{"class": "mb-8"},
			Div(
				Attrs{
					"id":    "counter-display",
					"class": "text-6xl font-bold text-indigo-600 mb-4",
				},
				"0",
			),
			P(Attrs{"class": "text-gray-600 dark:text-gray-300"}, "Click the button to increment the counter"),
		),
		Div(
			Attrs{"class": "space-x-4"},
			Button(
				Attrs{
					"class":   "px-6 py-3 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors duration-200 font-semibold shadow-lg hover:shadow-xl transform hover:-translate-y-0.5",
					"onclick": "incrementCounter()",
				},
				"+ Increment",
			),
			Button(
				Attrs{
					"class":   "px-6 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors duration-200 font-semibold shadow-lg hover:shadow-xl transform hover:-translate-y-0.5",
					"onclick": "resetCounter()",
				},
				"↻ Reset",
			),
		),
	)
}

// TodoApp creates a working todo application example
func TodoApp(props Attrs) *Element {
	return Div(
		Attrs{"class": "max-w-md mx-auto bg-white dark:bg-gray-800 dark:text-gray-100 rounded-xl shadow-lg p-6"},
		H2(Attrs{"class": "text-2xl font-bold text-gray-900 mb-6 text-center"}, "Todo App"),

		// Add todo form
		Div(
			Attrs{"class": "mb-6"},
			Div(
				Attrs{"class": "flex space-x-2"},
				Input(Attrs{
					"id":          "todo-input",
					"type":        "text",
					"placeholder": "Add a new todo...",
					"class":       "flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent",
					"onkeypress":  "handleTodoKeyPress(event)",
				}),
				Button(
					Attrs{
						"class":   "px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors duration-200 font-medium",
						"onclick": "addTodo()",
					},
					"Add",
				),
			),
		),

		// Todo list
		Div(
			Attrs{
				"id":    "todo-list",
				"class": "space-y-2",
			},
			// Initial todos will be added by JavaScript
			TodoItem("Learn GoWebComponents", false, 0),
			TodoItem("Build awesome apps", false, 1),
			TodoItem("Share with the world", false, 2),
		),

		// Stats
		Div(
			Attrs{"class": "mt-6 text-center text-sm text-gray-500"},
			Span(Attrs{"id": "todo-stats"}, "3 items remaining"),
		),
	)
}

// TodoItem creates a single todo item
func TodoItem(text string, completed bool, id int) *Element {
	completedClass := ""
	if completed {
		completedClass = "line-through text-gray-500"
	}

	return Div(
		Attrs{
			"class":        "flex items-center space-x-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors duration-200 dark:text-gray-100",
			"data-todo-id": string(rune(id)),
		},
		Input(Attrs{
			"type":     "checkbox",
			"class":    "w-4 h-4 text-indigo-600 rounded focus:ring-indigo-500",
			"onchange": "toggleTodo(" + string(rune(id)) + ")",
		}),
		Span(
			Attrs{"class": "flex-1 " + completedClass},
			text,
		),
		Button(
			Attrs{
				"class":   "text-red-500 hover:text-red-700 transition-colors duration-200",
				"onclick": "deleteTodo(" + string(rune(id)) + ")",
			},
			"🗑️",
		),
	)
}

// Dashboard creates a simple dashboard example
func Dashboard(props Attrs) *Element {
	return Div(
		Attrs{"class": "p-6 bg-gradient-to-br from-gray-50 to-blue-50 dark:from-gray-800 dark:to-gray-900 dark:text-gray-100 rounded-xl"},
		H2(Attrs{"class": "text-3xl font-bold text-gray-900 mb-8 text-center"}, "Dashboard"),

		// Stats cards
		Div(
			Attrs{"class": "grid grid-cols-1 md:grid-cols-3 gap-6 mb-8"},
			DashboardCard("👥 Users", "1,234", "↗ +12%", "text-green-500"),
			DashboardCard("📊 Revenue", "$45,678", "↗ +8%", "text-green-500"),
			DashboardCard("🚀 Growth", "23%", "↘ -2%", "text-red-500"),
		),

		// Activity feed
		Div(
			Attrs{"class": "bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6"},
			H3(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, "Recent Activity"),
			Div(
				Attrs{"class": "space-y-3"},
				ActivityItem("🎉", "New user registered", "2 minutes ago"),
				ActivityItem("💳", "Payment received", "5 minutes ago"),
				ActivityItem("📈", "Report generated", "10 minutes ago"),
				ActivityItem("🔧", "System maintenance", "1 hour ago"),
			),
		),
	)
}

// DashboardCard creates a stats card for the dashboard
func DashboardCard(title, value, change, changeColor string) *Element {
	return Div(
		Attrs{"class": "bg-white dark:bg-gray-800 dark:text-gray-100 rounded-lg shadow-lg p-6 hover:shadow-xl transition-shadow duration-300"},
		H4(Attrs{"class": "text-sm font-medium text-gray-500 mb-2"}, title),
		P(Attrs{"class": "text-3xl font-bold text-gray-900 mb-1"}, value),
		P(Attrs{"class": "text-sm " + changeColor}, change),
	)
}

// ActivityItem creates an activity feed item
func ActivityItem(icon, message, time string) *Element {
	return Div(
		Attrs{"class": "flex items-center space-x-3 p-3 hover:bg-gray-50 dark:hover:bg-gray-700 rounded-lg transition-colors duration-200 dark:text-gray-100"},
		Span(Attrs{"class": "text-2xl"}, icon),
		Div(
			Attrs{"class": "flex-1"},
			P(Attrs{"class": "text-sm font-medium text-gray-900"}, message),
			P(Attrs{"class": "text-xs text-gray-500"}, time),
		),
	)
}

// GetExampleContent returns the appropriate example component based on ID
func GetExampleContent(exampleId string) *Element {
	switch exampleId {
	case "click-counter":
		return ClickCounter(nil)
	case "todo-app":
		return TodoApp(nil)
	case "dashboard":
		return Dashboard(nil)
	default:
		return Div(
			Attrs{"class": "text-center p-8"},
			H3(Attrs{"class": "text-xl font-bold text-gray-900 mb-4"}, "Example Not Found"),
			P(Attrs{"class": "text-gray-600"}, "The requested example could not be loaded."),
		)
	}
}

// ExampleSourceCode returns the source code for examples
func GetExampleSourceCode(exampleId string) string {
	switch exampleId {
	case "click-counter":
		return `func ClickCounter(props Attrs) *Element {
    count, setCount := fiber.GoUseState(0)
    
    return Div(
        Attrs{"class": "text-center p-8"},
        H2(nil, "Click Counter"),
        Div(
            Attrs{"class": "text-6xl font-bold mb-4"},
            fmt.Sprintf("%d", count),
        ),
        Button(
            Attrs{
                "onclick": func() { setCount(count + 1) },
                "class": "px-6 py-3 bg-indigo-600 text-white rounded-lg",
            },
            "Increment",
        ),
    )
}`
	case "todo-app":
		return `func TodoApp(props Attrs) *Element {
    todos, setTodos := fiber.GoUseState([]Todo{})
    
    addTodo := func(text string) {
        newTodos := append(todos, Todo{
            ID: len(todos),
            Text: text,
            Done: false,
        })
        setTodos(newTodos)
    }
    
    return Div(
        Attrs{"class": "max-w-md mx-auto p-6"},
        H2(nil, "Todo App"),
        TodoForm(Attrs{"onAdd": addTodo}),
        TodoList(Attrs{"todos": todos, "onToggle": toggleTodo}),
    )
}`
	case "dashboard":
		return `func Dashboard(props Attrs) *Element {
    stats, setStats := fiber.GoUseState(getInitialStats())
    
    fiber.GoUseEffect(func() {
        // Update stats every 5 seconds
        timer := time.NewTicker(5 * time.Second)
        go func() {
            for range timer.C {
                setStats(updateStats())
            }
        }()
        return func() { timer.Stop() }
    }, []interface{}{})
    
    return Div(
        Attrs{"class": "p-6"},
        H2(nil, "Dashboard"),
        StatsGrid(Attrs{"stats": stats}),
        ActivityFeed(nil),
    )
}`
	default:
		return "// Example source code not available"
	}
}
