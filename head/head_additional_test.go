package head

import (
	"reflect"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestLinkRelAndHreflangRenderValidTags(t *testing.T) {
	markup, err := ui.RenderToString(ui.Fragment(
		LinkRel("canonical", "https://example.com/docs"),
		Hreflang("en", "https://example.com/docs"),
	))
	if err != nil {
		t.Fatalf("render link helpers: %v", err)
	}
	if !strings.Contains(markup, `rel="canonical"`) || !strings.Contains(markup, `href="https://example.com/docs"`) {
		t.Fatalf("expected canonical link markup, got %q", markup)
	}
	if !strings.Contains(markup, `rel="alternate"`) || !strings.Contains(markup, `hreflang="en"`) {
		t.Fatalf("expected hreflang markup, got %q", markup)
	}
}

func TestRenderJSONLDBranchesAndRenderToStringErrorPath(t *testing.T) {
	if markup, err := RenderJSONLD(nil, "ignored"); err != nil || markup != "" {
		t.Fatalf("expected nil JSON-LD value to render empty markup, got %q err=%v", markup, err)
	}

	markup, err := RenderJSONLD(map[string]any{"@type": "Thing"}, "")
	if err != nil {
		t.Fatalf("render jsonld without script id: %v", err)
	}
	if strings.Contains(markup, `id="`) {
		t.Fatalf("expected no script id attribute when empty script ID is provided, got %q", markup)
	}

	if _, err := RenderJSONLD(map[string]any{"bad": func() {}}, "bad-jsonld"); err == nil {
		t.Fatal("expected JSON-LD marshal error for unsupported value")
	}
	if _, err := RenderToString(Document{
		JSONLD: []JSONLDBlock{{ID: "bad-jsonld", Value: map[string]any{"bad": func() {}}}},
	}); err == nil {
		t.Fatal("expected RenderToString to return JSON-LD render error")
	}

	markup, err = RenderToString(Document{
		Metadata: router.Metadata{Title: "Docs"},
		Robots:   "index,follow",
	})
	if err != nil {
		t.Fatalf("expected RenderToString fast path without JSON-LD to succeed, got %v", err)
	}
	if !strings.Contains(markup, `content="index,follow"`) {
		t.Fatalf("expected RenderToString fast path markup, got %q", markup)
	}

	markup, err = RenderToString(Document{
		Extras: []ui.Node{ui.CreateElement(123)},
	})
	if err == nil {
		t.Fatal("expected RenderToString to propagate ui.RenderToString failure")
	}
	if markup != "" {
		t.Fatalf("expected empty markup on RenderToString render failure, got %q", markup)
	}
}

func TestResolveNoLayersAndMergeExtrasReplace(t *testing.T) {
	if resolved := Resolve(); !reflect.DeepEqual(resolved, Document{}) {
		t.Fatalf("expected zero-value document when resolving no layers, got %+v", resolved)
	}

	base := []ui.Node{MetaName("theme-color", "#000")}
	override := []ui.Node{MetaName("color-scheme", "dark")}
	merged := mergeExtras(base, override, MergeOptions{ReplaceExtras: true})
	if len(merged) != 1 {
		t.Fatalf("expected extras replacement to keep override entries only, got %d", len(merged))
	}
	markup, err := ui.RenderToString(ui.Fragment(merged...))
	if err != nil {
		t.Fatalf("render merged extras: %v", err)
	}
	if strings.Contains(markup, "theme-color") || !strings.Contains(markup, "color-scheme") {
		t.Fatalf("expected replaced extras markup, got %q", markup)
	}
}

func TestAlternateAndResourceHintOptionalAttributes(t *testing.T) {
	altNode := alternateLinkNode(AlternateLink{
		Href:     "https://example.com/docs",
		HrefLang: "en",
		Media:    "(min-width: 768px)",
		Type:     "text/html",
		Title:    "Documentation",
	})
	hintNode := resourceHintNode(ResourceHint{
		Rel:         "preload",
		Href:        "https://cdn.example.com/app.css",
		As:          "style",
		CrossOrigin: "anonymous",
		Type:        "text/css",
		Media:       "screen",
	})
	markup, err := ui.RenderToString(ui.Fragment(altNode, hintNode))
	if err != nil {
		t.Fatalf("render optional attribute nodes: %v", err)
	}
	for _, snippet := range []string{
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
		if !strings.Contains(markup, snippet) {
			t.Fatalf("expected %q in markup, got %q", snippet, markup)
		}
	}

	if fragmentNonNil() != nil {
		t.Fatal("expected fragmentNonNil() to return nil for empty input")
	}
	single := html.Meta(html.Props{Raw: map[string]any{"name": "robots", "content": "index"}})
	if fragmentNonNil(single) == nil {
		t.Fatal("expected fragmentNonNil(single) to return non-nil")
	}
}
