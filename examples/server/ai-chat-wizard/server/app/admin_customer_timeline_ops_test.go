package app

import (
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseSeedAdminCustomerTimelineSupportMessage seeds one support-ticket message for one target user when a ticket exists.
func parseSeedAdminCustomerTimelineSupportMessage(parseT *testing.T, parseStore *Store, parseUserID int64) {
	parseT.Helper()
	parseTicketRows, parseErr := parseStore.parseListSupportTickets(parseAdminScopedScanLimit)
	if parseErr != nil {
		parseT.Fatalf("parseListSupportTickets: %v", parseErr)
	}
	parseTicketID := int64(0)
	for _, parseTicketRow := range parseTicketRows {
		if parseTicketRow.UserID != parseUserID {
			continue
		}
		parseTicketID = parseTicketRow.ID
		break
	}
	if parseTicketID <= 0 {
		return
	}
	if _, parseErr = parseStore.parseCreateSupportTicketMessage(parseSupportTicketMessageWrite{
		TicketID:     parseTicketID,
		AuthorUserID: parseUserID,
		MessageType:  "internal_note",
		Body:         "Customer requested billing assistance.",
		IsInternal:   true,
	}); parseErr != nil {
		parseT.Fatalf("parseCreateSupportTicketMessage: %v", parseErr)
	}
}

// parseHasAdminCustomerTimelineSource reports whether one timeline response contains one expected source key.
func parseHasAdminCustomerTimelineSource(parseEvents []*chatpb.AdminCustomerTimelineEvent, parseSource string) bool {
	for _, parseEvent := range parseEvents {
		if strings.EqualFold(strings.TrimSpace(parseEvent.GetSource()), strings.TrimSpace(parseSource)) {
			return true
		}
	}
	return false
}

// TestGetAdminCustomerAccountTimeline verifies merged timeline events include chat, support, billing, auth-session, and audit sources with source-specific linkage ids.
func TestGetAdminCustomerAccountTimeline(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseAliceAuth, parseErr := parseStore.getUserAuthByEmail("alice@example.com")
	if parseErr != nil {
		parseT.Fatalf("getUserAuthByEmail alice: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseAliceAuth.ID)
	parseSeedAdminUserControlSignals(parseT, parseStore, parseAliceAuth.ID, "ws-alice-customer-timeline")
	parseSeedAdminCustomerTimelineSupportMessage(parseT, parseStore, parseAliceAuth.ID)

	parseCtx := parseBindAuthUser(parseServer, "peer-admin-customer-timeline-alice", parseAliceAuth.ID, parseAliceAuth.Email)
	parseResp, parseErr := parseServer.GetAdminCustomerAccountTimeline(parseCtx, &chatpb.GetAdminCustomerAccountTimelineRequest{
		UserId:       parseAliceAuth.ID,
		LookbackDays: 30,
		Limit:        50,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminCustomerAccountTimeline: %v", parseErr)
	}
	if parseResp.GetUser() == nil || parseResp.GetUser().GetUserId() != parseAliceAuth.ID {
		parseT.Fatalf("expected alice user summary in timeline response, got %+v", parseResp.GetUser())
	}
	if len(parseResp.GetEvents()) == 0 {
		parseT.Fatalf("expected non-empty timeline events")
	}
	parseExpectedSources := []string{
		string(parseAdminCustomerTimelineSourceChatContent),
		string(parseAdminCustomerTimelineSourceSupport),
		string(parseAdminCustomerTimelineSourceBilling),
		string(parseAdminCustomerTimelineSourceAuthSession),
		string(parseAdminCustomerTimelineSourceAuditEvent),
	}
	for _, parseExpectedSource := range parseExpectedSources {
		if !parseHasAdminCustomerTimelineSource(parseResp.GetEvents(), parseExpectedSource) {
			parseT.Fatalf("expected timeline source %q in events", parseExpectedSource)
		}
	}
	for _, parseEvent := range parseResp.GetEvents() {
		if strings.TrimSpace(parseEvent.GetTimelineId()) == "" {
			parseT.Fatalf("expected non-empty timeline id for event %+v", parseEvent)
		}
		switch strings.TrimSpace(parseEvent.GetSource()) {
		case string(parseAdminCustomerTimelineSourceChatContent):
			if parseEvent.GetConversationId() <= 0 {
				parseT.Fatalf("expected chat timeline event conversation id, got %+v", parseEvent)
			}
		case string(parseAdminCustomerTimelineSourceSupport):
			if parseEvent.GetSupportTicketId() <= 0 {
				parseT.Fatalf("expected support timeline event ticket id, got %+v", parseEvent)
			}
		case string(parseAdminCustomerTimelineSourceBilling):
			if parseEvent.GetBillingCustomerId() <= 0 {
				parseT.Fatalf("expected billing timeline event customer id, got %+v", parseEvent)
			}
		case string(parseAdminCustomerTimelineSourceAuthSession):
			if parseEvent.GetAuthSessionId() <= 0 {
				parseT.Fatalf("expected auth-session timeline event session id, got %+v", parseEvent)
			}
		case string(parseAdminCustomerTimelineSourceAuditEvent):
			if parseEvent.GetAuditLogId() <= 0 {
				parseT.Fatalf("expected audit timeline event audit id, got %+v", parseEvent)
			}
		}
	}
}

// TestGetAdminCustomerAccountTimelineScopeGuards verifies workspace-admin redaction behavior and out-of-scope denial for the unified customer timeline RPC.
func TestGetAdminCustomerAccountTimelineScopeGuards(parseT *testing.T) {
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
	parseSeedAdminUserControlSignals(parseT, parseStore, parseBobAuth.ID, "ws-bob-customer-timeline")

	parseBobCtx := parseBindAuthUser(parseServer, "peer-admin-customer-timeline-bob", parseBobAuth.ID, parseBobAuth.Email)
	parseInScopeResp, parseErr := parseServer.GetAdminCustomerAccountTimeline(parseBobCtx, &chatpb.GetAdminCustomerAccountTimelineRequest{
		UserId:       parseBobAuth.ID,
		LookbackDays: 30,
		Limit:        25,
	})
	if parseErr != nil {
		parseT.Fatalf("GetAdminCustomerAccountTimeline in-scope bob: %v", parseErr)
	}
	if parseInScopeResp.GetUser() == nil || parseInScopeResp.GetUser().GetUserId() != parseBobAuth.ID {
		parseT.Fatalf("expected bob timeline user summary, got %+v", parseInScopeResp.GetUser())
	}
	if len(parseInScopeResp.GetEvents()) == 0 {
		parseT.Fatalf("expected non-empty in-scope timeline events for bob")
	}
	for _, parseEvent := range parseInScopeResp.GetEvents() {
		if strings.TrimSpace(parseEvent.GetDetailJson()) != "{}" {
			parseT.Fatalf("expected redacted detail_json for workspace-admin timeline event, got %+v", parseEvent)
		}
		if strings.EqualFold(strings.TrimSpace(parseEvent.GetSource()), string(parseAdminCustomerTimelineSourceChatContent)) &&
			strings.TrimSpace(parseEvent.GetSummary()) != parseWorkspaceScopeRedactionText {
			parseT.Fatalf("expected redacted chat summary for workspace-admin timeline event, got %+v", parseEvent)
		}
	}

	if _, parseErr = parseServer.GetAdminCustomerAccountTimeline(parseBobCtx, &chatpb.GetAdminCustomerAccountTimelineRequest{
		UserId:       parseAliceAuth.ID,
		LookbackDays: 30,
		Limit:        25,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("out-of-scope timeline status code=%v want=%v", status.Code(parseErr), codes.PermissionDenied)
	}
	if _, parseErr = parseServer.GetAdminCustomerAccountTimeline(parseBobCtx, &chatpb.GetAdminCustomerAccountTimelineRequest{
		UserId:       0,
		LookbackDays: 30,
		Limit:        25,
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("missing user id status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
}
