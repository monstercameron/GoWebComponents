package app

import (
	"context"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestRenameConversation persists one customer-managed title and returns updated conversation metadata.
func TestRenameConversation(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "rename-conversation@example.com")
	parseConversationID, parseErr := parseStore.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation: %v", parseErr)
	}
	if parseErr = parseStore.parseSaveConversationMessage(parseUser.ID, parseConversationID, "user", "Original preview", "", 0, 0); parseErr != nil {
		parseT.Fatalf("parseSaveConversationMessage: %v", parseErr)
	}
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseCtx := parseBindAuthUser(parseServer, "peer-rename-conversation", parseUser.ID, parseUser.Email)

	parseRenameResp, parseErr := parseServer.RenameConversation(parseCtx, &chatpb.RenameConversationRequest{
		Id:    parseConversationID,
		Title: "Customer managed title",
	})
	if parseErr != nil {
		parseT.Fatalf("RenameConversation: %v", parseErr)
	}
	if parseRenameResp.GetConversation() == nil || parseRenameResp.GetConversation().GetId() != parseConversationID {
		parseT.Fatalf("unexpected rename response conversation: %+v", parseRenameResp.GetConversation())
	}
	if parseRenameResp.GetConversation().GetPreview() != "Customer managed title" {
		parseT.Fatalf("expected renamed preview in response, got %+v", parseRenameResp.GetConversation())
	}

	parseListResp, parseErr := parseServer.ListConversations(parseCtx, &chatpb.ListConversationsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListConversations: %v", parseErr)
	}
	if len(parseListResp.GetConversations()) == 0 || parseListResp.GetConversations()[0].GetPreview() != "Customer managed title" {
		parseT.Fatalf("expected renamed preview in list response, got %+v", parseListResp.GetConversations())
	}
}

// TestRenameConversationValidation verifies title and scope validation behavior.
func TestRenameConversationValidation(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "rename-validation-owner@example.com")
	parseOtherUser := parseMustCreateUser(parseT, parseStore, "rename-validation-other@example.com")
	parseConversationID, parseErr := parseStore.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation: %v", parseErr)
	}
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger(), "all")
	parseOwnerCtx := parseBindAuthUser(parseServer, "peer-rename-validation-owner", parseUser.ID, parseUser.Email)
	parseOtherCtx := parseBindAuthUser(parseServer, "peer-rename-validation-other", parseOtherUser.ID, parseOtherUser.Email)

	if _, parseErr = parseServer.RenameConversation(parseOwnerCtx, &chatpb.RenameConversationRequest{
		Id:    parseConversationID,
		Title: "   ",
	}); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("empty title status code=%v want=%v", status.Code(parseErr), codes.InvalidArgument)
	}
	if _, parseErr = parseServer.RenameConversation(parseOtherCtx, &chatpb.RenameConversationRequest{
		Id:    parseConversationID,
		Title: "out of scope",
	}); status.Code(parseErr) != codes.NotFound {
		parseT.Fatalf("out-of-scope rename status code=%v want=%v", status.Code(parseErr), codes.NotFound)
	}
	if _, parseErr = parseServer.RenameConversation(context.Background(), &chatpb.RenameConversationRequest{
		Id:    parseConversationID,
		Title: "unauthenticated",
	}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("unauthenticated rename status code=%v want=%v", status.Code(parseErr), codes.Unauthenticated)
	}
}
