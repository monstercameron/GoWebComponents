//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

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

type PriorityBadgeProps struct {
	Priority string
}

type CategoryBadgeProps struct {
	Category string
}

type DueDateDisplayProps struct {
	DueDate string
}

type TodoItemProps struct {
	Todo     Todo
	OnToggle func(int)
	OnDelete func(int)
}

type TodoInputProps struct {
	OnAdd func(Todo)
}

type TodoFiltersProps struct {
	Filter   TodoFilter
	OnChange func(TodoFilter)
}

func PriorityBadge(props PriorityBadgeProps) ui.Node {
	priority := props.Priority
	if priority == "" {
		priority = "medium"
	}

	label := "Medium"
	colorClass := "bg-amber-500/20 text-amber-400 border border-amber-500/30"

	switch priority {
	case "high":
		label = "High"
		colorClass = "bg-rose-500/20 text-rose-400 border border-rose-500/30"
	case "low":
		label = "Low"
		colorClass = "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
	}

	return html.Span(html.PropsOf(
		html.Class(fmt.Sprintf("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium %s", colorClass)),
	), html.Text(label))
}

func CategoryBadge(props CategoryBadgeProps) ui.Node {
	if props.Category == "" {
		return nil
	}

	return html.Span(html.PropsOf(
		html.Class("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-indigo-500/20 text-indigo-400 border border-indigo-500/30 ml-2"),
	), html.Text(props.Category))
}

func DueDateDisplay(props DueDateDisplayProps) ui.Node {
	if props.DueDate == "" {
		return nil
	}

	isOverdue := false
	displayDate := props.DueDate
	if parsedDate, err := time.Parse("2006-01-02", props.DueDate); err == nil {
		if parsedDate.Before(time.Now()) {
			isOverdue = true
		}
		displayDate = parsedDate.Format("Jan 2, 2006")
	}

	colorClass := "text-gray-500"
	if isOverdue {
		colorClass = "text-red-400"
	}

	return html.Span(html.PropsOf(
		html.Class(fmt.Sprintf("text-xs %s ml-2", colorClass)),
		html.Title("Due date"),
	), html.Text(displayDate))
}

func TodoItem(props TodoItemProps) ui.Node {
	handleToggle := ui.UseEvent(func() {
		props.OnToggle(props.Todo.ID)
	})

	handleDelete := ui.UseEvent(html.Prevent(func() {
		props.OnDelete(props.Todo.ID)
	}))

	textClass := "flex-1 ml-3 text-gray-200"
	if props.Todo.Completed {
		textClass += " line-through text-gray-500"
	}

	return html.Li(html.Props{
		Class: "flex items-center p-4 border-b border-white/5 hover:bg-white/5 transition-colors group",
		Key:   strconv.Itoa(props.Todo.ID),
	},
		html.Input(html.PropsOf(
			html.Type("checkbox"),
			html.Checked(props.Todo.Completed),
			html.OnChange(handleToggle),
			html.Class("h-5 w-5 text-indigo-500 focus:ring-indigo-500 border-gray-600 rounded bg-black/20"),
		)),
		html.Div(html.PropsOf(html.Class(textClass)),
			html.P(html.PropsOf(html.Class("font-medium")), html.Text(props.Todo.Text)),
			html.Div(html.PropsOf(html.Class("flex items-center mt-2")),
				ui.CreateElement(PriorityBadge, PriorityBadgeProps{Priority: props.Todo.Priority}),
				ui.CreateElement(CategoryBadge, CategoryBadgeProps{Category: props.Todo.Category}),
				ui.CreateElement(DueDateDisplay, DueDateDisplayProps{DueDate: props.Todo.DueDate}),
			),
		),
		html.Button(html.PropsOf(
			html.OnClick(handleDelete),
			html.Class("p-2 text-gray-500 hover:text-red-400 transition-colors opacity-0 group-hover:opacity-100"),
			html.Title("Delete todo"),
		), html.Text("✕")),
	)
}

func TodoInput(props TodoInputProps) ui.Node {
	text := ui.UseState("")
	priority := ui.UseState("medium")
	category := ui.UseState("")
	dueDate := ui.UseState("")

	handleSubmit := ui.UseEvent(html.Prevent(func() {
		trimmed := strings.TrimSpace(text.Get())
		if trimmed == "" {
			return
		}

		props.OnAdd(Todo{
			Text:      trimmed,
			Priority:  priority.Get(),
			Category:  category.Get(),
			DueDate:   dueDate.Get(),
			Completed: false,
			CreatedAt: time.Now(),
		})

		text.Set("")
		priority.Set("medium")
		category.Set("")
		dueDate.Set("")
	}))

	return html.Form(html.PropsOf(
		html.OnSubmit(handleSubmit),
		html.Class("bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm mb-6"),
	),
		html.H3(html.PropsOf(html.Class("text-lg font-bold mb-4 text-white")), html.Text("Add Todo")),
		html.Div(html.PropsOf(html.Class("mb-4")),
			html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Text")),
			html.Input(html.PropsOf(
				html.Type("text"),
				html.Value(text.Get()),
				html.OnInput(ui.UseEvent(func(val string) { text.Set(val) })),
				html.Placeholder("What needs to be done?"),
				html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600"),
			)),
		),
		html.Div(html.PropsOf(html.Class("grid grid-cols-3 gap-4 mb-6")),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Priority")),
				html.Select(html.PropsOf(
					html.Value(priority.Get()),
					html.OnChange(ui.UseEvent(func(e ui.Event) { priority.Set(e.GetValue()) })),
					html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white"),
				),
					html.Option(html.PropsOf(html.Value("low"), html.SelectedIf(priority.Get() == "low")), html.Text("Low")),
					html.Option(html.PropsOf(html.Value("medium"), html.SelectedIf(priority.Get() == "medium")), html.Text("Medium")),
					html.Option(html.PropsOf(html.Value("high"), html.SelectedIf(priority.Get() == "high")), html.Text("High")),
				),
			),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Category")),
				html.Input(html.PropsOf(
					html.Type("text"),
					html.Value(category.Get()),
					html.OnInput(ui.UseEvent(func(e ui.Event) { category.Set(e.GetValue()) })),
					html.Placeholder("Work, Personal..."),
					html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600"),
				)),
			),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Due Date")),
				html.Input(html.PropsOf(
					html.Type("date"),
					html.Value(dueDate.Get()),
					html.OnInput(ui.UseEvent(func(e ui.Event) { dueDate.Set(e.GetValue()) })),
					html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white"),
				)),
			),
		),
		html.Button(html.PropsOf(
			html.Type("submit"),
			html.Class("w-full px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-lg hover:opacity-90 transition-opacity font-semibold shadow-lg shadow-purple-500/20"),
		), html.Text("Add Todo")),
	)
}

func TodoFilters(props TodoFiltersProps) ui.Node {
	buttonClass := func(active bool) string {
		if active {
			return "px-4 py-2 bg-blue-500 text-white rounded-lg font-semibold text-sm"
		}
		return "px-4 py-2 bg-white/5 text-gray-400 rounded-lg hover:bg-white/10 text-sm"
	}

	return html.Div(html.PropsOf(html.Class("bg-white/5 border border-white/10 p-6 rounded-xl backdrop-blur-sm mb-6")),
		html.H3(html.PropsOf(html.Class("text-lg font-bold mb-4 text-white")), html.Text("Filters")),
		html.Div(html.PropsOf(html.Class("grid grid-cols-2 gap-4")),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Status")),
				html.Div(html.PropsOf(html.Class("flex gap-2")),
					html.Button(html.PropsOf(html.OnClick(html.Prevent(func() {
						next := props.Filter
						next.Status = "all"
						props.OnChange(next)
					})), html.Class(buttonClass(props.Filter.Status == "all"))), html.Text("All")),
					html.Button(html.PropsOf(html.OnClick(html.Prevent(func() {
						next := props.Filter
						next.Status = "active"
						props.OnChange(next)
					})), html.Class(buttonClass(props.Filter.Status == "active"))), html.Text("Active")),
					html.Button(html.PropsOf(html.OnClick(html.Prevent(func() {
						next := props.Filter
						next.Status = "completed"
						props.OnChange(next)
					})), html.Class(buttonClass(props.Filter.Status == "completed"))), html.Text("Completed")),
				),
			),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Search")),
				html.Input(html.PropsOf(
					html.Type("text"),
					html.Value(props.Filter.Search),
					html.OnInput(ui.UseEvent(func(e ui.Event) { next := props.Filter; next.Search = e.GetValue(); props.OnChange(next) })),
					html.Placeholder("Search todos..."),
					html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600"),
				)),
			),
		),
	)
}

func TodoApp() ui.Node {
	todos := ui.UseState([]Todo{})
	filter := ui.UseState(TodoFilter{Status: "all"})
	nextID := ui.UseState(1)

	addTodo := func(todo Todo) {
		id := nextID.Get()
		todo.ID = id
		nextID.Set(id + 1)
		todos.Update(func(prev []Todo) []Todo {
			next := make([]Todo, 0, len(prev)+1)
			next = append(next, prev...)
			next = append(next, todo)
			return next
		})
	}

	toggleTodo := func(id int) {
		todos.Update(func(prev []Todo) []Todo {
			next := make([]Todo, len(prev))
			for index, todo := range prev {
				if todo.ID == id {
					todo.Completed = !todo.Completed
				}
				next[index] = todo
			}
			return next
		})
	}

	deleteTodo := func(id int) {
		todos.Update(func(prev []Todo) []Todo {
			next := make([]Todo, 0, len(prev))
			for _, todo := range prev {
				if todo.ID != id {
					next = append(next, todo)
				}
			}
			return next
		})
	}

	currentTodos := todos.Get()
	currentFilter := filter.Get()

	filteredTodos := make([]Todo, 0, len(currentTodos))
	for _, todo := range currentTodos {
		if currentFilter.Status == "active" && todo.Completed {
			continue
		}
		if currentFilter.Status == "completed" && !todo.Completed {
			continue
		}
		if currentFilter.Search != "" && !strings.Contains(strings.ToLower(todo.Text), strings.ToLower(currentFilter.Search)) {
			continue
		}
		filteredTodos = append(filteredTodos, todo)
	}

	todoItems := html.MapKeyed(filteredTodos, func(todo Todo) interface{} { return todo.ID }, func(todo Todo) ui.Node {
		return ui.CreateElement(TodoItem, TodoItemProps{
			Todo:     todo,
			OnToggle: toggleTodo,
			OnDelete: deleteTodo,
		})
	})

	total := len(currentTodos)
	active := 0
	completed := 0
	for _, todo := range currentTodos {
		if todo.Completed {
			completed++
		} else {
			active++
		}
	}

	return html.Div(html.PropsOf(html.Class("min-h-screen bg-[#0a0a0a] text-white p-8")),
		html.Div(html.PropsOf(html.Class("max-w-4xl mx-auto")),
			html.H1(html.PropsOf(html.Class("text-4xl font-bold mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500")), html.Text("Advanced Todo App")),
			ui.CreateElement(TodoInput, TodoInputProps{OnAdd: addTodo}),
			ui.CreateElement(TodoFilters, TodoFiltersProps{
				Filter:   currentFilter,
				OnChange: func(next TodoFilter) { filter.Set(next) },
			}),
			html.Div(html.PropsOf(html.Class("bg-white/5 border border-white/10 p-4 rounded-lg mb-6 flex gap-6 text-sm text-gray-400")),
				html.Span(html.Props{}, html.Textf("Total: %d", total)),
				html.Span(html.Props{}, html.Textf("Active: %d", active)),
				html.Span(html.Props{}, html.Textf("Completed: %d", completed)),
			),
			html.Div(html.PropsOf(html.Class("bg-white/5 border border-white/10 rounded-xl overflow-hidden")),
				html.Ul(html.PropsOf(html.Class("divide-y divide-white/5")), todoItems...),
			),
		),
	)
}

func main() {
	ui.Render(ui.CreateElement(TodoApp), "body")
	select {}
}
