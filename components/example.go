package components

import (
	"fmt"
	"syscall/js"
)

func Example1() {
    fmt.Println("Starting the example 1")

    // Create the component using MakeComponent
    component := MakeComponent(func(self *Component, props int, children ...*Component) *Component {
        // Setup state for the counter, initializing with the props value
        counter, setCounter := AddState(self, "counter", props)

        // Setup the component when it is mounted
        Setup(self, func() {
            fmt.Println("Component has been set up.")
            fmt.Println("Initial counter value:", *counter)
        })

        // Define the click handler using the Function helper
        handleClick := Function(self, "handleClick", func(event js.Value) {
            // Update the counter state
			fmt.Println("handleClick called")
            setCounter(*counter + 1)
			fmt.Println("Counter updated to:", *counter)
        })

        RenderTemplate(self, Tag("div", Attributes{},
            Tag("p", Attributes{}, Text(fmt.Sprintf("Counter: %d", *counter))),
            Tag("button", Attributes{
                "onclick": handleClick,
                "class":   "btn",
            }, Text("Click me to increment")),
        ))

        return self
    })

    // Use the InsertComponentIntoDOM function to render and insert the component into the DOM
    InsertComponentIntoDOM("root", component(0)) // Initial counter value starts at 0
}
