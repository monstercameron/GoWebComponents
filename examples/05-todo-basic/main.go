//go:build js && wasm

package main

import (
	"fmt"
	"strconv"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

func TodoItem(props Attrs) *Element {
	text := props["text"].(string)
	onRemove := props["onRemove"].(func(dom.GoEvent))

	handleRemove := hooks.GoUseFunc(onRemove)

	return dom.Li(Attrs{
		"class": "flex items-center justify-between p-4 bg-white/5 border border-white/5 rounded-lg mb-2 group hover:border-white/10 transition-all",
	},
		dom.Span(Attrs{
			"class": "text-gray-200",
		}, dom.Text(text)),
		dom.Button(Attrs{
			"onclick": handleRemove,
			"class":   "px-3 py-1 bg-red-500/10 text-red-400 border border-red-500/20 rounded hover:bg-red-500/20 transition-colors text-sm opacity-0 group-hover:opacity-100",
		}, dom.Text("Remove")),
	)
}

func TodoList(_ Attrs) *Element {
	todos, setTodos := hooks.UseState([]string{})
	newTodo, setNewTodo := hooks.UseState("")
	currentTodos := todos()
	currentNewTodo := newTodo()

	handleInput := hooks.GoUseFunc(func(event dom.GoEvent) {
		setNewTodo(event.GetValue())
	})

	addTodo := hooks.GoUseFunc(func(event dom.GoEvent) {
		if currentNewTodo != "" {
			newTodos := make([]string, len(currentTodos)+1)
			copy(newTodos, currentTodos)
			newTodos[len(currentTodos)] = currentNewTodo
			setTodos(newTodos)
			setNewTodo("")
		}
	})

	clearAll := hooks.GoUseFunc(func(event dom.GoEvent) {
		setTodos([]string{})
	})

	createRemoveHandler := func(index int) func(dom.GoEvent) {
		return func(e dom.GoEvent) {
			newTodos := make([]string, 0, len(currentTodos)-1)
			newTodos = append(newTodos, currentTodos[:index]...)
			newTodos = append(newTodos, currentTodos[index+1:]...)
			setTodos(newTodos)
		}
	}

	todoItems := make([]interface{}, 0, len(currentTodos))
	for i, todo := range currentTodos {
		todoItems = append(todoItems, dom.CreateElement(TodoItem, Attrs{
			"key":      strconv.Itoa(i),
			"text":     todo,
			"onRemove": createRemoveHandler(i),
		}))
	}

	return dom.Div(Attrs{
		"class": "min-h-screen flex items-center justify-center bg-[#0a0a0a] text-white p-4",
	},
		dom.Div(Attrs{
			"class": "max-w-2xl w-full bg-white/5 border border-white/10 rounded-xl backdrop-blur-sm p-8 shadow-2xl",
		},
			dom.H2(Attrs{
				"class": "text-3xl font-bold mb-8 bg-clip-text text-transparent bg-gradient-to-r from-blue-400 to-purple-500",
			}, dom.Text("Todo List")),

			dom.Div(Attrs{"class": "flex gap-3 mb-8"},
				dom.Input(Attrs{
					"type":        "text",
					"value":       currentNewTodo,
					"oninput":     handleInput,
					"placeholder": "Enter a new todo...",
					"class":       "flex-1 px-4 py-3 bg-black/20 border border-white/10 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-white placeholder-gray-600 transition-all",
				}),
				dom.Button(Attrs{
					"onclick": addTodo,
					"class":   "px-6 py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold rounded-lg hover:opacity-90 transition-opacity shadow-lg shadow-purple-500/20",
				}, dom.Text("Add")),
			),

			dom.Div(Attrs{
				"class": "flex justify-between items-center mb-4 px-1",
			},
				dom.P(Attrs{
					"class": "text-gray-400 text-sm font-medium",
				}, dom.Text(fmt.Sprintf("Tasks: %d", len(currentTodos)))),

				dom.Button(Attrs{
					"onclick": clearAll,
					"class":   "text-xs text-gray-500 hover:text-red-400 transition-colors",
				}, dom.Text("Clear All")),
			),

			dom.Ul(Attrs{
				"class": "space-y-2",
			}, todoItems...),
		),
	)
}

func main() {
	fmt.Println("🚀 Todo Basic Example Started")
	render.To(dom.CreateElement(TodoList, nil), "#app")
	fmt.Println("✅ Todo Basic Example Rendered")
	select {}
}
