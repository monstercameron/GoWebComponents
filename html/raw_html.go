package html

import (
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/monstercameron/GoWebComponents/v6/sanitize"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Raw / markup nodes (G3).
//
// There is deliberately no innerHTML sink on the render path (see the runtime's
// TestDOMAdapterHasNoRawHTMLSink guard): untrusted strings must never be parsed
// as markup by the browser. RawHTML therefore parses a markup string into the
// framework's own node tree in Go — every element/text/attribute is rebuilt
// through the safe Tag/Text constructors — so "render this HTML" (Markdown
// output, SVG, rich text) works without ever reopening an XSS sink.
//
// RawHTML sanitizes its input with the sanitize package's default allowlist
// (scripts, event-handler attributes, javascript: URLs, <svg>/<style>/<iframe>,
// etc. are stripped). Use it for any content that is not fully under your
// control. For trusted, author-controlled markup that the allowlist would strip
// (notably inline SVG for icons/charts), use RawHTMLUnsafe.

// RawHTML parses sanitized markup into nodes. Safe for untrusted input.
//
//	Div(html.RawHTML(userSuppliedMarkdownHTML)...)
func RawHTML(parseMarkup string) []ui.Node {
	return parseMarkupToNodes(sanitize.Sanitize(parseMarkup))
}

// RawHTMLWith parses markup sanitized under a caller-provided policy.
func RawHTMLWith(parseMarkup string, parsePolicy sanitize.Policy) []ui.Node {
	return parseMarkupToNodes(sanitize.Sanitize(parseMarkup, parsePolicy))
}

// RawHTMLUnsafe parses markup WITHOUT sanitization. The content is still built as
// a real node tree (no innerHTML), but nothing is stripped — only pass markup you
// fully control (e.g. a bundled SVG icon). The name is intentionally alarming so
// its use is greppable in review.
func RawHTMLUnsafe(parseMarkup string) []ui.Node {
	return parseMarkupToNodes(parseMarkup)
}

// parseMarkupToNodes parses an HTML fragment and converts it to ui.Nodes built
// through the safe constructors. Malformed/empty input yields no nodes.
func parseMarkupToNodes(parseMarkup string) []ui.Node {
	if strings.TrimSpace(parseMarkup) == "" {
		return nil
	}
	parseContext := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	parseParsed, parseErr := xhtml.ParseFragment(strings.NewReader(parseMarkup), parseContext)
	if parseErr != nil {
		return nil
	}
	var parseOut []ui.Node
	for _, parseNode := range parseParsed {
		if parseConverted := convertMarkupNode(parseNode); parseConverted != nil {
			parseOut = append(parseOut, parseConverted)
		}
	}
	return parseOut
}

// convertMarkupNode maps one parsed HTML node to a ui.Node, or nil for comments
// and doctype nodes.
func convertMarkupNode(parseNode *xhtml.Node) ui.Node {
	switch parseNode.Type {
	case xhtml.TextNode:
		return Text(parseNode.Data)
	case xhtml.ElementNode:
		parseProps := markupAttrsToProps(parseNode.Attr)
		var parseChildren []ui.Node
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			if parseConverted := convertMarkupNode(parseChild); parseConverted != nil {
				parseChildren = append(parseChildren, parseConverted)
			}
		}
		return Tag(parseNode.Data, parseProps, parseChildren...)
	default:
		return nil
	}
}

// markupAttrsToProps maps parsed attributes onto Props. class/id/style get their
// typed homes (style is parsed into the Style map so it never hits the style
// differ as a bare string); everything else goes through Raw, which the prop
// differ applies as plain attributes via SetAttribute.
func markupAttrsToProps(parseAttrs []xhtml.Attribute) Props {
	parseProps := Props{}
	var parseRaw map[string]any
	for _, parseAttr := range parseAttrs {
		parseName := parseAttr.Key
		if parseAttr.Namespace != "" {
			parseName = parseAttr.Namespace + ":" + parseAttr.Key
		}
		switch parseName {
		case "class":
			parseProps.Class = parseAttr.Val
		case "id":
			parseProps.ID = parseAttr.Val
		case "style":
			if parseStyle := parseInlineStyle(parseAttr.Val); len(parseStyle) > 0 {
				parseProps.Style = parseStyle
			}
		default:
			if parseRaw == nil {
				parseRaw = map[string]any{}
			}
			parseRaw[parseName] = parseAttr.Val
		}
	}
	if parseRaw != nil {
		parseProps.Raw = parseRaw
	}
	return parseProps
}

// parseInlineStyle splits a "prop: value; prop: value" string into a map.
func parseInlineStyle(parseValue string) map[string]string {
	var parseStyle map[string]string
	for parseDecl := range strings.SplitSeq(parseValue, ";") {
		parseKey, parseRawVal, parseFound := strings.Cut(parseDecl, ":")
		if !parseFound {
			continue
		}
		parseProp := strings.TrimSpace(parseKey)
		parseVal := strings.TrimSpace(parseRawVal)
		if parseProp == "" || parseVal == "" {
			continue
		}
		if parseStyle == nil {
			parseStyle = map[string]string{}
		}
		parseStyle[parseProp] = parseVal
	}
	return parseStyle
}
