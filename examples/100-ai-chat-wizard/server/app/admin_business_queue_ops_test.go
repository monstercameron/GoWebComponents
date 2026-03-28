package app

import (
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseSeedAdminBusinessQueueDetailRows seeds one invoice-line and first-chat funnel fixture set for one target user.
func parseSeedAdminBusinessQueueDetailRows(parseT *testing.T, parseStore *Store, parseUserID int64, parseWorkspaceID int64) {
	parseT.Helper()
	parseCustomerRow, hasParseCustomerRow, parseErr := parseStore.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !hasParseCustomerRow {
		parseT.Fatalf("expected billing customer for user %d", parseUserID)
	}
	parseSubscriptionRows, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, 1)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: %v", parseErr)
	}
	if len(parseSubscriptionRows) == 0 {
		parseT.Fatalf("expected subscription rows for customer %d", parseCustomerRow.ID)
	}
	parseInvoiceRows, parseErr := parseStore.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, 1)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingInvoicesByCustomer: %v", parseErr)
	}
	if len(parseInvoiceRows) == 0 {
		parseT.Fatalf("expected invoice rows for customer %d", parseCustomerRow.ID)
	}
	if _, parseErr = parseStore.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseCustomerRow.ID,
		InvoiceID:       parseInvoiceRows[0].ID,
		UsageEventID:    "evt-alice-openai",
		LineType:        "usage",
		Description:     "Admin business queue seeded usage line",
		Quantity:        1,
		UnitAmountCents: 125,
		AmountCents:     125,
		Currency:        "usd",
		PeriodStart:     time.Now().UTC().Format(time.RFC3339),
		PeriodEnd:       time.Now().UTC().Format(time.RFC3339),
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingInvoiceLineItem: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateProductAnalyticsEvent(parseProductAnalyticsEventWrite{
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseUserID,
		SessionKey:     "session-admin-business-queue",
		EventName:      "funnel.visit_to_first_chat.first_reply_completed",
		FunnelKey:      "visit_to_first_chat",
		StepKey:        "first_reply_completed",
		ExperimentKey:  "exp-dashboard-active",
		VariantKey:     "treatment",
		EventPropsJSON: `{"source":"admin-business-queue-test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateProductAnalyticsEvent: %v", parseErr)
	}
}

// TestGetAdminBusinessQueueAndAccountDetail verifies typed business queue and account-detail RPCs for failed-payment, dunning, and funnel drill-downs.
func TestGetAdminBusinessQueueAndAccountDetail(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseMembershipRows, parseErr := parseStore.parseListWorkspaceMembershipsByUser(parseAliceAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaceMembershipsByUser alice: %v", parseErr)
	}
	if len(parseMembershipRows) == 0 {
		parseT.Fatalf("expected workspace membership for alice")
	}
	parseWorkspaceID := parseMembershipRows[0].WorkspaceID
	parseSeedAdminBusinessQueueDetailRows(parseT, parseStore, parseAliceAuth.ID, parseWorkspaceID)

	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-admin-business-queue-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseQueueResp, parseErr := parseServer.GetAdminBusinessQueue(parseSuperuserCtx, &chatpb.GetAdminBusinessQueueRequest{
		LookbackDays: 30,
		Limit:        25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminBusinessQueue: %v", parseErr)
	}
	if parseQueueResp.GetSummary() == nil {
		parseT.Fatalf("expected business queue summary payload")
	}
	if len(parseQueueResp.GetFailedPaymentEvents()) == 0 ||
		len(parseQueueResp.GetDunningEvents()) == 0 ||
		len(parseQueueResp.GetTopAccounts()) == 0 ||
		len(parseQueueResp.GetFirstChatFunnelEvents()) == 0 {
		parseT.Fatalf(
			"expected non-empty business queue slices, got failed_payments=%d dunning=%d top_accounts=%d funnel=%d",
			len(parseQueueResp.GetFailedPaymentEvents()),
			len(parseQueueResp.GetDunningEvents()),
			len(parseQueueResp.GetTopAccounts()),
			len(parseQueueResp.GetFirstChatFunnelEvents()),
		)
	}

	parseDetailResp, parseErr := parseServer.GetAdminBusinessAccountDetail(parseSuperuserCtx, &chatpb.GetAdminBusinessAccountDetailRequest{
		UserId:       parseAliceAuth.ID,
		LookbackDays: 30,
		Limit:        25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminBusinessAccountDetail: %v", parseErr)
	}
	if parseDetailResp.GetUser() == nil || parseDetailResp.GetUser().GetUserId() != parseAliceAuth.ID {
		parseT.Fatalf("expected business account user detail for alice, got %+v", parseDetailResp.GetUser())
	}
	if parseDetailResp.GetCustomer() == nil || parseDetailResp.GetCustomer().GetId() <= 0 {
		parseT.Fatalf("expected business account customer detail, got %+v", parseDetailResp.GetCustomer())
	}
	if len(parseDetailResp.GetSubscriptions()) == 0 ||
		len(parseDetailResp.GetInvoices()) == 0 ||
		len(parseDetailResp.GetInvoiceLineItems()) == 0 ||
		len(parseDetailResp.GetBillingEvents()) == 0 ||
		len(parseDetailResp.GetDunningEvents()) == 0 ||
		len(parseDetailResp.GetFirstChatFunnelEvents()) == 0 {
		parseT.Fatalf(
			"expected non-empty business account detail slices, got subscriptions=%d invoices=%d line_items=%d billing_events=%d dunning=%d funnel=%d",
			len(parseDetailResp.GetSubscriptions()),
			len(parseDetailResp.GetInvoices()),
			len(parseDetailResp.GetInvoiceLineItems()),
			len(parseDetailResp.GetBillingEvents()),
			len(parseDetailResp.GetDunningEvents()),
			len(parseDetailResp.GetFirstChatFunnelEvents()),
		)
	}
}

// TestGetAdminBusinessQueueAndAccountDetailScopeGuards verifies workspace-admin callers are denied business queue/detail access.
func TestGetAdminBusinessQueueAndAccountDetailScopeGuards(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-business-queue-scope")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-business-queue-bob", parseBobAuth.ID, parseBobAuth.Email)

	if _, parseErr = parseServer.GetAdminBusinessQueue(parseBobCtx, &chatpb.GetAdminBusinessQueueRequest{
		LookbackDays: 30,
		Limit:        10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminBusinessQueue workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.GetAdminBusinessAccountDetail(parseBobCtx, &chatpb.GetAdminBusinessAccountDetailRequest{
		UserId:       parseBobAuth.ID,
		LookbackDays: 30,
		Limit:        10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminBusinessAccountDetail workspace-admin status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
