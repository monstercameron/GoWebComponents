//go:build js && wasm

package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
)

func TestParseNormalizeAdminListQueryClampsPageAndTrims(parseT *testing.T) {
	parseT.Parallel()
	parseQuery := parseNormalizeAdminListQuery(adminListQueryState{
		Search: "  failed payment  ",
		Filter: " open ",
		Sort:   " updated ",
		Page:   -3,
	})
	if parseQuery.Search != "failed payment" {
		parseT.Fatalf("Search = %q, want trimmed value", parseQuery.Search)
	}
	if parseQuery.Filter != "open" || parseQuery.Sort != "updated" {
		parseT.Fatalf("Filter/Sort = %q/%q, want open/updated", parseQuery.Filter, parseQuery.Sort)
	}
	if parseQuery.Page != 0 {
		parseT.Fatalf("Page = %d, want 0", parseQuery.Page)
	}
}

func TestParseBuildAdminListQueryUsesOffsetAndDirection(parseT *testing.T) {
	parseT.Parallel()
	parseListQuery := parseBuildAdminListQuery(adminListQueryState{
		Search: "workspace",
		Sort:   "status",
		Page:   2,
	}, 25)
	if parseListQuery.GetSearch() != "workspace" {
		parseT.Fatalf("Search = %q, want workspace", parseListQuery.GetSearch())
	}
	if parseListQuery.GetSortBy() != "status" {
		parseT.Fatalf("SortBy = %q, want status", parseListQuery.GetSortBy())
	}
	if parseListQuery.GetSortDirection() != "desc" {
		parseT.Fatalf("SortDirection = %q, want desc", parseListQuery.GetSortDirection())
	}
	if parseListQuery.GetLimit() != 25 || parseListQuery.GetOffset() != 50 {
		parseT.Fatalf("Limit/Offset = %d/%d, want 25/50", parseListQuery.GetLimit(), parseListQuery.GetOffset())
	}
}

func TestParsePaginateAdminRowsBounds(parseT *testing.T) {
	parseT.Parallel()
	parseRows := make([][]string, 0, adminOperationsPageSize+2)
	for parseI := 0; parseI < adminOperationsPageSize+2; parseI++ {
		parseRows = append(parseRows, []string{formatDashboardInt(parseI)})
	}
	parsePageRows, parsePage, parseTotalPages := parsePaginateAdminRows(parseRows, 99)
	if parsePage != 1 {
		parseT.Fatalf("Page = %d, want last page 1", parsePage)
	}
	if parseTotalPages != 2 {
		parseT.Fatalf("TotalPages = %d, want 2", parseTotalPages)
	}
	if len(parsePageRows) != 2 {
		parseT.Fatalf("len(PageRows) = %d, want 2", len(parsePageRows))
	}
}

func TestParseMarshalAdminSupportDetailMapsMessagesAndActions(parseT *testing.T) {
	parseT.Parallel()
	parseSnap := parseMarshalAdminSupportDetail(&chatpb.GetAdminSupportTicketDetailResponse{
		Detail: &chatpb.AdminSupportTicketDetail{
			Ticket: &chatpb.SupportTicketEntry{
				Id:          42,
				TicketKey:   "SUP-42",
				WorkspaceId: 7,
				UserId:      9,
				Status:      "open",
				Priority:    "urgent",
				Subject:     "Cannot send messages",
				UpdatedAt:   "2026-06-12T13:00:00Z",
			},
			Messages: []*chatpb.SupportTicketMessageEntry{
				{AuthorUserId: 9, MessageType: "customer", Body: "Help", IsInternal: false, CreatedAt: "2026-06-12T13:01:00Z"},
				{AuthorUserId: 1, MessageType: "note", Body: "Checking workspace suspension", IsInternal: true, CreatedAt: "2026-06-12T13:02:00Z"},
			},
			AccountActionHistory: []*chatpb.AuditLogEntry{
				{EventType: "admin.user.disable", Summary: "Disabled for abuse review", CreatedAt: "2026-06-12T13:03:00Z"},
			},
		},
	})
	if !parseSnap.HasData {
		parseT.Fatal("HasData = false, want true")
	}
	if parseSnap.Ticket.TicketID != 42 || parseSnap.Ticket.Priority != "urgent" {
		parseT.Fatalf("Ticket = %+v, want id 42 urgent", parseSnap.Ticket)
	}
	if len(parseSnap.Messages) != 2 || !parseSnap.Messages[1].IsInternal {
		parseT.Fatalf("Messages = %+v, want two messages with second internal", parseSnap.Messages)
	}
	if len(parseSnap.AccountActions) != 1 || parseSnap.AccountActions[0].EventType != "admin.user.disable" {
		parseT.Fatalf("AccountActions = %+v, want admin.user.disable", parseSnap.AccountActions)
	}
}
