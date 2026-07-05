package mockdom

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// CreateHTMLSubtree parses one serialized HTML subtree into mock nodes so the
// native suite exercises the same serialized-mount strategy the browser
// adapter uses (one parse, then the runtime's binding walk).
func (parseA *MockDOMAdapter) CreateHTMLSubtree(parseHTML string) runtime.DOMNode {
	parseContext := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	parseNodes, parseErr := html.ParseFragment(strings.NewReader(parseHTML), parseContext)
	if parseErr != nil || len(parseNodes) == 0 {
		return (*MockDOMNode)(nil)
	}
	parseRoot := parseA.buildParsedNode(parseNodes[0])
	if parseRoot == nil {
		return (*MockDOMNode)(nil)
	}
	return parseRoot
}

// buildParsedNode converts one parsed html.Node into a mock node through the
// adapter's regular creation methods so operation recording stays consistent.
func (parseA *MockDOMAdapter) buildParsedNode(parseSrc *html.Node) *MockDOMNode {
	switch parseSrc.Type {
	case html.ElementNode:
		parseNode := parseA.CreateElement(parseSrc.Data).(*MockDOMNode)
		for _, parseAttr := range parseSrc.Attr {
			parseA.SetAttribute(parseNode, parseAttr.Key, parseAttr.Val)
		}
		for parseChild := parseSrc.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			if parseBuilt := parseA.buildParsedNode(parseChild); parseBuilt != nil {
				parseA.AppendChild(parseNode, parseBuilt)
			}
		}
		return parseNode
	case html.TextNode:
		parseText := parseA.CreateTextNode(parseSrc.Data).(*MockDOMNode)
		return parseText
	default:
		return nil
	}
}
