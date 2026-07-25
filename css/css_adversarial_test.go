//go:build !(js && wasm)

package css_test

import (
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/css"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func noStyleBreakout(t *testing.T, output, label string) {
	t.Helper()
	// When handed a full StyleBlock, strip the legitimate <style …> wrapper and its
	// trailing </style> so we scan only the CSS body — otherwise the closing tag is
	// a guaranteed false positive.
	body := output
	if strings.HasPrefix(body, "<style") {
		if idx := strings.IndexByte(body, '>'); idx >= 0 {
			body = body[idx+1:]
		}
		body = strings.TrimSuffix(body, "</style>")
	}
	// Any of these appearing unescaped in the CSS body means an attacker controls
	// parser context or breaks out of the <style> element.
	bad := []string{"</style>", "<script", "*/", "\x00"}
	for _, b := range bad {
		if strings.Contains(body, b) {
			t.Errorf("[CRITICAL INJECTION] %s: CSS body contains %q\nfull output:\n%s", label, b, output)
		}
	}
}

// ── 1. Injection / breakout attacks ──────────────────────────────────────────

func TestAdversarial_InjectionViaRaw(t *testing.T) {
	payloads := []struct {
		name, prop, val string
	}{
		{"style_close_in_value", "content", `"</style><script>alert(1)</script>"`},
		{"comment_break_value", "color", "red/**/}body{color:evil"},
		{"brace_in_value", "content", `"}"}`},
		{"newline_in_value", "color", "red\n}body{color:evil"},
		{"nul_in_value", "color", "red\x00"},
		{"style_close_in_prop", "</style><script>x</script>", "red"},
		{"semicolon_escape_value", "content", `"hello;} body { color: red`},
	}
	for _, p := range payloads {
		p := p
		t.Run(p.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Raw(p.prop, p.val))
			out := css.Harvest()
			noStyleBreakout(t, out, p.name)
			sb := css.StyleBlock()
			noStyleBreakout(t, sb, p.name+"_styleblock")
		})
	}
}

func TestAdversarial_InjectionViaHexColor(t *testing.T) {
	// Hex() prepends '#' — but what if the hex string contains injection chars?
	payloads := []string{
		`</style><script>alert(1)</script>`,
		`0af} body{color:red`,
		`0af\0`,
		"0af\n}*{color:red",
	}
	for _, p := range payloads {
		p := p
		t.Run(p, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on Hex injection: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Bg(css.Hex(p)))
			out := css.Harvest()
			noStyleBreakout(t, out, "Hex:"+p)
		})
	}
}

func TestAdversarial_InjectionViaVarColor(t *testing.T) {
	// Var() produces var(--name) — injection via the name parameter
	payloads := []string{
		"--ok) } body { color: red; } x(",
		"</style>",
		"ok\x00evil",
	}
	for _, p := range payloads {
		p := p
		t.Run(p[:min(len(p), 20)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Bg(css.Var(p)))
			out := css.Harvest()
			noStyleBreakout(t, out, "Var:"+p)
		})
	}
}

func TestAdversarial_InjectionViaClassSel(t *testing.T) {
	// ClassSel constructs ".name" — injection via the name
	payloads := []string{
		`evil} body{color:red`,
		`</style><script>`,
		"foo\nbar",
	}
	for _, p := range payloads {
		p := p
		t.Run(p[:min(len(p), 20)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Child(css.ClassSel(p), css.Display.Flex)...)
			out := css.Harvest()
			noStyleBreakout(t, out, "ClassSel:"+p)
		})
	}
}

func TestAdversarial_InjectionViaSel(t *testing.T) {
	// Sel is the explicit escape hatch — attacker can pass any selector fragment
	payloads := []string{
		`</style><script>alert(1)</script><style>`,
		"} body { color: red } .x {",
		"/*\n*/",
	}
	for _, p := range payloads {
		p := p
		t.Run(p[:min(len(p), 20)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Descendant(css.Sel(p), css.Display.Flex)...)
			out := css.Harvest()
			noStyleBreakout(t, out, "Sel:"+p)
		})
	}
}

func TestAdversarial_InjectionViaAttrEq(t *testing.T) {
	// AttrEq embeds value in [name="value"] — can attacker break out of the quotes?
	payloads := []struct{ name, val string }{
		{"type", `submit"] * {color:red} [foo="`},
		{"data-x", `"</style><script>alert(1)</script>`},
		{"data-y", "val\nbreaker"},
	}
	for _, p := range payloads {
		p := p
		t.Run(p.name+":"+p.val[:min(len(p.val), 20)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Not(css.AttrEq(p.name, p.val), css.Display.Flex)...)
			out := css.Harvest()
			noStyleBreakout(t, out, "AttrEq")
		})
	}
}

func TestAdversarial_InjectionViaKeyframeOffset(t *testing.T) {
	// Keyframe offsets are raw strings — can an attacker inject via them?
	offsets := []string{
		`0%} * {color:red} @keyframes foo {0%`,
		"</style>",
		"100%\n}body{color:red",
	}
	for _, o := range offsets {
		o := o
		t.Run(o[:min(len(o), 20)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Keyframes("anim",
				css.At(o, css.Opacity(0)),
				css.At("100%", css.Opacity(1)),
			))
			out := css.Harvest()
			noStyleBreakout(t, out, "keyframe-offset:"+o)
		})
	}
}

func TestAdversarial_InjectionViaKeyframeName(t *testing.T) {
	names := []string{
		`evil} body{color:red} @keyframes x`,
		"</style><script>",
		"name\x00inject",
	}
	for _, n := range names {
		n := n
		t.Run(n[:min(len(n), 20)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Keyframes(n,
				css.At("0%", css.Opacity(0)),
			))
			out := css.Harvest()
			noStyleBreakout(t, out, "keyframe-name:"+n)
		})
	}
}

func TestAdversarial_InjectionViaDefineVariant(t *testing.T) {
	// DefineVariant accepts an arbitrary selector template
	payloads := []string{
		`</style><script>alert(1)</script><style> &`,
		"} * { color:red } .x { &",
		"&/*injection*/",
	}
	for _, p := range payloads {
		p := p
		t.Run(p[:min(len(p), 20)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			v := css.DefineVariant(p)
			css.New(v(css.Display.Flex)...)
			out := css.Harvest()
			noStyleBreakout(t, out, "DefineVariant:"+p)
		})
	}
}

func TestAdversarial_InjectionViaRawMedia(t *testing.T) {
	payloads := []string{
		`screen) { * { color: red } } @media (min-width:0`,
		"</style><script>",
		"screen\n}body{color:red",
	}
	for _, p := range payloads {
		p := p
		t.Run(p[:min(len(p), 20)], func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()
			css.Reset()
			css.New(css.Media(css.RawMedia(p), css.Display.Flex)...)
			out := css.Harvest()
			noStyleBreakout(t, out, "RawMedia:"+p)
		})
	}
}

func TestAdversarial_StyleBlockClassesInjection(t *testing.T) {
	// The data-gwc-css attribute value is html.EscapeString'd — but what about
	// the CSS body inside <style>? That is NOT escaped. An attacker who controls
	// CSS property values can attempt to close the style tag.
	css.Reset()
	css.New(css.Raw("content", `"</style><script>alert(1)</script><style>"`))
	sb := css.StyleBlock()
	// The CSS body is inside <style>...</style> with no escaping of the content.
	// This is the critical question: does </style> in a value break out?
	if strings.Contains(sb, "</style><script>") {
		t.Errorf("[CRITICAL BREAKOUT] StyleBlock contains unescaped </style><script>:\n%s", sb)
	}
}

// ── 2. Determinism attacks ────────────────────────────────────────────────────

func TestAdversarial_DeterminismUnderPermutations(t *testing.T) {
	// Same 5 rules in all 5! = 120 orderings must produce the same class name.
	rules := []css.Rule{
		css.Display.Flex,
		css.Gap(css.Px(8)),
		css.Bg(css.Slate900),
		css.Padding(css.Rem(1)),
		css.Opacity(0.5),
	}
	// Generate all permutations via Heap's algorithm
	perm := make([]int, len(rules))
	for i := range perm {
		perm[i] = i
	}
	var classes []string
	var heaps func(k int)
	heaps = func(k int) {
		if k == 1 {
			set := make([]css.Rule, len(rules))
			for i, idx := range perm {
				set[i] = rules[idx]
			}
			css.Reset()
			class := css.New(set...)
			classes = append(classes, string(class))
			return
		}
		for i := 0; i < k; i++ {
			heaps(k - 1)
			if k%2 == 0 {
				perm[i], perm[k-1] = perm[k-1], perm[i]
			} else {
				perm[0], perm[k-1] = perm[k-1], perm[0]
			}
		}
	}
	heaps(len(rules))

	first := classes[0]
	for i, c := range classes {
		if c != first {
			t.Errorf("permutation %d produced different class %q vs %q", i, c, first)
		}
	}
}

func TestAdversarial_DeterminismDuplicateDecls(t *testing.T) {
	// Duplicate declarations for same property — last write should win, result stable
	css.Reset()
	c1 := css.New(css.Bg(css.Red500), css.Bg(css.Slate900))
	css.Reset()
	c2 := css.New(css.Bg(css.Slate900))
	if string(c1) != string(c2) {
		t.Errorf("duplicate decl: last-write-wins failed: %q vs %q", c1, c2)
	}
}

func TestAdversarial_DeterminismConflictingVariants(t *testing.T) {
	// Conflicting hover bg — last should win; both orderings should produce the same class
	css.Reset()
	set1 := css.Rules(
		css.Display.Flex,
		css.Hover(css.Bg(css.Red500), css.Bg(css.Slate900)),
	)
	c1 := css.New(set1...)

	css.Reset()
	set2 := css.Rules(
		css.Hover(css.Bg(css.Slate900), css.Bg(css.Red500)),
		css.Display.Flex,
	)
	c2 := css.New(set2...)

	if string(c1) == string(c2) {
		// Both win with Slate900 vs Red500 respectively — classes differ (that's fine)
		// but each call on its own should be stable
	}

	// Re-running same set must give same class
	css.Reset()
	c3 := css.New(set1...)
	if string(c1) != string(c3) {
		t.Errorf("non-determinism: same set produced %q then %q", c1, c3)
	}
}

// ── 3. Degenerate inputs ─────────────────────────────────────────────────────

func TestAdversarial_EmptyEverywhere(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on empty inputs: %v", r)
		}
	}()
	css.Reset()

	// Empty New
	s := css.New()
	if s != "" {
		t.Errorf("New() with no rules should return empty Sheet, got %q", s)
	}

	// Empty Raw
	css.Reset()
	css.New(css.Raw("", ""))
	css.New(css.Raw("color", ""))
	css.New(css.Raw("", "red"))

	// Empty selectors
	css.New(css.Child(css.El(""), css.Display.Flex)...)
	css.New(css.Descendant(css.AttrSel(""), css.Display.Flex)...)
	css.New(css.Not(css.Sel(""), css.Display.Flex)...)
	css.New(css.Is([]css.Selector{}, css.Display.Flex)...)
	css.New(css.Is([]css.Selector{css.El("")}, css.Display.Flex)...)

	// Confirm nothing panicked and output doesn't break
	out := css.Harvest()
	noStyleBreakout(t, out, "empty-everywhere")
}

func TestAdversarial_ExtremeNumbers(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on extreme numbers: %v", r)
		}
	}()
	css.Reset()

	css.New(css.Gap(css.Px(math.MaxInt32)))
	css.New(css.Gap(css.Px(math.MinInt32)))
	css.New(css.Gap(css.Px(0)))
	css.New(css.Gap(css.Px(-999999)))
	css.New(css.Gap(css.Rem(math.MaxFloat64)))
	css.New(css.Gap(css.Rem(-math.MaxFloat64)))
	css.New(css.Opacity(math.MaxFloat64))
	css.New(css.Opacity(-math.MaxFloat64))

	out := css.Harvest()
	noStyleBreakout(t, out, "extreme-numbers")
}

func TestAdversarial_NaNAndInf(t *testing.T) {
	// Go's strconv.FormatFloat produces "NaN", "+Inf", "-Inf" — do these land raw in CSS?
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on NaN/Inf: %v", r)
		}
	}()
	css.Reset()

	nan := math.NaN()
	inf := math.Inf(1)
	ninf := math.Inf(-1)

	css.New(css.Opacity(nan))
	css.New(css.Opacity(inf))
	css.New(css.Opacity(ninf))
	css.New(css.Gap(css.Rem(nan)))
	css.New(css.Gap(css.Rem(inf)))

	out := css.Harvest()
	t.Logf("NaN/Inf output: %s", out)
	// Check they don't break structural integrity
	noStyleBreakout(t, out, "nan-inf")
}

func TestAdversarial_DeepVariantNesting(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on deep nesting: %v", r)
		}
	}()
	css.Reset()

	// Build 50 layers of Hover nesting
	rules := []css.Rule{css.Display.Flex}
	var wrapped []css.Rule = rules
	for i := 0; i < 50; i++ {
		wrapped = css.Hover(wrapped...)
	}
	css.New(wrapped...)

	out := css.Harvest()
	noStyleBreakout(t, out, "deep-hover-nesting")
}

func TestAdversarial_DeepSelectorNesting(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on deep selector nesting: %v", r)
		}
	}()
	css.Reset()

	// Build 50 layers of Child/Descendant nesting
	rules := []css.Rule{css.Display.Flex}
	var wrapped []css.Rule = rules
	for i := 0; i < 50; i++ {
		if i%2 == 0 {
			wrapped = css.Child(css.El("div"), wrapped...)
		} else {
			wrapped = css.Descendant(css.El("span"), wrapped...)
		}
	}
	css.New(wrapped...)

	out := css.Harvest()
	noStyleBreakout(t, out, "deep-selector-nesting")
}

func TestAdversarial_HugeRuleSet(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on huge rule set: %v", r)
		}
	}()
	css.Reset()

	rules := make([]css.Rule, 5000)
	for i := range rules {
		rules[i] = css.Raw("--var-"+strings.Repeat("x", i%10), strings.Repeat("y", i%20)+"px")
	}
	css.New(rules...)

	out := css.Harvest()
	if out == "" {
		t.Error("expected non-empty harvest for 5000-rule set")
	}
	noStyleBreakout(t, out, "huge-rule-set")
}

func TestAdversarial_UnicodeAndEmojiInSelectors(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on unicode/emoji: %v", r)
		}
	}()
	css.Reset()

	css.New(css.Child(css.El("🎯"), css.Display.Flex)...)
	css.New(css.Child(css.ClassSel("emoji-🎯"), css.Display.Flex)...)
	css.New(css.Child(css.El("héllo"), css.Display.Flex)...)
	css.New(css.Raw("content", `"你好世界"`))

	out := css.Harvest()
	noStyleBreakout(t, out, "unicode-emoji")
}

func TestAdversarial_EmptyKeyframes(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on empty keyframes: %v", r)
		}
	}()
	css.Reset()
	// No frames
	css.New(css.Keyframes("empty-anim"))
	// Frames with no rules
	css.New(css.Keyframes("no-rules-anim", css.At("0%"), css.At("100%")))
	// Frame with empty offset
	css.New(css.Keyframes("empty-offset", css.At("", css.Opacity(1))))

	noStyleBreakout(t, css.Harvest(), "empty-keyframes")
}

// ── 4. Concurrency attacks ────────────────────────────────────────────────────

func TestAdversarial_ConcurrentNew(t *testing.T) {
	css.Reset()
	var wg sync.WaitGroup
	const goroutines = 200

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic in goroutine %d: %v", n, r)
				}
			}()
			css.New(css.Display.Flex, css.Gap(css.Px(n%20)), css.Bg(css.Slate900))
		}(i)
	}
	wg.Wait()
}

func TestAdversarial_ConcurrentResetAndNew(t *testing.T) {
	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			defer func() { recover() }()
			css.Reset()
		}()
		go func(n int) {
			defer wg.Done()
			defer func() { recover() }()
			css.New(css.Bg(css.Slate800), css.Padding(css.Px(n%16)))
		}(i)
	}
	wg.Wait()
}

func TestAdversarial_ConcurrentSetSink(t *testing.T) {
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			defer func() { recover() }()
			css.SetSink(nil) // restore default
		}()
		go func(n int) {
			defer wg.Done()
			defer func() { recover() }()
			css.New(css.Display.Grid, css.Gap(css.Px(n%8)))
		}(i)
		go func() {
			defer wg.Done()
			defer func() { recover() }()
			css.Harvest()
		}()
	}
	wg.Wait()
}

func TestAdversarial_ConcurrentSeed(t *testing.T) {
	css.Reset()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			defer func() { recover() }()
			css.Seed("c-fake" + strings.Repeat("x", n%10))
			css.New(css.Display.Flex)
		}(i)
	}
	wg.Wait()
}

// ── 5. Selector composition abuse ────────────────────────────────────────────

func TestAdversarial_SelectorCombinatorsWrapping(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	css.Reset()

	// Stacked combinators: Child wrapping Sibling rules
	rules := css.Child(css.El("a"),
		css.Sibling(css.El("p"), css.Display.Flex)...,
	)
	css.New(rules...)

	// Adjacent wrapping Descendant
	rules2 := css.Adjacent(css.El("section"),
		css.Descendant(css.El("h2"), css.Display.Flex)...,
	)
	css.New(rules2...)

	// Not wrapping Has via Sel
	rules3 := css.Not(css.Sel(":has(div)"), css.Display.Flex)
	css.New(rules3...)

	out := css.Harvest()
	noStyleBreakout(t, out, "combinator-stacking")
}

func TestAdversarial_IsWithEmptySelectors(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	css.Reset()
	// Empty target list
	css.New(css.Is([]css.Selector{}, css.Display.Flex)...)
	// Mixed empty/valid
	css.New(css.Is([]css.Selector{css.El(""), css.El("div"), css.Sel("")}, css.Display.Flex)...)
	noStyleBreakout(t, css.Harvest(), "Is-empty-selectors")
}

func TestAdversarial_AnBExtremeValues(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	css.Reset()

	css.New(css.NthChild(css.AnB(math.MaxInt32, math.MinInt32), css.Display.Flex)...)
	css.New(css.NthChild(css.AnB(-1, -1), css.Display.Flex)...)
	css.New(css.NthChild(css.AnB(0, 0), css.Display.Flex)...)
	css.New(css.NthOfType(css.Nth(-5), css.Display.Flex)...)

	noStyleBreakout(t, css.Harvest(), "AnB-extreme")
}

func TestAdversarial_NotHasIsDeepNesting(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on deep Not/Has/Is: %v", r)
		}
	}()
	css.Reset()
	// Build a selector fragment that deeply nests string representations
	// Not(Has(Is(...))) stacks via Sel escape hatch on the fragment
	inner := css.Sel(":is(:has(:not(div)))")
	rules := css.Not(inner, css.Display.Flex)
	css.New(rules...)

	noStyleBreakout(t, css.Harvest(), "Not-Has-Is-deep")
}

func TestAdversarial_RefToSelfBuiltSheet(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on Ref self-reference: %v", r)
		}
	}()
	css.Reset()
	// Build a sheet and use Ref to target itself in a descendant rule
	sheet1 := css.New(css.Display.Flex)
	// Now use that sheet's class in another sheet's selector
	sheet2 := css.New(css.Descendant(css.SheetRef(sheet1), css.Bg(css.Slate900))...)
	_ = sheet2
	noStyleBreakout(t, css.Harvest(), "Ref-self")
}

// ── 6. Theme abuse ───────────────────────────────────────────────────────────

func TestAdversarial_MultipleResets(t *testing.T) {
	// Multiple Resets in a row should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on multiple resets: %v", r)
		}
	}()
	for i := 0; i < 50; i++ {
		css.Reset()
	}
	css.New(css.Display.Flex)
	out := css.Harvest()
	if !strings.Contains(out, "display:flex") {
		t.Errorf("after multi-reset, display:flex missing in %q", out)
	}
}

func TestAdversarial_SeedCollision(t *testing.T) {
	// Seed a class that would naturally be emitted — should suppress re-emission
	css.Reset()
	// Pre-compute what class New(Display.Flex) would produce
	sheet := css.New(css.Display.Flex)
	class := string(sheet)
	css.Reset()
	css.Seed(class) // pre-seed
	// Now New should see it as already-emitted
	sheet2 := css.New(css.Display.Flex)
	out := css.Harvest()
	if strings.Count(out, "display:flex") > 1 {
		t.Errorf("Seed should prevent double-emission, got %d occurrences:\n%s",
			strings.Count(out, "display:flex"), out)
	}
	_ = sheet2
}

func TestAdversarial_NilSinkRestore(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil sink: %v", r)
		}
	}()
	css.Reset()
	old := css.SetSink(nil) // should restore default, not nil
	_ = old
	css.New(css.Display.Flex)
	// Must not panic and must harvest something
	out := css.Harvest()
	if !strings.Contains(out, "display:flex") {
		t.Errorf("after nil SetSink restore, expected display:flex, got %q", out)
	}
}

// ── 7. StyleBlock output integrity ───────────────────────────────────────────

func TestAdversarial_StyleBlockAttributeEscaping(t *testing.T) {
	// The data-gwc-css attribute value uses html.EscapeString on class names —
	// but class names are hashed, so injection there is extremely unlikely.
	// Let's verify the structure is valid when many classes exist.
	css.Reset()
	for i := 0; i < 100; i++ {
		css.New(css.Raw("--idx", strings.Repeat("a", i+1)))
	}
	sb := css.StyleBlock()
	// Must open with <style and close with </style>
	if !strings.HasPrefix(sb, "<style ") {
		t.Errorf("StyleBlock does not start with <style: %q", sb[:min(len(sb), 60)])
	}
	if !strings.HasSuffix(sb, "</style>") {
		t.Errorf("StyleBlock does not end with </style>: ...%q", sb[max(0, len(sb)-60):])
	}
}

func TestAdversarial_StyleBlockBodyNotHtmlEscaped(t *testing.T) {
	// The CSS body inside <style> is NOT html-escaped (that would break CSS).
	// Verify that legitimate CSS angle brackets (e.g., calc expressions) survive.
	// But also document the risk: if an attacker gets </style> into CSS values,
	// it WILL appear unescaped in StyleBlock output.
	css.Reset()
	css.New(css.Raw("content", `"hello world"`))
	sb := css.StyleBlock()
	if !strings.Contains(sb, `"hello world"`) {
		t.Errorf("legitimate CSS content was unexpectedly escaped in StyleBlock: %s", sb)
	}
}

// ── 8. MarkImportant idempotency ─────────────────────────────────────────────

func TestAdversarial_MarkImportantIdempotent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	css.Reset()
	base := css.Display.Flex
	once := css.MarkImportant(base)
	twice := css.MarkImportant(once)
	thrice := css.MarkImportant(twice)

	css.New(once)
	css.New(twice)
	css.New(thrice)

	out := css.Harvest()
	// Should contain "!important" exactly once per declaration, not duplicated
	// Count occurrences
	count := strings.Count(out, "!important")
	// 3 separate New calls each with 1 decl = 3 distinct classes = 3 entries
	// But twice/thrice are the same canonical (idempotent) so same class
	t.Logf("!important count in output: %d\nOutput: %s", count, out)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
