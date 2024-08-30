package vdom

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Component represents a reusable UI component
type Component struct {
	stateValues  map[string]interface{}
	stateSetters map[string]func(interface{})
	node         NodeInterface
	parent       *Component
	mu           sync.RWMutex
}

// NewComponent creates a new component
func NewComponent() *Component {
	return &Component{
		stateValues:  make(map[string]interface{}),
		stateSetters: make(map[string]func(interface{})),
	}
}

// AddState adds a new state to the component
func AddState[T any](c *Component, initialValue T) (*T, func(T)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("state_%d", len(c.stateValues))

	value := new(T)
	*value = initialValue

	c.stateValues[key] = value

	setter := func(newValue T) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if !reflect.DeepEqual(*value, newValue) {
			*value = newValue
			c.update()
		}
	}

	c.stateSetters[key] = func(i interface{}) {
		if v, ok := i.(T); ok {
			setter(v)
		}
	}

	return value, setter
}

// Render sets the node for the component
func Render(c *Component, node NodeInterface) {
	c.node = node
	c.update()
}

// update updates the component and its parent
func (c *Component) update() {
	if c.parent != nil {
		c.parent.update()
	}
}

func nodesToInterfaces(nodes []NodeInterface) []interface{} {
	interfaces := make([]interface{}, len(nodes))
	for i, node := range nodes {
		interfaces[i] = node
	}
	return interfaces
}

// Example usage
func ExampleUsage() {
	// Counter component
	type CounterProps struct {
		InitialCount int
	}

	Counter := func(props CounterProps) *Component {
		component := NewComponent()
		count, _ := AddState(component, props.InitialCount)

		Render(component, Tag("div", nil,
			Tag("p", nil, Text(fmt.Sprintf("Count: %d", *count))),
			Tag("button", map[string]string{"onClick": "increment"}, Text("Increment")),
		))

		return component
	}

	// Usage of Counter component
	counterComponent := Counter(CounterProps{InitialCount: 0})

	// Initial render
	fmt.Println("Initial state:")
	counterComponent.node.PrintTree(0)

	// Greeting component
	type GreetingProps struct {
		Name string
	}

	Greeting := func(props GreetingProps) *Component {
		component := NewComponent()
		greeting, _ := AddState(component, fmt.Sprintf("Hello, %s!", props.Name))

		Render(component, Tag("div", nil,
			Tag("p", nil, Text(*greeting)),
			Tag("button", map[string]string{"onClick": "changeGreeting"}, Text("Change Greeting")),
		))

		return component
	}

	// Usage of Greeting component
	greetingComponent := Greeting(GreetingProps{Name: "World"})

	// Initial render
	fmt.Println("\nInitial greeting:")
	greetingComponent.node.PrintTree(0)
}

func ExampleUsage2() {
	// ... [Previous example remains unchanged] ...

	// Example 2: Composable Select and Container components
	fmt.Println("\nExample 2: Composable Components")

	// Select component
	type SelectProps struct {
		Options []string
	}

	Select := func(props SelectProps) *Component {
		component := NewComponent()

		optionNodes := make([]NodeInterface, len(props.Options))
		for i, option := range props.Options {
			optionNodes[i] = Tag("option", map[string]string{"value": option}, Text(option))
		}

		Render(component, Tag("select", nil, nodesToInterfaces(optionNodes)...))

		return component
	}

	// Container component
	type ContainerProps struct {
		Title    string
		Children []NodeInterface
	}

	Container := func(props ContainerProps) *Component {
		component := NewComponent()

		children := append(
			[]NodeInterface{Tag("h2", nil, Text(props.Title))},
			props.Children...,
		)

		Render(component, Tag("div", map[string]string{"class": "container"}, nodesToInterfaces(children)...))

		return component
	}

	// Usage of composable components
	fruits := []string{"Apple", "Banana", "Cherry", "Date"}
	colors := []string{"Red", "Green", "Blue", "Yellow"}

	fruitSelect := Select(SelectProps{Options: fruits})
	colorSelect := Select(SelectProps{Options: colors})

	container := Container(ContainerProps{
		Title: "My Selects",
		Children: []NodeInterface{
			fruitSelect.node,
			colorSelect.node,
		},
	})

	// Render the composed components
	fmt.Println("Composed components structure:")
	container.node.PrintTree(0)

	// Render the actual HTML
	fmt.Println("\nRendered HTML:")
	fmt.Println(container.node.Render(0))
}

// Concise example usage
func ExampleUsage3() {
	fmt.Println("Concise Example: Composable Components")

	// Concise Select component
	Select := func(options []string) *Component {
		c := NewComponent()
		Render(c, Tag("select", nil, nodesToInterfaces(
			Map(options, func(opt string) NodeInterface {
				return Tag("option", map[string]string{"value": opt}, Text(opt))
			}))...))
		return c
	}

	// Concise Container component
	Container := func(title string, children ...NodeInterface) *Component {
		c := NewComponent()
		Render(c, Tag("div", map[string]string{"class": "container"},
			append([]interface{}{Tag("h2", nil, Text(title))},
				nodesToInterfaces(children)...)...))
		return c
	}

	// Usage of concise composable components
	container := Container(
		"My Selects",
		Select([]string{"Apple", "Banana", "Cherry"}).node,
		Select([]string{"Red", "Green", "Blue"}).node,
	)

	// Render the composed components
	fmt.Println("Composed components structure:")
	container.node.PrintTree(0)

	fmt.Println("\nRendered HTML:")
	fmt.Println(container.node.Render(0))
}

// DynamicNode represents a node with dynamic content
type DynamicNode struct {
	getContent func() NodeInterface
}

// Implement NodeInterface methods for DynamicNode
func (d *DynamicNode) SetValue(value interface{}) {}
func (d *DynamicNode) GetValue() interface{}      { return nil }
func (d *DynamicNode) SetTagName(tagName string) error { return nil }
func (d *DynamicNode) GetTagName() string              { return "" }
func (d *DynamicNode) SetAttribute(key, value string) error { return nil }
func (d *DynamicNode) GetAttributes() map[string]string     { return nil }
func (d *DynamicNode) AddChild(child NodeInterface)         {}
func (d *DynamicNode) GetChildren() []NodeInterface         { return nil }
func (d *DynamicNode) FindByID(id interface{}) (NodeInterface, error) { return nil, nil }
func (d *DynamicNode) PrintTree(level int) {
	d.getContent().PrintTree(level)
}
func (d *DynamicNode) Render(level int) string {
	return d.getContent().Render(level)
}

func ExampleUsage4() {
	fmt.Println("Concise Example: Select with State Management")

	// Concise Select component with state
	Select := func(props struct{ Options []string }) *Component {
		c := NewComponent()
		options, setOptions := AddState(c, props.Options)

		Render(c, &DynamicNode{
			getContent: func() NodeInterface {
				return Tag("select", nil, nodesToInterfaces(
					Map(*options, func(opt string) NodeInterface {
						return Tag("option", map[string]string{"value": opt}, Text(opt))
					}))...)
			},
		})

		// Simulate an update after 1 second (for demonstration purposes)
		go func() {
			time.Sleep(time.Second)
			newOptions := append(*options, "New Option")
			setOptions(newOptions)
		}()

		return c
	}

	// Usage of the Select component
	selectComponent := Select(struct{ Options []string }{
		Options: []string{"Apple", "Banana", "Cherry"},
	})

	// Initial render
	fmt.Println("Initial render:")
	selectComponent.node.PrintTree(0)

	// Wait for the update
	time.Sleep(time.Second * 2)

	// Render after update
	fmt.Println("\nRender after update:")
	selectComponent.node.PrintTree(0)
}