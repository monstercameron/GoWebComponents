package css

// Declaring custom properties (CSS variables).
//
// Var/VarLength/… already covered *reading* a custom property with full type safety,
// but there was no typed way to DECLARE one: Root's own doc comment pointed callers
// at css.Raw("--accent", "#4f46e5") — a gap admitting itself, and one where a typo in
// the "--" prefix produces a dead declaration rather than a compile error.
//
// These constructors normalize and sanitize the name exactly like Var does, so a
// declaration and its reference can never disagree about the spelling, and an
// untrusted token name cannot inject CSS:
//
//	css.Root(
//	    css.CustomColor("accent", css.Hex("4f46e5")),   // --accent: #4f46e5
//	    css.CustomLength("radius", css.Px(12)),         // --radius: 12px
//	)
//	css.New(css.Bg(css.Var("accent")))                  // background-color: var(--accent)

// CustomColor / CustomLength / CustomNumber / CustomDuration / CustomAngle declare a
// custom property from a typed value. The name may be written with or without the
// leading "--" (both "accent" and "--accent" produce `--accent`).
func CustomColor(parseName string, parseValue Color) Rule {
	return decl(customPropertyName(parseName), string(parseValue))
}
func CustomLength(parseName string, parseValue Length) Rule {
	return decl(customPropertyName(parseName), string(parseValue))
}
func CustomNumber(parseName string, parseValue Number) Rule {
	return decl(customPropertyName(parseName), string(parseValue))
}
func CustomDuration(parseName string, parseValue Duration) Rule {
	return decl(customPropertyName(parseName), string(parseValue))
}
func CustomAngle(parseName string, parseValue Angle) Rule {
	return decl(customPropertyName(parseName), string(parseValue))
}
func CustomFontStack(parseName string, parseValue FontStack) Rule {
	return decl(customPropertyName(parseName), string(parseValue))
}
func CustomShadow(parseName string, parseValue ShadowToken) Rule {
	return decl(customPropertyName(parseName), string(parseValue))
}

// Custom declares a custom property from a raw value string — the escape hatch for
// token values with no typed constructor (a font stack fragment, a `1px solid` line,
// a whole gradient). Only the VALUE is author-trusted; the NAME is still sanitized,
// so this is strictly safer than Raw("--"+name, value) and stays greppable.
func Custom(parseName string, parseValue string) Rule {
	return decl(customPropertyName(parseName), parseValue)
}

// customPropertyName sanitizes a custom-property name to CSS identifier characters
// and guarantees the "--" prefix. Shared with varExpr so Var("accent") and
// CustomColor("accent", …) can never disagree: a name that sanitizes to the same
// identifier resolves to the same property.
func customPropertyName(parseName string) string {
	name := cssIdentSanitize(parseName, "")
	// cssIdentSanitize keeps '-', so a caller-supplied "--accent" arrives intact and
	// only a bare "accent" needs the prefix added.
	for len(name) > 0 && name[0] == '-' {
		name = name[1:]
	}
	return "--" + name
}
