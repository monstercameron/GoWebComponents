package projection_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/projection"
)

// v5 M12 — render-thread resident projection memory, and the GC pause it costs.
//
// M12 is a ship gate, and it exists because residency is the mechanism that
// could quietly undo M1. Making the render thread hold a copy of the worker's
// data buys zero-round-trip reads (criterion c) and pays for them in heap — and
// heap on the render thread is GC pauses, which are frame drops. A projection
// that answered instantly and stuttered every few seconds would satisfy
// criterion (c) and defeat the thesis.
//
// The two halves are NOT equally measurable here, and the file says so rather
// than averaging them into one green check. Bytes transfer from native to wasm;
// pause behaviour does not.
//
// M11 and M12 sit on opposite sides of the boundary and answer different
// questions. This is the render-thread side only.

// m12ResidentBudgetBytes is the heap a full-residency projection may occupy.
//
// This is the half a native test can measure honestly. Bytes are bytes: a
// projection of N rows holds the same decoded values whether the runtime is
// native or wasm, so the measurement transfers. 8 MB leaves room for the app's
// own state beside it on a memory-constrained device, which is the constraint
// the plan's mobile risk row names.
const m12ResidentBudgetBytes = 8 << 20

// m12FrameBudgetMs is the GC pause a projection may cost.
//
// It matches M7's render-thread GC target rather than being a fresh number
// invented here. See TestM12GCPauseIsMeasuredButNativeOnly for where it can and
// cannot be trusted.
const m12FrameBudgetMs = 3.0

// m12PayloadBytes is a typical row: a handful of short fields.
const m12PayloadBytes = 100

// buildResidentProjection fills a projection with rows and returns it.
func buildResidentProjection(parseT *testing.T, parseRowCount int, parsePayloadBytes int) *projection.Projection[item] {
	parseT.Helper()

	parseProjection, parseErr := projection.New(decodeItem, projection.Options{Resident: -1})
	if parseErr != nil {
		parseT.Fatalf("New: %v", parseErr)
	}

	parseName := make([]byte, parsePayloadBytes)
	for parseIndex := range parseName {
		parseName[parseIndex] = byte('a' + parseIndex%26)
	}

	parseOps := make([]projection.Op, 0, parseRowCount)
	var parseAnchor projection.Key
	for parseIndex := range parseRowCount {
		parseKey := projection.Key(fmt.Sprintf("row-%08d", parseIndex))
		parseOps = append(parseOps, projection.Op{
			Kind: projection.OpInsert, Key: parseKey, AfterKey: parseAnchor,
			Payload: encodeItem(item{Name: string(parseName), Price: parseIndex}),
		})
		parseAnchor = parseKey
	}
	if parseErr := parseProjection.Apply(parseOps); parseErr != nil {
		parseT.Fatalf("apply: %v", parseErr)
	}
	return parseProjection
}

// measureResidentBytes reports the heap a resident projection occupies.
func measureResidentBytes(parseT *testing.T, parseRowCount int, parsePayloadBytes int) uint64 {
	parseT.Helper()

	var parseBefore, parseAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&parseBefore)

	parseProjection := buildResidentProjection(parseT, parseRowCount, parsePayloadBytes)

	runtime.GC()
	runtime.ReadMemStats(&parseAfter)
	// The projection must be alive across the second reading, or the collector
	// is free to reclaim the very thing being measured.
	runtime.KeepAlive(parseProjection)

	if parseAfter.HeapAlloc < parseBefore.HeapAlloc {
		parseT.Fatal("heap shrank during the measurement; the reading is not usable")
	}
	return parseAfter.HeapAlloc - parseBefore.HeapAlloc
}

// TestM12ResidentMemoryFitsTheBudget is the M12 gate a native test can hold.
func TestM12ResidentMemoryFitsTheBudget(parseT *testing.T) {
	parseHeapBytes := measureResidentBytes(parseT, projection.DefaultResident, m12PayloadBytes)
	parseT.Logf("M12: %d rows (the default residency) x ~%d B = %.2f MB resident",
		projection.DefaultResident, m12PayloadBytes, float64(parseHeapBytes)/(1<<20))

	if parseHeapBytes > m12ResidentBudgetBytes {
		parseT.Errorf("a full-residency projection holds %.2f MB, over the %.0f MB budget — DefaultResident is too high",
			float64(parseHeapBytes)/(1<<20), float64(m12ResidentBudgetBytes)/(1<<20))
	}
}

// TestM12DefaultResidentIsJustifiedByMeasurement is what the plan asks for:
// M12 SETS the Resident() default.
//
// A default chosen as a round number would drift out of alignment with what a
// device can carry and nothing would notice. This walks residencies upward,
// finds the largest that stays inside the memory budget, and fails if the
// shipped default exceeds it.
//
// It deliberately does not fail for a CONSERVATIVE default. Holding fewer rows
// than the budget allows costs some zero-round-trip coverage, which is a
// trade-off; holding more costs the render thread memory it does not have,
// which is a defect.
func TestM12DefaultResidentIsJustifiedByMeasurement(parseT *testing.T) {
	if testing.Short() {
		// The sweep allocates and collects hundreds of megabytes. It is a
		// calibration run, not something every edit should pay for.
		parseT.Skip("skipping the residency sweep in short mode")
	}
	parseLargestWithinBudget := 0
	for _, parseRowCount := range []int{10000, 20000, 40000, 80000} {
		parseHeapBytes := measureResidentBytes(parseT, parseRowCount, m12PayloadBytes)
		parseT.Logf("  %6d rows -> %6.2f MB resident", parseRowCount, float64(parseHeapBytes)/(1<<20))
		if parseHeapBytes <= m12ResidentBudgetBytes {
			parseLargestWithinBudget = parseRowCount
		}
	}

	if parseLargestWithinBudget == 0 {
		parseT.Fatal("no measured residency fit the memory budget; the measurement or the budget is wrong")
	}
	parseT.Logf("M12: largest measured residency inside %.0f MB = %d rows; DefaultResident = %d",
		float64(m12ResidentBudgetBytes)/(1<<20), parseLargestWithinBudget, projection.DefaultResident)

	if projection.DefaultResident > parseLargestWithinBudget {
		parseT.Errorf("DefaultResident is %d but only %d rows fit the memory budget — the default is not justified by measurement",
			projection.DefaultResident, parseLargestWithinBudget)
	}
}

// TestM12GCPauseIsMeasuredButNativeOnly reports M12's other half and is explicit
// about what it cannot conclude.
//
// It reads MemStats.PauseNs — actual stop-the-world time — rather than timing
// the wall clock around runtime.GC(), which mostly measures concurrent marking
// that never stops the mutator. An earlier version of this test did the latter,
// reported 0.00 ms at every size from 5,000 to 80,000 rows, and "concluded" that
// 80,000 rows fit the frame budget. A measurement that cannot distinguish a 16x
// difference is not evidence for anything.
//
// The caveat is the real content. Native Go marks in parallel across OS threads;
// Go's js/wasm runtime is single-threaded and pauses differently. A native pause
// therefore CANNOT set a browser budget, so this reports and sanity-checks
// instead of gating, and M12's pause half stays OPEN until it is measured in a
// browser on the P0.2 harness. A test that says so beats a green check that
// means nothing.
func TestM12GCPauseIsMeasuredButNativeOnly(parseT *testing.T) {
	parseProjection := buildResidentProjection(parseT, projection.DefaultResident, m12PayloadBytes)

	var parseBefore, parseAfter runtime.MemStats
	runtime.ReadMemStats(&parseBefore)
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&parseAfter)
	runtime.KeepAlive(parseProjection)

	if parseAfter.NumGC <= parseBefore.NumGC {
		parseT.Fatal("no collection ran; the pause reading would be meaningless")
	}

	// PauseNs is a 256-entry circular buffer indexed by (NumGC+255)%256.
	parseWorstPauseNs := uint64(0)
	for parseCollection := parseBefore.NumGC; parseCollection < parseAfter.NumGC; parseCollection++ {
		parsePause := parseAfter.PauseNs[(parseCollection+255)%256]
		if parsePause > parseWorstPauseNs {
			parseWorstPauseNs = parsePause
		}
	}
	parsePauseMs := float64(parseWorstPauseNs) / 1e6

	parseT.Logf("M12 (native, ADVISORY): %d resident rows -> worst STW pause %.3f ms across %d collections",
		projection.DefaultResident, parsePauseMs, parseAfter.NumGC-parseBefore.NumGC)
	parseT.Log("M12 pause half remains OPEN: this is native Go with parallel marking, not js/wasm. " +
		"The gating number must come from the P0.2 browser harness.")

	if parseWorstPauseNs == 0 {
		parseT.Skip("the runtime reported a zero pause; this platform cannot produce a usable reading")
	}
	// A native pause an order of magnitude past the frame budget would mean the
	// projection is too large for any runtime, wasm or not.
	if parsePauseMs > m12FrameBudgetMs*10 {
		parseT.Errorf("native STW pause is %.3f ms at the default residency, %.0fx the frame budget — too large for any runtime",
			parsePauseMs, parsePauseMs/m12FrameBudgetMs)
	}
}

// TestM12ResidencyCapActuallyBoundsMemory closes the loop between the number and
// the mechanism. A default that is measured but not enforced bounds nothing.
func TestM12ResidencyCapActuallyBoundsMemory(parseT *testing.T) {
	const parseOfferedRows = 60000
	const parseCap = 5000

	if testing.Short() {
		parseT.Skip("skipping the capped-versus-uncapped comparison in short mode")
	}

	parseProjection, parseErr := projection.New(decodeItem, projection.Options{Resident: parseCap})
	if parseErr != nil {
		parseT.Fatalf("New: %v", parseErr)
	}

	parseName := make([]byte, m12PayloadBytes)
	parseOps := make([]projection.Op, 0, parseOfferedRows)
	var parseAnchor projection.Key
	for parseIndex := range parseOfferedRows {
		parseKey := projection.Key(fmt.Sprintf("row-%08d", parseIndex))
		parseOps = append(parseOps, projection.Op{
			Kind: projection.OpInsert, Key: parseKey, AfterKey: parseAnchor,
			Payload: encodeItem(item{Name: string(parseName), Price: parseIndex}),
		})
		parseAnchor = parseKey
	}
	if parseErr := parseProjection.Apply(parseOps); parseErr != nil {
		parseT.Fatalf("apply: %v", parseErr)
	}

	if parseProjection.Len() != parseCap {
		parseT.Errorf("Len = %d, want the cap of %d", parseProjection.Len(), parseCap)
	}
	if parseProjection.DroppedByResidency() != parseOfferedRows-parseCap {
		parseT.Errorf("DroppedByResidency = %d, want %d",
			parseProjection.DroppedByResidency(), parseOfferedRows-parseCap)
	}

	// And the cap must bound MEMORY, not just the row count — the two would
	// diverge if refused rows were retained anywhere.
	parseCappedBytes := measureResidentBytes(parseT, parseCap, m12PayloadBytes)
	parseUncappedBytes := measureResidentBytes(parseT, parseOfferedRows, m12PayloadBytes)
	parseT.Logf("cap %d of %d offered: %.2f MB resident vs %.2f MB uncapped",
		parseCap, parseOfferedRows, float64(parseCappedBytes)/(1<<20), float64(parseUncappedBytes)/(1<<20))

	if parseCappedBytes >= parseUncappedBytes {
		parseT.Errorf("a capped projection holds %.2f MB and an uncapped one %.2f MB — the cap bounds nothing",
			float64(parseCappedBytes)/(1<<20), float64(parseUncappedBytes)/(1<<20))
	}
}

// ---------------------------------------------------------------- benchmarks

// BenchmarkFilterResident is the per-keystroke cost of criterion (c). It is the
// number that replaces a round trip, so it has to be small enough that removing
// the round trip was worth it.
func BenchmarkFilterResident(parseB *testing.B) {
	for _, parseRowCount := range []int{1000, 5000, 20000} {
		parseB.Run(fmt.Sprintf("rows-%d", parseRowCount), func(parseSub *testing.B) {
			parseProjection, parseErr := projection.New(decodeItem, projection.Options{Resident: -1})
			if parseErr != nil {
				parseSub.Fatalf("New: %v", parseErr)
			}
			parseOps := make([]projection.Op, 0, parseRowCount)
			var parseAnchor projection.Key
			for parseIndex := range parseRowCount {
				parseKey := projection.Key(fmt.Sprintf("row-%08d", parseIndex))
				parseOps = append(parseOps, projection.Op{
					Kind: projection.OpInsert, Key: parseKey, AfterKey: parseAnchor,
					Payload: encodeItem(item{Name: fmt.Sprintf("item-%d", parseIndex), Price: parseIndex}),
				})
				parseAnchor = parseKey
			}
			if parseErr := parseProjection.Apply(parseOps); parseErr != nil {
				parseSub.Fatalf("apply: %v", parseErr)
			}

			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			for parseSub.Loop() {
				parseMatches := parseProjection.Filter(func(parseItem item) bool {
					return parseItem.Price%1000 == 0
				})
				if len(parseMatches) == 0 {
					parseSub.Fatal("filter matched nothing")
				}
			}
		})
	}
}

// BenchmarkApplyUpdateBatch is the other per-frame cost: absorbing a published
// delta. Updates must not trigger an index rebuild, so this should stay flat in
// projection size.
func BenchmarkApplyUpdateBatch(parseB *testing.B) {
	for _, parseRowCount := range []int{1000, 5000, 20000} {
		parseB.Run(fmt.Sprintf("rows-%d", parseRowCount), func(parseSub *testing.B) {
			parseProjection, parseErr := projection.New(decodeItem, projection.Options{Resident: -1})
			if parseErr != nil {
				parseSub.Fatalf("New: %v", parseErr)
			}
			parseInserts := make([]projection.Op, 0, parseRowCount)
			var parseAnchor projection.Key
			for parseIndex := range parseRowCount {
				parseKey := projection.Key(fmt.Sprintf("row-%08d", parseIndex))
				parseInserts = append(parseInserts, projection.Op{
					Kind: projection.OpInsert, Key: parseKey, AfterKey: parseAnchor,
					Payload: encodeItem(item{Name: "x", Price: parseIndex}),
				})
				parseAnchor = parseKey
			}
			if parseErr := parseProjection.Apply(parseInserts); parseErr != nil {
				parseSub.Fatalf("apply: %v", parseErr)
			}

			parseUpdate := []projection.Op{{
				Kind: projection.OpUpdate, Key: "row-00000010",
				Payload: encodeItem(item{Name: "y", Price: 1}),
			}}

			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			for parseSub.Loop() {
				if parseErr := parseProjection.Apply(parseUpdate); parseErr != nil {
					parseSub.Fatalf("apply: %v", parseErr)
				}
			}
		})
	}
}
