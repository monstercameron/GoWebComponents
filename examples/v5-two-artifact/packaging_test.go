package twoartifact_test

import (
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

// v5 P3.10 — two-artifact packaging, acceptance M5 and M10.
//
// M5:  app.wasm gzipped under 1.6 MB (baseline 2.37 MB).
// M10: services.wasm size, and time to first command under 400 ms.
//
// The split only means anything if the engine actually stays out of app.wasm,
// so that is checked on the dependency graph rather than inferred from a size
// that could be small for unrelated reasons.

const appPackage = "github.com/monstercameron/GoWebComponents/v6/examples/v5-two-artifact/app"
const servicesPackage = "github.com/monstercameron/GoWebComponents/v6/examples/v5-two-artifact/services"

// m5BudgetBytes is M5's target for the render-thread binary, gzipped.
const m5BudgetBytes = 1_600_000

// buildWasm compiles one package for js/wasm and reports the artifact size and
// its gzipped size.
func buildWasm(parseT *testing.T, parsePackage string) (int64, int64) {
	parseT.Helper()

	if _, parseErr := exec.LookPath("go"); parseErr != nil {
		parseT.Skip("the go toolchain is not available")
	}

	parseOutput := filepath.Join(parseT.TempDir(), "out.wasm")
	parseCommand := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", parseOutput, parsePackage)
	parseCommand.Env = append(parseCommand.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseCombined, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("building %s:\n%s", parsePackage, parseCombined)
	}

	parseBytes, parseReadErr := os.ReadFile(parseOutput)
	if parseReadErr != nil {
		parseT.Fatalf("reading the artifact: %v", parseReadErr)
	}

	var parseCompressed bytes.Buffer
	parseWriter, _ := gzip.NewWriterLevel(&parseCompressed, gzip.BestCompression)
	if _, parseErr := parseWriter.Write(parseBytes); parseErr != nil {
		parseT.Fatalf("gzip: %v", parseErr)
	}
	if parseErr := parseWriter.Close(); parseErr != nil {
		parseT.Fatalf("gzip close: %v", parseErr)
	}

	return int64(len(parseBytes)), int64(parseCompressed.Len())
}

// compressBrotli reports a payload's brotli size at maximum quality.
//
// P6.1 pairs wasm-opt with brotli. wasm-opt is a Binaryen tool and is not
// installed here, so that half is not measured — see
// TestP61CompressionDeltaIsRecorded. Brotli is a dependency of this module
// already, so its delta over gzip can be measured exactly.
func compressBrotli(parseT *testing.T, parseBytes []byte) int64 {
	parseT.Helper()

	var parseCompressed bytes.Buffer
	parseWriter := brotli.NewWriterLevel(&parseCompressed, brotli.BestCompression)
	if _, parseErr := parseWriter.Write(parseBytes); parseErr != nil {
		parseT.Fatalf("brotli write: %v", parseErr)
	}
	if parseErr := parseWriter.Close(); parseErr != nil {
		parseT.Fatalf("brotli close: %v", parseErr)
	}
	return int64(parseCompressed.Len())
}

// buildAndRead compiles a package and returns its raw bytes.
func buildAndRead(parseT *testing.T, parsePackage string) []byte {
	parseT.Helper()

	if _, parseErr := exec.LookPath("go"); parseErr != nil {
		parseT.Skip("the go toolchain is not available")
	}
	parseOutput := filepath.Join(parseT.TempDir(), "out.wasm")
	parseCommand := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", parseOutput, parsePackage)
	parseCommand.Env = append(parseCommand.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseCombined, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("building %s: %s", parsePackage, parseCombined)
	}
	parseBytes, parseReadErr := os.ReadFile(parseOutput)
	if parseReadErr != nil {
		parseT.Fatalf("reading the artifact: %v", parseReadErr)
	}
	return parseBytes
}

// TestP61CompressionDeltaIsRecorded measures P6.1's compression half.
//
// The criterion is "measured size delta recorded per step". Two steps, and only
// one of them can run here:
//
//   - wasm-opt -Oz: a Binaryen tool, NOT INSTALLED in this environment, so its
//     delta is unmeasured and stays unmeasured rather than estimated. A number
//     invented for a tool that did not run would be worse than a gap.
//   - brotli over gzip: measured exactly, since brotli is already a dependency.
func TestP61CompressionDeltaIsRecorded(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping wasm builds in short mode")
	}

	if _, parseErr := exec.LookPath("wasm-opt"); parseErr == nil {
		parseT.Log("wasm-opt IS available here; its delta should be added to this measurement")
	} else {
		parseT.Log("wasm-opt is not installed, so the -Oz step is UNMEASURED and is not estimated")
	}

	for _, parseCase := range []struct {
		label string
		pkg   string
	}{
		{"app.wasm", appPackage},
		{"services.wasm", servicesPackage},
	} {
		parseBytes := buildAndRead(parseT, parseCase.pkg)

		var parseGzip bytes.Buffer
		parseWriter, _ := gzip.NewWriterLevel(&parseGzip, gzip.BestCompression)
		parseWriter.Write(parseBytes)
		parseWriter.Close()

		parseGzipSize := int64(parseGzip.Len())
		parseBrotliSize := compressBrotli(parseT, parseBytes)
		parseSaving := 100 * float64(parseGzipSize-parseBrotliSize) / float64(parseGzipSize)

		parseT.Logf("P6.1 %s: raw %d B, gzip %d B, brotli %d B (%.1f%% smaller than gzip)",
			parseCase.label, len(parseBytes), parseGzipSize, parseBrotliSize, parseSaving)

		if parseBrotliSize >= parseGzipSize {
			parseT.Errorf("%s: brotli (%d B) is not smaller than gzip (%d B); the step is not worth taking",
				parseCase.label, parseBrotliSize, parseGzipSize)
		}
	}
}

// listDeps returns a package's transitive js/wasm dependencies.
func listDeps(parseT *testing.T, parsePackage string) []string {
	parseT.Helper()

	parseCommand := exec.Command("go", "list", "-deps", parsePackage)
	parseCommand.Env = append(parseCommand.Environ(), "GOOS=js", "GOARCH=wasm")
	parseOutput, parseErr := parseCommand.Output()
	if parseErr != nil {
		parseT.Skipf("go list unavailable: %v", parseErr)
	}
	return strings.Split(strings.TrimSpace(string(parseOutput)), "\n")
}

// TestAppArtifactDoesNotLinkTheEngine is the structural claim the whole split
// rests on.
//
// Checked on the dependency graph, not on the size: a small binary could be
// small for reasons unrelated to the engine, and would stop being small the
// moment someone added a convenience import.
func TestAppArtifactDoesNotLinkTheEngine(parseT *testing.T) {
	for _, parseDep := range listDeps(parseT, appPackage) {
		for _, parseEngine := range []string{
			"github.com/tetratelabs/wazero",
			"github.com/ncruces/go-sqlite3",
			"modernc.org/sqlite",
			"github.com/monstercameron/GoWebComponents/v6/db/sqlite",
		} {
			if parseDep == parseEngine || strings.HasPrefix(parseDep, parseEngine+"/") {
				parseT.Errorf("app.wasm links %q — the two-artifact split is not holding", parseDep)
			}
		}
	}
}

// TestServicesArtifactCarriesTheEngine is the other half: the engine must live
// somewhere, and that somewhere is the worker binary.
func TestServicesArtifactCarriesTheEngine(parseT *testing.T) {
	hasEngine := false
	for _, parseDep := range listDeps(parseT, servicesPackage) {
		if strings.HasPrefix(parseDep, "github.com/ncruces/go-sqlite3") ||
			strings.HasPrefix(parseDep, "github.com/tetratelabs/wazero") {
			hasEngine = true
			break
		}
	}
	if !hasEngine {
		parseT.Error("services.wasm carries no engine — the split has separated the app from nothing")
	}
}

// m5MeasuredCeilingBytes is the gzipped app.wasm size recorded on 2026-07-25.
//
// A RATCHET, not the target. M5's target is 1.6 MB and this example measures
// 1.73 MB, so the budget is missed — see TestM5IsMeasuredAgainstItsBudget for
// the breakdown and why. Gating on the target would leave the suite permanently
// red, which hides real failures; gating on the measured value catches the
// regression that actually matters, which is app.wasm quietly growing.
//
// Lower it when the size drops. Raising it is a deliberate act that should
// arrive with a reason.
const m5MeasuredCeilingBytes = 1_800_000

// goWasmFloorGzipBytes is the gzipped size of an EMPTY Go js/wasm program,
// measured with the same flags on the same toolchain.
//
// It is the part of M5 no framework or application work can remove: the Go
// runtime, its scheduler, and its garbage collector. Quoting a total without it
// makes the addressable portion look larger than it is.
const goWasmFloorGzipBytes = 540_766

// TestM5IsMeasuredAgainstItsBudget records M5 and its gap.
//
// M5 is an ADVISORY gate — the plan's own words are that it "measures against a
// floor (Go wasm size)" and that a miss is a recorded trade-off rather than a
// release veto. So this reports the number, the budget, and the share of each
// that is unavoidable, then gates on a ratchet.
func TestM5IsMeasuredAgainstItsBudget(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping a wasm build in short mode")
	}

	parseRaw, parseGzipped := buildWasm(parseT, appPackage)

	parseAddressable := parseGzipped - goWasmFloorGzipBytes
	parseAddressableBudget := int64(m5BudgetBytes) - goWasmFloorGzipBytes

	parseT.Logf("M5: app.wasm = %d B raw, %d B gzipped; budget %d B", parseRaw, parseGzipped, m5BudgetBytes)
	parseT.Logf("M5 breakdown: %d B is the Go wasm floor (%.0f%% of the budget, unavoidable); "+
		"framework + app is %d B against %d B addressable",
		goWasmFloorGzipBytes, 100*float64(goWasmFloorGzipBytes)/float64(m5BudgetBytes),
		parseAddressable, parseAddressableBudget)

	if parseGzipped > m5BudgetBytes {
		parseT.Logf("M5 MISSED by %d B (%.0f%% of the total, %.0f%% of the addressable budget). "+
			"Recorded as an advisory miss; the two-artifact split alone does not reach 1.6 MB.",
			parseGzipped-m5BudgetBytes,
			100*float64(parseGzipped-m5BudgetBytes)/float64(m5BudgetBytes),
			100*float64(parseGzipped-m5BudgetBytes)/float64(parseAddressableBudget))
	}

	// The ratchet: growth is the failure worth catching automatically.
	if parseGzipped > m5MeasuredCeilingBytes {
		parseT.Errorf("app.wasm grew to %d B gzipped, past the recorded ceiling of %d B",
			parseGzipped, m5MeasuredCeilingBytes)
	}
}

// TestM10ServicesArtifactIsMeasured records the services.wasm half of M10.
//
// It reports rather than gates. M10's stated target is time-to-first-command
// under 400 ms, and that is a browser measurement — instantiation, compilation,
// and the first round trip. Size is the input to it that can be measured here;
// asserting a size budget invented in this file would be a number with no
// authority behind it.
func TestM10ServicesArtifactIsMeasured(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping a wasm build in short mode")
	}

	parseRaw, parseGzipped := buildWasm(parseT, servicesPackage)
	parseT.Logf("M10: services.wasm = %.2f MB raw, %.2f MB gzipped",
		float64(parseRaw)/(1<<20), float64(parseGzipped)/(1<<20))
	parseT.Log("M10's time-to-first-command half is a browser measurement and remains OPEN; " +
		"this records the size that feeds it.")

	if parseRaw == 0 {
		parseT.Error("services.wasm is empty")
	}
}

// TestTheTwoArtifactsAreSeparatelyBuildable is the packaging claim itself.
//
// Two binaries from one module, each with its own dependency closure. If they
// could not be built independently there would be no split, only two entry
// points into one artifact.
func TestTheTwoArtifactsAreSeparatelyBuildable(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping wasm builds in short mode")
	}

	parseAppRaw, parseAppGz := buildWasm(parseT, appPackage)
	parseServicesRaw, parseServicesGz := buildWasm(parseT, servicesPackage)

	parseT.Logf("app      %.2f MB raw / %.2f MB gz", float64(parseAppRaw)/(1<<20), float64(parseAppGz)/(1<<20))
	parseT.Logf("services %.2f MB raw / %.2f MB gz", float64(parseServicesRaw)/(1<<20), float64(parseServicesGz)/(1<<20))

	// The engine has real mass, so the binary carrying it must be visibly
	// larger. If they were within a few percent, both would be carrying it.
	if parseServicesRaw <= parseAppRaw {
		parseT.Errorf("services.wasm (%d B) is not larger than app.wasm (%d B); the engine may be in both",
			parseServicesRaw, parseAppRaw)
	}
}
