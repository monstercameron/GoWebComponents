//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	rt "github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/state"
)

// Header shows shared atom data (using UseAtom)
func Header(parseProps Attrs) *Element {
	parseTotal := state.UseAtom("todoCount", 0)
	parseTheme := state.UseAtom("appTheme", "light")

	parseToggleTheme := GoUseFunc(func() {
		if parseTheme.Get() == "light" {
			parseTheme.Set("dark")
		} else {
			parseTheme.Set("light")
		}
	})

	// UseEffect to set body data-theme
	UseEffect(func() func() {
		js.Global().Get("document").Get("body").Call("setAttribute", "data-theme", parseTheme.Get())
		return nil
	}, parseTheme.Get())

	// Log current fiber pointer for diagnostic purposes
	fmt.Println("Header render total:", parseTotal.Get(), "fiber:", rt.GetCurrentFiber())
	// UseEffect to log whenever the total atom changes and the header re-renders
	UseEffect(func() func() {
		fmt.Println("Header UseEffect total value: ", parseTotal.Get())
		return nil
	}, parseTotal.Get())
	return Div(Attrs{"class": "flex items-center justify-between"},
		H1(Attrs{"id": "todo-app-heading"}, Text("Todo App")),
		Div(nil,
			Span(Attrs{"id": "todo-header-count"}, Text(fmt.Sprintf("Total Todos: %d", parseTotal.Get()))),
			Button(Attrs{"onclick": parseToggleTheme, "class": "ml-2 px-2 py-1 bg-gray-200"}, Text("Toggle Theme")),
		),
	)
}
