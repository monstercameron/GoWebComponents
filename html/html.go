package html

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// TextNode represents plain text content within an HTML structure.
type TextNode struct {
	Text         string
	id           string // Unique ID for the node
	Dependencies []string
}

// Render returns the escaped text content of a TextNode, wrapped in a span with data attributes.
func (t *TextNode) Render() (string, error) {
	var htmlBuilder strings.Builder

	// Open span tag with data attributes
	htmlBuilder.WriteString("<span")
	htmlBuilder.WriteString(fmt.Sprintf(` data-node-id="%s"`, t.id))

	fmt.Println("Text Node Dependencies:", t.Dependencies)
	if len(t.Dependencies) > 0 {
		htmlBuilder.WriteString(fmt.Sprintf(` data-dependencies="%s"`, strings.Join(t.Dependencies, ",")))
	}

	htmlBuilder.WriteString(">")

	// Add escaped text content
	htmlBuilder.WriteString(html.EscapeString(t.Text))

	// Close span tag
	htmlBuilder.WriteString("</span>")

	return htmlBuilder.String(), nil
}

// GetDependencies returns an empty slice since TextNode does not have dependencies.
func (t *TextNode) GetDependencies() []string {
	return []string{}
}

// GetID returns the unique ID of the TextNode.
func (t *TextNode) GetID() string {
	return t.id
}

// ElementNode represents an HTML element with its tag name, attributes, and child nodes.
type ElementNode struct {
	TagName      string
	Attributes   map[string]string
	Children     []Node
	Dependencies []string
	id           string // Unique ID for the node
}

// GetDependencies returns the list of dependencies of the ElementNode.
func (e *ElementNode) GetDependencies() []string {
	return e.Dependencies
}

// GetID returns the unique ID of the ElementNode.
func (e *ElementNode) GetID() string {
	return e.id
}

// Render constructs and returns the HTML string representation of an ElementNode.
func (e *ElementNode) Render() (string, error) {
	// Validate the tag name to ensure it's a valid HTML tag.
	if err := validateTagName(e.TagName); err != nil {
		return "", err
	}

	var htmlBuilder strings.Builder

	// Write opening tag.
	htmlBuilder.WriteString("<")
	htmlBuilder.WriteString(e.TagName)

	// Add attributes to the tag.
	for attrName, attrValue := range e.Attributes {
		// Validate each attribute name to ensure it's safe for HTML.
		if err := validateAttributeName(attrName); err != nil {
			return "", err
		}
		// Use fmt.Sprintf for properly formatted attributes.
		htmlBuilder.WriteString(fmt.Sprintf(` %s="%s"`, attrName, html.EscapeString(attrValue)))
	}

	// Handle void elements (self-closing tags).
	if len(e.Children) == 0 && isVoidElement(e.TagName) {
		htmlBuilder.WriteString("/>")
		return htmlBuilder.String(), nil
	}

	htmlBuilder.WriteString(">")

	// Render child nodes recursively.
	for _, childNode := range e.Children {
		childHTML, err := childNode.Render()
		if err != nil {
			return "", fmt.Errorf("error rendering child of <%s>: %w", e.TagName, err)
		}
		htmlBuilder.WriteString(childHTML)
	}

	// Write closing tag.
	htmlBuilder.WriteString("</")
	htmlBuilder.WriteString(e.TagName)
	htmlBuilder.WriteString(">")

	return htmlBuilder.String(), nil
}

// AddChild appends a child node to the ElementNode's Children slice and registers it.
func (e *ElementNode) AddChild(child Node) {
	e.Children = append(e.Children, child)
	registerNode(child)
	fmt.Printf("Child added to <%s> element\n", e.TagName) // Added console log
}

// RemoveChild removes a child node from the ElementNode's Children slice and unregisters it.
func (e *ElementNode) RemoveChild(child Node) {
	for i, n := range e.Children {
		if n.GetID() == child.GetID() {
			// Remove child from slice
			e.Children = append(e.Children[:i], e.Children[i+1:]...)
			unregisterNode(child.GetID())
			fmt.Printf("Child removed from <%s> element\n", e.TagName) // Added console log
			return
		}
	}
}

// HTML creates and returns a new ElementNode with the given tag name, attributes, children, and dependencies.
func HTML(tagName string, attributes map[string]string, dependencies []string, children ...Node) *ElementNode {
	if attributes == nil {
		attributes = make(map[string]string)
	}

	// Add dependencies as a data attribute
	if len(dependencies) > 0 {
		attributes["data-dependencies"] = strings.Join(dependencies, ",")
	}

	node := &ElementNode{
		TagName:      tagName,
		Attributes:   attributes,
		Children:     children,
		Dependencies: dependencies,
		id:           fmt.Sprintf("%s-%s", tagName, GenerateUUID()),
	}

	// Add the node ID as a data attribute
	node.Attributes["data-node-id"] = node.id

	registerNode(node) // Register the node

	// Register all children nodes
	for _, child := range children {
		registerNode(child)
	}

	return node
}

// Text creates and returns a new TextNode with the given text content.
func Text(text string, dependencies []string) *TextNode {
	node := &TextNode{
		Text:         text,
		id:           fmt.Sprintf("text-%s", GenerateUUID()), // Assign a unique ID
		Dependencies: dependencies,
	}

	registerNode(node) // Register the node

	return node
}

// validateTagName checks if the tag name is valid using a regular expression.
// Valid HTML tag names must start with a letter and can only contain letters and numbers.
func validateTagName(tag string) error {
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`).MatchString(tag) {
		return fmt.Errorf("tag name '%s' is invalid: must start with a letter and contain only letters and numbers", tag)
	}
	return nil
}

// validateAttributeName checks if the attribute name is valid using a regular expression.
// Valid HTML attribute names must start with a letter and can contain letters, numbers, hyphens, and underscores.
func validateAttributeName(attr string) error {
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-_]*$`).MatchString(attr) {
		return fmt.Errorf("attribute name '%s' is invalid: must start with a letter and contain only letters, numbers, hyphens, and underscores", attr)
	}
	return nil
}

// isVoidElement checks if the given tag is a void element (self-closing tag).
// Void elements are HTML elements that do not have a closing tag.
func isVoidElement(tag string) bool {
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}
	return voidElements[tag]
}
