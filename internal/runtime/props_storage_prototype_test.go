package runtime

import "testing"

// v5 PA.2 — prototype the three props-storage designs and compare them.
//
// PA.1 established the baseline and eliminated candidate (a): migrating call
// sites to CreateElementOwned is already done in first-party code, so it has
// no headroom left. That leaves:
//
//	(b) copy-on-write props
//	(c) slice-backed small-map for the common <=4-key case
//
// These benchmarks measure the STORAGE strategies in isolation, so PA.3 can
// choose against numbers instead of intuition. Isolating them is deliberate:
// integrating a candidate into the element pipeline is PA.3's work, and doing
// it three times to find out which one wins would cost far more than measuring
// the primitive each one is built on.

// propPair is candidate (c)'s storage element: a key/value pair in a flat
// slice, ordered as the caller supplied it.
type propPair struct {
	Key   string
	Value any
}

// smallPropsMax is the occupancy below which a slice beats a map. Chosen from
// PA.1's finding that a 1-key map costs the same 336 B / 2 allocs as a 4-key
// one, so everything up to 4 keys is paying almost entirely for structure.
const smallPropsMax = 4

var (
	propsMapSink   map[string]any
	propsSliceSink []propPair
)

// clonePropsToSlice is candidate (c): copy into a flat slice, one allocation,
// sized exactly to the contents.
func clonePropsToSlice(parseProps map[string]any) []propPair {
	if len(parseProps) == 0 {
		return nil
	}
	getPairs := make([]propPair, 0, len(parseProps))
	for parseKey, parseValue := range parseProps {
		getPairs = append(getPairs, propPair{Key: parseKey, Value: parseValue})
	}
	return getPairs
}

// lookupSliceProp is the read path candidate (c) trades for. At <=4 entries a
// linear scan beats a hash: no hashing, no indirection, and the whole slice
// sits in one cache line.
func lookupSliceProp(parsePairs []propPair, parseKey string) (any, bool) {
	for parseIndex := range parsePairs {
		if parsePairs[parseIndex].Key == parseKey {
			return parsePairs[parseIndex].Value, true
		}
	}
	return nil, false
}

// BenchmarkPropsStorage_MapVsSlice is the PA.2 comparison table.
func BenchmarkPropsStorage_MapVsSlice(parseB *testing.B) {
	for _, parseSize := range []int{1, 2, 4, 8} {
		parseProps := propsOfSize(parseSize)

		parseB.Run("map/"+sizeLabel(parseSize), func(parseInner *testing.B) {
			parseInner.ReportAllocs()
			for parseInner.Loop() {
				propsMapSink = cloneElementProps(parseProps)
			}
		})

		parseB.Run("slice/"+sizeLabel(parseSize), func(parseInner *testing.B) {
			parseInner.ReportAllocs()
			for parseInner.Loop() {
				propsSliceSink = clonePropsToSlice(parseProps)
			}
		})
	}
}

// BenchmarkPropsStorage_ReadPath measures what candidate (c) gives up. A
// storage win that made every reconciler prop read slower would be a net loss,
// since the diff path reads props far more often than it builds them.
func BenchmarkPropsStorage_ReadPath(parseB *testing.B) {
	for _, parseSize := range []int{2, 4, 8} {
		parseProps := propsOfSize(parseSize)
		parsePairs := clonePropsToSlice(parseProps)
		// Worst case for the slice: the key that sorts last in iteration order.
		parseKey := parsePairs[len(parsePairs)-1].Key

		parseB.Run("map/"+sizeLabel(parseSize), func(parseInner *testing.B) {
			for parseInner.Loop() {
				_, _ = parseProps[parseKey]
			}
		})

		parseB.Run("slice/"+sizeLabel(parseSize), func(parseInner *testing.B) {
			for parseInner.Loop() {
				_, _ = lookupSliceProp(parsePairs, parseKey)
			}
		})
	}
}

// TestPropsStorage_SliceRoundTrips keeps the prototype honest: a storage form
// that loses or reorders entries is not a candidate regardless of its numbers.
func TestPropsStorage_SliceRoundTrips(parseT *testing.T) {
	parseProps := map[string]any{"class": "a", "id": "b", "role": "c"}
	parsePairs := clonePropsToSlice(parseProps)

	if len(parsePairs) != len(parseProps) {
		parseT.Fatalf("slice has %d entries, want %d", len(parsePairs), len(parseProps))
	}
	for parseKey, parseWant := range parseProps {
		parseGot, parseOk := lookupSliceProp(parsePairs, parseKey)
		if !parseOk || parseGot != parseWant {
			parseT.Errorf("key %q = %#v (found=%t), want %#v", parseKey, parseGot, parseOk, parseWant)
		}
	}
	if _, parseOk := lookupSliceProp(parsePairs, "absent"); parseOk {
		parseT.Error("lookup reported a key that was never stored")
	}
	if parsePairs := clonePropsToSlice(nil); parsePairs != nil {
		parseT.Error("empty props must stay nil, matching the map path's free case")
	}
}

// TestPropsStorage_CopyOnWriteIsUnsoundForThisContract records why candidate
// (b) is eliminated by reasoning rather than measurement.
//
// Copy-on-write means: keep the caller's map, clone only when someone writes.
// The runtime never writes to Props after construction, so a runtime-side CoW
// would elide every clone — and that is precisely the problem. The contract
// CreateElement's clone protects is that the CALLER may retain and mutate the
// map it passed. Go gives the runtime no way to observe a caller's write to a
// map it handed over, so CoW cannot trigger on the only mutation that matters.
//
// It would be fast and silently wrong: an app that reuses one props map across
// renders would see earlier elements mutate retroactively. This test documents
// the hazard by demonstrating it on the Owned path, which has the same
// aliasing semantics CoW would have.
func TestPropsStorage_CopyOnWriteIsUnsoundForThisContract(parseT *testing.T) {
	parseShared := map[string]any{"class": "before"}

	// Owned aliases the caller's map, exactly as a CoW scheme would.
	parseElem := CreateElementOwned("div", parseShared)

	// The caller mutates its own map afterwards — legal, and invisible to us.
	parseShared["class"] = "after"

	if parseElem.Props["class"] != "after" {
		parseT.Fatal("expected the aliased element to observe the caller's later write")
	}

	// The cloning constructor is immune, which is the contract worth keeping.
	parseShared2 := map[string]any{"class": "before"}
	parseCloned := CreateElement("div", parseShared2)
	parseShared2["class"] = "after"

	if parseCloned.Props["class"] != "before" {
		parseT.Error("CreateElement must isolate the element from later caller writes; that isolation is what candidate (b) cannot preserve")
	}
}
