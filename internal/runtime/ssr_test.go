package runtime

import (
	"strings"
	"testing"
)

func TestRenderToStringHostTree(parseT *testing.T) {
	parseElement := CreateElement("div", map[string]any{
		"id":        "root",
		"className": "panel",
		"style": map[string]string{
			"background": "black",
			"color":      "white",
		},
	},
		CreateElement("label", map[string]any{"htmlFor": "email"}, "Email"),
		CreateElement("input", map[string]any{"id": "email", "disabled": true, "checked": false}),
	)

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	parseWant := `<div class="panel" id="root" style="background:black;color:white"><label for="email">Email</label><input disabled id="email"></div>`
	if parseHtml != parseWant {
		parseT.Fatalf("unexpected html\nwant: %s\ngot:  %s", parseWant, parseHtml)
	}
}

func TestRenderToStringEscapesTextAndAttributes(parseT *testing.T) {
	parseElement := CreateElement("div", map[string]any{
		"title": `<unsafe "quote">`,
	}, `hello <world> & friends`)

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	parseWant := `<div title="&lt;unsafe &#34;quote&#34;&gt;">hello &lt;world&gt; &amp; friends</div>`
	if parseHtml != parseWant {
		parseT.Fatalf("unexpected escaped html\nwant: %s\ngot:  %s", parseWant, parseHtml)
	}
}

func TestRenderToStringFragment(parseT *testing.T) {
	parseElement := CreateElement("FRAGMENT", nil,
		CreateElement("span", nil, "one"),
		CreateElement("span", nil, "two"),
	)

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	if parseHtml != `<span>one</span><span>two</span>` {
		parseT.Fatalf("unexpected fragment html: %s", parseHtml)
	}
}

func TestRenderToStringSkipsChildrenKeyAndHandlers(parseT *testing.T) {
	parseElement := &Element{
		Type: "button",
		Props: map[string]any{
			"id":       "save",
			"key":      "button-1",
			"children": []any{"bad"},
			"onclick":  func() {},
		},
		Children: []any{"Save"},
	}

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	if parseHtml != `<button id="save">Save</button>` {
		parseT.Fatalf("unexpected button html: %s", parseHtml)
	}
}

// TestSerializeSSRAttrRejectsMaliciousName is a regression test for finding #57.
// A crafted attribute name like `x onmouseover=alert(1) y` must be silently
// dropped rather than written unescaped into the HTML output.
func TestSerializeSSRAttrRejectsMaliciousName(parseT *testing.T) {
	parseMalicious := `x onmouseover=alert(1) y`
	parseOut, parseOk := serializeSSRAttr(parseMalicious, "value")
	if parseOk {
		parseT.Fatalf("serializeSSRAttr() accepted malicious name %q, got %q", parseMalicious, parseOut)
	}
	if parseOut != "" {
		parseT.Fatalf("serializeSSRAttr() returned non-empty output for malicious name: %q", parseOut)
	}

	// Ensure the attr is also absent from rendered HTML.
	parseElement := CreateElement("div", map[string]any{
		parseMalicious: "injected",
		"id":           "safe",
	})
	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}
	if strings.Contains(parseHtml, "onmouseover") {
		parseT.Fatalf("rendered HTML contains injected attribute name: %s", parseHtml)
	}
	if !strings.Contains(parseHtml, `id="safe"`) {
		parseT.Fatalf("rendered HTML missing safe attribute: %s", parseHtml)
	}
}

// TestSerializeSSRAttrMapDeterministicOrder is a regression test for finding #58.
// A map[string]string attribute value that is not "style" must serialize with
// sorted keys so the output is deterministic across runs.
func TestSerializeSSRAttrMapDeterministicOrder(parseT *testing.T) {
	parseAttr1, parseOk1 := serializeSSRAttr("data-info", map[string]string{"z": "last", "a": "first", "m": "mid"})
	parseAttr2, parseOk2 := serializeSSRAttr("data-info", map[string]string{"m": "mid", "z": "last", "a": "first"})
	if !parseOk1 || !parseOk2 {
		parseT.Fatal("serializeSSRAttr() unexpectedly rejected valid map attribute")
	}
	if parseAttr1 != parseAttr2 {
		parseT.Fatalf("serializeSSRAttr() produced different outputs for same map with different insertion order:\n  %s\n  %s", parseAttr1, parseAttr2)
	}
}

func TestRenderToStringFunctionComponent(parseT *testing.T) {
	parseComponent := func(parseProps Attrs) *Element {
		return CreateElement("section", map[string]any{"id": parseProps["id"]}, parseProps["label"])
	}
	parseElement := CreateElement(parseComponent, map[string]any{"id": "hero", "label": "Welcome"})

	parseHtml, parseErr := RenderToString(parseElement)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	if parseHtml != `<section id="hero">Welcome</section>` {
		parseT.Fatalf("unexpected component html: %s", parseHtml)
	}
}

func TestRenderToStringAsyncBoundaryDoesNotLeakPartialMarkupOnSuspension(parseT *testing.T) {
	parseDone := make(chan struct{})
	parseSuspendingChild := CreateElement(func() *Element {
		SuspendUntil(parseDone, "late child")
		return CreateElement("em", nil, "late")
	}, nil)
	parseContent := CreateElement("div", nil, "before", parseSuspendingChild, "after")
	parseFallback := CreateElement("span", nil, "loading")
	parseRoot := CreateElement(AsyncBoundaryNodeType, map[string]any{
		"content":  parseContent,
		"fallback": parseFallback,
	})

	parseHTML, parseErr := RenderToString(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	if parseHTML != `<span>loading</span>` {
		parseT.Fatalf("expected only fallback markup after suspension, got %q", parseHTML)
	}
	if strings.Contains(parseHTML, "before") || strings.Contains(parseHTML, "after") {
		parseT.Fatalf("suspended boundary leaked partial content: %s", parseHTML)
	}
}

func TestRenderToStringErrorBoundaryDoesNotLeakPartialMarkupOnPanic(parseT *testing.T) {
	parsePanickingChild := CreateElement(func() *Element {
		panic("child render failure")
	}, nil)
	parseContent := CreateElement("div", nil, "before", parsePanickingChild, "after")
	parseFallback := CreateElement("span", nil, "recovered")
	parseRoot := CreateElement(NewErrorBoundaryType(), map[string]any{
		"fallback": parseFallback,
	}, parseContent)

	parseHTML, parseErr := RenderToString(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	if parseHTML != `<span>recovered</span>` {
		parseT.Fatalf("expected only fallback markup after child panic, got %q", parseHTML)
	}
	if strings.Contains(parseHTML, "before") || strings.Contains(parseHTML, "<div") {
		parseT.Fatalf("error boundary leaked partial content: %s", parseHTML)
	}
}
