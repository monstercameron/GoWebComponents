//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

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

func TodoItem(props Attrs) *Element {
	todo := props["todo"].(Todo)
	onToggle := props["onToggle"].(func(int))
	onDelete := props["onDelete"].(func(int))

	handleToggle := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		onToggle(todo.ID)
	})

	handleDelete := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		onDelete(todo.ID)
	})

	itemClass := "flex items-center p-4 border-b border-slate-200 hover:bg-slate-50 transition-colors"
	textClass := "flex-1 ml-3 text-slate-900"
	if todo.Completed {
		textClass += " line-through text-slate-500"
	}

	return dom.Li(Attrs{"class": itemClass, "key": strconv.Itoa(todo.ID)},
		dom.Input(Attrs{
			"type":     "checkbox",
			"checked":  todo.Completed,
			"onchange": handleToggle,
			"class":    "h-5 w-5 text-indigo-600 focus:ring-indigo-500 border-slate-300 rounded",
		}),
		dom.Div(Attrs{"class": textClass},
			dom.P(Attrs{"class": "font-medium"}, dom.Text(todo.Text)),
			dom.Div(Attrs{"class": "flex items-center mt-1"},
				PriorityBadge(Attrs{"priority": todo.Priority}),
				CategoryBadge(Attrs{"category": todo.Category}),
				DueDateDisplay(Attrs{"dueDate": todo.DueDate}),
			),
		),
		dom.Button(Attrs{
			"onclick": handleDelete,
			"class":   "p-1 text-gray-400 hover:text-red-600 transition-colors",
			"title":   "Delete todo",
		}, dom.Text("🗑️")),
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
		"class":    "bg-white p-6 rounded-lg shadow-lg mb-6",
	},
		dom.H3(Attrs{
			"class": "text-lg font-bold mb-4 text-gray-800",
		}, dom.Text("Add Todo")),

		dom.Div(Attrs{"class": "mb-4"},
			dom.Label(Attrs{
				"class": "block text-sm font-medium text-gray-700 mb-2",
			}, dom.Text("Text")),
			dom.Input(Attrs{
				"type":        "text",
				"value":       text(),
				"oninput":     hooks.GoUseFunc(func(e dom.GoEvent) { setText(e.GetValue()) }),
				"placeholder": "What needs to be done?",
				"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500",
			}),
		),

		dom.Div(Attrs{"class": "grid grid-cols-3 gap-4 mb-4"},
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-700 mb-2",
				}, dom.Text("Priority")),
				dom.Select(Attrs{
					"value":    priority(),
					"onchange": hooks.GoUseFunc(func(e dom.GoEvent) { setPriority(e.GetValue()) }),
					"class":    "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500",
				},
					dom.Option(Attrs{"value": "low"}, dom.Text("Low")),
					dom.Option(Attrs{"value": "medium"}, dom.Text("Medium")),
					dom.Option(Attrs{"value": "high"}, dom.Text("High")),
				),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-700 mb-2",
				}, dom.Text("Category")),
				dom.Input(Attrs{
					"type":        "text",
					"value":       category(),
					"oninput":     hooks.GoUseFunc(func(e dom.GoEvent) { setCategory(e.GetValue()) }),
					"placeholder": "Work, Personal...",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500",
				}),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-700 mb-2",
				}, dom.Text("Due Date")),
				dom.Input(Attrs{
					"type":     "date",
					"value":    dueDate(),
					"oninput":  hooks.GoUseFunc(func(e dom.GoEvent) { setDueDate(e.GetValue()) }),
					"class":    "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500",
				}),
			),
		),

		dom.Button(Attrs{
			"type":  "submit",
			"class": "w-full px-6 py-3 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors font-semibold",
		}, dom.Text("Add Todo")),
	)
}

func TodoFilters(props Attrs) *Element {
	filter := props["filter"].(TodoFilter)
	onChange := props["onChange"].(func(TodoFilter))

	return dom.Div(Attrs{
		"class": "bg-white p-4 rounded-lg shadow-lg mb-6",
	},
		dom.H3(Attrs{
			"class": "text-lg font-bold mb-4 text-gray-800",
		}, dom.Text("Filters")),

		dom.Div(Attrs{"class": "grid grid-cols-2 gap-4"},
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-700 mb-2",
				}, dom.Text("Status")),
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
								return "px-4 py-2 bg-indigo-600 text-white rounded-lg font-semibold"
							}
							return "px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
						}(),
					}, dom.Text("All")),
					dom.Button(Attrs{
						"onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
							e.PreventDefault()
							newFilter := filter
							newFilter.Status = "active"
							onChange(newFilter)
						}),
						"class": func() string {
							if filter.Status == "active" {
								return "px-4 py-2 bg-indigo-600 text-white rounded-lg font-semibold"
							}
							return "px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
						}(),
					}, dom.Text("Active")),
					dom.Button(Attrs{
						"onclick": hooks.GoUseFunc(func(e dom.GoEvent) {
							e.PreventDefault()
							newFilter := filter
							newFilter.Status = "completed"
							onChange(newFilter)
						}),
						"class": func() string {
							if filter.Status == "completed" {
								return "px-4 py-2 bg-indigo-600 text-white rounded-lg font-semibold"
							}
							return "px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
						}(),
					}, dom.Text("Completed")),
				),
			),
			dom.Div(nil,
				dom.Label(Attrs{
					"class": "block text-sm font-medium text-gray-700 mb-2",
				}, dom.Text("Search")),
				dom.Input(Attrs{
					"type":        "text",
					"value":       filter.Search,
					"oninput":     hooks.GoUseFunc(func(e dom.GoEvent) {
						newFilter := filter
						newFilter.Search = e.GetValue()
						onChange(newFilter)
					}),
					"placeholder": "Search todos...",
					"class":       "w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500",
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
		todoItems = append(todoItems, TodoItem(Attrs{
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
		"class": "max-w-4xl mx-auto mt-8 p-6",
	},
		dom.H1(Attrs{
			"class": "text-4xl font-bold mb-8 text-gray-800",
		}, dom.Text("Advanced Todo App")),

		TodoInput(Attrs{"onAdd": addTodo}),

		TodoFilters(Attrs{
			"filter":   currentFilter,
			"onChange": setFilter,
		}),

		dom.Div(Attrs{
			"class": "bg-white p-4 rounded-lg shadow-lg mb-6",
		},
			dom.Div(Attrs{"class": "flex gap-6 text-sm text-gray-600"},
				dom.Span(nil, dom.Text(fmt.Sprintf("Total: %d", total))),
				dom.Span(nil, dom.Text(fmt.Sprintf("Active: %d", active))),
				dom.Span(nil, dom.Text(fmt.Sprintf("Completed: %d", completed))),
			),
		),

		dom.Div(Attrs{
			"class": "bg-white rounded-lg shadow-lg overflow-hidden",
		},
			dom.Ul(Attrs{"class": "divide-y divide-slate-200"}, todoItems...),
		),
	)
}

func main() {
	container := js.Global().Get("document").Call("getElementById", "app")
	element := dom.CreateElement(TodoApp, nil)
	render.ToElement(element, container)
	select {}
}
