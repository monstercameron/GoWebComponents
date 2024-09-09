package components

import (
	"fmt"
	"sync"
	"syscall/js"
)

// Component represents a UI component
type Component struct {
	previousState   map[string]interface{}
	state           map[string]interface{}
	stateLock       sync.Mutex
	lifecycle       map[string]func()
	rootNode        *Node
	proposedNode    *Node
	updateStateFunc func()
	setupDone       bool
	registered      bool
	cachedValues    map[string]map[string]interface{} // To store the cached values and their dependencies
}

// NewComponent creates a new Component with a given root Node
func NewComponent(root *Node) *Component {
	return &Component{
		state:        make(map[string]interface{}),
		lifecycle:    make(map[string]func()),
		rootNode:     root,
		setupDone:    false,
		registered:   false,
		cachedValues: make(map[string]map[string]interface{}), // Initialize the cached values map
	}
}

func AddState[T any](c *Component, key string, initialValue T) (*T, func(T)) {
	// Initialize previousState map if it's nil
	if c.previousState == nil {
		c.previousState = make(map[string]interface{})
	}
	
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	// Check if the state already has a value for the given key
	if existingValue, exists := c.state[key]; exists {
		// Return the existing value and the setter function
		return existingValue.(*T), func(newValue T) {
			c.stateLock.Lock()
			c.previousState[key] = *existingValue.(*T)
			*(c.state[key].(*T)) = newValue
			c.stateLock.Unlock()

			// Trigger the re-render and DOM update
			c.updateStateFunc()
		}
	}

	// If the key doesn't exist, initialize it with the initial value
	value := initialValue
	c.state[key] = &value

	// Return the newly created value and the setter function
	return &value, func(newValue T) {
		c.stateLock.Lock()
		c.previousState[key] = value
		*(c.state[key].(*T)) = newValue
		c.stateLock.Unlock()

		// Trigger the re-render and DOM update
		c.updateStateFunc()
	}
}

// Setup registers a lifecycle function to run when the component is mounted
func Setup(self *Component, fn func()) {
	self.lifecycle["setup"] = fn
}

// Function registers a JavaScript event handler and returns its call signature
func Function(c *Component, id string, fn func(js.Value)) string {
	js.Global().Set(id, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			fn(args[0])
		}
		return nil
	}))
	return id + "(event)"
}

// MakeComponent handles the creation of components, taking generic props and child components
func MakeComponent[P any](f func(*Component, P, ...*Component) *Component) func(P, ...*Component) *Component {
	return func(props P, children ...*Component) *Component {
		self := &Component{
			state:        make(map[string]interface{}),
			lifecycle:    make(map[string]func()),
			cachedValues: make(map[string]map[string]interface{}),
		}

		self.updateStateFunc = func() {
			// Call the component's render function
			f(self, props, children...)

			if !self.setupDone {
				// Run the setup lifecycle function
				if setup, exists := self.lifecycle["setup"]; exists {
					setup()
				}

				self.setupDone = true
			}

			// Update the DOM after rendering
			if self.rootNode != nil {
				fmt.Println("Updating DOM")
				UpdateDOM(self)
			}
		}

		return self
	}
}

// RenderTemplate renders the component's HTML structure
func RenderTemplate(self *Component, node *Node) {
	self.proposedNode = node
}

// InsertComponentIntoDOM inserts the rendered component into the DOM
func InsertComponentIntoDOM(component *Component) {
	component.updateStateFunc()
	UpdateDOM(component)
}

func Watch(self *Component, callback func(), deps ...string) {
	// Placeholder map to simulate dependency tracking
	// This would track the previous state of the dependencies (in real-world cases, this could be part of the state system)
	previousValues := make(map[string]interface{})

	// Iterate over the dependencies
	for _, dep := range deps {
		// Get the current value of the dependency
		currentValue, exists := self.state[dep]
		if !exists {
			fmt.Printf("Dependency %s does not exist in state.\n", dep)
			continue
		}

		// Check if the dependency has a previous value stored
		prevValue, hasPrev := self.previousState[dep]

		// If the value has changed, execute the callback function
		if !hasPrev || currentValue != prevValue {
			fmt.Printf("Dependency %s has changed, calling the callback.\n", dep)
			callback()

			// Update the previous value with the current one
			previousValues[dep] = currentValue
		} else {
			fmt.Printf("No change detected for dependency %s.\n", dep)
		}
	}
}
