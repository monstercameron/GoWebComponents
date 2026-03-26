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

func TestStoreConversationAndPreferenceLifecycle(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "demo@example.com")

	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	if parseErr2 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "user", "Hello", "", 0, 0); parseErr2 != nil {
		parseT.Fatalf("saveConversationMessage user: %v", parseErr2)
	}
	if parseErr3 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "assistant", "Hi there", modelGPT54Mini, 11, 7); parseErr3 != nil {
		parseT.Fatalf("saveConversationMessage assistant: %v", parseErr3)
	}
	if parseErr4 := store.parseSaveConversationTitle(parseUser.ID, parseConversationID, "Greeting thread"); parseErr4 != nil {
		parseT.Fatalf("saveConversationTitle: %v", parseErr4)
	}

	parseOwned, parseErr := store.parseConversationOwnedByUser(parseUser.ID, parseConversationID)
	if parseErr != nil {
		parseT.Fatalf("conversationOwnedByUser: %v", parseErr)
	}
	if !parseOwned {
		parseT.Fatal("expected conversation to belong to user")
	}

	parseConversations, parseErr := store.parseListConversations(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations: %v", parseErr)
	}
	if len(parseConversations) != 1 {
		parseT.Fatalf("expected one conversation, got %d", len(parseConversations))
	}
	if parseConversations[0].PublicID == "" {
		parseT.Fatal("expected conversation public_id to be populated")
	}
	if _, parseErr5 := uuid.Parse(parseConversations[0].PublicID); parseErr5 != nil {
		parseT.Fatalf("expected conversation public_id to be a UUID, got %q: %v", parseConversations[0].PublicID, parseErr5)
	}
	if parseConversations[0].Preview != "Greeting thread" {
		parseT.Fatalf("expected title-backed preview, got %q", parseConversations[0].Preview)
	}

	parseMessages, parseErr := store.parseLoadConversation(parseUser.ID, parseConversationID)
	if parseErr != nil {
		parseT.Fatalf("loadConversation: %v", parseErr)
	}
	if len(parseMessages) != 2 {
		parseT.Fatalf("expected two messages, got %d", len(parseMessages))
	}
	if parseMessages[1].ModelID != modelGPT54Mini || parseMessages[1].PromptTokens != 11 || parseMessages[1].CompletionTokens != 7 {
		parseT.Fatalf("assistant message metadata mismatch: %+v", parseMessages[1])
	}

	parseUpdatedAt := time.Now().Unix()
	if parseErr6 := store.setUserName(parseUser.ID, "Renamed User", parseUpdatedAt); parseErr6 != nil {
		parseT.Fatalf("setUserName: %v", parseErr6)
	}
	parseName, parseGotUpdatedAt, parseErr := store.getUserName(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("getUserName: %v", parseErr)
	}
	if parseName != "Renamed User" || parseGotUpdatedAt != parseUpdatedAt {
		parseT.Fatalf("unexpected user name state: name=%q updatedAt=%d", parseName, parseGotUpdatedAt)
	}

	if parseErr7 := store.setSelectedModel(parseUser.ID, modelGPT54); parseErr7 != nil {
		parseT.Fatalf("setSelectedModel: %v", parseErr7)
	}
	if parseErr8 := store.setSelectedTone(parseUser.ID, "professional"); parseErr8 != nil {
		parseT.Fatalf("setSelectedTone: %v", parseErr8)
	}
	if parseErr9 := store.setSelectedThinkingEnabled(parseUser.ID, false); parseErr9 != nil {
		parseT.Fatalf("setSelectedThinkingEnabled: %v", parseErr9)
	}
	if parseErr10 := store.setSelectedThinkingEffort(parseUser.ID, "high"); parseErr10 != nil {
		parseT.Fatalf("setSelectedThinkingEffort: %v", parseErr10)
	}
	if parseErr11 := store.setSelectedSystemPrompt(parseUser.ID, "Always answer with short bullet points."); parseErr11 != nil {
		parseT.Fatalf("setSelectedSystemPrompt: %v", parseErr11)
	}
	if parseErr12 := store.parseUpsertUserMemory(parseUser.ID, userMemoryRow{
		Key:             "preference-editor",
		Category:        "preference",
		Summary:         "Prefers Neovim",
		Detail:          "Uses Neovim daily for coding",
		SourceMessage:   "I use Neovim for most of my work.",
		UsefulnessScore: 84,
		ConfidenceScore: 0.91,
		RubricReason:    "Stable tooling preference",
	}); parseErr12 != nil {
		parseT.Fatalf("upsertUserMemory: %v", parseErr12)
	}

	parseSelectedModel, parseErr := store.getSelectedModel(parseUser.ID, modelGPT54Mini)
	if parseErr != nil || parseSelectedModel != modelGPT54 {
		parseT.Fatalf("getSelectedModel: model=%q err=%v", parseSelectedModel, parseErr)
	}
	parseSelectedTone, parseErr := store.getSelectedTone(parseUser.ID, defaultToneID)
	if parseErr != nil || parseSelectedTone != "professional" {
		parseT.Fatalf("getSelectedTone: tone=%q err=%v", parseSelectedTone, parseErr)
	}
	parseThinkingEnabled, parseErr := store.getSelectedThinkingEnabled(parseUser.ID, true)
	if parseErr != nil || parseThinkingEnabled {
		parseT.Fatalf("getSelectedThinkingEnabled: enabled=%v err=%v", parseThinkingEnabled, parseErr)
	}
	parseThinkingEffort, parseErr := store.getSelectedThinkingEffort(parseUser.ID, defaultThinkingEffort)
	if parseErr != nil || parseThinkingEffort != "high" {
		parseT.Fatalf("getSelectedThinkingEffort: effort=%q err=%v", parseThinkingEffort, parseErr)
	}
	parseCustomSystemPrompt, parseErr := store.getSelectedSystemPrompt(parseUser.ID, "")
	if parseErr != nil || parseCustomSystemPrompt != "Always answer with short bullet points." {
		parseT.Fatalf("getSelectedSystemPrompt: prompt=%q err=%v", parseCustomSystemPrompt, parseErr)
	}
	parseMemories, parseErr := store.parseListUserMemories(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listUserMemories: %v", parseErr)
	}
	if len(parseMemories) != 1 || parseMemories[0].Summary != "Prefers Neovim" {
		parseT.Fatalf("unexpected stored memories: %+v", parseMemories)
	}

	if parseErr13 := store.parseDeleteConversation(parseUser.ID, parseConversationID); parseErr13 != nil {
		parseT.Fatalf("deleteConversation: %v", parseErr13)
	}
	parseRemaining, parseErr := store.parseListConversations(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations after delete: %v", parseErr)
	}
	if len(parseRemaining) != 0 {
		parseT.Fatalf("expected zero conversations after delete, got %d", len(parseRemaining))
	}
}

func TestStoreFallbacksAndUniqueness(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUserID, parseErr := store.parseCreateUser("fallback@example.com", "hash", "")
	if parseErr != nil {
		parseT.Fatalf("createUser: %v", parseErr)
	}
	if _, parseErr2 := store.parseCreateUser("fallback@example.com", "hash", ""); parseErr2 != errUserAlreadyExists {
		parseT.Fatalf("expected duplicate user error, got %v", parseErr2)
	}

	parseName, parseUpdatedAt, parseErr := store.getUserName(parseUserID + 100)
	if parseErr != nil {
		parseT.Fatalf("getUserName fallback: %v", parseErr)
	}
	if parseName != "User" || parseUpdatedAt != 0 {
		parseT.Fatalf("unexpected fallback profile: name=%q updatedAt=%d", parseName, parseUpdatedAt)
	}

	parseSelectedModel, parseErr := store.getSelectedModel(parseUserID+100, modelGPT54Mini)
	if parseErr != nil || parseSelectedModel != modelGPT54Mini {
		parseT.Fatalf("fallback selected model mismatch: model=%q err=%v", parseSelectedModel, parseErr)
	}
	parseSelectedTone, parseErr := store.getSelectedTone(parseUserID+100, defaultToneID)
	if parseErr != nil || parseSelectedTone != defaultToneID {
		parseT.Fatalf("fallback selected tone mismatch: tone=%q err=%v", parseSelectedTone, parseErr)
	}
	parseThinkingEnabled, parseErr := store.getSelectedThinkingEnabled(parseUserID+100, true)
	if parseErr != nil || !parseThinkingEnabled {
		parseT.Fatalf("fallback thinking enabled mismatch: enabled=%v err=%v", parseThinkingEnabled, parseErr)
	}
	parseThinkingEffort, parseErr := store.getSelectedThinkingEffort(parseUserID+100, defaultThinkingEffort)
	if parseErr != nil || parseThinkingEffort != defaultThinkingEffort {
		parseT.Fatalf("fallback thinking effort mismatch: effort=%q err=%v", parseThinkingEffort, parseErr)
	}
	parseCustomSystemPrompt, parseErr := store.getSelectedSystemPrompt(parseUserID+100, "")
	if parseErr != nil || parseCustomSystemPrompt != "" {
		parseT.Fatalf("fallback custom system prompt mismatch: prompt=%q err=%v", parseCustomSystemPrompt, parseErr)
	}

	parseDerivedName, parseDerivedUpdatedAt, parseErr := store.getUserName(parseUserID)
	if parseErr != nil {
		parseT.Fatalf("getUserName derived: %v", parseErr)
	}
	if parseDerivedName != "fallback" || parseDerivedUpdatedAt == 0 {
		parseT.Fatalf("expected default display name derived from email, got name=%q updatedAt=%d", parseDerivedName, parseDerivedUpdatedAt)
	}

	parseOwned, parseErr := store.parseConversationOwnedByUser(parseUserID, parseUserID+999)
	if parseErr != nil {
		parseT.Fatalf("conversationOwnedByUser false branch: %v", parseErr)
	}
	if parseOwned {
		parseT.Fatal("expected unrelated conversation to not belong to user")
	}
}

func TestCreateConversationRetriesPublicIDConflicts(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "uuid-retry@example.com")

	parseOriginalGenerator := newConversationPublicID
	defer func() { newConversationPublicID = parseOriginalGenerator }()

	parseCollisionID := uuid.NewString()
	parseRecoveredID := uuid.NewString()
	parseCallCount := 0
	newConversationPublicID = func() string {
		parseCallCount++
		if parseCallCount <= 2 {
			return parseCollisionID
		}
		return parseRecoveredID
	}

	parseFirstConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation first: %v", parseErr)
	}
	parseSecondConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation second: %v", parseErr)
	}
	if parseFirstConversationID == parseSecondConversationID {
		parseT.Fatalf("expected distinct conversation rows, got %d and %d", parseFirstConversationID, parseSecondConversationID)
	}

	parseConversations, parseErr := store.parseListConversations(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations: %v", parseErr)
	}
	if len(parseConversations) != 2 {
		parseT.Fatalf("expected two conversations, got %d", len(parseConversations))
	}

	parsePublicIDs := map[string]bool{}
	for _, parseConversation := range parseConversations {
		if parseConversation.PublicID == "" {
			parseT.Fatal("expected non-empty public_id")
		}
		if _, parseErr2 := uuid.Parse(parseConversation.PublicID); parseErr2 != nil {
			parseT.Fatalf("expected parseable UUID, got %q: %v", parseConversation.PublicID, parseErr2)
		}
		if parsePublicIDs[parseConversation.PublicID] {
			parseT.Fatalf("expected unique public_ids, got duplicate %q", parseConversation.PublicID)
		}
		parsePublicIDs[parseConversation.PublicID] = true
	}
	if !parsePublicIDs[parseCollisionID] {
		parseT.Fatalf("expected initial conversation to keep first generated UUID %q", parseCollisionID)
	}
	if !parsePublicIDs[parseRecoveredID] {
		parseT.Fatalf("expected retry path to use fallback UUID %q", parseRecoveredID)
	}
	if parseCallCount < 3 {
		parseT.Fatalf("expected generator to be called at least 3 times, got %d", parseCallCount)
	}
}

func TestCreateConversationReturnsUserMissingWhenParentUserDoesNotExist(parseT *testing.T) {
	store := parseNewTestStore(parseT)

	_, parseErr := store.parseCreateConversation(999999)
	if !errors.Is(parseErr, errStoreUserMissing) {
		parseT.Fatalf("expected errStoreUserMissing, got %v", parseErr)
	}
}

func TestSaveConversationMessageReturnsConversationMissingWhenParentConversationDeleted(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "missing-conversation@example.com")
	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	if parseErr2 := store.parseDeleteConversation(parseUser.ID, parseConversationID); parseErr2 != nil {
		parseT.Fatalf("deleteConversation: %v", parseErr2)
	}

	parseErr = store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "user", "hello", "", 0, 0)
	if !errors.Is(parseErr, errStoreConversationMissing) {
		parseT.Fatalf("expected errStoreConversationMissing, got %v", parseErr)
	}
}

func TestResolveConversationRouteIsOwnerScoped(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseOwner := parseMustCreateUser(parseT, store, "route-owner@example.com")
	parseOther := parseMustCreateUser(parseT, store, "route-other@example.com")

	parseConversationID, parseErr := store.parseCreateConversation(parseOwner.ID)
	if parseErr != nil {
		parseT.Fatalf("createConversation: %v", parseErr)
	}
	parseConversations, parseErr := store.parseListConversations(parseOwner.ID)
	if parseErr != nil {
		parseT.Fatalf("listConversations: %v", parseErr)
	}
	if len(parseConversations) != 1 {
		parseT.Fatalf("expected one conversation, got %d", len(parseConversations))
	}
	parsePublicID := parseConversations[0].PublicID

	parseSummary, parseOk, parseErr := store.parseResolveConversationRoute(parseOwner.ID, parsePublicID)
	if parseErr != nil {
		parseT.Fatalf("resolveConversationRoute owner: %v", parseErr)
	}
	if !parseOk || parseSummary.ID != parseConversationID || parseSummary.PublicID != parsePublicID {
		parseT.Fatalf("unexpected owner route resolution: ok=%v summary=%+v", parseOk, parseSummary)
	}

	parseSummary, parseOk, parseErr = store.parseResolveConversationRoute(parseOther.ID, parsePublicID)
	if parseErr != nil {
		parseT.Fatalf("resolveConversationRoute other: %v", parseErr)
	}
	if parseOk {
		parseT.Fatalf("expected non-owner route resolution to be inaccessible, got %+v", parseSummary)
	}
}

func TestOpenChatStoreRecoversFromIncompatibleLegacySchema(parseT *testing.T) {
	parseDbPath := filepath.Join(parseT.TempDir(), "chat_history.db")

	parseDb, parseErr := sql.Open("sqlite3", "file:"+parseDbPath)
	if parseErr != nil {
		parseT.Fatalf("sql.Open: %v", parseErr)
	}
	if _, parseErr2 := parseDb.Exec(`CREATE TABLE user_profile (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`); parseErr2 != nil {
		_ = parseDb.Close()
		parseT.Fatalf("create legacy schema: %v", parseErr2)
	}
	if parseErr3 := parseDb.Close(); parseErr3 != nil {
		parseT.Fatalf("close legacy db: %v", parseErr3)
	}

	store, parseErr := parseOpenChatStore(parseDbPath)
	if parseErr != nil {
		parseT.Fatalf("openChatStore recovery: %v", parseErr)
	}
	defer store.parseClose()

	parseMatches, parseErr := filepath.Glob(parseDbPath + ".incompatible-*.bak")
	if parseErr != nil {
		parseT.Fatalf("glob backup: %v", parseErr)
	}
	if len(parseMatches) != 1 {
		parseT.Fatalf("expected one backup file, got %v", parseMatches)
	}
	if _, parseErr4 := os.Stat(parseDbPath); parseErr4 != nil {
		parseT.Fatalf("expected recreated db at %q: %v", parseDbPath, parseErr4)
	}

	parseUserID, parseErr := store.parseCreateUser("recover@example.com", "hash", "Recover")
	if parseErr != nil {
		parseT.Fatalf("createUser after recovery: %v", parseErr)
	}
	if parseUserID <= 0 {
		parseT.Fatalf("expected valid recovered user id, got %d", parseUserID)
	}
}

func TestOpenChatStoreCreatesMissingParentDirectory(parseT *testing.T) {
	parseDbPath := filepath.Join(parseT.TempDir(), "missing", "runtime", "chat_history.db")
	parseDbDir := filepath.Dir(parseDbPath)

	if _, parseErr := os.Stat(parseDbDir); !errors.Is(parseErr, os.ErrNotExist) {
		parseT.Fatalf("expected missing db directory before open, stat err=%v", parseErr)
	}

	store, parseErr2 := parseOpenChatStore(parseDbPath)
	if parseErr2 != nil {
		parseT.Fatalf("openChatStore create parent dir: %v", parseErr2)
	}
	defer store.parseClose()

	if _, parseErr3 := os.Stat(parseDbDir); parseErr3 != nil {
		parseT.Fatalf("expected db directory to exist after open, got %v", parseErr3)
	}
	if _, parseErr4 := os.Stat(parseDbPath); parseErr4 != nil {
		parseT.Fatalf("expected db file to exist after open, got %v", parseErr4)
	}
}
