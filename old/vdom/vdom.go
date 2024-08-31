package vdom

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
)

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
	FindByID(id interface{}) (NodeInterface, error)
	PrintTree(level int)
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

// TextNode represents a text node
type TextNode struct {
	Content string
}

var (
	validTagNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)
	validAttrKeyRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-_]*$`)
)

func validateTagName(tagName string) error {
	if tagName == "" {
		return errors.New("tag name cannot be empty")
	}
	if !validTagNameRegex.MatchString(tagName) {
		return fmt.Errorf("invalid tag name: %s", tagName)
	}
	return nil
}

func validateAttributeKey(key string) error {
	if key == "" {
		return errors.New("attribute key cannot be empty")
	}
	if !validAttrKeyRegex.MatchString(key) {
		return fmt.Errorf("invalid attribute key: %s", key)
	}
	return nil
}

func validateAttributeValue(value string) error {
	if strings.ContainsAny(value, "\u0000\u0001\u0002\u0003\u0004") {
		return errors.New("attribute value contains invalid characters")
	}
	return nil
}

// Tag creates a new ElementNode with attributes and children
func Tag(tagName string, attributes map[string]string, children ...interface{}) NodeInterface {
	node := &ElementNode{
		ID:         GenerateID(tagName),
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
	if err := validateTagName(tagName); err != nil {
		return err
	}
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
	if err := validateAttributeKey(key); err != nil {
		return err
	}
	if err := validateAttributeValue(value); err != nil {
		return err
	}
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

func (n *ElementNode) FindByID(id interface{}) (NodeInterface, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.ID == id {
		return n, nil
	}

	for _, child := range n.Children {
		if elem, ok := child.(*ElementNode); ok {
			found, err := elem.FindByID(id)
			if err == nil && found != nil {
				return found, nil
			}
		}
	}

	return nil, fmt.Errorf("node with ID %v not found", id)
}

func (n *ElementNode) PrintTree(level int) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	indent := strings.Repeat("  ", level)
	fmt.Printf("%s%s\n", indent, n.TagName)
	for _, child := range n.Children {
		child.PrintTree(level + 1)
	}
}

func (n *ElementNode) Render(level int) string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	indent := strings.Repeat("  ", level)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s<%s", indent, html.EscapeString(n.TagName)))

	for key, value := range n.Attributes {
		sb.WriteString(fmt.Sprintf(" %s=\"%s\"", html.EscapeString(key), html.EscapeString(value)))
	}

	if len(n.Children) == 0 && n.Value == nil {
		sb.WriteString(" />")
	} else {
		sb.WriteString(">")
		if n.Value != nil {
			sb.WriteString(html.EscapeString(fmt.Sprintf("%v", n.Value)))
		}
		if len(n.Children) > 0 {
			sb.WriteString("\n")
			for _, child := range n.Children {
				sb.WriteString(child.Render(level + 1))
			}
			sb.WriteString(indent)
		}
		sb.WriteString(fmt.Sprintf("</%s>", html.EscapeString(n.TagName)))
	}
	sb.WriteString("\n")

	return sb.String()
}

// TextNode methods

func (n *TextNode) SetValue(value interface{}) {
	n.Content = fmt.Sprintf("%v", value)
}

func (n *TextNode) GetValue() interface{} {
	return n.Content
}

func (n *TextNode) SetTagName(tagName string) error {
	return errors.New("cannot set tag name on a text node")
}

func (n *TextNode) GetTagName() string {
	return ""
}

func (n *TextNode) SetAttribute(key, value string) error {
	return errors.New("cannot set attribute on a text node")
}

func (n *TextNode) GetAttributes() map[string]string {
	return nil
}

func (n *TextNode) AddChild(child NodeInterface) {
	// Do nothing, text nodes can't have children
}

func (n *TextNode) GetChildren() []NodeInterface {
	return nil
}

func (n *TextNode) FindByID(id interface{}) (NodeInterface, error) {
	return nil, errors.New("text node cannot have an ID")
}

func (n *TextNode) PrintTree(level int) {
	indent := strings.Repeat("  ", level)
	fmt.Printf("%s%s\n", indent, n.Content)
}

func (n *TextNode) Render(_ int) string {
	return html.EscapeString(n.Content)
}

func isTextNode(n NodeInterface) bool {
	_, ok := n.(*TextNode)
	return ok
}