//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"
	rt "github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/state"
)

// Header shows shared atom data (using UseAtom)
func Header(props Attrs) *Element {
	total := state.UseAtom("todoCount", 0)
	theme := state.UseAtom("appTheme", "light")

	toggleTheme := GoUseFunc(func() {
		if theme.Get() == "light" {
			theme.Set("dark")
		} else {
			theme.Set("light")
		}
	})

	// UseEffect to set body data-theme
	UseEffect(func() func() {
		js.Global().Get("document").Get("body").Call("setAttribute", "data-theme", theme.Get())
		return nil
	}, theme.Get())

	// Log current fiber pointer for diagnostic purposes
	fmt.Println("Header render total:", total.Get(), "fiber:", rt.GetCurrentFiber())
	// UseEffect to log whenever the total atom changes and the header re-renders
	UseEffect(func() func() {
		fmt.Println("Header UseEffect total value: ", total.Get())
		return nil
	}, total.Get())
	return Div(Attrs{"class": "flex items-center justify-between"},
		H1(Attrs{"id": "todo-app-heading"}, Text("Todo App")),
		Div(nil,
			Span(Attrs{"id": "todo-header-count"}, Text(fmt.Sprintf("Total Todos: %d", total.Get()))),
			Button(Attrs{"onclick": toggleTheme, "class": "ml-2 px-2 py-1 bg-gray-200"}, Text("Toggle Theme")),
		),
	)
}

