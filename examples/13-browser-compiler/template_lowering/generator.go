package templatelowering

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"strings"

	nethtml "golang.org/x/net/html"
)

type TemplateConfig struct {
	PackageName string
	StructName  string
	FuncName    string
}

func GenerateFromFile(path string, config TemplateConfig) (string, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Generate(input, config)
}

func Generate(input []byte, config TemplateConfig) (string, error) {
	if strings.TrimSpace(config.PackageName) == "" || strings.TrimSpace(config.StructName) == "" || strings.TrimSpace(config.FuncName) == "" {
		return "", fmt.Errorf("template lowering requires package, struct, and function names")
	}

	root, err := parseTemplate(input)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	buf.WriteString("package " + config.PackageName + "\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/monstercameron/GoWebComponents/html\"\n")
	buf.WriteString("\t\"github.com/monstercameron/GoWebComponents/ui\"\n")
	buf.WriteString(")\n\n")
	buf.WriteString("type " + config.StructName + " struct {\n")
	for _, field := range collectFields(root) {
		buf.WriteString("\t" + field + " string\n")
	}
	buf.WriteString("}\n\n")
	buf.WriteString("func " + config.FuncName + "(props " + config.StructName + ") ui.Node {\n")
	buf.WriteString("\treturn " + renderNode(root, 1) + "\n")
	buf.WriteString("}\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("format generated source: %w", err)
	}
	return string(formatted), nil
}

func parseTemplate(input []byte) (*nethtml.Node, error) {
	doc, err := nethtml.Parse(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	var root *nethtml.Node
	var visit func(*nethtml.Node)
	visit = func(node *nethtml.Node) {
		if root != nil {
			return
		}
		if node.Type == nethtml.ElementNode && node.Data != "html" && node.Data != "head" && node.Data != "body" {
			root = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)
	if root == nil {
		return nil, fmt.Errorf("no root element found")
	}
	return root, nil
}

func collectFields(root *nethtml.Node) []string {
	seen := map[string]bool{}
	fields := make([]string, 0, 8)
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.TextNode {
			if field, ok := placeholderField(node.Data); ok && !seen[field] {
				seen[field] = true
				fields = append(fields, field)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return fields
}

func renderNode(node *nethtml.Node, depth int) string {
	indent := strings.Repeat("\t", depth)
	if node.Type == nethtml.TextNode {
		text := strings.TrimSpace(node.Data)
		if text == "" {
			return ""
		}
		if field, ok := placeholderField(text); ok {
			return "html.Text(props." + field + ")"
		}
		return "html.Text(" + quote(text) + ")"
	}
	if node.Type != nethtml.ElementNode {
		return ""
	}

	builder := &strings.Builder{}
	builder.WriteString(tagFunc(node.Data))
	builder.WriteString("(html.Props{")
	propsParts := make([]string, 0, 2)
	for _, attr := range node.Attr {
		switch attr.Key {
		case "class":
			propsParts = append(propsParts, "Class: "+quote(strings.TrimSpace(attr.Val)))
		case "id":
			propsParts = append(propsParts, "ID: "+quote(strings.TrimSpace(attr.Val)))
		}
	}
	builder.WriteString(strings.Join(propsParts, ", "))
	builder.WriteString("}")

	children := renderChildren(node, depth+1)
	if len(children) == 0 {
		builder.WriteString(")")
		return builder.String()
	}
	builder.WriteString(",\n")
	for index, child := range children {
		builder.WriteString(indent)
		builder.WriteString(child)
		if index < len(children)-1 {
			builder.WriteString(",\n")
		}
	}
	builder.WriteString(")")
	return builder.String()
}

func renderChildren(node *nethtml.Node, depth int) []string {
	children := make([]string, 0, 4)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		rendered := renderNode(child, depth)
		if rendered == "" {
			continue
		}
		children = append(children, rendered)
	}
	return children
}

func placeholderField(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{{.") || !strings.HasSuffix(trimmed, "}}") {
		return "", false
	}
	field := strings.TrimSuffix(strings.TrimPrefix(trimmed, "{{."), "}}")
	field = strings.TrimSpace(field)
	if field == "" || strings.ContainsAny(field, " .-") {
		return "", false
	}
	return field, true
}

func tagFunc(tag string) string {
	switch tag {
	case "section":
		return "html.Section"
	case "p":
		return "html.P"
	case "h1":
		return "html.H1"
	case "div":
		return "html.Div"
	case "span":
		return "html.Span"
	default:
		return "html.Tag(" + quote(tag) + ", "
	}
}

func quote(value string) string {
	return fmt.Sprintf("%q", value)
}
