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
	// declHash is a content hash of decls, computed once at construction so the
	// fold fast path (fastfold.go) costs O(rules) instead of O(total bytes).
	//
	// ZERO MEANS "NOT COMPUTED", not "empty". Any constructor that builds a Rule
	// literal without setting it, or that mutates decls afterwards, must leave or
	// reset it to 0 — computeFoldKey then hashes that rule's contents the slow way,
	// which is correct, just not fast. A STALE non-zero hash is the one thing that
	// would be wrong, so when in doubt set it to 0.
	//
	// It deliberately covers only decls, not scope: variant helpers copy a rule and
	// rewrite its scope, and excluding scope means those copies keep a valid hash
	// instead of silently carrying a stale one.
	declHash uint64
}

// hashDecls computes the content hash stored in Rule.declHash. It must stay in
// step with how computeFoldKey mixes declarations, or a precomputed hash and a
// fallback hash would disagree for identical content and the same rule-set would
// fold to two different classes depending on how it was built.
func hashDecls(parseDecls []declaration) uint64 {
	parseHash := fnvOffset1
	for _, parseDecl := range parseDecls {
		for i := 0; i < len(parseDecl.property); i++ {
			parseHash = (parseHash ^ uint64(parseDecl.property[i])) * fnvPrime1
		}
		parseHash = (parseHash ^ 0xfe) * fnvPrime1
		for i := 0; i < len(parseDecl.value); i++ {
			parseHash = (parseHash ^ uint64(parseDecl.value[i])) * fnvPrime1
		}
		parseHash = (parseHash ^ 0xfd) * fnvPrime1
	}
	// Never hand back 0 from a real computation: 0 is the "not computed" sentinel,
	// and a rule that legitimately hashed to 0 would silently take the slow path
	// forever. Mapping it to 1 costs nothing and keeps the sentinel unambiguous.
	if parseHash == 0 {
		return 1
	}
	return parseHash
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
	parseDecls := []declaration{{property: property, value: value}}
	return Rule{decls: parseDecls, declHash: hashDecls(parseDecls)}
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
	// The values just changed, so the copied declHash describes the ORIGINAL rule.
	// Recompute rather than zero it: MarkImportant is used inside bundles that get
	// folded on every render, and leaving it at 0 would push those rules onto the
	// slow hashing path for the life of the process.
	next.declHash = hashDecls(next.decls)
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
//
// TWO CONSEQUENCES OF THAT DETERMINISM ARE INTENTIONAL AND LOAD-BEARING. Both are
// surprising the first time they bite; neither is a bug, and "fixing" either one
// would break class-name stability, which the determinism fuzz test and the class
// churn watchdog exist to protect.
//
//  1. BLOCKS ARE SORTED BY (at-rule, selector), NOT BY SOURCE ORDER. Within one
//     folded class, source order therefore cannot break a specificity tie: two
//     blocks at equal specificity always resolve in sorted order, whatever order the
//     author wrote them in. This is what makes the fold reorderable — New(a, b) and
//     New(b, a) MUST produce the same class — and it is also usable as a tool: the
//     empty at-rule sorts before "@media …", so a media block reliably lands after
//     the unconditional one and wins on order without !important. (Whether one
//     PSEUDO-class beats another is still specificity, not this sort. When you need a
//     guaranteed order across selectors, reach for cascade layers via Layer.)
//
//  2. DECLARATIONS ARE SORTED BY PROPERTY NAME WITHIN A BLOCK. Shorthand-then-
//     longhand works by construction, because every longhand of a shorthand starts
//     with the shorthand's own name and so sorts after it ("border" < "border-style",
//     "background-image" after "background"). What it CANNOT express is one longhand
//     overriding a sibling longhand: "border-bottom" sorts before "border-top", so
//     border-bottom can never beat border-top inside a single block, no matter which
//     was written first. Two rules that fight over the same physical side are
//     unorderable here by design — express the intent with one declaration per side,
//     or separate the losing rule into its own scope (a variant, or a Layer).
//
// A repeated property within the same scope is still last-write-wins at fold time
// (see the bucket assignment below), so the sort applies to distinct properties only.
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
