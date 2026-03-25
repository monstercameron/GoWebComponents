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

func (parseId opaqueID) MarshalText() ([]byte, error) {
	return []byte(string(parseId)), nil
}

func TestBootstrapVersionDefaultsAndRejectsFutureSchema(parseT *testing.T) {
	parsePayload, parseErr := UnmarshalSSRBootstrap([]byte(`{"route":{"path":"/docs"}}`))
	if parseErr != nil {
		parseT.Fatalf("unexpected unmarshal error: %v", parseErr)
	}
	if parsePayload.Version != CurrentSSRBootstrapVersion {
		parseT.Fatalf("expected default bootstrap version %d, got %d", CurrentSSRBootstrapVersion, parsePayload.Version)
	}

	if _, parseErr2 := UnmarshalSSRBootstrap([]byte(`{"version":99,"route":{"path":"/docs"}}`)); parseErr2 == nil {
		parseT.Fatal("expected future bootstrap version to be rejected")
	}
	if _, parseErr3 := UnmarshalSSRBootstrapReference([]byte(`{"url":"/bootstrap.json","version":99}`)); parseErr3 == nil {
		parseT.Fatal("expected future bootstrap reference version to be rejected")
	}
}

func TestRegisterAndReadBootstrapPayloadSupportsJSONTextBinaryAndTime(parseT *testing.T) {
	parseBootstrap := SSRBootstrap{}
	parseStamp := time.Date(2026, 3, 20, 12, 0, 0, 123456789, time.UTC)

	if parseErr := RegisterBootstrapPayload(&parseBootstrap, "profile", stateTransferProfile{Name: "Atlas", Count: 4}, SSRPayloadOptions{Scope: SSRPayloadScopeRoute, Target: "/shop/frame-bench"}); parseErr != nil {
		parseT.Fatalf("unexpected json payload registration error: %v", parseErr)
	}
	if parseErr2 := RegisterBootstrapPayload(&parseBootstrap, "raw-bytes", []byte{1, 2, 3, 4}); parseErr2 != nil {
		parseT.Fatalf("unexpected binary payload registration error: %v", parseErr2)
	}
	if parseErr3 := RegisterBootstrapPayload(&parseBootstrap, "stamp", parseStamp); parseErr3 != nil {
		parseT.Fatalf("unexpected time payload registration error: %v", parseErr3)
	}
	if parseErr4 := RegisterBootstrapPayload(&parseBootstrap, "opaque-id", opaqueID("sku_42")); parseErr4 != nil {
		parseT.Fatalf("unexpected text payload registration error: %v", parseErr4)
	}

	parseProfile, parseOk, parseErr5 := ReadBootstrapPayload[stateTransferProfile](parseBootstrap, "profile")
	if parseErr5 != nil || !parseOk {
		parseT.Fatalf("expected typed json payload, got ok=%t err=%v", parseOk, parseErr5)
	}
	if parseProfile.Value.Name != "Atlas" || parseProfile.Scope != SSRPayloadScopeRoute || parseProfile.Target != "/shop/frame-bench" {
		parseT.Fatalf("unexpected profile payload: %+v", parseProfile)
	}

	parseRawBytes, parseOk, parseErr5 := ReadBootstrapPayload[[]byte](parseBootstrap, "raw-bytes")
	if parseErr5 != nil || !parseOk {
		parseT.Fatalf("expected typed binary payload, got ok=%t err=%v", parseOk, parseErr5)
	}
	if len(parseRawBytes.Value) != 4 || parseRawBytes.Value[3] != 4 {
		parseT.Fatalf("unexpected byte payload: %+v", parseRawBytes)
	}

	parseDecodedStamp, parseOk, parseErr5 := ReadBootstrapPayload[time.Time](parseBootstrap, "stamp")
	if parseErr5 != nil || !parseOk {
		parseT.Fatalf("expected typed time payload, got ok=%t err=%v", parseOk, parseErr5)
	}
	if !parseDecodedStamp.Value.Equal(parseStamp) {
		parseT.Fatalf("expected timestamp %s, got %s", parseStamp, parseDecodedStamp.Value)
	}

	parseDecodedID, parseOk, parseErr5 := ReadBootstrapPayload[opaqueID](parseBootstrap, "opaque-id")
	if parseErr5 != nil || !parseOk {
		parseT.Fatalf("expected typed opaque id payload, got ok=%t err=%v", parseOk, parseErr5)
	}
	if parseDecodedID.Value != opaqueID("sku_42") {
		parseT.Fatalf("unexpected opaque id payload: %+v", parseDecodedID)
	}
}

func TestTypedBootstrapHelpersInspectAndLegacyFallback(parseT *testing.T) {
	parseBootstrap := SSRBootstrap{Data: map[string]interface{}{"legacy-message": "hello"}}
	if parseErr := RegisterRouteBootstrapData(&parseBootstrap, "catalog", "/products", stateTransferProfile{Name: "SSR", Count: 2}); parseErr != nil {
		parseT.Fatalf("unexpected route payload registration error: %v", parseErr)
	}
	if parseErr2 := RegisterFormBootstrapDefaults(&parseBootstrap, "checkout", map[string]string{"email": "cam@example.com"}); parseErr2 != nil {
		parseT.Fatalf("unexpected form defaults registration error: %v", parseErr2)
	}
	if parseErr3 := RegisterCacheBootstrapSeed(&parseBootstrap, "products:list", stateTransferProfile{Name: "seed", Count: 1}); parseErr3 != nil {
		parseT.Fatalf("unexpected cache seed registration error: %v", parseErr3)
	}
	if parseErr4 := RegisterSessionBootstrapHint(&parseBootstrap, "viewer", map[string]string{"role": "operator"}); parseErr4 != nil {
		parseT.Fatalf("unexpected session hint registration error: %v", parseErr4)
	}

	parseLegacy, parseOk, parseErr5 := ReadBootstrapPayload[string](parseBootstrap, "legacy-message")
	if parseErr5 != nil || !parseOk || parseLegacy.Value != "hello" {
		parseT.Fatalf("expected legacy payload fallback, got %+v ok=%t err=%v", parseLegacy, parseOk, parseErr5)
	}

	parseItems, parseErr5 := InspectBootstrapPayloads(parseBootstrap, SSRPayloadFilter{})
	if parseErr5 != nil {
		parseT.Fatalf("unexpected payload inspection error: %v", parseErr5)
	}
	if len(parseItems) != 5 {
		parseT.Fatalf("expected 5 inspected payloads, got %d", len(parseItems))
	}
	parseRouteItems, parseErr5 := InspectBootstrapPayloads(parseBootstrap, SSRPayloadFilter{Scope: SSRPayloadScopeRoute})
	if parseErr5 != nil {
		parseT.Fatalf("unexpected filtered inspection error: %v", parseErr5)
	}
	if len(parseRouteItems) != 2 {
		parseT.Fatalf("expected 2 route-scoped payloads, got %+v", parseRouteItems)
	}
}

func TestStateUpdateTextBinaryRoundTripAndApply(parseT *testing.T) {
	parseBootstrap := SSRBootstrap{}
	parseUpdate := SSRStateUpdate{CorrelationID: "req-88", Scope: SSRPayloadScopeSubtree, Target: "cart-panel"}
	if parseErr := RegisterStateUpdatePayload(&parseUpdate, "cart-summary", stateTransferProfile{Name: "updated", Count: 3}, SSRPayloadOptions{Kind: SSRPayloadKindData}); parseErr != nil {
		parseT.Fatalf("unexpected state update registration error: %v", parseErr)
	}
	parseUpdate.Deletes = []string{"old-key"}

	parseTextEncoded, parseErr2 := MarshalSSRStateUpdateText(parseUpdate)
	if parseErr2 != nil {
		parseT.Fatalf("unexpected text update marshal error: %v", parseErr2)
	}
	if !strings.Contains(string(parseTextEncoded), `"correlationId":"req-88"`) {
		parseT.Fatalf("expected correlation id in text update, got %q", parseTextEncoded)
	}
	parseDecodedText, parseErr2 := UnmarshalSSRStateUpdateText(parseTextEncoded)
	if parseErr2 != nil {
		parseT.Fatalf("unexpected text update unmarshal error: %v", parseErr2)
	}
	if parseDecodedText.Version != CurrentSSRStateUpdateVersion || parseDecodedText.Target != "cart-panel" {
		parseT.Fatalf("unexpected decoded text update: %+v", parseDecodedText)
	}

	parseBinaryEncoded, parseErr2 := MarshalSSRStateUpdateBinary(parseUpdate)
	if parseErr2 != nil {
		parseT.Fatalf("unexpected binary update marshal error: %v", parseErr2)
	}
	parseDecodedBinary, parseErr2 := UnmarshalSSRStateUpdateBinary(parseBinaryEncoded)
	if parseErr2 != nil {
		parseT.Fatalf("unexpected binary update unmarshal error: %v", parseErr2)
	}
	if parseErr3 := ApplySSRStateUpdate(&parseBootstrap, parseDecodedBinary); parseErr3 != nil {
		parseT.Fatalf("unexpected update apply error: %v", parseErr3)
	}
	parseValue, parseOk, parseErr2 := ReadBootstrapPayload[stateTransferProfile](parseBootstrap, "cart-summary")
	if parseErr2 != nil || !parseOk {
		parseT.Fatalf("expected applied payload, got ok=%t err=%v", parseOk, parseErr2)
	}
	if parseValue.Value.Count != 3 || parseValue.Target != "cart-panel" || parseValue.Scope != SSRPayloadScopeSubtree {
		parseT.Fatalf("unexpected applied payload metadata: %+v", parseValue)
	}
}

func TestAnalyzeSSRBootstrapSizeReportsRecommendation(parseT *testing.T) {
	parseBootstrap := SSRBootstrap{}
	if parseErr := RegisterBootstrapPayload(&parseBootstrap, "large", strings.Repeat("atlas", 128)); parseErr != nil {
		parseT.Fatalf("unexpected payload registration error: %v", parseErr)
	}
	parseReport, parseErr2 := InspectSSRBootstrapSize(parseBootstrap, SSRBootstrapBudget{
		InlineWarnBytes:   64,
		InlineErrorBytes:  96,
		SidecarWarnBytes:  80,
		SidecarErrorBytes: 112,
		BinaryWarnBytes:   64,
		BinaryErrorBytes:  96,
	})
	if parseErr2 != nil {
		parseT.Fatalf("unexpected bootstrap size analysis error: %v", parseErr2)
	}
	if parseReport.JSONPayloadBytes <= 0 || parseReport.BinaryPayloadBytes <= 0 || parseReport.InlineScriptBytes <= 0 {
		parseT.Fatalf("expected positive size report, got %+v", parseReport)
	}
	if parseReport.Recommendation == "inline-json" {
		parseT.Fatalf("expected a non-inline recommendation under tight budgets, got %+v", parseReport)
	}
	if len(parseReport.Warnings) == 0 && len(parseReport.Errors) == 0 {
		parseT.Fatalf("expected budget warnings or errors, got %+v", parseReport)
	}
	if len(parseReport.Payloads) != 1 {
		parseT.Fatalf("expected one payload inspection entry, got %+v", parseReport.Payloads)
	}
}
