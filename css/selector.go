package css

import (
	"strconv"
	"strings"
)

// Selector is a typed CSS selector fragment used as the target of a combinator
// (Child/Descendant/…) or a functional pseudo-class (Not/Has/Is). Build one with
// the typed constructors — never a free string — so composition stays checkable.
type Selector string

func (s Selector) String() string { return string(s) }

// El targets an HTML element by tag name: css.El("h3") -> "h3".
func El(parseTag string) Selector { return Selector(parseTag) }

// SheetRef targets another generated class, composing one Sheet against another with
// full type-safety: css.SheetRef(titleSheet) -> ".c-abc".
func SheetRef(parseSheet Sheet) Selector { return Selector("." + string(parseSheet)) }

// ClassSel targets a literal class name (interop with existing string classes):
// css.ClassSel("title") -> ".title".
func ClassSel(parseName string) Selector {
	return Selector("." + strings.TrimPrefix(parseName, "."))
}

// Attr targets an attribute-presence selector: css.AttrSel("data-open") -> "[data-open]".
func AttrSel(parseName string) Selector { return Selector("[" + parseName + "]") }

// AttrEq targets an attribute-equals selector: css.AttrEq("type","submit") ->
// `[type="submit"]`.
func AttrEq(parseName, parseValue string) Selector {
	return Selector("[" + parseName + "=\"" + parseValue + "\"]")
}

// Sel is the explicit selector-template escape hatch for a fragment the typed
// builders don't cover. The fragment is appended after the class anchor; use "&"
// only via DefineVariant for parent-context selectors. Greppable on purpose.
func Sel(parseFragment string) Selector { return Selector(parseFragment) }

// --- combinators --------------------------------------------------------------
//
// Each combinator scopes the inner rules under "& <combinator> <target>", folded
// into the same hashed class (no extra registry entries). Composition follows
// nesting order, e.g. Hover(Child(El("a"), …)) -> ".c:hover > a".

// Child scopes rules to a direct child: & > target.
func Child(target Selector, rules ...Rule) []Rule {
	return applyVariant("& > "+string(target), "", rules)
}

// Descendant scopes rules to any descendant: & target.
func Descendant(target Selector, rules ...Rule) []Rule {
	return applyVariant("& "+string(target), "", rules)
}

// Adjacent scopes rules to the next sibling: & + target.
func Adjacent(target Selector, rules ...Rule) []Rule {
	return applyVariant("& + "+string(target), "", rules)
}

// Sibling scopes rules to any following sibling: & ~ target.
func Sibling(target Selector, rules ...Rule) []Rule {
	return applyVariant("& ~ "+string(target), "", rules)
}

// --- functional / embedded pseudo-classes -------------------------------------

// Not scopes rules to :not(selector).
func Not(target Selector, rules ...Rule) []Rule {
	return applyVariant("&:not("+string(target)+")", "", rules)
}

// Has scopes rules to :has(selector).
func Has(target Selector, rules ...Rule) []Rule {
	return applyVariant("&:has("+string(target)+")", "", rules)
}

// Is scopes rules to :is(a, b, …).
func Is(targets []Selector, rules ...Rule) []Rule {
	parts := make([]string, 0, len(targets))
	for _, t := range targets {
		parts = append(parts, string(t))
	}
	return applyVariant("&:is("+strings.Join(parts, ", ")+")", "", rules)
}

// NthArg is a typed argument to NthChild/NthOfType: Odd, Even, or AnB(a,b).
type NthArg string

const (
	Odd  NthArg = "odd"
	Even NthArg = "even"
)

// AnB builds an An+B nth-child argument: AnB(3,1) -> "3n+1".
func AnB(a, b int) NthArg {
	sign := "+"
	if b < 0 {
		sign = "-"
		b = -b
	}
	return NthArg(strconv.Itoa(a) + "n" + sign + strconv.Itoa(b))
}

// Nth builds a literal nth argument from an index: Nth(2) -> "2".
func Nth(n int) NthArg { return NthArg(strconv.Itoa(n)) }

// NthChild scopes rules to :nth-child(arg).
func NthChild(arg NthArg, rules ...Rule) []Rule {
	return applyVariant("&:nth-child("+string(arg)+")", "", rules)
}

// NthOfType scopes rules to :nth-of-type(arg).
func NthOfType(arg NthArg, rules ...Rule) []Rule {
	return applyVariant("&:nth-of-type("+string(arg)+")", "", rules)
}
