package app

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestBillingInterventionOpsSuperuserFlow verifies billing intervention snapshot, override apply, and failed-payment resolution for superusers.
func TestBillingInterventionOpsSuperuserFlow(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseSuperuser := parseMustCreateUser(parseT, parseStore, "billing-superuser@example.com")
	parseTargetUser := parseMustCreateUser(parseT, parseStore, "billing-target@example.com")
	parseGrantSuperuserRole(parseT, parseStore, parseSuperuser.ID)
	parseMustAssignBillingPlan(parseT, parseStore, parseTargetUser.ID, "free")

	parseCustomerRow, isParseCustomerFound, parseErr := parseStore.parseGetBillingCustomerByUser(parseTargetUser.ID)
	if parseErr != nil || !isParseCustomerFound {
		parseT.Fatalf("parseGetBillingCustomerByUser: customer=%+v found=%v err=%v", parseCustomerRow, isParseCustomerFound, parseErr)
	}
	parseSubscriptionRows, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomerRow.ID, 1)
	if parseErr != nil || len(parseSubscriptionRows) == 0 {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: rows=%+v err=%v", parseSubscriptionRows, parseErr)
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseInvoiceRow, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomerRow.ID,
		SubscriptionID:    parseSubscriptionRows[0].ID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-billing-intervention-1",
		Status:            "open",
		Currency:          "usd",
		SubtotalCents:     1000,
		TotalCents:        1000,
		AmountDueCents:    1000,
		AmountPaidCents:   0,
		PeriodStart:       parseNow,
		PeriodEnd:         parseNow,
		DueAt:             parseNow,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice: %v", parseErr)
	}
	if _, parseErr = parseStore.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:       parseCustomerRow.ID,
		SubscriptionID:   parseSubscriptionRows[0].ID,
		InvoiceID:        parseInvoiceRow.ID,
		EventType:        "dunning.email_sent",
		EventSource:      "system",
		EventSummary:     "Dunning notice sent",
		EventPayloadJSON: "{}",
		ActorUserID:      parseSuperuser.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent dunning: %v", parseErr)
	}

	parseCtx := parseBindAuthUser(parseServer, "peer-billing-intervention-superuser", parseSuperuser.ID, parseSuperuser.Email)
	parseSnapshot, parseErr := parseServer.parseGetBillingInterventionSnapshotByAdmin(parseCtx, parseTargetUser.ID, 20)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingInterventionSnapshotByAdmin: %v", parseErr)
	}
	if parseSnapshot.parseCustomerRow.ID != parseCustomerRow.ID {
		parseT.Fatalf("expected billing customer id=%d, got %+v", parseCustomerRow.ID, parseSnapshot.parseCustomerRow)
	}
	if len(parseSnapshot.parseInvoiceRows) == 0 || len(parseSnapshot.parseEventRows) == 0 || len(parseSnapshot.parseDunningEventRows) == 0 {
		parseT.Fatalf("expected invoice/event/dunning rows, got invoices=%d events=%d dunning=%d", len(parseSnapshot.parseInvoiceRows), len(parseSnapshot.parseEventRows), len(parseSnapshot.parseDunningEventRows))
	}

	if parseErr = parseServer.parseApplyBillingAccessOverrideByAdmin(
		parseCtx,
		parseTargetUser.ID,
		"usage.monthly_token_limit",
		"999999",
		"temporary enterprise grace period",
		true,
		"",
		"",
	); parseErr != nil {
		parseT.Fatalf("parseApplyBillingAccessOverrideByAdmin: %v", parseErr)
	}
	parseOverrideRows, parseErr := parseStore.parseListBillingAccessOverridesByCustomer(parseCustomerRow.ID)
	if parseErr != nil || len(parseOverrideRows) == 0 {
		parseT.Fatalf("parseListBillingAccessOverridesByCustomer: rows=%+v err=%v", parseOverrideRows, parseErr)
	}

	if parseErr = parseServer.parseResolveBillingFailedPaymentByAdmin(
		parseCtx,
		parseTargetUser.ID,
		parseInvoiceRow.ID,
		"Payment method updated and invoice settled",
	); parseErr != nil {
		parseT.Fatalf("parseResolveBillingFailedPaymentByAdmin: %v", parseErr)
	}
	parseResolvedInvoice, isParseInvoiceFound, parseErr := parseStore.parseGetBillingInvoiceByProvider(parseInvoiceRow.ProviderInvoiceID)
	if parseErr != nil || !isParseInvoiceFound {
		parseT.Fatalf("parseGetBillingInvoiceByProvider resolved: row=%+v found=%v err=%v", parseResolvedInvoice, isParseInvoiceFound, parseErr)
	}
	if parseResolvedInvoice.Status != "paid" || parseResolvedInvoice.AmountDueCents != 0 {
		parseT.Fatalf("expected paid invoice with zero due, got %+v", parseResolvedInvoice)
	}
	parseDunningRows, parseErr := parseServer.parseListBillingDunningEventsByAdmin(parseCtx, parseTargetUser.ID, 20)
	if parseErr != nil || len(parseDunningRows) == 0 {
		parseT.Fatalf("parseListBillingDunningEventsByAdmin: rows=%+v err=%v", parseDunningRows, parseErr)
	}
}

// TestBillingInterventionOpsWorkspaceScopeDenied verifies workspace-admin interventions fail closed for out-of-scope users.
func TestBillingInterventionOpsWorkspaceScopeDenied(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseWorkspaceAdmin := parseMustCreateUser(parseT, parseStore, "billing-workspace-admin@example.com")
	parseScopedUser := parseMustCreateUser(parseT, parseStore, "billing-scoped-user@example.com")
	parseOutOfScopeUser := parseMustCreateUser(parseT, parseStore, "billing-outscope-user@example.com")
	parseWorkspaceID := parseMustEnsureWorkspaceMembership(parseT, parseStore, parseWorkspaceAdmin.ID, "ws-billing-intervention")
	if parseErr := parseStore.parseUpsertWorkspaceMembership(parseWorkspaceMembershipWrite{
		WorkspaceID:     parseWorkspaceID,
		UserID:          parseScopedUser.ID,
		RoleKey:         "member",
		Status:          "active",
		InvitedByUserID: parseWorkspaceAdmin.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspaceMembership scoped user: %v", parseErr)
	}
	parseMustEnsureWorkspaceMembership(parseT, parseStore, parseOutOfScopeUser.ID, "ws-billing-intervention-outscope")
	parseMustAssignBillingPlan(parseT, parseStore, parseScopedUser.ID, "free")
	parseMustAssignBillingPlan(parseT, parseStore, parseOutOfScopeUser.ID, "free")

	parseCtx := parseBindAuthUser(parseServer, "peer-billing-workspace-admin", parseWorkspaceAdmin.ID, parseWorkspaceAdmin.Email)
	if _, parseErr := parseServer.parseGetBillingInterventionSnapshotByAdmin(parseCtx, parseOutOfScopeUser.ID, 10); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("parseGetBillingInterventionSnapshotByAdmin out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if parseErr := parseServer.parseApplyBillingAccessOverrideByAdmin(
		parseCtx,
		parseOutOfScopeUser.ID,
		"usage.monthly_token_limit",
		"5000",
		"out-of-scope should fail",
		true,
		"",
		"",
	); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("parseApplyBillingAccessOverrideByAdmin out-of-scope status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
}
