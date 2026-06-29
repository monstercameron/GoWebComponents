package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/server/provider"
)

type benchmarkProvider struct {
	model string
}

func (parseP benchmarkProvider) ParseID() string { return "benchmark" }

func (parseP benchmarkProvider) ParseAvailable() bool { return true }

func (parseP benchmarkProvider) ParseInfo() provider.ProviderInfo {
	return provider.ProviderInfo{
		ID:                 parseP.ParseID(),
		Label:              "Benchmark",
		AuthConfigured:     true,
		Available:          true,
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   false,
	}
}

func (parseP benchmarkProvider) ParseDefaultModel() string { return parseP.model }

func (parseP benchmarkProvider) ParseSupportsModel(parseModel string) bool {
	return parseNormalizeSelectedModelID(parseModel) == parseP.model
}

func (parseP benchmarkProvider) ParseModelOptions() []provider.ModelOption {
	parseCapabilities := parseP.ParseCapabilities(parseP.model)
	return []provider.ModelOption{{
		ID:           parseP.model,
		Label:        "Benchmark Model",
		Note:         "Synthetic benchmark provider",
		Capabilities: parseCapabilities,
	}}
}

func (parseP benchmarkProvider) ParseModelMetadata(parseModel string) (provider.ModelMetadata, bool) {
	if !parseP.ParseSupportsModel(parseModel) {
		return provider.ModelMetadata{}, false
	}
	parseCapabilities := parseP.ParseCapabilities(parseModel)
	return provider.ModelMetadata{
		ID:                 parseP.model,
		DisplayName:        "Benchmark Model",
		ProviderID:         parseP.ParseID(),
		ProviderLabel:      "Benchmark",
		ProviderFamily:     "benchmark",
		Capabilities:       parseCapabilities,
		StreamingSupported: true,
		ReasoningSupported: parseCapabilities.SupportsThinking,
		OnboardingReady:    true,
	}, true
}

func (parseP benchmarkProvider) ParseCapabilities(string) provider.ModelCapabilities {
	return provider.ModelCapabilities{
		ProviderID:       parseP.ParseID(),
		ProviderLabel:    "Benchmark",
		SupportsThinking: true,
	}
}

func (parseP benchmarkProvider) ParseHealth() provider.ProviderHealth {
	return provider.ProviderHealth{
		ProviderID: parseP.ParseID(),
		Status:     provider.ProviderHealthUnknown,
	}
}

func (parseP benchmarkProvider) ParseCurrentRateLimits() provider.RateLimitSnapshot {
	return provider.RateLimitSnapshot{}
}

func (parseP benchmarkProvider) ParseStreamChat(_ context.Context, _ provider.ChatRequest, parseEmit func(provider.ChatEvent) error) (provider.ChatResult, error) {
	if parseErr := parseEmit(provider.ChatEvent{ThoughtDelta: "Inspecting request"}); parseErr != nil {
		return provider.ChatResult{}, parseErr
	}
	if parseErr2 := parseEmit(provider.ChatEvent{ThoughtDone: true}); parseErr2 != nil {
		return provider.ChatResult{}, parseErr2
	}
	if parseErr3 := parseEmit(provider.ChatEvent{TextDelta: "Benchmark response."}); parseErr3 != nil {
		return provider.ChatResult{}, parseErr3
	}
	return provider.ChatResult{
		Model:            parseP.model,
		PromptTokens:     64,
		CompletionTokens: 24,
	}, nil
}

func (parseP benchmarkProvider) ParseGenerateTitle(context.Context, provider.TitleRequest) (string, error) {
	return "Benchmark thread", nil
}

func (parseP benchmarkProvider) ParseExtractUserMemories(context.Context, provider.MemoryExtractionRequest) ([]provider.UserMemoryCandidate, error) {
	return nil, nil
}

func (parseP benchmarkProvider) ParseSynthesizeSpeech(context.Context, provider.SpeechRequest, func(provider.SpeechChunk) error) (provider.SpeechResult, error) {
	return provider.SpeechResult{}, nil
}

func parseNewBenchmarkLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func parseNewBenchmarkStore(parseB *testing.B) *Store {
	parseB.Helper()
	store, parseErr := parseOpenChatStore(filepath.Join(parseB.TempDir(), "benchmark-chat.db"))
	if parseErr != nil {
		parseB.Fatalf("openChatStore: %v", parseErr)
	}
	parseB.Cleanup(store.parseClose)
	return store
}

func parseMustCreateBenchmarkUser(parseB *testing.B, store *Store, parseEmail string) authUser {
	parseB.Helper()
	parseUserID, parseErr := store.parseCreateUser(parseEmail, "hash", "Benchmark User")
	if parseErr != nil {
		parseB.Fatalf("createUser: %v", parseErr)
	}
	parseMustAssignBillingPlanBenchmark(parseB, store, parseUserID, "free")
	return authUser{ID: parseUserID, Email: parseNormalizeAuthEmail(parseEmail)}
}

func parseMustAssignBillingPlanBenchmark(parseB *testing.B, parseStore *Store, parseUserID int64, parsePlanCode string) {
	parseB.Helper()
	parsePlanCode = parseResolveSeedBillingPlanCode(parsePlanCode)
	parseNow := time.Now().UTC()
	parseCustomer, parseErr := parseStore.parseUpsertBillingCustomer(parseBillingCustomerWrite{
		UserID:             parseUserID,
		ProviderID:         "stripe",
		ProviderCustomerID: fmt.Sprintf("cus-bench-%d-%s", parseUserID, strings.TrimSpace(parsePlanCode)),
		DefaultCurrency:    "usd",
	})
	if parseErr != nil {
		parseB.Fatalf("parseUpsertBillingCustomer: %v", parseErr)
	}
	if _, parseErr2 := parseStore.parseUpsertBillingSubscription(parseBillingSubscriptionWrite{
		CustomerID:             parseCustomer.ID,
		ProviderID:             "stripe",
		ProviderSubscriptionID: fmt.Sprintf("sub-bench-%d-%s", parseUserID, strings.TrimSpace(parsePlanCode)),
		PlanCode:               strings.TrimSpace(parsePlanCode),
		Status:                 "active",
		BillingInterval:        "month",
		CurrentPeriodStart:     parseNow.Format(time.RFC3339),
		CurrentPeriodEnd:       parseNow.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}); parseErr2 != nil {
		parseB.Fatalf("parseUpsertBillingSubscription: %v", parseErr2)
	}
}

func BenchmarkStoreCorePaths(parseB *testing.B) {
	store := parseNewBenchmarkStore(parseB)
	parseUser := parseMustCreateBenchmarkUser(parseB, store, "bench-store@example.com")

	parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseB.Fatalf("createConversation: %v", parseErr)
	}
	if parseErr2 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "user", "Seed question", "", 0, 0); parseErr2 != nil {
		parseB.Fatalf("saveConversationMessage seed user: %v", parseErr2)
	}
	if parseErr3 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "assistant", "Seed answer", modelGPT54Mini, 32, 12); parseErr3 != nil {
		parseB.Fatalf("saveConversationMessage seed assistant: %v", parseErr3)
	}

	parseB.Run("create_conversation", func(parseB2 *testing.B) {
		parseB2.ReportAllocs()
		parseB2.ResetTimer()
		for parseI := 0; parseI < parseB2.N; parseI++ {
			if _, parseErr4 := store.parseCreateConversation(parseUser.ID); parseErr4 != nil {
				parseB2.Fatalf("createConversation: %v", parseErr4)
			}
		}
	})

	parseB.Run("save_message", func(parseB3 *testing.B) {
		parseB3.ReportAllocs()
		parseB3.ResetTimer()
		for parseI2 := 0; parseI2 < parseB3.N; parseI2++ {
			if parseErr5 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, "assistant", "Synthetic benchmark reply", modelGPT54Mini, 64, 24); parseErr5 != nil {
				parseB3.Fatalf("saveConversationMessage: %v", parseErr5)
			}
		}
	})

	parseB.Run("save_usage_event", func(parseB3 *testing.B) {
		parseB3.ReportAllocs()
		parseB3.ResetTimer()
		for parseI2 := 0; parseI2 < parseB3.N; parseI2++ {
			if parseErr5 := store.parseSaveUsageEvent(parseUsageEventWrite{
				EventID:                 fmt.Sprintf("bench-usage-%d", parseI2),
				UserID:                  parseUser.ID,
				ConversationID:          parseConversationID,
				ProviderID:              "benchmark",
				ModelID:                 modelGPT54Mini,
				PromptTokens:            64,
				CompletionTokens:        24,
				UsageSource:             provider.UsageSourceExact,
				InputCostPerMillionUSD:  0.25,
				OutputCostPerMillionUSD: 2.00,
				PricingCurrency:         "USD",
				InputCostUSD:            0.000016,
				OutputCostUSD:           0.000048,
				TotalCostUSD:            0.000064,
				Status:                  "completed",
			}); parseErr5 != nil {
				parseB3.Fatalf("parseSaveUsageEvent: %v", parseErr5)
			}
		}
	})

	parseB.Run("list_conversations", func(parseB4 *testing.B) {
		parseB4.ReportAllocs()
		parseB4.ResetTimer()
		for parseI3 := 0; parseI3 < parseB4.N; parseI3++ {
			if _, parseErr6 := store.parseListConversations(parseUser.ID); parseErr6 != nil {
				parseB4.Fatalf("listConversations: %v", parseErr6)
			}
		}
	})

	parseB.Run("load_conversation", func(parseB5 *testing.B) {
		parseB5.ReportAllocs()
		parseB5.ResetTimer()
		for parseI4 := 0; parseI4 < parseB5.N; parseI4++ {
			if _, parseErr7 := store.parseLoadConversation(parseUser.ID, parseConversationID); parseErr7 != nil {
				parseB5.Fatalf("loadConversation: %v", parseErr7)
			}
		}
	})

	parseB.Run("list_usage_events", func(parseB6 *testing.B) {
		parseB6.ReportAllocs()
		parseB6.ResetTimer()
		for parseI5 := 0; parseI5 < parseB6.N; parseI5++ {
			if _, parseErr8 := store.parseListUsageEvents(parseUser.ID, 100); parseErr8 != nil {
				parseB6.Fatalf("parseListUsageEvents: %v", parseErr8)
			}
		}
	})

	parseB.Run("list_billing_effective_model_access_by_user", func(parseB7 *testing.B) {
		parseB7.ReportAllocs()
		parseB7.ResetTimer()
		for parseI6 := 0; parseI6 < parseB7.N; parseI6++ {
			if _, parseErr9 := store.parseListBillingEffectiveModelAccessByUser(parseUser.ID, time.Now().UTC()); parseErr9 != nil {
				parseB7.Fatalf("parseListBillingEffectiveModelAccessByUser: %v", parseErr9)
			}
		}
	})
}

func BenchmarkSendClientsPerCore(parseB *testing.B) {
	parseCpuCount := runtime.GOMAXPROCS(0)
	if parseCpuCount <= 0 {
		parseCpuCount = 1
	}

	for _, parseClientsPerCore := range []int{1, 2, 4} {
		parseB.Run(fmt.Sprintf("%d_clients_per_core", parseClientsPerCore), func(parseB2 *testing.B) {
			store := parseNewBenchmarkStore(parseB2)
			parseUser := parseMustCreateBenchmarkUser(parseB2, store, fmt.Sprintf("bench-send-%d@example.com", parseClientsPerCore))
			parseServer := &chatServer{
				providerRegistry:      provider.ParseNewRegistry(benchmarkProvider{model: modelGPT54Mini}),
				defaultModel:          modelGPT54Mini,
				store:                 store,
				logger:                parseNewBenchmarkLogger(),
				sessions:              map[string]*sessionState{},
				authUsers:             map[string]authUser{},
				memoryExtractionSlots: make(chan struct{}, 2),
			}
			var parseWorkerCounter atomic.Int64

			parseB2.ReportAllocs()
			parseB2.ReportMetric(float64(parseClientsPerCore), "clients/core")
			parseB2.ReportMetric(float64(parseClientsPerCore*parseCpuCount), "target_clients")
			parseB2.SetParallelism(parseClientsPerCore)
			parseB2.ResetTimer()

			parseB2.RunParallel(func(parsePb *testing.PB) {
				parseWorkerID := parseWorkerCounter.Add(1)
				parsePeerName := fmt.Sprintf("bench-peer-%d-%d", parseClientsPerCore, parseWorkerID)
				parseCtx := parseBindAuthUser(parseServer, parsePeerName, parseUser.ID, parseUser.Email)
				parseConversationID, parseErr := store.parseCreateConversation(parseUser.ID)
				if parseErr != nil {
					panic(fmt.Sprintf("createConversation: %v", parseErr))
				}
				parseHistory := []*chatpb.ChatMessage{
					{Role: "user", Content: "Seed request"},
					{Role: "assistant", Content: "Seed response", ModelId: modelGPT54Mini, PromptTokens: 16, CompletionTokens: 8},
				}
				for _, parseMessage := range parseHistory {
					if parseErr2 := store.parseSaveConversationMessage(parseUser.ID, parseConversationID, parseMessage.GetRole(), parseMessage.GetContent(), parseMessage.GetModelId(), parseMessage.GetPromptTokens(), parseMessage.GetCompletionTokens()); parseErr2 != nil {
						panic(fmt.Sprintf("saveConversationMessage seed: %v", parseErr2))
					}
				}
				parseStream := &fakeChatSendStream{ctx: parseCtx}
				parseReq := &chatpb.SendRequest{
					ConversationId:  parseConversationID,
					Model:           modelGPT54Mini,
					Message:         "Profile the bridge under benchmark load.",
					History:         parseHistory,
					Tone:            defaultToneID,
					ThinkingEnabled: true,
					ThinkingEffort:  defaultThinkingEffort,
				}

				for parsePb.Next() {
					parseStream.chunks = parseStream.chunks[:0]
					if parseErr3 := parseServer.Send(parseReq, parseStream); parseErr3 != nil {
						panic(fmt.Sprintf("Send: %v", parseErr3))
					}
				}

				parseServer.parseUnbindAuthenticatedPeer(parsePeerName)
			})
		})
	}
}
