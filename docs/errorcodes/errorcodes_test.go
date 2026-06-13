package errorcodes

import (
	"os"
	"path/filepath"
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
	parseSource := strings.ReplaceAll(readMetadataSource(parseT), "\n", "\r\n")
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
