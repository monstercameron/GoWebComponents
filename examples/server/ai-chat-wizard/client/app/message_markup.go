package app

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/sanitize"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	parseHTML "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// renderMessageMarkup makes rich text a sanitized, reconciler-owned node tree.
func renderMessageMarkup(parseMarkup string) ui.Node {
	parseContext := &parseHTML.Node{Type: parseHTML.ElementNode, Data: "div", DataAtom: atom.Div}
	parseNodes, parseErr := parseHTML.ParseFragment(strings.NewReader(sanitize.Sanitize(parseMarkup)), parseContext)
	if parseErr != nil {
		return html.Div(html.Props{}, html.Text(parseMarkup))
	}
	parseChildren := make([]ui.Node, 0, len(parseNodes))
	for parseIndex, parseNode := range parseNodes {
		parseChildren = append(parseChildren, renderMessageMarkupNode(parseNode, fmt.Sprint(parseIndex)))
	}
	return html.Div(html.Props{
		Class: "prose text-[1.0625rem] leading-[1.75] text-[#e8e7f2] min-w-0",
	}, parseChildren...)
}

// renderMessageMarkupNode converts sanitized markup and gives code controls their own lifecycle.
func renderMessageMarkupNode(parseNode *parseHTML.Node, parsePath string) ui.Node {
	if parseNode.Type == parseHTML.TextNode {
		return html.Text(parseNode.Data)
	}
	if parseNode.Type != parseHTML.ElementNode {
		return nil
	}
	parseProps := html.Props{Raw: map[string]any{}}
	for _, parseAttr := range parseNode.Attr {
		switch parseAttr.Key {
		case "class":
			parseProps.Class = parseAttr.Val
		case "id":
			parseProps.ID = parseAttr.Val
		default:
			parseProps.Raw[parseAttr.Key] = parseAttr.Val
		}
	}
	parseChildren := []ui.Node{}
	parseIndex := 0
	for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
		parseChildren = append(parseChildren, renderMessageMarkupNode(parseChild, fmt.Sprintf("%s.%d", parsePath, parseIndex)))
		parseIndex++
	}
	parseElement := html.Tag(parseNode.Data, parseProps, parseChildren...)
	if parseNode.Data == "pre" {
		// Mermaid remains in its existing renderer until the diagram migration;
		// plain code blocks no longer depend on a DOM-mutating observer.
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			for _, parseAttr := range parseChild.Attr {
				if parseAttr.Key == "class" && strings.Contains(" "+parseAttr.Val+" ", " language-mermaid ") {
					return parseElement
				}
			}
		}
		parseSource := getMessageMarkupText(parseNode)
		// A changed code sample must cancel its old clipboard task. Include its
		// position so identical sibling code samples do not collide as keys.
		parseKey := fmt.Sprintf("%s:%x", parsePath, sha256.Sum256([]byte(parseSource)))
		return html.WithKey(ui.CreateElement(renderMessageCodeBlock, messageCodeProps{parseElement: parseElement, parseSource: parseSource}), parseKey)
	}
	return parseElement
}

// getMessageMarkupText extracts decoded text without copying presentation markup.
func getMessageMarkupText(parseNode *parseHTML.Node) string {
	if parseNode.Type == parseHTML.TextNode {
		return parseNode.Data
	}
	var parseText strings.Builder
	for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
		parseText.WriteString(getMessageMarkupText(parseChild))
	}
	return parseText.String()
}

type messageCodeProps struct {
	parseElement ui.Node
	parseSource  string
}
