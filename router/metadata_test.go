package router

import (
	"strings"
	"testing"

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
