package app

import (
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestBuildUsageBasedBillingPreview verifies platform-fee + usage-cost + service-premium preview math.
func TestBuildUsageBasedBillingPreview(parseT *testing.T) {
	parsePreview, parseErr := parseBuildUsageBasedBillingPreview(2900, 1200, 500)
	if parseErr != nil {
		parseT.Fatalf("parseBuildUsageBasedBillingPreview: %v", parseErr)
	}
	if parsePreview.platformFeeCents != 2900 || parsePreview.usageCostCents != 1200 || parsePreview.servicePremiumCents != 60 || parsePreview.totalCents != 4160 {
		parseT.Fatalf("unexpected preview: %+v", parsePreview)
	}
	if parseSummary := parseBuildUsageBasedBillingSummary(parsePreview); parseSummary == "" {
		parseT.Fatal("expected non-empty usage-based billing summary")
	}
}

// TestValidateUsageBasedBillingPlanWrite verifies active non-enterprise plans require a non-zero platform fee.
func TestValidateUsageBasedBillingPlanWrite(parseT *testing.T) {
	if parseErr := parseValidateUsageBasedBillingPlanWrite(parseSuperuserBillingPlanWrite{
		PlanCode:              "pro",
		IsActive:              true,
		MonthlyBaseCents:      0,
		YearlyBaseCents:       0,
		IncludedTokensMonthly: 1000,
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("active plan without platform fee status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if parseErr := parseValidateUsageBasedBillingPlanWrite(parseSuperuserBillingPlanWrite{
		PlanCode:              "enterprise",
		IsActive:              true,
		MonthlyBaseCents:      0,
		YearlyBaseCents:       0,
		IncludedTokensMonthly: 0,
	}); parseErr != nil {
		parseT.Fatalf("enterprise zero platform fee expected allow: %v", parseErr)
	}
}

// TestValidateUsageBasedBillingOverrideWrite verifies overrides cannot reintroduce unlimited/flat-rate assumptions.
func TestValidateUsageBasedBillingOverrideWrite(parseT *testing.T) {
	if parseErr := parseValidateUsageBasedBillingOverrideWrite("billing.flat_rate.enabled", "true"); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("flat-rate override status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if parseErr := parseValidateUsageBasedBillingOverrideWrite(billingEntitlementUsageMonthlyTokenLimit, "unlimited"); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("monthly token unlimited override status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if parseErr := parseValidateUsageBasedBillingOverrideWrite("chat.send.enabled", "true"); parseErr != nil {
		parseT.Fatalf("standard access override expected allow: %v", parseErr)
	}
}

// TestValidatePlanBoundaryPlanWrite verifies canonical Pro/Team/Enterprise plan boundary enforcement.
func TestValidatePlanBoundaryPlanWrite(parseT *testing.T) {
	if parseErr := parseValidatePlanBoundaryPlanWrite(parseSuperuserBillingPlanWrite{
		PlanCode:              "pro",
		IncludedSeats:         1,
		MaxSeats:              1,
		SupportsTeamWorkspace: false,
		SupportsSSO:           false,
	}); parseErr != nil {
		parseT.Fatalf("pro boundary expected allow: %v", parseErr)
	}
	if parseErr := parseValidatePlanBoundaryPlanWrite(parseSuperuserBillingPlanWrite{
		PlanCode:              "pro",
		IncludedSeats:         1,
		MaxSeats:              3,
		SupportsTeamWorkspace: true,
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("pro boundary violation status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if parseErr := parseValidatePlanBoundaryPlanWrite(parseSuperuserBillingPlanWrite{
		PlanCode:              "team",
		IncludedSeats:         3,
		MaxSeats:              50,
		SupportsTeamWorkspace: false,
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("team collaboration boundary status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if parseErr := parseValidatePlanBoundaryPlanWrite(parseSuperuserBillingPlanWrite{
		PlanCode:              "enterprise",
		IncludedSeats:         10,
		MaxSeats:              500,
		SupportsTeamWorkspace: true,
		SupportsSSO:           true,
	}); parseErr != nil {
		parseT.Fatalf("enterprise boundary expected allow: %v", parseErr)
	}
	if parseErr := parseValidatePlanBoundaryPlanWrite(parseSuperuserBillingPlanWrite{
		PlanCode:              "enterprise",
		IncludedSeats:         5,
		MaxSeats:              5,
		SupportsTeamWorkspace: true,
		SupportsSSO:           false,
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("enterprise contract boundary status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestValidatePlanBoundaryEntitlementWrite verifies entitlement-level boundary constraints for canonical plans.
func TestValidatePlanBoundaryEntitlementWrite(parseT *testing.T) {
	if parseErr := parseValidatePlanBoundaryEntitlementWrite(parseSuperuserBillingPlanEntitlementWrite{
		PlanCode:         "pro",
		EntitlementKey:   "workspace.multi_user.enabled",
		EntitlementValue: "true",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("pro multi-user entitlement status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if parseErr := parseValidatePlanBoundaryEntitlementWrite(parseSuperuserBillingPlanEntitlementWrite{
		PlanCode:         "team",
		EntitlementKey:   "workspace.multi_user.enabled",
		EntitlementValue: "true",
	}); parseErr != nil {
		parseT.Fatalf("team multi-user entitlement expected allow: %v", parseErr)
	}
	if parseErr := parseValidatePlanBoundaryEntitlementWrite(parseSuperuserBillingPlanEntitlementWrite{
		PlanCode:         "team",
		EntitlementKey:   "sso.enabled",
		EntitlementValue: "true",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("team sso entitlement status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if parseErr := parseValidatePlanBoundaryEntitlementWrite(parseSuperuserBillingPlanEntitlementWrite{
		PlanCode:         "enterprise",
		EntitlementKey:   "sso.enabled",
		EntitlementValue: "true",
	}); parseErr != nil {
		parseT.Fatalf("enterprise sso entitlement expected allow: %v", parseErr)
	}
}

// TestCreateBillingInvoiceLineItemUsageBasedRules verifies invoice line items enforce normalized usage-based line taxonomy and amount math.
func TestCreateBillingInvoiceLineItemUsageBasedRules(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "billing-line-rules@example.com")
	parseNow := time.Now().UTC()

	parseCustomer, parseErr := parseStore.parseUpsertBillingCustomer(parseBillingCustomerWrite{
		UserID:             parseUser.ID,
		ProviderID:         "stripe",
		ProviderCustomerID: "cus-line-rules",
		DefaultCurrency:    "usd",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingCustomer: %v", parseErr)
	}
	parseSubscription, parseErr := parseStore.parseUpsertBillingSubscription(parseBillingSubscriptionWrite{
		CustomerID:             parseCustomer.ID,
		ProviderID:             "stripe",
		ProviderSubscriptionID: "sub-line-rules",
		PlanCode:               "team",
		Status:                 "active",
		BillingInterval:        "month",
		CurrentPeriodStart:     parseNow.Format(time.RFC3339),
		CurrentPeriodEnd:       parseNow.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingSubscription: %v", parseErr)
	}
	parseInvoice, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomer.ID,
		SubscriptionID:    parseSubscription.ID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-line-rules",
		Status:            "open",
		Currency:          "usd",
		TotalCents:        0,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice: %v", parseErr)
	}

	if _, parseErr = parseStore.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseCustomer.ID,
		InvoiceID:       parseInvoice.ID,
		LineType:        "opaque_total",
		Description:     "invalid type",
		Quantity:        1,
		UnitAmountCents: 100,
		AmountCents:     100,
		Currency:        "usd",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("invalid line type status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	if _, parseErr = parseStore.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseCustomer.ID,
		InvoiceID:       parseInvoice.ID,
		LineType:        "usage",
		Description:     "bad amount",
		Quantity:        2,
		UnitAmountCents: 100,
		AmountCents:     150,
		Currency:        "usd",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("line amount mismatch status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	if _, parseErr = parseStore.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseCustomer.ID,
		InvoiceID:       parseInvoice.ID,
		LineType:        "subscription",
		Description:     "platform fee",
		Quantity:        1,
		UnitAmountCents: 3900,
		AmountCents:     3900,
		Currency:        "usd",
	}); parseErr != nil {
		parseT.Fatalf("platform fee line expected allow: %v", parseErr)
	}
	parseLineItems, parseErr := parseStore.parseListBillingInvoiceLineItems(parseInvoice.ID, parseCustomer.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingInvoiceLineItems: %v", parseErr)
	}
	if len(parseLineItems) != 1 || parseLineItems[0].LineType != parseBillingLineTypePlatformFee {
		parseT.Fatalf("expected one normalized platform-fee line item, got %+v", parseLineItems)
	}
}

// parseSeedUsageBasedInvoiceLineFixture seeds one customer/subscription/invoice tuple for usage-based invoice-line tests.
func parseSeedUsageBasedInvoiceLineFixture(parseT *testing.T, parseStore *Store, parseUserEmail string) (parseBillingCustomerRow, parseBillingInvoiceRow) {
	parseT.Helper()
	parseUser := parseMustCreateUser(parseT, parseStore, parseUserEmail)
	parseNow := time.Now().UTC()
	parseCustomer, parseErr := parseStore.parseUpsertBillingCustomer(parseBillingCustomerWrite{
		UserID:             parseUser.ID,
		ProviderID:         "stripe",
		ProviderCustomerID: "cus-usage-based-lines-" + parseNormalizeSUKey(parseUserEmail),
		DefaultCurrency:    "usd",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingCustomer: %v", parseErr)
	}
	parseSubscription, parseErr := parseStore.parseUpsertBillingSubscription(parseBillingSubscriptionWrite{
		CustomerID:             parseCustomer.ID,
		ProviderID:             "stripe",
		ProviderSubscriptionID: "sub-usage-based-lines-" + parseNormalizeSUKey(parseUserEmail),
		PlanCode:               "team",
		Status:                 "active",
		BillingInterval:        "month",
		CurrentPeriodStart:     parseNow.Format(time.RFC3339),
		CurrentPeriodEnd:       parseNow.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingSubscription: %v", parseErr)
	}
	parseInvoice, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomer.ID,
		SubscriptionID:    parseSubscription.ID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-usage-based-lines-" + parseNormalizeSUKey(parseUserEmail),
		Status:            "open",
		Currency:          "usd",
		TotalCents:        0,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice: %v", parseErr)
	}
	return parseCustomer, parseInvoice
}

// TestCreateUsageBasedBillingInvoiceLineItemsPersistsClassification verifies generated invoice writes persist explicit platform/usage/premium rows.
func TestCreateUsageBasedBillingInvoiceLineItemsPersistsClassification(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseCustomer, parseInvoice := parseSeedUsageBasedInvoiceLineFixture(parseT, parseStore, "billing-generated-lines@example.com")
	parseLineIDs, parseErr := parseStore.parseCreateUsageBasedBillingInvoiceLineItems(parseUsageBasedBillingInvoiceLineItemsWrite{
		CustomerID:          parseCustomer.ID,
		InvoiceID:           parseInvoice.ID,
		Currency:            "usd",
		PlatformFeeCents:    2900,
		UsageCostCents:      1200,
		ServicePremiumCents: 60,
	})
	if parseErr != nil {
		parseT.Fatalf("parseCreateUsageBasedBillingInvoiceLineItems: %v", parseErr)
	}
	if len(parseLineIDs) != 3 {
		parseT.Fatalf("expected three inserted line ids, got %+v", parseLineIDs)
	}
	parseLineItems, parseErr := parseStore.parseListBillingInvoiceLineItems(parseInvoice.ID, parseCustomer.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingInvoiceLineItems: %v", parseErr)
	}
	if len(parseLineItems) != 3 {
		parseT.Fatalf("expected three persisted line items, got %+v", parseLineItems)
	}
	parseFoundTypes := map[string]bool{}
	for _, parseLineItem := range parseLineItems {
		parseFoundTypes[parseLineItem.LineType] = true
	}
	if !parseFoundTypes[parseBillingLineTypePlatformFee] || !parseFoundTypes[parseBillingLineTypeUsageCost] || !parseFoundTypes[parseBillingLineTypeServicePremium] {
		parseT.Fatalf("expected classified platform/usage/premium lines, got %+v", parseLineItems)
	}
}

// TestCreateUsageBasedBillingInvoiceLineItemsRejectsEmptyBreakdown verifies generated invoice writes fail closed when all billing components are zero.
func TestCreateUsageBasedBillingInvoiceLineItemsRejectsEmptyBreakdown(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseCustomer, parseInvoice := parseSeedUsageBasedInvoiceLineFixture(parseT, parseStore, "billing-generated-lines-empty@example.com")
	if _, parseErr := parseStore.parseCreateUsageBasedBillingInvoiceLineItems(parseUsageBasedBillingInvoiceLineItemsWrite{
		CustomerID:          parseCustomer.ID,
		InvoiceID:           parseInvoice.ID,
		Currency:            "usd",
		PlatformFeeCents:    0,
		UsageCostCents:      0,
		ServicePremiumCents: 0,
	}); parseErr == nil {
		parseT.Fatal("expected empty usage-based breakdown to fail")
	}
}

// TestAdminBillingOverrideRejectsUnlimitedMonthlyTokenLimit verifies admin override RPC blocks unlimited monthly token assumptions.
func TestAdminBillingOverrideRejectsUnlimitedMonthlyTokenLimit(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-billing-formula", parseAliceAuth.ID, parseAliceAuth.Email)

	if _, parseErr = parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   billingEntitlementUsageMonthlyTokenLimit,
		OverrideValue: "unlimited",
		Reason:        "attempt stale unlimited override",
		IsEnabled:     true,
		Confirm:       true,
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetAdminBillingAccessOverride unlimited monthly token status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestSuperuserBillingPlanRejectsZeroPlatformFee validates SetSuperuserBillingPlan fail-closes active non-enterprise plans without platform fees.
func TestSuperuserBillingPlanRejectsZeroPlatformFee(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-superuser-plan-formula", parseOwner.ID, parseOwner.Email)

	if _, parseErr := parseServer.SetSuperuserBillingPlan(parseSuperuserCtx, &chatpb.SetSuperuserBillingPlanRequest{
		PlanCode:              "pro",
		PlanName:              "Pro",
		PlanRank:              10,
		IsActive:              true,
		MonthlyBaseCents:      0,
		YearlyBaseCents:       0,
		IncludedTokensMonthly: 5000000,
		IncludedSeats:         1,
		MaxSeats:              1,
		Confirm:               true,
		Reason:                "invalid zero platform fee plan",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetSuperuserBillingPlan zero platform fee status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestSuperuserBillingPlanBoundaryMutations verifies boundary enforcement for canonical plan/entitlement mutations.
func TestSuperuserBillingPlanBoundaryMutations(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-superuser-plan-boundary", parseOwner.ID, parseOwner.Email)

	if _, parseErr := parseServer.SetSuperuserBillingPlan(parseSuperuserCtx, &chatpb.SetSuperuserBillingPlanRequest{
		PlanCode:              "pro",
		PlanName:              "Pro",
		PlanRank:              10,
		IsActive:              true,
		MonthlyBaseCents:      2900,
		YearlyBaseCents:       29000,
		IncludedTokensMonthly: 5000000,
		IncludedSeats:         1,
		MaxSeats:              3,
		SupportsTeamWorkspace: true,
		SupportsSso:           false,
		Confirm:               true,
		Reason:                "invalid pro boundary",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetSuperuserBillingPlan pro boundary status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}

	if _, parseErr := parseServer.SetSuperuserBillingPlanEntitlement(parseSuperuserCtx, &chatpb.SetSuperuserBillingPlanEntitlementRequest{
		PlanCode:         "pro",
		EntitlementKey:   "workspace.multi_user.enabled",
		EntitlementValue: "true",
		Confirm:          true,
		Reason:           "invalid pro entitlement boundary",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetSuperuserBillingPlanEntitlement pro multi-user status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// BenchmarkBuildUsageBasedBillingPreview measures preview formula computation overhead.
func BenchmarkBuildUsageBasedBillingPreview(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parsePreview, parseErr := parseBuildUsageBasedBillingPreview(2900, 12345, 500)
		if parseErr != nil {
			parseB.Fatalf("parseBuildUsageBasedBillingPreview: %v", parseErr)
		}
		if parsePreview.totalCents == 0 {
			parseB.Fatal("expected non-zero total preview cents")
		}
	}
}
