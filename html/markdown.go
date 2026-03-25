package html

import (
	"net/url"
	"path"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
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
}

// RenderMarkdown parses markdown text and returns semantic ui.Node values.
func RenderMarkdown(parseMarkdown string, parseOptions ...MarkdownRenderOptions) []ui.Node {
	parseConfig := MarkdownRenderOptions{}
	if len(parseOptions) > 0 {
		parseConfig = parseOptions[0]
	}
	parseSource := []byte(parseMarkdown)
	parseRoot := goldmark.New().Parser().Parse(text.NewReader(parseSource))
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
	case *ast.Blockquote:
		return Blockquote(propsWithClass(parseClasses.Blockquote), renderMarkdownBlocks(parseTyped, parseSource, parseConfig)...), true
	case *ast.List:
		parseItems := renderMarkdownBlocks(parseTyped, parseSource, parseConfig)
		parseClassName := strings.TrimSpace(parseClasses.List)
		if parseTyped.IsOrdered() {
			parseClassName = joinMarkdownClasses(parseClassName, parseClasses.OrderedList)
			parseProps := propsWithClass(parseClassName)
			if parseTyped.Start != 1 {
				parseProps.Raw = map[string]interface{}{"start": parseTyped.Start}
			}
			return Tag("ol", parseProps, parseItems...), true
		}
		parseClassName = joinMarkdownClasses(parseClassName, parseClasses.UnorderedList)
		return Ul(propsWithClass(parseClassName), parseItems...), true
	case *ast.ListItem:
		return Li(propsWithClass(parseClasses.ListItem), renderMarkdownBlocks(parseTyped, parseSource, parseConfig)...), true
	case *ast.FencedCodeBlock:
		return renderMarkdownCodeBlock(strings.TrimRight(markdownLinesText(parseTyped.Lines(), parseSource), "\n"), parseConfig), true
	case *ast.CodeBlock:
		return renderMarkdownCodeBlock(strings.TrimRight(markdownLinesText(parseTyped.Lines(), parseSource), "\n"), parseConfig), true
	case *ast.ThematicBreak:
		return Hr(propsWithClass(parseClasses.HorizontalRule)), true
	default:
		parseTextValue := strings.TrimSpace(markdownPlainText(parseNode, parseSource))
		if parseTextValue == "" {
			return nil, false
		}
		return P(propsWithClass(parseClasses.Paragraph), Text(parseTextValue)), true
	}
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
		parseProps.Href = resolveMarkdownHref(parseConfig, string(parseTyped.Destination))
		parseProps.Target = parseConfig.LinkTarget
		parseProps.Rel = parseConfig.LinkRel
		return []ui.Node{A(parseProps, parseChildren2...)}
	case *ast.AutoLink:
		parseHref := string(parseTyped.URL(parseSource))
		parseProps2 := propsWithClass(parseClasses.Link)
		parseProps2.Href = parseHref
		parseProps2.Target = parseConfig.LinkTarget
		parseProps2.Rel = parseConfig.LinkRel
		return []ui.Node{A(parseProps2, Text(parseHref))}
	default:
		parseTextValue2 := markdownPlainText(parseTyped, parseSource)
		if parseTextValue2 == "" {
			return nil
		}
		return []ui.Node{Text(parseTextValue2)}
	}
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
