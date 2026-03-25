package prerender

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestExportValidationErrors(t *testing.T) {
	outputDir := t.TempDir()

	if _, err := Export("   ", []Route{{Path: "/", Build: func(Target) (RouteOutput, error) { return RouteOutput{HTML: "ok"}, nil }}}); err == nil || !strings.Contains(err.Error(), "output directory is required") {
		t.Fatalf("expected output directory validation error, got %v", err)
	}
	if _, err := Export(outputDir, nil); err == nil || !strings.Contains(err.Error(), "at least one route is required") {
		t.Fatalf("expected routes validation error, got %v", err)
	}
}

func TestExportBuildCallbackAndResultErrors(t *testing.T) {
	outputDir := t.TempDir()

	if _, err := Export(outputDir, []Route{{Path: "/"}}); err == nil || !strings.Contains(err.Error(), "build callback is required") {
		t.Fatalf("expected missing build callback error, got %v", err)
	}

	if _, err := Export(outputDir, []Route{{
		Path: "/docs",
		Build: func(Target) (RouteOutput, error) {
			return RouteOutput{}, errors.New("boom")
		},
	}}); err == nil || !strings.Contains(err.Error(), "build \"/docs\": boom") {
		t.Fatalf("expected build callback wrapped error, got %v", err)
	}

	if _, err := Export(outputDir, []Route{{
		Path: "/docs",
		Build: func(Target) (RouteOutput, error) {
			return RouteOutput{HTML: "   "}, nil
		},
	}}); err == nil || !strings.Contains(err.Error(), "returned empty html") {
		t.Fatalf("expected empty html validation error, got %v", err)
	}
}

func TestExportSkipsBootstrapWhenUnavailable(t *testing.T) {
	outputDir := t.TempDir()
	summary, err := Export(outputDir, []Route{
		{
			Path: "/without-bootstrap-format",
			Build: func(target Target) (RouteOutput, error) {
				if target.BootstrapFile != "" || target.BootstrapURL != "" {
					t.Fatalf("expected no bootstrap target for empty format, got %+v", target)
				}
				return RouteOutput{HTML: "<html>ok</html>", Bootstrap: []byte(`{"ignored":true}`)}, nil
			},
		},
		{
			Path:            "/without-bootstrap-bytes",
			BootstrapFormat: ui.SSRBootstrapFormatJSON,
			Build: func(target Target) (RouteOutput, error) {
				if target.BootstrapFile == "" || target.BootstrapURL == "" {
					t.Fatalf("expected bootstrap target paths, got %+v", target)
				}
				return RouteOutput{HTML: "<html>ok</html>"}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(summary.HTMLFiles) != 2 {
		t.Fatalf("expected two html files, got %+v", summary.HTMLFiles)
	}
	if len(summary.BootstrapFiles) != 0 {
		t.Fatalf("expected no bootstrap files to be emitted, got %+v", summary.BootstrapFiles)
	}
}

func TestNormalizeRoutePathAndBuildTargetHelpers(t *testing.T) {
	if got, err := normalizeRoutePath(""); err != nil || got != "/" {
		t.Fatalf("expected blank route to normalize to root, got path=%q err=%v", got, err)
	}
	if got, err := normalizeRoutePath("/docs/"); err != nil || got != "/docs" {
		t.Fatalf("expected trailing slash trim, got path=%q err=%v", got, err)
	}

	rootTarget := buildTarget("/", ui.SSRBootstrapFormatJSON)
	if rootTarget.HTMLFile != "index.html" || rootTarget.BootstrapFile != "bootstrap/index.json" || rootTarget.BootstrapURL != "/bootstrap/index.json" {
		t.Fatalf("unexpected root target shape: %+v", rootTarget)
	}

	nestedTarget := buildTarget("/docs/getting-started", "custom-format")
	if nestedTarget.HTMLFile != filepath.ToSlash(filepath.Join("docs", "getting-started", "index.html")) {
		t.Fatalf("unexpected nested html target: %+v", nestedTarget)
	}
	if !strings.HasSuffix(nestedTarget.BootstrapFile, ".data") {
		t.Fatalf("expected unknown format to default to .data, got %+v", nestedTarget)
	}
}

func TestBootstrapExtensionVariants(t *testing.T) {
	if ext := bootstrapExtension(ui.SSRBootstrapFormatCBOR); ext != ".cbor" {
		t.Fatalf("expected cbor extension, got %q", ext)
	}
	if ext := bootstrapExtension(ui.SSRBootstrapFormatJSON); ext != ".json" {
		t.Fatalf("expected json extension, got %q", ext)
	}
	if ext := bootstrapExtension("custom"); ext != ".data" {
		t.Fatalf("expected fallback extension, got %q", ext)
	}
}

