//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	playwright "github.com/playwright-community/playwright-go"
)

// TestExample100AdminBusinessWorkflowRegression validates the business workflow from revenue summary through failed-payment intervention and filtered-queue route restoration.
func TestExample100AdminBusinessWorkflowRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseFixture := startExample100AdminOpsServer(parseT, parseRepoRoot, "18116")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseDashboardCtx, parseDashboardCancel := parseBuildExample100AdminMutationCallContext()
		parseDashboardResp, parseErr := parseClient.GetAdminDashboard(parseDashboardCtx, &chatpb.GetAdminDashboardRequest{
			LookbackDays: 30,
			TopLimit:     8,
			RecentLimit:  8,
		})
		parseDashboardCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminDashboard business summary: %v", parseErr)
		}
		if parseDashboardResp.GetSummary() == nil {
			parseT.Fatalf("GetAdminDashboard returned nil summary in business workflow")
		}
		if parseDashboardResp.GetSummary().GetWindowOpenInvoices() <= 0 && parseDashboardResp.GetSummary().GetWindowOpenDunningEvents() <= 0 {
			parseT.Fatalf("expected open invoice or dunning pressure in business summary, got summary=%+v", parseDashboardResp.GetSummary())
		}

		parseQueueCtx, parseQueueCancel := parseBuildExample100AdminMutationCallContext()
		parseQueueResp, parseErr := parseClient.ListAdminBillingDunningEvents(parseQueueCtx, &chatpb.ListAdminBillingDunningEventsRequest{
			UserId: parseFixture.GetTargetUserID,
			Status: "pending",
			ListQuery: &chatpb.AdminListQuery{
				Limit:         20,
				Offset:        0,
				SortBy:        "created_at",
				SortDirection: "desc",
			},
		})
		parseQueueCancel()
		if parseErr != nil {
			parseT.Fatalf("ListAdminBillingDunningEvents pending queue: %v", parseErr)
		}
		if len(parseQueueResp.GetEvents()) == 0 {
			parseT.Fatalf("expected failed-payment queue rows for business workflow, got none")
		}
		parseInvoiceID := parseQueueResp.GetEvents()[0].GetInvoiceId()
		if parseInvoiceID <= 0 {
			parseT.Fatalf("failed-payment queue row missing invoice id: %+v", parseQueueResp.GetEvents()[0])
		}

		parseDetailCtx, parseDetailCancel := parseBuildExample100AdminMutationCallContext()
		parseDetailResp, parseErr := parseClient.ListAdminBillingEvents(parseDetailCtx, &chatpb.ListAdminBillingEventsRequest{
			UserId: parseFixture.GetTargetUserID,
			Limit:  20,
			ListQuery: &chatpb.AdminListQuery{
				Limit:         20,
				Offset:        0,
				SortBy:        "created_at",
				SortDirection: "desc",
			},
		})
		parseDetailCancel()
		if parseErr != nil {
			parseT.Fatalf("ListAdminBillingEvents customer detail: %v", parseErr)
		}
		if len(parseDetailResp.GetEvents()) == 0 {
			parseT.Fatalf("expected billing event rows for customer billing detail, got none")
		}

		parseOverrideKey := fmt.Sprintf("business-workflow-override-%d", time.Now().UTC().UnixNano())
		parseOverrideCtx, parseOverrideCancel := parseBuildExample100AdminMutationCallContext()
		parseOverrideResp, parseErr := parseClient.SetAdminBillingAccessOverride(parseOverrideCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
			UserId:        parseFixture.GetTargetUserID,
			OverrideKey:   parseOverrideKey,
			OverrideValue: "enabled",
			Reason:        "playwright business workflow regression: access override",
			IsEnabled:     true,
			Confirm:       true,
		})
		parseOverrideCancel()
		if parseErr != nil {
			parseT.Fatalf("SetAdminBillingAccessOverride business workflow: %v", parseErr)
		}
		if strings.TrimSpace(parseOverrideResp.GetStatus()) == "" {
			parseT.Fatalf("SetAdminBillingAccessOverride returned blank status: %+v", parseOverrideResp)
		}

		parseResolveCtx, parseResolveCancel := parseBuildExample100AdminMutationCallContext()
		parseResolveResp, parseErr := parseClient.ResolveAdminBillingFailedPayment(parseResolveCtx, &chatpb.ResolveAdminBillingFailedPaymentRequest{
			UserId:    parseFixture.GetTargetUserID,
			InvoiceId: parseInvoiceID,
			Confirm:   true,
			Reason:    "playwright business workflow regression: resolve failed payment",
		})
		parseResolveCancel()
		if parseErr != nil {
			parseT.Fatalf("ResolveAdminBillingFailedPayment business workflow: %v", parseErr)
		}
		if strings.TrimSpace(parseResolveResp.GetInvoiceStatus()) == "" {
			parseT.Fatalf("ResolveAdminBillingFailedPayment returned blank invoice status: %+v", parseResolveResp)
		}

		parseQueueAfterCtx, parseQueueAfterCancel := parseBuildExample100AdminMutationCallContext()
		parseQueueAfterResp, parseErr := parseClient.ListAdminBillingDunningEvents(parseQueueAfterCtx, &chatpb.ListAdminBillingDunningEventsRequest{
			UserId: parseFixture.GetTargetUserID,
			Status: "pending",
			ListQuery: &chatpb.AdminListQuery{
				Limit:         20,
				Offset:        0,
				SortBy:        "created_at",
				SortDirection: "desc",
			},
		})
		parseQueueAfterCancel()
		if parseErr != nil {
			parseT.Fatalf("ListAdminBillingDunningEvents pending queue after resolve: %v", parseErr)
		}
		if len(parseQueueAfterResp.GetEvents()) != 0 {
			parseT.Fatalf("expected pending failed-payment queue to clear after resolve, got %d rows", len(parseQueueAfterResp.GetEvents()))
		}

		parseAssertExample100AdminBusinessWorkflowRoutes(parseT, parsePage, parseBaseURL, parseFixture.GetTargetUserID, parseInvoiceID)
		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-business workflow regression flow", parseEvidence)
	})
}

// parseAssertExample100AdminBusinessWorkflowRoutes verifies failed-payment queue route context persists across detail transitions and navigation replay.
func parseAssertExample100AdminBusinessWorkflowRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseUserID int64, parseInvoiceID int64) {
	parseT.Helper()
	parseRoutes := []string{
		"/app/dashboard/usage?tab=billing&lookback_days=30&status=pending&sort_by=created_at&sort_direction=desc&limit=20&offset=0",
		fmt.Sprintf("/app/dashboard/usage?tab=billing&lookback_days=30&status=pending&sort_by=created_at&sort_direction=desc&limit=20&offset=0&view=customer-billing-detail&user_id=%d&invoice_id=%d", parseUserID, parseInvoiceID),
		"/app/dashboard/usage?tab=billing&lookback_days=30&status=pending&sort_by=created_at&sort_direction=desc&limit=20&offset=0",
	}
	for _, parseRoute := range parseRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto business workflow route %s: %v", parseRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait business workflow route %s: %v", parseRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on business workflow route %s: %v", parseRoute, parseErr)
		}
	}
	parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, parseRoutes)
}
