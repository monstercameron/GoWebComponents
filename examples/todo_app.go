//go:build js && wasm
// +build js,wasm

package example

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

// Type aliases for convenience.
type Attrs = dom.Attrs
type Element = render.Element

// Todo represents a single todo item.
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

// TodoFilter represents filter criteria.
type TodoFilter struct {
	Status   string // "all", "active", "completed"
	Category string // "" for all, specific category name
	Priority string // "" for all, specific priority
	Search   string // text search
}

// TodoStats represents todo statistics.
type TodoStats struct {
	Total      int
	Active     int
	Completed  int
	ByPriority map[string]int
	ByCategory map[string]int
}

// Metrics tracking.
var appMetrics struct {
	TotalRenders    int
	HookUpdates     int
	ComponentMounts int
	FiberTime       float64 // in milliseconds
	LastRenderTime  time.Time
}

func logMetrics(operation string) {
	fmt.Printf("🔍 Metrics [%s]: Renders=%d, HookUpdates=%d, Mounts=%d, FiberTime=%.2fms\n",
		operation, appMetrics.TotalRenders, appMetrics.HookUpdates, appMetrics.ComponentMounts, appMetrics.FiberTime)
}

// Priority badge component.
func PriorityBadge(props Attrs) *Element {
	priority := "medium"
	if props != nil && props["priority"] != nil {
		priority = props["priority"].(string)
	}

	var colorClass string
	switch priority {
	case "high":
		colorClass = "bg-rose-100 text-rose-700"
	case "medium":
		colorClass = "bg-amber-100 text-amber-700"
	case "low":
		colorClass = "bg-emerald-100 text-emerald-700"
	default:
		colorClass = "bg-slate-100 text-slate-700"
	}

	return dom.Span(Attrs{
		"class": fmt.Sprintf("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium %s", colorClass),
	}, dom.Text(strings.Title(priority)))
}

// Category badge component.
func CategoryBadge(props Attrs) *Element {
	category := ""
	if props != nil && props["category"] != nil {
		category = props["category"].(string)
	}

	if category == "" {
		return dom.Span(nil)
	}

	return dom.Span(Attrs{
		"class": "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-indigo-100 text-indigo-700 ml-2",
	}, dom.Text(category))
}

// Due date component.
func DueDateDisplay(props Attrs) *Element {
	dueDate := ""
	if props != nil && props["dueDate"] != nil {
		dueDate = props["dueDate"].(string)
	}

	if dueDate == "" {
		return dom.Span(nil)
	}

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

	return dom.Span(Attrs{
		"class": fmt.Sprintf("text-xs %s ml-2", colorClass),
		"title": "Due date",
	}, dom.Text(fmt.Sprintf("📅 %s", displayDate)))
}

// TodoItem renders one todo row.
func TodoItem(props Attrs) *Element {
	todo := props["todo"].(Todo)
	onToggle := props["onToggle"].(func(int))
	onDelete := props["onDelete"].(func(int))
	onEdit := props["onEdit"].(func(int, string))

	isEditing, setIsEditing := hooks.UseState(false)
	editText, setEditText := hooks.UseState(todo.Text)

	handleToggle := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		onToggle(todo.ID)
	})

	handleDelete := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		onDelete(todo.ID)
	})

	handleEditStart := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setIsEditing(true)
		setEditText(todo.Text)
	})

	handleEditSave := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		if strings.TrimSpace(editText()) != "" {
			onEdit(todo.ID, editText())
		}
		setIsEditing(false)
	})

	handleEditCancel := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setIsEditing(false)
		setEditText(todo.Text)
	})

	handleInputChange := hooks.GoUseFunc(func(event dom.GoEvent) {
		setEditText(event.GetValue())
	})

	handleKeyPress := hooks.GoUseFunc(func(event dom.GoEvent) {
		if event.GetKey() == "Enter" {
			if strings.TrimSpace(editText()) != "" {
				onEdit(todo.ID, editText())
			}
			setIsEditing(false)
		} else if event.GetKey() == "Escape" {
			setIsEditing(false)
			setEditText(todo.Text)
		}
	})

	itemClass := "flex items-center p-4 border-b border-slate-200 hover:bg-slate-50 transition-colors"
	textClass := "flex-1 ml-3 text-slate-900"
	if todo.Completed {
		textClass += " line-through text-slate-500"
	}

	return dom.Li(Attrs{"class": itemClass, "key": todo.ID},
		dom.Input(Attrs{
			"type":     "checkbox",
			"checked":  todo.Completed,
			"onchange": handleToggle,
			"class":    "h-5 w-5 text-indigo-600 focus:ring-indigo-500 border-slate-300 rounded",
		}),
		dom.Div(Attrs{"class": textClass},
			func() *Element {
				if isEditing() {
					return dom.Div(Attrs{"class": "flex items-center gap-2"},
						dom.Input(Attrs{
							"type":        "text",
							"value":       editText(),
							"oninput":     handleInputChange,
							"onkeydown":   handleKeyPress,
							"class":       "flex-1 px-2 py-1 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-indigo-500",
							"placeholder": "Enter todo text...",
						}),
						dom.Button(Attrs{
							"onclick": handleEditSave,
							"class":   "px-3 py-1 bg-indigo-600 text-white rounded text-sm hover:bg-indigo-700",
						}, dom.Text("Save")),
						dom.Button(Attrs{
							"onclick": handleEditCancel,
							"class":   "px-3 py-1 bg-slate-500 text-white rounded text-sm hover:bg-slate-600",
						}, dom.Text("Cancel")),
					)
				}
				return dom.Div(nil,
					dom.P(Attrs{"class": "font-medium"}, dom.Text(todo.Text)),
					dom.Div(Attrs{"class": "flex items-center mt-1"},
						PriorityBadge(Attrs{"priority": todo.Priority}),
						CategoryBadge(Attrs{"category": todo.Category}),
						DueDateDisplay(Attrs{"dueDate": todo.DueDate}),
					),
				)
			}(),
		),
		dom.Div(Attrs{"class": "flex items-center gap-2 ml-2"},
			dom.Button(Attrs{
				"onclick": handleEditStart,
				"class":   "p-1 text-gray-400 hover:text-blue-600 transition-colors",
				"title":   "Edit todo",
			}, dom.Text("✏️")),
			dom.Button(Attrs{
				"onclick": handleDelete,
				"class":   "p-1 text-gray-400 hover:text-red-600 transition-colors",
				"title":   "Delete todo",
			}, dom.Text("🗑️")),
		),
	)
}

// TodoInput renders the form for adding todos.
func TodoInput(props Attrs) *Element {
	onAdd := props["onAdd"].(func(Todo))

	text, setText := hooks.UseState("")
	priority, setPriority := hooks.UseState("medium")
	category, setCategory := hooks.UseState("")
	dueDate, setDueDate := hooks.UseState("")

	handleSubmit := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		trimmed := strings.TrimSpace(text())
		if trimmed == "" {
			return
		}

		onAdd(Todo{
			Text:      trimmed,
			Priority:  priority(),
			Category:  category(),
			DueDate:   dueDate(),
			Completed: false,
			CreatedAt: time.Now(),
		})

		setText("")
		setPriority("medium")
		setCategory("")
		setDueDate("")
	})

	return dom.Form(Attrs{
		"onsubmit": handleSubmit,
		"style":    "background: white; padding: 1.5rem; border-radius: 0.75rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); display: flex; flex-direction: column; gap: 1rem;",
	},
		dom.H3(Attrs{
			"style": "font-size: 1.125rem; font-weight: 700; color: #0f172a;",
		}, dom.Text("Add Todo")),
		dom.Div(Attrs{
			"style": "display: flex; flex-direction: column; gap: 0.75rem;",
		},
			dom.Div(Attrs{
				"style": "flex: 1;",
			},
				dom.Label(Attrs{
					"style": "display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem;",
				}, dom.Text("Text")),
				dom.Input(Attrs{
					"type":        "text",
					"value":       text(),
					"oninput":     hooks.GoUseFunc(func(e dom.GoEvent) { setText(e.GetValue()) }),
					"placeholder": "What needs to be done?",
					"style":       "width: 100%; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; outline: none;",
				}),
			),
			dom.Div(Attrs{
				"style": "width: 100%;",
			},
				dom.Label(Attrs{
					"style": "display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem;",
				}, dom.Text("Priority")),
				dom.Select(Attrs{
					"value":    priority(),
					"onchange": hooks.GoUseFunc(func(e dom.GoEvent) { setPriority(e.GetValue()) }),
					"style":    "width: 100%; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; outline: none;",
				},
					dom.Option(Attrs{"value": "high"}, dom.Text("High")),
					dom.Option(Attrs{"value": "medium"}, dom.Text("Medium")),
					dom.Option(Attrs{"value": "low"}, dom.Text("Low")),
				),
			),
		),
		dom.Div(Attrs{
			"style": "display: flex; flex-direction: column; gap: 0.75rem;",
		},
			dom.Div(Attrs{
				"style": "flex: 1;",
			},
				dom.Label(Attrs{
					"style": "display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem;",
				}, dom.Text("Category")),
				dom.Input(Attrs{
					"type":        "text",
					"value":       category(),
					"oninput":     hooks.GoUseFunc(func(e dom.GoEvent) { setCategory(e.GetValue()) }),
					"placeholder": "Optional category (e.g., Work)",
					"style":       "width: 100%; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; outline: none;",
				}),
			),
			dom.Div(Attrs{
				"style": "width: 100%;",
			},
				dom.Label(Attrs{
					"style": "display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem;",
				}, dom.Text("Due Date")),
				dom.Input(Attrs{
					"type":     "date",
					"value":    dueDate(),
					"onchange": hooks.GoUseFunc(func(e dom.GoEvent) { setDueDate(e.GetValue()) }),
					"style":    "width: 100%; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; outline: none;",
				}),
			),
		),
		dom.Div(Attrs{
			"style": "display: flex; justify-content: flex-end;",
		},
			dom.Button(Attrs{
				"type":  "submit",
				"style": "padding: 0.5rem 1rem; background: #4f46e5; color: white; border-radius: 0.5rem; border: none; cursor: pointer; font-weight: 500;",
			}, dom.Text("Add Todo")),
		),
	)
}

// Filters component.
func TodoFilters(props Attrs) *Element {
	filter := props["filter"].(TodoFilter)
	onChange := props["onChange"].(func(TodoFilter))

	handleStatus := hooks.GoUseFunc(func(e dom.GoEvent) {
		filter.Status = e.GetValue()
		onChange(filter)
	})

	handlePriority := hooks.GoUseFunc(func(e dom.GoEvent) {
		filter.Priority = e.GetValue()
		onChange(filter)
	})

	handleCategory := hooks.GoUseFunc(func(e dom.GoEvent) {
		filter.Category = e.GetValue()
		onChange(filter)
	})

	handleSearch := hooks.GoUseFunc(func(e dom.GoEvent) {
		filter.Search = e.GetValue()
		onChange(filter)
	})

	return dom.Div(Attrs{
		"style": "background: white; padding: 1.5rem; border-radius: 0.75rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1);",
	},
		dom.H3(Attrs{
			"style": "font-size: 1.125rem; font-weight: 700; color: #0f172a; margin-bottom: 1rem;",
		}, dom.Text("Filter")),
		dom.Div(Attrs{
			"style": "display: grid; grid-template-columns: repeat(2, 1fr); gap: 1rem;",
		},
			dom.Div(nil,
				dom.Label(Attrs{
					"style": "display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem;",
				}, dom.Text("Status")),
				dom.Select(Attrs{
					"value":    filter.Status,
					"onchange": handleStatus,
					"style":    "width: 100%; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; outline: none;",
				},
					dom.Option(Attrs{"value": "all"}, dom.Text("All")),
					dom.Option(Attrs{"value": "active"}, dom.Text("Active")),
					dom.Option(Attrs{"value": "completed"}, dom.Text("Completed")),
				),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"style": "display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem;",
				}, dom.Text("Priority")),
				dom.Select(Attrs{
					"value":    filter.Priority,
					"onchange": handlePriority,
					"style":    "width: 100%; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; outline: none;",
				},
					dom.Option(Attrs{"value": ""}, dom.Text("All Priorities")),
					dom.Option(Attrs{"value": "high"}, dom.Text("High")),
					dom.Option(Attrs{"value": "medium"}, dom.Text("Medium")),
					dom.Option(Attrs{"value": "low"}, dom.Text("Low")),
				),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"style": "display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem;",
				}, dom.Text("Category")),
				dom.Input(Attrs{
					"type":        "text",
					"value":       filter.Category,
					"oninput":     handleCategory,
					"placeholder": "Filter by category",
					"style":       "width: 100%; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; outline: none;",
				}),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"style": "display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem;",
				}, dom.Text("Search")),
				dom.Input(Attrs{
					"type":        "text",
					"value":       filter.Search,
					"oninput":     handleSearch,
					"placeholder": "Search todos",
					"style":       "width: 100%; padding: 0.5rem; border: 1px solid #cbd5e1; border-radius: 0.5rem; outline: none;",
				}),
			),
		),
	)
}

// Statistics component.
func TodoStatsDisplay(props Attrs) *Element {
	stats := props["stats"].(TodoStats)

	return dom.Div(Attrs{
		"style": "background: linear-gradient(to bottom right, white, #f8fafc); padding: 1.5rem; border-radius: 0.75rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1);",
	},
		dom.H3(Attrs{
			"style": "font-size: 1.125rem; font-weight: 700; color: #0f172a; margin-bottom: 1rem;",
		}, dom.Text("Statistics")),
		dom.Div(Attrs{
			"style": "display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem;",
		},
			dom.Div(Attrs{
				"style": "text-align: center;",
			},
				dom.Div(Attrs{
					"style": "font-size: 1.875rem; font-weight: 700; color: #4f46e5;",
				}, dom.Text(strconv.Itoa(stats.Total))),
				dom.Div(Attrs{
					"style": "font-size: 0.875rem; color: #475569; margin-top: 0.25rem;",
				}, dom.Text("Total")),
			),
			dom.Div(Attrs{
				"style": "text-align: center;",
			},
				dom.Div(Attrs{
					"style": "font-size: 1.875rem; font-weight: 700; color: #d97706;",
				}, dom.Text(strconv.Itoa(stats.Active))),
				dom.Div(Attrs{
					"style": "font-size: 0.875rem; color: #475569; margin-top: 0.25rem;",
				}, dom.Text("Active")),
			),
			dom.Div(Attrs{
				"style": "text-align: center;",
			},
				dom.Div(Attrs{
					"style": "font-size: 1.875rem; font-weight: 700; color: #10b981;",
				}, dom.Text(strconv.Itoa(stats.Completed))),
				dom.Div(Attrs{
					"style": "font-size: 0.875rem; color: #475569; margin-top: 0.25rem;",
				}, dom.Text("Completed")),
			),
		),
		func() *Element {
			if stats.Total == 0 {
				return dom.Div(nil)
			}
			percentage := float64(stats.Completed) / float64(stats.Total) * 100
			return dom.Div(Attrs{
				"style": "margin-top: 1rem;",
			},
				dom.Div(Attrs{
					"style": "display: flex; justify-content: space-between; font-size: 0.875rem; color: #475569; margin-bottom: 0.25rem;",
				},
					dom.Text("Progress"),
					dom.Text(fmt.Sprintf("%.1f%%", percentage)),
				),
				dom.Div(Attrs{
					"style": "width: 100%; background: #cbd5e1; border-radius: 9999px; height: 0.5rem;",
				},
					dom.Div(Attrs{
						"style": fmt.Sprintf("background: #4f46e5; height: 0.5rem; border-radius: 9999px; width: %.1f%%;", percentage),
					}),
				),
			)
		}(),
	)
}

// Bulk actions component.
func BulkActions(props Attrs) *Element {
	onMarkAllComplete := props["onMarkAllComplete"].(func())
	onDeleteCompleted := props["onDeleteCompleted"].(func())
	onDeleteAll := props["onDeleteAll"].(func())

	handleMarkAllComplete := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		onMarkAllComplete()
	})

	handleDeleteCompleted := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		onDeleteCompleted()
	})

	handleDeleteAll := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		if js.Global().Call("confirm", "Are you sure you want to delete all todos?").Bool() {
			onDeleteAll()
		}
	})

	return dom.Div(Attrs{
		"style": "background: white; padding: 1.5rem; border-radius: 0.75rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1);",
	},
		dom.H3(Attrs{
			"style": "font-size: 1.125rem; font-weight: 700; color: #0f172a; margin-bottom: 1rem;",
		}, dom.Text("Bulk Actions")),
		dom.Div(Attrs{
			"style": "display: flex; flex-wrap: wrap; gap: 0.75rem;",
		},
			dom.Button(Attrs{
				"onclick": handleMarkAllComplete,
				"style":   "padding: 0.625rem 1.25rem; background: #4f46e5; color: white; font-weight: 500; border-radius: 0.5rem; border: none; cursor: pointer; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);",
			}, dom.Text("Mark All Complete")),
			dom.Button(Attrs{
				"onclick": handleDeleteCompleted,
				"style":   "padding: 0.625rem 1.25rem; background: #d97706; color: white; font-weight: 500; border-radius: 0.5rem; border: none; cursor: pointer; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);",
			}, dom.Text("Delete Completed")),
			dom.Button(Attrs{
				"onclick": handleDeleteAll,
				"style":   "padding: 0.625rem 1.25rem; background: #e11d48; color: white; font-weight: 500; border-radius: 0.5rem; border: none; cursor: pointer; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);",
			}, dom.Text("Delete All")),
		),
	)
}

// TodoApp is the main todo application component.
func TodoApp(props Attrs) *Element {
	if appMetrics.LastRenderTime.IsZero() {
		appMetrics.LastRenderTime = time.Now()
		appMetrics.ComponentMounts = 1
	} else {
		appMetrics.TotalRenders++
	}

	renderStart := time.Now()
	defer func() {
		appMetrics.FiberTime = float64(time.Since(renderStart).Nanoseconds()) / 1e6
		logMetrics("RENDER_COMPLETE")
	}()

	todos, setTodos := hooks.UseState([]Todo{})
	filter, setFilter := hooks.UseState(TodoFilter{Status: "all"})
	nextID := hooks.UseRef(1)

	// Load todos from localStorage once.
	hooks.UseEffect(func() func() {
		storage := js.Global().Get("localStorage")
		if storage.Truthy() {
			if raw := storage.Call("getItem", "gowebcomponents-todos"); raw.Truthy() {
				var loaded []Todo
				if err := json.Unmarshal([]byte(raw.String()), &loaded); err == nil {
					setTodos(loaded)
					maxID := 0
					for _, t := range loaded {
						if t.ID > maxID {
							maxID = t.ID
						}
					}
					nextID.Current = maxID + 1
				}
			}
		}
		return nil
	})

	// Persist todos when they change.
	hooks.UseEffect(func() func() {
		storage := js.Global().Get("localStorage")
		if storage.Truthy() {
			if data, err := json.Marshal(todos()); err == nil {
				storage.Call("setItem", "gowebcomponents-todos", string(data))
			}
		}
		return nil
	}, todos())

	// Derived state: filtered todos.
	filteredTodos := hooks.UseMemo(func() interface{} {
		current := todos()
		currentFilter := filter()
		filtered := make([]Todo, 0, len(current))

		for _, todo := range current {
			if currentFilter.Status == "active" && todo.Completed {
				continue
			}
			if currentFilter.Status == "completed" && !todo.Completed {
				continue
			}
			if currentFilter.Search != "" && !strings.Contains(strings.ToLower(todo.Text), strings.ToLower(currentFilter.Search)) {
				continue
			}
			if currentFilter.Category != "" && !strings.Contains(strings.ToLower(todo.Category), strings.ToLower(currentFilter.Category)) {
				continue
			}
			if currentFilter.Priority != "" && todo.Priority != currentFilter.Priority {
				continue
			}
			filtered = append(filtered, todo)
		}
		return filtered
	}, todos(), filter()).([]Todo)

	// Stats.
	stats := hooks.UseMemo(func() interface{} {
		current := todos()
		result := TodoStats{
			ByPriority: make(map[string]int),
			ByCategory: make(map[string]int),
		}
		for _, todo := range current {
			result.Total++
			if todo.Completed {
				result.Completed++
			} else {
				result.Active++
			}
			result.ByPriority[todo.Priority]++
			if todo.Category != "" {
				result.ByCategory[todo.Category]++
			}
		}
		return result
	}, todos()).(TodoStats)

	handleAddTodo := func(newTodo Todo) {
		id := nextID.Current.(int)
		nextID.Current = id + 1
		newTodo.ID = id
		setTodos(append(todos(), newTodo))
		appMetrics.HookUpdates++
	}

	handleToggleTodo := func(id int) {
		setTodos(func(prev []Todo) []Todo {
			updated := make([]Todo, len(prev))
			for i, todo := range prev {
				if todo.ID == id {
					todo.Completed = !todo.Completed
					if todo.Completed {
						now := time.Now()
						todo.CompletedAt = &now
					} else {
						todo.CompletedAt = nil
					}
				}
				updated[i] = todo
			}
			return updated
		})
		appMetrics.HookUpdates++
	}

	handleDeleteTodo := func(id int) {
		setTodos(func(prev []Todo) []Todo {
			out := make([]Todo, 0, len(prev))
			for _, todo := range prev {
				if todo.ID != id {
					out = append(out, todo)
				}
			}
			return out
		})
		appMetrics.HookUpdates++
	}

	handleEditTodo := func(id int, text string) {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return
		}
		setTodos(func(prev []Todo) []Todo {
			updated := make([]Todo, len(prev))
			for i, todo := range prev {
				if todo.ID == id {
					todo.Text = trimmed
				}
				updated[i] = todo
			}
			return updated
		})
		appMetrics.HookUpdates++
	}

	handleFilterChange := func(f TodoFilter) {
		setFilter(f)
	}

	handleMarkAllComplete := func() {
		setTodos(func(prev []Todo) []Todo {
			updated := make([]Todo, len(prev))
			for i, t := range prev {
				if !t.Completed {
					now := time.Now()
					t.Completed = true
					t.CompletedAt = &now
				}
				updated[i] = t
			}
			return updated
		})
		appMetrics.HookUpdates++
	}

	handleDeleteCompleted := func() {
		setTodos(func(prev []Todo) []Todo {
			out := make([]Todo, 0, len(prev))
			for _, t := range prev {
				if !t.Completed {
					out = append(out, t)
				}
			}
			return out
		})
		appMetrics.HookUpdates++
	}

	handleDeleteAll := func() {
		setTodos([]Todo{})
		appMetrics.HookUpdates++
	}

	items := make([]interface{}, len(filteredTodos))
	for i, todo := range filteredTodos {
		items[i] = TodoItem(Attrs{
			"todo":     todo,
			"onToggle": handleToggleTodo,
			"onDelete": handleDeleteTodo,
			"onEdit":   handleEditTodo,
			"key":      todo.ID,
		})
	}

	return dom.Div(Attrs{
		"style": "min-height: 100vh; background: linear-gradient(135deg, #f1f5f9 0%, #e2e8f0 100%); padding: 2rem 1rem;",
	},
		dom.Div(Attrs{
			"style": "max-width: 56rem; margin: 0 auto;",
		},
			dom.Div(Attrs{
				"style": "margin-bottom: 2rem;",
			},
				dom.H1(Attrs{
					"style": "font-size: 2.25rem; font-weight: 700; color: #0f172a; margin-bottom: 0.5rem;",
				}, dom.Text("📝 Advanced Todo App")),
				dom.P(Attrs{
					"style": "color: #475569; font-size: 1.125rem;",
				}, dom.Text("New public API version with filters, stats, persistence, and metrics.")),
			),
			dom.Div(Attrs{
				"style": "display: flex; flex-direction: column; gap: 1.5rem;",
			},
				TodoInput(Attrs{"onAdd": handleAddTodo}),
				TodoFilters(Attrs{"filter": filter(), "onChange": handleFilterChange}),
				TodoStatsDisplay(Attrs{"stats": stats}),
				dom.Div(Attrs{
					"style": "background: white; border-radius: 0.75rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); overflow: hidden;",
				},
					func() *Element {
						if len(items) == 0 {
							return dom.P(Attrs{
								"style": "padding: 1.5rem; text-align: center; color: #708090;",
							}, dom.Text("No todos match the current filters."))
						}
						return dom.Ul(Attrs{
							"style": "list-style: none; padding: 0; margin: 0;",
						}, items...)
					}(),
				),
				BulkActions(Attrs{
					"onMarkAllComplete": handleMarkAllComplete,
					"onDeleteCompleted": handleDeleteCompleted,
					"onDeleteAll":       handleDeleteAll,
				}),
			),
		),
	)
}
