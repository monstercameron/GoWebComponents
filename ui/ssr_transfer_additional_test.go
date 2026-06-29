package ui

import (
	"strings"
	"testing"
	"time"
)

type textOnlyID string

func (parseId textOnlyID) MarshalText() ([]byte, error) {
	return []byte(string(parseId)), nil
}

func TestSSRTransferHelperErrorBranches(parseT *testing.T) {
	if parseEscaped := escapeJSONForInlineScript("<>&"); parseEscaped != `\u003c\u003e\u0026` {
		parseT.Fatalf("escapeJSONForInlineScript() = %q, want escaped angle brackets and ampersand", parseEscaped)
	}
	if _, parseErr := normalizeSSRPayloadEnvelope(SSRPayloadEnvelope{Version: -1}); parseErr == nil {
		parseT.Fatal("normalizeSSRPayloadEnvelope() should reject negative versions")
	}
	if parseErr2 := validateSSRPayloadEnum("weird", SSRPayloadScopeApp, SSRPayloadReuseTrustOnFirstResume, SSRPayloadEncodingJSON); parseErr2 == nil {
		parseT.Fatal("validateSSRPayloadEnum() should reject invalid payload kind")
	}
	if parseGot := detectSSRPayloadEncoding([]byte{1}, SSRPayloadOptions{}); parseGot != SSRPayloadEncodingBinary {
		parseT.Fatalf("detectSSRPayloadEncoding([]byte) = %q, want binary", parseGot)
	}
	if parseGot2 := detectSSRPayloadEncoding(time.Unix(1, 0), SSRPayloadOptions{}); parseGot2 != SSRPayloadEncodingTimeRFC3339 {
		parseT.Fatalf("detectSSRPayloadEncoding(time) = %q, want time-rfc3339", parseGot2)
	}
	if parseGot3 := detectSSRPayloadEncoding(textOnlyID("sku-1"), SSRPayloadOptions{}); parseGot3 != SSRPayloadEncodingText {
		parseT.Fatalf("detectSSRPayloadEncoding(text marshaler) = %q, want text", parseGot3)
	}

	if _, parseErr3 := encodeSSRTextPayload(123); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "does not support text encoding") {
		parseT.Fatalf("encodeSSRTextPayload() error = %v, want text encoding error", parseErr3)
	}
	if _, parseErr4 := encodeSSRBinaryPayload("nope"); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "does not support binary encoding") {
		parseT.Fatalf("encodeSSRBinaryPayload() error = %v, want binary encoding error", parseErr4)
	}
	if _, parseErr5 := encodeSSRTimePayload("nope"); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "does not support time encoding") {
		parseT.Fatalf("encodeSSRTimePayload() error = %v, want time encoding error", parseErr5)
	}
	if _, parseErr6 := assignDecodedSSRValue[time.Time]("wrong-type"); parseErr6 == nil {
		parseT.Fatal("assignDecodedSSRValue() should reject incompatible decoded values")
	}

	if parseErr7 := RegisterBootstrapPayload(nil, "key", "value"); parseErr7 == nil {
		parseT.Fatal("RegisterBootstrapPayload(nil) should fail")
	}
	if parseErr8 := RegisterBootstrapPayload(&SSRBootstrap{}, "", "value"); parseErr8 == nil {
		parseT.Fatal("RegisterBootstrapPayload(empty key) should fail")
	}
	if _, _, parseErr9 := ReadBootstrapPayload[string](SSRBootstrap{}, ""); parseErr9 == nil {
		parseT.Fatal("ReadBootstrapPayload(empty key) should fail")
	}

	parseBootstrap := SSRBootstrap{}
	if parseErr10 := RegisterRouteBootstrapData(&parseBootstrap, "", "/docs", "payload"); parseErr10 != nil {
		parseT.Fatalf("RegisterRouteBootstrapData() error = %v", parseErr10)
	}
	parseRouteValue, parseOk, parseErr11 := ReadRouteBootstrapData[string](parseBootstrap, "", "/docs")
	if parseErr11 != nil || !parseOk || parseRouteValue.Key != routeBootstrapPayloadKey("", "/docs") || parseRouteValue.Value != "payload" {
		parseT.Fatalf("ReadRouteBootstrapData() = %+v, %t, %v; want default route payload key and value", parseRouteValue, parseOk, parseErr11)
	}

	if _, parseErr12 := normalizeSSRStateUpdate(SSRStateUpdate{Version: -1}); parseErr12 == nil {
		parseT.Fatal("normalizeSSRStateUpdate() should reject negative versions")
	}
	parseNormalized, parseErr11 := normalizeSSRStateUpdate(SSRStateUpdate{Deletes: []string{" a ", "", "a"}})
	if parseErr11 != nil {
		parseT.Fatalf("normalizeSSRStateUpdate() error = %v", parseErr11)
	}
	if len(parseNormalized.Deletes) != 1 || parseNormalized.Deletes[0] != "a" {
		parseT.Fatalf("normalizeSSRStateUpdate() deletes = %#v, want [a]", parseNormalized.Deletes)
	}
	if parseErr13 := RegisterStateUpdatePayload(nil, "key", "value"); parseErr13 == nil {
		parseT.Fatal("RegisterStateUpdatePayload(nil) should fail")
	}
	if parseErr14 := RegisterStateUpdatePayload(&SSRStateUpdate{}, "", "value"); parseErr14 == nil {
		parseT.Fatal("RegisterStateUpdatePayload(empty key) should fail")
	}
	if _, parseErr15 := UnmarshalSSRStateUpdateText(nil); parseErr15 != nil {
		parseT.Fatalf("UnmarshalSSRStateUpdateText(nil) error = %v", parseErr15)
	}
	if _, parseErr16 := UnmarshalSSRStateUpdateBinary(nil); parseErr16 != nil {
		parseT.Fatalf("UnmarshalSSRStateUpdateBinary(nil) error = %v", parseErr16)
	}
	if parseErr17 := ApplySSRStateUpdate(nil, SSRStateUpdate{}); parseErr17 == nil {
		parseT.Fatal("ApplySSRStateUpdate(nil) should fail")
	}

	parseBudget := normalizeSSRBootstrapBudget(SSRBootstrapBudget{InlineWarnBytes: 1, InlineErrorBytes: 0})
	if parseBudget.InlineErrorBytes < parseBudget.InlineWarnBytes || parseBudget.BinaryErrorBytes < parseBudget.BinaryWarnBytes || parseBudget.SidecarErrorBytes < parseBudget.SidecarWarnBytes {
		parseT.Fatalf("normalizeSSRBootstrapBudget() = %+v, want normalized thresholds", parseBudget)
	}
}

func TestScopedBootstrapReadHelpers(parseT *testing.T) {
	parseBootstrap := SSRBootstrap{}

	if parseErr := RegisterFormBootstrapDefaults(&parseBootstrap, " checkout ", map[string]string{"email": "cam@example.com"}); parseErr != nil {
		parseT.Fatalf("RegisterFormBootstrapDefaults() error = %v", parseErr)
	}
	if parseErr2 := RegisterCacheBootstrapSeed(&parseBootstrap, " products:list ", []string{"sku-1", "sku-2"}); parseErr2 != nil {
		parseT.Fatalf("RegisterCacheBootstrapSeed() error = %v", parseErr2)
	}
	if parseErr3 := RegisterSessionBootstrapHint(&parseBootstrap, " viewer ", map[string]string{"role": "operator"}); parseErr3 != nil {
		parseT.Fatalf("RegisterSessionBootstrapHint() error = %v", parseErr3)
	}

	parseFormDefaults, parseOk, parseErr4 := ReadFormBootstrapDefaults[map[string]string](parseBootstrap, " checkout ")
	if parseErr4 != nil || !parseOk {
		parseT.Fatalf("ReadFormBootstrapDefaults() = %+v, %t, %v; want payload", parseFormDefaults, parseOk, parseErr4)
	}
	if parseFormDefaults.Kind != SSRPayloadKindFormDefaults || parseFormDefaults.Scope != SSRPayloadScopeSubtree || parseFormDefaults.Target != "checkout" || parseFormDefaults.Value["email"] != "cam@example.com" {
		parseT.Fatalf("unexpected form defaults payload: %+v", parseFormDefaults)
	}

	cacheSeed, parseOk, parseErr4 := ReadCacheBootstrapSeed[[]string](parseBootstrap, " products:list ")
	if parseErr4 != nil || !parseOk {
		parseT.Fatalf("ReadCacheBootstrapSeed() = %+v, %t, %v; want payload", cacheSeed, parseOk, parseErr4)
	}
	if cacheSeed.Kind != SSRPayloadKindCacheSeed || cacheSeed.Scope != SSRPayloadScopeRoute || cacheSeed.Target != "products:list" || len(cacheSeed.Value) != 2 {
		parseT.Fatalf("unexpected cache seed payload: %+v", cacheSeed)
	}

	parseSessionHint, parseOk, parseErr4 := ReadSessionBootstrapHint[map[string]string](parseBootstrap, " viewer ")
	if parseErr4 != nil || !parseOk {
		parseT.Fatalf("ReadSessionBootstrapHint() = %+v, %t, %v; want payload", parseSessionHint, parseOk, parseErr4)
	}
	if parseSessionHint.Kind != SSRPayloadKindSessionHint || parseSessionHint.Scope != SSRPayloadScopeApp || parseSessionHint.Target != "viewer" || parseSessionHint.ReusePolicy != SSRPayloadReuseClientOwned || parseSessionHint.Value["role"] != "operator" {
		parseT.Fatalf("unexpected session hint payload: %+v", parseSessionHint)
	}

	if _, parseOk2, parseErr5 := ReadFormBootstrapDefaults[map[string]string](parseBootstrap, "missing"); parseErr5 != nil || parseOk2 {
		parseT.Fatalf("expected missing form defaults to return ok=false, got ok=%t err=%v", parseOk2, parseErr5)
	}
	if _, parseOk3, parseErr6 := ReadCacheBootstrapSeed[[]string](parseBootstrap, "missing"); parseErr6 != nil || parseOk3 {
		parseT.Fatalf("expected missing cache seed to return ok=false, got ok=%t err=%v", parseOk3, parseErr6)
	}
	if _, parseOk4, parseErr7 := ReadSessionBootstrapHint[map[string]string](parseBootstrap, "missing"); parseErr7 != nil || parseOk4 {
		parseT.Fatalf("expected missing session hint to return ok=false, got ok=%t err=%v", parseOk4, parseErr7)
	}
}

// TestEscapeJSONForInlineScriptLineSeparators is a regression test for finding #56.
// Raw U+2028 (LINE SEPARATOR) and U+2029 (PARAGRAPH SEPARATOR) embedded in JSON
// can break HTML inline script parsing. escapeJSONForInlineScript must replace
// both codepoints with their 6-char ASCII equivalents.
func TestEscapeJSONForInlineScriptLineSeparators(parseT *testing.T) {
	parseLineSep := string([]rune{0x2028})
	parseParaSep := string([]rune{0x2029})
	parseInput := "hello" + parseLineSep + "world" + parseParaSep + "end"
	parseOutput := escapeJSONForInlineScript(parseInput)

	// Must not contain the raw codepoints.
	if strings.ContainsRune(parseOutput, 0x2028) {
		parseT.Fatalf("escapeJSONForInlineScript() output contains raw U+2028: %q", parseOutput)
	}
	if strings.ContainsRune(parseOutput, 0x2029) {
		parseT.Fatalf("escapeJSONForInlineScript() output contains raw U+2029: %q", parseOutput)
	}

	// Must contain the 6-char ASCII escape sequences.
	if !strings.Contains(parseOutput, "\\u2028") {
		parseT.Fatalf("escapeJSONForInlineScript() output missing \u2028 escape: %q", parseOutput)
	}
	if !strings.Contains(parseOutput, "\\u2029") {
		parseT.Fatalf("escapeJSONForInlineScript() output missing \u2029 escape: %q", parseOutput)
	}
}

// TestNormalizeSSRStateUpdateKeyTrimmingAllApplied is a regression test for finding #59.
// When upsert keys require whitespace trimming, all keys must be normalized
// correctly even when the old-key and new-key sets overlap.
func TestNormalizeSSRStateUpdateKeyTrimmingAllApplied(parseT *testing.T) {
	parseUpdate := SSRStateUpdate{
		Upserts: map[string]SSRPayloadEnvelope{
			" alpha ": {Encoding: SSRPayloadEncodingText, Text: "a"},
			" beta ":  {Encoding: SSRPayloadEncodingText, Text: "b"},
			"gamma":   {Encoding: SSRPayloadEncodingText, Text: "c"},
		},
	}
	parseResult, parseErr := normalizeSSRStateUpdate(parseUpdate)
	if parseErr != nil {
		parseT.Fatalf("normalizeSSRStateUpdate() error = %v", parseErr)
	}

	// All three keys must be present under their trimmed form.
	for _, parseKey := range []string{"alpha", "beta", "gamma"} {
		if _, parseOk := parseResult.Upserts[parseKey]; !parseOk {
			parseT.Errorf("normalizeSSRStateUpdate() missing trimmed key %q; upserts = %v", parseKey, parseResult.Upserts)
		}
	}

	// Padded originals must have been removed.
	for _, parseOldKey := range []string{" alpha ", " beta "} {
		if _, parseOk := parseResult.Upserts[parseOldKey]; parseOk {
			parseT.Errorf("normalizeSSRStateUpdate() untrimmed key %q still present; upserts = %v", parseOldKey, parseResult.Upserts)
		}
	}
}
