package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"
)

const importedMountID = "gwc-import-root"

type importConfig struct {
	sourcePath string
	outputPath string
	json       bool
	sourceKind string
	sourceBase string

	document importedDocument
}

type importSummary struct {
	OK         bool   `json:"ok"`
	SourceKind string `json:"sourceKind"`
	SourcePath string `json:"sourcePath"`
	OutputPath string `json:"outputPath,omitempty"`
}

type importedDocument struct {
	SourceKind string
	Title      string
	Lang       string
	HeadNodes  []importedNode
	BodyAttrs  []importedAttr
	Roots      []importedNode
}

type importedNode struct {
	Kind     importedNodeKind
	Tag      string
	Attrs    []importedAttr
	Text     string
	Children []importedNode
}

type importedNodeKind string

const (
	importedNodeElement  importedNodeKind = "element"
	importedNodeText     importedNodeKind = "text"
	importedNodeFragment importedNodeKind = "fragment"
)

type importedAttr struct {
	Name  string
	Value importedValue
}

type importedValueKind string

const (
	importedValueString importedValueKind = "string"
	importedValueBool   importedValueKind = "bool"
	importedValueNumber importedValueKind = "number"
	importedValueNull   importedValueKind = "null"
	importedValueStyle  importedValueKind = "style"
)

type importedValue struct {
	Kind      importedValueKind
	String    string
	Bool      bool
	Number    string
	Style     map[string]string
	RawSource string
}

type scaffoldPlan struct {
	Selection         startSelection
	GoMod             string
	MainGo            string
	HTML              string
	README            string
	Metadata          scaffoldMetadata
	ExtraFiles        map[string][]byte
	SkipGoModTidy     bool
	SkipRuntimeAssets bool
}

type jsxParser struct {
	source string
	index  int
}

var launcherMkdirTemp = os.MkdirTemp

func (parseL launcher) runImport(parseArgs []string) error {
	parseFs := flag.NewFlagSet("import", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseSource := parseFs.String("src", "", "Path to a valid .html, .htm, .jsx, or .tsx file")
	parseOut := parseFs.String("out", "", "Path to write the generated main.go file; prints to stdout when omitted")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := parseL.resolveImportConfig(importConfig{
		sourcePath: *parseSource,
		outputPath: *parseOut,
		json:       *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseMainContents, parseSummary, parseErr2 := parseL.executeImport(parseConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	if strings.TrimSpace(parseConfig.outputPath) == "" {
		if parseConfig.json {
			return errors.New("import requires -out when -json is used")
		}
		fmt.Fprint(os.Stdout, parseMainContents)
		return nil
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printImportSummary(parseSummary)
	return nil
}

func (parseL launcher) resolveImportConfig(parseConfig importConfig) (importConfig, error) {
	parseResolved := parseConfig
	parseCwd, parseErr := buildGetwd()
	if parseErr != nil {
		return importConfig{}, parseErr
	}
	if strings.TrimSpace(parseResolved.sourcePath) == "" {
		return importConfig{}, errors.New("import requires -src")
	}
	parseResolved.sourcePath, parseErr = normalizeExistingPath(parseCwd, parseResolved.sourcePath)
	if parseErr != nil {
		return importConfig{}, fmt.Errorf("resolve source path: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseResolved.sourcePath)
	if parseErr != nil {
		return importConfig{}, fmt.Errorf("inspect source path: %w", parseErr)
	}
	if parseInfo.IsDir() {
		return importConfig{}, fmt.Errorf("import source must be a file: %s", parseResolved.sourcePath)
	}

	parseResolved.sourceBase = filepath.Base(parseResolved.sourcePath)
	parseResolved.sourceKind, parseErr = detectImportSourceKind(parseResolved.sourcePath)
	if parseErr != nil {
		return importConfig{}, parseErr
	}
	if strings.TrimSpace(parseResolved.outputPath) != "" {
		parseResolved.outputPath, parseErr = normalizePath(parseCwd, parseResolved.outputPath)
		if parseErr != nil {
			return importConfig{}, fmt.Errorf("resolve output path: %w", parseErr)
		}
	}

	parseSourceBytes, parseErr := os.ReadFile(parseResolved.sourcePath)
	if parseErr != nil {
		return importConfig{}, fmt.Errorf("read source file: %w", parseErr)
	}
	parseResolved.document, parseErr = parseImportedDocument(parseResolved.sourcePath, parseSourceBytes)
	if parseErr != nil {
		return importConfig{}, parseErr
	}
	return parseResolved, nil
}

func (parseL launcher) executeImport(parseConfig importConfig) (string, importSummary, error) {
	parseRepoModulePath, parseErr := parseL.readRepoModulePath()
	if parseErr != nil {
		return "", importSummary{}, parseErr
	}
	parseMainContents, parseErr := renderImportedMain(parseConfig.document, parseRepoModulePath)
	if parseErr != nil {
		return "", importSummary{}, parseErr
	}
	if strings.TrimSpace(parseConfig.outputPath) != "" {
		if parseErr2 := writeImportedMainFile(parseConfig.outputPath, parseMainContents); parseErr2 != nil {
			return "", importSummary{}, parseErr2
		}
	}
	parseSummary := importSummary{
		OK:         true,
		SourceKind: parseConfig.sourceKind,
		SourcePath: parseConfig.sourcePath,
		OutputPath: parseConfig.outputPath,
	}
	return parseMainContents, parseSummary, nil
}

func writeImportedMainFile(parseOutputPath string, parseMainContents string) error {
	if strings.TrimSpace(parseOutputPath) == "" {
		return errors.New("import output path is required")
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseOutputPath), 0755); parseErr != nil {
		return fmt.Errorf("create import output directory: %w", parseErr)
	}
	if parseErr2 := scaffoldWriteFile(parseOutputPath, []byte(parseMainContents), 0644); parseErr2 != nil {
		return fmt.Errorf("write imported main.go: %w", parseErr2)
	}
	if parseErr3 := scaffoldFormatMain(parseOutputPath); parseErr3 != nil {
		return fmt.Errorf("format imported main.go: %w", parseErr3)
	}
	return nil
}

func detectImportSourceKind(parsePath string) (string, error) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(parsePath))) {
	case ".html", ".htm":
		return "html", nil
	case ".jsx", ".tsx":
		return "jsx", nil
	default:
		return "", errors.New("import supports only .html, .htm, .jsx, or .tsx files")
	}
}

func defaultImportedProjectName(parseBase string) string {
	parseBase = strings.TrimSuffix(parseBase, filepath.Ext(parseBase))
	var parseBuilder strings.Builder
	isParseLastDash := false
	for _, parseR := range strings.ToLower(parseBase) {
		switch {
		case unicode.IsLetter(parseR) || unicode.IsDigit(parseR):
			parseBuilder.WriteRune(parseR)
			isParseLastDash = false
		case !isParseLastDash:
			parseBuilder.WriteByte('-')
			isParseLastDash = true
		}
	}
	parseName := strings.Trim(parseBuilder.String(), "-")
	if parseName == "" {
		return "imported-app"
	}
	return parseName
}

func parseImportedDocument(parseSourcePath string, parseSource []byte) (importedDocument, error) {
	parseSourceKind, parseErr := detectImportSourceKind(parseSourcePath)
	if parseErr != nil {
		return importedDocument{}, parseErr
	}
	switch parseSourceKind {
	case "html":
		return parseImportedHTMLDocument(parseSource)
	case "jsx":
		return parseImportedJSXDocument(string(parseSource))
	default:
		return importedDocument{}, fmt.Errorf("unsupported import kind %q", parseSourceKind)
	}
}

func parseImportedHTMLDocument(parseSource []byte) (importedDocument, error) {
	parseDoc, parseErr := xhtml.Parse(bytes.NewReader(parseSource))
	if parseErr != nil {
		return importedDocument{}, fmt.Errorf("parse html source: %w", parseErr)
	}
	parseHtmlNode := findHTMLElement(parseDoc, "html")
	parseHeadNode := findHTMLElement(parseDoc, "head")
	parseBodyNode := findHTMLElement(parseDoc, "body")
	if parseBodyNode == nil {
		return importedDocument{}, errors.New("parse html source: missing <body> element")
	}
	parseResult := importedDocument{SourceKind: "html"}
	if parseHtmlNode != nil {
		parseResult.Lang = strings.TrimSpace(importedHTMLAttrValue(parseHtmlNode, "lang"))
	}
	if parseHeadNode != nil {
		for parseChild := parseHeadNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			parseConverted := convertHTMLNode(parseChild, "head")
			if parseConverted == nil {
				continue
			}
			if parseConverted.Kind == importedNodeElement && strings.EqualFold(parseConverted.Tag, "title") {
				parseResult.Title = importedNodeTextContent(*parseConverted)
				continue
			}
			parseResult.HeadNodes = append(parseResult.HeadNodes, *parseConverted)
		}
	}
	parseResult.BodyAttrs = convertImportedAttrsFromHTML(parseBodyNode.Attr)
	for parseChild2 := parseBodyNode.FirstChild; parseChild2 != nil; parseChild2 = parseChild2.NextSibling {
		parseConverted2 := convertHTMLNode(parseChild2, parseBodyNode.Data)
		if parseConverted2 != nil {
			parseResult.Roots = append(parseResult.Roots, *parseConverted2)
		}
	}
	return parseResult, nil
}

func findHTMLElement(parseNode *xhtml.Node, parseName string) *xhtml.Node {
	if parseNode == nil {
		return nil
	}
	if parseNode.Type == xhtml.ElementNode && strings.EqualFold(parseNode.Data, parseName) {
		return parseNode
	}
	for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
		if parseFound := findHTMLElement(parseChild, parseName); parseFound != nil {
			return parseFound
		}
	}
	return nil
}

func importedHTMLAttrValue(parseNode *xhtml.Node, parseName string) string {
	for _, parseAttr := range parseNode.Attr {
		if strings.EqualFold(parseAttr.Key, parseName) {
			return parseAttr.Val
		}
	}
	return ""
}

func convertHTMLNode(parseNode *xhtml.Node, parseParentTag string) *importedNode {
	if parseNode == nil {
		return nil
	}
	switch parseNode.Type {
	case xhtml.TextNode:
		parseText := normalizeImportedText(parseNode.Data, parseParentTag)
		if parseText == "" {
			return nil
		}
		return &importedNode{Kind: importedNodeText, Text: parseText}
	case xhtml.ElementNode:
		parseConverted := &importedNode{
			Kind:  importedNodeElement,
			Tag:   strings.ToLower(strings.TrimSpace(parseNode.Data)),
			Attrs: convertImportedAttrsFromHTML(parseNode.Attr),
		}
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			parseConvertedChild := convertHTMLNode(parseChild, parseConverted.Tag)
			if parseConvertedChild != nil {
				parseConverted.Children = append(parseConverted.Children, *parseConvertedChild)
			}
		}
		return parseConverted
	default:
		return nil
	}
}

func convertImportedAttrsFromHTML(parseAttrs []xhtml.Attribute) []importedAttr {
	parseConverted := make([]importedAttr, 0, len(parseAttrs))
	for _, parseAttr := range parseAttrs {
		parseName := strings.TrimSpace(parseAttr.Key)
		if parseName == "" {
			continue
		}
		parseValue := importedValue{Kind: importedValueString, String: parseAttr.Val}
		if parseAttr.Val == "" && isImportedBooleanAttr(parseName) {
			parseValue = importedValue{Kind: importedValueBool, Bool: true}
		}
		parseConverted = append(parseConverted, importedAttr{Name: parseName, Value: parseValue})
	}
	return parseConverted
}

func parseImportedJSXDocument(parseSource string) (importedDocument, error) {
	parseStart, parseErr := findJSXStart(parseSource)
	if parseErr != nil {
		return importedDocument{}, parseErr
	}
	parseParser := &jsxParser{source: parseSource, index: parseStart}
	parseNode, parseErr := parseParser.parseNode()
	if parseErr != nil {
		return importedDocument{}, parseErr
	}
	if parseNode.Kind == "" {
		return importedDocument{}, errors.New("parse jsx source: no static JSX markup was found")
	}
	parseNodes := []importedNode{parseNode}
	if parseNode.Kind == importedNodeFragment {
		parseNodes = append([]importedNode(nil), parseNode.Children...)
	}
	if len(parseNodes) == 0 {
		return importedDocument{}, errors.New("parse jsx source: no static JSX markup was found")
	}
	parseResult := importedDocument{SourceKind: "jsx", Roots: parseNodes}
	if len(parseNodes) == 1 && parseNodes[0].Kind == importedNodeElement && strings.EqualFold(parseNodes[0].Tag, "html") {
		parseResult = importedDocument{SourceKind: "jsx"}
		for _, parseAttr := range parseNodes[0].Attrs {
			if strings.EqualFold(parseAttr.Name, "lang") {
				parseResult.Lang = importedValueAsString(parseAttr.Value)
			}
		}
		for _, parseChild := range parseNodes[0].Children {
			if parseChild.Kind != importedNodeElement {
				continue
			}
			switch strings.ToLower(parseChild.Tag) {
			case "head":
				for _, parseHeadChild := range parseChild.Children {
					if parseHeadChild.Kind == importedNodeElement && strings.EqualFold(parseHeadChild.Tag, "title") {
						parseResult.Title = importedNodeTextContent(parseHeadChild)
						continue
					}
					parseResult.HeadNodes = append(parseResult.HeadNodes, parseHeadChild)
				}
			case "body":
				parseResult.BodyAttrs = append(parseResult.BodyAttrs, parseChild.Attrs...)
				parseResult.Roots = append(parseResult.Roots, parseChild.Children...)
			}
		}
	}
	return parseResult, nil
}

func findJSXStart(parseSource string) (int, error) {
	parseTrimmed := strings.TrimSpace(parseSource)
	if parseTrimmed == "" {
		return 0, errors.New("parse jsx source: source file is empty")
	}
	if strings.HasPrefix(parseTrimmed, "<") {
		return strings.Index(parseSource, "<"), nil
	}
	parseBest := -1
	if parseIdx := findKeywordOutsideJSX(parseSource, "return"); parseIdx >= 0 {
		if parseNext := findCharOutsideJSX(parseSource[parseIdx+len("return"):], '<'); parseNext >= 0 {
			parseBest = parseIdx + len("return") + parseNext
		}
	}
	if parseBest >= 0 {
		return parseBest, nil
	}
	if parseIdx2 := findCharOutsideJSX(parseSource, '<'); parseIdx2 >= 0 {
		return parseIdx2, nil
	}
	return 0, errors.New("parse jsx source: could not find a static JSX root")
}

func findKeywordOutsideJSX(parseSource string, parseKeyword string) int {
	for parseIndex := 0; parseIndex < len(parseSource); parseIndex++ {
		if !strings.HasPrefix(parseSource[parseIndex:], parseKeyword) {
			continue
		}
		if parseIndex > 0 {
			parsePrevious, _ := utf8.DecodeLastRuneInString(parseSource[:parseIndex])
			if unicode.IsLetter(parsePrevious) || unicode.IsDigit(parsePrevious) || parsePrevious == '_' {
				continue
			}
		}
		parseNextIndex := parseIndex + len(parseKeyword)
		if parseNextIndex < len(parseSource) {
			parseNextRune, _ := utf8.DecodeRuneInString(parseSource[parseNextIndex:])
			if unicode.IsLetter(parseNextRune) || unicode.IsDigit(parseNextRune) || parseNextRune == '_' {
				continue
			}
		}
		return parseIndex
	}
	return -1
}

func findCharOutsideJSX(parseSource string, parseTarget byte) int {
	isParseInSingle := false
	isParseInDouble := false
	isParseInBacktick := false
	isParseInLineComment := false
	isParseInBlockComment := false
	isParseEscaped := false
	for parseIndex := 0; parseIndex < len(parseSource); parseIndex++ {
		parseChar := parseSource[parseIndex]
		parseNext := byte(0)
		if parseIndex+1 < len(parseSource) {
			parseNext = parseSource[parseIndex+1]
		}
		switch {
		case isParseInLineComment:
			if parseChar == '\n' {
				isParseInLineComment = false
			}
		case isParseInBlockComment:
			if parseChar == '*' && parseNext == '/' {
				isParseInBlockComment = false
				parseIndex++
			}
		case isParseInSingle:
			if isParseEscaped {
				isParseEscaped = false
				continue
			}
			if parseChar == '\\' {
				isParseEscaped = true
				continue
			}
			if parseChar == '\'' {
				isParseInSingle = false
			}
		case isParseInDouble:
			if isParseEscaped {
				isParseEscaped = false
				continue
			}
			if parseChar == '\\' {
				isParseEscaped = true
				continue
			}
			if parseChar == '"' {
				isParseInDouble = false
			}
		case isParseInBacktick:
			if isParseEscaped {
				isParseEscaped = false
				continue
			}
			if parseChar == '\\' {
				isParseEscaped = true
				continue
			}
			if parseChar == '`' {
				isParseInBacktick = false
			}
		default:
			if parseChar == '/' && parseNext == '/' {
				isParseInLineComment = true
				parseIndex++
				continue
			}
			if parseChar == '/' && parseNext == '*' {
				isParseInBlockComment = true
				parseIndex++
				continue
			}
			switch parseChar {
			case '\'':
				isParseInSingle = true
			case '"':
				isParseInDouble = true
			case '`':
				isParseInBacktick = true
			case parseTarget:
				return parseIndex
			}
		}
	}
	return -1
}

func (parseP *jsxParser) parseNodesUntil(parseClosingTag string) ([]importedNode, error) {
	parseNodes := []importedNode{}
	for {
		parseP.skipWhitespace()
		if parseP.index >= len(parseP.source) {
			if parseClosingTag != "" {
				return nil, parseP.errorf("expected closing tag </%s>", parseClosingTag)
			}
			return parseNodes, nil
		}
		if parseClosingTag == "" && strings.HasPrefix(parseP.source[parseP.index:], "</>") {
			return parseNodes, nil
		}
		if parseClosingTag != "" && strings.HasPrefix(parseP.source[parseP.index:], "</") {
			parseName, parseErr := parseP.parseClosingTag()
			if parseErr != nil {
				return nil, parseErr
			}
			if !strings.EqualFold(parseName, parseClosingTag) {
				return nil, parseP.errorf("expected closing tag </%s> but found </%s>", parseClosingTag, parseName)
			}
			return parseNodes, nil
		}
		parseNode, parseErr2 := parseP.parseNode()
		if parseErr2 != nil {
			return nil, parseErr2
		}
		if parseNode.Kind == "" {
			continue
		}
		parseNodes = append(parseNodes, parseNode)
	}
}

func (parseP *jsxParser) parseNode() (importedNode, error) {
	if parseP.index >= len(parseP.source) {
		return importedNode{}, nil
	}
	if strings.HasPrefix(parseP.source[parseP.index:], "<") {
		return parseP.parseElement()
	}
	if strings.HasPrefix(parseP.source[parseP.index:], "{") {
		parseContent, parseErr := parseP.readBalanced('{', '}')
		if parseErr != nil {
			return importedNode{}, parseErr
		}
		parseTrimmed := strings.TrimSpace(parseContent)
		if strings.HasPrefix(parseTrimmed, "/*") && strings.HasSuffix(parseTrimmed, "*/") {
			return importedNode{}, nil
		}
		if strings.HasPrefix(parseTrimmed, "<") {
			parseChildParser := &jsxParser{source: parseTrimmed}
			parseNodes, parseErr2 := parseChildParser.parseNodesUntil("")
			if parseErr2 != nil {
				return importedNode{}, parseErr2
			}
			if len(parseNodes) == 1 {
				return parseNodes[0], nil
			}
			return importedNode{Kind: importedNodeFragment, Children: parseNodes}, nil
		}
		parseValue, parseErr := parseImportedExpressionLiteral(parseTrimmed)
		if parseErr != nil {
			return importedNode{}, parseP.errorf("unsupported JSX child expression: %s", parseTrimmed)
		}
		if parseValue.Kind == importedValueNull || (parseValue.Kind == importedValueBool && !parseValue.Bool) {
			return importedNode{}, nil
		}
		if parseValue.Kind == importedValueBool {
			return importedNode{}, nil
		}
		return importedNode{Kind: importedNodeText, Text: importedValueAsString(parseValue)}, nil
	}
	parseText := parseP.readText()
	parseText = normalizeImportedText(parseText, "")
	if parseText == "" {
		return importedNode{}, nil
	}
	return importedNode{Kind: importedNodeText, Text: parseText}, nil
}

func (parseP *jsxParser) parseElement() (importedNode, error) {
	if !strings.HasPrefix(parseP.source[parseP.index:], "<") {
		return importedNode{}, parseP.errorf("expected element")
	}
	parseP.index++
	if strings.HasPrefix(parseP.source[parseP.index:], ">") {
		parseP.index++
		parseChildren, parseErr := parseP.parseNodesUntil("")
		if parseErr != nil {
			return importedNode{}, parseErr
		}
		if !strings.HasPrefix(parseP.source[parseP.index:], "</>") {
			return importedNode{}, parseP.errorf("expected closing fragment </>")
		}
		parseP.index += 3
		return importedNode{Kind: importedNodeFragment, Children: parseChildren}, nil
	}
	parseName := parseP.readTagName()
	if parseName == "" {
		return importedNode{}, parseP.errorf("expected tag name")
	}
	parseAttrs := []importedAttr{}
	for {
		parseP.skipWhitespace()
		if parseP.index >= len(parseP.source) {
			return importedNode{}, parseP.errorf("unexpected end of input inside <%s>", parseName)
		}
		if strings.HasPrefix(parseP.source[parseP.index:], "/>") {
			parseP.index += 2
			return importedNode{Kind: importedNodeElement, Tag: parseName, Attrs: parseAttrs}, nil
		}
		if strings.HasPrefix(parseP.source[parseP.index:], ">") {
			parseP.index++
			parseChildren2, parseErr2 := parseP.parseNodesUntil(parseName)
			if parseErr2 != nil {
				return importedNode{}, parseErr2
			}
			return importedNode{Kind: importedNodeElement, Tag: parseName, Attrs: parseAttrs, Children: parseChildren2}, nil
		}
		if strings.HasPrefix(parseP.source[parseP.index:], "{") {
			parseContent, parseErr3 := parseP.readBalanced('{', '}')
			if parseErr3 != nil {
				return importedNode{}, parseErr3
			}
			if strings.HasPrefix(strings.TrimSpace(parseContent), "...") {
				return importedNode{}, parseP.errorf("JSX spread attributes are not supported")
			}
			return importedNode{}, parseP.errorf("unsupported JSX attribute expression {%s}", strings.TrimSpace(parseContent))
		}
		parseAttrName := parseP.readAttrName()
		if parseAttrName == "" {
			return importedNode{}, parseP.errorf("expected attribute name in <%s>", parseName)
		}
		parseP.skipWhitespace()
		if parseP.index >= len(parseP.source) || parseP.source[parseP.index] != '=' {
			parseAttrs = append(parseAttrs, importedAttr{Name: parseAttrName, Value: importedValue{Kind: importedValueBool, Bool: true}})
			continue
		}
		parseP.index++
		parseP.skipWhitespace()
		parseValue, parseErr4 := parseP.parseAttrValue(parseAttrName)
		if parseErr4 != nil {
			return importedNode{}, parseErr4
		}
		parseAttrs = append(parseAttrs, importedAttr{Name: parseAttrName, Value: parseValue})
	}
}

func (parseP *jsxParser) parseAttrValue(parseAttrName string) (importedValue, error) {
	if parseP.index >= len(parseP.source) {
		return importedValue{}, parseP.errorf("expected attribute value for %s", parseAttrName)
	}
	switch parseP.source[parseP.index] {
	case '\'', '"':
		parseValue, parseErr := parseP.readQuotedString()
		if parseErr != nil {
			return importedValue{}, parseErr
		}
		return importedValue{Kind: importedValueString, String: parseValue}, nil
	case '{':
		parseContent, parseErr2 := parseP.readBalanced('{', '}')
		if parseErr2 != nil {
			return importedValue{}, parseErr2
		}
		parseTrimmed := strings.TrimSpace(parseContent)
		if strings.EqualFold(parseAttrName, "style") && strings.HasPrefix(parseTrimmed, "{") && strings.HasSuffix(parseTrimmed, "}") {
			parseStyle, parseStyleErr := parseImportedJSXStyleObject(parseTrimmed[1 : len(parseTrimmed)-1])
			if parseStyleErr != nil {
				return importedValue{}, parseP.errorf("unsupported JSX style object for %s: %v", parseAttrName, parseStyleErr)
			}
			return importedValue{Kind: importedValueStyle, Style: parseStyle, RawSource: parseTrimmed}, nil
		}
		parseValue2, parseErr := parseImportedExpressionLiteral(parseTrimmed)
		if parseErr != nil {
			return importedValue{}, parseP.errorf("unsupported JSX attribute expression for %s: %s", parseAttrName, parseTrimmed)
		}
		return parseValue2, nil
	default:
		return importedValue{}, parseP.errorf("expected quoted or braced attribute value for %s", parseAttrName)
	}
}

func (parseP *jsxParser) parseClosingTag() (string, error) {
	if !strings.HasPrefix(parseP.source[parseP.index:], "</") {
		return "", parseP.errorf("expected closing tag")
	}
	parseP.index += 2
	parseP.skipWhitespace()
	if strings.HasPrefix(parseP.source[parseP.index:], ">") {
		parseP.index++
		return "", nil
	}
	parseName := parseP.readTagName()
	parseP.skipWhitespace()
	if parseP.index >= len(parseP.source) || parseP.source[parseP.index] != '>' {
		return "", parseP.errorf("expected > to close </%s>", parseName)
	}
	parseP.index++
	return parseName, nil
}

func (parseP *jsxParser) readText() string {
	parseStart := parseP.index
	for parseP.index < len(parseP.source) {
		if parseP.source[parseP.index] == '<' || parseP.source[parseP.index] == '{' {
			break
		}
		parseP.index++
	}
	return parseP.source[parseStart:parseP.index]
}

func (parseP *jsxParser) readTagName() string {
	parseStart := parseP.index
	for parseP.index < len(parseP.source) {
		parseChar := parseP.source[parseP.index]
		if unicode.IsLetter(rune(parseChar)) || unicode.IsDigit(rune(parseChar)) || parseChar == '-' || parseChar == ':' || parseChar == '_' || parseChar == '.' {
			parseP.index++
			continue
		}
		break
	}
	return strings.TrimSpace(parseP.source[parseStart:parseP.index])
}

func (parseP *jsxParser) readAttrName() string {
	return parseP.readTagName()
}

func (parseP *jsxParser) readQuotedString() (string, error) {
	if parseP.index >= len(parseP.source) {
		return "", parseP.errorf("expected quoted string")
	}
	parseQuote := parseP.source[parseP.index]
	parseP.index++
	var parseBuilder strings.Builder
	isParseEscaped := false
	for parseP.index < len(parseP.source) {
		parseChar := parseP.source[parseP.index]
		parseP.index++
		if isParseEscaped {
			parseBuilder.WriteByte(parseChar)
			isParseEscaped = false
			continue
		}
		if parseChar == '\\' {
			isParseEscaped = true
			continue
		}
		if parseChar == parseQuote {
			return parseBuilder.String(), nil
		}
		parseBuilder.WriteByte(parseChar)
	}
	return "", parseP.errorf("unterminated string literal")
}

func (parseP *jsxParser) readBalanced(parseOpen byte, parseClose byte) (string, error) {
	if parseP.index >= len(parseP.source) || parseP.source[parseP.index] != parseOpen {
		return "", parseP.errorf("expected %c", parseOpen)
	}
	parseStart := parseP.index + 1
	parseP.index++
	parseDepth := 1
	isParseInSingle := false
	isParseInDouble := false
	isParseInBacktick := false
	isParseEscaped := false
	for parseP.index < len(parseP.source) {
		parseChar := parseP.source[parseP.index]
		if isParseEscaped {
			isParseEscaped = false
			parseP.index++
			continue
		}
		if parseChar == '\\' && (isParseInSingle || isParseInDouble || isParseInBacktick) {
			isParseEscaped = true
			parseP.index++
			continue
		}
		switch parseChar {
		case '\'':
			if !isParseInDouble && !isParseInBacktick {
				isParseInSingle = !isParseInSingle
			}
		case '"':
			if !isParseInSingle && !isParseInBacktick {
				isParseInDouble = !isParseInDouble
			}
		case '`':
			if !isParseInSingle && !isParseInDouble {
				isParseInBacktick = !isParseInBacktick
			}
		default:
			if !isParseInSingle && !isParseInDouble && !isParseInBacktick {
				switch parseChar {
				case parseOpen:
					parseDepth++
				case parseClose:
					parseDepth--
					if parseDepth == 0 {
						parseContent := parseP.source[parseStart:parseP.index]
						parseP.index++
						return parseContent, nil
					}
				}
			}
		}
		parseP.index++
	}
	return "", parseP.errorf("unterminated %c expression", parseOpen)
}

func (parseP *jsxParser) skipWhitespace() {
	for parseP.index < len(parseP.source) {
		if !unicode.IsSpace(rune(parseP.source[parseP.index])) {
			return
		}
		parseP.index++
	}
}

func (parseP *jsxParser) errorf(format string, parseArgs ...interface{}) error {
	parseLine := 1
	parseColumn := 1
	for _, parseChar := range parseP.source[:parseP.index] {
		if parseChar == '\n' {
			parseLine++
			parseColumn = 1
			continue
		}
		parseColumn++
	}
	return fmt.Errorf("parse jsx source:%d:%d: %s", parseLine, parseColumn, fmt.Sprintf(format, parseArgs...))
}

func parseImportedExpressionLiteral(parseRaw string) (importedValue, error) {
	parseTrimmed := strings.TrimSpace(parseRaw)
	if parseTrimmed == "" {
		return importedValue{Kind: importedValueNull}, nil
	}
	if strings.HasPrefix(parseTrimmed, "\"") || strings.HasPrefix(parseTrimmed, "'") {
		parseUnquoted, parseErr := strconv.Unquote(parseTrimmed)
		if parseErr != nil {
			return importedValue{}, parseErr
		}
		return importedValue{Kind: importedValueString, String: parseUnquoted}, nil
	}
	if strings.HasPrefix(parseTrimmed, "`") && strings.HasSuffix(parseTrimmed, "`") {
		if strings.Contains(parseTrimmed, "${") {
			return importedValue{}, errors.New("template literal interpolation is not supported")
		}
		return importedValue{Kind: importedValueString, String: strings.Trim(parseTrimmed, "`")}, nil
	}
	switch parseTrimmed {
	case "true":
		return importedValue{Kind: importedValueBool, Bool: true}, nil
	case "false":
		return importedValue{Kind: importedValueBool, Bool: false}, nil
	case "null", "undefined":
		return importedValue{Kind: importedValueNull}, nil
	}
	if _, parseErr2 := strconv.ParseFloat(parseTrimmed, 64); parseErr2 == nil {
		return importedValue{Kind: importedValueNumber, Number: parseTrimmed}, nil
	}
	return importedValue{}, errors.New("dynamic expressions are not supported")
}

func parseImportedJSXStyleObject(parseRaw string) (map[string]string, error) {
	parseStyle := map[string]string{}
	parseParts, parseErr := splitTopLevel(parseRaw, ',')
	if parseErr != nil {
		return nil, parseErr
	}
	for _, parsePart := range parseParts {
		parsePart = strings.TrimSpace(parsePart)
		if parsePart == "" {
			continue
		}
		parseSegments, parseSplitErr := splitTopLevel(parsePart, ':')
		if parseSplitErr != nil || len(parseSegments) < 2 {
			return nil, fmt.Errorf("invalid style entry %q", parsePart)
		}
		parseKey := strings.TrimSpace(parseSegments[0])
		parseValue := strings.TrimSpace(strings.Join(parseSegments[1:], ":"))
		if strings.HasPrefix(parseKey, "\"") || strings.HasPrefix(parseKey, "'") {
			parseUnquoted, parseUnquoteErr := strconv.Unquote(parseKey)
			if parseUnquoteErr != nil {
				return nil, parseUnquoteErr
			}
			parseKey = parseUnquoted
		}
		parseLiteral, parseLiteralErr := parseImportedExpressionLiteral(parseValue)
		if parseLiteralErr != nil {
			return nil, parseLiteralErr
		}
		parseStyle[camelToKebab(strings.TrimSpace(parseKey))] = importedValueAsString(parseLiteral)
	}
	return parseStyle, nil
}

func splitTopLevel(parseRaw string, parseDelimiter rune) ([]string, error) {
	parseParts := []string{}
	parseStart := 0
	parseDepth := 0
	isParseInSingle := false
	isParseInDouble := false
	isParseInBacktick := false
	isParseEscaped := false
	for parseIndex, parseChar := range parseRaw {
		if isParseEscaped {
			isParseEscaped = false
			continue
		}
		if parseChar == '\\' && (isParseInSingle || isParseInDouble || isParseInBacktick) {
			isParseEscaped = true
			continue
		}
		switch parseChar {
		case '\'':
			if !isParseInDouble && !isParseInBacktick {
				isParseInSingle = !isParseInSingle
			}
		case '"':
			if !isParseInSingle && !isParseInBacktick {
				isParseInDouble = !isParseInDouble
			}
		case '`':
			if !isParseInSingle && !isParseInDouble {
				isParseInBacktick = !isParseInBacktick
			}
		case '{', '[', '(':
			if !isParseInSingle && !isParseInDouble && !isParseInBacktick {
				parseDepth++
			}
		case '}', ']', ')':
			if !isParseInSingle && !isParseInDouble && !isParseInBacktick {
				if parseDepth == 0 {
					return nil, errors.New("unexpected closing delimiter")
				}
				parseDepth--
			}
		default:
			if parseChar == parseDelimiter && parseDepth == 0 && !isParseInSingle && !isParseInDouble && !isParseInBacktick {
				parseParts = append(parseParts, parseRaw[parseStart:parseIndex])
				parseStart = parseIndex + 1
			}
		}
	}
	if parseDepth != 0 || isParseInSingle || isParseInDouble || isParseInBacktick {
		return nil, errors.New("unterminated object literal")
	}
	parseParts = append(parseParts, parseRaw[parseStart:])
	return parseParts, nil
}

func camelToKebab(parseValue string) string {
	if strings.HasPrefix(parseValue, "--") {
		return parseValue
	}
	var parseBuilder strings.Builder
	for parseIndex, parseChar := range parseValue {
		if unicode.IsUpper(parseChar) {
			if parseIndex > 0 {
				parseBuilder.WriteByte('-')
			}
			parseBuilder.WriteRune(unicode.ToLower(parseChar))
			continue
		}
		parseBuilder.WriteRune(parseChar)
	}
	return parseBuilder.String()
}
