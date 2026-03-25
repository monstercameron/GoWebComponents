package main

import (
	"strings"
	"testing"
)

func TestImportHelpersParseAndNormalizeLiterals(t *testing.T) {
	if kind, err := detectImportSourceKind("catalog.HTML"); err != nil || kind != "html" {
		t.Fatalf("detectImportSourceKind(html) = %q, %v; want html,nil", kind, err)
	}
	if kind, err := detectImportSourceKind("catalog.tsx"); err != nil || kind != "jsx" {
		t.Fatalf("detectImportSourceKind(tsx) = %q, %v; want jsx,nil", kind, err)
	}
	if _, err := detectImportSourceKind("catalog.md"); err == nil {
		t.Fatal("detectImportSourceKind() should reject unsupported extensions")
	}

	if got := defaultImportedProjectName(" My App!.tsx "); got != "my-app" {
		t.Fatalf("defaultImportedProjectName() = %q, want my-app", got)
	}
	if got := camelToKebab("backgroundColor"); got != "background-color" {
		t.Fatalf("camelToKebab() = %q, want background-color", got)
	}
	if !preserveImportedWhitespace(" pre ") || preserveImportedWhitespace("div") {
		t.Fatal("preserveImportedWhitespace() returned unexpected values")
	}
	if got := normalizeImportedText("  hello   world  ", "div"); got != "hello world" {
		t.Fatalf("normalizeImportedText(div) = %q, want hello world", got)
	}
	if got := normalizeImportedText("  keep\n spacing  ", "pre"); got != "  keep\n spacing  " {
		t.Fatalf("normalizeImportedText(pre) = %q, want preserved text", got)
	}

	if value, err := parseImportedExpressionLiteral(`"atlas"`); err != nil || value.Kind != importedValueString || value.String != "atlas" {
		t.Fatalf("parseImportedExpressionLiteral(string) = %+v, %v", value, err)
	}
	if value, err := parseImportedExpressionLiteral("true"); err != nil || value.Kind != importedValueBool || !value.Bool {
		t.Fatalf("parseImportedExpressionLiteral(bool) = %+v, %v", value, err)
	}
	if value, err := parseImportedExpressionLiteral("42"); err != nil || value.Kind != importedValueNumber || value.Number != "42" {
		t.Fatalf("parseImportedExpressionLiteral(number) = %+v, %v", value, err)
	}
	if value, err := parseImportedExpressionLiteral("undefined"); err != nil || value.Kind != importedValueNull {
		t.Fatalf("parseImportedExpressionLiteral(undefined) = %+v, %v", value, err)
	}
	if _, err := parseImportedExpressionLiteral("`hi ${name}`"); err == nil {
		t.Fatal("parseImportedExpressionLiteral() should reject template interpolation")
	}
	if _, err := parseImportedExpressionLiteral("user.name"); err == nil {
		t.Fatal("parseImportedExpressionLiteral() should reject dynamic expressions")
	}

	style, err := parseImportedJSXStyleObject(`backgroundColor: "red", "--brand": "blue", zIndex: 5`)
	if err != nil {
		t.Fatalf("parseImportedJSXStyleObject() error = %v", err)
	}
	if style["background-color"] != "red" || style["--brand"] != "blue" || style["z-index"] != "5" {
		t.Fatalf("parseImportedJSXStyleObject() = %#v, want normalized style keys", style)
	}
	if _, err := parseImportedJSXStyleObject(`backgroundColor`); err == nil {
		t.Fatal("parseImportedJSXStyleObject() should reject invalid entries")
	}

	if parts, err := splitTopLevel(`a:{b:1}, c:2`, ','); err != nil || len(parts) != 2 {
		t.Fatalf("splitTopLevel() = %#v, %v; want 2 parts", parts, err)
	}
	if _, err := splitTopLevel(`a:{b:1`, ','); err == nil {
		t.Fatal("splitTopLevel() should reject unterminated input")
	}

	if findKeywordOutsideJSX(`returning = 1; return <div />`, "return") < 0 {
		t.Fatal("findKeywordOutsideJSX() should locate standalone keyword")
	}
	if findCharOutsideJSX(`const s = "<div>"; /* <ignore> */ return <main />`, '<') < 0 {
		t.Fatal("findCharOutsideJSX() should find JSX root outside quoted strings and comments")
	}
	if _, err := findJSXStart("   "); err == nil {
		t.Fatal("findJSXStart() should reject empty sources")
	}
}

func TestImportHelpersRenderAttrsAndValues(t *testing.T) {
	if got := importedValueAsString(importedValue{Kind: importedValueBool, Bool: true}); got != "true" {
		t.Fatalf("importedValueAsString(bool) = %q, want true", got)
	}
	if got := importedValueAsNumber(importedValue{Kind: importedValueString, String: "12"}); got != "12" {
		t.Fatalf("importedValueAsNumber() = %q, want 12", got)
	}
	if got := importedValueAsBool(importedValue{Kind: importedValueString, String: ""}); !got {
		t.Fatalf("importedValueAsBool(empty string) = %t, want true", got)
	}
	styleMap := importedValueAsStyleMap(importedValue{Kind: importedValueString, String: "color: red; padding: 8px"})
	if styleMap["color"] != "red" || styleMap["padding"] != "8px" {
		t.Fatalf("importedValueAsStyleMap() = %#v, want parsed style map", styleMap)
	}
	if got := parseImportedStyleString("color: red; broken; padding: 8px"); got["padding"] != "8px" || got["color"] != "red" {
		t.Fatalf("parseImportedStyleString() = %#v, want valid declarations only", got)
	}
	if got := renderImportedInterfaceValue(importedValue{Kind: importedValueNumber, Number: "7"}); got != "7" {
		t.Fatalf("renderImportedInterfaceValue(number) = %q, want 7", got)
	}

	attrs := []importedAttr{
		{Name: "id", Value: importedValue{Kind: importedValueString, String: importedMountID}},
		{Name: "checked", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "style", Value: importedValue{Kind: importedValueString, String: "color: red; padding: 8px"}},
		{Name: "data-mode", Value: importedValue{Kind: importedValueString, String: "demo"}},
	}
	rendered := renderImportedHTMLAttrs(attrs)
	if strings.Contains(rendered, importedMountID) || !strings.Contains(rendered, "checked") || !strings.Contains(rendered, `style="color: red; padding: 8px"`) || !strings.Contains(rendered, `data-mode="demo"`) {
		t.Fatalf("renderImportedHTMLAttrs() = %q, want filtered id plus boolean/style/data attrs", rendered)
	}

	node := importedNode{
		Kind: importedNodeElement,
		Tag:  "section",
		Children: []importedNode{
			{Kind: importedNodeText, Text: "Hello"},
			{Kind: importedNodeElement, Tag: "span", Children: []importedNode{{Kind: importedNodeText, Text: " World "}}},
		},
	}
	if got := importedNodeTextContent(node); got != "HelloWorld" {
		t.Fatalf("importedNodeTextContent() = %q, want HelloWorld", got)
	}
	if !isImportedVoidTag("img") || isImportedVoidTag("div") {
		t.Fatal("isImportedVoidTag() returned unexpected values")
	}
	if !isImportedBooleanAttr("disabled") || isImportedBooleanAttr("href") {
		t.Fatal("isImportedBooleanAttr() returned unexpected values")
	}
}
