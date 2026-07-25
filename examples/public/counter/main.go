//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

func counterToneClass(parseCurrentCount int) string {
	switch {
	case parseCurrentCount > 0:
		return "border-emerald-400/20 bg-emerald-400/10 text-emerald-100"
	case parseCurrentCount < 0:
		return "border-amber-400/20 bg-amber-400/10 text-amber-100"
	default:
		return "border-cyan-300/20 bg-cyan-400/10 text-cyan-100"
	}
}

func counterToneLabel(parseCurrentCount int) string {
	switch {
	case parseCurrentCount > 0:
		return "Positive"
	case parseCurrentCount < 0:
		return "Negative"
	default:
		return "Ready"
	}
}

func CounterExample() ui.Node {
	parseCount := ui.UseState(0)
	parseCurrentCount := parseCount.Get()
	parseEmbedded := isEmbeddedExample()

	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
	})

	parseDecrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrev2 int) int { return parsePrev2 - 1 })
	})

	reset := ui.UseEvent(func() {
		parseCount.Set(0)
	})

	parseToneLabel := counterToneLabel(parseCurrentCount)
	parseToneClass := counterToneClass(parseCurrentCount)
	parseExampleModeLabel := "Standalone example"
	if parseEmbedded {
		parseExampleModeLabel = "Embedded example"
	}
	parseButtonBaseClass := "transition-all duration-200 active:scale-95"
	parseHeaderClass := ClassNames(
		"border-b border-white/10",
		When(parseEmbedded, "pb-3"),
		When(!parseEmbedded, "flex flex-wrap items-center justify-between gap-3 pb-4"),
	)
	parseIconButtonClass := ClassNames(
		"h-14 w-14 flex items-center justify-center rounded-2xl text-xl font-semibold border",
		parseButtonBaseClass,
		"hover:-translate-y-0.5",
	)
	parseNeutralButtonClass := ClassNames(
		parseIconButtonClass,
		"border-white/10 bg-white/[0.05] text-slate-100 hover:bg-white/[0.08]",
	)
	parsePrimaryButtonClass := ClassNames(
		parseIconButtonClass,
		"border-cyan-300/30 bg-cyan-400/15 text-cyan-100 shadow-lg shadow-cyan-950/30 hover:bg-cyan-400/20",
	)
	resetButtonClass := ClassNames(
		"h-14 px-5 flex items-center justify-center rounded-2xl text-xs font-semibold uppercase tracking-[0.18em]",
		parseButtonBaseClass,
		"border-white/10 bg-slate-950/35 text-slate-200 hover:bg-white/[0.08] hover:text-white",
	)
	parseContainerClass := ClassNames(
		"box-border flex justify-center bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.16),transparent_26%),radial-gradient(circle_at_top_right,rgba(245,158,11,0.10),transparent_20%),linear-gradient(180deg,#020617_0%,#07111f_42%,#0f172a_100%)] text-white",
		When(parseEmbedded, "w-full items-start p-3 sm:p-4"),
		When(!parseEmbedded, "min-h-screen items-center p-4 sm:p-5"),
	)
	parseCardClass := ClassNames(
		"box-border w-full rounded-[24px] border border-white/10 bg-white/[0.05] shadow-2xl shadow-black/30 backdrop-blur-xl",
		When(parseEmbedded, "max-w-5xl p-4 sm:p-5"),
		When(!parseEmbedded, "max-w-xl p-5 sm:p-6"),
	)

	return Div(ClassStr(parseContainerClass),
		Div(ClassStr(parseCardClass),
			Div(ClassStr(parseHeaderClass),
				Div(ClassStr("space-y-2"),
					Div(ClassStr("inline-flex items-center gap-2 rounded-full border border-cyan-300/25 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em] text-cyan-100"),
						Span(ClassStr("h-2 w-2 rounded-full bg-cyan-300")),
						Text(parseExampleModeLabel),
					),
					H2(ClassStr("text-3xl font-semibold tracking-tight text-white"), Text("Counter Example")),
					P(ClassStr("max-w-md text-sm leading-6 text-slate-300"), Text("Reactive state, stable handlers, and direct DOM output from Go in the same design language as public-examples-site.")),
				),
				IfElse(!parseEmbedded,
					Div(ClassStr(ClassNames("rounded-full border px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em]", parseToneClass)), Text(parseToneLabel)),
					Fragment(),
				),
			),

			Div(ClassStr("mt-5"),
				Div(ClassStr("rounded-[22px] border border-emerald-400/20 bg-emerald-400/10 p-5"),
					Div(ClassStr("text-6xl font-semibold tracking-tight text-white font-mono"), Textf("%d", parseCurrentCount)),
					P(ClassStr("mt-2 text-sm text-emerald-50/90"), Text("Current count")),
					IfElse(parseCurrentCount == 0,
						P(ClassStr("mt-3 text-xs uppercase tracking-[0.16em] text-emerald-200"), Text("Counter is centered")),
						P(ClassStr("mt-3 text-xs uppercase tracking-[0.16em] text-emerald-100/80"), Textf("Offset from zero: %d", parseCurrentCount)),
					),
					Div(ClassStr("mt-5 flex flex-wrap gap-2"),
						Button(OnClick(parseDecrement), ClassStr(parseNeutralButtonClass), Text("-")),
						Button(OnClick(parseIncrement), ClassStr(parsePrimaryButtonClass), Text("+")),
						Button(OnClick(reset), ClassStr(resetButtonClass), Text("Reset")),
					),
				),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(CounterExample))
	exampleboot.WaitExampleRuntime()
}

func isEmbeddedExample() bool {
	return exampleboot.HasExampleMountSelector()
}
