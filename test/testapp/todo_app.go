//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	rt "github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/state"
)

// Header shows shared atom data (using UseAtom)
func Header(props dom.Attrs) *dom.Element {
	total := state.UseAtom("todoCount", 0)
	theme := state.UseAtom("appTheme", "light")

	toggleTheme := hooks.GoUseFunc(func() {
		if theme.Get() == "light" {
			theme.Set("dark")
		} else {
			theme.Set("light")
		}
	})

	// UseEffect to set body data-theme
	hooks.UseEffect(func() func() {
		js.Global().Get("document").Get("body").Call("setAttribute", "data-theme", theme.Get())
		return nil
	}, theme.Get())

	// Log current fiber pointer for diagnostic purposes
	fmt.Println("Header render total:", total.Get(), "fiber:", rt.GetCurrentFiber())
	// UseEffect to log whenever the total atom changes and the header re-renders
	hooks.UseEffect(func() func() {
		fmt.Println("Header UseEffect total value: ", total.Get())
		return nil
	}, total.Get())
	return dom.Div(dom.Attrs{"class": "flex items-center justify-between"},
		dom.H1(dom.Attrs{"id": "todo-app-heading"}, dom.Text("Todo App")),
		dom.Div(nil,
			dom.Span(dom.Attrs{"id": "todo-header-count"}, dom.Text(fmt.Sprintf("Total Todos: %d", total.Get()))),
			dom.Button(dom.Attrs{"onclick": toggleTheme, "class": "ml-2 px-2 py-1 bg-gray-200"}, dom.Text("Toggle Theme")),
		),
	)
}
