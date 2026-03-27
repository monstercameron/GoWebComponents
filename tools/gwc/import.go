package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	stdhtml "html"
	"os"
	"path/filepath"
	"sort"
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
	select {}
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
