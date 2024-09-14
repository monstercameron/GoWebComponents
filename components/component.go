package components

import (
	"fmt"
	"sync"
	"syscall/js"
)

// Component represents a UI component.
// It includes state management, lifecycle hooks, caching, and DOM manipulation functionality.
type Component struct {
	previousState   map[string]interface{}         // Tracks the previous state to detect changes.
	state           map[string]interface{}         // Holds the current state of the component.
	stateLock       sync.Mutex                     // Synchronizes state access across multiple goroutines.
	lifecycle       map[string]func()              // Stores lifecycle functions (e.g., setup).
	rootNode        *Node                          // The root node of the component in the virtual DOM.
	proposedNode    *Node                          // The proposed node to render in the virtual DOM.
	updateStateFunc func()                         // Function to trigger re-rendering and state updates.
	setupDone       bool                           // Tracks whether the setup function has been run.
	registered      bool                           // Tracks whether the component is registered in the DOM.
	cachedValues    map[string]map[string]interface{} // Stores cached values and dependencies.
}

// NewComponent creates and initializes a new Component with a given root Node.
// It initializes state, lifecycle hooks, and cached values.
func NewComponent(root *Node) *Component {
	return &Component{
		state:        make(map[string]interface{}),
		lifecycle:    make(map[string]func()),
		rootNode:     root,
		setupDone:    false,
		registered:   false,
		cachedValues: make(map[string]map[string]interface{}), // Initialize the cached values map.
	}
}

// AddState adds a state variable to the component and provides a getter and setter function for the state.
// It takes a generic type T for flexibility in the type of state managed.
func AddState[T any](c *Component, key string, initialValue T) (*T, func(T)) {
	// Initialize previousState map if it's nil.
	if c.previousState == nil {
		c.previousState = make(map[string]interface{})
	}

	// Lock the state to ensure thread-safe access.
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	// Check if the state already has a value for the given key.
	if existingValue, exists := c.state[key]; exists {
		// Return the existing value and the setter function.
		return existingValue.(*T), func(newValue T) {
			c.stateLock.Lock()
			c.previousState[key] = *existingValue.(*T) // Store the previous value before changing it.
			*(c.state[key].(*T)) = newValue            // Update the current state.
			c.stateLock.Unlock()

			// Trigger the re-render and DOM update.
			c.updateStateFunc()
		}
	}

	// If the key doesn't exist, initialize it with the initial value.
	value := initialValue
	c.state[key] = &value

	// Return the newly created value and the setter function.
	return &value, func(newValue T) {
		c.stateLock.Lock()
		c.previousState[key] = value            // Store the previous value.
		*(c.state[key].(*T)) = newValue         // Update the current state.
		c.stateLock.Unlock()

		// Trigger the re-render and DOM update.
		c.updateStateFunc()
	}
}

func (t *Component) RenderNode() *Node {
    return t.rootNode
}

// Setup registers a lifecycle function to run when the component is mounted.
// It ensures the function is only run once, the first time the component is mounted.
func Setup(self *Component, fn func()) {
	self.lifecycle["setup"] = fn // Store the setup function in the lifecycle map.
	// If the component is not yet mounted, run the setup function and mark it as done.
	if !self.setupDone {
		self.setupDone = true
		fn()
	}
}

// Function registers a JavaScript event handler and returns its call signature.
// It allows us to bind Go functions to JavaScript events in WebAssembly.
func Function(c *Component, id string, fn func(js.Value)) string {
	// Register the event handler in the global JavaScript environment.
	js.Global().Set(id, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Check if there are arguments, and if so, pass the first one to the Go callback function.
		if len(args) > 0 {
			fn(args[0])
		}
		return nil
	}))
	// Return the JavaScript call signature for the event (e.g., "click(event)").
	return id + "(event)"
}

// MakeComponent creates and initializes a new Component with generic props and child components.
// It sets up state management, lifecycle functions, and rendering logic.
func MakeComponent[P any](f func(*Component, P, ...*Component)) func(P, ...*Component) *Component {
    return func(props P, children ...*Component) *Component {
        var self *Component

        // Check if a Component instance already exists for this component
        // You might need a registry or a map to keep track of existing instances
        // For simplicity, let's assume a singleton component

        // Initialize the Component only if it's nil
        if self == nil {
            self = &Component{
                state:        make(map[string]interface{}),
                lifecycle:    make(map[string]func()),
                cachedValues: make(map[string]map[string]interface{}),
                setupDone:    false,
                registered:   false,
            }
        }

        // Define the function to update the component's state and re-render.
        self.updateStateFunc = func() {
            // Call the component's render function.
            f(self, props, children...)

            // Update the DOM after rendering.
            if self.rootNode != nil {
                fmt.Println("Updating DOM")
                UpdateDOM(self)
            }
        }

        // Initial render.
        self.updateStateFunc()

        return self
    }
}



// RenderTemplate sets the proposedNode to the passed-in node for future rendering.
// This is where the component's HTML structure is defined.
func RenderTemplate(self *Component, node *Node) {
	self.proposedNode = node
}

// InsertComponentIntoDOM triggers the initial rendering of the component and inserts it into the DOM.
// It calls the component's updateStateFunc to perform the rendering and then updates the DOM.
func InsertComponentIntoDOM(component *Component) {
	component.updateStateFunc() // Trigger the component's render process.
	UpdateDOM(component)        // Perform the actual DOM update.
}

// Watch listens for changes in specified state dependencies and triggers a callback if any of them change.
// It takes a component, a callback function, and a list of state keys to watch for changes.
func Watch(self *Component, callback func(), deps ...string) {
	// Placeholder map to simulate dependency tracking.
	previousValues := make(map[string]interface{})

	// Iterate over the dependencies.
	for _, dep := range deps {
		// Get the current value of the dependency from the component's state.
		currentValue, exists := self.state[dep]
		if !exists {
			fmt.Printf("Dependency %s does not exist in state.\n", dep)
			continue
		}

		// Check if the dependency has a previous value stored.
		prevValue, hasPrev := self.previousState[dep]

		// If the value has changed, execute the callback function.
		if !hasPrev || currentValue != prevValue {
			callback()                    // Trigger the callback for the changed dependency.
			previousValues[dep] = currentValue // Update the previous value with the current one.
		} else {
			fmt.Printf("No change detected for dependency %s.\n", dep)
		}
	}
}
