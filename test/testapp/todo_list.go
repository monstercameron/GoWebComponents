//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/state"
)

type Todo struct {
	ID        int
	Text      string
	Completed bool
}

// TodoList holds and renders a list of todos using UseState and UseAtom to share count
func TodoList(props dom.Attrs) *dom.Element {
	// shared atom for total todos
	getTotal, setTotal := state.UseAtom("todoCount", 0)

	todos, setTodos := hooks.UseState([]Todo{})
	todosVersion, setTodosVersion := hooks.UseState(0)
	filter, setFilter := hooks.UseState("")

	// UseMemo for filtered list
	filtered := hooks.UseMemo(func() interface{} {
		fmt.Println("UseMemo computing: filtered")
		f := filter()
		list := []Todo{}
		for _, t := range todos() {
			if f == "" || contains(t.Text, f) {
				list = append(list, t)
			}
		}
		return list
	}, len(todos()), todosVersion(), filter()).([]Todo)

	// Ensure total atom syncs
	hooks.UseEffect(func() func() {
		curLen := len(todos())
		fmt.Println("SetTotal called, len(todos()) =", curLen)
		setTotal(curLen)
		return nil
	}, len(todos()))

	// handlers
	// helper to update todos in an immutable way and bump version
	updateTodos := func(updater func([]Todo) []Todo) {
		setTodos(func(prev []Todo) []Todo {
			// create new slice to avoid in-place mutations causing subtle issues
			newSlice := make([]Todo, len(prev))
			copy(newSlice, prev)
			newSlice = updater(newSlice)
			return newSlice
		})
		// bump version to signal memoized computations to re-evaluate
		setTodosVersion(func(prev int) int { return prev + 1 })
	}

	addHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// read input from the input element, since event target may be the button
		val := js.Global().Get("document").Call("getElementById", "todo-input").Get("value").String()
		if val == "" {
			return nil
		}
		fmt.Println("AddHandler called, val:", val)
		updateTodos(func(prev []Todo) []Todo {
			id := 1
			if len(prev) > 0 {
				id = prev[len(prev)-1].ID + 1
			}
			nv := append(prev, Todo{ID: id, Text: val, Completed: false})
			// clear input via DOM
			js.Global().Get("document").Call("getElementById", "todo-input").Set("value", "")
			return nv
		})
		// Set atom total synchronously for immediate feedback
		setTotal(len(todos()))
		fmt.Println("SetTotal in addHandler:", len(todos()))
		return nil
	})

	// toggle and remove closures
	toggle := func(id int) js.Func {
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			updateTodos(func(prev []Todo) []Todo {
				for i := range prev {
					if prev[i].ID == id {
						prev[i].Completed = !prev[i].Completed
						fmt.Println("Toggled todo", id, "to", prev[i].Completed)
						break
					}
				}
				return prev
			})
			return nil
		})
	}

	remove := func(id int) js.Func {
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			var newLength int
			updateTodos(func(prev []Todo) []Todo {
				newList := []Todo{}
				for _, t := range prev {
					if t.ID != id {
						newList = append(newList, t)
					}
				}
				newLength = len(newList)
				return newList
			})
			// Set atom to the new length (not calling len(todos()) which would be stale)
			setTotal(newLength)
			return nil
		})
	}

	// Input change handler for filter
	filterOnChange := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			val := args[0].Get("target").Get("value").String()
			setFilter(val)
		}
		return nil
	})

	// build list items
	children := []interface{}{}
	for _, t := range filtered {
		children = append(children, &dom.Element{Type: TodoItem, Props: dom.Attrs{"id": t.ID, "text": t.Text, "completed": t.Completed, "toggle": toggle(t.ID), "remove": remove(t.ID)}})
	}

	result := dom.Div(dom.Attrs{"id": "todo-list", "class": "mt-6"},
		dom.H2(nil, dom.Text("Todo List")),
		dom.Div(nil,
			dom.Input(dom.Attrs{"id": "todo-input", "type": "text", "class": "border p-2"}),
			dom.Button(dom.Attrs{"id": "todo-add", "onclick": addHandler, "class": "ml-2 px-3 py-2 bg-blue-500 text-white"}, dom.Text("Add")),
		),
		dom.Div(dom.Attrs{"class": "mt-4"},
			dom.Input(dom.Attrs{"id": "todo-filter", "type": "text", "onchange": filterOnChange, "placeholder": "Filter todos"}),
		),
		dom.Div(dom.Attrs{"id": "todo-items", "class": "mt-4"}, children...),
		dom.P(dom.Attrs{"id": "todo-count"}, dom.Text(fmt.Sprintf("Total: %d", getTotal()))),
	)
	return result
}

func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	return len(s) >= len(sub) && (s == sub || stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
