//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/state"
)

type Todo struct {
	ID        int
	Text      string
	Completed bool
}

// TodoList holds and renders a list of todos using UseState and UseAtom to share count
func TodoList(props Attrs) *Element {
	// shared atom for total todos
	total := state.UseAtom("todoCount", 0)

	todos, setTodos := UseState([]Todo{})
	todosVersion, setTodosVersion := UseState(0)
	filter, setFilter := UseState("")
	statusFilter, setStatusFilter := UseState("all")

	// UseMemo for filtered list
	filtered := UseMemo(func() []Todo {
		fmt.Println("UseMemo computing: filtered")
		f := filter()
		s := statusFilter()
		list := []Todo{}
		for _, t := range todos() {
			// Text filter
			matchesText := f == "" || contains(t.Text, f)

			// Status filter
			matchesStatus := true
			if s == "active" {
				matchesStatus = !t.Completed
			} else if s == "completed" {
				matchesStatus = t.Completed
			}

			if matchesText && matchesStatus {
				list = append(list, t)
			}
		}
		return list
	}, len(todos()), todosVersion(), filter(), statusFilter())

	// Ensure total atom syncs
	UseEffect(func() func() {
		curLen := len(todos())
		fmt.Println("SetTotal called, len(todos()) =", curLen)
		total.Set(curLen)
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

	addHandler := GoUseFunc(func() {
		// read input from the input element, since event target may be the button
		val := js.Global().Get("document").Call("getElementById", "todo-input").Get("value").String()
		if val == "" {
			return
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
		total.Set(len(todos()))
		fmt.Println("SetTotal in addHandler:", len(todos()))
	})

	// toggle and remove closures
	toggle := func(id int) func() {
		return func() {
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
		}
	}

	remove := func(id int) func() {
		return func() {
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
			total.Set(newLength)
		}
	}

	// Input change handler for filter
	filterOnChange := GoUseFunc(func(val string) {
		setFilter(val)
	})

	// Status change handler
	statusOnChange := GoUseFunc(func(val string) {
		setStatusFilter(val)
	})

	// build list items
	children := []interface{}{}
	if len(filtered) == 0 {
		children = append(children, P(nil, Text("No todos match the current filters")))
	} else {
		for _, t := range filtered {
			children = append(children, &Element{Type: TodoItem, Props: Attrs{"key": t.ID, "id": t.ID, "text": t.Text, "completed": t.Completed, "toggle": toggle(t.ID), "remove": remove(t.ID)}})
		}
	}

	// Calculate statistics
	activeCount := 0
	completedCount := 0
	for _, t := range todos() {
		if t.Completed {
			completedCount++
		} else {
			activeCount++
		}
	}

	result := Div(Attrs{"id": "todo-list", "class": "mt-6"},
		H2(nil, Text("Todo List")),
		Div(nil,
			Input(Attrs{"id": "todo-input", "type": "text", "class": "border p-2", "placeholder": "What needs to be done?"}),
			Button(Attrs{"id": "todo-add", "onclick": addHandler, "class": "ml-2 px-3 py-2 bg-blue-500 text-white"}, Text("Add Todo")),
		),
		Div(Attrs{"class": "mt-4 flex gap-2"},
			Input(Attrs{"id": "todo-filter", "type": "text", "oninput": filterOnChange, "placeholder": "Filter todos"}),
			Select(Attrs{"id": "status-filter", "onchange": statusOnChange, "class": "border p-2"},
				Option(Attrs{"value": "all"}, Text("All")),
				Option(Attrs{"value": "active"}, Text("Active")),
				Option(Attrs{"value": "completed"}, Text("Completed")),
			),
		),
		Div(Attrs{"id": "todo-items", "class": "mt-4"}, children...),
		Div(Attrs{"class": "mt-6 border-t pt-4"},
			H3(nil, Text("Statistics")),
			P(Attrs{"id": "todo-count"}, Text(fmt.Sprintf("Total: %d", total.Get()))),
			P(nil, Text(fmt.Sprintf("Active: %d", activeCount))),
			P(nil, Text(fmt.Sprintf("Completed: %d", completedCount))),
		),
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
