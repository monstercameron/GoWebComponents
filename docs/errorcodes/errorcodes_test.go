package errorcodes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/errcode"
)

// referencePageRel is the committed generated page, relative to this package.
const referencePageRel = "../REFERENCE_MANUAL/error-codes.md"

// TestRenderPageCoversEveryRegisteredCode checks every code appears as a
// heading, so the lowercase Docs anchor of each entry resolves on the page.
func TestRenderPageCoversEveryRegisteredCode(parseT *testing.T) {
	parseEntries := errcode.All()
	if len(parseEntries) < 30 {
		parseT.Fatalf("expected the full diagnostic-code set, got only %d", len(parseEntries))
	}
	parsePage := RenderPage(parseEntries)
	for _, parseEntry := range parseEntries {
		if !strings.Contains(parsePage, "\n## "+string(parseEntry.Code)+"\n") {
			parseT.Fatalf("page is missing heading for %s", parseEntry.Code)
		}
		parseAnchor := strings.ToLower(string(parseEntry.Code))
		if !strings.HasSuffix(parseEntry.Docs, "#"+parseAnchor) {
			parseT.Fatalf("%s docs %q does not end in #%s", parseEntry.Code, parseEntry.Docs, parseAnchor)
		}
		if !strings.Contains(parsePage, "- **next:** "+parseEntry.Remediation+"\n") {
			parseT.Fatalf("page is missing remediation for %s", parseEntry.Code)
		}
	}
}

// TestErrorCodeReferenceIsGenerated is the drift guard: the committed page must
// byte-match the output generated from the registry. Run with ERRORCODES_WRITE=1
// to regenerate after adding or changing a code.
func TestErrorCodeReferenceIsGenerated(parseT *testing.T) {
	parseExpected := RenderPage(errcode.All())
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
