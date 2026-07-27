// Package css is GoWebComponents' typed, type-safe CSS layer. The raw-CSS layer
// (Rule/value/prop/variant) is the foundation; the Tailwind-shaped utility engine
// (subpackage css/u) is a typed convenience built on top. Emission v1 is runtime
// <style> injection on wasm and an in-memory buffer (read back by SSR and tests)
// on native.
//
// The two public entry points cross into the html package without any edits to
// it: New returns a Sheet (a fmt.Stringer that flows through html.ClassNames,
// ClassMap keys, SSR, and attribute strings) and Class returns an
// html.PropOption that delegates to the existing html.Class setter.
//
//	// today
//	Div(html.Class("flex gap-2 hover:bg-slate-900"), …)
//	// typed, same shape
//	Div(css.Class(css.Display.Flex, css.Gap(css.Px(8)), css.Hover(css.Bg(css.Slate900))), …)
package css

import (
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v5/html"
)

// Sheet is a generated class name (or space-separated names). It satisfies
// fmt.Stringer, so it already flows through html.ClassNames(...),
// appendClassFragments, ClassMap keys, SSR, and attribute strings.
type Sheet string

// String implements fmt.Stringer.
func (s Sheet) String() string { return string(s) }

// New folds a rule-set into a single content-hashed class, registers it (deduped
// process-wide), and emits its compiled CSS through the active Sink exactly once.
// It is safe and cheap to call inside render loops: N identical New(...) calls
// produce one registry entry and one injected rule.
//
// Variant helpers (css.Hover, css.Media, …) return []Rule; spread them into New
// with the ... operator, or pass them through the rules... variadic directly —
// New accepts both single Rules and rule slices via the Rules helper.
func New(rules ...Rule) Sheet {
	// Fast path: digest the INPUT rules and look that up before doing any work.
	// canonicalize allocates maps, sorts and serializes, so consulting the cache
	// after it — which is what this function used to do — meant a repeat fold still
	// paid for the expensive half. See fastfold.go for the measurements and for why
	// the digest must be order-sensitive.
	foldedKey := computeFoldKey(rules)
	if cached, ok := foldCache.Load(foldedKey); ok {
		return cached.(Sheet)
	}

	canonical, groups, raws := canonicalize(rules)
	if canonical == "" {
		// Not cached: an empty fold is already cheap, and caching it would mean
		// storing a key for every distinct no-op rule-set a caller constructs.
		return ""
	}
	// Second-level cache: two different rule ORDERS canonicalize to the same text
	// when they do not conflict, so this still collapses them onto one class even
	// though their fold keys differ.
	if cached, ok := newCache.Load(canonical); ok {
		sheet := cached.(Sheet)
		foldCache.Store(foldedKey, sheet)
		return sheet
	}
	class := hashClass(canonical)

	var cssText strings.Builder
	for _, g := range groups {
		cssText.WriteString(g.render(class))
	}
	for _, raw := range raws {
		cssText.WriteString(raw)
	}
	registerAndEmit(class, hardenCSS(cssText.String()))
	sheet := Sheet(class)
	newCache.Store(canonical, sheet)
	foldCache.Store(foldedKey, sheet)
	return sheet
}

// newCache memoizes canonical-serialization -> Sheet so repeat folds skip the
// hash + CSS-text build. Reset clears it alongside the registry.
var newCache sync.Map

// hardenCSS neutralizes the few sequences that could let CSS text break out of
// the <style> raw-text element (and thus become HTML/JS execution) once the
// emitted text is serialized into a document — via SSR StyleBlock or the wasm DOM
// sink. It is applied to ALL emitted CSS so every sink is breakout-safe.
//
// This is a security boundary, NOT a CSS validator: the escape hatches (Raw, Sel,
// DefineVariant, RawMedia) remain author-trusted for ordinary CSS-level content
// (an injected "}" can still restyle the sheet — that's the author's problem, like
// dangerouslySetInnerHTML). What it guarantees is that no input can terminate the
// <style> element or a CSS comment. The inserted backslashes are valid CSS string
// escapes (\/ == /, \! == !, \s == s), so legitimate content inside CSS strings is
// preserved byte-for-byte; outside strings these sequences were invalid CSS anyway.
func hardenCSS(parseCSS string) string {
	if !strings.ContainsRune(parseCSS, 0) &&
		!strings.Contains(parseCSS, "<") &&
		!strings.Contains(parseCSS, "*/") {
		return parseCSS
	}
	// Drop NUL FIRST, in a separate pass. A NUL is never valid and is a classic
	// filter-bypass byte: if it were merely skipped inline, an input like
	// "*\x00/" or "<\x00/style>" would slip past the lookahead guards below (which
	// see the NUL, not the dangerous next byte) and then be removed — silently
	// reconstituting "*/" or "</style>" in the output. Removing NULs up front means
	// the guards run on the final byte stream.
	if strings.ContainsRune(parseCSS, 0) {
		parseCSS = strings.ReplaceAll(parseCSS, "\x00", "")
	}
	var b strings.Builder
	b.Grow(len(parseCSS) + 8)
	for i := 0; i < len(parseCSS); i++ {
		c := parseCSS[i]
		switch {
		case c == '<' && i+1 < len(parseCSS) && isTagStart(parseCSS[i+1]):
			// "</style", "<script", "<!--" … — break the HTML token.
			b.WriteString("<\\")
		case c == '*' && i+1 < len(parseCSS) && parseCSS[i+1] == '/':
			// "*/" — break a CSS comment close.
			b.WriteString("*\\")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// isTagStart reports whether b could begin an HTML tag/comment after '<'.
func isTagStart(b byte) bool {
	return b == '/' || b == '!' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// Rules flattens a mix of single Rules and rule slices (the []Rule returned by
// variant helpers) into a flat []Rule, so authoring can freely intermix them:
//
//	css.New(css.Rules(
//	    css.Display.Flex,
//	    css.Hover(css.Bg(css.Slate800)),   // []Rule
//	    css.Gap(css.Px(8)),
//	)...)
func Rules(parts ...any) []Rule {
	var out []Rule
	for _, p := range parts {
		switch t := p.(type) {
		case Rule:
			out = append(out, t)
		case []Rule:
			out = append(out, t...)
		}
	}
	return out
}

// Class is the unified PropOption entry point for Div(...)/shorthand argument
// lists — the typed analog of JSX's className/clsx. It accepts a mixed list of:
//
//   - string  — a literal class name (migration / third-party / utility strings)
//   - Rule    — a single typed rule
//   - []Rule  — the slice returned by variants (Hover, Media, Child, …)
//   - Sheet   — a pre-folded class from New
//   - []Sheet — several pre-folded classes
//
// Typed rules are folded together into one hashed class via New; strings and
// Sheets pass through as class names. Everything is joined into the class
// attribute. Mixing is the whole point:
//
//	Div(Class("legacy-util", Display.Flex, Gap(Px(8)), Hover(Bg(Slate900)), someSheet))
//
// Type-safety note: the compile-checked guarantees live in the rule builders
// (Display.Flex, Gap(Px(8)), …); the argument list is `...any` because the class
// attribute is an inherently stringy boundary (exactly like clsx). Unrecognized
// argument types are ignored.
func Class(parts ...any) html.PropOption {
	var names []string
	var rules []Rule
	collectClassParts(parts, &names, &rules)
	if len(rules) > 0 {
		if s := New(rules...); s != "" {
			names = append(names, string(s))
		}
	}
	return html.Class(strings.Join(names, " "))
}

func collectClassParts(parts []any, names *[]string, rules *[]Rule) {
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			if v != "" {
				*names = append(*names, v)
			}
		case Rule:
			*rules = append(*rules, v)
		case []Rule:
			*rules = append(*rules, v...)
		case Sheet:
			if v != "" {
				*names = append(*names, string(v))
			}
		case []Sheet:
			for _, s := range v {
				if s != "" {
					*names = append(*names, string(s))
				}
			}
		case []any:
			collectClassParts(v, names, rules)
		}
	}
}
