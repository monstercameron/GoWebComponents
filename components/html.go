package components

import (
	"fmt"
	"syscall/js"
)

type Attributes map[string]string

// Global domRegistry to store references to DOM nodes
var domRegistry = make(map[string]js.Value)

// NodeInterface is the interface that all nodes must implement
type NodeInterface interface {
	Render() string
	Print(indent int) string
}

// Node represents an HTML tag node with attributes and children
type Node struct {
	Tag        string
	Attributes Attributes
	Children   []NodeInterface
}

// TextNode represents a text node that cannot have attributes
type TextNode struct {
	content string
}

// NewTextNode creates a new TextNode
func NewTextNode(content string) *TextNode {
	return &TextNode{
		content: content,
	}
}

// Text creates a new TextNode (used in Example1)
func Text(content string) *TextNode {
	return NewTextNode(content)
}

// Render returns the text content for TextNode
func (t *TextNode) Render() string {
	return t.content
}

// Print returns the text content for TextNode with appropriate indentation
func (t *TextNode) Print(indent int) string {
	return t.content
}

// GetBindingID returns the binding ID for the Node, which is automatically generated
func (n *Node) GetBindingID() string {
	// Check if the binding ID exists in the node's attributes
	if bindingID, ok := n.Attributes["data-go_binding_id"]; ok {
		return bindingID
	}

	// If it doesn't exist, you could either return an empty string or generate a new one
	return ""
}

// Render returns the HTML representation of the Node
func (n *Node) Render() string {
	attributes := ""
	for key, value := range n.Attributes {
		attributes += fmt.Sprintf(` %s="%s"`, key, value)
	}

	result := fmt.Sprintf("<%s%s>", n.Tag, attributes)
	for _, child := range n.Children {
		result += child.Render()
	}
	result += fmt.Sprintf("</%s>", n.Tag)
	return result
}

// Print returns a string representation of the Node for debugging
func (n *Node) Print(indent int) string {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}
	result := fmt.Sprintf("%s<%s>\n", prefix, n.Tag)
	for _, child := range n.Children {
		result += child.Print(indent + 1)
	}
	result += fmt.Sprintf("%s</%s>\n", prefix, n.Tag)
	return result
}

// ValidateNode runs validation rules on the node tree
func ValidateNode(node *Node) error {
	if node == nil {
		return nil
	}

	// Validate the current node's tag
	if err := validateTag(node); err != nil {
		return err
	}

	// Recursively validate children
	for _, child := range node.Children {
		switch n := child.(type) {
		case *Node:
			if err := ValidateNode(n); err != nil {
				return err
			}
		case *TextNode:
			// No validation required for TextNodes
			continue
		}
	}

	return nil
}

// validateTag validates if the node's tag is valid HTML
func validateTag(node *Node) error {
	validTags := []string{"div", "span", "p", "a", "button", "select", "option", "input", "form", "label"} // Extend as needed
	for _, validTag := range validTags {
		if node.Tag == validTag || isVoidTag(node.Tag) {
			return nil
		}
	}
	return fmt.Errorf("invalid tag: %s", node.Tag)
}

// Tag creates a new HTML node (used in Example1)
func Tag(tag string, attributes Attributes, children ...NodeInterface) *Node {
	return &Node{
		Tag:        tag,
		Attributes: attributes,
		Children:   children,
	}
}

// RenderAndValidateNodeTree renders the node tree to an HTML string and validates it
func RenderAndValidateNodeTree(root NodeInterface) (string, error) {
	// Check if the node is a TextNode
	if textNode, ok := root.(*TextNode); ok {
		// Return the raw text content for TextNode
		return textNode.Render(), nil
	}

	// Otherwise, treat it as a regular node
	html := root.Render()

	// Validate the node tree if it's a Node (and not a TextNode)
	if elementNode, ok := root.(*Node); ok {
		if err := ValidateNode(elementNode); err != nil {
			return "", err
		}
	}

	return html, nil
}

// PrintNodeTree prints the node tree with appropriate indentation
func PrintNodeTree(root NodeInterface) {
	fmt.Print(root.Print(0))
}

// Println prints the node tree with appropriate indentation and adds a newline at the end
func Println(root NodeInterface) {
	fmt.Println(root.Print(0))
}

// isVoidTag checks if the tag is a void HTML element
func isVoidTag(tag string) bool {
	voidTags := []string{"img", "br", "hr", "meta", "input", "link"}
	for _, t := range voidTags {
		if tag == t {
			return true
		}
	}
	return false
}

// UpdateDOM updates the DOM with the changes in the component's node structure
func UpdateDOM(component *Component) {
	if component.rootNode == nil {
		return
	}

	var walkAndApply func(node NodeInterface, parent js.Value)
	walkAndApply = func(node NodeInterface, parent js.Value) {
		switch n := node.(type) {
		case *Node:
			bindingID := n.GetBindingID()

			// Handle regular element nodes (with binding ID)
			element := js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-go_binding_id="%s"]`, bindingID))
			if !element.IsNull() {
				// Update the node if it exists
				// Update children as well
				for _, child := range n.Children {
					walkAndApply(child, element)
				}
			} else {
				// Register the node if not already present
				domRegistry[bindingID] = element
			}

		case *TextNode:
			// Handle text nodes by appending them directly to the parent element
			if !parent.IsNull() {
				parent.Set("textContent", n.Render()) // Use Render() instead of Text()
			}
		}
	}

	// Start walking and applying updates from the root node
	walkAndApply(component.rootNode, js.Null())
}

// RegisterTagReference is a placeholder function to register the tag reference from the DOM
func RegisterTagReference(bindingID string) {
	element := js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-go_binding_id="%s"]`, bindingID))
	domRegistry[bindingID] = element
	fmt.Printf("Registering tag reference for %s (Placeholder)\n", bindingID)
}
