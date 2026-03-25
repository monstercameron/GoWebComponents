package ui

import (
	"strings"
	"testing"
	"time"
)

type textOnlyID string

func (id textOnlyID) MarshalText() ([]byte, error) {
	return []byte(string(id)), nil
}

func TestSSRTransferHelperErrorBranches(t *testing.T) {
	if escaped := escapeJSONForInlineScript("<>&"); escaped != `\u003c\u003e\u0026` {
		t.Fatalf("escapeJSONForInlineScript() = %q, want escaped angle brackets and ampersand", escaped)
	}
	if _, err := normalizeSSRPayloadEnvelope(SSRPayloadEnvelope{Version: -1}); err == nil {
		t.Fatal("normalizeSSRPayloadEnvelope() should reject negative versions")
	}
	if err := validateSSRPayloadEnum("weird", SSRPayloadScopeApp, SSRPayloadReuseTrustOnFirstResume, SSRPayloadEncodingJSON); err == nil {
		t.Fatal("validateSSRPayloadEnum() should reject invalid payload kind")
	}
	if got := detectSSRPayloadEncoding([]byte{1}, SSRPayloadOptions{}); got != SSRPayloadEncodingBinary {
		t.Fatalf("detectSSRPayloadEncoding([]byte) = %q, want binary", got)
	}
	if got := detectSSRPayloadEncoding(time.Unix(1, 0), SSRPayloadOptions{}); got != SSRPayloadEncodingTimeRFC3339 {
		t.Fatalf("detectSSRPayloadEncoding(time) = %q, want time-rfc3339", got)
	}
	if got := detectSSRPayloadEncoding(textOnlyID("sku-1"), SSRPayloadOptions{}); got != SSRPayloadEncodingText {
		t.Fatalf("detectSSRPayloadEncoding(text marshaler) = %q, want text", got)
	}

	if _, err := encodeSSRTextPayload(123); err == nil || !strings.Contains(err.Error(), "does not support text encoding") {
		t.Fatalf("encodeSSRTextPayload() error = %v, want text encoding error", err)
	}
	if _, err := encodeSSRBinaryPayload("nope"); err == nil || !strings.Contains(err.Error(), "does not support binary encoding") {
		t.Fatalf("encodeSSRBinaryPayload() error = %v, want binary encoding error", err)
	}
	if _, err := encodeSSRTimePayload("nope"); err == nil || !strings.Contains(err.Error(), "does not support time encoding") {
		t.Fatalf("encodeSSRTimePayload() error = %v, want time encoding error", err)
	}
	if _, err := assignDecodedSSRValue[time.Time]("wrong-type"); err == nil {
		t.Fatal("assignDecodedSSRValue() should reject incompatible decoded values")
	}

	if err := RegisterBootstrapPayload(nil, "key", "value"); err == nil {
		t.Fatal("RegisterBootstrapPayload(nil) should fail")
	}
	if err := RegisterBootstrapPayload(&SSRBootstrap{}, "", "value"); err == nil {
		t.Fatal("RegisterBootstrapPayload(empty key) should fail")
	}
	if _, _, err := ReadBootstrapPayload[string](SSRBootstrap{}, ""); err == nil {
		t.Fatal("ReadBootstrapPayload(empty key) should fail")
	}

	bootstrap := SSRBootstrap{}
	if err := RegisterRouteBootstrapData(&bootstrap, "", "/docs", "payload"); err != nil {
		t.Fatalf("RegisterRouteBootstrapData() error = %v", err)
	}
	routeValue, ok, err := ReadRouteBootstrapData[string](bootstrap, "", "/docs")
	if err != nil || !ok || routeValue.Key != routeBootstrapPayloadKey("", "/docs") || routeValue.Value != "payload" {
		t.Fatalf("ReadRouteBootstrapData() = %+v, %t, %v; want default route payload key and value", routeValue, ok, err)
	}

	if _, err := normalizeSSRStateUpdate(SSRStateUpdate{Version: -1}); err == nil {
		t.Fatal("normalizeSSRStateUpdate() should reject negative versions")
	}
	normalized, err := normalizeSSRStateUpdate(SSRStateUpdate{Deletes: []string{" a ", "", "a"}})
	if err != nil {
		t.Fatalf("normalizeSSRStateUpdate() error = %v", err)
	}
	if len(normalized.Deletes) != 1 || normalized.Deletes[0] != "a" {
		t.Fatalf("normalizeSSRStateUpdate() deletes = %#v, want [a]", normalized.Deletes)
	}
	if err := AddStateUpdatePayload(nil, "key", "value"); err == nil {
		t.Fatal("AddStateUpdatePayload(nil) should fail")
	}
	if err := AddStateUpdatePayload(&SSRStateUpdate{}, "", "value"); err == nil {
		t.Fatal("AddStateUpdatePayload(empty key) should fail")
	}
	if _, err := UnmarshalSSRStateUpdateText(nil); err != nil {
		t.Fatalf("UnmarshalSSRStateUpdateText(nil) error = %v", err)
	}
	if _, err := UnmarshalSSRStateUpdateBinary(nil); err != nil {
		t.Fatalf("UnmarshalSSRStateUpdateBinary(nil) error = %v", err)
	}
	if err := ApplySSRStateUpdate(nil, SSRStateUpdate{}); err == nil {
		t.Fatal("ApplySSRStateUpdate(nil) should fail")
	}

	budget := normalizeSSRBootstrapBudget(SSRBootstrapBudget{InlineWarnBytes: 1, InlineErrorBytes: 0})
	if budget.InlineErrorBytes < budget.InlineWarnBytes || budget.BinaryErrorBytes < budget.BinaryWarnBytes || budget.SidecarErrorBytes < budget.SidecarWarnBytes {
		t.Fatalf("normalizeSSRBootstrapBudget() = %+v, want normalized thresholds", budget)
	}
}

func TestScopedBootstrapReadHelpers(t *testing.T) {
	bootstrap := SSRBootstrap{}

	if err := RegisterFormBootstrapDefaults(&bootstrap, " checkout ", map[string]string{"email": "cam@example.com"}); err != nil {
		t.Fatalf("RegisterFormBootstrapDefaults() error = %v", err)
	}
	if err := RegisterCacheBootstrapSeed(&bootstrap, " products:list ", []string{"sku-1", "sku-2"}); err != nil {
		t.Fatalf("RegisterCacheBootstrapSeed() error = %v", err)
	}
	if err := RegisterSessionBootstrapHint(&bootstrap, " viewer ", map[string]string{"role": "operator"}); err != nil {
		t.Fatalf("RegisterSessionBootstrapHint() error = %v", err)
	}

	formDefaults, ok, err := ReadFormBootstrapDefaults[map[string]string](bootstrap, " checkout ")
	if err != nil || !ok {
		t.Fatalf("ReadFormBootstrapDefaults() = %+v, %t, %v; want payload", formDefaults, ok, err)
	}
	if formDefaults.Kind != SSRPayloadKindFormDefaults || formDefaults.Scope != SSRPayloadScopeSubtree || formDefaults.Target != "checkout" || formDefaults.Value["email"] != "cam@example.com" {
		t.Fatalf("unexpected form defaults payload: %+v", formDefaults)
	}

	cacheSeed, ok, err := ReadCacheBootstrapSeed[[]string](bootstrap, " products:list ")
	if err != nil || !ok {
		t.Fatalf("ReadCacheBootstrapSeed() = %+v, %t, %v; want payload", cacheSeed, ok, err)
	}
	if cacheSeed.Kind != SSRPayloadKindCacheSeed || cacheSeed.Scope != SSRPayloadScopeRoute || cacheSeed.Target != "products:list" || len(cacheSeed.Value) != 2 {
		t.Fatalf("unexpected cache seed payload: %+v", cacheSeed)
	}

	sessionHint, ok, err := ReadSessionBootstrapHint[map[string]string](bootstrap, " viewer ")
	if err != nil || !ok {
		t.Fatalf("ReadSessionBootstrapHint() = %+v, %t, %v; want payload", sessionHint, ok, err)
	}
	if sessionHint.Kind != SSRPayloadKindSessionHint || sessionHint.Scope != SSRPayloadScopeApp || sessionHint.Target != "viewer" || sessionHint.ReusePolicy != SSRPayloadReuseClientOwned || sessionHint.Value["role"] != "operator" {
		t.Fatalf("unexpected session hint payload: %+v", sessionHint)
	}

	if _, ok, err := ReadFormBootstrapDefaults[map[string]string](bootstrap, "missing"); err != nil || ok {
		t.Fatalf("expected missing form defaults to return ok=false, got ok=%t err=%v", ok, err)
	}
	if _, ok, err := ReadCacheBootstrapSeed[[]string](bootstrap, "missing"); err != nil || ok {
		t.Fatalf("expected missing cache seed to return ok=false, got ok=%t err=%v", ok, err)
	}
	if _, ok, err := ReadSessionBootstrapHint[map[string]string](bootstrap, "missing"); err != nil || ok {
		t.Fatalf("expected missing session hint to return ok=false, got ok=%t err=%v", ok, err)
	}
}
