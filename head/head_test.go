package head

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestComposeRendersManagedAndExplicitHeadTags(t *testing.T) {
	markup, err := ui.RenderToString(Compose(
		router.Metadata{
			Title:        "Docs",
			Description:  "Searchable docs",
			CanonicalURL: "https://example.com/docs",
		},
		Robots("index,follow"),
		OpenGraph("title", "Docs"),
		html.Preconnect("https://cdn.example.com"),
	))
	if err != nil {
		t.Fatalf("unexpected head render error: %v", err)
	}

	checks := []string{
		`<title data-gwc-router-managed="true">Docs</title>`,
		`<meta content="Searchable docs" data-gwc-router-managed="true" name="description">`,
		`<link data-gwc-router-managed="true" href="https://example.com/docs" rel="canonical">`,
		`<meta content="index,follow" name="robots">`,
		`<meta content="Docs" property="og:title">`,
		`<link href="https://cdn.example.com" rel="preconnect">`,
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected rendered head markup to contain %q, got %q", check, markup)
		}
	}
}

func TestSocialTagsDefaultTwitterCardAndOmitEmptyFields(t *testing.T) {
	markup, err := ui.RenderToString(SocialTags(SocialMetadata{
		Title:       "Docs",
		Description: "Searchable docs",
		ImageURL:    "https://example.com/preview.png",
		URL:         "https://example.com/docs",
	}))
	if err != nil {
		t.Fatalf("unexpected social tag render error: %v", err)
	}

	checks := []string{
		`<meta content="Docs" property="og:title">`,
		`<meta content="Searchable docs" property="og:description">`,
		`<meta content="https://example.com/preview.png" property="og:image">`,
		`<meta content="https://example.com/docs" property="og:url">`,
		`<meta content="summary_large_image" name="twitter:card">`,
		`<meta content="Docs" name="twitter:title">`,
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected social tag markup to contain %q, got %q", check, markup)
		}
	}

	if strings.Contains(markup, `property="og:type"`) {
		t.Fatalf("expected empty social fields to be omitted, got %q", markup)
	}
}

func TestRenderBundlesNonRouterHeadMetadata(t *testing.T) {
	markup, err := Render(Document{
		Metadata: router.Metadata{
			Title:        "Docs",
			Description:  "Searchable docs",
			CanonicalURL: "https://example.com/docs",
		},
		Robots: "index,follow",
		Social: SocialMetadata{
			Title:       "Docs",
			Description: "Searchable docs",
			ImageURL:    "https://example.com/preview.png",
			URL:         "https://example.com/docs",
		},
		Alternates: []AlternateLink{
			{HrefLang: "en", Href: "https://example.com/docs"},
			{HrefLang: "fr", Href: "https://example.com/fr/docs"},
		},
		ResourceHints: []ResourceHint{
			{Rel: "preconnect", Href: "https://cdn.example.com"},
			{Rel: "preload", Href: "https://cdn.example.com/app.js", As: "script"},
		},
		JSONLD: []JSONLDBlock{
			{ID: "article-jsonld", Value: map[string]interface{}{
				"@context": "https://schema.org",
				"@type":    "Article",
				"headline": "Docs <Guide>",
			}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected head render error: %v", err)
	}

	checks := []string{
		`<title data-gwc-router-managed="true">Docs</title>`,
		`<meta content="index,follow" name="robots">`,
		`<meta content="Docs" property="og:title">`,
		`<meta content="summary_large_image" name="twitter:card">`,
		`<link href="https://example.com/docs" hreflang="en" rel="alternate">`,
		`<link href="https://example.com/fr/docs" hreflang="fr" rel="alternate">`,
		`<link href="https://cdn.example.com" rel="preconnect">`,
		`<link as="script" href="https://cdn.example.com/app.js" rel="preload">`,
		`<script id="article-jsonld" type="application/ld+json">{"@context":"https://schema.org","@type":"Article","headline":"Docs \u003cGuide\u003e"}</script>`,
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected rendered head markup to contain %q, got %q", check, markup)
		}
	}
}

func TestRenderJSONLDEscapesScriptTermination(t *testing.T) {
	markup, err := RenderJSONLD(map[string]interface{}{
		"text": `</script><script>alert("x")</script>`,
	}, "dangerous")
	if err != nil {
		t.Fatalf("unexpected JSON-LD render error: %v", err)
	}

	checks := []string{
		`<script id="dangerous" type="application/ld+json">`,
		`{"text":"\u003c/script\u003e\u003cscript\u003ealert(\"x\")\u003c/script\u003e"}`,
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected JSON-LD markup to contain %q, got %q", check, markup)
		}
	}
}

func TestMergeAppliesRouteDefaultsOverridesAndReplacementRules(t *testing.T) {
	base := Document{
		Metadata: router.Metadata{
			Title:        "Docs",
			Description:  "Base description",
			CanonicalURL: "https://example.com/docs",
		},
		Robots: "index,follow",
		Social: SocialMetadata{
			Title:       "Docs",
			Description: "Base social description",
			URL:         "https://example.com/docs",
		},
		Alternates: []AlternateLink{
			{HrefLang: "en", Href: "https://example.com/docs"},
		},
		ResourceHints: []ResourceHint{
			{Rel: "preconnect", Href: "https://cdn.example.com"},
		},
		JSONLD: []JSONLDBlock{
			{ID: "docs-jsonld", Value: map[string]interface{}{"@type": "WebPage"}},
		},
		Extras: []ui.Node{
			MetaName("theme-color", "#08111d"),
		},
	}

	override := Document{
		Metadata: router.Metadata{
			Title:        "Guide",
			CanonicalURL: "https://example.com/docs/guide",
		},
		Social: SocialMetadata{
			Title:    "Guide social",
			ImageURL: "https://example.com/guide.png",
		},
		Alternates: []AlternateLink{
			{HrefLang: "fr", Href: "https://example.com/fr/guide"},
		},
		ResourceHints: []ResourceHint{
			{Rel: "preload", Href: "https://cdn.example.com/guide.js", As: "script"},
		},
		JSONLD: []JSONLDBlock{
			{ID: "guide-jsonld", Value: map[string]interface{}{"@type": "Article"}},
		},
		Extras: []ui.Node{
			MetaName("color-scheme", "dark"),
		},
	}

	merged := Merge(base, override, MergeOptions{
		ReplaceAlternates:    true,
		ReplaceResourceHints: true,
		ReplaceJSONLD:        true,
	})

	if merged.Metadata.Title != "Guide" {
		t.Fatalf("expected title override, got %#v", merged.Metadata)
	}
	if merged.Metadata.Description != "Base description" {
		t.Fatalf("expected description default to survive, got %#v", merged.Metadata)
	}
	if merged.Metadata.CanonicalURL != "https://example.com/docs/guide" {
		t.Fatalf("expected canonical override, got %#v", merged.Metadata)
	}
	if merged.Robots != "index,follow" {
		t.Fatalf("expected robots default to survive, got %q", merged.Robots)
	}
	if merged.Social.Title != "Guide social" || merged.Social.Description != "Base social description" || merged.Social.ImageURL != "https://example.com/guide.png" {
		t.Fatalf("expected social fields to merge, got %#v", merged.Social)
	}
	if len(merged.Alternates) != 1 || merged.Alternates[0].HrefLang != "fr" {
		t.Fatalf("expected alternates to be replaced, got %#v", merged.Alternates)
	}
	if len(merged.ResourceHints) != 1 || merged.ResourceHints[0].Rel != "preload" {
		t.Fatalf("expected resource hints to be replaced, got %#v", merged.ResourceHints)
	}
	if len(merged.JSONLD) != 1 || merged.JSONLD[0].ID != "guide-jsonld" {
		t.Fatalf("expected JSON-LD to be replaced, got %#v", merged.JSONLD)
	}
	if len(merged.Extras) != 2 {
		t.Fatalf("expected extras to append by default, got %d", len(merged.Extras))
	}
}

func TestResolveSupportsClearAndReplaceRulesAcrossRouteLayers(t *testing.T) {
	resolved := Resolve(
		RouteLayer{
			Document: Document{
				Metadata: router.Metadata{
					Title:        "Docs",
					Description:  "Base description",
					CanonicalURL: "https://example.com/docs",
				},
				Robots: "index,follow",
				Social: SocialMetadata{
					Title:       "Docs social",
					Description: "Base social",
				},
				Alternates: []AlternateLink{
					{HrefLang: "en", Href: "https://example.com/docs"},
				},
				ResourceHints: []ResourceHint{
					{Rel: "preconnect", Href: "https://cdn.example.com"},
				},
				JSONLD: []JSONLDBlock{
					{ID: "base-jsonld", Value: map[string]interface{}{"@type": "WebPage"}},
				},
			},
		},
		RouteLayer{
			Document: Document{
				Metadata: router.Metadata{
					Title: "Guide",
				},
				Social: SocialMetadata{
					Title: "Guide social",
				},
			},
		},
		RouteLayer{
			Document: Document{},
			Merge: MergeOptions{
				ClearRobots:          true,
				ClearSocial:          true,
				ReplaceAlternates:    true,
				ReplaceResourceHints: true,
				ReplaceJSONLD:        true,
			},
		},
	)

	if resolved.Metadata.Title != "Guide" || resolved.Metadata.Description != "Base description" {
		t.Fatalf("expected metadata defaults plus leaf override, got %#v", resolved.Metadata)
	}
	if resolved.Robots != "" {
		t.Fatalf("expected robots to be cleared, got %q", resolved.Robots)
	}
	if resolved.Social != (SocialMetadata{}) {
		t.Fatalf("expected social metadata to be cleared, got %#v", resolved.Social)
	}
	if len(resolved.Alternates) != 0 {
		t.Fatalf("expected alternates to be cleared, got %#v", resolved.Alternates)
	}
	if len(resolved.ResourceHints) != 0 {
		t.Fatalf("expected resource hints to be cleared, got %#v", resolved.ResourceHints)
	}
	if len(resolved.JSONLD) != 0 {
		t.Fatalf("expected JSON-LD to be cleared, got %#v", resolved.JSONLD)
	}
}

func TestHelpersReturnNilForBlankInputs(t *testing.T) {
	if MetaName("", "value") != nil {
		t.Fatal("expected MetaName to return nil for blank name")
	}
	if MetaProperty("og:title", "") != nil {
		t.Fatal("expected MetaProperty to return nil for blank content")
	}
	if LinkRel("canonical", "") != nil {
		t.Fatal("expected LinkRel to return nil for blank href")
	}
	if Hreflang("", "https://example.com/docs") != nil {
		t.Fatal("expected Hreflang to return nil for blank locale")
	}
	if OpenGraph("", "Docs") != nil {
		t.Fatal("expected OpenGraph to return nil for blank field")
	}
	if Twitter("", "summary") != nil {
		t.Fatal("expected Twitter to return nil for blank field")
	}
	if SocialTags(SocialMetadata{}) != nil {
		t.Fatal("expected SocialTags to return nil when all fields are empty")
	}
	if AlternateLinks(AlternateLink{}) != nil {
		t.Fatal("expected AlternateLinks to return nil when all hrefs are empty")
	}
	if ResourceHints(ResourceHint{}) != nil {
		t.Fatal("expected ResourceHints to return nil when all rels or hrefs are empty")
	}
}
