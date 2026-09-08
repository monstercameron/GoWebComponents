package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAdminOpsMutationRPCs verifies ops mutation RPCs for site-config and webhook-behavior controls.
func TestAdminOpsMutationRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-ops-mutation-owner", parseOwner.ID, parseOwner.Email)
	parseSiteConfigResp, parseErr := parseServer.SetAdminSiteConfig(parseSuperuserCtx, &chatpb.SetAdminSiteConfigRequest{
		ConfigKey:   "ops.maintenance.mode",
		ConfigValue: "enabled",
		ValueType:   "string",
		Description: "Ops maintenance mode toggle",
		Confirm:     true,
		Reason:      "ops maintenance rollout",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminSiteConfig: %v", parseErr)
	}
	if parseSiteConfigResp.GetSiteConfig() == nil || parseSiteConfigResp.GetSiteConfig().GetConfigKey() != "ops.maintenance.mode" {
		parseT.Fatalf("unexpected site-config response: %+v", parseSiteConfigResp)
	}

	parseSiteConfigRows, parseErr := parseStore.parseListSiteConfigs()
	if parseErr != nil {
		parseT.Fatalf("parseListSiteConfigs: %v", parseErr)
	}
	parseValue, hasParseValue := parseFindSiteConfigValueByKey(parseSiteConfigRows, "ops.maintenance.mode")
	if !hasParseValue || parseValue != "enabled" {
		parseT.Fatalf("expected persisted site config value enabled, got value=%q exists=%v", parseValue, hasParseValue)
	}

	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints: %v", parseErr)
	}
	if len(parseWebhookRows) == 0 {
		parseT.Fatalf("expected seeded webhook endpoints")
	}
	parseWebhookRow := parseWebhookRows[0]
	if parseErr = parseStore.parseUpsertWebhookEndpoint(parseWebhookEndpointWrite{
		WorkspaceID:    parseWebhookRow.WorkspaceID,
		Label:          parseWebhookRow.Label,
		TargetURL:      parseWebhookRow.TargetURL,
		SecretHash:     parseWebhookRow.SecretHash,
		EventsJSON:     parseWebhookRow.EventsJSON,
		IsEnabled:      true,
		LastDeliveryAt: parseWebhookRow.LastDeliveryAt,
		FailureCount:   4,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWebhookEndpoint seed failure_count: %v", parseErr)
	}
	parseWebhookBehaviorResp, parseErr := parseServer.SetAdminWebhookBehavior(parseSuperuserCtx, &chatpb.SetAdminWebhookBehaviorRequest{
		EndpointId:        parseWebhookRow.ID,
		IsEnabled:         false,
		ResetFailureCount: true,
		Confirm:           true,
		Reason:            "pause noisy endpoint",
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminWebhookBehavior: %v", parseErr)
	}
	if parseWebhookBehaviorResp.GetWebhookEndpoint() == nil || parseWebhookBehaviorResp.GetWebhookEndpoint().GetId() != parseWebhookRow.ID {
		parseT.Fatalf("unexpected webhook-behavior response: %+v", parseWebhookBehaviorResp)
	}
	if parseWebhookBehaviorResp.GetWebhookEndpoint().GetIsEnabled() {
		parseT.Fatalf("expected webhook endpoint to be disabled, got %+v", parseWebhookBehaviorResp.GetWebhookEndpoint())
	}
	if parseWebhookBehaviorResp.GetWebhookEndpoint().GetFailureCount() != 0 {
		parseT.Fatalf("expected webhook endpoint failure_count reset to 0, got %+v", parseWebhookBehaviorResp.GetWebhookEndpoint())
	}
}

// TestAdminOpsMutationRPCScopeGuards verifies workspace-admin callers are denied ops mutation RPC access.
func TestAdminOpsMutationRPCScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWebhookRows, parseErr := parseStore.parseListWebhookEndpoints(10)
	if parseErr != nil {
		parseT.Fatalf("parseListWebhookEndpoints: %v", parseErr)
	}
	if len(parseWebhookRows) == 0 {
		parseT.Fatalf("expected seeded webhook endpoints")
	}
	parseEndpointID := parseWebhookRows[0].ID

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "workspace-admin-ops-mutation@example.com")
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-ops-mutation-scope")
	parseWorkspaceAdminCtx := parseBindAuthUser(parseServer, "peer-admin-ops-mutation-workspace", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)

	if _, parseErr = parseServer.SetAdminSiteConfig(parseWorkspaceAdminCtx, &chatpb.SetAdminSiteConfigRequest{
		ConfigKey:   "ops.maintenance.mode",
		ConfigValue: "disabled",
		ValueType:   "string",
		Confirm:     true,
		Reason:      "workspace-admin attempt",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminSiteConfig workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetAdminWebhookBehavior(parseWorkspaceAdminCtx, &chatpb.SetAdminWebhookBehaviorRequest{
		EndpointId:        parseEndpointID,
		IsEnabled:         false,
		ResetFailureCount: true,
		Confirm:           true,
		Reason:            "workspace-admin attempt",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetAdminWebhookBehavior workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
