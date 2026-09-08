//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	playwright "github.com/mxschmitt/playwright-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const example100AdminMutationWorkspaceKey = "ws-admin-guard-empty"

// TestExample100AdminMutationRegression covers disable/restore user and suspend/restore workspace flows with browser auth-shell transitions.
func TestExample100AdminMutationRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100AdminGuardServer(parseT, parseRepoRoot, "18109")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseTargetUserID, parseTargetWorkspaceID := parseResolveExample100AdminMutationTargetIDs(parseT, parseBaseURL, parseSuperuserToken)

		parseClearExample100AdminMutationAuthState(parseT, parsePage)
		loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminGuardWorkspaceEmail, example100AdminGuardWorkspacePassword)

		parseMutationConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseMutationConn.Close()
		parseMutationClient := chatpb.NewChatServiceClient(parseMutationConn)

		parseDisableCtx, parseDisableCancel := parseBuildExample100AdminMutationCallContext()
		parseDisableResp, parseErr := parseMutationClient.DisableAdminUser(parseDisableCtx, &chatpb.AdminUserMutationRequest{
			UserId:  parseTargetUserID,
			Confirm: true,
			Reason:  "playwright regression: disable target user",
		})
		parseDisableCancel()
		if parseErr != nil {
			parseT.Fatalf("DisableAdminUser: %v", parseErr)
		}
		if strings.TrimSpace(strings.ToLower(parseDisableResp.GetStatus())) != "disabled" {
			parseT.Fatalf("DisableAdminUser status=%q want=disabled", parseDisableResp.GetStatus())
		}
		parseAssertExample100AdminMutationAuthRoute(parseT, parsePage, parseBaseURL, "disable-user transition")

		parseRestoreUserCtx, parseRestoreUserCancel := parseBuildExample100AdminMutationCallContext()
		parseRestoreUserResp, parseErr := parseMutationClient.RestoreAdminUser(parseRestoreUserCtx, &chatpb.AdminUserMutationRequest{
			UserId:  parseTargetUserID,
			Confirm: true,
			Reason:  "playwright regression: restore target user",
		})
		parseRestoreUserCancel()
		if parseErr != nil {
			parseT.Fatalf("RestoreAdminUser: %v", parseErr)
		}
		if strings.TrimSpace(strings.ToLower(parseRestoreUserResp.GetStatus())) != "active" {
			parseT.Fatalf("RestoreAdminUser status=%q want=active", parseRestoreUserResp.GetStatus())
		}
		loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminGuardWorkspaceEmail, example100AdminGuardWorkspacePassword)

		parseSuspendCtx, parseSuspendCancel := parseBuildExample100AdminMutationCallContext()
		parseSuspendResp, parseErr := parseMutationClient.SuspendAdminWorkspace(parseSuspendCtx, &chatpb.AdminWorkspaceMutationRequest{
			WorkspaceId: parseTargetWorkspaceID,
			Confirm:     true,
			Reason:      "playwright regression: suspend target workspace",
		})
		parseSuspendCancel()
		if parseErr != nil {
			parseT.Fatalf("SuspendAdminWorkspace: %v", parseErr)
		}
		if strings.TrimSpace(strings.ToLower(parseSuspendResp.GetStatus())) != "suspended" {
			parseT.Fatalf("SuspendAdminWorkspace status=%q want=suspended", parseSuspendResp.GetStatus())
		}
		parseAssertExample100AdminMutationAuthRoute(parseT, parsePage, parseBaseURL, "suspend-workspace transition")

		parseRestoreWorkspaceCtx, parseRestoreWorkspaceCancel := parseBuildExample100AdminMutationCallContext()
		parseRestoreWorkspaceResp, parseErr := parseMutationClient.RestoreAdminWorkspace(parseRestoreWorkspaceCtx, &chatpb.AdminWorkspaceMutationRequest{
			WorkspaceId:             parseTargetWorkspaceID,
			Confirm:                 true,
			Reason:                  "playwright regression: restore target workspace",
			RestoreApiKeys:          true,
			RestoreWebhookEndpoints: true,
			RestoreBackgroundJobs:   true,
		})
		parseRestoreWorkspaceCancel()
		if parseErr != nil {
			parseT.Fatalf("RestoreAdminWorkspace: %v", parseErr)
		}
		if strings.TrimSpace(strings.ToLower(parseRestoreWorkspaceResp.GetStatus())) != "active" {
			parseT.Fatalf("RestoreAdminWorkspace status=%q want=active", parseRestoreWorkspaceResp.GetStatus())
		}
		loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminGuardWorkspaceEmail, example100AdminGuardWorkspacePassword)
		parseAssertExample100AdminMutationAppRoute(parseT, parsePage, "post-restore workspace login")

		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-mutation regression flow", parseEvidence)
	})
}

// parseOpenExample100AdminMutationConn opens one tunnel-backed gRPC connection with bearer metadata attached.
func parseOpenExample100AdminMutationConn(parseT *testing.T, parseBaseURL string, parseAuthToken string) *grpc.ClientConn {
	parseT.Helper()
	parseToken := strings.TrimSpace(parseAuthToken)
	if parseToken == "" {
		parseT.Fatalf("open mutation grpc conn: missing auth token")
	}
	parseDialCtx, parseCancelDial := context.WithTimeout(context.Background(), 12*time.Second)
	defer parseCancelDial()

	parseDialOptions := grpctunnel.ApplyTunnelInsecureCredentials([]grpc.DialOption{
		grpc.WithUnaryInterceptor(func(parseCtx context.Context, parseMethod string, parseReq interface{}, parseReply interface{}, parseConn *grpc.ClientConn, parseInvoker grpc.UnaryInvoker, parseOpts ...grpc.CallOption) error {
			parseAuthCtx := metadata.AppendToOutgoingContext(parseCtx, "authorization", "Bearer "+parseToken)
			return parseInvoker(parseAuthCtx, parseMethod, parseReq, parseReply, parseConn, parseOpts...)
		}),
		grpc.WithBlock(),
	})
	parseConn, parseErr := grpctunnel.BuildTunnelConn(parseDialCtx, grpctunnel.TunnelConfig{
		Target:      buildExample100AdminJourneyTunnelURL(parseBaseURL),
		GRPCOptions: parseDialOptions,
	})
	if parseErr != nil {
		parseT.Fatalf("open mutation grpc conn: %v", parseErr)
	}
	return parseConn
}

// parseResolveExample100AdminMutationTargetIDs resolves one deterministic target user/workspace pair for mutation assertions.
func parseResolveExample100AdminMutationTargetIDs(parseT *testing.T, parseBaseURL string, parseAuthToken string) (int64, int64) {
	parseT.Helper()
	parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseAuthToken)
	defer parseConn.Close()
	parseClient := chatpb.NewChatServiceClient(parseConn)

	parseSearchCtx, parseSearchCancel := parseBuildExample100AdminMutationCallContext()
	parseSearchResp, parseErr := parseClient.SearchAdminUsers(parseSearchCtx, &chatpb.SearchAdminUsersRequest{
		Query: example100AdminGuardWorkspaceEmail,
		Limit: 10,
	})
	parseSearchCancel()
	if parseErr != nil {
		parseT.Fatalf("SearchAdminUsers: %v", parseErr)
	}
	parseTargetUserID := int64(0)
	for _, parseUserRow := range parseSearchResp.GetUsers() {
		if strings.EqualFold(strings.TrimSpace(parseUserRow.GetEmail()), example100AdminGuardWorkspaceEmail) {
			parseTargetUserID = parseUserRow.GetUserId()
			break
		}
	}
	if parseTargetUserID <= 0 {
		parseT.Fatalf("resolve target user id: workspace-admin user not found in SearchAdminUsers response: %+v", parseSearchResp.GetUsers())
	}

	parseSlicesCtx, parseSlicesCancel := parseBuildExample100AdminMutationCallContext()
	parseSlicesResp, parseErr := parseClient.GetSuperuserSlices(parseSlicesCtx, &chatpb.GetSuperuserSlicesRequest{
		Limit: 200,
		WorkspaceListQuery: &chatpb.AdminListQuery{
			Limit:  200,
			Search: example100AdminMutationWorkspaceKey,
		},
	})
	parseSlicesCancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices: %v", parseErr)
	}
	parseTargetWorkspaceID := int64(0)
	for _, parseWorkspaceRow := range parseSlicesResp.GetWorkspaces() {
		if strings.EqualFold(strings.TrimSpace(parseWorkspaceRow.GetWorkspaceKey()), example100AdminMutationWorkspaceKey) ||
			strings.EqualFold(strings.TrimSpace(parseWorkspaceRow.GetSlug()), example100AdminMutationWorkspaceKey) {
			parseTargetWorkspaceID = parseWorkspaceRow.GetId()
			break
		}
	}
	if parseTargetWorkspaceID <= 0 {
		parseT.Fatalf("resolve target workspace id: workspace %q not found in superuser slices: %+v", example100AdminMutationWorkspaceKey, parseSlicesResp.GetWorkspaces())
	}
	return parseTargetUserID, parseTargetWorkspaceID
}

// parseBuildExample100AdminMutationCallContext builds one short-lived RPC context for mutation operations.
func parseBuildExample100AdminMutationCallContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// parseClearExample100AdminMutationAuthState clears local auth token and cookies so the next login flow starts from the auth shell.
func parseClearExample100AdminMutationAuthState(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	if parseErr := parsePage.Context().ClearCookies(); parseErr != nil {
		parseT.Fatalf("clear auth cookies: %v", parseErr)
	}
	if _, parseErr := parsePage.Evaluate(
		fmt.Sprintf(`() => { window.localStorage.removeItem(%q); window.sessionStorage.clear(); }`, example100AdminJourneyAuthTokenStorageKey),
	); parseErr != nil {
		parseT.Fatalf("clear auth storage: %v", parseErr)
	}
}

// parseAssertExample100AdminMutationAuthRoute asserts that the auth shell is visible after one mutation-driven revocation transition.
func parseAssertExample100AdminMutationAuthRoute(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseLabel string) {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("%s goto /app: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => {
			const path = String(window.location.pathname || "");
			const hasAuthInput = !!document.querySelector("#auth-email-input");
			const hasLandingLogin = !!document.querySelector("#landing-login-link");
			const hasLandingSignup = !!document.querySelector("#landing-signup-link");
			if (hasAuthInput || hasLandingLogin || hasLandingSignup) {
				return true;
			}
			return path === "/" || path === "/home" || path === "/pricing" || path === "/signup";
		}`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("%s wait auth/public transition: %v", parseLabel, parseErr)
	}
}

// parseAssertExample100AdminMutationAppRoute asserts that the authenticated app shell is visible after restore and login.
func parseAssertExample100AdminMutationAppRoute(parseT *testing.T, parsePage playwright.Page, parseLabel string) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("%s wait chat input: %v", parseLabel, parseErr)
	}
}
