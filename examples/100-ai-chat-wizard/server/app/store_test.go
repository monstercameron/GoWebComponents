package app

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStoreConversationAndPreferenceLifecycle(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "demo@example.com")

	conversationID, err := store.createConversation(user.ID)
	if err != nil {
		t.Fatalf("createConversation: %v", err)
	}
	if err := store.saveConversationMessage(user.ID, conversationID, "user", "Hello", "", 0, 0); err != nil {
		t.Fatalf("saveConversationMessage user: %v", err)
	}
	if err := store.saveConversationMessage(user.ID, conversationID, "assistant", "Hi there", modelGPT54Mini, 11, 7); err != nil {
		t.Fatalf("saveConversationMessage assistant: %v", err)
	}
	if err := store.saveConversationTitle(user.ID, conversationID, "Greeting thread"); err != nil {
		t.Fatalf("saveConversationTitle: %v", err)
	}

	owned, err := store.conversationOwnedByUser(user.ID, conversationID)
	if err != nil {
		t.Fatalf("conversationOwnedByUser: %v", err)
	}
	if !owned {
		t.Fatal("expected conversation to belong to user")
	}

	conversations, err := store.listConversations(user.ID)
	if err != nil {
		t.Fatalf("listConversations: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("expected one conversation, got %d", len(conversations))
	}
	if conversations[0].PublicID == "" {
		t.Fatal("expected conversation public_id to be populated")
	}
	if _, err := uuid.Parse(conversations[0].PublicID); err != nil {
		t.Fatalf("expected conversation public_id to be a UUID, got %q: %v", conversations[0].PublicID, err)
	}
	if conversations[0].Preview != "Greeting thread" {
		t.Fatalf("expected title-backed preview, got %q", conversations[0].Preview)
	}

	messages, err := store.loadConversation(user.ID, conversationID)
	if err != nil {
		t.Fatalf("loadConversation: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected two messages, got %d", len(messages))
	}
	if messages[1].ModelID != modelGPT54Mini || messages[1].PromptTokens != 11 || messages[1].CompletionTokens != 7 {
		t.Fatalf("assistant message metadata mismatch: %+v", messages[1])
	}

	updatedAt := time.Now().Unix()
	if err := store.setUserName(user.ID, "Renamed User", updatedAt); err != nil {
		t.Fatalf("setUserName: %v", err)
	}
	name, gotUpdatedAt, err := store.getUserName(user.ID)
	if err != nil {
		t.Fatalf("getUserName: %v", err)
	}
	if name != "Renamed User" || gotUpdatedAt != updatedAt {
		t.Fatalf("unexpected user name state: name=%q updatedAt=%d", name, gotUpdatedAt)
	}

	if err := store.setSelectedModel(user.ID, modelGPT54); err != nil {
		t.Fatalf("setSelectedModel: %v", err)
	}
	if err := store.setSelectedTone(user.ID, "professional"); err != nil {
		t.Fatalf("setSelectedTone: %v", err)
	}
	if err := store.setSelectedThinkingEnabled(user.ID, false); err != nil {
		t.Fatalf("setSelectedThinkingEnabled: %v", err)
	}
	if err := store.setSelectedThinkingEffort(user.ID, "high"); err != nil {
		t.Fatalf("setSelectedThinkingEffort: %v", err)
	}
	if err := store.setSelectedSystemPrompt(user.ID, "Always answer with short bullet points."); err != nil {
		t.Fatalf("setSelectedSystemPrompt: %v", err)
	}
	if err := store.upsertUserMemory(user.ID, userMemoryRow{
		Key:             "preference-editor",
		Category:        "preference",
		Summary:         "Prefers Neovim",
		Detail:          "Uses Neovim daily for coding",
		SourceMessage:   "I use Neovim for most of my work.",
		UsefulnessScore: 84,
		ConfidenceScore: 0.91,
		RubricReason:    "Stable tooling preference",
	}); err != nil {
		t.Fatalf("upsertUserMemory: %v", err)
	}

	selectedModel, err := store.getSelectedModel(user.ID, modelGPT54Mini)
	if err != nil || selectedModel != modelGPT54 {
		t.Fatalf("getSelectedModel: model=%q err=%v", selectedModel, err)
	}
	selectedTone, err := store.getSelectedTone(user.ID, defaultToneID)
	if err != nil || selectedTone != "professional" {
		t.Fatalf("getSelectedTone: tone=%q err=%v", selectedTone, err)
	}
	thinkingEnabled, err := store.getSelectedThinkingEnabled(user.ID, true)
	if err != nil || thinkingEnabled {
		t.Fatalf("getSelectedThinkingEnabled: enabled=%v err=%v", thinkingEnabled, err)
	}
	thinkingEffort, err := store.getSelectedThinkingEffort(user.ID, defaultThinkingEffort)
	if err != nil || thinkingEffort != "high" {
		t.Fatalf("getSelectedThinkingEffort: effort=%q err=%v", thinkingEffort, err)
	}
	customSystemPrompt, err := store.getSelectedSystemPrompt(user.ID, "")
	if err != nil || customSystemPrompt != "Always answer with short bullet points." {
		t.Fatalf("getSelectedSystemPrompt: prompt=%q err=%v", customSystemPrompt, err)
	}
	memories, err := store.listUserMemories(user.ID)
	if err != nil {
		t.Fatalf("listUserMemories: %v", err)
	}
	if len(memories) != 1 || memories[0].Summary != "Prefers Neovim" {
		t.Fatalf("unexpected stored memories: %+v", memories)
	}

	if err := store.deleteConversation(user.ID, conversationID); err != nil {
		t.Fatalf("deleteConversation: %v", err)
	}
	remaining, err := store.listConversations(user.ID)
	if err != nil {
		t.Fatalf("listConversations after delete: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected zero conversations after delete, got %d", len(remaining))
	}
}

func TestStoreFallbacksAndUniqueness(t *testing.T) {
	store := newTestStore(t)
	userID, err := store.createUser("fallback@example.com", "hash", "")
	if err != nil {
		t.Fatalf("createUser: %v", err)
	}
	if _, err := store.createUser("fallback@example.com", "hash", ""); err != errUserAlreadyExists {
		t.Fatalf("expected duplicate user error, got %v", err)
	}

	name, updatedAt, err := store.getUserName(userID + 100)
	if err != nil {
		t.Fatalf("getUserName fallback: %v", err)
	}
	if name != "User" || updatedAt != 0 {
		t.Fatalf("unexpected fallback profile: name=%q updatedAt=%d", name, updatedAt)
	}

	selectedModel, err := store.getSelectedModel(userID+100, modelGPT54Mini)
	if err != nil || selectedModel != modelGPT54Mini {
		t.Fatalf("fallback selected model mismatch: model=%q err=%v", selectedModel, err)
	}
	selectedTone, err := store.getSelectedTone(userID+100, defaultToneID)
	if err != nil || selectedTone != defaultToneID {
		t.Fatalf("fallback selected tone mismatch: tone=%q err=%v", selectedTone, err)
	}
	thinkingEnabled, err := store.getSelectedThinkingEnabled(userID+100, true)
	if err != nil || !thinkingEnabled {
		t.Fatalf("fallback thinking enabled mismatch: enabled=%v err=%v", thinkingEnabled, err)
	}
	thinkingEffort, err := store.getSelectedThinkingEffort(userID+100, defaultThinkingEffort)
	if err != nil || thinkingEffort != defaultThinkingEffort {
		t.Fatalf("fallback thinking effort mismatch: effort=%q err=%v", thinkingEffort, err)
	}
	customSystemPrompt, err := store.getSelectedSystemPrompt(userID+100, "")
	if err != nil || customSystemPrompt != "" {
		t.Fatalf("fallback custom system prompt mismatch: prompt=%q err=%v", customSystemPrompt, err)
	}

	derivedName, derivedUpdatedAt, err := store.getUserName(userID)
	if err != nil {
		t.Fatalf("getUserName derived: %v", err)
	}
	if derivedName != "fallback" || derivedUpdatedAt == 0 {
		t.Fatalf("expected default display name derived from email, got name=%q updatedAt=%d", derivedName, derivedUpdatedAt)
	}

	owned, err := store.conversationOwnedByUser(userID, userID+999)
	if err != nil {
		t.Fatalf("conversationOwnedByUser false branch: %v", err)
	}
	if owned {
		t.Fatal("expected unrelated conversation to not belong to user")
	}
}

func TestCreateConversationRetriesPublicIDConflicts(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "uuid-retry@example.com")

	originalGenerator := newConversationPublicID
	defer func() { newConversationPublicID = originalGenerator }()

	collisionID := uuid.NewString()
	recoveredID := uuid.NewString()
	callCount := 0
	newConversationPublicID = func() string {
		callCount++
		if callCount <= 2 {
			return collisionID
		}
		return recoveredID
	}

	firstConversationID, err := store.createConversation(user.ID)
	if err != nil {
		t.Fatalf("createConversation first: %v", err)
	}
	secondConversationID, err := store.createConversation(user.ID)
	if err != nil {
		t.Fatalf("createConversation second: %v", err)
	}
	if firstConversationID == secondConversationID {
		t.Fatalf("expected distinct conversation rows, got %d and %d", firstConversationID, secondConversationID)
	}

	conversations, err := store.listConversations(user.ID)
	if err != nil {
		t.Fatalf("listConversations: %v", err)
	}
	if len(conversations) != 2 {
		t.Fatalf("expected two conversations, got %d", len(conversations))
	}

	publicIDs := map[string]bool{}
	for _, conversation := range conversations {
		if conversation.PublicID == "" {
			t.Fatal("expected non-empty public_id")
		}
		if _, err := uuid.Parse(conversation.PublicID); err != nil {
			t.Fatalf("expected parseable UUID, got %q: %v", conversation.PublicID, err)
		}
		if publicIDs[conversation.PublicID] {
			t.Fatalf("expected unique public_ids, got duplicate %q", conversation.PublicID)
		}
		publicIDs[conversation.PublicID] = true
	}
	if !publicIDs[collisionID] {
		t.Fatalf("expected initial conversation to keep first generated UUID %q", collisionID)
	}
	if !publicIDs[recoveredID] {
		t.Fatalf("expected retry path to use fallback UUID %q", recoveredID)
	}
	if callCount < 3 {
		t.Fatalf("expected generator to be called at least 3 times, got %d", callCount)
	}
}

func TestCreateConversationReturnsUserMissingWhenParentUserDoesNotExist(t *testing.T) {
	store := newTestStore(t)

	_, err := store.createConversation(999999)
	if !errors.Is(err, errStoreUserMissing) {
		t.Fatalf("expected errStoreUserMissing, got %v", err)
	}
}

func TestSaveConversationMessageReturnsConversationMissingWhenParentConversationDeleted(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "missing-conversation@example.com")
	conversationID, err := store.createConversation(user.ID)
	if err != nil {
		t.Fatalf("createConversation: %v", err)
	}
	if err := store.deleteConversation(user.ID, conversationID); err != nil {
		t.Fatalf("deleteConversation: %v", err)
	}

	err = store.saveConversationMessage(user.ID, conversationID, "user", "hello", "", 0, 0)
	if !errors.Is(err, errStoreConversationMissing) {
		t.Fatalf("expected errStoreConversationMissing, got %v", err)
	}
}

func TestResolveConversationRouteIsOwnerScoped(t *testing.T) {
	store := newTestStore(t)
	owner := mustCreateUser(t, store, "route-owner@example.com")
	other := mustCreateUser(t, store, "route-other@example.com")

	conversationID, err := store.createConversation(owner.ID)
	if err != nil {
		t.Fatalf("createConversation: %v", err)
	}
	conversations, err := store.listConversations(owner.ID)
	if err != nil {
		t.Fatalf("listConversations: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("expected one conversation, got %d", len(conversations))
	}
	publicID := conversations[0].PublicID

	summary, ok, err := store.resolveConversationRoute(owner.ID, publicID)
	if err != nil {
		t.Fatalf("resolveConversationRoute owner: %v", err)
	}
	if !ok || summary.ID != conversationID || summary.PublicID != publicID {
		t.Fatalf("unexpected owner route resolution: ok=%v summary=%+v", ok, summary)
	}

	summary, ok, err = store.resolveConversationRoute(other.ID, publicID)
	if err != nil {
		t.Fatalf("resolveConversationRoute other: %v", err)
	}
	if ok {
		t.Fatalf("expected non-owner route resolution to be inaccessible, got %+v", summary)
	}
}

func TestOpenChatStoreRecoversFromIncompatibleLegacySchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "chat_history.db")

	db, err := sql.Open("sqlite3", "file:"+dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE user_profile (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`); err != nil {
		_ = db.Close()
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	store, err := openChatStore(dbPath)
	if err != nil {
		t.Fatalf("openChatStore recovery: %v", err)
	}
	defer store.close()

	matches, err := filepath.Glob(dbPath + ".incompatible-*.bak")
	if err != nil {
		t.Fatalf("glob backup: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one backup file, got %v", matches)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected recreated db at %q: %v", dbPath, err)
	}

	userID, err := store.createUser("recover@example.com", "hash", "Recover")
	if err != nil {
		t.Fatalf("createUser after recovery: %v", err)
	}
	if userID <= 0 {
		t.Fatalf("expected valid recovered user id, got %d", userID)
	}
}

func TestOpenChatStoreCreatesMissingParentDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "missing", "runtime", "chat_history.db")
	dbDir := filepath.Dir(dbPath)

	if _, err := os.Stat(dbDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing db directory before open, stat err=%v", err)
	}

	store, err := openChatStore(dbPath)
	if err != nil {
		t.Fatalf("openChatStore create parent dir: %v", err)
	}
	defer store.close()

	if _, err := os.Stat(dbDir); err != nil {
		t.Fatalf("expected db directory to exist after open, got %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db file to exist after open, got %v", err)
	}
}
