package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
)

type benchmarkProvider struct {
	model string
}

func (p benchmarkProvider) ID() string { return "benchmark" }

func (p benchmarkProvider) Available() bool { return true }

func (p benchmarkProvider) Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		ID:                 p.ID(),
		Label:              "Benchmark",
		AuthConfigured:     true,
		Available:          true,
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   false,
	}
}

func (p benchmarkProvider) DefaultModel() string { return p.model }

func (p benchmarkProvider) SupportsModel(model string) bool { return normalizeSelectedModelID(model) == p.model }

func (p benchmarkProvider) ModelOptions() []provider.ModelOption {
	capabilities := p.Capabilities(p.model)
	return []provider.ModelOption{{
		ID:           p.model,
		Label:        "Benchmark Model",
		Note:         "Synthetic benchmark provider",
		Capabilities: capabilities,
	}}
}

func (p benchmarkProvider) ModelMetadata(model string) (provider.ModelMetadata, bool) {
	if !p.SupportsModel(model) {
		return provider.ModelMetadata{}, false
	}
	capabilities := p.Capabilities(model)
	return provider.ModelMetadata{
		ID:                 p.model,
		DisplayName:        "Benchmark Model",
		ProviderID:         p.ID(),
		ProviderLabel:      "Benchmark",
		ProviderFamily:     "benchmark",
		Capabilities:       capabilities,
		StreamingSupported: true,
		ReasoningSupported: capabilities.SupportsThinking,
		OnboardingReady:    true,
	}, true
}

func (p benchmarkProvider) Capabilities(string) provider.ModelCapabilities {
	return provider.ModelCapabilities{
		ProviderID:       p.ID(),
		ProviderLabel:    "Benchmark",
		SupportsThinking: true,
	}
}

func (p benchmarkProvider) Health() provider.ProviderHealth {
	return provider.ProviderHealth{
		ProviderID: p.ID(),
		Status:     provider.ProviderHealthUnknown,
	}
}

func (p benchmarkProvider) CurrentRateLimits() provider.RateLimitSnapshot {
	return provider.RateLimitSnapshot{}
}

func (p benchmarkProvider) StreamChat(_ context.Context, _ provider.ChatRequest, emit func(provider.ChatEvent) error) (provider.ChatResult, error) {
	if err := emit(provider.ChatEvent{ThoughtDelta: "Inspecting request"}); err != nil {
		return provider.ChatResult{}, err
	}
	if err := emit(provider.ChatEvent{ThoughtDone: true}); err != nil {
		return provider.ChatResult{}, err
	}
	if err := emit(provider.ChatEvent{TextDelta: "Benchmark response."}); err != nil {
		return provider.ChatResult{}, err
	}
	return provider.ChatResult{
		Model:            p.model,
		PromptTokens:     64,
		CompletionTokens: 24,
	}, nil
}

func (p benchmarkProvider) GenerateTitle(context.Context, provider.TitleRequest) (string, error) {
	return "Benchmark thread", nil
}

func (p benchmarkProvider) ExtractUserMemories(context.Context, provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
	return nil, nil
}

func (p benchmarkProvider) SynthesizeSpeech(context.Context, provider.SpeechRequest, func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
	return provider.SpeechResult{}, nil
}

func newBenchmarkLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newBenchmarkStore(b *testing.B) *Store {
	b.Helper()
	store, err := openChatStore(filepath.Join(b.TempDir(), "benchmark-chat.db"))
	if err != nil {
		b.Fatalf("openChatStore: %v", err)
	}
	b.Cleanup(store.close)
	return store
}

func mustCreateBenchmarkUser(b *testing.B, store *Store, email string) authUser {
	b.Helper()
	userID, err := store.createUser(email, "hash", "Benchmark User")
	if err != nil {
		b.Fatalf("createUser: %v", err)
	}
	return authUser{ID: userID, Email: normalizeAuthEmail(email)}
}

func BenchmarkStoreCorePaths(b *testing.B) {
	store := newBenchmarkStore(b)
	user := mustCreateBenchmarkUser(b, store, "bench-store@example.com")

	conversationID, err := store.createConversation(user.ID)
	if err != nil {
		b.Fatalf("createConversation: %v", err)
	}
	if err := store.saveConversationMessage(user.ID, conversationID, "user", "Seed question", "", 0, 0); err != nil {
		b.Fatalf("saveConversationMessage seed user: %v", err)
	}
	if err := store.saveConversationMessage(user.ID, conversationID, "assistant", "Seed answer", modelGPT54Mini, 32, 12); err != nil {
		b.Fatalf("saveConversationMessage seed assistant: %v", err)
	}

	b.Run("create_conversation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := store.createConversation(user.ID); err != nil {
				b.Fatalf("createConversation: %v", err)
			}
		}
	})

	b.Run("save_message", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := store.saveConversationMessage(user.ID, conversationID, "assistant", "Synthetic benchmark reply", modelGPT54Mini, 64, 24); err != nil {
				b.Fatalf("saveConversationMessage: %v", err)
			}
		}
	})

	b.Run("list_conversations", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := store.listConversations(user.ID); err != nil {
				b.Fatalf("listConversations: %v", err)
			}
		}
	})

	b.Run("load_conversation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := store.loadConversation(user.ID, conversationID); err != nil {
				b.Fatalf("loadConversation: %v", err)
			}
		}
	})
}

func BenchmarkSendClientsPerCore(b *testing.B) {
	cpuCount := runtime.GOMAXPROCS(0)
	if cpuCount <= 0 {
		cpuCount = 1
	}

	for _, clientsPerCore := range []int{1, 2, 4} {
		b.Run(fmt.Sprintf("%d_clients_per_core", clientsPerCore), func(b *testing.B) {
			store := newBenchmarkStore(b)
			user := mustCreateBenchmarkUser(b, store, fmt.Sprintf("bench-send-%d@example.com", clientsPerCore))
			server := &chatServer{
				providerRegistry:      provider.NewRegistry(benchmarkProvider{model: modelGPT54Mini}),
				defaultModel:          modelGPT54Mini,
				store:                 store,
				logger:                newBenchmarkLogger(),
				sessions:              map[string]*sessionState{},
				authUsers:             map[string]authUser{},
				memoryExtractionSlots: make(chan struct{}, 2),
			}
			var workerCounter atomic.Int64

			b.ReportAllocs()
			b.ReportMetric(float64(clientsPerCore), "clients/core")
			b.ReportMetric(float64(clientsPerCore*cpuCount), "target_clients")
			b.SetParallelism(clientsPerCore)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				workerID := workerCounter.Add(1)
				peerName := fmt.Sprintf("bench-peer-%d-%d", clientsPerCore, workerID)
				ctx := bindAuthUser(server, peerName, user.ID, user.Email)
				conversationID, err := store.createConversation(user.ID)
				if err != nil {
					panic(fmt.Sprintf("createConversation: %v", err))
				}
				history := []*chatpb.ChatMessage{
					{Role: "user", Content: "Seed request"},
					{Role: "assistant", Content: "Seed response", ModelId: modelGPT54Mini, PromptTokens: 16, CompletionTokens: 8},
				}
				for _, message := range history {
					if err := store.saveConversationMessage(user.ID, conversationID, message.GetRole(), message.GetContent(), message.GetModelId(), message.GetPromptTokens(), message.GetCompletionTokens()); err != nil {
						panic(fmt.Sprintf("saveConversationMessage seed: %v", err))
					}
				}
				stream := &fakeChatSendStream{ctx: ctx}
				req := &chatpb.SendRequest{
					ConversationId:  conversationID,
					Model:           modelGPT54Mini,
					Message:         "Profile the bridge under benchmark load.",
					History:         history,
					Tone:            defaultToneID,
					ThinkingEnabled: true,
					ThinkingEffort:  defaultThinkingEffort,
				}

				for pb.Next() {
					stream.chunks = stream.chunks[:0]
					if err := server.Send(req, stream); err != nil {
						panic(fmt.Sprintf("Send: %v", err))
					}
				}

				server.unbindAuthenticatedPeer(peerName)
			})
		})
	}
}
