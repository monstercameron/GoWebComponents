// Bill Splitter — HTMX implementation.
//
// HTMX is server-driven hypermedia: there is no client-side reactive framework.
// State lives on the server; every interaction posts to an endpoint that mutates
// state and returns the re-rendered <main> fragment, which htmx morph-swaps into
// place (focus preserved). This is the honest shape of an htmx app and a useful
// contrast to the client-wasm / client-JS implementations in this study.
//
// Single shared in-memory state (one user) keeps the demo small; a real app would
// scope it per session.
package main

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type state struct {
	bill    float64
	tip     float64
	people  int
	roundUp bool
	theme   string
}

var (
	mu sync.Mutex
	st = state{bill: 0, tip: 18, people: 1, roundUp: false, theme: "light"}
)

const presetsCSV = "10,15,18,20,25"

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/styles.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../../shared/styles.css") // canonical shared design
	})
	http.HandleFunc("/bill", mutating(func(r *http.Request) { st.bill = parseNum(r.FormValue("value")) }))
	http.HandleFunc("/tip", mutating(func(r *http.Request) { st.tip = parseNum(r.FormValue("value")) }))
	http.HandleFunc("/preset", mutating(func(r *http.Request) { st.tip = parseNum(r.URL.Query().Get("p")) }))
	http.HandleFunc("/people/inc", mutating(func(r *http.Request) { st.people++ }))
	http.HandleFunc("/people/dec", mutating(func(r *http.Request) {
		if st.people > 1 {
			st.people--
		}
	}))
	http.HandleFunc("/toggle/theme", mutating(func(r *http.Request) {
		if st.theme == "dark" {
			st.theme = "light"
		} else {
			st.theme = "dark"
		}
	}))
	http.HandleFunc("/toggle/roundup", mutating(func(r *http.Request) { st.roundUp = !st.roundUp }))

	addr := "127.0.0.1:8099"
	fmt.Println("HTMX Bill Splitter on http://" + addr)
	_ = http.ListenAndServe(addr, nil)
}

// mutating wraps a state mutation and writes back the morphed <main> fragment.
func mutating(apply func(r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		_ = r.ParseForm()
		apply(r)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, renderMain(st))
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Bill Splitter — HTMX</title>
    <link rel="stylesheet" href="/styles.css" />
    <script src="https://unpkg.com/htmx.org@2.0.1"></script>
    <script src="https://unpkg.com/idiomorph@0.3.0/dist/idiomorph-ext.min.js"></script>
  </head>
  <!-- All controls inherit these: post returns the morphed <main>, focus preserved. -->
  <body hx-target="#bs-main" hx-swap="morph:outerHTML" hx-ext="morph">
    %s
  </body>
</html>`, renderMain(st))
}

// renderMain returns the full interactive view as one morph-swappable fragment.
func renderMain(s state) string {
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

	var b strings.Builder
	fmt.Fprintf(&b, `<main id="bs-main" class="bs-app" data-theme="%s">`, s.theme)

	// header
	fmt.Fprintf(&b, `<header class="bs-header"><h1 class="bs-title">Bill Splitter</h1><div class="bs-header-actions">`+
		`<button class="bs-toggle" aria-pressed="%t" hx-post="/toggle/roundup">Round up</button>`+
		`<button class="bs-toggle" aria-pressed="%t" hx-post="/toggle/theme">Dark</button>`+
		`</div></header>`, s.roundUp, s.theme == "dark")

	// inputs
	b.WriteString(`<section class="bs-card bs-inputs">`)
	fmt.Fprintf(&b, `<label class="bs-field"><span class="bs-label">Bill amount</span><div class="bs-input-wrap"><span class="bs-prefix">$</span>`+
		`<input class="bs-input" type="number" min="0" step="0.01" name="value" value="%s" hx-post="/bill" hx-trigger="input changed delay:200ms" /></div></label>`,
		numStr(s.bill))

	b.WriteString(`<div class="bs-field"><span class="bs-label">Tip</span><div class="bs-presets">`)
	for _, p := range strings.Split(presetsCSV, ",") {
		pv, _ := strconv.ParseFloat(p, 64)
		active := ""
		if s.tip == pv {
			active = " bs-preset--active"
		}
		fmt.Fprintf(&b, `<button class="bs-preset%s" hx-post="/preset?p=%s">%s%%</button>`, active, p, p)
	}
	fmt.Fprintf(&b, `<input class="bs-preset-custom" type="number" min="0" placeholder="Custom %%" name="value" value="%s" hx-post="/tip" hx-trigger="input changed delay:200ms" /></div></div>`,
		numStr(s.tip))

	fmt.Fprintf(&b, `<div class="bs-field"><span class="bs-label">People</span><div class="bs-stepper">`+
		`<button class="bs-step" aria-label="Fewer people"%s hx-post="/people/dec">−</button>`+
		`<span class="bs-count">%d</span>`+
		`<button class="bs-step" aria-label="More people" hx-post="/people/inc">+</button>`+
		`</div></div>`, disabledAttr(s.people <= 1), s.people)
	b.WriteString(`</section>`)

	// results
	b.WriteString(`<section class="bs-card bs-results">`)
	fmt.Fprintf(&b, `<div class="bs-result-row"><span>Tip</span><span>%s</span></div>`, usd(tipAmount))
	fmt.Fprintf(&b, `<div class="bs-result-row"><span>Total</span><span>%s</span></div>`, usd(total))
	fmt.Fprintf(&b, `<div class="bs-result-hero"><span class="bs-result-hero-label">Per person</span><span class="bs-result-hero-value">%s</span></div>`, usd(perPerson))
	if s.roundUp && roundingExtra > 0 {
		fmt.Fprintf(&b, `<p class="bs-note">Rounding up collects %s extra · effective tip %s%%</p>`, usd(roundingExtra), strconv.FormatFloat(effTip, 'f', 1, 64))
	}
	if s.bill <= 0 {
		b.WriteString(`<p class="bs-empty">Enter a bill amount to begin.</p>`)
	}
	b.WriteString(`</section>`)

	// breakdown
	b.WriteString(`<section class="bs-card bs-breakdown"><h2 class="bs-subtitle">Per-person breakdown</h2><ul class="bs-people">`)
	for n := 1; n <= s.people; n++ {
		fmt.Fprintf(&b, `<li class="bs-person"><span>Person %d</span><span>%s</span></li>`, n, usd(perPerson))
	}
	b.WriteString(`</ul></section>`)

	// footer
	fmt.Fprintf(&b, `<footer class="bs-footer">Splitting %s between %d · %s theme</footer>`, usd(total), s.people, s.theme)

	b.WriteString(`</main>`)
	return b.String()
}

func disabledAttr(disabled bool) string {
	if disabled {
		return " disabled"
	}
	return ""
}

func parseNum(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
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
	dollars := strconv.FormatInt(cents/100, 10)
	var grouped []byte
	for i := 0; i < len(dollars); i++ {
		if i > 0 && (len(dollars)-i)%3 == 0 {
			grouped = append(grouped, ',')
		}
		grouped = append(grouped, dollars[i])
	}
	frac := cents % 100
	fs := strconv.FormatInt(frac, 10)
	if len(fs) < 2 {
		fs = "0" + fs
	}
	return "$" + string(grouped) + "." + fs
}
