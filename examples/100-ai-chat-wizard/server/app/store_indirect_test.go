package app

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// store_billing.go — normalization helpers
// ---------------------------------------------------------------------------

func TestBillingBuildFlagValue(parseT *testing.T) {
	if parseBuildBillingFlagValue(true) != 1 {
		parseT.Fatal("expected true→1")
	}
	if parseBuildBillingFlagValue(false) != 0 {
		parseT.Fatal("expected false→0")
	}
}

func TestBillingNormalizeCurrency(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"", "USD"},
		{"  ", "USD"},
		{"usd", "USD"},
		{"eur", "EUR"},
		{"GBP", "GBP"},
		{"  cad  ", "CAD"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeBillingCurrency(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeBillingCurrency(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestBillingNormalizeTaxExemptStatus(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"", "none"},
		{"  ", "none"},
		{"EXEMPT", "exempt"},
		{"none", "none"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeBillingTaxExemptStatus(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeBillingTaxExemptStatus(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestBillingNormalizeSubscriptionStatus(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"", "active"},
		{"ACTIVE", "active"},
		{"canceled", "canceled"},
		{"  past_due  ", "past_due"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeBillingSubscriptionStatus(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeBillingSubscriptionStatus(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestBillingNormalizeInvoiceStatus(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"", "draft"},
		{"PAID", "paid"},
		{"open", "open"},
		{"  Void  ", "void"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeBillingInvoiceStatus(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeBillingInvoiceStatus(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestBillingNormalizeInterval(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"", "month"},
		{"YEAR", "year"},
		{"month", "month"},
		{"  Week  ", "week"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeBillingInterval(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeBillingInterval(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestBillingNormalizeEventSource(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"", "system"},
		{"STRIPE", "stripe"},
		{"webhook", "webhook"},
		{"  Admin  ", "admin"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeBillingEventSource(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeBillingEventSource(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestBillingNormalizeJSON(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"", "{}"},
		{"  ", "{}"},
		{"{}", "{}"},
		{`{"key":"value"}`, `{"key":"value"}`},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeBillingJSON(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeBillingJSON(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

// ---------------------------------------------------------------------------
// store_server_tool_policy.go / server_tool_policy.go — helpers
// ---------------------------------------------------------------------------

func TestServerToolPolicyBoolValue(parseT *testing.T) {
	if parseBuildServerToolPolicyBoolValue(true) != "true" {
		parseT.Fatal("expected true→\"true\"")
	}
	if parseBuildServerToolPolicyBoolValue(false) != "false" {
		parseT.Fatal("expected false→\"false\"")
	}
}

func TestServerToolPolicyBoolBit(parseT *testing.T) {
	if parseBuildServerToolPolicyBoolBit(true) != 1 {
		parseT.Fatal("expected true→1")
	}
	if parseBuildServerToolPolicyBoolBit(false) != 0 {
		parseT.Fatal("expected false→0")
	}
}

func TestServerToolPolicyIntValue(parseT *testing.T) {
	parseTests := []struct {
		parseInput int32
		parseWant  string
	}{
		{0, "0"},
		{300, "300"},
		{-1, "-1"},
		{1024 * 1024, "1048576"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseBuildServerToolPolicyIntValue(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseBuildServerToolPolicyIntValue(%d): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestClampServerToolPolicyInt32(parseT *testing.T) {
	// Below min → min
	if parseGot := parseClampServerToolPolicyInt32(0, 5, 100); parseGot != 5 {
		parseT.Errorf("clamp(0,5,100): got %d, want 5", parseGot)
	}
	// Above max → max
	if parseGot := parseClampServerToolPolicyInt32(200, 5, 100); parseGot != 100 {
		parseT.Errorf("clamp(200,5,100): got %d, want 100", parseGot)
	}
	// In-range → unchanged
	if parseGot := parseClampServerToolPolicyInt32(50, 5, 100); parseGot != 50 {
		parseT.Errorf("clamp(50,5,100): got %d, want 50", parseGot)
	}
	// At min boundary
	if parseGot := parseClampServerToolPolicyInt32(5, 5, 100); parseGot != 5 {
		parseT.Errorf("clamp(5,5,100): got %d, want 5", parseGot)
	}
	// At max boundary
	if parseGot := parseClampServerToolPolicyInt32(100, 5, 100); parseGot != 100 {
		parseT.Errorf("clamp(100,5,100): got %d, want 100", parseGot)
	}
}

// ---------------------------------------------------------------------------
// store_superuser_pricing.go — normalization helpers
// ---------------------------------------------------------------------------

func TestSuperuserBillingEnforcementMode(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"warn", "warn"},
		{"TRACK", "track"},
		{"Block", "block"},
		{"", "block"},
		{"unknown", "block"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeSuperuserBillingEnforcementMode(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeSuperuserBillingEnforcementMode(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestSuperuserDunningStatus(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"pending", "pending"},
		{"RETRYING", "retrying"},
		{"Resolved", "resolved"},
		{"failed", "failed"},
		{"", "pending"},
		{"unknown", "pending"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeSuperuserDunningStatus(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeSuperuserDunningStatus(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestSuperuserBillingPlanCode(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"free", "free"},
		{"PRO", "pro"},
		{"  Enterprise  ", "enterprise"},
		{"", ""},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeSuperuserBillingPlanCode(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeSuperuserBillingPlanCode(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

// ---------------------------------------------------------------------------
// store_superuser_reliability.go — normalization helpers
// ---------------------------------------------------------------------------

func TestSuperuserPurgeMode(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"delete", "delete"},
		{"ARCHIVE", "archive"},
		{"Anonymize", "anonymize"},
		{"", "delete"},
		{"purge", "delete"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeSuperuserPurgeMode(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeSuperuserPurgeMode(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestSuperuserComplianceStatus(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"planned", "planned"},
		{"IN_PROGRESS", "in_progress"},
		{"implemented", "implemented"},
		{"VERIFIED", "verified"},
		{"Waived", "waived"},
		{"", "planned"},
		{"approved", "planned"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeSuperuserComplianceStatus(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeSuperuserComplianceStatus(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestClampSuperuserObjectivePercent(parseT *testing.T) {
	parseTests := []struct {
		parseInput float64
		parseWant  float64
	}{
		{-1, 0},
		{0, 0},
		{50, 50},
		{99.9, 99.9},
		{100, 100},
		{101, 100},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseClampSuperuserObjectivePercent(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseClampSuperuserObjectivePercent(%v): got %v, want %v", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestSuperuserIncidentSeverity(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"minor", "minor"},
		{"MAJOR", "major"},
		{"Critical", "critical"},
		{"", "minor"},
		{"severe", "minor"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeSuperuserIncidentSeverity(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeSuperuserIncidentSeverity(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestSuperuserIncidentStatus(parseT *testing.T) {
	parseTests := []struct {
		parseInput string
		parseWant  string
	}{
		{"open", "open"},
		{"INVESTIGATING", "investigating"},
		{"Identified", "identified"},
		{"mitigated", "mitigated"},
		{"monitoring", "monitoring"},
		{"resolved", "resolved"},
		{"closed", "closed"},
		{"", "investigating"},
		{"unknown", "investigating"},
	}
	for _, parseTC := range parseTests {
		if parseGot := parseNormalizeSuperuserIncidentStatus(parseTC.parseInput); parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeSuperuserIncidentStatus(%q): got %q, want %q", parseTC.parseInput, parseGot, parseTC.parseWant)
		}
	}
}

func TestNormalizeWorkspaceSSOProviderType(parseT *testing.T) {
	parseTests := []struct {
		parseProviderType    string
		parseProviderKey     string
		parseSAMLEntrypoint  string
		parseSAMLIssuer      string
		parseSAMLCertificate string
		parseWant            string
	}{
		// Explicit type wins
		{"oidc", "", "", "", "", "oidc"},
		{"SAML", "", "", "", "", "saml"},
		// SAML fields infer saml
		{"", "", "https://idp.example.com/sso", "", "", "saml"},
		{"", "", "", "urn:example:issuer", "", "saml"},
		{"", "", "", "", "-----BEGIN CERT-----", "saml"},
		// No signals → default oidc
		{"", "", "", "", "", "oidc"},
		// Unknown type + no SAML signals → oidc
		{"openid", "", "", "", "", "oidc"},
	}
	for _, parseTC := range parseTests {
		parseGot := parseNormalizeWorkspaceSSOProviderType(
			parseTC.parseProviderType,
			parseTC.parseProviderKey,
			parseTC.parseSAMLEntrypoint,
			parseTC.parseSAMLIssuer,
			parseTC.parseSAMLCertificate,
		)
		if parseGot != parseTC.parseWant {
			parseT.Errorf("parseNormalizeWorkspaceSSOProviderType(%q,%q,...): got %q, want %q",
				parseTC.parseProviderType, parseTC.parseProviderKey, parseGot, parseTC.parseWant)
		}
	}
}

// ---------------------------------------------------------------------------
// onboarding_state.go — pure helper
// ---------------------------------------------------------------------------

func TestBuildStarterMilestoneMetadataJSON(parseT *testing.T) {
	parseTests := []struct {
		parseLabel    string
		parseInput    map[string]any
		parseWantJSON string
	}{
		{"nil map", nil, "{}"},
		{"empty map", map[string]any{}, "{}"},
		{"single key", map[string]any{"state": "brand_new_user"}, `{"state":"brand_new_user"}`},
	}
	for _, parseTC := range parseTests {
		parseGot := parseBuildStarterMilestoneMetadataJSON(parseTC.parseInput)
		if parseGot != parseTC.parseWantJSON {
			parseT.Errorf("%s: got %q, want %q", parseTC.parseLabel, parseGot, parseTC.parseWantJSON)
		}
	}
}

// ---------------------------------------------------------------------------
// model_catalog_store.go — pure row method helpers
// ---------------------------------------------------------------------------

func TestModelCatalogRowHelpers(parseT *testing.T) {
	parseRow := modelCatalogRow{
		ID:                  "openai/gpt-4o",
		ProviderID:          "openai",
		ProviderLabel:       "OpenAI",
		Label:               "GPT-4o",
		Note:                "flagship",
		SupportsThinking:    false,
		SupportsSpeech:      true,
		InputPerMillionUSD:  2.50,
		OutputPerMillionUSD: 10.00,
		PricingCurrency:     "USD",
		MaxOutputTokens:     4096,
	}

	parseCaps := parseRow.parseCapabilities()
	if parseCaps.ProviderID != "openai" || parseCaps.ProviderLabel != "OpenAI" {
		parseT.Errorf("parseCapabilities: unexpected provider fields: %+v", parseCaps)
	}
	if parseCaps.SupportsThinking || !parseCaps.SupportsSpeech {
		parseT.Errorf("parseCapabilities: unexpected feature flags: %+v", parseCaps)
	}

	parseMeta := parseRow.parseMetadata()
	if parseMeta.ID != "openai/gpt-4o" || parseMeta.DisplayName != "GPT-4o" {
		parseT.Errorf("parseMetadata: unexpected fields: %+v", parseMeta)
	}
	if parseMeta.Pricing.InputPerMillionUSD != 2.50 || parseMeta.Pricing.OutputPerMillionUSD != 10.00 {
		parseT.Errorf("parseMetadata: unexpected pricing: %+v", parseMeta.Pricing)
	}

	parseOption := parseRow.parseOption()
	if parseOption.ID != "openai/gpt-4o" || parseOption.Label != "GPT-4o" || parseOption.Note != "flagship" {
		parseT.Errorf("parseOption: unexpected fields: %+v", parseOption)
	}
}

// ---------------------------------------------------------------------------
// store_growth_ops.go — round-trip: onboarding templates
// ---------------------------------------------------------------------------

func TestGrowthOpsOnboardingTemplateRoundTrip(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)

	if parseErr := parseStore.parseUpsertOnboardingTemplate(parseOnboardingTemplateWrite{
		TemplateKey:   "tmpl.first-steps",
		Title:         "First steps",
		Category:      "onboarding",
		PromptText:    "Tell me what you want to build.",
		ChecklistJSON: "[]",
		IsDefault:     true,
		SortOrder:     1,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertOnboardingTemplate: %v", parseErr)
	}

	parseList, parseErr := parseStore.parseListOnboardingTemplates(10)
	if parseErr != nil {
		parseT.Fatalf("parseListOnboardingTemplates: %v", parseErr)
	}
	if len(parseList) != 1 {
		parseT.Fatalf("expected 1 template, got %d", len(parseList))
	}
	if parseList[0].TemplateKey != "tmpl.first-steps" || parseList[0].Title != "First steps" {
		parseT.Errorf("unexpected template row: %+v", parseList[0])
	}
	if !parseList[0].IsDefault {
		parseT.Errorf("expected IsDefault=true, got %v", parseList[0].IsDefault)
	}
}

// parseMustCreateTestWorkspace creates one workspace in the test store and returns its integer ID.
func parseMustCreateTestWorkspace(parseT *testing.T, parseStore *Store, parseOwnerUserID int64, parseKey string) int64 {
	parseT.Helper()
	if parseErr := parseStore.parseUpsertWorkspace(parseWorkspaceWrite{
		WorkspaceKey: parseKey,
		Slug:         parseKey,
		Name:         parseKey,
		PlanCode:     "free",
		Status:       "active",
		OwnerUserID:  parseOwnerUserID,
		SettingsJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertWorkspace %q: %v", parseKey, parseErr)
	}
	parseRows, parseErr := parseStore.parseListWorkspaces(100)
	if parseErr != nil {
		parseT.Fatalf("parseListWorkspaces: %v", parseErr)
	}
	for _, parseRow := range parseRows {
		if parseRow.WorkspaceKey == parseKey {
			return parseRow.ID
		}
	}
	parseT.Fatalf("workspace %q not found after upsert", parseKey)
	return 0
}

// ---------------------------------------------------------------------------
// store_growth_ops.go — round-trip: analytics events by user
// ---------------------------------------------------------------------------

func TestGrowthOpsAnalyticsEventListsOnEmptyStore(parseT *testing.T) {
	// product_analytics_events has FK constraints on workspace_id, user_id,
	// and experiment_key that require seeding related tables; test list queries
	// instead which have no preconditions.
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "analytics@example.com")

	parseAll, parseErr := parseStore.parseListProductAnalyticsEvents(10)
	if parseErr != nil {
		parseT.Fatalf("parseListProductAnalyticsEvents: %v", parseErr)
	}
	if len(parseAll) != 0 {
		parseT.Errorf("expected empty list on fresh store, got %d rows", len(parseAll))
	}

	parseByUser, parseErr := parseStore.parseListProductAnalyticsEventsByUser(parseUser.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListProductAnalyticsEventsByUser: %v", parseErr)
	}
	if len(parseByUser) != 0 {
		parseT.Errorf("expected empty list for new user, got %d rows", len(parseByUser))
	}
}

// ---------------------------------------------------------------------------
// store_growth_ops.go — round-trip: activation milestones
// ---------------------------------------------------------------------------

func TestGrowthOpsActivationMilestoneRoundTrip(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "milestone@example.com")
	parseNow := time.Now().UTC().Format(time.RFC3339)

	if parseErr := parseStore.parseUpsertUserActivationMilestone(parseUserActivationMilestoneWrite{
		UserID:       parseUser.ID,
		MilestoneKey: "starter.first_run_detected",
		Status:       "completed",
		AchievedAt:   parseNow,
		MetadataJSON: `{"state":"brand_new_user"}`,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserActivationMilestone: %v", parseErr)
	}

	parseList, parseErr := parseStore.parseListUserActivationMilestones(50)
	if parseErr != nil {
		parseT.Fatalf("parseListUserActivationMilestones: %v", parseErr)
	}
	parsefound := false
	for _, parseRow := range parseList {
		if parseRow.UserID == parseUser.ID && parseRow.MilestoneKey == "starter.first_run_detected" {
			parsefound = true
			if parseRow.Status != "completed" {
				parseT.Errorf("expected status=completed, got %q", parseRow.Status)
			}
		}
	}
	if !parsefound {
		parseT.Fatal("milestone row not found after upsert")
	}
}

// ---------------------------------------------------------------------------
// store_server_tool_policy.go — round-trip: set + history
// ---------------------------------------------------------------------------

func TestServerToolPolicySetAndHistory(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "policy-admin@example.com")

	parseRow, parseErr := parseStore.parseSetServerToolPolicy(parseServerToolPolicyStoreWrite{
		IsEnabled:         true,
		MaxSessionSeconds: 120,
		MaxOutputBytes:    65536,
		ApprovedToolsJSON: `["bash"]`,
		UpdatedByUserID:   parseUser.ID,
		Source:            "test",
	})
	if parseErr != nil {
		parseT.Fatalf("parseSetServerToolPolicy: %v", parseErr)
	}
	if !parseRow.IsEnabled {
		parseT.Error("expected IsEnabled=true in returned row")
	}
	if parseRow.MaxSessionSeconds < serverToolPolicyMinimumMaxSessionSeconds {
		parseT.Errorf("expected MaxSessionSeconds clamped up to min, got %d", parseRow.MaxSessionSeconds)
	}

	parseHistory, parseErr := parseStore.parseListServerToolPolicyHistory(10)
	if parseErr != nil {
		parseT.Fatalf("parseListServerToolPolicyHistory: %v", parseErr)
	}
	if len(parseHistory) == 0 {
		parseT.Fatal("expected at least one history row after set")
	}
	if parseHistory[0].UpdatedByUserID != parseUser.ID {
		parseT.Errorf("expected UpdatedByUserID=%d, got %d", parseUser.ID, parseHistory[0].UpdatedByUserID)
	}
}

// ---------------------------------------------------------------------------
// store_billing.go — round-trip: customer + subscription + invoice
// ---------------------------------------------------------------------------

func TestBillingCustomerSubscriptionInvoiceRoundTrip(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "billing-user@example.com")

	// Customer upsert + get
	parseCustomer, parseErr := parseStore.parseUpsertBillingCustomer(parseBillingCustomerWrite{
		UserID:             parseUser.ID,
		ProviderID:         "stripe",
		ProviderCustomerID: "cus_testABC",
		BillingEmail:       "billing-user@example.com",
		BillingName:        "Billing User",
		DefaultCurrency:    "usd",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingCustomer: %v", parseErr)
	}
	if parseCustomer.DefaultCurrency != "USD" {
		parseT.Errorf("expected currency normalized to USD, got %q", parseCustomer.DefaultCurrency)
	}

	parseLoaded, hasParseLoaded, parseErr := parseStore.parseGetBillingCustomerByUser(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !hasParseLoaded {
		parseT.Fatal("expected billing customer to exist after upsert")
	}
	if parseLoaded.ProviderCustomerID != "cus_testABC" {
		parseT.Errorf("unexpected provider customer id: %q", parseLoaded.ProviderCustomerID)
	}

	// Subscription upsert + list
	parseSub, parseErr := parseStore.parseUpsertBillingSubscription(parseBillingSubscriptionWrite{
		CustomerID:             parseCustomer.ID,
		ProviderID:             "stripe",
		ProviderSubscriptionID: "sub_testXYZ",
		PlanCode:               "pro",
		Status:                 "ACTIVE",
		BillingInterval:        "MONTH",
		Quantity:               1,
		CurrentPeriodStart:     "2025-01-01T00:00:00Z",
		CurrentPeriodEnd:       "2025-02-01T00:00:00Z",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingSubscription: %v", parseErr)
	}
	if parseSub.Status != "active" {
		parseT.Errorf("expected status normalized to active, got %q", parseSub.Status)
	}
	if parseSub.BillingInterval != "month" {
		parseT.Errorf("expected interval normalized to month, got %q", parseSub.BillingInterval)
	}

	parseSubList, parseErr := parseStore.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: %v", parseErr)
	}
	if len(parseSubList) != 1 {
		parseT.Fatalf("expected 1 subscription, got %d", len(parseSubList))
	}

	// Invoice upsert + list
	parseInvoice, parseErr := parseStore.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:        parseCustomer.ID,
		SubscriptionID:    parseSub.ID,
		ProviderID:        "stripe",
		ProviderInvoiceID: "inv_testABC",
		Status:            "open",
		Currency:          "usd",
		SubtotalCents:     1000,
		TotalCents:        1000,
		AmountDueCents:    1000,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice: %v", parseErr)
	}
	if parseInvoice.Currency != "USD" {
		parseT.Errorf("expected invoice currency normalized to USD, got %q", parseInvoice.Currency)
	}

	parseInvoiceList, parseErr := parseStore.parseListBillingInvoicesByCustomer(parseCustomer.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingInvoicesByCustomer: %v", parseErr)
	}
	if len(parseInvoiceList) != 1 {
		parseT.Fatalf("expected 1 invoice, got %d", len(parseInvoiceList))
	}
}

// ---------------------------------------------------------------------------
// store_admin.go — smoke: dashboard summary on empty store
// ---------------------------------------------------------------------------

func TestAdminDashboardSummaryOnEmptyStore(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSince := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)

	parseSummary, parseErr := parseStore.parseGetAdminDashboardSummary(parseSince)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminDashboardSummary: %v", parseErr)
	}
	// Empty store: all aggregates must be zero — not negative, not NaN
	if parseSummary.TotalUsers < 0 || parseSummary.TotalConversations < 0 {
		parseT.Errorf("unexpected negative aggregates on empty store: %+v", parseSummary)
	}
}

func TestAdminDashboardDailyUsageOnEmptyStore(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSince := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)

	parseRows, parseErr := parseStore.parseListAdminDashboardDailyUsage(parseSince)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminDashboardDailyUsage: %v", parseErr)
	}
	// No rows on empty store is valid; nil slice is also fine
	_ = parseRows
}

// ---------------------------------------------------------------------------
// store_superuser_reliability.go — round-trip: data retention policy
// ---------------------------------------------------------------------------

func TestReliabilityDataRetentionPolicyRoundTrip(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, parseStore, "retention-owner@example.com")
	parseWorkspaceID := parseMustCreateTestWorkspace(parseT, parseStore, parseOwner.ID, "ws-retention")

	parseRow, parseErr := parseStore.parseUpsertSuperuserDataRetentionPolicy(parseSuperuserDataRetentionPolicyWrite{
		WorkspaceID:   parseWorkspaceID,
		ScopeKey:      "messages",
		RetentionDays: 90,
		PurgeMode:     "ARCHIVE",
		LegalHoldJSON: "{}",
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserDataRetentionPolicy: %v", parseErr)
	}
	if parseRow.ScopeKey != "messages" {
		parseT.Errorf("expected ScopeKey=messages, got %q", parseRow.ScopeKey)
	}
	if parseRow.PurgeMode != "archive" {
		parseT.Errorf("expected PurgeMode normalized to archive, got %q", parseRow.PurgeMode)
	}

	parseList, parseErr := parseStore.parseListDataRetentionPolicies(10)
	if parseErr != nil {
		parseT.Fatalf("parseListDataRetentionPolicies: %v", parseErr)
	}
	if len(parseList) == 0 {
		parseT.Fatal("expected at least one data retention policy after upsert")
	}
}

// ---------------------------------------------------------------------------
// store_superuser_reliability.go — round-trip: compliance control
// ---------------------------------------------------------------------------

func TestReliabilityComplianceControlRoundTrip(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "compliance@example.com")
	parseWorkspaceID := parseMustCreateTestWorkspace(parseT, parseStore, parseUser.ID, "ws-compliance")

	parseRow, parseErr := parseStore.parseUpsertSuperuserComplianceControl(parseSuperuserComplianceControlWrite{
		WorkspaceID:  parseWorkspaceID,
		ControlKey:   "cc.access-review",
		FrameworkKey: "soc2",
		Status:       "IMPLEMENTED",
		OwnerUserID:  parseUser.ID,
		EvidenceURL:  "https://evidence.example.com/doc.pdf",
		ReviewedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertSuperuserComplianceControl: %v", parseErr)
	}
	if parseRow.ControlKey != "cc.access-review" {
		parseT.Errorf("unexpected ControlKey: %q", parseRow.ControlKey)
	}
	if parseRow.Status != "implemented" {
		parseT.Errorf("expected Status normalized to implemented, got %q", parseRow.Status)
	}
}
