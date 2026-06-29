//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	playwright "github.com/playwright-community/playwright-go"
)

// TestExample100AdminProvidersWorkflowRegression validates the providers workflow from summary through model drill-down, policy mutation, blast-radius preview, and summary refresh.
func TestExample100AdminProvidersWorkflowRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseFixture := startExample100AdminOpsServer(parseT, parseRepoRoot, "18119")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseSummaryCtx, parseSummaryCancel := parseBuildExample100AdminMutationCallContext()
		parseSummaryResp, parseErr := parseClient.GetAdminDashboard(parseSummaryCtx, &chatpb.GetAdminDashboardRequest{
			LookbackDays: 30,
			TopLimit:     10,
			RecentLimit:  10,
		})
		parseSummaryCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminDashboard providers summary: %v", parseErr)
		}
		if len(parseSummaryResp.GetProviders()) == 0 || len(parseSummaryResp.GetModels()) == 0 {
			parseT.Fatalf("expected provider/model summary rows, got providers=%d models=%d", len(parseSummaryResp.GetProviders()), len(parseSummaryResp.GetModels()))
		}
		parseProviderID := strings.TrimSpace(parseSummaryResp.GetProviders()[0].GetProviderId())
		if parseProviderID == "" {
			parseT.Fatalf("provider summary row had blank provider id")
		}
		parseModelID := strings.TrimSpace(parseSummaryResp.GetModels()[0].GetModelId())
		if parseModelID == "" {
			parseT.Fatalf("model summary row had blank model id")
		}

		parseModelDrilldownCtx, parseModelDrilldownCancel := parseBuildExample100AdminMutationCallContext()
		parseModelDrilldownResp, parseErr := parseClient.ListAdminUsageEvents(parseModelDrilldownCtx, &chatpb.ListAdminUsageEventsRequest{
			LookbackDays: 30,
			Limit:        25,
			ProviderId:   parseProviderID,
			ListQuery: &chatpb.AdminListQuery{
				Limit:         25,
				Offset:        0,
				Search:        parseModelID,
				SortBy:        "created_at",
				SortDirection: "desc",
			},
		})
		parseModelDrilldownCancel()
		if parseErr != nil {
			parseT.Fatalf("ListAdminUsageEvents provider/model drilldown: %v", parseErr)
		}
		if len(parseModelDrilldownResp.GetEvents()) == 0 {
			parseT.Fatalf("expected provider/model drilldown rows for provider=%q model=%q, got none", parseProviderID, parseModelID)
		}

		parseFlagKey := "provider-fallback-" + strings.NewReplacer("/", "-", " ", "-", "_", "-").Replace(strings.ToLower(parseProviderID))
		parsePolicyCtx, parsePolicyCancel := parseBuildExample100AdminMutationCallContext()
		parsePolicyResp, parseErr := parseClient.SetAdminFeatureFlag(parsePolicyCtx, &chatpb.SetAdminFeatureFlagRequest{
			FlagKey:        parseFlagKey,
			IsEnabled:      true,
			RolloutPercent: 100,
			AudienceJson:   "{\"workspace\":\"all\"}",
			PayloadJson:    fmt.Sprintf("{\"provider\":\"%s\",\"policy\":\"fallback_enabled\"}", parseProviderID),
			Confirm:        true,
			Reason:         "playwright providers workflow regression: provider fallback visibility policy",
		})
		parsePolicyCancel()
		if parseErr != nil {
			parseT.Fatalf("SetAdminFeatureFlag providers workflow policy mutation: %v", parseErr)
		}
		if parsePolicyResp.GetFeatureFlag() == nil || parsePolicyResp.GetFeatureFlag().GetFlagKey() != parseFlagKey || !parsePolicyResp.GetFeatureFlag().GetIsEnabled() {
			parseT.Fatalf("SetAdminFeatureFlag returned invalid provider policy row: %+v", parsePolicyResp)
		}

		parseBlastCtx, parseBlastCancel := parseBuildExample100AdminMutationCallContext()
		parseBlastResp, parseErr := parseClient.GetAdminIncidentBlastRadius(parseBlastCtx, &chatpb.GetAdminIncidentBlastRadiusRequest{
			WorkspaceId:  parseFixture.GetTargetWorkspaceID,
			LookbackDays: 30,
		})
		parseBlastCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminIncidentBlastRadius providers workflow blast preview: %v", parseErr)
		}
		if parseBlastResp.GetBlastRadius() == nil || parseBlastResp.GetBlastRadius().GetWorkspaceId() != parseFixture.GetTargetWorkspaceID {
			parseT.Fatalf("GetAdminIncidentBlastRadius returned invalid blast preview payload: %+v", parseBlastResp)
		}

		parseRefreshCtx, parseRefreshCancel := parseBuildExample100AdminMutationCallContext()
		parseRefreshResp, parseErr := parseClient.GetAdminDashboard(parseRefreshCtx, &chatpb.GetAdminDashboardRequest{
			LookbackDays: 30,
			TopLimit:     10,
			RecentLimit:  10,
		})
		parseRefreshCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminDashboard providers summary refresh: %v", parseErr)
		}
		if len(parseRefreshResp.GetProviders()) == 0 {
			parseT.Fatalf("expected provider rows after summary refresh, got none")
		}

		parseAssertExample100AdminProvidersWorkflowRoutes(parseT, parsePage, parseBaseURL, parseProviderID, parseModelID, parseFixture.GetTargetWorkspaceID)
		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-providers workflow regression flow", parseEvidence)
	})
}

// parseAssertExample100AdminProvidersWorkflowRoutes verifies provider filter/time-range state persists across drill-down, blast-preview, and summary-refresh route transitions.
func parseAssertExample100AdminProvidersWorkflowRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseProviderID string, parseModelID string, parseWorkspaceID int64) {
	parseT.Helper()
	parseQueueRoute := fmt.Sprintf("/app/dashboard/usage?tab=providers&lookback_days=30&provider_id=%s&model_id=%s&sort_by=provider_id&sort_direction=asc&limit=25&offset=0", parseProviderID, parseModelID)
	parseRoutes := []string{
		parseQueueRoute,
		fmt.Sprintf("%s&view=model-detail", parseQueueRoute),
		fmt.Sprintf("%s&view=blast-radius-preview&workspace_id=%d", parseQueueRoute, parseWorkspaceID),
		parseQueueRoute,
	}
	for _, parseRoute := range parseRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto providers workflow route %s: %v", parseRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait providers workflow route %s: %v", parseRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on providers workflow route %s: %v", parseRoute, parseErr)
		}
	}
	parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, parseRoutes)
}
