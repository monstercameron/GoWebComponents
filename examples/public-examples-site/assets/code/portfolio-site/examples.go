//go:build js && wasm
// +build js,wasm

package main

// ExampleModal renders a full-screen modal for showcasing interactive examples.
// Features source code viewing, example reset functionality, and responsive design
// with backdrop blur and smooth animations.
func ExampleModal(_ Attrs) *Element {
	return Div(
		Attrs{
			"id":    "example-modal",
			"class": "fixed inset-0 bg-black/80 backdrop-blur-sm z-50 hidden opacity-0 transition-all duration-300",
		},
		Div(
			Attrs{
				"class":   "flex items-center justify-center min-h-screen p-4",
				"onclick": "closeExampleModal(event)",
			},
			Div(
				Attrs{
					"class":   "bg-[#0a0a0a] border border-white/10 rounded-2xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden",
					"onclick": "event.stopPropagation()",
				},

				// Modal header
				Div(
					Attrs{"class": "flex items-center justify-between p-6 border-b border-white/10 bg-white/5"},
					H3(
						Attrs{"id": "modal-title", "class": "text-2xl font-bold text-white"},
						"Example",
					),
					Button(
						Attrs{
							"class":   "p-2 hover:bg-white/10 rounded-full transition-colors duration-200 text-gray-400 hover:text-white",
							"onclick": "closeExampleModal()",
						},
						Span(Attrs{"class": "text-2xl"}, "×"),
					),
				),

				// Modal content
				Div(
					Attrs{
						"id":    "modal-content",
						"class": "p-6 overflow-y-auto max-h-[70vh] bg-transparent text-gray-100",
					},
					P(Attrs{"class": "text-gray-400"}, "Loading example..."),
				),

				// Modal footer
				Div(
					Attrs{"class": "flex justify-between items-center p-6 border-t border-white/10 bg-white/5"},
					Div(
						Attrs{"class": "flex space-x-3"},
						Button(
							Attrs{
								"id":      "view-source-btn",
								"class":   "px-4 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors duration-200 font-medium",
								"onclick": "toggleSourceView()",
							},
							"View Source",
						),
						Button(
							Attrs{
								"id":      "reset-example-btn",
								"class":   "px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200 font-medium",
								"onclick": "resetExample()",
							},
							"Reset",
						),
					),
					Button(
						Attrs{
							"class":   "px-6 py-2 bg-white/10 text-white rounded-lg hover:bg-white/20 transition-colors duration-200 font-medium",
							"onclick": "closeExampleModal()",
						},
						"Close",
					),
				),
			),
		),
	)
}

// ClickCounter demonstrates basic state management and event handling.
// Shows fundamental GoWebComponents patterns with increment/reset functionality
// and animated visual feedback.
func ClickCounter(_ Attrs) *Element {
	// This would use UseState in a real implementation
	return Div(
		Attrs{"class": "text-center p-8 bg-gradient-to-br from-blue-900/20 to-indigo-900/20 border border-blue-500/30 rounded-xl"},
		H2(Attrs{"class": "text-3xl font-bold text-white mb-6"}, "Click Counter"),
		Div(
			Attrs{"class": "mb-8"},
			Div(
				Attrs{
					"id":    "counter-display",
					"class": "text-6xl font-bold text-blue-400 mb-4",
				},
				"0",
			),
			P(Attrs{"class": "text-gray-400"}, "Click the button to increment the counter"),
		),
		Div(
			Attrs{"class": "space-x-4"},
			Button(
				Attrs{
					"class":   "px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200 font-semibold shadow-lg hover:shadow-xl transform hover:-translate-y-0.5",
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

// TodoApp showcases comprehensive CRUD operations and form handling.
// Includes add/delete/toggle functionality with keyboard shortcuts
// and dynamic stats display.
func TodoApp(_ Attrs) *Element {
	return Div(
		Attrs{"class": "max-w-md mx-auto bg-white/5 border border-white/10 rounded-xl shadow-lg p-6"},
		H2(Attrs{"class": "text-2xl font-bold text-white mb-6 text-center"}, "Todo App"),

		// Add todo form
		Div(
			Attrs{"class": "mb-6"},
			Div(
				Attrs{"class": "flex space-x-2"},
				Input(Attrs{
					"id":          "todo-input",
					"type":        "text",
					"placeholder": "Add a new todo...",
					"class":       "flex-1 px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-white placeholder-gray-500",
					"onkeypress":  "handleTodoKeyPress(event)",
				}),
				Button(
					Attrs{
						"class":   "px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200 font-medium",
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
			Attrs{"class": "mt-6 text-center text-sm text-gray-400"},
			Span(Attrs{"id": "todo-stats"}, "3 items remaining"),
		),
	)
}

// TodoItem renders an individual todo with checkbox, text, and delete button.
// Handles state-based styling for completed items and provides interaction handlers.
func TodoItem(parseText string, isCompleted bool, parseId int) *Element {
	parseCompletedClass := ""
	if isCompleted {
		parseCompletedClass = "line-through text-gray-500"
	}

	return Div(
		Attrs{
			"class":        "flex items-center space-x-3 p-3 bg-white/5 rounded-lg hover:bg-white/10 transition-colors duration-200 text-gray-200",
			"data-todo-id": string(rune(parseId)),
		},
		Input(Attrs{
			"type":     "checkbox",
			"class":    "w-4 h-4 text-blue-600 rounded focus:ring-blue-500 bg-black/20 border-white/10",
			"onchange": "toggleTodo(" + string(rune(parseId)) + ")",
		}),
		Span(
			Attrs{"class": "flex-1 " + parseCompletedClass},
			parseText,
		),
		Button(
			Attrs{
				"class":   "text-red-500 hover:text-red-400 transition-colors duration-200",
				"onclick": "deleteTodo(" + string(rune(parseId)) + ")",
			},
			"🗑️",
		),
	)
}

// Dashboard demonstrates complex UI composition with stats cards and activity feeds.
// Shows how to structure data-driven interfaces with responsive grid layouts
// and consistent visual hierarchy.
func Dashboard(_ Attrs) *Element {
	return Div(
		Attrs{"class": "p-6 bg-gradient-to-br from-gray-900/50 to-blue-900/20 border border-white/10 rounded-xl"},
		H2(Attrs{"class": "text-3xl font-bold text-white mb-8 text-center"}, "Dashboard"),

		// Stats cards
		Div(
			Attrs{"class": "grid grid-cols-1 md:grid-cols-3 gap-6 mb-8"},
			DashboardCard("👥 Users", "1,234", "↗ +12%", "text-green-400"),
			DashboardCard("📊 Revenue", "$45,678", "↗ +8%", "text-green-400"),
			DashboardCard("🚀 Growth", "23%", "↘ -2%", "text-red-400"),
		),

		// Activity feed
		Div(
			Attrs{"class": "bg-white/5 border border-white/10 rounded-lg shadow-lg p-6"},
			H3(Attrs{"class": "text-xl font-bold text-white mb-4"}, "Recent Activity"),
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

// DashboardCard renders a metric display with value, title, and trend indicator.
// Supports color-coded change indicators and hover animations.
func DashboardCard(parseTitle, parseValue, parseChange, parseChangeColor string) *Element {
	return Div(
		Attrs{"class": "bg-white/5 border border-white/10 rounded-lg shadow-lg p-6 hover:shadow-xl transition-shadow duration-300"},
		H4(Attrs{"class": "text-sm font-medium text-gray-400 mb-2"}, parseTitle),
		P(Attrs{"class": "text-3xl font-bold text-white mb-1"}, parseValue),
		P(Attrs{"class": "text-sm " + parseChangeColor}, parseChange),
	)
}

// ActivityItem displays a single activity with icon, message, and timestamp.
// Provides consistent formatting for activity streams and notification lists.
func ActivityItem(parseIcon, parseMessage, parseTime string) *Element {
	return Div(
		Attrs{"class": "flex items-center space-x-3 p-3 hover:bg-white/5 rounded-lg transition-colors duration-200 text-gray-200"},
		Span(Attrs{"class": "text-2xl"}, parseIcon),
		Div(
			Attrs{"class": "flex-1"},
			P(Attrs{"class": "text-sm font-medium text-gray-200"}, parseMessage),
			P(Attrs{"class": "text-xs text-gray-500"}, parseTime),
		),
	)
}

// GetExampleContent returns the appropriate example component based on ID
func GetExampleContent(parseExampleId string) *Element {
	switch parseExampleId {
	case "click-counter":
		return ClickCounter(nil)
	case "todo-app":
		return TodoApp(nil)
	case "dashboard":
		return Dashboard(nil)
	default:
		return Div(
			Attrs{"class": "text-center p-8"},
			H3(Attrs{"class": "text-xl font-bold text-white mb-4"}, "Example Not Found"),
			P(Attrs{"class": "text-gray-400"}, "The requested example could not be loaded."),
		)
	}
}

// ExampleSourceCode returns the source code for examples
func GetExampleSourceCode(parseExampleId string) string {
	switch parseExampleId {
	case "click-counter":
		return `func ClickCounter(props Attrs) *Element {
    count, setCount := UseState(0)
    
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
    todos, setTodos := UseState([]Todo{})
    
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
    stats, setStats := UseState(getInitialStats())
    
    UseEffect(func() {
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
