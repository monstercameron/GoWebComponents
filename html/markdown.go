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
func RenderMarkdown(markdown string, options ...MarkdownRenderOptions) []ui.Node {
	config := MarkdownRenderOptions{}
	if len(options) > 0 {
		config = options[0]
	}
	source := []byte(markdown)
	root := goldmark.New().Parser().Parse(text.NewReader(source))
	return renderMarkdownBlocks(root, source, config)
}

// ResolveMarkdownHref resolves a markdown destination against the source document path.
func ResolveMarkdownHref(sourcePath, destination string) string {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return ""
	}
	parsedDestination, err := url.Parse(destination)
	if err != nil {
		return destination
	}
	if parsedDestination.IsAbs() || strings.HasPrefix(destination, "#") || strings.HasPrefix(destination, "/") {
		return destination
	}
	base := strings.TrimSpace(sourcePath)
	if base == "" {
		return destination
	}
	parsedBase, err := url.Parse(base)
	if err == nil && (parsedBase.IsAbs() || strings.HasPrefix(base, "/")) {
		return parsedBase.ResolveReference(parsedDestination).String()
	}
	basePath := strings.ReplaceAll(base, "\\", "/")
	resolved := *parsedDestination
	resolved.Path = path.Clean(path.Join(path.Dir(basePath), parsedDestination.Path))
	if strings.HasPrefix(basePath, "/") && !strings.HasPrefix(resolved.Path, "/") {
		resolved.Path = "/" + resolved.Path
	}
	return resolved.String()
}

func renderMarkdownBlocks(parent ast.Node, source []byte, config MarkdownRenderOptions) []ui.Node {
	nodes := make([]ui.Node, 0)
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if rendered, ok := renderMarkdownBlock(child, source, config); ok {
			nodes = append(nodes, rendered)
		}
	}
	return nodes
}

func renderMarkdownBlock(node ast.Node, source []byte, config MarkdownRenderOptions) (ui.Node, bool) {
	classes := config.Classes
	switch typed := node.(type) {
	case *ast.Heading:
		children := renderMarkdownInlines(typed, source, config)
		switch typed.Level {
		case 1:
			return H1(propsWithClass(classes.Heading1), children...), true
		case 2:
			return H2(propsWithClass(classes.Heading2), children...), true
		case 3:
			return H3(propsWithClass(classes.Heading3), children...), true
		case 4:
			return H4(propsWithClass(classes.Heading4), children...), true
		case 5:
			return H5(propsWithClass(classes.Heading5), children...), true
		default:
			return H6(propsWithClass(classes.Heading6), children...), true
		}
	case *ast.Paragraph:
		return P(propsWithClass(classes.Paragraph), renderMarkdownInlines(typed, source, config)...), true
	case *ast.Blockquote:
		return Blockquote(propsWithClass(classes.Blockquote), renderMarkdownBlocks(typed, source, config)...), true
	case *ast.List:
		items := renderMarkdownBlocks(typed, source, config)
		className := strings.TrimSpace(classes.List)
		if typed.IsOrdered() {
			className = joinMarkdownClasses(className, classes.OrderedList)
			props := propsWithClass(className)
			if typed.Start != 1 {
				props.Raw = map[string]interface{}{"start": typed.Start}
			}
			return Tag("ol", props, items...), true
		}
		className = joinMarkdownClasses(className, classes.UnorderedList)
		return Ul(propsWithClass(className), items...), true
	case *ast.ListItem:
		return Li(propsWithClass(classes.ListItem), renderMarkdownBlocks(typed, source, config)...), true
	case *ast.FencedCodeBlock:
		return renderMarkdownCodeBlock(strings.TrimRight(markdownLinesText(typed.Lines(), source), "\n"), config), true
	case *ast.CodeBlock:
		return renderMarkdownCodeBlock(strings.TrimRight(markdownLinesText(typed.Lines(), source), "\n"), config), true
	case *ast.ThematicBreak:
		return Hr(propsWithClass(classes.HorizontalRule)), true
	default:
		textValue := strings.TrimSpace(markdownPlainText(node, source))
		if textValue == "" {
			return nil, false
		}
		return P(propsWithClass(classes.Paragraph), Text(textValue)), true
	}
}

func renderMarkdownCodeBlock(code string, config MarkdownRenderOptions) ui.Node {
	children := make([]ui.Node, 0, 2)
	if strings.TrimSpace(config.CodeBlockLabel) != "" {
		children = append(children, Div(propsWithClass(config.Classes.CodeBlockLabel), Text(config.CodeBlockLabel)))
	}
	children = append(children, Pre(propsWithClass(config.Classes.CodeBlockPre), Code(Props{}, Text(code))))
	return Div(propsWithClass(config.Classes.CodeBlockContainer), children...)
}

func renderMarkdownInlines(parent ast.Node, source []byte, config MarkdownRenderOptions) []ui.Node {
	nodes := make([]ui.Node, 0)
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		nodes = append(nodes, renderMarkdownInline(child, source, config)...)
	}
	return nodes
}

func renderMarkdownInline(node ast.Node, source []byte, config MarkdownRenderOptions) []ui.Node {
	classes := config.Classes
	switch typed := node.(type) {
	case *ast.Text:
		textValue := string(typed.Segment.Value(source))
		nodes := []ui.Node{Text(textValue)}
		if typed.HardLineBreak() || typed.SoftLineBreak() {
			nodes = append(nodes, Br(Props{}))
		}
		return nodes
	case *ast.CodeSpan:
		return []ui.Node{Code(propsWithClass(classes.InlineCode), Text(markdownPlainText(typed, source)))}
	case *ast.Emphasis:
		children := renderMarkdownInlines(typed, source, config)
		if typed.Level == 2 {
			return []ui.Node{Tag("strong", propsWithClass(classes.Strong), children...)}
		}
		return []ui.Node{Em(propsWithClass(classes.Emphasis), children...)}
	case *ast.Link:
		children := renderMarkdownInlines(typed, source, config)
		props := propsWithClass(classes.Link)
		props.Href = resolveMarkdownHref(config, string(typed.Destination))
		props.Target = config.LinkTarget
		props.Rel = config.LinkRel
		return []ui.Node{A(props, children...)}
	case *ast.AutoLink:
		href := string(typed.URL(source))
		props := propsWithClass(classes.Link)
		props.Href = href
		props.Target = config.LinkTarget
		props.Rel = config.LinkRel
		return []ui.Node{A(props, Text(href))}
	default:
		textValue := markdownPlainText(typed, source)
		if textValue == "" {
			return nil
		}
		return []ui.Node{Text(textValue)}
	}
}

func markdownLinesText(lines *text.Segments, source []byte) string {
	if lines == nil {
		return ""
	}
	var builder strings.Builder
	for index := 0; index < lines.Len(); index++ {
		segment := lines.At(index)
		builder.Write(segment.Value(source))
	}
	return builder.String()
}

func markdownPlainText(node ast.Node, source []byte) string {
	var builder strings.Builder
	var walk func(ast.Node)
	walk = func(current ast.Node) {
		if current == nil {
			return
		}
		switch typed := current.(type) {
		case *ast.Text:
			builder.Write(typed.Segment.Value(source))
			if typed.HardLineBreak() || typed.SoftLineBreak() {
				builder.WriteString("\n")
			}
		case *ast.FencedCodeBlock:
			builder.WriteString(markdownLinesText(typed.Lines(), source))
		case *ast.CodeBlock:
			builder.WriteString(markdownLinesText(typed.Lines(), source))
		default:
			for child := current.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child)
			}
		}
	}
	walk(node)
	return strings.TrimSpace(builder.String())
}

func resolveMarkdownHref(config MarkdownRenderOptions, destination string) string {
	if config.ResolveHref != nil {
		return config.ResolveHref(config.SourcePath, destination)
	}
	return ResolveMarkdownHref(config.SourcePath, destination)
}

func propsWithClass(className string) Props {
	className = strings.TrimSpace(className)
	if className == "" {
		return Props{}
	}
	return Props{Class: className}
}

func joinMarkdownClasses(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " ")
}
