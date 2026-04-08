package app

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/server/provider"
)

func TestCustomPromptAndMemoryHelperFunctions(parseT *testing.T) {
	parseTooLongPrompt := "  " + string(make([]rune, maxCustomSystemPromptRunes+25))
	parseTooLongRunes := []rune(parseTooLongPrompt)
	for parseIndex := range parseTooLongRunes {
		if parseTooLongRunes[parseIndex] == 0 {
			parseTooLongRunes[parseIndex] = 'x'
		}
	}
	parseNormalizedPrompt := parseNormalizeCustomSystemPrompt(string(parseTooLongRunes))
	if len([]rune(parseNormalizedPrompt)) != maxCustomSystemPromptRunes {
		parseT.Fatalf("expected custom system prompt to be truncated to %d runes, got %d", maxCustomSystemPromptRunes, len([]rune(parseNormalizedPrompt)))
	}
	if parseNormalizeCustomSystemPrompt("   ") != "" {
		parseT.Fatal("expected blank custom system prompt to normalize to empty")
	}

	if parseGot := parseNormalizeUserMemoryCategory(" Preference "); parseGot != "preference" {
		parseT.Fatalf("unexpected normalized category: %q", parseGot)
	}
	if parseGot2 := parseNormalizeUserMemoryCategory("unknown"); parseGot2 != "other" {
		parseT.Fatalf("expected unknown category to normalize to other, got %q", parseGot2)
	}
	if parseGot3 := parseNormalizeUserMemoryKey("", "preference", "Uses Neovim daily!"); parseGot3 != "preference-uses-neovim-daily" {
		parseT.Fatalf("unexpected derived memory key: %q", parseGot3)
	}
	if parseGot4 := parseNormalizeUserMemoryKey("  Team/Role  ", "profile", "ignored"); parseGot4 != "team-role" {
		parseT.Fatalf("unexpected explicit memory key normalization: %q", parseGot4)
	}
	if parseGot5 := parseClampUsefulnessScore(-4); parseGot5 != 0 {
		parseT.Fatalf("expected low usefulness score clamp to 0, got %d", parseGot5)
	}
	if parseGot6 := parseClampUsefulnessScore(101); parseGot6 != 100 {
		parseT.Fatalf("expected high usefulness score clamp to 100, got %d", parseGot6)
	}
	if parseGot7 := parseClampConfidenceScore(-0.1); parseGot7 != 0 {
		parseT.Fatalf("expected low confidence clamp to 0, got %v", parseGot7)
	}
	if parseGot8 := parseClampConfidenceScore(1.4); parseGot8 != 1 {
		parseT.Fatalf("expected high confidence clamp to 1, got %v", parseGot8)
	}

	parseFiltered := filterUsefulUserMemories([]provider.UserMemoryCandidate{
		{Category: "Preference", Summary: " Prefers concise answers ", Detail: " Prefers concise answers ", UsefulnessScore: 80, ConfidenceScore: 0.8, RubricReason: " Stable preference "},
		{Category: "constraint", Summary: "", UsefulnessScore: 90, ConfidenceScore: 0.9},
		{Category: "project", Summary: "Low confidence", UsefulnessScore: 90, ConfidenceScore: 0.2},
		{Category: "profile", Summary: "Too low usefulness", UsefulnessScore: 20, ConfidenceScore: 0.9},
	})
	if len(parseFiltered) != 1 {
		parseT.Fatalf("expected one useful memory candidate, got %+v", parseFiltered)
	}
	if parseFiltered[0].Category != "preference" || parseFiltered[0].Key != "preference-prefers-concise-answers" {
		parseT.Fatalf("unexpected normalized useful memory candidate: %+v", parseFiltered[0])
	}
	parseDedupedCandidates := filterUsefulUserMemories([]provider.UserMemoryCandidate{
		{Key: "pref-short-low", Category: "preference", Summary: "Prefers concise answers", Detail: "Asked for concise responses", UsefulnessScore: 72, ConfidenceScore: 0.82},
		{Key: "pref-short-high", Category: "preference", Summary: " Prefers   concise answers ", Detail: "Asked for concise responses", UsefulnessScore: 94, ConfidenceScore: 0.96},
	})
	if len(parseDedupedCandidates) != 1 {
		parseT.Fatalf("expected deduped memory candidates length 1, got %+v", parseDedupedCandidates)
	}
	if parseDedupedCandidates[0].UsefulnessScore != 94 || parseDedupedCandidates[0].Key != "pref-short-high" {
		parseT.Fatalf("expected strongest duplicate memory candidate to win, got %+v", parseDedupedCandidates[0])
	}

	parseMemoryBlock := buildUserMemoryPromptBlock([]userMemoryRow{
		{Summary: "Prefers concise answers", Detail: "Prefers concise answers"},
		{Summary: "Uses Neovim", Detail: "Daily editor"},
	})
	if parseMemoryBlock == "" {
		parseT.Fatal("expected memory prompt block to be built")
	}
	if isParseContains := parseMemoryBlock == "- Prefers concise answers\n- Uses Neovim (Daily editor)"; !isParseContains {
		parseT.Fatalf("unexpected memory prompt block: %q", parseMemoryBlock)
	}
	parseDedupedMemoryBlock := buildUserMemoryPromptBlock([]userMemoryRow{
		{Category: "preference", Summary: "Prefers concise answers", Detail: "Asked for concise responses"},
		{Category: "preference", Summary: " Prefers concise answers ", Detail: "Asked for concise responses"},
	})
	if parseDedupedMemoryBlock != "- Prefers concise answers (Asked for concise responses)" {
		parseT.Fatalf("expected deduped memory prompt block to keep one line, got %q", parseDedupedMemoryBlock)
	}

	parseLongMemories := make([]userMemoryRow, 0, maxInjectedUserMemoryCount+3)
	for parseIndex2 := 0; parseIndex2 < maxInjectedUserMemoryCount+3; parseIndex2++ {
		parseLongMemories = append(parseLongMemories, userMemoryRow{Summary: "Memory", Detail: "Detail"})
	}
	parseTrimmedBlock := buildUserMemoryPromptBlock(parseLongMemories)
	parseLineCount := 0
	for _, parseCurrentRune := range parseTrimmedBlock {
		if parseCurrentRune == '\n' {
			parseLineCount++
		}
	}
	if parseLineCount+1 > maxInjectedUserMemoryCount {
		parseT.Fatalf("expected memory prompt block to cap items at %d, got %d lines", maxInjectedUserMemoryCount, parseLineCount+1)
	}
	if len([]rune(parseTrimmedBlock)) > maxInjectedUserMemoryRunes {
		parseT.Fatalf("expected memory prompt block to cap runes at %d, got %d", maxInjectedUserMemoryRunes, len([]rune(parseTrimmedBlock)))
	}
}

func TestExtractAndStoreUserMemoriesBranches(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "memory-helper@example.com")
	parseFake := parseNewFakeProvider()
	parseCallCount := 0
	parseFake.extractUserMemories = func(_ context.Context, parseReq provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
		parseCallCount++
		if parseReq.Model != modelGPT54 {
			parseT.Fatalf("expected extraction model %q, got %q", modelGPT54, parseReq.Model)
		}
		if parseReq.UserMessage != "Remember that I like direct answers." {
			parseT.Fatalf("unexpected extraction message: %q", parseReq.UserMessage)
		}
		return []provider.UserMemoryCandidate{
			{Category: "Preference", Summary: "Likes direct answers", Detail: "Asked for direct answers", UsefulnessScore: 81, ConfidenceScore: 0.91, RubricReason: "Stable preference"},
			{Category: "project", Summary: "Ignore me", UsefulnessScore: 10, ConfidenceScore: 0.9},
		}, nil
	}

	parseServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(parseFake),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                parseNewTestLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}

	parseServer.parseExtractAndStoreUserMemories(context.Background(), parseUser.ID, "Remember that I like direct answers.")
	parseDeadline := time.Now().Add(2 * time.Second)
	for {
		parseMemories, parseErr := store.parseListUserMemories(parseUser.ID)
		if parseErr != nil {
			parseT.Fatalf("listUserMemories: %v", parseErr)
		}
		if len(parseMemories) == 1 {
			if parseMemories[0].Category != "preference" || parseMemories[0].Key != "preference-likes-direct-answers" {
				parseT.Fatalf("unexpected stored extracted memory: %+v", parseMemories[0])
			}
			break
		}
		if time.Now().After(parseDeadline) {
			parseT.Fatalf("timed out waiting for extracted memory, stored=%d", len(parseMemories))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if parseCallCount != 1 {
		parseT.Fatalf("expected one extraction call, got %d", parseCallCount)
	}

	parseQueueFullServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(parseFake),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                parseNewTestLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}
	parseQueueFullServer.memoryExtractionSlots <- struct{}{}
	parseQueueFullServer.parseExtractAndStoreUserMemories(context.Background(), parseUser.ID, "This should skip")
	if parseCallCount != 1 {
		parseT.Fatalf("expected queue-full extraction skip, got %d calls", parseCallCount)
	}

	parseServer.parseExtractAndStoreUserMemories(context.Background(), parseUser.ID, "   ")
	if parseCallCount != 1 {
		parseT.Fatalf("expected blank message extraction skip, got %d calls", parseCallCount)
	}
}

// TestExtractAndStoreUserMemoriesReusesExistingKeys verifies extraction dedupe reuses stable keys so user edits stay coherent.
func TestExtractAndStoreUserMemoriesReusesExistingKeys(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "memory-reuse@example.com")
	if parseErr := store.parseUpsertUserMemory(parseUser.ID, userMemoryRow{
		Key:             "manual-memory-key",
		Category:        "preference",
		Summary:         "Prefers concise answers",
		Detail:          "Asked for concise responses",
		SourceMessage:   "manual",
		UsefulnessScore: 77,
		ConfidenceScore: 0.78,
		RubricReason:    "seed",
	}); parseErr != nil {
		parseT.Fatalf("parseUpsertUserMemory seed: %v", parseErr)
	}

	parseFake := parseNewFakeProvider()
	parseFake.extractUserMemories = func(_ context.Context, _ provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
		return []provider.UserMemoryCandidate{
			{Key: "llm-memory-low", Category: "preference", Summary: "Prefers concise answers", Detail: "Asked for concise responses", UsefulnessScore: 81, ConfidenceScore: 0.81},
			{Key: "llm-memory-high", Category: "preference", Summary: " Prefers   concise answers ", Detail: "Asked for concise responses", UsefulnessScore: 95, ConfidenceScore: 0.95},
		}, nil
	}

	parseServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(parseFake),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                parseNewTestLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}
	parseServer.parseExtractAndStoreUserMemories(context.Background(), parseUser.ID, "Remember that I prefer concise answers.")

	parseMemories, parseErr := store.parseListUserMemories(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListUserMemories: %v", parseErr)
	}
	if len(parseMemories) != 1 {
		parseT.Fatalf("expected one deduped memory row, got %+v", parseMemories)
	}
	if parseMemories[0].Key != "manual-memory-key" {
		parseT.Fatalf("expected extraction to reuse existing key, got %+v", parseMemories[0])
	}
	if parseMemories[0].UsefulnessScore != 95 || parseMemories[0].ConfidenceScore != 0.95 {
		parseT.Fatalf("expected strongest duplicate candidate values to persist, got %+v", parseMemories[0])
	}
}

func TestExtractAndStoreUserMemoriesLogsLifecycle(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "memory-logs@example.com")
	parseFake := parseNewFakeProvider()
	parseFake.extractUserMemories = func(_ context.Context, parseReq provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
		if parseReq.Model != modelGPT54 {
			parseT.Fatalf("expected extraction model %q, got %q", modelGPT54, parseReq.Model)
		}
		return []provider.UserMemoryCandidate{
			{
				Category:        "preference",
				Summary:         "Prefers short answers",
				Detail:          "Asked for concise responses",
				UsefulnessScore: 91,
				ConfidenceScore: 0.9,
				RubricReason:    "stable preference",
			},
		}, nil
	}

	var parseLogs bytes.Buffer
	parseServer := &chatServer{
		providerRegistry:      provider.ParseNewRegistry(parseFake),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                slog.New(slog.NewTextHandler(&parseLogs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}

	parseServer.parseExtractAndStoreUserMemories(context.Background(), parseUser.ID, "Remember that I prefer concise answers.")

	parseLogOutput := parseLogs.String()
	if !strings.Contains(parseLogOutput, "memory extraction started") {
		parseT.Fatalf("expected lifecycle start log, got logs:\n%s", parseLogOutput)
	}
	if !strings.Contains(parseLogOutput, "memory extraction completed") {
		parseT.Fatalf("expected lifecycle completion log, got logs:\n%s", parseLogOutput)
	}
	if !strings.Contains(parseLogOutput, "saved_count=1") {
		parseT.Fatalf("expected saved_count=1 in completion log, got logs:\n%s", parseLogOutput)
	}
}
