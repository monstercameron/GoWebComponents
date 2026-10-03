package prerender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExportRejectsBackslashTraversalRoute pins that a backslash-separated ".." segment cannot
// escape the export directory. On Windows filepath.Join treats "\" as a separator and cleans
// "a\..\..\evil" to "..\evil", so a route that passes the "/"-only ".." check would write outside
// the output directory.
func TestExportRejectsBackslashTraversalRoute(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutputDir := filepath.Join(parseRoot, "out")
	if parseErr := os.MkdirAll(parseOutputDir, 0o755); parseErr != nil {
		parseT.Fatal(parseErr)
	}

	for _, parsePath := range []string{`/a\..\..\evil`, `/..\evil`, `/a/b\..\..\..\evil`, "/nul\x00byte"} {
		_, parseErr := Export(parseOutputDir, []Route{{
			Path: parsePath,
			Build: func(Target) (RouteOutput, error) {
				return RouteOutput{HTML: "<html>pwned</html>"}, nil
			},
		}})
		if parseErr == nil {
			parseT.Errorf("route %q was accepted; expected an error", parsePath)
		}
	}

	// Whatever the platform, nothing may be written next to (outside) the output directory.
	parseEntries, parseErr := os.ReadDir(parseRoot)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	for _, parseEntry := range parseEntries {
		if parseEntry.Name() != "out" {
			parseT.Errorf("export wrote %q outside the output directory", parseEntry.Name())
		}
	}
}

// TestExportKeepsEveryOutputInsideOutputDir enumerates awkward but accepted route shapes and
// asserts every emitted file stays under the output directory.
func TestExportKeepsEveryOutputInsideOutputDir(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()
	parseRoutes := []Route{}
	for _, parsePath := range []string{"/", "/a//b", "/a/./b", "/%2e%2e/x", "/..a/b", "/a/", "//double"} {
		parseRoutes = append(parseRoutes, Route{
			Path:            parsePath,
			BootstrapFormat: "json",
			Build: func(Target) (RouteOutput, error) {
				return RouteOutput{HTML: "<html></html>", Bootstrap: []byte("{}")}, nil
			},
		})
	}
	parseSummary, parseErr := Export(parseOutputDir, parseRoutes)
	if parseErr != nil {
		parseT.Fatalf("export: %v", parseErr)
	}
	parseBase := filepath.Clean(parseOutputDir) + string(filepath.Separator)
	for _, parseFile := range append(parseSummary.HTMLFiles, parseSummary.BootstrapFiles...) {
		if !strings.HasPrefix(filepath.Clean(parseFile)+string(filepath.Separator), parseBase) {
			parseT.Errorf("file %q escapes %q", parseFile, parseOutputDir)
		}
	}
}
