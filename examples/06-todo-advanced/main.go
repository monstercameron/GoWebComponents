//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = dom.Element

type Todo struct {
	ID        int
	Text      string
	Completed bool
	Priority  string
	Category  string
	DueDate   string
	CreatedAt time.Time
}

type TodoFilter struct {
	Status   string
	Category string
	Priority string
	Search   string
}

func PriorityBadge(props Attrs) *Element {
	priority := "medium"
	if props != nil && props["priority"] != nil {
		priority = props["priority"].(string)
	}

	var colorClass string
	switch priority {
	case "high":
		colorClass = "bg-rose-500/20 text-rose-400 border border-rose-500/30"
	case "medium":
		colorClass = "bg-amber-500/20 text-amber-400 border border-amber-500/30"
	case "low":
		colorClass = "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
	default:
		colorClass = "bg-slate-500/20 text-slate-400 border border-slate-500/30"
	}

	return dom.Span(Attrs{
		"class": fmt.Sprintf("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium %s", colorClass),
	}, strings.Title(priority))
}

func CategoryBadge(props Attrs) *Element {
	category := ""
	if props != nil && props["category"] != nil {
		category = props["category"].(string)
	}

	if category == "" {
		return dom.Span(nil)
	}

	return dom.Span(Attrs{
		"class": "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-indigo-500/20 text-indigo-400 border border-indigo-500/30 ml-2",
	}, category)
}

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

	colorClass := "text-gray-500"
	if isOverdue {
		colorClass = "text-red-400"
	}

	return dom.Span(Attrs{
		"class": fmt.Sprintf("text-xs %s ml-2", colorClass),
		"title": "Due date",
	}, fmt.Sprintf(" %s", displayDate))
}

func TodoItem(props Attrs) *Element {
	todo := props["todo"].(Todo)
	onToggle := props["onToggle"].(func(int))
	onDelete := props["onDelete"].(func(int))

	handleToggle := hooks.GoUseFunc(func(event dom.GoEvent) {
		onToggle(todo.ID)
	})

	handleDelete := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		onDelete(todo.ID)
	})

	itemClass := "flex items-center p-4 border-b border-white/5 hover:bg-white/5 transition-colors group"
	textClass := "flex-1 ml-3 text-gray-200"
	if todo.Completed {
		textClass += " line-through text-gray-500"
	}

	return dom.Li(Attrs{"class": itemClass, "key": strconv.Itoa(todo.ID)},
		dom.Input(Attrs{
			"type":     "checkbox",
			"checked":  todo.Completed,
			"onchange": handleToggle,
			"class":    "h-5 w-5 text-indigo-500 focus:ring-indigo-500 border-gray-600 rounded bg-black/20",
		}),
		dom.Div(Attrs{"class": textClass},
			dom.P(Attrs{"class": "font-medium"}, todo.Text),
			dom.Div(Attrs{"class": "flex items-center mt-2"},
				dom.CreateElement(PriorityBadge, Attrs{"priority": todo.Priority}),
				dom.CreateElement(CategoryBadge, Attrs{"category": todo.Category}),
				dom.CreateElement(DueDateDisplay, Attrs{"dueDate": todo.DueDate}),
			),
		),
		dom.Button(Attrs{
			"onclick": handleDelete,
			"class":   "p-2 text-gray-500 hover:text-red-400 transition-colors opacity-0 group-hover:opacity-100",
			"title":   "Delete todo",
		}, "✕"),
	)
}

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
		"class":    "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm mb-6",
	},
		dom.H3(Attrs{
			"class": "text-lg font-bold mb-4 text-white",
		}, "Add Todo"),

		dom.Div(Attrs{"class": "mb-4"},
			dom.Label(Attrs{
				"class": "block text-sm font-medium text-gray-400 mb-2",
			}, "Text"),
			dom.Input(Attrs{
				"type":  "text",
				"value": text(),
				"oninput": hooks.GoUseFunc(func(val string) {
					setText(val)
				}),
				"placeholder": "What needs to be done?",
				"class":       "w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600",
			}),
		),

		dom.Div(Attrs{"class": "grid grid-cols-3 gap-4 mb-6"},
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-400 mb-2",
				}, "Priority"),
				dom.Select(Attrs{
					"value":    priority(),
					"onchange": hooks.GoUseFunc(func(e dom.GoEvent) { setPriority(e.GetValue()) }),
					"class":    "w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white",
				},
					dom.Option(Attrs{"value": "low"}, "Low"),
					dom.Option(Attrs{"value": "medium"}, "Medium"),
					dom.Option(Attrs{"value": "high"}, "High"),
				),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-400 mb-2",
				}, "Category"),
				dom.Input(Attrs{
					"type":        "text",
					"value":       category(),
					"oninput":     hooks.GoUseFunc(func(e dom.GoEvent) { setCategory(e.GetValue()) }),
					"placeholder": "Work, Personal...",
					"class":       "w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600",
				}),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-400 mb-2",
				}, "Due Date"),
				dom.Input(Attrs{
					"type":    "date",
					"value":   dueDate(),
					"oninput": hooks.GoUseFunc(func(e dom.GoEvent) { setDueDate(e.GetValue()) }),
					"class":   "w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white",
				}),
			),
		),

		dom.Button(Attrs{
			"type":  "submit",
			"class": "w-full px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-lg hover:opacity-90 transition-opacity font-semibold shadow-lg shadow-purple-500/20",
		}, "Add Todo"),
	)
}

func TodoFilters(props Attrs) *Element {
	filter := props["filter"].(TodoFilter)
	onChange := props["onChange"].(func(TodoFilter))

	return dom.Div(Attrs{
		"class": "bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm mb-6",
	},
		dom.H3(Attrs{
			"class": "text-lg font-bold mb-4 text-white",
		}, "Filters"),

		dom.Div(Attrs{"class": "grid grid-cols-2 gap-4"},
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-400 mb-2",
				}, "Status"),
				dom.Div(Attrs{"class": "flex gap-2"},
					dom.Button(Attrs{
						"onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
							e.PreventDefault()
							newFilter := filter
							newFilter.Status = "all"
							onChange(newFilter)
						}),
						"class": func() string {
							if filter.Status == "all" {
								return "px-4 py-2 bg-blue-500 text-white rounded-lg font-semibold text-sm"
							}
							return "px-4 py-2 bg-white/5 text-gray-400 rounded-lg hover:bg-white/10 text-sm"
						}(),
					}, "All"),
					dom.Button(Attrs{
						"onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
							e.PreventDefault()
							newFilter := filter
							newFilter.Status = "active"
							onChange(newFilter)
						}),
						"class": func() string {
							if filter.Status == "active" {
								return "px-4 py-2 bg-blue-500 text-white rounded-lg font-semibold text-sm"
							}
							return "px-4 py-2 bg-white/5 text-gray-400 rounded-lg hover:bg-white/10 text-sm"
						}(),
					}, "Active"),
					dom.Button(Attrs{
						"onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
							e.PreventDefault()
							newFilter := filter
							newFilter.Status = "completed"
							onChange(newFilter)
						}),
						"class": func() string {
							if filter.Status == "completed" {
								return "px-4 py-2 bg-blue-500 text-white rounded-lg font-semibold text-sm"
							}
							return "px-4 py-2 bg-white/5 text-gray-400 rounded-lg hover:bg-white/10 text-sm"
						}(),
					}, "Completed"),
				),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-400 mb-2",
				}, "Search"),
				dom.Input(Attrs{
					"type":  "text",
					"value": filter.Search,
					"oninput": hooks.GoUseFunc(func(e dom.GoEvent) {
						newFilter := filter
						newFilter.Search = e.GetValue()
						onChange(newFilter)
					}),
					"placeholder": "Search todos...",
					"class":       "w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600",
				}),
			),
		),
	)
}

func TodoApp(_ Attrs) *Element {
	todos, setTodos := hooks.UseState([]Todo{})
	filter, setFilter := hooks.UseState(TodoFilter{Status: "all"})
	nextID, setNextID := hooks.UseState(1)

	currentTodos := todos()
	currentFilter := filter()

	addTodo := func(todo Todo) {
		todo.ID = nextID()
		setNextID(nextID() + 1)
		newTodos := append(currentTodos, todo)
		setTodos(newTodos)
	}

	toggleTodo := func(id int) {
		newTodos := make([]Todo, len(currentTodos))
		for i, t := range currentTodos {
			if t.ID == id {
				t.Completed = !t.Completed
			}
			newTodos[i] = t
		}
		setTodos(newTodos)
	}

	deleteTodo := func(id int) {
		newTodos := []Todo{}
		for _, t := range currentTodos {
			if t.ID != id {
				newTodos = append(newTodos, t)
			}
		}
		setTodos(newTodos)
	}

	// Filter todos
	filteredTodos := []Todo{}
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

		filteredTodos = append(filteredTodos, todo)
	}

	// Create todo items
	todoItems := make([]interface{}, 0, len(filteredTodos))
	for _, todo := range filteredTodos {
		todoItems = append(todoItems, dom.CreateElement(TodoItem, Attrs{
			"todo":     todo,
			"onToggle": toggleTodo,
			"onDelete": deleteTodo,
		}))
	}

	total := len(currentTodos)
	active := 0
	completed := 0
	for _, t := range currentTodos {
		if t.Completed {
			completed++
		} else {
			active++
		}
	}

	return dom.Div(Attrs{
		"class": "min-h-screen bg-[#0a0a0a] text-white p-8",
	},
		dom.Div(Attrs{
			"class": "max-w-4xl mx-auto",
		},
			dom.H1(Attrs{
				"class": "text-4xl font-bold mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, "Advanced Todo App"),

			dom.CreateElement(TodoInput, Attrs{"onAdd": addTodo}),

			dom.CreateElement(TodoFilters, Attrs{
				"filter":   currentFilter,
				"onChange": func(f TodoFilter) { setFilter(f) },
			}),

			dom.Div(Attrs{
				"class": "bg-white/5 border border-white/10 p-4 rounded-lg mb-6 flex gap-6 text-sm text-gray-400",
			},
				dom.Span(nil, fmt.Sprintf("Total: %d", total)),
				dom.Span(nil, fmt.Sprintf("Active: %d", active)),
				dom.Span(nil, fmt.Sprintf("Completed: %d", completed)),
			),

			dom.Div(Attrs{
				"class": "bg-white/5 border border-white/10 rounded-xl overflow-hidden",
			},
				dom.Ul(Attrs{"class": "divide-y divide-white/5"}, todoItems...),
			),
		),
	)
}

func main() {
	render.To(dom.CreateElement(TodoApp, nil), "body")
	select {}
}
