package components

import (
	"fmt"
	"strings"
	"sync"
)

// Props is a struct to pass properties to components
type Props struct {
	InitialValue string
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
}

// ElementNode represents an HTML element
type ElementNode struct {
	ID         interface{}
	Value      interface{}
	TagName    string
	Attributes map[string]string
	Children   []NodeInterface
	mu         sync.RWMutex
}

type Component struct {
	Children     []*Component      // Children components
	Nodes        []NodeInterface   // Node children
	RootNode     NodeInterface
	RenderedHTML string
	State        map[string]interface{}
	mu           sync.Mutex
}

// NewComponent initializes a new Component
func NewComponent() *Component {
	return &Component{
		State:    make(map[string]interface{}),
		Children: make([]*Component, 0),
		Nodes:    make([]NodeInterface, 0),
	}
}


func CreateComponent(f func(*Component, Props, ...*Component) *Component) func(Props, ...*Component) *Component {
	return func(props Props, children ...*Component) *Component {
		c := NewComponent() // Create the parent component

		// Register each child component
		for _, child := range children {
			if child != nil {
				c.Children = append(c.Children, child)
			}
		}

		// Call the provided function with the parent component, props, and children
		return f(c, props, children...)
	}
}



func AddState[T any](c *Component, initialState T) (*T, func(T)) {
    c.mu.Lock()
    defer c.mu.Unlock()

    key := generateUniqueKey(c)
    valuePtr := new(T)
    *valuePtr = initialState
    c.State[key] = valuePtr

    // Setter function that updates the state and triggers re-render
    setValue := func(newValue T) {
        c.mu.Lock()
        defer c.mu.Unlock()
        *valuePtr = newValue
        c.RenderedHTML = "" // Invalidate the rendered HTML

        // Trigger a re-render of the component and its subtree
        c.RenderedHTML = renderComponentTree(c)
    }

    return valuePtr, setValue
}

func renderComponentTree(c *Component) string {
	if c.RootNode == nil {
		return ""
	}

	// Start with the current component's rendered HTML
	renderedHTML := c.RootNode.Render(0)

	// Render each child component recursively
	for _, child := range c.Children {
		renderedHTML += renderComponentTree(child)
	}

	return renderedHTML
}


func Render(c *Component, rootNode NodeInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.RootNode = rootNode
	c.Nodes = append(c.Nodes, rootNode)

	// Render and cache the HTML for the component and its subtree
	c.RenderedHTML = renderComponentTree(c)
}

// Render method for Component generates and returns the rendered HTML
func (c *Component) Render() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.RootNode == nil {
		return "", fmt.Errorf("component has no root node")
	}

	if c.RenderedHTML == "" {
		c.RenderedHTML = c.RootNode.Render(0)
	}

	return c.RenderedHTML, nil
}

// generateUniqueKey generates a unique key for the state
func generateUniqueKey(c *Component) string {
	return fmt.Sprintf("state-%d", len(c.State))
}

// ElementNode methods

func (n *ElementNode) SetValue(value interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Value = value
}

func (n *ElementNode) GetValue() interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Value
}

func (n *ElementNode) SetTagName(tagName string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.TagName = tagName
	return nil
}

func (n *ElementNode) GetTagName() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.TagName
}

func (n *ElementNode) SetAttribute(key, value string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Attributes[key] = value
	return nil
}

func (n *ElementNode) GetAttributes() map[string]string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	attrs := make(map[string]string)
	for k, v := range n.Attributes {
		attrs[k] = v
	}
	return attrs
}

func (n *ElementNode) AddChild(child NodeInterface) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Children = append(n.Children, child)
}

func (n *ElementNode) GetChildren() []NodeInterface {
	n.mu.RLock()
	defer n.mu.RUnlock()
	children := make([]NodeInterface, len(n.Children))
	copy(children, n.Children)
	return children
}

func (n *ElementNode) Render(level int) string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	indent := strings.Repeat("  ", level)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s<%s", indent, n.TagName))

	for key, value := range n.Attributes {
		sb.WriteString(fmt.Sprintf(" %s=\"%s\"", key, value))
	}

	if len(n.Children) == 0 && n.Value == nil {
		sb.WriteString(" />")
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
		sb.WriteString(fmt.Sprintf("</%s>", n.TagName))
	}
	sb.WriteString("\n")

	return sb.String()
}

// Tag creates a new ElementNode with attributes and children
func Tag(tagName string, attributes map[string]string, children ...interface{}) NodeInterface {
	node := &ElementNode{
		TagName:    tagName,
		Attributes: make(map[string]string),
		Children:   make([]NodeInterface, 0),
	}

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

// Text creates a new TextNode
func Text(content string) NodeInterface {
	return &TextNode{Content: content}
}

// TextNode represents a text content node
type TextNode struct {
	Content string
}

// Implement NodeInterface methods for TextNode
func (n *TextNode) SetValue(value interface{})              { n.Content = fmt.Sprintf("%v", value) }
func (n *TextNode) GetValue() interface{}                   { return n.Content }
func (n *TextNode) SetTagName(tagName string) error         { return nil }
func (n *TextNode) GetTagName() string                      { return "" }
func (n *TextNode) SetAttribute(key, value string) error    { return nil }
func (n *TextNode) GetAttributes() map[string]string        { return nil }
func (n *TextNode) AddChild(child NodeInterface)            {}
func (n *TextNode) GetChildren() []NodeInterface            { return nil }
func (n *TextNode) Render(level int) string                 { return n.Content }
