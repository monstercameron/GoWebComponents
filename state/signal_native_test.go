//go:build !(js && wasm)

package state_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/state"
)

// TestSignalSeedAndGet proves NewSignal seeds its initial value and Get reads it.
func TestSignalSeedAndGet(parseT *testing.T) {
	parseSig := state.NewSignal(7)
	if parseGot := parseSig.Get(); parseGot != 7 {
		parseT.Fatalf("expected seeded 7, got %d", parseGot)
	}
	if parseGot := parseSig.Peek(); parseGot != 7 {
		parseT.Fatalf("expected Peek 7, got %d", parseGot)
	}
}

// TestSignalSetUpdateRoundTrip proves Set and Update mutate the observed value.
func TestSignalSetUpdateRoundTrip(parseT *testing.T) {
	parseSig := state.NewSignal("idle")
	parseSig.Set("busy")
	if parseGot := parseSig.Get(); parseGot != "busy" {
		parseT.Fatalf("expected 'busy' after Set, got %q", parseGot)
	}
	parseCount := state.NewSignal(10)
	parseCount.Update(func(parsePrev int) int { return parsePrev + 5 })
	if parseGot := parseCount.Get(); parseGot != 15 {
		parseT.Fatalf("expected 15 after Update, got %d", parseGot)
	}
}

// TestSignalAnonymousIdentitiesAreUnique proves two NewSignal calls never collide,
// so anonymous signals are independent without caller-managed ids.
func TestSignalAnonymousIdentitiesAreUnique(parseT *testing.T) {
	parseA := state.NewSignal(1)
	parseB := state.NewSignal(2)
	if parseA.ID() == parseB.ID() {
		parseT.Fatalf("expected distinct signal ids, both were %q", parseA.ID())
	}
	parseA.Set(100)
	if parseB.Get() != 2 {
		parseT.Fatalf("writing signal A leaked into signal B: B=%d", parseB.Get())
	}
}

// TestKeyedSignalSharesIdentity proves NewKeyedSignal handles with the same id
// address one shared reactive value (like UseAtom).
func TestKeyedSignalSharesIdentity(parseT *testing.T) {
	parseWriter := state.NewKeyedSignal("test:sig:shared", 0)
	parseReader := state.NewKeyedSignal("test:sig:shared", -1) // late handle must not clobber
	parseWriter.Set(42)
	if parseGot := parseReader.Get(); parseGot != 42 {
		parseT.Fatalf("expected shared keyed value 42, got %d", parseGot)
	}
}

// TestSignalTextReturnsReactiveNode proves the fine-grained text binding produces a
// real element (the node that updates in place without re-rendering its owner).
func TestSignalTextReturnsReactiveNode(parseT *testing.T) {
	parseSig := state.NewSignal(3)
	parseNode := parseSig.Text(func(parseN int) string { return fmt.Sprintf("%d", parseN) })
	if parseNode == nil {
		parseT.Fatal("expected a reactive text node, got nil")
	}
}

// TestSignalReactiveRegionSourceIDs proves a Signal exposes its id as a region
// source so ui.ReactiveRegion can subscribe to it.
func TestSignalReactiveRegionSourceIDs(parseT *testing.T) {
	parseSig := state.NewKeyedSignal("test:sig:region", 0)
	parseIDs := parseSig.ReactiveRegionSourceIDs()
	if len(parseIDs) != 1 || parseIDs[0] != "test:sig:region" {
		parseT.Fatalf("expected [test:sig:region], got %v", parseIDs)
	}
}

// TestComputedRecomputesFromLiveSources proves NewComputed reflects current source
// values on each read (lazy, explicit-dependency derivation).
func TestComputedRecomputesFromLiveSources(parseT *testing.T) {
	parseFirst := state.NewSignal("Ada")
	parseLast := state.NewSignal("Lovelace")
	parseFull := state.NewComputed(func() string {
		return parseFirst.Get() + " " + parseLast.Get()
	}, parseFirst, parseLast)

	if parseGot := parseFull.Get(); parseGot != "Ada Lovelace" {
		parseT.Fatalf("expected 'Ada Lovelace', got %q", parseGot)
	}
	parseLast.Set("Byron")
	if parseGot := parseFull.Get(); parseGot != "Ada Byron" {
		parseT.Fatalf("expected computed to track source change, got %q", parseGot)
	}
}

// TestComputedDeclaresEveryDependency proves a multi-source computed exposes ALL
// declared source ids to ui.ReactiveRegion (so any dependency change refreshes it),
// de-duplicated and order-preserving.
func TestComputedDeclaresEveryDependency(parseT *testing.T) {
	parseA := state.NewKeyedSignal("test:sig:depA", 0)
	parseB := state.NewKeyedSignal("test:sig:depB", 0)
	parseComputed := state.NewComputed(func() int { return parseA.Get() + parseB.Get() }, parseA, parseB, parseA /* dup */)

	parseIDs := parseComputed.ReactiveRegionSourceIDs()
	if len(parseIDs) != 2 || parseIDs[0] != "test:sig:depA" || parseIDs[1] != "test:sig:depB" {
		parseT.Fatalf("expected deduped [depA depB], got %v", parseIDs)
	}
}

// TestComputedComposesAsSource proves a ComputedSignal can itself be a dependency
// of another computed (fine-grained graphs compose).
func TestComputedComposesAsSource(parseT *testing.T) {
	parsePrice := state.NewSignal(100)
	parseWithTax := state.NewComputed(func() int { return parsePrice.Get() * 110 / 100 }, parsePrice)
	parseLabel := state.NewComputed(func() string { return fmt.Sprintf("$%d", parseWithTax.Get()) }, parseWithTax)

	if parseGot := parseLabel.Get(); parseGot != "$110" {
		parseT.Fatalf("expected composed computed '$110', got %q", parseGot)
	}
	parsePrice.Set(200)
	if parseGot := parseLabel.Get(); parseGot != "$220" {
		parseT.Fatalf("expected composed computed to track upstream change, got %q", parseGot)
	}
}

// TestComputedNilCompute proves a zero-value-safe computed returns the zero value.
func TestComputedNilCompute(parseT *testing.T) {
	var parseComputed state.ComputedSignal[int]
	if parseGot := parseComputed.Get(); parseGot != 0 {
		parseT.Fatalf("expected zero value from nil computed, got %d", parseGot)
	}
	if parseIDs := parseComputed.ReactiveRegionSourceIDs(); parseIDs != nil {
		parseT.Fatalf("expected nil source ids from empty computed, got %v", parseIDs)
	}
}

// TestSignalComposesWithReactiveSystem proves a Signal satisfies the same source
// contract as Atom/Derived: it is accepted by NewComputed (which takes the
// reactive-source contract) and exposes a region source id. (UseSelector itself is
// a render-time hook and is exercised in the wasm/browser lanes, not here.)
func TestSignalComposesWithReactiveSystem(parseT *testing.T) {
	parseSig := state.NewKeyedSignal("test:sig:selsrc", 5)
	// Accepted as a NewComputed dependency — same source contract as Atom/Derived.
	parseDoubled := state.NewComputed(func() string {
		return fmt.Sprintf("%d", parseSig.Get()*2)
	}, parseSig)
	if parseGot := parseDoubled.Get(); parseGot != "10" {
		parseT.Fatalf("expected doubled '10', got %q", parseGot)
	}
	if parseIDs := parseSig.ReactiveRegionSourceIDs(); len(parseIDs) != 1 || parseIDs[0] != "test:sig:selsrc" {
		parseT.Fatalf("expected single region source [test:sig:selsrc], got %v", parseIDs)
	}
}

// TestSignalTextValueZeroArg proves the zero-argument TextValue shortcut binds a reactive
// text node without a custom formatter (the FA1 audit ergonomics niggle).
func TestSignalTextValueZeroArg(parseT *testing.T) {
	if parseNode := state.NewSignal("Ada").TextValue(); parseNode == nil {
		parseT.Fatal("Signal.TextValue should return a reactive text node")
	}
	parseSrc := state.NewSignal(2)
	parseComputed := state.NewComputed(func() int { return parseSrc.Get() * 2 }, parseSrc)
	if parseNode := parseComputed.TextValue(); parseNode == nil {
		parseT.Fatal("ComputedSignal.TextValue should return a reactive text node")
	}
}

// TestSignalTextRendersValueNatively proves the .Text reactive-text-node getter produces the
// current value when mounted natively (previously only exercised under js && wasm).
func TestSignalTextRendersValueNatively(parseT *testing.T) {
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")

	parseSig := state.NewSignal("hello-signal")
	parseNode := parseSig.Text(func(parseV string) string { return parseV + "!" })
	if parseErr := parseRuntime.RenderInto(parseRoot, parseNode); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	if parseText := collectSignalText(parseAdapter, parseRoot); !strings.Contains(parseText, "hello-signal!") {
		parseT.Fatalf("expected the .Text getter to render hello-signal!, got %q", parseText)
	}
}

func collectSignalText(parseAdapter *mockdom.MockDOMAdapter, parseNode runtime.DOMNode) string {
	parseMock, parseOk := parseNode.(*mockdom.MockDOMNode)
	if !parseOk {
		return ""
	}
	parseText := parseMock.TextContent
	for _, parseChild := range parseAdapter.GetChildren(parseNode) {
		parseText += collectSignalText(parseAdapter, parseChild)
	}
	return parseText
}

// TestComputedTextSubscribesAllSources proves the FA1 multi-source fix: a ComputedSignal.Text
// node subscribes to EVERY declared source (not only the first), so a change to any dependency
// flushes the text. We assert it both structurally (the reactive-text node carries all source
// ids) and end-to-end (rendered text reflects the live computed value).
func TestComputedTextSubscribesAllSources(parseT *testing.T) {
	parseFirst := state.NewKeyedSignal("test:fa1:first", "Ada")
	parseLast := state.NewKeyedSignal("test:fa1:last", "Lovelace")
	parseFull := state.NewComputed(func() string {
		return parseFirst.Get() + " " + parseLast.Get()
	}, parseFirst, parseLast)

	parseNode := parseFull.Text(func(parseV string) string { return parseV })
	// Structural: the node's atom-id prop carries BOTH source ids (comma-joined), so the
	// reconciler subscribes the text to each dependency.
	parseIDs, _ := parseNode.Props["__gwc_reactive_text_atom_id"].(string)
	if !strings.Contains(parseIDs, "test:fa1:first") || !strings.Contains(parseIDs, "test:fa1:last") {
		parseT.Fatalf("multi-source .Text must subscribe to both sources, got id prop %q", parseIDs)
	}

	// End-to-end: mount and confirm the initial computed value renders.
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})
	parseRoot := parseAdapter.CreateElement("div")
	if parseErr := parseRuntime.RenderInto(parseRoot, parseFull.Text(func(parseV string) string { return parseV })); parseErr != nil {
		parseT.Fatalf("RenderInto: %v", parseErr)
	}
	if parseText := collectSignalText(parseAdapter, parseRoot); !strings.Contains(parseText, "Ada Lovelace") {
		parseT.Fatalf("expected computed text 'Ada Lovelace', got %q", parseText)
	}
}

// TestNewAutoComputedDiscoversSources proves opt-in auto-tracking: NewAutoComputed runs its
// compute once and auto-discovers every signal it read as a source, with the right value —
// while NewComputed (explicit) remains the default path.
func TestNewAutoComputedDiscoversSources(parseT *testing.T) {
	parseA := state.NewSignal(2)
	parseB := state.NewSignal(3)

	parseSum := state.NewAutoComputed(func() int { return parseA.Get() + parseB.Get() })
	if parseSum.Get() != 5 {
		parseT.Fatalf("auto-computed value = %d, want 5", parseSum.Get())
	}
	// Both signals were discovered as reactive sources (so a region bound to the computed
	// refreshes when either changes).
	parseIDs := parseSum.ReactiveRegionSourceIDs()
	if len(parseIDs) != 2 {
		parseT.Fatalf("auto-computed should have discovered 2 sources, got %d: %v", len(parseIDs), parseIDs)
	}
	parseA.Set(10)
	if parseSum.Get() != 13 {
		parseT.Fatalf("auto-computed should recompute to 13 after a change, got %d", parseSum.Get())
	}
}
