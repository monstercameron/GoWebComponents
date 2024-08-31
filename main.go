package main

import (
	"fmt"
	. "goHTML/components" // Alias out components to just use the exported functions
)

func main() {
	// Child component using the CreateComponent technique
	childComponent := CreateComponent(func(c *Component, props Props, children ...*Component) *Component {
		Render(c, Tag("option", map[string]string{"value": "child1"}, Text("Child Option 1")))
		return c
	})

	// Main select component using the CreateComponent technique
	selectComponent := CreateComponent(func(c *Component, props Props, children ...*Component) *Component {
		// Define state to keep track of the selected option
		selectedValue, setSelectedValue := AddState(c, props.InitialValue)

		// Create the select tag with options and any children passed in
		Render(c, Tag("select", map[string]string{
			"class": "form-select",
			"id":    "exampleSelect",
			"name":  "exampleSelect",
		},
			Tag("option", map[string]string{"value": "option1"}, Text("Option 1")),
			Tag("option", map[string]string{"value": "option2"}, Text("Option 2")),
			Tag("option", map[string]string{"value": *selectedValue}, Text(fmt.Sprintf("Option %s", *selectedValue))),
		))

		// Simulate selecting "updated value from setSelectedValue"
		setSelectedValue("updated value from setSelectedValue")

		return c
	})

	// Props to pass to the select component
	props := Props{
		InitialValue: "test value from props",
	}

	// Render the main component with a child component and print the result
	renderedHTML, err := selectComponent(props, childComponent(Props{})).Render() // Pass empty Props to childComponent
	if err != nil {
		fmt.Println("Error rendering component:", err)
		return
	}
	fmt.Println(renderedHTML)
}
