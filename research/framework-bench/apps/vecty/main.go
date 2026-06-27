// Bill Splitter — Vecty (Go → wasm, React-like component model).
//
// One root component holds all state; mutations call vecty.Rerender. Build with
// `GOARCH=wasm GOOS=js go build -o app.wasm` and serve with wasm_exec.js + the
// canonical ../../shared/styles.css.
//
// Status: to-spec scaffold, not build-verified here (run `go mod tidy` first).
package main

import (
	"math"
	"strconv"

	"github.com/hexops/vecty"
	"github.com/hexops/vecty/elem"
	"github.com/hexops/vecty/event"
	"github.com/hexops/vecty/prop"
)

var presets = []int{10, 15, 18, 20, 25}

type Splitter struct {
	vecty.Core
	bill    float64
	tip     float64
	people  int
	theme   string
	roundUp bool
}

func (c *Splitter) toggleTheme() {
	if c.theme == "dark" {
		c.theme = "light"
	} else {
		c.theme = "dark"
	}
	vecty.Rerender(c)
}
func (c *Splitter) toggleRoundUp() { c.roundUp = !c.roundUp; vecty.Rerender(c) }
func (c *Splitter) inc()           { c.people++; vecty.Rerender(c) }
func (c *Splitter) dec() {
	if c.people > 1 {
		c.people--
	}
	vecty.Rerender(c)
}

func (c *Splitter) Render() vecty.ComponentOrHTML {
	tipAmount := c.bill * c.tip / 100
	total := c.bill + tipAmount
	perPersonRaw := 0.0
	if c.people > 0 {
		perPersonRaw = total / float64(c.people)
	}
	perPerson := perPersonRaw
	totalCollected := total
	if c.roundUp {
		perPerson = math.Ceil(perPersonRaw)
		totalCollected = perPerson * float64(c.people)
	}
	roundingExtra := math.Max(0, totalCollected-total)
	effTip := c.tip
	if c.bill > 0 {
		effTip = (totalCollected - c.bill) / c.bill * 100
	}

	presetButtons := make([]vecty.MarkupOrChild, 0, len(presets)+1)
	for _, p := range presets {
		pp := p
		presetButtons = append(presetButtons, elem.Button(
			vecty.Markup(
				vecty.ClassMap{"bs-preset": true, "bs-preset--active": c.tip == float64(pp)},
				event.Click(func(*vecty.Event) { c.tip = float64(pp); vecty.Rerender(c) }),
			),
			vecty.Text(strconv.Itoa(pp)+"%"),
		))
	}
	presetButtons = append(presetButtons, elem.Input(vecty.Markup(
		vecty.Class("bs-preset-custom"), prop.Type(prop.TypeNumber), vecty.Property("min", "0"),
		prop.Placeholder("Custom %"), prop.Value(numStr(c.tip)),
		event.Input(func(e *vecty.Event) { c.tip = parseNum(e.Target.Get("value").String()); vecty.Rerender(c) }),
	)))

	people := make([]vecty.MarkupOrChild, 0, c.people)
	for i := 1; i <= c.people; i++ {
		people = append(people, elem.ListItem(
			vecty.Markup(vecty.Class("bs-person")),
			elem.Span(vecty.Text("Person "+strconv.Itoa(i))),
			elem.Span(vecty.Text(usd(perPerson))),
		))
	}

	return elem.Main(
		vecty.Markup(vecty.Class("bs-app"), vecty.Attribute("data-theme", c.theme)),
		elem.Header(
			vecty.Markup(vecty.Class("bs-header")),
			elem.Heading1(vecty.Markup(vecty.Class("bs-title")), vecty.Text("Bill Splitter")),
			elem.Div(
				vecty.Markup(vecty.Class("bs-header-actions")),
				elem.Button(vecty.Markup(vecty.Class("bs-toggle"), event.Click(func(*vecty.Event) { c.toggleRoundUp() })), vecty.Text("Round up")),
				elem.Button(vecty.Markup(vecty.Class("bs-toggle"), event.Click(func(*vecty.Event) { c.toggleTheme() })), vecty.Text("Dark")),
			),
		),
		elem.Section(
			vecty.Markup(vecty.Class("bs-card", "bs-inputs")),
			elem.Label(
				vecty.Markup(vecty.Class("bs-field")),
				elem.Span(vecty.Markup(vecty.Class("bs-label")), vecty.Text("Bill amount")),
				elem.Div(
					vecty.Markup(vecty.Class("bs-input-wrap")),
					elem.Span(vecty.Markup(vecty.Class("bs-prefix")), vecty.Text("$")),
					elem.Input(vecty.Markup(
						vecty.Class("bs-input"), prop.Type(prop.TypeNumber), prop.Value(numStr(c.bill)),
						event.Input(func(e *vecty.Event) { c.bill = parseNum(e.Target.Get("value").String()); vecty.Rerender(c) }),
					)),
				),
			),
			elem.Div(
				vecty.Markup(vecty.Class("bs-field")),
				elem.Span(vecty.Markup(vecty.Class("bs-label")), vecty.Text("Tip")),
				elem.Div(append([]vecty.MarkupOrChild{vecty.Markup(vecty.Class("bs-presets"))}, presetButtons...)...),
			),
			elem.Div(
				vecty.Markup(vecty.Class("bs-field")),
				elem.Span(vecty.Markup(vecty.Class("bs-label")), vecty.Text("People")),
				elem.Div(
					vecty.Markup(vecty.Class("bs-stepper")),
					elem.Button(vecty.Markup(vecty.Class("bs-step"), prop.Disabled(c.people <= 1), event.Click(func(*vecty.Event) { c.dec() })), vecty.Text("−")),
					elem.Span(vecty.Markup(vecty.Class("bs-count")), vecty.Text(strconv.Itoa(c.people))),
					elem.Button(vecty.Markup(vecty.Class("bs-step"), event.Click(func(*vecty.Event) { c.inc() })), vecty.Text("+")),
				),
			),
		),
		elem.Section(
			vecty.Markup(vecty.Class("bs-card", "bs-results")),
			resultRow("Tip", usd(tipAmount)),
			resultRow("Total", usd(total)),
			elem.Div(
				vecty.Markup(vecty.Class("bs-result-hero")),
				elem.Span(vecty.Markup(vecty.Class("bs-result-hero-label")), vecty.Text("Per person")),
				elem.Span(vecty.Markup(vecty.Class("bs-result-hero-value")), vecty.Text(usd(perPerson))),
			),
			vecty.If(c.roundUp && roundingExtra > 0, elem.Paragraph(vecty.Markup(vecty.Class("bs-note")),
				vecty.Text("Rounding up collects "+usd(roundingExtra)+" extra · effective tip "+strconv.FormatFloat(effTip, 'f', 1, 64)+"%"))),
			vecty.If(c.bill <= 0, elem.Paragraph(vecty.Markup(vecty.Class("bs-empty")), vecty.Text("Enter a bill amount to begin."))),
		),
		elem.Section(
			vecty.Markup(vecty.Class("bs-card", "bs-breakdown")),
			elem.Heading2(vecty.Markup(vecty.Class("bs-subtitle")), vecty.Text("Per-person breakdown")),
			elem.UnorderedList(append([]vecty.MarkupOrChild{vecty.Markup(vecty.Class("bs-people"))}, people...)...),
		),
		elem.Footer(vecty.Markup(vecty.Class("bs-footer")),
			vecty.Text("Splitting "+usd(total)+" between "+strconv.Itoa(c.people)+" · "+c.theme+" theme")),
	)
}

func resultRow(label, value string) *vecty.HTML {
	return elem.Div(vecty.Markup(vecty.Class("bs-result-row")), elem.Span(vecty.Text(label)), elem.Span(vecty.Text(value)))
}

func main() {
	vecty.SetTitle("Bill Splitter — Vecty")
	vecty.RenderInto("#app", &Splitter{tip: 18, people: 1, theme: "light"})
}

func parseNum(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}
func numStr(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
func usd(v float64) string {
	cents := int64(math.Round(v * 100))
	if cents < 0 {
		cents = 0
	}
	d := strconv.FormatInt(cents/100, 10)
	var g []byte
	for i := 0; i < len(d); i++ {
		if i > 0 && (len(d)-i)%3 == 0 {
			g = append(g, ',')
		}
		g = append(g, d[i])
	}
	fs := strconv.FormatInt(cents%100, 10)
	if len(fs) < 2 {
		fs = "0" + fs
	}
	return "$" + string(g) + "." + fs
}
