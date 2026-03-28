package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequireUserEntitlementDenyByDefault(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := &chatServer{store: parseStore, logger: parseNewTestLogger()}
	parseUser := parseMustCreateUser(parseT, parseStore, "deny-by-default@example.com")

	parseErr := parseServer.parseRequireUserEntitlement(parseUser.ID, billingEntitlementChatSendEnabled)
	if status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected missing entitlement to deny, got code=%v err=%v", status.Code(parseErr), parseErr)
	}
}

func TestRequireUserEntitlementAllowsEffectiveAccess(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := &chatServer{store: parseStore, logger: parseNewTestLogger()}
	parseUser := parseMustCreateUser(parseT, parseStore, "entitled@example.com")
	parseMustAssignBillingPlan(parseT, parseStore, parseUser.ID, "free")

	if parseErr := parseServer.parseRequireUserEntitlement(parseUser.ID, billingEntitlementChatSendEnabled); parseErr != nil {
		parseT.Fatalf("expected active entitlement to pass, got %v", parseErr)
	}
}

// TestRequireUsageBudgetEnforcesMonthlyTokenLimit validates monthly token quota blocking.
func TestRequireUsageBudgetEnforcesMonthlyTokenLimit(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := &chatServer{store: parseStore, logger: parseNewTestLogger()}
	parseUser := parseMustCreateUser(parseT, parseStore, "usage-monthly@example.com")
	parseMustAssignBillingPlan(parseT, parseStore, parseUser.ID, "free")
	parseMustUpsertBillingOverride(parseT, parseStore, parseUser.ID, billingEntitlementUsageMonthlyTokenLimit, "10")
	parseMustSaveUsageEventTokens(parseT, parseStore, parseUser.ID, 8, 5)

	parseReleaseBudget, parseErr := parseServer.parseRequireUsageBudget(parseUser.ID)
	parseReleaseBudget()
	if status.Code(parseErr) != codes.ResourceExhausted {
		parseT.Fatalf("expected resource exhausted for monthly limit, got code=%v err=%v", status.Code(parseErr), parseErr)
	}
}

// TestRequireUsageBudgetEnforcesPerUserRate validates per-minute send-rate blocking.
func TestRequireUsageBudgetEnforcesPerUserRate(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := &chatServer{store: parseStore, logger: parseNewTestLogger()}
	parseUser := parseMustCreateUser(parseT, parseStore, "usage-rate@example.com")
	parseMustAssignBillingPlan(parseT, parseStore, parseUser.ID, "free")
	parseMustUpsertBillingOverride(parseT, parseStore, parseUser.ID, billingEntitlementUsageSendsPerMinute, "1")
	parseMustUpsertBillingOverride(parseT, parseStore, parseUser.ID, billingEntitlementUsageConcurrentSends, "unlimited")

	parseReleaseBudget, parseErr := parseServer.parseRequireUsageBudget(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("expected first budget check to pass, got %v", parseErr)
	}
	parseReleaseBudget()

	parseReleaseBudget2, parseErr := parseServer.parseRequireUsageBudget(parseUser.ID)
	parseReleaseBudget2()
	if status.Code(parseErr) != codes.ResourceExhausted {
		parseT.Fatalf("expected resource exhausted for per-user rate, got code=%v err=%v", status.Code(parseErr), parseErr)
	}
}

// TestRequireUsageBudgetEnforcesPerUserConcurrency validates in-flight send concurrency blocking.
func TestRequireUsageBudgetEnforcesPerUserConcurrency(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseServer := &chatServer{store: parseStore, logger: parseNewTestLogger()}
	parseUser := parseMustCreateUser(parseT, parseStore, "usage-concurrency@example.com")
	parseMustAssignBillingPlan(parseT, parseStore, parseUser.ID, "free")
	parseMustUpsertBillingOverride(parseT, parseStore, parseUser.ID, billingEntitlementUsageSendsPerMinute, "unlimited")
	parseMustUpsertBillingOverride(parseT, parseStore, parseUser.ID, billingEntitlementUsageConcurrentSends, "1")

	parseReleaseBudget, parseErr := parseServer.parseRequireUsageBudget(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("expected first budget check to pass, got %v", parseErr)
	}
	parseReleaseBudget2, parseErr := parseServer.parseRequireUsageBudget(parseUser.ID)
	parseReleaseBudget2()
	if status.Code(parseErr) != codes.ResourceExhausted {
		parseReleaseBudget()
		parseT.Fatalf("expected resource exhausted for per-user concurrency, got code=%v err=%v", status.Code(parseErr), parseErr)
	}
	parseReleaseBudget()

	parseReleaseBudget3, parseErr := parseServer.parseRequireUsageBudget(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("expected budget check to pass after release, got %v", parseErr)
	}
	parseReleaseBudget3()
}

// BenchmarkRequireUsageBudget measures per-call overhead for usage budget checks.
func BenchmarkRequireUsageBudget(parseB *testing.B) {
	parseStore := parseNewBenchmarkStore(parseB)
	parseUser := parseMustCreateBenchmarkUser(parseB, parseStore, "budget-bench@example.com")
	parseServer := &chatServer{store: parseStore, logger: parseNewBenchmarkLogger()}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseReleaseBudget, parseErr := parseServer.parseRequireUsageBudget(parseUser.ID)
		if parseErr != nil {
			parseB.Fatalf("parseRequireUsageBudget: %v", parseErr)
		}
		parseReleaseBudget()
	}
}

// parseMustUpsertBillingOverride writes one billing access override for one test user.
func parseMustUpsertBillingOverride(parseT *testing.T, parseStore *Store, parseUserID int64, parseOverrideKey, parseOverrideValue string) {
	parseT.Helper()
	parseCustomer, hasParseCustomer, parseErr := parseStore.parseGetBillingCustomerByUser(parseUserID)
	if parseErr != nil {
		parseT.Fatalf("parseGetBillingCustomerByUser: %v", parseErr)
	}
	if !hasParseCustomer {
		parseT.Fatalf("missing billing customer for user %d", parseUserID)
	}
	if parseErr2 := parseStore.parseUpsertBillingAccessOverride(parseBillingAccessOverrideWrite{
		CustomerID:    parseCustomer.ID,
		OverrideKey:   parseOverrideKey,
		OverrideValue: parseOverrideValue,
		Reason:        "test override",
		IsEnabled:     true,
		StartsAt:      time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		EndsAt:        time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		ActorUserID:   parseUserID,
	}); parseErr2 != nil {
		parseT.Fatalf("parseUpsertBillingAccessOverride: %v", parseErr2)
	}
}

// parseMustSaveUsageEventTokens writes one usage event row with token totals for one user.
func parseMustSaveUsageEventTokens(parseT *testing.T, parseStore *Store, parseUserID int64, parsePromptTokens, parseCompletionTokens int64) {
	parseT.Helper()
	parseConversationID, parseErr := parseStore.parseCreateConversation(parseUserID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation: %v", parseErr)
	}
	parseEventID := fmt.Sprintf("usage-budget-%d", time.Now().UTC().UnixNano())
	if parseErr2 := parseStore.parseSaveUsageEvent(parseUsageEventWrite{
		EventID:          parseEventID,
		UserID:           parseUserID,
		ConversationID:   parseConversationID,
		ProviderID:       "test",
		ModelID:          modelGPT54Mini,
		PromptTokens:     parsePromptTokens,
		CompletionTokens: parseCompletionTokens,
		UsageSource:      provider.UsageSourceExact,
		Status:           "completed",
	}); parseErr2 != nil {
		parseT.Fatalf("parseSaveUsageEvent: %v", parseErr2)
	}
}
