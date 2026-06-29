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

func GenerateFromFile(parsePath string, parseConfig TemplateConfig) (string, error) {
	parseInput, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return "", parseErr
	}
	return Generate(parseInput, parseConfig)
}

func Generate(parseInput []byte, parseConfig TemplateConfig) (string, error) {
	if strings.TrimSpace(parseConfig.PackageName) == "" || strings.TrimSpace(parseConfig.StructName) == "" || strings.TrimSpace(parseConfig.FuncName) == "" {
		return "", fmt.Errorf("template lowering requires package, struct, and function names")
	}

	parseRoot, parseErr := parseTemplate(parseInput)
	if parseErr != nil {
		return "", parseErr
	}

	var parseBuf bytes.Buffer
	parseBuf.WriteString("package " + parseConfig.PackageName + "\n\n")
	parseBuf.WriteString("import (\n")
	parseBuf.WriteString("\t\"github.com/monstercameron/GoWebComponents/v4/html\"\n")
	parseBuf.WriteString("\t\"github.com/monstercameron/GoWebComponents/v4/ui\"\n")
	parseBuf.WriteString(")\n\n")
	parseBuf.WriteString("type " + parseConfig.StructName + " struct {\n")
	for _, parseField := range collectFields(parseRoot) {
		parseBuf.WriteString("\t" + parseField + " string\n")
	}
	parseBuf.WriteString("}\n\n")
	parseBuf.WriteString("func " + parseConfig.FuncName + "(props " + parseConfig.StructName + ") ui.Node {\n")
	parseBuf.WriteString("\treturn " + renderNode(parseRoot, 1) + "\n")
	parseBuf.WriteString("}\n")

	parseFormatted, parseErr := format.Source(parseBuf.Bytes())
	if parseErr != nil {
		return "", fmt.Errorf("format generated source: %w", parseErr)
	}
	return string(parseFormatted), nil
}

func parseTemplate(parseInput []byte) (*nethtml.Node, error) {
	parseDoc, parseErr := nethtml.Parse(bytes.NewReader(parseInput))
	if parseErr != nil {
		return nil, parseErr
	}
	var parseRoot *nethtml.Node
	var parseVisit func(*nethtml.Node)
	parseVisit = func(parseNode *nethtml.Node) {
		if parseRoot != nil {
			return
		}
		if parseNode.Type == nethtml.ElementNode && parseNode.Data != "html" && parseNode.Data != "head" && parseNode.Data != "body" {
			parseRoot = parseNode
			return
		}
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			parseVisit(parseChild)
		}
	}
	parseVisit(parseDoc)
	if parseRoot == nil {
		return nil, fmt.Errorf("no root element found")
	}
	return parseRoot, nil
}

func collectFields(parseRoot *nethtml.Node) []string {
	parseSeen := map[string]bool{}
	parseFields := make([]string, 0, 8)
	var parseWalk func(*nethtml.Node)
	parseWalk = func(parseNode *nethtml.Node) {
		if parseNode.Type == nethtml.TextNode {
			if parseField, parseOk := placeholderField(parseNode.Data); parseOk && !parseSeen[parseField] {
				parseSeen[parseField] = true
				parseFields = append(parseFields, parseField)
			}
		}
		for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
			parseWalk(parseChild)
		}
	}
	parseWalk(parseRoot)
	return parseFields
}

func renderNode(parseNode *nethtml.Node, parseDepth int) string {
	parseIndent := strings.Repeat("\t", parseDepth)
	if parseNode.Type == nethtml.TextNode {
		parseText := strings.TrimSpace(parseNode.Data)
		if parseText == "" {
			return ""
		}
		if parseField, parseOk := placeholderField(parseText); parseOk {
			return "html.Text(props." + parseField + ")"
		}
		return "html.Text(" + quote(parseText) + ")"
	}
	if parseNode.Type != nethtml.ElementNode {
		return ""
	}

	parseBuilder := &strings.Builder{}
	parseBuilder.WriteString(tagFunc(parseNode.Data))
	parseBuilder.WriteString("(html.Props{")
	parsePropsParts := make([]string, 0, 2)
	for _, parseAttr := range parseNode.Attr {
		switch parseAttr.Key {
		case "class":
			parsePropsParts = append(parsePropsParts, "Class: "+quote(strings.TrimSpace(parseAttr.Val)))
		case "id":
			parsePropsParts = append(parsePropsParts, "ID: "+quote(strings.TrimSpace(parseAttr.Val)))
		}
	}
	parseBuilder.WriteString(strings.Join(parsePropsParts, ", "))
	parseBuilder.WriteString("}")

	parseChildren := renderChildren(parseNode, parseDepth+1)
	if len(parseChildren) == 0 {
		parseBuilder.WriteString(")")
		return parseBuilder.String()
	}
	parseBuilder.WriteString(",\n")
	for parseIndex, parseChild := range parseChildren {
		parseBuilder.WriteString(parseIndent)
		parseBuilder.WriteString(parseChild)
		if parseIndex < len(parseChildren)-1 {
			parseBuilder.WriteString(",\n")
		}
	}
	parseBuilder.WriteString(")")
	return parseBuilder.String()
}

func renderChildren(parseNode *nethtml.Node, parseDepth int) []string {
	parseChildren := make([]string, 0, 4)
	for parseChild := parseNode.FirstChild; parseChild != nil; parseChild = parseChild.NextSibling {
		parseRendered := renderNode(parseChild, parseDepth)
		if parseRendered == "" {
			continue
		}
		parseChildren = append(parseChildren, parseRendered)
	}
	return parseChildren
}

func placeholderField(parseText string) (string, bool) {
	parseTrimmed := strings.TrimSpace(parseText)
	if !strings.HasPrefix(parseTrimmed, "{{.") || !strings.HasSuffix(parseTrimmed, "}}") {
		return "", false
	}
	parseField := strings.TrimSuffix(strings.TrimPrefix(parseTrimmed, "{{."), "}}")
	parseField = strings.TrimSpace(parseField)
	if parseField == "" || strings.ContainsAny(parseField, " .-") {
		return "", false
	}
	return parseField, true
}

func tagFunc(parseTag string) string {
	switch parseTag {
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
		return "html.Tag(" + quote(parseTag) + ", "
	}
}

func quote(parseValue string) string {
	return fmt.Sprintf("%q", parseValue)
}
