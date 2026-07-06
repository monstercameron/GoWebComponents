package css

import (
	"sort"
	"strings"
)

// Rule is one typed CSS declaration (or a small group of declarations) optionally
// wrapped in a variant scope (a pseudo-selector and/or one or more at-rules).
//
// Rules are opaque values: build them with the typed property constructors
// (css.Gap, css.Bg, the css.Display namespace, …), wrap them with variants
// (css.Hover, css.Media, …), and fold a set of them into a single hashed class
// with css.New. A Rule carries no class identity of its own — identity is minted
// at New time from the canonical serialization of the folded rule-set.
type Rule struct {
	decls []declaration
	scope scope
	// raw holds complete, class-independent CSS blocks (e.g. an @keyframes body)
	// that must travel with the rule and participate in hashing/emission.
	raw []string
}

// declaration is a single "property: value" pair.
type declaration struct {
	property string
	value    string
}

// scope captures the variant wrapping for a rule's declarations. selectorTemplate
// is a selector with a single "&" placeholder that stands in for the generated
// class selector (default "&"); atRules are at-rule headers wrapping the block,
// innermost first.
type scope struct {
	selectorTemplate string
	atRules          []string
}

func (s scope) template() string {
	if s.selectorTemplate == "" {
		return "&"
	}
	return s.selectorTemplate
}

// decl is the internal constructor shared by every typed property.
func decl(property, value string) Rule {
	return Rule{decls: []declaration{{property: property, value: value}}}
}

// MarkImportant returns a copy of the rule with "!important" appended to every
// declaration value (idempotent). It is the typed analog of Tailwind's "!"
// modifier.
func MarkImportant(parseRule Rule) Rule {
	next := parseRule
	next.decls = make([]declaration, len(parseRule.decls))
	for i, d := range parseRule.decls {
		value := d.value
		if !strings.Contains(value, "!important") {
			value += " !important"
		}
		next.decls[i] = declaration{property: d.property, value: value}
	}
	return next
}

// applyVariant returns copies of rules with an outer selector template and/or an
// at-rule layered on. Composition substitutes the OUTER template into the inner
// rule's "&" — so nesting order equals selector order, exactly like SCSS:
//
//	Hover(Descendant(h3, …))  →  inner "& h3", outer "&:hover"  →  "&:hover h3"
//	Descendant(h3, Hover(…))  →  inner "&:hover", outer "& h3"  →  "& h3:hover"
//
// (Pure pseudo-stacking like md(hover(x)) is direction-insensitive.)
func applyVariant(selectorTemplate, atRule string, rules []Rule) []Rule {
	out := make([]Rule, 0, len(rules))
	for _, r := range rules {
		next := r
		if selectorTemplate != "" {
			next.scope.selectorTemplate = strings.ReplaceAll(r.scope.template(), "&", selectorTemplate)
		}
		if atRule != "" {
			next.scope.atRules = append(append([]string{}, r.scope.atRules...), atRule)
		}
		out = append(out, next)
	}
	return out
}

// groupKey is the canonical scope signature used to bucket declarations that
// share the same selector/at-rule scope.
type groupKey struct {
	selectorTemplate string
	atRules          string
}

// atRuleSep joins/splits a scope's stacked at-rules. It is the ASCII unit
// separator, which cannot legally appear in a CSS at-rule — the old "||"
// delimiter collided with an at-rule value that literally contained "||",
// merging unrelated rules and mis-splitting the nesting on render.
const atRuleSep = "\x1f"

// canonicalize folds rules into a deterministic CSS-source representation plus the
// ordered list of raw blocks. The representation is selector-template-relative
// (uses "&", not the final class) so it is identity-free and stable: identical
// rule-sets in any order produce identical text, which is what the content hash
// keys on.
func canonicalize(rules []Rule) (canonical string, groups []renderGroup, raws []string) {
	buckets := map[groupKey]map[string]string{}
	order := []groupKey{}
	seenRaw := map[string]bool{}

	for _, r := range rules {
		for _, block := range r.raw {
			if !seenRaw[block] {
				seenRaw[block] = true
				raws = append(raws, block)
			}
		}
		if len(r.decls) == 0 {
			continue
		}
		key := groupKey{
			selectorTemplate: r.scope.template(),
			atRules:          strings.Join(r.scope.atRules, atRuleSep),
		}
		bucket, ok := buckets[key]
		if !ok {
			bucket = map[string]string{}
			buckets[key] = bucket
			order = append(order, key)
		}
		for _, d := range r.decls {
			// last write wins for a repeated property within the same scope.
			bucket[d.property] = d.value
		}
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].atRules != order[j].atRules {
			return order[i].atRules < order[j].atRules
		}
		return order[i].selectorTemplate < order[j].selectorTemplate
	})
	sort.Strings(raws)

	var b strings.Builder
	for _, key := range order {
		bucket := buckets[key]
		props := make([]string, 0, len(bucket))
		for p := range bucket {
			props = append(props, p)
		}
		sort.Strings(props)

		var body strings.Builder
		for _, p := range props {
			body.WriteString(p)
			body.WriteByte(':')
			body.WriteString(bucket[p])
			body.WriteByte(';')
		}
		groups = append(groups, renderGroup{
			selectorTemplate: key.selectorTemplate,
			atRules:          splitAtRules(key.atRules),
			body:             body.String(),
		})

		b.WriteString(key.atRules)
		b.WriteByte('{')
		b.WriteString(key.selectorTemplate)
		b.WriteByte('{')
		b.WriteString(body.String())
		b.WriteString("}}")
	}
	for _, raw := range raws {
		b.WriteString(raw)
	}
	return b.String(), groups, raws
}

func splitAtRules(joined string) []string {
	if joined == "" {
		return nil
	}
	return strings.Split(joined, atRuleSep)
}

// renderGroup is a single emittable block keyed to the (not-yet-known) class.
type renderGroup struct {
	selectorTemplate string
	atRules          []string
	body             string
}

// render produces the final CSS text for a group given the resolved class name.
func (g renderGroup) render(class string) string {
	selector := strings.ReplaceAll(g.selectorTemplate, "&", "."+class)
	block := selector + "{" + g.body + "}"
	for _, at := range g.atRules { // innermost first
		block = at + "{" + block + "}"
	}
	return block
}
