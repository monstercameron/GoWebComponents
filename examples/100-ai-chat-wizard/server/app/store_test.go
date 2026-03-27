package app

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
)

func TestStoreConversationAndPreferenceLifecycle(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "demo@example.com")

	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	if parseErr2 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "user", "Hello", "", 0, 0); parseErr2 != nil {
		parseT.Fatalf("saveConversationMessage user: %v", parseErr2)
	}
	if parseErr3 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "assistant", "Hi there", modelGPT54Mini, 11, 7); parseErr3 != nil {
		parseT.Fatalf("saveConversationMessage assistant: %v", parseErr3)
	}
	if parseErr4 := store.parseSaveConversationTitle(parseUser.ID, parseConversationID, "Greeting thread"); parseErr4 != nil {
		parseT.Fatalf("saveConversationTitle: %v", parseErr4)
	}

	parseOwned, parseErr := store.parseConversationOwnedByUser(parseUser.ID, parseConversationID)
	if parseErr != nil {
		parseT.Fatalf("conversationOwnedByUser: %v", parseErr)
	}
	if !parseOwned {
		parseT.Fatal("expected conversation to belong to user")
	}

	parseConversations, parseErr := store.parseListConversations(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations: %v", parseErr)
	}
	if len(parseConversations) != 1 {
		parseT.Fatalf("expected one conversation, got %d", len(parseConversations))
	}
	if parseConversations[0].PublicID == "" {
		parseT.Fatal("expected conversation public_id to be populated")
	}
	if _, parseErr5 := uuid.Parse(parseConversations[0].PublicID); parseErr5 != nil {
		parseT.Fatalf("expected conversation public_id to be a UUID, got %q: %v", parseConversations[0].PublicID, parseErr5)
	}
	if parseConversations[0].Preview != "Greeting thread" {
		parseT.Fatalf("expected title-backed preview, got %q", parseConversations[0].Preview)
	}

	parseMessages, parseErr := store.parseLoadConversation(parseUser.ID, parseConversationID)
	if parseErr != nil {
		parseT.Fatalf("loadConversation: %v", parseErr)
	}
	if len(parseMessages) != 2 {
		parseT.Fatalf("expected two messages, got %d", len(parseMessages))
	}
	if parseMessages[1].ModelID != modelGPT54Mini || parseMessages[1].PromptTokens != 11 || parseMessages[1].CompletionTokens != 7 {
		parseT.Fatalf("assistant message metadata mismatch: %+v", parseMessages[1])
	}

	parseUpdatedAt := time.Now().Unix()
	if parseErr6 := store.setUserName(parseUser.ID, "Renamed User", parseUpdatedAt); parseErr6 != nil {
		parseT.Fatalf("setUserName: %v", parseErr6)
	}
	parseName, parseGotUpdatedAt, parseErr := store.getUserName(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("getUserName: %v", parseErr)
	}
	if parseName != "Renamed User" || parseGotUpdatedAt != parseUpdatedAt {
		parseT.Fatalf("unexpected user name state: name=%q updatedAt=%d", parseName, parseGotUpdatedAt)
	}

	if parseErr7 := store.setSelectedModel(parseUser.ID, modelGPT54); parseErr7 != nil {
		parseT.Fatalf("setSelectedModel: %v", parseErr7)
	}
	if parseErr8 := store.setSelectedTone(parseUser.ID, "professional"); parseErr8 != nil {
		parseT.Fatalf("setSelectedTone: %v", parseErr8)
	}
	if parseErr9 := store.setSelectedThinkingEnabled(parseUser.ID, false); parseErr9 != nil {
		parseT.Fatalf("setSelectedThinkingEnabled: %v", parseErr9)
	}
	if parseErr10 := store.setSelectedThinkingEffort(parseUser.ID, "high"); parseErr10 != nil {
		parseT.Fatalf("setSelectedThinkingEffort: %v", parseErr10)
	}
	if parseErr11 := store.setSelectedSystemPrompt(parseUser.ID, "Always answer with short bullet points."); parseErr11 != nil {
		parseT.Fatalf("setSelectedSystemPrompt: %v", parseErr11)
	}
	if parseErr12 := store.parseUpsertUserMemory(parseUser.ID, userMemoryRow{
		Key:             "preference-editor",
		Category:        "preference",
		Summary:         "Prefers Neovim",
		Detail:          "Uses Neovim daily for coding",
		SourceMessage:   "I use Neovim for most of my work.",
		UsefulnessScore: 84,
		ConfidenceScore: 0.91,
		RubricReason:    "Stable tooling preference",
	}); parseErr12 != nil {
		parseT.Fatalf("upsertUserMemory: %v", parseErr12)
	}

	parseSelectedModel, parseErr := store.getSelectedModel(parseUser.ID, modelGPT54Mini)
	if parseErr != nil || parseSelectedModel != modelGPT54 {
		parseT.Fatalf("getSelectedModel: model=%q err=%v", parseSelectedModel, parseErr)
	}
	parseSelectedTone, parseErr := store.getSelectedTone(parseUser.ID, defaultToneID)
	if parseErr != nil || parseSelectedTone != "professional" {
		parseT.Fatalf("getSelectedTone: tone=%q err=%v", parseSelectedTone, parseErr)
	}
	parseThinkingEnabled, parseErr := store.getSelectedThinkingEnabled(parseUser.ID, true)
	if parseErr != nil || parseThinkingEnabled {
		parseT.Fatalf("getSelectedThinkingEnabled: enabled=%v err=%v", parseThinkingEnabled, parseErr)
	}
	parseThinkingEffort, parseErr := store.getSelectedThinkingEffort(parseUser.ID, defaultThinkingEffort)
	if parseErr != nil || parseThinkingEffort != "high" {
		parseT.Fatalf("getSelectedThinkingEffort: effort=%q err=%v", parseThinkingEffort, parseErr)
	}
	parseCustomSystemPrompt, parseErr := store.getSelectedSystemPrompt(parseUser.ID, "")
	if parseErr != nil || parseCustomSystemPrompt != "Always answer with short bullet points." {
		parseT.Fatalf("getSelectedSystemPrompt: prompt=%q err=%v", parseCustomSystemPrompt, parseErr)
	}
	parseMemories, parseErr := store.parseListUserMemories(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listUserMemories: %v", parseErr)
	}
	if len(parseMemories) != 1 || parseMemories[0].Summary != "Prefers Neovim" {
		parseT.Fatalf("unexpected stored memories: %+v", parseMemories)
	}

	if parseErr13 := store.parseDeleteConversation(parseUser.ID, parseConversationID); parseErr13 != nil {
		parseT.Fatalf("deleteConversation: %v", parseErr13)
	}
	parseRemaining, parseErr := store.parseListConversations(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations after delete: %v", parseErr)
	}
	if len(parseRemaining) != 0 {
		parseT.Fatalf("expected zero conversations after delete, got %d", len(parseRemaining))
	}
}

func TestStoreFallbacksAndUniqueness(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUserID, parseErr := store.parseCreateUser("fallback@example.com", "hash", "")
	if parseErr != nil {
		parseT.Fatalf("createUser: %v", parseErr)
	}
	if _, parseErr2 := store.parseCreateUser("fallback@example.com", "hash", ""); parseErr2 != errUserAlreadyExists {
		parseT.Fatalf("expected duplicate user error, got %v", parseErr2)
	}

	parseName, parseUpdatedAt, parseErr := store.getUserName(parseUserID + 100)
	if parseErr != nil {
		parseT.Fatalf("getUserName fallback: %v", parseErr)
	}
	if parseName != "User" || parseUpdatedAt != 0 {
		parseT.Fatalf("unexpected fallback profile: name=%q updatedAt=%d", parseName, parseUpdatedAt)
	}

	parseSelectedModel, parseErr := store.getSelectedModel(parseUserID+100, modelGPT54Mini)
	if parseErr != nil || parseSelectedModel != modelGPT54Mini {
		parseT.Fatalf("fallback selected model mismatch: model=%q err=%v", parseSelectedModel, parseErr)
	}
	parseSelectedTone, parseErr := store.getSelectedTone(parseUserID+100, defaultToneID)
	if parseErr != nil || parseSelectedTone != defaultToneID {
		parseT.Fatalf("fallback selected tone mismatch: tone=%q err=%v", parseSelectedTone, parseErr)
	}
	parseThinkingEnabled, parseErr := store.getSelectedThinkingEnabled(parseUserID+100, true)
	if parseErr != nil || !parseThinkingEnabled {
		parseT.Fatalf("fallback thinking enabled mismatch: enabled=%v err=%v", parseThinkingEnabled, parseErr)
	}
	parseThinkingEffort, parseErr := store.getSelectedThinkingEffort(parseUserID+100, defaultThinkingEffort)
	if parseErr != nil || parseThinkingEffort != defaultThinkingEffort {
		parseT.Fatalf("fallback thinking effort mismatch: effort=%q err=%v", parseThinkingEffort, parseErr)
	}
	parseCustomSystemPrompt, parseErr := store.getSelectedSystemPrompt(parseUserID+100, "")
	if parseErr != nil || parseCustomSystemPrompt != "" {
		parseT.Fatalf("fallback custom system prompt mismatch: prompt=%q err=%v", parseCustomSystemPrompt, parseErr)
	}

	parseDerivedName, parseDerivedUpdatedAt, parseErr := store.getUserName(parseUserID)
	if parseErr != nil {
		parseT.Fatalf("getUserName derived: %v", parseErr)
	}
	if parseDerivedName != "fallback" || parseDerivedUpdatedAt == 0 {
		parseT.Fatalf("expected default display name derived from email, got name=%q updatedAt=%d", parseDerivedName, parseDerivedUpdatedAt)
	}

	parseOwned, parseErr := store.parseConversationOwnedByUser(parseUserID, parseUserID+999)
	if parseErr != nil {
		parseT.Fatalf("conversationOwnedByUser false branch: %v", parseErr)
	}
	if parseOwned {
		parseT.Fatal("expected unrelated conversation to not belong to user")
	}
}

func TestCreateConversationRetriesPublicIDConflicts(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "uuid-retry@example.com")

	parseOriginalGenerator := newConversationPublicID
	defer func() { newConversationPublicID = parseOriginalGenerator }()

	parseCollisionID := uuid.NewString()
	parseRecoveredID := uuid.NewString()
	parseCallCount := 0
	newConversationPublicID = func() string {
		parseCallCount++
		if parseCallCount <= 2 {
			return parseCollisionID
		}
		return parseRecoveredID
	}

	parseFirstConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation first: %v", parseErr)
	}
	parseSecondConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation second: %v", parseErr)
	}
	if parseFirstConversationID == parseSecondConversationID {
		parseT.Fatalf("expected distinct conversation rows, got %d and %d", parseFirstConversationID, parseSecondConversationID)
	}

	parseConversations, parseErr := store.parseListConversations(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations: %v", parseErr)
	}
	if len(parseConversations) != 2 {
		parseT.Fatalf("expected two conversations, got %d", len(parseConversations))
	}

	parsePublicIDs := map[string]bool{}
	for _, parseConversation := range parseConversations {
		if parseConversation.PublicID == "" {
			parseT.Fatal("expected non-empty public_id")
		}
		if _, parseErr2 := uuid.Parse(parseConversation.PublicID); parseErr2 != nil {
			parseT.Fatalf("expected parseable UUID, got %q: %v", parseConversation.PublicID, parseErr2)
		}
		if parsePublicIDs[parseConversation.PublicID] {
			parseT.Fatalf("expected unique public_ids, got duplicate %q", parseConversation.PublicID)
		}
		parsePublicIDs[parseConversation.PublicID] = true
	}
	if !parsePublicIDs[parseCollisionID] {
		parseT.Fatalf("expected initial conversation to keep first generated UUID %q", parseCollisionID)
	}
	if !parsePublicIDs[parseRecoveredID] {
		parseT.Fatalf("expected retry path to use fallback UUID %q", parseRecoveredID)
	}
	if parseCallCount < 3 {
		parseT.Fatalf("expected generator to be called at least 3 times, got %d", parseCallCount)
	}
}

func TestCreateConversationReturnsUserMissingWhenParentUserDoesNotExist(parseT *testing.T) {
	store := parseNewTestStore(parseT)

	_, parseErr := store.parseCreateConversation(999999)
	if !errors.Is(parseErr, errStoreUserMissing) {
		parseT.Fatalf("expected errStoreUserMissing, got %v", parseErr)
	}
}

func TestSaveConversationMessageReturnsConversationMissingWhenParentConversationDeleted(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "missing-conversation@example.com")
	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	if parseErr2 := store.parseDeleteConversation(parseUser.ID, parseConversationID); parseErr2 != nil {
		parseT.Fatalf("deleteConversation: %v", parseErr2)
	}

	parseErr = store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "user", "hello", "", 0, 0)
	if !errors.Is(parseErr, errStoreConversationMissing) {
		parseT.Fatalf("expected errStoreConversationMissing, got %v", parseErr)
	}
}

// TestStoreUsageEventLifecycleAndUserScoping verifies persisted usage events are immutable and user-scoped.
func TestStoreUsageEventLifecycleAndUserScoping(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, store, "usage-owner@example.com")
	parseOther := parseMustCreateUser(parseT, store, "usage-other@example.com")

	parseConversationID, parseErr := store.parseCreateConversation(parseOwner.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation owner: %v", parseErr)
	}
	if _, parseErr2 := store.parseCreateConversation(parseOther.ID); parseErr2 != nil {
		parseT.Fatalf("createConversation other: %v", parseErr2)
	}

	parseEventID := uuid.NewString()
	parseUsage := parseUsageEventWrite{
		EventID:                 parseEventID,
		UserID:                  parseOwner.ID,
		ConversationID:          parseConversationID,
		ProviderID:              "openai",
		ModelID:                 modelGPT54Mini,
		PromptTokens:            150,
		CompletionTokens:        30,
		UsageSource:             "exact",
		ProviderRequestID:       "resp_123",
		InputCostPerMillionUSD:  0.25,
		OutputCostPerMillionUSD: 2.00,
		PricingCurrency:         "USD",
		InputCostUSD:            0.0000375,
		OutputCostUSD:           0.00006,
		TotalCostUSD:            0.0000975,
		ClientID:                uuid.NewString(),
		TraceID:                 "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:                  "00f067aa0ba902b7",
		TraceState:              "vendor=relay",
		Status:                  "completed",
		ErrorMessage:            "",
	}
	if parseErr3 := store.parseSaveUsageEvent(parseUsage); parseErr3 != nil {
		parseT.Fatalf("parseSaveUsageEvent: %v", parseErr3)
	}

	parseOwnerEvents, parseErr := store.parseListUsageEvents(parseOwner.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListUsageEvents owner: %v", parseErr)
	}
	if len(parseOwnerEvents) != 1 {
		parseT.Fatalf("expected one owner usage event, got %d", len(parseOwnerEvents))
	}
	parseSaved := parseOwnerEvents[0]
	if parseSaved.EventID != parseEventID || parseSaved.ProviderID != "openai" || parseSaved.ModelID != modelGPT54Mini {
		parseT.Fatalf("unexpected usage event identity: %+v", parseSaved)
	}
	if parseSaved.PromptTokens != 150 || parseSaved.CompletionTokens != 30 || parseSaved.UsageSource != "exact" {
		parseT.Fatalf("unexpected usage tokens/source: %+v", parseSaved)
	}
	if parseSaved.Status != "completed" || parseSaved.CreatedAt == "" {
		parseT.Fatalf("unexpected usage status/timestamp: %+v", parseSaved)
	}

	parseOtherEvents, parseErr := store.parseListUsageEvents(parseOther.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListUsageEvents other: %v", parseErr)
	}
	if len(parseOtherEvents) != 0 {
		parseT.Fatalf("expected zero usage events for other user, got %d", len(parseOtherEvents))
	}
}

// TestStoreUsageEventValidatesConversationOwnership verifies usage writes cannot target missing conversations.
func TestStoreUsageEventValidatesConversationOwnership(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "usage-missing@example.com")
	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	if parseErr2 := store.parseDeleteConversation(parseUser.ID, parseConversationID); parseErr2 != nil {
		parseT.Fatalf("deleteConversation: %v", parseErr2)
	}

	parseErr = store.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:        uuid.NewString(),
		UserID:         parseUser.ID,
		ConversationID: parseConversationID,
		ProviderID:     "openai",
		ModelID:        modelGPT54Mini,
		Status:         "failed",
	})
	if !errors.Is(parseErr, errStoreConversationMissing) {
		parseT.Fatalf("expected errStoreConversationMissing, got %v", parseErr)
	}
}

// TestStoreGetModelPricingForModel verifies model pricing snapshots resolve from the catalog.
func TestStoreGetModelPricingForModel(parseT *testing.T) {
	store := parseNewTestStore(parseT)

	parseProviderID, parsePricing, hasParsePricing, parseErr := store.parseGetModelPricingForModel(modelGPT54Mini)
	if parseErr != nil {
		parseT.Fatalf("parseGetModelPricingForModel existing: %v", parseErr)
	}
	if !hasParsePricing {
		parseT.Fatal("expected existing model pricing to be found")
	}
	if parseProviderID != "openai" {
		parseT.Fatalf("unexpected provider id: %q", parseProviderID)
	}
	if parsePricing.InputPerMillionUSD <= 0 || parsePricing.OutputPerMillionUSD <= 0 || parsePricing.Currency != "USD" {
		parseT.Fatalf("unexpected pricing payload: %+v", parsePricing)
	}

	parseProviderID, parsePricing, hasParsePricing, parseErr = store.parseGetModelPricingForModel("missing-model")
	if parseErr != nil {
		parseT.Fatalf("parseGetModelPricingForModel missing: %v", parseErr)
	}
	if hasParsePricing || parseProviderID != "" || (parsePricing != (provider.ModelPricing{})) {
		parseT.Fatalf("unexpected missing pricing result: provider=%q pricing=%+v found=%v", parseProviderID, parsePricing, hasParsePricing)
	}
}

func TestResolveConversationRouteIsOwnerScoped(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, store, "route-owner@example.com")
	parseOther := parseMustCreateUser(parseT, store, "route-other@example.com")

	parseConversationID, parseErr := store.parseCreateConversation(parseOwner.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	parseConversations, parseErr := store.parseListConversations(parseOwner.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations: %v", parseErr)
	}
	if len(parseConversations) != 1 {
		parseT.Fatalf("expected one conversation, got %d", len(parseConversations))
	}
	parsePublicID := parseConversations[0].PublicID

	parseSummary, parseOk, parseErr := store.parseResolveConversationRoute(parseOwner.ID, parsePublicID)
	if parseErr != nil {
		parseT.Fatalf("resolveConversationRoute owner: %v", parseErr)
	}
	if !parseOk || parseSummary.ID != parseConversationID || parseSummary.PublicID != parsePublicID {
		parseT.Fatalf("unexpected owner route resolution: ok=%v summary=%+v", parseOk, parseSummary)
	}

	parseSummary, parseOk, parseErr = store.parseResolveConversationRoute(parseOther.ID, parsePublicID)
	if parseErr != nil {
		parseT.Fatalf("resolveConversationRoute other: %v", parseErr)
	}
	if parseOk {
		parseT.Fatalf("expected non-owner route resolution to be inaccessible, got %+v", parseSummary)
	}
}

// TestStoreBillingVisibilityAccessAndControlLifecycle verifies billing persistence supports audit visibility and explicit access control.
func TestStoreBillingVisibilityAccessAndControlLifecycle(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "billing-owner@example.com")

	parseCustomer, parseErr := store.parseUpsertBillingCustomer(parseBillingCustomerWrite{
		UserID:               parseUser.ID,
		ProviderID:           "stripe",
		ProviderCustomerID:   "cus_test_123",
		BillingEmail:         "billing-owner@example.com",
		BillingName:          "Billing Owner",
		BillingCountry:       "US",
		BillingRegion:        "NY",
		DefaultCurrency:      "usd",
		TaxExemptStatus:      "none",
		ExternalMetadataJSON: `{"segment":"beta"}`,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingCustomer: %v", parseErr)
	}
	if parseCustomer.ID <= 0 || parseCustomer.DefaultCurrency != "USD" {
		parseT.Fatalf("unexpected customer row: %+v", parseCustomer)
	}

	parseCustomerRead, hasParseCustomerRead, parseErr := store.parseGetBillingCustomerByUser(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !hasParseCustomerRead || parseCustomerRead.ID != parseCustomer.ID {
		parseT.Fatalf("expected to resolve customer by user, got row=%+v found=%v", parseCustomerRead, hasParseCustomerRead)
	}

	parseNow := time.Now().UTC()
	parseSubscription, parseErr := store.parseUpsertBillingSubscription(parseBillingSubscriptionWrite{
		CustomerID:             parseCustomer.ID,
		ProviderID:             "stripe",
		ProviderSubscriptionID: "sub_test_123",
		PlanCode:               "team",
		PriceCode:              "price_team_monthly",
		Status:                 "active",
		BillingInterval:        "month",
		Quantity:               3,
		CurrentPeriodStart:     parseNow.Format(time.RFC3339),
		CurrentPeriodEnd:       parseNow.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingSubscription: %v", parseErr)
	}
	if parseSubscription.ID <= 0 || parseSubscription.PlanCode != "team" {
		parseT.Fatalf("unexpected subscription row: %+v", parseSubscription)
	}

	parseSubscriptions, parseErr := store.parseListBillingSubscriptionsByCustomer(parseCustomer.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingSubscriptionsByCustomer: %v", parseErr)
	}
	if len(parseSubscriptions) != 1 || parseSubscriptions[0].ProviderSubscriptionID != "sub_test_123" {
		parseT.Fatalf("unexpected subscription list: %+v", parseSubscriptions)
	}

	parseInvoice, parseErr := store.parseUpsertBillingInvoice(parseBillingInvoiceWrite{
		CustomerID:           parseCustomer.ID,
		SubscriptionID:       parseSubscription.ID,
		ProviderID:           "stripe",
		ProviderInvoiceID:    "in_test_123",
		Status:               "open",
		Currency:             "usd",
		SubtotalCents:        9900,
		TaxCents:             800,
		DiscountCents:        1000,
		TotalCents:           9700,
		AmountDueCents:       9700,
		AmountPaidCents:      0,
		PeriodStart:          parseNow.Format(time.RFC3339),
		PeriodEnd:            parseNow.Add(30 * 24 * time.Hour).Format(time.RFC3339),
		DueAt:                parseNow.Add(7 * 24 * time.Hour).Format(time.RFC3339),
		PaidAt:               "",
		HostedInvoiceURL:     "https://example.test/invoice/in_test_123",
		ExternalMetadataJSON: `{"source":"unit-test"}`,
	})
	if parseErr != nil {
		parseT.Fatalf("parseUpsertBillingInvoice: %v", parseErr)
	}
	if parseInvoice.ID <= 0 || parseInvoice.Currency != "USD" {
		parseT.Fatalf("unexpected invoice row: %+v", parseInvoice)
	}

	parseInvoices, parseErr := store.parseListBillingInvoicesByCustomer(parseCustomer.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingInvoicesByCustomer: %v", parseErr)
	}
	if len(parseInvoices) != 1 || parseInvoices[0].ProviderInvoiceID != "in_test_123" {
		parseT.Fatalf("unexpected invoice list: %+v", parseInvoices)
	}

	if _, parseErr = store.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseCustomer.ID + 777,
		InvoiceID:       parseInvoice.ID,
		LineType:        "usage",
		Description:     "wrong customer should fail",
		Quantity:        1,
		UnitAmountCents: 100,
		AmountCents:     100,
		Currency:        "usd",
	}); !errors.Is(parseErr, errStoreBillingInvoiceMissing) {
		parseT.Fatalf("expected ownership guard for invoice line item, got %v", parseErr)
	}

	parseLineItemID, parseErr := store.parseCreateBillingInvoiceLineItem(parseBillingInvoiceLineItemWrite{
		CustomerID:      parseCustomer.ID,
		InvoiceID:       parseInvoice.ID,
		UsageEventID:    "",
		LineType:        "subscription",
		Description:     "Team monthly seat bundle",
		Quantity:        3,
		UnitAmountCents: 3300,
		AmountCents:     9900,
		Currency:        "usd",
		PeriodStart:     parseNow.Format(time.RFC3339),
		PeriodEnd:       parseNow.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	})
	if parseErr != nil {
		parseT.Fatalf("parseCreateBillingInvoiceLineItem: %v", parseErr)
	}
	if parseLineItemID <= 0 {
		parseT.Fatalf("expected line item id, got %d", parseLineItemID)
	}

	parseLineItems, parseErr := store.parseListBillingInvoiceLineItems(parseInvoice.ID, parseCustomer.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingInvoiceLineItems: %v", parseErr)
	}
	if len(parseLineItems) != 1 || parseLineItems[0].AmountCents != 9900 {
		parseT.Fatalf("unexpected line item list: %+v", parseLineItems)
	}

	if parseErr2 := store.parseUpsertBillingAccessOverride(parseBillingAccessOverrideWrite{
		CustomerID:    parseCustomer.ID,
		OverrideKey:   "chat.send.enabled",
		OverrideValue: "false",
		Reason:        "temporary abuse hold",
		IsEnabled:     true,
		StartsAt:      parseNow.Add(-1 * time.Hour).Format(time.RFC3339),
		EndsAt:        parseNow.Add(24 * time.Hour).Format(time.RFC3339),
		ActorUserID:   7,
	}); parseErr2 != nil {
		parseT.Fatalf("parseUpsertBillingAccessOverride: %v", parseErr2)
	}

	parseOverrides, parseErr := store.parseListBillingAccessOverridesByCustomer(parseCustomer.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingAccessOverridesByCustomer: %v", parseErr)
	}
	if len(parseOverrides) != 1 || parseOverrides[0].OverrideKey != "chat.send.enabled" {
		parseT.Fatalf("unexpected overrides: %+v", parseOverrides)
	}

	parseEventID, parseErr := store.parseCreateBillingEvent(parseBillingEventWrite{
		CustomerID:       parseCustomer.ID,
		SubscriptionID:   parseSubscription.ID,
		InvoiceID:        parseInvoice.ID,
		EventType:        "subscription.updated",
		EventSource:      "admin",
		EventSummary:     "Downgraded sending access",
		EventPayloadJSON: `{"override_key":"chat.send.enabled","value":"false"}`,
		ActorUserID:      7,
	})
	if parseErr != nil {
		parseT.Fatalf("parseCreateBillingEvent: %v", parseErr)
	}
	if parseEventID <= 0 {
		parseT.Fatalf("expected billing event id, got %d", parseEventID)
	}

	parseEvents, parseErr := store.parseListBillingEventsByCustomer(parseCustomer.ID, 10)
	if parseErr != nil {
		parseT.Fatalf("parseListBillingEventsByCustomer: %v", parseErr)
	}
	if len(parseEvents) != 1 || parseEvents[0].EventType != "subscription.updated" {
		parseT.Fatalf("unexpected billing events: %+v", parseEvents)
	}

	parseEntitlements, parseErr := store.parseListBillingPlanEntitlements("team")
	if parseErr != nil {
		parseT.Fatalf("parseListBillingPlanEntitlements: %v", parseErr)
	}
	if len(parseEntitlements) == 0 {
		parseT.Fatal("expected team plan entitlements to be seeded")
	}

	parseAccessControl, parseErr := store.parseGetBillingAccessControlByUser(parseUser.ID, parseNow)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingAccessControlByUser: %v", parseErr)
	}
	parseChatControl, hasParseChatControl := parseAccessControl["chat.send.enabled"]
	if !hasParseChatControl || parseChatControl.AccessValue != "false" || parseChatControl.SourceType != "override" {
		parseT.Fatalf("expected override to control chat.send.enabled, got %+v", parseChatControl)
	}
	parseTokenControl, hasParseTokenControl := parseAccessControl["usage.monthly_token_limit"]
	if !hasParseTokenControl || parseTokenControl.AccessValue != "20000000" {
		parseT.Fatalf("expected plan token entitlement, got %+v", parseTokenControl)
	}
}

func TestOpenChatStoreRecoversFromIncompatibleLegacySchema(parseT *testing.T) {
	parseDbPath := filepath.Join(parseT.TempDir(), "chat_history.db")

	parseDb, parseErr := sql.Open("sqlite3", "file:"+parseDbPath)
	if parseErr != nil {
		parseT.Fatalf("sql.Open: %v", parseErr)
	}
	if _, parseErr2 := parseDb.Exec(`CREATE TABLE user_profile (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`); parseErr2 != nil {
		_ = parseDb.Close()
		parseT.Fatalf("create legacy schema: %v", parseErr2)
	}
	if parseErr3 := parseDb.Close(); parseErr3 != nil {
		parseT.Fatalf("close legacy db: %v", parseErr3)
	}

	store, parseErr := parseOpenChatStore(parseDbPath)
	if parseErr != nil {
		parseT.Fatalf("openChatStore recovery: %v", parseErr)
	}
	defer store.parseClose()

	parseMatches, parseErr := filepath.Glob(parseDbPath + ".incompatible-*.bak")
	if parseErr != nil {
		parseT.Fatalf("glob backup: %v", parseErr)
	}
	if len(parseMatches) != 1 {
		parseT.Fatalf("expected one backup file, got %v", parseMatches)
	}
	if _, parseErr4 := os.Stat(parseDbPath); parseErr4 != nil {
		parseT.Fatalf("expected recreated db at %q: %v", parseDbPath, parseErr4)
	}

	parseUserID, parseErr := store.parseCreateUser("recover@example.com", "hash", "Recover")
	if parseErr != nil {
		parseT.Fatalf("createUser after recovery: %v", parseErr)
	}
	if parseUserID <= 0 {
		parseT.Fatalf("expected valid recovered user id, got %d", parseUserID)
	}
}

func TestOpenChatStoreCreatesMissingParentDirectory(parseT *testing.T) {
	parseDbPath := filepath.Join(parseT.TempDir(), "missing", "runtime", "chat_history.db")
	parseDbDir := filepath.Dir(parseDbPath)

	if _, parseErr := os.Stat(parseDbDir); !errors.Is(parseErr, os.ErrNotExist) {
		parseT.Fatalf("expected missing db directory before open, stat err=%v", parseErr)
	}

	store, parseErr2 := parseOpenChatStore(parseDbPath)
	if parseErr2 != nil {
		parseT.Fatalf("openChatStore create parent dir: %v", parseErr2)
	}
	defer store.parseClose()

	if _, parseErr3 := os.Stat(parseDbDir); parseErr3 != nil {
		parseT.Fatalf("expected db directory to exist after open, got %v", parseErr3)
	}
	if _, parseErr4 := os.Stat(parseDbPath); parseErr4 != nil {
		parseT.Fatalf("expected db file to exist after open, got %v", parseErr4)
	}
}
