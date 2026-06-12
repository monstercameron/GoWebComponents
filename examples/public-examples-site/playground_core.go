package main

import (
	"encoding/base64"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"html"
	"net/url"
	"strconv"
	"strings"
)

const (
	playgroundSnippetParam = "snippet"

	defaultPlaygroundSnippet = `package main

func App() ui.Node {
	return Div(
		H1("Hello from the playground"),
		P("Edit this GoWebComponents snippet and share the URL."),
		Button("Compile and render"),
	)
}
`
)

type playgroundDiagnostic struct {
	Code     string
	Severity string
	Message  string
	Line     int
	Column   int
}

type playgroundNode struct {
	Tag      string
	Text     string
	Children []playgroundNode
}

type playgroundCompileResult struct {
	Source     string
	OK         bool
	Root       playgroundNode
	Diagnostic playgroundDiagnostic
}

var playgroundAllowedElements = map[string]string{
	"Article":    "article",
	"Aside":      "aside",
	"Blockquote": "blockquote",
	"Button":     "button",
	"Code":       "code",
	"Div":        "div",
	"Em":         "em",
	"Footer":     "footer",
	"H1":         "h1",
	"H2":         "h2",
	"H3":         "h3",
	"Header":     "header",
	"Li":         "li",
	"Main":       "main",
	"Mark":       "mark",
	"Ol":         "ol",
	"P":          "p",
	"Pre":        "pre",
	"Section":    "section",
	"Small":      "small",
	"Span":       "span",
	"Strong":     "strong",
	"Ul":         "ul",
}

func compilePlaygroundSnippet(parseSource string) playgroundCompileResult {
	parseSource = strings.TrimSpace(parseSource)
	if parseSource == "" {
		return playgroundCompileResult{
			Source: parseSource,
			Diagnostic: playgroundDiagnostic{
				Code:     "GWC-PLAYGROUND-EMPTY",
				Severity: "error",
				Message:  "Snippet source is empty.",
			},
		}
	}

	parseFset := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFset, "playground.go", parseSource, parser.AllErrors)
	if parseErr != nil {
		parsePosition := parseDiagnosticPosition(parseErr)
		return playgroundCompileResult{
			Source: parseSource,
			Diagnostic: playgroundDiagnostic{
				Code:     "GWC-PLAYGROUND-SYNTAX",
				Severity: "error",
				Message:  parseErr.Error(),
				Line:     parsePosition.Line,
				Column:   parsePosition.Column,
			},
		}
	}

	parseRenderExpr, parsePosition, parseOk := findPlaygroundReturnExpr(parseFset, parseFile)
	if !parseOk {
		return playgroundCompileResult{
			Source: parseSource,
			Diagnostic: playgroundDiagnostic{
				Code:     "GWC-PLAYGROUND-NO-APP",
				Severity: "error",
				Message:  "Define func App() ui.Node with one return expression.",
				Line:     parsePosition.Line,
				Column:   parsePosition.Column,
			},
		}
	}

	parseRoot, parseDiagnostic, parseCompileOK := compilePlaygroundExpr(parseFset, parseRenderExpr)
	if !parseCompileOK {
		return playgroundCompileResult{Source: parseSource, Diagnostic: parseDiagnostic}
	}
	return playgroundCompileResult{Source: parseSource, OK: true, Root: parseRoot}
}

func findPlaygroundReturnExpr(parseFset *token.FileSet, parseFile *ast.File) (ast.Expr, token.Position, bool) {
	parseFallback := parseFset.Position(parseFile.Pos())
	for _, parseDecl := range parseFile.Decls {
		parseFunc, parseOk := parseDecl.(*ast.FuncDecl)
		if !parseOk || parseFunc.Name == nil || parseFunc.Name.Name != "App" || parseFunc.Body == nil {
			continue
		}
		parseFallback = parseFset.Position(parseFunc.Pos())
		for _, parseStmt := range parseFunc.Body.List {
			parseReturn, parseReturnOK := parseStmt.(*ast.ReturnStmt)
			if !parseReturnOK {
				continue
			}
			if len(parseReturn.Results) != 1 {
				return nil, parseFset.Position(parseReturn.Pos()), false
			}
			return parseReturn.Results[0], parseFset.Position(parseReturn.Results[0].Pos()), true
		}
		return nil, parseFset.Position(parseFunc.Body.Lbrace), false
	}
	return nil, parseFallback, false
}

func compilePlaygroundExpr(parseFset *token.FileSet, parseExpr ast.Expr) (playgroundNode, playgroundDiagnostic, bool) {
	switch parseTyped := parseExpr.(type) {
	case *ast.BasicLit:
		if parseTyped.Kind != token.STRING {
			return playgroundNode{}, playgroundUnsupportedDiagnostic(parseFset, parseTyped.Pos(), "Only string literals can become text nodes."), false
		}
		parseText, parseErr := strconv.Unquote(parseTyped.Value)
		if parseErr != nil {
			return playgroundNode{}, playgroundUnsupportedDiagnostic(parseFset, parseTyped.Pos(), "String literal could not be decoded."), false
		}
		return playgroundNode{Text: parseText}, playgroundDiagnostic{}, true
	case *ast.CallExpr:
		parseName := playgroundCallName(parseTyped.Fun)
		if parseName == "Text" {
			return compilePlaygroundTextCall(parseFset, parseTyped)
		}
		parseTag, parseKnown := playgroundAllowedElements[parseName]
		if !parseKnown {
			return playgroundNode{}, playgroundUnsupportedDiagnostic(parseFset, parseTyped.Pos(), fmt.Sprintf("%s is not supported in the browser playground subset.", parseName)), false
		}
		parseNode := playgroundNode{Tag: parseTag}
		for _, parseArg := range parseTyped.Args {
			parseChild, parseDiagnostic, parseOk := compilePlaygroundExpr(parseFset, parseArg)
			if !parseOk {
				return playgroundNode{}, parseDiagnostic, false
			}
			parseNode.Children = append(parseNode.Children, parseChild)
		}
		return parseNode, playgroundDiagnostic{}, true
	default:
		return playgroundNode{}, playgroundUnsupportedDiagnostic(parseFset, parseExpr.Pos(), "Only whitelisted element calls and string literals are supported."), false
	}
}

func compilePlaygroundTextCall(parseFset *token.FileSet, parseCall *ast.CallExpr) (playgroundNode, playgroundDiagnostic, bool) {
	if len(parseCall.Args) != 1 {
		return playgroundNode{}, playgroundUnsupportedDiagnostic(parseFset, parseCall.Pos(), "Text requires exactly one string literal."), false
	}
	return compilePlaygroundExpr(parseFset, parseCall.Args[0])
}

func playgroundCallName(parseExpr ast.Expr) string {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name
	case *ast.SelectorExpr:
		if parseTyped.Sel == nil {
			return ""
		}
		return parseTyped.Sel.Name
	default:
		return ""
	}
}

func playgroundUnsupportedDiagnostic(parseFset *token.FileSet, parsePos token.Pos, parseMessage string) playgroundDiagnostic {
	parsePosition := parseFset.Position(parsePos)
	return playgroundDiagnostic{
		Code:     "GWC-PLAYGROUND-UNSUPPORTED",
		Severity: "error",
		Message:  parseMessage,
		Line:     parsePosition.Line,
		Column:   parsePosition.Column,
	}
}

func parseDiagnosticPosition(parseErr error) token.Position {
	if parseList, parseOk := parseErr.(scanner.ErrorList); parseOk && len(parseList) > 0 {
		return parseList[0].Pos
	}
	return token.Position{}
}

func playgroundNodeHTML(parseNode playgroundNode) string {
	if parseNode.Tag == "" {
		return html.EscapeString(parseNode.Text)
	}
	var parseBuilder strings.Builder
	parseBuilder.WriteString("<")
	parseBuilder.WriteString(parseNode.Tag)
	parseBuilder.WriteString(">")
	for _, parseChild := range parseNode.Children {
		parseBuilder.WriteString(playgroundNodeHTML(parseChild))
	}
	parseBuilder.WriteString("</")
	parseBuilder.WriteString(parseNode.Tag)
	parseBuilder.WriteString(">")
	return parseBuilder.String()
}

func playgroundSandboxHTML(parseResult playgroundCompileResult) string {
	if !parseResult.OK {
		return ""
	}
	return `<!doctype html><html><head><meta charset="utf-8"><style>
body{margin:0;background:#f8fafc;color:#0f172a;font:15px system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
.playground-root{min-height:100vh;padding:24px;box-sizing:border-box}
.playground-root>*{max-width:680px}
h1,h2,h3,p{margin:0 0 12px}
button{border:0;border-radius:8px;background:#0891b2;color:white;padding:10px 14px;font:inherit}
section,article,main,div{display:block}
pre,code{font-family:ui-monospace,SFMono-Regular,Consolas,monospace}
</style></head><body><main class="playground-root">` + playgroundNodeHTML(parseResult.Root) + `</main></body></html>`
}

func encodePlaygroundSource(parseSource string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(parseSource))
}

func decodePlaygroundSource(parseEncoded string) (string, bool, error) {
	parseEncoded = strings.TrimSpace(parseEncoded)
	if parseEncoded == "" {
		return "", false, nil
	}
	parseBytes, parseErr := base64.RawURLEncoding.DecodeString(parseEncoded)
	if parseErr != nil {
		return "", true, parseErr
	}
	return string(parseBytes), true, nil
}

func buildPlaygroundShareURL(parseCurrentHref string, parseSource string) string {
	parseURL, parseErr := url.Parse(parseCurrentHref)
	if parseErr != nil {
		return ""
	}
	parseQuery := parseURL.Query()
	if strings.TrimSpace(parseSource) == strings.TrimSpace(defaultPlaygroundSnippet) {
		parseQuery.Del(playgroundSnippetParam)
	} else {
		parseQuery.Set(playgroundSnippetParam, encodePlaygroundSource(parseSource))
	}
	parseURL.RawQuery = parseQuery.Encode()
	parseURL.Fragment = "playground"
	return parseURL.String()
}
