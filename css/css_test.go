//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
)

// reset gives each test a clean process-wide registry/sink.
func reset(t *testing.T) {
	t.Helper()
	css.Reset()
}

func TestNewIsDeterministicAndOrderIndependent(t *testing.T) {
	reset(t)
	a := css.New(css.Display.Flex, css.Gap(css.Px(8)), css.Padding(css.Px(16)))
	reset(t)
	b := css.New(css.Padding(css.Px(16)), css.Gap(css.Px(8)), css.Display.Flex)
	if a == "" {
		t.Fatal("New returned empty class for non-empty rule set")
	}
	if a != b {
		t.Fatalf("class should be order-independent: %q != %q", a, b)
	}
	if !strings.HasPrefix(string(a), "c-") {
		t.Fatalf("class should carry the c- prefix, got %q", a)
	}
}

func TestNewEmitsCanonicalCSS(t *testing.T) {
	reset(t)
	class := css.New(css.Display.Flex, css.Gap(css.Px(8)))
	got := css.Harvest()
	// declarations sorted by property within the block.
	want := "." + string(class) + "{display:flex;gap:8px;}"
	if got != want {
		t.Fatalf("unexpected CSS\nwant: %s\ngot:  %s", want, got)
	}
}

func TestDedupAcrossRepeatedCalls(t *testing.T) {
	reset(t)
	first := css.New(css.Bg(css.Slate900))
	for i := 0; i < 50; i++ {
		again := css.New(css.Bg(css.Slate900))
		if again != first {
			t.Fatalf("repeated New should return the same class: %q != %q", again, first)
		}
	}
	// Only one rule should have been emitted.
	if got := strings.Count(css.Harvest(), "{"); got != 1 {
		t.Fatalf("expected exactly one emitted block, got %d: %q", got, css.Harvest())
	}
	if classes := css.HarvestedClasses(); len(classes) != 1 {
		t.Fatalf("expected one harvested class, got %v", classes)
	}
}

func TestDistinctValuesProduceDistinctClasses(t *testing.T) {
	reset(t)
	a := css.New(css.Gap(css.Px(8)))
	b := css.New(css.Gap(css.Px(12)))
	if a == b {
		t.Fatalf("distinct values must hash to distinct classes: %q == %q", a, b)
	}
}

func TestHoverVariantEmitsPseudoSelector(t *testing.T) {
	reset(t)
	class := css.New(css.Rules(
		css.Bg(css.Slate900),
		css.Hover(css.Bg(css.Slate800)),
	)...)
	got := css.Harvest()
	if !strings.Contains(got, "."+string(class)+"{background-color:#0f172a;}") {
		t.Fatalf("missing base rule: %q", got)
	}
	if !strings.Contains(got, "."+string(class)+":hover{background-color:#1e293b;}") {
		t.Fatalf("missing hover rule: %q", got)
	}
}

func TestMediaVariantEmitsAtRule(t *testing.T) {
	reset(t)
	class := css.New(css.Media(css.MinW(768), css.Display.Flex)...)
	got := css.Harvest()
	want := "@media (min-width:768px){." + string(class) + "{display:flex;}}"
	if got != want {
		t.Fatalf("unexpected media CSS\nwant: %s\ngot:  %s", want, got)
	}
}

func TestVariantStackingNestsScopes(t *testing.T) {
	reset(t)
	class := css.New(css.Media(css.MinW(768), css.Hover(css.Display.Flex)...)...)
	got := css.Harvest()
	want := "@media (min-width:768px){." + string(class) + ":hover{display:flex;}}"
	if got != want {
		t.Fatalf("unexpected stacked CSS\nwant: %s\ngot:  %s", want, got)
	}
}

func TestDefineVariantGroupHover(t *testing.T) {
	reset(t)
	groupHover := css.DefineVariant(".group:hover &")
	class := css.New(groupHover(css.Display.Block)...)
	got := css.Harvest()
	want := ".group:hover ." + string(class) + "{display:block;}"
	if got != want {
		t.Fatalf("unexpected group-hover CSS\nwant: %s\ngot:  %s", want, got)
	}
}

func TestImportantAppendsBang(t *testing.T) {
	reset(t)
	class := css.New(css.MarkImportant(css.Display.Block))
	if !strings.Contains(css.Harvest(), "."+string(class)+"{display:block !important;}") {
		t.Fatalf("missing !important: %q", css.Harvest())
	}
}

func TestKeyframesEmitsAtRuleAndAnimationName(t *testing.T) {
	reset(t)
	class := css.New(
		css.Keyframes("spin",
			css.At("from", css.Raw("transform", "rotate(0deg)")),
			css.At("to", css.Raw("transform", "rotate(360deg)")),
		),
		css.Animation(css.Ms(1000), css.Linear),
	)
	got := css.Harvest()
	if !strings.Contains(got, "@keyframes spin-") {
		t.Fatalf("missing @keyframes block: %q", got)
	}
	if !strings.Contains(got, "animation-name:spin-") {
		t.Fatalf("missing animation-name: %q", got)
	}
	if !strings.Contains(got, "animation-duration:1000ms") {
		t.Fatalf("missing animation-duration: %q", got)
	}
	_ = class
}

func TestValuesFormatting(t *testing.T) {
	cases := []struct {
		got  string
		want string
	}{
		{string(css.Px(8)), "8px"},
		{string(css.Rem(0.5)), "0.5rem"},
		{string(css.Rem(1)), "1rem"},
		{string(css.Percent(50)), "50%"},
		{string(css.Hex("0af")), "#0af"},
		{string(css.Hex("#0af")), "#0af"},
		{string(css.RGB(1, 2, 3)), "rgb(1,2,3)"},
		{string(css.RGBA(1, 2, 3, 0.5)), "rgba(1,2,3,0.5)"},
		{string(css.Var("accent")), "var(--accent)"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("value mismatch: got %q want %q", c.got, c.want)
		}
	}
}

func TestEmptyRuleSetReturnsEmptyClass(t *testing.T) {
	reset(t)
	if got := css.New(); got != "" {
		t.Fatalf("empty rule set should yield empty class, got %q", got)
	}
}

func TestSeedSuppressesEmission(t *testing.T) {
	reset(t)
	// Pre-compute the class for a rule set without emitting, then seed it.
	class := css.New(css.Display.Grid)
	reset(t)
	css.Seed(string(class))
	again := css.New(css.Display.Grid)
	if again != class {
		t.Fatalf("seeded class changed: %q != %q", again, class)
	}
	if css.Harvest() != "" {
		t.Fatalf("seeded class should not re-emit, got %q", css.Harvest())
	}
}

func TestCustomSinkReceivesEmissions(t *testing.T) {
	reset(t)
	captured := map[string]string{}
	prev := css.SetSink(sinkFunc(func(class, cssText string) { captured[class] = cssText }))
	defer css.SetSink(prev)

	class := css.New(css.Display.Flex)
	if _, ok := captured[string(class)]; !ok {
		t.Fatalf("custom sink did not receive emission; captured=%v", captured)
	}
}

type sinkFunc func(class, cssText string)

func (f sinkFunc) Emit(class, cssText string) { f(class, cssText) }

// TestTypedVarConstructors proves the Var* siblings let a CSS custom property flow into typed
// non-color positions (Length/Duration/Angle/Number) with the same sanitizing as Var, so callers
// don't fall through to Raw(...). A name without the -- prefix is normalized; injection chars are stripped.
func TestTypedVarConstructors(t *testing.T) {
	if got := css.VarLength("--size"); got != css.Length("var(--size)") {
		t.Fatalf("VarLength = %q, want var(--size)", got)
	}
	if got := css.VarDuration("speed"); got != css.Duration("var(--speed)") {
		t.Fatalf("VarDuration (no prefix) = %q, want var(--speed)", got)
	}
	if got := css.VarAngle("--rot"); got != css.Angle("var(--rot)") {
		t.Fatalf("VarAngle = %q, want var(--rot)", got)
	}
	if got := css.VarNumber("--n"); got != css.Number("var(--n)") {
		t.Fatalf("VarNumber = %q, want var(--n)", got)
	}
	// Injection attempt is sanitized away, exactly like Var(Color).
	if got := css.VarLength("x);}evil{"); got != css.Length("var(--xevil)") {
		t.Fatalf("VarLength sanitize = %q, want var(--xevil)", got)
	}
	// Typed Var flows into a typed prop without Raw fallthrough (compile-checks).
	_ = css.W(css.VarLength("--w"))
	_ = css.FontSize(css.VarLength("--fs"))
}
