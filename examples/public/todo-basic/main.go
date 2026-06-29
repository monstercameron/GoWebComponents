//go:build js && wasm

package main

import (
	"fmt"
	"strconv"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type TodoItemProps struct {
	Text     string
	OnRemove func()
}

type todoRow struct {
	Index int
	Text  string
}

func TodoItem(parseProps TodoItemProps) ui.Node {
	parseRemove := ui.UseEvent(parseProps.OnRemove)

	return html.Li(html.Props{
		Class: "flex items-center justify-between gap-3 rounded-[20px] border border-white/10 bg-slate-950/60 px-4 py-3",
	},
		html.Span(html.Props{
			Class: "min-w-0 break-words text-slate-100",
		}, html.Text(parseProps.Text)),
		html.Button(html.Props{
			OnClick: parseRemove,
			Class:   "rounded-2xl border border-rose-400/20 bg-rose-400/10 px-3 py-2 text-sm font-medium text-rose-100 transition hover:bg-rose-400/15",
		}, html.Text("Remove")),
	)
}

func TodoList() ui.Node {
	parseTodos := ui.UseState([]string{})
	parseNewTodo := ui.UseState("")
	parseCurrentTodos := parseTodos.Get()
	parseCurrentNewTodo := parseNewTodo.Get()

	handleInput := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseNewTodo.Set(parseEvent.GetValue())
	})

	parseAddTodo := ui.UseEvent(func() {
		if parseCurrentNewTodo != "" {
			parseNewTodos := make([]string, len(parseCurrentTodos)+1)
			copy(parseNewTodos, parseCurrentTodos)
			parseNewTodos[len(parseCurrentTodos)] = parseCurrentNewTodo
			parseTodos.Set(parseNewTodos)
			parseNewTodo.Set("")
		}
	})

	clearAll := ui.UseEvent(func() {
		parseTodos.Set([]string{})
	})

	parseTodoRows := make([]todoRow, len(parseCurrentTodos))
	for parseI, parseTodo := range parseCurrentTodos {
		parseTodoRows[parseI] = todoRow{Index: parseI, Text: parseTodo}
	}
	parseTodoItems := html.Map(parseTodoRows, func(parseRow todoRow) ui.Node {
		return ui.CreateElement(TodoItem, TodoItemProps{
			Text: parseRow.Text,
			OnRemove: func() {
				parseNewTodos2 := make([]string, 0, len(parseCurrentTodos)-1)
				parseNewTodos2 = append(parseNewTodos2, parseCurrentTodos[:parseRow.Index]...)
				parseNewTodos2 = append(parseNewTodos2, parseCurrentTodos[parseRow.Index+1:]...)
				parseTodos.Set(parseNewTodos2)
			},
		})
	})

	return shared.ExamplePage(
		"Todo Basic",
		"ui.UseState",
		"Manage one local slice of todos with add, remove, and reset actions.",
		shared.ExamplePanel("Composer",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				html.Input(html.Props{
					Type:        "text",
					Value:       parseCurrentNewTodo,
					OnInput:     handleInput,
					Placeholder: "Enter a new todo...",
					Class:       "min-w-[220px] flex-1 rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
				}),
				html.Button(html.Props{
					OnClick: parseAddTodo,
					Class:   "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-2 text-sm font-medium text-cyan-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-cyan-400/20 active:translate-y-0 active:scale-95",
				}, html.Text("Add")),
			),
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
				shared.ExampleStat("Tasks", fmt.Sprintf("%d", len(parseCurrentTodos))),
				shared.ExampleStat("Input", func() string {
					if parseCurrentNewTodo == "" {
						return "Empty"
					}
					return "Ready"
				}()),
			),
		),
		shared.ExamplePanel("List",
			html.Div(html.Props{
				Class: "flex items-center justify-between gap-3",
			},
				html.Button(html.Props{
					OnClick: clearAll,
					Class:   "rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-xs font-medium uppercase tracking-[0.16em] text-slate-200 transition hover:border-cyan-300/30 hover:text-cyan-100",
				}, html.Text("Clear All")),
			),
			html.If(len(parseCurrentTodos) == 0,
				html.P(html.Props{Class: "rounded-[20px] border border-dashed border-white/10 px-4 py-6 text-center text-sm text-slate-400"}, html.Text("No tasks yet. Add your first item above.")),
			),
			html.Unless(len(parseCurrentTodos) == 0,
				html.Ul(html.Props{
					Class: "space-y-2",
					ID:    "todo-list-" + strconv.Itoa(len(parseCurrentTodos)),
				}, parseTodoItems...),
			),
		),
	)
}

func main() {
	fmt.Println("ðŸš€ Todo Basic Example Started")
	exampleboot.RenderExampleRoot(ui.CreateElement(TodoList))
	fmt.Println("âœ… Todo Basic Example Rendered")
	exampleboot.WaitExampleRuntime()
}
