package capabilities

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot resolves the module root by walking up from the test working
// directory until a directory containing go.mod is found.
func repoRoot(parseT *testing.T) string {
	parseT.Helper()
	parseWd, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("getwd: %v", parseErr)
	}
	parseDir := parseWd
	for parseI := 0; parseI < 12; parseI++ {
		if _, parseStatErr := os.Stat(filepath.Join(parseDir, "go.mod")); parseStatErr == nil {
			return parseDir
		}
		parseParent := filepath.Dir(parseDir)
		if parseParent == parseDir {
			break
		}
		parseDir = parseParent
	}
	parseT.Fatalf("module root (go.mod) not found from %s", parseWd)
	return ""
}

// TestCapabilityReferencesResolve is the filesystem guard: for every Capability
// row in the curated table, it asserts that the example directory exists on disk
// (when ExampleSlug is non-empty) and that the chapter file exists under
// docs/REFERENCE_MANUAL/. This is the check that fails CI the moment a slug or
// chapter filename is renamed without updating the table.
func TestCapabilityReferencesResolve(parseT *testing.T) {
	parseRoot := repoRoot(parseT)
	parseCaps := Capabilities()
	if len(parseCaps) == 0 {
		parseT.Fatal("Capabilities() returned an empty slice; the table is missing")
	}
	for _, parseCap := range parseCaps {
		parseCap := parseCap // capture for clarity
		parseT.Run(parseCap.Name, func(parseInner *testing.T) {
			if parseCap.ExampleSlug != "" {
				parseExampleDir := filepath.Join(parseRoot, "examples", "public", parseCap.ExampleSlug)
				if _, parseStatErr := os.Stat(parseExampleDir); parseStatErr != nil {
					parseInner.Errorf("example directory %q not found on disk (ExampleSlug=%q): %v",
						parseExampleDir, parseCap.ExampleSlug, parseStatErr)
				}
			}
			parseChapterPath := filepath.Join(parseRoot, "docs", "REFERENCE_MANUAL", parseCap.Chapter)
			if _, parseStatErr := os.Stat(parseChapterPath); parseStatErr != nil {
				parseInner.Errorf("chapter file %q not found on disk (Chapter=%q): %v",
					parseChapterPath, parseCap.Chapter, parseStatErr)
			}
		})
	}
}

// TestCapabilityMatrixIsGenerated is the drift guard: the committed page must
// byte-match the output of RenderPage(Capabilities()). Run with
// CAPABILITIES_WRITE=1 to regenerate after updating the capability table.
func TestCapabilityMatrixIsGenerated(parseT *testing.T) {
	parseRoot := repoRoot(parseT)
	parseExpected := RenderPage(Capabilities())
	parsePath := filepath.Join(parseRoot, "docs", "REFERENCE_MANUAL", "capability-matrix.md")

	if os.Getenv("CAPABILITIES_WRITE") != "" {
		if parseWriteErr := os.WriteFile(parsePath, []byte(parseExpected), 0o644); parseWriteErr != nil {
			parseT.Fatalf("write capability matrix page: %v", parseWriteErr)
		}
		parseT.Logf("regenerated %s", parsePath)
		return
	}

	parseActual, parseReadErr := os.ReadFile(parsePath)
	if parseReadErr != nil {
		parseT.Fatalf("read capability matrix page (regenerate with CAPABILITIES_WRITE=1 go test ./docs/capabilities/): %v", parseReadErr)
	}
	if strings.ReplaceAll(string(parseActual), "\r\n", "\n") != parseExpected {
		parseT.Fatalf("capability-matrix.md is stale; regenerate with CAPABILITIES_WRITE=1 go test ./docs/capabilities/")
	}
}
