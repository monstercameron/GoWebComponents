package main

import (
	"errors"
	"fmt"
	stdhtml "html"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func normalizeImportedText(parseText string, parseParentTag string) string {
	if preserveImportedWhitespace(parseParentTag) {
		if parseText == "" {
			return ""
		}
		return parseText
	}
	parseTrimmed := strings.TrimSpace(parseText)
	if parseTrimmed == "" {
		return ""
	}
	return strings.Join(strings.Fields(parseTrimmed), " ")
}

func preserveImportedWhitespace(parseTag string) bool {
	switch strings.ToLower(strings.TrimSpace(parseTag)) {
	case "pre", "code", "textarea", "style", "script":
		return true
	default:
		return false
	}
}

func importedNodeTextContent(parseNode importedNode) string {
	if parseNode.Kind == importedNodeText {
		return parseNode.Text
	}
	var parseBuilder strings.Builder
	for _, parseChild := range parseNode.Children {
		parseBuilder.WriteString(importedNodeTextContent(parseChild))
	}
	return strings.TrimSpace(parseBuilder.String())
}

func renderImportedMain(parseDocument importedDocument, parseRepoModulePath string) (string, error) {
	parseRootExpr, parseErr := renderImportedRootExpression(parseDocument.Roots, "\t")
	if parseErr != nil {
		return "", parseErr
	}
	return fmt.Sprintf(`//go:build js && wasm
// +build js,wasm

package main

import (
	%q
	%q
	%q
)

func App() ui.Node {
	return %s
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(App), "#%s")
	utils.WaitForever()
}
`, parseRepoModulePath+"/html", parseRepoModulePath+"/ui", parseRepoModulePath+"/utils", parseRootExpr, importedMountID), nil
}

func renderImportedRootExpression(parseNodes []importedNode, parseIndent string) (string, error) {
	if len(parseNodes) == 0 {
		return "html.Fragment()", nil
	}
	if len(parseNodes) == 1 {
		return renderImportedNodeExpression(parseNodes[0], parseIndent)
	}
	parseFragment := importedNode{Kind: importedNodeFragment, Children: parseNodes}
	return renderImportedNodeExpression(parseFragment, parseIndent)
}

func renderImportedNodeExpression(parseNode importedNode, parseIndent string) (string, error) {
	switch parseNode.Kind {
	case importedNodeText:
		return fmt.Sprintf("html.Text(%s)", strconv.Quote(parseNode.Text)), nil
	case importedNodeFragment:
		if len(parseNode.Children) == 0 {
			return "html.Fragment()", nil
		}
		parseChildLines := []string{"html.Fragment("}
		for _, parseChild := range parseNode.Children {
			parseRendered, parseErr := renderImportedNodeExpression(parseChild, parseIndent+"\t")
			if parseErr != nil {
				return "", parseErr
			}
			parseChildLines = append(parseChildLines, parseIndent+parseRendered+",")
		}
		parseChildLines = append(parseChildLines, strings.TrimRight(parseIndent, "\t")+")")
		return strings.Join(parseChildLines, "\n"), nil
	case importedNodeElement:
		if parseErr2 := validateImportedTagName(parseNode.Tag); parseErr2 != nil {
			return "", parseErr2
		}
		parsePropsLiteral, parseErr3 := renderImportedPropsLiteral(parseNode.Attrs, parseIndent)
		if parseErr3 != nil {
			return "", parseErr3
		}
		parseBuilder, parseTyped := importedBuilderName(parseNode.Tag)
		parseArgs := []string{}
		if parseTyped {
			parseArgs = append(parseArgs, parsePropsLiteral)
		} else {
			parseArgs = append(parseArgs, strconv.Quote(parseNode.Tag), parsePropsLiteral)
		}
		for _, parseChild2 := range parseNode.Children {
			parseRendered2, parseChildErr := renderImportedNodeExpression(parseChild2, parseIndent+"\t")
			if parseChildErr != nil {
				return "", parseChildErr
			}
			parseArgs = append(parseArgs, parseRendered2)
		}
		parsePrefix := "html." + parseBuilder
		if !parseTyped {
			parsePrefix = "html.Tag"
		}
		if len(parseNode.Children) == 0 && !strings.Contains(parsePropsLiteral, "\n") {
			return fmt.Sprintf("%s(%s)", parsePrefix, strings.Join(parseArgs, ", ")), nil
		}
		parseLines := []string{parsePrefix + "("}
		for _, parseArg := range parseArgs {
			parseArgLines := strings.Split(parseArg, "\n")
			if len(parseArgLines) == 1 {
				parseLines = append(parseLines, parseIndent+parseArg+",")
				continue
			}
			for _, parseArgLine := range parseArgLines {
				parseLines = append(parseLines, parseIndent+parseArgLine)
			}
			parseLines[len(parseLines)-1] += ","
		}
		parseLines = append(parseLines, strings.TrimRight(parseIndent, "\t")+")")
		return strings.Join(parseLines, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported imported node kind %q", parseNode.Kind)
	}
}

func validateImportedTagName(parseTag string) error {
	parseTrimmed := strings.TrimSpace(parseTag)
	if parseTrimmed == "" {
		return errors.New("empty tag name is not supported")
	}
	if unicode.IsUpper(rune(parseTrimmed[0])) || strings.Contains(parseTrimmed, ".") {
		return fmt.Errorf("JSX component tags are not supported in gwc import; rewrite %q as static HTML or a custom element", parseTag)
	}
	return nil
}

func importedBuilderName(parseTag string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(parseTag)) {
	case "a":
		return "A", true
	case "article":
		return "Article", true
	case "aside":
		return "Aside", true
	case "blockquote":
		return "Blockquote", true
	case "br":
		return "Br", true
	case "button":
		return "Button", true
	case "code":
		return "Code", true
	case "dialog":
		return "Dialog", true
	case "div":
		return "Div", true
	case "em":
		return "Em", true
	case "fieldset":
		return "Fieldset", true
	case "footer":
		return "Footer", true
	case "form":
		return "Form", true
	case "h1":
		return "H1", true
	case "h2":
		return "H2", true
	case "h3":
		return "H3", true
	case "h4":
		return "H4", true
	case "h5":
		return "H5", true
	case "h6":
		return "H6", true
	case "header":
		return "Header", true
	case "hr":
		return "Hr", true
	case "img":
		return "Img", true
	case "input":
		return "Input", true
	case "label":
		return "Label", true
	case "legend":
		return "Legend", true
	case "li":
		return "Li", true
	case "main":
		return "Main", true
	case "nav":
		return "Nav", true
	case "option":
		return "Option", true
	case "p":
		return "P", true
	case "pre":
		return "Pre", true
	case "section":
		return "Section", true
	case "select":
		return "Select", true
	case "small":
		return "Small", true
	case "span":
		return "Span", true
	case "strong":
		return "Strong", true
	case "textarea":
		return "Textarea", true
	case "time":
		return "Time", true
	case "ul":
		return "Ul", true
	default:
		return parseTag, false
	}
}

func renderImportedPropsLiteral(parseAttrs []importedAttr, parseIndent string) (string, error) {
	if len(parseAttrs) == 0 {
		return "html.Props{}", nil
	}
	type renderedProps struct {
		strings map[string]string
		ints    map[string]string
		bools   map[string]bool
		style   map[string]string
		data    map[string]string
		aria    map[string]string
		raw     map[string]importedValue
	}
	parseProps := renderedProps{
		strings: map[string]string{},
		ints:    map[string]string{},
		bools:   map[string]bool{},
		style:   map[string]string{},
		data:    map[string]string{},
		aria:    map[string]string{},
		raw:     map[string]importedValue{},
	}
	for _, parseAttr := range parseAttrs {
		parseName := strings.TrimSpace(parseAttr.Name)
		if parseName == "" || parseAttr.Value.Kind == importedValueNull {
			continue
		}
		parseLower := strings.ToLower(parseName)
		switch parseLower {
		case "id":
			parseProps.strings["ID"] = importedValueAsString(parseAttr.Value)
		case "class", "classname":
			parseProps.strings["Class"] = importedValueAsString(parseAttr.Value)
		case "key":
			parseProps.strings["Key"] = importedValueAsString(parseAttr.Value)
		case "slot":
			parseProps.strings["Slot"] = importedValueAsString(parseAttr.Value)
		case "title":
			parseProps.strings["Title"] = importedValueAsString(parseAttr.Value)
		case "type":
			parseProps.strings["Type"] = importedValueAsString(parseAttr.Value)
		case "name":
			parseProps.strings["Name"] = importedValueAsString(parseAttr.Value)
		case "value":
			parseProps.strings["Value"] = importedValueAsString(parseAttr.Value)
		case "placeholder":
			parseProps.strings["Placeholder"] = importedValueAsString(parseAttr.Value)
		case "accept":
			parseProps.strings["Accept"] = importedValueAsString(parseAttr.Value)
		case "href":
			parseProps.strings["Href"] = importedValueAsString(parseAttr.Value)
		case "src":
			parseProps.strings["Src"] = importedValueAsString(parseAttr.Value)
		case "alt":
			parseProps.strings["Alt"] = importedValueAsString(parseAttr.Value)
		case "for", "htmlfor":
			parseProps.strings["For"] = importedValueAsString(parseAttr.Value)
		case "role":
			parseProps.strings["Role"] = importedValueAsString(parseAttr.Value)
		case "target":
			parseProps.strings["Target"] = importedValueAsString(parseAttr.Value)
		case "rel":
			parseProps.strings["Rel"] = importedValueAsString(parseAttr.Value)
		case "as":
			parseProps.strings["As"] = importedValueAsString(parseAttr.Value)
		case "action":
			parseProps.strings["Action"] = importedValueAsString(parseAttr.Value)
		case "method":
			parseProps.strings["Method"] = importedValueAsString(parseAttr.Value)
		case "enctype", "encType":
			parseProps.strings["EncType"] = importedValueAsString(parseAttr.Value)
		case "autocomplete", "autoComplete":
			parseProps.strings["AutoComplete"] = importedValueAsString(parseAttr.Value)
		case "min":
			parseProps.strings["Min"] = importedValueAsString(parseAttr.Value)
		case "max":
			parseProps.strings["Max"] = importedValueAsString(parseAttr.Value)
		case "step":
			parseProps.strings["Step"] = importedValueAsString(parseAttr.Value)
		case "rows":
			parseProps.ints["Rows"] = importedValueAsNumber(parseAttr.Value)
		case "cols":
			parseProps.ints["Cols"] = importedValueAsNumber(parseAttr.Value)
		case "checked":
			parseProps.bools["Checked"] = importedValueAsBool(parseAttr.Value)
		case "disabled":
			parseProps.bools["Disabled"] = importedValueAsBool(parseAttr.Value)
		case "selected":
			parseProps.bools["Selected"] = importedValueAsBool(parseAttr.Value)
		case "required":
			parseProps.bools["Required"] = importedValueAsBool(parseAttr.Value)
		case "readonly", "readOnly":
			parseProps.bools["ReadOnly"] = importedValueAsBool(parseAttr.Value)
		case "hidden":
			parseProps.bools["Hidden"] = importedValueAsBool(parseAttr.Value)
		case "multiple":
			parseProps.bools["Multiple"] = importedValueAsBool(parseAttr.Value)
		case "autofocus", "autoFocus":
			parseProps.bools["AutoFocus"] = importedValueAsBool(parseAttr.Value)
		case "style":
			parseStyle := importedValueAsStyleMap(parseAttr.Value)
			for parseKey, parseValue := range parseStyle {
				parseProps.style[parseKey] = parseValue
			}
		default:
			if strings.HasPrefix(parseLower, "data-") {
				parseProps.data[strings.TrimPrefix(parseName, "data-")] = importedValueAsString(parseAttr.Value)
				continue
			}
			if strings.HasPrefix(parseLower, "aria-") {
				parseProps.aria[strings.TrimPrefix(parseName, "aria-")] = importedValueAsString(parseAttr.Value)
				continue
			}
			parseProps.raw[parseName] = parseAttr.Value
		}
	}
	parseLines := []string{"html.Props{"}
	parseAppendStringField := func(parseField4 string) {
		parseValue2, parseOk := parseProps.strings[parseField4]
		if parseOk && parseValue2 != "" {
			parseLines = append(parseLines, parseIndent+parseField4+": "+strconv.Quote(parseValue2)+",")
		}
	}
	parseAppendIntField := func(parseField5 string) {
		parseValue3, parseOk2 := parseProps.ints[parseField5]
		if parseOk2 && parseValue3 != "" {
			parseLines = append(parseLines, parseIndent+parseField5+": "+parseValue3+",")
		}
	}
	parseAppendBoolField := func(parseField6 string) {
		if parseProps.bools[parseField6] {
			parseLines = append(parseLines, parseIndent+parseField6+": true,")
		}
	}
	for _, parseField := range []string{"ID", "Class", "Key", "Slot", "Title", "Type", "Name", "Value", "Placeholder", "Accept", "Href", "Src", "Alt", "For", "Role", "Target", "Rel", "As", "Action", "Method", "EncType", "AutoComplete", "Min", "Max", "Step"} {
		parseAppendStringField(parseField)
	}
	for _, parseField2 := range []string{"Rows", "Cols"} {
		parseAppendIntField(parseField2)
	}
	for _, parseField3 := range []string{"Checked", "Disabled", "Selected", "Required", "ReadOnly", "Hidden", "Multiple", "AutoFocus"} {
		parseAppendBoolField(parseField3)
	}
	parseAppendRenderedStringMap := func(parseField7 string, parseValues map[string]string) {
		if len(parseValues) == 0 {
			return
		}
		parseKeys := make([]string, 0, len(parseValues))
		for parseKey2 := range parseValues {
			parseKeys = append(parseKeys, parseKey2)
		}
		sort.Strings(parseKeys)
		parseLines = append(parseLines, parseIndent+parseField7+": map[string]string{")
		for _, parseKey3 := range parseKeys {
			parseLines = append(parseLines, parseIndent+"\t"+strconv.Quote(parseKey3)+": "+strconv.Quote(parseValues[parseKey3])+",")
		}
		parseLines = append(parseLines, parseIndent+"},")
	}
	parseAppendRenderedStringMap("Style", parseProps.style)
	parseAppendRenderedStringMap("Data", parseProps.data)
	parseAppendRenderedStringMap("Aria", parseProps.aria)
	if len(parseProps.raw) > 0 {
		parseKeys2 := make([]string, 0, len(parseProps.raw))
		for parseKey4 := range parseProps.raw {
			parseKeys2 = append(parseKeys2, parseKey4)
		}
		sort.Strings(parseKeys2)
		parseLines = append(parseLines, parseIndent+"Raw: map[string]interface{}{")
		for _, parseKey5 := range parseKeys2 {
			parseLines = append(parseLines, parseIndent+"\t"+strconv.Quote(parseKey5)+": "+renderImportedInterfaceValue(parseProps.raw[parseKey5])+",")
		}
		parseLines = append(parseLines, parseIndent+"},")
	}
	if len(parseLines) == 1 {
		return "html.Props{}", nil
	}
	parseLines = append(parseLines, strings.TrimRight(parseIndent, "\t")+"}")
	return strings.Join(parseLines, "\n"), nil
}

func importedValueAsString(parseValue importedValue) string {
	switch parseValue.Kind {
	case importedValueString:
		return parseValue.String
	case importedValueBool:
		if parseValue.Bool {
			return "true"
		}
		return "false"
	case importedValueNumber:
		return parseValue.Number
	case importedValueStyle:
		parseKeys := make([]string, 0, len(parseValue.Style))
		for parseKey := range parseValue.Style {
			parseKeys = append(parseKeys, parseKey)
		}
		sort.Strings(parseKeys)
		parseParts := make([]string, 0, len(parseKeys))
		for _, parseKey2 := range parseKeys {
			parseParts = append(parseParts, parseKey2+": "+parseValue.Style[parseKey2])
		}
		return strings.Join(parseParts, "; ")
	default:
		return ""
	}
}

func importedValueAsNumber(parseValue importedValue) string {
	switch parseValue.Kind {
	case importedValueNumber:
		return parseValue.Number
	case importedValueString:
		parseTrimmed := strings.TrimSpace(parseValue.String)
		if parseTrimmed == "" {
			return ""
		}
		if _, parseErr := strconv.Atoi(parseTrimmed); parseErr == nil {
			return parseTrimmed
		}
	}
	return ""
}

func importedValueAsBool(parseValue importedValue) bool {
	switch parseValue.Kind {
	case importedValueBool:
		return parseValue.Bool
	case importedValueString:
		if parseValue.String == "" {
			return true
		}
		return strings.EqualFold(strings.TrimSpace(parseValue.String), "true")
	default:
		return false
	}
}

func importedValueAsStyleMap(parseValue importedValue) map[string]string {
	if parseValue.Kind == importedValueStyle {
		return parseValue.Style
	}
	parseStyle := map[string]string{}
	for parseKey, parseVal := range parseImportedStyleString(importedValueAsString(parseValue)) {
		parseStyle[parseKey] = parseVal
	}
	return parseStyle
}

func parseImportedStyleString(parseRaw string) map[string]string {
	parseStyle := map[string]string{}
	for _, parsePart := range strings.Split(parseRaw, ";") {
		parsePart = strings.TrimSpace(parsePart)
		if parsePart == "" {
			continue
		}
		parseSegments := strings.SplitN(parsePart, ":", 2)
		if len(parseSegments) != 2 {
			continue
		}
		parseKey := strings.TrimSpace(parseSegments[0])
		parseValue := strings.TrimSpace(parseSegments[1])
		if parseKey == "" || parseValue == "" {
			continue
		}
		parseStyle[parseKey] = parseValue
	}
	return parseStyle
}

func renderImportedInterfaceValue(parseValue importedValue) string {
	switch parseValue.Kind {
	case importedValueBool:
		if parseValue.Bool {
			return "true"
		}
		return "false"
	case importedValueNumber:
		return parseValue.Number
	default:
		return strconv.Quote(importedValueAsString(parseValue))
	}
}

func renderImportedIndexHTML(parseSelection startSelection, parseDocument importedDocument) (string, error) {
	parseTitle := strings.TrimSpace(parseDocument.Title)
	if parseTitle == "" {
		parseTitle = parseSelection.ProjectName
	}
	parseLang := strings.TrimSpace(parseDocument.Lang)
	if parseLang == "" {
		parseLang = "en"
	}
	parseHeadLines := []string{
		"\t<meta charset=\"UTF-8\">",
		"\t<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">",
		"\t<title>" + stdhtml.EscapeString(parseTitle) + "</title>",
	}
	for _, parseNode := range parseDocument.HeadNodes {
		parseRendered, parseErr := renderImportedNodeAsHTML(parseNode)
		if parseErr != nil {
			return "", parseErr
		}
		if strings.TrimSpace(parseRendered) == "" {
			continue
		}
		for _, parseLine := range strings.Split(strings.TrimSuffix(parseRendered, "\n"), "\n") {
			parseHeadLines = append(parseHeadLines, "\t"+parseLine)
		}
	}
	parseHeadLines = append(parseHeadLines, "\t<script src=\"./wasm_exec.js\"></script>")
	parseBodyAttrs := renderImportedHTMLAttrs(parseDocument.BodyAttrs)
	if parseBodyAttrs != "" {
		parseBodyAttrs = " " + parseBodyAttrs
	}
	return "<!DOCTYPE html>\n<html lang=\"" + stdhtml.EscapeString(parseLang) + "\">\n<head>\n" + strings.Join(parseHeadLines, "\n") + "\n</head>\n<body" + parseBodyAttrs + ">\n\t<div id=\"" + importedMountID + "\"></div>\n\t<div id=\"boot-error\" hidden></div>\n\t<script>\n\t\tconst go = new Go();\n\t\tconst errorBox = document.getElementById('boot-error');\n\t\tWebAssembly.instantiateStreaming(fetch('./bin/main.wasm'), go.importObject)\n\t\t\t.then(result => go.run(result.instance))\n\t\t\t.catch(error => {\n\t\t\t\terrorBox.hidden = false;\n\t\t\t\terrorBox.textContent = 'Failed to start wasm app: ' + String(error);\n\t\t\t\tconsole.error(error);\n\t\t\t});\n\t</script>\n</body>\n</html>\n", nil
}

func renderImportedNodeAsHTML(parseNode importedNode) (string, error) {
	switch parseNode.Kind {
	case importedNodeText:
		return stdhtml.EscapeString(parseNode.Text), nil
	case importedNodeFragment:
		parseParts := make([]string, 0, len(parseNode.Children))
		for _, parseChild := range parseNode.Children {
			parseRendered, parseErr := renderImportedNodeAsHTML(parseChild)
			if parseErr != nil {
				return "", parseErr
			}
			parseParts = append(parseParts, parseRendered)
		}
		return strings.Join(parseParts, ""), nil
	case importedNodeElement:
		if parseErr2 := validateImportedTagName(parseNode.Tag); parseErr2 != nil {
			return "", parseErr2
		}
		parseAttrs := renderImportedHTMLAttrs(parseNode.Attrs)
		if parseAttrs != "" {
			parseAttrs = " " + parseAttrs
		}
		if len(parseNode.Children) == 0 && isImportedVoidTag(parseNode.Tag) {
			return "<" + parseNode.Tag + parseAttrs + ">", nil
		}
		parseParts2 := make([]string, 0, len(parseNode.Children))
		for _, parseChild2 := range parseNode.Children {
			parseRendered2, parseErr3 := renderImportedNodeAsHTML(parseChild2)
			if parseErr3 != nil {
				return "", parseErr3
			}
			parseParts2 = append(parseParts2, parseRendered2)
		}
		return "<" + parseNode.Tag + parseAttrs + ">" + strings.Join(parseParts2, "") + "</" + parseNode.Tag + ">", nil
	default:
		return "", fmt.Errorf("unsupported imported node kind %q", parseNode.Kind)
	}
}

func renderImportedHTMLAttrs(parseAttrs []importedAttr) string {
	parseParts := make([]string, 0, len(parseAttrs))
	for _, parseAttr := range parseAttrs {
		parseName := strings.TrimSpace(parseAttr.Name)
		if parseName == "" || strings.EqualFold(parseName, "id") && importedValueAsString(parseAttr.Value) == importedMountID {
			continue
		}
		parseValue := parseAttr.Value
		if parseValue.Kind == importedValueNull {
			continue
		}
		if isImportedBooleanAttr(parseName) {
			if importedValueAsBool(parseValue) {
				parseParts = append(parseParts, parseName)
			}
			continue
		}
		if strings.EqualFold(parseName, "style") {
			parseStyle := importedValueAsStyleMap(parseValue)
			if len(parseStyle) == 0 {
				continue
			}
			parseKeys := make([]string, 0, len(parseStyle))
			for parseKey := range parseStyle {
				parseKeys = append(parseKeys, parseKey)
			}
			sort.Strings(parseKeys)
			parseEntries := make([]string, 0, len(parseKeys))
			for _, parseKey2 := range parseKeys {
				parseEntries = append(parseEntries, parseKey2+": "+parseStyle[parseKey2])
			}
			parseParts = append(parseParts, parseName+"=\""+stdhtml.EscapeString(strings.Join(parseEntries, "; "))+"\"")
			continue
		}
		parseParts = append(parseParts, parseName+"=\""+stdhtml.EscapeString(importedValueAsString(parseValue))+"\"")
	}
	return strings.Join(parseParts, " ")
}

func isImportedVoidTag(parseTag string) bool {
	switch strings.ToLower(strings.TrimSpace(parseTag)) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

func isImportedBooleanAttr(parseName string) bool {
	switch strings.ToLower(strings.TrimSpace(parseName)) {
	case "checked", "disabled", "selected", "required", "readonly", "hidden", "multiple", "autofocus", "controls", "muted", "playsinline", "loop":
		return true
	default:
		return false
	}
}

func (parseL launcher) generateScaffoldProject(parsePlan scaffoldPlan) (scaffoldResult, error) {
	parseTargetDir := filepath.Clean(parsePlan.Selection.TargetDir)
	if parseErr := validateGeneratedTargetDir(parseTargetDir); parseErr != nil {
		return scaffoldResult{}, parseErr
	}
	if parseErr2 := ensureEmptyDir(parseTargetDir); parseErr2 != nil {
		return scaffoldResult{}, parseErr2
	}

	parseWasmExecSource := ""
	if !parsePlan.SkipRuntimeAssets {
		parseResolvedWasmExecSource, parseErr3 := scaffoldResolveWasmExecPath()
		if parseErr3 != nil {
			return scaffoldResult{}, parseErr3
		}
		parseWasmExecSource = parseResolvedWasmExecSource
	}

	parseMainPath := filepath.Join(parseTargetDir, "main.go")
	parseHtmlPath := filepath.Join(parseTargetDir, "index.html")
	parseMetadataPath := filepath.Join(parseTargetDir, "gwc-start.json")
	parseReadmePath := filepath.Join(parseTargetDir, "README.md")
	parseWasmExecPath := filepath.Join(parseTargetDir, "wasm_exec.js")
	parseGoModPath := filepath.Join(parseTargetDir, "go.mod")

	if parseErr4 := scaffoldWriteFile(parseGoModPath, []byte(parsePlan.GoMod), 0644); parseErr4 != nil {
		return scaffoldResult{}, fmt.Errorf("write go.mod: %w", parseErr4)
	}
	if parseErr5 := scaffoldWriteFile(parseMainPath, []byte(parsePlan.MainGo), 0644); parseErr5 != nil {
		return scaffoldResult{}, fmt.Errorf("write main.go: %w", parseErr5)
	}
	if parseErr6 := scaffoldWriteFile(parseHtmlPath, []byte(parsePlan.HTML), 0644); parseErr6 != nil {
		return scaffoldResult{}, fmt.Errorf("write index.html: %w", parseErr6)
	}
	parseMetadataBytes, parseErr7 := scaffoldMarshalIndent(parsePlan.Metadata, "", "  ")
	if parseErr7 != nil {
		return scaffoldResult{}, fmt.Errorf("encode gwc-start.json: %w", parseErr7)
	}
	parseMetadataBytes = append(parseMetadataBytes, '\n')
	if parseErr8 := scaffoldWriteFile(parseMetadataPath, parseMetadataBytes, 0644); parseErr8 != nil {
		return scaffoldResult{}, fmt.Errorf("write gwc-start.json: %w", parseErr8)
	}
	if parseErr9 := scaffoldWriteFile(parseReadmePath, []byte(parsePlan.README), 0644); parseErr9 != nil {
		return scaffoldResult{}, fmt.Errorf("write README.md: %w", parseErr9)
	}
	for parseRelativePath, parseContents := range parsePlan.ExtraFiles {
		parseTargetPath := filepath.Join(parseTargetDir, filepath.FromSlash(parseRelativePath))
		if parseErr10 := os.MkdirAll(filepath.Dir(parseTargetPath), 0755); parseErr10 != nil {
			return scaffoldResult{}, fmt.Errorf("create scaffold extra file directory: %w", parseErr10)
		}
		if parseErr11 := scaffoldWriteFile(parseTargetPath, parseContents, 0644); parseErr11 != nil {
			return scaffoldResult{}, fmt.Errorf("write scaffold extra file %s: %w", parseRelativePath, parseErr11)
		}
	}
	if !parsePlan.SkipRuntimeAssets {
		parseWasmExecBytes, parseErr12 := scaffoldReadFile(parseWasmExecSource)
		if parseErr12 != nil {
			return scaffoldResult{}, fmt.Errorf("read wasm_exec.js: %w", parseErr12)
		}
		if parseErr13 := scaffoldWriteFile(parseWasmExecPath, parseWasmExecBytes, 0644); parseErr13 != nil {
			return scaffoldResult{}, fmt.Errorf("write wasm_exec.js: %w", parseErr13)
		}
	}
	if parseErr14 := parseL.seedScaffoldGoSum(parseTargetDir); parseErr14 != nil {
		return scaffoldResult{}, parseErr14
	}
	if !parsePlan.SkipGoModTidy {
		if parseErr15 := scaffoldTidyModule(parseL, parseTargetDir); parseErr15 != nil {
			return scaffoldResult{}, parseErr15
		}
	}
	if parseErr16 := scaffoldFormatMain(parseMainPath); parseErr16 != nil {
		return scaffoldResult{}, fmt.Errorf("format generated main.go: %w", parseErr16)
	}
	return scaffoldResult{TargetDir: parseTargetDir, AppPath: parseMainPath, HTMLPath: parseHtmlPath}, nil
}

func defaultScaffoldMetadata(parseSelection startSelection) scaffoldMetadata {
	parseEnterpriseSections := make([]scaffoldEnterpriseSectionMetadata, 0, len(parseSelection.EnterpriseSections))
	for _, parseSection := range parseSelection.EnterpriseSections {
		parseEnterpriseSections = append(parseEnterpriseSections, scaffoldEnterpriseSectionMetadata{
			Title:    parseSection.Title,
			Summary:  parseSection.Summary,
			Features: append([]string(nil), parseSection.Features...),
		})
	}
	return scaffoldMetadata{
		SchemaVersion: currentScaffoldMetadataSchemaVersion,
		ProjectName:   parseSelection.ProjectName,
		ModulePath:    parseSelection.ModulePath,
		Author:        parseSelection.Author,
		Version:       parseSelection.Version,
		Description:   parseSelection.Description,
		TargetDir:     parseSelection.TargetDir,
		Preset: scaffoldPresetMetadata{
			Key:         parseSelection.Preset.Key,
			Name:        parseSelection.Preset.Name,
			Summary:     parseSelection.Preset.Summary,
			Description: parseSelection.Preset.Description,
			Features:    parseSelection.Preset.Features,
		},
		Enterprise: scaffoldEnterpriseMetadata{
			EnabledSections: append([]string(nil), parseSelection.EnabledEnterpriseSections...),
			Features:        append([]string(nil), parseSelection.EnterpriseFeatures...),
			Sections:        parseEnterpriseSections,
		},
		Ownership: scaffoldOwnershipMetadata{
			ProjectOwnership:    selectionProjectOwnership(parseSelection.ProjectMode),
			FrameworkSourceMode: selectionFrameworkSourceMode(parseSelection.ProjectMode),
		},
		Tooling: scaffoldToolingMetadata{
			AppPath:             filepath.ToSlash("main.go"),
			HTMLPath:            filepath.ToSlash("index.html"),
			WASMPath:            filepath.ToSlash(scaffoldWASMOutputPath()),
			DevHost:             defaultHost,
			DevPort:             "8080",
			DefaultBuildProfile: defaultScaffoldBuildProfile(),
			ReleaseOutDir:       defaultScaffoldReleaseOutDir(),
			ReleaseBinaryName:   defaultScaffoldReleaseBinaryName(),
			ReleaseCompression:  defaultScaffoldReleaseCompression(),
		},
	}
}

func printImportSummary(parseSummary importSummary) {
	fmt.Println("GWC import")
	fmt.Printf("  source kind:    %s\n", parseSummary.SourceKind)
	fmt.Printf("  source:         %s\n", parseSummary.SourcePath)
	if strings.TrimSpace(parseSummary.OutputPath) != "" {
		fmt.Printf("  output:         %s\n", parseSummary.OutputPath)
	}
}

func createLauncherTempDir(parseRootPath string, parsePrefix string) (string, error) {
	parseRootPath = strings.TrimSpace(parseRootPath)
	if parseRootPath == "" {
		parseCwd, parseErr := buildGetwd()
		if parseErr != nil {
			return "", parseErr
		}
		parseRootPath = parseCwd
	}
	parseTempRoot, _, parseErr2 := resolveLauncherTempRoot(parseRootPath)
	if parseErr2 != nil {
		return "", parseErr2
	}
	if parseErr3 := os.MkdirAll(parseTempRoot, 0755); parseErr3 != nil {
		return "", fmt.Errorf("create launcher temp root: %w", parseErr3)
	}
	parsePath, parseErr2 := launcherMkdirTemp(parseTempRoot, parsePrefix)
	if parseErr2 != nil {
		return "", fmt.Errorf("create launcher temp directory: %w", parseErr2)
	}
	return parsePath, nil
}
