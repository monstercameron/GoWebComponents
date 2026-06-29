package app

import (
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseSeedAdminProvidersDrilldownRows seeds one compact provider-routing and guardrail fixture set for two workspaces.
func parseSeedAdminProvidersDrilldownRows(parseT *testing.T, parseStore *Store, parsePrimaryWorkspaceID int64, parseSecondaryWorkspaceID int64) {
	parseT.Helper()
	parseUpdatedAt := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseStore.db.Exec(
		`INSERT INTO workspace_model_routing_policies (workspace_id, policy_key, default_model_id, fallback_model_id, max_input_cost_per_million_usd, max_output_cost_per_million_usd, requires_approval, rules_json, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		parsePrimaryWorkspaceID,
		"default",
		modelGPT54,
		modelGPT54Mini,
		5.0,
		25.0,
		1,
		`{"mode":"balanced"}`,
		parseUpdatedAt,
	); parseErr != nil {
		parseT.Fatalf("seed workspace_model_routing_policies primary: %v", parseErr)
	}
	if _, parseErr := parseStore.db.Exec(
		`INSERT INTO workspace_model_routing_policies (workspace_id, policy_key, default_model_id, fallback_model_id, max_input_cost_per_million_usd, max_output_cost_per_million_usd, requires_approval, rules_json, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		parseSecondaryWorkspaceID,
		"default",
		modelGPT54Mini,
		modelGPT54,
		2.5,
		20.0,
		0,
		`{"mode":"cost-first"}`,
		parseUpdatedAt,
	); parseErr != nil {
		parseT.Fatalf("seed workspace_model_routing_policies secondary: %v", parseErr)
	}
	if _, parseErr := parseStore.db.Exec(
		`INSERT INTO workspace_cost_guardrails (workspace_id, guardrail_key, daily_budget_cents, monthly_budget_cents, max_cost_per_request_cents, alert_threshold_percent, action_mode, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		parsePrimaryWorkspaceID,
		"default",
		1500,
		30000,
		500,
		80,
		"notify",
		parseUpdatedAt,
	); parseErr != nil {
		parseT.Fatalf("seed workspace_cost_guardrails primary: %v", parseErr)
	}
	if _, parseErr := parseStore.db.Exec(
		`INSERT INTO workspace_cost_guardrails (workspace_id, guardrail_key, daily_budget_cents, monthly_budget_cents, max_cost_per_request_cents, alert_threshold_percent, action_mode, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		parseSecondaryWorkspaceID,
		"default",
		1200,
		25000,
		350,
		75,
		"block",
		parseUpdatedAt,
	); parseErr != nil {
		parseT.Fatalf("seed workspace_cost_guardrails secondary: %v", parseErr)
	}
}

// TestGetAdminProvidersDrilldown verifies typed providers drill-down reads provider usage, catalog, routing, and guardrail rows.
func TestGetAdminProvidersDrilldown(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseAliceAuth.ID, "ws-alice-providers-drilldown")
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-providers-drilldown")
	parseSeedAdminProvidersDrilldownRows(parseT, parseStore, parseAliceWorkspaceID, parseBobWorkspaceID)

	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-providers-drilldown-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseResp, parseErr := parseServer.GetAdminProvidersDrilldown(parseAliceCtx, &chatpb.GetAdminProvidersDrilldownRequest{
		LookbackDays: 180,
		Limit:        50,
		WorkspaceId:  parseAliceWorkspaceID,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminProvidersDrilldown: %v", parseErr)
	}
	if parseResp.GetSummary() == nil {
		parseT.Fatalf("expected providers summary payload")
	}
	if len(parseResp.GetProviders()) == 0 || len(parseResp.GetModels()) == 0 || len(parseResp.GetUsageEvents()) == 0 {
		parseT.Fatalf(
			"expected non-empty provider usage/model/event slices, got providers=%d models=%d usage_events=%d",
			len(parseResp.GetProviders()),
			len(parseResp.GetModels()),
			len(parseResp.GetUsageEvents()),
		)
	}
	if len(parseResp.GetModelCatalog()) == 0 || len(parseResp.GetProviderSnapshots()) == 0 {
		parseT.Fatalf(
			"expected model catalog + provider snapshots, got model_catalog=%d provider_snapshots=%d",
			len(parseResp.GetModelCatalog()),
			len(parseResp.GetProviderSnapshots()),
		)
	}
	if len(parseResp.GetWorkspaceModelRoutingPolicies()) == 0 || len(parseResp.GetWorkspaceCostGuardrails()) == 0 {
		parseT.Fatalf(
			"expected workspace routing + guardrail rows, got routing=%d guardrails=%d",
			len(parseResp.GetWorkspaceModelRoutingPolicies()),
			len(parseResp.GetWorkspaceCostGuardrails()),
		)
	}
	for _, parseRoutingRow := range parseResp.GetWorkspaceModelRoutingPolicies() {
		if parseRoutingRow.GetWorkspaceId() != parseAliceWorkspaceID {
			parseT.Fatalf("expected routing rows scoped to workspace %d, got %+v", parseAliceWorkspaceID, parseRoutingRow)
		}
	}
	for _, parseGuardrailRow := range parseResp.GetWorkspaceCostGuardrails() {
		if parseGuardrailRow.GetWorkspaceId() != parseAliceWorkspaceID {
			parseT.Fatalf("expected guardrail rows scoped to workspace %d, got %+v", parseAliceWorkspaceID, parseGuardrailRow)
		}
	}
	if parseResp.GetSummary().GetProviderUsageCount() == 0 || parseResp.GetSummary().GetModelCatalogCount() == 0 {
		parseT.Fatalf("expected providers summary counts, got %+v", parseResp.GetSummary())
	}

	parseGlobalResp, parseErr := parseServer.GetAdminProvidersDrilldown(parseAliceCtx, &chatpb.GetAdminProvidersDrilldownRequest{
		LookbackDays: 180,
		Limit:        50,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminProvidersDrilldown global: %v", parseErr)
	}
	if parseGlobalResp.GetSummary() == nil || parseGlobalResp.GetSummary().GetUsageEventCount() == 0 {
		parseT.Fatalf("expected providers global summary from raw/rollup path, got %+v", parseGlobalResp.GetSummary())
	}
}

// TestGetAdminProvidersDrilldownScopeGuards verifies workspace-admin callers are denied provider-surface drill-down access.
func TestGetAdminProvidersDrilldownScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-providers-scope")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-providers-drilldown-bob", parseBobAuth.ID, parseBobAuth.Email)

	if _, parseErr = parseServer.GetAdminProvidersDrilldown(parseBobCtx, &chatpb.GetAdminProvidersDrilldownRequest{
		LookbackDays: 30,
		Limit:        10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminProvidersDrilldown workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
