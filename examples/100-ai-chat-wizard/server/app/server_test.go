package app

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateConversationSessionForkCreatesNewConversation(t *testing.T) {
	store, err := openChatStore(filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatalf("openChatStore: %v", err)
	}
	defer store.close()

	server := newChatServiceServer("", "", modelGPT54Mini, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	userID, err := store.createUser("test@example.com", "hash", "Test User")
	if err != nil {
		t.Fatalf("createUser: %v", err)
	}

	firstConversationID, savedCount, err := server.loadOrCreateConversationSession("peer-1", userID, 0, 0)
	if err != nil {
		t.Fatalf("first loadOrCreateConversationSession: %v", err)
	}
	if savedCount != 0 {
		t.Fatalf("expected first conversation saved count 0, got %d", savedCount)
	}
	server.markConversationMessagesSaved("peer-1", 4)

	forkConversationID, forkSavedCount, err := server.loadOrCreateConversationSession("peer-1", userID, 0, 2)
	if err != nil {
		t.Fatalf("fork loadOrCreateConversationSession: %v", err)
	}
	if forkConversationID == firstConversationID {
		t.Fatalf("expected fork to create a new conversation, reused %d", forkConversationID)
	}
	if forkSavedCount != 0 {
		t.Fatalf("expected fork saved count 0 for a new conversation, got %d", forkSavedCount)
	}
}
