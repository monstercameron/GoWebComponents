package main

import (
	"strings"
	"testing"
)

// TestImportParserBranchesCoverFallbacks verifies parser helper fallback and error branches.
func TestImportParserBranchesCoverFallbacks(parseT *testing.T) {
	if parseGot := findCharOutsideJSX("'<' `>` // <\n/* < */", '<'); parseGot != -1 {
		parseT.Fatalf("findCharOutsideJSX() = %d, want -1 when target stays inside quoted/commented content", parseGot)
	}

	parseParser := &jsxParser{source: "{true}"}
	parseNode, parseErr := parseParser.parseNode()
	if parseErr != nil {
		parseT.Fatalf("parseNode(bool expression): %v", parseErr)
	}
	if parseNode.Kind != "" {
		parseT.Fatalf("parseNode(bool expression) = %#v, want empty node", parseNode)
	}

	parseParser = &jsxParser{source: "<div></span>"}
	if _, parseErr2 := parseParser.parseNodesUntil(""); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "expected closing tag </div> but found </span>") {
		parseT.Fatalf("parseNodesUntil(mismatched close) error = %v", parseErr2)
	}

	parseParser = &jsxParser{source: "<div title={user.name}></div>"}
	if _, parseErr3 := parseParser.parseElement(); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "unsupported JSX attribute expression for title") {
		parseT.Fatalf("parseElement(dynamic attr) error = %v", parseErr3)
	}

	parseParser = &jsxParser{source: "<div style={{broken}}></div>"}
	if _, parseErr4 := parseParser.parseElement(); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "unsupported JSX style object for style") {
		parseT.Fatalf("parseElement(invalid style object) error = %v", parseErr4)
	}

	parseParser = &jsxParser{source: "</div"}
	if _, parseErr5 := parseParser.parseClosingTag(); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "expected > to close </div>") {
		parseT.Fatalf("parseClosingTag() error = %v", parseErr5)
	}

	parseParser = &jsxParser{source: "div"}
	if _, parseErr6 := parseParser.readBalanced('{', '}'); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "expected {") {
		parseT.Fatalf("readBalanced() error = %v", parseErr6)
	}
}

// TestImportRenderBranchesCoverCustomTags verifies renderer branches for custom tags and invalid nodes.
func TestImportRenderBranchesCoverCustomTags(parseT *testing.T) {
	if parseErr := validateImportedTagName(""); parseErr == nil || !strings.Contains(parseErr.Error(), "empty tag name") {
		parseT.Fatalf("validateImportedTagName(empty) error = %v", parseErr)
	}
	if parseErr := validateImportedTagName("Widget"); parseErr == nil || !strings.Contains(parseErr.Error(), "component tags are not supported") {
		parseT.Fatalf("validateImportedTagName(component) error = %v", parseErr)
	}

	if parseBuilder, isParseTyped := importedBuilderName("price-badge"); parseBuilder != "price-badge" || isParseTyped {
		parseT.Fatalf("importedBuilderName(custom) = (%q, %t), want custom,false", parseBuilder, isParseTyped)
	}

	parseFragmentExpr, parseErr := renderImportedNodeExpression(importedNode{Kind: importedNodeFragment}, "\t")
	if parseErr != nil {
		parseT.Fatalf("renderImportedNodeExpression(empty fragment): %v", parseErr)
	}
	if parseFragmentExpr != "html.Fragment()" {
		parseT.Fatalf("renderImportedNodeExpression(empty fragment) = %q, want html.Fragment()", parseFragmentExpr)
	}

	parseCustomExpr, parseErr := renderImportedNodeExpression(importedNode{
		Kind: importedNodeElement,
		Tag:  "price-badge",
		Attrs: []importedAttr{
			{Name: "data-tone", Value: importedValue{Kind: importedValueString, String: "sale"}},
		},
	}, "\t")
	if parseErr != nil {
		parseT.Fatalf("renderImportedNodeExpression(custom): %v", parseErr)
	}
	if !strings.Contains(parseCustomExpr, "html.Tag(") || !strings.Contains(parseCustomExpr, `"price-badge"`) || !strings.Contains(parseCustomExpr, `Data: map[string]string{`) {
		parseT.Fatalf("renderImportedNodeExpression(custom) = %q, want html.Tag custom element output", parseCustomExpr)
	}

	if _, parseErr2 := renderImportedNodeExpression(importedNode{Kind: importedNodeKind("mystery")}, "\t"); parseErr2 == nil || !strings.Contains(parseErr2.Error(), `unsupported imported node kind "mystery"`) {
		parseT.Fatalf("renderImportedNodeExpression(unsupported) error = %v", parseErr2)
	}

	parseHTML, parseErr := renderImportedNodeAsHTML(importedNode{
		Kind: importedNodeFragment,
		Children: []importedNode{
			{Kind: importedNodeText, Text: "<safe>"},
			{Kind: importedNodeElement, Tag: "img", Attrs: []importedAttr{{Name: "alt", Value: importedValue{Kind: importedValueString, String: `sale "hero"`}}}},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("renderImportedNodeAsHTML(fragment): %v", parseErr)
	}
	if parseHTML != `&lt;safe&gt;<img alt="sale &#34;hero&#34;">` {
		parseT.Fatalf("renderImportedNodeAsHTML(fragment) = %q, want escaped text plus void tag", parseHTML)
	}

	if _, parseErr3 := renderImportedNodeAsHTML(importedNode{Kind: importedNodeElement, Tag: "Widget"}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "component tags are not supported") {
		parseT.Fatalf("renderImportedNodeAsHTML(component) error = %v", parseErr3)
	}
	if _, parseErr4 := renderImportedNodeAsHTML(importedNode{Kind: importedNodeKind("mystery")}); parseErr4 == nil || !strings.Contains(parseErr4.Error(), `unsupported imported node kind "mystery"`) {
		parseT.Fatalf("renderImportedNodeAsHTML(unsupported) error = %v", parseErr4)
	}
}

// TestImportIndexHTMLBranchesCoverDefaults verifies HTML shell rendering defaults and error propagation.
func TestImportIndexHTMLBranchesCoverDefaults(parseT *testing.T) {
	parseHTML, parseErr := renderImportedIndexHTML(startSelection{ProjectName: "Atlas Demo"}, importedDocument{
		HeadNodes: []importedNode{
			{Kind: importedNodeText, Text: ""},
			{Kind: importedNodeElement, Tag: "meta", Attrs: []importedAttr{{Name: "name", Value: importedValue{Kind: importedValueString, String: "theme-color"}}, {Name: "content", Value: importedValue{Kind: importedValueString, String: "#111"}}}},
		},
		BodyAttrs: []importedAttr{
			{Name: "hidden", Value: importedValue{Kind: importedValueBool, Bool: true}},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("renderImportedIndexHTML(defaults): %v", parseErr)
	}
	if !strings.Contains(parseHTML, "<html lang=\"en\">") || !strings.Contains(parseHTML, "<title>Atlas Demo</title>") || !strings.Contains(parseHTML, "<meta name=\"theme-color\" content=\"#111\">") || !strings.Contains(parseHTML, "<body hidden>") {
		parseT.Fatalf("renderImportedIndexHTML(defaults) = %q, want default title/lang and rendered head/body attrs", parseHTML)
	}

	_, parseErr = renderImportedIndexHTML(startSelection{ProjectName: "Atlas"}, importedDocument{
		HeadNodes: []importedNode{{Kind: importedNodeElement, Tag: "Widget"}},
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "component tags are not supported") {
		parseT.Fatalf("renderImportedIndexHTML(invalid head) error = %v", parseErr)
	}
}
