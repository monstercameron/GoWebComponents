package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseFindSiteConfigValueByKey returns one normalized site-config value by key for test assertions.
func parseFindSiteConfigValueByKey(parseRows []parseSiteConfigRow, parseKey string) (string, bool) {
	parseNormalizedKey := parseNormalizeSUKey(parseKey)
	for _, parseRow := range parseRows {
		if parseNormalizeSUKey(parseRow.ConfigKey) != parseNormalizedKey {
			continue
		}
		return parseRow.ConfigValue, true
	}
	return "", false
}

// TestAdminProviderMutationRPCs verifies provider mutation RPCs persist site-config, routing, and guardrail controls.
func TestAdminProviderMutationRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	if len(parseWorkspaces) == 0 {
		parseT.Fatalf("expected seeded workspace rows")
	}
	parseWorkspaceID := parseWorkspaces[0].ID

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-provider-mutation-owner", parseOwner.ID, parseOwner.Email)
	parseEnabledResp, parseErr := parseServer.SetAdminProviderEnabled(parseSuperuserCtx, &chatpb.SetAdminProviderEnabledRequest{
		ProviderId: "openai",
		IsEnabled:  false,
		Confirm:    true,
		Reason:     "maintenance window",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminProviderEnabled: %v", parseErr)
	}
	if parseEnabledResp.GetProviderId() != "openai" || parseEnabledResp.GetIsEnabled() {
		parseT.Fatalf("unexpected provider-enabled response: %+v", parseEnabledResp)
	}

	parseVisibilityResp, parseErr := parseServer.SetAdminProviderModelVisibility(parseSuperuserCtx, &chatpb.SetAdminProviderModelVisibilityRequest{
		ModelId:   modelGPT54Mini,
		IsVisible: false,
		Confirm:   true,
		Reason:    "temporary hide low-tier model",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminProviderModelVisibility: %v", parseErr)
	}
	if parseVisibilityResp.GetModelId() != modelGPT54Mini || parseVisibilityResp.GetIsVisible() {
		parseT.Fatalf("unexpected provider-model visibility response: %+v", parseVisibilityResp)
	}

	parseLimitResp, parseErr := parseServer.SetAdminProviderLimit(parseSuperuserCtx, &chatpb.SetAdminProviderLimitRequest{
		ProviderId:        "openai",
		RequestsPerMinute: 120,
		TokensPerMinute:   60000,
		RequestsPerDay:    25000,
		TokensPerDay:      8000000,
		Confirm:           true,
		Reason:            "set provider ceilings",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminProviderLimit: %v", parseErr)
	}
	if parseLimitResp.GetProviderId() != "openai" || parseLimitResp.GetRequestsPerMinute() != 120 {
		parseT.Fatalf("unexpected provider-limit response: %+v", parseLimitResp)
	}

	parseFallbackResp, parseErr := parseServer.SetAdminProviderFallbackRouting(parseSuperuserCtx, &chatpb.SetAdminProviderFallbackRoutingRequest{
		WorkspaceId:                parseWorkspaceID,
		PolicyKey:                  "default",
		DefaultModelId:             modelGPT54,
		FallbackModelId:            modelGPT54Mini,
		MaxInputCostPerMillionUsd:  5.0,
		MaxOutputCostPerMillionUsd: 20.0,
		RequiresApproval:           true,
		RulesJson:                  `{"route":"balanced"}`,
		Confirm:                    true,
		Reason:                     "set workspace fallback routing",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminProviderFallbackRouting: %v", parseErr)
	}
	if parseFallbackResp.GetRoutingPolicy() == nil || parseFallbackResp.GetRoutingPolicy().GetWorkspaceId() != parseWorkspaceID {
		parseT.Fatalf("unexpected fallback-routing response: %+v", parseFallbackResp)
	}

	parseGuardrailResp, parseErr := parseServer.SetAdminProviderCostGuardrail(parseSuperuserCtx, &chatpb.SetAdminProviderCostGuardrailRequest{
		WorkspaceId:            parseWorkspaceID,
		GuardrailKey:           "default",
		DailyBudgetCents:       1500,
		MonthlyBudgetCents:     30000,
		MaxCostPerRequestCents: 500,
		AlertThresholdPercent:  80,
		ActionMode:             "notify",
		Confirm:                true,
		Reason:                 "set workspace cost guardrail",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminProviderCostGuardrail: %v", parseErr)
	}
	if parseGuardrailResp.GetGuardrail() == nil || parseGuardrailResp.GetGuardrail().GetWorkspaceId() != parseWorkspaceID {
		parseT.Fatalf("unexpected cost-guardrail response: %+v", parseGuardrailResp)
	}

	parseConfigRows, parseErr := parseStore.parseListSiteConfigs()
	if parseErr != nil {
		parseT.Fatalf("parseListSiteConfigs: %v", parseErr)
	}
	if parseValue, hasParseValue := parseFindSiteConfigValueByKey(parseConfigRows, parseBuildAdminProviderEnabledConfigKey("openai")); !hasParseValue || parseValue != "false" {
		parseT.Fatalf("expected provider enabled site config false, got value=%q exists=%v", parseValue, hasParseValue)
	}
	if parseValue, hasParseValue := parseFindSiteConfigValueByKey(parseConfigRows, parseBuildAdminProviderModelVisibilityConfigKey(modelGPT54Mini)); !hasParseValue || parseValue != "false" {
		parseT.Fatalf("expected provider model visibility site config false, got value=%q exists=%v", parseValue, hasParseValue)
	}
	if _, hasParseValue := parseFindSiteConfigValueByKey(parseConfigRows, parseBuildAdminProviderLimitConfigKey("openai")); !hasParseValue {
		parseT.Fatalf("expected provider limit site config row")
	}
}

// TestAdminProviderMutationRPCScopeGuards verifies provider mutation RPCs reject workspace-admin callers.
func TestAdminProviderMutationRPCScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaces, parseErr := parseStore.parseListWorkspaces(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	if len(parseWorkspaces) == 0 {
		parseT.Fatalf("expected seeded workspace rows")
	}
	parseWorkspaceID := parseWorkspaces[0].ID

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-provider-mutation@example.com")
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-provider-mutation-scope")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-provider-mutation-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)
	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-provider-mutation-owner", parseOwner.ID, parseOwner.Email)

	if _, parseErr = parseServer.SetAdminProviderEnabled(parseWorkspaceAdminCtx, &chatpb.SetAdminProviderEnabledRequest{
		ProviderId: "openai",
		IsEnabled:  true,
		Confirm:    true,
		Reason:     "workspace-admin attempt",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminProviderEnabled workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetAdminProviderModelVisibility(parseWorkspaceAdminCtx, &chatpb.SetAdminProviderModelVisibilityRequest{
		ModelId:   modelGPT54Mini,
		IsVisible: true,
		Confirm:   true,
		Reason:    "workspace-admin attempt",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminProviderModelVisibility workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetAdminProviderLimit(parseWorkspaceAdminCtx, &chatpb.SetAdminProviderLimitRequest{
		ProviderId:        "openai",
		RequestsPerMinute: 10,
		TokensPerMinute:   1000,
		RequestsPerDay:    100,
		TokensPerDay:      100000,
		Confirm:           true,
		Reason:            "workspace-admin attempt",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminProviderLimit workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetAdminProviderFallbackRouting(parseWorkspaceAdminCtx, &chatpb.SetAdminProviderFallbackRoutingRequest{
		WorkspaceId:     parseWorkspaceID,
		PolicyKey:       "default",
		DefaultModelId:  modelGPT54,
		FallbackModelId: modelGPT54Mini,
		Confirm:         true,
		Reason:          "workspace-admin attempt",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminProviderFallbackRouting workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetAdminProviderCostGuardrail(parseWorkspaceAdminCtx, &chatpb.SetAdminProviderCostGuardrailRequest{
		WorkspaceId:            parseWorkspaceID,
		GuardrailKey:           "default",
		DailyBudgetCents:       1000,
		MonthlyBudgetCents:     20000,
		MaxCostPerRequestCents: 200,
		AlertThresholdPercent:  80,
		ActionMode:             "notify",
		Confirm:                true,
		Reason:                 "workspace-admin attempt",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminProviderCostGuardrail workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.SetAdminProviderEnabled(parseSuperuserCtx, &chatpb.SetAdminProviderEnabledRequest{
		ProviderId: "openai",
		IsEnabled:  true,
		Confirm:    true,
		Reason:     "superuser control check",
	}); parseErr != nil {
		parseT.Fatalf("SetAdminProviderEnabled superuser expected allow: %v", parseErr)
	}
}
