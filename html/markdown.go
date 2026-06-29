package html

import (
	"net/url"
	"path"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownClasses configures optional class names applied to rendered markdown nodes.
type MarkdownClasses struct {
	Heading1           string
	Heading2           string
	Heading3           string
	Heading4           string
	Heading5           string
	Heading6           string
	Paragraph          string
	Blockquote         string
	List               string
	OrderedList        string
	UnorderedList      string
	ListItem           string
	CodeBlockContainer string
	CodeBlockLabel     string
	CodeBlockPre       string
	InlineCode         string
	Strong             string
	Emphasis           string
	Link               string
	HorizontalRule     string
}

// MarkdownRenderOptions configures markdown rendering into ui.Node values.
type MarkdownRenderOptions struct {
	SourcePath     string
	ResolveHref    func(sourcePath, destination string) string
	Classes        MarkdownClasses
	CodeBlockLabel string
	LinkTarget     string
	LinkRel        string
	// AllowedURLSchemes overrides the scheme allowlist applied to link, image,
	// and autolink destinations. When nil the default allowlist is used
	// (http, https, mailto); relative paths, absolute paths, and fragments
	// carry no scheme and are always allowed. Any other scheme - notably
	// javascript:, data:, and vbscript: - is dropped so user-supplied markdown
	// cannot smuggle an executable URL into an href or src sink. Supply schemes
	// without the trailing colon (for example []string{"http", "https",
	// "mailto", "data"} to additionally permit inline data: images).
	AllowedURLSchemes []string
}

// defaultMarkdownSchemes is the scheme allowlist used when a caller does not
// override it. It deliberately excludes javascript:, data:, vbscript:, and
// every other active scheme.
var defaultMarkdownSchemes = map[string]bool{"http": true, "https": true, "mailto": true}

// RenderMarkdown parses markdown text and returns semantic ui.Node values.
func RenderMarkdown(parseMarkdown string, parseOptions ...MarkdownRenderOptions) []ui.Node {
	parseConfig := MarkdownRenderOptions{}
	if len(parseOptions) > 0 {
		parseConfig = parseOptions[0]
	}
	parseSource := []byte(parseMarkdown)
	parseRoot := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(parseSource))
	return renderMarkdownBlocks(parseRoot, parseSource, parseConfig)
}

// ResolveMarkdownHref resolves a markdown destination against the source document path.
func ResolveMarkdownHref(parseSourcePath, parseDestination string) string {
	parseDestination = strings.TrimSpace(parseDestination)
	if parseDestination == "" {
		return ""
	}
	parseParsedDestination, parseErr := url.Parse(parseDestination)
	if parseErr != nil {
		return parseDestination
	}
	if parseParsedDestination.IsAbs() || strings.HasPrefix(parseDestination, "#") || strings.HasPrefix(parseDestination, "/") {
		return parseDestination
	}
	parseBase := strings.TrimSpace(parseSourcePath)
	if parseBase == "" {
		return parseDestination
	}
	parseParsedBase, parseErr := url.Parse(parseBase)
	if parseErr == nil && (parseParsedBase.IsAbs() || strings.HasPrefix(parseBase, "/")) {
		return parseParsedBase.ResolveReference(parseParsedDestination).String()
	}
	parseBasePath := strings.ReplaceAll(parseBase, "\\", "/")
	parseResolved := *parseParsedDestination
	parseResolved.Path = path.Clean(path.Join(path.Dir(parseBasePath), parseParsedDestination.Path))
	if strings.HasPrefix(parseBasePath, "/") && !strings.HasPrefix(parseResolved.Path, "/") {
		parseResolved.Path = "/" + parseResolved.Path
	}
	return parseResolved.String()
}

// renderMarkdownBlocks is a core package helper.
func renderMarkdownBlocks(parseParent ast.Node, parseSource []byte, parseConfig MarkdownRenderOptions) []ui.Node {
	parseNodes := make([]ui.Node, 0)
	for parseChild := parseParent.FirstChild(); parseChild != nil; parseChild = parseChild.NextSibling() {
		if parseRendered, parseOk := renderMarkdownBlock(parseChild, parseSource, parseConfig); parseOk {
			parseNodes = append(parseNodes, parseRendered)
		}
	}
	return parseNodes
}

// renderMarkdownBlock is a core package helper.
func renderMarkdownBlock(parseNode ast.Node, parseSource []byte, parseConfig MarkdownRenderOptions) (ui.Node, bool) {
	parseClasses := parseConfig.Classes
	switch parseTyped := parseNode.(type) {
	case *ast.Heading:
		parseChildren := renderMarkdownInlines(parseTyped, parseSource, parseConfig)
		switch parseTyped.Level {
		case 1:
			return H1(propsWithClass(parseClasses.Heading1), parseChildren...), true
		case 2:
			return H2(propsWithClass(parseClasses.Heading2), parseChildren...), true
		case 3:
			return H3(propsWithClass(parseClasses.Heading3), parseChildren...), true
		case 4:
			return H4(propsWithClass(parseClasses.Heading4), parseChildren...), true
		case 5:
			return H5(propsWithClass(parseClasses.Heading5), parseChildren...), true
		default:
			return H6(propsWithClass(parseClasses.Heading6), parseChildren...), true
		}
	case *ast.Paragraph:
		return P(propsWithClass(parseClasses.Paragraph), renderMarkdownInlines(parseTyped, parseSource, parseConfig)...), true
	case *ast.TextBlock:
		return P(propsWithClass(parseClasses.Paragraph), renderMarkdownInlines(parseTyped, parseSource, parseConfig)...), true
	case *ast.Blockquote:
		return Blockquote(propsWithClass(parseClasses.Blockquote), renderMarkdownBlocks(parseTyped, parseSource, parseConfig)...), true
	case *ast.List:
		parseItems := renderMarkdownBlocks(parseTyped, parseSource, parseConfig)
		parseClassName := strings.TrimSpace(parseClasses.List)
		if parseTyped.IsOrdered() {
			parseClassName = joinMarkdownClasses(parseClassName, parseClasses.OrderedList)
			parseProps := propsWithClass(parseClassName)
			if parseTyped.Start != 1 {
				parseProps.Raw = map[string]any{"start": parseTyped.Start}
			}
			return Tag("ol", parseProps, parseItems...), true
		}
		parseClassName = joinMarkdownClasses(parseClassName, parseClasses.UnorderedList)
		return Ul(propsWithClass(parseClassName), parseItems...), true
	case *ast.ListItem:
		parseItems := renderMarkdownBlocks(parseTyped, parseSource, parseConfig)
		if !markdownNodeContainsTaskCheckBox(parseTyped) {
			if isParseChecked, isParseTask := markdownTaskListItemState(parseTyped, parseSource); isParseTask {
				parseItems = append([]ui.Node{renderMarkdownTaskCheckBox(&extast.TaskCheckBox{IsChecked: isParseChecked})}, parseItems...)
			}
		}
		return Li(propsWithClass(parseClasses.ListItem), parseItems...), true
	case *ast.FencedCodeBlock:
		return renderMarkdownCodeBlock(strings.TrimRight(markdownLinesText(parseTyped.Lines(), parseSource), "\n"), parseConfig), true
	case *ast.CodeBlock:
		return renderMarkdownCodeBlock(strings.TrimRight(markdownLinesText(parseTyped.Lines(), parseSource), "\n"), parseConfig), true
	case *ast.ThematicBreak:
		return Hr(propsWithClass(parseClasses.HorizontalRule)), true
	case *ast.HTMLBlock:
		parseRaw := strings.TrimSpace(markdownRawHTMLBlockText(parseTyped, parseSource))
		if parseRaw == "" {
			return nil, false
		}
		return P(propsWithClass(parseClasses.Paragraph), Text(parseRaw)), true
	case *extast.TaskCheckBox:
		return renderMarkdownTaskCheckBox(parseTyped), true
	case *extast.Table:
		return renderMarkdownTable(parseTyped, parseSource, parseConfig), true
	default:
		parseTextValue := strings.TrimSpace(markdownPlainText(parseNode, parseSource))
		if parseTextValue == "" {
			return nil, false
		}
		return P(propsWithClass(parseClasses.Paragraph), Text(parseTextValue)), true
	}
}

// renderMarkdownTable converts a GFM table into thead/tbody markup.
func renderMarkdownTable(parseTable *extast.Table, parseSource []byte, parseConfig MarkdownRenderOptions) ui.Node {
	var parseHeadRows []ui.Node
	var parseBodyRows []ui.Node
	for parseChild := parseTable.FirstChild(); parseChild != nil; parseChild = parseChild.NextSibling() {
		switch parseRow := parseChild.(type) {
		case *extast.TableHeader:
			parseHeadRows = append(parseHeadRows, renderMarkdownTableRow(parseRow, parseSource, parseConfig, true))
		case *extast.TableRow:
			parseBodyRows = append(parseBodyRows, renderMarkdownTableRow(parseRow, parseSource, parseConfig, false))
		}
	}
	parseSections := make([]ui.Node, 0, 2)
	if len(parseHeadRows) > 0 {
		parseSections = append(parseSections, Tag("thead", Props{}, parseHeadRows...))
	}
	if len(parseBodyRows) > 0 {
		parseSections = append(parseSections, Tag("tbody", Props{}, parseBodyRows...))
	}
	return Tag("table", Props{}, parseSections...)
}

// renderMarkdownTableRow converts one table row's cells into th/td nodes.
func renderMarkdownTableRow(parseRow ast.Node, parseSource []byte, parseConfig MarkdownRenderOptions, isParseHeader bool) ui.Node {
	parseCellTag := "td"
	if isParseHeader {
		parseCellTag = "th"
	}
	var parseCells []ui.Node
	for parseCell := parseRow.FirstChild(); parseCell != nil; parseCell = parseCell.NextSibling() {
		parseCells = append(parseCells, Tag(parseCellTag, Props{}, renderMarkdownInlines(parseCell, parseSource, parseConfig)...))
	}
	return Tag("tr", Props{}, parseCells...)
}

// renderMarkdownCodeBlock is a core package helper.
func renderMarkdownCodeBlock(parseCode string, parseConfig MarkdownRenderOptions) ui.Node {
	parseChildren := make([]ui.Node, 0, 2)
	if strings.TrimSpace(parseConfig.CodeBlockLabel) != "" {
		parseChildren = append(parseChildren, Div(propsWithClass(parseConfig.Classes.CodeBlockLabel), Text(parseConfig.CodeBlockLabel)))
	}
	parseChildren = append(parseChildren, Pre(propsWithClass(parseConfig.Classes.CodeBlockPre), Code(Props{}, Text(parseCode))))
	return Div(propsWithClass(parseConfig.Classes.CodeBlockContainer), parseChildren...)
}

// renderMarkdownInlines is a core package helper.
func renderMarkdownInlines(parseParent ast.Node, parseSource []byte, parseConfig MarkdownRenderOptions) []ui.Node {
	parseNodes := make([]ui.Node, 0)
	for parseChild := parseParent.FirstChild(); parseChild != nil; parseChild = parseChild.NextSibling() {
		parseNodes = append(parseNodes, renderMarkdownInline(parseChild, parseSource, parseConfig)...)
	}
	return parseNodes
}

// renderMarkdownInline is a core package helper.
func renderMarkdownInline(parseNode ast.Node, parseSource []byte, parseConfig MarkdownRenderOptions) []ui.Node {
	parseClasses := parseConfig.Classes
	switch parseTyped := parseNode.(type) {
	case *ast.Text:
		parseTextValue := string(parseTyped.Segment.Value(parseSource))
		parseNodes := []ui.Node{Text(parseTextValue)}
		if parseTyped.HardLineBreak() || parseTyped.SoftLineBreak() {
			parseNodes = append(parseNodes, Br(Props{}))
		}
		return parseNodes
	case *ast.CodeSpan:
		return []ui.Node{Code(propsWithClass(parseClasses.InlineCode), Text(markdownPlainText(parseTyped, parseSource)))}
	case *ast.Emphasis:
		parseChildren := renderMarkdownInlines(parseTyped, parseSource, parseConfig)
		if parseTyped.Level == 2 {
			return []ui.Node{Tag("strong", propsWithClass(parseClasses.Strong), parseChildren...)}
		}
		return []ui.Node{Em(propsWithClass(parseClasses.Emphasis), parseChildren...)}
	case *ast.Link:
		parseChildren2 := renderMarkdownInlines(parseTyped, parseSource, parseConfig)
		parseProps := propsWithClass(parseClasses.Link)
		parseProps.Href = safeMarkdownHref(parseConfig, string(parseTyped.Destination))
		parseProps.Target = parseConfig.LinkTarget
		parseProps.Rel = parseConfig.LinkRel
		return []ui.Node{A(parseProps, parseChildren2...)}
	case *ast.Image:
		parseProps3 := propsWithClass(parseClasses.Link)
		parseProps3.Src = safeMarkdownHref(parseConfig, string(parseTyped.Destination))
		parseProps3.Alt = markdownPlainText(parseTyped, parseSource)
		return []ui.Node{Img(parseProps3)}
	case *ast.AutoLink:
		parseHref := string(parseTyped.URL(parseSource))
		parseProps2 := propsWithClass(parseClasses.Link)
		parseProps2.Href = SanitizeMarkdownHref(parseHref, parseConfig.AllowedURLSchemes)
		parseProps2.Target = parseConfig.LinkTarget
		parseProps2.Rel = parseConfig.LinkRel
		return []ui.Node{A(parseProps2, Text(parseHref))}
	case *ast.RawHTML:
		parseRaw := strings.TrimSpace(string(parseTyped.Segments.Value(parseSource)))
		if parseRaw == "" {
			return nil
		}
		return []ui.Node{Text(parseRaw)}
	case *extast.Strikethrough:
		return []ui.Node{Del(Props{}, renderMarkdownInlines(parseTyped, parseSource, parseConfig)...)}
	case *extast.TaskCheckBox:
		return []ui.Node{renderMarkdownTaskCheckBox(parseTyped)}
	default:
		parseTextValue2 := markdownPlainText(parseTyped, parseSource)
		if parseTextValue2 == "" {
			return nil
		}
		return []ui.Node{Text(parseTextValue2)}
	}
}

func renderMarkdownTaskCheckBox(parseNode *extast.TaskCheckBox) ui.Node {
	parseProps := Props{Type: "checkbox", Disabled: true}
	if parseNode != nil && parseNode.IsChecked {
		parseProps.Checked = true
	}
	return Input(parseProps)
}

func markdownNodeContainsTaskCheckBox(parseNode ast.Node) bool {
	if parseNode == nil {
		return false
	}
	for parseChild := parseNode.FirstChild(); parseChild != nil; parseChild = parseChild.NextSibling() {
		if _, parseOk := parseChild.(*extast.TaskCheckBox); parseOk {
			return true
		}
		if markdownNodeContainsTaskCheckBox(parseChild) {
			return true
		}
	}
	return false
}

func markdownTaskListItemState(parseItem *ast.ListItem, parseSource []byte) (bool, bool) {
	if parseItem == nil || parseItem.FirstChild() == nil {
		return false, false
	}
	parseLines := parseItem.FirstChild().Lines()
	if parseLines == nil || parseLines.Len() == 0 {
		return false, false
	}
	parseStart := parseLines.At(0).Start
	if parseStart > len(parseSource) {
		return false, false
	}
	parseLineStart := parseStart
	for parseLineStart > 0 && parseSource[parseLineStart-1] != '\n' && parseSource[parseLineStart-1] != '\r' {
		parseLineStart--
	}
	parsePrefix := string(parseSource[parseLineStart:parseStart])
	parseMarkerStart := strings.LastIndex(parsePrefix, "[")
	if parseMarkerStart < 0 {
		return false, false
	}
	switch strings.TrimSpace(parsePrefix[parseMarkerStart:]) {
	case "[x]", "[X]":
		return true, true
	case "[ ]":
		return false, true
	default:
		return false, false
	}
}

func markdownRawHTMLBlockText(parseBlock *ast.HTMLBlock, parseSource []byte) string {
	if parseBlock == nil {
		return ""
	}
	parseValue := markdownLinesText(parseBlock.Lines(), parseSource)
	if parseBlock.HasClosure() {
		parseValue += string(parseBlock.ClosureLine.Value(parseSource))
	}
	return parseValue
}

// markdownLinesText is a core package helper.
func markdownLinesText(parseLines *text.Segments, parseSource []byte) string {
	if parseLines == nil {
		return ""
	}
	var parseBuilder strings.Builder
	for parseIndex := 0; parseIndex < parseLines.Len(); parseIndex++ {
		parseSegment := parseLines.At(parseIndex)
		parseBuilder.Write(parseSegment.Value(parseSource))
	}
	return parseBuilder.String()
}

// markdownPlainText is a core package helper.
func markdownPlainText(parseNode ast.Node, parseSource []byte) string {
	var parseBuilder strings.Builder
	var parseWalk func(ast.Node)
	parseWalk = func(parseCurrent ast.Node) {
		if parseCurrent == nil {
			return
		}
		switch parseTyped := parseCurrent.(type) {
		case *ast.Text:
			parseBuilder.Write(parseTyped.Segment.Value(parseSource))
			if parseTyped.HardLineBreak() || parseTyped.SoftLineBreak() {
				parseBuilder.WriteString("\n")
			}
		case *ast.FencedCodeBlock:
			parseBuilder.WriteString(markdownLinesText(parseTyped.Lines(), parseSource))
		case *ast.CodeBlock:
			parseBuilder.WriteString(markdownLinesText(parseTyped.Lines(), parseSource))
		default:
			for parseChild := parseCurrent.FirstChild(); parseChild != nil; parseChild = parseChild.NextSibling() {
				parseWalk(parseChild)
			}
		}
	}
	parseWalk(parseNode)
	return strings.TrimSpace(parseBuilder.String())
}

// resolveMarkdownHref is a core package helper.
func resolveMarkdownHref(parseConfig MarkdownRenderOptions, parseDestination string) string {
	if parseConfig.ResolveHref != nil {
		return parseConfig.ResolveHref(parseConfig.SourcePath, parseDestination)
	}
	return ResolveMarkdownHref(parseConfig.SourcePath, parseDestination)
}

// safeMarkdownHref resolves a destination and then drops it if its URL scheme is
// not allowed, so dangerous schemes never reach an href or src attribute.
func safeMarkdownHref(parseConfig MarkdownRenderOptions, parseDestination string) string {
	return SanitizeMarkdownHref(resolveMarkdownHref(parseConfig, parseDestination), parseConfig.AllowedURLSchemes)
}

// SanitizeMarkdownHref returns destination unchanged when it carries no scheme
// (relative, absolute-path, or fragment) or an allowed scheme, and returns ""
// otherwise. The allowed argument lists schemes without the trailing colon; a
// nil or empty list uses the default allowlist (http, https, mailto). Scheme
// detection tolerates the usual obfuscations - leading/embedded whitespace and
// control characters (java\tscript:), mixed case (JavaScript:), and a leading
// space ( javascript:) - by normalizing the scheme token before the check.
func SanitizeMarkdownHref(parseDestination string, parseAllowed []string) string {
	parseTrimmed := strings.TrimSpace(parseDestination)
	if parseTrimmed == "" {
		return ""
	}
	parseScheme, parseHasScheme := markdownURLScheme(parseTrimmed)
	if !parseHasScheme {
		return parseDestination
	}
	if markdownAllowedSchemes(parseAllowed)[normalizeMarkdownScheme(parseScheme)] {
		return parseDestination
	}
	return ""
}

// markdownURLScheme returns the scheme token of a destination (the text before
// the first ':' that precedes any '/', '?', '#', or '\\'). It reports
// hasScheme=false for relative paths, absolute paths, and fragments, which have
// no scheme separator.
func markdownURLScheme(parseDestination string) (parseScheme string, parseHasScheme bool) {
	for parseIndex := 0; parseIndex < len(parseDestination); parseIndex++ {
		switch parseDestination[parseIndex] {
		case ':':
			return parseDestination[:parseIndex], true
		case '/', '?', '#', '\\':
			return "", false
		}
	}
	return "", false
}

// normalizeMarkdownScheme lowercases a scheme token and strips whitespace and
// control characters so obfuscated schemes collapse onto their real form.
func normalizeMarkdownScheme(parseScheme string) string {
	var parseBuilder strings.Builder
	for parseIndex := 0; parseIndex < len(parseScheme); parseIndex++ {
		parseByte := parseScheme[parseIndex]
		if parseByte <= ' ' || parseByte == 0x7f {
			continue
		}
		parseBuilder.WriteByte(parseByte)
	}
	return strings.ToLower(parseBuilder.String())
}

// markdownAllowedSchemes returns the effective scheme allowlist, defaulting to
// http/https/mailto when no override is supplied.
func markdownAllowedSchemes(parseAllowed []string) map[string]bool {
	if len(parseAllowed) == 0 {
		return defaultMarkdownSchemes
	}
	parseSet := make(map[string]bool, len(parseAllowed))
	for _, parseScheme := range parseAllowed {
		parseSet[normalizeMarkdownScheme(parseScheme)] = true
	}
	return parseSet
}

// propsWithClass is a core package helper.
func propsWithClass(parseClassName string) Props {
	parseClassName = strings.TrimSpace(parseClassName)
	if parseClassName == "" {
		return Props{}
	}
	return Props{Class: parseClassName}
}

// joinMarkdownClasses is a core package helper.
func joinMarkdownClasses(parseValues ...string) string {
	parseParts := make([]string, 0, len(parseValues))
	for _, parseValue := range parseValues {
		parseValue = strings.TrimSpace(parseValue)
		if parseValue != "" {
			parseParts = append(parseParts, parseValue)
		}
	}
	return strings.Join(parseParts, " ")
}
