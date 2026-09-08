//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"
	"unicode"

	. "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderHeroCodeSnippet renders a compact, syntax-colored Go counter example for the landing hero.
func renderHeroCodeSnippet() ui.Node {
	return Div(ClassStr("rounded-[24px] border border-cyan-300/15 bg-[#050d18]/85 p-4 shadow-xl shadow-black/20"),
		Div(ClassStr("flex items-center justify-between gap-3"),
			Div(ClassStr("text-[11px] uppercase tracking-[0.18em] text-cyan-200"), Text("Quick Counter Example")),
			Div(ClassStr("rounded-full border border-white/10 bg-white/[0.04] px-2.5 py-1 text-[10px] uppercase tracking-[0.16em] text-slate-400"), Text("Go + hooks + typed HTML")),
		),
		Pre(ClassStr("mt-3 overflow-x-auto rounded-[18px] border border-white/10 bg-black/20 p-4 text-[13px] leading-6 text-slate-200"),
			renderQuickCounterSnippetCode(),
		),
	)
}

func renderQuickCounterSnippetCode() ui.Node {
	return Code(
		Span(ClassStr("text-violet-300"), Text("func")),
		Text(" "),
		Span(ClassStr("text-cyan-200"), Text("Counter")),
		Text("() "),
		Span(ClassStr("text-violet-300"), Text("ui.Node")),
		Text(" {\n  "),
		Span(ClassStr("text-amber-200"), Text("count")),
		Text(" := "),
		Span(ClassStr("text-cyan-300"), Text("ui.UseState")),
		Text("("),
		Span(ClassStr("text-emerald-300"), Text("0")),
		Text(")\n  "),
		Span(ClassStr("text-amber-200"), Text("currentCount")),
		Text(" := "),
		Span(ClassStr("text-amber-200"), Text("count")),
		Text("."),
		Span(ClassStr("text-cyan-200"), Text("Get")),
		Text("()\n  "),
		Span(ClassStr("text-amber-200"), Text("increment")),
		Text(" := "),
		Span(ClassStr("text-cyan-300"), Text("ui.UseEvent")),
		Text("("),
		Span(ClassStr("text-violet-300"), Text("func")),
		Text("() {\n    "),
		Span(ClassStr("text-amber-200"), Text("count")),
		Text("."),
		Span(ClassStr("text-cyan-200"), Text("Set")),
		Text("("),
		Span(ClassStr("text-amber-200"), Text("count")),
		Text("."),
		Span(ClassStr("text-cyan-200"), Text("Get")),
		Text("() + "),
		Span(ClassStr("text-emerald-300"), Text("1")),
		Text(")\n  })\n\n  "),
		Span(ClassStr("text-cyan-300"), Text("return")),
		Text(" "),
		Span(ClassStr("text-cyan-200"), Text("Button")),
		Text("(\n    "),
		Span(ClassStr("text-cyan-200"), Text("OnClick")),
		Text("("),
		Span(ClassStr("text-amber-200"), Text("increment")),
		Text("),\n    "),
		Span(ClassStr("text-cyan-200"), Text("Class")),
		Text("("),
		Span(ClassStr("text-emerald-300"), Text("\"rounded-xl px-4 py-2\"")),
		Text("),\n    "),
		Span(ClassStr("text-cyan-200"), Text("Textf")),
		Text("("),
		Span(ClassStr("text-emerald-300"), Text("\"Clicked %d times\"")),
		Text(", "),
		Span(ClassStr("text-amber-200"), Text("currentCount")),
		Text("),\n  )\n}"),
	)
}

func isQuickCounterSnippet(parseSource string) bool {
	parseNormalized := strings.ReplaceAll(parseSource, "\r\n", "\n")
	return strings.Contains(parseNormalized, "func Counter() ui.Node {") &&
		strings.Contains(parseNormalized, "count.Set(count.Get() + 1)") &&
		strings.Contains(parseNormalized, "Textf(\"Clicked %d times\", currentCount)")
}

// isSourceHighlightEnabled reports whether the custom highlighter should run for one source blob.
func isSourceHighlightEnabled(parseSource string) bool {
	parseNormalized := strings.ReplaceAll(parseSource, "\r\n", "\n")
	if len(parseNormalized) > 12000 {
		return false
	}
	if strings.Count(parseNormalized, "\n") > 220 {
		return false
	}
	return true
}

func isGoIdentifierStart(parseChar byte) bool {
	return parseChar == '_' || unicode.IsLetter(rune(parseChar))
}

func isGoIdentifierPart(parseChar byte) bool {
	return parseChar == '_' || unicode.IsLetter(rune(parseChar)) || unicode.IsDigit(rune(parseChar))
}

func renderHighlightedGoSource(parseSource string) ui.Node {
	parseKeywordClasses := map[string]string{
		"break":       "text-violet-300",
		"case":        "text-violet-300",
		"default":     "text-violet-300",
		"defer":       "text-violet-300",
		"else":        "text-violet-300",
		"fallthrough": "text-violet-300",
		"for":         "text-violet-300",
		"func":        "text-violet-300",
		"go":          "text-violet-300",
		"if":          "text-violet-300",
		"import":      "text-violet-300",
		"package":     "text-violet-300",
		"range":       "text-violet-300",
		"return":      "text-violet-300",
		"select":      "text-violet-300",
		"switch":      "text-violet-300",
		"type":        "text-violet-300",
		"var":         "text-violet-300",
	}
	parseCallClasses := map[string]string{
		"Button":          "text-cyan-200",
		"Class":           "text-cyan-200",
		"ClassNames":      "text-cyan-200",
		"CounterExample":  "text-cyan-200",
		"CreateElement":   "text-cyan-200",
		"DisableAllDebug": "text-cyan-200",
		"Div":             "text-cyan-200",
		"Fragment":        "text-cyan-200",
		"H2":              "text-cyan-200",
		"IfElse":          "text-cyan-200",
		"LookupString":    "text-cyan-200",
		"OnClick":         "text-cyan-200",
		"P":               "text-cyan-200",
		"Render":          "text-cyan-200",
		"Span":            "text-cyan-200",
		"String":          "text-cyan-200",
		"Text":            "text-cyan-200",
		"Textf":           "text-cyan-200",
		"UseEvent":        "text-cyan-300",
		"UseState":        "text-cyan-300",
		"WaitForever":     "text-cyan-200",
		"When":            "text-cyan-200",
	}
	parseModuleClasses := map[string]string{
		"interop": "text-cyan-200",
		"ui":      "text-cyan-200",
		"utils":   "text-cyan-200",
	}
	var parseNodes []ui.Node
	parseNormalized := strings.ReplaceAll(parseSource, "\r\n", "\n")

	parseAppendText := func(parseText string) {
		if parseText != "" {
			parseNodes = append(parseNodes, Text(parseText))
		}
	}
	parseAppendClassed := func(parseClassName, parseText2 string) {
		if parseText2 == "" {
			return
		}
		parseNodes = append(parseNodes, Span(ClassStr(parseClassName), Text(parseText2)))
	}

	for parseIndex := 0; parseIndex < len(parseNormalized); {
		if strings.HasPrefix(parseNormalized[parseIndex:], "//") {
			parseEnd := parseIndex
			for parseEnd < len(parseNormalized) && parseNormalized[parseEnd] != '\n' {
				parseEnd++
			}
			parseAppendClassed("text-slate-500", parseNormalized[parseIndex:parseEnd])
			parseIndex = parseEnd
			continue
		}
		if parseNormalized[parseIndex] == '"' {
			parseEnd2 := parseIndex + 1
			for parseEnd2 < len(parseNormalized) {
				if parseNormalized[parseEnd2] == '\\' && parseEnd2+1 < len(parseNormalized) {
					parseEnd2 += 2
					continue
				}
				if parseNormalized[parseEnd2] == '"' {
					parseEnd2++
					break
				}
				parseEnd2++
			}
			parseAppendClassed("text-emerald-300", parseNormalized[parseIndex:parseEnd2])
			parseIndex = parseEnd2
			continue
		}
		if parseNormalized[parseIndex] == '`' {
			parseEnd3 := parseIndex + 1
			for parseEnd3 < len(parseNormalized) && parseNormalized[parseEnd3] != '`' {
				parseEnd3++
			}
			if parseEnd3 < len(parseNormalized) {
				parseEnd3++
			}
			parseAppendClassed("text-emerald-300", parseNormalized[parseIndex:parseEnd3])
			parseIndex = parseEnd3
			continue
		}
		if unicode.IsDigit(rune(parseNormalized[parseIndex])) {
			parseEnd4 := parseIndex + 1
			for parseEnd4 < len(parseNormalized) && (unicode.IsDigit(rune(parseNormalized[parseEnd4])) || parseNormalized[parseEnd4] == '.') {
				parseEnd4++
			}
			parseAppendClassed("text-emerald-300", parseNormalized[parseIndex:parseEnd4])
			parseIndex = parseEnd4
			continue
		}
		if isGoIdentifierStart(parseNormalized[parseIndex]) {
			parseEnd5 := parseIndex + 1
			for parseEnd5 < len(parseNormalized) && isGoIdentifierPart(parseNormalized[parseEnd5]) {
				parseEnd5++
			}
			parseToken := parseNormalized[parseIndex:parseEnd5]
			switch {
			case parseKeywordClasses[parseToken] != "":
				parseAppendClassed(parseKeywordClasses[parseToken], parseToken)
			case parseCallClasses[parseToken] != "":
				parseAppendClassed(parseCallClasses[parseToken], parseToken)
			case parseModuleClasses[parseToken] != "":
				parseAppendClassed(parseModuleClasses[parseToken], parseToken)
			case unicode.IsUpper(rune(parseToken[0])):
				parseAppendClassed("text-cyan-200", parseToken)
			default:
				parseAppendClassed("text-amber-200", parseToken)
			}
			parseIndex = parseEnd5
			continue
		}
		parseAppendText(string(parseNormalized[parseIndex]))
		parseIndex++
	}

	parseArgs := make([]interface{}, 0, len(parseNodes))
	for _, parseNode := range parseNodes {
		parseArgs = append(parseArgs, parseNode)
	}
	return Code(parseArgs...)
}

func renderSourceSnippetCard(parseTitle, parseBadge, parseSource string) ui.Node {
	parseSourceNode := Code(Text(parseSource))
	switch {
	case isQuickCounterSnippet(parseSource):
		parseSourceNode = renderQuickCounterSnippetCode()
	case isSourceHighlightEnabled(parseSource):
		parseSourceNode = renderHighlightedGoSource(parseSource)
	}
	return Div(ClassStr("min-w-0 rounded-[24px] border border-cyan-300/15 bg-[#050d18]/85 p-4 shadow-xl shadow-black/20"),
		Div(ClassStr("flex items-center justify-between gap-3"),
			Div(ClassStr("text-[11px] uppercase tracking-[0.18em] text-cyan-200"), Text(parseTitle)),
			Div(ClassStr("rounded-full border border-white/10 bg-white/[0.04] px-2.5 py-1 text-[10px] uppercase tracking-[0.16em] text-slate-400"), Text(parseBadge)),
		),
		Pre(ClassStr("mt-3 overflow-x-auto rounded-[18px] border border-white/10 bg-black/20 p-4 text-[13px] leading-6 text-slate-200"),
			parseSourceNode,
		),
	)
}
