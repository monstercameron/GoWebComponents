package css

import "strings"

// Global emits CSS rules under a literal, top-level selector — an element
// (`body`, `h3`), the universal selector (`*`), a `:root` token block, or a
// stable semantic class (`.nav-link`, `.bento`) that other markup or a runtime
// theme engine targets by name. Unlike [New], the rules are NOT scoped under a
// generated hashed class: the selector you pass is emitted verbatim.
//
//	css.Global("body", css.Margin(css.Px(0)), css.Font(css.SansStack))
//	css.Global(".nav-link", css.Display.Flex, css.Hover(css.Bg(css.Slate800)))
//
// Variants compose against the selector exactly as they do against `&` in New:
// the `&` placeholder is replaced by the literal selector, so
// css.Hover(...) under Global(".btn", ...) emits `.btn:hover { … }` and
// css.Descendant(h3, ...) emits `.btn h3 { … }`.
//
// Emission is deduped process-wide on the (selector + rules) content, flows
// through the same Sink as New (so SSR StyleBlock/Harvest include it and the
// wasm DOM sink injects it), and is hardened against <style> breakout. Calling
// Global repeatedly with the same selector and rules is cheap and emits once.
func Global(parseSelector string, parseRules ...Rule) {
	emitGlobal("", parseSelector, parseRules)
}

// emitGlobal is the shared emission path for Global and LayerGlobal. atRule, when
// non-empty (e.g. "@layer overrides"), wraps the emitted block.
func emitGlobal(parseAtRule string, parseSelector string, parseRules []Rule) {
	if strings.TrimSpace(parseSelector) == "" {
		return
	}
	scoped := applyVariant(parseSelector, parseAtRule, parseRules)
	canonical, groups, raws := canonicalize(scoped)
	if canonical == "" {
		return
	}
	key := "g-" + shortHash(parseAtRule+"\x00"+parseSelector+"\x00"+canonical)

	var cssText strings.Builder
	for _, g := range groups {
		// The templates no longer contain "&" (applyVariant substituted the literal
		// selector), so render's class substitution is a no-op — the literal
		// selector is emitted.
		cssText.WriteString(g.render(""))
	}
	for _, raw := range raws {
		cssText.WriteString(raw)
	}
	registerAndEmit(key, hardenCSS(cssText.String()))
}

// Layer folds rules into a hashed class emitted inside the named cascade layer
// (`@layer <name> { .c-xxx { … } }`), giving declared override precedence
// instead of registration-order accident (CSS4). Declare the layer order once
// with DeclareLayers so later layers reliably win.
//
//	css.DeclareLayers("base", "components", "overrides")
//	btn := css.Layer("components", css.Display.Flex, css.Padding(css.Px(8)))
func Layer(parseName string, parseRules ...Rule) Sheet {
	if strings.TrimSpace(parseName) == "" {
		return New(parseRules...)
	}
	return New(applyVariant("", "@layer "+parseName, parseRules)...)
}

// LayerGlobal emits global rules (element/:root/semantic-class selectors) inside
// a named cascade layer — the typed home for a light-theme override layer that
// must beat base rules by layer order rather than specificity hacks (CSS4).
//
//	css.LayerGlobal("overrides", `[data-theme="light"] .card`, css.Bg(css.White))
func LayerGlobal(parseName string, parseSelector string, parseRules ...Rule) {
	if strings.TrimSpace(parseName) == "" {
		emitGlobal("", parseSelector, parseRules)
		return
	}
	emitGlobal("@layer "+parseName, parseSelector, parseRules)
}

// DeclareLayers emits an `@layer a, b, c;` statement establishing layer order
// (earlier = lower precedence). Call it once at startup before other emissions so
// the order statement leads the stylesheet. Identical declarations dedupe.
//
//	css.DeclareLayers("base", "components", "overrides")
func DeclareLayers(parseNames ...string) {
	if len(parseNames) == 0 {
		return
	}
	statement := "@layer " + strings.Join(parseNames, ",") + ";"
	registerAndEmit("layer-decl-"+shortHash(statement), hardenCSS(statement))
}

// Root emits a `:root { … }` block — the canonical home for a custom-property
// (design-token) palette authored in typed Go:
//
//	css.Root(css.Raw("--accent", "#4f46e5"), css.Raw("--radius", "12px"))
//
// It is shorthand for Global(":root", rules...). Author custom properties with
// the existing css.Raw(property, value) escape hatch. A runtime theme engine can
// then override individual tokens with element.style.setProperty without
// regenerating any classes.
func Root(parseRules ...Rule) {
	Global(":root", parseRules...)
}

// Within scopes rules to apply only when an ancestor matches selector — the
// `<ancestor> &` form that powers attribute-state theming
// (`[data-theme="light"] &`, `html[data-density="compact"] &`). It is the
// ancestor-state counterpart to the self-state variants (Hover/Focus/…):
//
//	css.New(css.Within(`[data-theme="light"]`, css.Bg(css.White), css.Color(css.Slate900)))
//
// emits `[data-theme="light"] .c-xxx { … }`.
func Within(parseAncestor string, parseRules ...Rule) []Rule {
	parseAncestor = strings.TrimSpace(parseAncestor)
	if parseAncestor == "" {
		return parseRules
	}
	return applyVariant(parseAncestor+" &", "", parseRules)
}

// DataTheme is a convenience for Within(`[data-theme="<name>"]`, rules...) — the
// common ancestor-attribute theming case. (Named DataTheme to avoid colliding
// with the Theme token-config type.)
//
//	css.New(css.Bg(css.Slate900), css.DataTheme("light", css.Bg(css.White)))
func DataTheme(parseName string, parseRules ...Rule) []Rule {
	return Within(`[data-theme="`+parseName+`"]`, parseRules...)
}

// Inject installs an arbitrary CSS string as a managed <style> element, keyed by
// id and idempotent (the first call for an id wins; later calls for the same id
// are no-ops). It is the runtime counterpart to Global for CSS that must be a
// raw string rather than typed rules — `@font-face` blocks for user-uploaded
// fonts, a third-party widget's stylesheet, etc.
//
//	css.Inject("app-fonts", `@font-face{font-family:"My Font";src:url(...)}`)
//
// On wasm it creates (once) a <style id="..."> in <head>; on native it records
// the CSS into the buffer sink so SSR/tests can read it back. The CSS is hardened
// against <style> breakout. To replace injected CSS, use a new id.
func Inject(parseID string, parseCSS string) {
	if strings.TrimSpace(parseID) == "" || parseCSS == "" {
		return
	}
	injectStyleElement(parseID, hardenCSS(parseCSS))
}
