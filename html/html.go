package html

import (
	"fmt"
	"strings"
)

// Node interface for both text and element nodes
type Node interface {
	Render() string
}

// TextNode struct represents plain text content within an HTML structure
type TextNode struct {
	Text string // Represents the text content
}

// Render method for TextNode returns the text content
func (t *TextNode) Render() string {
	return t.Text
}

// ElementNode struct represents an HTML element
type ElementNode struct {
	TagName    string            // Represents the HTML tag name
	Attributes map[string]string // Represents HTML attributes
	Children   []Node            // Can hold both TextNode and ElementNode, representing mixed content
}

// Render method for ElementNode constructs the HTML string
func (e *ElementNode) Render() string {
	var htmlBuilder strings.Builder

	// Render opening tag with attributes
	var attrBuilder strings.Builder
	for attrName, attrValue := range e.Attributes {
		attrBuilder.WriteString(fmt.Sprintf(` %s="%s"`, attrName, attrValue))
	}
	htmlBuilder.WriteString(fmt.Sprintf("<%s%s>", e.TagName, attrBuilder.String()))

	// Render all children (both text and nested elements)
	for _, childNode := range e.Children {
		htmlBuilder.WriteString(childNode.Render())
	}

	// Render closing tag
	htmlBuilder.WriteString(fmt.Sprintf("</%s>", e.TagName))

	return htmlBuilder.String()
}

// AddChild method to add a child node to the ElementNode
func (e *ElementNode) AddChild(child Node) {
	e.Children = append(e.Children, child)
}

// Html function creates a new ElementNode, making the API more intuitive
func HTML(tagName string, attributes map[string]string, children ...Node) *ElementNode {
	return &ElementNode{
		TagName:    tagName,
		Attributes: attributes,
		Children:   children,
	}
}

// Text function creates a new TextNode, making the API more intuitive
func Text(text string) *TextNode {
	return &TextNode{
		Text: text,
	}
}

// Example usage of the composable HTML rendering
// func main() {
// 	// Create a <div> element with mixed content: text and a <strong> element
// 	div := Html("div", map[string]string{"class": "container"},
// 		Text("Hello, "),
// 		Html("strong", nil,
// 			Text("world"),
// 		),
// 		Text("!"),
// 	)

// 	// More complex example: <select> with multiple <option> elements
// 	selectElement := Html("select", map[string]string{"name": "options", "id": "selectElement"},
// 		Html("option", map[string]string{"value": "1"}, Text("Option 1")),
// 		Html("option", map[string]string{"value": "2"}, Text("Option 2")),
// 		Html("option", map[string]string{"value": "3"}, Text("Option 3")),
// 	)

// 	// Render the HTML and print it
// 	fmt.Println(div.Render())
// 	fmt.Println(selectElement.Render())
// }
