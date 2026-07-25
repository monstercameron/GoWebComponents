package delta

import (
	"fmt"
	"runtime"
	"testing"
)

// v5 M11 — worker resident memory: the engine's index at 20,000 rows, against a
// target of "index < 2x payload".
//
// The measurement matters because the whole point of a delta engine is to avoid
// re-sending data, and the obvious way to do that is to keep a copy of what was
// sent — which would put a second copy of the projection in the worker. This
// engine keeps one integer per key instead, so the test asserts that architecture
// held rather than trusting that it did.
//
// M11 and M12 sit on opposite sides of the boundary and answer different
// questions; this is the worker side only.

// m11RowCount is the scale M11 is specified at.
const m11RowCount = 20000

// buildSizedRows builds rows whose payloads are a given size, so the ratio can
// be measured against a stated payload rather than an accidental one.
func buildSizedRows(parseCount int, parsePayloadBytes int) []Row {
	parseRows := make([]Row, 0, parseCount)
	for parseIndex := range parseCount {
		parsePayload := make([]byte, parsePayloadBytes)
		for parseByteIndex := range parsePayload {
			parsePayload[parseByteIndex] = byte('a' + parseByteIndex%26)
		}
		parseRows = append(parseRows, Row{
			Key:     Key(fmt.Sprintf("row-%08d", parseIndex)),
			Version: 1,
			Payload: parsePayload,
		})
	}
	return parseRows
}

// measureIndexBytes reports how much heap the engine's own state occupies.
//
// The rows are built and retained OUTSIDE the measurement window, so what is
// measured is the engine's incremental cost — its key order and version map —
// and not the projection data itself, which the worker holds regardless of
// whether a delta engine exists.
func measureIndexBytes(parseT *testing.T, parseRows []Row) uint64 {
	parseT.Helper()

	var parseBefore, parseAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&parseBefore)

	parseEngine := New()
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseT.Fatalf("Publish: %v", parseErr)
	}

	runtime.GC()
	runtime.ReadMemStats(&parseAfter)

	// The engine must be alive across the second reading, or the GC is free to
	// collect the very thing being measured and the answer is zero.
	runtime.KeepAlive(parseEngine)
	runtime.KeepAlive(parseRows)

	if parseAfter.HeapAlloc < parseBefore.HeapAlloc {
		parseT.Fatalf("heap shrank during the measurement (%d -> %d); the reading is not usable",
			parseBefore.HeapAlloc, parseAfter.HeapAlloc)
	}
	return parseAfter.HeapAlloc - parseBefore.HeapAlloc
}

// TestM11IndexStaysUnderTwiceThePayload is the M11 gate.
//
// 200 bytes per row is a realistic serialized table row — a handful of columns
// with short strings. The ratio depends on payload size, which is why the size
// is stated rather than incidental, and why the companion test below reports
// where the ratio crosses over.
func TestM11IndexStaysUnderTwiceThePayload(parseT *testing.T) {
	const parsePayloadBytes = 200

	parseRows := buildSizedRows(m11RowCount, parsePayloadBytes)
	parsePayloadTotal := uint64(m11RowCount * parsePayloadBytes)
	parseIndexBytes := measureIndexBytes(parseT, parseRows)

	parseRatio := float64(parseIndexBytes) / float64(parsePayloadTotal)
	parseT.Logf("M11: %d rows x %d B payload = %.2f MB; index = %.2f MB; ratio = %.2fx",
		m11RowCount, parsePayloadBytes,
		float64(parsePayloadTotal)/(1<<20), float64(parseIndexBytes)/(1<<20), parseRatio)

	if parseRatio >= 2.0 {
		parseT.Errorf("index is %.2fx the payload, want under 2x (M11)", parseRatio)
	}
}

// TestM11EngineDoesNotRetainPayloads is the architectural claim underneath the
// ratio, and the one that would silently break first.
//
// Adding a payload copy to the engine — the natural way to implement a diff —
// would keep every op-count test passing while roughly doubling worker memory.
// So this measures the index at two payload sizes: if payloads were retained,
// the index would grow with them.
func TestM11EngineDoesNotRetainPayloads(parseT *testing.T) {
	const parseSmallPayload = 64
	const parseLargePayload = 1024
	const parseRowCount = 10000

	parseSmallIndex := measureIndexBytes(parseT, buildSizedRows(parseRowCount, parseSmallPayload))
	parseLargeIndex := measureIndexBytes(parseT, buildSizedRows(parseRowCount, parseLargePayload))

	parseT.Logf("index at %dB payload = %d KB; at %dB payload = %d KB",
		parseSmallPayload, parseSmallIndex/1024, parseLargePayload, parseLargeIndex/1024)

	// A 16x payload increase must not move the index appreciably. The tolerance
	// is generous because heap measurement is noisy; retaining payloads would
	// show up as a ~16x difference, not a 50% one.
	parseGrowth := float64(parseLargeIndex) / float64(parseSmallIndex)
	if parseGrowth > 2.0 {
		parseT.Errorf("index grew %.2fx when payloads grew 16x — the engine is retaining payload data", parseGrowth)
	}
}

// TestM11ReportsTheCrossoverPayloadSize records where the ratio stops holding,
// so the M11 number is a property of the design rather than of one payload size
// that happened to be chosen.
//
// It reports rather than gates: a projection of very small rows genuinely has a
// larger index than payload, and that is a fact about the workload, not a
// regression in the engine.
func TestM11ReportsTheCrossoverPayloadSize(parseT *testing.T) {
	const parseRowCount = 10000

	for _, parsePayloadBytes := range []int{16, 32, 64, 128, 256, 512} {
		parseIndexBytes := measureIndexBytes(parseT, buildSizedRows(parseRowCount, parsePayloadBytes))
		parsePayloadTotal := uint64(parseRowCount * parsePayloadBytes)
		parseT.Logf("payload %4d B/row -> index %.2fx payload", parsePayloadBytes,
			float64(parseIndexBytes)/float64(parsePayloadTotal))
	}
}

// ------------------------------------------------------------- op-cost bench

// BenchmarkPublishOneEditIn20k measures the incremental publish path at M11
// scale. It is the number P3.7's criterion (a) — flat publish time from 1k to
// 20k — will be measured against.
func BenchmarkPublishOneEditIn20k(parseB *testing.B) {
	parseRows := buildSizedRows(m11RowCount, 200)
	parseEngine := New()
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseB.Fatalf("initial publish: %v", parseErr)
	}

	parseB.ResetTimer()
	parseB.ReportAllocs()
	parseIteration := 0
	for parseB.Loop() {
		parseIteration++
		parseRows[parseIteration%m11RowCount].Version++
		if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
			parseB.Fatalf("publish: %v", parseErr)
		}
	}
}

// BenchmarkPublishScaling documents the snapshot path as O(N), which P3.7's
// criterion (b) requires be measured rather than assumed.
//
// Both paths emit O(change) ops — that is P3.15a's criterion and it holds for
// each. But only Apply is O(change) in TIME. Publish must read the whole new
// state to discover what changed, so its cost tracks the projection size no
// matter how small the edit. The numbers across these three sizes are the
// evidence.
func BenchmarkPublishScaling(parseB *testing.B) {
	for _, parseRowCount := range []int{1000, 5000, 20000} {
		parseB.Run(fmt.Sprintf("rows-%d", parseRowCount), func(parseSub *testing.B) {
			parseRows := buildSizedRows(parseRowCount, 200)
			parseEngine := New()
			if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
				parseSub.Fatalf("initial publish: %v", parseErr)
			}

			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			parseIteration := 0
			for parseSub.Loop() {
				parseIteration++
				parseRows[parseIteration%parseRowCount].Version++
				if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
					parseSub.Fatalf("publish: %v", parseErr)
				}
			}
		})
	}
}

// BenchmarkApplyScaling is the same sweep for the change-driven path, which is
// what P3.7's criterion (a) — flat publish time from 1k to 20k — will need.
func BenchmarkApplyScaling(parseB *testing.B) {
	for _, parseRowCount := range []int{1000, 5000, 20000} {
		parseB.Run(fmt.Sprintf("rows-%d", parseRowCount), func(parseSub *testing.B) {
			parseRows := buildSizedRows(parseRowCount, 200)
			parseEngine := New()
			if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
				parseSub.Fatalf("initial publish: %v", parseErr)
			}
			parseChange := []Change{{Kind: ChangeUpsert, Row: Row{Key: parseRows[0].Key, Version: 1}}}

			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			for parseSub.Loop() {
				parseChange[0].Row.Version++
				if _, parseErr := parseEngine.Apply(parseChange); parseErr != nil {
					parseSub.Fatalf("apply: %v", parseErr)
				}
			}
		})
	}
}

// BenchmarkApplyOneEdit measures the change-driven path, which does not scan.
// The gap between this and BenchmarkPublishOneEditIn20k is exactly what a
// producer buys by describing its own changes.
func BenchmarkApplyOneEdit(parseB *testing.B) {
	parseRows := buildSizedRows(m11RowCount, 200)
	parseEngine := New()
	if _, parseErr := parseEngine.Publish(parseRows); parseErr != nil {
		parseB.Fatalf("initial publish: %v", parseErr)
	}
	parseChange := []Change{{Kind: ChangeUpsert, Row: Row{Key: "row-00000500", Version: 1, Payload: []byte("edited")}}}

	parseB.ResetTimer()
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseChange[0].Row.Version++
		if _, parseErr := parseEngine.Apply(parseChange); parseErr != nil {
			parseB.Fatalf("apply: %v", parseErr)
		}
	}
}
