package ssr

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/head"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/prerender"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestRenderSnapshotsHTML(parseT *testing.T) {
	parseSnapshot := Render(parseT, html.Div(html.Props{ID: "ssr-root"}, html.Text("SSR Ready")))
	if !parseSnapshot.Contains("SSR Ready") || !parseSnapshot.Contains("id=\"ssr-root\"") {
		parseT.Fatalf("expected snapshot HTML to contain rendered markup, got %q", parseSnapshot.HTML)
	}
}

func TestRequirePayloadReadsTypedBootstrapValue(parseT *testing.T) {
	parseBootstrap := ui.SSRBootstrap{}
	if parseErr := ui.RegisterBootstrapPayload(&parseBootstrap, "profile", map[string]string{"name": "Cam"}); parseErr != nil {
		parseT.Fatalf("expected bootstrap registration to succeed, got %v", parseErr)
	}
	parseProfile := RequirePayload[map[string]string](parseT, parseBootstrap, "profile")
	if parseProfile.Value["name"] != "Cam" {
		parseT.Fatalf("expected typed bootstrap payload, got %+v", parseProfile.Value)
	}
}

func TestStructuredSnapshotParsesHeadMetadataAndJSONLD(parseT *testing.T) {
	parseMarkup, parseErr := head.RenderToString(head.Document{
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
	if parseErr != nil {
		parseT.Fatalf("expected head render to succeed, got %v", parseErr)
	}

	parseStructured := Snapshot{HTML: parseMarkup}.Structured(parseT)
	parseStructured.ApplyStructuredTitle(parseT, "Docs")
	parseStructured.ApplyStructuredMetaName(parseT, "description", "Searchable docs")
	parseStructured.ApplyStructuredMetaProperty(parseT, "og:title", "Docs Social")
	parseStructured.ApplyStructuredMetaProperty(parseT, "og:description", "Share docs")
	parseStructured.ApplyStructuredCanonicalURL(parseT, "https://example.com/docs")
	parseStructured.ApplyStructuredJSONLDType(parseT, "docs-jsonld", "Article")
}

// TestStructuredSnapshotCollectsMetadataAnywhereInTree pins that the structured
// collector walks the ENTIRE parsed tree, so head-level tags are found regardless
// of where the HTML parser places them (Structured wraps the snapshot in <div> and
// full-document-parses it, which can relocate head tags). A future change that
// only walked <head> would silently drop metadata a component rendered inline —
// a false-pass for every Apply* head-assertion helper. Here the tags are buried
// deep in nested body content, never in a <head>.
func TestStructuredSnapshotCollectsMetadataAnywhereInTree(parseT *testing.T) {
	parseHTML := "<section><article><div>" +
		"<title>Deep Title</title>" +
		`<meta name="description" content="deep desc">` +
		`<meta property="og:title" content="deep og">` +
		`<link rel="canonical" href="https://example.com/deep">` +
		`<script id="deep-ld" type="application/ld+json">{"@type":"Thing"}</script>` +
		"</div></article></section>"

	parseStructured := Snapshot{HTML: parseHTML}.Structured(parseT)
	parseStructured.ApplyStructuredTitle(parseT, "Deep Title")
	parseStructured.ApplyStructuredMetaName(parseT, "description", "deep desc")
	parseStructured.ApplyStructuredMetaProperty(parseT, "og:title", "deep og")
	parseStructured.ApplyStructuredCanonicalURL(parseT, "https://example.com/deep")
	if parseStructured.JSONLD("deep-ld") == "" {
		parseT.Fatal("expected the JSON-LD script buried in body content to be collected")
	}
}

func TestLoadStaticExportReadsExportedRoutesAndBootstrapSidecars(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()
	parseHeadMarkup, parseErr := head.RenderToString(head.Document{
		Metadata: router.Metadata{
			Title:        "Docs",
			Description:  "Exported docs",
			CanonicalURL: "https://example.com/docs",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected head render to succeed, got %v", parseErr)
	}
	_, parseErr = prerender.Export(parseOutputDir, []prerender.Route{
		{
			Path:            "/",
			BootstrapFormat: ui.SSRBootstrapFormatJSON,
			Build: func(parseTarget prerender.Target) (prerender.RouteOutput, error) {
				return prerender.RouteOutput{
					HTML:      "<!doctype html><html><body>home</body></html>",
					Bootstrap: []byte(`{"page":"home"}`),
				}, nil
			},
		},
		{
			Path:            "/docs",
			BootstrapFormat: ui.SSRBootstrapFormatCBOR,
			Build: func(parseTarget2 prerender.Target) (prerender.RouteOutput, error) {
				return prerender.RouteOutput{
					HTML:      "<!doctype html><html><head>" + parseHeadMarkup + "</head><body>docs</body></html>",
					Bootstrap: []byte{1, 2, 3},
				}, nil
			},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected prerender export to succeed, got %v", parseErr)
	}

	parseExport := LoadStaticExport(parseT, parseOutputDir)
	parseHome, parseErr := parseExport.Route("/")
	if parseErr != nil {
		parseT.Fatalf("expected home export, got %v", parseErr)
	}
	if parseHome.HTMLFile != "index.html" || string(parseHome.Bootstrap) != `{"page":"home"}` {
		parseT.Fatalf("unexpected home export: %+v", parseHome)
	}

	parseDocs, parseErr := parseExport.Route("/docs")
	if parseErr != nil {
		parseT.Fatalf("expected docs export, got %v", parseErr)
	}
	if parseDocs.HTMLFile != filepath.ToSlash(filepath.Join("docs", "index.html")) {
		parseT.Fatalf("unexpected docs html file: %+v", parseDocs)
	}
	if parseDocs.BootstrapFile != filepath.ToSlash(filepath.Join("bootstrap", "docs.cbor")) {
		parseT.Fatalf("unexpected docs bootstrap file: %+v", parseDocs)
	}
	parseStructured := parseDocs.Snapshot.Structured(parseT)
	parseStructured.ApplyStructuredTitle(parseT, "Docs")
	parseStructured.ApplyStructuredCanonicalURL(parseT, "https://example.com/docs")
}

func TestStructuredSnapshotHelperFallbacks(parseT *testing.T) {
	parseSnapshot := StructuredSnapshot{
		ScriptsByType: map[string][]ScriptTag{
			"application/ld+json": {
				{Type: "application/ld+json", Content: `{"@type":"FAQPage"}`},
			},
		},
	}
	if parseGot := parseSnapshot.MetaName("missing"); parseGot != "" {
		parseT.Fatalf("expected missing meta name to return empty string, got %q", parseGot)
	}
	if parseGot2 := parseSnapshot.MetaProperty("missing"); parseGot2 != "" {
		parseT.Fatalf("expected missing meta property to return empty string, got %q", parseGot2)
	}
	if parseGot3 := parseSnapshot.CanonicalURL(); parseGot3 != "" {
		parseT.Fatalf("expected missing canonical URL to return empty string, got %q", parseGot3)
	}
	if parseGot4 := parseSnapshot.JSONLD("unknown-id"); !strings.Contains(parseGot4, `"@type":"FAQPage"`) {
		parseT.Fatalf("expected JSONLD type fallback to return first script content, got %q", parseGot4)
	}
	if parseGot5 := (StructuredSnapshot{}).JSONLD("none"); parseGot5 != "" {
		parseT.Fatalf("expected empty structured snapshot JSONLD to be empty, got %q", parseGot5)
	}
}

func TestStructuredSnapshotParsesBootstrapScriptPayload(parseT *testing.T) {
	parseScript, parseErr := ui.RenderBootstrapScript(ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: "/docs"},
		Atoms: map[string]any{"theme": "dark"},
	}, "")
	if parseErr != nil {
		parseT.Fatalf("expected bootstrap script render to succeed, got %v", parseErr)
	}

	parseStructured := Snapshot{HTML: "<!doctype html><html><head>" + parseScript + "</head></html>"}.Structured(parseT)
	parseTag := parseStructured.ApplyStructuredScriptID(parseT, ui.DefaultBootstrapScriptID)
	if parseTag.Type != "application/json" {
		parseT.Fatalf("expected bootstrap script type application/json, got %q", parseTag.Type)
	}

	parsePayload := parseStructured.ParseStructuredBootstrapScript(parseT, ui.DefaultBootstrapScriptID)
	parseRoute, parseOk := parsePayload["route"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected bootstrap payload route object, got %#v", parsePayload["route"])
	}
	if parsePath, _ := parseRoute["path"].(string); parsePath != "/docs" {
		parseT.Fatalf("expected bootstrap route path /docs, got %q", parsePath)
	}
}

func TestStaticExportRouteNormalizationAndMissingBootstrap(parseT *testing.T) {
	parseExport := StaticExport{
		HTMLFiles: map[string]Snapshot{
			"index.html":      {HTML: "<html><body>home</body></html>"},
			"docs/index.html": {HTML: "<html><body>docs</body></html>"},
		},
		Bootstrap: map[string][]byte{
			"bootstrap/index.json": []byte(`{"page":"home"}`),
		},
	}

	parseRoot, parseErr := parseExport.Route("")
	if parseErr != nil {
		parseT.Fatalf("expected empty route path to normalize to root, got %v", parseErr)
	}
	if parseRoot.Path != "/" || parseRoot.HTMLFile != "index.html" {
		parseT.Fatalf("unexpected root route resolution: %+v", parseRoot)
	}

	parseDocs, parseErr := parseExport.Route("/docs/")
	if parseErr != nil {
		parseT.Fatalf("expected trailing slash route to normalize, got %v", parseErr)
	}
	if parseDocs.Path != "/docs" || parseDocs.HTMLFile != filepath.ToSlash(filepath.Join("docs", "index.html")) {
		parseT.Fatalf("unexpected docs route resolution: %+v", parseDocs)
	}
	if parseDocs.BootstrapFile != "" || len(parseDocs.Bootstrap) != 0 {
		parseT.Fatalf("expected docs route without matching bootstrap sidecar, got file=%q bytes=%d", parseDocs.BootstrapFile, len(parseDocs.Bootstrap))
	}

	if _, parseErr2 := parseExport.Route("docs"); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "must start with '/'") {
		parseT.Fatalf("expected invalid route path error for missing leading slash, got %v", parseErr2)
	}
	if _, parseErr3 := parseExport.Route("/unknown"); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "missing html file") {
		parseT.Fatalf("expected missing html file error, got %v", parseErr3)
	}
}

func TestSnapshotNodeTextNilBranch(parseT *testing.T) {
	if parseGot := nodeText(nil); parseGot != "" {
		parseT.Fatalf("expected nil nodeText to return empty string, got %q", parseGot)
	}
}
