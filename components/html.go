package components

import (
    "fmt"
    "syscall/js"
)

// Attributes represents a map of HTML attributes for a given node.
type Attributes map[string]string

// NodeInterface is the interface that all nodes must implement.
type NodeInterface interface {
    Render() string
    Print(indent int) string
    GetBindingID() string
    SetBindingID(string)
}

// Node represents an HTML tag node with attributes and children.
type Node struct {
    Tag        string
    Attributes Attributes
    Children   []NodeInterface
    bindingID  string // Store the binding ID explicitly
}

// TextNode represents a text node.
type TextNode struct {
    content   string
    bindingID string // For consistency, though text nodes don't need binding IDs
}

// NewTextNode creates a new TextNode with the given content.
func NewTextNode(content string) *TextNode {
    return &TextNode{
        content: content,
    }
}

// Text creates a new TextNode.
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

// GetBindingID returns the binding ID for the TextNode.
func (t *TextNode) GetBindingID() string {
    return t.bindingID
}

// SetBindingID sets the binding ID for the TextNode.
func (t *TextNode) SetBindingID(id string) {
    t.bindingID = id
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
    if !isVoidTag(n.Tag) {
        result += fmt.Sprintf("</%s>", n.Tag)
    }
    return result
}

// Print returns a string representation of the Node for debugging purposes.
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

// GetBindingID returns the binding ID for the Node.
func (n *Node) GetBindingID() string {
    return n.bindingID
}

// SetBindingID sets the binding ID for the Node.
func (n *Node) SetBindingID(id string) {
    n.bindingID = id
}

// Tag creates a new HTML node with the given tag, attributes, and children.
func Tag(tag string, attributes Attributes, children ...NodeInterface) *Node {
    return &Node{
        Tag:        tag,
        Attributes: attributes,
        Children:   children,
    }
}

// isVoidTag checks if the provided tag is a void HTML element.
func isVoidTag(tag string) bool {
    voidTags := []string{"img", "br", "hr", "meta", "input", "link", "area", "base", "col", "embed", "param", "source", "track", "wbr"}
    for _, t := range voidTags {
        if tag == t {
            return true
        }
    }
    return false
}

// Global domRegistry to store references to DOM nodes.
var domRegistry = make(map[string]js.Value)

// incrementCounter is a global counter for generating unique binding IDs.
var incrementCounter = 0

// EnsureBindingIDs traverses the node tree and assigns binding IDs only to nodes that don't have one.
func EnsureBindingIDs(node NodeInterface) {
    if node.GetBindingID() == "" {
        newID := fmt.Sprintf("go_%d", incrementCounter)
        incrementCounter++
        node.SetBindingID(newID)
    }
    switch n := node.(type) {
    case *Node:
        // Add data-go_binding_id attribute to node's attributes
        if n.Attributes == nil {
            n.Attributes = make(Attributes)
        }
        n.Attributes["data-go_binding_id"] = n.GetBindingID()
        for _, child := range n.Children {
            EnsureBindingIDs(child)
        }
    case *TextNode:
        // For TextNodes, we can skip adding the binding ID as an attribute
        // Since they don't have attributes
    }
}

// UpdateDOM updates the DOM based on changes in the component's node structure.
func UpdateDOM(component *Component) {
    rootElement := js.Global().Get("document").Call("getElementById", "root")
    if rootElement.IsNull() {
        fmt.Println("Root element not found in the DOM.")
        return
    }

    // First render: render the entire tree.
    if component.rootNode == nil {
        EnsureBindingIDs(component.proposedNode)
        component.rootNode = component.proposedNode
        domElement := renderNodeToDOM(component.proposedNode)
        rootElement.Set("innerHTML", "")
        rootElement.Call("appendChild", domElement)
    } else {
        // Subsequent renders: diff and update.
        fmt.Println("Updating DOM")
        diffAndUpdate(component.rootNode, component.proposedNode)
        component.rootNode = component.proposedNode
    }
}

// diffAndUpdate recursively diffs the old and new nodes and updates the DOM accordingly.
func diffAndUpdate(oldNode NodeInterface, newNode NodeInterface) {
    // If the nodes are the same object, do nothing
    if oldNode == newNode {
        return
    }

    // Copy binding ID from old node to new node
    newNode.SetBindingID(oldNode.GetBindingID())

    // Get the DOM element corresponding to the oldNode
    domElement := getDOMElement(oldNode)
    if domElement.IsUndefined() || domElement.IsNull() {
        fmt.Println("DOM Element is undefined or null in diffAndUpdate")
        return
    }

    // Check if the nodes are of different types (Node vs TextNode)
    switch oldNodeTyped := oldNode.(type) {
    case *TextNode:
        switch newNodeTyped := newNode.(type) {
        case *TextNode:
            // Both are TextNodes
            if oldNodeTyped.content != newNodeTyped.content {
                domElement.Set("nodeValue", newNodeTyped.content)
            }
            // Register the new node
            registerDOMElement(newNode, domElement)
        default:
            // Replace text node with new element
            newDomElement := renderNodeToDOM(newNode)
            parent := domElement.Get("parentNode")
            parent.Call("replaceChild", newDomElement, domElement)
            registerDOMElement(newNode, newDomElement)
            unregisterDOMElement(oldNode)
        }
    case *Node:
        switch newNodeTyped := newNode.(type) {
        case *Node:
            // Both are Nodes
            if oldNodeTyped.Tag != newNodeTyped.Tag {
                // Replace the entire node
                newDomElement := renderNodeToDOM(newNodeTyped)
                parent := domElement.Get("parentNode")
                parent.Call("replaceChild", newDomElement, domElement)
                registerDOMElement(newNodeTyped, newDomElement)
                unregisterDOMElement(oldNodeTyped)
            } else {
                // Same tag: update attributes and children
                updateAttributes(domElement, oldNodeTyped.Attributes, newNodeTyped.Attributes)
                registerDOMElement(newNode, domElement)
                diffChildren(oldNodeTyped.Children, newNodeTyped.Children, domElement)
            }
        default:
            // Replace element node with text node
            newDomElement := renderNodeToDOM(newNode)
            parent := domElement.Get("parentNode")
            parent.Call("replaceChild", newDomElement, domElement)
            registerDOMElement(newNode, newDomElement)
            unregisterDOMElement(oldNode)
        }
    }
}

// diffChildren diffs the children of a node and updates the DOM accordingly.
func diffChildren(oldChildren []NodeInterface, newChildren []NodeInterface, parent js.Value) {
    oldLen := len(oldChildren)
    newLen := len(newChildren)
    maxLen := oldLen
    if newLen > maxLen {
        maxLen = newLen
    }

    oldChildMap := make(map[string]NodeInterface)
    for _, child := range oldChildren {
        oldChildMap[child.GetBindingID()] = child
    }

    for i := 0; i < maxLen; i++ {
        if i >= oldLen {
            // New child added
            EnsureBindingIDs(newChildren[i])
            newChildDom := renderNodeToDOM(newChildren[i])
            parent.Call("appendChild", newChildDom)
            registerDOMElement(newChildren[i], newChildDom)
        } else if i >= newLen {
            // Old child removed
            oldChildDom := getDOMElement(oldChildren[i])
            if !oldChildDom.IsUndefined() && !oldChildDom.IsNull() {
                parent.Call("removeChild", oldChildDom)
            }
            unregisterDOMElement(oldChildren[i])
        } else {
            // Both children exist: diff them
            diffAndUpdate(oldChildren[i], newChildren[i])
        }
    }
}

// updateAttributes updates the attributes of a DOM element based on the differences.
func updateAttributes(domElement js.Value, oldAttrs, newAttrs Attributes) {
    if domElement.IsUndefined() || domElement.IsNull() {
        fmt.Println("DOM Element is undefined or null in updateAttributes")
        return
    }
    // Remove attributes not present in newAttrs
    for key := range oldAttrs {
        if _, exists := newAttrs[key]; !exists {
            domElement.Call("removeAttribute", key)
        }
    }

    // Set new or changed attributes
    for key, newValue := range newAttrs {
        oldValue, exists := oldAttrs[key]
        if !exists || oldValue != newValue {
            domElement.Call("setAttribute", key, newValue)
        }
    }
}

// renderNodeToDOM creates a DOM element from a NodeInterface.
func renderNodeToDOM(node NodeInterface) js.Value {
    switch n := node.(type) {
    case *TextNode:
        domElement := js.Global().Get("document").Call("createTextNode", n.content)
        registerDOMElement(n, domElement)
        return domElement
    case *Node:
        element := js.Global().Get("document").Call("createElement", n.Tag)
        // Set attributes
        for key, value := range n.Attributes {
            element.Call("setAttribute", key, value)
        }
        // Register in domRegistry
        registerDOMElement(n, element)
        // Recursively append children
        for _, child := range n.Children {
            childElement := renderNodeToDOM(child)
            element.Call("appendChild", childElement)
        }
        return element
    default:
        return js.Value{}
    }
}

// getDOMElement retrieves the DOM element corresponding to a NodeInterface using its binding ID.
func getDOMElement(node NodeInterface) js.Value {
    bindingID := node.GetBindingID()
    if bindingID == "" {
        return js.Value{}
    }
    if element, exists := domRegistry[bindingID]; exists && !element.IsUndefined() && !element.IsNull() {
        return element
    }
    // As a fallback, query the DOM
    element := js.Global().Get("document").Call("querySelector", fmt.Sprintf(`[data-go_binding_id="%s"]`, bindingID))
    if element.IsUndefined() || element.IsNull() {
        fmt.Printf("Element with binding ID %s not found in the DOM.\n", bindingID)
        return js.Value{}
    }
    domRegistry[bindingID] = element
    return element
}

// registerDOMElement registers a DOM element for a given NodeInterface in the domRegistry.
func registerDOMElement(node NodeInterface, element js.Value) {
    bindingID := node.GetBindingID()
    if bindingID != "" {
        domRegistry[bindingID] = element
    }
}

// unregisterDOMElement removes a NodeInterface from the domRegistry.
func unregisterDOMElement(node NodeInterface) {
    bindingID := node.GetBindingID()
    if bindingID != "" {
        delete(domRegistry, bindingID)
    }
}
