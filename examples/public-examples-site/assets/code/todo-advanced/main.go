//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"

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

func PriorityBadge(parseProps PriorityBadgeProps) ui.Node {
	parsePriority := parseProps.Priority
	if parsePriority == "" {
		parsePriority = "medium"
	}

	parseLabel := "Medium"
	parseColorClass := "bg-amber-500/20 text-amber-400 border border-amber-500/30"

	switch parsePriority {
	case "high":
		parseLabel = "High"
		parseColorClass = "bg-rose-500/20 text-rose-400 border border-rose-500/30"
	case "low":
		parseLabel = "Low"
		parseColorClass = "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
	}

	return html.Span(html.PropsOf(
		html.Class(fmt.Sprintf("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium %s", parseColorClass)),
	), html.Text(parseLabel))
}

func CategoryBadge(parseProps CategoryBadgeProps) ui.Node {
	if parseProps.Category == "" {
		return nil
	}

	return html.Span(html.PropsOf(
		html.Class("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-indigo-500/20 text-indigo-400 border border-indigo-500/30 ml-2"),
	), html.Text(parseProps.Category))
}

func DueDateDisplay(parseProps DueDateDisplayProps) ui.Node {
	if parseProps.DueDate == "" {
		return nil
	}

	isOverdue := false
	parseDisplayDate := parseProps.DueDate
	if parseParsedDate, parseErr := time.Parse("2006-01-02", parseProps.DueDate); parseErr == nil {
		if parseParsedDate.Before(time.Now()) {
			isOverdue = true
		}
		parseDisplayDate = parseParsedDate.Format("Jan 2, 2006")
	}

	parseColorClass := "text-gray-500"
	if isOverdue {
		parseColorClass = "text-red-400"
	}

	return html.Span(html.PropsOf(
		html.Class(fmt.Sprintf("text-xs %s ml-2", parseColorClass)),
		html.Title("Due date"),
	), html.Text(parseDisplayDate))
}

func TodoItem(parseProps TodoItemProps) ui.Node {
	handleToggle := ui.UseEvent(func() {
		parseProps.OnToggle(parseProps.Todo.ID)
	})

	handleDelete := ui.UseEvent(html.Prevent(func() {
		parseProps.OnDelete(parseProps.Todo.ID)
	}))

	parseTextClass := "flex-1 ml-3 text-gray-200"
	if parseProps.Todo.Completed {
		parseTextClass += " line-through text-gray-500"
	}

	return html.Li(html.Props{
		Class: "flex items-center p-4 border-b border-white/5 hover:bg-white/5 transition-colors group",
		Key:   strconv.Itoa(parseProps.Todo.ID),
	},
		html.Input(html.PropsOf(
			html.Type("checkbox"),
			html.Checked(parseProps.Todo.Completed),
			html.OnChange(handleToggle),
			html.Class("h-5 w-5 text-indigo-500 focus:ring-indigo-500 border-gray-600 rounded bg-black/20"),
		)),
		html.Div(html.PropsOf(html.Class(parseTextClass)),
			html.P(html.PropsOf(html.Class("font-medium")), html.Text(parseProps.Todo.Text)),
			html.Div(html.PropsOf(html.Class("flex items-center mt-2")),
				ui.CreateElement(PriorityBadge, PriorityBadgeProps{Priority: parseProps.Todo.Priority}),
				ui.CreateElement(CategoryBadge, CategoryBadgeProps{Category: parseProps.Todo.Category}),
				ui.CreateElement(DueDateDisplay, DueDateDisplayProps{DueDate: parseProps.Todo.DueDate}),
			),
		),
		html.Button(html.PropsOf(
			html.OnClick(handleDelete),
			html.Class("p-2 text-gray-500 hover:text-red-400 transition-colors opacity-0 group-hover:opacity-100"),
			html.Title("Delete todo"),
		), html.Text("✕")),
	)
}

func TodoInput(parseProps TodoInputProps) ui.Node {
	parseText := ui.UseState("")
	parsePriority := ui.UseState("medium")
	parseCategory := ui.UseState("")
	parseDueDate := ui.UseState("")

	handleSubmit := ui.UseEvent(html.Prevent(func() {
		parseTrimmed := strings.TrimSpace(parseText.Get())
		if parseTrimmed == "" {
			return
		}

		parseProps.OnAdd(Todo{
			Text:      parseTrimmed,
			Priority:  parsePriority.Get(),
			Category:  parseCategory.Get(),
			DueDate:   parseDueDate.Get(),
			Completed: false,
			CreatedAt: time.Now(),
		})

		parseText.Set("")
		parsePriority.Set("medium")
		parseCategory.Set("")
		parseDueDate.Set("")
	}))

	return html.Form(html.PropsOf(
		html.OnSubmit(handleSubmit),
		html.Class("rounded-[22px] border border-white/10 bg-slate-950/60 p-6"),
	),
		html.H3(html.PropsOf(html.Class("text-lg font-bold mb-4 text-white")), html.Text("Add Todo")),
		html.Div(html.PropsOf(html.Class("mb-4")),
			html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Text")),
			html.Input(html.PropsOf(
				html.Type("text"),
				html.Value(parseText.Get()),
				html.OnInput(ui.UseEvent(func(parseVal string) { parseText.Set(parseVal) })),
				html.Placeholder("What needs to be done?"),
				html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600"),
			)),
		),
		html.Div(html.PropsOf(html.Class("grid grid-cols-3 gap-4 mb-6")),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Priority")),
				html.Select(html.PropsOf(
					html.Value(parsePriority.Get()),
					html.OnChange(ui.UseEvent(func(parseE ui.Event) { parsePriority.Set(parseE.GetValue()) })),
					html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white"),
				),
					html.Option(html.PropsOf(html.Value("low"), html.SelectedIf(parsePriority.Get() == "low")), html.Text("Low")),
					html.Option(html.PropsOf(html.Value("medium"), html.SelectedIf(parsePriority.Get() == "medium")), html.Text("Medium")),
					html.Option(html.PropsOf(html.Value("high"), html.SelectedIf(parsePriority.Get() == "high")), html.Text("High")),
				),
			),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Category")),
				html.Input(html.PropsOf(
					html.Type("text"),
					html.Value(parseCategory.Get()),
					html.OnInput(ui.UseEvent(func(parseE2 ui.Event) { parseCategory.Set(parseE2.GetValue()) })),
					html.Placeholder("Work, Personal..."),
					html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600"),
				)),
			),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Due Date")),
				html.Input(html.PropsOf(
					html.Type("date"),
					html.Value(parseDueDate.Get()),
					html.OnInput(ui.UseEvent(func(parseE3 ui.Event) { parseDueDate.Set(parseE3.GetValue()) })),
					html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white"),
				)),
			),
		),
		html.Button(html.PropsOf(
			html.Type("submit"),
			html.Class("w-full rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-6 py-3 font-semibold text-cyan-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0 active:scale-95"),
		), html.Text("Add Todo")),
	)
}

func TodoFilters(parseProps TodoFiltersProps) ui.Node {
	parseButtonClass := func(isActive bool) string {
		if isActive {
			return "px-4 py-2 bg-blue-500 text-white rounded-lg font-semibold text-sm"
		}
		return "px-4 py-2 bg-white/5 text-gray-400 rounded-lg hover:bg-white/10 text-sm"
	}

	return html.Div(html.PropsOf(html.Class("rounded-[22px] border border-white/10 bg-slate-950/60 p-6")),
		html.H3(html.PropsOf(html.Class("text-lg font-bold mb-4 text-white")), html.Text("Filters")),
		html.Div(html.PropsOf(html.Class("grid grid-cols-2 gap-4")),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Status")),
				html.Div(html.PropsOf(html.Class("flex gap-2")),
					html.Button(html.PropsOf(html.OnClick(html.Prevent(func() {
						parseNext := parseProps.Filter
						parseNext.Status = "all"
						parseProps.OnChange(parseNext)
					})), html.Class(parseButtonClass(parseProps.Filter.Status == "all"))), html.Text("All")),
					html.Button(html.PropsOf(html.OnClick(html.Prevent(func() {
						parseNext2 := parseProps.Filter
						parseNext2.Status = "active"
						parseProps.OnChange(parseNext2)
					})), html.Class(parseButtonClass(parseProps.Filter.Status == "active"))), html.Text("Active")),
					html.Button(html.PropsOf(html.OnClick(html.Prevent(func() {
						parseNext3 := parseProps.Filter
						parseNext3.Status = "completed"
						parseProps.OnChange(parseNext3)
					})), html.Class(parseButtonClass(parseProps.Filter.Status == "completed"))), html.Text("Completed")),
				),
			),
			html.Div(html.Props{},
				html.Label(html.PropsOf(html.Class("block text-sm font-medium text-gray-400 mb-2")), html.Text("Search")),
				html.Input(html.PropsOf(
					html.Type("text"),
					html.Value(parseProps.Filter.Search),
					html.OnInput(ui.UseEvent(func(parseE ui.Event) {
						parseNext4 := parseProps.Filter
						parseNext4.Search = parseE.GetValue()
						parseProps.OnChange(parseNext4)
					})),
					html.Placeholder("Search todos..."),
					html.Class("w-full px-4 py-2 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600"),
				)),
			),
		),
	)
}

func TodoApp() ui.Node {
	parseTodos := ui.UseState([]Todo{})
	filter := ui.UseState(TodoFilter{Status: "all"})
	parseNextID := ui.UseState(1)

	parseAddTodo := func(parseTodo5 Todo) {
		parseId := parseNextID.Get()
		parseTodo5.ID = parseId
		parseNextID.Set(parseId + 1)
		parseTodos.Update(func(parsePrev []Todo) []Todo {
			parseNext := make([]Todo, 0, len(parsePrev)+1)
			parseNext = append(parseNext, parsePrev...)
			parseNext = append(parseNext, parseTodo5)
			return parseNext
		})
	}

	parseToggleTodo := func(parseId2 int) {
		parseTodos.Update(func(parsePrev2 []Todo) []Todo {
			parseNext2 := make([]Todo, len(parsePrev2))
			for parseIndex, parseTodo := range parsePrev2 {
				if parseTodo.ID == parseId2 {
					parseTodo.Completed = !parseTodo.Completed
				}
				parseNext2[parseIndex] = parseTodo
			}
			return parseNext2
		})
	}

	parseDeleteTodo := func(parseId3 int) {
		parseTodos.Update(func(parsePrev3 []Todo) []Todo {
			parseNext3 := make([]Todo, 0, len(parsePrev3))
			for _, parseTodo2 := range parsePrev3 {
				if parseTodo2.ID != parseId3 {
					parseNext3 = append(parseNext3, parseTodo2)
				}
			}
			return parseNext3
		})
	}

	parseCurrentTodos := parseTodos.Get()
	parseCurrentFilter := filter.Get()

	parseFilteredTodos := make([]Todo, 0, len(parseCurrentTodos))
	for _, parseTodo3 := range parseCurrentTodos {
		if parseCurrentFilter.Status == "active" && parseTodo3.Completed {
			continue
		}
		if parseCurrentFilter.Status == "completed" && !parseTodo3.Completed {
			continue
		}
		if parseCurrentFilter.Search != "" && !strings.Contains(strings.ToLower(parseTodo3.Text), strings.ToLower(parseCurrentFilter.Search)) {
			continue
		}
		parseFilteredTodos = append(parseFilteredTodos, parseTodo3)
	}

	parseTodoItems := html.MapKeyed(parseFilteredTodos, func(parseTodo6 Todo) interface{} { return parseTodo6.ID }, func(parseTodo7 Todo) ui.Node {
		return ui.CreateElement(TodoItem, TodoItemProps{
			Todo:     parseTodo7,
			OnToggle: parseToggleTodo,
			OnDelete: parseDeleteTodo,
		})
	})

	parseTotal := len(parseCurrentTodos)
	parseActive := 0
	parseCompleted := 0
	for _, parseTodo4 := range parseCurrentTodos {
		if parseTodo4.Completed {
			parseCompleted++
		} else {
			parseActive++
		}
	}

	return shared.ExamplePage(
		"Todo Advanced",
		"ui.UseState + html.MapKeyed",
		"Manage richer todo data with filtering, priority badges, categories, due dates, and keyed list updates.",
		shared.ExamplePanel("Composer",
			ui.CreateElement(TodoInput, TodoInputProps{OnAdd: parseAddTodo}),
		),
		shared.ExamplePanel("Filters",
			ui.CreateElement(TodoFilters, TodoFiltersProps{
				Filter:   parseCurrentFilter,
				OnChange: func(parseNext4 TodoFilter) { filter.Set(parseNext4) },
			}),
		),
		shared.ExamplePanel("List",
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-3"},
				shared.ExampleStat("Total", fmt.Sprintf("%d", parseTotal)),
				shared.ExampleStat("Active", fmt.Sprintf("%d", parseActive)),
				shared.ExampleStat("Completed", fmt.Sprintf("%d", parseCompleted)),
			),
			html.Div(html.PropsOf(html.Class("rounded-[22px] border border-white/10 bg-slate-950/60 overflow-hidden")),
				html.Ul(html.PropsOf(html.Class("divide-y divide-white/5")), parseTodoItems...),
			),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(TodoApp))
	exampleboot.WaitExampleRuntime()
}
