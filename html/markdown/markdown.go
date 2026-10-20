// Package markdown is the goldmark-backed markdown engine for html.Markdown and
// html.RenderMarkdown. It is a separate package so apps that never render markdown
// do not pay for goldmark, regexp and the unicode tables it drags in. Import it once:
//
//	import _ "github.com/monstercameron/GoWebComponents/v7/html/markdown"
//
// or call [Enable] for readers who dislike blank imports.
package markdown

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v7/html"
	"github.com/monstercameron/GoWebComponents/v7/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

func init() {
	Enable()
}

// Enable registers the goldmark engine with package html. It is idempotent and
// exists so an app can opt in without a blank import; importing this package
// already calls it.
func Enable() {
	html.RegisterMarkdownEngine(renderMarkdownConfig)
}

// renderMarkdownConfig renders markdown text with a resolved configuration.
func renderMarkdownConfig(parseMarkdown string, parseConfig html.MarkdownRenderOptions) []ui.Node {
	parseSource := []byte(parseMarkdown)
	parseRoot := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(parseSource))
	return renderMarkdownBlocks(parseRoot, parseSource, parseConfig, 0)
}

// maxMarkdownRenderDepth bounds the recursive AST walk. goldmark does not cap
// blockquote/list nesting, so adversarial input like "> " repeated tens of
// thousands of times yields an AST deep enough to overflow the render stack.
// The cap sits far above any real document (deeply nested lists rarely exceed
// a few dozen levels) while stopping pathological input from crashing the process.
const maxMarkdownRenderDepth = 512

// renderMarkdownBlocks is a core package helper.
func renderMarkdownBlocks(parseParent ast.Node, parseSource []byte, parseConfig html.MarkdownRenderOptions, parseDepth int) []ui.Node {
	if parseDepth > maxMarkdownRenderDepth {
		// Adversarially deep nesting: truncate with a visible marker rather than
		// recursing until the stack overflows.
		return []ui.Node{html.P(propsWithClass(parseConfig.Classes.Paragraph), html.Text("[markdown nesting too deep — truncated]"))}
	}
	parseNodes := make([]ui.Node, 0)
	for parseChild := parseParent.FirstChild(); parseChild != nil; parseChild = parseChild.NextSibling() {
		if parseRendered, parseOk := renderMarkdownBlock(parseChild, parseSource, parseConfig, parseDepth); parseOk {
			parseNodes = append(parseNodes, parseRendered)
		}
	}
	return parseNodes
}

// renderMarkdownBlock is a core package helper.
func renderMarkdownBlock(parseNode ast.Node, parseSource []byte, parseConfig html.MarkdownRenderOptions, parseDepth int) (ui.Node, bool) {
	parseClasses := parseConfig.Classes
	switch parseTyped := parseNode.(type) {
	case *ast.Heading:
		parseChildren := renderMarkdownInlines(parseTyped, parseSource, parseConfig, parseDepth)
		switch parseTyped.Level {
		case 1:
			return html.H1(propsWithClass(parseClasses.Heading1), parseChildren...), true
		case 2:
			return html.H2(propsWithClass(parseClasses.Heading2), parseChildren...), true
		case 3:
			return html.H3(propsWithClass(parseClasses.Heading3), parseChildren...), true
		case 4:
			return html.H4(propsWithClass(parseClasses.Heading4), parseChildren...), true
		case 5:
			return html.H5(propsWithClass(parseClasses.Heading5), parseChildren...), true
		default:
			return html.H6(propsWithClass(parseClasses.Heading6), parseChildren...), true
		}
	case *ast.Paragraph:
		return html.P(propsWithClass(parseClasses.Paragraph), renderMarkdownInlines(parseTyped, parseSource, parseConfig, parseDepth)...), true
	case *ast.TextBlock:
		return html.P(propsWithClass(parseClasses.Paragraph), renderMarkdownInlines(parseTyped, parseSource, parseConfig, parseDepth)...), true
	case *ast.Blockquote:
		return html.Blockquote(propsWithClass(parseClasses.Blockquote), renderMarkdownBlocks(parseTyped, parseSource, parseConfig, parseDepth+1)...), true
	case *ast.List:
		parseItems := renderMarkdownBlocks(parseTyped, parseSource, parseConfig, parseDepth+1)
		parseClassName := strings.TrimSpace(parseClasses.List)
		if parseTyped.IsOrdered() {
			parseClassName = joinMarkdownClasses(parseClassName, parseClasses.OrderedList)
			parseProps := propsWithClass(parseClassName)
			if parseTyped.Start != 1 {
				parseProps.Raw = map[string]any{"start": parseTyped.Start}
			}
			return html.Tag("ol", parseProps, parseItems...), true
		}
		parseClassName = joinMarkdownClasses(parseClassName, parseClasses.UnorderedList)
		return html.Ul(propsWithClass(parseClassName), parseItems...), true
	case *ast.ListItem:
		parseItems := renderMarkdownBlocks(parseTyped, parseSource, parseConfig, parseDepth+1)
		if !markdownNodeContainsTaskCheckBox(parseTyped) {
			if isParseChecked, isParseTask := markdownTaskListItemState(parseTyped, parseSource); isParseTask {
				parseItems = append([]ui.Node{renderMarkdownTaskCheckBox(&extast.TaskCheckBox{IsChecked: isParseChecked})}, parseItems...)
			}
		}
		return html.Li(propsWithClass(parseClasses.ListItem), parseItems...), true
	case *ast.FencedCodeBlock:
		return renderMarkdownCodeBlock(strings.TrimRight(markdownLinesText(parseTyped.Lines(), parseSource), "\n"), parseConfig), true
	case *ast.CodeBlock:
		return renderMarkdownCodeBlock(strings.TrimRight(markdownLinesText(parseTyped.Lines(), parseSource), "\n"), parseConfig), true
	case *ast.ThematicBreak:
		return html.Hr(propsWithClass(parseClasses.HorizontalRule)), true
	case *ast.HTMLBlock:
		parseRaw := strings.TrimSpace(markdownRawHTMLBlockText(parseTyped, parseSource))
		if parseRaw == "" {
			return nil, false
		}
		return html.P(propsWithClass(parseClasses.Paragraph), html.Text(parseRaw)), true
	case *extast.TaskCheckBox:
		return renderMarkdownTaskCheckBox(parseTyped), true
	case *extast.Table:
		return renderMarkdownTable(parseTyped, parseSource, parseConfig, parseDepth+1), true
	default:
		parseTextValue := strings.TrimSpace(markdownPlainText(parseNode, parseSource))
		if parseTextValue == "" {
			return nil, false
		}
		return html.P(propsWithClass(parseClasses.Paragraph), html.Text(parseTextValue)), true
	}
}

// renderMarkdownTable converts a GFM table into thead/tbody markup.
func renderMarkdownTable(parseTable *extast.Table, parseSource []byte, parseConfig html.MarkdownRenderOptions, parseDepth int) ui.Node {
	var parseHeadRows []ui.Node
	var parseBodyRows []ui.Node
	for parseChild := parseTable.FirstChild(); parseChild != nil; parseChild = parseChild.NextSibling() {
		switch parseRow := parseChild.(type) {
		case *extast.TableHeader:
			parseHeadRows = append(parseHeadRows, renderMarkdownTableRow(parseRow, parseSource, parseConfig, true, parseDepth))
		case *extast.TableRow:
			parseBodyRows = append(parseBodyRows, renderMarkdownTableRow(parseRow, parseSource, parseConfig, false, parseDepth))
		}
	}
	parseSections := make([]ui.Node, 0, 2)
	if len(parseHeadRows) > 0 {
		parseSections = append(parseSections, html.Tag("thead", html.Props{}, parseHeadRows...))
	}
	if len(parseBodyRows) > 0 {
		parseSections = append(parseSections, html.Tag("tbody", html.Props{}, parseBodyRows...))
	}
	return html.Tag("table", html.Props{}, parseSections...)
}

// renderMarkdownTableRow converts one table row's cells into th/td nodes.
func renderMarkdownTableRow(parseRow ast.Node, parseSource []byte, parseConfig html.MarkdownRenderOptions, isParseHeader bool, parseDepth int) ui.Node {
	parseCellTag := "td"
	if isParseHeader {
		parseCellTag = "th"
	}
	var parseCells []ui.Node
	for parseCell := parseRow.FirstChild(); parseCell != nil; parseCell = parseCell.NextSibling() {
		parseCells = append(parseCells, html.Tag(parseCellTag, html.Props{}, renderMarkdownInlines(parseCell, parseSource, parseConfig, parseDepth)...))
	}
	return html.Tag("tr", html.Props{}, parseCells...)
}

// renderMarkdownCodeBlock is a core package helper.
func renderMarkdownCodeBlock(parseCode string, parseConfig html.MarkdownRenderOptions) ui.Node {
	parseChildren := make([]ui.Node, 0, 2)
	if strings.TrimSpace(parseConfig.CodeBlockLabel) != "" {
		parseChildren = append(parseChildren, html.Div(propsWithClass(parseConfig.Classes.CodeBlockLabel), html.Text(parseConfig.CodeBlockLabel)))
	}
	parseChildren = append(parseChildren, html.Pre(propsWithClass(parseConfig.Classes.CodeBlockPre), html.Code(html.Props{}, html.Text(parseCode))))
	return html.Div(propsWithClass(parseConfig.Classes.CodeBlockContainer), parseChildren...)
}

// renderMarkdownInlines is a core package helper.
func renderMarkdownInlines(parseParent ast.Node, parseSource []byte, parseConfig html.MarkdownRenderOptions, parseDepth int) []ui.Node {
	if parseDepth > maxMarkdownRenderDepth {
		// Adversarially deep inline nesting (e.g. runaway emphasis/link nesting):
		// stop recursing before the stack overflows.
		return nil
	}
	parseNodes := make([]ui.Node, 0)
	for parseChild := parseParent.FirstChild(); parseChild != nil; parseChild = parseChild.NextSibling() {
		parseNodes = append(parseNodes, renderMarkdownInline(parseChild, parseSource, parseConfig, parseDepth)...)
	}
	return parseNodes
}

// renderMarkdownInline is a core package helper.
func renderMarkdownInline(parseNode ast.Node, parseSource []byte, parseConfig html.MarkdownRenderOptions, parseDepth int) []ui.Node {
	parseClasses := parseConfig.Classes
	switch parseTyped := parseNode.(type) {
	case *ast.Text:
		parseTextValue := string(parseTyped.Segment.Value(parseSource))
		parseNodes := []ui.Node{html.Text(parseTextValue)}
		if parseTyped.HardLineBreak() || parseTyped.SoftLineBreak() {
			parseNodes = append(parseNodes, html.Br(html.Props{}))
		}
		return parseNodes
	case *ast.CodeSpan:
		return []ui.Node{html.Code(propsWithClass(parseClasses.InlineCode), html.Text(markdownPlainText(parseTyped, parseSource)))}
	case *ast.Emphasis:
		parseChildren := renderMarkdownInlines(parseTyped, parseSource, parseConfig, parseDepth+1)
		if parseTyped.Level == 2 {
			return []ui.Node{html.Tag("strong", propsWithClass(parseClasses.Strong), parseChildren...)}
		}
		return []ui.Node{html.Em(propsWithClass(parseClasses.Emphasis), parseChildren...)}
	case *ast.Link:
		parseChildren2 := renderMarkdownInlines(parseTyped, parseSource, parseConfig, parseDepth+1)
		parseProps := propsWithClass(parseClasses.Link)
		parseProps.Href = safeMarkdownHref(parseConfig, string(parseTyped.Destination))
		parseProps.Target = parseConfig.LinkTarget
		parseProps.Rel = parseConfig.LinkRel
		return []ui.Node{html.A(parseProps, parseChildren2...)}
	case *ast.Image:
		parseProps3 := propsWithClass(parseClasses.Link)
		parseProps3.Src = safeMarkdownHref(parseConfig, string(parseTyped.Destination))
		parseProps3.Alt = markdownPlainText(parseTyped, parseSource)
		return []ui.Node{html.Img(parseProps3)}
	case *ast.AutoLink:
		parseHref := string(parseTyped.URL(parseSource))
		if parseTyped.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(parseHref), "mailto:") {
			// An email autolink carries a bare address; without the scheme the
			// browser would resolve it as a relative path.
			parseHref = "mailto:" + parseHref
		}
		parseProps2 := propsWithClass(parseClasses.Link)
		parseProps2.Href = html.SanitizeMarkdownHref(parseHref, parseConfig.GetAllowedSchemes())
		parseProps2.Target = parseConfig.LinkTarget
		parseProps2.Rel = parseConfig.LinkRel
		return []ui.Node{html.A(parseProps2, html.Text(string(parseTyped.Label(parseSource))))}
	case *ast.RawHTML:
		parseRaw := strings.TrimSpace(string(parseTyped.Segments.Value(parseSource)))
		if parseRaw == "" {
			return nil
		}
		return []ui.Node{html.Text(parseRaw)}
	case *extast.Strikethrough:
		return []ui.Node{html.Del(html.Props{}, renderMarkdownInlines(parseTyped, parseSource, parseConfig, parseDepth+1)...)}
	case *extast.TaskCheckBox:
		return []ui.Node{renderMarkdownTaskCheckBox(parseTyped)}
	default:
		parseTextValue2 := markdownPlainText(parseTyped, parseSource)
		if parseTextValue2 == "" {
			return nil
		}
		return []ui.Node{html.Text(parseTextValue2)}
	}
}

func renderMarkdownTaskCheckBox(parseNode *extast.TaskCheckBox) ui.Node {
	parseProps := html.Props{Type: "checkbox", Disabled: true}
	if parseNode != nil && parseNode.IsChecked {
		parseProps.Checked = true
	}
	return html.Input(parseProps)
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
	_, parseMarkerTail, isMarkerFound := strings.CutLast(parsePrefix, "[")
	if !isMarkerFound {
		return false, false
	}
	switch strings.TrimSpace("[" + parseMarkerTail) {
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

// safeMarkdownHref resolves a destination and then drops it if its URL scheme is
// not allowed, so dangerous schemes never reach an href or src attribute.
func safeMarkdownHref(parseConfig html.MarkdownRenderOptions, parseDestination string) string {
	return parseConfig.ResolveSafeHref(parseDestination)
}

// propsWithClass is a core package helper.
func propsWithClass(parseClassName string) html.Props {
	parseClassName = strings.TrimSpace(parseClassName)
	if parseClassName == "" {
		return html.Props{}
	}
	return html.Props{Class: parseClassName}
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
