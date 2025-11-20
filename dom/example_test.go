//go:build js && wasm
// +build js,wasm

package dom_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
)

func ExampleDiv() {
	// Create a simple div with text content
	element := dom.Div(
		dom.Attrs{"class": "container"},
		dom.H1(nil, dom.Text("Hello World")),
		dom.P(nil, dom.Text("This is a paragraph")),
	)

	// In a real application, this element would be rendered to the DOM
	// using render.ToElement(element, container)
	fmt.Printf("Created element: %v\n", element != nil)
	// Output: Created element: true
}

func ExampleButton() {
	// Create a button with an onclick handler
	// Note: In a real app, you would use hooks.GoUseFunc for the handler
	btn := dom.Button(
		dom.Attrs{
			"class": "btn btn-primary",
			"type":  "button",
		},
		dom.Text("Click Me"),
	)

	fmt.Printf("Created button: %v\n", btn != nil)
	// Output: Created button: true
}

func ExampleForm() {
	// Create a login form
	form := dom.Form(
		dom.Attrs{"class": "login-form"},
		dom.Div(
			dom.Attrs{"class": "form-group"},
			dom.Label(dom.Attrs{"for": "username"}, dom.Text("Username")),
			dom.Input(dom.Attrs{
				"type": "text",
				"id":   "username",
				"name": "username",
			}),
		),
		dom.Div(
			dom.Attrs{"class": "form-group"},
			dom.Label(dom.Attrs{"for": "password"}, dom.Text("Password")),
			dom.Input(dom.Attrs{
				"type": "password",
				"id":   "password",
				"name": "password",
			}),
		),
		dom.Button(
			dom.Attrs{"type": "submit"},
			dom.Text("Login"),
		),
	)

	fmt.Printf("Created form: %v\n", form != nil)
	// Output: Created form: true
}

func ExampleUl() {
	// Create an unordered list
	list := dom.Ul(
		dom.Attrs{"class": "item-list"},
		dom.Li(nil, dom.Text("Item 1")),
		dom.Li(nil, dom.Text("Item 2")),
		dom.Li(nil, dom.Text("Item 3")),
	)

	fmt.Printf("Created list: %v\n", list != nil)
	// Output: Created list: true
}
