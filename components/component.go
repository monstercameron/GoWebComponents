package components

import (
	"fmt"
	"sync"
	"syscall/js"
)

// Props is defined as a map of string to interface{}
type Props map[string]interface{}

// Component struct defines the structure for a UI component
type Component struct {
	Children      []*Component
	Nodes         []NodeInterface
	RootNode      NodeInterface
	RenderedHTML  string
	State         map[string]interface{}
	mu            sync.Mutex
	Parent        *Component
	OnStateChange func()
	renderFunc    func(*Component, Props, ...*Component) *Component
}

// NewComponent initializes a new Component
func NewComponent() *Component {
	return &Component{
		State:    make(map[string]interface{}),
		Children: make([]*Component, 0),
		Nodes:    make([]NodeInterface, 0),
	}
}

// CreateComponent handles the creation of components, taking Props and children components
func CreateComponent(f func(*Component, Props, ...*Component) *Component) func(Props, ...*Component) *Component {
    return func(props Props, children ...*Component) *Component {
        c := NewComponent()
        c.renderFunc = f // Store the reference to the render function

        for _, child := range children {
            if child != nil {
                c.Children = append(c.Children, child)
                child.Parent = c
            }
        }

        // Function to reuse existing AutoIDs
        reuseAutoIDs := func(oldComponent, newComponent *Component) {
            if oldComponent != nil && newComponent != nil {
                if oldComponent.RootNode != nil && newComponent.RootNode != nil {
                    newComponent.RootNode.(*ElementNode).AutoID = oldComponent.RootNode.GetAutoID()
                }

                for i := range oldComponent.Children {
                    if i < len(newComponent.Children) {
                        reuseAutoIDs(oldComponent.Children[i], newComponent.Children[i])
                    }
                }
            }
        }

        c.OnStateChange = func() {
            c.RenderedHTML = "" // Invalidate the rendered HTML
            c.Nodes = nil       // Clear previous nodes

            // Store reference to old component before re-render
            oldComponent := *c

            // Re-run the render function to get the updated component
            newComponent := c.renderFunc(c, props, children...)

            // Reuse AutoIDs from the old component tree
            reuseAutoIDs(&oldComponent, newComponent)

            // Manually copy the fields from newComponent to c
            c.Children = newComponent.Children
            c.Nodes = newComponent.Nodes
            c.RootNode = newComponent.RootNode
            c.RenderedHTML = newComponent.RenderedHTML
            c.OnStateChange = newComponent.OnStateChange
            c.renderFunc = newComponent.renderFunc

            // Re-render the component tree with the updated component
            c.RenderedHTML = renderComponentTree(c)

            // Propagate update to parent
            if c.Parent != nil && c.Parent.OnStateChange != nil {
                c.Parent.OnStateChange()
            }
        }

        return c.renderFunc(c, props, children...)
    }
}

// Utility function to copy IDs from old nodes to new ones
func reuseAutoIDs(oldComponent, newComponent *Component) {
    if oldComponent != nil && newComponent != nil {
        if oldComponent.RootNode != nil && newComponent.RootNode != nil {
            oldElementNode, okOld := oldComponent.RootNode.(*ElementNode)
            newElementNode, okNew := newComponent.RootNode.(*ElementNode)
            if okOld && okNew {
                newElementNode.AutoID = oldElementNode.AutoID
            }
        }

        for i := range oldComponent.Children {
            if i < len(newComponent.Children) {
                reuseAutoIDs(oldComponent.Children[i], newComponent.Children[i])
            }
        }
    }
}

// AddState manages the component state, identified by a unique key generated from the pointer address.
func AddState[T any](c *Component, key string, initialState T) (*T, func(T)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the state already exists to prevent overwriting
	if existingValue, exists := c.State[key]; exists {
		valuePtr := existingValue.(*T)
		return valuePtr, func(newValue T) {
			c.mu.Lock()
			*valuePtr = newValue
			c.mu.Unlock()

			// // Trigger the OnStateChange callback
			// if c.OnStateChange != nil {
			// 	c.OnStateChange()
			// }
		}
	}

	// Initialize the state for the first time
	valuePtr := new(T)
	*valuePtr = initialState
	c.State[key] = valuePtr

	// Setter function to update the state and trigger re-rendering
	setValue := func(newValue T) {
		c.mu.Lock()
		*valuePtr = newValue
		c.mu.Unlock()

		// Trigger the OnStateChange callback
		if c.OnStateChange != nil {
			c.OnStateChange()
		}
	}

	return valuePtr, setValue
}

// Render function update
func Render(c *Component, rootNode NodeInterface) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.RootNode = rootNode
	c.Nodes = append(c.Nodes, rootNode)

	// Render and cache the HTML for the component and its subtree
	c.RenderedHTML = renderComponentTree(c)
}

// renderComponentTree function (helper function) recursively renders the component tree
func renderComponentTree(c *Component) string {
	if c.RootNode == nil {
		return ""
	}

	renderedHTML := c.RootNode.Render(0)

	for _, child := range c.Children {
		renderedHTML += renderComponentTree(child)
	}

	return renderedHTML
}

func (c *Component) Render() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.RootNode == nil {
		return "", fmt.Errorf("component has no root node")
	}

	c.RenderedHTML = c.RootNode.Render(0)
	return c.RenderedHTML, nil
}

// generateUniqueKey generates a unique key for the state
func generateUniqueKey(c *Component) string {
	return fmt.Sprintf("state-%d", len(c.State))
}

// RenderAndUpdateDom renders the component and updates the DOM node
func (c *Component) RenderAndUpdateDom() {
	// Render the component to HTML
	c.RenderedHTML, _ = c.Render()

	// Ensure the DOM update happens after rendering with a slight delay
	js.Global().Call("setTimeout", js.FuncOf(func(js.Value, []js.Value) interface{} {
		if c.RootNode != nil {
			// Update the specific DOM node
			c.UpdateDomNodeByAutoID(c.RootNode.GetAutoID(), c.RenderedHTML)

			// Register nodes only after updating the DOM
			if elementNode, ok := c.RootNode.(*ElementNode); ok {
				elementNode.RegisterNode()
			}
		}

		// Recursively update and register children
		for _, child := range c.Children {
			child.RenderAndUpdateDom()
		}

		return nil
	}), 0) // Use a 0ms delay to defer execution until after the current event loop
}


// UpdateDomNodeByAutoID updates a DOM node by its AutoID with the given content
func (c *Component) UpdateDomNodeByAutoID(autoID string, content string) {
	mu.Lock()
	element, exists := nodeMap[autoID]
	mu.Unlock()
	if exists {
		element.Set("innerHTML", content)
	} else {
		fmt.Printf("Element with AutoID %s not found\n", autoID)
	}
}

// RenderToBody replaces the innerHTML of the document body with the rendered HTML of the given component.
func RenderToBody(c *Component) {
	if renderedHTML, err := c.Render(); err == nil {
		// Replace the innerHTML of the body element
		js.Global().Get("document").Get("body").Set("innerHTML", renderedHTML)
	} else {
		fmt.Println("Error rendering component:", err)
	}
}