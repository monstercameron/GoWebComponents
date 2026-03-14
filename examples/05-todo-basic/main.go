//go:build js && wasm

package main

import (
	"fmt"
	"strconv"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type TodoItemProps struct {
	Text     string
	OnRemove func()
}

func TodoItem(props TodoItemProps) ui.Node {
	remove := ui.UseEvent(props.OnRemove)

	return html.Li(html.Props{
		Class: "flex items-center justify-between p-4 bg-white/5 border border-white/5 rounded-lg mb-2 group hover:border-white/10 transition-all",
	},
		html.Span(html.Props{
			Class: "text-gray-200",
		}, html.Text(props.Text)),
		html.Button(html.Props{
			OnClick: remove,
			Class:   "px-3 py-1 bg-red-500/10 text-red-400 border border-red-500/20 rounded hover:bg-red-500/20 transition-colors text-sm opacity-0 group-hover:opacity-100",
		}, html.Text("Remove")),
	)
}

func TodoList() ui.Node {
	todos := ui.UseState([]string{})
	newTodo := ui.UseState("")
	currentTodos := todos.Get()
	currentNewTodo := newTodo.Get()

	handleInput := ui.UseEvent(func(event ui.InputEvent) {
		newTodo.Set(event.GetValue())
	})

	addTodo := ui.UseEvent(func() {
		if currentNewTodo != "" {
			newTodos := make([]string, len(currentTodos)+1)
			copy(newTodos, currentTodos)
			newTodos[len(currentTodos)] = currentNewTodo
			todos.Set(newTodos)
			newTodo.Set("")
		}
	})

	clearAll := ui.UseEvent(func() {
		todos.Set([]string{})
	})

	todoItems := make([]ui.Node, 0, len(currentTodos))
	for i, todo := range currentTodos {
		index := i
		todoItems = append(todoItems, ui.CreateElement(TodoItem, TodoItemProps{
			Text: todo,
			OnRemove: func() {
				newTodos := make([]string, 0, len(currentTodos)-1)
				newTodos = append(newTodos, currentTodos[:index]...)
				newTodos = append(newTodos, currentTodos[index+1:]...)
				todos.Set(newTodos)
			},
		}))
	}

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
					Value:       currentNewTodo,
					OnInput:     handleInput,
					Placeholder: "Enter a new todo...",
					Class:       "flex-1 px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
				}),
				html.Button(html.Props{
					OnClick: addTodo,
					Class:   "px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold rounded-lg hover:opacity-90 transition-opacity shadow-lg shadow-purple-500/20",
				}, html.Text("Add")),
			),

			html.Div(html.Props{
				Class: "flex justify-between items-center mb-4 px-1",
			},
				html.P(html.Props{
					Class: "text-gray-400 text-sm font-medium",
				}, html.Text(fmt.Sprintf("Tasks: %d", len(currentTodos)))),

				html.Button(html.Props{
					OnClick: clearAll,
					Class:   "text-xs text-gray-500 hover:text-red-400 transition-colors",
				}, html.Text("Clear All")),
			),

			html.Ul(html.Props{
				Class: "space-y-2",
				ID:    "todo-list-" + strconv.Itoa(len(currentTodos)),
			}, todoItems...),
		),
	)
}

func main() {
	fmt.Println("ðŸš€ Todo Basic Example Started")
	ui.Render(ui.CreateElement(TodoList), "#app")
	fmt.Println("âœ… Todo Basic Example Rendered")
	select {}
}
