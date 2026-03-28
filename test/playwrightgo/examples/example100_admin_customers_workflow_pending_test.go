//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	playwright "github.com/playwright-community/playwright-go"
)

// TestExample100AdminCustomersWorkflowRegression validates the customers workflow from filtered search through detail views and disable/suspend restore actions.
func TestExample100AdminCustomersWorkflowRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100AdminGuardServer(parseT, parseRepoRoot, "18117")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)
		parseTargetUserID, parseTargetWorkspaceID := parseResolveExample100AdminMutationTargetIDs(parseT, parseBaseURL, parseSuperuserToken)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseSearchCtx, parseSearchCancel := parseBuildExample100AdminMutationCallContext()
		parseSearchResp, parseErr := parseClient.SearchAdminUsers(parseSearchCtx, &chatpb.SearchAdminUsersRequest{
			Query: example100AdminGuardWorkspaceEmail,
			Limit: 20,
		})
		parseSearchCancel()
		if parseErr != nil {
			parseT.Fatalf("SearchAdminUsers customers workflow: %v", parseErr)
		}
		if len(parseSearchResp.GetUsers()) == 0 {
			parseT.Fatalf("expected filtered customer search rows for %q, got none", example100AdminGuardWorkspaceEmail)
		}

		parseUserDetailCtx, parseUserDetailCancel := parseBuildExample100AdminMutationCallContext()
		parseUserDetailResp, parseErr := parseClient.GetAdminUserDetail(parseUserDetailCtx, &chatpb.GetAdminUserDetailRequest{
			UserId:       parseTargetUserID,
			LookbackDays: 30,
			Limit:        25,
		})
		parseUserDetailCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminUserDetail customers workflow: %v", parseErr)
		}
		if parseUserDetailResp.GetDetail() == nil || parseUserDetailResp.GetDetail().GetUser() == nil || parseUserDetailResp.GetDetail().GetUser().GetUserId() != parseTargetUserID {
			parseT.Fatalf("GetAdminUserDetail returned invalid detail payload: %+v", parseUserDetailResp)
		}

		parseWorkspaceDetailCtx, parseWorkspaceDetailCancel := parseBuildExample100AdminMutationCallContext()
		parseWorkspaceDetailResp, parseErr := parseClient.GetAdminWorkspaceDetail(parseWorkspaceDetailCtx, &chatpb.GetAdminWorkspaceDetailRequest{
			WorkspaceId:  parseTargetWorkspaceID,
			LookbackDays: 30,
			Limit:        25,
		})
		parseWorkspaceDetailCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminWorkspaceDetail customers workflow: %v", parseErr)
		}
		if parseWorkspaceDetailResp.GetDetail() == nil || parseWorkspaceDetailResp.GetDetail().GetWorkspace() == nil || parseWorkspaceDetailResp.GetDetail().GetWorkspace().GetId() != parseTargetWorkspaceID {
			parseT.Fatalf("GetAdminWorkspaceDetail returned invalid detail payload: %+v", parseWorkspaceDetailResp)
		}

		parseDisableCtx, parseDisableCancel := parseBuildExample100AdminMutationCallContext()
		parseDisableResp, parseErr := parseClient.DisableAdminUser(parseDisableCtx, &chatpb.AdminUserMutationRequest{
			UserId:  parseTargetUserID,
			Confirm: true,
			Reason:  "playwright customers workflow regression: disable user",
		})
		parseDisableCancel()
		if parseErr != nil {
			parseT.Fatalf("DisableAdminUser customers workflow: %v", parseErr)
		}
		if parseDisableResp.GetStatus() != "disabled" {
			parseT.Fatalf("DisableAdminUser status=%q want=disabled", parseDisableResp.GetStatus())
		}

		parseSearchAfterDisableCtx, parseSearchAfterDisableCancel := parseBuildExample100AdminMutationCallContext()
		parseSearchAfterDisableResp, parseErr := parseClient.SearchAdminUsers(parseSearchAfterDisableCtx, &chatpb.SearchAdminUsersRequest{
			Query: example100AdminGuardWorkspaceEmail,
			Limit: 20,
		})
		parseSearchAfterDisableCancel()
		if parseErr != nil {
			parseT.Fatalf("SearchAdminUsers after disable: %v", parseErr)
		}
		if len(parseSearchAfterDisableResp.GetUsers()) == 0 {
			parseT.Fatalf("expected filtered customer list rows after disable, got none")
		}

		parseRestoreUserCtx, parseRestoreUserCancel := parseBuildExample100AdminMutationCallContext()
		parseRestoreUserResp, parseErr := parseClient.RestoreAdminUser(parseRestoreUserCtx, &chatpb.AdminUserMutationRequest{
			UserId:  parseTargetUserID,
			Confirm: true,
			Reason:  "playwright customers workflow regression: restore user",
		})
		parseRestoreUserCancel()
		if parseErr != nil {
			parseT.Fatalf("RestoreAdminUser customers workflow: %v", parseErr)
		}
		if parseRestoreUserResp.GetStatus() != "active" {
			parseT.Fatalf("RestoreAdminUser status=%q want=active", parseRestoreUserResp.GetStatus())
		}

		parseSuspendCtx, parseSuspendCancel := parseBuildExample100AdminMutationCallContext()
		parseSuspendResp, parseErr := parseClient.SuspendAdminWorkspace(parseSuspendCtx, &chatpb.AdminWorkspaceMutationRequest{
			WorkspaceId: parseTargetWorkspaceID,
			Confirm:     true,
			Reason:      "playwright customers workflow regression: suspend workspace",
		})
		parseSuspendCancel()
		if parseErr != nil {
			parseT.Fatalf("SuspendAdminWorkspace customers workflow: %v", parseErr)
		}
		if strings.TrimSpace(parseSuspendResp.GetStatus()) != "suspended" {
			parseT.Fatalf("SuspendAdminWorkspace status=%q want=suspended", parseSuspendResp.GetStatus())
		}

		parseWorkspaceAfterSuspendCtx, parseWorkspaceAfterSuspendCancel := parseBuildExample100AdminMutationCallContext()
		parseWorkspaceAfterSuspendResp, parseErr := parseClient.GetAdminWorkspaceDetail(parseWorkspaceAfterSuspendCtx, &chatpb.GetAdminWorkspaceDetailRequest{
			WorkspaceId:  parseTargetWorkspaceID,
			LookbackDays: 30,
			Limit:        25,
		})
		parseWorkspaceAfterSuspendCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminWorkspaceDetail after suspend: %v", parseErr)
		}
		if parseWorkspaceAfterSuspendResp.GetDetail() == nil || parseWorkspaceAfterSuspendResp.GetDetail().GetWorkspace() == nil || parseWorkspaceAfterSuspendResp.GetDetail().GetWorkspace().GetStatus() != "suspended" {
			parseT.Fatalf("expected suspended workspace detail after suspend, got %+v", parseWorkspaceAfterSuspendResp.GetDetail())
		}

		parseRestoreWorkspaceCtx, parseRestoreWorkspaceCancel := parseBuildExample100AdminMutationCallContext()
		parseRestoreWorkspaceResp, parseErr := parseClient.RestoreAdminWorkspace(parseRestoreWorkspaceCtx, &chatpb.AdminWorkspaceMutationRequest{
			WorkspaceId:             parseTargetWorkspaceID,
			RestoreApiKeys:          true,
			RestoreWebhookEndpoints: true,
			RestoreBackgroundJobs:   true,
			Confirm:                 true,
			Reason:                  "playwright customers workflow regression: restore workspace",
		})
		parseRestoreWorkspaceCancel()
		if parseErr != nil {
			parseT.Fatalf("RestoreAdminWorkspace customers workflow: %v", parseErr)
		}
		if strings.TrimSpace(parseRestoreWorkspaceResp.GetStatus()) != "active" {
			parseT.Fatalf("RestoreAdminWorkspace status=%q want=active", parseRestoreWorkspaceResp.GetStatus())
		}

		parseWorkspaceAfterRestoreCtx, parseWorkspaceAfterRestoreCancel := parseBuildExample100AdminMutationCallContext()
		parseWorkspaceAfterRestoreResp, parseErr := parseClient.GetAdminWorkspaceDetail(parseWorkspaceAfterRestoreCtx, &chatpb.GetAdminWorkspaceDetailRequest{
			WorkspaceId:  parseTargetWorkspaceID,
			LookbackDays: 30,
			Limit:        25,
		})
		parseWorkspaceAfterRestoreCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminWorkspaceDetail after restore: %v", parseErr)
		}
		if parseWorkspaceAfterRestoreResp.GetDetail() == nil || parseWorkspaceAfterRestoreResp.GetDetail().GetWorkspace() == nil || parseWorkspaceAfterRestoreResp.GetDetail().GetWorkspace().GetStatus() != "active" {
			parseT.Fatalf("expected active workspace detail after restore, got %+v", parseWorkspaceAfterRestoreResp.GetDetail())
		}

		parseAssertExample100AdminCustomersWorkflowRoutes(parseT, parsePage, parseBaseURL, parseTargetUserID, parseTargetWorkspaceID)
		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-customers workflow regression flow", parseEvidence)
	})
}

// parseAssertExample100AdminCustomersWorkflowRoutes verifies filtered customer/workspace list context persists across detail route transitions and navigation replay.
func parseAssertExample100AdminCustomersWorkflowRoutes(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseUserID int64, parseWorkspaceID int64) {
	parseT.Helper()
	parseRoutes := []string{
		"/app/admin/users?tab=customers&lookback_days=30&search=workspace-admin%40example.com&sort_by=email&sort_direction=asc&limit=20&offset=0",
		fmt.Sprintf("/app/admin/users?tab=customers&lookback_days=30&search=workspace-admin%%40example.com&sort_by=email&sort_direction=asc&limit=20&offset=0&view=user-detail&user_id=%d", parseUserID),
		"/app/dashboard/usage?tab=workspaces&lookback_days=30&search=ws-admin-guard-empty&sort_by=workspace_key&sort_direction=asc&limit=20&offset=0",
		fmt.Sprintf("/app/dashboard/usage?tab=workspaces&lookback_days=30&search=ws-admin-guard-empty&sort_by=workspace_key&sort_direction=asc&limit=20&offset=0&view=workspace-detail&workspace_id=%d", parseWorkspaceID),
	}
	for _, parseRoute := range parseRoutes {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto customers workflow route %s: %v", parseRoute, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRoute)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait customers workflow route %s: %v", parseRoute, parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait chat input on customers workflow route %s: %v", parseRoute, parseErr)
		}
	}
	parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, parseRoutes)
}
