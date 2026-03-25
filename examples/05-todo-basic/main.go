//go:build js && wasm

package main

import (
	"fmt"
	"strconv"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

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
		Class: "flex items-center justify-between p-4 bg-white/5 border border-white/5 rounded-lg mb-2 group hover:border-white/10 transition-all",
	},
		html.Span(html.Props{
			Class: "text-gray-200",
		}, html.Text(parseProps.Text)),
		html.Button(html.Props{
			OnClick: parseRemove,
			Class:   "px-3 py-1 bg-red-500/10 text-red-400 border border-red-500/20 rounded hover:bg-red-500/20 transition-colors text-sm opacity-0 group-hover:opacity-100",
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

	return html.Div(html.Props{
		Class: "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		html.Div(html.Props{
			Class: "max-w-2xl w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			html.H2(html.Props{
				Class: "text-3xl font-bold mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, html.Text("Todo List")),

			html.Div(html.Props{Class: "flex gap-3 mb-8"},
				html.Input(html.Props{
					Type:        "text",
					Value:       parseCurrentNewTodo,
					OnInput:     handleInput,
					Placeholder: "Enter a new todo...",
					Class:       "flex-1 px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
				}),
				html.Button(html.Props{
					OnClick: parseAddTodo,
					Class:   "px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold rounded-lg hover:opacity-90 transition-opacity shadow-lg shadow-purple-500/20",
				}, html.Text("Add")),
			),

			html.Div(html.Props{
				Class: "flex justify-between items-center mb-4 px-1",
			},
				html.P(html.Props{
					Class: "text-gray-400 text-sm font-medium",
				}, html.Textf("Tasks: %d", len(parseCurrentTodos))),

				html.Button(html.Props{
					OnClick: clearAll,
					Class:   "text-xs text-gray-500 hover:text-red-400 transition-colors",
				}, html.Text("Clear All")),
			),

			html.If(len(parseCurrentTodos) == 0,
				html.P(html.Props{Class: "rounded-lg border border-dashed border-white/10 px-4 py-6 text-center text-sm text-gray-500"}, html.Text("No tasks yet. Add your first item above.")),
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
	ui.Render(ui.CreateElement(TodoList), "#app")
	fmt.Println("âœ… Todo Basic Example Rendered")
	select {}
}
