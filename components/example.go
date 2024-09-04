package components

import (
	"fmt"
)

func Example1() {
	var setCounterHoisted func(int)

	// example1 is a simple example of a component that increments a counter when a button is clicked.
	component := ThisIsAComponent(func(self *Component, props int, children ...*Component) *Component {
		// Setup state
		counter, setCounter := AddState(self, "counter", props)
		setCounterHoisted = setCounter

		// Setup the component when it is mounted
		Setup(self, func() {
			fmt.Println("Component has been set up.")
			fmt.Println("Initial counter value:", *counter)
		})

		// Watch for changes in the counter state
		// Watch(self, func() {
		//     fmt.Println("Counter value changed:", *counter)
		// }, "counter")

		// Example of a cached value that depends on the counter state
		cachedValue := Cached(self, "cachedValue", func() interface{} {
			return fmt.Sprintf("Computed Value: %d", *counter*2)
		}, []string{"counter"})

		fmt.Println("Cached value:", cachedValue)

		// Cleanup logic when the component is unmounted
		Cleanup(self, func() {
			fmt.Println("Component is being cleaned up.")
		})

		// Render the HTML structure
		RenderTemplate(self, Tag("select", Attributes{"class": "dropdown"},
			Tag("option", Attributes{"value": "1"}, Text("Option 1")),
			Tag("option", Attributes{"value": "2"}, Text("Option 2")),
			Tag("option", Attributes{"value": "3"}, Text(fmt.Sprintf("Option %d", *counter))),
		))

		return self
	})

	myComponent := component(111)
	myComponent.updateStateFunc()
	PrintNodeTree(myComponent.rootNode)

	setCounterHoisted(777)
	myComponent.updateStateFunc()
	PrintNodeTree(myComponent.rootNode)
}

func Example3() {
	// Create a simple select dropdown with options
	selectNode := Tag("select", Attributes{"class": "dropdown"},
		Tag("option", Attributes{"value": "1"}, Text("Option 1")),
		Tag("option", Attributes{"value": "2"}, Text("Option 2")),
		Tag("option", Attributes{"value": "3"}, Text("Option 3")),
	)

	PrintNodeTree(selectNode)
}

func Example4() {
	// Create a complex HTML structure with void tags and mixed content
	htmlStructure := Tag("div", Attributes{"class": "container"},
		Text("Welcome to the site! "),
		Tag("img", Attributes{"src": "logo.png", "alt": "Site Logo"}), // Void tag
		Tag("p", Attributes{"class": "description"},
			Text("This is an example of "),
			Tag("a", Attributes{"href": "https://example.com"}, Text("a link")),
			Text(" with some "),
			Tag("strong", Attributes{}, Text("bold text")),
			Text(" and an inline image."),
		),
		Tag("input", Attributes{"type": "text", "placeholder": "Enter text here"}), // Void tag
		Tag("div", Attributes{"class": "footer"},
			Text("Thank you for visiting!"),
			Tag("br", Attributes{}), // Void tag
			Text("We hope you enjoy your stay."),
		),
	)

	PrintNodeTree(htmlStructure)
}
