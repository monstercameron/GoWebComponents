//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	playwright "github.com/playwright-community/playwright-go"
)

// TestExample100AdminListMechanicsRegression validates admin list-query mechanics and list-state navigation stability.
func TestExample100AdminListMechanicsRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseFixture := startExample100AdminListMechanicsServer(parseT, parseRepoRoot, "18111")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseBillingKeyA := fmt.Sprintf("list-mech-billing-a-%d", time.Now().UTC().UnixNano())
		parseBillingKeyB := fmt.Sprintf("list-mech-billing-b-%d", time.Now().UTC().UnixNano())
		parseCreateExample100BillingOverride(parseT, parseClient, parseFixture.GetTargetUserID, parseBillingKeyA)
		parseCreateExample100BillingOverride(parseT, parseClient, parseFixture.GetTargetUserID, parseBillingKeyB)

		parseSLOKey := fmt.Sprintf("list-mech-slo-%d", time.Now().UTC().UnixNano())
		parseSLOCtx, parseSLOCancel := parseBuildExample100AdminMutationCallContext()
		_, parseErr := parseClient.SetSuperuserServiceLevelObjective(parseSLOCtx, &chatpb.SetSuperuserServiceLevelObjectiveRequest{
			SloKey:             parseSLOKey,
			ServiceName:        "list-mechanics-regression",
			ObjectivePercent:   99.9,
			WindowDays:         30,
			ErrorBudgetMinutes: 43,
			StatusPageUrl:      "https://status.example.invalid/list-mechanics-regression",
			Confirm:            true,
			Reason:             "playwright list mechanics regression: seed slo",
		})
		parseSLOCancel()
		if parseErr != nil {
			parseT.Fatalf("SetSuperuserServiceLevelObjective: %v", parseErr)
		}
		parseCreateExample100Incident(parseT, parseClient, parseSLOKey, "list-mech-incident-a")
		parseCreateExample100Incident(parseT, parseClient, parseSLOKey, "list-mech-incident-b")

		parseAssertExample100AdminUserListMechanics(parseT, parseClient)
		parseAssertExample100AdminWorkspaceListMechanics(parseT, parseClient)
		parseAssertExample100AdminSupportListMechanics(parseT, parseClient)
		parseAssertExample100AdminIncidentListMechanics(parseT, parseClient)
		parseAssertExample100AdminBillingListMechanics(parseT, parseClient, parseFixture.GetTargetUserID)

		parseAssertExample100AdminListURLStateNavigation(parseT, parsePage, parseBaseURL, []string{
			"/app/admin/users?lookback_days=30&search=admin&sort_by=email&sort_direction=asc&limit=1&offset=1",
			"/app/dashboard/usage?lookback_days=30&tab=workspaces&search=ws-list-mech&sort_by=workspace_key&sort_direction=asc&limit=1&offset=1",
			"/app/dashboard/usage?lookback_days=30&tab=support&search=list-mech-support&sort_by=ticket_key&sort_direction=asc&limit=1&offset=1",
			"/app/dashboard/usage?lookback_days=30&tab=incidents&search=list-mech-incident&sort_by=incident_key&sort_direction=asc&limit=1&offset=1",
			"/app/dashboard/usage?lookback_days=30&tab=billing&search=list-mech-billing&sort_by=override_key&sort_direction=asc&limit=1&offset=1",
		})

		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-list-mechanics regression flow", parseEvidence)
	})
}

// startExample100AdminListMechanicsServer starts one seeded server with deterministic list-friendly fixtures for users/workspaces/support/billing slices.
func startExample100AdminListMechanicsServer(parseT *testing.T, parseRepoRoot string, parsePort string) (string, example100AdminOpsFixture) {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "admin_list_mechanics.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseBinaryPath := filepath.Join(parseRuntimeDir, "example100-admin-list-mechanics-server")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)

	seedExample100HappyPathDatabase(parseT, parseRepoRoot, parseDBPath)
	grantExample100AdminJourneySuperuserRole(parseT, parseDBPath)
	seedExample100AdminGuardWorkspaceAdminUser(parseT, parseDBPath)
	parseFixture := seedExample100AdminOpsFixtures(parseT, parseDBPath)
	seedExample100AdminListMechanicsFixtures(parseT, parseDBPath, parseFixture)
	buildExample100HappyPathServerBinary(parseT, parseRepoRoot, parseBinaryPath)

	parseStop := startExamplesCommandWithEnv(
		parseT,
		parseRepoRoot,
		[]string{
			"LISTEN_ADDR=" + parseAddress,
			"CHAT_DB_PATH=" + parseDBPath,
			"CHAT_LOG_DIR=" + parseLogDir,
		},
		parseBinaryPath,
	)
	parseT.Cleanup(parseStop)

	parseBaseURL := "http://" + parseAddress
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 120*time.Second)
	return parseBaseURL, parseFixture
}

// seedExample100AdminListMechanicsFixtures inserts deterministic workspace and support rows for list-query pagination/sort coverage.
func seedExample100AdminListMechanicsFixtures(parseT *testing.T, parseDBPath string, parseFixture example100AdminOpsFixture) {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open admin-list-mechanics sqlite db: %v", parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)

	parseNow := time.Now().UTC()
	parseNowRFC3339 := parseNow.Format(time.RFC3339)
	parseWorkspaceKeys := []string{"ws-list-mech-a", "ws-list-mech-b"}
	for _, parseWorkspaceKey := range parseWorkspaceKeys {
		parseWorkspaceResult, parseErr := parseDB.Exec(
			`INSERT INTO workspaces (workspace_key, slug, name, plan_code, status, owner_user_id, settings_json, created_at, updated_at)
			 VALUES (?, ?, ?, 'team', 'active', ?, '{}', ?, ?)`,
			parseWorkspaceKey,
			parseWorkspaceKey,
			"List Mechanics "+strings.ToUpper(strings.TrimPrefix(parseWorkspaceKey, "ws-list-mech-")),
			parseFixture.GetTargetUserID,
			parseNowRFC3339,
			parseNowRFC3339,
		)
		if parseErr != nil {
			parseT.Fatalf("insert list-mechanics workspace (%s): %v", parseWorkspaceKey, parseErr)
		}
		parseWorkspaceID, parseErr := parseWorkspaceResult.LastInsertId()
		if parseErr != nil {
			parseT.Fatalf("read list-mechanics workspace id (%s): %v", parseWorkspaceKey, parseErr)
		}
		if _, parseErr := parseDB.Exec(
			`INSERT INTO workspace_memberships (workspace_id, user_id, role_key, status, invited_by_user_id, created_at, updated_at)
			 VALUES (?, ?, 'admin', 'active', ?, ?, ?)`,
			parseWorkspaceID,
			parseFixture.GetTargetUserID,
			parseFixture.GetTargetUserID,
			parseNowRFC3339,
			parseNowRFC3339,
		); parseErr != nil {
			parseT.Fatalf("insert list-mechanics workspace membership (%s): %v", parseWorkspaceKey, parseErr)
		}
		if _, parseErr := parseDB.Exec(
			`INSERT INTO support_tickets (ticket_key, workspace_id, user_id, status, priority, subject, body, assignee_user_id, resolution_note, created_at, updated_at)
			 VALUES (?, ?, ?, 'open', 'normal', ?, ?, ?, '', ?, ?)`,
			fmt.Sprintf("ticket-%s-%d", parseWorkspaceKey, parseNow.UnixNano()),
			parseWorkspaceID,
			parseFixture.GetTargetUserID,
			"List mechanics support "+parseWorkspaceKey,
			"Seeded list-mechanics support ticket",
			1,
			parseNowRFC3339,
			parseNowRFC3339,
		); parseErr != nil {
			parseT.Fatalf("insert list-mechanics support ticket (%s): %v", parseWorkspaceKey, parseErr)
		}
	}
}

// parseCreateExample100BillingOverride stores one billing access override used for billing list-query pagination checks.
func parseCreateExample100BillingOverride(parseT *testing.T, parseClient chatpb.ChatServiceClient, parseUserID int64, parseOverrideKey string) {
	parseT.Helper()
	parseCtx, parseCancel := parseBuildExample100AdminMutationCallContext()
	parseResp, parseErr := parseClient.SetAdminBillingAccessOverride(parseCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseUserID,
		OverrideKey:   parseOverrideKey,
		OverrideValue: "enabled",
		Reason:        "playwright list mechanics regression: seed billing override",
		IsEnabled:     true,
		Confirm:       true,
	})
	parseCancel()
	if parseErr != nil {
		parseT.Fatalf("SetAdminBillingAccessOverride(%s): %v", parseOverrideKey, parseErr)
	}
	if strings.TrimSpace(parseResp.GetStatus()) == "" {
		parseT.Fatalf("SetAdminBillingAccessOverride(%s) returned blank status: %+v", parseOverrideKey, parseResp)
	}
}

// parseCreateExample100Incident stores one incident used for incident list-query pagination checks.
func parseCreateExample100Incident(parseT *testing.T, parseClient chatpb.ChatServiceClient, parseSLOKey string, parseIncidentSuffix string) {
	parseT.Helper()
	parseCtx, parseCancel := parseBuildExample100AdminMutationCallContext()
	parseResp, parseErr := parseClient.SetSuperuserIncident(parseCtx, &chatpb.SetSuperuserIncidentRequest{
		IncidentKey: fmt.Sprintf("%s-%d", parseIncidentSuffix, time.Now().UTC().UnixNano()),
		SloKey:      parseSLOKey,
		Severity:    "major",
		Status:      "open",
		Title:       "List mechanics incident " + parseIncidentSuffix,
		Summary:     "Seeded incident for list mechanics pagination checks",
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		Confirm:     true,
		Reason:      "playwright list mechanics regression: seed incident",
	})
	parseCancel()
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserIncident(%s): %v", parseIncidentSuffix, parseErr)
	}
	if parseResp.GetIncident() == nil || parseResp.GetIncident().GetId() <= 0 {
		parseT.Fatalf("SetSuperuserIncident(%s) returned invalid incident: %+v", parseIncidentSuffix, parseResp)
	}
}

// parseAssertExample100AdminUserListMechanics verifies typed user list search/sort/pagination behavior.
func parseAssertExample100AdminUserListMechanics(parseT *testing.T, parseClient chatpb.ChatServiceClient) {
	parseT.Helper()
	parsePage0Ctx, parsePage0Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage0Resp, parseErr := parseClient.ListAdminUsers(parsePage0Ctx, &chatpb.ListAdminUsersRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "admin",
			SortBy:        "email",
			SortDirection: "asc",
		},
	})
	parsePage0Cancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers page0: %v", parseErr)
	}
	parsePage1Ctx, parsePage1Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage1Resp, parseErr := parseClient.ListAdminUsers(parsePage1Ctx, &chatpb.ListAdminUsersRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "admin",
			SortBy:        "email",
			SortDirection: "asc",
		},
	})
	parsePage1Cancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers page1: %v", parseErr)
	}
	if len(parsePage0Resp.GetUsers()) == 0 || len(parsePage1Resp.GetUsers()) == 0 {
		parseT.Fatalf("expected paged user rows, got page0=%d page1=%d", len(parsePage0Resp.GetUsers()), len(parsePage1Resp.GetUsers()))
	}
	if parsePage0Resp.GetUsers()[0].GetUserId() == parsePage1Resp.GetUsers()[0].GetUserId() {
		parseT.Fatalf("expected distinct user rows across pagination pages, got same user id=%d", parsePage0Resp.GetUsers()[0].GetUserId())
	}
}

// parseAssertExample100AdminWorkspaceListMechanics verifies typed workspace slice search/sort/pagination behavior.
func parseAssertExample100AdminWorkspaceListMechanics(parseT *testing.T, parseClient chatpb.ChatServiceClient) {
	parseT.Helper()
	parsePage0Ctx, parsePage0Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage0Resp, parseErr := parseClient.GetSuperuserSlices(parsePage0Ctx, &chatpb.GetSuperuserSlicesRequest{
		Limit: 200,
		WorkspaceListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "ws-list-mech",
			SortBy:        "workspace_key",
			SortDirection: "asc",
		},
		WorkspaceStatus: "active",
	})
	parsePage0Cancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices workspace page0: %v", parseErr)
	}
	parsePage1Ctx, parsePage1Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage1Resp, parseErr := parseClient.GetSuperuserSlices(parsePage1Ctx, &chatpb.GetSuperuserSlicesRequest{
		Limit: 200,
		WorkspaceListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "ws-list-mech",
			SortBy:        "workspace_key",
			SortDirection: "asc",
		},
		WorkspaceStatus: "active",
	})
	parsePage1Cancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices workspace page1: %v", parseErr)
	}
	if len(parsePage0Resp.GetWorkspaces()) == 0 || len(parsePage1Resp.GetWorkspaces()) == 0 {
		parseT.Fatalf("expected paged workspace rows, got page0=%d page1=%d", len(parsePage0Resp.GetWorkspaces()), len(parsePage1Resp.GetWorkspaces()))
	}
	if parsePage0Resp.GetWorkspaces()[0].GetId() == parsePage1Resp.GetWorkspaces()[0].GetId() {
		parseT.Fatalf("expected distinct workspace rows across pagination pages, got same id=%d", parsePage0Resp.GetWorkspaces()[0].GetId())
	}
}

// parseAssertExample100AdminSupportListMechanics verifies typed support slice search/sort/pagination behavior.
func parseAssertExample100AdminSupportListMechanics(parseT *testing.T, parseClient chatpb.ChatServiceClient) {
	parseT.Helper()
	parsePage0Ctx, parsePage0Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage0Resp, parseErr := parseClient.GetSuperuserSlices(parsePage0Ctx, &chatpb.GetSuperuserSlicesRequest{
		Limit: 200,
		SupportListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "list-mechanics support",
			SortBy:        "ticket_key",
			SortDirection: "asc",
		},
		SupportStatus: "open",
	})
	parsePage0Cancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices support page0: %v", parseErr)
	}
	parsePage1Ctx, parsePage1Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage1Resp, parseErr := parseClient.GetSuperuserSlices(parsePage1Ctx, &chatpb.GetSuperuserSlicesRequest{
		Limit: 200,
		SupportListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "list-mechanics support",
			SortBy:        "ticket_key",
			SortDirection: "asc",
		},
		SupportStatus: "open",
	})
	parsePage1Cancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices support page1: %v", parseErr)
	}
	if len(parsePage0Resp.GetSupportTickets()) == 0 || len(parsePage1Resp.GetSupportTickets()) == 0 {
		parseT.Fatalf("expected paged support rows, got page0=%d page1=%d", len(parsePage0Resp.GetSupportTickets()), len(parsePage1Resp.GetSupportTickets()))
	}
	if parsePage0Resp.GetSupportTickets()[0].GetId() == parsePage1Resp.GetSupportTickets()[0].GetId() {
		parseT.Fatalf("expected distinct support rows across pagination pages, got same id=%d", parsePage0Resp.GetSupportTickets()[0].GetId())
	}
}

// parseAssertExample100AdminIncidentListMechanics verifies typed incident slice search/sort/pagination behavior.
func parseAssertExample100AdminIncidentListMechanics(parseT *testing.T, parseClient chatpb.ChatServiceClient) {
	parseT.Helper()
	parsePage0Ctx, parsePage0Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage0Resp, parseErr := parseClient.GetSuperuserSlices(parsePage0Ctx, &chatpb.GetSuperuserSlicesRequest{
		Limit: 200,
		IncidentListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "list-mech-incident",
			SortBy:        "incident_key",
			SortDirection: "asc",
		},
		IncidentStatus: "open",
	})
	parsePage0Cancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices incident page0: %v", parseErr)
	}
	parsePage1Ctx, parsePage1Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage1Resp, parseErr := parseClient.GetSuperuserSlices(parsePage1Ctx, &chatpb.GetSuperuserSlicesRequest{
		Limit: 200,
		IncidentListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "list-mech-incident",
			SortBy:        "incident_key",
			SortDirection: "asc",
		},
		IncidentStatus: "open",
	})
	parsePage1Cancel()
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices incident page1: %v", parseErr)
	}
	if len(parsePage0Resp.GetIncidents()) == 0 || len(parsePage1Resp.GetIncidents()) == 0 {
		parseT.Fatalf("expected paged incident rows, got page0=%d page1=%d", len(parsePage0Resp.GetIncidents()), len(parsePage1Resp.GetIncidents()))
	}
	if parsePage0Resp.GetIncidents()[0].GetId() == parsePage1Resp.GetIncidents()[0].GetId() {
		parseT.Fatalf("expected distinct incident rows across pagination pages, got same id=%d", parsePage0Resp.GetIncidents()[0].GetId())
	}
}

// parseAssertExample100AdminBillingListMechanics verifies typed billing list search/sort/pagination behavior.
func parseAssertExample100AdminBillingListMechanics(parseT *testing.T, parseClient chatpb.ChatServiceClient, parseUserID int64) {
	parseT.Helper()
	parsePage0Ctx, parsePage0Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage0Resp, parseErr := parseClient.ListAdminBillingAccessOverrides(parsePage0Ctx, &chatpb.ListAdminBillingAccessOverridesRequest{
		UserId: parseUserID,
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "list-mech-billing",
			SortBy:        "override_key",
			SortDirection: "asc",
		},
	})
	parsePage0Cancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingAccessOverrides page0: %v", parseErr)
	}
	parsePage1Ctx, parsePage1Cancel := parseBuildExample100AdminMutationCallContext()
	parsePage1Resp, parseErr := parseClient.ListAdminBillingAccessOverrides(parsePage1Ctx, &chatpb.ListAdminBillingAccessOverridesRequest{
		UserId: parseUserID,
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "list-mech-billing",
			SortBy:        "override_key",
			SortDirection: "asc",
		},
	})
	parsePage1Cancel()
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingAccessOverrides page1: %v", parseErr)
	}
	if len(parsePage0Resp.GetOverrides()) == 0 || len(parsePage1Resp.GetOverrides()) == 0 {
		parseT.Fatalf("expected paged billing override rows, got page0=%d page1=%d", len(parsePage0Resp.GetOverrides()), len(parsePage1Resp.GetOverrides()))
	}
	if parsePage0Resp.GetOverrides()[0].GetId() == parsePage1Resp.GetOverrides()[0].GetId() {
		parseT.Fatalf("expected distinct billing override rows across pagination pages, got same id=%d", parsePage0Resp.GetOverrides()[0].GetId())
	}
}

// parseAssertExample100AdminListURLStateNavigation verifies list-state query-string stability across refresh/back/forward browser navigation.
func parseAssertExample100AdminListURLStateNavigation(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseRelativeURLs []string) {
	parseT.Helper()
	if len(parseRelativeURLs) == 0 {
		parseT.Fatalf("list URL-state navigation requires at least one route")
	}
	for _, parseRelativeURL := range parseRelativeURLs {
		if _, parseErr := parsePage.Goto(parseBaseURL+parseRelativeURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto %s: %v", parseRelativeURL, parseErr)
		}
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRelativeURL)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait exact route state %s: %v", parseRelativeURL, parseErr)
		}
		if _, parseErr := parsePage.Reload(playwright.PageReloadOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("reload %s: %v", parseRelativeURL, parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait exact route state after reload %s: %v", parseRelativeURL, parseErr)
		}
	}
	for parseIndex := len(parseRelativeURLs) - 2; parseIndex >= 0; parseIndex-- {
		if _, parseErr := parsePage.GoBack(playwright.PageGoBackOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("back navigation index %d: %v", parseIndex, parseErr)
		}
		parseRelativeURL := parseRelativeURLs[parseIndex]
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRelativeURL)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait exact route state after back %s: %v", parseRelativeURL, parseErr)
		}
	}
	for parseIndex := 1; parseIndex < len(parseRelativeURLs); parseIndex++ {
		if _, parseErr := parsePage.GoForward(playwright.PageGoForwardOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("forward navigation index %d: %v", parseIndex, parseErr)
		}
		parseRelativeURL := parseRelativeURLs[parseIndex]
		parseWaitJS := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseRelativeURL)
		if _, parseErr := parsePage.WaitForFunction(parseWaitJS, nil); parseErr != nil {
			parseT.Fatalf("wait exact route state after forward %s: %v", parseRelativeURL, parseErr)
		}
	}
}
