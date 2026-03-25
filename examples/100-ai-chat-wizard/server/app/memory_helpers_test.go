package app

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
)

func TestCustomPromptAndMemoryHelperFunctions(t *testing.T) {
	tooLongPrompt := "  " + string(make([]rune, maxCustomSystemPromptRunes+25))
	tooLongRunes := []rune(tooLongPrompt)
	for index := range tooLongRunes {
		if tooLongRunes[index] == 0 {
			tooLongRunes[index] = 'x'
		}
	}
	normalizedPrompt := normalizeCustomSystemPrompt(string(tooLongRunes))
	if len([]rune(normalizedPrompt)) != maxCustomSystemPromptRunes {
		t.Fatalf("expected custom system prompt to be truncated to %d runes, got %d", maxCustomSystemPromptRunes, len([]rune(normalizedPrompt)))
	}
	if normalizeCustomSystemPrompt("   ") != "" {
		t.Fatal("expected blank custom system prompt to normalize to empty")
	}

	if got := normalizeUserMemoryCategory(" Preference "); got != "preference" {
		t.Fatalf("unexpected normalized category: %q", got)
	}
	if got := normalizeUserMemoryCategory("unknown"); got != "other" {
		t.Fatalf("expected unknown category to normalize to other, got %q", got)
	}
	if got := normalizeUserMemoryKey("", "preference", "Uses Neovim daily!"); got != "preference-uses-neovim-daily" {
		t.Fatalf("unexpected derived memory key: %q", got)
	}
	if got := normalizeUserMemoryKey("  Team/Role  ", "profile", "ignored"); got != "team-role" {
		t.Fatalf("unexpected explicit memory key normalization: %q", got)
	}
	if got := clampUsefulnessScore(-4); got != 0 {
		t.Fatalf("expected low usefulness score clamp to 0, got %d", got)
	}
	if got := clampUsefulnessScore(101); got != 100 {
		t.Fatalf("expected high usefulness score clamp to 100, got %d", got)
	}
	if got := clampConfidenceScore(-0.1); got != 0 {
		t.Fatalf("expected low confidence clamp to 0, got %v", got)
	}
	if got := clampConfidenceScore(1.4); got != 1 {
		t.Fatalf("expected high confidence clamp to 1, got %v", got)
	}

	filtered := filterUsefulUserMemories([]provider.UserMemoryCandidate{
		{Category: "Preference", Summary: " Prefers concise answers ", Detail: " Prefers concise answers ", UsefulnessScore: 80, ConfidenceScore: 0.8, RubricReason: " Stable preference "},
		{Category: "constraint", Summary: "", UsefulnessScore: 90, ConfidenceScore: 0.9},
		{Category: "project", Summary: "Low confidence", UsefulnessScore: 90, ConfidenceScore: 0.2},
		{Category: "profile", Summary: "Too low usefulness", UsefulnessScore: 20, ConfidenceScore: 0.9},
	})
	if len(filtered) != 1 {
		t.Fatalf("expected one useful memory candidate, got %+v", filtered)
	}
	if filtered[0].Category != "preference" || filtered[0].Key != "preference-prefers-concise-answers" {
		t.Fatalf("unexpected normalized useful memory candidate: %+v", filtered[0])
	}

	memoryBlock := buildUserMemoryPromptBlock([]userMemoryRow{
		{Summary: "Prefers concise answers", Detail: "Prefers concise answers"},
		{Summary: "Uses Neovim", Detail: "Daily editor"},
	})
	if memoryBlock == "" {
		t.Fatal("expected memory prompt block to be built")
	}
	if contains := memoryBlock == "- Prefers concise answers\n- Uses Neovim (Daily editor)"; !contains {
		t.Fatalf("unexpected memory prompt block: %q", memoryBlock)
	}

	longMemories := make([]userMemoryRow, 0, maxInjectedUserMemoryCount+3)
	for index := 0; index < maxInjectedUserMemoryCount+3; index++ {
		longMemories = append(longMemories, userMemoryRow{Summary: "Memory", Detail: "Detail"})
	}
	trimmedBlock := buildUserMemoryPromptBlock(longMemories)
	lineCount := 0
	for _, currentRune := range trimmedBlock {
		if currentRune == '\n' {
			lineCount++
		}
	}
	if lineCount+1 > maxInjectedUserMemoryCount {
		t.Fatalf("expected memory prompt block to cap items at %d, got %d lines", maxInjectedUserMemoryCount, lineCount+1)
	}
	if len([]rune(trimmedBlock)) > maxInjectedUserMemoryRunes {
		t.Fatalf("expected memory prompt block to cap runes at %d, got %d", maxInjectedUserMemoryRunes, len([]rune(trimmedBlock)))
	}
}

func TestExtractAndStoreUserMemoriesBranches(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "memory-helper@example.com")
	fake := newFakeProvider()
	callCount := 0
	fake.extractUserMemories = func(_ context.Context, req provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
		callCount++
		if req.Model != modelGPT54 {
			t.Fatalf("expected extraction model %q, got %q", modelGPT54, req.Model)
		}
		if req.UserMessage != "Remember that I like direct answers." {
			t.Fatalf("unexpected extraction message: %q", req.UserMessage)
		}
		return []provider.UserMemoryCandidate{
			{Category: "Preference", Summary: "Likes direct answers", Detail: "Asked for direct answers", UsefulnessScore: 81, ConfidenceScore: 0.91, RubricReason: "Stable preference"},
			{Category: "project", Summary: "Ignore me", UsefulnessScore: 10, ConfidenceScore: 0.9},
		}, nil
	}

	server := &chatServer{
		providerRegistry:      provider.NewRegistry(fake),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                newTestLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}

	server.extractAndStoreUserMemories(user.ID, "Remember that I like direct answers.")
	deadline := time.Now().Add(2 * time.Second)
	for {
		memories, err := store.listUserMemories(user.ID)
		if err != nil {
			t.Fatalf("listUserMemories: %v", err)
		}
		if len(memories) == 1 {
			if memories[0].Category != "preference" || memories[0].Key != "preference-likes-direct-answers" {
				t.Fatalf("unexpected stored extracted memory: %+v", memories[0])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for extracted memory, stored=%d", len(memories))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if callCount != 1 {
		t.Fatalf("expected one extraction call, got %d", callCount)
	}

	queueFullServer := &chatServer{
		providerRegistry:      provider.NewRegistry(fake),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                newTestLogger(),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}
	queueFullServer.memoryExtractionSlots <- struct{}{}
	queueFullServer.extractAndStoreUserMemories(user.ID, "This should skip")
	if callCount != 1 {
		t.Fatalf("expected queue-full extraction skip, got %d calls", callCount)
	}

	server.extractAndStoreUserMemories(user.ID, "   ")
	if callCount != 1 {
		t.Fatalf("expected blank message extraction skip, got %d calls", callCount)
	}
}

func TestExtractAndStoreUserMemoriesLogsLifecycle(t *testing.T) {
	store := newTestStore(t)
	user := mustCreateUser(t, store, "memory-logs@example.com")
	fake := newFakeProvider()
	fake.extractUserMemories = func(_ context.Context, req provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
		if req.Model != modelGPT54 {
			t.Fatalf("expected extraction model %q, got %q", modelGPT54, req.Model)
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

	var logs bytes.Buffer
	server := &chatServer{
		providerRegistry:      provider.NewRegistry(fake),
		defaultModel:          modelGPT54Mini,
		store:                 store,
		logger:                slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		sessions:              map[string]*sessionState{},
		authUsers:             map[string]authUser{},
		memoryExtractionSlots: make(chan struct{}, 1),
		memoryExtractionModel: modelGPT54,
	}

	server.extractAndStoreUserMemories(user.ID, "Remember that I prefer concise answers.")

	logOutput := logs.String()
	if !strings.Contains(logOutput, "memory extraction started") {
		t.Fatalf("expected lifecycle start log, got logs:\n%s", logOutput)
	}
	if !strings.Contains(logOutput, "memory extraction completed") {
		t.Fatalf("expected lifecycle completion log, got logs:\n%s", logOutput)
	}
	if !strings.Contains(logOutput, "saved_count=1") {
		t.Fatalf("expected saved_count=1 in completion log, got logs:\n%s", logOutput)
	}
}
