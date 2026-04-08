package app

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type parseAdminSupportTicketDetail struct {
	parseTicketRow      parseSupportTicketRow
	parseMessageRows    []parseSupportTicketMessageRow
	parseAccountActions []parseAuditLogRow
}

// parseBuildAdminSupportTicketEntry maps one support-ticket row into protobuf form.
func parseBuildAdminSupportTicketEntry(parseRow parseSupportTicketRow) *chatpb.SupportTicketEntry {
	return &chatpb.SupportTicketEntry{
		Id:             parseRow.ID,
		TicketKey:      parseRow.TicketKey,
		WorkspaceId:    parseRow.WorkspaceID,
		UserId:         parseRow.UserID,
		Status:         parseRow.Status,
		Priority:       parseRow.Priority,
		Subject:        parseRow.Subject,
		Body:           parseRow.Body,
		AssigneeUserId: parseRow.AssigneeUserID,
		ResolutionNote: parseRow.ResolutionNote,
		CreatedAt:      parseRow.CreatedAt,
		UpdatedAt:      parseRow.UpdatedAt,
	}
}

// parseBuildAdminSupportTicketMessageEntry maps one support-ticket message row into protobuf form.
func parseBuildAdminSupportTicketMessageEntry(parseRow parseSupportTicketMessageRow) *chatpb.SupportTicketMessageEntry {
	return &chatpb.SupportTicketMessageEntry{
		Id:           parseRow.ID,
		TicketId:     parseRow.TicketID,
		AuthorUserId: parseRow.AuthorUserID,
		MessageType:  parseRow.MessageType,
		Body:         parseRow.Body,
		IsInternal:   parseRow.IsInternal,
		CreatedAt:    parseRow.CreatedAt,
		UpdatedAt:    parseRow.UpdatedAt,
	}
}

// parseRequireAdminSupportActionConfirmation enforces explicit confirmation and one non-empty reason for support-ticket mutations.
func parseRequireAdminSupportActionConfirmation(isParseConfirmed bool, parseReason string) (string, error) {
	if !isParseConfirmed {
		return "", status.Error(codes.InvalidArgument, "support ticket mutation confirmation is required")
	}
	parseReason = strings.TrimSpace(parseReason)
	if parseReason == "" {
		return "", status.Error(codes.InvalidArgument, "support ticket mutation reason is required")
	}
	return parseReason, nil
}

// parseGetSupportTicketRowByID resolves one support-ticket row by id.
func (parseS *chatServer) parseGetSupportTicketRowByID(parseTicketID int64) (parseSupportTicketRow, bool, error) {
	if parseTicketID <= 0 {
		return parseSupportTicketRow{}, false, nil
	}
	if parseS == nil || parseS.store == nil {
		return parseSupportTicketRow{}, false, status.Error(codes.Unavailable, "store unavailable")
	}
	parseTicketRows, parseErr := parseS.store.parseListSupportTickets(parseAdminScopedScanLimit)
	if parseErr != nil {
		return parseSupportTicketRow{}, false, parseErr
	}
	return parseFindSupportTicketRowByID(parseTicketRows, parseTicketID)
}

// parseRequireAdminSupportTicketScope enforces caller scope for one support-ticket id and resolves the ticket row.
func (parseS *chatServer) parseRequireAdminSupportTicketScope(parseCtx context.Context, parseTicketID int64, parseSliceKey string) (parseAdminAccessScope, parseSupportTicketRow, error) {
	if parseTicketID <= 0 {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.InvalidArgument, "ticket id is required")
	}
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, parseSliceKey)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, parseErr
	}
	parseTicketRow, isParseFound, parseErr := parseS.parseGetSupportTicketRowByID(parseTicketID)
	if parseErr != nil {
		if status.Code(parseErr) == codes.Unavailable {
			return parseAdminAccessScope{}, parseSupportTicketRow{}, parseErr
		}
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Errorf(codes.Internal, "get support ticket: %v", parseErr)
	}
	if !isParseFound {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.NotFound, "support ticket not found")
	}
	if !parseScope.isPlatformScope {
		if _, hasParseWorkspace := parseScope.workspaceIDs[parseTicketRow.WorkspaceID]; !hasParseWorkspace {
			return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.PermissionDenied, "support ticket outside workspace-admin scope")
		}
	}
	return parseScope, parseTicketRow, nil
}

// parseListSupportTicketQueueByAdmin lists one scoped support queue with typed status/priority/assignee filters.
func (parseS *chatServer) parseListSupportTicketQueueByAdmin(parseCtx context.Context, parseListQueryShape parseAdminListQueryShape, parseStatusFilter string, parsePriorityFilter string, parseAssigneeUserID int64) (parseAdminAccessScope, []parseSupportTicketRow, error) {
	parseScope, parseErr := parseS.parseRequireAdminSliceScope(parseCtx, "dashboard.support.queue")
	if parseErr != nil {
		return parseAdminAccessScope{}, nil, parseErr
	}
	if parseS == nil || parseS.store == nil {
		return parseScope, nil, status.Error(codes.Unavailable, "store unavailable")
	}
	parseQueryLimit := parseAdminScopedScanLimit
	parseTicketRows, parseErr := parseS.store.parseListSupportTickets(parseQueryLimit)
	if parseErr != nil {
		return parseAdminAccessScope{}, nil, status.Errorf(codes.Internal, "list support tickets: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseTicketRows = parseFilterSupportTicketRowsByWorkspaceScope(parseTicketRows, parseScope.workspaceIDs)
	}
	parseTicketRows = parseFilterSupportTicketRows(parseTicketRows, parseStatusFilter, parsePriorityFilter, parseAssigneeUserID)
	parseTicketRows = parseFilterSupportTicketRowsBySearch(parseTicketRows, parseListQueryShape.parseSearch)
	parseSortSupportTicketRows(parseTicketRows, parseListQueryShape.parseSortBy, parseListQueryShape.isParseSortAscending)
	parseTicketRows = parseApplyAdminSliceWindow(parseTicketRows, parseListQueryShape.parseOffset, parseListQueryShape.parseLimit)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.slice.view",
		"slice",
		"support-queue",
		"Admin support queue viewed",
		"{}",
		parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs),
	)
	return parseScope, parseTicketRows, nil
}

// parseGetSupportTicketDetailByAdmin resolves one scoped support-ticket detail with ticket messages and account-linked action history.
func (parseS *chatServer) parseGetSupportTicketDetailByAdmin(parseCtx context.Context, parseTicketID int64, parseLimit int32) (parseAdminAccessScope, parseAdminSupportTicketDetail, error) {
	parseScope, parseTicketRow, parseErr := parseS.parseRequireAdminSupportTicketScope(parseCtx, parseTicketID, "dashboard.support.detail")
	if parseErr != nil {
		return parseAdminAccessScope{}, parseAdminSupportTicketDetail{}, parseErr
	}
	parseLimit = parseClampAdminListLimit(parseLimit)
	if parseS == nil || parseS.store == nil {
		return parseScope, parseAdminSupportTicketDetail{}, status.Error(codes.Unavailable, "store unavailable")
	}
	parseQueryLimit := int64(parseLimit)
	if !parseScope.isPlatformScope {
		parseQueryLimit = parseAdminScopedScanLimit
	}
	parseMessageRows, parseErr := parseS.store.parseListSupportTicketMessages(parseQueryLimit)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseAdminSupportTicketDetail{}, status.Errorf(codes.Internal, "list support ticket messages: %v", parseErr)
	}
	parseMessageRows = parseFilterSupportTicketMessageRowsByTicket(parseMessageRows, parseTicketID)
	parseMessageRows = parseLimitSupportTicketMessageRows(parseMessageRows, parseLimit)

	parseAuditRows, parseErr := parseS.store.parseListAdminSupportAccountActionHistoryByTicketID(parseTicketID, parseQueryLimit)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseAdminSupportTicketDetail{}, status.Errorf(codes.Internal, "list account action history: %v", parseErr)
	}
	if !parseScope.isPlatformScope {
		parseAuditRows = parseFilterAdminAuditRowsByWorkspaceScope(parseAuditRows, parseScope.workspaceIDs)
	}
	parseAuditRows = parseLimitAdminAuditRows(parseAuditRows, parseLimit)
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.dashboard.drilldown.access",
		"support_ticket",
		strconv.FormatInt(parseTicketID, 10),
		"Admin support ticket detail viewed",
		"{}",
		parseTicketRow.WorkspaceID,
	)
	return parseScope, parseAdminSupportTicketDetail{
		parseTicketRow:      parseTicketRow,
		parseMessageRows:    parseMessageRows,
		parseAccountActions: parseAuditRows,
	}, nil
}

// parseAddSupportInternalNoteByAdmin appends one internal note to one scoped support ticket.
func (parseS *chatServer) parseAddSupportInternalNoteByAdmin(parseCtx context.Context, parseTicketID int64, parseBody string, isParseConfirmed bool, parseReason string) (parseAdminAccessScope, int64, error) {
	parseScope, parseTicketRow, parseErr := parseS.parseRequireAdminSupportTicketScope(parseCtx, parseTicketID, "dashboard.support.note.add")
	if parseErr != nil {
		return parseAdminAccessScope{}, 0, parseErr
	}
	parseBody = strings.TrimSpace(parseBody)
	if parseBody == "" {
		return parseAdminAccessScope{}, 0, status.Error(codes.InvalidArgument, "note body is required")
	}
	parseReason, parseErr = parseRequireAdminSupportActionConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return parseAdminAccessScope{}, 0, parseErr
	}
	parseMessageID, parseErr := parseS.store.parseCreateSupportTicketMessage(parseSupportTicketMessageWrite{
		TicketID:     parseTicketID,
		AuthorUserID: parseScope.adminUserID,
		MessageType:  "internal_note",
		Body:         parseBody,
		IsInternal:   true,
	})
	if parseErr != nil {
		return parseAdminAccessScope{}, 0, status.Errorf(codes.Internal, "create support internal note: %v", parseErr)
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.support.internal_note.added",
		"support_ticket",
		strconv.FormatInt(parseTicketID, 10),
		"Admin support internal note added: "+parseReason,
		"{}",
		parseTicketRow.WorkspaceID,
	)
	return parseScope, parseMessageID, nil
}

// parseAssignSupportTicketByAdmin sets one assignee for one scoped support ticket.
func (parseS *chatServer) parseAssignSupportTicketByAdmin(parseCtx context.Context, parseTicketID int64, parseAssigneeUserID int64, isParseConfirmed bool, parseReason string) (parseAdminAccessScope, parseSupportTicketRow, error) {
	parseScope, parseTicketRow, parseErr := parseS.parseRequireAdminSupportTicketScope(parseCtx, parseTicketID, "dashboard.support.assignment")
	if parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, parseErr
	}
	if parseAssigneeUserID <= 0 {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.InvalidArgument, "assignee user id is required")
	}
	parseReason, parseErr = parseRequireAdminSupportActionConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, parseErr
	}
	if !parseScope.isPlatformScope {
		if _, hasParseUser := parseScope.userIDs[parseAssigneeUserID]; !hasParseUser {
			return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.PermissionDenied, "assignee user outside workspace-admin scope")
		}
	}
	_, hasParseUser, parseErr := parseS.store.parseGetAdminUserSummaryByUserID(parseAssigneeUserID)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Errorf(codes.Internal, "get assignee user: %v", parseErr)
	}
	if !hasParseUser {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.NotFound, "assignee user not found")
	}

	if parseErr = parseS.store.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      parseTicketRow.TicketKey,
		WorkspaceID:    parseTicketRow.WorkspaceID,
		UserID:         parseTicketRow.UserID,
		Status:         parseTicketRow.Status,
		Priority:       parseTicketRow.Priority,
		Subject:        parseTicketRow.Subject,
		Body:           parseTicketRow.Body,
		AssigneeUserID: parseAssigneeUserID,
		ResolutionNote: parseTicketRow.ResolutionNote,
	}); parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Errorf(codes.Internal, "assign support ticket: %v", parseErr)
	}
	parseUpdatedTicket, isParseFound, parseErr := parseS.parseGetSupportTicketRowByID(parseTicketID)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Errorf(codes.Internal, "reload assigned support ticket: %v", parseErr)
	}
	if !isParseFound {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.NotFound, "support ticket not found after assignment")
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.support.ticket.assigned",
		"support_ticket",
		strconv.FormatInt(parseTicketID, 10),
		fmt.Sprintf("Support ticket assigned to user %d: %s", parseAssigneeUserID, parseReason),
		"{}",
		parseTicketRow.WorkspaceID,
	)
	return parseScope, parseUpdatedTicket, nil
}

// parseEscalateSupportTicketByAdmin applies one escalation update for one scoped support ticket.
func (parseS *chatServer) parseEscalateSupportTicketByAdmin(parseCtx context.Context, parseTicketID int64, parsePriority string, parseStatusValue string, parseResolutionNote string, isParseConfirmed bool, parseReason string) (parseAdminAccessScope, parseSupportTicketRow, error) {
	parseScope, parseTicketRow, parseErr := parseS.parseRequireAdminSupportTicketScope(parseCtx, parseTicketID, "dashboard.support.escalation")
	if parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, parseErr
	}
	parsePriority = strings.TrimSpace(parsePriority)
	parseStatusValue = strings.TrimSpace(parseStatusValue)
	parseResolutionNote = strings.TrimSpace(parseResolutionNote)
	if parsePriority == "" {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.InvalidArgument, "escalation priority is required")
	}
	if parseStatusValue == "" {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.InvalidArgument, "escalation status is required")
	}
	if parseResolutionNote == "" {
		parseResolutionNote = parseTicketRow.ResolutionNote
	}
	parseReason, parseErr = parseRequireAdminSupportActionConfirmation(isParseConfirmed, parseReason)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, parseErr
	}

	if parseErr = parseS.store.parseUpsertSupportTicket(parseSupportTicketWrite{
		TicketKey:      parseTicketRow.TicketKey,
		WorkspaceID:    parseTicketRow.WorkspaceID,
		UserID:         parseTicketRow.UserID,
		Status:         parseStatusValue,
		Priority:       parsePriority,
		Subject:        parseTicketRow.Subject,
		Body:           parseTicketRow.Body,
		AssigneeUserID: parseTicketRow.AssigneeUserID,
		ResolutionNote: parseResolutionNote,
	}); parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Errorf(codes.Internal, "escalate support ticket: %v", parseErr)
	}
	parseUpdatedTicket, isParseFound, parseErr := parseS.parseGetSupportTicketRowByID(parseTicketID)
	if parseErr != nil {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Errorf(codes.Internal, "reload escalated support ticket: %v", parseErr)
	}
	if !isParseFound {
		return parseAdminAccessScope{}, parseSupportTicketRow{}, status.Error(codes.NotFound, "support ticket not found after escalation")
	}
	parseS.parseTrackAdminAuditEvent(
		parseScope,
		"admin.support.ticket.escalated",
		"support_ticket",
		strconv.FormatInt(parseTicketID, 10),
		fmt.Sprintf("Support ticket escalated to status=%s priority=%s: %s", parseStatusValue, parsePriority, parseReason),
		"{}",
		parseTicketRow.WorkspaceID,
	)
	return parseScope, parseUpdatedTicket, nil
}

// ListAdminSupportTickets returns one scoped support queue slice.
func (parseS *chatServer) ListAdminSupportTickets(parseCtx context.Context, parseReq *chatpb.ListAdminSupportTicketsRequest) (*chatpb.ListAdminSupportTicketsResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "ListAdminSupportTickets"))
	var parseLimit int32
	var parseStatusFilter string
	var parsePriorityFilter string
	var parseAssigneeUserID int64
	var parseListQuery *chatpb.AdminListQuery
	if parseReq != nil {
		parseLimit = parseReq.GetLimit()
		parseStatusFilter = parseReq.GetStatus()
		parsePriorityFilter = parseReq.GetPriority()
		parseAssigneeUserID = parseReq.GetAssigneeUserId()
		parseListQuery = parseReq.GetListQuery()
	}
	parseListQueryShape := parseBuildAdminListQueryShape(parseLimit, parseListQuery)
	parseScope, parseTicketRows, parseErr := parseS.parseListSupportTicketQueueByAdmin(parseCtx, parseListQueryShape, parseStatusFilter, parsePriorityFilter, parseAssigneeUserID)
	if parseErr != nil {
		if status.Code(parseErr) == codes.Unavailable {
			parseLogger.Warn("rpc.ListAdminSupportTickets: store unavailable")
			return &chatpb.ListAdminSupportTicketsResponse{}, nil
		}
		return nil, parseErr
	}
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseResponse := &chatpb.ListAdminSupportTicketsResponse{
		Tickets: make([]*chatpb.SupportTicketEntry, 0, len(parseTicketRows)),
	}
	for _, parseTicketRow := range parseTicketRows {
		parseResponse.Tickets = append(parseResponse.Tickets, parseRedactAdminSupportTicketEntryByScope(parseScope, parseBuildAdminSupportTicketEntry(parseTicketRow)))
	}
	parseLogger.Info(
		"rpc.ListAdminSupportTickets: complete",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int("count", len(parseResponse.Tickets)),
	)
	return parseResponse, nil
}

// GetAdminSupportTicketDetail returns one scoped support ticket detail with messages and account-linked action history.
func (parseS *chatServer) GetAdminSupportTicketDetail(parseCtx context.Context, parseReq *chatpb.GetAdminSupportTicketDetailRequest) (*chatpb.GetAdminSupportTicketDetailResponse, error) {
	parseLogger := parseS.logger.With(slog.String("rpc", "GetAdminSupportTicketDetail"))
	parseFetchStart := time.Now()
	var parseTicketID int64
	var parseLimit int32
	if parseReq != nil {
		parseTicketID = parseReq.GetTicketId()
		parseLimit = parseReq.GetLimit()
	}
	parseScope, parseDetail, parseErr := parseS.parseGetSupportTicketDetailByAdmin(parseCtx, parseTicketID, parseLimit)
	if parseErr != nil {
		if status.Code(parseErr) == codes.Unavailable {
			parseLogger.Warn("rpc.GetAdminSupportTicketDetail: store unavailable")
			return &chatpb.GetAdminSupportTicketDetailResponse{}, nil
		}
		return nil, parseErr
	}
	parseScopeType := "workspace"
	if parseScope.isPlatformScope {
		parseScopeType = "platform"
	}
	parseResponse := &chatpb.GetAdminSupportTicketDetailResponse{
		Detail: &chatpb.AdminSupportTicketDetail{
			Ticket:               parseRedactAdminSupportTicketEntryByScope(parseScope, parseBuildAdminSupportTicketEntry(parseDetail.parseTicketRow)),
			Messages:             make([]*chatpb.SupportTicketMessageEntry, 0, len(parseDetail.parseMessageRows)),
			AccountActionHistory: make([]*chatpb.AuditLogEntry, 0, len(parseDetail.parseAccountActions)),
		},
	}
	for _, parseMessageRow := range parseDetail.parseMessageRows {
		parseResponse.Detail.Messages = append(parseResponse.Detail.Messages, parseRedactAdminSupportTicketMessageEntryByScope(parseScope, parseBuildAdminSupportTicketMessageEntry(parseMessageRow)))
	}
	for _, parseAuditRow := range parseDetail.parseAccountActions {
		parseResponse.Detail.AccountActionHistory = append(parseResponse.Detail.AccountActionHistory, parseRedactAdminAuditLogEntryByScope(parseScope, parseBuildAdminAuditLogEntry(parseAuditRow)))
	}
	parseFetchDuration := time.Since(parseFetchStart)
	parseAggregateCount := len(parseResponse.Detail.Messages) + len(parseResponse.Detail.AccountActionHistory)
	parseLogAdminFetchOutcome(parseLogger, "rpc.GetAdminSupportTicketDetail", "drilldown", parseScopeType, parseFetchDuration, parseAggregateCount)
	parseLogger.Info(
		"rpc.GetAdminSupportTicketDetail: complete",
		slog.Int64("admin_user_id", parseScope.adminUserID),
		slog.String("scope", parseScopeType),
		slog.Int64("ticket_id", parseTicketID),
		slog.Int("messages", len(parseResponse.Detail.Messages)),
		slog.Int("account_history", len(parseResponse.Detail.AccountActionHistory)),
		slog.Duration("duration", parseFetchDuration),
	)
	return parseResponse, nil
}

// AddAdminSupportInternalNote appends one internal note to one scoped support ticket.
func (parseS *chatServer) AddAdminSupportInternalNote(parseCtx context.Context, parseReq *chatpb.AddAdminSupportInternalNoteRequest) (*chatpb.AddAdminSupportInternalNoteResponse, error) {
	var parseTicketID int64
	var parseBody string
	var parseConfirm bool
	var parseReason string
	if parseReq != nil {
		parseTicketID = parseReq.GetTicketId()
		parseBody = parseReq.GetBody()
		parseConfirm = parseReq.GetConfirm()
		parseReason = parseReq.GetReason()
	}
	_, parseMessageID, parseErr := parseS.parseAddSupportInternalNoteByAdmin(parseCtx, parseTicketID, parseBody, parseConfirm, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	return &chatpb.AddAdminSupportInternalNoteResponse{
		TicketId:  parseTicketID,
		MessageId: parseMessageID,
		Status:    "created",
	}, nil
}

// AssignAdminSupportTicket sets one assignee for one scoped support ticket.
func (parseS *chatServer) AssignAdminSupportTicket(parseCtx context.Context, parseReq *chatpb.AssignAdminSupportTicketRequest) (*chatpb.AssignAdminSupportTicketResponse, error) {
	var parseTicketID int64
	var parseAssigneeUserID int64
	var parseConfirm bool
	var parseReason string
	if parseReq != nil {
		parseTicketID = parseReq.GetTicketId()
		parseAssigneeUserID = parseReq.GetAssigneeUserId()
		parseConfirm = parseReq.GetConfirm()
		parseReason = parseReq.GetReason()
	}
	_, parseTicketRow, parseErr := parseS.parseAssignSupportTicketByAdmin(parseCtx, parseTicketID, parseAssigneeUserID, parseConfirm, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	return &chatpb.AssignAdminSupportTicketResponse{
		TicketId:       parseTicketID,
		AssigneeUserId: parseTicketRow.AssigneeUserID,
		Status:         parseTicketRow.Status,
	}, nil
}

// EscalateAdminSupportTicket escalates one scoped support ticket with updated status and priority.
func (parseS *chatServer) EscalateAdminSupportTicket(parseCtx context.Context, parseReq *chatpb.EscalateAdminSupportTicketRequest) (*chatpb.EscalateAdminSupportTicketResponse, error) {
	var parseTicketID int64
	var parsePriority string
	var parseStatusValue string
	var parseResolutionNote string
	var parseConfirm bool
	var parseReason string
	if parseReq != nil {
		parseTicketID = parseReq.GetTicketId()
		parsePriority = parseReq.GetPriority()
		parseStatusValue = parseReq.GetStatus()
		parseResolutionNote = parseReq.GetResolutionNote()
		parseConfirm = parseReq.GetConfirm()
		parseReason = parseReq.GetReason()
	}
	_, parseTicketRow, parseErr := parseS.parseEscalateSupportTicketByAdmin(parseCtx, parseTicketID, parsePriority, parseStatusValue, parseResolutionNote, parseConfirm, parseReason)
	if parseErr != nil {
		return nil, parseErr
	}
	return &chatpb.EscalateAdminSupportTicketResponse{
		TicketId: parseTicketID,
		Priority: parseTicketRow.Priority,
		Status:   parseTicketRow.Status,
	}, nil
}

// parseFilterSupportTicketRowsByWorkspaceScope filters support-ticket rows to one workspace-admin scope.
func parseFilterSupportTicketRowsByWorkspaceScope(parseRows []parseSupportTicketRow, parseWorkspaceIDs map[int64]struct{}) []parseSupportTicketRow {
	parseFilteredRows := make([]parseSupportTicketRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if _, hasParseWorkspace := parseWorkspaceIDs[parseRow.WorkspaceID]; !hasParseWorkspace {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterSupportTicketRows filters support-ticket rows by optional status, priority, and assignee.
func parseFilterSupportTicketRows(parseRows []parseSupportTicketRow, parseStatusFilter string, parsePriorityFilter string, parseAssigneeUserID int64) []parseSupportTicketRow {
	parseStatusFilter = strings.TrimSpace(strings.ToLower(parseStatusFilter))
	parsePriorityFilter = strings.TrimSpace(strings.ToLower(parsePriorityFilter))
	parseFilteredRows := make([]parseSupportTicketRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseStatusFilter != "" && strings.TrimSpace(strings.ToLower(parseRow.Status)) != parseStatusFilter {
			continue
		}
		if parsePriorityFilter != "" && strings.TrimSpace(strings.ToLower(parseRow.Priority)) != parsePriorityFilter {
			continue
		}
		if parseAssigneeUserID > 0 && parseRow.AssigneeUserID != parseAssigneeUserID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseFilterSupportTicketRowsBySearch filters support-ticket rows by one optional search term.
func parseFilterSupportTicketRowsBySearch(parseRows []parseSupportTicketRow, parseSearch string) []parseSupportTicketRow {
	parseSearch = strings.TrimSpace(strings.ToLower(parseSearch))
	if parseSearch == "" {
		return parseRows
	}
	parseFilteredRows := make([]parseSupportTicketRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if strings.Contains(strconv.FormatInt(parseRow.ID, 10), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.TicketKey)), parseSearch) ||
			strings.Contains(strconv.FormatInt(parseRow.WorkspaceID, 10), parseSearch) ||
			strings.Contains(strconv.FormatInt(parseRow.UserID, 10), parseSearch) ||
			strings.Contains(strconv.FormatInt(parseRow.AssigneeUserID, 10), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Status)), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Priority)), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Subject)), parseSearch) ||
			strings.Contains(strings.ToLower(strings.TrimSpace(parseRow.Body)), parseSearch) {
			parseFilteredRows = append(parseFilteredRows, parseRow)
		}
	}
	return parseFilteredRows
}

// parseSortSupportTicketRows sorts support-ticket rows by one typed sort key and direction when provided.
func parseSortSupportTicketRows(parseRows []parseSupportTicketRow, parseSortBy string, isParseSortAscending bool) {
	parseSortBy = strings.TrimSpace(strings.ToLower(parseSortBy))
	if parseSortBy == "" {
		return
	}
	sort.SliceStable(parseRows, func(parseLeftIndex, parseRightIndex int) bool {
		parseLeftRow := parseRows[parseLeftIndex]
		parseRightRow := parseRows[parseRightIndex]
		switch parseSortBy {
		case "id":
			return parseCompareAdminInt64(parseLeftRow.ID, parseRightRow.ID, isParseSortAscending)
		case "workspace_id":
			return parseCompareAdminInt64(parseLeftRow.WorkspaceID, parseRightRow.WorkspaceID, isParseSortAscending)
		case "user_id":
			return parseCompareAdminInt64(parseLeftRow.UserID, parseRightRow.UserID, isParseSortAscending)
		case "assignee_user_id":
			return parseCompareAdminInt64(parseLeftRow.AssigneeUserID, parseRightRow.AssigneeUserID, isParseSortAscending)
		case "ticket_key":
			return parseCompareAdminString(parseLeftRow.TicketKey, parseRightRow.TicketKey, isParseSortAscending)
		case "status":
			return parseCompareAdminString(parseLeftRow.Status, parseRightRow.Status, isParseSortAscending)
		case "priority":
			return parseCompareAdminString(parseLeftRow.Priority, parseRightRow.Priority, isParseSortAscending)
		case "created_at":
			return parseCompareAdminString(parseLeftRow.CreatedAt, parseRightRow.CreatedAt, isParseSortAscending)
		case "updated_at":
			return parseCompareAdminString(parseLeftRow.UpdatedAt, parseRightRow.UpdatedAt, isParseSortAscending)
		default:
			return false
		}
	})
}

// parseLimitSupportTicketRows truncates support-ticket rows to one RPC-safe limit.
func parseLimitSupportTicketRows(parseRows []parseSupportTicketRow, parseLimit int32) []parseSupportTicketRow {
	if parseLimit <= 0 || len(parseRows) <= int(parseLimit) {
		return parseRows
	}
	return parseRows[:parseLimit]
}

// parseFindSupportTicketRowByID resolves one support-ticket row by id from one in-memory row set.
func parseFindSupportTicketRowByID(parseRows []parseSupportTicketRow, parseTicketID int64) (parseSupportTicketRow, bool, error) {
	for _, parseRow := range parseRows {
		if parseRow.ID != parseTicketID {
			continue
		}
		return parseRow, true, nil
	}
	return parseSupportTicketRow{}, false, nil
}

// parseFilterSupportTicketMessageRowsByTicket filters support-ticket message rows to one ticket id.
func parseFilterSupportTicketMessageRowsByTicket(parseRows []parseSupportTicketMessageRow, parseTicketID int64) []parseSupportTicketMessageRow {
	parseFilteredRows := make([]parseSupportTicketMessageRow, 0, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.TicketID != parseTicketID {
			continue
		}
		parseFilteredRows = append(parseFilteredRows, parseRow)
	}
	return parseFilteredRows
}

// parseLimitSupportTicketMessageRows truncates support-ticket message rows to one RPC-safe limit.
func parseLimitSupportTicketMessageRows(parseRows []parseSupportTicketMessageRow, parseLimit int32) []parseSupportTicketMessageRow {
	if parseLimit <= 0 || len(parseRows) <= int(parseLimit) {
		return parseRows
	}
	return parseRows[:parseLimit]
}
