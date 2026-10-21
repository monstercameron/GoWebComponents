package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// makeBenchSuiteFixture builds a synthetic report with one native entry and one size row.
func makeBenchSuiteFixture(parseMedian float64, parseMAD float64, parseArch string, parseGzip int64) benchSuiteReport {
	return benchSuiteReport{
		SchemaVersion: benchSuiteSchemaVersion,
		Suite:         "fixture",
		GeneratedAt:   "2026-01-01T00:00:00Z",
		Machine:       &benchmarkMachine{OS: "windows", Arch: parseArch, HostArch: "arm64", CPUModel: "cpu-x"},
		Run:           benchSuiteRun{Count: 5, Warmup: 1, Seed: 7, Order: []string{"native/./a"}, Parallel: 1},
		Lanes: []benchSuiteLane{
			{Lane: "native", Layer: "native-micro", Status: "ok", Entries: []benchSuiteEntry{
				{ID: "native/./a/BenchmarkX", Unit: "ns/op", Samples: []float64{parseMedian}, Median: parseMedian, MAD: parseMAD, N: 5},
			}},
			{Lane: "browser", Layer: "browser", Status: "not-run", Reason: "later", Entries: []benchSuiteEntry{}},
		},
		Size:   []benchSuiteSize{{Artifact: "./app", RawBytes: 1000, GzipBytes: parseGzip}},
		Refs:   map[string]any{},
		Health: map[string]any{},
	}
}

func TestBenchSuiteSchemaRoundTrip(parseT *testing.T) {
	parseReport := makeBenchSuiteFixture(100, 1, "amd64", 400)
	parseReport.Build = &benchSuiteBuild{Pattern: "./app", Target: "wasm", ColdMs: 900, WarmMs: 40}
	parsePath := filepath.Join(parseT.TempDir(), "nested", "suite.json")
	if parseErr := writeBenchSuiteReport(parsePath, parseReport); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseLoaded, parseErr2 := loadBenchSuiteReport(parsePath)
	if parseErr2 != nil {
		parseT.Fatal(parseErr2)
	}
	if !reflect.DeepEqual(parseReport, parseLoaded) {
		parseT.Fatalf("round trip mismatch:\n%+v\n%+v", parseReport, parseLoaded)
	}
	parseRaw, _ := os.ReadFile(parsePath)
	for _, parseKey := range []string{`"schemaVersion": "gwc.bench.suite/v1"`, `"refs": {}`, `"health": {}`, `"coldMs"`} {
		if !strings.Contains(string(parseRaw), parseKey) {
			parseT.Fatalf("report missing %s", parseKey)
		}
	}
}

func TestBenchSuiteRejectsWrongSchema(parseT *testing.T) {
	parsePath := filepath.Join(parseT.TempDir(), "old.json")
	_ = os.WriteFile(parsePath, []byte(`{"schemaVersion":"other/v9"}`), 0o644)
	if _, parseErr := loadBenchSuiteReport(parsePath); parseErr == nil || !strings.Contains(parseErr.Error(), "schemaVersion") {
		parseT.Fatalf("want schema error, got %v", parseErr)
	}
}

func TestBenchSuiteShuffleDeterministicBySeed(parseT *testing.T) {
	parseIDs := []string{"native/./a", "native/./b", "native/./c", "native/./d", "wasm/./a", "wasm/./b", "wasm/./c", "wasm/./d"}
	parseOne := shuffleBenchSuiteOrder(parseIDs, 42)
	parseTwo := shuffleBenchSuiteOrder(parseIDs, 42)
	if !reflect.DeepEqual(parseOne, parseTwo) {
		parseT.Fatal("same seed produced different orders")
	}
	hasDifferent := false
	for parseSeed := int64(1); parseSeed < 20; parseSeed++ {
		if !reflect.DeepEqual(shuffleBenchSuiteOrder(parseIDs, parseSeed), parseOne) {
			hasDifferent = true
		}
	}
	if !hasDifferent {
		parseT.Fatal("different seeds never changed the order")
	}
	if parseIDs[0] != "native/./a" {
		parseT.Fatal("input slice was mutated")
	}
}

func TestBenchSuiteCompareVerdicts(parseT *testing.T) {
	parseBase := makeBenchSuiteFixture(100, 0.5, "amd64", 400)

	parseNoise := compareBenchSuiteReports(parseBase, makeBenchSuiteFixture(101, 0.5, "amd64", 400))
	if parseNoise.Noise != 3 || parseNoise.Regressed != 0 || parseNoise.Improved != 0 {
		parseT.Fatalf("1%% change should be noise: %+v", parseNoise)
	}

	parseRegress := compareBenchSuiteReports(parseBase, makeBenchSuiteFixture(110, 0.5, "amd64", 400))
	if parseRegress.Regressed != 1 || parseRegress.Deltas[len(parseRegress.Deltas)-1].Verdict == "" {
		parseT.Fatalf("10%% slower should regress: %+v", parseRegress)
	}

	parseImprove := compareBenchSuiteReports(parseBase, makeBenchSuiteFixture(80, 0.5, "amd64", 300))
	if parseImprove.Improved != 2 {
		parseT.Fatalf("faster and smaller should both improve: %+v", parseImprove)
	}

	// A noisy baseline widens the threshold: 10% is inside 3*MAD (combined MAD 5 -> 15%).
	parseNoisy := compareBenchSuiteReports(makeBenchSuiteFixture(100, 5, "amd64", 400), makeBenchSuiteFixture(110, 0, "amd64", 400))
	if parseNoisy.Regressed != 0 {
		parseT.Fatalf("noisy entry must not be flagged: %+v", parseNoisy)
	}
	for _, parseLayer := range parseRegress.Layers {
		if parseLayer.Layer == "native-micro" && (parseLayer.GeomeanPct < 9.9 || parseLayer.GeomeanPct > 10.1) {
			parseT.Fatalf("geomean wrong: %+v", parseLayer)
		}
	}
}

func TestBenchSuiteCompareGateAndUnmatched(parseT *testing.T) {
	parseBase := makeBenchSuiteFixture(100, 0, "amd64", 400)
	parseCand := makeBenchSuiteFixture(120, 0, "amd64", 400)
	parseCand.Lanes[0].Entries = append(parseCand.Lanes[0].Entries, benchSuiteEntry{ID: "native/./a/BenchmarkNew", Median: 5})
	parseComparison := compareBenchSuiteReports(parseBase, parseCand)
	if len(parseComparison.OnlyCandidate) != 1 {
		parseT.Fatalf("expected one candidate-only id: %+v", parseComparison.OnlyCandidate)
	}
	if parseErr := getBenchSuiteRegressionGate(parseComparison, 0, 0); parseErr != nil {
		parseT.Fatalf("gate disabled by default: %v", parseErr)
	}
	if parseErr := getBenchSuiteRegressionGate(parseComparison, 25, 0); parseErr != nil {
		parseT.Fatalf("20%% is under a 25%% gate: %v", parseErr)
	}
	if parseErr := getBenchSuiteRegressionGate(parseComparison, 5, 0); parseErr == nil {
		parseT.Fatal("20% should trip a 5% gate")
	}
	if parseErr := getBenchSuiteRegressionGate(parseComparison, 5, 101); parseErr != nil {
		parseT.Fatalf("sub-floor timing should remain advisory: %v", parseErr)
	}
	if parseErr := getBenchSuiteRegressionGate(parseComparison, 5, 100); parseErr == nil {
		parseT.Fatal("timing at the baseline floor should still trip the gate")
	}
	parseComparison.Deltas = append(parseComparison.Deltas, benchSuiteDelta{
		ID: "size/wasm", Unit: "bytes", Baseline: 10, Candidate: 20,
		DeltaPct: 100, Verdict: benchSuiteVerdictRegressed,
	})
	if parseErr := getBenchSuiteRegressionGate(parseComparison, 5, 101); parseErr == nil {
		parseT.Fatal("non-timing metric must remain gated below the timing floor")
	}
}

func TestBenchSuiteCompareCommandMachineMismatch(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseA := filepath.Join(parseDir, "a.json")
	parseB := filepath.Join(parseDir, "b.json")
	_ = writeBenchSuiteReport(parseA, makeBenchSuiteFixture(100, 1, "amd64", 400))
	_ = writeBenchSuiteReport(parseB, makeBenchSuiteFixture(100, 1, "arm64", 400))

	parseErr := runBenchSuiteCompare([]string{parseA, parseB})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "machine.arch") {
		parseT.Fatalf("want machine mismatch error, got %v", parseErr)
	}
	if parseErr2 := runBenchSuiteCompare([]string{parseA, parseB, "-allow-machine-mismatch"}); parseErr2 != nil {
		parseT.Fatalf("override should compare: %v", parseErr2)
	}
	if parseErr3 := runBenchSuiteCompare([]string{"-max-regress-pct", "5", parseA, parseA}); parseErr3 != nil {
		parseT.Fatalf("identical reports pass the gate: %v", parseErr3)
	}
}

func TestBenchSuiteNotRunLanesAlwaysPresent(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	_ = os.MkdirAll(filepath.Join(parseRoot, "pkg"), 0o755)
	_ = os.WriteFile(filepath.Join(parseRoot, "pkg", "x_test.go"), []byte("package pkg\nfunc BenchmarkX() {}\n"), 0o644)
	parseOriginal := benchSuiteRunPackage
	parseT.Cleanup(func() { benchSuiteRunPackage = parseOriginal })
	benchSuiteRunPackage = func(parseL launcher, parseConfig benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
		return benchmarkPackageReport{Lane: parseLane, Package: parsePackagePath, OK: true, Benchmarks: parseBenchmarkOutput(
			"BenchmarkX-8 10 100 ns/op 8 B/op 1 allocs/op\nBenchmarkX-8 10 110 ns/op 8 B/op 1 allocs/op\nBenchmarkX-8 10 90 ns/op 8 B/op 1 allocs/op\n")}
	}
	parseConfig, parseErr := resolveBenchSuiteConfig(benchSuiteConfig{rootPath: parseRoot, outPath: filepath.Join(parseRoot, "bin", "bench-suite.json"), count: 3, warmup: 1, lanes: []string{"native"}, seed: 9})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseReport, parseErr2 := launcher{}.executeBenchSuite(parseConfig)
	if parseErr2 != nil {
		parseT.Fatal(parseErr2)
	}
	parseStatus := map[string]string{}
	for _, parseLane := range parseReport.Lanes {
		parseStatus[parseLane.Lane] = parseLane.Status
		if parseLane.Status == benchSuiteStatusNotRun && parseLane.Reason == "" {
			parseT.Fatalf("lane %s not-run without reason", parseLane.Lane)
		}
	}
	parseWant := map[string]string{"native": "ok", "wasm": "not-run", "size": "not-run", "build": "not-run", "browser": "not-run"}
	if !reflect.DeepEqual(parseStatus, parseWant) {
		parseT.Fatalf("lane statuses %v, want %v", parseStatus, parseWant)
	}
	parseEntry := parseReport.Lanes[0].Entries[0]
	if parseEntry.ID != "native/./pkg/BenchmarkX" || parseEntry.Median != 100 || parseEntry.MAD != 10 || parseEntry.N != 3 || parseEntry.AllocsPerOp == nil {
		parseT.Fatalf("unexpected entry %+v", parseEntry)
	}
	if !reflect.DeepEqual(parseReport.Run.Order, []string{"native/./pkg"}) {
		parseT.Fatalf("order %v", parseReport.Run.Order)
	}
	if _, parseErr3 := json.Marshal(parseReport); parseErr3 != nil {
		parseT.Fatal(parseErr3)
	}
}

func TestBenchSuiteConfigRejectsBadInput(parseT *testing.T) {
	for _, parseConfig := range []benchSuiteConfig{
		{count: 0},
		{count: 1, warmup: -1},
		{count: 1, lanes: []string{"browser"}},
		{count: 1, lanes: []string{"bogus"}},
	} {
		if _, parseErr := resolveBenchSuiteConfig(parseConfig); parseErr == nil {
			parseT.Fatalf("config %+v should be rejected", parseConfig)
		}
	}
}

func TestBenchSuiteSmokeNativeLane(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("smoke test runs go test -bench")
	}
	parseRoot := parseT.TempDir()
	_ = os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module smoke\n\ngo 1.21\n"), 0o644)
	_ = os.WriteFile(filepath.Join(parseRoot, "s.go"), []byte("package smoke\n"), 0o644)
	_ = os.WriteFile(filepath.Join(parseRoot, "s_test.go"), []byte("package smoke\n\nimport \"testing\"\n\nfunc BenchmarkSmoke(b *testing.B) {\n\tfor i := 0; i < b.N; i++ {\n\t}\n}\n"), 0o644)
	parseOut := filepath.Join(parseRoot, "out", "suite.json")
	parseErr := launcher{}.runBenchmarkSuite([]string{"-root", parseRoot, "-lane", "native", "-count", "2", "-benchtime", "1x", "-warmup", "0", "-out", parseOut})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseReport, parseErr2 := loadBenchSuiteReport(parseOut)
	if parseErr2 != nil {
		parseT.Fatal(parseErr2)
	}
	if len(parseReport.Lanes[0].Entries) != 1 || parseReport.Lanes[0].Entries[0].N != 2 {
		parseT.Fatalf("unexpected native entries: %+v", parseReport.Lanes[0])
	}
}
