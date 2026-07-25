package router

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestMetadataNodeRendersManagedHeadTags(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(BuildMetadataNode(Metadata{
		Title:        "Docs",
		Description:  "Searchable docs",
		CanonicalURL: "https://example.com/docs?q=go&lang=en",
	}))
	if parseErr != nil {
		parseT.Fatalf("unexpected metadata render error: %v", parseErr)
	}

	parseChecks := []string{
		`<title data-gwc-router-managed="true">Docs</title>`,
		`<meta content="Searchable docs" data-gwc-router-managed="true" name="description">`,
		`<link data-gwc-router-managed="true" href="https://example.com/docs?q=go&amp;lang=en" rel="canonical">`,
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected metadata markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}
}

func TestMetadataNodeOmitsEmptyFields(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(BuildMetadataNode(Metadata{}))
	if parseErr != nil {
		parseT.Fatalf("unexpected metadata render error: %v", parseErr)
	}
	if parseMarkup != "" {
		parseT.Fatalf("expected empty metadata markup, got %q", parseMarkup)
	}
}

func TestMetadataNodeComposesWithExplicitSSRHeadTags(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(ui.Fragment(
		BuildMetadataNode(Metadata{
			Title:        "Docs",
			Description:  "Searchable docs",
			CanonicalURL: "https://example.com/docs",
		}),
		html.Meta(html.Props{Raw: map[string]any{
			"name":    "robots",
			"content": "index,follow",
		}}),
		html.Meta(html.Props{Raw: map[string]any{
			"property": "og:title",
			"content":  "Docs",
		}}),
		html.Link(html.Props{Raw: map[string]any{
			"rel":  "preconnect",
			"href": "https://cdn.example.com",
		}}),
	))
	if parseErr != nil {
		parseT.Fatalf("unexpected composed head render error: %v", parseErr)
	}

	parseChecks := []string{
		`<meta content="index,follow" name="robots">`,
		`<meta content="Docs" property="og:title">`,
		`<link href="https://cdn.example.com" rel="preconnect">`,
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected composed head markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}
}
