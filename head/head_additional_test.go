package head

import (
	"reflect"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/html"
	"github.com/monstercameron/GoWebComponents/v7/router"
	"github.com/monstercameron/GoWebComponents/v7/ui"
)

func TestLinkRelAndHreflangRenderValidTags(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(ui.Fragment(
		LinkRel("canonical", "https://example.com/docs"),
		Hreflang("en", "https://example.com/docs"),
	))
	if parseErr != nil {
		parseT.Fatalf("render link helpers: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, `rel="canonical"`) || !strings.Contains(parseMarkup, `href="https://example.com/docs"`) {
		parseT.Fatalf("expected canonical link markup, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, `rel="alternate"`) || !strings.Contains(parseMarkup, `hreflang="en"`) {
		parseT.Fatalf("expected hreflang markup, got %q", parseMarkup)
	}
}

func TestRenderJSONLDBranchesAndRenderToStringErrorPath(parseT *testing.T) {
	if parseMarkup, parseErr := RenderJSONLD(nil, "ignored"); parseErr != nil || parseMarkup != "" {
		parseT.Fatalf("expected nil JSON-LD value to render empty markup, got %q err=%v", parseMarkup, parseErr)
	}

	parseMarkup2, parseErr2 := RenderJSONLD(map[string]any{"@type": "Thing"}, "")
	if parseErr2 != nil {
		parseT.Fatalf("render jsonld without script id: %v", parseErr2)
	}
	if strings.Contains(parseMarkup2, `id="`) {
		parseT.Fatalf("expected no script id attribute when empty script ID is provided, got %q", parseMarkup2)
	}

	if _, parseErr3 := RenderJSONLD(map[string]any{"bad": func() {}}, "bad-jsonld"); parseErr3 == nil {
		parseT.Fatal("expected JSON-LD marshal error for unsupported value")
	}
	if _, parseErr4 := RenderToString(Document{
		JSONLD: []JSONLDBlock{{ID: "bad-jsonld", Value: map[string]any{"bad": func() {}}}},
	}); parseErr4 == nil {
		parseT.Fatal("expected RenderToString to return JSON-LD render error")
	}

	parseMarkup2, parseErr2 = RenderToString(Document{
		Metadata: router.Metadata{Title: "Docs"},
		Robots:   "index,follow",
	})
	if parseErr2 != nil {
		parseT.Fatalf("expected RenderToString fast path without JSON-LD to succeed, got %v", parseErr2)
	}
	if !strings.Contains(parseMarkup2, `content="index,follow"`) {
		parseT.Fatalf("expected RenderToString fast path markup, got %q", parseMarkup2)
	}

	parseMarkup2, parseErr2 = RenderToString(Document{
		Extras: []ui.Node{ui.CreateElement(123)},
	})
	if parseErr2 == nil {
		parseT.Fatal("expected RenderToString to propagate ui.RenderToString failure")
	}
	if parseMarkup2 != "" {
		parseT.Fatalf("expected empty markup on RenderToString render failure, got %q", parseMarkup2)
	}
}

func TestResolveNoLayersAndMergeExtrasReplace(parseT *testing.T) {
	if parseResolved := Resolve(); !reflect.DeepEqual(parseResolved, Document{}) {
		parseT.Fatalf("expected zero-value document when resolving no layers, got %+v", parseResolved)
	}

	parseBase := []ui.Node{MetaName("theme-color", "#000")}
	parseOverride := []ui.Node{MetaName("color-scheme", "dark")}
	parseMerged := mergeExtras(parseBase, parseOverride, MergeOptions{ReplaceExtras: true})
	if len(parseMerged) != 1 {
		parseT.Fatalf("expected extras replacement to keep override entries only, got %d", len(parseMerged))
	}
	parseMarkup, parseErr := ui.RenderToString(ui.Fragment(parseMerged...))
	if parseErr != nil {
		parseT.Fatalf("render merged extras: %v", parseErr)
	}
	if strings.Contains(parseMarkup, "theme-color") || !strings.Contains(parseMarkup, "color-scheme") {
		parseT.Fatalf("expected replaced extras markup, got %q", parseMarkup)
	}
}

func TestAlternateAndResourceHintOptionalAttributes(parseT *testing.T) {
	parseAltNode := alternateLinkNode(AlternateLink{
		Href:     "https://example.com/docs",
		HrefLang: "en",
		Media:    "(min-width: 768px)",
		Type:     "text/html",
		Title:    "Documentation",
	})
	parseHintNode := resourceHintNode(ResourceHint{
		Rel:         "preload",
		Href:        "https://cdn.example.com/app.css",
		As:          "style",
		CrossOrigin: "anonymous",
		Type:        "text/css",
		Media:       "screen",
	})
	parseMarkup, parseErr := ui.RenderToString(ui.Fragment(parseAltNode, parseHintNode))
	if parseErr != nil {
		parseT.Fatalf("render optional attribute nodes: %v", parseErr)
	}
	for _, parseSnippet := range []string{
		`hreflang="en"`,
		`media="(min-width: 768px)"`,
		`type="text/html"`,
		`title="Documentation"`,
		`rel="preload"`,
		`as="style"`,
		`crossorigin="anonymous"`,
		`type="text/css"`,
		`media="screen"`,
	} {
		if !strings.Contains(parseMarkup, parseSnippet) {
			parseT.Fatalf("expected %q in markup, got %q", parseSnippet, parseMarkup)
		}
	}

	if fragmentNonNil() != nil {
		parseT.Fatal("expected fragmentNonNil() to return nil for empty input")
	}
	parseSingle := html.Meta(html.Props{Raw: map[string]any{"name": "robots", "content": "index"}})
	if fragmentNonNil(parseSingle) == nil {
		parseT.Fatal("expected fragmentNonNil(single) to return non-nil")
	}
}

// TestMergeDedupesRepeatedHeadEntries pins that a leaf layer redeclaring an
// alternate, resource hint or JSON-LD id replaces the parent's entry in place
// instead of emitting a duplicate tag.
func TestMergeDedupesRepeatedHeadEntries(parseT *testing.T) {
	parseBase := Document{
		Alternates:    []AlternateLink{{HrefLang: "en", Href: "https://x.test/en"}, {HrefLang: "fr", Href: "https://x.test/fr"}},
		ResourceHints: []ResourceHint{{Rel: "preconnect", Href: "https://cdn.test"}},
		JSONLD:        []JSONLDBlock{{ID: "page", Value: map[string]any{"@type": "WebPage"}}, {Value: "anon-base"}},
	}
	parseOverride := Document{
		Alternates:    []AlternateLink{{HrefLang: "EN", Href: "https://x.test/en/guide"}, {HrefLang: "de", Href: "https://x.test/de"}},
		ResourceHints: []ResourceHint{{Rel: "preconnect", Href: "https://cdn.test"}},
		JSONLD:        []JSONLDBlock{{ID: "page", Value: map[string]any{"@type": "Article"}}, {Value: "anon-override"}},
	}
	parseMerged := Merge(parseBase, parseOverride)
	if len(parseMerged.Alternates) != 3 || parseMerged.Alternates[0].Href != "https://x.test/en/guide" || parseMerged.Alternates[2].HrefLang != "de" {
		parseT.Fatalf("alternates not deduped by hreflang: %#v", parseMerged.Alternates)
	}
	if len(parseMerged.ResourceHints) != 1 {
		parseT.Fatalf("identical resource hint duplicated: %#v", parseMerged.ResourceHints)
	}
	if len(parseMerged.JSONLD) != 3 || parseMerged.JSONLD[0].Value.(map[string]any)["@type"] != "Article" {
		parseT.Fatalf("JSON-LD with the same id not replaced (anonymous blocks must both survive): %#v", parseMerged.JSONLD)
	}
}
