package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxConversationTitleRunes = 120

// parseNormalizeConversationTitle validates one customer-managed conversation title.
func parseNormalizeConversationTitle(parseTitle string) (string, error) {
	parseTitle = strings.TrimSpace(parseTitle)
	if parseTitle == "" {
		return "", status.Error(codes.InvalidArgument, "title must not be empty")
	}
	if utf8.RuneCountInString(parseTitle) > maxConversationTitleRunes {
		return "", status.Errorf(codes.InvalidArgument, "title must not exceed %d characters", maxConversationTitleRunes)
	}
	return parseTitle, nil
}

// parseBuildConversationSummaryEntry maps one conversation summary store row into protobuf form.
func parseBuildConversationSummaryEntry(parseRow conversationSummaryRow) *chatpb.ConversationSummary {
	parsePreview := parseRow.Preview
	if len(parsePreview) > 60 {
		parsePreview = parsePreview[:60] + "..."
	}
	return &chatpb.ConversationSummary{
		Id:        parseRow.ID,
		PublicId:  parseRow.PublicID,
		StartedAt: parseRow.StartedAt,
		Preview:   parsePreview,
	}
}

// parseFindConversationSummaryByID resolves one conversation summary row by conversation id.
func parseFindConversationSummaryByID(parseRows []conversationSummaryRow, parseConversationID int64) (conversationSummaryRow, bool) {
	for _, parseRow := range parseRows {
		if parseRow.ID == parseConversationID {
			return parseRow, true
		}
	}
	return conversationSummaryRow{}, false
}

// RenameConversation persists one customer-managed title for one user-owned conversation.
func (parseS *chatServer) RenameConversation(parseCtx context.Context, parseReq *chatpb.RenameConversationRequest) (*chatpb.RenameConversationResponse, error) {
	parseConversationID := int64(0)
	parseRequestedTitle := ""
	if parseReq != nil {
		parseConversationID = parseReq.GetId()
		parseRequestedTitle = parseReq.GetTitle()
	}
	parseLogger := parseS.logger.With(slog.String("rpc", "RenameConversation"), slog.Int64("conv_id", parseConversationID))
	if parseConversationID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "conversation id is required")
	}
	parseTitle, parseErr := parseNormalizeConversationTitle(parseRequestedTitle)
	if parseErr != nil {
		return nil, parseErr
	}
	parseUserID, parseErr := parseS.parseRequireAuthenticatedUserID(parseCtx)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseS.store == nil {
		parseLogger.Warn("rpc.RenameConversation: store unavailable")
		return nil, status.Error(codes.Unavailable, "conversation persistence unavailable")
	}
	isOwned, parseErr := parseS.store.parseConversationOwnedByUser(parseUserID, parseConversationID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "resolve conversation ownership: %v", parseErr)
	}
	if !isOwned {
		return nil, status.Error(codes.NotFound, "conversation not found")
	}
	if parseErr = parseS.store.parseSaveConversationTitle(parseUserID, parseConversationID, parseTitle); parseErr != nil {
		if errors.Is(parseErr, errStoreConversationMissing) {
			return nil, status.Error(codes.NotFound, "conversation not found")
		}
		if isStoreGuardError(parseErr) {
			return nil, parseStatusForStoreGuard(parseErr)
		}
		return nil, status.Errorf(codes.Internal, "save conversation title: %v", parseErr)
	}
	parseConversationRows, parseErr := parseS.store.parseListConversations(parseUserID)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "list conversations: %v", parseErr)
	}
	parseSummaryRow, isHasSummary := parseFindConversationSummaryByID(parseConversationRows, parseConversationID)
	if !isHasSummary {
		return nil, status.Error(codes.NotFound, "conversation not found")
	}
	parseLogger.Info("rpc.RenameConversation: complete", slog.String("title", parseTitle))
	return &chatpb.RenameConversationResponse{Conversation: parseBuildConversationSummaryEntry(parseSummaryRow)}, nil
}
