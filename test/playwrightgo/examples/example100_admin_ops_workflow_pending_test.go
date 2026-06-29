//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	playwright "github.com/playwright-community/playwright-go"
)

// TestExample100AdminOpsWorkflowRegression validates the ops workflow from incident summary through queue detail, action, audit confirmation, and queue-context restoration.
func TestExample100AdminOpsWorkflowRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseFixture := startExample100AdminOpsServer(parseT, parseRepoRoot, "18120")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseSLOKey := fmt.Sprintf("ops-workflow-slo-%d", time.Now().UTC().UnixNano())
		parseSLOCtx, parseSLOCancel := parseBuildExample100AdminMutationCallContext()
		_, parseErr := parseClient.SetSuperuserServiceLevelObjective(parseSLOCtx, &chatpb.SetSuperuserServiceLevelObjectiveRequest{
			SloKey:             parseSLOKey,
			ServiceName:        "ops-workflow-regression",
			ObjectivePercent:   99.9,
			WindowDays:         30,
			ErrorBudgetMinutes: 43,
			StatusPageUrl:      "https://status.example.invalid/ops-workflow-regression",
			Confirm:            true,
			Reason:             "playwright ops workflow regression: seed slo",
		})
		parseSLOCancel()
		if parseErr != nil {
			parseT.Fatalf("SetSuperuserServiceLevelObjective ops workflow: %v", parseErr)
		}

		parseIncidentKey := fmt.Sprintf("ops-workflow-incident-%d", time.Now().UTC().UnixNano())
		parseIncidentSetCtx, parseIncidentSetCancel := parseBuildExample100AdminMutationCallContext()
		parseIncidentSetResp, parseErr := parseClient.SetSuperuserIncident(parseIncidentSetCtx, &chatpb.SetSuperuserIncidentRequest{
			IncidentKey: parseIncidentKey,
			SloKey:      parseSLOKey,
			Severity:    "major",
			Status:      "open",
			Title:       "Ops workflow regression incident",
			Summary:     "Seeded incident for ops queue/replay workflow checks",
			StartedAt:   time.Now().UTC().Format(time.RFC3339),
			Confirm:     true,
			Reason:      "playwright ops workflow regression: seed incident",
		})
		parseIncidentSetCancel()
		if parseErr != nil {
			parseT.Fatalf("SetSuperuserIncident ops workflow: %v", parseErr)
		}
		if parseIncidentSetResp.GetIncident() == nil || parseIncidentSetResp.GetIncident().GetId() <= 0 {
			parseT.Fatalf("SetSuperuserIncident returned invalid incident payload: %+v", parseIncidentSetResp)
		}
		parseIncidentID := parseIncidentSetResp.GetIncident().GetId()

		parseSummaryCtx, parseSummaryCancel := parseBuildExample100AdminMutationCallContext()
		parseSummaryResp, parseErr := parseClient.GetSuperuserSlices(parseSummaryCtx, &chatpb.GetSuperuserSlicesRequest{
			LookbackDays: 30,
			Limit:        100,
			IncidentListQuery: &chatpb.AdminListQuery{
				Limit:         20,
				Offset:        0,
				Search:        parseIncidentKey,
				SortBy:        "incident_key",
				SortDirection: "asc",
			},
			IncidentStatus: "open",
		})
		parseSummaryCancel()
		if parseErr != nil {
			parseT.Fatalf("GetSuperuserSlices incident summary: %v", parseErr)
		}
		if len(parseSummaryResp.GetIncidents()) == 0 {
			parseT.Fatalf("expected incident queue rows for ops workflow summary, got none")
		}

		parseDetailCtx, parseDetailCancel := parseBuildExample100AdminMutationCallContext()
		parseDetailResp, parseErr := parseClient.GetAdminIncidentBlastRadius(parseDetailCtx, &chatpb.GetAdminIncidentBlastRadiusRequest{
			WorkspaceId:  parseFixture.GetTargetWorkspaceID,
			LookbackDays: 30,
		})
		parseDetailCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminIncidentBlastRadius ops workflow detail: %v", parseErr)
		}
		if parseDetailResp.GetBlastRadius() == nil || parseDetailResp.GetBlastRadius().GetWorkspaceId() != parseFixture.GetTargetWorkspaceID {
			parseT.Fatalf("GetAdminIncidentBlastRadius returned invalid detail payload: %+v", parseDetailResp)
		}

		parseActionCtx, parseActionCancel := parseBuildExample100AdminMutationCallContext()
		parseActionResp, parseErr := parseClient.UpdateAdminIncident(parseActionCtx, &chatpb.UpdateAdminIncidentRequest{
			IncidentId:  parseIncidentID,
			WorkspaceId: parseFixture.GetTargetWorkspaceID,
			Status:      "mitigating",
			Message:     "playwright ops workflow regression replay action",
			IsPublic:    false,
			Confirm:     true,
			Reason:      "playwright ops workflow regression: incident replay action",
		})
		parseActionCancel()
		if parseErr != nil {
			parseT.Fatalf("UpdateAdminIncident ops workflow action: %v", parseErr)
		}
		if parseActionResp.GetIncident() == nil || parseActionResp.GetIncident().GetStatus() != "mitigating" {
			parseT.Fatalf("UpdateAdminIncident returned invalid incident payload: %+v", parseActionResp)
		}

		parseAuditCtx, parseAuditCancel := parseBuildExample100AdminMutationCallContext()
		parseAuditResp, parseErr := parseClient.GetSuperuserControlPlane(parseAuditCtx, &chatpb.GetSuperuserControlPlaneRequest{
			Limit: 250,
		})
		parseAuditCancel()
		if parseErr != nil {
			parseT.Fatalf("GetSuperuserControlPlane audit confirmation: %v", parseErr)
		}
		parseIncidentTargetID := strconv.FormatInt(parseIncidentID, 10)
		if !parseHasExample100IncidentAuditRow(parseAuditResp.GetAuditLogs(), parseIncidentTargetID) {
			parseT.Fatalf("expected incident update audit row for incident id=%s after replay action", parseIncidentTargetID)
		}

		parseAssertExample100AdminOpsWorkflowRoutes(parseT, parsePage, parseBaseURL, parseIncidentKey, parseIncidentID, parseFixture.GetTargetWorkspaceID)
		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-ops workflow regression flow", parseEvidence)
	})
}

// parseHasExample100IncidentAuditRow reports whether one incident-update audit row exists for the supplied incident target id.
func parseHasExample100IncidentAuditRow(parseRows []*chatpb.AuditLogEntry, parseIncidentTargetID string) bool {
	for _, parseRow := range parseRows {
		if parseRow == nil {
			continue
		}
		if parseRow.GetEventType() == "admin.control.incident.update" && strings.TrimSpace(parseRow.GetTargetId()) == strings.TrimSpace(parseIncidentTargetID) {
			return true
		}
	}
	return false
}

// parseAssertExample100AdminOpsWorkflowRoutes verifies queue/time-range route state persists across detail and replay-action transitions.
func parseAssertExample100AdminOpsWorkflowRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseIncidentKey string, parseIncidentID int64, parseWorkspaceID int64) {
	parseT.Helper()
	parseQueueRoute := fmt.Sprintf("/app/dashboard/usage?tab=ops&lookback_days=30&status=open&search=%s&sort_by=incident_key&sort_direction=asc&limit=20&offset=0", parseIncidentKey)
	parseRoutes := []string{
		parseQueueRoute,
		fmt.Sprintf("%s&view=queue-detail&incident_id=%d", parseQueueRoute, parseIncidentID),
		fmt.Sprintf("%s&view=replay-action&incident_id=%d&workspace_id=%d", parseQueueRoute, parseIncidentID, parseWorkspaceID),
		parseQueueRoute,
	}
	for _, parseRoute := range parseRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto ops workflow route %s: %v", parseRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait ops workflow route %s: %v", parseRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on ops workflow route %s: %v", parseRoute, parseErr)
		}
	}
	parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, parseRoutes)
}
