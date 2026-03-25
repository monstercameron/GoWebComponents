package prerender

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/ui"
)

// Target describes the computed output locations for one prerendered route.
type Target struct {
	RoutePath    string
	HTMLFile     string
	BootstrapFile string
	BootstrapURL string
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
func Export(outputDir string, routes []Route) (ExportSummary, error) {
	outputDir = strings.TrimSpace(outputDir)
	if outputDir == "" {
		return ExportSummary{}, fmt.Errorf("prerender: output directory is required")
	}
	if len(routes) == 0 {
		return ExportSummary{}, fmt.Errorf("prerender: at least one route is required")
	}

	summary := ExportSummary{
		HTMLFiles:      make([]string, 0, len(routes)),
		BootstrapFiles: make([]string, 0, len(routes)),
	}
	for _, route := range routes {
		normalizedPath, err := normalizeRoutePath(route.Path)
		if err != nil {
			return ExportSummary{}, err
		}
		if route.Build == nil {
			return ExportSummary{}, fmt.Errorf("prerender: build callback is required for route %q", normalizedPath)
		}

		target := buildTarget(normalizedPath, route.BootstrapFormat)
		result, err := route.Build(target)
		if err != nil {
			return ExportSummary{}, fmt.Errorf("prerender: build %q: %w", normalizedPath, err)
		}
		if strings.TrimSpace(result.HTML) == "" {
			return ExportSummary{}, fmt.Errorf("prerender: route %q returned empty html", normalizedPath)
		}

		htmlPath := filepath.Join(outputDir, filepath.FromSlash(target.HTMLFile))
		if err := writeFile(htmlPath, []byte(result.HTML)); err != nil {
			return ExportSummary{}, err
		}
		summary.HTMLFiles = append(summary.HTMLFiles, htmlPath)

		if target.BootstrapFile == "" || len(result.Bootstrap) == 0 {
			continue
		}
		bootstrapPath := filepath.Join(outputDir, filepath.FromSlash(target.BootstrapFile))
		if err := writeFile(bootstrapPath, result.Bootstrap); err != nil {
			return ExportSummary{}, err
		}
		summary.BootstrapFiles = append(summary.BootstrapFiles, bootstrapPath)
	}

	return summary, nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prerender: create directory for %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("prerender: write %q: %w", path, err)
	}
	return nil
}

func normalizeRoutePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("prerender: route %q must start with '/'", path)
	}
	if path != "/" {
		path = strings.TrimRight(path, "/")
	}
	return path, nil
}

func buildTarget(routePath string, format string) Target {
	trimmed := strings.Trim(routePath, "/")
	htmlFile := "index.html"
	if trimmed != "" {
		htmlFile = filepath.ToSlash(filepath.Join(trimmed, "index.html"))
	}

	target := Target{
		RoutePath: routePath,
		HTMLFile:  htmlFile,
	}
	if format == "" {
		return target
	}

	ext := bootstrapExtension(format)
	bootstrapFile := filepath.ToSlash(filepath.Join("bootstrap", trimmed))
	if bootstrapFile == "bootstrap" {
		bootstrapFile = filepath.ToSlash(filepath.Join("bootstrap", "index"))
	}
	bootstrapFile += ext
	target.BootstrapFile = bootstrapFile
	target.BootstrapURL = "/" + strings.TrimPrefix(bootstrapFile, "/")
	return target
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
