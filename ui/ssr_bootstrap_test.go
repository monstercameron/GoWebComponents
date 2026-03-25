package ui

import (
	"strings"
	"testing"
)

func TestMarshalSSRBootstrapEscapesScriptSensitiveCharacters(t *testing.T) {
	payload := SSRBootstrap{
		Route: SSRRouteBootstrap{Path: "/docs"},
		Data:  map[string]interface{}{"snippet": "</script><div>&"},
	}

	encoded, err := MarshalSSRBootstrap(payload)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	text := string(encoded)
	if strings.Contains(text, "</script>") {
		t.Fatalf("expected closing script tag to be escaped, got %q", text)
	}
	if !strings.Contains(text, `\u003c/script\u003e\u003cdiv\u003e\u0026`) {
		t.Fatalf("expected script-sensitive characters to be escaped, got %q", text)
	}
}

func TestRenderBootstrapScriptUsesDefaultID(t *testing.T) {
	script, err := RenderBootstrapScript(SSRBootstrap{Route: SSRRouteBootstrap{Path: "/home"}}, "")
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if !strings.Contains(script, `"version":1`) {
		t.Fatalf("expected versioned bootstrap payload in script tag, got %q", script)
	}
	if !strings.Contains(script, `id="__GWC_BOOTSTRAP__"`) {
		t.Fatalf("expected default bootstrap script id, got %q", script)
	}
	if !strings.Contains(script, `type="application/json"`) {
		t.Fatalf("expected application/json script type, got %q", script)
	}
	if !strings.Contains(script, `"path":"/home"`) {
		t.Fatalf("expected payload json in script tag, got %q", script)
	}
}

func TestRenderBootstrapScriptEscapesCustomScriptID(t *testing.T) {
	scriptID := `boot"><img src=x onerror=alert(1)>`
	script, err := RenderBootstrapScript(SSRBootstrap{Route: SSRRouteBootstrap{Path: "/home"}}, scriptID)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	if strings.Contains(script, `<img src=x onerror=alert(1)>`) {
		t.Fatalf("expected custom script id to be escaped, got %q", script)
	}
	if !strings.Contains(script, `id="boot&#34;&gt;&lt;img src=x onerror=alert(1)&gt;"`) {
		t.Fatalf("expected escaped script id in output, got %q", script)
	}
}

func TestUnmarshalSSRBootstrapInitializesMaps(t *testing.T) {
	payload, err := UnmarshalSSRBootstrap([]byte(`{"route":{"path":"/x"}}`))
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if payload.Route.Query == nil || payload.Route.Params == nil || payload.Atoms == nil || payload.Data == nil || payload.I18n.Messages == nil {
		t.Fatalf("expected zero-value maps to be initialized, got %+v", payload)
	}
}

func TestMarshalSSRBootstrapIncludesI18nPayload(t *testing.T) {
	payload := SSRBootstrap{
		Route: SSRRouteBootstrap{Path: "/docs"},
		I18n: SSRI18nBootstrap{
			Locale:         "fr",
			FallbackLocale: "en",
			Direction:      "ltr",
			Messages: map[string]map[string]SSRI18nMessage{
				"fr": {
					"marketing.headline": {Text: "Bonjour {name}"},
				},
			},
		},
	}
	encoded, err := MarshalSSRBootstrap(payload)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	text := string(encoded)
	if !strings.Contains(text, `"i18n":{"locale":"fr"`) {
		t.Fatalf("expected i18n payload in bootstrap json, got %q", text)
	}
	if !strings.Contains(text, `"marketing.headline":{"text":"Bonjour {name}"}`) {
		t.Fatalf("expected i18n messages in bootstrap json, got %q", text)
	}
}

func TestMarshalAndUnmarshalSSRBootstrapBinaryRoundTrip(t *testing.T) {
	input := SSRBootstrap{
		Route: SSRRouteBootstrap{
			Path:   "/products/42",
			Query:  map[string][]string{"tab": {"specs"}},
			Params: map[string]string{"id": "42"},
		},
		Atoms:  map[string]interface{}{"theme": "dark", "count": uint64(3)},
		Data:   map[string]interface{}{"title": "Widget"},
		IDSeed: 7,
	}

	encoded, err := MarshalSSRBootstrapBinary(input)
	if err != nil {
		t.Fatalf("unexpected binary marshal error: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("expected binary bootstrap payload")
	}

	decoded, err := UnmarshalSSRBootstrapBinary(encoded)
	if err != nil {
		t.Fatalf("unexpected binary unmarshal error: %v", err)
	}

	if decoded.Route.Path != input.Route.Path {
		t.Fatalf("expected path %q, got %q", input.Route.Path, decoded.Route.Path)
	}
	if decoded.Route.Params["id"] != "42" {
		t.Fatalf("expected params to survive binary round-trip, got %+v", decoded.Route.Params)
	}
	if decoded.Atoms["theme"] != "dark" {
		t.Fatalf("expected theme atom to survive binary round-trip, got %#v", decoded.Atoms["theme"])
	}
	if decoded.IDSeed != 7 {
		t.Fatalf("expected id seed to survive binary round-trip, got %d", decoded.IDSeed)
	}
}

func TestRenderBootstrapReferenceScriptUsesDefaultID(t *testing.T) {
	script, err := RenderBootstrapReferenceScript(SSRBootstrapReference{URL: "/bootstrap.cbor", Format: SSRBootstrapFormatCBOR}, "")
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	if !strings.Contains(script, `id="__GWC_BOOTSTRAP_REF__"`) {
		t.Fatalf("expected default bootstrap reference script id, got %q", script)
	}
	if !strings.Contains(script, `data-gwc-bootstrap-ref="true"`) {
		t.Fatalf("expected bootstrap reference marker, got %q", script)
	}
	if !strings.Contains(script, `{"version":1,"url":"/bootstrap.cbor","format":"cbor"}`) {
		t.Fatalf("expected raw JSON reference payload in script tag, got %q", script)
	}
}

func TestRenderBootstrapReferenceScriptEscapesCustomScriptID(t *testing.T) {
	scriptID := `ref"><svg onload=alert(1)>`
	script, err := RenderBootstrapReferenceScript(SSRBootstrapReference{URL: "/bootstrap.cbor", Format: SSRBootstrapFormatCBOR}, scriptID)
	if err != nil {
		t.Fatalf("unexpected reference render error: %v", err)
	}
	if strings.Contains(script, `<svg onload=alert(1)>`) {
		t.Fatalf("expected reference script id to be escaped, got %q", script)
	}
	if !strings.Contains(script, `id="ref&#34;&gt;&lt;svg onload=alert(1)&gt;"`) {
		t.Fatalf("expected escaped reference script id in output, got %q", script)
	}
}

func TestUnmarshalSSRBootstrapReferenceDefaultsToJSON(t *testing.T) {
	ref, err := UnmarshalSSRBootstrapReference([]byte(`{"url":"/bootstrap.json"}`))
	if err != nil {
		t.Fatalf("unexpected reference unmarshal error: %v", err)
	}
	if ref.Format != SSRBootstrapFormatJSON {
		t.Fatalf("expected default bootstrap reference format %q, got %q", SSRBootstrapFormatJSON, ref.Format)
	}
}
