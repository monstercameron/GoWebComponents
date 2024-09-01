package components

import (
	"fmt"
	"sync"
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

// ComponentFunc is a function type that represents a component's creation logic
type ComponentFunc[P any] func(*Component, P, ...*Component) *Component

// CreateComponent handles the creation of components, taking generic Props and child components
func ThisIsAComponent[P any](f ComponentFunc[P]) func(P, ...*Component) *Component {

	return func(props P, children ...*Component) *Component {
		// Create a new component with no root node initially
		self := &Component{
			state:        make(map[string]interface{}),
			lifecycle:    make(map[string]func()),
			cachedValues: make(map[string]map[string]interface{}),
		}

		// Assign the updateStateFunc after the component has been created
		self.updateStateFunc = func() {
			// Invoke the render function with captured variables
			f(self, props, children...)
		}

		// Return the initialized component without calling updateStateFunc
		return self
	}
}

// RenderTemplate attaches the *Node markup to the component struct
func RenderTemplate(self *Component, node *Node) {
	// Attach the provided node as the root node of the component
	self.rootNode = node
}

// RenderHTML returns the HTML string of the rendered component
func RenderHTML(c *Component) string {
	if c.rootNode == nil {
		return ""
	}

	// Manually render the root node into an HTML string
	html := "<" + c.rootNode.Tag
	for key, value := range c.rootNode.Attributes {
		html += " " + key + "=\"" + value + "\""
	}
	html += ">"

	// Recursively render children
	// for _, child := range c.rootNode.Children {
	// 	html += RenderNode(child)
	// }

	html += "</" + c.rootNode.Tag + ">"

	return html
}

// RenderNode manually renders a Node and its children to an HTML string
func RenderNode(n *Node) string {
	if n == nil {
		return ""
	}

	// Render opening tag with attributes
	html := "<" + n.Tag
	for key, value := range n.Attributes {
		html += " " + key + "=\"" + value + "\""
	}
	html += ">"

	// Render children
	// for _, child := range n.Children {
	// 	html += RenderNode(child)
	// }

	// Render closing tag
	html += "</" + n.Tag + ">"

	return html
}

// Cached returns a cached value that only recalculates when dependencies change or if it's the first run
func Cached(c *Component, key string, calcFunc func() interface{}, deps []string) interface{} {
	// Retrieve or initialize the cache entry for the given key
	cache, exists := c.cachedValues[key]
	if !exists {
		cache = make(map[string]interface{})
		c.cachedValues[key] = cache
	}

	// Check if this is the first time running or if any dependency has changed
	needsRecalculation := false
	if _, resultExists := cache["result"]; !resultExists {
		needsRecalculation = true // First time running
	} else {
		for _, depKey := range deps {
			newVal := c.state[depKey]
			if cache[depKey] != newVal {
				needsRecalculation = true
				break // Stop checking further once a change is detected
			}
		}
	}

	// Recalculate if needed
	if needsRecalculation {
		result := calcFunc()

		// Update the cache with the new dependency values and the result
		for _, dk := range deps {
			cache[dk] = c.state[dk]
		}
		cache["result"] = result

		return result
	}

	// Return the existing cached result if no recalculation was necessary
	return cache["result"]
}

// AddState adds a state to the component
func AddState[T any](c *Component, key string, initialValue T) (*T, func(T)) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	// Store the initial value in the state map
	c.state[key] = initialValue

	// Return a pointer to the value and a setter function
	return c.state[key].(*T), func(newValue T) {
		c.stateLock.Lock()
		defer c.stateLock.Unlock()

		// Update the value in the state map
		c.state[key] = newValue

		// Trigger re-render or state update
		c.updateStateFunc()
	}
}

// Setup adds a setup function to be called when the component is mounted
func Setup(c *Component, setupFunc func()) {
	c.lifecycle["setup"] = setupFunc
}

// Cleanup adds a cleanup function to be called when the component is unmounted
func Cleanup(c *Component, cleanupFunc func()) {
	c.lifecycle["cleanup"] = cleanupFunc
}

// UnregisterComponent handles the cleanup of the component
func UnregisterComponent(c *Component) {
	if c.registered {
		if cleanupFunc, ok := c.lifecycle["cleanup"]; ok {
			cleanupFunc()
		}
		c.registered = false // Mark the component as unregistered (unmounted)
	}
}

// registerDOMReferences registers all nodes in the component's tree with the DOM
func registerDOMReferences(node *Node) {
	// Register the current node
	if bindingID, ok := node.Attributes["data-go_binding_id"]; ok {
		RegisterTagReference(bindingID)
	}

	// Recursively register child nodes
	for _, child := range node.Children {
		if elementNode, ok := child.(*Node); ok {
			registerDOMReferences(elementNode)
		}
	}
}

// RegisterTagReference is a placeholder for registering the tag reference
func RegisterTagReference(bindingID string) {
	// Placeholder for registering a tag reference in the DOM
	fmt.Println("Registering tag reference:", bindingID)
}
