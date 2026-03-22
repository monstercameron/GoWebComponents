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
	Selection  startSelection
	MainGo     string
	HTML       string
	README     string
	Metadata   scaffoldMetadata
	ExtraFiles map[string][]byte
}

type jsxParser struct {
	source string
	index  int
}

var launcherMkdirTemp = os.MkdirTemp

func (l launcher) runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	source := fs.String("src", "", "Path to a valid .html, .htm, .jsx, or .tsx file")
	out := fs.String("out", "", "Path to write the generated main.go file; prints to stdout when omitted")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := l.resolveImportConfig(importConfig{
		sourcePath: *source,
		outputPath: *out,
		json:       *jsonOutput,
	})
	if err != nil {
		return err
	}

	mainContents, summary, err := l.executeImport(config)
	if err != nil {
		return err
	}
	if strings.TrimSpace(config.outputPath) == "" {
		if config.json {
			return errors.New("import requires -out when -json is used")
		}
		fmt.Fprint(os.Stdout, mainContents)
		return nil
	}
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}
	printImportSummary(summary)
	return nil
}

func (l launcher) resolveImportConfig(config importConfig) (importConfig, error) {
	resolved := config
	cwd, err := buildGetwd()
	if err != nil {
		return importConfig{}, err
	}
	if strings.TrimSpace(resolved.sourcePath) == "" {
		return importConfig{}, errors.New("import requires -src")
	}
	resolved.sourcePath, err = normalizeExistingPath(cwd, resolved.sourcePath)
	if err != nil {
		return importConfig{}, fmt.Errorf("resolve source path: %w", err)
	}
	info, err := os.Stat(resolved.sourcePath)
	if err != nil {
		return importConfig{}, fmt.Errorf("inspect source path: %w", err)
	}
	if info.IsDir() {
		return importConfig{}, fmt.Errorf("import source must be a file: %s", resolved.sourcePath)
	}

	resolved.sourceBase = filepath.Base(resolved.sourcePath)
	resolved.sourceKind, err = detectImportSourceKind(resolved.sourcePath)
	if err != nil {
		return importConfig{}, err
	}
	if strings.TrimSpace(resolved.outputPath) != "" {
		resolved.outputPath, err = normalizePath(cwd, resolved.outputPath)
		if err != nil {
			return importConfig{}, fmt.Errorf("resolve output path: %w", err)
		}
	}

	sourceBytes, err := os.ReadFile(resolved.sourcePath)
	if err != nil {
		return importConfig{}, fmt.Errorf("read source file: %w", err)
	}
	resolved.document, err = parseImportedDocument(resolved.sourcePath, sourceBytes)
	if err != nil {
		return importConfig{}, err
	}
	return resolved, nil
}

func (l launcher) executeImport(config importConfig) (string, importSummary, error) {
	repoModulePath, err := l.readRepoModulePath()
	if err != nil {
		return "", importSummary{}, err
	}
	mainContents, err := renderImportedMain(config.document, repoModulePath)
	if err != nil {
		return "", importSummary{}, err
	}
	if strings.TrimSpace(config.outputPath) != "" {
		if err := writeImportedMainFile(config.outputPath, mainContents); err != nil {
			return "", importSummary{}, err
		}
	}
	summary := importSummary{
		OK:         true,
		SourceKind: config.sourceKind,
		SourcePath: config.sourcePath,
		OutputPath: config.outputPath,
	}
	return mainContents, summary, nil
}

func writeImportedMainFile(outputPath string, mainContents string) error {
	if strings.TrimSpace(outputPath) == "" {
		return errors.New("import output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create import output directory: %w", err)
	}
	if err := scaffoldWriteFile(outputPath, []byte(mainContents), 0644); err != nil {
		return fmt.Errorf("write imported main.go: %w", err)
	}
	if err := scaffoldFormatMain(outputPath); err != nil {
		return fmt.Errorf("format imported main.go: %w", err)
	}
	return nil
}

func detectImportSourceKind(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".html", ".htm":
		return "html", nil
	case ".jsx", ".tsx":
		return "jsx", nil
	default:
		return "", errors.New("import supports only .html, .htm, .jsx, or .tsx files")
	}
}

func defaultImportedProjectName(base string) string {
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(base) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case !lastDash:
			builder.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(builder.String(), "-")
	if name == "" {
		return "imported-app"
	}
	return name
}

func parseImportedDocument(sourcePath string, source []byte) (importedDocument, error) {
	sourceKind, err := detectImportSourceKind(sourcePath)
	if err != nil {
		return importedDocument{}, err
	}
	switch sourceKind {
	case "html":
		return parseImportedHTMLDocument(source)
	case "jsx":
		return parseImportedJSXDocument(string(source))
	default:
		return importedDocument{}, fmt.Errorf("unsupported import kind %q", sourceKind)
	}
}

func parseImportedHTMLDocument(source []byte) (importedDocument, error) {
	doc, err := xhtml.Parse(bytes.NewReader(source))
	if err != nil {
		return importedDocument{}, fmt.Errorf("parse html source: %w", err)
	}
	htmlNode := findHTMLElement(doc, "html")
	headNode := findHTMLElement(doc, "head")
	bodyNode := findHTMLElement(doc, "body")
	if bodyNode == nil {
		return importedDocument{}, errors.New("parse html source: missing <body> element")
	}
	result := importedDocument{SourceKind: "html"}
	if htmlNode != nil {
		result.Lang = strings.TrimSpace(importedHTMLAttrValue(htmlNode, "lang"))
	}
	if headNode != nil {
		for child := headNode.FirstChild; child != nil; child = child.NextSibling {
			converted := convertHTMLNode(child, "head")
			if converted == nil {
				continue
			}
			if converted.Kind == importedNodeElement && strings.EqualFold(converted.Tag, "title") {
				result.Title = importedNodeTextContent(*converted)
				continue
			}
			result.HeadNodes = append(result.HeadNodes, *converted)
		}
	}
	result.BodyAttrs = convertImportedAttrsFromHTML(bodyNode.Attr)
	for child := bodyNode.FirstChild; child != nil; child = child.NextSibling {
		converted := convertHTMLNode(child, bodyNode.Data)
		if converted != nil {
			result.Roots = append(result.Roots, *converted)
		}
	}
	return result, nil
}

func findHTMLElement(node *xhtml.Node, name string) *xhtml.Node {
	if node == nil {
		return nil
	}
	if node.Type == xhtml.ElementNode && strings.EqualFold(node.Data, name) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findHTMLElement(child, name); found != nil {
			return found
		}
	}
	return nil
}

func importedHTMLAttrValue(node *xhtml.Node, name string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}

func convertHTMLNode(node *xhtml.Node, parentTag string) *importedNode {
	if node == nil {
		return nil
	}
	switch node.Type {
	case xhtml.TextNode:
		text := normalizeImportedText(node.Data, parentTag)
		if text == "" {
			return nil
		}
		return &importedNode{Kind: importedNodeText, Text: text}
	case xhtml.ElementNode:
		converted := &importedNode{
			Kind:  importedNodeElement,
			Tag:   strings.ToLower(strings.TrimSpace(node.Data)),
			Attrs: convertImportedAttrsFromHTML(node.Attr),
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			convertedChild := convertHTMLNode(child, converted.Tag)
			if convertedChild != nil {
				converted.Children = append(converted.Children, *convertedChild)
			}
		}
		return converted
	default:
		return nil
	}
}

func convertImportedAttrsFromHTML(attrs []xhtml.Attribute) []importedAttr {
	converted := make([]importedAttr, 0, len(attrs))
	for _, attr := range attrs {
		name := strings.TrimSpace(attr.Key)
		if name == "" {
			continue
		}
		value := importedValue{Kind: importedValueString, String: attr.Val}
		if attr.Val == "" && isImportedBooleanAttr(name) {
			value = importedValue{Kind: importedValueBool, Bool: true}
		}
		converted = append(converted, importedAttr{Name: name, Value: value})
	}
	return converted
}

func parseImportedJSXDocument(source string) (importedDocument, error) {
	start, err := findJSXStart(source)
	if err != nil {
		return importedDocument{}, err
	}
	parser := &jsxParser{source: source, index: start}
	node, err := parser.parseNode()
	if err != nil {
		return importedDocument{}, err
	}
	if node.Kind == "" {
		return importedDocument{}, errors.New("parse jsx source: no static JSX markup was found")
	}
	nodes := []importedNode{node}
	if node.Kind == importedNodeFragment {
		nodes = append([]importedNode(nil), node.Children...)
	}
	if len(nodes) == 0 {
		return importedDocument{}, errors.New("parse jsx source: no static JSX markup was found")
	}
	result := importedDocument{SourceKind: "jsx", Roots: nodes}
	if len(nodes) == 1 && nodes[0].Kind == importedNodeElement && strings.EqualFold(nodes[0].Tag, "html") {
		result = importedDocument{SourceKind: "jsx"}
		for _, attr := range nodes[0].Attrs {
			if strings.EqualFold(attr.Name, "lang") {
				result.Lang = importedValueAsString(attr.Value)
			}
		}
		for _, child := range nodes[0].Children {
			if child.Kind != importedNodeElement {
				continue
			}
			switch strings.ToLower(child.Tag) {
			case "head":
				for _, headChild := range child.Children {
					if headChild.Kind == importedNodeElement && strings.EqualFold(headChild.Tag, "title") {
						result.Title = importedNodeTextContent(headChild)
						continue
					}
					result.HeadNodes = append(result.HeadNodes, headChild)
				}
			case "body":
				result.BodyAttrs = append(result.BodyAttrs, child.Attrs...)
				result.Roots = append(result.Roots, child.Children...)
			}
		}
	}
	return result, nil
}

func findJSXStart(source string) (int, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return 0, errors.New("parse jsx source: source file is empty")
	}
	if strings.HasPrefix(trimmed, "<") {
		return strings.Index(source, "<"), nil
	}
	best := -1
	if idx := findKeywordOutsideJSX(source, "return"); idx >= 0 {
		if next := findCharOutsideJSX(source[idx+len("return"):], '<'); next >= 0 {
			best = idx + len("return") + next
		}
	}
	if best >= 0 {
		return best, nil
	}
	if idx := findCharOutsideJSX(source, '<'); idx >= 0 {
		return idx, nil
	}
	return 0, errors.New("parse jsx source: could not find a static JSX root")
}

func findKeywordOutsideJSX(source string, keyword string) int {
	for index := 0; index < len(source); index++ {
		if !strings.HasPrefix(source[index:], keyword) {
			continue
		}
		if index > 0 {
			previous, _ := utf8.DecodeLastRuneInString(source[:index])
			if unicode.IsLetter(previous) || unicode.IsDigit(previous) || previous == '_' {
				continue
			}
		}
		nextIndex := index + len(keyword)
		if nextIndex < len(source) {
			nextRune, _ := utf8.DecodeRuneInString(source[nextIndex:])
			if unicode.IsLetter(nextRune) || unicode.IsDigit(nextRune) || nextRune == '_' {
				continue
			}
		}
		return index
	}
	return -1
}

func findCharOutsideJSX(source string, target byte) int {
	inSingle := false
	inDouble := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false
	escaped := false
	for index := 0; index < len(source); index++ {
		char := source[index]
		next := byte(0)
		if index+1 < len(source) {
			next = source[index+1]
		}
		switch {
		case inLineComment:
			if char == '\n' {
				inLineComment = false
			}
		case inBlockComment:
			if char == '*' && next == '/' {
				inBlockComment = false
				index++
			}
		case inSingle:
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '\'' {
				inSingle = false
			}
		case inDouble:
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inDouble = false
			}
		case inBacktick:
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '`' {
				inBacktick = false
			}
		default:
			if char == '/' && next == '/' {
				inLineComment = true
				index++
				continue
			}
			if char == '/' && next == '*' {
				inBlockComment = true
				index++
				continue
			}
			switch char {
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case '`':
				inBacktick = true
			case target:
				return index
			}
		}
	}
	return -1
}

func (p *jsxParser) parseNodesUntil(closingTag string) ([]importedNode, error) {
	nodes := []importedNode{}
	for {
		p.skipWhitespace()
		if p.index >= len(p.source) {
			if closingTag != "" {
				return nil, p.errorf("expected closing tag </%s>", closingTag)
			}
			return nodes, nil
		}
		if closingTag == "" && strings.HasPrefix(p.source[p.index:], "</>") {
			return nodes, nil
		}
		if closingTag != "" && strings.HasPrefix(p.source[p.index:], "</") {
			name, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !strings.EqualFold(name, closingTag) {
				return nil, p.errorf("expected closing tag </%s> but found </%s>", closingTag, name)
			}
			return nodes, nil
		}
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node.Kind == "" {
			continue
		}
		nodes = append(nodes, node)
	}
}

func (p *jsxParser) parseNode() (importedNode, error) {
	if p.index >= len(p.source) {
		return importedNode{}, nil
	}
	if strings.HasPrefix(p.source[p.index:], "<") {
		return p.parseElement()
	}
	if strings.HasPrefix(p.source[p.index:], "{") {
		content, err := p.readBalanced('{', '}')
		if err != nil {
			return importedNode{}, err
		}
		trimmed := strings.TrimSpace(content)
		if strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
			return importedNode{}, nil
		}
		if strings.HasPrefix(trimmed, "<") {
			childParser := &jsxParser{source: trimmed}
			nodes, err := childParser.parseNodesUntil("")
			if err != nil {
				return importedNode{}, err
			}
			if len(nodes) == 1 {
				return nodes[0], nil
			}
			return importedNode{Kind: importedNodeFragment, Children: nodes}, nil
		}
		value, err := parseImportedExpressionLiteral(trimmed)
		if err != nil {
			return importedNode{}, p.errorf("unsupported JSX child expression: %s", trimmed)
		}
		if value.Kind == importedValueNull || (value.Kind == importedValueBool && !value.Bool) {
			return importedNode{}, nil
		}
		if value.Kind == importedValueBool {
			return importedNode{}, nil
		}
		return importedNode{Kind: importedNodeText, Text: importedValueAsString(value)}, nil
	}
	text := p.readText()
	text = normalizeImportedText(text, "")
	if text == "" {
		return importedNode{}, nil
	}
	return importedNode{Kind: importedNodeText, Text: text}, nil
}

func (p *jsxParser) parseElement() (importedNode, error) {
	if !strings.HasPrefix(p.source[p.index:], "<") {
		return importedNode{}, p.errorf("expected element")
	}
	p.index++
	if strings.HasPrefix(p.source[p.index:], ">") {
		p.index++
		children, err := p.parseNodesUntil("")
		if err != nil {
			return importedNode{}, err
		}
		if !strings.HasPrefix(p.source[p.index:], "</>") {
			return importedNode{}, p.errorf("expected closing fragment </>")
		}
		p.index += 3
		return importedNode{Kind: importedNodeFragment, Children: children}, nil
	}
	name := p.readTagName()
	if name == "" {
		return importedNode{}, p.errorf("expected tag name")
	}
	attrs := []importedAttr{}
	for {
		p.skipWhitespace()
		if p.index >= len(p.source) {
			return importedNode{}, p.errorf("unexpected end of input inside <%s>", name)
		}
		if strings.HasPrefix(p.source[p.index:], "/>") {
			p.index += 2
			return importedNode{Kind: importedNodeElement, Tag: name, Attrs: attrs}, nil
		}
		if strings.HasPrefix(p.source[p.index:], ">") {
			p.index++
			children, err := p.parseNodesUntil(name)
			if err != nil {
				return importedNode{}, err
			}
			return importedNode{Kind: importedNodeElement, Tag: name, Attrs: attrs, Children: children}, nil
		}
		if strings.HasPrefix(p.source[p.index:], "{") {
			content, err := p.readBalanced('{', '}')
			if err != nil {
				return importedNode{}, err
			}
			if strings.HasPrefix(strings.TrimSpace(content), "...") {
				return importedNode{}, p.errorf("JSX spread attributes are not supported")
			}
			return importedNode{}, p.errorf("unsupported JSX attribute expression {%s}", strings.TrimSpace(content))
		}
		attrName := p.readAttrName()
		if attrName == "" {
			return importedNode{}, p.errorf("expected attribute name in <%s>", name)
		}
		p.skipWhitespace()
		if p.index >= len(p.source) || p.source[p.index] != '=' {
			attrs = append(attrs, importedAttr{Name: attrName, Value: importedValue{Kind: importedValueBool, Bool: true}})
			continue
		}
		p.index++
		p.skipWhitespace()
		value, err := p.parseAttrValue(attrName)
		if err != nil {
			return importedNode{}, err
		}
		attrs = append(attrs, importedAttr{Name: attrName, Value: value})
	}
}

func (p *jsxParser) parseAttrValue(attrName string) (importedValue, error) {
	if p.index >= len(p.source) {
		return importedValue{}, p.errorf("expected attribute value for %s", attrName)
	}
	switch p.source[p.index] {
	case '\'', '"':
		value, err := p.readQuotedString()
		if err != nil {
			return importedValue{}, err
		}
		return importedValue{Kind: importedValueString, String: value}, nil
	case '{':
		content, err := p.readBalanced('{', '}')
		if err != nil {
			return importedValue{}, err
		}
		trimmed := strings.TrimSpace(content)
		if strings.EqualFold(attrName, "style") && strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			style, styleErr := parseImportedJSXStyleObject(trimmed[1 : len(trimmed)-1])
			if styleErr != nil {
				return importedValue{}, p.errorf("unsupported JSX style object for %s: %v", attrName, styleErr)
			}
			return importedValue{Kind: importedValueStyle, Style: style, RawSource: trimmed}, nil
		}
		value, parseErr := parseImportedExpressionLiteral(trimmed)
		if parseErr != nil {
			return importedValue{}, p.errorf("unsupported JSX attribute expression for %s: %s", attrName, trimmed)
		}
		return value, nil
	default:
		return importedValue{}, p.errorf("expected quoted or braced attribute value for %s", attrName)
	}
}

func (p *jsxParser) parseClosingTag() (string, error) {
	if !strings.HasPrefix(p.source[p.index:], "</") {
		return "", p.errorf("expected closing tag")
	}
	p.index += 2
	p.skipWhitespace()
	if strings.HasPrefix(p.source[p.index:], ">") {
		p.index++
		return "", nil
	}
	name := p.readTagName()
	p.skipWhitespace()
	if p.index >= len(p.source) || p.source[p.index] != '>' {
		return "", p.errorf("expected > to close </%s>", name)
	}
	p.index++
	return name, nil
}

func (p *jsxParser) readText() string {
	start := p.index
	for p.index < len(p.source) {
		if p.source[p.index] == '<' || p.source[p.index] == '{' {
			break
		}
		p.index++
	}
	return p.source[start:p.index]
}

func (p *jsxParser) readTagName() string {
	start := p.index
	for p.index < len(p.source) {
		char := p.source[p.index]
		if unicode.IsLetter(rune(char)) || unicode.IsDigit(rune(char)) || char == '-' || char == ':' || char == '_' || char == '.' {
			p.index++
			continue
		}
		break
	}
	return strings.TrimSpace(p.source[start:p.index])
}

func (p *jsxParser) readAttrName() string {
	return p.readTagName()
}

func (p *jsxParser) readQuotedString() (string, error) {
	if p.index >= len(p.source) {
		return "", p.errorf("expected quoted string")
	}
	quote := p.source[p.index]
	p.index++
	var builder strings.Builder
	escaped := false
	for p.index < len(p.source) {
		char := p.source[p.index]
		p.index++
		if escaped {
			builder.WriteByte(char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == quote {
			return builder.String(), nil
		}
		builder.WriteByte(char)
	}
	return "", p.errorf("unterminated string literal")
}

func (p *jsxParser) readBalanced(open byte, close byte) (string, error) {
	if p.index >= len(p.source) || p.source[p.index] != open {
		return "", p.errorf("expected %c", open)
	}
	start := p.index + 1
	p.index++
	depth := 1
	inSingle := false
	inDouble := false
	inBacktick := false
	escaped := false
	for p.index < len(p.source) {
		char := p.source[p.index]
		if escaped {
			escaped = false
			p.index++
			continue
		}
		if char == '\\' && (inSingle || inDouble || inBacktick) {
			escaped = true
			p.index++
			continue
		}
		switch char {
		case '\'':
			if !inDouble && !inBacktick {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && !inBacktick {
				inDouble = !inDouble
			}
		case '`':
			if !inSingle && !inDouble {
				inBacktick = !inBacktick
			}
		default:
			if !inSingle && !inDouble && !inBacktick {
				if char == open {
					depth++
				} else if char == close {
					depth--
					if depth == 0 {
						content := p.source[start:p.index]
						p.index++
						return content, nil
					}
				}
			}
		}
		p.index++
	}
	return "", p.errorf("unterminated %c expression", open)
}

func (p *jsxParser) skipWhitespace() {
	for p.index < len(p.source) {
		if !unicode.IsSpace(rune(p.source[p.index])) {
			return
		}
		p.index++
	}
}

func (p *jsxParser) errorf(format string, args ...interface{}) error {
	line := 1
	column := 1
	for _, char := range p.source[:p.index] {
		if char == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return fmt.Errorf("parse jsx source:%d:%d: %s", line, column, fmt.Sprintf(format, args...))
}

func parseImportedExpressionLiteral(raw string) (importedValue, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return importedValue{Kind: importedValueNull}, nil
	}
	if strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "'") {
		unquoted, err := strconv.Unquote(trimmed)
		if err != nil {
			return importedValue{}, err
		}
		return importedValue{Kind: importedValueString, String: unquoted}, nil
	}
	if strings.HasPrefix(trimmed, "`") && strings.HasSuffix(trimmed, "`") {
		if strings.Contains(trimmed, "${") {
			return importedValue{}, errors.New("template literal interpolation is not supported")
		}
		return importedValue{Kind: importedValueString, String: strings.Trim(trimmed, "`")}, nil
	}
	switch trimmed {
	case "true":
		return importedValue{Kind: importedValueBool, Bool: true}, nil
	case "false":
		return importedValue{Kind: importedValueBool, Bool: false}, nil
	case "null", "undefined":
		return importedValue{Kind: importedValueNull}, nil
	}
	if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return importedValue{Kind: importedValueNumber, Number: trimmed}, nil
	}
	return importedValue{}, errors.New("dynamic expressions are not supported")
}

func parseImportedJSXStyleObject(raw string) (map[string]string, error) {
	style := map[string]string{}
	parts, err := splitTopLevel(raw, ',')
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments, splitErr := splitTopLevel(part, ':')
		if splitErr != nil || len(segments) < 2 {
			return nil, fmt.Errorf("invalid style entry %q", part)
		}
		key := strings.TrimSpace(segments[0])
		value := strings.TrimSpace(strings.Join(segments[1:], ":"))
		if strings.HasPrefix(key, "\"") || strings.HasPrefix(key, "'") {
			unquoted, unquoteErr := strconv.Unquote(key)
			if unquoteErr != nil {
				return nil, unquoteErr
			}
			key = unquoted
		}
		literal, literalErr := parseImportedExpressionLiteral(value)
		if literalErr != nil {
			return nil, literalErr
		}
		style[camelToKebab(strings.TrimSpace(key))] = importedValueAsString(literal)
	}
	return style, nil
}

func splitTopLevel(raw string, delimiter rune) ([]string, error) {
	parts := []string{}
	start := 0
	depth := 0
	inSingle := false
	inDouble := false
	inBacktick := false
	escaped := false
	for index, char := range raw {
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && (inSingle || inDouble || inBacktick) {
			escaped = true
			continue
		}
		switch char {
		case '\'':
			if !inDouble && !inBacktick {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && !inBacktick {
				inDouble = !inDouble
			}
		case '`':
			if !inSingle && !inDouble {
				inBacktick = !inBacktick
			}
		case '{', '[', '(':
			if !inSingle && !inDouble && !inBacktick {
				depth++
			}
		case '}', ']', ')':
			if !inSingle && !inDouble && !inBacktick {
				if depth == 0 {
					return nil, errors.New("unexpected closing delimiter")
				}
				depth--
			}
		default:
			if char == delimiter && depth == 0 && !inSingle && !inDouble && !inBacktick {
				parts = append(parts, raw[start:index])
				start = index + 1
			}
		}
	}
	if depth != 0 || inSingle || inDouble || inBacktick {
		return nil, errors.New("unterminated object literal")
	}
	parts = append(parts, raw[start:])
	return parts, nil
}

func camelToKebab(value string) string {
	if strings.HasPrefix(value, "--") {
		return value
	}
	var builder strings.Builder
	for index, char := range value {
		if unicode.IsUpper(char) {
			if index > 0 {
				builder.WriteByte('-')
			}
			builder.WriteRune(unicode.ToLower(char))
			continue
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func normalizeImportedText(text string, parentTag string) string {
	if preserveImportedWhitespace(parentTag) {
		if text == "" {
			return ""
		}
		return text
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	return strings.Join(strings.Fields(trimmed), " ")
}

func preserveImportedWhitespace(tag string) bool {
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case "pre", "code", "textarea", "style", "script":
		return true
	default:
		return false
	}
}

func importedNodeTextContent(node importedNode) string {
	if node.Kind == importedNodeText {
		return node.Text
	}
	var builder strings.Builder
	for _, child := range node.Children {
		builder.WriteString(importedNodeTextContent(child))
	}
	return strings.TrimSpace(builder.String())
}

func renderImportedMain(document importedDocument, repoModulePath string) (string, error) {
	rootExpr, err := renderImportedRootExpression(document.Roots, "\t")
	if err != nil {
		return "", err
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
`, repoModulePath+"/html", repoModulePath+"/ui", repoModulePath+"/utils", rootExpr, importedMountID), nil
}

func renderImportedRootExpression(nodes []importedNode, indent string) (string, error) {
	if len(nodes) == 0 {
		return "html.Fragment()", nil
	}
	if len(nodes) == 1 {
		return renderImportedNodeExpression(nodes[0], indent)
	}
	fragment := importedNode{Kind: importedNodeFragment, Children: nodes}
	return renderImportedNodeExpression(fragment, indent)
}

func renderImportedNodeExpression(node importedNode, indent string) (string, error) {
	switch node.Kind {
	case importedNodeText:
		return fmt.Sprintf("html.Text(%s)", strconv.Quote(node.Text)), nil
	case importedNodeFragment:
		if len(node.Children) == 0 {
			return "html.Fragment()", nil
		}
		childLines := []string{"html.Fragment("}
		for _, child := range node.Children {
			rendered, err := renderImportedNodeExpression(child, indent+"\t")
			if err != nil {
				return "", err
			}
			childLines = append(childLines, indent+rendered+",")
		}
		childLines = append(childLines, strings.TrimRight(indent, "\t")+")")
		return strings.Join(childLines, "\n"), nil
	case importedNodeElement:
		if err := validateImportedTagName(node.Tag); err != nil {
			return "", err
		}
		propsLiteral, err := renderImportedPropsLiteral(node.Attrs, indent)
		if err != nil {
			return "", err
		}
		builder, typed := importedBuilderName(node.Tag)
		args := []string{}
		if typed {
			args = append(args, propsLiteral)
		} else {
			args = append(args, strconv.Quote(node.Tag), propsLiteral)
		}
		for _, child := range node.Children {
			rendered, childErr := renderImportedNodeExpression(child, indent+"\t")
			if childErr != nil {
				return "", childErr
			}
			args = append(args, rendered)
		}
		prefix := "html." + builder
		if !typed {
			prefix = "html.Tag"
		}
		if len(node.Children) == 0 && !strings.Contains(propsLiteral, "\n") {
			return fmt.Sprintf("%s(%s)", prefix, strings.Join(args, ", ")), nil
		}
		lines := []string{prefix + "("}
		for _, arg := range args {
			argLines := strings.Split(arg, "\n")
			if len(argLines) == 1 {
				lines = append(lines, indent+arg+",")
				continue
			}
			for _, argLine := range argLines {
				lines = append(lines, indent+argLine)
			}
			lines[len(lines)-1] += ","
		}
		lines = append(lines, strings.TrimRight(indent, "\t")+")")
		return strings.Join(lines, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported imported node kind %q", node.Kind)
	}
}

func validateImportedTagName(tag string) error {
	trimmed := strings.TrimSpace(tag)
	if trimmed == "" {
		return errors.New("empty tag name is not supported")
	}
	if unicode.IsUpper(rune(trimmed[0])) || strings.Contains(trimmed, ".") {
		return fmt.Errorf("JSX component tags are not supported in gwc import; rewrite %q as static HTML or a custom element", tag)
	}
	return nil
}

func importedBuilderName(tag string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(tag)) {
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
		return tag, false
	}
}

func renderImportedPropsLiteral(attrs []importedAttr, indent string) (string, error) {
	if len(attrs) == 0 {
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
	props := renderedProps{
		strings: map[string]string{},
		ints:    map[string]string{},
		bools:   map[string]bool{},
		style:   map[string]string{},
		data:    map[string]string{},
		aria:    map[string]string{},
		raw:     map[string]importedValue{},
	}
	for _, attr := range attrs {
		name := strings.TrimSpace(attr.Name)
		if name == "" || attr.Value.Kind == importedValueNull {
			continue
		}
		lower := strings.ToLower(name)
		switch lower {
		case "id":
			props.strings["ID"] = importedValueAsString(attr.Value)
		case "class", "classname":
			props.strings["Class"] = importedValueAsString(attr.Value)
		case "key":
			props.strings["Key"] = importedValueAsString(attr.Value)
		case "slot":
			props.strings["Slot"] = importedValueAsString(attr.Value)
		case "title":
			props.strings["Title"] = importedValueAsString(attr.Value)
		case "type":
			props.strings["Type"] = importedValueAsString(attr.Value)
		case "name":
			props.strings["Name"] = importedValueAsString(attr.Value)
		case "value":
			props.strings["Value"] = importedValueAsString(attr.Value)
		case "placeholder":
			props.strings["Placeholder"] = importedValueAsString(attr.Value)
		case "accept":
			props.strings["Accept"] = importedValueAsString(attr.Value)
		case "href":
			props.strings["Href"] = importedValueAsString(attr.Value)
		case "src":
			props.strings["Src"] = importedValueAsString(attr.Value)
		case "alt":
			props.strings["Alt"] = importedValueAsString(attr.Value)
		case "for", "htmlfor":
			props.strings["For"] = importedValueAsString(attr.Value)
		case "role":
			props.strings["Role"] = importedValueAsString(attr.Value)
		case "target":
			props.strings["Target"] = importedValueAsString(attr.Value)
		case "rel":
			props.strings["Rel"] = importedValueAsString(attr.Value)
		case "as":
			props.strings["As"] = importedValueAsString(attr.Value)
		case "action":
			props.strings["Action"] = importedValueAsString(attr.Value)
		case "method":
			props.strings["Method"] = importedValueAsString(attr.Value)
		case "enctype", "encType":
			props.strings["EncType"] = importedValueAsString(attr.Value)
		case "autocomplete", "autoComplete":
			props.strings["AutoComplete"] = importedValueAsString(attr.Value)
		case "min":
			props.strings["Min"] = importedValueAsString(attr.Value)
		case "max":
			props.strings["Max"] = importedValueAsString(attr.Value)
		case "step":
			props.strings["Step"] = importedValueAsString(attr.Value)
		case "rows":
			props.ints["Rows"] = importedValueAsNumber(attr.Value)
		case "cols":
			props.ints["Cols"] = importedValueAsNumber(attr.Value)
		case "checked":
			props.bools["Checked"] = importedValueAsBool(attr.Value)
		case "disabled":
			props.bools["Disabled"] = importedValueAsBool(attr.Value)
		case "selected":
			props.bools["Selected"] = importedValueAsBool(attr.Value)
		case "required":
			props.bools["Required"] = importedValueAsBool(attr.Value)
		case "readonly", "readOnly":
			props.bools["ReadOnly"] = importedValueAsBool(attr.Value)
		case "hidden":
			props.bools["Hidden"] = importedValueAsBool(attr.Value)
		case "multiple":
			props.bools["Multiple"] = importedValueAsBool(attr.Value)
		case "autofocus", "autoFocus":
			props.bools["AutoFocus"] = importedValueAsBool(attr.Value)
		case "style":
			style := importedValueAsStyleMap(attr.Value)
			for key, value := range style {
				props.style[key] = value
			}
		default:
			if strings.HasPrefix(lower, "data-") {
				props.data[strings.TrimPrefix(name, "data-")] = importedValueAsString(attr.Value)
				continue
			}
			if strings.HasPrefix(lower, "aria-") {
				props.aria[strings.TrimPrefix(name, "aria-")] = importedValueAsString(attr.Value)
				continue
			}
			props.raw[name] = attr.Value
		}
	}
	lines := []string{"html.Props{"}
	appendStringField := func(field string) {
		value, ok := props.strings[field]
		if ok && value != "" {
			lines = append(lines, indent+field+": "+strconv.Quote(value)+",")
		}
	}
	appendIntField := func(field string) {
		value, ok := props.ints[field]
		if ok && value != "" {
			lines = append(lines, indent+field+": "+value+",")
		}
	}
	appendBoolField := func(field string) {
		if props.bools[field] {
			lines = append(lines, indent+field+": true,")
		}
	}
	for _, field := range []string{"ID", "Class", "Key", "Slot", "Title", "Type", "Name", "Value", "Placeholder", "Accept", "Href", "Src", "Alt", "For", "Role", "Target", "Rel", "As", "Action", "Method", "EncType", "AutoComplete", "Min", "Max", "Step"} {
		appendStringField(field)
	}
	for _, field := range []string{"Rows", "Cols"} {
		appendIntField(field)
	}
	for _, field := range []string{"Checked", "Disabled", "Selected", "Required", "ReadOnly", "Hidden", "Multiple", "AutoFocus"} {
		appendBoolField(field)
	}
	appendRenderedStringMap := func(field string, values map[string]string) {
		if len(values) == 0 {
			return
		}
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines = append(lines, indent+field+": map[string]string{")
		for _, key := range keys {
			lines = append(lines, indent+"\t"+strconv.Quote(key)+": "+strconv.Quote(values[key])+",")
		}
		lines = append(lines, indent+"},")
	}
	appendRenderedStringMap("Style", props.style)
	appendRenderedStringMap("Data", props.data)
	appendRenderedStringMap("Aria", props.aria)
	if len(props.raw) > 0 {
		keys := make([]string, 0, len(props.raw))
		for key := range props.raw {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines = append(lines, indent+"Raw: map[string]interface{}{")
		for _, key := range keys {
			lines = append(lines, indent+"\t"+strconv.Quote(key)+": "+renderImportedInterfaceValue(props.raw[key])+",")
		}
		lines = append(lines, indent+"},")
	}
	if len(lines) == 1 {
		return "html.Props{}", nil
	}
	lines = append(lines, strings.TrimRight(indent, "\t")+"}")
	return strings.Join(lines, "\n"), nil
}

func importedValueAsString(value importedValue) string {
	switch value.Kind {
	case importedValueString:
		return value.String
	case importedValueBool:
		if value.Bool {
			return "true"
		}
		return "false"
	case importedValueNumber:
		return value.Number
	case importedValueStyle:
		keys := make([]string, 0, len(value.Style))
		for key := range value.Style {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+": "+value.Style[key])
		}
		return strings.Join(parts, "; ")
	default:
		return ""
	}
}

func importedValueAsNumber(value importedValue) string {
	switch value.Kind {
	case importedValueNumber:
		return value.Number
	case importedValueString:
		trimmed := strings.TrimSpace(value.String)
		if trimmed == "" {
			return ""
		}
		if _, err := strconv.Atoi(trimmed); err == nil {
			return trimmed
		}
	}
	return ""
}

func importedValueAsBool(value importedValue) bool {
	switch value.Kind {
	case importedValueBool:
		return value.Bool
	case importedValueString:
		if value.String == "" {
			return true
		}
		return strings.EqualFold(strings.TrimSpace(value.String), "true")
	default:
		return false
	}
}

func importedValueAsStyleMap(value importedValue) map[string]string {
	if value.Kind == importedValueStyle {
		return value.Style
	}
	style := map[string]string{}
	for key, val := range parseImportedStyleString(importedValueAsString(value)) {
		style[key] = val
	}
	return style
}

func parseImportedStyleString(raw string) map[string]string {
	style := map[string]string{}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments := strings.SplitN(part, ":", 2)
		if len(segments) != 2 {
			continue
		}
		key := strings.TrimSpace(segments[0])
		value := strings.TrimSpace(segments[1])
		if key == "" || value == "" {
			continue
		}
		style[key] = value
	}
	return style
}

func renderImportedInterfaceValue(value importedValue) string {
	switch value.Kind {
	case importedValueBool:
		if value.Bool {
			return "true"
		}
		return "false"
	case importedValueNumber:
		return value.Number
	default:
		return strconv.Quote(importedValueAsString(value))
	}
}

func renderImportedIndexHTML(selection startSelection, document importedDocument) (string, error) {
	title := strings.TrimSpace(document.Title)
	if title == "" {
		title = selection.ProjectName
	}
	lang := strings.TrimSpace(document.Lang)
	if lang == "" {
		lang = "en"
	}
	headLines := []string{
		"\t<meta charset=\"UTF-8\">",
		"\t<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">",
		"\t<title>" + stdhtml.EscapeString(title) + "</title>",
	}
	for _, node := range document.HeadNodes {
		rendered, err := renderImportedNodeAsHTML(node)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		for _, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
			headLines = append(headLines, "\t"+line)
		}
	}
	headLines = append(headLines, "\t<script src=\"./wasm_exec.js\"></script>")
	bodyAttrs := renderImportedHTMLAttrs(document.BodyAttrs)
	if bodyAttrs != "" {
		bodyAttrs = " " + bodyAttrs
	}
	return "<!DOCTYPE html>\n<html lang=\"" + stdhtml.EscapeString(lang) + "\">\n<head>\n" + strings.Join(headLines, "\n") + "\n</head>\n<body" + bodyAttrs + ">\n\t<div id=\"" + importedMountID + "\"></div>\n\t<div id=\"boot-error\" hidden></div>\n\t<script>\n\t\tconst go = new Go();\n\t\tconst errorBox = document.getElementById('boot-error');\n\t\tWebAssembly.instantiateStreaming(fetch('./main.wasm'), go.importObject)\n\t\t\t.then(result => go.run(result.instance))\n\t\t\t.catch(error => {\n\t\t\t\terrorBox.hidden = false;\n\t\t\t\terrorBox.textContent = 'Failed to start wasm app: ' + String(error);\n\t\t\t\tconsole.error(error);\n\t\t\t});\n\t</script>\n</body>\n</html>\n", nil
}

func renderImportedNodeAsHTML(node importedNode) (string, error) {
	switch node.Kind {
	case importedNodeText:
		return stdhtml.EscapeString(node.Text), nil
	case importedNodeFragment:
		parts := make([]string, 0, len(node.Children))
		for _, child := range node.Children {
			rendered, err := renderImportedNodeAsHTML(child)
			if err != nil {
				return "", err
			}
			parts = append(parts, rendered)
		}
		return strings.Join(parts, ""), nil
	case importedNodeElement:
		if err := validateImportedTagName(node.Tag); err != nil {
			return "", err
		}
		attrs := renderImportedHTMLAttrs(node.Attrs)
		if attrs != "" {
			attrs = " " + attrs
		}
		if len(node.Children) == 0 && isImportedVoidTag(node.Tag) {
			return "<" + node.Tag + attrs + ">", nil
		}
		parts := make([]string, 0, len(node.Children))
		for _, child := range node.Children {
			rendered, err := renderImportedNodeAsHTML(child)
			if err != nil {
				return "", err
			}
			parts = append(parts, rendered)
		}
		return "<" + node.Tag + attrs + ">" + strings.Join(parts, "") + "</" + node.Tag + ">", nil
	default:
		return "", fmt.Errorf("unsupported imported node kind %q", node.Kind)
	}
}

func renderImportedHTMLAttrs(attrs []importedAttr) string {
	parts := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		name := strings.TrimSpace(attr.Name)
		if name == "" || strings.EqualFold(name, "id") && importedValueAsString(attr.Value) == importedMountID {
			continue
		}
		value := attr.Value
		if value.Kind == importedValueNull {
			continue
		}
		if isImportedBooleanAttr(name) {
			if importedValueAsBool(value) {
				parts = append(parts, name)
			}
			continue
		}
		if strings.EqualFold(name, "style") {
			style := importedValueAsStyleMap(value)
			if len(style) == 0 {
				continue
			}
			keys := make([]string, 0, len(style))
			for key := range style {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			entries := make([]string, 0, len(keys))
			for _, key := range keys {
				entries = append(entries, key+": "+style[key])
			}
			parts = append(parts, name+"=\""+stdhtml.EscapeString(strings.Join(entries, "; "))+"\"")
			continue
		}
		parts = append(parts, name+"=\""+stdhtml.EscapeString(importedValueAsString(value))+"\"")
	}
	return strings.Join(parts, " ")
}

func isImportedVoidTag(tag string) bool {
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

func isImportedBooleanAttr(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "checked", "disabled", "selected", "required", "readonly", "hidden", "multiple", "autofocus", "controls", "muted", "playsinline", "loop":
		return true
	default:
		return false
	}
}

func (l launcher) generateScaffoldProject(plan scaffoldPlan) (scaffoldResult, error) {
	targetDir := filepath.Clean(plan.Selection.TargetDir)
	if err := validateGeneratedTargetDir(targetDir); err != nil {
		return scaffoldResult{}, err
	}
	if err := ensureEmptyDir(targetDir); err != nil {
		return scaffoldResult{}, err
	}

	repoModulePath, err := l.readRepoModulePath()
	if err != nil {
		return scaffoldResult{}, err
	}
	relRepoRoot, err := scaffoldRel(targetDir, l.repoRoot)
	if err != nil {
		return scaffoldResult{}, fmt.Errorf("resolve repo replace path: %w", err)
	}
	wasmExecSource, err := scaffoldResolveWasmExecPath()
	if err != nil {
		return scaffoldResult{}, err
	}

	mainPath := filepath.Join(targetDir, "main.go")
	htmlPath := filepath.Join(targetDir, "index.html")
	metadataPath := filepath.Join(targetDir, "gwc-start.json")
	readmePath := filepath.Join(targetDir, "README.md")
	wasmExecPath := filepath.Join(targetDir, "wasm_exec.js")
	goModPath := filepath.Join(targetDir, "go.mod")

	if err := scaffoldWriteFile(goModPath, []byte(renderScaffoldGoMod(plan.Selection, repoModulePath, relRepoRoot)), 0644); err != nil {
		return scaffoldResult{}, fmt.Errorf("write go.mod: %w", err)
	}
	if err := scaffoldWriteFile(mainPath, []byte(plan.MainGo), 0644); err != nil {
		return scaffoldResult{}, fmt.Errorf("write main.go: %w", err)
	}
	if err := scaffoldWriteFile(htmlPath, []byte(plan.HTML), 0644); err != nil {
		return scaffoldResult{}, fmt.Errorf("write index.html: %w", err)
	}
	metadataBytes, err := scaffoldMarshalIndent(plan.Metadata, "", "  ")
	if err != nil {
		return scaffoldResult{}, fmt.Errorf("encode gwc-start.json: %w", err)
	}
	metadataBytes = append(metadataBytes, '\n')
	if err := scaffoldWriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return scaffoldResult{}, fmt.Errorf("write gwc-start.json: %w", err)
	}
	if err := scaffoldWriteFile(readmePath, []byte(plan.README), 0644); err != nil {
		return scaffoldResult{}, fmt.Errorf("write README.md: %w", err)
	}
	for relativePath, contents := range plan.ExtraFiles {
		targetPath := filepath.Join(targetDir, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return scaffoldResult{}, fmt.Errorf("create scaffold extra file directory: %w", err)
		}
		if err := scaffoldWriteFile(targetPath, contents, 0644); err != nil {
			return scaffoldResult{}, fmt.Errorf("write scaffold extra file %s: %w", relativePath, err)
		}
	}
	wasmExecBytes, err := scaffoldReadFile(wasmExecSource)
	if err != nil {
		return scaffoldResult{}, fmt.Errorf("read wasm_exec.js: %w", err)
	}
	if err := scaffoldWriteFile(wasmExecPath, wasmExecBytes, 0644); err != nil {
		return scaffoldResult{}, fmt.Errorf("write wasm_exec.js: %w", err)
	}
	if err := l.seedScaffoldGoSum(targetDir); err != nil {
		return scaffoldResult{}, err
	}
	if err := l.tidyScaffoldModule(targetDir); err != nil {
		return scaffoldResult{}, err
	}
	if err := scaffoldFormatMain(mainPath); err != nil {
		return scaffoldResult{}, fmt.Errorf("format generated main.go: %w", err)
	}
	return scaffoldResult{TargetDir: targetDir, AppPath: mainPath, HTMLPath: htmlPath}, nil
}

func defaultScaffoldMetadata(selection startSelection) scaffoldMetadata {
	return scaffoldMetadata{
		ProjectName: selection.ProjectName,
		ModulePath:  selection.ModulePath,
		Author:      selection.Author,
		Version:     selection.Version,
		Description: selection.Description,
		TargetDir:   selection.TargetDir,
		Preset: scaffoldPresetMetadata{
			Key:         selection.Preset.Key,
			Name:        selection.Preset.Name,
			Summary:     selection.Preset.Summary,
			Description: selection.Preset.Description,
			Features:    selection.Preset.Features,
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

func printImportSummary(summary importSummary) {
	fmt.Println("GWC import")
	fmt.Printf("  source kind:    %s\n", summary.SourceKind)
	fmt.Printf("  source:         %s\n", summary.SourcePath)
	if strings.TrimSpace(summary.OutputPath) != "" {
		fmt.Printf("  output:         %s\n", summary.OutputPath)
	}
}

func createLauncherTempDir(rootPath string, prefix string) (string, error) {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		cwd, err := buildGetwd()
		if err != nil {
			return "", err
		}
		rootPath = cwd
	}
	tempRoot := filepath.Join(rootPath, "bin", "tmp")
	if err := os.MkdirAll(tempRoot, 0755); err != nil {
		return "", fmt.Errorf("create launcher temp root: %w", err)
	}
	path, err := launcherMkdirTemp(tempRoot, prefix)
	if err != nil {
		return "", fmt.Errorf("create launcher temp directory: %w", err)
	}
	return path, nil
}
