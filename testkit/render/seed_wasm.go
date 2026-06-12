//go:build js && wasm

package render

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type SeededMarkup struct {
	HTML    string
	NodeIDs map[string]int
}

// SeedHTML replaces the fixture container children with parsed server markup.
func (parseF *Fixture) SeedHTML(parseMarkup string) SeededMarkup {
	parseF.tb.Helper()
	parseF.requireActive()
	clearMockChildren(parseF.container)

	parseResult := SeededMarkup{
		HTML:    parseMarkup,
		NodeIDs: map[string]int{},
	}
	parseNodes, parseErr := xhtml.ParseFragment(strings.NewReader(parseMarkup), &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"})
	if parseErr != nil {
		parseF.tb.Fatalf("render fixture failed to parse seeded HTML: %v", parseErr)
	}
	for _, parseNode := range parseNodes {
		appendHTMLNode(parseF, parseF.container, parseNode, parseResult.NodeIDs)
	}
	return parseResult
}

func clearMockChildren(parseParent *mockdom.MockDOMNode) {
	if parseParent == nil {
		return
	}
	for _, parseChild := range parseParent.Children {
		parseChild.Parent = nil
	}
	parseParent.Children = nil
	parseParent.InnerHTML = ""
	parseParent.TextContent = ""
}

func appendHTMLNode(parseF *Fixture, parseParent *mockdom.MockDOMNode, parseNode *xhtml.Node, parseIds map[string]int) {
	if parseNode == nil {
		return
	}
	switch parseNode.Type {
	case xhtml.TextNode:
		if strings.TrimSpace(parseNode.Data) == "" && parseNode.Data == "" {
			return
		}
		parseTextNode, _ := parseF.adapter.CreateTextNode(parseNode.Data).(*mockdom.MockDOMNode)
		parseF.adapter.AppendChild(parseParent, parseTextNode)
	case xhtml.ElementNode:
		parseElement, _ := parseF.adapter.CreateElement(parseNode.Data).(*mockdom.MockDOMNode)
		for _, parseAttr := range parseNode.Attr {
			parseF.adapter.SetAttribute(parseElement, parseAttr.Key, parseAttr.Val)
			if parseAttr.Key == "id" {
				parseIds[parseAttr.Val] = parseElement.ID
			}
		}
		parseF.adapter.AppendChild(parseParent, parseElement)
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			appendHTMLNode(parseF, parseElement, parseChild, parseIds)
		}
	}
}
