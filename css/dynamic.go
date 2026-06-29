package css

import "github.com/monstercameron/GoWebComponents/html"

// Dynamic pairs a static, hashed class that references a CSS custom property with
// the inline StyleVar PropOption that carries the live value. This is the
// static/runtime boundary: a rule built from a runtime value would mint a new
// class per distinct value and grow the registry unboundedly, so instead the
// class references var(--name) (one stable hashed class) and the per-element
// value is set inline.
//
//	d := css.Dynamic("--gap", "gap", state.Gap())   // state.Gap() is a css.Length
//	Div(d.Class(), d.Style(), …)                     // class is stable; value is live
//
// Construct one with DynamicLength (typed Length value) or DynamicVar (any string value);
// there is no NewDynamic — the type-specific factories are the intended constructors.
type Dynamic struct {
	varName  string
	property string
	value    string
}

// DynamicLength declares a class that sets parseProperty to var(--parseName), plus
// the live Length value to bind inline. parseName is normalized to a "--" custom
// property.
func DynamicLength(parseName, parseProperty string, parseValue Length) Dynamic {
	return DynamicVar(parseName, parseProperty, string(parseValue))
}

// DynamicVar builds a Dynamic for any property/value pair.
func DynamicVar(parseName, parseProperty, parseValue string) Dynamic {
	if len(parseName) < 2 || parseName[:2] != "--" {
		parseName = "--" + parseName
	}
	return Dynamic{varName: parseName, property: parseProperty, value: parseValue}
}

// Rule returns the static Layer-1 rule that references the custom property. Fold
// it into New alongside other rules, or use Class for a one-shot PropOption.
func (d Dynamic) Rule() Rule {
	return decl(d.property, "var("+d.varName+")")
}

// Class returns the PropOption that applies the var-referencing class.
func (d Dynamic) Class() html.PropOption {
	return Class(d.Rule())
}

// Style returns the PropOption that sets the live custom-property value inline,
// keeping the class and value in sync per element.
func (d Dynamic) Style() html.PropOption {
	return html.StyleVar(d.varName, d.value)
}
