package html

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
	"syscall/js"
)

// Package html provides a simple HTML generation library with a focus on safety and correctness.

// Node interface represents any element in the HTML structure, including both text and element nodes.
type Node interface {
	Render() (string, error)
}

type useState[T any] struct {
	value T
	mutex sync.RWMutex
}

func (s *useState[T]) get() T {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.value
}

func (s *useState[T]) set(newValue T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.value = newValue
}

// UseState creates a new state with an initial value and returns the value and a setter function
func UseState[T any](initialValue T) (*T, func(T)) {
	state := &useState[T]{value: initialValue}
	setter := func(newValue T) {
		state.set(newValue)
	}
	return &state.value, setter
}

// WasmFunc wraps a Go function to be callable from JavaScript and automatically sets it as a global function
func WasmFunc(name string, f func(args []js.Value) interface{}) {
    js.Global().Set(name, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        return f(args)
    }))
}

// UpdateElement updates the inner HTML of an element
func UpdateElement(id string, content interface{}) {
	js.Global().Get("document").Call("getElementById", id).Set("innerHTML", fmt.Sprint(content))
}

// TextNode represents plain text content within an HTML structure.
type TextNode struct {
	Text string
}

// Render returns the escaped text content of a TextNode.
func (t *TextNode) Render() (string, error) {
	return html.EscapeString(t.Text), nil
}

// ElementNode represents an HTML element with its tag name, attributes, and child nodes.
type ElementNode struct {
	TagName    string
	Attributes map[string]string
	Children   []Node
}

// Render constructs and returns the HTML string representation of an ElementNode.
func (e *ElementNode) Render() (string, error) {
	if err := validateTagName(e.TagName); err != nil {
		return "", err
	}

	var htmlBuilder strings.Builder

	// Write opening tag
	htmlBuilder.WriteString("<")
	htmlBuilder.WriteString(e.TagName)

	// Add attributes
	for attrName, attrValue := range e.Attributes {
		if err := validateAttributeName(attrName); err != nil {
			return "", err
		}
		// Use fmt.Sprintf for complex string formatting
		htmlBuilder.WriteString(fmt.Sprintf(` %s="%s"`, attrName, html.EscapeString(attrValue)))
	}

	// Handle void elements (self-closing tags)
	if len(e.Children) == 0 && isVoidElement(e.TagName) {
		htmlBuilder.WriteString("/>")
		return htmlBuilder.String(), nil
	}

	htmlBuilder.WriteString(">")

	// Render child nodes
	for _, childNode := range e.Children {
		childHTML, err := childNode.Render()
		if err != nil {
			return "", fmt.Errorf("error rendering child of <%s>: %w", e.TagName, err)
		}
		htmlBuilder.WriteString(childHTML)
	}

	// Write closing tag
	htmlBuilder.WriteString("</")
	htmlBuilder.WriteString(e.TagName)
	htmlBuilder.WriteString(">")

	return htmlBuilder.String(), nil
}

// AddChild appends a child node to the ElementNode's Children slice.
func (e *ElementNode) AddChild(child Node) {
	e.Children = append(e.Children, child)
}

// HTML creates and returns a new ElementNode with the given tag name, attributes, and children.
func HTML(tagName string, attributes map[string]string, children ...Node) *ElementNode {
	return &ElementNode{
		TagName:    tagName,
		Attributes: attributes,
		Children:   children,
	}
}

// Text creates and returns a new TextNode with the given text content.
func Text(text string) *TextNode {
	return &TextNode{
		Text: text,
	}
}

// isVoidElement checks if the given tag is a void element (self-closing tag).
func isVoidElement(tag string) bool {
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}
	return voidElements[tag]
}

// validateTagName checks if the tag name is valid using a regular expression.
func validateTagName(tag string) error {
	// Ensure tag starts with a letter and contains only letters and numbers
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`).MatchString(tag) {
		return fmt.Errorf("invalid tag name: %s", tag)
	}
	return nil
}

// validateAttributeName checks if the attribute name is valid using a regular expression.
func validateAttributeName(attr string) error {
	// Ensure attribute starts with a letter and contains only letters, numbers, hyphens, and underscores
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-_]*$`).MatchString(attr) {
		return fmt.Errorf("invalid attribute name: %s", attr)
	}
	return nil
}
