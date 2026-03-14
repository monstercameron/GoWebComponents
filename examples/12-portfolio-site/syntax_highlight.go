//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"
	"unicode"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// HighlightGoCode performs simple syntax highlighting for Go code
func HighlightGoCode(code string) *Element {
	lines := strings.Split(code, "\n")
	lineElements := make([]ui.Node, 0, len(lines))

	for _, line := range lines {
		lineElements = append(lineElements, highlightLine(line))
	}

	return html.Div(html.Props{
		Class: "text-xs font-mono leading-relaxed overflow-x-auto h-full pb-8 p-4 text-gray-300 bg-[#0a0a0a]",
		Style: map[string]string{"tab-size": "4"},
	}, lineElements...)
}

func highlightLine(line string) *Element {
	if len(line) == 0 {
		return html.Div(html.Props{Class: "h-4"})
	}

	var tokens []ui.Node

	commentIdx := strings.Index(line, "//")
	if commentIdx != -1 {
		if commentIdx > 0 {
			tokens = append(tokens, processCodeSegment(line[:commentIdx])...)
		}
		tokens = append(tokens, html.Span(html.Props{Class: "text-gray-500 italic"}, html.Text(line[commentIdx:])))
		return html.Div(html.Props{Class: "whitespace-pre"}, tokens...)
	}

	tokens = append(tokens, processCodeSegment(line)...)
	return html.Div(html.Props{Class: "whitespace-pre"}, tokens...)
}

func processCodeSegment(segment string) []ui.Node {
	var tokens []ui.Node

	current := ""
	inString := false
	stringChar := byte(0)

	flush := func() {
		if current != "" {
			tokens = append(tokens, colorizeToken(current))
			current = ""
		}
	}

	for i := 0; i < len(segment); i++ {
		char := segment[i]

		if inString {
			current += string(char)
			if char == stringChar && (i == 0 || segment[i-1] != '\\') {
				tokens = append(tokens, html.Span(html.Props{Class: "text-green-400"}, html.Text(current)))
				current = ""
				inString = false
			}
			continue
		}

		if char == '"' || char == '`' || char == '\'' {
			flush()
			current += string(char)
			inString = true
			stringChar = char
			continue
		}

		if isDelimiter(char) {
			flush()
			tokens = append(tokens, html.Span(html.Props{Class: "text-gray-500"}, html.Text(string(char))))
		} else if unicode.IsSpace(rune(char)) {
			flush()
			tokens = append(tokens, html.Text(string(char)))
		} else {
			current += string(char)
		}
	}
	flush()

	return tokens
}

func isDelimiter(char byte) bool {
	return strings.ContainsRune("(){}[],.;:", rune(char))
}

func colorizeToken(token string) *Element {
	keywords := map[string]bool{
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

	if keywords[token] {
		return html.Span(html.Props{Class: "text-purple-400 font-bold"}, html.Text(token))
	}

	if isNumber(token) {
		return html.Span(html.Props{Class: "text-orange-400"}, html.Text(token))
	}

	return html.Span(html.Props{Class: "text-blue-300"}, html.Text(token))
}

func isNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !unicode.IsDigit(c) && c != '.' {
			return false
		}
	}
	return true
}
