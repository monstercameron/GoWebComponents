package prerender

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestExportValidationErrors(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()

	if _, parseErr := Export("   ", []Route{{Path: "/", Build: func(Target) (RouteOutput, error) { return RouteOutput{HTML: "ok"}, nil }}}); parseErr == nil || !strings.Contains(parseErr.Error(), "output directory is required") {
		parseT.Fatalf("expected output directory validation error, got %v", parseErr)
	}
	if _, parseErr2 := Export(parseOutputDir, nil); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "at least one route is required") {
		parseT.Fatalf("expected routes validation error, got %v", parseErr2)
	}
}

func TestExportBuildCallbackAndResultErrors(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()

	if _, parseErr := Export(parseOutputDir, []Route{{Path: "/"}}); parseErr == nil || !strings.Contains(parseErr.Error(), "build callback is required") {
		parseT.Fatalf("expected missing build callback error, got %v", parseErr)
	}

	if _, parseErr2 := Export(parseOutputDir, []Route{{
		Path: "/docs",
		Build: func(Target) (RouteOutput, error) {
			return RouteOutput{}, errors.New("boom")
		},
	}}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "build \"/docs\": boom") {
		parseT.Fatalf("expected build callback wrapped error, got %v", parseErr2)
	}

	if _, parseErr3 := Export(parseOutputDir, []Route{{
		Path: "/docs",
		Build: func(Target) (RouteOutput, error) {
			return RouteOutput{HTML: "   "}, nil
		},
	}}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "returned empty html") {
		parseT.Fatalf("expected empty html validation error, got %v", parseErr3)
	}
}

func TestExportSkipsBootstrapWhenUnavailable(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()
	parseSummary, parseErr := Export(parseOutputDir, []Route{
		{
			Path: "/without-bootstrap-format",
			Build: func(parseTarget Target) (RouteOutput, error) {
				if parseTarget.BootstrapFile != "" || parseTarget.BootstrapURL != "" {
					parseT.Fatalf("expected no bootstrap target for empty format, got %+v", parseTarget)
				}
				return RouteOutput{HTML: "<html>ok</html>", Bootstrap: []byte(`{"ignored":true}`)}, nil
			},
		},
		{
			Path:            "/without-bootstrap-bytes",
			BootstrapFormat: ui.SSRBootstrapFormatJSON,
			Build: func(parseTarget2 Target) (RouteOutput, error) {
				if parseTarget2.BootstrapFile == "" || parseTarget2.BootstrapURL == "" {
					parseT.Fatalf("expected bootstrap target paths, got %+v", parseTarget2)
				}
				return RouteOutput{HTML: "<html>ok</html>"}, nil
			},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("Export() error = %v", parseErr)
	}
	if len(parseSummary.HTMLFiles) != 2 {
		parseT.Fatalf("expected two html files, got %+v", parseSummary.HTMLFiles)
	}
	if len(parseSummary.BootstrapFiles) != 0 {
		parseT.Fatalf("expected no bootstrap files to be emitted, got %+v", parseSummary.BootstrapFiles)
	}
}

func TestNormalizeRoutePathAndBuildTargetHelpers(parseT *testing.T) {
	if parseGot, parseErr := normalizeRoutePath(""); parseErr != nil || parseGot != "/" {
		parseT.Fatalf("expected blank route to normalize to root, got path=%q err=%v", parseGot, parseErr)
	}
	if parseGot2, parseErr2 := normalizeRoutePath("/docs/"); parseErr2 != nil || parseGot2 != "/docs" {
		parseT.Fatalf("expected trailing slash trim, got path=%q err=%v", parseGot2, parseErr2)
	}

	parseRootTarget := buildTarget("/", ui.SSRBootstrapFormatJSON)
	if parseRootTarget.HTMLFile != "index.html" || parseRootTarget.BootstrapFile != "bootstrap/index.json" || parseRootTarget.BootstrapURL != "/bootstrap/index.json" {
		parseT.Fatalf("unexpected root target shape: %+v", parseRootTarget)
	}

	parseNestedTarget := buildTarget("/docs/getting-started", "custom-format")
	if parseNestedTarget.HTMLFile != filepath.ToSlash(filepath.Join("docs", "getting-started", "index.html")) {
		parseT.Fatalf("unexpected nested html target: %+v", parseNestedTarget)
	}
	if !strings.HasSuffix(parseNestedTarget.BootstrapFile, ".data") {
		parseT.Fatalf("expected unknown format to default to .data, got %+v", parseNestedTarget)
	}
}

func TestBootstrapExtensionVariants(parseT *testing.T) {
	if parseExt := bootstrapExtension(ui.SSRBootstrapFormatCBOR); parseExt != ".cbor" {
		parseT.Fatalf("expected cbor extension, got %q", parseExt)
	}
	if parseExt2 := bootstrapExtension(ui.SSRBootstrapFormatJSON); parseExt2 != ".json" {
		parseT.Fatalf("expected json extension, got %q", parseExt2)
	}
	if parseExt3 := bootstrapExtension("custom"); parseExt3 != ".data" {
		parseT.Fatalf("expected fallback extension, got %q", parseExt3)
	}
}

// TestExportReturnsPartialSummaryOnMidLoopError proves a mid-loop failure returns the PARTIAL
// summary (files written before the error) rather than a zeroed one, so callers can clean up.
func TestExportReturnsPartialSummaryOnMidLoopError(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseRes, parseErr := Export(parseDir, []Route{
		{Path: "/", Build: func(Target) (RouteOutput, error) { return RouteOutput{HTML: "<p>ok</p>"}, nil }},
		{Path: "/bad", Build: func(Target) (RouteOutput, error) { return RouteOutput{HTML: ""}, nil }},
	})
	if parseErr == nil {
		parseT.Fatal("expected an error on the empty-html route")
	}
	if len(parseRes.HTMLFiles) != 1 {
		parseT.Fatalf("expected the first route's file in the partial summary, got %v", parseRes.HTMLFiles)
	}
}
