package html

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Node interface for both text and element nodes
type Node interface {
	Render() (string, error)
}

// TextNode struct represents plain text content within an HTML structure
type TextNode struct {
	Text string
}

// Render method for TextNode returns the escaped text content
func (t *TextNode) Render() (string, error) {
	return html.EscapeString(t.Text), nil
}

// ElementNode struct represents an HTML element
type ElementNode struct {
	TagName    string
	Attributes map[string]string
	Children   []Node
}

// Render method for ElementNode constructs the HTML string
func (e *ElementNode) Render() (string, error) {
	if err := validateTagName(e.TagName); err != nil {
		return "", err
	}

	var htmlBuilder strings.Builder

	htmlBuilder.WriteString("<")
	htmlBuilder.WriteString(e.TagName)

	for attrName, attrValue := range e.Attributes {
		if err := validateAttributeName(attrName); err != nil {
			return "", err
		}
		htmlBuilder.WriteString(fmt.Sprintf(` %s="%s"`, attrName, html.EscapeString(attrValue)))
	}

	if len(e.Children) == 0 && isVoidElement(e.TagName) {
		htmlBuilder.WriteString("/>")
		return htmlBuilder.String(), nil
	}

	htmlBuilder.WriteString(">")

	for _, childNode := range e.Children {
		childHTML, err := childNode.Render()
		if err != nil {
			return "", fmt.Errorf("error rendering child of <%s>: %w", e.TagName, err)
		}
		htmlBuilder.WriteString(childHTML)
	}

	htmlBuilder.WriteString("</")
	htmlBuilder.WriteString(e.TagName)
	htmlBuilder.WriteString(">")

	return htmlBuilder.String(), nil
}

// AddChild method to add a child node to the ElementNode
func (e *ElementNode) AddChild(child Node) {
	e.Children = append(e.Children, child)
}

// HTML function creates a new ElementNode
func HTML(tagName string, attributes map[string]string, children ...Node) *ElementNode {
	return &ElementNode{
		TagName:    tagName,
		Attributes: attributes,
		Children:   children,
	}
}

// Text function creates a new TextNode
func Text(text string) *TextNode {
	return &TextNode{
		Text: text,
	}
}

// isVoidElement checks if the given tag is a void element
func isVoidElement(tag string) bool {
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}
	return voidElements[tag]
}

// validateTagName checks if the tag name is valid
func validateTagName(tag string) error {
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`).MatchString(tag) {
		return fmt.Errorf("invalid tag name: %s", tag)
	}
	return nil
}

// validateAttributeName checks if the attribute name is valid
func validateAttributeName(attr string) error {
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-_]*$`).MatchString(attr) {
		return fmt.Errorf("invalid attribute name: %s", attr)
	}
	return nil
}