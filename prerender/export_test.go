package prerender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func TestExportWritesHTMLAndBootstrapFiles(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()

	parseSummary, parseErr := Export(parseOutputDir, []Route{
		{
			Path:            "/",
			BootstrapFormat: ui.SSRBootstrapFormatJSON,
			Build: func(parseTarget Target) (RouteOutput, error) {
				if parseTarget.HTMLFile != "index.html" {
					parseT.Fatalf("unexpected root html file: %s", parseTarget.HTMLFile)
				}
				if parseTarget.BootstrapURL != "/bootstrap/index.json" {
					parseT.Fatalf("unexpected root bootstrap url: %s", parseTarget.BootstrapURL)
				}
				return RouteOutput{
					HTML:      "<!doctype html><html><body>home</body></html>",
					Bootstrap: []byte(`{"page":"home"}`),
				}, nil
			},
		},
		{
			Path:            "/docs/getting-started",
			BootstrapFormat: ui.SSRBootstrapFormatCBOR,
			Build: func(parseTarget2 Target) (RouteOutput, error) {
				if parseTarget2.HTMLFile != "docs/getting-started/index.html" {
					parseT.Fatalf("unexpected nested html file: %s", parseTarget2.HTMLFile)
				}
				if parseTarget2.BootstrapURL != "/bootstrap/docs/getting-started.cbor" {
					parseT.Fatalf("unexpected nested bootstrap url: %s", parseTarget2.BootstrapURL)
				}
				return RouteOutput{
					HTML:      "<!doctype html><html><body>docs</body></html>",
					Bootstrap: []byte{1, 2, 3, 4},
				}, nil
			},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("Export() error = %v", parseErr)
	}

	if len(parseSummary.HTMLFiles) != 2 {
		parseT.Fatalf("expected 2 html files, got %d", len(parseSummary.HTMLFiles))
	}
	if len(parseSummary.BootstrapFiles) != 2 {
		parseT.Fatalf("expected 2 bootstrap files, got %d", len(parseSummary.BootstrapFiles))
	}

	assertFileContains(parseT, filepath.Join(parseOutputDir, "index.html"), "home")
	assertFileContains(parseT, filepath.Join(parseOutputDir, "docs", "getting-started", "index.html"), "docs")
	assertFileContains(parseT, filepath.Join(parseOutputDir, "bootstrap", "index.json"), `"page":"home"`)
	assertFileBytes(parseT, filepath.Join(parseOutputDir, "bootstrap", "docs", "getting-started.cbor"), []byte{1, 2, 3, 4})
}

func TestExportRejectsInvalidRoutes(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()

	_, parseErr := Export(parseOutputDir, []Route{{
		Path:  "docs",
		Build: func(Target) (RouteOutput, error) { return RouteOutput{HTML: "nope"}, nil },
	}})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "must start with '/'") {
		parseT.Fatalf("expected invalid route error, got %v", parseErr)
	}
}

func TestExportRejectsPathTraversalRoute(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()

	_, parseErr := Export(parseOutputDir, []Route{{
		Path:  "/../../escape",
		Build: func(Target) (RouteOutput, error) { return RouteOutput{HTML: "nope"}, nil },
	}})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "'..'") {
		parseT.Fatalf("expected path-traversal rejection, got %v", parseErr)
	}
}

func assertFileContains(parseT *testing.T, parsePath string, parseWant string) {
	parseT.Helper()
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("read %s: %v", parsePath, parseErr)
	}
	if !strings.Contains(string(parseData), parseWant) {
		parseT.Fatalf("%s does not contain %q: %s", parsePath, parseWant, string(parseData))
	}
}

func assertFileBytes(parseT *testing.T, parsePath string, parseWant []byte) {
	parseT.Helper()
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("read %s: %v", parsePath, parseErr)
	}
	if string(parseData) != string(parseWant) {
		parseT.Fatalf("%s bytes mismatch: got %v want %v", parsePath, parseData, parseWant)
	}
}
