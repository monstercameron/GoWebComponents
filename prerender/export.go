package prerender

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// Target describes the computed output locations for one prerendered route.
type Target struct {
	RoutePath     string
	HTMLFile      string
	BootstrapFile string
	BootstrapURL  string
}

// RouteOutput holds the rendered HTML and optional bootstrap payload for one route.
type RouteOutput struct {
	HTML      string
	Bootstrap []byte
}

// Route configures one prerendered route export.
type Route struct {
	Path            string
	BootstrapFormat string
	Build           func(Target) (RouteOutput, error)
}

// ExportSummary reports the files emitted by Export.
type ExportSummary struct {
	HTMLFiles      []string
	BootstrapFiles []string
}

// Export renders the provided routes and writes HTML files plus optional bootstrap sidecars.
func Export(parseOutputDir string, parseRoutes []Route) (ExportSummary, error) {
	parseOutputDir = strings.TrimSpace(parseOutputDir)
	if parseOutputDir == "" {
		return ExportSummary{}, fmt.Errorf("prerender: output directory is required")
	}
	if len(parseRoutes) == 0 {
		return ExportSummary{}, fmt.Errorf("prerender: at least one route is required")
	}

	parseSummary := ExportSummary{
		HTMLFiles:      make([]string, 0, len(parseRoutes)),
		BootstrapFiles: make([]string, 0, len(parseRoutes)),
	}
	// Mid-loop errors return the partial summary (not a zeroed one) so callers know which files
	// were already written to disk and can clean them up.
	for _, parseRoute := range parseRoutes {
		parseNormalizedPath, parseErr := normalizeRoutePath(parseRoute.Path)
		if parseErr != nil {
			return parseSummary, parseErr
		}
		if parseRoute.Build == nil {
			return parseSummary, fmt.Errorf("prerender: build callback is required for route %q", parseNormalizedPath)
		}

		parseTarget := buildTarget(parseNormalizedPath, parseRoute.BootstrapFormat)
		parseResult, parseErr := parseRoute.Build(parseTarget)
		if parseErr != nil {
			return parseSummary, fmt.Errorf("prerender: build %q: %w", parseNormalizedPath, parseErr)
		}
		if strings.TrimSpace(parseResult.HTML) == "" {
			return parseSummary, fmt.Errorf("prerender: route %q returned empty html", parseNormalizedPath)
		}

		parseHtmlPath := filepath.Join(parseOutputDir, filepath.FromSlash(parseTarget.HTMLFile))
		if parseErr2 := writeFile(parseHtmlPath, []byte(parseResult.HTML)); parseErr2 != nil {
			return parseSummary, parseErr2
		}
		parseSummary.HTMLFiles = append(parseSummary.HTMLFiles, parseHtmlPath)

		if parseTarget.BootstrapFile == "" || len(parseResult.Bootstrap) == 0 {
			continue
		}
		parseBootstrapPath := filepath.Join(parseOutputDir, filepath.FromSlash(parseTarget.BootstrapFile))
		if parseErr3 := writeFile(parseBootstrapPath, parseResult.Bootstrap); parseErr3 != nil {
			return parseSummary, parseErr3
		}
		parseSummary.BootstrapFiles = append(parseSummary.BootstrapFiles, parseBootstrapPath)
	}

	return parseSummary, nil
}

func writeFile(parsePath string, parseData []byte) error {
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0o755); parseErr != nil {
		return fmt.Errorf("prerender: create directory for %q: %w", parsePath, parseErr)
	}
	if parseErr2 := os.WriteFile(parsePath, parseData, 0o644); parseErr2 != nil {
		return fmt.Errorf("prerender: write %q: %w", parsePath, parseErr2)
	}
	return nil
}

func normalizeRoutePath(parsePath string) (string, error) {
	parsePath = strings.TrimSpace(parsePath)
	if parsePath == "" {
		parsePath = "/"
	}
	if !strings.HasPrefix(parsePath, "/") {
		return "", fmt.Errorf("prerender: route %q must start with '/'", parsePath)
	}
	// Reject ".." segments: the route path is turned into an on-disk file path
	// (buildTarget → filepath.Join → os.WriteFile), so a traversal segment would
	// let a route write outside the export output directory. Defense-in-depth —
	// routes are normally developer-authored, but must not be trusted to be.
	for _, parseSegment := range strings.Split(parsePath, "/") {
		if parseSegment == ".." {
			return "", fmt.Errorf("prerender: route %q must not contain '..' segments", parsePath)
		}
	}
	if parsePath != "/" {
		parsePath = strings.TrimRight(parsePath, "/")
	}
	return parsePath, nil
}

func buildTarget(parseRoutePath string, format string) Target {
	parseTrimmed := strings.Trim(parseRoutePath, "/")
	parseHtmlFile := "index.html"
	if parseTrimmed != "" {
		parseHtmlFile = filepath.ToSlash(filepath.Join(parseTrimmed, "index.html"))
	}

	parseTarget := Target{
		RoutePath: parseRoutePath,
		HTMLFile:  parseHtmlFile,
	}
	if format == "" {
		return parseTarget
	}

	parseExt := bootstrapExtension(format)
	parseBootstrapFile := filepath.ToSlash(filepath.Join("bootstrap", parseTrimmed))
	if parseBootstrapFile == "bootstrap" {
		parseBootstrapFile = filepath.ToSlash(filepath.Join("bootstrap", "index"))
	}
	parseBootstrapFile += parseExt
	parseTarget.BootstrapFile = parseBootstrapFile
	parseTarget.BootstrapURL = "/" + strings.TrimPrefix(parseBootstrapFile, "/")
	return parseTarget
}

func bootstrapExtension(format string) string {
	switch format {
	case ui.SSRBootstrapFormatCBOR:
		return ".cbor"
	case ui.SSRBootstrapFormatJSON:
		return ".json"
	default:
		return ".data"
	}
}
