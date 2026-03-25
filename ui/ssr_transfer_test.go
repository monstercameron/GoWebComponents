package ui

import (
	"strings"
	"testing"
	"time"
)

type stateTransferProfile struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type opaqueID string

func (id opaqueID) MarshalText() ([]byte, error) {
	return []byte(string(id)), nil
}

func TestBootstrapVersionDefaultsAndRejectsFutureSchema(t *testing.T) {
	payload, err := UnmarshalSSRBootstrap([]byte(`{"route":{"path":"/docs"}}`))
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if payload.Version != CurrentSSRBootstrapVersion {
		t.Fatalf("expected default bootstrap version %d, got %d", CurrentSSRBootstrapVersion, payload.Version)
	}

	if _, err := UnmarshalSSRBootstrap([]byte(`{"version":99,"route":{"path":"/docs"}}`)); err == nil {
		t.Fatal("expected future bootstrap version to be rejected")
	}
	if _, err := UnmarshalSSRBootstrapReference([]byte(`{"url":"/bootstrap.json","version":99}`)); err == nil {
		t.Fatal("expected future bootstrap reference version to be rejected")
	}
}

func TestRegisterAndReadBootstrapPayloadSupportsJSONTextBinaryAndTime(t *testing.T) {
	bootstrap := SSRBootstrap{}
	stamp := time.Date(2026, 3, 20, 12, 0, 0, 123456789, time.UTC)

	if err := RegisterBootstrapPayload(&bootstrap, "profile", stateTransferProfile{Name: "Atlas", Count: 4}, SSRPayloadOptions{Scope: SSRPayloadScopeRoute, Target: "/shop/frame-bench"}); err != nil {
		t.Fatalf("unexpected json payload registration error: %v", err)
	}
	if err := RegisterBootstrapPayload(&bootstrap, "raw-bytes", []byte{1, 2, 3, 4}); err != nil {
		t.Fatalf("unexpected binary payload registration error: %v", err)
	}
	if err := RegisterBootstrapPayload(&bootstrap, "stamp", stamp); err != nil {
		t.Fatalf("unexpected time payload registration error: %v", err)
	}
	if err := RegisterBootstrapPayload(&bootstrap, "opaque-id", opaqueID("sku_42")); err != nil {
		t.Fatalf("unexpected text payload registration error: %v", err)
	}

	profile, ok, err := ReadBootstrapPayload[stateTransferProfile](bootstrap, "profile")
	if err != nil || !ok {
		t.Fatalf("expected typed json payload, got ok=%t err=%v", ok, err)
	}
	if profile.Value.Name != "Atlas" || profile.Scope != SSRPayloadScopeRoute || profile.Target != "/shop/frame-bench" {
		t.Fatalf("unexpected profile payload: %+v", profile)
	}

	rawBytes, ok, err := ReadBootstrapPayload[[]byte](bootstrap, "raw-bytes")
	if err != nil || !ok {
		t.Fatalf("expected typed binary payload, got ok=%t err=%v", ok, err)
	}
	if len(rawBytes.Value) != 4 || rawBytes.Value[3] != 4 {
		t.Fatalf("unexpected byte payload: %+v", rawBytes)
	}

	decodedStamp, ok, err := ReadBootstrapPayload[time.Time](bootstrap, "stamp")
	if err != nil || !ok {
		t.Fatalf("expected typed time payload, got ok=%t err=%v", ok, err)
	}
	if !decodedStamp.Value.Equal(stamp) {
		t.Fatalf("expected timestamp %s, got %s", stamp, decodedStamp.Value)
	}

	decodedID, ok, err := ReadBootstrapPayload[opaqueID](bootstrap, "opaque-id")
	if err != nil || !ok {
		t.Fatalf("expected typed opaque id payload, got ok=%t err=%v", ok, err)
	}
	if decodedID.Value != opaqueID("sku_42") {
		t.Fatalf("unexpected opaque id payload: %+v", decodedID)
	}
}

func TestTypedBootstrapHelpersInspectAndLegacyFallback(t *testing.T) {
	bootstrap := SSRBootstrap{Data: map[string]interface{}{"legacy-message": "hello"}}
	if err := RegisterRouteBootstrapData(&bootstrap, "catalog", "/products", stateTransferProfile{Name: "SSR", Count: 2}); err != nil {
		t.Fatalf("unexpected route payload registration error: %v", err)
	}
	if err := RegisterFormBootstrapDefaults(&bootstrap, "checkout", map[string]string{"email": "cam@example.com"}); err != nil {
		t.Fatalf("unexpected form defaults registration error: %v", err)
	}
	if err := RegisterCacheBootstrapSeed(&bootstrap, "products:list", stateTransferProfile{Name: "seed", Count: 1}); err != nil {
		t.Fatalf("unexpected cache seed registration error: %v", err)
	}
	if err := RegisterSessionBootstrapHint(&bootstrap, "viewer", map[string]string{"role": "operator"}); err != nil {
		t.Fatalf("unexpected session hint registration error: %v", err)
	}

	legacy, ok, err := ReadBootstrapPayload[string](bootstrap, "legacy-message")
	if err != nil || !ok || legacy.Value != "hello" {
		t.Fatalf("expected legacy payload fallback, got %+v ok=%t err=%v", legacy, ok, err)
	}

	items, err := InspectBootstrapPayloads(bootstrap, SSRPayloadFilter{})
	if err != nil {
		t.Fatalf("unexpected payload inspection error: %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("expected 5 inspected payloads, got %d", len(items))
	}
	routeItems, err := InspectBootstrapPayloads(bootstrap, SSRPayloadFilter{Scope: SSRPayloadScopeRoute})
	if err != nil {
		t.Fatalf("unexpected filtered inspection error: %v", err)
	}
	if len(routeItems) != 2 {
		t.Fatalf("expected 2 route-scoped payloads, got %+v", routeItems)
	}
}

func TestStateUpdateTextBinaryRoundTripAndApply(t *testing.T) {
	bootstrap := SSRBootstrap{}
	update := SSRStateUpdate{CorrelationID: "req-88", Scope: SSRPayloadScopeSubtree, Target: "cart-panel"}
	if err := RegisterStateUpdatePayload(&update, "cart-summary", stateTransferProfile{Name: "updated", Count: 3}, SSRPayloadOptions{Kind: SSRPayloadKindData}); err != nil {
		t.Fatalf("unexpected state update registration error: %v", err)
	}
	update.Deletes = []string{"old-key"}

	textEncoded, err := MarshalSSRStateUpdateText(update)
	if err != nil {
		t.Fatalf("unexpected text update marshal error: %v", err)
	}
	if !strings.Contains(string(textEncoded), `"correlationId":"req-88"`) {
		t.Fatalf("expected correlation id in text update, got %q", textEncoded)
	}
	decodedText, err := UnmarshalSSRStateUpdateText(textEncoded)
	if err != nil {
		t.Fatalf("unexpected text update unmarshal error: %v", err)
	}
	if decodedText.Version != CurrentSSRStateUpdateVersion || decodedText.Target != "cart-panel" {
		t.Fatalf("unexpected decoded text update: %+v", decodedText)
	}

	binaryEncoded, err := MarshalSSRStateUpdateBinary(update)
	if err != nil {
		t.Fatalf("unexpected binary update marshal error: %v", err)
	}
	decodedBinary, err := UnmarshalSSRStateUpdateBinary(binaryEncoded)
	if err != nil {
		t.Fatalf("unexpected binary update unmarshal error: %v", err)
	}
	if err := ApplySSRStateUpdate(&bootstrap, decodedBinary); err != nil {
		t.Fatalf("unexpected update apply error: %v", err)
	}
	value, ok, err := ReadBootstrapPayload[stateTransferProfile](bootstrap, "cart-summary")
	if err != nil || !ok {
		t.Fatalf("expected applied payload, got ok=%t err=%v", ok, err)
	}
	if value.Value.Count != 3 || value.Target != "cart-panel" || value.Scope != SSRPayloadScopeSubtree {
		t.Fatalf("unexpected applied payload metadata: %+v", value)
	}
}

func TestAnalyzeSSRBootstrapSizeReportsRecommendation(t *testing.T) {
	bootstrap := SSRBootstrap{}
	if err := RegisterBootstrapPayload(&bootstrap, "large", strings.Repeat("atlas", 128)); err != nil {
		t.Fatalf("unexpected payload registration error: %v", err)
	}
	report, err := InspectSSRBootstrapSize(bootstrap, SSRBootstrapBudget{
		InlineWarnBytes:   64,
		InlineErrorBytes:  96,
		SidecarWarnBytes:  80,
		SidecarErrorBytes: 112,
		BinaryWarnBytes:   64,
		BinaryErrorBytes:  96,
	})
	if err != nil {
		t.Fatalf("unexpected bootstrap size analysis error: %v", err)
	}
	if report.JSONPayloadBytes <= 0 || report.BinaryPayloadBytes <= 0 || report.InlineScriptBytes <= 0 {
		t.Fatalf("expected positive size report, got %+v", report)
	}
	if report.Recommendation == "inline-json" {
		t.Fatalf("expected a non-inline recommendation under tight budgets, got %+v", report)
	}
	if len(report.Warnings) == 0 && len(report.Errors) == 0 {
		t.Fatalf("expected budget warnings or errors, got %+v", report)
	}
	if len(report.Payloads) != 1 {
		t.Fatalf("expected one payload inspection entry, got %+v", report.Payloads)
	}
}
