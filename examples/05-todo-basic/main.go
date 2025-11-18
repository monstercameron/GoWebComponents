//go:build js && wasm

package main

import (
	"fmt"
	"strconv"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

func TodoList(_ Attrs) *Element {
	todos, setTodos := hooks.UseState([]string{})
	newTodo, setNewTodo := hooks.UseState("")
	currentTodos := todos()
	currentNewTodo := newTodo()

	addTodo := func(this js.Value, args []js.Value) interface{} {
		todoText := currentNewTodo
		if todoText != "" {
			newTodos := make([]string, len(currentTodos)+1)
			copy(newTodos, currentTodos)
			newTodos[len(currentTodos)] = todoText
			setTodos(newTodos)
			setNewTodo("")
		}
		return nil
	}

	removeTodo := func(index int) js.Func {
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if index >= 0 && index < len(currentTodos) {
				newTodos := make([]string, 0, len(currentTodos)-1)
				for i, todo := range currentTodos {
					if i != index {
						newTodos = append(newTodos, todo)
					}
				}
				setTodos(newTodos)
			}
			return nil
		})
	}

	handleInput := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newText := args[0].Get("target").Get("value").String()
			setNewTodo(newText)
		}
		return nil
	}

	clearAll := func(this js.Value, args []js.Value) interface{} {
		setTodos([]string{})
		return nil
	}

	// Create todo items
	todoItems := make([]interface{}, 0, len(currentTodos))
	for i, todo := range currentTodos {
		todoItems = append(todoItems, dom.Li(Attrs{
			"key":   strconv.Itoa(i),
			"class": "flex items-center justify-between p-3 bg-gray-50 rounded-lg mb-2",
		},
			dom.Span(Attrs{
				"class": "text-gray-800",
			}, dom.Text(todo)),
			dom.Button(Attrs{
				"onclick": removeTodo(i),
				"class":   "px-3 py-1 bg-red-500 text-white rounded hover:bg-red-600 transition-colors text-sm",
			}, dom.Text("Remove")),
		))
	}

	return dom.Div(Attrs{
		"class": "max-w-2xl mx-auto mt-8 p-6 bg-white rounded-lg shadow-lg",
	},
		dom.H2(Attrs{
			"class": "text-2xl font-bold mb-6 text-gray-800",
		}, dom.Text("Todo List")),

		dom.Div(Attrs{"class": "flex gap-2 mb-4"},
			dom.Input(Attrs{
				"type":        "text",
				"value":       currentNewTodo,
				"oninput":     js.FuncOf(handleInput),
				"placeholder": "Enter a new todo...",
				"class":       "flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500",
			}),
			dom.Button(Attrs{
				"onclick": js.FuncOf(addTodo),
				"class":   "px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors",
			}, dom.Text("Add")),
			dom.Button(Attrs{
				"onclick": js.FuncOf(clearAll),
				"class":   "px-6 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors",
			}, dom.Text("Clear All")),
		),

		dom.P(Attrs{
			"class": "mb-4 text-gray-600 font-medium",
		}, dom.Text(fmt.Sprintf("Total todos: %d", len(currentTodos)))),

		dom.Ul(Attrs{
			"class": "space-y-2",
		}, todoItems...),
	)
}

func main() {
	container := js.Global().Get("document").Call("getElementById", "app")
	element := dom.CreateElement(TodoList, nil)
	render.ToElement(element, container)
	select {}
}
