package components

import (
	"fmt"
	"syscall/js"
	"time"
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

var incrementCounter = 0

// Iterate through the node tree and ensure every node has a binding ID
func EnsureBindingIDs(node *Node) {
	// Check if the current node has a binding ID, and if not, generate one
	if node.Attributes["data-go_binding_id"] == "" {
		newID := fmt.Sprintf("go_%d", incrementCounter)
		incrementCounter++
		node.Attributes["data-go_binding_id"] = newID
		fmt.Printf("Generated new binding ID: %s for node <%s>\n", newID, node.Tag)
	}

	// Recursively ensure all child nodes have binding IDs
	for _, child := range node.Children {
		if childNode, ok := child.(*Node); ok {
			EnsureBindingIDs(childNode) // Recur for child nodes
		}
	}
}

// UpdateDOM updates the DOM with the changes in the component's node structure
func UpdateDOM(component *Component) {
	// Diff the old and new node trees
	if component.rootNode == nil {
		component.rootNode = component.proposedNode
	} else {
		diffNode := DiffNodeTree(component.rootNode, component.proposedNode)
		component.rootNode = diffNode
	}
	// component.rootNode = component.proposedNode

	rootElement := js.Global().Get("document").Call("getElementById", "root")
	if !rootElement.IsNull() {
		if rootElement.Get("innerHTML").String() == "" {
			EnsureBindingIDs(component.rootNode)
			rootElement.Set("innerHTML", component.rootNode.Render())
			IterateAndRegisterTags(component.rootNode)
			return
		} else {
			renderDiff(component.rootNode)
		}
	}
}

// renderDiff checks the DOM element for attribute diffs and child changes, then updates the DOM if necessary
func renderDiff(node *Node) {
	// Retrieve or query for the DOM element based on the binding ID
	bindingID, exists := node.Attributes["data-go_binding_id"]
	if !exists || bindingID == "" {
		fmt.Println("Node does not have a binding ID, skipping.")
		return
	}

	// Check if the element is already in domRegistry
	element, found := domRegistry[bindingID]
	if !found {
		// If not found in domRegistry, query the DOM for the element
		element = js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-go_binding_id="%s"]`, bindingID))
		if element.IsNull() {
			fmt.Printf("Element with binding ID %s not found in the DOM.\n", bindingID)
			return
		}
		// Store the reference in domRegistry for future use
		domRegistry[bindingID] = element
		fmt.Printf("Element with binding ID %s found and stored in domRegistry.\n", bindingID)
	} else {
		fmt.Printf("Element with binding ID %s retrieved from domRegistry.\n", bindingID)
	}

	// Step 1: Attribute-Level Changes
	for key, newValue := range node.Attributes {
		currentValue := element.Get(key).String()
		if currentValue != newValue {
			fmt.Printf("Updating attribute %s (old: %s, new: %s) on element with binding ID %s.\n", key, currentValue, newValue, bindingID)
			element.Call("setAttribute", key, newValue)
		}
	}
	// Remove any attributes that exist on the DOM element but not in the node
	for i := 0; i < element.Get("attributes").Length(); i++ {
		attr := element.Get("attributes").Index(i).Get("name").String()
		if _, exists := node.Attributes[attr]; !exists && attr != "data-go_binding_id" {
			element.Call("removeAttribute", attr)
			fmt.Printf("Removed attribute %s from element with binding ID %s.\n", attr, bindingID)
		}
	}

	// Step 2: Child-Level Changes
	// If the node has children, check for differences and update only the changed children
	if len(node.Children) > 0 {
		for _, child := range node.Children {
			if childNode, ok := child.(*Node); ok {
				// Recursively check for diffs in child nodes
				renderDiff(childNode)
			} else {
				// If it's a TextNode or other type, check if innerHTML needs updating
				currentHTML := element.Get("innerHTML").String()
				renderedHTML := child.Render()

				if currentHTML != renderedHTML {
					fmt.Printf("Updating innerHTML on element with binding ID %s (old: %s, new: %s).\n", bindingID, currentHTML, renderedHTML)
					element.Set("innerHTML", renderedHTML)
				}
			}
		}
	} else {
		// If no children, ensure the innerHTML matches the node's content (text or empty)
		currentHTML := element.Get("innerHTML").String()
		renderedHTML := node.Render()

		if currentHTML != renderedHTML {
			fmt.Printf("Updating innerHTML on element with binding ID %s (no children, old: %s, new: %s).\n", bindingID, currentHTML, renderedHTML)
			element.Set("innerHTML", renderedHTML)
		}
	}
}

// DiffNodeTree compares two node trees and returns a new *Node with differences.
// If there are no changes, the original *Node is returned.
func DiffNodeTree(oldNode, newNode *Node) *Node {
	// Check if the tags are different; if so, return the new node
	if oldNode.Tag != newNode.Tag {
		return newNode
	}

	// Check and update the binding ID if missing
	if oldNode.Attributes["data-go_binding_id"] == "" {
		oldNode.Attributes["data-go_binding_id"] = fmt.Sprintf("go_%d", time.Now().UnixNano())
	}
	if newNode.Attributes["data-go_binding_id"] == "" {
		newNode.Attributes["data-go_binding_id"] = oldNode.Attributes["data-go_binding_id"]
	}

	// Check for attribute differences
	attrsChanged := false
	for key, oldValue := range oldNode.Attributes {
		if newValue, exists := newNode.Attributes[key]; !exists || newValue != oldValue {
			attrsChanged = true
			break
		}
	}

	// If attributes count differs, mark as changed
	if len(oldNode.Attributes) != len(newNode.Attributes) {
		attrsChanged = true
	}

	// Diff children nodes
	changedChildren := make([]NodeInterface, len(newNode.Children))
	childrenChanged := false
	for i := range newNode.Children {
		if i < len(oldNode.Children) {
			// Recursively diff children
			oldChild, okOld := oldNode.Children[i].(*Node)
			newChild, okNew := newNode.Children[i].(*Node)

			if okOld && okNew {
				changedChild := DiffNodeTree(oldChild, newChild)
				if changedChild != oldChild {
					childrenChanged = true
				}
				changedChildren[i] = changedChild
			} else {
				// Child types differ, mark as changed
				childrenChanged = true
				changedChildren[i] = newNode.Children[i]
			}
		} else {
			// New child, just append
			childrenChanged = true
			changedChildren[i] = newNode.Children[i]
		}
	}

	// If attributes or children have changed, return a new node
	if attrsChanged || childrenChanged {
		return &Node{
			Tag:        newNode.Tag,
			Attributes: newNode.Attributes,
			Children:   changedChildren,
		}
	}

	// No changes, return the old node
	return oldNode
}

// Iterate through the node tree and register tag references for each node
func IterateAndRegisterTags(node *Node) {
	// Register the current node's binding ID
	if bindingID, exists := node.Attributes["data-go_binding_id"]; exists && bindingID != "" {
		RegisterTagReference(bindingID)
		fmt.Printf("Registered tag reference for node with binding ID: %s\n", bindingID)
	}

	// Recursively register tag references for all child nodes
	for _, child := range node.Children {
		if childNode, ok := child.(*Node); ok {
			IterateAndRegisterTags(childNode) // Recur for child nodes
		}
	}
}

// RegisterTagReference is a placeholder function to register the tag reference from the DOM
func RegisterTagReference(bindingID string) {
	element := js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-go_binding_id="%s"]`, bindingID))
	domRegistry[bindingID] = element
	fmt.Printf("Registering tag reference for %s (Placeholder)\n", bindingID)
}
