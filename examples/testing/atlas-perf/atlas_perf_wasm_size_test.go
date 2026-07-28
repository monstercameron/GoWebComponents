package atlasperf

// M5-adjacent: how big is Atlas's client bundle, compressed?
//
// M5 targets app.wasm under 1.6 MB GZIPPED, measured on the two-artifact example
// with `-trimpath -ldflags="-s -w"` at maximum compression
// (docs/V5_SIZE_STORY.md). This test applies the same compression to Atlas's
// bundle so the two numbers are comparable, and it REFUSES to compare a
// development build against that budget, because a dev build carries the DWARF
// sections `-s -w` strips and would overstate the number by several megabytes.
//
// REPRODUCE:
//
//	# whatever bundle is currently served (usually a development build)
//	go test ./examples/testing/atlas-perf/ -run TestAtlasPerfWasmSize -v
//
//	# a budget-comparable build, into an ISOLATED path
//	#   never build into examples/static/bin from a measurement run: that path is
//	#   shared, other agents build it, and concurrent writes corrupt it
//	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" \
//	  -o "$TMPDIR/atlas_perf_client.wasm" ./examples/server/atlas-commerce-os/client
//	ATLAS_PERF_WASM="$TMPDIR/atlas_perf_client.wasm" ATLAS_PERF_WASM_STRIPPED=1 \
//	  go test ./examples/testing/atlas-perf/ -run TestAtlasPerfWasmSize -v

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybalholm/brotli"
)

// atlasPerfDefaultWasmPath is the bundle the Atlas server serves from
// examples/static/. Relative to this package.
const atlasPerfDefaultWasmPath = "../../static/bin/atlas-commerce-os.wasm"

// atlasPerfM5GzipBudgetBytes is M5 as written: 1.6 MB gzipped.
const atlasPerfM5GzipBudgetBytes = 1_600_000

// atlasPerfGoWasmFloorGzipBytes is the recorded Go js/wasm floor — an empty
// `func main() { select {} }` gzipped, from docs/V5_SIZE_STORY.md. Quoted so the
// addressable share of any Atlas number is visible next to it.
const atlasPerfGoWasmFloorGzipBytes = 541 * 1024

func TestAtlasPerfWasmSize(parseT *testing.T) {
	parsePath := os.Getenv("ATLAS_PERF_WASM")
	parseIsStripped := os.Getenv("ATLAS_PERF_WASM_STRIPPED") == "1"
	if parsePath == "" {
		// Opt-in when no bundle is named. Brotli at BestCompression over a
		// ~20 MB Atlas bundle takes ~60 s; making that a side effect of
		// `go test ./...` would tax every unrelated run in the repository.
		if os.Getenv("ATLAS_PERF_WASM_SIZE") != "1" {
			parseT.Skip("set ATLAS_PERF_WASM=<path> (or ATLAS_PERF_WASM_SIZE=1 for the served bundle); " +
				"brotli over a 20 MB bundle costs ~60 s and must not run in a plain `go test ./...`")
		}
		parsePath = atlasPerfDefaultWasmPath
	}
	parseRaw, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Skipf("no Atlas wasm bundle at %s (%v); build one first — see this file's header", parsePath, parseErr)
	}

	parseGzip := atlasPerfGzipSize(parseT, parseRaw)
	parseBrotli := atlasPerfBrotliSize(parseT, parseRaw)

	parseT.Logf("bundle: %s", filepath.Clean(parsePath))
	parseT.Logf("  raw    %s (%d B)", atlasPerfFormatBytes(len(parseRaw)), len(parseRaw))
	parseT.Logf("  gzip   %s (%d B)", atlasPerfFormatBytes(parseGzip), parseGzip)
	parseT.Logf("  brotli %s (%d B)  %.1f%% under gzip", atlasPerfFormatBytes(parseBrotli), parseBrotli,
		100*(1-float64(parseBrotli)/float64(parseGzip)))

	if !parseIsStripped {
		// Deliberately not an error, and deliberately not silent. A dev build's
		// number is real information about what the local server serves; it is
		// simply not the number M5 is about.
		parseT.Logf("NOT COMPARABLE TO M5: this build was not declared stripped. M5's 1.6 MB gzip budget was "+
			"measured with -trimpath -ldflags=\"-s -w\"; a development build carries DWARF and overstates the "+
			"number. Set ATLAS_PERF_WASM_STRIPPED=1 on a build made with those flags to compare. "+
			"(For reference the budget is %s and the Go js/wasm floor is %s gzipped.)",
			atlasPerfFormatBytes(atlasPerfM5GzipBudgetBytes), atlasPerfFormatBytes(atlasPerfGoWasmFloorGzipBytes))
		return
	}

	parseOverBudget := parseGzip - atlasPerfM5GzipBudgetBytes
	parseAddressable := atlasPerfM5GzipBudgetBytes - atlasPerfGoWasmFloorGzipBytes
	parseT.Logf("M5 (gzip < %s): %s — over by %s (%.0f%% of the %s addressable budget after the %s Go floor)",
		atlasPerfFormatBytes(atlasPerfM5GzipBudgetBytes),
		map[bool]string{true: "MET", false: "MISSED"}[parseGzip <= atlasPerfM5GzipBudgetBytes],
		atlasPerfFormatBytes(parseOverBudget),
		100*float64(parseOverBudget)/float64(parseAddressable),
		atlasPerfFormatBytes(parseAddressable),
		atlasPerfFormatBytes(atlasPerfGoWasmFloorGzipBytes))
	parseT.Logf("brotli is what a browser actually downloads (every wasm-capable browser negotiates it): %s",
		atlasPerfFormatBytes(parseBrotli))
}

func atlasPerfGzipSize(parseT *testing.T, parsePayload []byte) int {
	parseT.Helper()
	var parseBuffer bytes.Buffer
	parseWriter, parseErr := gzip.NewWriterLevel(&parseBuffer, gzip.BestCompression)
	if parseErr != nil {
		parseT.Fatalf("gzip writer: %v", parseErr)
	}
	if _, parseWriteErr := parseWriter.Write(parsePayload); parseWriteErr != nil {
		parseT.Fatalf("gzip write: %v", parseWriteErr)
	}
	if parseCloseErr := parseWriter.Close(); parseCloseErr != nil {
		parseT.Fatalf("gzip close: %v", parseCloseErr)
	}
	return parseBuffer.Len()
}

func atlasPerfBrotliSize(parseT *testing.T, parsePayload []byte) int {
	parseT.Helper()
	var parseBuffer bytes.Buffer
	parseWriter := brotli.NewWriterLevel(&parseBuffer, brotli.BestCompression)
	if _, parseErr := parseWriter.Write(parsePayload); parseErr != nil {
		parseT.Fatalf("brotli write: %v", parseErr)
	}
	if parseErr := parseWriter.Close(); parseErr != nil {
		parseT.Fatalf("brotli close: %v", parseErr)
	}
	return parseBuffer.Len()
}
