package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseBuildAdminProviderEnabledConfigKey returns one site-config key for provider enabled state.
func parseBuildAdminProviderEnabledConfigKey(parseProviderID string) string {
	return "providers." + strings.TrimSpace(strings.ToLower(parseProviderID)) + ".enabled"
}

// parseBuildAdminProviderModelVisibilityConfigKey returns one site-config key for model visibility state.
func parseBuildAdminProviderModelVisibilityConfigKey(parseModelID string) string {
	return "providers.models." + parseNormalizeSelectedModelID(parseModelID) + ".visible"
}

// parseBuildAdminProviderLimitConfigKey returns one site-config key for provider limit controls.
func parseBuildAdminProviderLimitConfigKey(parseProviderID string) string {
	return "providers." + strings.TrimSpace(strings.ToLower(parseProviderID)) + ".limits"
}

// parseBuildAdminProviderLimitConfigJSON serializes provider limit controls into one stable JSON payload.
func parseBuildAdminProviderLimitConfigJSON(parseReq *chatpb.SetAdminProviderLimitRequest) (string, error) {
	parseLimitConfig := map[string]int64{
		"requests_per_minute": parseReq.GetRequestsPerMinute(),
		"tokens_per_minute":   parseReq.GetTokensPerMinute(),
		"requests_per_day":    parseReq.GetRequestsPerDay(),
		"tokens_per_day":      parseReq.GetTokensPerDay(),
	}
	parsePayload, parseErr := json.Marshal(parseLimitConfig)
	if parseErr != nil {
		return "", parseErr
	}
	return string(parsePayload), nil
}

// parseFindWorkspaceModelRoutingPolicyRowByScope finds one routing-policy row by workspace id and policy key.
func parseFindWorkspaceModelRoutingPolicyRowByScope(parseRows []parseWorkspaceModelRoutingPolicyRow, parseWorkspaceID int64, parsePolicyKey string) (parseWorkspaceModelRoutingPolicyRow, bool) {
	parsePolicyKey = strings.TrimSpace(parsePolicyKey)
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		if strings.TrimSpace(parseRow.PolicyKey) != parsePolicyKey {
			continue
		}
		return parseRow, true
	}
	return parseWorkspaceModelRoutingPolicyRow{}, false
}

// parseFindWorkspaceCostGuardrailRowByScope finds one cost-guardrail row by workspace id and guardrail key.
func parseFindWorkspaceCostGuardrailRowByScope(parseRows []parseWorkspaceCostGuardrailRow, parseWorkspaceID int64, parseGuardrailKey string) (parseWorkspaceCostGuardrailRow, bool) {
	parseGuardrailKey = strings.TrimSpace(parseGuardrailKey)
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceID != parseWorkspaceID {
			continue
		}
		if strings.TrimSpace(parseRow.GuardrailKey) != parseGuardrailKey {
			continue
		}
		return parseRow, true
	}
	return parseWorkspaceCostGuardrailRow{}, false
}

// SetAdminProviderEnabled stores provider enable or disable controls.
func (parseS *chatServer) SetAdminProviderEnabled(parseCtx context.Context, parseReq *chatpb.SetAdminProviderEnabledRequest) (*chatpb.SetAdminProviderEnabledResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminProviderMutationScope(parseCtx, parseAdminProviderMutationSetProviderEnabled)
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseProviderID := strings.TrimSpace(strings.ToLower(parseReq.GetProviderId()))
	if parseProviderID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider id is required")
	}
	if parseErr = parseS.store.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey:       parseBuildAdminProviderEnabledConfigKey(parseProviderID),
		ConfigValue:     strconv.FormatBool(parseReq.GetIsEnabled()),
		ValueType:       "bool",
		Description:     "Provider enabled state",
		UpdatedByUserID: parseScope.adminUserID,
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set provider enabled state: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.provider.enabled.updated",
		"provider",
		parseProviderID,
		"Admin provider enabled state updated",
		"{}",
		0,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminProviderEnabled: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.String("provider_id", parseProviderID),
			slog.Bool("is_enabled", parseReq.GetIsEnabled()),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminProviderEnabledResponse{
		ProviderId: parseProviderID,
		IsEnabled:  parseReq.GetIsEnabled(),
		Status:     "updated",
	}, nil
}

// SetAdminProviderModelVisibility stores provider model visibility controls.
func (parseS *chatServer) SetAdminProviderModelVisibility(parseCtx context.Context, parseReq *chatpb.SetAdminProviderModelVisibilityRequest) (*chatpb.SetAdminProviderModelVisibilityResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminProviderMutationScope(parseCtx, parseAdminProviderMutationSetModelVisibility)
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseModelID := parseNormalizeSelectedModelID(parseReq.GetModelId())
	if parseModelID == "" {
		return nil, status.Error(codes.InvalidArgument, "model id is required")
	}
	if parseErr = parseS.store.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey:       parseBuildAdminProviderModelVisibilityConfigKey(parseModelID),
		ConfigValue:     strconv.FormatBool(parseReq.GetIsVisible()),
		ValueType:       "bool",
		Description:     "Provider model visibility state",
		UpdatedByUserID: parseScope.adminUserID,
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set provider model visibility: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.provider.model_visibility.updated",
		"model",
		parseModelID,
		"Admin provider model visibility updated",
		"{}",
		0,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminProviderModelVisibility: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.String("model_id", parseModelID),
			slog.Bool("is_visible", parseReq.GetIsVisible()),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminProviderModelVisibilityResponse{
		ModelId:   parseModelID,
		IsVisible: parseReq.GetIsVisible(),
		Status:    "updated",
	}, nil
}

// SetAdminProviderFallbackRouting stores workspace provider fallback-routing controls.
func (parseS *chatServer) SetAdminProviderFallbackRouting(parseCtx context.Context, parseReq *chatpb.SetAdminProviderFallbackRoutingRequest) (*chatpb.SetAdminProviderFallbackRoutingResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminProviderMutationScope(parseCtx, parseAdminProviderMutationSetFallbackRouting)
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseWorkspaceID := parseReq.GetWorkspaceId()
	if parseWorkspaceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	parsePolicyKey := strings.TrimSpace(parseReq.GetPolicyKey())
	if parsePolicyKey == "" {
		parsePolicyKey = "default"
	}
	parseDefaultModelID := parseNormalizeSelectedModelID(parseReq.GetDefaultModelId())
	parseFallbackModelID := parseNormalizeSelectedModelID(parseReq.GetFallbackModelId())
	if parseDefaultModelID == "" {
		return nil, status.Error(codes.InvalidArgument, "default model id is required")
	}
	if parseFallbackModelID == "" {
		parseFallbackModelID = parseDefaultModelID
	}
	if parseErr = parseS.store.parseUpsertWorkspaceModelRoutingPolicy(parseWorkspaceModelRoutingPolicyWrite{
		WorkspaceID:                parseWorkspaceID,
		PolicyKey:                  parsePolicyKey,
		DefaultModelID:             parseDefaultModelID,
		FallbackModelID:            parseFallbackModelID,
		MaxInputCostPerMillionUSD:  parseReq.GetMaxInputCostPerMillionUsd(),
		MaxOutputCostPerMillionUSD: parseReq.GetMaxOutputCostPerMillionUsd(),
		RequiresApproval:           parseReq.GetRequiresApproval(),
		RulesJSON:                  parseReq.GetRulesJson(),
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set provider fallback routing: %v", parseErr)
	}
	parseRoutingRows, parseErr := parseS.store.parseListWorkspaceModelRoutingPolicies(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list provider fallback routing: %v", parseErr)
	}
	parseRoutingRow, hasParseRoutingRow := parseFindWorkspaceModelRoutingPolicyRowByScope(parseRoutingRows, parseWorkspaceID, parsePolicyKey)
	if !hasParseRoutingRow {
		return nil, status.Error(codes.NotFound, "workspace model routing policy not found after update")
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.provider.fallback_routing.updated",
		"workspace",
		strconv.FormatInt(parseWorkspaceID, 10),
		"Admin provider fallback routing updated",
		"{}",
		parseWorkspaceID,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminProviderFallbackRouting: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.String("policy_key", parsePolicyKey),
			slog.String("default_model_id", parseDefaultModelID),
			slog.String("fallback_model_id", parseFallbackModelID),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminProviderFallbackRoutingResponse{
		RoutingPolicy: parseBuildWorkspaceModelRoutingPolicyEntry(parseRoutingRow),
		Status:        "updated",
	}, nil
}

// SetAdminProviderLimit stores provider limit controls.
func (parseS *chatServer) SetAdminProviderLimit(parseCtx context.Context, parseReq *chatpb.SetAdminProviderLimitRequest) (*chatpb.SetAdminProviderLimitResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminProviderMutationScope(parseCtx, parseAdminProviderMutationSetProviderLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseProviderID := strings.TrimSpace(strings.ToLower(parseReq.GetProviderId()))
	if parseProviderID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider id is required")
	}
	if parseReq.GetRequestsPerMinute() < 0 || parseReq.GetTokensPerMinute() < 0 || parseReq.GetRequestsPerDay() < 0 || parseReq.GetTokensPerDay() < 0 {
		return nil, status.Error(codes.InvalidArgument, "provider limits must be non-negative")
	}
	parseLimitConfigJSON, parseErr := parseBuildAdminProviderLimitConfigJSON(parseReq)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "marshal provider limit config: %v", parseErr)
	}
	if parseErr = parseS.store.parseUpsertSiteConfig(parseSiteConfigWrite{
		ConfigKey:       parseBuildAdminProviderLimitConfigKey(parseProviderID),
		ConfigValue:     parseLimitConfigJSON,
		ValueType:       "json",
		Description:     "Provider limit controls",
		UpdatedByUserID: parseScope.adminUserID,
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set provider limits: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.provider.limit.updated",
		"provider",
		parseProviderID,
		"Admin provider limit updated",
		"{}",
		0,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminProviderLimit: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.String("provider_id", parseProviderID),
			slog.Int64("requests_per_minute", parseReq.GetRequestsPerMinute()),
			slog.Int64("tokens_per_minute", parseReq.GetTokensPerMinute()),
			slog.Int64("requests_per_day", parseReq.GetRequestsPerDay()),
			slog.Int64("tokens_per_day", parseReq.GetTokensPerDay()),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminProviderLimitResponse{
		ProviderId:        parseProviderID,
		RequestsPerMinute: parseReq.GetRequestsPerMinute(),
		TokensPerMinute:   parseReq.GetTokensPerMinute(),
		RequestsPerDay:    parseReq.GetRequestsPerDay(),
		TokensPerDay:      parseReq.GetTokensPerDay(),
		Status:            "updated",
	}, nil
}

// SetAdminProviderCostGuardrail stores workspace provider cost-guardrail controls.
func (parseS *chatServer) SetAdminProviderCostGuardrail(parseCtx context.Context, parseReq *chatpb.SetAdminProviderCostGuardrailRequest) (*chatpb.SetAdminProviderCostGuardrailResponse, error) {
	if parseReq == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if parseS == nil || parseS.store == nil {
		return nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseScope, parseErr := parseS.parseAuthorizeAdminProviderMutationScope(parseCtx, parseAdminProviderMutationSetCostGuardrail)
	if parseErr != nil {
		return nil, parseErr
	}
	parseReason, parseErr := parseRequireAdminControlMutationConfirmation(parseReq.GetConfirm(), parseReq.GetReason())
	if parseErr != nil {
		return nil, parseErr
	}
	parseWorkspaceID := parseReq.GetWorkspaceId()
	if parseWorkspaceID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "workspace id is required")
	}
	parseGuardrailKey := strings.TrimSpace(parseReq.GetGuardrailKey())
	if parseGuardrailKey == "" {
		parseGuardrailKey = "default"
	}
	if parseErr = parseS.store.parseUpsertWorkspaceCostGuardrail(parseWorkspaceCostGuardrailWrite{
		WorkspaceID:            parseWorkspaceID,
		GuardrailKey:           parseGuardrailKey,
		DailyBudgetCents:       parseReq.GetDailyBudgetCents(),
		MonthlyBudgetCents:     parseReq.GetMonthlyBudgetCents(),
		MaxCostPerRequestCents: parseReq.GetMaxCostPerRequestCents(),
		AlertThresholdPercent:  parseReq.GetAlertThresholdPercent(),
		ActionMode:             parseReq.GetActionMode(),
	}); parseErr != nil {
		return nil, status.Errorf(codes.Internal, "set provider cost guardrail: %v", parseErr)
	}
	parseGuardrailRows, parseErr := parseS.store.parseListWorkspaceCostGuardrails(parseAdminScopedScanLimit)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list provider cost guardrails: %v", parseErr)
	}
	parseGuardrailRow, hasParseGuardrailRow := parseFindWorkspaceCostGuardrailRowByScope(parseGuardrailRows, parseWorkspaceID, parseGuardrailKey)
	if !hasParseGuardrailRow {
		return nil, status.Error(codes.NotFound, "workspace cost guardrail not found after update")
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.provider.cost_guardrail.updated",
		"workspace",
		strconv.FormatInt(parseWorkspaceID, 10),
		"Admin provider cost guardrail updated",
		"{}",
		parseWorkspaceID,
	)
	if parseS.logger != nil {
		parseS.logger.Info(
			"rpc.SetAdminProviderCostGuardrail: complete",
			slog.Int64("admin_user_id", parseScope.adminUserID),
			slog.Int64("workspace_id", parseWorkspaceID),
			slog.String("guardrail_key", parseGuardrailKey),
			slog.String("reason", parseReason),
		)
	}
	return &chatpb.SetAdminProviderCostGuardrailResponse{
		Guardrail: parseBuildWorkspaceCostGuardrailEntry(parseGuardrailRow),
		Status:    "updated",
	}, nil
}
