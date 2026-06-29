package app

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateConversationSessionForkCreatesNewConversation(parseT *testing.T) {
	store, parseErr := parseOpenChatStore(filepath.Join(parseT.TempDir(), "chat.db"))
	if parseErr != nil {
		parseT.Fatalf("openChatStore: %v", parseErr)
	}
	defer store.parseClose()

	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	parseUserID, parseErr := store.parseCreateUser("test@example.com", "hash", "Test User")
	if parseErr != nil {
		parseT.Fatalf("createUser: %v", parseErr)
	}

	parseFirstConversationID, parseSavedCount, parseErr := parseServer.parseLoadOrCreateConversationSession("peer-1", parseUserID, 0, 0)
	if parseErr != nil {
		parseT.Fatalf("first loadOrCreateConversationSession: %v", parseErr)
	}
	if parseSavedCount != 0 {
		parseT.Fatalf("expected first conversation saved count 0, got %d", parseSavedCount)
	}
	parseServer.parseMarkConversationMessagesSaved("peer-1", 4)

	parseForkConversationID, parseForkSavedCount, parseErr := parseServer.parseLoadOrCreateConversationSession("peer-1", parseUserID, 0, 2)
	if parseErr != nil {
		parseT.Fatalf("fork loadOrCreateConversationSession: %v", parseErr)
	}
	if parseForkConversationID == parseFirstConversationID {
		parseT.Fatalf("expected fork to create a new conversation, reused %d", parseForkConversationID)
	}
	if parseForkSavedCount != 0 {
		parseT.Fatalf("expected fork saved count 0 for a new conversation, got %d", parseForkSavedCount)
	}
}
