package app

import (
	"context"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// parseSeedSuperuserPricingInvoiceIDs seeds one customer/subscription/invoice tuple for superuser pricing tests.
func parseSeedSuperuserPricingInvoiceIDs(parseT *testing.T, parseStore *Store, parseUserID int64) (int64, int64, int64) {
	parseT.Helper()

	parseMustAssignBillingPlan(parseT, parseStore, parseUserID, "team")
	parseCustomer, hasParseCustomer, parseErr := parseStore.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !hasParseCustomer {
		parseT.Fatalf("expected billing customer for user %d", parseUserID)
	}
	parseSubscriptions, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 1)
	if parseErr != nil || len(parseSubscriptions) == 0 {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: rows=%+v err=%v", parseSubscriptions, parseErr)
	}
	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseInvoice, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomer.ID,
		SubscriptionID:    parseSubscriptions[0].ID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv-superuser-pricing",
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
	return parseCustomer.ID, parseSubscriptions[0].ID, parseInvoice.ID
}

// TestStoreSuperuserPricingControlFuncs verifies typed store pricing-control upsert/delete helpers.
func TestStoreSuperuserPricingControlFuncs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseCustomerID, parseSubscriptionID, parseInvoiceID := parseSeedSuperuserPricingInvoiceIDs(parseT, parseStore, parseOwner.ID)

	parseOverageRow, parseErr := parseStore.parseUpsertSuperuserBillingPlanOverage(parseSuperuserBillingPlanOverageWrite{
		PlanCode:          "team",
		MeterKey:          "usage.tokens.monthly",
		IncludedUnits:     1000,
		SoftLimitUnits:    2000,
		HardLimitUnits:    3000,
		OverageUnitSize:   100,
		OveragePriceCents: 5,
		BillingInterval:   "monthly",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserBillingPlanOverage: %v", parseErr)
	}
	if parseOverageRow.PlanCode != "team" || parseOverageRow.MeterKey != "usage.tokens.monthly" {
		parseT.Fatalf("unexpected overage row: %+v", parseOverageRow)
	}

	parsePolicyRow, parseErr := parseStore.parseUpsertSuperuserBillingQuotaPolicy(parseSuperuserBillingQuotaPolicyWrite{
		PlanCode:        "team",
		QuotaKey:        "requests.daily",
		SoftLimitValue:  1000,
		HardLimitValue:  1200,
		ResetInterval:   "daily",
		EnforcementMode: "warn",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserBillingQuotaPolicy: %v", parseErr)
	}
	if parsePolicyRow.PlanCode != "team" || parsePolicyRow.QuotaKey != "requests.daily" {
		parseT.Fatalf("unexpected quota policy row: %+v", parsePolicyRow)
	}

	parseTriggerRow, parseErr := parseStore.parseUpsertSuperuserBillingUpgradeTrigger(parseSuperuserBillingUpgradeTriggerWrite{
		PlanCode:         "team",
		TriggerKey:       "usage.tokens.threshold",
		ThresholdPercent: 85,
		UpgradePlanCode:  "pro",
		Message:          "Upgrade suggested",
		CTALabel:         "Upgrade",
		CTAURL:           "/app/settings?panel=settings-billing",
		IsEnabled:        true,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserBillingUpgradeTrigger: %v", parseErr)
	}
	if parseTriggerRow.PlanCode != "team" || parseTriggerRow.TriggerKey != "usage.tokens.threshold" {
		parseT.Fatalf("unexpected upgrade trigger row: %+v", parseTriggerRow)
	}

	parseDunningRow, parseErr := parseStore.parseUpsertSuperuserBillingDunningEvent(parseSuperuserBillingDunningEventWrite{
		CustomerID:     parseCustomerID,
		SubscriptionID: parseSubscriptionID,
		InvoiceID:      parseInvoiceID,
		Status:         "pending",
		AttemptCount:   1,
		FailureReason:  "card_declined",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserBillingDunningEvent create: %v", parseErr)
	}
	if parseDunningRow.ID <= 0 || parseDunningRow.CustomerID != parseCustomerID {
		parseT.Fatalf("unexpected dunning row: %+v", parseDunningRow)
	}
	parseDunningRow, parseErr = parseStore.parseUpsertSuperuserBillingDunningEvent(parseSuperuserBillingDunningEventWrite{
		ID:             parseDunningRow.ID,
		CustomerID:     parseCustomerID,
		SubscriptionID: parseSubscriptionID,
		InvoiceID:      parseInvoiceID,
		Status:         "resolved",
		AttemptCount:   2,
		FailureReason:  "manual_resolution",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserBillingDunningEvent update: %v", parseErr)
	}
	if parseDunningRow.Status != "resolved" || parseDunningRow.AttemptCount != 2 {
		parseT.Fatalf("expected resolved dunning row after update, got %+v", parseDunningRow)
	}

	if parseErr = parseStore.parseDeleteSuperuserBillingPlanOverage("team", "usage.tokens.monthly"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserBillingPlanOverage: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserBillingQuotaPolicy("team", "requests.daily"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserBillingQuotaPolicy: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserBillingUpgradeTrigger("team", "usage.tokens.threshold"); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserBillingUpgradeTrigger: %v", parseErr)
	}
	if parseErr = parseStore.parseDeleteSuperuserBillingDunningEvent(parseDunningRow.ID); parseErr != nil {
		parseT.Fatalf("parseDeleteSuperuserBillingDunningEvent: %v", parseErr)
	}

	parseOverageRows, parseErr := parseStore.parseListBillingPlanOverages(50)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingPlanOverages: %v", parseErr)
	}
	if parseHasSuperuserBillingPlanOverage(parseOverageRows, "team", "usage.tokens.monthly") {
		parseT.Fatalf("expected deleted overage row to be absent, rows=%+v", parseOverageRows)
	}
	parsePolicyRows, parseErr := parseStore.parseListBillingQuotaPolicies(50)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingQuotaPolicies: %v", parseErr)
	}
	if parseHasSuperuserBillingQuotaPolicy(parsePolicyRows, "team", "requests.daily") {
		parseT.Fatalf("expected deleted quota policy row to be absent, rows=%+v", parsePolicyRows)
	}
	parseTriggerRows, parseErr := parseStore.parseListBillingUpgradeTriggers(50)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingUpgradeTriggers: %v", parseErr)
	}
	if parseHasSuperuserBillingUpgradeTrigger(parseTriggerRows, "team", "usage.tokens.threshold") {
		parseT.Fatalf("expected deleted trigger row to be absent, rows=%+v", parseTriggerRows)
	}
	if _, hasParseDunningRow, parseErr := parseStore.parseGetBillingDunningEventByID(parseDunningRow.ID); parseErr != nil || hasParseDunningRow {
		parseT.Fatalf("expected deleted dunning row to be absent, found=%v err=%v", hasParseDunningRow, parseErr)
	}
}

// TestSuperuserPricingControlRPCs verifies typed superuser pricing CRUD RPC behavior and role gating.
func TestSuperuserPricingControlRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseSeedSuperuserControlPlaneData(parseT, parseStore)
	parseCustomerID, parseSubscriptionID, parseInvoiceID := parseSeedSuperuserPricingInvoiceIDs(parseT, parseStore, parseOwner.ID)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseSuperuserCtx := parseBindAuthUser(parseServer, "peer-superuser-pricing-rpc", parseOwner.ID, parseOwner.Email)

	parseOverageResp, parseErr := parseServer.SetSuperuserBillingPlanOverage(parseSuperuserCtx, &chatpb.SetSuperuserBillingPlanOverageRequest{
		PlanCode:          "team",
		MeterKey:          "usage.tokens.monthly",
		IncludedUnits:     2000,
		SoftLimitUnits:    2500,
		HardLimitUnits:    3000,
		OverageUnitSize:   100,
		OveragePriceCents: 10,
		BillingInterval:   "monthly",
		Confirm:           true,
		Reason:            "pricing calibration",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserBillingPlanOverage: %v", parseErr)
	}
	if parseOverageResp.GetOverage().GetPlanCode() != "team" || parseOverageResp.GetStatus() == "" {
		parseT.Fatalf("unexpected SetSuperuserBillingPlanOverage response: %+v", parseOverageResp)
	}

	parseQuotaResp, parseErr := parseServer.SetSuperuserBillingQuotaPolicy(parseSuperuserCtx, &chatpb.SetSuperuserBillingQuotaPolicyRequest{
		PlanCode:        "team",
		QuotaKey:        "requests.daily",
		SoftLimitValue:  1200,
		HardLimitValue:  1500,
		ResetInterval:   "daily",
		EnforcementMode: "warn",
		Confirm:         true,
		Reason:          "quota policy update",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserBillingQuotaPolicy: %v", parseErr)
	}
	if parseQuotaResp.GetPolicy().GetQuotaKey() != "requests.daily" || parseQuotaResp.GetStatus() == "" {
		parseT.Fatalf("unexpected SetSuperuserBillingQuotaPolicy response: %+v", parseQuotaResp)
	}

	parseTriggerResp, parseErr := parseServer.SetSuperuserBillingUpgradeTrigger(parseSuperuserCtx, &chatpb.SetSuperuserBillingUpgradeTriggerRequest{
		PlanCode:         "team",
		TriggerKey:       "usage.tokens.threshold",
		ThresholdPercent: 90,
		UpgradePlanCode:  "pro",
		Message:          "Upgrade now",
		CtaLabel:         "Upgrade",
		CtaUrl:           "/app/settings?panel=settings-billing",
		IsEnabled:        true,
		Confirm:          true,
		Reason:           "upgrade trigger update",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserBillingUpgradeTrigger: %v", parseErr)
	}
	if parseTriggerResp.GetTrigger().GetTriggerKey() != "usage.tokens.threshold" || parseTriggerResp.GetStatus() == "" {
		parseT.Fatalf("unexpected SetSuperuserBillingUpgradeTrigger response: %+v", parseTriggerResp)
	}

	parseDunningResp, parseErr := parseServer.SetSuperuserBillingDunningEvent(parseSuperuserCtx, &chatpb.SetSuperuserBillingDunningEventRequest{
		CustomerId:     parseCustomerID,
		SubscriptionId: parseSubscriptionID,
		InvoiceId:      parseInvoiceID,
		Status:         "pending",
		AttemptCount:   1,
		FailureReason:  "card_declined",
		Confirm:        true,
		Reason:         "manual dunning lifecycle test",
	})
	if parseErr != nil {
		parseT.Fatalf("SetSuperuserBillingDunningEvent: %v", parseErr)
	}
	if parseDunningResp.GetEvent().GetId() <= 0 || parseDunningResp.GetStatus() == "" {
		parseT.Fatalf("unexpected SetSuperuserBillingDunningEvent response: %+v", parseDunningResp)
	}

	if _, parseErr = parseServer.DeleteSuperuserBillingPlanOverage(parseSuperuserCtx, &chatpb.DeleteSuperuserBillingPlanOverageRequest{
		PlanCode: "team",
		MeterKey: "usage.tokens.monthly",
		Confirm:  true,
		Reason:   "cleanup overage",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserBillingPlanOverage: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserBillingQuotaPolicy(parseSuperuserCtx, &chatpb.DeleteSuperuserBillingQuotaPolicyRequest{
		PlanCode: "team",
		QuotaKey: "requests.daily",
		Confirm:  true,
		Reason:   "cleanup quota",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserBillingQuotaPolicy: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserBillingUpgradeTrigger(parseSuperuserCtx, &chatpb.DeleteSuperuserBillingUpgradeTriggerRequest{
		PlanCode:   "team",
		TriggerKey: "usage.tokens.threshold",
		Confirm:    true,
		Reason:     "cleanup trigger",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserBillingUpgradeTrigger: %v", parseErr)
	}
	if _, parseErr = parseServer.DeleteSuperuserBillingDunningEvent(parseSuperuserCtx, &chatpb.DeleteSuperuserBillingDunningEventRequest{
		Id:      parseDunningResp.GetEvent().GetId(),
		Confirm: true,
		Reason:  "cleanup dunning event",
	}); parseErr != nil {
		parseT.Fatalf("DeleteSuperuserBillingDunningEvent: %v", parseErr)
	}

	parseNonSuperuser := parseMustCreateUser(parseT, parseStore, "non-su-pricing@example.com")
	parseNonSuperuserCtx := parseBindAuthUser(parseServer, "peer-non-superuser-pricing-rpc", parseNonSuperuser.ID, parseNonSuperuser.Email)
	if _, parseErr = parseServer.SetSuperuserBillingPlanOverage(parseNonSuperuserCtx, &chatpb.SetSuperuserBillingPlanOverageRequest{
		PlanCode: "team",
		MeterKey: "usage.tokens.monthly",
		Confirm:  true,
		Reason:   "should fail",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("SetSuperuserBillingPlanOverage non-superuser status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.SetSuperuserBillingPlanOverage(parseSuperuserCtx, &chatpb.SetSuperuserBillingPlanOverageRequest{
		PlanCode: "team",
		MeterKey: "usage.tokens.monthly",
		Confirm:  false,
		Reason:   "missing confirm",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("SetSuperuserBillingPlanOverage missing confirm status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}

// TestSuperuserBusinessMutationsRequireFreshSession verifies stale superuser sessions cannot execute business pricing mutations.
func TestSuperuserBusinessMutationsRequireFreshSession(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger())
	parseServer.authManager = parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseSignupResp, parseErr := parseServer.Signup(context.Background(), &chatpb.SignupRequest{
		Email:       "superuser-business-fresh-session@example.com",
		Password:    "password123",
		DisplayName: "Business Fresh Session",
	})
	if parseErr != nil {
		parseT.Fatalf("Signup: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSignupResp.GetUserId())

	parseLoginResp, parseErr := parseServer.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "superuser-business-fresh-session@example.com",
		Password: "password123",
	})
	if parseErr != nil {
		parseT.Fatalf("Login: %v", parseErr)
	}

	parseFreshCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseLoginResp.GetAuthToken()))
	if _, parseErr = parseServer.SetSuperuserBillingPlanOverage(parseFreshCtx, &chatpb.SetSuperuserBillingPlanOverageRequest{
		PlanCode:          "team",
		MeterKey:          "usage.tokens.monthly",
		IncludedUnits:     1000,
		SoftLimitUnits:    1250,
		HardLimitUnits:    1500,
		OverageUnitSize:   100,
		OveragePriceCents: 10,
		BillingInterval:   "monthly",
		Confirm:           true,
		Reason:            "fresh-session control validation",
	}); parseErr != nil {
		parseT.Fatalf("SetSuperuserBillingPlanOverage fresh session: %v", parseErr)
	}

	parseStaleToken := parseBuildStaleSessionToken(parseT, parseServer.authManager, parseLoginResp.GetAuthToken(), time.Now().UTC().Add(-superuserMutationSessionMaxAge-time.Minute))
	parseStaleCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseStaleToken))
	if _, parseErr = parseServer.SetSuperuserBillingPlanOverage(parseStaleCtx, &chatpb.SetSuperuserBillingPlanOverageRequest{
		PlanCode:          "team",
		MeterKey:          "usage.tokens.monthly",
		IncludedUnits:     1000,
		SoftLimitUnits:    1250,
		HardLimitUnits:    1500,
		OverageUnitSize:   100,
		OveragePriceCents: 10,
		BillingInterval:   "monthly",
		Confirm:           true,
		Reason:            "stale-session should fail",
	}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("SetSuperuserBillingPlanOverage stale session status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
}

// parseHasSuperuserBillingPlanOverage reports whether one overage row slice contains one plan+meter pair.
func parseHasSuperuserBillingPlanOverage(parseRows []parseBillingPlanOverageRow, parsePlanCode string, parseMeterKey string) bool {
	for _, parseRow := range parseRows {
		if parseRow.PlanCode == parsePlanCode && parseRow.MeterKey == parseMeterKey {
			return true
		}
	}
	return false
}

// parseHasSuperuserBillingQuotaPolicy reports whether one quota-policy row slice contains one plan+quota pair.
func parseHasSuperuserBillingQuotaPolicy(parseRows []parseBillingQuotaPolicyRow, parsePlanCode string, parseQuotaKey string) bool {
	for _, parseRow := range parseRows {
		if parseRow.PlanCode == parsePlanCode && parseRow.QuotaKey == parseQuotaKey {
			return true
		}
	}
	return false
}

// parseHasSuperuserBillingUpgradeTrigger reports whether one trigger row slice contains one plan+trigger pair.
func parseHasSuperuserBillingUpgradeTrigger(parseRows []parseBillingUpgradeTriggerRow, parsePlanCode string, parseTriggerKey string) bool {
	for _, parseRow := range parseRows {
		if parseRow.PlanCode == parsePlanCode && parseRow.TriggerKey == parseTriggerKey {
			return true
		}
	}
	return false
}

// BenchmarkParseNormalizeSuperuserBillingEnforcementMode reports micro-benchmark throughput for superuser enforcement-mode normalization.
func BenchmarkParseNormalizeSuperuserBillingEnforcementMode(parseB *testing.B) {
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseNormalizeSuperuserBillingEnforcementMode("warn")
	}
}
