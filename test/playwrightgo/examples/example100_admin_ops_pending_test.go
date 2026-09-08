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

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	playwright "github.com/mxschmitt/playwright-go"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type example100AdminOpsFixture struct {
	GetTargetUserID      int64
	GetTargetWorkspaceID int64
	GetTargetTicketID    int64
}

// TestExample100AdminOpsRegression validates billing-intervention, support-triage, and incident-control mutations with refresh/back stability checks.
func TestExample100AdminOpsRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseFixture := startExample100AdminOpsServer(parseT, parseRepoRoot, "18110")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseBillingMutationKey := fmt.Sprintf("ops-billing-access-%d", time.Now().UTC().UnixNano())
		parseBillingSetCtx, parseBillingSetCancel := parseBuildExample100AdminMutationCallContext()
		parseBillingSetResp, parseErr := parseClient.SetAdminBillingAccessOverride(parseBillingSetCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
			UserId:        parseFixture.GetTargetUserID,
			OverrideKey:   parseBillingMutationKey,
			OverrideValue: "enabled",
			Reason:        "playwright admin ops regression: billing intervention",
			IsEnabled:     true,
			Confirm:       true,
		})
		parseBillingSetCancel()
		if parseErr != nil {
			parseT.Fatalf("SetAdminBillingAccessOverride: %v", parseErr)
		}
		if strings.TrimSpace(parseBillingSetResp.GetStatus()) == "" {
			parseT.Fatalf("SetAdminBillingAccessOverride status should not be blank: %+v", parseBillingSetResp)
		}
		parseBillingListCtx, parseBillingListCancel := parseBuildExample100AdminMutationCallContext()
		parseBillingListResp, parseErr := parseClient.ListAdminBillingAccessOverrides(parseBillingListCtx, &chatpb.ListAdminBillingAccessOverridesRequest{
			UserId:      parseFixture.GetTargetUserID,
			OverrideKey: parseBillingMutationKey,
			ListQuery: &chatpb.AdminListQuery{
				Limit: 20,
			},
		})
		parseBillingListCancel()
		if parseErr != nil {
			parseT.Fatalf("ListAdminBillingAccessOverrides: %v", parseErr)
		}
		if len(parseBillingListResp.GetOverrides()) == 0 {
			parseT.Fatalf("expected billing override rows for key %q, got none", parseBillingMutationKey)
		}
		parseAssertExample100AdminOpsBackRefreshNavigation(parseT, parsePage, parseBaseURL, "billing-intervention")

		parseSupportNoteBody := fmt.Sprintf("ops-note-%d", time.Now().UTC().UnixNano())
		parseSupportNoteCtx, parseSupportNoteCancel := parseBuildExample100AdminMutationCallContext()
		parseSupportNoteResp, parseErr := parseClient.AddAdminSupportInternalNote(parseSupportNoteCtx, &chatpb.AddAdminSupportInternalNoteRequest{
			TicketId: parseFixture.GetTargetTicketID,
			Body:     parseSupportNoteBody,
			Confirm:  true,
			Reason:   "playwright admin ops regression: support internal note",
		})
		parseSupportNoteCancel()
		if parseErr != nil {
			parseT.Fatalf("AddAdminSupportInternalNote: %v", parseErr)
		}
		if strings.TrimSpace(parseSupportNoteResp.GetStatus()) == "" {
			parseT.Fatalf("AddAdminSupportInternalNote status should not be blank: %+v", parseSupportNoteResp)
		}
		parseSupportAssignCtx, parseSupportAssignCancel := parseBuildExample100AdminMutationCallContext()
		parseSupportAssignResp, parseErr := parseClient.AssignAdminSupportTicket(parseSupportAssignCtx, &chatpb.AssignAdminSupportTicketRequest{
			TicketId:       parseFixture.GetTargetTicketID,
			AssigneeUserId: 1,
			Confirm:        true,
			Reason:         "playwright admin ops regression: support assignment",
		})
		parseSupportAssignCancel()
		if parseErr != nil {
			parseT.Fatalf("AssignAdminSupportTicket: %v", parseErr)
		}
		if strings.TrimSpace(parseSupportAssignResp.GetStatus()) == "" {
			parseT.Fatalf("AssignAdminSupportTicket status should not be blank: %+v", parseSupportAssignResp)
		}
		parseSupportEscalateCtx, parseSupportEscalateCancel := parseBuildExample100AdminMutationCallContext()
		parseSupportEscalateResp, parseErr := parseClient.EscalateAdminSupportTicket(parseSupportEscalateCtx, &chatpb.EscalateAdminSupportTicketRequest{
			TicketId:       parseFixture.GetTargetTicketID,
			Priority:       "high",
			Status:         "escalated",
			ResolutionNote: "playwright admin ops regression escalation",
			Confirm:        true,
			Reason:         "playwright admin ops regression: support escalation",
		})
		parseSupportEscalateCancel()
		if parseErr != nil {
			parseT.Fatalf("EscalateAdminSupportTicket: %v", parseErr)
		}
		if strings.TrimSpace(parseSupportEscalateResp.GetStatus()) == "" {
			parseT.Fatalf("EscalateAdminSupportTicket status should not be blank: %+v", parseSupportEscalateResp)
		}
		parseSupportDetailCtx, parseSupportDetailCancel := parseBuildExample100AdminMutationCallContext()
		parseSupportDetailResp, parseErr := parseClient.GetAdminSupportTicketDetail(parseSupportDetailCtx, &chatpb.GetAdminSupportTicketDetailRequest{
			TicketId: parseFixture.GetTargetTicketID,
			Limit:    50,
		})
		parseSupportDetailCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminSupportTicketDetail: %v", parseErr)
		}
		parseHasSupportNote := false
		for _, parseMessageRow := range parseSupportDetailResp.GetDetail().GetMessages() {
			if strings.Contains(parseMessageRow.GetBody(), parseSupportNoteBody) {
				parseHasSupportNote = true
				break
			}
		}
		if !parseHasSupportNote {
			parseT.Fatalf("expected support internal note body %q in ticket detail: %+v", parseSupportNoteBody, parseSupportDetailResp.GetDetail().GetMessages())
		}
		parseAssertExample100AdminOpsBackRefreshNavigation(parseT, parsePage, parseBaseURL, "support-triage")

		parseSLOKey := fmt.Sprintf("ops-slo-%d", time.Now().UTC().UnixNano())
		parseSLOSetCtx, parseSLOSetCancel := parseBuildExample100AdminMutationCallContext()
		parseSLOSetResp, parseErr := parseClient.SetSuperuserServiceLevelObjective(parseSLOSetCtx, &chatpb.SetSuperuserServiceLevelObjectiveRequest{
			SloKey:             parseSLOKey,
			ServiceName:        "admin-ops-regression",
			ObjectivePercent:   99.9,
			WindowDays:         30,
			ErrorBudgetMinutes: 43,
			StatusPageUrl:      "https://status.example.invalid/admin-ops-regression",
			Confirm:            true,
			Reason:             "playwright admin ops regression: seed slo",
		})
		parseSLOSetCancel()
		if parseErr != nil {
			parseT.Fatalf("SetSuperuserServiceLevelObjective: %v", parseErr)
		}
		if parseSLOSetResp.GetSlo() == nil || strings.TrimSpace(parseSLOSetResp.GetSlo().GetSloKey()) == "" {
			parseT.Fatalf("SetSuperuserServiceLevelObjective returned invalid payload: %+v", parseSLOSetResp)
		}

		parseIncidentKey := fmt.Sprintf("ops-incident-%d", time.Now().UTC().UnixNano())
		parseIncidentSetCtx, parseIncidentSetCancel := parseBuildExample100AdminMutationCallContext()
		parseIncidentSetResp, parseErr := parseClient.SetSuperuserIncident(parseIncidentSetCtx, &chatpb.SetSuperuserIncidentRequest{
			IncidentKey: parseIncidentKey,
			SloKey:      parseSLOKey,
			Severity:    "major",
			Status:      "open",
			Title:       "Playwright admin ops incident",
			Summary:     "Created during admin ops regression",
			StartedAt:   time.Now().UTC().Format(time.RFC3339),
			Confirm:     true,
			Reason:      "playwright admin ops regression: seed incident",
		})
		parseIncidentSetCancel()
		if parseErr != nil {
			parseT.Fatalf("SetSuperuserIncident: %v", parseErr)
		}
		if parseIncidentSetResp.GetIncident() == nil || parseIncidentSetResp.GetIncident().GetId() <= 0 {
			parseT.Fatalf("SetSuperuserIncident returned invalid incident: %+v", parseIncidentSetResp)
		}
		parseIncidentUpdateCtx, parseIncidentUpdateCancel := parseBuildExample100AdminMutationCallContext()
		parseIncidentUpdateResp, parseErr := parseClient.UpdateAdminIncident(parseIncidentUpdateCtx, &chatpb.UpdateAdminIncidentRequest{
			IncidentId:  parseIncidentSetResp.GetIncident().GetId(),
			WorkspaceId: parseFixture.GetTargetWorkspaceID,
			Status:      "investigating",
			Message:     "playwright admin ops incident update",
			IsPublic:    false,
			Confirm:     true,
			Reason:      "playwright admin ops regression: incident update",
		})
		parseIncidentUpdateCancel()
		if parseErr != nil {
			parseT.Fatalf("UpdateAdminIncident: %v", parseErr)
		}
		if parseIncidentUpdateResp.GetIncident() == nil || strings.TrimSpace(parseIncidentUpdateResp.GetIncident().GetStatus()) == "" {
			parseT.Fatalf("UpdateAdminIncident returned invalid incident payload: %+v", parseIncidentUpdateResp)
		}
		parseBlastCtx, parseBlastCancel := parseBuildExample100AdminMutationCallContext()
		parseBlastResp, parseErr := parseClient.GetAdminIncidentBlastRadius(parseBlastCtx, &chatpb.GetAdminIncidentBlastRadiusRequest{
			WorkspaceId:  parseFixture.GetTargetWorkspaceID,
			LookbackDays: 30,
		})
		parseBlastCancel()
		if parseErr != nil {
			parseT.Fatalf("GetAdminIncidentBlastRadius: %v", parseErr)
		}
		if parseBlastResp.GetBlastRadius() == nil || parseBlastResp.GetBlastRadius().GetWorkspaceId() != parseFixture.GetTargetWorkspaceID {
			parseT.Fatalf("unexpected blast radius payload: %+v", parseBlastResp)
		}
		parseAssertExample100AdminOpsBackRefreshNavigation(parseT, parsePage, parseBaseURL, "incident-control")

		assertExample100AdminGuardNoRuntimeErrors(parseT, "admin-ops regression flow", parseEvidence)
	})
}

// startExample100AdminOpsServer starts one seeded server with deterministic workspace-admin billing/support fixtures for admin-ops regression coverage.
func startExample100AdminOpsServer(parseT *testing.T, parseRepoRoot string, parsePort string) (string, example100AdminOpsFixture) {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "admin_ops.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseBinaryPath := filepath.Join(parseRuntimeDir, "example100-admin-ops-server")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)

	seedExample100HappyPathDatabase(parseT, parseRepoRoot, parseDBPath)
	grantExample100AdminJourneySuperuserRole(parseT, parseDBPath)
	seedExample100AdminGuardWorkspaceAdminUser(parseT, parseDBPath)
	parseFixture := seedExample100AdminOpsFixtures(parseT, parseDBPath)
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

// seedExample100AdminOpsFixtures inserts one billing customer path and one support ticket for the workspace-admin fixture account.
func seedExample100AdminOpsFixtures(parseT *testing.T, parseDBPath string) example100AdminOpsFixture {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open admin-ops sqlite db: %v", parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)

	var parseTargetUserID int64
	if parseErr := parseDB.QueryRow(`SELECT id FROM users WHERE email = ? COLLATE NOCASE`, example100AdminGuardWorkspaceEmail).Scan(&parseTargetUserID); parseErr != nil {
		parseT.Fatalf("resolve admin-ops target user id: %v", parseErr)
	}
	var parseTargetWorkspaceID int64
	if parseErr := parseDB.QueryRow(`SELECT id FROM workspaces WHERE workspace_key = ?`, example100AdminMutationWorkspaceKey).Scan(&parseTargetWorkspaceID); parseErr != nil {
		parseT.Fatalf("resolve admin-ops target workspace id: %v", parseErr)
	}
	var parseSuperuserID int64
	if parseErr := parseDB.QueryRow(`SELECT id FROM users WHERE email = ? COLLATE NOCASE`, example100AdminJourneyLoginEmail).Scan(&parseSuperuserID); parseErr != nil {
		parseT.Fatalf("resolve admin-ops superuser id: %v", parseErr)
	}

	parseNow := time.Now().UTC()
	parseNowRFC3339 := parseNow.Format(time.RFC3339)

	parseCustomerResult, parseErr := parseDB.Exec(
		`INSERT INTO billing_customers (user_id, provider_id, provider_customer_id, billing_email, billing_name, billing_country, billing_region, default_currency, tax_exempt_status, external_metadata_json, created_at, updated_at)
		 VALUES (?, 'stripe', ?, ?, ?, 'US', 'NY', 'USD', 'none', '{}', ?, ?)`,
		parseTargetUserID,
		fmt.Sprintf("cus-admin-ops-%d", parseNow.UnixNano()),
		example100AdminGuardWorkspaceEmail,
		"Workspace Admin",
		parseNowRFC3339,
		parseNowRFC3339,
	)
	if parseErr != nil {
		parseT.Fatalf("insert admin-ops billing customer: %v", parseErr)
	}
	parseCustomerID, parseErr := parseCustomerResult.LastInsertId()
	if parseErr != nil {
		parseT.Fatalf("read admin-ops billing customer id: %v", parseErr)
	}

	parseSubscriptionResult, parseErr := parseDB.Exec(
		`INSERT INTO billing_subscriptions (customer_id, provider_id, provider_subscription_id, plan_code, price_code, status, billing_interval, quantity, current_period_start, current_period_end, cancel_at_period_end, canceled_at, trial_ends_at, access_expires_at, created_at, updated_at)
		 VALUES (?, 'stripe', ?, 'pro', 'pro-monthly', 'active', 'month', 1, ?, ?, 0, '', '', '', ?, ?)`,
		parseCustomerID,
		fmt.Sprintf("sub-admin-ops-%d", parseNow.UnixNano()),
		parseNowRFC3339,
		parseNow.Add(30*24*time.Hour).Format(time.RFC3339),
		parseNowRFC3339,
		parseNowRFC3339,
	)
	if parseErr != nil {
		parseT.Fatalf("insert admin-ops subscription: %v", parseErr)
	}
	parseSubscriptionID, parseErr := parseSubscriptionResult.LastInsertId()
	if parseErr != nil {
		parseT.Fatalf("read admin-ops subscription id: %v", parseErr)
	}

	parseInvoiceResult, parseErr := parseDB.Exec(
		`INSERT INTO billing_invoices (customer_id, subscription_id, provider_id, provider_invoice_id, status, currency, subtotal_cents, tax_cents, discount_cents, total_cents, amount_due_cents, amount_paid_cents, period_start, period_end, due_at, paid_at, hosted_invoice_url, external_metadata_json, created_at, updated_at)
		 VALUES (?, ?, 'stripe', ?, 'open', 'USD', 1200, 0, 0, 1200, 1200, 0, ?, ?, ?, '', '', '{}', ?, ?)`,
		parseCustomerID,
		parseSubscriptionID,
		fmt.Sprintf("inv-admin-ops-%d", parseNow.UnixNano()),
		parseNowRFC3339,
		parseNow.Add(30*24*time.Hour).Format(time.RFC3339),
		parseNow.Add(7*24*time.Hour).Format(time.RFC3339),
		parseNowRFC3339,
		parseNowRFC3339,
	)
	if parseErr != nil {
		parseT.Fatalf("insert admin-ops invoice: %v", parseErr)
	}
	parseInvoiceID, parseErr := parseInvoiceResult.LastInsertId()
	if parseErr != nil {
		parseT.Fatalf("read admin-ops invoice id: %v", parseErr)
	}

	if _, parseErr := parseDB.Exec(
		`INSERT INTO billing_dunning_events (customer_id, subscription_id, invoice_id, status, attempt_count, failure_reason, next_attempt_at, resolved_at, created_at, updated_at)
		 VALUES (?, ?, ?, 'pending', 1, 'card_declined', ?, '', ?, ?)`,
		parseCustomerID,
		parseSubscriptionID,
		parseInvoiceID,
		parseNow.Add(24*time.Hour).Format(time.RFC3339),
		parseNowRFC3339,
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("insert admin-ops dunning event: %v", parseErr)
	}

	if _, parseErr := parseDB.Exec(
		`INSERT INTO billing_events (customer_id, subscription_id, invoice_id, event_type, event_source, event_summary, event_payload_json, actor_user_id, created_at)
		 VALUES (?, ?, ?, 'invoice.payment_failed', 'stripe', 'Card declined for open invoice', '{"failure_reason":"card_declined"}', 0, ?)`,
		parseCustomerID,
		parseSubscriptionID,
		parseInvoiceID,
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("insert admin-ops billing event: %v", parseErr)
	}

	parseTicketResult, parseErr := parseDB.Exec(
		`INSERT INTO support_tickets (ticket_key, workspace_id, user_id, status, priority, subject, body, assignee_user_id, resolution_note, created_at, updated_at)
		 VALUES (?, ?, ?, 'open', 'normal', 'Admin ops regression ticket', 'Seeded support ticket for triage mutation flow coverage', ?, '', ?, ?)`,
		fmt.Sprintf("ticket-admin-ops-%d", parseNow.UnixNano()),
		parseTargetWorkspaceID,
		parseTargetUserID,
		parseSuperuserID,
		parseNowRFC3339,
		parseNowRFC3339,
	)
	if parseErr != nil {
		parseT.Fatalf("insert admin-ops support ticket: %v", parseErr)
	}
	parseTicketID, parseErr := parseTicketResult.LastInsertId()
	if parseErr != nil {
		parseT.Fatalf("read admin-ops support ticket id: %v", parseErr)
	}

	return example100AdminOpsFixture{
		GetTargetUserID:      parseTargetUserID,
		GetTargetWorkspaceID: parseTargetWorkspaceID,
		GetTargetTicketID:    parseTicketID,
	}
}

// parseAssertExample100AdminOpsBackRefreshNavigation verifies admin route stability across refresh, back, and forward after one operator mutation flow.
func parseAssertExample100AdminOpsBackRefreshNavigation(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseLabel string) {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard/usage", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("%s goto /app/dashboard/usage: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/dashboard/usage")`, nil); parseErr != nil {
		parseT.Fatalf("%s wait /app/dashboard/usage: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/admin/users", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("%s goto /app/admin/users: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/admin/users")`, nil); parseErr != nil {
		parseT.Fatalf("%s wait /app/admin/users: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.Reload(playwright.PageReloadOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("%s reload /app/admin/users: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/admin/users")`, nil); parseErr != nil {
		parseT.Fatalf("%s wait /app/admin/users after reload: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.GoBack(playwright.PageGoBackOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("%s back navigation: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/dashboard/usage")`, nil); parseErr != nil {
		parseT.Fatalf("%s wait /app/dashboard/usage after back: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.GoForward(playwright.PageGoForwardOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("%s forward navigation: %v", parseLabel, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/admin/users")`, nil); parseErr != nil {
		parseT.Fatalf("%s wait /app/admin/users after forward: %v", parseLabel, parseErr)
	}
}
