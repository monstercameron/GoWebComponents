package ui

import "github.com/monstercameron/GoWebComponents/v5/internal/runtime"

// themeAtomID is the well-known atom holding the active theme name. UseTheme
// subscribes to it; SetTheme/CurrentTheme read and write it from anywhere.
const themeAtomID = "gwc:theme"

// UseTheme subscribes the calling component to the active theme name and returns
// it together with a setter. Switching the theme (via the returned setter or
// SetTheme) re-renders every UseTheme subscriber and applies the name to
// <html data-theme="..."> so `[data-theme="..."]` CSS rules and :root token
// overrides take effect. It is the reactive capstone over the typed-CSS token
// system (css.Theme.RootRules / css.Var): style with tokens, switch with UseTheme.
//
//	theme, setTheme := ui.UseTheme("dark")
//	onClick := ui.UseEvent(func() {
//	    if theme == "dark" { setTheme("light") } else { setTheme("dark") }
//	})
//
// Seed from the OS preference by passing it as the default:
//
//	theme, setTheme := ui.UseTheme(string(ui.UsePrefersColorScheme()))
func UseTheme(parseDefault string) (string, func(string)) {
	parseGet, parseSet := runtime.GoUseAtomGlobal(themeAtomID, parseDefault)
	parseCurrent := parseGet()
	// Apply the data-theme attribute after commit (layout effect: before paint),
	// keyed on the current theme so it only runs on change.
	UseLayoutEffect(func() func() {
		applyThemeAttribute(parseCurrent)
		return nil
	}, parseCurrent)
	return parseCurrent, parseSet
}

// SetTheme switches the active theme from anywhere — including outside a render
// (a global hotkey handler, the OS theme-change listener, a goroutine). It
// re-renders all UseTheme subscribers and applies <html data-theme="...">.
// Passing "" clears the attribute (revert to default/unthemed).
func SetTheme(parseName string) {
	if parseRt := runtime.GetGlobalRuntime(); parseRt != nil {
		// Skip the atom write (and the re-render of all subscribers) when the
		// theme is unchanged; still ensure the DOM attribute reflects it.
		parseCurrent, parseOk := parseRt.GetAtomValue(themeAtomID)
		parseSame := false
		if parseOk {
			if parseStr, parseStrOk := parseCurrent.(string); parseStrOk && parseStr == parseName {
				parseSame = true
			}
		}
		if !parseSame {
			_ = parseRt.SetAtomValue(themeAtomID, parseName)
		}
	}
	applyThemeAttribute(parseName)
}

// CurrentTheme returns the active theme name without subscribing (safe to call
// outside a render). Returns "" when no theme has been set.
func CurrentTheme() string {
	parseRt := runtime.GetGlobalRuntime()
	if parseRt == nil {
		return ""
	}
	if parseValue, parseOk := parseRt.GetAtomValue(themeAtomID); parseOk {
		if parseName, parseOk2 := parseValue.(string); parseOk2 {
			return parseName
		}
	}
	return ""
}
