package components

import (
	"fmt"
	"strings"
	"sync"
	"syscall/js"
)

var (
	nodeMap   = make(map[string]js.Value) // Global map to store node references
	autoIDSeq = 0                         // Counter for generating unique IDs
	mu        sync.Mutex
)

// GenerateAutoID generates a unique ID for each component/node
func GenerateAutoID() string {
	mu.Lock()
	defer mu.Unlock()
	autoIDSeq++
	return fmt.Sprintf("auto-id-%d", autoIDSeq)
}

// NodeInterface defines the interface for all node types
type NodeInterface interface {
	SetValue(value interface{})
	GetValue() interface{}
	SetTagName(tagName string) error
	GetTagName() string
	SetAttribute(key, value string) error
	GetAttributes() map[string]string
	AddChild(child NodeInterface)
	GetChildren() []NodeInterface
	Render(level int) string
	GetAutoID() string // Method to retrieve the auto-generated ID
}

// ElementNode represents an HTML element
type ElementNode struct {
	AutoID     string
	TagName    string
	Attributes map[string]string
	Children   []NodeInterface
	Value      interface{} // Added Value to store content
	mu         sync.RWMutex
}

// GetAutoID returns the auto-generated ID of the ElementNode
func (n *ElementNode) GetAutoID() string {
	return n.AutoID
}

// SetValue sets the value of the element node
func (n *ElementNode) SetValue(value interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Value = value
}

// GetValue gets the value of the element node
func (n *ElementNode) GetValue() interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Value
}

// SetTagName sets the tag name of the element node
func (n *ElementNode) SetTagName(tagName string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.TagName = tagName
	return nil
}

// GetTagName gets the tag name of the element node
func (n *ElementNode) GetTagName() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.TagName
}

// SetAttribute sets an attribute on the element node
func (n *ElementNode) SetAttribute(key, value string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Attributes[key] = value
	return nil
}

// GetAttributes gets all attributes of the element node
func (n *ElementNode) GetAttributes() map[string]string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	attrs := make(map[string]string)
	for k, v := range n.Attributes {
		attrs[k] = v
	}
	return attrs
}

// AddChild adds a child node to the element node
func (n *ElementNode) AddChild(child NodeInterface) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Children = append(n.Children, child)
}

// GetChildren gets all children of the element node
func (n *ElementNode) GetChildren() []NodeInterface {
	n.mu.RLock()
	defer n.mu.RUnlock()
	children := make([]NodeInterface, len(n.Children))
	copy(children, n.Children)
	return children
}

// Render renders the element node to a string with indentation
func (n *ElementNode) Render(level int) string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	indent := strings.Repeat("  ", level)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s<%s", indent, n.TagName))

	// Include all attributes, including the data-auto-id
	for key, value := range n.Attributes {
		sb.WriteString(fmt.Sprintf(" %s=\"%s\"", key, value))
	}

	if len(n.Children) == 0 && n.Value == nil {
		sb.WriteString(" />\n")
	} else {
		sb.WriteString(">")
		if n.Value != nil {
			sb.WriteString(fmt.Sprintf("%v", n.Value))
		}
		if len(n.Children) > 0 {
			sb.WriteString("\n")
			for _, child := range n.Children {
				sb.WriteString(child.Render(level + 1))
			}
			sb.WriteString(indent)
		}
		sb.WriteString(fmt.Sprintf("</%s>\n", n.TagName))
	}

	// Register the node in the global map after rendering
	n.RegisterNode()

	return sb.String()
}

// RegisterNode stores the reference to the DOM node in the map
func (n *ElementNode) RegisterNode() {
	js.Global().Call("requestAnimationFrame", js.FuncOf(func(js.Value, []js.Value) interface{} {
		document := js.Global().Get("document")
		selector := fmt.Sprintf("[data-auto-id='%s']", n.AutoID)

		// Query the DOM for the element using its auto-generated ID
		node := document.Call("querySelector", selector)

		if node.Truthy() {
			// Store the node reference in the global map
			mu.Lock()
			nodeMap[n.AutoID] = node
			mu.Unlock()
		} else {
			// fmt.Printf("Warning: Element with AutoID %s not found in the DOM.\n", n.AutoID)
		}

		return nil
	}))
}

// Tag creates a new ElementNode with attributes and children
func Tag(tagName string, attributes map[string]string, children ...interface{}) NodeInterface {
	node := &ElementNode{
		AutoID:     GenerateAutoID(),
		TagName:    tagName,
		Attributes: make(map[string]string),
		Children:   make([]NodeInterface, 0),
	}

	// Add the auto-ID as a data attribute
	node.Attributes["data-auto-id"] = node.AutoID

	for key, value := range attributes {
		node.Attributes[key] = value
	}

	for _, child := range children {
		switch v := child.(type) {
		case NodeInterface:
			node.Children = append(node.Children, v)
		case string:
			node.Children = append(node.Children, &TextNode{Content: v})
		default:
			node.Children = append(node.Children, &TextNode{Content: fmt.Sprintf("%v", v)})
		}
	}

	return node
}

// Text creates a simple TextNode
func Text(content string) NodeInterface {
	return &TextNode{Content: content}
}

// TextNode represents a text content node
type TextNode struct {
	Content string
}

// Implement NodeInterface methods for TextNode
func (n *TextNode) SetValue(value interface{})           { n.Content = fmt.Sprintf("%v", value) }
func (n *TextNode) GetValue() interface{}                { return n.Content }
func (n *TextNode) SetTagName(tagName string) error      { return nil }
func (n *TextNode) GetTagName() string                   { return "" }
func (n *TextNode) SetAttribute(key, value string) error { return nil }
func (n *TextNode) GetAttributes() map[string]string     { return nil }
func (n *TextNode) AddChild(child NodeInterface)         {}
func (n *TextNode) GetChildren() []NodeInterface         { return nil }
func (n *TextNode) Render(level int) string              { return n.Content }
func (n *TextNode) GetAutoID() string                    { return "" } // TextNodes don't need an AutoID
