package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// parseAssertMemoryExtractionParityCandidate verifies one normalized candidate shape shared across providers.
func parseAssertMemoryExtractionParityCandidate(parseT *testing.T, parseProviderLabel string, parseCandidates []UserMemoryCandidate, parseErr error) {
	parseT.Helper()
	if parseErr != nil {
		parseT.Fatalf("%s ParseExtractUserMemories: %v", parseProviderLabel, parseErr)
	}
	if len(parseCandidates) != 1 {
		parseT.Fatalf("%s candidate len=%d want=1 candidates=%+v", parseProviderLabel, len(parseCandidates), parseCandidates)
	}
	parseCandidate := parseCandidates[0]
	if parseCandidate.Key != "prefers-vim" {
		parseT.Fatalf("%s key=%q want=prefers-vim", parseProviderLabel, parseCandidate.Key)
	}
	if parseCandidate.Category != "other" {
		parseT.Fatalf("%s category=%q want=other", parseProviderLabel, parseCandidate.Category)
	}
	if parseCandidate.Summary != "Prefers Vim!!" {
		parseT.Fatalf("%s summary=%q want=Prefers Vim!!", parseProviderLabel, parseCandidate.Summary)
	}
	if parseCandidate.Detail != "Uses Vim every day" {
		parseT.Fatalf("%s detail=%q want=Uses Vim every day", parseProviderLabel, parseCandidate.Detail)
	}
	if parseCandidate.UsefulnessScore != 100 {
		parseT.Fatalf("%s usefulness_score=%d want=100", parseProviderLabel, parseCandidate.UsefulnessScore)
	}
	if parseCandidate.ConfidenceScore != 0 {
		parseT.Fatalf("%s confidence_score=%v want=0", parseProviderLabel, parseCandidate.ConfidenceScore)
	}
	if parseCandidate.RubricReason != "stable editing preference" {
		parseT.Fatalf("%s rubric_reason=%q want=stable editing preference", parseProviderLabel, parseCandidate.RubricReason)
	}
}

// TestMemoryExtractionParityAcrossProviders verifies OpenAI/Anthropic/Cerebras memory extraction returns one shared normalized candidate contract.
func TestMemoryExtractionParityAcrossProviders(parseT *testing.T) {
	parseT.Run("openai", func(parseT2 *testing.T) {
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			parseW.Header().Set("Content-Type", "application/json")
			_, _ = parseW.Write([]byte(`{"id":"resp_mem","object":"response","model":"gpt-5.4-mini","output":[{"id":"msg_mem","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"{\"memories\":[{\"key\":\"\",\"category\":\"unknown\",\"summary\":\"Prefers Vim!!\",\"detail\":\"Uses Vim every day\",\"usefulness_score\":150,\"confidence_score\":-2,\"rubric_reason\":\"stable editing preference\"}]}"}]}]}`))
		}))
		defer parseServer.Close()

		parseClient := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer.URL),
			option.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &OpenAIProvider{client: &parseClient, catalog: parseTestOpenAICatalog()}
		parseCandidates, parseErr := parseProvider.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{
			Model:       "gpt-5.4-mini",
			UserMessage: "I prefer Vim for coding.",
		})
		parseAssertMemoryExtractionParityCandidate(parseT2, "openai", parseCandidates, parseErr)
	})

	parseT.Run("anthropic", func(parseT2 *testing.T) {
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			parseW.Header().Set("Content-Type", "application/json")
			_, _ = parseW.Write([]byte(`{"id":"msg_mem","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"tool_use","id":"toolu_mem","name":"extract_user_memories","input":{"memories":[{"key":"","category":"unknown","summary":"Prefers Vim!!","detail":"Uses Vim every day","usefulness_score":150,"confidence_score":-2,"rubric_reason":"stable editing preference"}]}}],"usage":{"input_tokens":13,"output_tokens":9},"stop_reason":"tool_use","stop_sequence":""}`))
		}))
		defer parseServer.Close()

		parseClient := anthropic.NewClient(
			anthropicoption.WithAPIKey("test-key"),
			anthropicoption.WithBaseURL(parseServer.URL),
			anthropicoption.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &AnthropicProvider{client: &parseClient, catalog: parseTestAnthropicCatalog()}
		parseCandidates, parseErr := parseProvider.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{
			Model:       "claude-sonnet-4-5",
			UserMessage: "I prefer Vim for coding.",
		})
		parseAssertMemoryExtractionParityCandidate(parseT2, "anthropic", parseCandidates, parseErr)
	})

	parseT.Run("cerebras", func(parseT2 *testing.T) {
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			parseW.Header().Set("Content-Type", "application/json")
			_, _ = parseW.Write([]byte(`{"id":"chatcmpl_mem","object":"chat.completion","created":1,"model":"gpt-oss-120b","choices":[{"index":0,"message":{"role":"assistant","content":"{\"memories\":[{\"key\":\"\",\"category\":\"unknown\",\"summary\":\"Prefers Vim!!\",\"detail\":\"Uses Vim every day\",\"usefulness_score\":150,\"confidence_score\":-2,\"rubric_reason\":\"stable editing preference\"}]}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":8,"completion_tokens":5,"total_tokens":13}}`))
		}))
		defer parseServer.Close()

		parseClient := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer.URL),
			option.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &CerebrasProvider{client: &parseClient, catalog: parseTestCerebrasCatalog()}
		parseCandidates, parseErr := parseProvider.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{
			Model:       "gpt-oss-120b",
			UserMessage: "I prefer Vim for coding.",
		})
		parseAssertMemoryExtractionParityCandidate(parseT2, "cerebras", parseCandidates, parseErr)
	})
}
