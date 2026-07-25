// Bill Splitter — GoWebComponents implementation (the study's anchor).
//
// Exercises a broad slice of the framework surface against research/framework-bench/SPEC.md:
//   - html/shorthand tags + control flow (Map, If)
//   - typed css/u utilities for the whole design (no class strings, no Tailwind toolchain)
//   - ui hooks: UseState (bill/tip/people), UseEvent (handlers), CreateElement (composition)
//   - state atoms for SHARED state (theme + roundUp) consumed by several components
//   - ui.Run one-line browser entrypoint
package main

import (
	"math"
	"strconv"

	"github.com/monstercameron/GoWebComponents/v5/css/u"
	"github.com/monstercameron/GoWebComponents/v5/html"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/state"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// --- shared (global) state: theme + roundUp, keyed atoms shared across components ---

const (
	themeAtomID   = "bs.theme"
	roundUpAtomID = "bs.roundUp"
)

func useTheme() state.Atom[string] { return state.UseAtom(themeAtomID, "light") }
func useRoundUp() state.Atom[bool] { return state.UseAtom(roundUpAtomID, false) }

// --- design tokens (SPEC) reproduced through typed css/u color tokens ---

type palette struct {
	bg, surface, text, muted, border, accent, accentText u.Color
}

func paletteFor(theme string) palette {
	if theme == "dark" {
		return palette{
			bg: u.Slate900, surface: u.Slate800, text: u.Slate100,
			muted: u.Slate400, border: u.Slate700, accent: u.Sky400, accentText: u.Slate900,
		}
	}
	return palette{
		bg: u.Slate50, surface: u.White, text: u.Slate900,
		muted: u.Slate500, border: u.Slate200, accent: u.Sky500, accentText: u.White,
	}
}

var presets = []int{10, 15, 18, 20, 25}

// --- root component: owns local state, derives values, composes children ---

func App() ui.Node {
	bill := ui.UseState(0.0)
	tip := ui.UseState(18.0)
	people := ui.UseState(1)

	theme := useTheme().Get()
	round := useRoundUp().Get()
	pal := paletteFor(theme)

	tipAmount := bill.Get() * tip.Get() / 100
	total := bill.Get() + tipAmount
	perPersonRaw := 0.0
	if people.Get() > 0 {
		perPersonRaw = total / float64(people.Get())
	}
	perPerson := perPersonRaw
	if round {
		perPerson = math.Ceil(perPersonRaw)
	}

	return Main(
		u.Class(u.Flex, u.FlexCol, u.Gap(u.Spacing4), u.Bg(pal.bg), u.Fg(pal.text), u.PadX(u.Spacing4), u.PadY(u.Spacing6)),
		ui.CreateElement(HeaderBar, headerProps{Pal: pal}),
		ui.CreateElement(Inputs, inputsProps{Bill: bill, Tip: tip, People: people, Pal: pal}),
		ui.CreateElement(Results, resultsProps{Bill: bill.Get(), Tip: tip.Get(), People: people.Get(), Pal: pal}),
		ui.CreateElement(Breakdown, breakdownProps{PerPerson: perPerson, People: people.Get(), Pal: pal}),
		ui.CreateElement(FooterBar, footerProps{Total: total, People: people.Get(), Theme: theme, Pal: pal}),
	)
}

// --- header: theme + round-up toggles writing shared atoms ---

type headerProps struct{ Pal palette }

func HeaderBar(props headerProps) ui.Node {
	themeAtom := useTheme()
	roundAtom := useRoundUp()
	dark := themeAtom.Get() == "dark"
	round := roundAtom.Get()

	toggleTheme := ui.UseEvent(func() {
		themeAtom.Update(func(t string) string {
			if t == "dark" {
				return "light"
			}
			return "dark"
		})
	})
	toggleRound := ui.UseEvent(func() { roundAtom.Update(func(b bool) bool { return !b }) })

	return Header(
		u.Class(u.Flex, u.ItemsCenter, u.JustifyBetween, u.Gap(u.Spacing3)),
		H1(u.Class(u.TextSize(u.TextXl), u.FontBold, u.Fg(props.Pal.text)), "Bill Splitter"),
		Div(
			u.Class(u.Flex, u.Gap(u.Spacing2)),
			toggleButton("Round up", round, props.Pal, toggleRound),
			toggleButton("Dark", dark, props.Pal, toggleTheme),
		),
	)
}

// toggleButton is a small inline helper (no hooks) so it can be called inline; the
// click handler is created by the owning Header component and passed in.
func toggleButton(label string, active bool, pal palette, onClick ui.Handler) ui.Node {
	bg, fg, border := pal.surface, pal.text, pal.border
	if active {
		bg, fg, border = pal.accent, pal.accentText, pal.accent
	}
	return Button(
		Type("button"),
		Aria("pressed", boolStr(active)),
		OnClick(onClick),
		u.Class(u.Rounded(u.RadiusLg), u.PadX(u.Spacing3), u.PadY(u.Spacing1), u.TextSize(u.TextSm), u.FontSemibold, u.Bg(bg), u.Fg(fg), u.Border(border)),
		label,
	)
}

// --- inputs: bill, tip presets (keyed list of PresetButton), custom tip, stepper ---

type inputsProps struct {
	Bill   ui.State[float64]
	Tip    ui.State[float64]
	People ui.State[int]
	Pal    palette
}

func Inputs(props inputsProps) ui.Node {
	pal := props.Pal

	onBill := ui.UseEvent(func(e ui.InputEvent) { props.Bill.Set(parseNum(e.GetValue())) })
	onCustomTip := ui.UseEvent(func(e ui.InputEvent) { props.Tip.Set(parseNum(e.GetValue())) })
	dec := ui.UseEvent(func() {
		props.People.Update(func(n int) int {
			if n > 1 {
				return n - 1
			}
			return 1
		})
	})
	inc := ui.UseEvent(func() { props.People.Update(func(n int) int { return n + 1 }) })

	tipVal := props.Tip.Get()

	return Section(
		card(pal),
		// Bill amount
		Label(
			field(),
			Span(labelClass(pal), "Bill amount"),
			Div(
				u.Class(u.Flex, u.ItemsCenter, u.Gap(u.Spacing1), u.Rounded(u.RadiusLg), u.Border(pal.border), u.Bg(pal.bg), u.PadX(u.Spacing3), u.PadY(u.Spacing2)),
				Span(u.Class(u.Fg(pal.muted), u.FontSemibold), "$"),
				Input(Type("number"), Min("0"), Step("0.01"), Value(numStr(props.Bill.Get())), OnInput(onBill),
					u.Class(u.WFull, u.Bg(pal.bg), u.Fg(pal.text))),
			),
		),
		// Tip
		Div(
			field(),
			Span(labelClass(pal), "Tip"),
			Div(
				u.Class(u.Flex, u.Gap(u.Spacing2)),
				Map(presets, func(p int) ui.Node {
					return ui.CreateElement(PresetButton, presetProps{Percent: p, Active: tipVal == float64(p), Pal: pal, Tip: props.Tip})
				}),
				Input(Type("number"), Min("0"), Placeholder("Custom %"), Value(numStr(tipVal)), OnInput(onCustomTip),
					u.Class(u.Rounded(u.RadiusLg), u.Border(pal.border), u.Bg(pal.bg), u.Fg(pal.text), u.PadX(u.Spacing3), u.PadY(u.Spacing2), u.TextSize(u.TextSm))),
			),
		),
		// People
		Div(
			field(),
			Span(labelClass(pal), "People"),
			Div(
				u.Class(u.Flex, u.ItemsCenter, u.Gap(u.Spacing3)),
				Button(Type("button"), Aria("label", "Fewer people"), DisabledIf(props.People.Get() <= 1), OnClick(dec), stepClass(pal), "−"),
				Span(u.Class(u.TextSize(u.TextLg), u.FontBold), strconv.Itoa(props.People.Get())),
				Button(Type("button"), Aria("label", "More people"), OnClick(inc), stepClass(pal), "+"),
			),
		),
	)
}

// --- preset button: its own component so the OnClick hook is at a stable position ---

type presetProps struct {
	Percent int
	Active  bool
	Pal     palette
	Tip     ui.State[float64]
}

func PresetButton(props presetProps) ui.Node {
	onSelect := ui.UseEvent(func() { props.Tip.Set(float64(props.Percent)) })
	bg, fg, border := props.Pal.bg, props.Pal.text, props.Pal.border
	if props.Active {
		bg, fg, border = props.Pal.accent, props.Pal.accentText, props.Pal.accent
	}
	return Button(
		Type("button"),
		OnClick(onSelect),
		u.Class(u.Rounded(u.RadiusLg), u.Border(border), u.Bg(bg), u.Fg(fg), u.PadX(u.Spacing3), u.PadY(u.Spacing2), u.TextSize(u.TextSm), u.FontSemibold),
		strconv.Itoa(props.Percent)+"%",
	)
}

// --- results: derived display + rounding note (reads shared roundUp) ---

type resultsProps struct {
	Bill, Tip float64
	People    int
	Pal       palette
}

func Results(props resultsProps) ui.Node {
	round := useRoundUp().Get()
	pal := props.Pal

	tipAmount := props.Bill * props.Tip / 100
	total := props.Bill + tipAmount
	perPersonRaw := 0.0
	if props.People > 0 {
		perPersonRaw = total / float64(props.People)
	}
	perPerson := perPersonRaw
	totalCollected := total
	if round {
		perPerson = math.Ceil(perPersonRaw)
		totalCollected = perPerson * float64(props.People)
	}
	roundingExtra := math.Max(0, totalCollected-total)
	effTip := props.Tip
	if props.Bill > 0 {
		effTip = (totalCollected - props.Bill) / props.Bill * 100
	}

	return Section(
		card(pal),
		resultRow(pal, "Tip", usd(tipAmount)),
		resultRow(pal, "Total", usd(total)),
		Div(
			u.Class(u.Flex, u.ItemsCenter, u.JustifyBetween),
			Span(u.Class(u.TextSize(u.TextSm), u.Fg(pal.muted)), "Per person"),
			Span(u.Class(u.TextSize(u.Text2xl), u.FontBold, u.Fg(pal.accent)), usd(perPerson)),
		),
		If(round && roundingExtra > 0,
			P(u.Class(u.TextSize(u.TextXs), u.Fg(pal.muted)),
				"Rounding up collects "+usd(roundingExtra)+" extra · effective tip "+strconv.FormatFloat(effTip, 'f', 1, 64)+"%"),
		),
		If(props.Bill <= 0,
			P(u.Class(u.TextSize(u.TextSm), u.Fg(pal.muted)), "Enter a bill amount to begin."),
		),
	)
}

// --- breakdown: keyed per-person list ---

type breakdownProps struct {
	PerPerson float64
	People    int
	Pal       palette
}

func Breakdown(props breakdownProps) ui.Node {
	pal := props.Pal
	rows := make([]int, props.People)
	for i := range rows {
		rows[i] = i + 1
	}
	return Section(
		card(pal),
		H2(u.Class(u.TextSize(u.TextSm), u.FontBold), "Per-person breakdown"),
		Ul(
			u.Class(u.Flex, u.FlexCol, u.Gap(u.Spacing2)),
			MapKeyed(rows, func(n int) any { return n }, func(n int) ui.Node {
				return Li(
					u.Class(u.Flex, u.JustifyBetween, u.Rounded(u.RadiusLg), u.Border(pal.border), u.Bg(pal.bg), u.PadX(u.Spacing3), u.PadY(u.Spacing2), u.TextSize(u.TextSm)),
					Span(u.Class(u.Fg(pal.text)), "Person "+strconv.Itoa(n)),
					Span(u.Class(u.FontSemibold), usd(props.PerPerson)),
				)
			}),
		),
	)
}

// --- footer: reads shared theme/total ---

type footerProps struct {
	Total  float64
	People int
	Theme  string
	Pal    palette
}

func FooterBar(props footerProps) ui.Node {
	return Footer(
		u.Class(u.TextSize(u.TextXs), u.Fg(props.Pal.muted)),
		"Splitting "+usd(props.Total)+" between "+strconv.Itoa(props.People)+" · "+props.Theme+" theme",
	)
}

// --- small style + format helpers ---

func card(pal palette) html.PropOption {
	return u.Class(u.Rounded(u.RadiusXl), u.Border(pal.border), u.Bg(pal.surface), u.Pad(u.Spacing4), u.Flex, u.FlexCol, u.Gap(u.Spacing4))
}
func field() html.PropOption { return u.Class(u.Flex, u.FlexCol, u.Gap(u.Spacing2)) }
func labelClass(pal palette) html.PropOption {
	return u.Class(u.TextSize(u.TextSm), u.FontSemibold, u.Fg(pal.muted))
}
func stepClass(pal palette) html.PropOption {
	return u.Class(u.Rounded(u.RadiusLg), u.Border(pal.border), u.Bg(pal.bg), u.Fg(pal.text), u.PadX(u.Spacing3), u.PadY(u.Spacing2), u.TextSize(u.TextLg), u.FontBold)
}
func resultRow(pal palette, label, value string) ui.Node {
	return Div(
		u.Class(u.Flex, u.JustifyBetween, u.TextSize(u.TextSm)),
		Span(u.Class(u.Fg(pal.muted)), label),
		Span(u.Class(u.FontSemibold), value),
	)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func parseNum(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

// numStr renders an input value: empty for zero (so the placeholder shows), else the number.
func numStr(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// usd formats a non-negative amount as $1,234.56 (en-US currency, per SPEC).
func usd(v float64) string {
	cents := int64(math.Round(v * 100))
	if cents < 0 {
		cents = 0
	}
	dollars := strconv.FormatInt(cents/100, 10)
	var grouped []byte
	for i := 0; i < len(dollars); i++ {
		if i > 0 && (len(dollars)-i)%3 == 0 {
			grouped = append(grouped, ',')
		}
		grouped = append(grouped, dollars[i])
	}
	frac := cents % 100
	return "$" + string(grouped) + "." + pad2(frac)
}

func pad2(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) < 2 {
		return "0" + s
	}
	return s
}

func main() {
	ui.Run("#app", App)
}
