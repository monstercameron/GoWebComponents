package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseSeedCustomerBillingSnapshot seeds one user-scoped billing snapshot and returns customer, subscription, and invoice rows.
func parseSeedCustomerBillingSnapshot(parseT *testing.T, parseStore *Store, parseUserID int64) (parseBillingCustomerRow, parseBillingSubscriptionRow, parseBillingInvoiceRow) {
	parseT.Helper()
	parseMustAssignBillingPlan(parseT, parseStore, parseUserID, "team")

	parseCustomerRow, isHasCustomer, parseErr := parseStore.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !isHasCustomer {
		parseT.Fatalf("expected billing customer for user %d", parseUserID)
	}
	parseSubscriptionRows, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, 1)
	if parseErr != nil || len(parseSubscriptionRows) == 0 {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: rows=%+v err=%v", parseSubscriptionRows, parseErr)
	}
	parseNow := time.Now().UTC()
	parseInvoiceRow, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomerRow.ID,
		SubscriptionID:    parseSubscriptionRows[0].ID,
		ProviderID:        "stripe",
		ProviderInvoiceID: fmt.Sprintf("inv-customer-summary-%d", parseUserID),
		Status:            "open",
		Currency:          "usd",
		SubtotalCents:     4160,
		TotalCents:        4160,
		AmountDueCents:    4160,
		PeriodStart:       parseNow.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		PeriodEnd:         parseNow.Format(time.RFC3339),
		DueAt:             parseNow.Add(7 * 24 * time.Hour).Format(time.RFC3339),
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice: %v", parseErr)
	}
	return parseCustomerRow, parseSubscriptionRows[0], parseInvoiceRow
}

// TestGetCustomerBillingSummary returns one typed customer billing snapshot sourced from billing + usage rows.
func TestGetCustomerBillingSummary(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "customer-billing-summary@example.com")
	parseCustomerRow, _, parseInvoiceRow := parseSeedCustomerBillingSnapshot(parseT, parseStore, parseUser.ID)

	parseLineItems := []parseBillingInvoiceLineItemWrite{
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "platform_fee", Description: "Platform fee", Quantity: 1, UnitAmountCents: 2900, AmountCents: 2900, Currency: "usd"},
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "usage_cost", Description: "Usage", Quantity: 1, UnitAmountCents: 1200, AmountCents: 1200, Currency: "usd"},
		{CustomerID: parseCustomerRow.ID, InvoiceID: parseInvoiceRow.ID, LineType: "service_premium", Description: "Premium", Quantity: 1, UnitAmountCents: 60, AmountCents: 60, Currency: "usd"},
	}
	for _, parseLineItemWrite := range parseLineItems {
		if _, parseErr := parseStore.parseCreateBillingInvoiceLineItem(parseLineItemWrite); parseErr != nil {
			parseT.Fatalf("parseCreateBillingInvoiceLineItem: %v", parseErr)
		}
	}

	parseConversationID, parseErr := parseStore.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:          "evt-customer-billing-1",
		UserID:           parseUser.ID,
		ConversationID:   parseConversationID,
		ProviderID:       "openai",
		ModelID:          modelGPT54Mini,
		PromptTokens:     120,
		CompletionTokens: 80,
		PricingCurrency:  "usd",
		TotalCostUSD:     3.20,
		Status:           "completed",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent #1: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:          "evt-customer-billing-2",
		UserID:           parseUser.ID,
		ConversationID:   parseConversationID,
		ProviderID:       "openai",
		ModelID:          modelGPT54Mini,
		PromptTokens:     90,
		CompletionTokens: 60,
		PricingCurrency:  "usd",
		TotalCostUSD:     1.05,
		Status:           "completed",
	}); parseErr != nil {
		parseT.Fatalf("parseSaveUsageEvent #2: %v", parseErr)
	}

	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseCtx := parseBindAuthUser(parseServer, "peer-customer-billing-summary", parseUser.ID, parseUser.Email)
	parseResp, parseErr := parseServer.GetCustomerBillingSummary(parseCtx, &chatpb.GetCustomerBillingSummaryRequest{InvoiceLimit: 10, UsageLimit: 10})
	if parseErr != nil {
		parseT.Fatalf("GetCustomerBillingSummary: %v", parseErr)
	}
	if parseResp.GetCustomer() == nil || parseResp.GetCustomer().GetUserId() != parseUser.ID {
		parseT.Fatalf("unexpected customer payload: %+v", parseResp.GetCustomer())
	}
	if len(parseResp.GetSubscriptions()) == 0 || len(parseResp.GetPlans()) == 0 {
		parseT.Fatalf("expected subscriptions and plan metadata, got subscriptions=%d plans=%d", len(parseResp.GetSubscriptions()), len(parseResp.GetPlans()))
	}
	if len(parseResp.GetRecentInvoices()) == 0 || len(parseResp.GetRecentInvoiceLineItems()) != 3 {
		parseT.Fatalf("expected invoice + line-item snapshot, got invoices=%d line_items=%d", len(parseResp.GetRecentInvoices()), len(parseResp.GetRecentInvoiceLineItems()))
	}
	if parseResp.GetUsageSummary().GetEventCount() != 2 {
		parseT.Fatalf("expected usage summary event count=2, got %+v", parseResp.GetUsageSummary())
	}
	parseTotals := parseResp.GetTotals()
	if parseTotals.GetPlatformFeeCents() != 2900 || parseTotals.GetUsageCostCents() != 1200 || parseTotals.GetServicePremiumCents() != 60 || parseTotals.GetTotalCents() != 4160 {
		parseT.Fatalf("unexpected totals payload: %+v", parseTotals)
	}
}

// TestListCustomerInvoiceHistory returns one customer-scoped invoice history slice.
func TestListCustomerInvoiceHistory(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseScopedUser := parseMustCreateUser(parseT, parseStore, "customer-billing-history@example.com")
	parseScopedCustomer, _, parseScopedInvoice := parseSeedCustomerBillingSnapshot(parseT, parseStore, parseScopedUser.ID)
	if _, parseErr := parseStore.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseScopedCustomer.ID,
		InvoiceID:       parseScopedInvoice.ID,
		LineType:        "platform_fee",
		Description:     "Platform fee",
		Quantity:        1,
		UnitAmountCents: 2900,
		AmountCents:     2900,
		Currency:        "usd",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingInvoiceLineItem scoped: %v", parseErr)
	}

	parseOutOfScopeUser := parseMustCreateUser(parseT, parseStore, "customer-billing-history-other@example.com")
	parseOutOfScopeCustomer, _, parseOutOfScopeInvoice := parseSeedCustomerBillingSnapshot(parseT, parseStore, parseOutOfScopeUser.ID)
	if _, parseErr := parseStore.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseOutOfScopeCustomer.ID,
		InvoiceID:       parseOutOfScopeInvoice.ID,
		LineType:        "platform_fee",
		Description:     "Platform fee",
		Quantity:        1,
		UnitAmountCents: 3100,
		AmountCents:     3100,
		Currency:        "usd",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingInvoiceLineItem out-of-scope: %v", parseErr)
	}

	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseScopedCtx := parseBindAuthUser(parseServer, "peer-customer-billing-history", parseScopedUser.ID, parseScopedUser.Email)
	parseResp, parseErr := parseServer.ListCustomerInvoiceHistory(parseScopedCtx, &chatpb.ListCustomerInvoiceHistoryRequest{Limit: 10})
	if parseErr != nil {
		parseT.Fatalf("ListCustomerInvoiceHistory: %v", parseErr)
	}
	if len(parseResp.GetInvoices()) != 1 || parseResp.GetInvoices()[0].GetCustomerId() != parseScopedCustomer.ID {
		parseT.Fatalf("expected one scoped invoice row, got %+v", parseResp.GetInvoices())
	}
	if len(parseResp.GetInvoiceLineItems()) != 1 || parseResp.GetInvoiceLineItems()[0].GetInvoiceId() != parseScopedInvoice.ID {
		parseT.Fatalf("expected one scoped line-item row, got %+v", parseResp.GetInvoiceLineItems())
	}
}

// TestCustomerBillingRPCsRequireAuthentication verifies customer billing RPCs fail closed for unauthenticated callers.
func TestCustomerBillingRPCsRequireAuthentication(parseT *testing.T) {
	parseServer := &chatServer{logger: parseNewTestLogger()}
	if _, parseErr := parseServer.GetCustomerBillingSummary(context.Background(), &chatpb.GetCustomerBillingSummaryRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("GetCustomerBillingSummary status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
	if _, parseErr := parseServer.ListCustomerInvoiceHistory(context.Background(), &chatpb.ListCustomerInvoiceHistoryRequest{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("ListCustomerInvoiceHistory status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
}
