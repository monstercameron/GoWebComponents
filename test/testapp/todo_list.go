//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/state"
)

type Todo struct {
	ID        int
	Text      string
	Completed bool
}

// TodoList holds and renders a list of todos using UseState and UseAtom to share count
func TodoList(parseProps Attrs) *Element {
	// shared atom for total todos
	parseTotal := state.UseAtom("todoCount", 0)

	parseTodos, setTodos := UseState([]Todo{})
	parseTodosVersion, setTodosVersion := UseState(0)
	filter, setFilter := UseState("")
	parseStatusFilter, setStatusFilter := UseState("all")

	// UseMemo for filtered list
	parseFiltered := UseMemo(func() []Todo {
		fmt.Println("UseMemo computing: filtered")
		parseF := filter()
		parseS := parseStatusFilter()
		parseList := []Todo{}
		for _, parseT := range parseTodos() {
			// Text filter
			isParseMatchesText := parseF == "" || contains(parseT.Text, parseF)

			// Status filter
			isParseMatchesStatus := true
			switch parseS {
			case "active":
				isParseMatchesStatus = !parseT.Completed
			case "completed":
				isParseMatchesStatus = parseT.Completed
			}

			if isParseMatchesText && isParseMatchesStatus {
				parseList = append(parseList, parseT)
			}
		}
		return parseList
	}, len(parseTodos()), parseTodosVersion(), filter(), parseStatusFilter())

	// Ensure total atom syncs
	UseEffect(func() func() {
		parseCurLen := len(parseTodos())
		fmt.Println("SetTotal called, len(todos()) =", parseCurLen)
		parseTotal.Set(parseCurLen)
		return nil
	}, len(parseTodos()))

	// handlers
	// helper to update todos in an immutable way and bump version
	parseUpdateTodos := func(parseUpdater func([]Todo) []Todo) {
		setTodos(func(parsePrev []Todo) []Todo {
			// create new slice to avoid in-place mutations causing subtle issues
			parseNewSlice := make([]Todo, len(parsePrev))
			copy(parseNewSlice, parsePrev)
			parseNewSlice = parseUpdater(parseNewSlice)
			return parseNewSlice
		})
		// bump version to signal memoized computations to re-evaluate
		setTodosVersion(func(parsePrev2 int) int { return parsePrev2 + 1 })
	}

	parseAddHandler := GoUseFunc(func() {
		// read input from the input element, since event target may be the button
		parseVal := js.Global().Get("document").Call("getElementById", "todo-input").Get("value").String()
		if parseVal == "" {
			return
		}
		fmt.Println("AddHandler called, val:", parseVal)
		parseUpdateTodos(func(parsePrev3 []Todo) []Todo {
			parseId := 1
			if len(parsePrev3) > 0 {
				parseId = parsePrev3[len(parsePrev3)-1].ID + 1
			}
			parseNv := append(parsePrev3, Todo{ID: parseId, Text: parseVal, Completed: false})
			// clear input via DOM
			js.Global().Get("document").Call("getElementById", "todo-input").Set("value", "")
			return parseNv
		})
		// Set atom total synchronously for immediate feedback
		parseTotal.Set(len(parseTodos()))
		fmt.Println("SetTotal in addHandler:", len(parseTodos()))
	})

	// toggle and remove closures
	parseToggle := func(parseId2 int) func() {
		return func() {
			parseUpdateTodos(func(parsePrev4 []Todo) []Todo {
				for parseI := range parsePrev4 {
					if parsePrev4[parseI].ID == parseId2 {
						parsePrev4[parseI].Completed = !parsePrev4[parseI].Completed
						fmt.Println("Toggled todo", parseId2, "to", parsePrev4[parseI].Completed)
						break
					}
				}
				return parsePrev4
			})
		}
	}

	parseRemove := func(parseId3 int) func() {
		return func() {
			var parseNewLength int
			parseUpdateTodos(func(parsePrev5 []Todo) []Todo {
				parseNewList := []Todo{}
				for _, parseT2 := range parsePrev5 {
					if parseT2.ID != parseId3 {
						parseNewList = append(parseNewList, parseT2)
					}
				}
				parseNewLength = len(parseNewList)
				return parseNewList
			})
			// Set atom to the new length (not calling len(todos()) which would be stale)
			parseTotal.Set(parseNewLength)
		}
	}

	// Input change handler for filter
	filterOnChange := GoUseFunc(func(parseVal2 string) {
		setFilter(parseVal2)
	})

	// Status change handler
	parseStatusOnChange := GoUseFunc(func(parseVal3 string) {
		setStatusFilter(parseVal3)
	})

	// build list items
	parseChildren := []interface{}{}
	if len(parseFiltered) == 0 {
		parseChildren = append(parseChildren, P(nil, Text("No todos match the current filters")))
	} else {
		for _, parseT3 := range parseFiltered {
			parseChildren = append(parseChildren, &Element{Type: TodoItem, Props: Attrs{"key": parseT3.ID, "id": parseT3.ID, "text": parseT3.Text, "completed": parseT3.Completed, "toggle": parseToggle(parseT3.ID), "remove": parseRemove(parseT3.ID)}})
		}
	}

	// Calculate statistics
	parseActiveCount := 0
	parseCompletedCount := 0
	for _, parseT4 := range parseTodos() {
		if parseT4.Completed {
			parseCompletedCount++
		} else {
			parseActiveCount++
		}
	}

	parseResult := Div(Attrs{"id": "todo-list", "class": "mt-6"},
		H2(nil, Text("Todo List")),
		Div(nil,
			Input(Attrs{"id": "todo-input", "type": "text", "class": "border p-2", "placeholder": "What needs to be done?"}),
			Button(Attrs{"id": "todo-add", "onclick": parseAddHandler, "class": "ml-2 px-3 py-2 bg-blue-500 text-white"}, Text("Add Todo")),
		),
		Div(Attrs{"class": "mt-4 flex gap-2"},
			Input(Attrs{"id": "todo-filter", "type": "text", "oninput": filterOnChange, "placeholder": "Filter todos"}),
			Select(Attrs{"id": "status-filter", "onchange": parseStatusOnChange, "class": "border p-2"},
				Option(Attrs{"value": "all"}, Text("All")),
				Option(Attrs{"value": "active"}, Text("Active")),
				Option(Attrs{"value": "completed"}, Text("Completed")),
			),
		),
		Div(Attrs{"id": "todo-items", "class": "mt-4"}, parseChildren...),
		Div(Attrs{"class": "mt-6 border-t pt-4"},
			H3(nil, Text("Statistics")),
			P(Attrs{"id": "todo-count"}, Text(fmt.Sprintf("Total: %d", parseTotal.Get()))),
			P(nil, Text(fmt.Sprintf("Active: %d", parseActiveCount))),
			P(nil, Text(fmt.Sprintf("Completed: %d", parseCompletedCount))),
		),
	)
	return parseResult
}

func contains(parseS, parseSub string) bool {
	if parseSub == "" {
		return true
	}
	return len(parseS) >= len(parseSub) && (parseS == parseSub || stringContains(parseS, parseSub))
}

func stringContains(parseS, parseSub string) bool {
	for parseI := 0; parseI+len(parseSub) <= len(parseS); parseI++ {
		if parseS[parseI:parseI+len(parseSub)] == parseSub {
			return true
		}
	}
	return false
}
