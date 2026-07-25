//go:build js && wasm

package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
)

// ─── account_costs.go ─────────────────────────────────────────────────────────

// TestParseBuildAccountCostSummaryFromBillingSummaryNilReturnsZero verifies that
// passing a nil proto response returns a zeroed-out accountCostSummary using the
// provided default premium percent and platform fee only, with no cost fields set.
func TestParseBuildAccountCostSummaryFromBillingSummaryNilReturnsZero(parseT *testing.T) {
	parseT.Parallel()
	parseSummary := parseBuildAccountCostSummaryFromBillingSummary(nil, 10.0, 29.0)
	if parseSummary.PlatformFee != 0 {
		parseT.Errorf("PlatformFee = %v, want 0 for nil response", parseSummary.PlatformFee)
	}
	if parseSummary.UsageCost != 0 {
		parseT.Errorf("UsageCost = %v, want 0 for nil response", parseSummary.UsageCost)
	}
	if parseSummary.TotalCost != 0 {
		parseT.Errorf("TotalCost = %v, want 0 for nil response", parseSummary.TotalCost)
	}
	if parseSummary.HasAnyExactCosts {
		parseT.Error("HasAnyExactCosts = true, want false for nil response")
	}
}

// TestParseBuildAccountCostSummaryFromBillingSummaryCentConversion verifies that
// cent fields from the billing totals proto are correctly divided by 100 to dollars.
func TestParseBuildAccountCostSummaryFromBillingSummaryCentConversion(parseT *testing.T) {
	parseT.Parallel()
	parseResp := &chatpb.GetCustomerBillingSummaryResponse{
		Totals: &chatpb.CustomerBillingTotals{
			PlatformFeeCents:    2900, // $29.00
			UsageCostCents:      500,  // $5.00
			ServicePremiumCents: 25,   // $0.25
			TotalCents:          3425, // $34.25
		},
	}
	parseSummary := parseBuildAccountCostSummaryFromBillingSummary(parseResp, 5.0, 29.0)
	if parseSummary.PlatformFee != 29.0 {
		parseT.Errorf("PlatformFee = %v, want 29.0", parseSummary.PlatformFee)
	}
	if parseSummary.UsageCost != 5.0 {
		parseT.Errorf("UsageCost = %v, want 5.0", parseSummary.UsageCost)
	}
	if parseSummary.PremiumCost != 0.25 {
		parseT.Errorf("PremiumCost = %v, want 0.25", parseSummary.PremiumCost)
	}
	if parseSummary.TotalCost != 34.25 {
		parseT.Errorf("TotalCost = %v, want 34.25", parseSummary.TotalCost)
	}
	if !parseSummary.HasAnyExactCosts {
		parseT.Error("HasAnyExactCosts = false, want true when costs are positive")
	}
}

// TestParseBuildAccountCostSummaryFromBillingSummaryPlanFromPlans verifies that
// the plan label is extracted from the plans list when provided.
func TestParseBuildAccountCostSummaryFromBillingSummaryPlanFromPlans(parseT *testing.T) {
	parseT.Parallel()
	parseResp := &chatpb.GetCustomerBillingSummaryResponse{
		Plans: []*chatpb.BillingPlanEntry{
			{PlanName: "Enterprise"},
		},
	}
	parseSummary := parseBuildAccountCostSummaryFromBillingSummary(parseResp, 5.0, 29.0)
	if parseSummary.PlanLabel != "Enterprise" {
		parseT.Errorf("PlanLabel = %q, want %q", parseSummary.PlanLabel, "Enterprise")
	}
}

// TestParseBuildAccountCostSummaryFromBillingSummaryPlanFromSubscription verifies that
// the plan label falls back to the subscription plan code when plans list is empty.
func TestParseBuildAccountCostSummaryFromBillingSummaryPlanFromSubscription(parseT *testing.T) {
	parseT.Parallel()
	parseResp := &chatpb.GetCustomerBillingSummaryResponse{
		Subscriptions: []*chatpb.BillingSubscriptionEntry{
			{PlanCode: "team"},
		},
	}
	parseSummary := parseBuildAccountCostSummaryFromBillingSummary(parseResp, 5.0, 29.0)
	if parseSummary.PlanLabel != "team" {
		parseT.Errorf("PlanLabel = %q, want %q", parseSummary.PlanLabel, "team")
	}
}

// TestParseBuildAccountCostSummaryFromBillingSummaryUsageEvents verifies that the
// thread count is mapped from the usage summary event count.
func TestParseBuildAccountCostSummaryFromBillingSummaryUsageEvents(parseT *testing.T) {
	parseT.Parallel()
	parseResp := &chatpb.GetCustomerBillingSummaryResponse{
		UsageSummary: &chatpb.CustomerBillingUsageSummary{
			EventCount: 42,
		},
	}
	parseSummary := parseBuildAccountCostSummaryFromBillingSummary(parseResp, 5.0, 29.0)
	if parseSummary.ThreadCount != 42 {
		parseT.Errorf("ThreadCount = %d, want 42", parseSummary.ThreadCount)
	}
}

// ─── admin_data.go — parseMarshalAdminDashboardResp ──────────────────────────

// TestParseMarshalAdminDashboardRespNilReturnsError verifies the nil-safety guard:
// the function must return an error state with HasData=false when passed nil.
func TestParseMarshalAdminDashboardRespNilReturnsError(parseT *testing.T) {
	parseT.Parallel()
	parseData := parseMarshalAdminDashboardResp(nil)
	if parseData.HasData {
		parseT.Error("HasData = true for nil response, want false")
	}
	if parseData.Error == "" {
		parseT.Error("Error is empty for nil response, want a non-empty error string")
	}
}

// TestParseMarshalAdminDashboardRespEmptyResponseHasData verifies that a non-nil but
// completely empty proto response (no sub-messages set) still sets HasData=true.
func TestParseMarshalAdminDashboardRespEmptyResponseHasData(parseT *testing.T) {
	parseT.Parallel()
	parseData := parseMarshalAdminDashboardResp(&chatpb.GetAdminDashboardResponse{})
	if !parseData.HasData {
		parseT.Error("HasData = false for empty non-nil response, want true")
	}
	if parseData.Error != "" {
		parseT.Errorf("Error = %q for empty non-nil response, want empty", parseData.Error)
	}
}

// TestParseMarshalAdminDashboardRespSummaryFields verifies that summary scalars are
// correctly mapped from the proto AdminDashboardSummary message.
func TestParseMarshalAdminDashboardRespSummaryFields(parseT *testing.T) {
	parseT.Parallel()
	parseData := parseMarshalAdminDashboardResp(&chatpb.GetAdminDashboardResponse{
		Summary: &chatpb.AdminDashboardSummary{
			TotalUsers:         120,
			TotalConversations: 880,
			TotalMessages:      4400,
			WindowNewUsers:     5,
			WindowActiveUsers:  42,
			WindowTotalCostUsd: 12.75,
			OpenIncidents:      2,
		},
	})
	if parseData.Summary.TotalUsers != 120 {
		parseT.Errorf("Summary.TotalUsers = %d, want 120", parseData.Summary.TotalUsers)
	}
	if parseData.Summary.TotalConversations != 880 {
		parseT.Errorf("Summary.TotalConversations = %d, want 880", parseData.Summary.TotalConversations)
	}
	if parseData.Summary.WindowTotalCostUSD != 12.75 {
		parseT.Errorf("Summary.WindowTotalCostUSD = %v, want 12.75", parseData.Summary.WindowTotalCostUSD)
	}
	if parseData.Summary.OpenIncidents != 2 {
		parseT.Errorf("Summary.OpenIncidents = %d, want 2", parseData.Summary.OpenIncidents)
	}
}

// TestParseMarshalAdminDashboardRespTopUsers verifies top-user rows are flattened.
func TestParseMarshalAdminDashboardRespTopUsers(parseT *testing.T) {
	parseT.Parallel()
	parseData := parseMarshalAdminDashboardResp(&chatpb.GetAdminDashboardResponse{
		TopUsers: []*chatpb.AdminUserUsage{
			{UserId: 7, Email: "top@example.com", DisplayName: "Top User", TotalCostUsd: 99.5},
		},
	})
	if len(parseData.TopUsers) != 1 {
		parseT.Fatalf("len(TopUsers) = %d, want 1", len(parseData.TopUsers))
	}
	if parseData.TopUsers[0].UserID != 7 {
		parseT.Errorf("TopUsers[0].UserID = %d, want 7", parseData.TopUsers[0].UserID)
	}
	if parseData.TopUsers[0].Email != "top@example.com" {
		parseT.Errorf("TopUsers[0].Email = %q, want top@example.com", parseData.TopUsers[0].Email)
	}
	if parseData.TopUsers[0].TotalCostUSD != 99.5 {
		parseT.Errorf("TopUsers[0].TotalCostUSD = %v, want 99.5", parseData.TopUsers[0].TotalCostUSD)
	}
}

// TestParseMarshalAdminDashboardRespProviderSnaps verifies that provider availability
// and configuration flags are correctly mapped from ProviderSnapshots.
func TestParseMarshalAdminDashboardRespProviderSnaps(parseT *testing.T) {
	parseT.Parallel()
	parseData := parseMarshalAdminDashboardResp(&chatpb.GetAdminDashboardResponse{
		ProviderSnapshots: []*chatpb.AdminProviderSnapshot{
			{ProviderId: "openai", Label: "OpenAI", Available: true, AuthConfigured: true, Status: "healthy"},
			{ProviderId: "anthropic", Label: "Anthropic", Available: false, AuthConfigured: false, Status: "degraded"},
		},
	})
	if len(parseData.ProviderSnaps) != 2 {
		parseT.Fatalf("len(ProviderSnaps) = %d, want 2", len(parseData.ProviderSnaps))
	}
	if !parseData.ProviderSnaps[0].IsAvailable {
		parseT.Error("ProviderSnaps[0].IsAvailable = false, want true")
	}
	if !parseData.ProviderSnaps[0].IsConfigured {
		parseT.Error("ProviderSnaps[0].IsConfigured = false, want true")
	}
	if parseData.ProviderSnaps[1].IsAvailable {
		parseT.Error("ProviderSnaps[1].IsAvailable = true, want false (degraded provider)")
	}
	if parseData.ProviderSnaps[1].Status != "degraded" {
		parseT.Errorf("ProviderSnaps[1].Status = %q, want degraded", parseData.ProviderSnaps[1].Status)
	}
}

// TestParseMarshalAdminDashboardRespDailyUsage verifies that daily usage rows are mapped.
func TestParseMarshalAdminDashboardRespDailyUsage(parseT *testing.T) {
	parseT.Parallel()
	parseData := parseMarshalAdminDashboardResp(&chatpb.GetAdminDashboardResponse{
		DailyUsage: []*chatpb.AdminDashboardDailyUsage{
			{UsageDay: "2026-03-01", UsageEventCount: 55, TotalCostUsd: 3.14, ActiveUsers: 8},
		},
	})
	if len(parseData.DailyUsage) != 1 {
		parseT.Fatalf("len(DailyUsage) = %d, want 1", len(parseData.DailyUsage))
	}
	if parseData.DailyUsage[0].UsageDay != "2026-03-01" {
		parseT.Errorf("DailyUsage[0].UsageDay = %q, want 2026-03-01", parseData.DailyUsage[0].UsageDay)
	}
	if parseData.DailyUsage[0].EventCount != 55 {
		parseT.Errorf("DailyUsage[0].EventCount = %d, want 55", parseData.DailyUsage[0].EventCount)
	}
}

// ─── admin_data.go — parseMarshalAdminUserDetail ──────────────────────────────

// TestParseMarshalAdminUserDetailNilReturnsEmpty verifies the nil-safety guard.
func TestParseMarshalAdminUserDetailNilReturnsEmpty(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := parseMarshalAdminUserDetail(nil)
	if parseSnapshot.HasData {
		parseT.Error("HasData = true for nil response, want false")
	}
}

// TestParseMarshalAdminUserDetailNilDetailReturnsEmpty verifies that a non-nil
// response with a nil Detail sub-message also returns an empty snapshot.
func TestParseMarshalAdminUserDetailNilDetailReturnsEmpty(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := parseMarshalAdminUserDetail(&chatpb.GetAdminUserDetailResponse{})
	if parseSnapshot.HasData {
		parseT.Error("HasData = true when Detail is nil, want false")
	}
}

// TestParseMarshalAdminUserDetailSuccess verifies that user fields and sub-tables
// are correctly mapped from a fully populated response.
func TestParseMarshalAdminUserDetailSuccess(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := parseMarshalAdminUserDetail(&chatpb.GetAdminUserDetailResponse{
		Detail: &chatpb.AdminUserDetail{
			User: &chatpb.AdminUserSummary{
				UserId:       99,
				Email:        "user@example.com",
				DisplayName:  "Alice",
				TotalCostUsd: 7.42,
			},
			RecentSessions: []*chatpb.AuthSessionEntry{
				{UserAgent: "Firefox/124", IpAddress: "10.0.0.1", LastSeenAt: "2026-03-01"},
			},
			RecentUsageEvents: []*chatpb.AdminUsageEvent{
				{ProviderId: "openai", ModelId: "gpt-4o", TotalCostUsd: 0.12, Status: "ok"},
			},
			RecentAuditLogs: []*chatpb.AuditLogEntry{
				{EventType: "login", Summary: "Logged in from Firefox", CreatedAt: "2026-03-01T00:00:00Z"},
			},
		},
	})
	if !parseSnapshot.HasData {
		parseT.Fatal("HasData = false, want true")
	}
	if parseSnapshot.UserID != 99 {
		parseT.Errorf("UserID = %d, want 99", parseSnapshot.UserID)
	}
	if parseSnapshot.Email != "user@example.com" {
		parseT.Errorf("Email = %q, want user@example.com", parseSnapshot.Email)
	}
	if parseSnapshot.TotalCostUSD != 7.42 {
		parseT.Errorf("TotalCostUSD = %v, want 7.42", parseSnapshot.TotalCostUSD)
	}
	if len(parseSnapshot.Sessions) != 1 {
		parseT.Errorf("len(Sessions) = %d, want 1", len(parseSnapshot.Sessions))
	} else if parseSnapshot.Sessions[0].IPAddress != "10.0.0.1" {
		parseT.Errorf("Sessions[0].IPAddress = %q, want 10.0.0.1", parseSnapshot.Sessions[0].IPAddress)
	}
	if len(parseSnapshot.UsageEvents) != 1 {
		parseT.Errorf("len(UsageEvents) = %d, want 1", len(parseSnapshot.UsageEvents))
	} else if parseSnapshot.UsageEvents[0].ModelID != "gpt-4o" {
		parseT.Errorf("UsageEvents[0].ModelID = %q, want gpt-4o", parseSnapshot.UsageEvents[0].ModelID)
	}
	if len(parseSnapshot.AuditLogs) != 1 {
		parseT.Errorf("len(AuditLogs) = %d, want 1", len(parseSnapshot.AuditLogs))
	} else if parseSnapshot.AuditLogs[0].EventType != "login" {
		parseT.Errorf("AuditLogs[0].EventType = %q, want login", parseSnapshot.AuditLogs[0].EventType)
	}
}

// ─── admin_data.go — parseMarshalAdminWorkspaceDetail ────────────────────────

// TestParseMarshalAdminWorkspaceDetailNilReturnsEmpty verifies the nil-safety guard.
func TestParseMarshalAdminWorkspaceDetailNilReturnsEmpty(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := parseMarshalAdminWorkspaceDetail(nil)
	if parseSnapshot.HasData {
		parseT.Error("HasData = true for nil response, want false")
	}
}

// TestParseMarshalAdminWorkspaceDetailNilDetailReturnsEmpty verifies that a non-nil
// response with nil Detail also returns an empty snapshot.
func TestParseMarshalAdminWorkspaceDetailNilDetailReturnsEmpty(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := parseMarshalAdminWorkspaceDetail(&chatpb.GetAdminWorkspaceDetailResponse{})
	if parseSnapshot.HasData {
		parseT.Error("HasData = true when Detail is nil, want false")
	}
}

// TestParseMarshalAdminWorkspaceDetailSuccess verifies that workspace fields and
// member/key/webhook sub-tables are correctly flattened from the proto response.
func TestParseMarshalAdminWorkspaceDetailSuccess(parseT *testing.T) {
	parseT.Parallel()
	parseSnapshot := parseMarshalAdminWorkspaceDetail(&chatpb.GetAdminWorkspaceDetailResponse{
		Detail: &chatpb.AdminWorkspaceDetail{
			Workspace: &chatpb.WorkspaceEntry{
				Id:       55,
				Name:     "Acme Team",
				Slug:     "acme",
				PlanCode: "team",
				Status:   "active",
			},
			Memberships: []*chatpb.WorkspaceMembershipEntry{
				{UserId: 1, RoleKey: "admin", Status: "active"},
				{UserId: 2, RoleKey: "member", Status: "active"},
			},
			ApiKeys: []*chatpb.APIKeyEntry{
				{KeyId: "k1", Label: "CI key", KeyPrefix: "sk-ci"},
			},
			WebhookEndpoints: []*chatpb.WebhookEndpointEntry{
				{Label: "stripe-hook", TargetUrl: "https://example.com/hook", IsEnabled: true},
			},
		},
	})
	if !parseSnapshot.HasData {
		parseT.Fatal("HasData = false, want true")
	}
	if parseSnapshot.WorkspaceID != 55 {
		parseT.Errorf("WorkspaceID = %d, want 55", parseSnapshot.WorkspaceID)
	}
	if parseSnapshot.Name != "Acme Team" {
		parseT.Errorf("Name = %q, want Acme Team", parseSnapshot.Name)
	}
	if parseSnapshot.PlanCode != "team" {
		parseT.Errorf("PlanCode = %q, want team", parseSnapshot.PlanCode)
	}
	if len(parseSnapshot.Members) != 2 {
		parseT.Errorf("len(Members) = %d, want 2", len(parseSnapshot.Members))
	} else if parseSnapshot.Members[0].RoleKey != "admin" {
		parseT.Errorf("Members[0].RoleKey = %q, want admin", parseSnapshot.Members[0].RoleKey)
	}
	if len(parseSnapshot.APIKeys) != 1 {
		parseT.Errorf("len(APIKeys) = %d, want 1", len(parseSnapshot.APIKeys))
	} else if parseSnapshot.APIKeys[0].KeyPrefix != "sk-ci" {
		parseT.Errorf("APIKeys[0].KeyPrefix = %q, want sk-ci", parseSnapshot.APIKeys[0].KeyPrefix)
	}
	if len(parseSnapshot.Webhooks) != 1 {
		parseT.Errorf("len(Webhooks) = %d, want 1", len(parseSnapshot.Webhooks))
	} else if !parseSnapshot.Webhooks[0].IsEnabled {
		parseT.Error("Webhooks[0].IsEnabled = false, want true")
	}
}

// ─── helpers.go — memory helpers (covers memory_editor.go and profile.go) ────

// TestIsManagedUserNameMemoryManagedKey verifies that the managed key is recognised.
func TestIsManagedUserNameMemoryManagedKey(parseT *testing.T) {
	parseT.Parallel()
	parseMemory := editableUserMemory{Key: managedUserNameMemoryKey}
	if !isManagedUserNameMemory(parseMemory) {
		parseT.Errorf("isManagedUserNameMemory({Key:%q}) = false, want true", managedUserNameMemoryKey)
	}
}

// TestIsManagedUserNameMemoryOtherKey verifies that unrelated keys return false.
func TestIsManagedUserNameMemoryOtherKey(parseT *testing.T) {
	parseT.Parallel()
	parseMemory := editableUserMemory{Key: "some.other.key"}
	if isManagedUserNameMemory(parseMemory) {
		parseT.Error("isManagedUserNameMemory({Key:other.key}) = true, want false")
	}
}

// TestIsManagedUserNameMemoryTrimmed verifies that the key check is trim-safe.
func TestIsManagedUserNameMemoryTrimmed(parseT *testing.T) {
	parseT.Parallel()
	parseMemory := editableUserMemory{Key: "  " + managedUserNameMemoryKey + "  "}
	if !isManagedUserNameMemory(parseMemory) {
		parseT.Errorf("isManagedUserNameMemory({Key:whitespace-padded}) = false, want true (trim should match)")
	}
}

// TestParseEnsureManagedUserNameMemoryEmpty verifies that an empty display name
// strips managed entries and returns remaining user memories.
func TestParseEnsureManagedUserNameMemoryEmpty(parseT *testing.T) {
	parseT.Parallel()
	parseInput := []editableUserMemory{
		{Key: managedUserNameMemoryKey, Summary: "Old Name"},
		{Key: "custom.note", Summary: "keep this"},
	}
	parseResult := parseEnsureManagedUserNameMemory("", parseInput)
	for _, parseM := range parseResult {
		if isManagedUserNameMemory(parseM) {
			parseT.Error("managed memory entry present in result when display name is empty")
		}
	}
	if len(parseResult) != 1 || parseResult[0].Key != "custom.note" {
		parseT.Errorf("unexpected result when name is empty: %+v", parseResult)
	}
}

// TestParseEnsureManagedUserNameMemoryPrependsEntry verifies that a non-empty
// display name prepends exactly one managed entry at position 0.
func TestParseEnsureManagedUserNameMemoryPrependsEntry(parseT *testing.T) {
	parseT.Parallel()
	parseInput := []editableUserMemory{
		{Key: "custom.note", Summary: "keep this"},
	}
	parseResult := parseEnsureManagedUserNameMemory("Alice", parseInput)
	if len(parseResult) < 1 {
		parseT.Fatal("result is empty, want at least one entry")
	}
	if !isManagedUserNameMemory(parseResult[0]) {
		parseT.Errorf("parseResult[0].Key = %q, want managed memory key at position 0", parseResult[0].Key)
	}
	if parseResult[0].Summary != "Alice" {
		parseT.Errorf("parseResult[0].Summary = %q, want Alice", parseResult[0].Summary)
	}
}

// TestParseEnsureManagedUserNameMemoryDeduplicates verifies that if a managed
// entry already exists, it is replaced not duplicated.
func TestParseEnsureManagedUserNameMemoryDeduplicates(parseT *testing.T) {
	parseT.Parallel()
	parseInput := []editableUserMemory{
		{Key: managedUserNameMemoryKey, Summary: "OldName"},
		{Key: "custom.note", Summary: "keep"},
	}
	parseResult := parseEnsureManagedUserNameMemory("NewName", parseInput)
	parseManagedCount := 0
	for _, parseM := range parseResult {
		if isManagedUserNameMemory(parseM) {
			parseManagedCount++
			if parseM.Summary != "NewName" {
				parseT.Errorf("managed entry Summary = %q, want NewName", parseM.Summary)
			}
		}
	}
	if parseManagedCount != 1 {
		parseT.Errorf("managed entry count = %d, want exactly 1", parseManagedCount)
	}
}

// TestParseEnsureManagedUserNameMemoryPreservesOrder verifies that custom memories
// appear after the managed entry in the returned slice, not before.
func TestParseEnsureManagedUserNameMemoryPreservesOrder(parseT *testing.T) {
	parseT.Parallel()
	parseInput := []editableUserMemory{
		{Key: "a", Summary: "first"},
		{Key: "b", Summary: "second"},
	}
	parseResult := parseEnsureManagedUserNameMemory("Bob", parseInput)
	if len(parseResult) < 3 {
		parseT.Fatalf("len(result) = %d, want 3", len(parseResult))
	}
	if !isManagedUserNameMemory(parseResult[0]) {
		parseT.Error("managed entry is not at position 0")
	}
	if parseResult[1].Key != "a" || parseResult[2].Key != "b" {
		parseT.Errorf("custom memory order not preserved: [1]=%q [2]=%q", parseResult[1].Key, parseResult[2].Key)
	}
}
