//go:build js && wasm

package main

import (
	"strings"
	"unicode"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// HighlightGoCode performs simple syntax highlighting for Go code
func HighlightGoCode(parseCode string) *Element {
	parseLines := strings.Split(parseCode, "\n")
	parseLineElements := make([]ui.Node, 0, len(parseLines))

	for _, parseLine := range parseLines {
		parseLineElements = append(parseLineElements, highlightLine(parseLine))
	}

	return html.Div(html.Props{
		Class: "text-xs font-mono leading-relaxed overflow-x-auto h-full pb-8 p-4 text-gray-300 bg-[#0a0a0a]",
		Style: map[string]string{"tab-size": "4"},
	}, parseLineElements...)
}

func highlightLine(parseLine string) *Element {
	if len(parseLine) == 0 {
		return html.Div(html.Props{Class: "h-4"})
	}

	var parseTokens []ui.Node

	parseCommentIdx := strings.Index(parseLine, "//")
	if parseCommentIdx != -1 {
		if parseCommentIdx > 0 {
			parseTokens = append(parseTokens, processCodeSegment(parseLine[:parseCommentIdx])...)
		}
		parseTokens = append(parseTokens, html.Span(html.Props{Class: "text-gray-500 italic"}, html.Text(parseLine[parseCommentIdx:])))
		return html.Div(html.Props{Class: "whitespace-pre"}, parseTokens...)
	}

	parseTokens = append(parseTokens, processCodeSegment(parseLine)...)
	return html.Div(html.Props{Class: "whitespace-pre"}, parseTokens...)
}

func processCodeSegment(parseSegment string) []ui.Node {
	var parseTokens []ui.Node

	parseCurrent := ""
	isParseInString := false
	parseStringChar := byte(0)

	parseFlush := func() {
		if parseCurrent != "" {
			parseTokens = append(parseTokens, colorizeToken(parseCurrent))
			parseCurrent = ""
		}
	}

	for parseI := 0; parseI < len(parseSegment); parseI++ {
		parseChar := parseSegment[parseI]

		if isParseInString {
			parseCurrent += string(parseChar)
			if parseChar == parseStringChar && (parseI == 0 || parseSegment[parseI-1] != '\\') {
				parseTokens = append(parseTokens, html.Span(html.Props{Class: "text-green-400"}, html.Text(parseCurrent)))
				parseCurrent = ""
				isParseInString = false
			}
			continue
		}

		if parseChar == '"' || parseChar == '`' || parseChar == '\'' {
			parseFlush()
			parseCurrent += string(parseChar)
			isParseInString = true
			parseStringChar = parseChar
			continue
		}

		if isDelimiter(parseChar) {
			parseFlush()
			parseTokens = append(parseTokens, html.Span(html.Props{Class: "text-gray-500"}, html.Text(string(parseChar))))
		} else if unicode.IsSpace(rune(parseChar)) {
			parseFlush()
			parseTokens = append(parseTokens, html.Text(string(parseChar)))
		} else {
			parseCurrent += string(parseChar)
		}
	}
	parseFlush()

	return parseTokens
}

func isDelimiter(parseChar byte) bool {
	return strings.ContainsRune("(){}[],.;:", rune(parseChar))
}

func colorizeToken(parseToken string) *Element {
	parseKeywords := map[string]bool{
		"func": true, "return": true, "if": true, "else": true,
		"for": true, "range": true, "var": true, "const": true,
		"type": true, "struct": true, "interface": true, "package": true,
		"import": true, "go": true, "defer": true, "select": true,
		"case": true, "default": true, "switch": true, "break": true,
		"continue": true, "fallthrough": true, "goto": true, "map": true,
		"chan": true, "make": true, "new": true, "len": true, "cap": true,
		"append": true, "copy": true, "close": true, "delete": true,
		"complex": true, "real": true, "imag": true, "panic": true, "recover": true,
		"print": true, "println": true, "true": true, "false": true, "nil": true,
		"iota": true, "string": true, "int": true, "bool": true, "byte": true,
		"error": true, "float32": true, "float64": true, "int32": true, "int64": true,
	}

	if parseKeywords[parseToken] {
		return html.Span(html.Props{Class: "text-purple-400 font-bold"}, html.Text(parseToken))
	}

	if isNumber(parseToken) {
		return html.Span(html.Props{Class: "text-orange-400"}, html.Text(parseToken))
	}

	return html.Span(html.Props{Class: "text-blue-300"}, html.Text(parseToken))
}

func isNumber(parseS string) bool {
	if len(parseS) == 0 {
		return false
	}
	for _, parseC := range parseS {
		if !unicode.IsDigit(parseC) && parseC != '.' {
			return false
		}
	}
	return true
}
