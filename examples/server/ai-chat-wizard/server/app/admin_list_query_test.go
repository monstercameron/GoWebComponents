package app

import (
	"strconv"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
)

// TestAdminListQueryUsersUsageConversations verifies typed list-query search/filter/sort/pagination behavior for user, usage, and conversation admin RPCs.
func TestAdminListQueryUsersUsageConversations(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-list-query-users", parseAliceAuth.ID, parseAliceAuth.Email)

	parseUsersResp, parseErr := parseServer.ListAdminUsers(parseAliceCtx, &chatpb.ListAdminUsersRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Offset:        0,
			Search:        "alice",
			SortBy:        "email",
			SortDirection: "asc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsers list query: %v", parseErr)
	}
	if len(parseUsersResp.GetUsers()) != 1 || parseUsersResp.GetUsers()[0].GetEmail() != "alice@example.com" {
		parseT.Fatalf("unexpected users list-query response: %+v", parseUsersResp.GetUsers())
	}

	parseUsageResp, parseErr := parseServer.ListAdminUsageEvents(parseAliceCtx, &chatpb.ListAdminUsageEventsRequest{
		LookbackDays: 30,
		Status:       "completed",
		ProviderId:   "fake",
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Search:        "bob",
			SortBy:        "total_cost_usd",
			SortDirection: "desc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminUsageEvents list query: %v", parseErr)
	}
	if len(parseUsageResp.GetEvents()) != 1 {
		parseT.Fatalf("expected one filtered usage row, got %+v", parseUsageResp.GetEvents())
	}
	parseUsageRow := parseUsageResp.GetEvents()[0]
	if parseUsageRow.GetProviderId() != "fake" || parseUsageRow.GetStatus() != "completed" {
		parseT.Fatalf("unexpected usage row after list query filtering: %+v", parseUsageRow)
	}

	parseConversationsResp, parseErr := parseServer.ListAdminConversations(parseAliceCtx, &chatpb.ListAdminConversationsRequest{
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Search:        "refund",
			SortBy:        "conversation_id",
			SortDirection: "asc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminConversations list query: %v", parseErr)
	}
	if len(parseConversationsResp.GetConversations()) != 1 {
		parseT.Fatalf("expected one filtered conversation row, got %+v", parseConversationsResp.GetConversations())
	}
	if parseConversationsResp.GetConversations()[0].GetEmail() != "bob@example.com" {
		parseT.Fatalf("expected bob conversation row after search filter, got %+v", parseConversationsResp.GetConversations()[0])
	}
}

// TestAdminListQuerySupportBilling verifies typed list-query behavior for support and billing admin list RPCs.
func TestAdminListQuerySupportBilling(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-list-query-support-billing", parseAliceAuth.ID, parseAliceAuth.Email)

	parseSupportResp, parseErr := parseServer.ListAdminSupportTickets(parseAliceCtx, &chatpb.ListAdminSupportTicketsRequest{
		Status: "open",
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Search:        "billing",
			SortBy:        "ticket_key",
			SortDirection: "asc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminSupportTickets list query: %v", parseErr)
	}
	if len(parseSupportResp.GetTickets()) != 1 || parseSupportResp.GetTickets()[0].GetTicketKey() == "" {
		parseT.Fatalf("expected one support ticket row from list query, got %+v", parseSupportResp.GetTickets())
	}

	if _, parseErr = parseServer.SetAdminBillingAccessOverride(parseAliceCtx, &chatpb.AdminBillingAccessOverrideMutationRequest{
		UserId:        parseAliceAuth.ID,
		OverrideKey:   "access.priority_support",
		OverrideValue: "enabled",
		Reason:        "operator setup",
		IsEnabled:     true,
		Confirm:       true,
	}); parseErr != nil {
		parseT.Fatalf("SetAdminBillingAccessOverride setup: %v", parseErr)
	}
	if _, parseErr = parseServer.SetAdminBillingQuotaOverride(parseAliceCtx, &chatpb.AdminBillingQuotaOverrideMutationRequest{
		UserId:     parseAliceAuth.ID,
		QuotaKey:   "requests.daily",
		QuotaValue: "100000",
		Reason:     "operator setup",
		IsEnabled:  true,
		Confirm:    true,
	}); parseErr != nil {
		parseT.Fatalf("SetAdminBillingQuotaOverride setup: %v", parseErr)
	}

	parseBillingEventsResp, parseErr := parseServer.ListAdminBillingEvents(parseAliceCtx, &chatpb.ListAdminBillingEventsRequest{
		UserId:    parseAliceAuth.ID,
		EventType: "invoice.payment_failed",
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Search:        "payment failed",
			SortBy:        "created_at",
			SortDirection: "desc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingEvents list query: %v", parseErr)
	}
	if len(parseBillingEventsResp.GetEvents()) != 1 || parseBillingEventsResp.GetEvents()[0].GetEventType() != "invoice.payment_failed" {
		parseT.Fatalf("unexpected billing events list-query response: %+v", parseBillingEventsResp.GetEvents())
	}

	parseDunningResp, parseErr := parseServer.ListAdminBillingDunningEvents(parseAliceCtx, &chatpb.ListAdminBillingDunningEventsRequest{
		UserId: parseAliceAuth.ID,
		Status: "pending",
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			SortBy:        "updated_at",
			SortDirection: "desc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingDunningEvents list query: %v", parseErr)
	}
	if len(parseDunningResp.GetEvents()) != 1 || parseDunningResp.GetEvents()[0].GetStatus() == "" {
		parseT.Fatalf("unexpected billing dunning list-query response: %+v", parseDunningResp.GetEvents())
	}

	parseOverrideResp, parseErr := parseServer.ListAdminBillingAccessOverrides(parseAliceCtx, &chatpb.ListAdminBillingAccessOverridesRequest{
		UserId:      parseAliceAuth.ID,
		OverrideKey: "access.priority_support",
		ListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			SortBy:        "updated_at",
			SortDirection: "desc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminBillingAccessOverrides list query: %v", parseErr)
	}
	if len(parseOverrideResp.GetOverrides()) != 1 || parseOverrideResp.GetOverrides()[0].GetOverrideKey() != "access.priority_support" {
		parseT.Fatalf("unexpected billing override list-query response: %+v", parseOverrideResp.GetOverrides())
	}
}

// TestSuperuserSlicesListQuery verifies typed workspace/support/incident list-query filtering and pagination for superuser slices.
func TestSuperuserSlicesListQuery(parseT *testing.T) {
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
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)

	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-superuser-list-query")
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-bob-superuser-list-query",
		WorkspaceID:    parseBobWorkspaceID,
		UserID:         parseBobAuth.ID,
		Status:         "open",
		Priority:       "normal",
		Subject:        "Scoped support query ticket",
		Body:           "Support triage row for superuser list query filtering.",
		AssigneeUserID: parseBobAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket bob list-query: %v", parseErr)
	}

	parseAliceCtx := parseBindAuthUser(parseServer, "peer-superuser-list-query", parseAliceAuth.ID, parseAliceAuth.Email)
	parseSlicesResp, parseErr := parseServer.GetSuperuserSlices(parseAliceCtx, &chatpb.GetSuperuserSlicesRequest{
		Limit:           25,
		LookbackDays:    30,
		WorkspaceStatus: "active",
		SupportStatus:   "open",
		IncidentStatus:  "open",
		WorkspaceListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Search:        "ws-bob-superuser-list-query",
			SortBy:        "workspace_key",
			SortDirection: "asc",
		},
		SupportListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			Search:        "superuser-list-query",
			SortBy:        "ticket_key",
			SortDirection: "asc",
		},
		IncidentListQuery: &chatpb.AdminListQuery{
			Limit:         1,
			SortBy:        "updated_at",
			SortDirection: "desc",
		},
	})
	if parseErr != nil {
		parseT.Fatalf("GetSuperuserSlices list query: %v", parseErr)
	}
	if len(parseSlicesResp.GetWorkspaces()) != 1 || parseSlicesResp.GetWorkspaces()[0].GetWorkspaceKey() != "ws-bob-superuser-list-query" {
		parseT.Fatalf("unexpected workspace list-query response: %+v", parseSlicesResp.GetWorkspaces())
	}
	if len(parseSlicesResp.GetSupportTickets()) != 1 || parseSlicesResp.GetSupportTickets()[0].GetTicketKey() != "ticket-bob-superuser-list-query" {
		parseT.Fatalf("unexpected support list-query response: %+v", parseSlicesResp.GetSupportTickets())
	}
	if len(parseSlicesResp.GetIncidents()) != 1 || parseSlicesResp.GetIncidents()[0].GetStatus() == "" {
		parseT.Fatalf("unexpected incident list-query response: %+v", parseSlicesResp.GetIncidents())
	}
}

// BenchmarkParseFilterAdminUserRowsBySearch reports micro-benchmark throughput for admin user list search filtering.
func BenchmarkParseFilterAdminUserRowsBySearch(parseB *testing.B) {
	parseRows := make([]parseAdminUserRow, 0, 200)
	for parseIndex := range int64(200) {
		parseRows = append(parseRows, parseAdminUserRow{
			UserID:       parseIndex + 1,
			Email:        "user" + strconv.FormatInt(parseIndex, 10) + "@example.com",
			DisplayName:  "User " + strconv.FormatInt(parseIndex, 10),
			TotalCostUSD: float64(parseIndex) / 10,
		})
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseFilterAdminUserRowsBySearch(parseRows, "user19")
	}
}
