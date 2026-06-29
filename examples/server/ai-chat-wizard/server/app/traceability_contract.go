package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"strings"
)

type parseLogFieldSpec struct {
	ParseRoute             string
	ParseRPC               string
	ParseAction            string
	ParseProvider          string
	ParseActorScope        string
	ParseActorUserID       int64
	ParseWorkspaceID       int64
	ParseTargetScope       string
	ParseTargetID          string
	ParseTargetUserID      int64
	ParseTargetWorkspaceID int64
}

// parseResolveTraceabilityAttrs resolves one shared traceability attribute set from request metadata.
func parseResolveTraceabilityAttrs(parseCtx context.Context) []any {
	parseRequestID, parseCorrelationID := parseExtractCorrelationFromContext(parseCtx)
	parseTraceID, parseSpanID, _ := parseExtractTraceContextFromContext(parseCtx)
	parseSupportExposure := parseResolveSupportIDExposure(parseCtx, "", "")

	parseAttrs := make([]any, 0, 5)
	if strings.TrimSpace(parseRequestID) != "" {
		parseAttrs = append(parseAttrs, slog.String("request.id", strings.TrimSpace(parseRequestID)))
	}
	if strings.TrimSpace(parseCorrelationID) != "" {
		parseAttrs = append(parseAttrs, slog.String("correlation.id", strings.TrimSpace(parseCorrelationID)))
	}
	if strings.TrimSpace(parseTraceID) != "" {
		parseAttrs = append(parseAttrs, slog.String("trace.id", strings.TrimSpace(parseTraceID)))
	}
	if strings.TrimSpace(parseSpanID) != "" {
		parseAttrs = append(parseAttrs, slog.String("span.id", strings.TrimSpace(parseSpanID)))
	}
	if strings.TrimSpace(parseSupportExposure.ParseCustomerSupportID) != "" {
		parseAttrs = append(parseAttrs, slog.String("support.id", strings.TrimSpace(parseSupportExposure.ParseCustomerSupportID)))
	}
	return parseAttrs
}

// parseBuildLogFieldAttrs builds one shared actionable-log attribute set from traceability, route, RPC, mutation, and provider inputs.
func parseBuildLogFieldAttrs(parseCtx context.Context, parseSpec parseLogFieldSpec) []any {
	parseAttrs := parseResolveTraceabilityAttrs(parseCtx)
	parseSpec.ParseRoute = strings.TrimSpace(parseSpec.ParseRoute)
	parseSpec.ParseRPC = strings.TrimSpace(parseSpec.ParseRPC)
	parseSpec.ParseAction = strings.TrimSpace(parseSpec.ParseAction)
	parseSpec.ParseProvider = strings.TrimSpace(parseSpec.ParseProvider)
	parseSpec.ParseActorScope = strings.TrimSpace(parseSpec.ParseActorScope)
	parseSpec.ParseTargetScope = strings.TrimSpace(parseSpec.ParseTargetScope)
	parseSpec.ParseTargetID = strings.TrimSpace(parseSpec.ParseTargetID)

	if parseSpec.ParseRoute != "" {
		parseAttrs = append(parseAttrs, slog.String("route", parseSpec.ParseRoute))
	}
	if parseSpec.ParseRPC != "" {
		parseAttrs = append(parseAttrs, slog.String("rpc", parseSpec.ParseRPC))
	}
	if parseSpec.ParseAction != "" {
		parseAttrs = append(parseAttrs, slog.String("action", parseSpec.ParseAction))
	}
	if parseSpec.ParseProvider != "" {
		parseAttrs = append(parseAttrs, slog.String("provider", parseSpec.ParseProvider))
	}
	if parseSpec.ParseActorScope != "" {
		parseAttrs = append(parseAttrs, slog.String("actor.scope", parseSpec.ParseActorScope))
	}
	if parseSpec.ParseActorUserID > 0 {
		parseAttrs = append(parseAttrs, slog.Int64("actor.user_id", parseSpec.ParseActorUserID))
	}
	if parseSpec.ParseWorkspaceID > 0 {
		parseAttrs = append(parseAttrs, slog.Int64("workspace_id", parseSpec.ParseWorkspaceID))
	}
	if parseSpec.ParseTargetScope != "" {
		parseAttrs = append(parseAttrs, slog.String("target.scope", parseSpec.ParseTargetScope))
	}
	if parseSpec.ParseTargetID != "" {
		parseAttrs = append(parseAttrs, slog.String("target.id", parseSpec.ParseTargetID))
	}
	if parseSpec.ParseTargetUserID > 0 {
		parseAttrs = append(parseAttrs, slog.Int64("target.user_id", parseSpec.ParseTargetUserID))
	}
	if parseSpec.ParseTargetWorkspaceID > 0 {
		parseAttrs = append(parseAttrs, slog.Int64("target.workspace_id", parseSpec.ParseTargetWorkspaceID))
	}
	return parseAttrs
}

// parseBuildTraceabilityJSON builds one shared JSON traceability envelope from request metadata.
func parseBuildTraceabilityJSON(parseCtx context.Context) string {
	parseRequestID, parseCorrelationID := parseExtractCorrelationFromContext(parseCtx)
	parseTraceID, parseSpanID, parseTraceState := parseExtractTraceContextFromContext(parseCtx)
	parseSupportExposure := parseResolveSupportIDExposure(parseCtx, "", "")

	parseTraceability := map[string]any{}
	if strings.TrimSpace(parseRequestID) != "" {
		parseTraceability["request_id"] = strings.TrimSpace(parseRequestID)
	}
	if strings.TrimSpace(parseCorrelationID) != "" {
		parseTraceability["correlation_id"] = strings.TrimSpace(parseCorrelationID)
	}
	if strings.TrimSpace(parseTraceID) != "" {
		parseTraceability["trace_id"] = strings.TrimSpace(parseTraceID)
	}
	if strings.TrimSpace(parseSpanID) != "" {
		parseTraceability["span_id"] = strings.TrimSpace(parseSpanID)
	}
	if strings.TrimSpace(parseTraceState) != "" {
		parseTraceability["trace_state"] = strings.TrimSpace(parseTraceState)
	}
	if strings.TrimSpace(parseSupportExposure.ParseCustomerSupportID) != "" {
		parseTraceability["support_id"] = strings.TrimSpace(parseSupportExposure.ParseCustomerSupportID)
	}
	if len(parseTraceability) == 0 {
		return `{"traceability":{}}`
	}
	parseEnvelope := map[string]any{
		"traceability": parseTraceability,
	}
	parseJSON, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return `{"traceability":{}}`
	}
	return string(parseJSON)
}

// parseMergeJSONObjectStrings merges two JSON object strings and prefers overlay values for duplicate keys.
func parseMergeJSONObjectStrings(parseBaseJSON string, parseOverlayJSON string) string {
	parseBaseJSON = strings.TrimSpace(parseBaseJSON)
	parseOverlayJSON = strings.TrimSpace(parseOverlayJSON)
	switch {
	case parseBaseJSON == "":
		return parseOverlayJSON
	case parseOverlayJSON == "":
		return parseBaseJSON
	}

	parseBaseMap := map[string]any{}
	if parseErr := json.Unmarshal([]byte(parseBaseJSON), &parseBaseMap); parseErr != nil {
		return parseBaseJSON
	}
	parseOverlayMap := map[string]any{}
	if parseErr := json.Unmarshal([]byte(parseOverlayJSON), &parseOverlayMap); parseErr != nil {
		return parseBaseJSON
	}
	maps.Copy(parseBaseMap, parseOverlayMap)
	parseMergedJSON, parseErr := json.Marshal(parseBaseMap)
	if parseErr != nil {
		return parseBaseJSON
	}
	return string(parseMergedJSON)
}

// parseResolveActorTargetScopeAttrs builds one actor/target scope attribute set for privileged and customer-impacting flows.
func parseResolveActorTargetScopeAttrs(parseActorScope string, parseActorUserID int64, parseTargetScope string, parseTargetUserID int64, parseTargetWorkspaceID int64) []any {
	parseAttrs := make([]any, 0, 5)
	parseActorScope = strings.TrimSpace(parseActorScope)
	parseTargetScope = strings.TrimSpace(parseTargetScope)
	if parseActorScope != "" {
		parseAttrs = append(parseAttrs, slog.String("actor.scope", parseActorScope))
	}
	if parseActorUserID > 0 {
		parseAttrs = append(parseAttrs, slog.Int64("actor.user_id", parseActorUserID))
	}
	if parseTargetScope != "" {
		parseAttrs = append(parseAttrs, slog.String("target.scope", parseTargetScope))
	}
	if parseTargetUserID > 0 {
		parseAttrs = append(parseAttrs, slog.Int64("target.user_id", parseTargetUserID))
	}
	if parseTargetWorkspaceID > 0 {
		parseAttrs = append(parseAttrs, slog.Int64("target.workspace_id", parseTargetWorkspaceID))
	}
	return parseAttrs
}
