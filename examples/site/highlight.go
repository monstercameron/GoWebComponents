//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// highlightTokenClass maps chroma token categories onto the site's small
// token palette. Returning "" emits plain text.
func highlightTokenClass(parseType chroma.TokenType) string {
	switch {
	case parseType.InCategory(chroma.Keyword):
		return "tok-kw"
	case parseType.InCategory(chroma.String):
		return "tok-str"
	case parseType.InCategory(chroma.Comment):
		return "tok-com"
	case parseType.InCategory(chroma.Number):
		return "tok-num"
	case parseType == chroma.NameFunction:
		return "tok-fn"
	case parseType == chroma.NameClass || parseType == chroma.KeywordType:
		return "tok-typ"
	case parseType.InCategory(chroma.Operator), parseType.InCategory(chroma.Punctuation):
		return "tok-op"
	default:
		return ""
	}
}

// highlightSource tokenizes source code with chroma and returns styled nodes.
// Everything happens in Go inside the wasm module — no JS highlighter ships.
func highlightSource(parseSource string, parseLanguage string) []ui.Node {
	parseLexer := lexers.Get(parseLanguage)
	if parseLexer == nil {
		parseLexer = lexers.Fallback
	}
	parseIterator, parseErr := chroma.Coalesce(parseLexer).Tokenise(nil, parseSource)
	if parseErr != nil {
		return []ui.Node{Text(parseSource)}
	}

	var parseNodes []ui.Node
	for parseToken := parseIterator(); parseToken != chroma.EOF; parseToken = parseIterator() {
		parseValue := parseToken.Value
		if parseValue == "" {
			continue
		}
		parseClass := highlightTokenClass(parseToken.Type)
		if parseClass == "" || strings.TrimSpace(parseValue) == "" {
			parseNodes = append(parseNodes, Text(parseValue))
			continue
		}
		parseNodes = append(parseNodes, Span(ClassStr(parseClass), Text(parseValue)))
	}
	return parseNodes
}
