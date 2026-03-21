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
	if OpenGraph("", "Docs") != nil {
		t.Fatal("expected OpenGraph to return nil for blank field")
	}
	if Twitter("", "summary") != nil {
		t.Fatal("expected Twitter to return nil for blank field")
	}
	if SocialTags(SocialMetadata{}) != nil {
		t.Fatal("expected SocialTags to return nil when all fields are empty")
	}
}
