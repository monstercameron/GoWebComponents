package mockdom

import (
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// MockDOMNode implements runtime.DOMNode
type MockDOMNode struct {
	ID          int
	Tag         string
	TextContent string
	Attrs       map[string]string
	Props       map[string]any
	Styles      map[string]string
	InnerHTML   string
	Children    []*MockDOMNode
	Parent      *MockDOMNode
}

var _ runtime.DOMNode = (*MockDOMNode)(nil)

func (parseN *MockDOMNode) IsNull() bool {
	return parseN == nil
}

func (parseN *MockDOMNode) Equals(parseOther runtime.DOMNode) bool {
	if parseN == nil {
		return runtime.IsDOMNodeNull(parseOther)
	}
	if runtime.IsDOMNodeNull(parseOther) {
		return false
	}
	parseOtherMock, parseOk := parseOther.(*MockDOMNode)
	if !parseOk || parseOtherMock == nil {
		return false
	}
	return parseN.ID == parseOtherMock.ID
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
	Data      any
	Timestamp time.Time
}

// NewMockDOMAdapter creates an empty MockDOMAdapter suitable for unit testing DOM operations.
func NewMockDOMAdapter() *MockDOMAdapter {
	return &MockDOMAdapter{
		nodeMap:    make(map[int]*MockDOMNode),
		operations: make([]DOMOperation, 0),
	}
}

// recordOpLocked appends one operation entry while the adapter mutex is already held so DOM state and operation-log mutations stay serialized.
func (parseA *MockDOMAdapter) recordOpLocked(parseOpType string, parseNodeID int, parseData any) {
	parseA.operations = append(parseA.operations, DOMOperation{
		Type:      parseOpType,
		NodeID:    parseNodeID,
		Data:      parseData,
		Timestamp: time.Now(),
	})
}

func (parseA *MockDOMAdapter) CreateElement(parseTag string) runtime.DOMNode {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()

	parseA.nodeCounter++
	parseNode := &MockDOMNode{
		ID:       parseA.nodeCounter,
		Tag:      parseTag,
		Attrs:    make(map[string]string),
		Props:    make(map[string]any),
		Styles:   make(map[string]string),
		Children: make([]*MockDOMNode, 0),
	}
	parseA.nodeMap[parseNode.ID] = parseNode
	parseA.recordOpLocked("createElement", parseNode.ID, parseTag)
	return parseNode
}

func (parseA *MockDOMAdapter) CreateTextNode(parseText string) runtime.DOMNode {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()

	parseA.nodeCounter++
	parseNode := &MockDOMNode{
		ID:          parseA.nodeCounter,
		Tag:         "#text",
		TextContent: parseText,
		Attrs:       make(map[string]string),
		Props:       make(map[string]any),
	}
	parseA.nodeMap[parseNode.ID] = parseNode
	parseA.recordOpLocked("createTextNode", parseNode.ID, parseText)
	return parseNode
}

func (parseA *MockDOMAdapter) SetAttribute(parseNode runtime.DOMNode, parseName, parseValue string) {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		// Mirror the browser adapter: block javascript:/vbscript: URLs so the
		// client render path is testable for the same XSS guard natively.
		parseValue = runtime.SanitizeURLAttributeValue(parseName, parseValue)
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		parseN.Attrs[parseName] = parseValue
		parseA.recordOpLocked("setAttribute", parseN.ID, map[string]string{parseName: parseValue})
	}
}

// GetAttribute reports one stored attribute value for one mock DOM node.
func (parseA *MockDOMAdapter) GetAttribute(parseNode runtime.DOMNode, parseName string) string {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		return parseN.Attrs[parseName]
	}
	return ""
}

func (parseA *MockDOMAdapter) RemoveAttribute(parseNode runtime.DOMNode, parseName string) {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		delete(parseN.Attrs, parseName)
		parseA.recordOpLocked("removeAttribute", parseN.ID, parseName)
	}
}

func (parseA *MockDOMAdapter) SetProperty(parseNode runtime.DOMNode, parseName string, parseValue any) {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		parseN.Props[parseName] = parseValue
		parseA.recordOpLocked("setProperty", parseN.ID, map[string]any{parseName: parseValue})
	}
}

func (parseA *MockDOMAdapter) GetProperty(parseNode runtime.DOMNode, parseName string) any {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		switch parseName {
		case "nodeType":
			if parseN.Tag == "#text" {
				return 3
			}
			return 1
		case "tagName":
			return parseN.Tag
		case "nodeName":
			return parseN.Tag
		case "textContent":
			return parseN.TextContent
		case "className":
			return parseN.Attrs["class"]
		case "htmlFor":
			return parseN.Attrs["for"]
		}
		if parseValue, parseOk2 := parseN.Attrs[parseName]; parseOk2 {
			return parseValue
		}
		return parseN.Props[parseName]
	}
	return nil
}

// QuerySelector resolves one simple selector against the current mock DOM tree.
//
// The mock adapter only needs the subset used by framework tests: `#id` lookup
// plus a basic tag-name fallback when no id selector is requested.
func (parseA *MockDOMAdapter) QuerySelector(parseSelector string) any {
	parseTrimmedSelector := strings.TrimSpace(parseSelector)
	if parseTrimmedSelector == "" {
		return nil
	}

	parseA.mu.Lock()
	defer parseA.mu.Unlock()

	if after, ok := strings.CutPrefix(parseTrimmedSelector, "#"); ok {
		parseWantedID := strings.TrimSpace(after)
		if parseWantedID == "" {
			return nil
		}
		for _, parseNode := range parseA.nodeMap {
			if parseNode != nil && strings.TrimSpace(parseNode.Attrs["id"]) == parseWantedID {
				return parseNode
			}
		}
		return nil
	}

	parseWantedTag := strings.ToLower(parseTrimmedSelector)
	for _, parseNode := range parseA.nodeMap {
		if parseNode != nil && strings.ToLower(strings.TrimSpace(parseNode.Tag)) == parseWantedTag {
			return parseNode
		}
	}
	return nil
}

func (parseA *MockDOMAdapter) AppendChild(parseParent, parseChild runtime.DOMNode) {
	parseP, parsePok := parseParent.(*MockDOMNode)
	parseC, parseCok := parseChild.(*MockDOMNode)
	if parsePok && parseCok {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		if parseC.Parent != nil {
			for parseIndex, parseExistingChild := range parseC.Parent.Children {
				if parseExistingChild.ID == parseC.ID {
					parseC.Parent.Children = append(parseC.Parent.Children[:parseIndex], parseC.Parent.Children[parseIndex+1:]...)
					break
				}
			}
		}
		parseP.Children = append(parseP.Children, parseC)
		parseC.Parent = parseP
		parseA.recordOpLocked("appendChild", parseC.ID, map[string]int{"parentID": parseP.ID})
	}
}

func (parseA *MockDOMAdapter) RemoveChild(parseParent, parseChild runtime.DOMNode) {
	parseP, parsePok := parseParent.(*MockDOMNode)
	parseC, parseCok := parseChild.(*MockDOMNode)
	if parsePok && parseCok {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		for parseI, parseCh := range parseP.Children {
			if parseCh.ID == parseC.ID {
				parseP.Children = append(parseP.Children[:parseI], parseP.Children[parseI+1:]...)
				parseC.Parent = nil
				break
			}
		}
		parseA.recordOpLocked("removeChild", parseC.ID, map[string]int{"parentID": parseP.ID})
	}
}

func (parseA *MockDOMAdapter) InsertBefore(parseParent, parseNewNode, parseReferenceNode runtime.DOMNode) {
	parseP, parsePok := parseParent.(*MockDOMNode)
	parseN, parseNok := parseNewNode.(*MockDOMNode)
	parseR, parseRok := parseReferenceNode.(*MockDOMNode)
	if parsePok && parseNok && parseRok {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		if parseN.Parent != nil {
			for parseIndex, parseExistingChild := range parseN.Parent.Children {
				if parseExistingChild.ID == parseN.ID {
					parseN.Parent.Children = append(parseN.Parent.Children[:parseIndex], parseN.Parent.Children[parseIndex+1:]...)
					break
				}
			}
		}
		for parseI, parseCh := range parseP.Children {
			if parseCh.ID == parseR.ID {
				parseP.Children = append(parseP.Children[:parseI], append([]*MockDOMNode{parseN}, parseP.Children[parseI:]...)...)
				parseN.Parent = parseP
				break
			}
		}
		parseA.recordOpLocked("insertBefore", parseN.ID, map[string]int{"parentID": parseP.ID, "beforeID": parseR.ID})
	}
}

func (parseA *MockDOMAdapter) ReplaceChild(parseParent, parseNewNode, parseOldNode runtime.DOMNode) {
	parseP, parsePok := parseParent.(*MockDOMNode)
	parseN, parseNok := parseNewNode.(*MockDOMNode)
	parseO, parseOok := parseOldNode.(*MockDOMNode)
	if parsePok && parseNok && parseOok {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		for parseI, parseCh := range parseP.Children {
			if parseCh.ID == parseO.ID {
				parseP.Children[parseI] = parseN
				parseN.Parent = parseP
				parseO.Parent = nil
				break
			}
		}
		parseA.recordOpLocked("replaceChild", parseN.ID, map[string]int{"parentID": parseP.ID, "oldID": parseO.ID})
	}
}

func (parseA *MockDOMAdapter) GetParent(parseNode runtime.DOMNode) runtime.DOMNode {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		return parseN.Parent
	}
	return nil
}

func (parseA *MockDOMAdapter) GetChildren(parseNode runtime.DOMNode) []runtime.DOMNode {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		parseResult := make([]runtime.DOMNode, len(parseN.Children))
		for parseI, parseChild := range parseN.Children {
			parseResult[parseI] = parseChild
		}
		return parseResult
	}
	return nil
}

func (parseA *MockDOMAdapter) GetFirstChild(parseNode runtime.DOMNode) runtime.DOMNode {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		if len(parseN.Children) > 0 {
			return parseN.Children[0]
		}
	}
	return nil
}

func (parseA *MockDOMAdapter) GetNextSibling(parseNode runtime.DOMNode) runtime.DOMNode {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk && parseN.Parent != nil {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		for parseI, parseChild := range parseN.Parent.Children {
			if parseChild.ID == parseN.ID && parseI+1 < len(parseN.Parent.Children) {
				return parseN.Parent.Children[parseI+1]
			}
		}
	}
	return nil
}

func (parseA *MockDOMAdapter) SetStyle(parseNode runtime.DOMNode, parseProperty, parseValue string) {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		parseN.Styles[parseProperty] = parseValue
		parseA.recordOpLocked("setStyle", parseN.ID, map[string]string{parseProperty: parseValue})
	}
}

func (parseA *MockDOMAdapter) SetStyles(parseNode runtime.DOMNode, parseStyles map[string]string) {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		maps.Copy(parseN.Styles, parseStyles)
		parseA.recordOpLocked("setStyles", parseN.ID, parseStyles)
	}
}

func (parseA *MockDOMAdapter) SetInnerHTML(parseNode runtime.DOMNode, parseHtml string) {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		parseN.InnerHTML = parseHtml
		if parseHtml == "" {
			for _, parseChild := range parseN.Children {
				parseChild.Parent = nil
			}
			parseN.Children = parseN.Children[:0]
			parseN.TextContent = ""
		}
		parseA.recordOpLocked("setInnerHTML", parseN.ID, parseHtml)
	}
}

func (parseA *MockDOMAdapter) SetTextContent(parseNode runtime.DOMNode, parseText string) {
	if parseN, parseOk := parseNode.(*MockDOMNode); parseOk {
		parseA.mu.Lock()
		defer parseA.mu.Unlock()
		parseN.TextContent = parseText
		parseA.recordOpLocked("setTextContent", parseN.ID, parseText)
	}
}

func (parseA *MockDOMAdapter) WrapFunction(parseFn any) any {
	// For mock DOM, we just return the function as is
	return parseFn
}

// GetOperations returns one snapshot copy of the recorded DOM operation log for assertions.
func (parseA *MockDOMAdapter) GetOperations() []DOMOperation {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()
	return append([]DOMOperation(nil), parseA.operations...)
}

// GetNode returns one live node handle for read-only assertions against adapter-owned state.
func (parseA *MockDOMAdapter) GetNode(parseId int) *MockDOMNode {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()
	return parseA.nodeMap[parseId]
}

// ClearOperations resets the recorded DOM operation log.
func (parseA *MockDOMAdapter) ClearOperations() {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()
	parseA.operations = make([]DOMOperation, 0)
}

// AssertOperation verifies one recorded operation type at one log index.
func (parseA *MockDOMAdapter) AssertOperation(parseIndex int, parseExpectedType string) error {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()
	if parseIndex >= len(parseA.operations) {
		return fmt.Errorf("operation index %d out of bounds (have %d operations)", parseIndex, len(parseA.operations))
	}
	parseOp := parseA.operations[parseIndex]
	if parseOp.Type != parseExpectedType {
		return fmt.Errorf("expected operation type %s, got %s", parseExpectedType, parseOp.Type)
	}
	return nil
}
