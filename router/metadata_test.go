package router

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestMetadataNodeRendersManagedHeadTags(t *testing.T) {
	markup, err := ui.RenderToString(MetadataNode(Metadata{
		Title:        "Docs",
		Description:  "Searchable docs",
		CanonicalURL: "https://example.com/docs?q=go&lang=en",
	}))
	if err != nil {
		t.Fatalf("unexpected metadata render error: %v", err)
	}

	checks := []string{
		`<title data-gwc-router-managed="true">Docs</title>`,
		`<meta content="Searchable docs" data-gwc-router-managed="true" name="description">`,
		`<link data-gwc-router-managed="true" href="https://example.com/docs?q=go&amp;lang=en" rel="canonical">`,
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected metadata markup to contain %q, got %q", check, markup)
		}
	}
}

func TestMetadataNodeOmitsEmptyFields(t *testing.T) {
	markup, err := ui.RenderToString(MetadataNode(Metadata{}))
	if err != nil {
		t.Fatalf("unexpected metadata render error: %v", err)
	}
	if markup != "" {
		t.Fatalf("expected empty metadata markup, got %q", markup)
	}
}

func TestMetadataNodeComposesWithExplicitSSRHeadTags(t *testing.T) {
	markup, err := ui.RenderToString(ui.Fragment(
		MetadataNode(Metadata{
			Title:        "Docs",
			Description:  "Searchable docs",
			CanonicalURL: "https://example.com/docs",
		}),
		html.Meta(html.Props{Raw: map[string]interface{}{
			"name":    "robots",
			"content": "index,follow",
		}}),
		html.Meta(html.Props{Raw: map[string]interface{}{
			"property": "og:title",
			"content":  "Docs",
		}}),
		html.Link(html.Props{Raw: map[string]interface{}{
			"rel":  "preconnect",
			"href": "https://cdn.example.com",
		}}),
	))
	if err != nil {
		t.Fatalf("unexpected composed head render error: %v", err)
	}

	checks := []string{
		`<meta content="index,follow" name="robots">`,
		`<meta content="Docs" property="og:title">`,
		`<link href="https://cdn.example.com" rel="preconnect">`,
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected composed head markup to contain %q, got %q", check, markup)
		}
	}
}
