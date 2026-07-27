package css

// Benchmarks for the fold fast path (see fastfold.go).
//
// BenchmarkFoldWarmFourBundlesOldOrdering deliberately reproduces the ORDER the
// code used before the fast path existed — canonicalize, then consult the cache —
// so the improvement can be re-measured on any machine instead of trusting a
// number in a commit message. Keep it: if someone reverts the fast path believing
// it is redundant, this benchmark is what shows them the cost.
//
// Measured windows/arm64, -benchtime 3000x, medians of 3:
//
//	OldOrdering  ~2200 ns/op  2248 B/op  29 allocs/op
//	Fast path    ~1150 ns/op  1016 B/op  16 allocs/op
//	Construction  ~270 ns/op    40 B/op   5 allocs/op
//
// The third line is the caller's own cost. It reads far lower than its share of
// the second because the slice does not escape when nothing consumes it; passing
// it to New forces it to the heap. That gap is the argument for hoisting folded
// classes to package vars, which removes the construction as well as the fold.

import "testing"

// Mirrors the shape the profile flagged: a four-bundle fold, rules rebuilt every
// call the way a render loop does it.
func BenchmarkFoldWarmFourBundles(b *testing.B) {
	Reset()
	build := func() []Rule {
		return []Rule{
			Display.Flex, Items.Center, Gap(Px(8)), Padding(Px(12)),
			TextColor(Hex("#14161A")), Bg(Hex("#F2F1ED")),
			Rounded(Px(2)), FontSize(Rem(0.875)),
		}
	}
	_ = New(build()...) // warm
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New(build()...)
	}
}

// Replicates the OLD ordering — canonicalize first, then consult the cache — so
// the before/after comparison is on one machine in one run.
func BenchmarkFoldWarmFourBundlesOldOrdering(b *testing.B) {
	Reset()
	build := func() []Rule {
		return []Rule{
			Display.Flex, Items.Center, Gap(Px(8)), Padding(Px(12)),
			TextColor(Hex("#14161A")), Bg(Hex("#F2F1ED")),
			Rounded(Px(2)), FontSize(Rem(0.875)),
		}
	}
	_ = New(build()...) // warm both caches
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rules := build()
		canonical, _, _ := canonicalize(rules)
		if cached, ok := newCache.Load(canonical); ok {
			_ = cached.(Sheet)
			continue
		}
		b.Fatal("expected warm cache hit")
	}
}

// Isolates the caller's own cost: constructing the rule values, with no fold at
// all. Whatever this costs is what hoisting to a package var removes.
func BenchmarkFoldRuleConstructionOnly(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rules := []Rule{
			Display.Flex, Items.Center, Gap(Px(8)), Padding(Px(12)),
			TextColor(Hex("#14161A")), Bg(Hex("#F2F1ED")),
			Rounded(Px(2)), FontSize(Rem(0.875)),
		}
		_ = rules
	}
}
