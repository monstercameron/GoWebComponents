package main

import (
	"fmt"
	. "goHTML/components" // Alias out components to just use the exported functions
)

func main() {
	var setSelectedValue func(string) // Declare the setter outside of the function

	// Child component using the CreateComponent technique
	childComponent := CreateComponent(func(c *Component, props Props, children ...*Component) *Component {
		Render(c, Tag("option", map[string]string{"value": "child1"}, Text("Child Option 1")))
		return c
	})

	// Main select component using the CreateComponent technique
	selectComponent := CreateComponent(func(c *Component, props Props, children ...*Component) *Component {
		// Extract the initial value from props and cast it to a string
		selectedValue, setSelectedValueInternal := AddState(c, "selectedValue", props["InitialValue"].(string))
		setSelectedValue = setSelectedValueInternal // Hoist the setter outside

		fmt.Printf("Selected value: %s\n", *selectedValue)

		// Create the select tag with options
		selectNode := Tag("select", map[string]string{
			"class": "form-select",
			"id":    "exampleSelect",
			"name":  "exampleSelect",
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
		"InitialValue":    "test value from props",
		"AdditionalData":  "Some other value",
	}

	// Render the main component with a child component and print the result
	mainComponent := selectComponent(props, childComponent(Props{}))
	renderedHTML, err := mainComponent.Render() // Initial render
	if err != nil {
		fmt.Println("Error rendering component:", err)
		return
	}
	fmt.Println("Initial Rendered HTML:")
	fmt.Println(renderedHTML)

	// Update the state one more time and render the component again
	fmt.Println("Setting value to: hello world")
	setSelectedValue("hello world")
	renderedHTML, err = mainComponent.Render() // Re-render after updating state
	if err != nil {
		fmt.Println("Error rendering component:", err)
		return
	}
	fmt.Println("Updated Rendered HTML:")
	fmt.Println(renderedHTML)
}
