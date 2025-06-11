package fiber

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

// Todo represents a single todo item
type Todo struct {
	ID          int        `json:"id"`
	Text        string     `json:"text"`
	Completed   bool       `json:"completed"`
	Priority    string     `json:"priority"` // "low", "medium", "high"
	Category    string     `json:"category"`
	DueDate     string     `json:"dueDate"` // ISO date string
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// TodoFilter represents filter criteria
type TodoFilter struct {
	Status   string // "all", "active", "completed"
	Category string // "" for all, specific category name
	Priority string // "" for all, specific priority
	Search   string // text search
}

// TodoStats represents todo statistics
type TodoStats struct {
	Total      int
	Active     int
	Completed  int
	ByPriority map[string]int
	ByCategory map[string]int
}

// Priority badge component
func PriorityBadge(props Attrs) *Element {
	priority := "medium"
	if props != nil && props["priority"] != nil {
		priority = props["priority"].(string)
	}

	var colorClass string
	switch priority {
	case "high":
		colorClass = "bg-red-100 text-red-800"
	case "medium":
		colorClass = "bg-yellow-100 text-yellow-800"
	case "low":
		colorClass = "bg-green-100 text-green-800"
	default:
		colorClass = "bg-gray-100 text-gray-800"
	}

	return Span(Attrs{
		"class": fmt.Sprintf("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium %s", colorClass),
	}, Text(strings.Title(priority)))
}

// Category badge component
func CategoryBadge(props Attrs) *Element {
	category := ""
	if props != nil && props["category"] != nil {
		category = props["category"].(string)
	}

	if category == "" {
		return Span(nil)
	}

	return Span(Attrs{
		"class": "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 ml-2",
	}, Text(category))
}

// Due date component
func DueDateDisplay(props Attrs) *Element {
	dueDate := ""
	if props != nil && props["dueDate"] != nil {
		dueDate = props["dueDate"].(string)
	}

	if dueDate == "" {
		return Span(nil)
	}

	// Parse date and check if overdue
	isOverdue := false
	displayDate := dueDate
	if parsedDate, err := time.Parse("2006-01-02", dueDate); err == nil {
		if parsedDate.Before(time.Now()) {
			isOverdue = true
		}
		displayDate = parsedDate.Format("Jan 2, 2006")
	}

	colorClass := "text-gray-600"
	if isOverdue {
		colorClass = "text-red-600"
	}

	return Span(Attrs{
		"class": fmt.Sprintf("text-xs %s ml-2", colorClass),
		"title": "Due date",
	}, Text(fmt.Sprintf("📅 %s", displayDate)))
}

// Individual todo item component
func TodoItem(props Attrs) *Element {
	todo := props["todo"].(Todo)
	onToggle := props["onToggle"]
	onDelete := props["onDelete"]
	onEdit := props["onEdit"]

	// Local state for editing
	isEditing, setIsEditing := GoUseState(false)
	editText, setEditText := GoUseState(todo.Text)

	// Handle toggle completion
	handleToggle := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if onToggle != nil {
			onToggle.(func(int))(todo.ID)
		}
	})

	// Handle delete
	handleDelete := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if onDelete != nil {
			onDelete.(func(int))(todo.ID)
		}
	})

	// Handle edit start
	handleEditStart := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		setIsEditing(true)
		setEditText(todo.Text)
	})

	// Handle edit save
	handleEditSave := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if strings.TrimSpace(editText()) != "" {
			if onEdit != nil {
				onEdit.(func(int, string))(todo.ID, editText())
			}
		}
		setIsEditing(false)
	})

	// Handle edit cancel
	handleEditCancel := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		setIsEditing(false)
		setEditText(todo.Text)
	})

	// Handle input change
	handleInputChange := GoUseFunc(func(event GoEvent) {
		setEditText(event.GetValue())
	})

	// Handle key press
	handleKeyPress := GoUseFunc(func(event GoEvent) {
		if event.GetKey() == "Enter" {
			handleEditSave.Call("call", nil, event.Raw())
		} else if event.GetKey() == "Escape" {
			handleEditCancel.Call("call", nil, event.Raw())
		}
	})

	// Determine styling based on completion
	itemClass := "flex items-center p-4 border-b border-gray-200 hover:bg-gray-50 transition-colors"
	textClass := "flex-1 ml-3"
	if todo.Completed {
		textClass += " line-through text-gray-500"
	}

	return Li(Attrs{"class": itemClass},
		// Checkbox
		Input(Attrs{
			"type":     "checkbox",
			"checked":  todo.Completed,
			"onchange": handleToggle,
			"class":    "h-5 w-5 text-blue-600 focus:ring-blue-500 border-gray-300 rounded",
		}),

		// Todo content
		Div(Attrs{"class": textClass},
			func() *Element {
				if isEditing() {
					return Div(Attrs{"class": "flex items-center gap-2"},
						Input(Attrs{
							"type":        "text",
							"value":       editText(),
							"oninput":     handleInputChange,
							"onkeydown":   handleKeyPress,
							"class":       "flex-1 px-2 py-1 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500",
							"placeholder": "Enter todo text...",
						}),
						Button(Attrs{
							"onclick": handleEditSave,
							"class":   "px-3 py-1 bg-green-500 text-white rounded text-sm hover:bg-green-600",
						}, Text("Save")),
						Button(Attrs{
							"onclick": handleEditCancel,
							"class":   "px-3 py-1 bg-gray-500 text-white rounded text-sm hover:bg-gray-600",
						}, Text("Cancel")),
					)
				}
				return Div(nil,
					P(Attrs{"class": "font-medium"}, Text(todo.Text)),
					Div(Attrs{"class": "flex items-center mt-1"},
						PriorityBadge(Attrs{"priority": todo.Priority}),
						CategoryBadge(Attrs{"category": todo.Category}),
						DueDateDisplay(Attrs{"dueDate": todo.DueDate}),
					),
				)
			}(),
		),

		// Action buttons
		Div(Attrs{"class": "flex items-center gap-2 ml-2"},
			Button(Attrs{
				"onclick": handleEditStart,
				"class":   "p-1 text-gray-400 hover:text-blue-600 transition-colors",
				"title":   "Edit todo",
			}, Text("✏️")),
			Button(Attrs{
				"onclick": handleDelete,
				"class":   "p-1 text-gray-400 hover:text-red-600 transition-colors",
				"title":   "Delete todo",
			}, Text("🗑️")),
		),
	)
}

// Todo input form component
func TodoInput(props Attrs) *Element {
	onAdd := props["onAdd"]

	// Form state
	text, setText := GoUseState("")
	priority, setPriority := GoUseState("medium")
	category, setCategory := GoUseState("")
	dueDate, setDueDate := GoUseState("")

	// Handle form submission
	handleSubmit := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		trimmedText := strings.TrimSpace(text())
		if trimmedText != "" && onAdd != nil {
			newTodo := Todo{
				Text:      trimmedText,
				Priority:  priority(),
				Category:  category(),
				DueDate:   dueDate(),
				Completed: false,
				CreatedAt: time.Now(),
			}
			onAdd.(func(Todo))(newTodo)
			// Reset form
			setText("")
			setPriority("medium")
			setCategory("")
			setDueDate("")
		}
	})

	// Handle input changes
	handleTextChange := GoUseFunc(func(event GoEvent) {
		setText(event.GetValue())
	})

	handlePriorityChange := GoUseFunc(func(event GoEvent) {
		setPriority(event.GetValue())
	})

	handleCategoryChange := GoUseFunc(func(event GoEvent) {
		setCategory(event.GetValue())
	})

	handleDueDateChange := GoUseFunc(func(event GoEvent) {
		setDueDate(event.GetValue())
	})

	return Form(Attrs{
		"onsubmit": handleSubmit,
		"class":    "bg-white p-6 rounded-lg shadow-md mb-6",
	},
		H3(Attrs{"class": "text-lg font-semibold mb-4"}, Text("Add New Todo")),

		Div(Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-4 mb-4"},
			// Text input
			Div(Attrs{"class": "md:col-span-2"},
				Input(Attrs{
					"type":        "text",
					"value":       text(),
					"oninput":     handleTextChange,
					"placeholder": "What needs to be done?",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent",
					"required":    true,
				}),
			),

			// Priority select
			Div(nil,
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Priority")),
				Select(Attrs{
					"value":    priority(),
					"onchange": handlePriorityChange,
					"class":    "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				},
					Option(Attrs{"value": "low"}, Text("Low")),
					Option(Attrs{"value": "medium", "selected": priority() == "medium"}, Text("Medium")),
					Option(Attrs{"value": "high"}, Text("High")),
				),
			),

			// Category input
			Div(nil,
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Category")),
				Input(Attrs{
					"type":        "text",
					"value":       category(),
					"oninput":     handleCategoryChange,
					"placeholder": "e.g., Work, Personal",
					"class":       "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				}),
			),

			// Due date
			Div(nil,
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Due Date")),
				Input(Attrs{
					"type":     "date",
					"value":    dueDate(),
					"onchange": handleDueDateChange,
					"class":    "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				}),
			),
		),

		Button(Attrs{
			"type":  "submit",
			"class": "w-full bg-blue-600 text-white py-2 px-4 rounded-lg font-medium hover:bg-blue-700 transition-colors",
		}, Text("Add Todo")),
	)
}

// Filter controls component
func TodoFilters(props Attrs) *Element {
	filter := props["filter"].(TodoFilter)
	onFilterChange := props["onFilterChange"]

	// Handle filter changes
	handleStatusChange := GoUseFunc(func(event GoEvent) {
		newFilter := filter
		newFilter.Status = event.GetValue()
		if onFilterChange != nil {
			onFilterChange.(func(TodoFilter))(newFilter)
		}
	})

	handleSearchChange := GoUseFunc(func(event GoEvent) {
		newFilter := filter
		newFilter.Search = event.GetValue()
		if onFilterChange != nil {
			onFilterChange.(func(TodoFilter))(newFilter)
		}
	})

	handleCategoryChange := GoUseFunc(func(event GoEvent) {
		newFilter := filter
		newFilter.Category = event.GetValue()
		if onFilterChange != nil {
			onFilterChange.(func(TodoFilter))(newFilter)
		}
	})

	handlePriorityChange := GoUseFunc(func(event GoEvent) {
		newFilter := filter
		newFilter.Priority = event.GetValue()
		if onFilterChange != nil {
			onFilterChange.(func(TodoFilter))(newFilter)
		}
	})

	return Div(Attrs{"class": "bg-white p-4 rounded-lg shadow-md mb-6"},
		H3(Attrs{"class": "text-lg font-semibold mb-4"}, Text("Filters")),

		Div(Attrs{"class": "grid grid-cols-1 md:grid-cols-4 gap-4"},
			// Search
			Div(nil,
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Search")),
				Input(Attrs{
					"type":        "text",
					"value":       filter.Search,
					"oninput":     handleSearchChange,
					"placeholder": "Search todos...",
					"class":       "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				}),
			),

			// Status filter
			Div(nil,
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Status")),
				Select(Attrs{
					"value":    filter.Status,
					"onchange": handleStatusChange,
					"class":    "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				},
					Option(Attrs{"value": "all"}, Text("All")),
					Option(Attrs{"value": "active"}, Text("Active")),
					Option(Attrs{"value": "completed"}, Text("Completed")),
				),
			),

			// Category filter
			Div(nil,
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Category")),
				Input(Attrs{
					"type":        "text",
					"value":       filter.Category,
					"oninput":     handleCategoryChange,
					"placeholder": "Filter by category",
					"class":       "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				}),
			),

			// Priority filter
			Div(nil,
				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Priority")),
				Select(Attrs{
					"value":    filter.Priority,
					"onchange": handlePriorityChange,
					"class":    "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				},
					Option(Attrs{"value": ""}, Text("All Priorities")),
					Option(Attrs{"value": "high"}, Text("High")),
					Option(Attrs{"value": "medium"}, Text("Medium")),
					Option(Attrs{"value": "low"}, Text("Low")),
				),
			),
		),
	)
}

// Statistics component
func TodoStatsDisplay(props Attrs) *Element {
	stats := props["stats"].(TodoStats)

	return Div(Attrs{"class": "bg-white p-6 rounded-lg shadow-md mb-6"},
		H3(Attrs{"class": "text-lg font-semibold mb-4"}, Text("Statistics")),

		Div(Attrs{"class": "grid grid-cols-2 md:grid-cols-3 gap-4"},
			// Total
			Div(Attrs{"class": "text-center"},
				Div(Attrs{"class": "text-2xl font-bold text-blue-600"}, Text(strconv.Itoa(stats.Total))),
				Div(Attrs{"class": "text-sm text-gray-600"}, Text("Total")),
			),

			// Active
			Div(Attrs{"class": "text-center"},
				Div(Attrs{"class": "text-2xl font-bold text-orange-600"}, Text(strconv.Itoa(stats.Active))),
				Div(Attrs{"class": "text-sm text-gray-600"}, Text("Active")),
			),

			// Completed
			Div(Attrs{"class": "text-center"},
				Div(Attrs{"class": "text-2xl font-bold text-green-600"}, Text(strconv.Itoa(stats.Completed))),
				Div(Attrs{"class": "text-sm text-gray-600"}, Text("Completed")),
			),
		),

		// Progress bar
		func() *Element {
			if stats.Total == 0 {
				return Div(nil)
			}
			percentage := float64(stats.Completed) / float64(stats.Total) * 100
			return Div(Attrs{"class": "mt-4"},
				Div(Attrs{"class": "flex justify-between text-sm text-gray-600 mb-1"},
					Text("Progress"),
					Text(fmt.Sprintf("%.1f%%", percentage)),
				),
				Div(Attrs{"class": "w-full bg-gray-200 rounded-full h-2"},
					Div(Attrs{
						"class": "bg-green-600 h-2 rounded-full transition-all duration-300",
						"style": fmt.Sprintf("width: %.1f%%", percentage),
					}),
				),
			)
		}(),
	)
}

// Bulk actions component
func BulkActions(props Attrs) *Element {
	onMarkAllComplete := props["onMarkAllComplete"]
	onDeleteCompleted := props["onDeleteCompleted"]
	onDeleteAll := props["onDeleteAll"]

	handleMarkAllComplete := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if onMarkAllComplete != nil {
			onMarkAllComplete.(func())()
		}
	})

	handleDeleteCompleted := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if onDeleteCompleted != nil {
			onDeleteCompleted.(func())()
		}
	})

	handleDeleteAll := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if confirm("Are you sure you want to delete all todos?") {
			if onDeleteAll != nil {
				onDeleteAll.(func())()
			}
		}
	})

	return Div(Attrs{"class": "bg-white p-4 rounded-lg shadow-md mb-6"},
		H3(Attrs{"class": "text-lg font-semibold mb-4"}, Text("Bulk Actions")),

		Div(Attrs{"class": "flex flex-wrap gap-2"},
			Button(Attrs{
				"onclick": handleMarkAllComplete,
				"class":   "px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors",
			}, Text("Mark All Complete")),

			Button(Attrs{
				"onclick": handleDeleteCompleted,
				"class":   "px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors",
			}, Text("Delete Completed")),

			Button(Attrs{
				"onclick": handleDeleteAll,
				"class":   "px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors",
			}, Text("Delete All")),
		),
	)
}

// Helper function for confirm dialog (would be replaced with a proper modal in production)
func confirm(message string) bool {
	return js.Global().Call("confirm", message).Bool()
}

// Main todo application component
func MainTodoApp(props Attrs) *Element {
	// State management using GoUseState
	todos, setTodos := GoUseState([]Todo{})
	filter, setFilter := GoUseState(TodoFilter{Status: "all"})
	nextID, setNextID := GoUseState(1)

	// Load todos from localStorage on mount using GoUseEffect
	GoUseEffect(func() {
		fmt.Println("TodoApp: Loading todos from localStorage")
		storage := js.Global().Get("localStorage")
		if !storage.IsUndefined() && !storage.IsNull() {
			todosJSON := storage.Call("getItem", "gowebcomponents-todos")
			if !todosJSON.IsNull() && !todosJSON.IsUndefined() {
				var loadedTodos []Todo
				if err := json.Unmarshal([]byte(todosJSON.String()), &loadedTodos); err == nil {
					setTodos(loadedTodos)
					// Find the next ID
					maxID := 0
					for _, todo := range loadedTodos {
						if todo.ID > maxID {
							maxID = todo.ID
						}
					}
					setNextID(maxID + 1)
					fmt.Printf("TodoApp: Loaded %d todos from localStorage\n", len(loadedTodos))
				}
			}
		}
	}, []interface{}{}) // Empty deps = run once on mount

	// Save todos to localStorage whenever todos change using GoUseEffect
	GoUseEffect(func() {
		fmt.Println("TodoApp: Saving todos to localStorage")
		storage := js.Global().Get("localStorage")
		if !storage.IsUndefined() && !storage.IsNull() {
			todosJSON, err := json.Marshal(todos())
			if err == nil {
				storage.Call("setItem", "gowebcomponents-todos", string(todosJSON))
			}
		}
	}, []interface{}{todos()}) // Runs when todos change

	// Compute filtered todos using GoUseMemo
	filteredTodos := GoUseMemo(func() interface{} {
		currentTodos := todos()
		currentFilter := filter()

		var filtered []Todo
		for _, todo := range currentTodos {
			// Status filter
			if currentFilter.Status == "active" && todo.Completed {
				continue
			}
			if currentFilter.Status == "completed" && !todo.Completed {
				continue
			}

			// Search filter
			if currentFilter.Search != "" {
				if !strings.Contains(strings.ToLower(todo.Text), strings.ToLower(currentFilter.Search)) {
					continue
				}
			}

			// Category filter
			if currentFilter.Category != "" {
				if !strings.Contains(strings.ToLower(todo.Category), strings.ToLower(currentFilter.Category)) {
					continue
				}
			}

			// Priority filter
			if currentFilter.Priority != "" && todo.Priority != currentFilter.Priority {
				continue
			}

			filtered = append(filtered, todo)
		}
		return filtered
	}, []interface{}{todos(), filter()}).([]Todo)

	// Compute statistics using GoUseMemo
	stats := GoUseMemo(func() interface{} {
		currentTodos := todos()
		totalStats := TodoStats{
			ByPriority: make(map[string]int),
			ByCategory: make(map[string]int),
		}

		for _, todo := range currentTodos {
			totalStats.Total++
			if todo.Completed {
				totalStats.Completed++
			} else {
				totalStats.Active++
			}
			totalStats.ByPriority[todo.Priority]++
			if todo.Category != "" {
				totalStats.ByCategory[todo.Category]++
			}
		}

		return totalStats
	}, []interface{}{todos()}).(TodoStats)

	// Event handlers
	handleAddTodo := func(newTodo Todo) {
		newTodo.ID = nextID()
		setTodos(append(todos(), newTodo))
		setNextID(nextID() + 1)
		fmt.Printf("TodoApp: Added todo: %s\n", newTodo.Text)
	}

	handleToggleTodo := func(id int) {
		newTodos := make([]Todo, len(todos()))
		for i, todo := range todos() {
			if todo.ID == id {
				todo.Completed = !todo.Completed
				if todo.Completed {
					now := time.Now()
					todo.CompletedAt = &now
				} else {
					todo.CompletedAt = nil
				}
			}
			newTodos[i] = todo
		}
		setTodos(newTodos)
		fmt.Printf("TodoApp: Toggled todo %d\n", id)
	}

	handleDeleteTodo := func(id int) {
		var newTodos []Todo
		for _, todo := range todos() {
			if todo.ID != id {
				newTodos = append(newTodos, todo)
			}
		}
		setTodos(newTodos)
		fmt.Printf("TodoApp: Deleted todo %d\n", id)
	}

	handleEditTodo := func(id int, newText string) {
		newTodos := make([]Todo, len(todos()))
		for i, todo := range todos() {
			if todo.ID == id {
				todo.Text = newText
			}
			newTodos[i] = todo
		}
		setTodos(newTodos)
		fmt.Printf("TodoApp: Edited todo %d\n", id)
	}

	handleFilterChange := func(newFilter TodoFilter) {
		setFilter(newFilter)
		fmt.Printf("TodoApp: Filter changed: %+v\n", newFilter)
	}

	handleMarkAllComplete := func() {
		newTodos := make([]Todo, len(todos()))
		now := time.Now()
		for i, todo := range todos() {
			if !todo.Completed {
				todo.Completed = true
				todo.CompletedAt = &now
			}
			newTodos[i] = todo
		}
		setTodos(newTodos)
		fmt.Println("TodoApp: Marked all todos as complete")
	}

	handleDeleteCompleted := func() {
		var newTodos []Todo
		for _, todo := range todos() {
			if !todo.Completed {
				newTodos = append(newTodos, todo)
			}
		}
		setTodos(newTodos)
		fmt.Println("TodoApp: Deleted all completed todos")
	}

	handleDeleteAll := func() {
		setTodos([]Todo{})
		setNextID(1)
		fmt.Println("TodoApp: Deleted all todos")
	}

	return Html(Attrs{"lang": "en"},
		Head(nil,
			Meta(Attrs{"charset": "UTF-8"}),
			Meta(Attrs{
				"name":    "viewport",
				"content": "width=device-width, initial-scale=1.0",
			}),
			Title(nil, Text("Todo App - GoWebComponents")),
			Script(Attrs{"src": "https://cdn.tailwindcss.com"}),
		),
		Body(Attrs{"class": "bg-gray-100 min-h-screen"},
			Div(Attrs{"class": "container mx-auto px-4 py-8 max-w-4xl"},
				// Header
				Header(Attrs{"class": "text-center mb-8"},
					H1(Attrs{"class": "text-4xl font-bold text-gray-900 mb-2"}, Text("📝 Todo App")),
					P(Attrs{"class": "text-gray-600"}, Text("Built with GoWebComponents • Feature-packed & Reactive")),
				),

				// Statistics
				TodoStatsDisplay(Attrs{"stats": stats}),

				// Add todo form
				TodoInput(Attrs{"onAdd": handleAddTodo}),

				// Filters
				TodoFilters(Attrs{
					"filter":         filter(),
					"onFilterChange": handleFilterChange,
				}),

				// Bulk actions
				BulkActions(Attrs{
					"onMarkAllComplete": handleMarkAllComplete,
					"onDeleteCompleted": handleDeleteCompleted,
					"onDeleteAll":       handleDeleteAll,
				}),

				// Todo list
				Div(Attrs{"class": "bg-white rounded-lg shadow-md overflow-hidden"},
					func() *Element {
						if len(filteredTodos) == 0 {
							return Div(Attrs{"class": "p-8 text-center text-gray-500"},
								P(Attrs{"class": "text-lg"}, Text("No todos found")),
								P(Attrs{"class": "text-sm"}, Text("Add a new todo or adjust your filters")),
							)
						}

						return Ul(Attrs{"class": "divide-y divide-gray-200"},
							func() []interface{} {
								var items []interface{}
								for _, todo := range filteredTodos {
									items = append(items, TodoItem(Attrs{
										"todo":     todo,
										"onToggle": handleToggleTodo,
										"onDelete": handleDeleteTodo,
										"onEdit":   handleEditTodo,
									}))
								}
								return items
							}()...,
						)
					}(),
				),

				// Footer
				Footer(Attrs{"class": "text-center mt-8 text-gray-500 text-sm"},
					P(nil, Text("Powered by GoWebComponents • Using all Go-branded hooks")),
				),
			),
		),
	)
}

// TodoApp creates a comprehensive todo application demonstrating GoWebComponents
func TodoApp() {
	fmt.Println("TodoApp: Starting to render feature-packed todo application")

	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("TodoApp: Error - No element with id 'root' found in the DOM")
		return
	}

	fmt.Println("TodoApp: Rendering todo application into the container")
	render(createElement(MainTodoApp, nil), container)
}

// Example7 creates a comprehensive todo application demonstrating GoWebComponents
func Example7() {
	fmt.Println("Example7: Starting to render simplified todo application")

	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example7: Error - No element with id 'root' found in the DOM")
		return
	}

	fmt.Println("Example7: Rendering simplified todo application into the container")
	render(createElement(SimpleTodoApp, nil), container)
}

// Simplified todo application component with fewer hooks
func SimpleTodoApp(props Attrs) *Element {
	// Single state for todos
	todos, setTodos := GoUseState([]Todo{})

	// Handle add todo
	handleAddTodo := func(text string) {
		if strings.TrimSpace(text) != "" {
			newTodo := Todo{
				ID:        len(todos()) + 1,
				Text:      strings.TrimSpace(text),
				Completed: false,
				Priority:  "medium",
				CreatedAt: time.Now(),
			}
			setTodos(append(todos(), newTodo))
		}
	}

	// Handle toggle todo
	handleToggleTodo := func(id int) {
		newTodos := make([]Todo, len(todos()))
		for i, todo := range todos() {
			if todo.ID == id {
				todo.Completed = !todo.Completed
			}
			newTodos[i] = todo
		}
		setTodos(newTodos)
	}

	// Handle delete todo
	handleDeleteTodo := func(id int) {
		var newTodos []Todo
		for _, todo := range todos() {
			if todo.ID != id {
				newTodos = append(newTodos, todo)
			}
		}
		setTodos(newTodos)
	}

	return Html(Attrs{"lang": "en"},
		Head(nil,
			Meta(Attrs{"charset": "UTF-8"}),
			Meta(Attrs{
				"name":    "viewport",
				"content": "width=device-width, initial-scale=1.0",
			}),
			Title(nil, Text("Simple Todo App - GoWebComponents")),
			Script(Attrs{"src": "https://cdn.tailwindcss.com"}),
		),
		Body(Attrs{"class": "bg-gray-100 min-h-screen p-8"},
			Div(Attrs{"class": "max-w-2xl mx-auto"},
				// Header
				H1(Attrs{"class": "text-3xl font-bold text-center mb-8"}, Text("📝 Simple Todo App")),

				// Add todo form
				SimpleTodoInput(Attrs{"onAdd": handleAddTodo}),

				// Todo list
				Div(Attrs{"class": "bg-white rounded-lg shadow-md"},
					func() *Element {
						if len(todos()) == 0 {
							return Div(Attrs{"class": "p-8 text-center text-gray-500"},
								P(nil, Text("No todos yet. Add one above!")),
							)
						}

						var items []interface{}
						for _, todo := range todos() {
							items = append(items, SimpleTodoItem(Attrs{
								"todo":     todo,
								"onToggle": handleToggleTodo,
								"onDelete": handleDeleteTodo,
							}))
						}

						return Ul(Attrs{"class": "divide-y divide-gray-200"}, items...)
					}(),
				),

				// Footer
				P(Attrs{"class": "text-center mt-8 text-gray-500 text-sm"},
					Text("Built with GoWebComponents")),
			),
		),
	)
}

// Simple todo input component
func SimpleTodoInput(props Attrs) *Element {
	onAdd := props["onAdd"]
	text, setText := GoUseState("")

	handleSubmit := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if onAdd != nil && strings.TrimSpace(text()) != "" {
			onAdd.(func(string))(text())
			setText("")
		}
	})

	handleChange := GoUseFunc(func(event GoEvent) {
		setText(event.GetValue())
	})

	return Form(Attrs{
		"onsubmit": handleSubmit,
		"class":    "bg-white p-6 rounded-lg shadow-md mb-6",
	},
		Div(Attrs{"class": "flex gap-4"},
			Input(Attrs{
				"type":        "text",
				"value":       text(),
				"oninput":     handleChange,
				"placeholder": "What needs to be done?",
				"class":       "flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
				"required":    true,
			}),
			Button(Attrs{
				"type":  "submit",
				"class": "bg-blue-600 text-white px-6 py-2 rounded-lg font-medium hover:bg-blue-700 transition-colors",
			}, Text("Add Todo")),
		),
	)
}

// Simple todo item component
func SimpleTodoItem(props Attrs) *Element {
	todo := props["todo"].(Todo)
	onToggle := props["onToggle"]
	onDelete := props["onDelete"]

	handleToggle := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if onToggle != nil {
			onToggle.(func(int))(todo.ID)
		}
	})

	handleDelete := GoUseFunc(func(event GoEvent) {
		event.PreventDefault()
		if onDelete != nil {
			onDelete.(func(int))(todo.ID)
		}
	})

	textClass := "flex-1 ml-3"
	if todo.Completed {
		textClass += " line-through text-gray-500"
	}

	return Li(Attrs{"class": "flex items-center p-4 hover:bg-gray-50"},
		Input(Attrs{
			"type":     "checkbox",
			"checked":  todo.Completed,
			"onchange": handleToggle,
			"class":    "h-5 w-5 text-blue-600 focus:ring-blue-500 border-gray-300 rounded",
		}),

		Div(Attrs{"class": textClass},
			P(Attrs{"class": "font-medium"}, Text(todo.Text)),
		),

		Button(Attrs{
			"onclick": handleDelete,
			"class":   "p-2 text-red-500 hover:text-red-700 transition-colors",
			"title":   "Delete todo",
		}, Text("🗑️")),
	)
}
