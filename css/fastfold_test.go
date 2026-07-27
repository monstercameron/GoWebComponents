package css

import "testing"

// The fast fold path must be indistinguishable from the slow one. These tests are
// written against observable behaviour (the class New returns, and the CSS that
// gets registered) rather than against foldCache internals, so they keep meaning
// if the caching strategy changes again.

// A fold key is only allowed to be reused when the rules genuinely produce the
// same CSS. The dangerous direction is a FALSE HIT: two different rule-sets
// digesting the same and the second caller receiving the first one's class.
func TestFastFoldDoesNotCollideAcrossDifferentRuleSets(parseT *testing.T) {
	Reset()

	parseCases := []struct {
		name  string
		rules []Rule
	}{
		{"color red", []Rule{TextColor(Hex("#ff0000"))}},
		{"color blue", []Rule{TextColor(Hex("#0000ff"))}},
		{"bg red", []Rule{Bg(Hex("#ff0000"))}},
		{"two decls", []Rule{TextColor(Hex("#ff0000")), Bg(Hex("#0000ff"))}},
		{"same decls, other order", []Rule{Bg(Hex("#0000ff")), TextColor(Hex("#ff0000"))}},
		{"padding 4", []Rule{Padding(Px(4))}},
		{"padding 40", []Rule{Padding(Px(40))}},
		{"empty", []Rule{}},
	}

	parseSeen := map[string]string{}
	for _, parseCase := range parseCases {
		parseClass := string(New(parseCase.rules...))
		if parseClass == "" {
			continue
		}
		// Two cases may legitimately share a class only if their CSS is identical —
		// which is true for the two orderings of non-conflicting declarations.
		if parseOther, parseDup := parseSeen[parseClass]; parseDup {
			parseBothOrderings := parseCase.name == "same decls, other order" && parseOther == "two decls"
			if !parseBothOrderings {
				parseT.Fatalf("distinct rule-sets %q and %q folded to the same class %q",
					parseOther, parseCase.name, parseClass)
			}
			continue
		}
		parseSeen[parseClass] = parseCase.name
	}
}

// This is the case an order-INDEPENDENT digest gets wrong, and the reason
// fastfold.go digests in order.
//
// canonicalize resolves a conflict by argument order (later wins), so these two
// folds must produce different classes. A sum/xor digest would key them
// identically and hand the second caller the first one's colour.
func TestFastFoldRespectsConflictOrdering(parseT *testing.T) {
	Reset()

	parseRedThenBlue := string(New(TextColor(Hex("#ff0000")), TextColor(Hex("#0000ff"))))
	parseBlueThenRed := string(New(TextColor(Hex("#0000ff")), TextColor(Hex("#ff0000"))))

	if parseRedThenBlue == "" || parseBlueThenRed == "" {
		parseT.Fatal("expected both conflicting folds to produce classes")
	}
	if parseRedThenBlue == parseBlueThenRed {
		parseT.Fatalf("conflicting declarations must fold by argument order, but both orders gave %q — "+
			"the fold digest has become order-independent", parseRedThenBlue)
	}

	parseSheet := StyleBlock()
	if !contains(parseSheet, "#0000ff") || !contains(parseSheet, "#ff0000") {
		parseT.Fatalf("both winning colours should be registered; got %q", parseSheet)
	}
}

// A repeat fold must return the identical class, and must not register a second
// copy of the CSS. This is the property the cache exists for.
func TestFastFoldRepeatIsStableAndEmitsOnce(parseT *testing.T) {
	Reset()

	parseRules := func() []Rule {
		// Rebuilt every call on purpose: a render loop constructs fresh Rule values
		// each pass, so the cache must key on CONTENT, not on identity.
		return []Rule{TextColor(Hex("#123456")), Padding(Px(8)), Gap(Px(4))}
	}

	parseFirst := string(New(parseRules()...))
	if parseFirst == "" {
		parseT.Fatal("expected a class")
	}
	parseBlockAfterFirst := StyleBlock()

	for i := 0; i < 25; i++ {
		if parseRepeat := string(New(parseRules()...)); parseRepeat != parseFirst {
			parseT.Fatalf("repeat fold %d returned %q, want %q", i, parseRepeat, parseFirst)
		}
	}
	if parseBlockAfter := StyleBlock(); parseBlockAfter != parseBlockAfterFirst {
		parseT.Fatalf("repeat folds re-emitted CSS.\n before: %q\n  after: %q",
			parseBlockAfterFirst, parseBlockAfter)
	}
}

// Reset must clear the fold cache with everything else. A surviving entry hands
// back a class whose CSS is no longer registered — markup that still carries the
// class name but has lost its styling.
func TestFastFoldIsClearedByReset(parseT *testing.T) {
	Reset()
	parseClass := string(New(TextColor(Hex("#abcdef"))))
	if parseClass == "" {
		parseT.Fatal("expected a class")
	}
	if !contains(StyleBlock(), "#abcdef") {
		parseT.Fatal("expected the colour to be registered before Reset")
	}

	Reset()
	if parseBlock := StyleBlock(); contains(parseBlock, "#abcdef") {
		parseT.Fatalf("Reset should have dropped the registered CSS; got %q", parseBlock)
	}
	// Folding the same rules again must re-register the CSS rather than returning a
	// cached class with nothing behind it.
	if parseAgain := string(New(TextColor(Hex("#abcdef")))); parseAgain != parseClass {
		parseT.Fatalf("post-Reset fold returned %q, want the same content-hashed class %q", parseAgain, parseClass)
	}
	if !contains(StyleBlock(), "#abcdef") {
		parseT.Fatal("post-Reset fold returned a class but did not re-emit its CSS")
	}
}

func contains(parseHaystack, parseNeedle string) bool {
	return len(parseNeedle) == 0 || indexOf(parseHaystack, parseNeedle) >= 0
}

func indexOf(parseHaystack, parseNeedle string) int {
	for i := 0; i+len(parseNeedle) <= len(parseHaystack); i++ {
		if parseHaystack[i:i+len(parseNeedle)] == parseNeedle {
			return i
		}
	}
	return -1
}
