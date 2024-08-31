package main

import (
	"fmt"
	html "goHTML/components" // Alias out components to just use the exported functions
	"sync"
	"syscall/js"
	"time"
)

var (
	// Alias out the components package to just use the exported functions
	CreateComponent = html.CreateComponent
	Tag             = html.Tag
	Text            = html.Text
	Render          = html.Render
	Function        = html.Function
)

type (
	Component = html.Component
	Node      = html.NodeInterface
	Props     = html.Props
)

func main() {
	// Create a wait group to keep the main function running
	var wg sync.WaitGroup
	wg.Add(1)

	// Declare the setter in the parent scope
	var setSelectedValueInternal func(string)

	// Create a child component using the CreateComponent technique
	childComponent := CreateComponent(func(c *html.Component, props Props, children ...*html.Component) *html.Component {
		Render(c, Tag("option", map[string]string{"value": "child1"}, Text("Child Option 1")))
		return c
	})

	// Create the main select component using the CreateComponent technique
	selectComponent := CreateComponent(func(c *html.Component, props Props, children ...*html.Component) *html.Component {
		// Extract the initial value from props and cast it to a string
		selectedValue, setter := html.AddState(c, "selectedValue", props["InitialValue"].(string))
		setSelectedValueInternal = setter // Hoist the setter to the parent scope

		handleSelectChange := Function(c, "handleSelectChange", func(event js.Value) {
			fmt.Println("Select value changed:", event.Get("target").Get("value").String())
		})

		// Create the select tag with options
		selectNode := Tag("select", map[string]string{
			"class": "form-select",
			"id":    "exampleSelect",
			"name":  "exampleSelect",
			"onchange": handleSelectChange,
		},
			Tag("option", map[string]string{"value": "option1"}, Text("Option 1")),
			Tag("option", map[string]string{"value": "option2"}, Text("Option 2")),
			Tag("option", map[string]string{"value": *selectedValue}, Text(fmt.Sprintf("Option %s", *selectedValue))),
		)

		// Add child nodes directly to the selectNode
		for _, child := range children {
			if child != nil && child.RootNode != nil {
				selectNode.AddChild(child.RootNode)
			}
		}

		// Render the select component with its children
		Render(c, selectNode)

		return c
	})

	// Define Props with various generic properties
	props := Props{
		"InitialValue":   "test value from props",
		"AdditionalData": "Some other value",
	}

	// Render the main component with a child component
	mainComponent := selectComponent(props, childComponent(Props{}))
	renderedHTML, err := mainComponent.Render() // Initial render
	if err != nil {
		fmt.Println("Error rendering component:", err)
		return
	}

	// Insert the rendered HTML into the document body
	document := js.Global().Get("document")
	document.Get("body").Set("innerHTML", renderedHTML)

	// Create a goroutine to wait 3 seconds and then update the selected value
	go func() {
		time.Sleep(3 * time.Second)             // Wait for 3 seconds
		setSelectedValueInternal("hello world") // Call the setter to update the state

		mainComponent.RenderAndUpdateDom() // Re-render and update the DOM
	}()

	// Wait indefinitely to keep the program running (or you could have other logic to exit)
	wg.Wait()
}
