//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	playwright "github.com/mxschmitt/playwright-go"
)

// TestExample100DashboardSettingsMutationsRegression validates dashboard settings mutations and verifies refreshed slices return updated rows without stale data.
func TestExample100DashboardSettingsMutationsRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseFixture := startExample100AdminListMechanicsServer(parseT, parseRepoRoot, "18113")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parsePlanCode := fmt.Sprintf("reg-plan-%d", time.Now().UTC().UnixNano())
		parseQuotaKey := fmt.Sprintf("reg-quota-%d", time.Now().UTC().UnixNano())
		parseFlagKey := fmt.Sprintf("reg-flag-%d", time.Now().UTC().UnixNano())
		parseIncidentKey := fmt.Sprintf("reg-incident-%d", time.Now().UTC().UnixNano())
		parseSLOKey := fmt.Sprintf("reg-slo-%d", time.Now().UTC().UnixNano())

		parseApplyExample100DashboardSettingsMutations(parseT, parseClient, parseFixture.GetTargetWorkspaceID, parsePlanCode, parseQuotaKey, parseFlagKey, parseSLOKey, parseIncidentKey)
		parseAssertExample100DashboardSettingsRows(parseT, parseClient, parsePlanCode, parseQuotaKey, parseFlagKey, parseIncidentKey)
		parseAssertExample100DashboardSettingsRoutes(parseT, parsePage, parseBaseURL, parsePlanCode, parseFlagKey, parseIncidentKey)
		parseAssertExample100DashboardSettingsRows(parseT, parseClient, parsePlanCode, parseQuotaKey, parseFlagKey, parseIncidentKey)

		assertExample100AdminGuardNoRuntimeErrors(parseT, "dashboard settings mutation regression flow", parseEvidence)
	})
}

// parseApplyExample100DashboardSettingsMutations executes billing, feature-flag, and incident settings mutations used by the regression.
func parseApplyExample100DashboardSettingsMutations(parseT *testing.T, parseClient chatpb.ChatServiceClient, parseWorkspaceID int64, parsePlanCode string, parseQuotaKey string, parseFlagKey string, parseSLOKey string, parseIncidentKey string) {
	parseT.Helper()

	parseOverageCtx, parseOverageCancel := parseBuildExample100AdminMutationCallContext()
	parseOverageResp, parseErr := parseClient.SetSuperuserBillingPlanOverage(parseOverageCtx, &chatpb.SetSuperuserBillingPlanOverageRequest{
		PlanCode:          parsePlanCode,
		MeterKey:          "tokens",
		IncludedUnits:     1000,
		SoftLimitUnits:    2000,
		HardLimitUnits:    4000,
		OverageUnitSize:   100,
		OveragePriceCents: 5,
		BillingInterval:   "monthly",
		Confirm:           true,
		Reason:            "playwright dashboard-settings regression: set overage",
	})
	parseOverageCancel()
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserBillingPlanOverage: %v", parseErr)
	}
	if parseOverageResp.GetOverage() == nil || parseOverageResp.GetOverage().GetPlanCode() != parsePlanCode {
		parseT.Fatalf("SetSuperuserBillingPlanOverage returned invalid row: %+v", parseOverageResp)
	}

	parseQuotaCtx, parseQuotaCancel := parseBuildExample100AdminMutationCallContext()
	parseQuotaResp, parseErr := parseClient.SetSuperuserBillingQuotaPolicy(parseQuotaCtx, &chatpb.SetSuperuserBillingQuotaPolicyRequest{
		PlanCode:        parsePlanCode,
		QuotaKey:        parseQuotaKey,
		SoftLimitValue:  80,
		HardLimitValue:  100,
		ResetInterval:   "monthly",
		EnforcementMode: "hard",
		Confirm:         true,
		Reason:          "playwright dashboard-settings regression: set quota policy",
	})
	parseQuotaCancel()
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserBillingQuotaPolicy: %v", parseErr)
	}
	if parseQuotaResp.GetPolicy() == nil || parseQuotaResp.GetPolicy().GetPlanCode() != parsePlanCode || parseQuotaResp.GetPolicy().GetQuotaKey() != parseQuotaKey {
		parseT.Fatalf("SetSuperuserBillingQuotaPolicy returned invalid row: %+v", parseQuotaResp)
	}

	parseFlagCtx, parseFlagCancel := parseBuildExample100AdminMutationCallContext()
	parseFlagResp, parseErr := parseClient.SetAdminFeatureFlag(parseFlagCtx, &chatpb.SetAdminFeatureFlagRequest{
		FlagKey:        parseFlagKey,
		IsEnabled:      true,
		RolloutPercent: 100,
		AudienceJson:   "{\"workspace\":\"all\"}",
		PayloadJson:    "{\"source\":\"dashboard-settings-regression\"}",
		Confirm:        true,
		Reason:         "playwright dashboard-settings regression: set feature flag",
	})
	parseFlagCancel()
	if parseErr != nil {
		parseT.Fatalf("SetAdminFeatureFlag: %v", parseErr)
	}
	if parseFlagResp.GetFeatureFlag() == nil || parseFlagResp.GetFeatureFlag().GetFlagKey() != parseFlagKey || !parseFlagResp.GetFeatureFlag().GetIsEnabled() {
		parseT.Fatalf("SetAdminFeatureFlag returned invalid row: %+v", parseFlagResp)
	}

	parseSLOCtx, parseSLOCancel := parseBuildExample100AdminMutationCallContext()
	_, parseErr = parseClient.SetSuperuserServiceLevelObjective(parseSLOCtx, &chatpb.SetSuperuserServiceLevelObjectiveRequest{
		SloKey:             parseSLOKey,
		ServiceName:        "dashboard-settings-regression",
		ObjectivePercent:   99.9,
		WindowDays:         30,
		ErrorBudgetMinutes: 43,
		StatusPageUrl:      "https://status.example.invalid/dashboard-settings-regression",
		Confirm:            true,
		Reason:             "playwright dashboard-settings regression: seed slo",
	})
	parseSLOCancel()
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserServiceLevelObjective: %v", parseErr)
	}

	parseIncidentSetCtx, parseIncidentSetCancel := parseBuildExample100AdminMutationCallContext()
	parseIncidentSetResp, parseErr := parseClient.SetSuperuserIncident(parseIncidentSetCtx, &chatpb.SetSuperuserIncidentRequest{
		IncidentKey: parseIncidentKey,
		SloKey:      parseSLOKey,
		Severity:    "major",
		Status:      "open",
		Title:       "Dashboard settings mutation regression incident",
		Summary:     "Seed incident for mutation refresh regression checks",
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		Confirm:     true,
		Reason:      "playwright dashboard-settings regression: seed incident",
	})
	parseIncidentSetCancel()
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserIncident: %v", parseErr)
	}
	if parseIncidentSetResp.GetIncident() == nil || parseIncidentSetResp.GetIncident().GetId() <= 0 {
		parseT.Fatalf("SetSuperuserIncident returned invalid row: %+v", parseIncidentSetResp)
	}

	parseIncidentUpdateCtx, parseIncidentUpdateCancel := parseBuildExample100AdminMutationCallContext()
	parseIncidentUpdateResp, parseErr := parseClient.UpdateAdminIncident(parseIncidentUpdateCtx, &chatpb.UpdateAdminIncidentRequest{
		IncidentId:  parseIncidentSetResp.GetIncident().GetId(),
		WorkspaceId: parseWorkspaceID,
		Status:      "investigating",
		Message:     "dashboard settings regression update",
		IsPublic:    false,
		Confirm:     true,
		Reason:      "playwright dashboard-settings regression: update incident",
	})
	parseIncidentUpdateCancel()
	if parseErr != nil {
		parseT.Fatalf("UpdateAdminIncident: %v", parseErr)
	}
	if parseIncidentUpdateResp.GetIncident() == nil || parseIncidentUpdateResp.GetIncident().GetStatus() != "investigating" {
		parseT.Fatalf("UpdateAdminIncident returned invalid row: %+v", parseIncidentUpdateResp)
	}
}

// parseAssertExample100DashboardSettingsRows verifies slice/control-plane reads reflect recent settings mutations.
func parseAssertExample100DashboardSettingsRows(parseT *testing.T, parseClient chatpb.ChatServiceClient, parsePlanCode string, parseQuotaKey string, parseFlagKey string, parseIncidentKey string) {
	parseT.Helper()

	parseSlicesCtx, parseSlicesCancel := parseBuildExample100AdminMutationCallContext()
	parseSlicesResp, parseErr := parseClient.GetSuperuserSlices(parseSlicesCtx, &chatpb.GetSuperuserSlicesRequest{
		LookbackDays: 30,
		Limit:        300,
		IncidentListQuery: &chatpb.AdminListQuery{
			Limit:         25,
			Offset:        0,
			Search:        parseIncidentKey,
			SortBy:        "incident_key",
			SortDirection: "asc",
		},
		IncidentStatus: "investigating",
	})
	parseSlicesCancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices: %v", parseErr)
	}
	if !parseHasExample100BillingPlanOverage(parseSlicesResp.GetBillingPlanOverages(), parsePlanCode, "tokens") {
		parseT.Fatalf("expected billing overage row for plan=%q meter=tokens in slices response", parsePlanCode)
	}
	if !parseHasExample100BillingQuotaPolicy(parseSlicesResp.GetBillingQuotaPolicies(), parsePlanCode, parseQuotaKey) {
		parseT.Fatalf("expected billing quota row for plan=%q quota=%q in slices response", parsePlanCode, parseQuotaKey)
	}
	if !parseHasExample100IncidentStatus(parseSlicesResp.GetIncidents(), parseIncidentKey, "investigating") {
		parseT.Fatalf("expected incident row for key=%q status=investigating in slices response", parseIncidentKey)
	}

	parseControlCtx, parseControlCancel := parseBuildExample100AdminMutationCallContext()
	parseControlResp, parseErr := parseClient.GetSuperuserControlPlane(parseControlCtx, &chatpb.GetSuperuserControlPlaneRequest{
		Limit: 300,
	})
	parseControlCancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserControlPlane: %v", parseErr)
	}
	if !parseHasExample100FeatureFlag(parseControlResp.GetFeatureFlags(), parseFlagKey, true) {
		parseT.Fatalf("expected feature flag row for flag=%q enabled=true in control-plane response", parseFlagKey)
	}
}

// parseAssertExample100DashboardSettingsRoutes verifies settings-surface route state persists across refresh/back/forward after mutations.
func parseAssertExample100DashboardSettingsRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parsePlanCode string, parseFlagKey string, parseIncidentKey string) {
	parseT.Helper()
	parseRoutes := []string{
		fmt.Sprintf("/app/dashboard/usage?tab=billing&lookback_days=30&search=%s&sort_by=plan_code&sort_direction=asc&limit=5&offset=0", parsePlanCode),
		fmt.Sprintf("/app/dashboard/usage?tab=ops&lookback_days=30&search=%s&sort_by=incident_key&sort_direction=asc&limit=5&offset=0", parseIncidentKey),
		fmt.Sprintf("/app/dashboard?tab=ops-settings&lookback_days=30&search=%s", parseFlagKey),
	}
	for _, parseRoute := range parseRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto dashboard settings route %s: %v", parseRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait dashboard settings route %s: %v", parseRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on dashboard settings route %s: %v", parseRoute, parseErr)
		}
	}
	parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, parseRoutes)
}

// parseHasExample100BillingPlanOverage reports whether one billing-plan overage row exists for the supplied plan and meter.
func parseHasExample100BillingPlanOverage(parseRows []*chatpb.BillingPlanOverageEntry, parsePlanCode string, parseMeterKey string) bool {
	for _, parseRow := range parseRows {
		if parseRow == nil {
			continue
		}
		if parseRow.GetPlanCode() == parsePlanCode && parseRow.GetMeterKey() == parseMeterKey {
			return true
		}
	}
	return false
}

// parseHasExample100BillingQuotaPolicy reports whether one billing quota-policy row exists for the supplied plan and quota key.
func parseHasExample100BillingQuotaPolicy(parseRows []*chatpb.BillingQuotaPolicyEntry, parsePlanCode string, parseQuotaKey string) bool {
	for _, parseRow := range parseRows {
		if parseRow == nil {
			continue
		}
		if parseRow.GetPlanCode() == parsePlanCode && parseRow.GetQuotaKey() == parseQuotaKey {
			return true
		}
	}
	return false
}

// parseHasExample100IncidentStatus reports whether one incident row exists for the supplied incident key and status.
func parseHasExample100IncidentStatus(parseRows []*chatpb.IncidentEntry, parseIncidentKey string, parseStatus string) bool {
	for _, parseRow := range parseRows {
		if parseRow == nil {
			continue
		}
		if parseRow.GetIncidentKey() == parseIncidentKey && parseRow.GetStatus() == parseStatus {
			return true
		}
	}
	return false
}

// parseHasExample100FeatureFlag reports whether one feature-flag row exists for the supplied flag key and enabled state.
func parseHasExample100FeatureFlag(parseRows []*chatpb.FeatureFlagEntry, parseFlagKey string, isParseEnabled bool) bool {
	for _, parseRow := range parseRows {
		if parseRow == nil {
			continue
		}
		if parseRow.GetFlagKey() == parseFlagKey && parseRow.GetIsEnabled() == isParseEnabled {
			return true
		}
	}
	return false
}
