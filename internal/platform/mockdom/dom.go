package mockdom

import (
	"fmt"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// MockDOMNode implements runtime.DOMNode
type MockDOMNode struct {
	ID          int
	Tag         string
	TextContent string
	Attrs       map[string]string
	Props       map[string]interface{}
	Styles      map[string]string
	InnerHTML   string
	Children    []*MockDOMNode
	Parent      *MockDOMNode
}

var _ runtime.DOMNode = (*MockDOMNode)(nil)

func (n *MockDOMNode) IsNull() bool {
	return n == nil
}

func (n *MockDOMNode) Equals(other runtime.DOMNode) bool {
	if other == nil {
		return n == nil
	}
	otherMock, ok := other.(*MockDOMNode)
	if !ok {
		return false
	}
	return n.ID == otherMock.ID
}

// MockDOMAdapter implements runtime.DOMAdapter
type MockDOMAdapter struct {
	mu          sync.Mutex
	nodeCounter int
	operations  []DOMOperation
	nodeMap     map[int]*MockDOMNode
}

var _ runtime.DOMAdapter = (*MockDOMAdapter)(nil)

// DOMOperation records a DOM operation for testing
type DOMOperation struct {
	Type      string
	NodeID    int
	ParentID  int
	Data      interface{}
	Timestamp time.Time
}

// NewMockDOMAdapter creates an empty MockDOMAdapter suitable for unit testing DOM operations.
func NewMockDOMAdapter() *MockDOMAdapter {
	return &MockDOMAdapter{
		nodeMap:    make(map[int]*MockDOMNode),
		operations: make([]DOMOperation, 0),
	}
}

func (a *MockDOMAdapter) recordOp(opType string, nodeID int, data interface{}) {
	// TODO: guard operations with a lock; concurrent adapter calls can race when appending
	a.operations = append(a.operations, DOMOperation{
		Type:      opType,
		NodeID:    nodeID,
		Data:      data,
		Timestamp: time.Now(),
	})
}

func (a *MockDOMAdapter) CreateElement(tag string) runtime.DOMNode {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.nodeCounter++
	node := &MockDOMNode{
		ID:       a.nodeCounter,
		Tag:      tag,
		Attrs:    make(map[string]string),
		Props:    make(map[string]interface{}),
		Styles:   make(map[string]string),
		Children: make([]*MockDOMNode, 0),
	}
	a.nodeMap[node.ID] = node
	a.recordOp("createElement", node.ID, tag)
	return node
}

func (a *MockDOMAdapter) CreateTextNode(text string) runtime.DOMNode {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.nodeCounter++
	node := &MockDOMNode{
		ID:          a.nodeCounter,
		Tag:         "#text",
		TextContent: text,
		Attrs:       make(map[string]string),
		Props:       make(map[string]interface{}),
	}
	a.nodeMap[node.ID] = node
	a.recordOp("createTextNode", node.ID, text)
	return node
}

func (a *MockDOMAdapter) SetAttribute(node runtime.DOMNode, name, value string) {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		n.Attrs[name] = value
		a.recordOp("setAttribute", n.ID, map[string]string{name: value})
	}
}

func (a *MockDOMAdapter) RemoveAttribute(node runtime.DOMNode, name string) {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		delete(n.Attrs, name)
		a.recordOp("removeAttribute", n.ID, name)
	}
}

func (a *MockDOMAdapter) SetProperty(node runtime.DOMNode, name string, value interface{}) {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		n.Props[name] = value
		a.recordOp("setProperty", n.ID, map[string]interface{}{name: value})
	}
}

func (a *MockDOMAdapter) GetProperty(node runtime.DOMNode, name string) interface{} {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		switch name {
		case "nodeType":
			if n.Tag == "#text" {
				return 3
			}
			return 1
		case "tagName":
			return n.Tag
		case "nodeName":
			return n.Tag
		case "textContent":
			return n.TextContent
		case "className":
			return n.Attrs["class"]
		case "htmlFor":
			return n.Attrs["for"]
		}
		if value, ok := n.Attrs[name]; ok {
			return value
		}
		return n.Props[name]
	}
	return nil
}

func (a *MockDOMAdapter) AppendChild(parent, child runtime.DOMNode) {
	p, pok := parent.(*MockDOMNode)
	c, cok := child.(*MockDOMNode)
	if pok && cok {
		a.mu.Lock()
		defer a.mu.Unlock()
		p.Children = append(p.Children, c)
		c.Parent = p
		a.recordOp("appendChild", c.ID, map[string]int{"parentID": p.ID})
	}
}

func (a *MockDOMAdapter) RemoveChild(parent, child runtime.DOMNode) {
	p, pok := parent.(*MockDOMNode)
	c, cok := child.(*MockDOMNode)
	if pok && cok {
		a.mu.Lock()
		defer a.mu.Unlock()
		for i, ch := range p.Children {
			if ch.ID == c.ID {
				p.Children = append(p.Children[:i], p.Children[i+1:]...)
				c.Parent = nil
				break
			}
		}
		a.recordOp("removeChild", c.ID, map[string]int{"parentID": p.ID})
	}
}

func (a *MockDOMAdapter) InsertBefore(parent, newNode, referenceNode runtime.DOMNode) {
	p, pok := parent.(*MockDOMNode)
	n, nok := newNode.(*MockDOMNode)
	r, rok := referenceNode.(*MockDOMNode)
	if pok && nok && rok {
		a.mu.Lock()
		defer a.mu.Unlock()
		for i, ch := range p.Children {
			if ch.ID == r.ID {
				p.Children = append(p.Children[:i], append([]*MockDOMNode{n}, p.Children[i:]...)...)
				n.Parent = p
				break
			}
		}
		a.recordOp("insertBefore", n.ID, map[string]int{"parentID": p.ID, "beforeID": r.ID})
	}
}

func (a *MockDOMAdapter) ReplaceChild(parent, newNode, oldNode runtime.DOMNode) {
	p, pok := parent.(*MockDOMNode)
	n, nok := newNode.(*MockDOMNode)
	o, ook := oldNode.(*MockDOMNode)
	if pok && nok && ook {
		a.mu.Lock()
		defer a.mu.Unlock()
		for i, ch := range p.Children {
			if ch.ID == o.ID {
				p.Children[i] = n
				n.Parent = p
				o.Parent = nil
				break
			}
		}
		a.recordOp("replaceChild", n.ID, map[string]int{"parentID": p.ID, "oldID": o.ID})
	}
}

func (a *MockDOMAdapter) GetParent(node runtime.DOMNode) runtime.DOMNode {
	if n, ok := node.(*MockDOMNode); ok {
		return n.Parent
	}
	return nil
}

func (a *MockDOMAdapter) GetChildren(node runtime.DOMNode) []runtime.DOMNode {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		result := make([]runtime.DOMNode, len(n.Children))
		for i, child := range n.Children {
			result[i] = child
		}
		return result
	}
	return nil
}

func (a *MockDOMAdapter) GetFirstChild(node runtime.DOMNode) runtime.DOMNode {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		if len(n.Children) > 0 {
			return n.Children[0]
		}
	}
	return nil
}

func (a *MockDOMAdapter) GetNextSibling(node runtime.DOMNode) runtime.DOMNode {
	if n, ok := node.(*MockDOMNode); ok && n.Parent != nil {
		a.mu.Lock()
		defer a.mu.Unlock()
		for i, child := range n.Parent.Children {
			if child.ID == n.ID && i+1 < len(n.Parent.Children) {
				return n.Parent.Children[i+1]
			}
		}
	}
	return nil
}

func (a *MockDOMAdapter) SetStyle(node runtime.DOMNode, property, value string) {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		n.Styles[property] = value
		a.recordOp("setStyle", n.ID, map[string]string{property: value})
	}
}

func (a *MockDOMAdapter) SetStyles(node runtime.DOMNode, styles map[string]string) {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		for k, v := range styles {
			n.Styles[k] = v
		}
		a.recordOp("setStyles", n.ID, styles)
	}
}

func (a *MockDOMAdapter) SetInnerHTML(node runtime.DOMNode, html string) {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		n.InnerHTML = html
		if html == "" {
			for _, child := range n.Children {
				child.Parent = nil
			}
			n.Children = n.Children[:0]
			n.TextContent = ""
		}
		a.recordOp("setInnerHTML", n.ID, html)
	}
}

func (a *MockDOMAdapter) SetTextContent(node runtime.DOMNode, text string) {
	if n, ok := node.(*MockDOMNode); ok {
		a.mu.Lock()
		defer a.mu.Unlock()
		n.TextContent = text
		a.recordOp("setTextContent", n.ID, text)
	}
}

func (a *MockDOMAdapter) WrapFunction(fn interface{}) interface{} {
	// For mock DOM, we just return the function as is
	return fn
}

// Helper methods for testing
func (a *MockDOMAdapter) GetOperations() []DOMOperation {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]DOMOperation(nil), a.operations...)
}

func (a *MockDOMAdapter) GetNode(id int) *MockDOMNode {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.nodeMap[id]
}

func (a *MockDOMAdapter) ClearOperations() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.operations = make([]DOMOperation, 0)
}

func (a *MockDOMAdapter) AssertOperation(index int, expectedType string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index >= len(a.operations) {
		return fmt.Errorf("operation index %d out of bounds (have %d operations)", index, len(a.operations))
	}
	op := a.operations[index]
	if op.Type != expectedType {
		return fmt.Errorf("expected operation type %s, got %s", expectedType, op.Type)
	}
	return nil
}
