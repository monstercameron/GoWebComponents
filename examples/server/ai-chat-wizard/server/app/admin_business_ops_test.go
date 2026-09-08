package app

import (
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestGetAdminBusinessDrilldownRPCs verifies typed business drill-down payloads include billing, usage, analytics, and churn data.
func TestGetAdminBusinessDrilldownRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-business-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseMembershipRows, parseErr := parseStore.parseListWorkspaceMembershipsByUser(parseAliceAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaceMembershipsByUser alice: %v", parseErr)
	}
	if len(parseMembershipRows) == 0 {
		parseT.Fatalf("expected workspace membership for user %d", parseAliceAuth.ID)
	}
	parseWorkspaceID := parseMembershipRows[0].WorkspaceID

	parseCustomerRow, hasParseCustomerRow, parseErr := parseStore.parseGetBillingCustomerByUser(parseAliceAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser alice: %v", parseErr)
	}
	if !hasParseCustomerRow {
		parseT.Fatalf("expected billing customer for user %d", parseAliceAuth.ID)
	}
	parseSubscriptionRows, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, 1)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer alice: %v", parseErr)
	}
	if len(parseSubscriptionRows) == 0 {
		parseT.Fatalf("expected billing subscription for customer %d", parseCustomerRow.ID)
	}
	parseInvoiceRows, parseErr := parseStore.parseListBillingInvoicesByCustomer(parseCustomerRow.ID, 1)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingInvoicesByCustomer alice: %v", parseErr)
	}
	if len(parseInvoiceRows) == 0 {
		parseT.Fatalf("expected billing invoice for customer %d", parseCustomerRow.ID)
	}
	parseInvoiceRow := parseInvoiceRows[0]

	if _, parseErr = parseStore.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseCustomerRow.ID,
		InvoiceID:       parseInvoiceRow.ID,
		UsageEventID:    "evt-alice-openai",
		LineType:        "usage",
		Description:     "OpenAI usage",
		Quantity:        1,
		UnitAmountCents: 250,
		AmountCents:     250,
		Currency:        "usd",
		PeriodStart:     time.Now().UTC().Format(time.RFC3339),
		PeriodEnd:       time.Now().UTC().Format(time.RFC3339),
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingInvoiceLineItem: %v", parseErr)
	}
	if _, parseErr = parseStore.parseSetAdminBillingAccessOverrideByUser(parseAliceAuth.ID, parseBillingAccessOverrideWrite{
		OverrideKey:   "access.priority_support",
		OverrideValue: "enabled",
		Reason:        "admin business drilldown seed",
		IsEnabled:     true,
		ActorUserID:   parseAliceAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseSetAdminBillingAccessOverrideByUser: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateProductAnalyticsEvent(parseProductAnalyticsEventWrite{
		WorkspaceID:    parseWorkspaceID,
		UserID:         parseAliceAuth.ID,
		SessionKey:     "session-admin-business-alice",
		EventName:      "funnel.visit_to_first_chat.first_reply_completed",
		FunnelKey:      "visit_to_first_chat",
		StepKey:        "first_reply_completed",
		ExperimentKey:  "exp-dashboard-active",
		VariantKey:     "treatment",
		EventPropsJSON: `{"source":"admin-business-test"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateProductAnalyticsEvent: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateSubscriptionChurnFeedback(parseSubscriptionChurnFeedbackWrite{
		CustomerID:       parseCustomerRow.ID,
		SubscriptionID:   parseSubscriptionRows[0].ID,
		WorkspaceID:      parseWorkspaceID,
		ReasonKey:        "price",
		Detail:           "Too expensive",
		RecoveryOfferKey: "discount_20",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateSubscriptionChurnFeedback: %v", parseErr)
	}

	parseResp, parseErr := parseServer.GetAdminBusinessDrilldown(parseAliceCtx, &chatpb.GetAdminBusinessDrilldownRequest{
		UserId: parseAliceAuth.ID,
		Limit:  25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminBusinessDrilldown: %v", parseErr)
	}
	if parseResp.GetUser() == nil || parseResp.GetUser().GetUserId() != parseAliceAuth.ID {
		parseT.Fatalf("expected typed admin user summary for alice, got %+v", parseResp.GetUser())
	}
	if parseResp.GetCustomer() == nil || parseResp.GetCustomer().GetId() <= 0 {
		parseT.Fatalf("expected billing customer entry, got %+v", parseResp.GetCustomer())
	}
	if len(parseResp.GetSubscriptions()) == 0 || len(parseResp.GetInvoices()) == 0 || len(parseResp.GetInvoiceLineItems()) == 0 {
		parseT.Fatalf("expected billing subscriptions/invoices/line-items, got subs=%d invoices=%d line_items=%d", len(parseResp.GetSubscriptions()), len(parseResp.GetInvoices()), len(parseResp.GetInvoiceLineItems()))
	}
	if len(parseResp.GetAccessOverrides()) == 0 || len(parseResp.GetBillingEvents()) == 0 {
		parseT.Fatalf("expected billing overrides/events, got overrides=%d events=%d", len(parseResp.GetAccessOverrides()), len(parseResp.GetBillingEvents()))
	}
	if len(parseResp.GetUsageEvents()) == 0 || len(parseResp.GetProductAnalyticsEvents()) == 0 || len(parseResp.GetChurnFeedback()) == 0 {
		parseT.Fatalf("expected usage/analytics/churn rows, got usage=%d analytics=%d churn=%d", len(parseResp.GetUsageEvents()), len(parseResp.GetProductAnalyticsEvents()), len(parseResp.GetChurnFeedback()))
	}
}

// TestGetAdminBusinessDrilldownScopeGuards verifies workspace-admin callers are denied out-of-scope targets.
func TestGetAdminBusinessDrilldownScopeGuards(parseT *testing.T) {
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
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-business-scope")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-business-bob", parseBobAuth.ID, parseBobAuth.Email)

	if _, parseErr = parseServer.GetAdminBusinessDrilldown(parseBobCtx, &chatpb.GetAdminBusinessDrilldownRequest{
		UserId: parseAliceAuth.ID,
		Limit:  10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminBusinessDrilldown out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}

	if _, parseErr = parseServer.GetAdminBusinessDrilldown(parseBobCtx, &chatpb.GetAdminBusinessDrilldownRequest{
		UserId: parseBobAuth.ID,
		Limit:  10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminBusinessDrilldown workspace-admin own-user status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
