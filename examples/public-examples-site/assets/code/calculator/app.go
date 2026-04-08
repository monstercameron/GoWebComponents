//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	calculatorThemeAtom  = "calculator-theme"
	calculatorAngleAtom  = "calculator-angle"
	calculatorMemoryAtom = "calculator-memory"
)

type keySpec struct {
	Label string
	Tone  string
	Token string
	Class string
}

var quickInsertKeySpecs = []keySpec{
	{Label: "sin(", Tone: "ghost", Token: "sin("},
	{Label: "cos(", Tone: "ghost", Token: "cos("},
	{Label: "tan(", Tone: "ghost", Token: "tan("},
	{Label: "sqrt(", Tone: "ghost", Token: "sqrt("},
	{Label: "pow(", Tone: "ghost", Token: "pow("},
	{Label: "log(", Tone: "ghost", Token: "log("},
	{Label: "abs(", Tone: "ghost", Token: "abs("},
	{Label: "pi", Tone: "ghost", Token: "pi"},
	{Label: "e", Tone: "ghost", Token: "e"},
	{Label: ",", Tone: "ghost", Token: ","},
	{Label: "(", Tone: "ghost", Token: "("},
	{Label: ")", Tone: "ghost", Token: ")"},
}

var keypadSpecs = []keySpec{
	{Label: "C", Tone: "ghost", Class: "text-sm tracking-[0.18em]"},
	{Label: "DEL", Tone: "ghost", Class: "text-sm tracking-[0.12em]"},
	{Label: "^", Tone: "accent", Token: "^"},
	{Label: "/", Tone: "accent", Token: "/"},
	{Label: "7", Tone: "primary", Token: "7"},
	{Label: "8", Tone: "primary", Token: "8"},
	{Label: "9", Tone: "primary", Token: "9"},
	{Label: "*", Tone: "accent", Token: "*"},
	{Label: "4", Tone: "primary", Token: "4"},
	{Label: "5", Tone: "primary", Token: "5"},
	{Label: "6", Tone: "primary", Token: "6"},
	{Label: "-", Tone: "accent", Token: "-"},
	{Label: "1", Tone: "primary", Token: "1"},
	{Label: "2", Tone: "primary", Token: "2"},
	{Label: "3", Tone: "primary", Token: "3"},
	{Label: "+", Tone: "accent", Token: "+"},
	{Label: "0", Tone: "primary", Token: "0", Class: "col-span-2"},
	{Label: ".", Tone: "primary", Token: "."},
	{Label: "=", Tone: "solve"},
}

var quickDeleteFunctionTokens = []string{
	"sqrt(",
	"pow(",
	"sin(",
	"cos(",
	"tan(",
	"log(",
	"abs(",
}

func readStorageValue(parseStorage js.Value, parseKey string) string {
	if !parseStorage.Truthy() {
		return ""
	}

	parseValue := parseStorage.Call("getItem", parseKey)
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return ""
	}

	return strings.TrimSpace(parseValue.String())
}

func normalizeCalculatorTheme(parseValue string) string {
	if strings.TrimSpace(parseValue) == "midnight" {
		return "midnight"
	}
	return "graphite"
}

func App() ui.Node {
	parseTheme := state.UseAtom(calculatorThemeAtom, "graphite")
	parseAngle := state.UseAtom(calculatorAngleAtom, "deg")
	parseMemory := state.UseAtom(calculatorMemoryAtom, "0")

	parseExpression := ui.UseState("sin(45)+pow(2,5)")
	parsePrecision := ui.UseState(4)
	parseLastAction := ui.UseState("Ready for input")
	parseEvaluationCount := ui.UseState(0)
	parseHydrated := ui.UseState(false)

	parseSessionRef := ui.UseRef("CALC-" + time.Now().Format("150405"))
	parseLastStableResult := ui.UseRef("0")
	parseExpressionID := ui.UseId()

	parseEvaluation := ui.UseMemo(func() Evaluation {
		return EvaluateExpression(parseExpression.Get(), parseAngle.Get(), parsePrecision.Get())
	}, parseExpression.Get(), parseAngle.Get(), parsePrecision.Get())

	parseStatsText := ui.UseMemo(func() string {
		parseTokens := countTokens(parseExpression.Get())
		return fmt.Sprintf("%d chars | %d tokens | %d solves", len(strings.TrimSpace(parseExpression.Get())), parseTokens, parseEvaluationCount.Get())
	}, parseExpression.Get(), parseEvaluationCount.Get())

	parseRootClass := "relative min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 text-slate-50"
	parseCardClass := "overflow-hidden rounded-3xl bg-white/[0.02] shadow-2xl backdrop-blur-xl ring-1 ring-white/[0.05]"
	parseHeroCardClass := "overflow-hidden rounded-3xl bg-gradient-to-b from-white/[0.03] to-white/[0.01] shadow-2xl backdrop-blur-xl ring-1 ring-white/10"
	parseTextPrimary := "text-white"
	parseTextSecondary := "text-slate-400"
	parseTextMuted := "text-slate-600"
	parseDisplayBg := "rounded-2xl bg-gradient-to-br from-black/50 to-black/30 p-8 shadow-inner ring-1 ring-black/40"
	parseDisplayResult := "break-all text-5xl font-black tracking-tight text-white sm:text-6xl"
	parseInputClass := "w-full rounded-2xl bg-black/30 px-5 py-4 font-mono text-sm leading-relaxed text-slate-200 shadow-inner placeholder:text-slate-700 outline-none ring-1 ring-white/5 transition-all focus:ring-cyan-500/40"
	parseSelectClass := "cursor-pointer rounded-xl bg-white/5 px-4 py-2 text-xs font-semibold text-slate-300 outline-none transition-all hover:bg-white/10"
	parseLabelClass := "text-xs font-semibold uppercase tracking-wide text-slate-500"
	parseQuickBtnClass := "rounded-xl bg-white/5 px-4 py-3 text-sm font-medium text-slate-300 transition-all hover:bg-cyan-500/20 hover:text-cyan-100 active:scale-95"
	parseKeyBtnAccent := "rounded-2xl bg-cyan-500/15 px-4 py-4 text-lg font-bold text-cyan-200 shadow-lg transition-all hover:bg-cyan-500/25 hover:text-white active:scale-95"
	parseKeyBtnGhost := "rounded-2xl bg-white/5 px-4 py-4 text-sm font-semibold text-slate-400 transition-all hover:bg-white/10 hover:text-slate-200 active:scale-95"
	parseKeyBtnPrimary := "rounded-2xl bg-white/10 px-4 py-4 text-xl font-black text-white transition-all hover:bg-white/15 active:scale-95"
	parseBadgeClass := "rounded-xl bg-white/5 px-4 py-2 text-xs font-medium"
	parseCodeBtnClass := "inline-flex items-center justify-center rounded-xl bg-cyan-500/20 px-4 py-2 text-xs font-semibold text-cyan-200 transition-all hover:bg-cyan-500/30 hover:text-white active:scale-95"

	if parseTheme.Get() == "midnight" {
		parseCardClass = "overflow-hidden rounded-3xl bg-cyan-950/20 shadow-2xl backdrop-blur-xl ring-1 ring-cyan-500/10"
		parseHeroCardClass = "overflow-hidden rounded-3xl bg-gradient-to-b from-cyan-950/30 to-cyan-950/10 shadow-2xl backdrop-blur-xl ring-1 ring-cyan-500/20"
	}

	parseResultValue := parseEvaluation.Formatted
	parseStatusTone := "text-emerald-200"
	parseStatusLabel := "Live preview synced"
	if parseEvaluation.Error != "" {
		parseResultValue = parseLastStableResult.Get()
		parseStatusTone = "text-amber-200"
		parseStatusLabel = parseEvaluation.Error
	}

	parseOpenSource := ui.UseEvent(func() {
		parseOpen := js.Global().Get("openCalculatorSourceModal")
		if parseOpen.Truthy() {
			parseOpen.Invoke()
		}
	})

	parseAppendToken := ui.UseCallback(func(parseToken2 string) {
		parseExpression.Set(parseExpression.Get() + parseToken2)
		parseLastAction.Set("Inserted " + parseToken2)
	}, parseExpression.Get())

	clearExpression := ui.UseCallback(func() {
		parseExpression.Set("")
		parseLastAction.Set("Cleared expression")
	}, nil)

	parseDeleteLast := ui.UseCallback(func() {
		parseCurrent := parseExpression.Get()
		if parseCurrent == "" {
			parseLastAction.Set("Nothing to delete")
			return
		}

		for _, parseToken := range quickDeleteFunctionTokens {
			if strings.HasSuffix(parseCurrent, parseToken) {
				parseExpression.Set(strings.TrimSuffix(parseCurrent, parseToken))
				parseLastAction.Set("Deleted " + parseToken)
				return
			}
		}

		parseRunes := []rune(parseCurrent)
		parseExpression.Set(string(parseRunes[:len(parseRunes)-1]))
		parseLastAction.Set("Deleted last character")
	}, parseExpression.Get())

	parseSolve := ui.UseCallback(func() {
		if parseEvaluation.Error != "" {
			parseLastAction.Set(parseEvaluation.Error)
			return
		}

		parseStamp := time.Now().Format("3:04:05 PM")
		parseEvaluationCount.Update(func(parseV int) int { return parseV + 1 })
		parseMemory.Set(parseEvaluation.Formatted)
		parseLastStableResult.Set(parseEvaluation.Formatted)
		parseLastAction.Set("Solved and stored in memory at " + parseStamp)
	}, parseExpression.Get(), parseEvaluation.Formatted, parseEvaluation.Error)

	parseUpdateExpression := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseExpression.Set(parseEvent.GetValue())
		parseLastAction.Set("Expression updated")
	})

	handleAngle := ui.UseEvent(func(parseEvent2 ui.ChangeEvent) {
		parseAngle.Set(parseEvent2.GetValue())
		parseLastAction.Set("Angle mode switched to " + strings.ToUpper(parseEvent2.GetValue()))
	})

	handleTheme := ui.UseEvent(func(parseEvent3 ui.ChangeEvent) {
		parseNextTheme := normalizeCalculatorTheme(parseEvent3.GetValue())
		parseTheme.Set(parseNextTheme)
		parseLastAction.Set("Theme changed to " + parseNextTheme)
	})

	handleExpressionKey := ui.UseEvent(func(parseEvent4 ui.KeyboardEvent) {
		switch parseEvent4.GetKey() {
		case "Enter":
			parseEvent4.PreventDefault()
			parseSolve()
		case "Escape":
			parseEvent4.PreventDefault()
			clearExpression()
		}
	})

	ui.UseEffect(func() func() {
		parseStorage := js.Global().Get("localStorage")
		if parseStorage.Truthy() {
			if parseSavedExpression := readStorageValue(parseStorage, "calculator-expression"); parseSavedExpression != "" {
				parseExpression.Set(parseSavedExpression)
			}
			if parseSavedPrecision := readStorageValue(parseStorage, "calculator-precision"); parseSavedPrecision != "" {
				if parseParsed, parseErr := strconv.Atoi(parseSavedPrecision); parseErr == nil {
					parsePrecision.Set(parseParsed)
				}
			}
			if parseSavedTheme := readStorageValue(parseStorage, "calculator-theme"); parseSavedTheme != "" {
				parseTheme.Set(normalizeCalculatorTheme(parseSavedTheme))
			}
			if parseSavedAngle := readStorageValue(parseStorage, "calculator-angle"); parseSavedAngle != "" {
				parseAngle.Set(parseSavedAngle)
			}
			if parseSavedMemory := readStorageValue(parseStorage, "calculator-memory"); parseSavedMemory != "" {
				parseMemory.Set(parseSavedMemory)
			}
		}

		parseHydrated.Set(true)
		return nil
	}, "calculator-boot")

	ui.UseEffect(func() func() {
		parseDocument := js.Global().Get("document")
		if parseDocument.Truthy() {
			parseDocument.Set("title", "Signal Calculator - GoWebComponents")
			parseDocumentElement := parseDocument.Get("documentElement")
			if parseDocumentElement.Truthy() {
				parseDocumentElement.Call("setAttribute", "data-calculator-theme", parseTheme.Get())
			}
			parseBody := parseDocument.Get("body")
			if parseBody.Truthy() {
				parseBody.Call("setAttribute", "data-calculator-theme", parseTheme.Get())
			}
		}
		return nil
	}, parseTheme.Get())

	ui.UseEffect(func() func() {
		if parseEvaluation.Error == "" {
			parseLastStableResult.Set(parseEvaluation.Formatted)
		}
		return nil
	}, parseEvaluation.Formatted, parseEvaluation.Error)

	ui.UseEffect(func() func() {
		if !parseHydrated.Get() {
			return nil
		}
		parseStorage2 := js.Global().Get("localStorage")
		if !parseStorage2.Truthy() {
			return nil
		}
		parseStorage2.Call("setItem", "calculator-expression", parseExpression.Get())
		parseStorage2.Call("setItem", "calculator-precision", strconv.Itoa(parsePrecision.Get()))
		parseStorage2.Call("setItem", "calculator-theme", parseTheme.Get())
		parseStorage2.Call("setItem", "calculator-angle", parseAngle.Get())
		parseStorage2.Call("setItem", "calculator-memory", parseMemory.Get())
		return nil
	}, parseHydrated.Get(), parseExpression.Get(), parsePrecision.Get(), parseTheme.Get(), parseAngle.Get(), parseMemory.Get())

	parseQuickInsertNodes := make([]ui.Node, 0, len(quickInsertKeySpecs))
	for _, parseSpec := range quickInsertKeySpecs {
		parseButtonSpec := parseSpec
		parseQuickInsertNodes = append(parseQuickInsertNodes, html.Button(html.Props{
			Type:    "button",
			OnClick: ui.UseEvent(func() { parseAppendToken(parseButtonSpec.Token) }),
			Class:   parseQuickBtnClass,
		}, html.Text(parseButtonSpec.Label)))
	}

	parseKeypadNodes := make([]ui.Node, 0, len(keypadSpecs))
	for _, parseSpec2 := range keypadSpecs {
		parseButtonSpec2 := parseSpec2
		var parseOnPress func()
		switch parseButtonSpec2.Label {
		case "C":
			parseOnPress = clearExpression
		case "DEL":
			parseOnPress = parseDeleteLast
		case "=":
			parseOnPress = parseSolve
		default:
			parseOnPress = func() { parseAppendToken(parseButtonSpec2.Token) }
		}

		// Build button class without display modifiers
		var parseBtnClass string
		switch parseButtonSpec2.Tone {
		case "accent":
			parseBtnClass = parseKeyBtnAccent
		case "ghost":
			parseBtnClass = parseKeyBtnGhost
		case "solve":
			parseBtnClass = "rounded-2xl bg-gradient-to-br from-cyan-500 to-blue-600 px-6 py-4 text-xl font-black text-white shadow-xl transition-all hover:shadow-2xl hover:shadow-cyan-500/50 active:scale-95"
		default:
			parseBtnClass = parseKeyBtnPrimary
		}

		if parseButtonSpec2.Class != "" {
			parseBtnClass += " " + parseButtonSpec2.Class
		}

		parseKeypadNodes = append(parseKeypadNodes, html.Button(html.Props{
			Type:    "button",
			OnClick: ui.UseEvent(parseOnPress),
			Class:   parseBtnClass,
		}, html.Text(parseButtonSpec2.Label)))
	}

	// Calculator Display - The LCD-style result display
	parseDisplaySection := html.Section(html.Props{Class: parseHeroCardClass},
		html.Div(html.Props{Class: "p-8"},
			html.Div(html.Props{Class: "mb-6 flex items-center justify-between"},
				html.Div(html.Props{Class: "flex items-center gap-4"},
					html.H1(html.Props{Class: "text-xl font-bold " + parseTextPrimary}, html.Text("Calculator")),
					html.Div(html.Props{Class: "flex gap-2"},
						html.Select(html.Props{Value: parseTheme.Get(), OnChange: handleTheme, Class: parseSelectClass},
							html.Option(html.Props{Value: "graphite"}, html.Text("Graphite")),
							html.Option(html.Props{Value: "midnight"}, html.Text("Midnight")),
						),
						html.Select(html.Props{Value: parseAngle.Get(), OnChange: handleAngle, Class: parseSelectClass},
							html.Option(html.Props{Value: "deg"}, html.Text("DEG")),
							html.Option(html.Props{Value: "rad"}, html.Text("RAD")),
						),
					),
				),
				html.Button(html.Props{Type: "button", OnClick: parseOpenSource, Class: parseCodeBtnClass}, html.Text("Code")),
			),
			// Main display
			html.Div(html.Props{Class: parseDisplayBg},
				html.Div(html.Props{Class: "mb-4"},
					html.P(html.Props{Class: "break-all font-mono text-sm leading-relaxed " + parseTextMuted}, html.Text(func() string {
						if parseExpression.Get() == "" {
							return "Enter expression"
						}
						return parseExpression.Get()
					}())),
				),
				html.Div(html.Props{Class: "flex items-end justify-between gap-6"},
					html.P(html.Props{Class: parseDisplayResult}, html.Text(parseResultValue)),
					html.Div(html.Props{Class: "text-right"},
						html.P(html.Props{Class: "text-sm font-semibold " + parseStatusTone}, html.Text(parseStatusLabel)),
						html.P(html.Props{Class: "mt-1 text-xs " + parseTextMuted}, html.Text(fmt.Sprintf("%d solves", parseEvaluationCount.Get()))),
					),
				),
			),
			// Quick stats
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				html.Div(html.Props{Class: parseBadgeClass},
					html.Span(html.Props{Class: "text-xs " + parseTextSecondary}, html.Text("Precision: ")),
					html.Span(html.Props{Class: "text-xs font-bold " + parseTextPrimary}, html.Text(strconv.Itoa(parsePrecision.Get()))),
				),
			),
		),
	)

	// Main calculator interface
	parseComposerSection := html.Section(html.Props{Class: parseCardClass},
		html.Div(html.Props{Class: "p-8"},
			html.Div(html.Props{Class: "grid gap-6 lg:grid-cols-2"},
				// Quick insert functions
				html.Div(html.Props{},
					html.Label(html.Props{Class: "mb-3 block " + parseLabelClass}, html.Text("Functions")),
					html.Div(html.Props{Class: "grid grid-cols-2 gap-2", Style: map[string]string{"display": "grid"}}, parseQuickInsertNodes...),
				),
				// Number pad
				html.Div(html.Props{},
					html.Label(html.Props{Class: "mb-3 block " + parseLabelClass}, html.Text("Keypad")),
					html.Div(html.Props{Class: "grid gap-2", Style: map[string]string{
						"display":               "grid",
						"grid-template-columns": "repeat(4, 1fr)",
					}}, parseKeypadNodes...),
				),
			),
			// Expression input
			html.Div(html.Props{Class: "mt-6"},
				html.Label(html.Props{For: parseExpressionID, Class: "mb-3 flex items-center justify-between " + parseLabelClass},
					html.Text("Expression"),
					html.Span(html.Props{Class: "text-xs font-normal normal-case " + parseTextMuted}, html.Text("Enter to solve • Esc to clear")),
				),
				html.Textarea(html.Props{
					ID:          parseExpressionID,
					Value:       parseExpression.Get(),
					Rows:        3,
					OnInput:     parseUpdateExpression,
					OnKeyDown:   handleExpressionKey,
					Placeholder: "sin(45) + sqrt(81)",
					Class:       parseInputClass,
				}),
			),
		),
	)

	// About section
	parseAboutSection := html.Section(html.Props{Class: parseCardClass},
		html.Div(html.Props{Class: "p-8"},
			html.H3(html.Props{Class: "mb-4 " + parseLabelClass}, html.Text("About")),
			html.P(html.Props{Class: "mb-4 text-sm leading-relaxed " + parseTextSecondary}, html.Text("Modern calculator built with GoWebComponents - featuring reactive state, effects, atoms, and composition.")),
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				html.Span(html.Props{Class: parseBadgeClass + " " + parseTextSecondary}, html.Text("Session: "+parseSessionRef.Get())),
				html.Span(html.Props{Class: parseBadgeClass + " " + parseTextSecondary}, html.Text(parseStatsText)),
			),
		),
	)

	parseMainContent := html.Main(html.Props{Class: "mx-auto max-w-4xl space-y-6"}, parseDisplaySection, parseComposerSection, parseAboutSection)

	return html.Div(html.Props{Class: parseRootClass},
		html.Div(html.Props{Class: "pointer-events-none absolute inset-0 bg-gradient-to-br from-cyan-500/5 via-transparent to-blue-500/5"}),
		html.Div(html.Props{Class: "relative z-10 mx-auto max-w-[90rem] px-6 py-8"}, parseMainContent),
	)
}

func countTokens(parseExpression string) int {
	parseCount := 0
	isParseInToken := false
	for _, parseCh := range parseExpression {
		switch {
		case parseCh == ' ' || parseCh == '\t' || parseCh == '\n':
			isParseInToken = false
		case parseCh == '+' || parseCh == '-' || parseCh == '*' || parseCh == '/' || parseCh == '^' || parseCh == '(' || parseCh == ')' || parseCh == ',':
			parseCount++
			isParseInToken = false
		default:
			if !isParseInToken {
				parseCount++
			}
			isParseInToken = true
		}
	}
	return parseCount
}
