package errorcodes

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// metadataSourceRel is the runtime source of truth, relative to this package.
const metadataSourceRel = "../../internal/runtime/diagnostic_metadata.go"

// referencePageRel is the committed generated page, relative to this package.
const referencePageRel = "../REFERENCE_MANUAL/error-codes.md"

func readMetadataSource(parseT *testing.T) string {
	parseT.Helper()
	parseData, parseErr := os.ReadFile(filepath.FromSlash(metadataSourceRel))
	if parseErr != nil {
		parseT.Fatalf("read metadata source: %v", parseErr)
	}
	return string(parseData)
}

// TestExtractCodesFindsKnownCodes sanity-checks the extractor against codes that
// must always exist, including the panic-phase-only GWC-RUNTIME-PANIC-ASYNC that
// has no descriptive switch entry.
func TestExtractCodesFindsKnownCodes(parseT *testing.T) {
	parseCodes := ExtractCodes(readMetadataSource(parseT))
	if len(parseCodes) < 15 {
		parseT.Fatalf("expected the full diagnostic-code set, got only %d", len(parseCodes))
	}
	parseSeen := map[string]ErrorCode{}
	for _, parseCode := range parseCodes {
		parseSeen[parseCode.Code] = parseCode
	}
	for _, parseWant := range []string{
		"GWC-RUNTIME-PANIC-RENDER", "GWC-RUNTIME-PANIC-ASYNC",
		"GWC-ROUTER-REDIRECT-LOOP", "GWC-HYDRATION-TEXT-MISMATCH",
		"GWC-RUNTIME-CONTAINER-NOT-FOUND",
	} {
		parseEntry, parseOk := parseSeen[parseWant]
		if !parseOk {
			parseT.Fatalf("expected code %q to be extracted", parseWant)
		}
		if parseEntry.Remediation == "" {
			parseT.Fatalf("code %q has no remediation", parseWant)
		}
	}
}

func TestExtractCodesHandlesCRLFSource(parseT *testing.T) {
	parseSource := strings.ReplaceAll(readMetadataSource(parseT), "\r\n", "\n")
	parseSource = strings.ReplaceAll(parseSource, "\n", "\r\n")
	parseCodes := ExtractCodes(parseSource)
	parseSeen := map[string]ErrorCode{}
	for _, parseCode := range parseCodes {
		parseSeen[parseCode.Code] = parseCode
	}
	parseAsync := parseSeen["GWC-RUNTIME-PANIC-ASYNC"]
	if parseAsync.Docs != "ACTIONABLE_ERRORS.md#gwc-runtime-panic-async" {
		parseT.Fatalf("async panic docs = %q, want ACTIONABLE_ERRORS.md#gwc-runtime-panic-async", parseAsync.Docs)
	}
}

// TestExtractorMatchesIndependentLiteralScan makes the drift guard
// NON-self-referential. TestErrorCodeReferenceIsGenerated compares the committed
// page against RenderPage(ExtractCodes(...)) — BOTH derived from the same
// extractor — so if ExtractCodes silently drops a code (a broken .Code/return
// regex), the regenerated page and the committed page omit it identically and the
// guard passes. Here an INDEPENDENT naive scan collects every quoted GWC-<UPPER>
// code literal in the source and asserts the structured extractor found all of
// them, so an extractor regression that drops a real code fails the suite.
func TestExtractorMatchesIndependentLiteralScan(parseT *testing.T) {
	parseSource := readMetadataSource(parseT)

	parseLiteral := regexp.MustCompile(`"(GWC-[A-Z0-9-]+)"`)
	parseNaive := map[string]bool{}
	for _, parseMatch := range parseLiteral.FindAllStringSubmatch(parseSource, -1) {
		parseCode := parseMatch[1]
		if parseCode == "GWC-RUNTIME-PANIC" {
			continue // the generic fallback ExtractCodes intentionally skips
		}
		parseNaive[parseCode] = true
	}
	if len(parseNaive) < 15 {
		parseT.Fatalf("independent literal scan found only %d codes; the cross-check pattern is broken", len(parseNaive))
	}

	parseExtracted := map[string]bool{}
	for _, parseCode := range ExtractCodes(parseSource) {
		parseExtracted[parseCode.Code] = true
	}
	for parseCode := range parseNaive {
		if !parseExtracted[parseCode] {
			parseT.Fatalf("code %q appears as a literal in the runtime source but the structured extractor dropped it — the drift guard is self-referential and would not catch this", parseCode)
		}
	}
}

// TestErrorCodeReferenceIsGenerated is the drift guard: the committed page must
// byte-match the output generated from the runtime source. Run with
// ERRORCODES_WRITE=1 to regenerate after adding or changing a code.
func TestErrorCodeReferenceIsGenerated(parseT *testing.T) {
	parseExpected := RenderPage(ExtractCodes(readMetadataSource(parseT)))
	parsePath := filepath.FromSlash(referencePageRel)

	if os.Getenv("ERRORCODES_WRITE") != "" {
		if parseErr := os.WriteFile(parsePath, []byte(parseExpected), 0o644); parseErr != nil {
			parseT.Fatalf("write reference page: %v", parseErr)
		}
		parseT.Logf("regenerated %s", parsePath)
		return
	}

	parseActual, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("read reference page (regenerate with ERRORCODES_WRITE=1): %v", parseErr)
	}
	if strings.ReplaceAll(string(parseActual), "\r\n", "\n") != parseExpected {
		parseT.Fatalf("error-code reference page is stale; regenerate with ERRORCODES_WRITE=1 go test ./docs/errorcodes/")
	}
}
