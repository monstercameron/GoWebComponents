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
// This package is designed for use in WebAssembly (Wasm) environments, enabling state management
// and dynamic HTML generation directly from Go.

// Node interface represents any element in the HTML structure, including both text and element nodes.
type Node interface {
	Render() (string, error)
}

// useState is a thread-safe structure to manage state values.
type useState[T any] struct {
	value T
	mutex sync.RWMutex
}

// get returns the current state value in a thread-safe manner.
func (s *useState[T]) get() T {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.value
}

// set updates the state value in a thread-safe manner and triggers an update of associated elements.
func (s *useState[T]) set(newValue T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.value = newValue
}

// UpdateStateElements updates all elements in the DOM that are associated with a specific state variable.
func updateElementsWithState(stateID string, value string) {
	selector := fmt.Sprintf("[data-state~='%s']", stateID)
	elements := js.Global().Get("document").Call("querySelectorAll", selector)
	length := elements.Length()
	
	for i := 0; i < length; i++ {
		element := elements.Index(i)
		element.Set("textContent", value)
	}
}

// UseState creates a new state with an initial value and returns a pointer to the state value
// and a setter function to update the state.
func UseState[T any](initialValue T) (*T, func(T), string) {
	state := &useState[T]{value: initialValue}
	stateID := fmt.Sprintf("state-%p", state) // Generate a unique ID based on the state's memory address
	
	setter := func(newValue T) {
		state.set(newValue)
		updateElementsWithState(stateID, fmt.Sprint(newValue))
	}
	
	return &state.value, setter, stateID
}

// WasmFunc wraps a Go function to be callable from JavaScript and automatically sets it as a global function.
func WasmFunc(name string, f func()) {
	js.Global().Set(name, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		f()
		return nil
	}))
}

// UpdateElement updates the inner HTML of an element with the given ID.
// It accepts any type as value and converts it to a string.
func UpdateElement(id string, value interface{}) {
	js.Global().Get("document").Call("getElementById", id).Set("innerHTML", fmt.Sprint(value))
}

// TextNode represents plain text content within an HTML structure.
type TextNode struct {
	Text string
}

// Render returns the escaped text content of a TextNode.
// It ensures that any special HTML characters are properly escaped.
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

// AddChild appends a child node to the ElementNode's Children slice.
func (e *ElementNode) AddChild(child Node) {
	e.Children = append(e.Children, child)
}

// HTML creates and returns a new ElementNode with the given tag name, attributes, and children.
// This function helps to construct a DOM tree dynamically.
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

// validateTagName checks if the tag name is valid using a regular expression.
// Valid HTML tag names must start with a letter and can only contain letters and numbers.
func validateTagName(tag string) error {
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`).MatchString(tag) {
		return fmt.Errorf("invalid tag name: %s", tag)
	}
	return nil
}

// validateAttributeName checks if the attribute name is valid using a regular expression.
// Valid HTML attribute names must start with a letter and can contain letters, numbers, hyphens, and underscores.
func validateAttributeName(attr string) error {
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-_]*$`).MatchString(attr) {
		return fmt.Errorf("invalid attribute name: %s", attr)
	}
	return nil
}
