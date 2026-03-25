package ssr

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/head"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/prerender"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestRenderSnapshotsHTML(t *testing.T) {
	snapshot := Render(t, html.Div(html.Props{ID: "ssr-root"}, html.Text("SSR Ready")))
	if !snapshot.Contains("SSR Ready") || !snapshot.Contains("id=\"ssr-root\"") {
		t.Fatalf("expected snapshot HTML to contain rendered markup, got %q", snapshot.HTML)
	}
}

func TestRequirePayloadReadsTypedBootstrapValue(t *testing.T) {
	bootstrap := ui.SSRBootstrap{}
	if err := ui.RegisterBootstrapPayload(&bootstrap, "profile", map[string]string{"name": "Cam"}); err != nil {
		t.Fatalf("expected bootstrap registration to succeed, got %v", err)
	}
	profile := RequirePayload[map[string]string](t, bootstrap, "profile")
	if profile.Value["name"] != "Cam" {
		t.Fatalf("expected typed bootstrap payload, got %+v", profile.Value)
	}
}

func TestStructuredSnapshotParsesHeadMetadataAndJSONLD(t *testing.T) {
	markup, err := head.RenderToString(head.Document{
		Metadata: router.Metadata{
			Title:        "Docs",
			Description:  "Searchable docs",
			CanonicalURL: "https://example.com/docs",
		},
		Social: head.SocialMetadata{
			Title:       "Docs Social",
			Description: "Share docs",
		},
		JSONLD: []head.JSONLDBlock{{
			ID: "docs-jsonld",
			Value: map[string]any{
				"@context": "https://schema.org",
				"@type":    "Article",
			},
		}},
	})
	if err != nil {
		t.Fatalf("expected head render to succeed, got %v", err)
	}

	structured := Snapshot{HTML: markup}.Structured(t)
	structured.ApplyStructuredTitle(t, "Docs")
	structured.ApplyStructuredMetaName(t, "description", "Searchable docs")
	structured.ApplyStructuredMetaProperty(t, "og:title", "Docs Social")
	structured.ApplyStructuredMetaProperty(t, "og:description", "Share docs")
	structured.ApplyStructuredCanonicalURL(t, "https://example.com/docs")
	structured.ApplyStructuredJSONLDType(t, "docs-jsonld", "Article")
}

func TestLoadStaticExportReadsExportedRoutesAndBootstrapSidecars(t *testing.T) {
	outputDir := t.TempDir()
	headMarkup, err := head.RenderToString(head.Document{
		Metadata: router.Metadata{
			Title:        "Docs",
			Description:  "Exported docs",
			CanonicalURL: "https://example.com/docs",
		},
	})
	if err != nil {
		t.Fatalf("expected head render to succeed, got %v", err)
	}
	_, err = prerender.Export(outputDir, []prerender.Route{
		{
			Path:            "/",
			BootstrapFormat: ui.SSRBootstrapFormatJSON,
			Build: func(target prerender.Target) (prerender.RouteOutput, error) {
				return prerender.RouteOutput{
					HTML:      "<!doctype html><html><body>home</body></html>",
					Bootstrap: []byte(`{"page":"home"}`),
				}, nil
			},
		},
		{
			Path:            "/docs",
			BootstrapFormat: ui.SSRBootstrapFormatCBOR,
			Build: func(target prerender.Target) (prerender.RouteOutput, error) {
				return prerender.RouteOutput{
					HTML:      "<!doctype html><html><head>" + headMarkup + "</head><body>docs</body></html>",
					Bootstrap: []byte{1, 2, 3},
				}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("expected prerender export to succeed, got %v", err)
	}

	export := LoadStaticExport(t, outputDir)
	home, err := export.Route("/")
	if err != nil {
		t.Fatalf("expected home export, got %v", err)
	}
	if home.HTMLFile != "index.html" || string(home.Bootstrap) != `{"page":"home"}` {
		t.Fatalf("unexpected home export: %+v", home)
	}

	docs, err := export.Route("/docs")
	if err != nil {
		t.Fatalf("expected docs export, got %v", err)
	}
	if docs.HTMLFile != filepath.ToSlash(filepath.Join("docs", "index.html")) {
		t.Fatalf("unexpected docs html file: %+v", docs)
	}
	if docs.BootstrapFile != filepath.ToSlash(filepath.Join("bootstrap", "docs.cbor")) {
		t.Fatalf("unexpected docs bootstrap file: %+v", docs)
	}
	structured := docs.Snapshot.Structured(t)
	structured.ApplyStructuredTitle(t, "Docs")
	structured.ApplyStructuredCanonicalURL(t, "https://example.com/docs")
}

func TestStructuredSnapshotHelperFallbacks(t *testing.T) {
	snapshot := StructuredSnapshot{
		ScriptsByType: map[string][]ScriptTag{
			"application/ld+json": {
				{Type: "application/ld+json", Content: `{"@type":"FAQPage"}`},
			},
		},
	}
	if got := snapshot.MetaName("missing"); got != "" {
		t.Fatalf("expected missing meta name to return empty string, got %q", got)
	}
	if got := snapshot.MetaProperty("missing"); got != "" {
		t.Fatalf("expected missing meta property to return empty string, got %q", got)
	}
	if got := snapshot.CanonicalURL(); got != "" {
		t.Fatalf("expected missing canonical URL to return empty string, got %q", got)
	}
	if got := snapshot.JSONLD("unknown-id"); !strings.Contains(got, `"@type":"FAQPage"`) {
		t.Fatalf("expected JSONLD type fallback to return first script content, got %q", got)
	}
	if got := (StructuredSnapshot{}).JSONLD("none"); got != "" {
		t.Fatalf("expected empty structured snapshot JSONLD to be empty, got %q", got)
	}
}

func TestStructuredSnapshotParsesBootstrapScriptPayload(t *testing.T) {
	script, err := ui.RenderBootstrapScript(ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: "/docs"},
		Atoms: map[string]any{"theme": "dark"},
	}, "")
	if err != nil {
		t.Fatalf("expected bootstrap script render to succeed, got %v", err)
	}

	structured := Snapshot{HTML: "<!doctype html><html><head>" + script + "</head></html>"}.Structured(t)
	tag := structured.ApplyStructuredScriptID(t, ui.DefaultBootstrapScriptID)
	if tag.Type != "application/json" {
		t.Fatalf("expected bootstrap script type application/json, got %q", tag.Type)
	}

	payload := structured.ParseStructuredBootstrapScript(t, ui.DefaultBootstrapScriptID)
	route, ok := payload["route"].(map[string]any)
	if !ok {
		t.Fatalf("expected bootstrap payload route object, got %#v", payload["route"])
	}
	if path, _ := route["path"].(string); path != "/docs" {
		t.Fatalf("expected bootstrap route path /docs, got %q", path)
	}
}

func TestStaticExportRouteNormalizationAndMissingBootstrap(t *testing.T) {
	export := StaticExport{
		HTMLFiles: map[string]Snapshot{
			"index.html":      {HTML: "<html><body>home</body></html>"},
			"docs/index.html": {HTML: "<html><body>docs</body></html>"},
		},
		Bootstrap: map[string][]byte{
			"bootstrap/index.json": []byte(`{"page":"home"}`),
		},
	}

	root, err := export.Route("")
	if err != nil {
		t.Fatalf("expected empty route path to normalize to root, got %v", err)
	}
	if root.Path != "/" || root.HTMLFile != "index.html" {
		t.Fatalf("unexpected root route resolution: %+v", root)
	}

	docs, err := export.Route("/docs/")
	if err != nil {
		t.Fatalf("expected trailing slash route to normalize, got %v", err)
	}
	if docs.Path != "/docs" || docs.HTMLFile != filepath.ToSlash(filepath.Join("docs", "index.html")) {
		t.Fatalf("unexpected docs route resolution: %+v", docs)
	}
	if docs.BootstrapFile != "" || len(docs.Bootstrap) != 0 {
		t.Fatalf("expected docs route without matching bootstrap sidecar, got file=%q bytes=%d", docs.BootstrapFile, len(docs.Bootstrap))
	}

	if _, err := export.Route("docs"); err == nil || !strings.Contains(err.Error(), "must start with '/'") {
		t.Fatalf("expected invalid route path error for missing leading slash, got %v", err)
	}
	if _, err := export.Route("/unknown"); err == nil || !strings.Contains(err.Error(), "missing html file") {
		t.Fatalf("expected missing html file error, got %v", err)
	}
}

func TestSnapshotNodeTextNilBranch(t *testing.T) {
	if got := nodeText(nil); got != "" {
		t.Fatalf("expected nil nodeText to return empty string, got %q", got)
	}
}
