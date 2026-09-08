package head

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestComposeRendersManagedAndExplicitHeadTags(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(Compose(
		router.Metadata{
			Title:        "Docs",
			Description:  "Searchable docs",
			CanonicalURL: "https://example.com/docs",
		},
		Robots("index,follow"),
		OpenGraph("title", "Docs"),
		html.Preconnect("https://cdn.example.com"),
	))
	if parseErr != nil {
		parseT.Fatalf("unexpected head render error: %v", parseErr)
	}

	parseChecks := []string{
		`<title data-gwc-router-managed="true">Docs</title>`,
		`<meta content="Searchable docs" data-gwc-router-managed="true" name="description">`,
		`<link data-gwc-router-managed="true" href="https://example.com/docs" rel="canonical">`,
		`<meta content="index,follow" name="robots">`,
		`<meta content="Docs" property="og:title">`,
		`<link href="https://cdn.example.com" rel="preconnect">`,
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected rendered head markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}
}

func TestSocialTagsDefaultTwitterCardAndOmitEmptyFields(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(SocialTags(SocialMetadata{
		Title:       "Docs",
		Description: "Searchable docs",
		ImageURL:    "https://example.com/preview.png",
		URL:         "https://example.com/docs",
	}))
	if parseErr != nil {
		parseT.Fatalf("unexpected social tag render error: %v", parseErr)
	}

	parseChecks := []string{
		`<meta content="Docs" property="og:title">`,
		`<meta content="Searchable docs" property="og:description">`,
		`<meta content="https://example.com/preview.png" property="og:image">`,
		`<meta content="https://example.com/docs" property="og:url">`,
		`<meta content="summary_large_image" name="twitter:card">`,
		`<meta content="Docs" name="twitter:title">`,
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected social tag markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}

	if strings.Contains(parseMarkup, `property="og:type"`) {
		parseT.Fatalf("expected empty social fields to be omitted, got %q", parseMarkup)
	}
}

func TestRenderBundlesNonRouterHeadMetadata(parseT *testing.T) {
	parseMarkup, parseErr := RenderToString(Document{
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
			{ID: "article-jsonld", Value: map[string]any{
				"@context": "https://schema.org",
				"@type":    "Article",
				"headline": "Docs <Guide>",
			}},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("unexpected head render error: %v", parseErr)
	}

	parseChecks := []string{
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
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected rendered head markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}
}

func TestRenderJSONLDEscapesScriptTermination(parseT *testing.T) {
	parseMarkup, parseErr := RenderJSONLD(map[string]any{
		"text": `</script><script>alert("x")</script>`,
	}, "dangerous")
	if parseErr != nil {
		parseT.Fatalf("unexpected JSON-LD render error: %v", parseErr)
	}

	parseChecks := []string{
		`<script id="dangerous" type="application/ld+json">`,
		`{"text":"\u003c/script\u003e\u003cscript\u003ealert(\"x\")\u003c/script\u003e"}`,
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected JSON-LD markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}
}

func TestMergeAppliesRouteDefaultsOverridesAndReplacementRules(parseT *testing.T) {
	parseBase := Document{
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
			{ID: "docs-jsonld", Value: map[string]any{"@type": "WebPage"}},
		},
		Extras: []ui.Node{
			MetaName("theme-color", "#08111d"),
		},
	}

	parseOverride := Document{
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
			{ID: "guide-jsonld", Value: map[string]any{"@type": "Article"}},
		},
		Extras: []ui.Node{
			MetaName("color-scheme", "dark"),
		},
	}

	parseMerged := Merge(parseBase, parseOverride, MergeOptions{
		ReplaceAlternates:    true,
		ReplaceResourceHints: true,
		ReplaceJSONLD:        true,
	})

	if parseMerged.Metadata.Title != "Guide" {
		parseT.Fatalf("expected title override, got %#v", parseMerged.Metadata)
	}
	if parseMerged.Metadata.Description != "Base description" {
		parseT.Fatalf("expected description default to survive, got %#v", parseMerged.Metadata)
	}
	if parseMerged.Metadata.CanonicalURL != "https://example.com/docs/guide" {
		parseT.Fatalf("expected canonical override, got %#v", parseMerged.Metadata)
	}
	if parseMerged.Robots != "index,follow" {
		parseT.Fatalf("expected robots default to survive, got %q", parseMerged.Robots)
	}
	if parseMerged.Social.Title != "Guide social" || parseMerged.Social.Description != "Base social description" || parseMerged.Social.ImageURL != "https://example.com/guide.png" {
		parseT.Fatalf("expected social fields to merge, got %#v", parseMerged.Social)
	}
	if len(parseMerged.Alternates) != 1 || parseMerged.Alternates[0].HrefLang != "fr" {
		parseT.Fatalf("expected alternates to be replaced, got %#v", parseMerged.Alternates)
	}
	if len(parseMerged.ResourceHints) != 1 || parseMerged.ResourceHints[0].Rel != "preload" {
		parseT.Fatalf("expected resource hints to be replaced, got %#v", parseMerged.ResourceHints)
	}
	if len(parseMerged.JSONLD) != 1 || parseMerged.JSONLD[0].ID != "guide-jsonld" {
		parseT.Fatalf("expected JSON-LD to be replaced, got %#v", parseMerged.JSONLD)
	}
	if len(parseMerged.Extras) != 2 {
		parseT.Fatalf("expected extras to append by default, got %d", len(parseMerged.Extras))
	}
}

func TestResolveSupportsClearAndReplaceRulesAcrossRouteLayers(parseT *testing.T) {
	parseResolved := Resolve(
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
					{ID: "base-jsonld", Value: map[string]any{"@type": "WebPage"}},
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

	if parseResolved.Metadata.Title != "Guide" || parseResolved.Metadata.Description != "Base description" {
		parseT.Fatalf("expected metadata defaults plus leaf override, got %#v", parseResolved.Metadata)
	}
	if parseResolved.Robots != "" {
		parseT.Fatalf("expected robots to be cleared, got %q", parseResolved.Robots)
	}
	if parseResolved.Social != (SocialMetadata{}) {
		parseT.Fatalf("expected social metadata to be cleared, got %#v", parseResolved.Social)
	}
	if len(parseResolved.Alternates) != 0 {
		parseT.Fatalf("expected alternates to be cleared, got %#v", parseResolved.Alternates)
	}
	if len(parseResolved.ResourceHints) != 0 {
		parseT.Fatalf("expected resource hints to be cleared, got %#v", parseResolved.ResourceHints)
	}
	if len(parseResolved.JSONLD) != 0 {
		parseT.Fatalf("expected JSON-LD to be cleared, got %#v", parseResolved.JSONLD)
	}
}

func TestHelpersReturnNilForBlankInputs(parseT *testing.T) {
	if MetaName("", "value") != nil {
		parseT.Fatal("expected MetaName to return nil for blank name")
	}
	if MetaProperty("og:title", "") != nil {
		parseT.Fatal("expected MetaProperty to return nil for blank content")
	}
	if LinkRel("canonical", "") != nil {
		parseT.Fatal("expected LinkRel to return nil for blank href")
	}
	if Hreflang("", "https://example.com/docs") != nil {
		parseT.Fatal("expected Hreflang to return nil for blank locale")
	}
	if OpenGraph("", "Docs") != nil {
		parseT.Fatal("expected OpenGraph to return nil for blank field")
	}
	if Twitter("", "summary") != nil {
		parseT.Fatal("expected Twitter to return nil for blank field")
	}
	if SocialTags(SocialMetadata{}) != nil {
		parseT.Fatal("expected SocialTags to return nil when all fields are empty")
	}
	if AlternateLinks(AlternateLink{}) != nil {
		parseT.Fatal("expected AlternateLinks to return nil when all hrefs are empty")
	}
	if ResourceHints(ResourceHint{}) != nil {
		parseT.Fatal("expected ResourceHints to return nil when all rels or hrefs are empty")
	}
}
