package components

import (
	"fmt"
	"syscall/js"
	"time"
)

// Attributes represents a map of HTML attributes for a given node.
type Attributes map[string]string

// Global domRegistry to store references to DOM nodes
var domRegistry = make(map[string]js.Value)

// NodeInterface is the interface that all nodes must implement.
type NodeInterface interface {
	// Render returns the HTML representation of the node.
	Render() string
	// Print returns a string representation of the node with indentation.
	Print(indent int) string
}

// Node represents an HTML tag node with attributes and children.
type Node struct {
	Tag        string
	Attributes Attributes
	Children   []NodeInterface
}

// TextNode represents a text node, which does not have attributes.
type TextNode struct {
	content string
}

// NewTextNode creates a new TextNode with the given content.
func NewTextNode(content string) *TextNode {
	return &TextNode{
		content: content,
	}
}

// Text creates a new TextNode (used in Example1).
func Text(content string) *TextNode {
	return NewTextNode(content)
}

// Render returns the text content for a TextNode.
func (t *TextNode) Render() string {
	return t.content
}

// Print returns the text content for a TextNode with appropriate indentation.
func (t *TextNode) Print(indent int) string {
	return t.content
}

// GetBindingID returns the binding ID for the Node, which is automatically generated.
// It checks if the binding ID is already in the node's attributes, otherwise returns an empty string.
func (n *Node) GetBindingID() string {
	if bindingID, ok := n.Attributes["data-go_binding_id"]; ok {
		return bindingID
	}
	// If the binding ID doesn't exist, return an empty string or generate a new one.
	return ""
}

// Render returns the HTML representation of the Node with its attributes and children.
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

// Print returns a string representation of the Node for debugging purposes.
// It recursively prints the node and its children with appropriate indentation.
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

// ValidateNode runs validation rules on the node tree recursively.
// It ensures that the node and its children contain valid HTML tags.
func ValidateNode(node *Node) error {
	if node == nil {
		return nil
	}

	// Validate the current node's tag.
	if err := validateTag(node); err != nil {
		return err
	}

	// Recursively validate children.
	for _, child := range node.Children {
		switch n := child.(type) {
		case *Node:
			if err := ValidateNode(n); err != nil {
				return err
			}
		case *TextNode:
			// No validation required for TextNodes.
			continue
		}
	}

	return nil
}

// validateTag checks if the node's tag is a valid HTML tag.
func validateTag(node *Node) error {
	// A list of valid HTML tags (extend as needed).
	validTags := []string{"div", "span", "p", "a", "button", "select", "option", "input", "form", "label"}
	for _, validTag := range validTags {
		if node.Tag == validTag || isVoidTag(node.Tag) {
			return nil
		}
	}
	return fmt.Errorf("invalid tag: %s", node.Tag)
}

// Tag creates a new HTML node with the given tag, attributes, and children.
// This is used for constructing the virtual DOM.
func Tag(tag string, attributes Attributes, children ...NodeInterface) *Node {
	return &Node{
		Tag:        tag,
		Attributes: attributes,
		Children:   children,
	}
}

// RenderAndValidateNodeTree renders the node tree to an HTML string and validates it.
// It returns the rendered HTML and an error if validation fails.
func RenderAndValidateNodeTree(root NodeInterface) (string, error) {
	// Check if the root node is a TextNode.
	if textNode, ok := root.(*TextNode); ok {
		// Return the raw text content for TextNode.
		return textNode.Render(), nil
	}

	// Otherwise, render the node as HTML.
	html := root.Render()

	// Validate the node tree if it's a regular node (and not a TextNode).
	if elementNode, ok := root.(*Node); ok {
		if err := ValidateNode(elementNode); err != nil {
			return "", err
		}
	}

	return html, nil
}

// PrintNodeTree prints the node tree with appropriate indentation for debugging.
func PrintNodeTree(root NodeInterface) {
	fmt.Print(root.Print(0))
}

// Println prints the node tree with appropriate indentation and adds a newline at the end.
func Println(root NodeInterface) {
	fmt.Println(root.Print(0))
}

// isVoidTag checks if the provided tag is a void HTML element (i.e., self-closing).
func isVoidTag(tag string) bool {
	voidTags := []string{"img", "br", "hr", "meta", "input", "link"}
	for _, t := range voidTags {
		if tag == t {
			return true
		}
	}
	return false
}

// incrementCounter is a global counter for generating unique binding IDs.
var incrementCounter = 0

// EnsureBindingIDs traverses the node tree and ensures every node has a binding ID.
// If a node does not have a binding ID, one is generated.
func EnsureBindingIDs(node *Node) {
	// Check if the current node has a binding ID, and if not, generate one.
	if node.Attributes["data-go_binding_id"] == "" {
		newID := fmt.Sprintf("go_%d", incrementCounter)
		incrementCounter++
		node.Attributes["data-go_binding_id"] = newID
	}

	// Recursively ensure all child nodes have binding IDs.
	for _, child := range node.Children {
		if childNode, ok := child.(*Node); ok {
			EnsureBindingIDs(childNode) // Recur for child nodes.
		}
	}
}

// UpdateDOM updates the DOM based on changes in the component's node structure.
// It diffs the old and new node trees and renders only the changes.
func UpdateDOM(component *Component) {
	// Diff the old and new node trees.
	if component.rootNode == nil {
		component.rootNode = component.proposedNode
	} else {
		diffNode := DiffNodeTree(component.rootNode, component.proposedNode)
		component.rootNode = diffNode
	}

	rootElement := js.Global().Get("document").Call("getElementById", "root")
	if !rootElement.IsNull() {
		// If the root element is empty, render and register the entire tree.
		if rootElement.Get("innerHTML").String() == "" {
			EnsureBindingIDs(component.rootNode)
			rootElement.Set("innerHTML", component.rootNode.Render())
			IterateAndRegisterTags(component.rootNode)
			return
		} else {
			// Otherwise, perform a diff and update the DOM.
			renderDiff(component.rootNode)
		}
	}
}

// renderDiff checks for differences between the virtual DOM and the actual DOM.
// It updates the DOM only where necessary, based on attribute or child changes.
func renderDiff(node *Node) {
	// Retrieve or query for the DOM element based on the binding ID.
	bindingID, exists := node.Attributes["data-go_binding_id"]
	if !exists || bindingID == "" {
		fmt.Println("Node does not have a binding ID, skipping.")
		return
	}

	// Check if the element is already in domRegistry.
	element, found := domRegistry[bindingID]
	if !found {
		// If not found in domRegistry, query the DOM for the element.
		element = js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-go_binding_id="%s"]`, bindingID))
		if element.IsNull() {
			fmt.Printf("Element with binding ID %s not found in the DOM.\n", bindingID)
			return
		}
		// Store the reference in domRegistry for future use.
		domRegistry[bindingID] = element
	}

	// Step 1: Attribute-Level Changes.
	for key, newValue := range node.Attributes {
		currentValue := element.Get(key).String()
		if currentValue != newValue {
			element.Call("setAttribute", key, newValue)
		}
	}
	// Remove any attributes that exist on the DOM element but not in the node.
	for i := 0; i < element.Get("attributes").Length(); i++ {
		attr := element.Get("attributes").Index(i).Get("name").String()
		if _, exists := node.Attributes[attr]; !exists && attr != "data-go_binding_id" {
			element.Call("removeAttribute", attr)
		}
	}

	// Step 2: Child-Level Changes.
	// If the node has children, check for differences and update only the changed children.
	if len(node.Children) > 0 {
		for _, child := range node.Children {
			if childNode, ok := child.(*Node); ok {
				renderDiff(childNode) // Recur for child nodes.
			} else {
				// If it's a TextNode or other type, check if innerHTML needs updating.
				currentHTML := element.Get("innerHTML").String()
				renderedHTML := child.Render()

				if currentHTML != renderedHTML {
					element.Set("innerHTML", renderedHTML)
				}
			}
		}
	} else {
		// If no children, ensure the innerHTML matches the node's content (text or empty).
		currentHTML := element.Get("innerHTML").String()
		renderedHTML := node.Render()

		if currentHTML != renderedHTML {
			element.Set("innerHTML", renderedHTML)
		}
	}
}

// DiffNodeTree compares two node trees and returns a new *Node with differences.
// If there are no changes, the original *Node is returned.
func DiffNodeTree(oldNode, newNode *Node) *Node {
	// Check if the tags are different; if so, return the new node.
	if oldNode.Tag != newNode.Tag {
		return newNode
	}

	// Ensure binding IDs are set.
	if oldNode.Attributes["data-go_binding_id"] == "" {
		oldNode.Attributes["data-go_binding_id"] = fmt.Sprintf("go_%d", time.Now().UnixNano())
	}
	if newNode.Attributes["data-go_binding_id"] == "" {
		newNode.Attributes["data-go_binding_id"] = oldNode.Attributes["data-go_binding_id"]
	}

	// Check for attribute differences.
	attrsChanged := false
	for key, oldValue := range oldNode.Attributes {
		if newValue, exists := newNode.Attributes[key]; !exists || newValue != oldValue {
			attrsChanged = true
			break
		}
	}

	// If the number of attributes differs, mark as changed.
	if len(oldNode.Attributes) != len(newNode.Attributes) {
		attrsChanged = true
	}

	// Diff children nodes.
	changedChildren := make([]NodeInterface, len(newNode.Children))
	childrenChanged := false
	for i := range newNode.Children {
		if i < len(oldNode.Children) {
			// Recursively diff children.
			oldChild, okOld := oldNode.Children[i].(*Node)
			newChild, okNew := newNode.Children[i].(*Node)

			if okOld && okNew {
				changedChild := DiffNodeTree(oldChild, newChild)
				if changedChild != oldChild {
					childrenChanged = true
				}
				changedChildren[i] = changedChild
			} else {
				// Child types differ, mark as changed.
				childrenChanged = true
				changedChildren[i] = newNode.Children[i]
			}
		} else {
			// New child, just append.
			childrenChanged = true
			changedChildren[i] = newNode.Children[i]
		}
	}

	// If attributes or children have changed, return a new node.
	if attrsChanged || childrenChanged {
		return &Node{
			Tag:        newNode.Tag,
			Attributes: newNode.Attributes,
			Children:   changedChildren,
		}
	}

	// No changes, return the old node.
	return oldNode
}

// IterateAndRegisterTags traverses the node tree and registers tag references for each node.
// It ensures all nodes are registered in the domRegistry.
func IterateAndRegisterTags(node *Node) {
	// Register the current node's binding ID.
	if bindingID, exists := node.Attributes["data-go_binding_id"]; exists && bindingID != "" {
		RegisterTagReference(bindingID)
	}

	// Recursively register tag references for all child nodes.
	for _, child := range node.Children {
		if childNode, ok := child.(*Node); ok {
			IterateAndRegisterTags(childNode) // Recur for child nodes.
		}
	}
}

// RegisterTagReference is a placeholder function to register the tag reference from the DOM.
// It stores the reference in the domRegistry.
func RegisterTagReference(bindingID string) {
	element := js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-go_binding_id="%s"]`, bindingID))
	domRegistry[bindingID] = element
}
