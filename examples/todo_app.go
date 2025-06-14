package examples

// import (
// 	"encoding/json"
// 	"fmt"
// 	"strconv"
// 	"strings"
// 	"syscall/js"
// 	"time"
// )

// // Performance Optimizations Applied:
// // 1. Fixed metrics tracking to distinguish renders vs mounts
// // 2. Implemented 1000ms debounced input to reduce unnecessary parent re-renders
// // 3. Immediate UI updates with delayed state propagation for responsiveness
// // 4. Reduced logging frequency to minimize console noise (every 5th character)
// // 5. Simplified hook usage to prevent index out of bounds errors
// // 6. Timer cleanup on form submission to prevent memory leaks

// // Todo represents a single todo item
// type Todo struct {
// 	ID          int        `json:"id"`
// 	Text        string     `json:"text"`
// 	Completed   bool       `json:"completed"`
// 	Priority    string     `json:"priority"` // "low", "medium", "high"
// 	Category    string     `json:"category"`
// 	DueDate     string     `json:"dueDate"` // ISO date string
// 	CreatedAt   time.Time  `json:"createdAt"`
// 	CompletedAt *time.Time `json:"completedAt,omitempty"`
// }

// // TodoFilter represents filter criteria
// type TodoFilter struct {
// 	Status   string // "all", "active", "completed"
// 	Category string // "" for all, specific category name
// 	Priority string // "" for all, specific priority
// 	Search   string // text search
// }

// // TodoStats represents todo statistics
// type TodoStats struct {
// 	Total      int
// 	Active     int
// 	Completed  int
// 	ByPriority map[string]int
// 	ByCategory map[string]int
// }

// // Global metrics tracking
// var appMetrics struct {
// 	TotalRenders    int
// 	HookUpdates     int
// 	ComponentMounts int
// 	FiberTime       float64 // in milliseconds
// 	LastRenderTime  time.Time
// }

// // Helper function to log metrics
// func logMetrics(operation string) {
// 	fmt.Printf("🔍 Metrics [%s]: Renders=%d, HookUpdates=%d, Mounts=%d, FiberTime=%.2fms\n",
// 		operation, appMetrics.TotalRenders, appMetrics.HookUpdates, appMetrics.ComponentMounts, appMetrics.FiberTime)
// }

// // Metrics panel component (stateless to avoid hook conflicts)
// func MetricsPanel(props Attrs) *Element {
// 	// Calculate derived metrics without hooks
// 	renderRate := float64(appMetrics.TotalRenders)
// 	if !appMetrics.LastRenderTime.IsZero() {
// 		elapsed := time.Since(appMetrics.LastRenderTime).Seconds()
// 		if elapsed > 0 {
// 			renderRate = float64(appMetrics.TotalRenders) / elapsed
// 		}
// 	}

// 	uptime := time.Since(appMetrics.LastRenderTime).Truncate(time.Second)
// 	if appMetrics.LastRenderTime.IsZero() {
// 		uptime = 0
// 	}

// 	// Determine position based on props
// 	var positionStyle string
// 	if props != nil && props["position"] == "bottom" {
// 		positionStyle = "position: fixed; bottom: 80px; left: 50%; transform: translateX(-50%); min-width: 320px; max-width: 90vw; background: rgba(255, 255, 255, 0.1); backdrop-filter: blur(10px); border: 1px solid rgba(255, 255, 255, 0.3); border-radius: 12px; padding: 16px; z-index: 50; animation: slideUp 0.3s ease-out;"
// 	} else {
// 		positionStyle = "position: fixed; top: 16px; right: 16px; min-width: 280px; background: rgba(255, 255, 255, 0.1); backdrop-filter: blur(10px); border: 1px solid rgba(255, 255, 255, 0.3); border-radius: 12px; padding: 16px; z-index: 50;"
// 	}

// 	return Div(Attrs{
// 		"class": "metrics-panel",
// 		"style": positionStyle,
// 	},
// 		// Header
// 		Div(Attrs{
// 			"style": "display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px;",
// 		},
// 			H3(Attrs{
// 				"style": "font-size: 18px; font-weight: 600; color: white; margin: 0;",
// 			}, Text("⚡ Fiber Metrics")),
// 			Div(Attrs{
// 				"style": "width: 8px; height: 8px; background: #4ade80; border-radius: 50%; animation: pulse 2s infinite;",
// 			}),
// 		),

// 		// Metrics grid
// 		Div(Attrs{
// 			"style": "display: grid; grid-template-columns: 1fr 1fr; gap: 12px;",
// 		},
// 			// Total Renders
// 			Div(Attrs{
// 				"style": "background: rgba(255, 255, 255, 0.1); border-radius: 8px; padding: 12px; text-align: center;",
// 			},
// 				Div(Attrs{
// 					"style": "font-size: 24px; font-weight: bold; color: #60a5fa;",
// 				}, Text(fmt.Sprintf("%d", appMetrics.TotalRenders))),
// 				Div(Attrs{
// 					"style": "font-size: 12px; color: #d1d5db;",
// 				}, Text("Renders")),
// 			),

// 			// Hook Updates
// 			Div(Attrs{
// 				"style": "background: rgba(255, 255, 255, 0.1); border-radius: 8px; padding: 12px; text-align: center;",
// 			},
// 				Div(Attrs{
// 					"style": "font-size: 24px; font-weight: bold; color: #c084fc;",
// 				}, Text(fmt.Sprintf("%d", appMetrics.HookUpdates))),
// 				Div(Attrs{
// 					"style": "font-size: 12px; color: #d1d5db;",
// 				}, Text("Hook Updates")),
// 			),

// 			// Component Mounts
// 			Div(Attrs{
// 				"style": "background: rgba(255, 255, 255, 0.1); border-radius: 8px; padding: 12px; text-align: center;",
// 			},
// 				Div(Attrs{
// 					"style": "font-size: 24px; font-weight: bold; color: #4ade80;",
// 				}, Text(fmt.Sprintf("%d", appMetrics.ComponentMounts))),
// 				Div(Attrs{
// 					"style": "font-size: 12px; color: #d1d5db;",
// 				}, Text("Mounts")),
// 			),

// 			// Fiber Time
// 			Div(Attrs{
// 				"style": "background: rgba(255, 255, 255, 0.1); border-radius: 8px; padding: 12px; text-align: center;",
// 			},
// 				Div(Attrs{
// 					"style": "font-size: 24px; font-weight: bold; color: #fb923c;",
// 				}, Text(fmt.Sprintf("%.1f", appMetrics.FiberTime))),
// 				Div(Attrs{
// 					"style": "font-size: 12px; color: #d1d5db;",
// 				}, Text("Fiber ms")),
// 			),
// 		),

// 		// Additional info
// 		Div(Attrs{
// 			"style": "margin-top: 16px; padding-top: 12px; border-top: 1px solid rgba(255, 255, 255, 0.2);",
// 		},
// 			Div(Attrs{
// 				"style": "font-size: 12px; color: #d1d5db; text-align: center;",
// 			}, Text(fmt.Sprintf("Render Rate: %.1f/sec", renderRate))),
// 			Div(Attrs{
// 				"style": "font-size: 12px; color: #9ca3af; text-align: center; margin-top: 4px;",
// 			}, Text(fmt.Sprintf("Uptime: %v", uptime))),
// 		),
// 	)
// }

// // Enhanced state setter that tracks metrics
// func createTrackedGoUseState[T any](initialValue T) (func() T, func(T)) {
// 	getter, setter := GoUseState(initialValue)

// 	trackedSetter := func(newValue T) {
// 		appMetrics.HookUpdates++
// 		logMetrics("HOOK_UPDATE")
// 		setter(newValue)
// 	}

// 	return getter, trackedSetter
// }

// // Optimized state setter that reduces re-render frequency
// func createOptimizedGoUseState[T any](initialValue T) (func() T, func(T)) {
// 	getter, setter := GoUseState(initialValue)

// 	optimizedSetter := func(newValue T) {
// 		// Update state normally but track metrics more efficiently
// 		setter(newValue)
// 		appMetrics.HookUpdates++
// 		// Only log every few updates to reduce noise
// 		if appMetrics.HookUpdates%3 == 0 {
// 			logMetrics("OPTIMIZED_UPDATE")
// 		}
// 	}

// 	return getter, optimizedSetter
// }

// // Priority badge component
// func PriorityBadge(props Attrs) *Element {
// 	priority := "medium"
// 	if props != nil && props["priority"] != nil {
// 		priority = props["priority"].(string)
// 	}

// 	var colorClass string
// 	switch priority {
// 	case "high":
// 		colorClass = "bg-red-100 text-red-800"
// 	case "medium":
// 		colorClass = "bg-yellow-100 text-yellow-800"
// 	case "low":
// 		colorClass = "bg-green-100 text-green-800"
// 	default:
// 		colorClass = "bg-gray-100 text-gray-800"
// 	}

// 	return Span(Attrs{
// 		"class": fmt.Sprintf("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium %s", colorClass),
// 	}, Text(strings.Title(priority)))
// }

// // Category badge component
// func CategoryBadge(props Attrs) *Element {
// 	category := ""
// 	if props != nil && props["category"] != nil {
// 		category = props["category"].(string)
// 	}

// 	if category == "" {
// 		return Span(nil)
// 	}

// 	return Span(Attrs{
// 		"class": "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 ml-2",
// 	}, Text(category))
// }

// // Due date component
// func DueDateDisplay(props Attrs) *Element {
// 	dueDate := ""
// 	if props != nil && props["dueDate"] != nil {
// 		dueDate = props["dueDate"].(string)
// 	}

// 	if dueDate == "" {
// 		return Span(nil)
// 	}

// 	// Parse date and check if overdue
// 	isOverdue := false
// 	displayDate := dueDate
// 	if parsedDate, err := time.Parse("2006-01-02", dueDate); err == nil {
// 		if parsedDate.Before(time.Now()) {
// 			isOverdue = true
// 		}
// 		displayDate = parsedDate.Format("Jan 2, 2006")
// 	}

// 	colorClass := "text-gray-600"
// 	if isOverdue {
// 		colorClass = "text-red-600"
// 	}

// 	return Span(Attrs{
// 		"class": fmt.Sprintf("text-xs %s ml-2", colorClass),
// 		"title": "Due date",
// 	}, Text(fmt.Sprintf("📅 %s", displayDate)))
// }

// // Individual todo item component
// func TodoItem(props Attrs) *Element {
// 	todo := props["todo"].(Todo)
// 	onToggle := props["onToggle"]
// 	onDelete := props["onDelete"]
// 	onEdit := props["onEdit"]

// 	// Local state for editing
// 	isEditing, setIsEditing := GoUseState(false)
// 	editText, setEditText := GoUseState(todo.Text)

// 	// Handle toggle completion
// 	handleToggle := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		if onToggle != nil {
// 			onToggle.(func(int))(todo.ID)
// 		}
// 	})

// 	// Handle delete
// 	handleDelete := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		if onDelete != nil {
// 			onDelete.(func(int))(todo.ID)
// 		}
// 	})

// 	// Handle edit start
// 	handleEditStart := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		setIsEditing(true)
// 		setEditText(todo.Text)
// 	})

// 	// Handle edit save
// 	handleEditSave := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		if strings.TrimSpace(editText()) != "" {
// 			if onEdit != nil {
// 				onEdit.(func(int, string))(todo.ID, editText())
// 			}
// 		}
// 		setIsEditing(false)
// 	})

// 	// Handle edit cancel
// 	handleEditCancel := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		setIsEditing(false)
// 		setEditText(todo.Text)
// 	})

// 	// Handle input change
// 	handleInputChange := GoUseFunc(func(event GoEvent) {
// 		setEditText(event.GetValue())
// 	})

// 	// Handle key press
// 	handleKeyPress := GoUseFunc(func(event GoEvent) {
// 		if event.GetKey() == "Enter" {
// 			handleEditSave.Call("call", nil, event.Raw())
// 		} else if event.GetKey() == "Escape" {
// 			handleEditCancel.Call("call", nil, event.Raw())
// 		}
// 	})

// 	// Determine styling based on completion
// 	itemClass := "flex items-center p-4 border-b border-gray-200 hover:bg-gray-50 transition-colors"
// 	textClass := "flex-1 ml-3"
// 	if todo.Completed {
// 		textClass += " line-through text-gray-500"
// 	}

// 	return Li(Attrs{"class": itemClass},
// 		// Checkbox
// 		Input(Attrs{
// 			"type":     "checkbox",
// 			"checked":  todo.Completed,
// 			"onchange": handleToggle,
// 			"class":    "h-5 w-5 text-blue-600 focus:ring-blue-500 border-gray-300 rounded",
// 		}),

// 		// Todo content
// 		Div(Attrs{"class": textClass},
// 			func() *Element {
// 				if isEditing() {
// 					return Div(Attrs{"class": "flex items-center gap-2"},
// 						Input(Attrs{
// 							"type":        "text",
// 							"value":       editText(),
// 							"oninput":     handleInputChange,
// 							"onkeydown":   handleKeyPress,
// 							"class":       "flex-1 px-2 py-1 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500",
// 							"placeholder": "Enter todo text...",
// 						}),
// 						Button(Attrs{
// 							"onclick": handleEditSave,
// 							"class":   "px-3 py-1 bg-green-500 text-white rounded text-sm hover:bg-green-600",
// 						}, Text("Save")),
// 						Button(Attrs{
// 							"onclick": handleEditCancel,
// 							"class":   "px-3 py-1 bg-gray-500 text-white rounded text-sm hover:bg-gray-600",
// 						}, Text("Cancel")),
// 					)
// 				}
// 				return Div(nil,
// 					P(Attrs{"class": "font-medium"}, Text(todo.Text)),
// 					Div(Attrs{"class": "flex items-center mt-1"},
// 						PriorityBadge(Attrs{"priority": todo.Priority}),
// 						CategoryBadge(Attrs{"category": todo.Category}),
// 						DueDateDisplay(Attrs{"dueDate": todo.DueDate}),
// 					),
// 				)
// 			}(),
// 		),

// 		// Action buttons
// 		Div(Attrs{"class": "flex items-center gap-2 ml-2"},
// 			Button(Attrs{
// 				"onclick": handleEditStart,
// 				"class":   "p-1 text-gray-400 hover:text-blue-600 transition-colors",
// 				"title":   "Edit todo",
// 			}, Text("✏️")),
// 			Button(Attrs{
// 				"onclick": handleDelete,
// 				"class":   "p-1 text-gray-400 hover:text-red-600 transition-colors",
// 				"title":   "Delete todo",
// 			}, Text("🗑️")),
// 		),
// 	)
// }

// // Todo input form component
// func TodoInput(props Attrs) *Element {
// 	onAdd := props["onAdd"]

// 	// Form state
// 	text, setText := GoUseState("")
// 	priority, setPriority := GoUseState("medium")
// 	category, setCategory := GoUseState("")
// 	dueDate, setDueDate := GoUseState("")

// 	// Handle form submission
// 	handleSubmit := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		trimmedText := strings.TrimSpace(text())
// 		if trimmedText != "" && onAdd != nil {
// 			newTodo := Todo{
// 				Text:      trimmedText,
// 				Priority:  priority(),
// 				Category:  category(),
// 				DueDate:   dueDate(),
// 				Completed: false,
// 				CreatedAt: time.Now(),
// 			}
// 			onAdd.(func(Todo))(newTodo)
// 			// Reset form
// 			setText("")
// 			setPriority("medium")
// 			setCategory("")
// 			setDueDate("")
// 		}
// 	})

// 	// Handle input changes
// 	handleTextChange := GoUseFunc(func(event GoEvent) {
// 		setText(event.GetValue())
// 	})

// 	handlePriorityChange := GoUseFunc(func(event GoEvent) {
// 		setPriority(event.GetValue())
// 	})

// 	handleCategoryChange := GoUseFunc(func(event GoEvent) {
// 		setCategory(event.GetValue())
// 	})

// 	handleDueDateChange := GoUseFunc(func(event GoEvent) {
// 		setDueDate(event.GetValue())
// 	})

// 	return Form(Attrs{
// 		"onsubmit": handleSubmit,
// 		"class":    "bg-white p-6 rounded-lg shadow-md mb-6",
// 	},
// 		H3(Attrs{"class": "text-lg font-semibold mb-4"}, Text("Add New Todo")),

// 		Div(Attrs{"class": "grid grid-cols-1 md:grid-cols-2 gap-4 mb-4"},
// 			// Text input
// 			Div(Attrs{"class": "md:col-span-2"},
// 				Input(Attrs{
// 					"type":        "text",
// 					"value":       text(),
// 					"oninput":     handleTextChange,
// 					"placeholder": "What needs to be done?",
// 					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent",
// 					"required":    true,
// 				}),
// 			),

// 			// Priority select
// 			Div(nil,
// 				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Priority")),
// 				Select(Attrs{
// 					"value":    priority(),
// 					"onchange": handlePriorityChange,
// 					"class":    "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
// 				},
// 					Option(Attrs{"value": "low"}, Text("Low")),
// 					Option(Attrs{"value": "medium", "selected": priority() == "medium"}, Text("Medium")),
// 					Option(Attrs{"value": "high"}, Text("High")),
// 				),
// 			),

// 			// Category input
// 			Div(nil,
// 				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Category")),
// 				Input(Attrs{
// 					"type":        "text",
// 					"value":       category(),
// 					"oninput":     handleCategoryChange,
// 					"placeholder": "e.g., Work, Personal",
// 					"class":       "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
// 				}),
// 			),

// 			// Due date
// 			Div(nil,
// 				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Due Date")),
// 				Input(Attrs{
// 					"type":     "date",
// 					"value":    dueDate(),
// 					"onchange": handleDueDateChange,
// 					"class":    "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
// 				}),
// 			),
// 		),

// 		Button(Attrs{
// 			"type":  "submit",
// 			"class": "w-full bg-blue-600 text-white py-2 px-4 rounded-lg font-medium hover:bg-blue-700 transition-colors",
// 		}, Text("Add Todo")),
// 	)
// }

// // Filter controls component
// func TodoFilters(props Attrs) *Element {
// 	filter := props["filter"].(TodoFilter)
// 	onFilterChange := props["onFilterChange"]

// 	// Handle filter changes
// 	handleStatusChange := GoUseFunc(func(event GoEvent) {
// 		newFilter := filter
// 		newFilter.Status = event.GetValue()
// 		if onFilterChange != nil {
// 			onFilterChange.(func(TodoFilter))(newFilter)
// 		}
// 	})

// 	handleSearchChange := GoUseFunc(func(event GoEvent) {
// 		newFilter := filter
// 		newFilter.Search = event.GetValue()
// 		if onFilterChange != nil {
// 			onFilterChange.(func(TodoFilter))(newFilter)
// 		}
// 	})

// 	handleCategoryChange := GoUseFunc(func(event GoEvent) {
// 		newFilter := filter
// 		newFilter.Category = event.GetValue()
// 		if onFilterChange != nil {
// 			onFilterChange.(func(TodoFilter))(newFilter)
// 		}
// 	})

// 	handlePriorityChange := GoUseFunc(func(event GoEvent) {
// 		newFilter := filter
// 		newFilter.Priority = event.GetValue()
// 		if onFilterChange != nil {
// 			onFilterChange.(func(TodoFilter))(newFilter)
// 		}
// 	})

// 	return Div(Attrs{"class": "bg-white p-4 rounded-lg shadow-md mb-6"},
// 		H3(Attrs{"class": "text-lg font-semibold mb-4"}, Text("Filters")),

// 		Div(Attrs{"class": "grid grid-cols-1 md:grid-cols-4 gap-4"},
// 			// Search
// 			Div(nil,
// 				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Search")),
// 				Input(Attrs{
// 					"type":        "text",
// 					"value":       filter.Search,
// 					"oninput":     handleSearchChange,
// 					"placeholder": "Search todos...",
// 					"class":       "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
// 				}),
// 			),

// 			// Status filter
// 			Div(nil,
// 				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Status")),
// 				Select(Attrs{
// 					"value":    filter.Status,
// 					"onchange": handleStatusChange,
// 					"class":    "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
// 				},
// 					Option(Attrs{"value": "all"}, Text("All")),
// 					Option(Attrs{"value": "active"}, Text("Active")),
// 					Option(Attrs{"value": "completed"}, Text("Completed")),
// 				),
// 			),

// 			// Category filter
// 			Div(nil,
// 				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Category")),
// 				Input(Attrs{
// 					"type":        "text",
// 					"value":       filter.Category,
// 					"oninput":     handleCategoryChange,
// 					"placeholder": "Filter by category",
// 					"class":       "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
// 				}),
// 			),

// 			// Priority filter
// 			Div(nil,
// 				Label(Attrs{"class": "block text-sm font-medium text-gray-700 mb-1"}, Text("Priority")),
// 				Select(Attrs{
// 					"value":    filter.Priority,
// 					"onchange": handlePriorityChange,
// 					"class":    "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
// 				},
// 					Option(Attrs{"value": ""}, Text("All Priorities")),
// 					Option(Attrs{"value": "high"}, Text("High")),
// 					Option(Attrs{"value": "medium"}, Text("Medium")),
// 					Option(Attrs{"value": "low"}, Text("Low")),
// 				),
// 			),
// 		),
// 	)
// }

// // Statistics component
// func TodoStatsDisplay(props Attrs) *Element {
// 	stats := props["stats"].(TodoStats)

// 	return Div(Attrs{"class": "bg-white p-6 rounded-lg shadow-md mb-6"},
// 		H3(Attrs{"class": "text-lg font-semibold mb-4"}, Text("Statistics")),

// 		Div(Attrs{"class": "grid grid-cols-2 md:grid-cols-3 gap-4"},
// 			// Total
// 			Div(Attrs{"class": "text-center"},
// 				Div(Attrs{"class": "text-2xl font-bold text-blue-600"}, Text(strconv.Itoa(stats.Total))),
// 				Div(Attrs{"class": "text-sm text-gray-600"}, Text("Total")),
// 			),

// 			// Active
// 			Div(Attrs{"class": "text-center"},
// 				Div(Attrs{"class": "text-2xl font-bold text-orange-600"}, Text(strconv.Itoa(stats.Active))),
// 				Div(Attrs{"class": "text-sm text-gray-600"}, Text("Active")),
// 			),

// 			// Completed
// 			Div(Attrs{"class": "text-center"},
// 				Div(Attrs{"class": "text-2xl font-bold text-green-600"}, Text(strconv.Itoa(stats.Completed))),
// 				Div(Attrs{"class": "text-sm text-gray-600"}, Text("Completed")),
// 			),
// 		),

// 		// Progress bar
// 		func() *Element {
// 			if stats.Total == 0 {
// 				return Div(nil)
// 			}
// 			percentage := float64(stats.Completed) / float64(stats.Total) * 100
// 			return Div(Attrs{"class": "mt-4"},
// 				Div(Attrs{"class": "flex justify-between text-sm text-gray-600 mb-1"},
// 					Text("Progress"),
// 					Text(fmt.Sprintf("%.1f%%", percentage)),
// 				),
// 				Div(Attrs{"class": "w-full bg-gray-200 rounded-full h-2"},
// 					Div(Attrs{
// 						"class": "bg-green-600 h-2 rounded-full transition-all duration-300",
// 						"style": fmt.Sprintf("width: %.1f%%", percentage),
// 					}),
// 				),
// 			)
// 		}(),
// 	)
// }

// // Bulk actions component
// func BulkActions(props Attrs) *Element {
// 	onMarkAllComplete := props["onMarkAllComplete"]
// 	onDeleteCompleted := props["onDeleteCompleted"]
// 	onDeleteAll := props["onDeleteAll"]

// 	handleMarkAllComplete := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		if onMarkAllComplete != nil {
// 			onMarkAllComplete.(func())()
// 		}
// 	})

// 	handleDeleteCompleted := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		if onDeleteCompleted != nil {
// 			onDeleteCompleted.(func())()
// 		}
// 	})

// 	handleDeleteAll := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		if confirm("Are you sure you want to delete all todos?") {
// 			if onDeleteAll != nil {
// 				onDeleteAll.(func())()
// 			}
// 		}
// 	})

// 	return Div(Attrs{"class": "bg-white p-4 rounded-lg shadow-md mb-6"},
// 		H3(Attrs{"class": "text-lg font-semibold mb-4"}, Text("Bulk Actions")),

// 		Div(Attrs{"class": "flex flex-wrap gap-2"},
// 			Button(Attrs{
// 				"onclick": handleMarkAllComplete,
// 				"class":   "px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors",
// 			}, Text("Mark All Complete")),

// 			Button(Attrs{
// 				"onclick": handleDeleteCompleted,
// 				"class":   "px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors",
// 			}, Text("Delete Completed")),

// 			Button(Attrs{
// 				"onclick": handleDeleteAll,
// 				"class":   "px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors",
// 			}, Text("Delete All")),
// 		),
// 	)
// }

// // Helper function for confirm dialog (would be replaced with a proper modal in production)
// func confirm(message string) bool {
// 	return js.Global().Call("confirm", message).Bool()
// }

// // Main todo application component
// func MainTodoApp(props Attrs) *Element {
// 	// State management using GoUseState
// 	todos, setTodos := GoUseState([]Todo{})
// 	filter, setFilter := GoUseState(TodoFilter{Status: "all"})
// 	nextID, setNextID := GoUseState(1)

// 	// Load todos from localStorage on mount using GoUseEffect
// 	GoUseEffect(func() {
// 		fmt.Println("TodoApp: Loading todos from localStorage")
// 		storage := js.Global().Get("localStorage")
// 		if !storage.IsUndefined() && !storage.IsNull() {
// 			todosJSON := storage.Call("getItem", "gowebcomponents-todos")
// 			if !todosJSON.IsNull() && !todosJSON.IsUndefined() {
// 				var loadedTodos []Todo
// 				if err := json.Unmarshal([]byte(todosJSON.String()), &loadedTodos); err == nil {
// 					setTodos(loadedTodos)
// 					// Find the next ID
// 					maxID := 0
// 					for _, todo := range loadedTodos {
// 						if todo.ID > maxID {
// 							maxID = todo.ID
// 						}
// 					}
// 					setNextID(maxID + 1)
// 					fmt.Printf("TodoApp: Loaded %d todos from localStorage\n", len(loadedTodos))
// 				}
// 			}
// 		}
// 	}, []interface{}{}) // Empty deps = run once on mount

// 	// Save todos to localStorage whenever todos change using GoUseEffect
// 	GoUseEffect(func() {
// 		fmt.Println("TodoApp: Saving todos to localStorage")
// 		storage := js.Global().Get("localStorage")
// 		if !storage.IsUndefined() && !storage.IsNull() {
// 			todosJSON, err := json.Marshal(todos())
// 			if err == nil {
// 				storage.Call("setItem", "gowebcomponents-todos", string(todosJSON))
// 			}
// 		}
// 	}, []interface{}{todos()}) // Runs when todos change

// 	// Compute filtered todos using GoUseMemo
// 	filteredTodos := GoUseMemo(func() interface{} {
// 		currentTodos := todos()
// 		currentFilter := filter()

// 		var filtered []Todo
// 		for _, todo := range currentTodos {
// 			// Status filter
// 			if currentFilter.Status == "active" && todo.Completed {
// 				continue
// 			}
// 			if currentFilter.Status == "completed" && !todo.Completed {
// 				continue
// 			}

// 			// Search filter
// 			if currentFilter.Search != "" {
// 				if !strings.Contains(strings.ToLower(todo.Text), strings.ToLower(currentFilter.Search)) {
// 					continue
// 				}
// 			}

// 			// Category filter
// 			if currentFilter.Category != "" {
// 				if !strings.Contains(strings.ToLower(todo.Category), strings.ToLower(currentFilter.Category)) {
// 					continue
// 				}
// 			}

// 			// Priority filter
// 			if currentFilter.Priority != "" && todo.Priority != currentFilter.Priority {
// 				continue
// 			}

// 			filtered = append(filtered, todo)
// 		}
// 		return filtered
// 	}, []interface{}{todos(), filter()}).([]Todo)

// 	// Compute statistics using GoUseMemo
// 	stats := GoUseMemo(func() interface{} {
// 		currentTodos := todos()
// 		totalStats := TodoStats{
// 			ByPriority: make(map[string]int),
// 			ByCategory: make(map[string]int),
// 		}

// 		for _, todo := range currentTodos {
// 			totalStats.Total++
// 			if todo.Completed {
// 				totalStats.Completed++
// 			} else {
// 				totalStats.Active++
// 			}
// 			totalStats.ByPriority[todo.Priority]++
// 			if todo.Category != "" {
// 				totalStats.ByCategory[todo.Category]++
// 			}
// 		}

// 		return totalStats
// 	}, []interface{}{todos()}).(TodoStats)

// 	// Event handlers
// 	handleAddTodo := func(newTodo Todo) {
// 		newTodo.ID = nextID()
// 		setTodos(append(todos(), newTodo))
// 		setNextID(nextID() + 1)
// 		fmt.Printf("TodoApp: Added todo: %s\n", newTodo.Text)
// 	}

// 	handleToggleTodo := func(id int) {
// 		newTodos := make([]Todo, len(todos()))
// 		for i, todo := range todos() {
// 			if todo.ID == id {
// 				todo.Completed = !todo.Completed
// 				if todo.Completed {
// 					now := time.Now()
// 					todo.CompletedAt = &now
// 				} else {
// 					todo.CompletedAt = nil
// 				}
// 			}
// 			newTodos[i] = todo
// 		}
// 		setTodos(newTodos)
// 		fmt.Printf("TodoApp: Toggled todo %d\n", id)
// 	}

// 	handleDeleteTodo := func(id int) {
// 		var newTodos []Todo
// 		for _, todo := range todos() {
// 			if todo.ID != id {
// 				newTodos = append(newTodos, todo)
// 			}
// 		}
// 		setTodos(newTodos)
// 		fmt.Printf("TodoApp: Deleted todo %d\n", id)
// 	}

// 	handleEditTodo := func(id int, newText string) {
// 		newTodos := make([]Todo, len(todos()))
// 		for i, todo := range todos() {
// 			if todo.ID == id {
// 				todo.Text = newText
// 			}
// 			newTodos[i] = todo
// 		}
// 		setTodos(newTodos)
// 		fmt.Printf("TodoApp: Edited todo %d\n", id)
// 	}

// 	handleFilterChange := func(newFilter TodoFilter) {
// 		setFilter(newFilter)
// 		fmt.Printf("TodoApp: Filter changed: %+v\n", newFilter)
// 	}

// 	handleMarkAllComplete := func() {
// 		newTodos := make([]Todo, len(todos()))
// 		now := time.Now()
// 		for i, todo := range todos() {
// 			if !todo.Completed {
// 				todo.Completed = true
// 				todo.CompletedAt = &now
// 			}
// 			newTodos[i] = todo
// 		}
// 		setTodos(newTodos)
// 		fmt.Println("TodoApp: Marked all todos as complete")
// 	}

// 	handleDeleteCompleted := func() {
// 		var newTodos []Todo
// 		for _, todo := range todos() {
// 			if !todo.Completed {
// 				newTodos = append(newTodos, todo)
// 			}
// 		}
// 		setTodos(newTodos)
// 		fmt.Println("TodoApp: Deleted all completed todos")
// 	}

// 	handleDeleteAll := func() {
// 		setTodos([]Todo{})
// 		setNextID(1)
// 		fmt.Println("TodoApp: Deleted all todos")
// 	}

// 	return Html(Attrs{"lang": "en"},
// 		Head(nil,
// 			Meta(Attrs{"charset": "UTF-8"}),
// 			Meta(Attrs{
// 				"name":    "viewport",
// 				"content": "width=device-width, initial-scale=1.0",
// 			}),
// 			Title(nil, Text("Todo App - GoWebComponents")),
// 			Script(Attrs{"src": "https://cdn.tailwindcss.com"}),
// 		),
// 		Body(Attrs{"class": "bg-gray-100 min-h-screen"},
// 			Div(Attrs{"class": "container mx-auto px-4 py-8 max-w-4xl"},
// 				// Header
// 				Header(Attrs{"class": "text-center mb-8"},
// 					H1(Attrs{"class": "text-4xl font-bold text-gray-900 mb-2"}, Text("📝 Todo App")),
// 					P(Attrs{"class": "text-gray-600"}, Text("Built with GoWebComponents • Feature-packed & Reactive")),
// 				),

// 				// Statistics
// 				TodoStatsDisplay(Attrs{"stats": stats}),

// 				// Add todo form
// 				TodoInput(Attrs{"onAdd": handleAddTodo}),

// 				// Filters
// 				TodoFilters(Attrs{
// 					"filter":         filter(),
// 					"onFilterChange": handleFilterChange,
// 				}),

// 				// Bulk actions
// 				BulkActions(Attrs{
// 					"onMarkAllComplete": handleMarkAllComplete,
// 					"onDeleteCompleted": handleDeleteCompleted,
// 					"onDeleteAll":       handleDeleteAll,
// 				}),

// 				// Todo list
// 				Div(Attrs{"class": "bg-white rounded-lg shadow-md overflow-hidden"},
// 					func() *Element {
// 						if len(filteredTodos) == 0 {
// 							return Div(Attrs{"class": "p-8 text-center text-gray-500"},
// 								P(Attrs{"class": "text-lg"}, Text("No todos found")),
// 								P(Attrs{"class": "text-sm"}, Text("Add a new todo or adjust your filters")),
// 							)
// 						}

// 						return Ul(Attrs{"class": "divide-y divide-gray-200"},
// 							func() []interface{} {
// 								var items []interface{}
// 								for _, todo := range filteredTodos {
// 									items = append(items, TodoItem(Attrs{
// 										"todo":     todo,
// 										"onToggle": handleToggleTodo,
// 										"onDelete": handleDeleteTodo,
// 										"onEdit":   handleEditTodo,
// 									}))
// 								}
// 								return items
// 							}()...,
// 						)
// 					}(),
// 				),

// 				// Footer
// 				Footer(Attrs{"class": "text-center mt-8 text-gray-500 text-sm"},
// 					P(nil, Text("Powered by GoWebComponents • Using all Go-branded hooks")),
// 				),
// 			),
// 		),
// 	)
// }

// // TodoApp creates a comprehensive todo application demonstrating GoWebComponents
// func TodoApp() {
// 	fmt.Println("TodoApp: Starting to render feature-packed todo application")

// 	container := js.Global().Get("document").Call("getElementById", "root")
// 	if container.IsUndefined() || container.IsNull() {
// 		fmt.Println("TodoApp: Error - No element with id 'root' found in the DOM")
// 		return
// 	}

// 	fmt.Println("TodoApp: Rendering todo application into the container")
// 	render(createElement(MainTodoApp, nil), container)
// }

// // Example7 creates a comprehensive todo application demonstrating GoWebComponents
// func Example7() {
// 	fmt.Println("Example7: Starting to render simplified todo application")

// 	container := js.Global().Get("document").Call("getElementById", "root")
// 	if container.IsUndefined() || container.IsNull() {
// 		fmt.Println("Example7: Error - No element with id 'root' found in the DOM")
// 		return
// 	}

// 	fmt.Println("Example7: Rendering simplified todo application into the container")
// 	render(createElement(SimpleTodoApp, nil), container)
// }

// // Simplified todo application component with modern dark mode styling and metrics
// func SimpleTodoApp(props Attrs) *Element {
// 	// Initialize metrics timing
// 	if appMetrics.LastRenderTime.IsZero() {
// 		appMetrics.LastRenderTime = time.Now()
// 		appMetrics.ComponentMounts = 1 // Only count actual component mounts
// 		fmt.Println("🚀 TodoApp: Metrics tracking initialized")
// 	} else {
// 		// Only increment renders, not mounts
// 		appMetrics.TotalRenders++
// 	}

// 	// Track render start time
// 	renderStart := time.Now()

// 	// Single state for todos with regular state (no extra tracking needed here)
// 	todos, setTodos := GoUseState([]Todo{})

// 	// State for metrics panel visibility
// 	metricsVisible, setMetricsVisible := GoUseState(true)

// 	// Track render end time
// 	defer func() {
// 		renderTime := float64(time.Since(renderStart).Nanoseconds()) / 1000000.0 // Convert to milliseconds
// 		appMetrics.FiberTime = renderTime
// 		logMetrics("RENDER_COMPLETE")
// 	}()

// 	// Simple handlers - avoid memoization complexity that breaks hooks
// 	handleAddTodo := func(text string) {
// 		if strings.TrimSpace(text) != "" {
// 			newTodo := Todo{
// 				ID:        len(todos()) + 1,
// 				Text:      strings.TrimSpace(text),
// 				Completed: false,
// 				Priority:  "medium",
// 				CreatedAt: time.Now(),
// 			}
// 			fmt.Printf("📝 TodoApp: Adding new todo - '%s'\n", newTodo.Text)
// 			setTodos(append(todos(), newTodo))
// 			appMetrics.HookUpdates++
// 			logMetrics("ADD_TODO")
// 		}
// 	}

// 	handleToggleTodo := func(id int) {
// 		newTodos := make([]Todo, len(todos()))
// 		for i, todo := range todos() {
// 			if todo.ID == id {
// 				todo.Completed = !todo.Completed
// 				fmt.Printf("✅ TodoApp: Toggled todo %d - completed: %v\n", id, todo.Completed)
// 			}
// 			newTodos[i] = todo
// 		}
// 		setTodos(newTodos)
// 		appMetrics.HookUpdates++
// 		logMetrics("TOGGLE_TODO")
// 	}

// 	handleDeleteTodo := func(id int) {
// 		var newTodos []Todo
// 		for _, todo := range todos() {
// 			if todo.ID != id {
// 				newTodos = append(newTodos, todo)
// 			} else {
// 				fmt.Printf("🗑️ TodoApp: Deleted todo %d - '%s'\n", id, todo.Text)
// 			}
// 		}
// 		setTodos(newTodos)
// 		appMetrics.HookUpdates++
// 		logMetrics("DELETE_TODO")
// 	}

// 	handleToggleMetrics := func() {
// 		fmt.Printf("📊 TodoApp: Toggling metrics panel visibility to %v\n", !metricsVisible())
// 		setMetricsVisible(!metricsVisible())
// 		appMetrics.HookUpdates++
// 		logMetrics("TOGGLE_METRICS")
// 	}

// 	fmt.Printf("🔄 TodoApp: Rendering with %d todos\n", len(todos()))

// 	return Html(Attrs{"lang": "en"},
// 		Head(nil,
// 			Meta(Attrs{"charset": "UTF-8"}),
// 			Meta(Attrs{
// 				"name":    "viewport",
// 				"content": "width=device-width, initial-scale=1.0",
// 			}),
// 			Title(nil, Text("Modern Todo App - GoWebComponents")),
// 			Script(Attrs{"src": "https://cdn.tailwindcss.com"}),
// 			Style(nil, Text(`
// 				@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
// 				body { font-family: 'Inter', sans-serif; }
// 				.glassmorphism {
// 					background: rgba(255, 255, 255, 0.1);
// 					backdrop-filter: blur(10px);
// 					border: 1px solid rgba(255, 255, 255, 0.2);
// 				}
// 				.todo-gradient {
// 					background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
// 				}
// 				.dark-gradient {
// 					background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
// 				}
// 				.animate-fade-in {
// 					animation: fadeIn 0.3s ease-in-out;
// 				}
// 				@keyframes fadeIn {
// 					from { opacity: 0; transform: translateY(10px); }
// 					to { opacity: 1; transform: translateY(0); }
// 				}
// 				@keyframes slideUp {
// 					from { opacity: 0; transform: translateX(-50%) translateY(20px); }
// 					to { opacity: 1; transform: translateX(-50%) translateY(0); }
// 				}
// 				@keyframes pulse {
// 					0%, 100% { opacity: 1; }
// 					50% { opacity: 0.5; }
// 				}
// 				button:hover {
// 					transform: scale(1.05);
// 				}
// 			`)),
// 		),
// 		Body(Attrs{"class": "bg-gradient-to-br from-gray-900 via-purple-900 to-violet-900 min-h-screen p-4 md:p-8"},
// 			Div(Attrs{"class": "max-w-4xl mx-auto"},
// 				// Header with gradient and glassmorphism
// 				Div(Attrs{"class": "text-center mb-8 glassmorphism rounded-2xl p-8 backdrop-blur-xl"},
// 					H1(Attrs{"class": "text-4xl md:text-6xl font-bold bg-gradient-to-r from-blue-400 via-purple-400 to-pink-400 bg-clip-text text-transparent mb-4"},
// 						Text("✨ Modern Todo")),
// 					P(Attrs{"class": "text-gray-300 text-lg font-light"},
// 						Text("Beautiful task management with GoWebComponents + Metrics")),
// 				),

// 				// Add todo form with modern styling
// 				SimpleTodoInput(Attrs{"onAdd": handleAddTodo}),

// 				// Stats section - simplified without memoization
// 				func() *Element {
// 					completedCount := 0
// 					for _, todo := range todos() {
// 						if todo.Completed {
// 							completedCount++
// 						}
// 					}

// 					if len(todos()) == 0 {
// 						return Div(nil)
// 					}

// 					// Only log stats every few renders to reduce noise
// 					if len(todos())%2 == 0 {
// 						fmt.Printf("📊 TodoApp: Stats - Total: %d, Completed: %d, Remaining: %d\n",
// 							len(todos()), completedCount, len(todos())-completedCount)
// 					}

// 					return Div(Attrs{"class": "grid grid-cols-1 md:grid-cols-3 gap-4 mb-8"},
// 						// Total tasks
// 						Div(Attrs{"class": "glassmorphism rounded-xl p-6 text-center"},
// 							Div(Attrs{"class": "text-3xl font-bold text-blue-400 mb-2"},
// 								Text(fmt.Sprintf("%d", len(todos())))),
// 							Div(Attrs{"class": "text-gray-300 text-sm font-medium"},
// 								Text("Total Tasks")),
// 						),

// 						// Completed tasks
// 						Div(Attrs{"class": "glassmorphism rounded-xl p-6 text-center"},
// 							Div(Attrs{"class": "text-3xl font-bold text-green-400 mb-2"},
// 								Text(fmt.Sprintf("%d", completedCount))),
// 							Div(Attrs{"class": "text-gray-300 text-sm font-medium"},
// 								Text("Completed")),
// 						),

// 						// Remaining tasks
// 						Div(Attrs{"class": "glassmorphism rounded-xl p-6 text-center"},
// 							Div(Attrs{"class": "text-3xl font-bold text-orange-400 mb-2"},
// 								Text(fmt.Sprintf("%d", len(todos())-completedCount))),
// 							Div(Attrs{"class": "text-gray-300 text-sm font-medium"},
// 								Text("Remaining")),
// 						),
// 					)
// 				}(),

// 				// Todo list with modern glassmorphism
// 				Div(Attrs{"class": "glassmorphism rounded-2xl overflow-hidden backdrop-blur-xl"},
// 					func() *Element {
// 						if len(todos()) == 0 {
// 							return Div(Attrs{"class": "p-16 text-center"},
// 								Div(Attrs{"class": "text-6xl mb-4"}, Text("🎯")),
// 								P(Attrs{"class": "text-xl text-gray-300 mb-2"}, Text("No tasks yet")),
// 								P(Attrs{"class": "text-gray-400"}, Text("Add your first task above to get started")),
// 							)
// 						}

// 						var items []interface{}
// 						for _, todo := range todos() {
// 							items = append(items, SimpleTodoItem(Attrs{
// 								"todo":     todo,
// 								"onToggle": handleToggleTodo,
// 								"onDelete": handleDeleteTodo,
// 							}))
// 						}

// 						fmt.Printf("📋 TodoApp: Rendering %d todo items\n", len(items))

// 						return Ul(Attrs{"class": "divide-y divide-white/10"}, items...)
// 					}(),
// 				),

// 				// Footer with modern styling
// 				Footer(Attrs{"class": "text-center mt-12 glassmorphism rounded-xl p-6"},
// 					P(Attrs{"class": "text-gray-300 mb-2"},
// 						Text("Built with ❤️ using GoWebComponents")),
// 					P(Attrs{"class": "text-gray-400 text-sm"},
// 						Text("Modern • Reactive • Beautiful • Monitored")),
// 				),
// 			),

// 			// Metrics toggle button - fixed position
// 			Button(Attrs{
// 				"onclick": GoUseFunc(func(event GoEvent) {
// 					handleToggleMetrics()
// 				}),
// 				"style": "position: fixed; bottom: 20px; right: 20px; width: 50px; height: 50px; background: rgba(99, 102, 241, 0.8); color: white; border: none; border-radius: 50%; font-size: 20px; cursor: pointer; backdrop-filter: blur(10px); z-index: 60; transition: all 0.3s ease;",
// 				"title": func() string {
// 					if metricsVisible() {
// 						return "Hide Metrics Panel"
// 					}
// 					return "Show Metrics Panel"
// 				}(),
// 			}, Text(func() string {
// 				if metricsVisible() {
// 					return "📊"
// 				}
// 				return "📈"
// 			}())),

// 			// Metrics Panel - Bottom positioned with visibility toggle
// 			func() *Element {
// 				if metricsVisible() {
// 					return MetricsPanel(Attrs{"position": "bottom"})
// 				}
// 				return Div(nil) // Return empty div when hidden
// 			}(),
// 		),
// 	)
// }

// // Modern todo input component with glassmorphism styling and 1000ms debounced rendering
// func SimpleTodoInput(props Attrs) *Element {
// 	onAdd := props["onAdd"]

// 	// Local immediate state for UI responsiveness
// 	displayText, setDisplayText := GoUseState("")
// 	// Debounced state that triggers parent re-renders
// 	debouncedText, setDebouncedText := GoUseState("")
// 	// Timer reference for debouncing
// 	timerRef, setTimerRef := GoUseState(js.Null())

// 	// Reduce logging frequency to minimize console noise
// 	if len(displayText())%5 == 0 || displayText() == "" {
// 		fmt.Printf("🔤 SimpleTodoInput: Rendering with display='%s', debounced='%s'\n", displayText(), debouncedText())
// 	}

// 	// Submit handler uses current display text
// 	handleSubmit := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		inputText := strings.TrimSpace(displayText())
// 		fmt.Printf("📤 SimpleTodoInput: Form submitted with text='%s'\n", inputText)

// 		if onAdd != nil && inputText != "" {
// 			onAdd.(func(string))(inputText)
// 			setDisplayText("")
// 			setDebouncedText("")
// 			// Clear any pending timer
// 			if !timerRef().IsNull() {
// 				js.Global().Call("clearTimeout", timerRef())
// 				setTimerRef(js.Null())
// 			}
// 			appMetrics.HookUpdates++
// 			logMetrics("FORM_SUBMIT")
// 		}
// 	})

// 	// Debounced change handler (1000ms delay)
// 	handleChange := GoUseFunc(func(event GoEvent) {
// 		newValue := event.GetValue()

// 		// Update display immediately for UI responsiveness
// 		setDisplayText(newValue)

// 		// Clear existing timer
// 		if !timerRef().IsNull() {
// 			js.Global().Call("clearTimeout", timerRef())
// 		}

// 		// Set new timer for debounced update (1000ms)
// 		newTimer := js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
// 			// Only log significant changes to reduce noise
// 			if len(newValue)%5 == 0 || newValue == "" {
// 				fmt.Printf("⌨️ SimpleTodoInput: Debounced text changed to='%s'\n", newValue)
// 			}
// 			setDebouncedText(newValue)
// 			setTimerRef(js.Null())
// 			appMetrics.HookUpdates++
// 			logMetrics("DEBOUNCED_INPUT")
// 			return nil
// 		}), 1000)

// 		setTimerRef(newTimer)
// 	})

// 	return Form(Attrs{
// 		"onsubmit": handleSubmit,
// 		"class":    "glassmorphism rounded-2xl p-6 mb-8 backdrop-blur-xl animate-fade-in",
// 	},
// 		Div(Attrs{"class": "flex flex-col md:flex-row gap-4"},
// 			Input(Attrs{
// 				"type":        "text",
// 				"value":       displayText(),
// 				"oninput":     handleChange,
// 				"placeholder": "What needs to be accomplished today?",
// 				"class":       "flex-1 px-6 py-4 bg-white/10 border border-white/20 rounded-xl text-white placeholder-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-400 focus:border-transparent backdrop-blur-sm transition-all duration-200",
// 				"required":    true,
// 			}),
// 			Button(Attrs{
// 				"type":  "submit",
// 				"class": "px-8 py-4 bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold rounded-xl hover:from-blue-600 hover:to-purple-700 focus:outline-none focus:ring-2 focus:ring-blue-400 transform hover:scale-105 transition-all duration-200 shadow-lg",
// 			}, Text("✨ Add Task")),
// 		),
// 	)
// }

// // Modern todo item component with sleek dark styling and optimized rendering
// func SimpleTodoItem(props Attrs) *Element {
// 	todo := props["todo"].(Todo)
// 	onToggle := props["onToggle"]
// 	onDelete := props["onDelete"]

// 	// Only log when item actually changes, not on every app render
// 	fmt.Printf("📝 SimpleTodoItem: Rendering todo %d - '%s' (completed: %v)\n",
// 		todo.ID, todo.Text, todo.Completed)

// 	// Simple handlers without memoization
// 	handleToggle := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		fmt.Printf("🔄 SimpleTodoItem: Toggle clicked for todo %d\n", todo.ID)
// 		if onToggle != nil {
// 			onToggle.(func(int))(todo.ID)
// 			appMetrics.HookUpdates++
// 			logMetrics("ITEM_TOGGLE")
// 		}
// 	})

// 	handleDelete := GoUseFunc(func(event GoEvent) {
// 		event.PreventDefault()
// 		fmt.Printf("🗑️ SimpleTodoItem: Delete clicked for todo %d\n", todo.ID)
// 		if onDelete != nil {
// 			onDelete.(func(int))(todo.ID)
// 			appMetrics.HookUpdates++
// 			logMetrics("ITEM_DELETE")
// 		}
// 	})

// 	// Dynamic styling based on completion
// 	itemClass := "flex items-center p-6 hover:bg-white/5 transition-all duration-200 group animate-fade-in"
// 	textClass := "flex-1 ml-4 transition-all duration-200"
// 	checkboxClass := "h-5 w-5 rounded-md border-2 border-gray-400 bg-transparent checked:bg-blue-500 checked:border-blue-500 focus:ring-2 focus:ring-blue-400 transition-all duration-200"

// 	if todo.Completed {
// 		textClass += " line-through text-gray-400"
// 		checkboxClass = "h-5 w-5 rounded-md border-2 border-green-400 bg-green-500 checked:bg-green-500 checked:border-green-500 focus:ring-2 focus:ring-green-400 transition-all duration-200"
// 	} else {
// 		textClass += " text-white"
// 	}

// 	return Li(Attrs{"class": itemClass},
// 		// Custom styled checkbox
// 		Label(Attrs{"class": "flex items-center cursor-pointer"},
// 			Input(Attrs{
// 				"type":     "checkbox",
// 				"checked":  todo.Completed,
// 				"onchange": handleToggle,
// 				"class":    checkboxClass,
// 			}),

// 			Div(Attrs{"class": textClass},
// 				P(Attrs{"class": "font-medium text-lg"}, Text(todo.Text)),
// 				P(Attrs{"class": "text-sm text-gray-400 mt-1"},
// 					Text(todo.CreatedAt.Format("Jan 2, 2006 at 3:04 PM"))),
// 			),
// 		),

// 		// Modern delete button with hover effects
// 		Button(Attrs{
// 			"onclick": handleDelete,
// 			"class":   "p-3 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-xl transition-all duration-200 opacity-0 group-hover:opacity-100 transform hover:scale-110",
// 			"title":   "Delete task",
// 		}, Text("🗑️")),
// 	)
// }
