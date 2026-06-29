package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/server/provider"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// TestServerConstructionAndHelperBranches verifies server construction and helper fallback branches.
func TestServerConstructionAndHelperBranches(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "helper-branches@example.com")

	parseServer := parseNewChatServiceServer("", "", "", "missing-model", parseStore, parseNewTestLogger(), "all")
	if parseServer.defaultModel == "" || parseServer.defaultModel == "missing-model" {
		parseT.Fatalf("expected default model fallback, got %q", parseServer.defaultModel)
	}
	if parseServer.memoryExtractionModel == "" {
		parseT.Fatal("expected memory extraction model to be populated")
	}
	if parseServer.authManager == nil {
		parseT.Fatal("expected auth manager to be initialized")
	}

	parseCatalog := provider.Catalog{DefaultModel: modelGPT54Mini}
	parseDefaultOpenAI := parseSelectRuntimeProvider("openai", "", nil, parseCatalog)
	if parseDefaultOpenAI == nil || parseDefaultOpenAI.ParseID() != "openai" {
		parseT.Fatalf("expected default openai provider, got %#v", parseDefaultOpenAI)
	}
	if parseUnknownWithKey := parseSelectRuntimeProvider("unknown", "key", nil, parseCatalog); parseUnknownWithKey != nil {
		parseT.Fatalf("expected unknown provider with key to be nil, got %#v", parseUnknownWithKey)
	}

	if parseDisplayName := parseServer.parseDisplayNameForUser(0, "helper-branches@example.com"); parseDisplayName != "helper-branches" {
		parseT.Fatalf("display name fallback = %q", parseDisplayName)
	}
	if parseDisplayName := parseServer.parseDisplayNameForUser(parseUser.ID, parseUser.Email); parseDisplayName != "Demo User" {
		parseT.Fatalf("display name from store = %q", parseDisplayName)
	}
	parseStore.parseClose()
	if parseDisplayName := parseServer.parseDisplayNameForUser(parseUser.ID, parseUser.Email); parseDisplayName != "helper-branches" {
		parseT.Fatalf("display name closed-store fallback = %q", parseDisplayName)
	}

	for parseInput, parseWant := range map[string]string{
		" LOW ":  "low",
		"MEDIUM": "medium",
		"High":   "high",
		"other":  defaultThinkingEffort,
		"   ":    defaultThinkingEffort,
	} {
		if parseGot := parseNormalizeSelectedThinkingEffort(parseInput); parseGot != parseWant {
			parseT.Fatalf("parseNormalizeSelectedThinkingEffort(%q) = %q, want %q", parseInput, parseGot, parseWant)
		}
	}
}

// TestConversationSessionHelperBranches verifies ownership, error, and creation branches for conversation sessions.
func TestConversationSessionHelperBranches(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "session-owner@example.com")
	parseOtherUser := parseMustCreateUser(parseT, parseStore, "session-other@example.com")
	parseConversationID, parseErr := parseStore.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation: %v", parseErr)
	}

	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())

	parseResolvedConversationID, parseSavedCount, parseErr := parseServer.parseLoadOrCreateConversationSession("peer-owned", parseUser.ID, parseConversationID, 4)
	if parseErr != nil || parseResolvedConversationID != parseConversationID || parseSavedCount != 4 {
		parseT.Fatalf("owned conversation session = (%d, %d, %v)", parseResolvedConversationID, parseSavedCount, parseErr)
	}
	if parseSession := parseServer.sessions["peer-owned"]; parseSession == nil || parseSession.savedMessageCount != 4 {
		parseT.Fatalf("expected saved session state, got %#v", parseSession)
	}

	if _, _, parseErr2 := parseServer.parseLoadOrCreateConversationSession("peer-other", parseOtherUser.ID, parseConversationID, 1); status.Code(parseErr2) != codes.NotFound {
		parseT.Fatalf("expected not found for unowned conversation, got %v", status.Code(parseErr2))
	}

	parseNewConversationID, parseNewSavedCount, parseErr3 := parseServer.parseLoadOrCreateConversationSession("peer-new", parseUser.ID, 0, 9)
	if parseErr3 != nil || parseNewConversationID == 0 || parseNewSavedCount != 0 {
		parseT.Fatalf("new conversation session = (%d, %d, %v)", parseNewConversationID, parseNewSavedCount, parseErr3)
	}

	parseStore.parseClose()
	if _, _, parseErr4 := parseServer.parseLoadOrCreateConversationSession("peer-closed", parseUser.ID, parseConversationID, 0); parseErr4 == nil {
		parseT.Fatal("expected closed store session lookup to fail")
	}
	if _, _, parseErr5 := parseServer.parseLoadOrCreateConversationSession("peer-create-closed", parseUser.ID, 0, 0); parseErr5 == nil {
		parseT.Fatal("expected closed store conversation creation to fail")
	}
}

// TestAuthSessionRPCBranchesCoverUnavailableAuthAndLogout verifies auth-unavailable and logout cleanup branches.
func TestAuthSessionRPCBranchesCoverUnavailableAuthAndLogout(parseT *testing.T) {
	parseServer := &chatServer{
		logger:    parseNewTestLogger(),
		sessions:  map[string]*sessionState{"peer-auth": {userID: 7, conversationID: 9}},
		authUsers: map[string]authUser{"peer-auth": {ID: 7, Email: "logout@example.com"}},
	}

	if _, parseErr := parseServer.Signup(context.Background(), nil); status.Code(parseErr) != codes.Unavailable {
		parseT.Fatalf("expected signup auth-unavailable error, got %v", status.Code(parseErr))
	}
	if _, parseErr := parseServer.Login(context.Background(), nil); status.Code(parseErr) != codes.Unavailable {
		parseT.Fatalf("expected login auth-unavailable error, got %v", status.Code(parseErr))
	}
	if _, parseErr := parseServer.RefreshSession(context.Background(), &emptypb.Empty{}); status.Code(parseErr) != codes.Unavailable {
		parseT.Fatalf("expected refresh auth-unavailable error, got %v", status.Code(parseErr))
	}

	parsePeerCtx := parseNewAuthenticatedContext("peer-auth")
	if _, parseErr := parseServer.Logout(parsePeerCtx, &emptypb.Empty{}); parseErr != nil {
		parseT.Fatalf("Logout(peer): %v", parseErr)
	}
	if _, parseFound := parseServer.sessions["peer-auth"]; parseFound {
		parseT.Fatalf("expected logout to clear peer session, got %#v", parseServer.sessions)
	}
	if _, parseFound := parseServer.authUsers["peer-auth"]; parseFound {
		parseT.Fatalf("expected logout to clear peer auth user cache, got %#v", parseServer.authUsers)
	}
	if _, parseErr := parseServer.Logout(context.Background(), &emptypb.Empty{}); parseErr != nil {
		parseT.Fatalf("Logout(no peer): %v", parseErr)
	}
}

// TestMemoryExtractionFailureBranches verifies provider-resolution, provider-failure, and save-failure extraction branches.
func TestMemoryExtractionFailureBranches(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "memory-failures@example.com")

	parseMissingModelProvider := parseNewFakeProvider()
	parseMissingModelServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(parseMissingModelProvider),
		defaultModel:          modelGPT54Mini,
		store:                 parseStore,
		logger:                parseNewTestLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: "missing-model",
	}
	parseMissingModelServer.parseExtractAndStoreUserMemories(context.Background(), parseUser.ID, "Remember this.")
	parseMemories, parseErr := parseStore.parseListUserMemories(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListUserMemories missing-model: %v", parseErr)
	}
	if len(parseMemories) != 0 {
		parseT.Fatalf("expected no stored memories on provider resolve failure, got %+v", parseMemories)
	}

	parseFailingProvider := parseNewFakeProvider()
	parseFailingProvider.extractUserMemories = func(context.Context, provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
		return nil, errors.New("extract failed")
	}
	parseFailingServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(parseFailingProvider),
		defaultModel:          modelGPT54Mini,
		store:                 parseStore,
		logger:                parseNewTestLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}
	parseFailingServer.parseExtractAndStoreUserMemories(context.Background(), parseUser.ID, "Remember this too.")
	parseMemories, parseErr = parseStore.parseListUserMemories(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListUserMemories failing-provider: %v", parseErr)
	}
	if len(parseMemories) != 0 {
		parseT.Fatalf("expected no stored memories on extraction failure, got %+v", parseMemories)
	}

	parseSaveFailureProvider := parseNewFakeProvider()
	parseSaveFailureProvider.extractUserMemories = func(context.Context, provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
		return []provider.UserMemoryCandidate{{
			Category:        "preference",
			Summary:         "Prefers concise answers",
			Detail:          "Asked for concise replies",
			UsefulnessScore: 90,
			ConfidenceScore: 0.9,
			RubricReason:    "stable preference",
		}}, nil
	}
	var parseLogs bytes.Buffer
	parseSaveFailureServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(parseSaveFailureProvider),
		defaultModel:          modelGPT54Mini,
		store:                 parseStore,
		logger:                slog.New(slog.NewTextHandler(&parseLogs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}
	parseStore.parseClose()
	parseSaveFailureServer.parseExtractAndStoreUserMemories(context.Background(), parseUser.ID, "Remember that I prefer concise replies.")
	parseLogOutput := parseLogs.String()
	if !strings.Contains(parseLogOutput, "memory extraction save failed") || !strings.Contains(parseLogOutput, "save_failure_count=1") {
		parseT.Fatalf("expected save-failure logs, got:\n%s", parseLogOutput)
	}
}
