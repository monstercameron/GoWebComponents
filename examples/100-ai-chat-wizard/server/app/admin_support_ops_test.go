package app

import (
	"strconv"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestStoreAdminSupportTriageFuncs verifies typed store helpers for support queue/detail, linked action history, assignment, and escalation.
func TestStoreAdminSupportTriageFuncs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}

	parseTicketRows, parseErr := parseStore.parseListAdminSupportTickets(25)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminSupportTickets: %v", parseErr)
	}
	parseTicketID := parseFindAdminSupportTicketIDByKey(parseTicketRows, "ticket-admin-dashboard-open")
	if parseTicketID <= 0 {
		parseT.Fatalf("expected seeded support ticket id, rows=%+v", parseTicketRows)
	}

	parseTicketRow, hasParseTicket, parseErr := parseStore.parseGetAdminSupportTicketByID(parseTicketID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAdminSupportTicketByID: %v", parseErr)
	}
	if !hasParseTicket {
		parseT.Fatalf("expected support ticket %d", parseTicketID)
	}

	parseMessageID, parseErr := parseStore.parseCreateSupportTicketMessage(parseSupportTicketMessageWrite{
		TicketID:     parseTicketID,
		AuthorUserID: parseAliceAuth.ID,
		MessageType:  "internal_note",
		Body:         "Investigating payment failure details.",
		IsInternal:   true,
	})
	if parseErr != nil {
		parseT.Fatalf("parseCreateSupportTicketMessage: %v", parseErr)
	}
	parseMessageRows, parseErr := parseStore.parseListAdminSupportTicketMessagesByTicketID(parseTicketID, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminSupportTicketMessagesByTicketID: %v", parseErr)
	}
	if !parseHasAdminSupportTicketMessageID(parseMessageRows, parseMessageID) {
		parseT.Fatalf("expected support message id=%d in rows=%+v", parseMessageID, parseMessageRows)
	}

	if _, parseErr = parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseBobAuth.ID,
		WorkspaceID: parseTicketRow.WorkspaceID,
		EventType:   "admin.support.ticket.assigned",
		TargetType:  "support_ticket",
		TargetID:    strconv.FormatInt(parseTicketID, 10),
		Summary:     "Ticket assignment audited",
		PayloadJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog support_ticket target: %v", parseErr)
	}
	parseActionRows, parseErr := parseStore.parseListAdminSupportAccountActionHistoryByTicketID(parseTicketID, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminSupportAccountActionHistoryByTicketID: %v", parseErr)
	}
	if !parseHasAdminAuditTarget(parseActionRows, "support_ticket", strconv.FormatInt(parseTicketID, 10)) {
		parseT.Fatalf("expected support_ticket audit target for ticket %d, rows=%+v", parseTicketID, parseActionRows)
	}
	if _, parseErr = parseStore.parseCreateAuditLog(parseAuditLogWrite{
		ActorUserID: parseBobAuth.ID,
		WorkspaceID: parseTicketRow.WorkspaceID,
		EventType:   "admin.support.account.reviewed",
		TargetType:  "user",
		TargetID:    strconv.FormatInt(parseAliceAuth.ID, 10),
		Summary:     "Account history reviewed",
		PayloadJSON: "{}",
	}); parseErr != nil {
		parseT.Fatalf("parseCreateAuditLog user target: %v", parseErr)
	}
	parseUserActionRows, parseErr := parseStore.parseListAdminSupportAccountActionHistoryByUser(parseAliceAuth.ID, 25)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminSupportAccountActionHistoryByUser: %v", parseErr)
	}
	if !parseHasAdminAuditTarget(parseUserActionRows, "user", strconv.FormatInt(parseAliceAuth.ID, 10)) {
		parseT.Fatalf("expected user-target audit action row for user %d, rows=%+v", parseAliceAuth.ID, parseUserActionRows)
	}

	parseAssignedRow, parseErr := parseStore.parseStoreAdminSupportTicketAssignment(parseTicketID, parseBobAuth.ID)
	if parseErr != nil {
		parseT.Fatalf("parseStoreAdminSupportTicketAssignment: %v", parseErr)
	}
	if parseAssignedRow.AssigneeUserID != parseBobAuth.ID {
		parseT.Fatalf("expected assignee user id %d, got %+v", parseBobAuth.ID, parseAssignedRow)
	}

	parseEscalatedRow, parseErr := parseStore.parseStoreAdminSupportTicketEscalation(parseTicketID, "urgent", "escalated", "Escalated to billing operations.")
	if parseErr != nil {
		parseT.Fatalf("parseStoreAdminSupportTicketEscalation: %v", parseErr)
	}
	if parseEscalatedRow.Status != "escalated" || parseEscalatedRow.Priority != "urgent" {
		parseT.Fatalf("expected escalated urgent support ticket, got %+v", parseEscalatedRow)
	}
}

// TestStoreAdminSupportAccountActionHistoryByUser verifies user-linked account action history delegates to audit-log listing by user.
func TestStoreAdminSupportAccountActionHistoryByUser(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)

	parseBobAuth, parseErr := parseStore.getUserAuthByEmail("bob@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail bob: %v", parseErr)
	}
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-support-history")

	parseActionRows, parseErr := parseStore.parseListAdminSupportAccountActionHistoryByUser(parseBobAuth.ID, 1)
	if parseErr != nil {
		parseT.Fatalf("parseListAdminSupportAccountActionHistoryByUser: %v", parseErr)
	}
	if len(parseActionRows) != 1 || parseActionRows[0].TargetType != "user" || parseActionRows[0].TargetID != strconv.FormatInt(parseBobAuth.ID, 10) {
		parseT.Fatalf("unexpected account action history rows: %+v", parseActionRows)
	}
}

// TestAdminSupportTriageRPCs verifies superuser support-triage RPC queue, detail, note, assignment, and escalation behavior.
func TestAdminSupportTriageRPCs(parseT *testing.T) {
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
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-support-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseQueueResp, parseErr := parseServer.ListAdminSupportTickets(parseAliceCtx, &chatpb.ListAdminSupportTicketsRequest{
		Limit:          25,
		Status:         "open",
		Priority:       "high",
		AssigneeUserId: parseAliceAuth.ID,
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminSupportTickets: %v", parseErr)
	}
	parseTicketID := parseFindAdminSupportTicketProtoIDByKey(parseQueueResp.GetTickets(), "ticket-admin-dashboard-open")
	if parseTicketID <= 0 {
		parseT.Fatalf("expected seeded support ticket in queue response, rows=%+v", parseQueueResp.GetTickets())
	}

	parseDetailResp, parseErr := parseServer.GetAdminSupportTicketDetail(parseAliceCtx, &chatpb.GetAdminSupportTicketDetailRequest{
		TicketId: parseTicketID,
		Limit:    25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminSupportTicketDetail: %v", parseErr)
	}
	if parseDetailResp.GetDetail().GetTicket().GetId() != parseTicketID {
		parseT.Fatalf("expected detail ticket id %d, got %+v", parseTicketID, parseDetailResp.GetDetail())
	}

	parseNoteResp, parseErr := parseServer.AddAdminSupportInternalNote(parseAliceCtx, &chatpb.AddAdminSupportInternalNoteRequest{
		TicketId: parseTicketID,
		Body:     "Checking invoice reconciliation now.",
		Confirm:  true,
		Reason:   "Operator internal triage note",
	})
	if parseErr != nil {
		parseT.Fatalf("AddAdminSupportInternalNote: %v", parseErr)
	}
	if parseNoteResp.GetStatus() != "created" || parseNoteResp.GetMessageId() <= 0 {
		parseT.Fatalf("unexpected note mutation response: %+v", parseNoteResp)
	}

	parseAssignResp, parseErr := parseServer.AssignAdminSupportTicket(parseAliceCtx, &chatpb.AssignAdminSupportTicketRequest{
		TicketId:       parseTicketID,
		AssigneeUserId: parseBobAuth.ID,
		Confirm:        true,
		Reason:         "Routing to billing specialist",
	})
	if parseErr != nil {
		parseT.Fatalf("AssignAdminSupportTicket: %v", parseErr)
	}
	if parseAssignResp.GetAssigneeUserId() != parseBobAuth.ID {
		parseT.Fatalf("expected assigned user %d, got %+v", parseBobAuth.ID, parseAssignResp)
	}

	parseEscalateResp, parseErr := parseServer.EscalateAdminSupportTicket(parseAliceCtx, &chatpb.EscalateAdminSupportTicketRequest{
		TicketId:       parseTicketID,
		Priority:       "urgent",
		Status:         "escalated",
		ResolutionNote: "Escalated to L2 support and billing ops.",
		Confirm:        true,
		Reason:         "Customer blocked on payment flow",
	})
	if parseErr != nil {
		parseT.Fatalf("EscalateAdminSupportTicket: %v", parseErr)
	}
	if parseEscalateResp.GetStatus() != "escalated" || parseEscalateResp.GetPriority() != "urgent" {
		parseT.Fatalf("unexpected escalation response: %+v", parseEscalateResp)
	}

	parseDetailAfterResp, parseErr := parseServer.GetAdminSupportTicketDetail(parseAliceCtx, &chatpb.GetAdminSupportTicketDetailRequest{
		TicketId: parseTicketID,
		Limit:    25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminSupportTicketDetail after mutations: %v", parseErr)
	}
	if len(parseDetailAfterResp.GetDetail().GetMessages()) == 0 {
		parseT.Fatalf("expected support messages after note mutation, detail=%+v", parseDetailAfterResp.GetDetail())
	}
	if len(parseDetailAfterResp.GetDetail().GetAccountActionHistory()) == 0 {
		parseT.Fatalf("expected account action history after triage mutations, detail=%+v", parseDetailAfterResp.GetDetail())
	}
}

// TestAdminSupportTriageListQueryRPCs verifies typed support queue search, sort, and pagination behavior.
func TestAdminSupportTriageListQueryRPCs(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseAliceCtx := parseBindAuthUser(parseServer, "peer-admin-support-list-query-alice", parseAliceAuth.ID, parseAliceAuth.Email)

	parseSeedTicketRows, parseErr := parseStore.parseListSupportTickets(25)
	if parseErr != nil {
		parseT.Fatalf("parseListSupportTickets seed lookup: %v", parseErr)
	}
	parseSeedWorkspaceID := int64(0)
	for _, parseSeedTicketRow := range parseSeedTicketRows {
		if parseSeedTicketRow.TicketKey != "ticket-admin-dashboard-open" {
			continue
		}
		parseSeedWorkspaceID = parseSeedTicketRow.WorkspaceID
		break
	}
	if parseSeedWorkspaceID <= 0 {
		parseT.Fatalf("expected seeded workspace id from support ticket rows=%+v", parseSeedTicketRows)
	}
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-admin-dashboard-open-2",
		WorkspaceID:    parseSeedWorkspaceID,
		UserID:         parseAliceAuth.ID,
		Status:         "open",
		Priority:       "high",
		Subject:        "Follow-up billing interruption",
		Body:           "Second open support item for list-query coverage.",
		AssigneeUserID: parseAliceAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket list-query seed: %v", parseErr)
	}

	parseQueueResp, parseErr := parseServer.ListAdminSupportTickets(parseAliceCtx, &chatpb.ListAdminSupportTicketsRequest{
		Status:   "open",
		Priority: "high",
		ListQuery: &chatpb.AdminListQuery{
			Search:        "ticket-admin-dashboard-open",
			SortBy:        "ticket_key",
			SortDirection: "asc",
			Limit:         1,
			Offset:        1,
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminSupportTickets list query: %v", parseErr)
	}
	if len(parseQueueResp.GetTickets()) != 1 || parseQueueResp.GetTickets()[0].GetTicketKey() != "ticket-admin-dashboard-open-2" {
		parseT.Fatalf("expected second sorted support ticket row, got %+v", parseQueueResp.GetTickets())
	}
}

// TestAdminSupportTriageRPCsWorkspaceAdminScope verifies workspace-admin support triage is scoped and denies out-of-scope ticket actions.
func TestAdminSupportTriageRPCsWorkspaceAdminScope(parseT *testing.T) {
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
	parseAliceTicketRows, parseErr := parseStore.parseListSupportTickets(25)
	if parseErr != nil {
		parseT.Fatalf("parseListSupportTickets: %v", parseErr)
	}
	parseAliceTicketID := parseFindAdminSupportTicketIDByKey(parseAliceTicketRows, "ticket-admin-dashboard-open")
	if parseAliceTicketID <= 0 {
		parseT.Fatalf("expected alice seeded support ticket id, rows=%+v", parseAliceTicketRows)
	}
	parseBobWorkspaceID := parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-admin-support-scope")
	if parseErr = parseStore.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      "ticket-bob-scope-only",
		WorkspaceID:    parseBobWorkspaceID,
		UserID:         parseBobAuth.ID,
		Status:         "open",
		Priority:       "normal",
		Subject:        "Workspace scoped support issue",
		Body:           "Bob workspace needs a scoped support check.",
		AssigneeUserID: parseBobAuth.ID,
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertSupportTicket bob scoped: %v", parseErr)
	}
	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-support-bob", parseBobAuth.ID, parseBobAuth.Email)

	parseQueueResp, parseErr := parseServer.ListAdminSupportTickets(parseBobCtx, &chatpb.ListAdminSupportTicketsRequest{
		Limit: 25,
	})
	if parseErr != nil {
		parseT.Fatalf("ListAdminSupportTickets workspace-admin: %v", parseErr)
	}
	if parseHasAdminSupportTicketProtoID(parseQueueResp.GetTickets(), parseAliceTicketID) {
		parseT.Fatalf("workspace-admin queue leaked out-of-scope ticket id=%d rows=%+v", parseAliceTicketID, parseQueueResp.GetTickets())
	}
	if parseFindAdminSupportTicketProtoIDByKey(parseQueueResp.GetTickets(), "ticket-bob-scope-only") <= 0 {
		parseT.Fatalf("expected in-scope bob support ticket in queue, rows=%+v", parseQueueResp.GetTickets())
	}

	if _, parseErr = parseServer.GetAdminSupportTicketDetail(parseBobCtx, &chatpb.GetAdminSupportTicketDetailRequest{
		TicketId: parseAliceTicketID,
		Limit:    25,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("GetAdminSupportTicketDetail out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.AddAdminSupportInternalNote(parseBobCtx, &chatpb.AddAdminSupportInternalNoteRequest{
		TicketId: parseAliceTicketID,
		Body:     "out of scope mutation",
		Confirm:  true,
		Reason:   "scope test",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("AddAdminSupportInternalNote out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.AssignAdminSupportTicket(parseBobCtx, &chatpb.AssignAdminSupportTicketRequest{
		TicketId:       parseAliceTicketID,
		AssigneeUserId: parseBobAuth.ID,
		Confirm:        true,
		Reason:         "scope test",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("AssignAdminSupportTicket out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.EscalateAdminSupportTicket(parseBobCtx, &chatpb.EscalateAdminSupportTicketRequest{
		TicketId:       parseAliceTicketID,
		Priority:       "urgent",
		Status:         "escalated",
		ResolutionNote: "scope test",
		Confirm:        true,
		Reason:         "scope test",
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("EscalateAdminSupportTicket out-of-scope status code = %v, want %v", status.Code(parseErr), codes.PermissionDenied)
	}
	_ = parseAliceAuth
}

// TestParseLimitSupportTicketRows verifies support-ticket row limiting respects non-positive and positive limits.
func TestParseLimitSupportTicketRows(parseT *testing.T) {
	parseRows := []parseSupportTicketRow{
		{ID: 1, TicketKey: "ticket-1"},
		{ID: 2, TicketKey: "ticket-2"},
		{ID: 3, TicketKey: "ticket-3"},
	}
	parseUnlimitedRows := parseLimitSupportTicketRows(parseRows, 0)
	if len(parseUnlimitedRows) != 3 {
		parseT.Fatalf("expected non-positive limit to keep all rows, got %d", len(parseUnlimitedRows))
	}
	parseLimitedRows := parseLimitSupportTicketRows(parseRows, 2)
	if len(parseLimitedRows) != 2 || parseLimitedRows[0].ID != 1 || parseLimitedRows[1].ID != 2 {
		parseT.Fatalf("expected deterministic two-row limit result, got %+v", parseLimitedRows)
	}
	parseLargeLimitRows := parseLimitSupportTicketRows(parseRows, 10)
	if len(parseLargeLimitRows) != 3 {
		parseT.Fatalf("expected large limit to keep all rows, got %d", len(parseLargeLimitRows))
	}
}

// TestParseLimitSupportTicketMessageRows verifies support-ticket message limiting respects non-positive and positive limits.
func TestParseLimitSupportTicketMessageRows(parseT *testing.T) {
	parseRows := []parseSupportTicketMessageRow{
		{ID: 1, TicketID: 10, Body: "message-1"},
		{ID: 2, TicketID: 10, Body: "message-2"},
		{ID: 3, TicketID: 10, Body: "message-3"},
	}
	parseUnlimitedRows := parseLimitSupportTicketMessageRows(parseRows, 0)
	if len(parseUnlimitedRows) != 3 {
		parseT.Fatalf("expected non-positive limit to keep all message rows, got %d", len(parseUnlimitedRows))
	}
	parseLimitedRows := parseLimitSupportTicketMessageRows(parseRows, 2)
	if len(parseLimitedRows) != 2 || parseLimitedRows[0].ID != 1 || parseLimitedRows[1].ID != 2 {
		parseT.Fatalf("expected deterministic two-message limit result, got %+v", parseLimitedRows)
	}
	parseLargeLimitRows := parseLimitSupportTicketMessageRows(parseRows, 10)
	if len(parseLargeLimitRows) != 3 {
		parseT.Fatalf("expected large limit to keep all message rows, got %d", len(parseLargeLimitRows))
	}
}

// parseFindAdminSupportTicketIDByKey returns one support-ticket id for one ticket key from store rows.
func parseFindAdminSupportTicketIDByKey(parseRows []parseSupportTicketRow, parseTicketKey string) int64 {
	for _, parseRow := range parseRows {
		if parseRow.TicketKey == parseTicketKey {
			return parseRow.ID
		}
	}
	return 0
}

// parseFindAdminSupportTicketProtoIDByKey returns one support-ticket id for one ticket key from protobuf rows.
func parseFindAdminSupportTicketProtoIDByKey(parseRows []*chatpb.SupportTicketEntry, parseTicketKey string) int64 {
	for _, parseRow := range parseRows {
		if parseRow.GetTicketKey() == parseTicketKey {
			return parseRow.GetId()
		}
	}
	return 0
}

// parseHasAdminSupportTicketProtoID reports whether one protobuf support-ticket slice contains one ticket id.
func parseHasAdminSupportTicketProtoID(parseRows []*chatpb.SupportTicketEntry, parseTicketID int64) bool {
	for _, parseRow := range parseRows {
		if parseRow.GetId() == parseTicketID {
			return true
		}
	}
	return false
}

// parseHasAdminSupportTicketMessageID reports whether one support-ticket message row slice contains one message id.
func parseHasAdminSupportTicketMessageID(parseRows []parseSupportTicketMessageRow, parseMessageID int64) bool {
	for _, parseRow := range parseRows {
		if parseRow.ID == parseMessageID {
			return true
		}
	}
	return false
}

// parseHasAdminAuditTarget reports whether one audit row slice contains one target pair.
func parseHasAdminAuditTarget(parseRows []parseAuditLogRow, parseTargetType, parseTargetID string) bool {
	for _, parseRow := range parseRows {
		if parseRow.TargetType != parseTargetType || parseRow.TargetID != parseTargetID {
			continue
		}
		return true
	}
	return false
}

// BenchmarkParseBuildAdminSupportTicketEntry reports micro-benchmark throughput for support-ticket protobuf mapping.
func BenchmarkParseBuildAdminSupportTicketEntry(parseB *testing.B) {
	parseRow := parseSupportTicketRow{
		ID:             1,
		TicketKey:      "ticket-bench",
		WorkspaceID:    10,
		UserID:         20,
		Status:         "open",
		Priority:       "high",
		Subject:        "Subject",
		Body:           "Body",
		AssigneeUserID: 30,
		ResolutionNote: "",
		CreatedAt:      "2026-01-01T00:00:00Z",
		UpdatedAt:      "2026-01-01T00:00:00Z",
	}
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		_ = parseBuildAdminSupportTicketEntry(parseRow)
	}
}
