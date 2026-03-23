//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func counterToneClass(currentCount int) string {
	switch {
	case currentCount > 0:
		return "border-emerald-400/20 bg-emerald-400/10 text-emerald-100"
	case currentCount < 0:
		return "border-amber-400/20 bg-amber-400/10 text-amber-100"
	default:
		return "border-cyan-300/20 bg-cyan-400/10 text-cyan-100"
	}
}

func counterToneLabel(currentCount int) string {
	switch {
	case currentCount > 0:
		return "Positive"
	case currentCount < 0:
		return "Negative"
	default:
		return "Ready"
	}
}

func CounterExample() ui.Node {
	count := ui.UseState(0)
	currentCount := count.Get()
	embedded := isEmbeddedExample()

	increment := ui.UseEvent(func() {
		count.Update(func(prev int) int { return prev + 1 })
	})

	decrement := ui.UseEvent(func() {
		count.Update(func(prev int) int { return prev - 1 })
	})

	reset := ui.UseEvent(func() {
		count.Set(0)
	})

	toneLabel := counterToneLabel(currentCount)
	toneClass := counterToneClass(currentCount)
	buttonBaseClass := "transition-all duration-200 active:scale-95"
	headerClass := ClassNames(
		"border-b border-white/10",
		When(embedded, "pb-3"),
		When(!embedded, "flex flex-wrap items-center justify-between gap-3 pb-4"),
	)
	iconButtonClass := ClassNames(
		"h-14 w-14 flex items-center justify-center rounded-2xl text-xl font-semibold border",
		buttonBaseClass,
		"hover:-translate-y-0.5",
	)
	neutralButtonClass := ClassNames(
		iconButtonClass,
		"border-white/10 bg-white/[0.05] text-slate-100 hover:bg-white/[0.08]",
	)
	primaryButtonClass := ClassNames(
		iconButtonClass,
		"border-cyan-300/30 bg-cyan-400/15 text-cyan-100 shadow-lg shadow-cyan-950/30 hover:bg-cyan-400/20",
	)
	resetButtonClass := ClassNames(
		"h-14 px-5 flex items-center justify-center rounded-2xl text-xs font-semibold uppercase tracking-[0.18em]",
		buttonBaseClass,
		"border-white/10 bg-slate-950/35 text-slate-200 hover:bg-white/[0.08] hover:text-white",
	)
	containerClass := ClassNames(
		"flex justify-center bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.16),transparent_26%),radial-gradient(circle_at_top_right,rgba(245,158,11,0.10),transparent_20%),linear-gradient(180deg,#020617_0%,#07111f_42%,#0f172a_100%)] text-white",
		When(embedded, "w-full items-start p-3 sm:p-4"),
		When(!embedded, "min-h-screen items-center p-4 sm:p-5"),
	)
	cardClass := ClassNames(
		"w-full rounded-[24px] border border-white/10 bg-white/[0.05] shadow-2xl shadow-black/30 backdrop-blur-xl",
		When(embedded, "max-w-none p-4 sm:p-5"),
		When(!embedded, "max-w-xl p-5 sm:p-6"),
	)

	return Div(Class(containerClass),
		Div(Class(cardClass),
			Div(Class(headerClass),
				Div(Class("space-y-2"),
					Div(Class("inline-flex items-center gap-2 rounded-full border border-cyan-300/25 bg-cyan-400/10 px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em] text-cyan-100"),
						Span(Class("h-2 w-2 rounded-full bg-cyan-300")),
						Text("Embedded example"),
					),
					H2(Class("text-3xl font-semibold tracking-tight text-white"), Text("Counter Example")),
					P(Class("max-w-md text-sm leading-6 text-slate-300"), Text("Reactive state, stable handlers, and direct DOM output from Go in the same design language as example-0.")),
				),
				IfElse(!embedded,
					Div(Class(ClassNames("rounded-full border px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em]", toneClass)), Text(toneLabel)),
					Fragment(),
				),
			),

			Div(Class("mt-5"),
				Div(Class("rounded-[22px] border border-emerald-400/20 bg-emerald-400/10 p-5"),
					Div(Class("text-6xl font-semibold tracking-tight text-white font-mono"), Textf("%d", currentCount)),
					P(Class("mt-2 text-sm text-emerald-50/90"), Text("Current count")),
					IfElse(currentCount == 0,
						P(Class("mt-3 text-xs uppercase tracking-[0.16em] text-emerald-200"), Text("Counter is centered")),
						P(Class("mt-3 text-xs uppercase tracking-[0.16em] text-emerald-100/80"), Textf("Offset from zero: %d", currentCount)),
					),
					Div(Class("mt-5 flex flex-wrap gap-2"),
						Button(OnClick(decrement), Class(neutralButtonClass), Text("-")),
						Button(OnClick(increment), Class(primaryButtonClass), Text("+")),
						Button(OnClick(reset), Class(resetButtonClass), Text("Reset")),
					),
				),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(CounterExample), resolveMountSelector())
	utils.WaitForever()
}

func resolveMountSelector() string {
	return interop.SharedWindowEnv().String("__gwcExampleMountSelector", "#app")
}

func isEmbeddedExample() bool {
	_, ok := interop.SharedWindowEnv().LookupString("__gwcExampleMountSelector")
	return ok
}
