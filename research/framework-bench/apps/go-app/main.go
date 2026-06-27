// Bill Splitter — go-app (maxence-charriere/go-app), Go compiled to wasm.
//
// go-app is a GWC peer: the whole UI is Go→wasm. The same binary builds to wasm
// for the browser and runs the server that ships it. State is component fields;
// shared theme/roundUp use go-app's context state (SetState/ObserveState).
//
// Build & run:
//   GOARCH=wasm GOOS=js go build -o web/app.wasm
//   go run .            # serves on :8100
package main

import (
	"math"
	"net/http"
	"strconv"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

var presets = []int{10, 15, 18, 20, 25}

// header is its own component sharing theme/roundUp through context state.
type header struct {
	app.Compo
	theme   string
	roundUp bool
}

func (h *header) OnMount(ctx app.Context) {
	ctx.ObserveState("theme", &h.theme)
	ctx.ObserveState("roundUp", &h.roundUp)
}

func (h *header) Render() app.UI {
	return app.Header().Class("bs-header").Body(
		app.H1().Class("bs-title").Text("Bill Splitter"),
		app.Div().Class("bs-header-actions").Body(
			app.Button().Class("bs-toggle").Aria("pressed", h.roundUp).
				OnClick(func(ctx app.Context, e app.Event) { ctx.SetState("roundUp", !h.roundUp).Broadcast() }).
				Text("Round up"),
			app.Button().Class("bs-toggle").Aria("pressed", h.theme == "dark").
				OnClick(func(ctx app.Context, e app.Event) {
					next := "dark"
					if h.theme == "dark" {
						next = "light"
					}
					ctx.SetState("theme", next).Broadcast()
				}).Text("Dark"),
		),
	)
}

type splitter struct {
	app.Compo
	bill    float64
	tip     float64
	people  int
	theme   string
	roundUp bool
}

func (s *splitter) OnMount(ctx app.Context) {
	s.tip, s.people, s.theme = 18, 1, "light"
	ctx.SetState("theme", "light")
	ctx.SetState("roundUp", false)
	ctx.ObserveState("theme", &s.theme)
	ctx.ObserveState("roundUp", &s.roundUp)
}

func (s *splitter) Render() app.UI {
	tipAmount := s.bill * s.tip / 100
	total := s.bill + tipAmount
	perPersonRaw := 0.0
	if s.people > 0 {
		perPersonRaw = total / float64(s.people)
	}
	perPerson := perPersonRaw
	totalCollected := total
	if s.roundUp {
		perPerson = math.Ceil(perPersonRaw)
		totalCollected = perPerson * float64(s.people)
	}
	roundingExtra := math.Max(0, totalCollected-total)
	effTip := s.tip
	if s.bill > 0 {
		effTip = (totalCollected - s.bill) / s.bill * 100
	}

	return app.Main().Class("bs-app").DataSet("theme", s.theme).Body(
		&header{},
		// inputs
		app.Section().Class("bs-card bs-inputs").Body(
			app.Label().Class("bs-field").Body(
				app.Span().Class("bs-label").Text("Bill amount"),
				app.Div().Class("bs-input-wrap").Body(
					app.Span().Class("bs-prefix").Text("$"),
					app.Input().Class("bs-input").Type("number").Min(0).Step(0.01).Value(numStr(s.bill)).
						OnInput(func(ctx app.Context, e app.Event) { s.bill = parseNum(ctx.JSSrc().Get("value").String()) }),
				),
			),
			app.Div().Class("bs-field").Body(
				app.Span().Class("bs-label").Text("Tip"),
				app.Div().Class("bs-presets").Body(
					app.Range(presets).Slice(func(i int) app.UI {
						p := presets[i]
						cls := "bs-preset"
						if s.tip == float64(p) {
							cls = "bs-preset bs-preset--active"
						}
						return app.Button().Class(cls).
							OnClick(func(ctx app.Context, e app.Event) { s.tip = float64(p) }).
							Text(strconv.Itoa(p) + "%")
					}),
					app.Input().Class("bs-preset-custom").Type("number").Min(0).Placeholder("Custom %").Value(numStr(s.tip)).
						OnInput(func(ctx app.Context, e app.Event) { s.tip = parseNum(ctx.JSSrc().Get("value").String()) }),
				),
			),
			app.Div().Class("bs-field").Body(
				app.Span().Class("bs-label").Text("People"),
				app.Div().Class("bs-stepper").Body(
					app.Button().Class("bs-step").Aria("label", "Fewer people").Disabled(s.people <= 1).
						OnClick(func(ctx app.Context, e app.Event) {
							if s.people > 1 {
								s.people--
							}
						}).Text("−"),
					app.Span().Class("bs-count").Text(strconv.Itoa(s.people)),
					app.Button().Class("bs-step").Aria("label", "More people").
						OnClick(func(ctx app.Context, e app.Event) { s.people++ }).Text("+"),
				),
			),
		),
		// results
		app.Section().Class("bs-card bs-results").Body(
			resultRow("Tip", usd(tipAmount)),
			resultRow("Total", usd(total)),
			app.Div().Class("bs-result-hero").Body(
				app.Span().Class("bs-result-hero-label").Text("Per person"),
				app.Span().Class("bs-result-hero-value").Text(usd(perPerson)),
			),
			app.If(s.roundUp && roundingExtra > 0, func() app.UI {
				return app.P().Class("bs-note").Text("Rounding up collects " + usd(roundingExtra) + " extra · effective tip " + strconv.FormatFloat(effTip, 'f', 1, 64) + "%")
			}),
			app.If(s.bill <= 0, func() app.UI {
				return app.P().Class("bs-empty").Text("Enter a bill amount to begin.")
			}),
		),
		// breakdown
		app.Section().Class("bs-card bs-breakdown").Body(
			app.H2().Class("bs-subtitle").Text("Per-person breakdown"),
			app.Ul().Class("bs-people").Body(
				app.Range(make([]int, s.people)).Slice(func(i int) app.UI {
					return app.Li().Class("bs-person").Body(
						app.Span().Text("Person "+strconv.Itoa(i+1)),
						app.Span().Text(usd(perPerson)),
					)
				}),
			),
		),
		app.Footer().Class("bs-footer").Text("Splitting "+usd(total)+" between "+strconv.Itoa(s.people)+" · "+s.theme+" theme"),
	)
}

func resultRow(label, value string) app.UI {
	return app.Div().Class("bs-result-row").Body(
		app.Span().Text(label),
		app.Span().Text(value),
	)
}

func main() {
	app.Route("/", func() app.Composer { return &splitter{} })
	app.RunWhenOnBrowser()
	http.Handle("/", &app.Handler{
		Name:   "Bill Splitter",
		Styles: []string{"/web/styles.css"}, // copy ../../shared/styles.css into web/
	})
	_ = http.ListenAndServe(":8100", nil)
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
