//go:build js && wasm
// +build js,wasm

package render

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	xhtml "golang.org/x/net/html"
)

type SeededMarkup struct {
	HTML    string
	NodeIDs map[string]int
}

// SeedHTML replaces the fixture container children with parsed server markup.
func (f *Fixture) SeedHTML(markup string) SeededMarkup {
	f.tb.Helper()
	f.requireActive()
	clearMockChildren(f.container)

	result := SeededMarkup{
		HTML:    markup,
		NodeIDs: map[string]int{},
	}
	nodes, err := xhtml.ParseFragment(strings.NewReader(markup), &xhtml.Node{Type: xhtml.ElementNode, Data: "div"})
	if err != nil {
		f.tb.Fatalf("render fixture failed to parse seeded HTML: %v", err)
	}
	for _, node := range nodes {
		appendHTMLNode(f, f.container, node, result.NodeIDs)
	}
	return result
}

func clearMockChildren(parent *mockdom.MockDOMNode) {
	if parent == nil {
		return
	}
	for _, child := range parent.Children {
		child.Parent = nil
	}
	parent.Children = nil
	parent.InnerHTML = ""
	parent.TextContent = ""
}

func appendHTMLNode(f *Fixture, parent *mockdom.MockDOMNode, node *xhtml.Node, ids map[string]int) {
	if node == nil {
		return
	}
	switch node.Type {
	case xhtml.TextNode:
		if strings.TrimSpace(node.Data) == "" && node.Data == "" {
			return
		}
		textNode, _ := f.adapter.CreateTextNode(node.Data).(*mockdom.MockDOMNode)
		f.adapter.AppendChild(parent, textNode)
	case xhtml.ElementNode:
		element, _ := f.adapter.CreateElement(node.Data).(*mockdom.MockDOMNode)
		for _, attr := range node.Attr {
			f.adapter.SetAttribute(element, attr.Key, attr.Val)
			if attr.Key == "id" {
				ids[attr.Val] = element.ID
			}
		}
		f.adapter.AppendChild(parent, element)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			appendHTMLNode(f, element, child, ids)
		}
	}
}
