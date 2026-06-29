package main

import (
	"strings"
	"testing"
)

func TestImportHelpersParseAndNormalizeLiterals(parseT *testing.T) {
	if parseKind, parseErr := detectImportSourceKind("catalog.HTML"); parseErr != nil || parseKind != "html" {
		parseT.Fatalf("detectImportSourceKind(html) = %q, %v; want html,nil", parseKind, parseErr)
	}
	if parseKind2, parseErr2 := detectImportSourceKind("catalog.tsx"); parseErr2 != nil || parseKind2 != "jsx" {
		parseT.Fatalf("detectImportSourceKind(tsx) = %q, %v; want jsx,nil", parseKind2, parseErr2)
	}
	if _, parseErr3 := detectImportSourceKind("catalog.md"); parseErr3 == nil {
		parseT.Fatal("detectImportSourceKind() should reject unsupported extensions")
	}

	if parseGot := defaultImportedProjectName(" My App!.tsx "); parseGot != "my-app" {
		parseT.Fatalf("defaultImportedProjectName() = %q, want my-app", parseGot)
	}
	if parseGot2 := camelToKebab("backgroundColor"); parseGot2 != "background-color" {
		parseT.Fatalf("camelToKebab() = %q, want background-color", parseGot2)
	}
	if !preserveImportedWhitespace(" pre ") || preserveImportedWhitespace("div") {
		parseT.Fatal("preserveImportedWhitespace() returned unexpected values")
	}
	if parseGot3 := normalizeImportedText("  hello   world  ", "div"); parseGot3 != "hello world" {
		parseT.Fatalf("normalizeImportedText(div) = %q, want hello world", parseGot3)
	}
	if parseGot4 := normalizeImportedText("  keep\n spacing  ", "pre"); parseGot4 != "  keep\n spacing  " {
		parseT.Fatalf("normalizeImportedText(pre) = %q, want preserved text", parseGot4)
	}

	if parseValue, parseErr4 := parseImportedExpressionLiteral(`"atlas"`); parseErr4 != nil || parseValue.Kind != importedValueString || parseValue.String != "atlas" {
		parseT.Fatalf("parseImportedExpressionLiteral(string) = %+v, %v", parseValue, parseErr4)
	}
	if parseValue2, parseErr5 := parseImportedExpressionLiteral("true"); parseErr5 != nil || parseValue2.Kind != importedValueBool || !parseValue2.Bool {
		parseT.Fatalf("parseImportedExpressionLiteral(bool) = %+v, %v", parseValue2, parseErr5)
	}
	if parseValue3, parseErr6 := parseImportedExpressionLiteral("42"); parseErr6 != nil || parseValue3.Kind != importedValueNumber || parseValue3.Number != "42" {
		parseT.Fatalf("parseImportedExpressionLiteral(number) = %+v, %v", parseValue3, parseErr6)
	}
	if parseValue4, parseErr7 := parseImportedExpressionLiteral("undefined"); parseErr7 != nil || parseValue4.Kind != importedValueNull {
		parseT.Fatalf("parseImportedExpressionLiteral(undefined) = %+v, %v", parseValue4, parseErr7)
	}
	if _, parseErr8 := parseImportedExpressionLiteral("`hi ${name}`"); parseErr8 == nil {
		parseT.Fatal("parseImportedExpressionLiteral() should reject template interpolation")
	}
	if _, parseErr9 := parseImportedExpressionLiteral("user.name"); parseErr9 == nil {
		parseT.Fatal("parseImportedExpressionLiteral() should reject dynamic expressions")
	}

	parseStyle, parseErr10 := parseImportedJSXStyleObject(`backgroundColor: "red", "--brand": "blue", zIndex: 5`)
	if parseErr10 != nil {
		parseT.Fatalf("parseImportedJSXStyleObject() error = %v", parseErr10)
	}
	if parseStyle["background-color"] != "red" || parseStyle["--brand"] != "blue" || parseStyle["z-index"] != "5" {
		parseT.Fatalf("parseImportedJSXStyleObject() = %#v, want normalized style keys", parseStyle)
	}
	if _, parseErr11 := parseImportedJSXStyleObject(`backgroundColor`); parseErr11 == nil {
		parseT.Fatal("parseImportedJSXStyleObject() should reject invalid entries")
	}

	if parseParts, parseErr12 := splitTopLevel(`a:{b:1}, c:2`, ','); parseErr12 != nil || len(parseParts) != 2 {
		parseT.Fatalf("splitTopLevel() = %#v, %v; want 2 parts", parseParts, parseErr12)
	}
	if _, parseErr13 := splitTopLevel(`a:{b:1`, ','); parseErr13 == nil {
		parseT.Fatal("splitTopLevel() should reject unterminated input")
	}

	if findKeywordOutsideJSX(`returning = 1; return <div />`, "return") < 0 {
		parseT.Fatal("findKeywordOutsideJSX() should locate standalone keyword")
	}
	if findCharOutsideJSX(`const s = "<div>"; /* <ignore> */ return <main />`, '<') < 0 {
		parseT.Fatal("findCharOutsideJSX() should find JSX root outside quoted strings and comments")
	}
	if _, parseErr14 := findJSXStart("   "); parseErr14 == nil {
		parseT.Fatal("findJSXStart() should reject empty sources")
	}
}

func TestImportHelpersRenderAttrsAndValues(parseT *testing.T) {
	if parseGot := importedValueAsString(importedValue{Kind: importedValueBool, Bool: true}); parseGot != "true" {
		parseT.Fatalf("importedValueAsString(bool) = %q, want true", parseGot)
	}
	if parseGot2 := importedValueAsNumber(importedValue{Kind: importedValueString, String: "12"}); parseGot2 != "12" {
		parseT.Fatalf("importedValueAsNumber() = %q, want 12", parseGot2)
	}
	if parseGot3 := importedValueAsBool(importedValue{Kind: importedValueString, String: ""}); !parseGot3 {
		parseT.Fatalf("importedValueAsBool(empty string) = %t, want true", parseGot3)
	}
	parseStyleMap := importedValueAsStyleMap(importedValue{Kind: importedValueString, String: "color: red; padding: 8px"})
	if parseStyleMap["color"] != "red" || parseStyleMap["padding"] != "8px" {
		parseT.Fatalf("importedValueAsStyleMap() = %#v, want parsed style map", parseStyleMap)
	}
	if parseGot4 := parseImportedStyleString("color: red; broken; padding: 8px"); parseGot4["padding"] != "8px" || parseGot4["color"] != "red" {
		parseT.Fatalf("parseImportedStyleString() = %#v, want valid declarations only", parseGot4)
	}
	if parseGot5 := renderImportedInterfaceValue(importedValue{Kind: importedValueNumber, Number: "7"}); parseGot5 != "7" {
		parseT.Fatalf("renderImportedInterfaceValue(number) = %q, want 7", parseGot5)
	}

	parseAttrs := []importedAttr{
		{Name: "id", Value: importedValue{Kind: importedValueString, String: importedMountID}},
		{Name: "checked", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "style", Value: importedValue{Kind: importedValueString, String: "color: red; padding: 8px"}},
		{Name: "data-mode", Value: importedValue{Kind: importedValueString, String: "demo"}},
	}
	parseRendered := renderImportedHTMLAttrs(parseAttrs)
	if strings.Contains(parseRendered, importedMountID) || !strings.Contains(parseRendered, "checked") || !strings.Contains(parseRendered, `style="color: red; padding: 8px"`) || !strings.Contains(parseRendered, `data-mode="demo"`) {
		parseT.Fatalf("renderImportedHTMLAttrs() = %q, want filtered id plus boolean/style/data attrs", parseRendered)
	}

	parseNode := importedNode{
		Kind: importedNodeElement,
		Tag:  "section",
		Children: []importedNode{
			{Kind: importedNodeText, Text: "Hello"},
			{Kind: importedNodeElement, Tag: "span", Children: []importedNode{{Kind: importedNodeText, Text: " World "}}},
		},
	}
	if parseGot6 := importedNodeTextContent(parseNode); parseGot6 != "HelloWorld" {
		parseT.Fatalf("importedNodeTextContent() = %q, want HelloWorld", parseGot6)
	}
	if !isImportedVoidTag("img") || isImportedVoidTag("div") {
		parseT.Fatal("isImportedVoidTag() returned unexpected values")
	}
	if !isImportedBooleanAttr("disabled") || isImportedBooleanAttr("href") {
		parseT.Fatal("isImportedBooleanAttr() returned unexpected values")
	}
}
