package ui

import (
	"strings"
	"testing"
)

func TestMarshalSSRBootstrapEscapesScriptSensitiveCharacters(parseT *testing.T) {
	parsePayload := SSRBootstrap{
		Route: SSRRouteBootstrap{Path: "/docs"},
		Data:  map[string]any{"snippet": "</script><div>&"},
	}

	parseEncoded, parseErr := MarshalSSRBootstrap(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("unexpected marshal error: %v", parseErr)
	}

	parseText := string(parseEncoded)
	if strings.Contains(parseText, "</script>") {
		parseT.Fatalf("expected closing script tag to be escaped, got %q", parseText)
	}
	if !strings.Contains(parseText, `\u003c/script\u003e\u003cdiv\u003e\u0026`) {
		parseT.Fatalf("expected script-sensitive characters to be escaped, got %q", parseText)
	}
}

func TestMarshalSSRBootstrapPreservesEmptyNestedObjects(parseT *testing.T) {
	parseEncoded, parseErr := MarshalSSRBootstrap(SSRBootstrap{})
	if parseErr != nil {
		parseT.Fatalf("unexpected marshal error: %v", parseErr)
	}
	parseText := string(parseEncoded)
	for _, parseExpected := range []string{`"route":{}`, `"i18n":{}`} {
		if !strings.Contains(parseText, parseExpected) {
			parseT.Fatalf("expected bootstrap payload to preserve %s, got %q", parseExpected, parseText)
		}
	}
}

func TestRenderBootstrapScriptUsesDefaultID(parseT *testing.T) {
	parseScript, parseErr := RenderBootstrapScript(SSRBootstrap{Route: SSRRouteBootstrap{Path: "/home"}}, "")
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	if !strings.Contains(parseScript, `"version":1`) {
		parseT.Fatalf("expected versioned bootstrap payload in script tag, got %q", parseScript)
	}
	if !strings.Contains(parseScript, `id="__GWC_BOOTSTRAP__"`) {
		parseT.Fatalf("expected default bootstrap script id, got %q", parseScript)
	}
	if !strings.Contains(parseScript, `type="application/json"`) {
		parseT.Fatalf("expected application/json script type, got %q", parseScript)
	}
	if !strings.Contains(parseScript, `"path":"/home"`) {
		parseT.Fatalf("expected payload json in script tag, got %q", parseScript)
	}
}

func TestRenderBootstrapScriptEscapesCustomScriptID(parseT *testing.T) {
	parseScriptID := `boot"><img src=x onerror=alert(1)>`
	parseScript, parseErr := RenderBootstrapScript(SSRBootstrap{Route: SSRRouteBootstrap{Path: "/home"}}, parseScriptID)
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}
	if strings.Contains(parseScript, `<img src=x onerror=alert(1)>`) {
		parseT.Fatalf("expected custom script id to be escaped, got %q", parseScript)
	}
	if !strings.Contains(parseScript, `id="boot&#34;&gt;&lt;img src=x onerror=alert(1)&gt;"`) {
		parseT.Fatalf("expected escaped script id in output, got %q", parseScript)
	}
}

func TestUnmarshalSSRBootstrapInitializesMaps(parseT *testing.T) {
	parsePayload, parseErr := UnmarshalSSRBootstrap([]byte(`{"route":{"path":"/x"}}`))
	if parseErr != nil {
		parseT.Fatalf("unexpected unmarshal error: %v", parseErr)
	}

	if parsePayload.Route.Query == nil || parsePayload.Route.Params == nil || parsePayload.Atoms == nil || parsePayload.Data == nil || parsePayload.I18n.Messages == nil {
		parseT.Fatalf("expected zero-value maps to be initialized, got %+v", parsePayload)
	}
}

func TestMarshalSSRBootstrapIncludesI18nPayload(parseT *testing.T) {
	parsePayload := SSRBootstrap{
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
	parseEncoded, parseErr := MarshalSSRBootstrap(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("unexpected marshal error: %v", parseErr)
	}
	parseText := string(parseEncoded)
	if !strings.Contains(parseText, `"i18n":{"locale":"fr"`) {
		parseT.Fatalf("expected i18n payload in bootstrap json, got %q", parseText)
	}
	if !strings.Contains(parseText, `"marketing.headline":{"text":"Bonjour {name}"}`) {
		parseT.Fatalf("expected i18n messages in bootstrap json, got %q", parseText)
	}
}

func TestMarshalAndUnmarshalSSRBootstrapBinaryRoundTrip(parseT *testing.T) {
	parseInput := SSRBootstrap{
		Route: SSRRouteBootstrap{
			Path:   "/products/42",
			Query:  map[string][]string{"tab": {"specs"}},
			Params: map[string]string{"id": "42"},
		},
		Atoms:  map[string]any{"theme": "dark", "count": uint64(3)},
		Data:   map[string]any{"title": "Widget"},
		IDSeed: 7,
	}

	parseEncoded, parseErr := MarshalSSRBootstrapBinary(parseInput)
	if parseErr != nil {
		parseT.Fatalf("unexpected binary marshal error: %v", parseErr)
	}
	if len(parseEncoded) == 0 {
		parseT.Fatal("expected binary bootstrap payload")
	}

	parseDecoded, parseErr := UnmarshalSSRBootstrapBinary(parseEncoded)
	if parseErr != nil {
		parseT.Fatalf("unexpected binary unmarshal error: %v", parseErr)
	}

	if parseDecoded.Route.Path != parseInput.Route.Path {
		parseT.Fatalf("expected path %q, got %q", parseInput.Route.Path, parseDecoded.Route.Path)
	}
	if parseDecoded.Route.Params["id"] != "42" {
		parseT.Fatalf("expected params to survive binary round-trip, got %+v", parseDecoded.Route.Params)
	}
	if parseDecoded.Atoms["theme"] != "dark" {
		parseT.Fatalf("expected theme atom to survive binary round-trip, got %#v", parseDecoded.Atoms["theme"])
	}
	if parseDecoded.IDSeed != 7 {
		parseT.Fatalf("expected id seed to survive binary round-trip, got %d", parseDecoded.IDSeed)
	}
}

func TestRenderBootstrapReferenceScriptUsesDefaultID(parseT *testing.T) {
	parseScript, parseErr := RenderBootstrapReferenceScript(SSRBootstrapReference{URL: "/bootstrap.cbor", Format: SSRBootstrapFormatCBOR}, "")
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	if !strings.Contains(parseScript, `id="__GWC_BOOTSTRAP_REF__"`) {
		parseT.Fatalf("expected default bootstrap reference script id, got %q", parseScript)
	}
	if !strings.Contains(parseScript, `data-gwc-bootstrap-ref="true"`) {
		parseT.Fatalf("expected bootstrap reference marker, got %q", parseScript)
	}
	if !strings.Contains(parseScript, `{"version":1,"url":"/bootstrap.cbor","format":"cbor"}`) {
		parseT.Fatalf("expected raw JSON reference payload in script tag, got %q", parseScript)
	}
}

func TestRenderBootstrapReferenceScriptEscapesCustomScriptID(parseT *testing.T) {
	parseScriptID := `ref"><svg onload=alert(1)>`
	parseScript, parseErr := RenderBootstrapReferenceScript(SSRBootstrapReference{URL: "/bootstrap.cbor", Format: SSRBootstrapFormatCBOR}, parseScriptID)
	if parseErr != nil {
		parseT.Fatalf("unexpected reference render error: %v", parseErr)
	}
	if strings.Contains(parseScript, `<svg onload=alert(1)>`) {
		parseT.Fatalf("expected reference script id to be escaped, got %q", parseScript)
	}
	if !strings.Contains(parseScript, `id="ref&#34;&gt;&lt;svg onload=alert(1)&gt;"`) {
		parseT.Fatalf("expected escaped reference script id in output, got %q", parseScript)
	}
}

func TestUnmarshalSSRBootstrapReferenceDefaultsToJSON(parseT *testing.T) {
	parseRef, parseErr := UnmarshalSSRBootstrapReference([]byte(`{"url":"/bootstrap.json"}`))
	if parseErr != nil {
		parseT.Fatalf("unexpected reference unmarshal error: %v", parseErr)
	}
	if parseRef.Format != SSRBootstrapFormatJSON {
		parseT.Fatalf("expected default bootstrap reference format %q, got %q", SSRBootstrapFormatJSON, parseRef.Format)
	}
}
