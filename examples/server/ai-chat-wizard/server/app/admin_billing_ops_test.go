package app

import (
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestStoreAdminBillingInterventionFuncs verifies typed store helpers for billing override, dunning review, and failed-payment resolution.
func TestStoreAdminBillingInterventionFuncs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseBillingEvents, parseErr := parseStore.parseListAdminBillingEventsByUser(parseAliceAuth.ID, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminBillingEventsByUser: %v", parseErr)
	}
	if len(parseBillingEvents) == 0 {
		parseT.Fatalf("expected seeded billing events for alice user %d", parseAliceAuth.ID)
	}
	parseDunningEvents, parseErr := parseStore.parseListAdminBillingDunningEventsByUser(parseAliceAuth.ID, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminBillingDunningEventsByUser: %v", parseErr)
	}
	if len(parseDunningEvents) == 0 || parseDunningEvents[0].InvoiceID <= 0 {
		parseT.Fatalf("expected seeded dunning rows with invoice id, got %+v", parseDunningEvents)
	}
	parseResolvedInvoiceID := parseDunningEvents[0].InvoiceID

	parseAccessOverrideRow, parseErr := parseStore.parseSetAdminBillingAccessOverrideByUser(parseAliceAuth.ID, parseBillingAccessOverrideWrite{
		OverrideKey:   "access.support_priority",
		OverrideValue: "enabled",
		Reason:        "temporary support unblock",
		IsEnabled:     true,
		ActorUserID:   parseAliceAuth.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseSetAdminBillingAccessOverrideByUser: %v", parseErr)
	}
	if parseAccessOverrideRow.OverrideKey != "access.support_priority" || !parseAccessOverrideRow.IsEnabled {
		parseT.Fatalf("unexpected access override row: %+v", parseAccessOverrideRow)
	}

	parseQuotaOverrideRow, parseErr := parseStore.parseSetAdminBillingQuotaOverrideByUser(parseAliceAuth.ID, parseBillingAccessOverrideWrite{
		OverrideKey:   "tokens.monthly",
		OverrideValue: "500000",
		Reason:        "support granted quota headroom",
		IsEnabled:     true,
		ActorUserID:   parseAliceAuth.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("parseSetAdminBillingQuotaOverrideByUser: %v", parseErr)
	}
	if !strings.HasPrefix(parseQuotaOverrideRow.OverrideKey, "quota.") {
		parseT.Fatalf("expected normalized quota override key prefix, got %+v", parseQuotaOverrideRow)
	}

	parseOverrides, parseErr := parseStore.parseListAdminBillingAccessOverridesByUser(parseAliceAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminBillingAccessOverridesByUser: %v", parseErr)
	}
	if !parseHasAdminBillingOverrideKey(parseOverrides, "access.support_priority") {
		parseT.Fatalf("expected access override in override list, got %+v", parseOverrides)
	}
	if !parseHasAdminBillingOverrideKey(parseOverrides, "quota.tokens.monthly") {
		parseT.Fatalf("expected quota override in override list, got %+v", parseOverrides)
	}

	parseResolvedInvoice, parseErr := parseStore.parseResolveAdminBillingFailedPaymentByUser(parseAliceAuth.ID, parseResolvedInvoiceID, parseAliceAuth.ID, "manual settlement")
	if parseErr != nil {
		parseT.Fatalf("parseResolveAdminBillingFailedPaymentByUser: %v", parseErr)
	}
	if !strings.EqualFold(parseResolvedInvoice.Status, "paid") {
		parseT.Fatalf("expected paid invoice status after resolution, got %+v", parseResolvedInvoice)
	}

	parseDunningEventsAfter, parseErr := parseStore.parseListAdminBillingDunningEventsByUser(parseAliceAuth.ID, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminBillingDunningEventsByUser after resolution: %v", parseErr)
	}
	if !parseHasResolvedAdminDunningEventForInvoice(parseDunningEventsAfter, parseResolvedInvoiceID) {
		parseT.Fatalf("expected resolved dunning event for invoice %d, rows=%+v", parseResolvedInvoiceID, parseDunningEventsAfter)
	}

	parseBillingEventsAfter, parseErr := parseStore.parseListAdminBillingEventsByUser(parseAliceAuth.ID, 50)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminBillingEventsByUser after mutations: %v", parseErr)
	}
	if !parseHasAdminBillingEventType(parseBillingEventsAfter, "billing.access_override.updated") {
		parseT.Fatalf("expected access override billing event, rows=%+v", parseBillingEventsAfter)
	}
	if !parseHasAdminBillingEventType(parseBillingEventsAfter, "billing.quota_override.updated") {
		parseT.Fatalf("expected quota override billing event, rows=%+v", parseBillingEventsAfter)
	}
	if !parseHasAdminBillingEventType(parseBillingEventsAfter, "invoice.payment_resolved") {
		parseT.Fatalf("expected failed-payment resolution billing event, rows=%+v", parseBillingEventsAfter)
	}
}

// TestAdminBillingInterventionRPCs verifies typed billing intervention RPC behavior for superuser and workspace-admin callers.
func TestAdminBillingInterventionRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-billing-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseBillingDunningResp, parseErr := parseServer.ListAdminBillingDunningEvents(parseAliceCtx, &chatpb.ListAdminBillingDunningEventsRequest{
		UserId: parseAliceAuth.ID,
		Limit:  25,
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingDunningEvents: %v", parseErr)
	}
	if len(parseBillingDunningResp.GetEvents()) == 0 || parseBillingDunningResp.GetEvents()[0].GetInvoiceId() <= 0 {
		parseT.Fatalf("expected seeded dunning events from rpc, got %+v", parseBillingDunningResp.GetEvents())
	}
	parseInvoiceID := parseBillingDunningResp.GetEvents()[0].GetInvoiceId()

	parseBillingEventsResp, parseErr := parseServer.ListAdminBillingEvents(parseAliceCtx, &chatpb.ListAdminBillingEventsRequest{
		UserId: parseAliceAuth.ID,
		Limit:  25,
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingEvents: %v", parseErr)
	}
	if len(parseBillingEventsResp.GetEvents()) == 0 {
		parseT.Fatalf("expected seeded billing events from rpc, got %+v", parseBillingEventsResp.GetEvents())
	}

	parseAccessResp, parseErr := parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   "access.priority_support",
		OverrideValue: "enabled",
		Reason:        "operator grant",
		IsEnabled:     true,
		Confirm:       true,
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminBillingAccessOverride: %v", parseErr)
	}
	if parseAccessResp.GetStatus() != "enabled" {
		parseT.Fatalf("expected enabled access override status, got %+v", parseAccessResp)
	}

	parseQuotaResp, parseErr := parseServer.SetAdminBillingQuotaOverride(parseAliceCtx, &chatpb.AdminBillingQuotaOverrideMutationRequest{
		UserId:     parseAliceAuth.ID,
		QuotaKey:   "requests.daily",
		QuotaValue: "100000",
		Reason:     "operator grant",
		IsEnabled:  true,
		Confirm:    true,
	})
	if parseErr != nil {
		parseT.Fatalf("SetAdminBillingQuotaOverride: %v", parseErr)
	}
	if !strings.HasPrefix(parseQuotaResp.GetQuotaKey(), "quota.") || parseQuotaResp.GetStatus() != "enabled" {
		parseT.Fatalf("expected enabled normalized quota override status, got %+v", parseQuotaResp)
	}

	parseResolveResp, parseErr := parseServer.ResolveAdminBillingFailedPayment(parseAliceCtx, &chatpb.ResolveAdminBillingFailedPaymentRequest{
		UserId:    parseAliceAuth.ID,
		InvoiceId: parseInvoiceID,
		Confirm:   true,
		Reason:    "manual payment settlement",
	})
	if parseErr != nil {
		parseT.Fatalf("ResolveAdminBillingFailedPayment: %v", parseErr)
	}
	if !strings.EqualFold(parseResolveResp.GetInvoiceStatus(), "paid") {
		parseT.Fatalf("expected paid invoice status from resolution rpc, got %+v", parseResolveResp)
	}

	parseOverridesResp, parseErr := parseServer.ListAdminBillingAccessOverrides(parseAliceCtx, &chatpb.ListAdminBillingAccessOverridesRequest{
		UserId: parseAliceAuth.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingAccessOverrides: %v", parseErr)
	}
	if len(parseOverridesResp.GetOverrides()) < 2 {
		parseT.Fatalf("expected at least two overrides after access+quota mutation, got %+v", parseOverridesResp.GetOverrides())
	}

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-admin-billing-scope")
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-billing-bob", parseBobAuth.ID, parseBobAuth.Email)
	if _, parseErr = parseServer.ListAdminBillingEvents(parseBobCtx, &chatpb.ListAdminBillingEventsRequest{
		UserId: parseAliceAuth.ID,
		Limit:  10,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("ListAdminBillingEvents out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
}

// TestAdminBillingOverrideMutationsRequireReasonAndConfirmation verifies billing override mutations fail closed without explicit confirmation and reason.
func TestAdminBillingOverrideMutationsRequireReasonAndConfirmation(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-billing-reason-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	if _, parseErr = parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   "access.priority_support",
		OverrideValue: "enabled",
		Confirm:       false,
		Reason:        "requires explicit confirm",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetAdminBillingAccessOverride missing confirm status code = %v, want %v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr = parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   "access.priority_support",
		OverrideValue: "enabled",
		Confirm:       true,
		Reason:        "   ",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetAdminBillingAccessOverride missing reason status code = %v, want %v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr = parseServer.SetAdminBillingQuotaOverride(parseAliceCtx, &chatpb.AdminBillingQuotaOverrideMutationRequest{
		UserId:     parseAliceAuth.ID,
		QuotaKey:   "requests.daily",
		QuotaValue: "100000",
		Confirm:    false,
		Reason:     "requires explicit confirm",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetAdminBillingQuotaOverride missing confirm status code = %v, want %v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr = parseServer.SetAdminBillingQuotaOverride(parseAliceCtx, &chatpb.AdminBillingQuotaOverrideMutationRequest{
		UserId:     parseAliceAuth.ID,
		QuotaKey:   "requests.daily",
		QuotaValue: "100000",
		Confirm:    true,
		Reason:     "",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetAdminBillingQuotaOverride missing reason status code = %v, want %v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestAdminBillingEventsWorkspaceScopeRedactsPayload verifies workspace-admin billing event views redact raw provider payload JSON.
func TestAdminBillingEventsWorkspaceScopeRedactsPayload(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-billing-redaction")
	parseMustAssignBillingPlan(parseT, parseStore, parseBobAuth.ID, "free")
	parseBobCustomer, hasParseCustomer, parseErr := parseStore.parseGetBillingCustomerByUser(parseBobAuth.ID)
	if parseErr != nil || !hasParseCustomer {
		parseT.Fatalf("parseGetBillingCustomerByUser bob: customer=%+v found=%v err=%v", parseBobCustomer, hasParseCustomer, parseErr)
	}
	parseSubscriptions, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseBobCustomer.ID, 1)
	if parseErr != nil || len(parseSubscriptions) == 0 {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer bob: rows=%+v err=%v", parseSubscriptions, parseErr)
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseInvoice, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseBobCustomer.ID,
		SubscriptionID:    parseSubscriptions[0].ID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-bob-redaction",
		Status:            "open",
		Currency:          "usd",
		SubtotalCents:     500,
		TotalCents:        500,
		AmountDueCents:    500,
		AmountPaidCents:   0,
		PeriodStart:       parseNow,
		PeriodEnd:         parseNow,
		DueAt:             parseNow,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:       parseBobCustomer.ID,
		SubscriptionID:   parseSubscriptions[0].ID,
		InvoiceID:        parseInvoice.ID,
		EventType:        "invoice.payment_failed",
		EventSource:      "provider",
		EventSummary:     "payment failed with provider payload",
		EventPayloadJSON: `{"provider_error":"raw-card-declined-payload"}`,
		ActorUserID:      parseBobAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent: %v", parseErr)
	}

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-billing-redaction-bob", parseBobAuth.ID, parseBobAuth.Email)
	parseEventsResp, parseErr := parseServer.ListAdminBillingEvents(parseBobCtx, &chatpb.ListAdminBillingEventsRequest{
		UserId: parseBobAuth.ID,
		Limit:  25,
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingEvents workspace-admin: %v", parseErr)
	}
	if len(parseEventsResp.GetEvents()) == 0 {
		parseT.Fatalf("expected billing events for workspace-admin redaction check, got %+v", parseEventsResp.GetEvents())
	}
	for _, parseEventEntry := range parseEventsResp.GetEvents() {
		if parseEventEntry.GetEventPayloadJson() != "{}" {
			parseT.Fatalf("expected workspace-scoped billing payload redaction, got %+v", parseEventEntry)
		}
	}
}

// TestAdminBillingListQueryRPCs verifies typed billing list-query behavior for override, event, and dunning list RPCs.
func TestAdminBillingListQueryRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-billing-list-query-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	if _, parseErr = parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   "access.alpha",
		OverrideValue: "enabled",
		Reason:        "alpha grant",
		IsEnabled:     true,
		Confirm:       true,
	}); parseErr != nil {
		parseT.Fatalf("SetAdminBillingAccessOverride alpha: %v", parseErr)
	}
	if _, parseErr = parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   "access.beta",
		OverrideValue: "enabled",
		Reason:        "beta grant",
		IsEnabled:     true,
		Confirm:       true,
	}); parseErr != nil {
		parseT.Fatalf("SetAdminBillingAccessOverride beta: %v", parseErr)
	}

	parseOverridesResp, parseErr := parseServer.ListAdminBillingAccessOverrides(parseAliceCtx, &chatpb.ListAdminBillingAccessOverridesRequest{
		UserId:      parseAliceAuth.ID,
		OverrideKey: "access.beta",
		ListQuery: &chatpb.AdminListQuery{
			Search:        "beta",
			SortBy:        "override_key",
			SortDirection: "asc",
			Limit:         1,
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingAccessOverrides list query: %v", parseErr)
	}
	if len(parseOverridesResp.GetOverrides()) != 1 || parseOverridesResp.GetOverrides()[0].GetOverrideKey() != "access.beta" {
		parseT.Fatalf("expected one filtered override row for access.beta, got %+v", parseOverridesResp.GetOverrides())
	}

	parseEventsResp, parseErr := parseServer.ListAdminBillingEvents(parseAliceCtx, &chatpb.ListAdminBillingEventsRequest{
		UserId:    parseAliceAuth.ID,
		EventType: "billing.access_override.updated",
		ListQuery: &chatpb.AdminListQuery{
			SortBy:        "id",
			SortDirection: "asc",
			Limit:         1,
			Offset:        1,
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingEvents list query: %v", parseErr)
	}
	if len(parseEventsResp.GetEvents()) != 1 || !strings.EqualFold(parseEventsResp.GetEvents()[0].GetEventType(), "billing.access_override.updated") {
		parseT.Fatalf("expected one paged access-override event row, got %+v", parseEventsResp.GetEvents())
	}
	if !strings.Contains(strings.ToLower(parseEventsResp.GetEvents()[0].GetEventSummary()), "beta") {
		parseT.Fatalf("expected paged event summary to include beta marker, got %+v", parseEventsResp.GetEvents()[0])
	}

	parseCustomer, hasParseCustomer, parseErr := parseStore.parseGetBillingCustomerByUser(parseAliceAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !hasParseCustomer {
		parseT.Fatalf("expected billing customer for user id %d", parseAliceAuth.ID)
	}
	parseSubscriptions, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 1)
	if parseErr != nil || len(parseSubscriptions) == 0 {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: rows=%+v err=%v", parseSubscriptions, parseErr)
	}
	parseInvoices, parseErr := parseStore.parseListBillingInvoicesByCustomer(parseCustomer.ID, 1)
	if parseErr != nil || len(parseInvoices) == 0 {
		parseT.Fatalf("parseListBillingInvoicesByCustomer: rows=%+v err=%v", parseInvoices, parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO billing_dunning_events (customer_id, subscription_id, invoice_id, status, attempt_count, failure_reason, next_attempt_at, resolved_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		parseCustomer.ID,
		parseSubscriptions[0].ID,
		parseInvoices[0].ID,
		"pending",
		2,
		"manual_followup",
		"2026-03-28T01:00:00Z",
		"",
		"2026-03-28T00:00:00Z",
		"2026-03-28T00:00:00Z",
	); parseErr != nil {
		parseT.Fatalf("seed billing_dunning_events list query row: %v", parseErr)
	}

	parseDunningResp, parseErr := parseServer.ListAdminBillingDunningEvents(parseAliceCtx, &chatpb.ListAdminBillingDunningEventsRequest{
		UserId: parseAliceAuth.ID,
		Status: "pending",
		ListQuery: &chatpb.AdminListQuery{
			Search:        "manual_followup",
			SortBy:        "attempt_count",
			SortDirection: "desc",
			Limit:         1,
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingDunningEvents list query: %v", parseErr)
	}
	if len(parseDunningResp.GetEvents()) != 1 || !strings.Contains(strings.ToLower(parseDunningResp.GetEvents()[0].GetFailureReason()), "manual_followup") {
		parseT.Fatalf("expected one filtered dunning row for manual_followup, got %+v", parseDunningResp.GetEvents())
	}
}

// TestAdminBillingListRPCsApplyTypedListQuery verifies billing list RPC search, filter, sort, and pagination behavior.
func TestAdminBillingListRPCsApplyTypedListQuery(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-billing-list-query-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseCustomer, isParseCustomerFound, parseErr := parseStore.parseGetBillingCustomerByUser(parseAliceAuth.ID)
	if parseErr != nil || !isParseCustomerFound {
		parseT.Fatalf("parseGetBillingCustomerByUser: found=%v row=%+v err=%v", isParseCustomerFound, parseCustomer, parseErr)
	}
	parseSubscriptions, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 10)
	if parseErr != nil || len(parseSubscriptions) == 0 {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: rows=%+v err=%v", parseSubscriptions, parseErr)
	}
	parseInvoices, parseErr := parseStore.parseListBillingInvoicesByCustomer(parseCustomer.ID, 10)
	if parseErr != nil || len(parseInvoices) == 0 {
		parseT.Fatalf("parseListBillingInvoicesByCustomer: rows=%+v err=%v", parseInvoices, parseErr)
	}
	parseSubscriptionID := parseSubscriptions[0].ID
	parseInvoiceID := parseInvoices[0].ID
	parseNow := time.Now().UTC().Format(time.RFC3339)

	if _, parseErr = parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   "access.listquery.alpha",
		OverrideValue: "enabled",
		Reason:        "list-query seed alpha",
		IsEnabled:     true,
		Confirm:       true,
	}); parseErr != nil {
		parseT.Fatalf("SetAdminBillingAccessOverride alpha: %v", parseErr)
	}
	if _, parseErr = parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   "access.listquery.beta",
		OverrideValue: "enabled",
		Reason:        "list-query seed beta",
		IsEnabled:     true,
		Confirm:       true,
	}); parseErr != nil {
		parseT.Fatalf("SetAdminBillingAccessOverride beta: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:     parseCustomer.ID,
		SubscriptionID: parseSubscriptionID,
		InvoiceID:      parseInvoiceID,
		EventType:      "invoice.payment_failed",
		EventSource:    "operator",
		EventSummary:   "list-query-event-alpha",
		ActorUserID:    parseAliceAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent alpha: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:     parseCustomer.ID,
		SubscriptionID: parseSubscriptionID,
		InvoiceID:      parseInvoiceID,
		EventType:      "invoice.payment_failed",
		EventSource:    "operator",
		EventSummary:   "list-query-event-beta",
		ActorUserID:    parseAliceAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent beta: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO billing_dunning_events (customer_id, subscription_id, invoice_id, status, attempt_count, failure_reason, next_attempt_at, resolved_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		parseCustomer.ID,
		parseSubscriptionID,
		parseInvoiceID,
		"pending",
		1,
		"list-query-dunning-alpha",
		parseNow,
		"",
		parseNow,
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed billing_dunning_events alpha: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(
		`INSERT INTO billing_dunning_events (customer_id, subscription_id, invoice_id, status, attempt_count, failure_reason, next_attempt_at, resolved_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		parseCustomer.ID,
		parseSubscriptionID,
		parseInvoiceID,
		"pending",
		2,
		"list-query-dunning-beta",
		parseNow,
		"",
		parseNow,
		parseNow,
	); parseErr != nil {
		parseT.Fatalf("seed billing_dunning_events beta: %v", parseErr)
	}

	parseOverrideResp, parseErr := parseServer.ListAdminBillingAccessOverrides(parseAliceCtx, &chatpb.ListAdminBillingAccessOverridesRequest{
		UserId: parseAliceAuth.ID,
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "access.listquery",
			SortBy:        "override_key",
			SortDirection: "asc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingAccessOverrides list query: %v", parseErr)
	}
	if len(parseOverrideResp.GetOverrides()) != 1 || parseOverrideResp.GetOverrides()[0].GetOverrideKey() != "access.listquery.beta" {
		parseT.Fatalf("unexpected override list-query rows: %+v", parseOverrideResp.GetOverrides())
	}

	parseEventResp, parseErr := parseServer.ListAdminBillingEvents(parseAliceCtx, &chatpb.ListAdminBillingEventsRequest{
		UserId:    parseAliceAuth.ID,
		EventType: "invoice.payment_failed",
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "list-query-event",
			SortBy:        "event_summary",
			SortDirection: "asc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingEvents list query: %v", parseErr)
	}
	if len(parseEventResp.GetEvents()) != 1 || parseEventResp.GetEvents()[0].GetEventSummary() != "list-query-event-beta" {
		parseT.Fatalf("unexpected billing-event list-query rows: %+v", parseEventResp.GetEvents())
	}

	parseDunningResp, parseErr := parseServer.ListAdminBillingDunningEvents(parseAliceCtx, &chatpb.ListAdminBillingDunningEventsRequest{
		UserId: parseAliceAuth.ID,
		Status: "pending",
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        1,
			Search:        "list-query-dunning",
			SortBy:        "attempt_count",
			SortDirection: "asc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingDunningEvents list query: %v", parseErr)
	}
	if len(parseDunningResp.GetEvents()) != 1 || parseDunningResp.GetEvents()[0].GetAttemptCount() != 2 {
		parseT.Fatalf("unexpected dunning list-query rows: %+v", parseDunningResp.GetEvents())
	}
}

// TestAdminBillingDunningFallbackListQuery verifies fallback dunning rows map and sort correctly when table rows are unavailable.
func TestAdminBillingDunningFallbackListQuery(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-billing-dunning-fallback", parseAliceAuth.ID, parseAliceAuth.Email)

	parseCustomer, hasParseCustomer, parseErr := parseStore.parseGetBillingCustomerByUser(parseAliceAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !hasParseCustomer {
		parseT.Fatalf("expected billing customer for user %d", parseAliceAuth.ID)
	}
	parseSubscriptionRows, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 1)
	if parseErr != nil || len(parseSubscriptionRows) == 0 {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: rows=%+v err=%v", parseSubscriptionRows, parseErr)
	}
	parseSubscriptionID := parseSubscriptionRows[0].ID
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseFallbackInvoiceAlpha, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomer.ID,
		SubscriptionID:    parseSubscriptionID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-dunning-fallback-001",
		Status:            "open",
		Currency:          "usd",
		SubtotalCents:     1500,
		TotalCents:        1500,
		AmountDueCents:    1500,
		AmountPaidCents:   0,
		PeriodStart:       parseNow,
		PeriodEnd:         parseNow,
		DueAt:             parseNow,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice fallback alpha: %v", parseErr)
	}
	parseFallbackInvoiceBeta, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomer.ID,
		SubscriptionID:    parseSubscriptionID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-dunning-fallback-002",
		Status:            "open",
		Currency:          "usd",
		SubtotalCents:     2000,
		TotalCents:        2000,
		AmountDueCents:    2000,
		AmountPaidCents:   0,
		PeriodStart:       parseNow,
		PeriodEnd:         parseNow,
		DueAt:             parseNow,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice fallback beta: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:     parseCustomer.ID,
		SubscriptionID: parseSubscriptionID,
		InvoiceID:      parseFallbackInvoiceBeta.ID,
		EventType:      "invoice.payment_failed.resolved",
		EventSource:    "provider",
		EventSummary:   "fallback-beta",
		ActorUserID:    parseAliceAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent fallback-beta: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:     parseCustomer.ID,
		SubscriptionID: parseSubscriptionID,
		InvoiceID:      parseFallbackInvoiceAlpha.ID,
		EventType:      "invoice.payment_failed",
		EventSource:    "provider",
		EventSummary:   "fallback-alpha",
		ActorUserID:    parseAliceAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent fallback-alpha: %v", parseErr)
	}
	if _, parseErr = parseStore.db.Exec(`DELETE FROM billing_dunning_events WHERE customer_id = ?`, parseCustomer.ID); parseErr != nil {
		parseT.Fatalf("delete billing_dunning_events: %v", parseErr)
	}

	parseDunningResp, parseErr := parseServer.ListAdminBillingDunningEvents(parseAliceCtx, &chatpb.ListAdminBillingDunningEventsRequest{
		UserId: parseAliceAuth.ID,
		ListQuery: &chatpb.AdminListQuery{
			Search:        "fallback-",
			SortBy:        "invoice_id",
			SortDirection: "desc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingDunningEvents fallback: %v", parseErr)
	}
	if len(parseDunningResp.GetEvents()) != 2 {
		parseT.Fatalf("expected two fallback dunning rows, got %+v", parseDunningResp.GetEvents())
	}
	if parseDunningResp.GetEvents()[0].GetInvoiceId() < parseDunningResp.GetEvents()[1].GetInvoiceId() {
		parseT.Fatalf("expected fallback dunning rows sorted by invoice id desc, got %+v", parseDunningResp.GetEvents())
	}
	parseHasResolvedFallback := false
	for _, parseEvent := range parseDunningResp.GetEvents() {
		if !strings.Contains(strings.ToLower(parseEvent.GetFailureReason()), "fallback-beta") {
			continue
		}
		parseHasResolvedFallback = true
		if parseEvent.GetStatus() != "resolved" || parseEvent.GetResolvedAt() == "" {
			parseT.Fatalf("expected resolved fallback row from event mapping, got %+v", parseEvent)
		}
	}
	if !parseHasResolvedFallback {
		parseT.Fatalf("expected resolved fallback row from invoice.payment_failed.resolved event, got %+v", parseDunningResp.GetEvents())
	}
}

// TestBuildAdminBillingDunningEventEntryFromBillingEvent verifies fallback dunning-entry mapping for open and resolved billing-event types.
func TestBuildAdminBillingDunningEventEntryFromBillingEvent(parseT *testing.T) {
	parseOpenEntry := parseBuildAdminBillingDunningEventEntryFromBillingEvent(parseBillingEventRow{
		ID:             10,
		CustomerID:     20,
		SubscriptionID: 30,
		InvoiceID:      40,
		EventType:      "invoice.payment_failed",
		EventSummary:   "card declined",
		CreatedAt:      "2026-03-28T00:00:00Z",
	})
	if parseOpenEntry.GetStatus() != "open" || parseOpenEntry.GetResolvedAt() != "" || parseOpenEntry.GetFailureReason() != "card declined" {
		parseT.Fatalf("unexpected open fallback dunning entry: %+v", parseOpenEntry)
	}

	parseResolvedEntry := parseBuildAdminBillingDunningEventEntryFromBillingEvent(parseBillingEventRow{
		ID:             11,
		CustomerID:     21,
		SubscriptionID: 31,
		InvoiceID:      41,
		EventType:      "invoice.payment_failed.resolved",
		EventSummary:   "manual settlement completed",
		CreatedAt:      "2026-03-28T01:00:00Z",
	})
	if parseResolvedEntry.GetStatus() != "resolved" || parseResolvedEntry.GetResolvedAt() != "2026-03-28T01:00:00Z" {
		parseT.Fatalf("unexpected resolved fallback dunning entry: %+v", parseResolvedEntry)
	}
}

// TestSortAdminBillingDunningEntryRows verifies fallback dunning-entry sorting for typed keys and nil-row safety.
func TestSortAdminBillingDunningEntryRows(parseT *testing.T) {
	parseRows := []*chatpb.AdminBillingDunningEventEntry{
		{Id: 3, InvoiceId: 300, Status: "open", CreatedAt: "2026-03-28T03:00:00Z", UpdatedAt: "2026-03-28T03:00:00Z"},
		{Id: 1, InvoiceId: 100, Status: "resolved", CreatedAt: "2026-03-28T01:00:00Z", UpdatedAt: "2026-03-28T01:00:00Z"},
		{Id: 2, InvoiceId: 200, Status: "pending", CreatedAt: "2026-03-28T02:00:00Z", UpdatedAt: "2026-03-28T02:00:00Z"},
	}
	parseSortAdminBillingDunningEntryRows(parseRows, "invoice_id", true)
	if parseRows[0].GetInvoiceId() != 100 || parseRows[1].GetInvoiceId() != 200 || parseRows[2].GetInvoiceId() != 300 {
		parseT.Fatalf("unexpected asc invoice sort order: %+v", parseRows)
	}

	parseSortAdminBillingDunningEntryRows(parseRows, "created_at", false)
	if parseRows[0].GetCreatedAt() != "2026-03-28T03:00:00Z" || parseRows[2].GetCreatedAt() != "2026-03-28T01:00:00Z" {
		parseT.Fatalf("unexpected desc created_at sort order: %+v", parseRows)
	}

	parseNilRows := []*chatpb.AdminBillingDunningEventEntry{nil, nil}
	parseSortAdminBillingDunningEntryRows(parseNilRows, "id", true)
	if len(parseNilRows) != 2 {
		parseT.Fatalf("expected nil row slice length to remain stable, got %d", len(parseNilRows))
	}
}

// BenchmarkParseNormalizeAdminBillingQuotaOverrideKey reports micro-benchmark throughput for quota-key normalization.
func BenchmarkParseNormalizeAdminBillingQuotaOverrideKey(parseB *testing.B) {
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseNormalizeAdminBillingQuotaOverrideKey("requests.daily")
	}
}

// BenchmarkParseFilterAdminBillingEventRows reports micro-benchmark throughput for billing-event list-query filtering.
func BenchmarkParseFilterAdminBillingEventRows(parseB *testing.B) {
	parseRows := []parseBillingEventRow{
		{
			ID:             1,
			CustomerID:     10,
			SubscriptionID: 11,
			InvoiceID:      12,
			EventType:      "billing.access_override.updated",
			EventSource:    "admin",
			EventSummary:   "alpha grant",
			ActorUserID:    20,
			CreatedAt:      "2026-03-28T00:00:00Z",
		},
		{
			ID:             2,
			CustomerID:     10,
			SubscriptionID: 11,
			InvoiceID:      12,
			EventType:      "invoice.payment_failed",
			EventSource:    "provider",
			EventSummary:   "card declined",
			ActorUserID:    20,
			CreatedAt:      "2026-03-28T00:01:00Z",
		},
	}
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseFilterAdminBillingEventRows(parseRows, "billing.access_override.updated", "alpha")
	}
}

// parseHasAdminBillingOverrideKey reports whether one override slice contains one normalized override key.
func parseHasAdminBillingOverrideKey(parseRows []parseBillingAccessOverrideRow, parseOverrideKey string) bool {
	parseOverrideKey = strings.TrimSpace(strings.ToLower(parseOverrideKey))
	for _, parseRow := range parseRows {
		if strings.TrimSpace(strings.ToLower(parseRow.OverrideKey)) == parseOverrideKey {
			return true
		}
	}
	return false
}

// parseHasResolvedAdminDunningEventForInvoice reports whether one dunning slice contains one invoice row in resolved state.
func parseHasResolvedAdminDunningEventForInvoice(parseRows []parseBillingDunningEventRow, parseInvoiceID int64) bool {
	for _, parseRow := range parseRows {
		if parseRow.InvoiceID != parseInvoiceID {
			continue
		}
		if strings.EqualFold(parseRow.Status, "resolved") {
			return true
		}
	}
	return false
}

// parseHasAdminBillingEventType reports whether one billing-event slice contains one target event type.
func parseHasAdminBillingEventType(parseRows []parseBillingEventRow, parseEventType string) bool {
	parseEventType = strings.TrimSpace(strings.ToLower(parseEventType))
	for _, parseRow := range parseRows {
		if strings.TrimSpace(strings.ToLower(parseRow.EventType)) == parseEventType {
			return true
		}
	}
	return false
}
