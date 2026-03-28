package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"google.golang.org/grpc/metadata"
)

// TestResolveTraceabilityAttrsIncludesRequestCorrelationAndSupport verifies request/correlation metadata maps to traceability attrs.
func TestResolveTraceabilityAttrsIncludesRequestCorrelationAndSupport(parseT *testing.T) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-traceability-123",
		correlationIDMetadataKey, "corr-traceability-456",
	))
	parseAttrs := parseResolveTraceabilityAttrs(parseCtx)
	parseAttrValues := map[string]string{}
	for _, parseEntry := range parseAttrs {
		parseAttr, isParseAttr := parseEntry.(slog.Attr)
		if !isParseAttr {
			continue
		}
		parseAttrValues[parseAttr.Key] = parseAttr.Value.Resolve().String()
	}
	if parseAttrValues["request.id"] != "req-traceability-123" {
		parseT.Fatalf("request.id = %q, want req-traceability-123", parseAttrValues["request.id"])
	}
	if parseAttrValues["correlation.id"] != "corr-traceability-456" {
		parseT.Fatalf("correlation.id = %q, want corr-traceability-456", parseAttrValues["correlation.id"])
	}
	if parseSupportID := parseAttrValues["support.id"]; parseSupportID == "" || parseSupportID == parseSupportIDUnknown {
		parseT.Fatalf("support.id = %q, want derived support id", parseSupportID)
	}
}

// TestBuildAdminMutationTraceabilityPayloadJSON verifies typed mutation traceability payload content.
func TestBuildAdminMutationTraceabilityPayloadJSON(parseT *testing.T) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-audit-trace-123",
		correlationIDMetadataKey, "corr-audit-trace-456",
	))
	parsePayloadJSON := parseBuildAdminMutationTraceabilityPayloadJSON(parseCtx, "admin", "user")
	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parsePayloadJSON), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal(payload): %v", parseErr)
	}
	parseTraceabilityPayload, isParseMap := parsePayload["traceability"].(map[string]any)
	if !isParseMap {
		parseT.Fatalf("traceability payload missing: %+v", parsePayload)
	}
	if parseTraceabilityPayload["request_id"] != "req-audit-trace-123" || parseTraceabilityPayload["correlation_id"] != "corr-audit-trace-456" {
		parseT.Fatalf("unexpected traceability payload ids: %+v", parseTraceabilityPayload)
	}
	if parseTraceabilityPayload["actor_scope"] != "admin" || parseTraceabilityPayload["target_scope"] != "user" {
		parseT.Fatalf("unexpected traceability actor/target scope payload: %+v", parseTraceabilityPayload)
	}
}

// TestBuildLogFieldAttrsIncludesActionableFields verifies the shared builder carries route, rpc, mutation, provider, and traceability fields together.
func TestBuildLogFieldAttrsIncludesActionableFields(parseT *testing.T) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-log-fields-123",
		correlationIDMetadataKey, "corr-log-fields-456",
		traceParentMetadataKey, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		traceStateMetadataKey, "vendor=relay",
	))
	parseAttrs := parseBuildLogFieldAttrs(parseCtx, parseLogFieldSpec{
		ParseRoute:             "/app/admin/users",
		ParseRPC:               "Signup",
		ParseAction:            "workspace.suspend",
		ParseProvider:          "openai",
		ParseActorScope:        "admin",
		ParseActorUserID:       42,
		ParseWorkspaceID:       7,
		ParseTargetScope:       "workspace",
		ParseTargetID:          "ws-123",
		ParseTargetUserID:      43,
		ParseTargetWorkspaceID: 44,
	})
	parseAttrValues := map[string]string{}
	for _, parseEntry := range parseAttrs {
		parseAttr, isParseAttr := parseEntry.(slog.Attr)
		if !isParseAttr {
			continue
		}
		parseAttrValues[parseAttr.Key] = parseAttr.Value.Resolve().String()
	}
	if parseAttrValues["request.id"] != "req-log-fields-123" || parseAttrValues["correlation.id"] != "corr-log-fields-456" {
		parseT.Fatalf("unexpected request/correlation attrs: %+v", parseAttrValues)
	}
	if parseAttrValues["trace.id"] != "4bf92f3577b34da6a3ce929d0e0e4736" || parseAttrValues["span.id"] != "00f067aa0ba902b7" {
		parseT.Fatalf("unexpected trace attrs: %+v", parseAttrValues)
	}
	if parseAttrValues["support.id"] == "" || parseAttrValues["support.id"] == parseSupportIDUnknown {
		parseT.Fatalf("expected derived support id, got %q", parseAttrValues["support.id"])
	}
	if parseAttrValues["route"] != "/app/admin/users" {
		parseT.Fatalf("route = %q, want /app/admin/users", parseAttrValues["route"])
	}
	if parseAttrValues["rpc"] != "Signup" {
		parseT.Fatalf("rpc = %q, want Signup", parseAttrValues["rpc"])
	}
	if parseAttrValues["action"] != "workspace.suspend" {
		parseT.Fatalf("action = %q, want workspace.suspend", parseAttrValues["action"])
	}
	if parseAttrValues["provider"] != "openai" {
		parseT.Fatalf("provider = %q, want openai", parseAttrValues["provider"])
	}
	if parseAttrValues["actor.scope"] != "admin" || parseAttrValues["actor.user_id"] != "42" {
		parseT.Fatalf("unexpected actor attrs: %+v", parseAttrValues)
	}
	if parseAttrValues["workspace_id"] != "7" {
		parseT.Fatalf("workspace_id = %q, want 7", parseAttrValues["workspace_id"])
	}
	if parseAttrValues["target.scope"] != "workspace" || parseAttrValues["target.id"] != "ws-123" || parseAttrValues["target.user_id"] != "43" || parseAttrValues["target.workspace_id"] != "44" {
		parseT.Fatalf("unexpected target attrs: %+v", parseAttrValues)
	}
}

// BenchmarkBuildLogFieldAttrs measures the shared actionable-log builder overhead.
func BenchmarkBuildLogFieldAttrs(parseB *testing.B) {
	parseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "req-log-bench-123",
		correlationIDMetadataKey, "corr-log-bench-456",
		traceParentMetadataKey, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		traceStateMetadataKey, "vendor=relay",
	))
	parseSpec := parseLogFieldSpec{
		ParseRoute:             "/app/admin/users",
		ParseRPC:               "Signup",
		ParseAction:            "workspace.suspend",
		ParseProvider:          "openai",
		ParseActorScope:        "admin",
		ParseActorUserID:       42,
		ParseWorkspaceID:       7,
		ParseTargetScope:       "workspace",
		ParseTargetID:          "ws-123",
		ParseTargetUserID:      43,
		ParseTargetWorkspaceID: 44,
	}
	parseB.ResetTimer()
	for parseN := 0; parseN < parseB.N; parseN++ {
		_ = parseBuildLogFieldAttrs(parseCtx, parseSpec)
	}
}
