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

func readStorageValue(storage js.Value, key string) string {
	if !storage.Truthy() {
		return ""
	}

	value := storage.Call("getItem", key)
	if value.IsUndefined() || value.IsNull() {
		return ""
	}

	return strings.TrimSpace(value.String())
}

func App() ui.Node {
	theme := state.UseAtom(calculatorThemeAtom, "graphite")
	angle := state.UseAtom(calculatorAngleAtom, "deg")
	memory := state.UseAtom(calculatorMemoryAtom, "0")

	expression := ui.UseState("sin(45)+pow(2,5)")
	precision := ui.UseState(4)
	lastAction := ui.UseState("Ready for input")
	evaluationCount := ui.UseState(0)
	hydrated := ui.UseState(false)

	sessionRef := ui.UseRef("CALC-" + time.Now().Format("150405"))
	lastStableResult := ui.UseRef("0")
	expressionID := ui.UseId()

	evaluation := ui.UseMemo(func() Evaluation {
		return EvaluateExpression(expression.Get(), angle.Get(), precision.Get())
	}, expression.Get(), angle.Get(), precision.Get())

	statsText := ui.UseMemo(func() string {
		tokens := countTokens(expression.Get())
		return fmt.Sprintf("%d chars | %d tokens | %d solves", len(strings.TrimSpace(expression.Get())), tokens, evaluationCount.Get())
	}, expression.Get(), evaluationCount.Get())

	rootClass := "relative min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 text-slate-50"
	cardClass := "overflow-hidden rounded-3xl bg-white/[0.02] shadow-2xl backdrop-blur-xl ring-1 ring-white/[0.05]"
	heroCardClass := "overflow-hidden rounded-3xl bg-gradient-to-b from-white/[0.03] to-white/[0.01] shadow-2xl backdrop-blur-xl ring-1 ring-white/10"
	textPrimary := "text-white"
	textSecondary := "text-slate-400"
	textMuted := "text-slate-600"
	displayBg := "rounded-2xl bg-gradient-to-br from-black/50 to-black/30 p-8 shadow-inner ring-1 ring-black/40"
	displayResult := "break-all text-5xl font-black tracking-tight text-white sm:text-6xl"
	inputClass := "w-full rounded-2xl bg-black/30 px-5 py-4 font-mono text-sm leading-relaxed text-slate-200 shadow-inner placeholder:text-slate-700 outline-none ring-1 ring-white/5 transition-all focus:ring-cyan-500/40"
	selectClass := "cursor-pointer rounded-xl bg-white/5 px-4 py-2 text-xs font-semibold text-slate-300 outline-none transition-all hover:bg-white/10"
	labelClass := "text-xs font-semibold uppercase tracking-wide text-slate-500"
	quickBtnClass := "rounded-xl bg-white/5 px-4 py-3 text-sm font-medium text-slate-300 transition-all hover:bg-cyan-500/20 hover:text-cyan-100 active:scale-95"
	keyBtnAccent := "rounded-2xl bg-cyan-500/15 px-4 py-4 text-lg font-bold text-cyan-200 shadow-lg transition-all hover:bg-cyan-500/25 hover:text-white active:scale-95"
	keyBtnGhost := "rounded-2xl bg-white/5 px-4 py-4 text-sm font-semibold text-slate-400 transition-all hover:bg-white/10 hover:text-slate-200 active:scale-95"
	keyBtnPrimary := "rounded-2xl bg-white/10 px-4 py-4 text-xl font-black text-white transition-all hover:bg-white/15 active:scale-95"
	badgeClass := "rounded-xl bg-white/5 px-4 py-2 text-xs font-medium"
	codeBtnClass := "inline-flex items-center justify-center rounded-xl bg-cyan-500/20 px-4 py-2 text-xs font-semibold text-cyan-200 transition-all hover:bg-cyan-500/30 hover:text-white active:scale-95"

	if theme.Get() == "midnight" {
		cardClass = "overflow-hidden rounded-3xl bg-cyan-950/20 shadow-2xl backdrop-blur-xl ring-1 ring-cyan-500/10"
		heroCardClass = "overflow-hidden rounded-3xl bg-gradient-to-b from-cyan-950/30 to-cyan-950/10 shadow-2xl backdrop-blur-xl ring-1 ring-cyan-500/20"
	} else if theme.Get() == "light" {
		rootClass = "relative min-h-screen bg-gradient-to-br from-slate-50 via-slate-100 to-slate-50 text-slate-900"
		cardClass = "overflow-hidden rounded-3xl bg-white shadow-2xl ring-1 ring-slate-200"
		heroCardClass = "overflow-hidden rounded-3xl bg-gradient-to-b from-white to-slate-50 shadow-2xl ring-1 ring-slate-200"
		textPrimary = "text-slate-900"
		textSecondary = "text-slate-600"
		textMuted = "text-slate-500"
		displayBg = "rounded-2xl bg-slate-100 p-8 ring-1 ring-slate-200"
		displayResult = "break-all text-5xl font-black tracking-tight text-slate-900 sm:text-6xl"
		inputClass = "w-full rounded-2xl bg-white px-5 py-4 font-mono text-sm leading-relaxed text-slate-900 ring-1 ring-slate-200 transition-all focus:ring-cyan-500/50 outline-none placeholder:text-slate-400"
		selectClass = "cursor-pointer rounded-xl bg-slate-100 px-4 py-2 text-xs font-semibold text-slate-700 outline-none transition-all hover:bg-slate-200 ring-1 ring-slate-200"
		labelClass = "text-xs font-semibold uppercase tracking-wide text-slate-500"
		quickBtnClass = "rounded-xl bg-slate-100 px-4 py-3 text-sm font-medium text-slate-700 transition-all hover:bg-cyan-100 hover:text-cyan-700 active:scale-95 ring-1 ring-slate-200"
		keyBtnAccent = "rounded-2xl bg-cyan-500/20 px-4 py-4 text-lg font-bold text-cyan-700 shadow-sm transition-all hover:bg-cyan-500/30 hover:text-cyan-800 active:scale-95"
		keyBtnGhost = "rounded-2xl bg-slate-100 px-4 py-4 text-sm font-semibold text-slate-600 transition-all hover:bg-slate-200 hover:text-slate-800 active:scale-95 ring-1 ring-slate-200"
		keyBtnPrimary = "rounded-2xl bg-white px-4 py-4 text-xl font-black text-slate-900 shadow-sm transition-all hover:bg-slate-50 active:scale-95 ring-1 ring-slate-200"
		badgeClass = "rounded-xl bg-slate-100 px-4 py-2 text-xs font-medium text-slate-600 ring-1 ring-slate-200"
		codeBtnClass = "inline-flex items-center justify-center rounded-xl bg-cyan-500/20 px-4 py-2 text-xs font-semibold text-cyan-700 transition-all hover:bg-cyan-500/30 hover:text-cyan-800 active:scale-95"
	}

	resultValue := evaluation.Formatted
	statusTone := "text-emerald-200"
	statusLabel := "Live preview synced"
	if evaluation.Error != "" {
		resultValue = lastStableResult.Get()
		statusTone = "text-amber-200"
		statusLabel = evaluation.Error
	}

	if theme.Get() == "light" {
		if evaluation.Error != "" {
			statusTone = "text-amber-600"
		} else {
			statusTone = "text-emerald-600"
		}
	}

	openSource := ui.UseEvent(func() {
		open := js.Global().Get("openCalculatorSourceModal")
		if open.Truthy() {
			open.Invoke()
		}
	})

	appendToken := ui.UseCallback(func(token string) {
		expression.Set(expression.Get() + token)
		lastAction.Set("Inserted " + token)
	}, expression.Get())

	clearExpression := ui.UseCallback(func() {
		expression.Set("")
		lastAction.Set("Cleared expression")
	}, nil)

	deleteLast := ui.UseCallback(func() {
		current := expression.Get()
		if current == "" {
			lastAction.Set("Nothing to delete")
			return
		}

		for _, token := range quickDeleteFunctionTokens {
			if strings.HasSuffix(current, token) {
				expression.Set(strings.TrimSuffix(current, token))
				lastAction.Set("Deleted " + token)
				return
			}
		}

		runes := []rune(current)
		expression.Set(string(runes[:len(runes)-1]))
		lastAction.Set("Deleted last character")
	}, expression.Get())

	solve := ui.UseCallback(func() {
		if evaluation.Error != "" {
			lastAction.Set(evaluation.Error)
			return
		}

		stamp := time.Now().Format("3:04:05 PM")
		evaluationCount.Update(func(v int) int { return v + 1 })
		memory.Set(evaluation.Formatted)
		lastStableResult.Set(evaluation.Formatted)
		lastAction.Set("Solved and stored in memory at " + stamp)
	}, expression.Get(), evaluation.Formatted, evaluation.Error)

	updateExpression := ui.UseEvent(func(event ui.InputEvent) {
		expression.Set(event.GetValue())
		lastAction.Set("Expression updated")
	})

	handleAngle := ui.UseEvent(func(event ui.ChangeEvent) {
		angle.Set(event.GetValue())
		lastAction.Set("Angle mode switched to " + strings.ToUpper(event.GetValue()))
	})

	handleTheme := ui.UseEvent(func(event ui.ChangeEvent) {
		theme.Set(event.GetValue())
		lastAction.Set("Theme changed to " + event.GetValue())
	})

	handleExpressionKey := ui.UseEvent(func(event ui.KeyboardEvent) {
		switch event.GetKey() {
		case "Enter":
			event.PreventDefault()
			solve()
		case "Escape":
			event.PreventDefault()
			clearExpression()
		}
	})

	ui.UseEffect(func() func() {
		storage := js.Global().Get("localStorage")
		if storage.Truthy() {
			if savedExpression := readStorageValue(storage, "calculator-expression"); savedExpression != "" {
				expression.Set(savedExpression)
			}
			if savedPrecision := readStorageValue(storage, "calculator-precision"); savedPrecision != "" {
				if parsed, err := strconv.Atoi(savedPrecision); err == nil {
					precision.Set(parsed)
				}
			}
			if savedTheme := readStorageValue(storage, "calculator-theme"); savedTheme != "" {
				theme.Set(savedTheme)
			}
			if savedAngle := readStorageValue(storage, "calculator-angle"); savedAngle != "" {
				angle.Set(savedAngle)
			}
			if savedMemory := readStorageValue(storage, "calculator-memory"); savedMemory != "" {
				memory.Set(savedMemory)
			}
		}

		hydrated.Set(true)
		return nil
	}, "calculator-boot")

	ui.UseEffect(func() func() {
		document := js.Global().Get("document")
		if document.Truthy() {
			document.Set("title", "Signal Calculator - GoWebComponents")
			documentElement := document.Get("documentElement")
			if documentElement.Truthy() {
				documentElement.Call("setAttribute", "data-calculator-theme", theme.Get())
			}
			body := document.Get("body")
			if body.Truthy() {
				body.Call("setAttribute", "data-calculator-theme", theme.Get())
			}
		}
		return nil
	}, theme.Get())

	ui.UseEffect(func() func() {
		if evaluation.Error == "" {
			lastStableResult.Set(evaluation.Formatted)
		}
		return nil
	}, evaluation.Formatted, evaluation.Error)

	ui.UseEffect(func() func() {
		if !hydrated.Get() {
			return nil
		}
		storage := js.Global().Get("localStorage")
		if !storage.Truthy() {
			return nil
		}
		storage.Call("setItem", "calculator-expression", expression.Get())
		storage.Call("setItem", "calculator-precision", strconv.Itoa(precision.Get()))
		storage.Call("setItem", "calculator-theme", theme.Get())
		storage.Call("setItem", "calculator-angle", angle.Get())
		storage.Call("setItem", "calculator-memory", memory.Get())
		return nil
	}, hydrated.Get(), expression.Get(), precision.Get(), theme.Get(), angle.Get(), memory.Get())

	quickInsertNodes := make([]ui.Node, 0, len(quickInsertKeySpecs))
	for _, spec := range quickInsertKeySpecs {
		buttonSpec := spec
		quickInsertNodes = append(quickInsertNodes, html.Button(html.Props{
			Type:    "button",
			OnClick: ui.UseEvent(func() { appendToken(buttonSpec.Token) }),
			Class:   quickBtnClass,
		}, html.Text(buttonSpec.Label)))
	}

	keypadNodes := make([]ui.Node, 0, len(keypadSpecs))
	for _, spec := range keypadSpecs {
		buttonSpec := spec
		var onPress func()
		switch buttonSpec.Label {
		case "C":
			onPress = clearExpression
		case "DEL":
			onPress = deleteLast
		case "=":
			onPress = solve
		default:
			onPress = func() { appendToken(buttonSpec.Token) }
		}

		// Build button class without display modifiers
		var btnClass string
		switch buttonSpec.Tone {
		case "accent":
			btnClass = keyBtnAccent
		case "ghost":
			btnClass = keyBtnGhost
		case "solve":
			btnClass = "rounded-2xl bg-gradient-to-br from-cyan-500 to-blue-600 px-6 py-4 text-xl font-black text-white shadow-xl transition-all hover:shadow-2xl hover:shadow-cyan-500/50 active:scale-95"
		default:
			btnClass = keyBtnPrimary
		}

		if buttonSpec.Class != "" {
			btnClass += " " + buttonSpec.Class
		}

		keypadNodes = append(keypadNodes, html.Button(html.Props{
			Type:    "button",
			OnClick: ui.UseEvent(onPress),
			Class:   btnClass,
		}, html.Text(buttonSpec.Label)))
	}

	// Calculator Display - The LCD-style result display
	displaySection := html.Section(html.Props{Class: heroCardClass},
		html.Div(html.Props{Class: "p-8"},
			html.Div(html.Props{Class: "mb-6 flex items-center justify-between"},
				html.Div(html.Props{Class: "flex items-center gap-4"},
					html.H1(html.Props{Class: "text-xl font-bold " + textPrimary}, html.Text("Calculator")),
					html.Div(html.Props{Class: "flex gap-2"},
						html.Select(html.Props{Value: theme.Get(), OnChange: handleTheme, Class: selectClass},
							html.Option(html.Props{Value: "graphite"}, html.Text("Graphite")),
							html.Option(html.Props{Value: "midnight"}, html.Text("Midnight")),
							html.Option(html.Props{Value: "light"}, html.Text("Light")),
						),
						html.Select(html.Props{Value: angle.Get(), OnChange: handleAngle, Class: selectClass},
							html.Option(html.Props{Value: "deg"}, html.Text("DEG")),
							html.Option(html.Props{Value: "rad"}, html.Text("RAD")),
						),
					),
				),
				html.Button(html.Props{Type: "button", OnClick: openSource, Class: codeBtnClass}, html.Text("Code")),
			),
			// Main display
			html.Div(html.Props{Class: displayBg},
				html.Div(html.Props{Class: "mb-4"},
					html.P(html.Props{Class: "break-all font-mono text-sm leading-relaxed " + textMuted}, html.Text(func() string {
						if expression.Get() == "" {
							return "Enter expression"
						}
						return expression.Get()
					}())),
				),
				html.Div(html.Props{Class: "flex items-end justify-between gap-6"},
					html.P(html.Props{Class: displayResult}, html.Text(resultValue)),
					html.Div(html.Props{Class: "text-right"},
						html.P(html.Props{Class: "text-sm font-semibold " + statusTone}, html.Text(statusLabel)),
						html.P(html.Props{Class: "mt-1 text-xs " + textMuted}, html.Text(fmt.Sprintf("%d solves", evaluationCount.Get()))),
					),
				),
			),
			// Quick stats
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				html.Div(html.Props{Class: badgeClass},
					html.Span(html.Props{Class: "text-xs " + textSecondary}, html.Text("Precision: ")),
					html.Span(html.Props{Class: "text-xs font-bold " + textPrimary}, html.Text(strconv.Itoa(precision.Get()))),
				),
			),
		),
	)

	// Main calculator interface
	composerSection := html.Section(html.Props{Class: cardClass},
		html.Div(html.Props{Class: "p-8"},
			html.Div(html.Props{Class: "grid gap-6 lg:grid-cols-2"},
				// Quick insert functions
				html.Div(html.Props{},
					html.Label(html.Props{Class: "mb-3 block " + labelClass}, html.Text("Functions")),
					html.Div(html.Props{Class: "grid grid-cols-2 gap-2", Style: map[string]string{"display": "grid"}}, quickInsertNodes...),
				),
				// Number pad
				html.Div(html.Props{},
					html.Label(html.Props{Class: "mb-3 block " + labelClass}, html.Text("Keypad")),
					html.Div(html.Props{Class: "grid gap-2", Style: map[string]string{
						"display":               "grid",
						"grid-template-columns": "repeat(4, 1fr)",
					}}, keypadNodes...),
				),
			),
			// Expression input
			html.Div(html.Props{Class: "mt-6"},
				html.Label(html.Props{For: expressionID, Class: "mb-3 flex items-center justify-between " + labelClass},
					html.Text("Expression"),
					html.Span(html.Props{Class: "text-xs font-normal normal-case " + textMuted}, html.Text("Enter to solve • Esc to clear")),
				),
				html.Textarea(html.Props{
					ID:          expressionID,
					Value:       expression.Get(),
					Rows:        3,
					OnInput:     updateExpression,
					OnKeyDown:   handleExpressionKey,
					Placeholder: "sin(45) + sqrt(81)",
					Class:       inputClass,
				}),
			),
		),
	)

	// About section
	aboutSection := html.Section(html.Props{Class: cardClass},
		html.Div(html.Props{Class: "p-8"},
			html.H3(html.Props{Class: "mb-4 " + labelClass}, html.Text("About")),
			html.P(html.Props{Class: "mb-4 text-sm leading-relaxed " + textSecondary}, html.Text("Modern calculator built with GoWebComponents - featuring reactive state, effects, atoms, and composition.")),
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				html.Span(html.Props{Class: badgeClass + " " + textSecondary}, html.Text("Session: "+sessionRef.Get())),
				html.Span(html.Props{Class: badgeClass + " " + textSecondary}, html.Text(statsText)),
			),
		),
	)

	mainContent := html.Main(html.Props{Class: "mx-auto max-w-4xl space-y-6"}, displaySection, composerSection, aboutSection)

	return html.Div(html.Props{Class: rootClass},
		html.Div(html.Props{Class: "pointer-events-none absolute inset-0 bg-gradient-to-br from-cyan-500/5 via-transparent to-blue-500/5"}),
		html.Div(html.Props{Class: "relative z-10 mx-auto max-w-[90rem] px-6 py-8"}, mainContent),
	)
}

func countTokens(expression string) int {
	count := 0
	inToken := false
	for _, ch := range expression {
		switch {
		case ch == ' ' || ch == '\t' || ch == '\n':
			inToken = false
		case ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '^' || ch == '(' || ch == ')' || ch == ',':
			count++
			inToken = false
		default:
			if !inToken {
				count++
			}
			inToken = true
		}
	}
	return count
}
