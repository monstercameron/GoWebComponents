package css

// Preflight emits a small, modern CSS reset (a Tailwind-preflight-equivalent) as
// global rules (CSS5). It is opt-in — call it once at startup if you want the
// base layer managed by the framework instead of hand-maintained inline. Rules
// are deduped, hardened, and flow through the same sink as the rest of css, so
// they appear in SSR output and inject on wasm.
//
//	func main() { css.Preflight(); /* … */ }
//
// To place the reset in a cascade layer (so app styles reliably win), see
// PreflightInLayer.
func Preflight() {
	preflightInto("")
}

// PreflightInLayer emits the reset inside a named cascade layer (e.g. "base"), so
// it sits below component/override layers regardless of source order (CSS4+CSS5).
//
//	css.DeclareLayers("base", "components", "overrides")
//	css.PreflightInLayer("base")
func PreflightInLayer(parseLayer string) {
	preflightInto(parseLayer)
}

func preflightInto(parseLayer string) {
	parseEmit := func(parseSelector string, parseRules ...Rule) {
		if parseLayer == "" {
			Global(parseSelector, parseRules...)
			return
		}
		LayerGlobal(parseLayer, parseSelector, parseRules...)
	}

	parseEmit("*,::before,::after", Raw("box-sizing", "border-box"))
	parseEmit("*", Raw("margin", "0"))
	parseEmit("html", Raw("-webkit-text-size-adjust", "100%"), Raw("tab-size", "4"))
	parseEmit("body", Raw("line-height", "1.5"), Raw("-webkit-font-smoothing", "antialiased"))
	parseEmit("img,picture,video,canvas,svg", Raw("display", "block"), Raw("max-width", "100%"))
	parseEmit("input,button,textarea,select", Raw("font", "inherit"), Raw("color", "inherit"))
	parseEmit("button", Raw("cursor", "pointer"))
	parseEmit("p,h1,h2,h3,h4,h5,h6", Raw("overflow-wrap", "break-word"))
	parseEmit("h1,h2,h3,h4,h5,h6", Raw("font-size", "inherit"), Raw("font-weight", "inherit"))
	parseEmit("a", Raw("color", "inherit"), Raw("text-decoration", "inherit"))
}
