package components

import (
	"fmt"
	"sync"
	"syscall/js"
)

// Component represents a UI component
type Component struct {
	state           map[string]interface{}
	stateLock       sync.Mutex
	lifecycle       map[string]func()
	rootNode        *Node
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
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	// Check if the state already has a value for the given key
	if existingValue, exists := c.state[key]; exists {
		// Return the existing value and the setter function
		return existingValue.(*T), func(newValue T) {
			c.stateLock.Lock()
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
	self.rootNode = node
}

// InsertComponentIntoDOM inserts the rendered component into the DOM
// InsertComponentIntoDOM inserts the rendered component into the DOM
func InsertComponentIntoDOM(domID string, component *Component) {
	component.updateStateFunc()
	
    if component.rootNode == nil {
        panic("Component rootNode is nil. Ensure RenderTemplate is called before inserting into DOM.")
    }

    rootElement := js.Global().Get("document").Call("getElementById", domID)
    if !rootElement.IsNull() {
        rootElement.Set("innerHTML", component.rootNode.Render())
    }
}
