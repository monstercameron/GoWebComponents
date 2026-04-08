package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// parseMustRequestJSON decodes an HTTP request body into a generic JSON object for test assertions.
func parseMustRequestJSON(parseT *testing.T, parseR *http.Request) map[string]any {
	parseT.Helper()

	parseBody, parseErr := io.ReadAll(parseR.Body)
	if parseErr != nil {
		parseT.Fatalf("ReadAll request body: %v", parseErr)
	}

	var parsePayload map[string]any
	if parseErr2 := json.Unmarshal(parseBody, &parsePayload); parseErr2 != nil {
		parseT.Fatalf("json.Unmarshal request body: %v\nbody=%s", parseErr2, string(parseBody))
	}
	return parsePayload
}

// buildParseSSEStream renders a simple SSE response body from alternating event names and JSON payloads.
func buildParseSSEStream(parsePairs ...string) string {
	var parseBuilder strings.Builder
	for parseIndex := 0; parseIndex+1 < len(parsePairs); parseIndex += 2 {
		parseEvent := strings.TrimSpace(parsePairs[parseIndex])
		parseData := parsePairs[parseIndex+1]
		if parseEvent != "" {
			parseBuilder.WriteString("event: ")
			parseBuilder.WriteString(parseEvent)
			parseBuilder.WriteString("\n")
		}
		parseBuilder.WriteString("data: ")
		parseBuilder.WriteString(parseData)
		parseBuilder.WriteString("\n\n")
	}
	return parseBuilder.String()
}

// TestOpenAIProviderStreamingHTTPBackedBranches covers response-stream success, retry, and emit-failure paths.
func TestOpenAIProviderStreamingHTTPBackedBranches(parseT *testing.T) {
	parseT.Run("success and retry fallback", func(parseT2 *testing.T) {
		parseRequestBodies := make([]map[string]any, 0, 3)
		parseRequestCount := 0
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			if !strings.HasSuffix(parseR.URL.Path, "/responses") {
				parseT2.Fatalf("unexpected path: %s", parseR.URL.Path)
			}

			parseRequestCount++
			parseRequestBody := parseMustRequestJSON(parseT2, parseR)
			parseRequestBodies = append(parseRequestBodies, parseRequestBody)

			if parseRequestCount == 2 {
				parseW.Header().Set("Content-Type", "application/json")
				parseW.WriteHeader(http.StatusBadRequest)
				_, _ = parseW.Write([]byte(`{"error":{"message":"reasoning.summary unsupported_value for this account","type":"invalid_request_error"}}`))
				return
			}

			parseW.Header().Set("Content-Type", "text/event-stream")
			parseW.Header().Set("Cache-Control", "no-cache")
			if parseRequestCount == 1 {
				_, _ = parseW.Write([]byte(buildParseSSEStream(
					"response.reasoning_summary_text.delta", `{"type":"response.reasoning_summary_text.delta","sequence_number":1,"item_id":"rs_1","output_index":0,"summary_index":0,"delta":"trace "}`,
					"response.output_text.delta", `{"type":"response.output_text.delta","sequence_number":2,"item_id":"out_1","output_index":0,"content_index":0,"delta":"answer","logprobs":[]}`,
					"response.completed", `{"type":"response.completed","sequence_number":3,"response":{"model":"gpt-5.4-mini","usage":{"input_tokens":11,"output_tokens":7}}}`,
				)))
				return
			}

			_, _ = parseW.Write([]byte(buildParseSSEStream(
				"response.output_text.delta", `{"type":"response.output_text.delta","sequence_number":1,"item_id":"out_2","output_index":0,"content_index":0,"delta":"fallback","logprobs":[]}`,
				"response.completed", `{"type":"response.completed","sequence_number":2,"response":{"model":"gpt-5.4-mini","usage":{"input_tokens":13,"output_tokens":5}}}`,
			)))
		}))
		defer parseServer.Close()

		parseClient := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer.URL),
			option.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &OpenAIProvider{client: &parseClient, catalog: parseTestOpenAICatalog()}

		parseEvents := make([]ChatEvent, 0, 4)
		parseResult, parseErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
			Model:           "gpt-5.4-mini",
			SystemPrompt:    "system",
			UserMessage:     "hello",
			ThinkingEnabled: true,
			ThinkingEffort:  "high",
		}, func(parseEvent ChatEvent) error {
			parseEvents = append(parseEvents, parseEvent)
			return nil
		})
		if parseErr != nil {
			parseT2.Fatalf("ParseStreamChat(success): %v", parseErr)
		}
		parseExpectedEvents := []ChatEvent{
			{ThoughtDelta: "trace "},
			{ThoughtDone: true},
			{TextDelta: "answer"},
		}
		if !reflect.DeepEqual(parseEvents, parseExpectedEvents) {
			parseT2.Fatalf("success events = %#v, want %#v", parseEvents, parseExpectedEvents)
		}
		if parseResult.Model != "gpt-5.4-mini" || parseResult.PromptTokens != 11 || parseResult.CompletionTokens != 7 || parseResult.UsageSource != UsageSourceExact {
			parseT2.Fatalf("unexpected success result: %+v", parseResult)
		}

		parseRetryEvents := make([]ChatEvent, 0, 4)
		parseRetryResult, parseRetryErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
			Model:           "gpt-5.4-mini",
			SystemPrompt:    "system",
			UserMessage:     "retry",
			ThinkingEnabled: true,
			ThinkingEffort:  "low",
		}, func(parseEvent ChatEvent) error {
			parseRetryEvents = append(parseRetryEvents, parseEvent)
			return nil
		})
		if parseRetryErr != nil {
			parseT2.Fatalf("ParseStreamChat(retry): %v", parseRetryErr)
		}
		parseExpectedRetryEvents := []ChatEvent{
			{ThoughtDelta: "Reasoning summaries are unavailable for this account, so continuing without live thought output."},
			{ThoughtDone: true},
			{TextDelta: "fallback"},
		}
		if !reflect.DeepEqual(parseRetryEvents, parseExpectedRetryEvents) {
			parseT2.Fatalf("retry events = %#v, want %#v", parseRetryEvents, parseExpectedRetryEvents)
		}
		if parseRetryResult.Model != "gpt-5.4-mini" || parseRetryResult.PromptTokens != 13 || parseRetryResult.CompletionTokens != 5 || parseRetryResult.UsageSource != UsageSourceExact {
			parseT2.Fatalf("unexpected retry result: %+v", parseRetryResult)
		}
		if parseRequestCount != 3 {
			parseT2.Fatalf("request count = %d, want 3", parseRequestCount)
		}
		if parseReasoning, parseOk := parseRequestBodies[0]["reasoning"].(map[string]any); !parseOk || parseReasoning["summary"] != "detailed" {
			parseT2.Fatalf("expected first request to include detailed reasoning summary, got %#v", parseRequestBodies[0]["reasoning"])
		}
		if parseReasoning, parseOk := parseRequestBodies[2]["reasoning"].(map[string]any); !parseOk || parseReasoning["summary"] != nil {
			parseT2.Fatalf("expected retried request to omit reasoning summary, got %#v", parseRequestBodies[2]["reasoning"])
		}
	})

	parseT.Run("emit failure", func(parseT3 *testing.T) {
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			parseW.Header().Set("Content-Type", "text/event-stream")
			_, _ = parseW.Write([]byte(buildParseSSEStream(
				"response.output_text.delta", `{"type":"response.output_text.delta","sequence_number":1,"item_id":"out_3","output_index":0,"content_index":0,"delta":"boom","logprobs":[]}`,
			)))
		}))
		defer parseServer.Close()

		parseClient := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer.URL),
			option.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &OpenAIProvider{client: &parseClient, catalog: parseTestOpenAICatalog()}
		parseExpectedErr := errors.New("emit failed")

		if _, parseErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
			Model:       "gpt-5.4-mini",
			UserMessage: "hello",
		}, func(ChatEvent) error {
			return parseExpectedErr
		}); !errors.Is(parseErr, parseExpectedErr) {
			parseT3.Fatalf("ParseStreamChat(emit failure) error = %v, want %v", parseErr, parseExpectedErr)
		}
	})
}

// TestAnthropicProviderStreamingHTTPBackedBranches covers message-stream success, retry, and available-provider branches.
func TestAnthropicProviderStreamingHTTPBackedBranches(parseT *testing.T) {
	parseT.Run("success and retry fallback", func(parseT2 *testing.T) {
		parseRequestBodies := make([]map[string]any, 0, 3)
		parseRequestCount := 0
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			if !strings.HasSuffix(parseR.URL.Path, "/v1/messages") {
				parseT2.Fatalf("unexpected path: %s", parseR.URL.Path)
			}

			parseRequestCount++
			parseRequestBody := parseMustRequestJSON(parseT2, parseR)
			parseRequestBodies = append(parseRequestBodies, parseRequestBody)

			if parseRequestCount == 2 {
				parseW.Header().Set("Content-Type", "application/json")
				parseW.WriteHeader(http.StatusBadRequest)
				_, _ = parseW.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"thinking unsupported invalid_request_error"}}`))
				return
			}

			parseW.Header().Set("Content-Type", "text/event-stream")
			parseW.Header().Set("Cache-Control", "no-cache")
			if parseRequestCount == 1 {
				_, _ = parseW.Write([]byte(buildParseSSEStream(
					"message_start", `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":17,"output_tokens":0}}}`,
					"content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
					"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"trace "}}`,
					"content_block_stop", `{"type":"content_block_stop","index":0}`,
					"content_block_start", `{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
					"content_block_delta", `{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"answer"}}`,
					"content_block_stop", `{"type":"content_block_stop","index":1}`,
					"message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"input_tokens":17,"output_tokens":9}}`,
					"message_stop", `{"type":"message_stop"}`,
				)))
				return
			}

			_, _ = parseW.Write([]byte(buildParseSSEStream(
				"message_start", `{"type":"message_start","message":{"id":"msg_2","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":19,"output_tokens":0}}}`,
				"content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
				"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"fallback"}}`,
				"content_block_stop", `{"type":"content_block_stop","index":0}`,
				"message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"input_tokens":19,"output_tokens":6}}`,
				"message_stop", `{"type":"message_stop"}`,
			)))
		}))
		defer parseServer.Close()

		parseClient := anthropic.NewClient(
			anthropicoption.WithAPIKey("test-key"),
			anthropicoption.WithBaseURL(parseServer.URL),
			anthropicoption.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &AnthropicProvider{client: &parseClient, catalog: parseTestAnthropicCatalog()}

		parseEvents := make([]ChatEvent, 0, 4)
		parseResult, parseErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
			Model:           "claude-sonnet-4-5",
			SystemPrompt:    "system",
			UserMessage:     "hello",
			ThinkingEnabled: true,
			ThinkingEffort:  "high",
		}, func(parseEvent ChatEvent) error {
			parseEvents = append(parseEvents, parseEvent)
			return nil
		})
		if parseErr != nil {
			parseT2.Fatalf("ParseStreamChat(success): %v", parseErr)
		}
		parseExpectedEvents := []ChatEvent{
			{ThoughtDelta: "trace "},
			{ThoughtDone: true},
			{TextDelta: "answer"},
		}
		if !reflect.DeepEqual(parseEvents, parseExpectedEvents) {
			parseT2.Fatalf("success events = %#v, want %#v", parseEvents, parseExpectedEvents)
		}
		if parseResult.Model != "claude-sonnet-4-5" || parseResult.PromptTokens != 17 || parseResult.CompletionTokens != 9 || parseResult.UsageSource != UsageSourceExact || parseResult.ProviderRequestID != "msg_1" {
			parseT2.Fatalf("unexpected success result: %+v", parseResult)
		}

		parseRetryEvents := make([]ChatEvent, 0, 4)
		parseRetryResult, parseRetryErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
			Model:           "claude-sonnet-4-5",
			SystemPrompt:    "system",
			UserMessage:     "retry",
			ThinkingEnabled: true,
			ThinkingEffort:  "low",
		}, func(parseEvent ChatEvent) error {
			parseRetryEvents = append(parseRetryEvents, parseEvent)
			return nil
		})
		if parseRetryErr != nil {
			parseT2.Fatalf("ParseStreamChat(retry): %v", parseRetryErr)
		}
		parseExpectedRetryEvents := []ChatEvent{
			{ThoughtDelta: "Extended thinking is unavailable for this Claude configuration, so continuing without live thought output."},
			{ThoughtDone: true},
			{TextDelta: "fallback"},
		}
		if !reflect.DeepEqual(parseRetryEvents, parseExpectedRetryEvents) {
			parseT2.Fatalf("retry events = %#v, want %#v", parseRetryEvents, parseExpectedRetryEvents)
		}
		if parseRetryResult.Model != "claude-sonnet-4-5" || parseRetryResult.PromptTokens != 19 || parseRetryResult.CompletionTokens != 6 || parseRetryResult.UsageSource != UsageSourceExact || parseRetryResult.ProviderRequestID != "msg_2" {
			parseT2.Fatalf("unexpected retry result: %+v", parseRetryResult)
		}
		if parseRequestCount != 3 {
			parseT2.Fatalf("request count = %d, want 3", parseRequestCount)
		}
		if _, parseHasThinking := parseRequestBodies[0]["thinking"]; !parseHasThinking {
			parseT2.Fatalf("expected first request to include thinking config, got %#v", parseRequestBodies[0])
		}
		if _, parseHasThinking := parseRequestBodies[2]["thinking"]; parseHasThinking {
			parseT2.Fatalf("expected retried request to omit thinking config, got %#v", parseRequestBodies[2])
		}
	})

	parseT.Run("available provider errors", func(parseT3 *testing.T) {
		parseProvider := ParseNewAnthropicProvider("test-key", parseTestAnthropicCatalog())

		if parseMemories, parseErr := parseProvider.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{
			UserMessage: "   ",
		}); parseErr != nil || parseMemories != nil {
			parseT3.Fatalf("ParseExtractUserMemories(blank) = %+v, %v; want nil,nil", parseMemories, parseErr)
		}

		parseSpeechErr := &UnsupportedCapabilityError{}
		if _, parseErr2 := parseProvider.ParseSynthesizeSpeech(context.Background(), SpeechRequest{
			Model: "claude-sonnet-4-5",
			Text:  "hello",
		}, func(SpeechChunk) error {
			return nil
		}); !errors.As(parseErr2, &parseSpeechErr) {
			parseT3.Fatalf("ParseSynthesizeSpeech() error = %v, want UnsupportedCapabilityError", parseErr2)
		}
	})
}

// TestCerebrasProviderStreamingHTTPBackedBranches covers chat-completion streaming and available-provider error branches.
func TestCerebrasProviderStreamingHTTPBackedBranches(parseT *testing.T) {
	parseT.Run("success", func(parseT2 *testing.T) {
		parseRequestBodies := make([]map[string]any, 0, 2)
		parseRequestCount := 0
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			if !strings.HasSuffix(parseR.URL.Path, "/chat/completions") {
				parseT2.Fatalf("unexpected path: %s", parseR.URL.Path)
			}

			parseRequestCount++
			parseRequestBodies = append(parseRequestBodies, parseMustRequestJSON(parseT2, parseR))

			parseW.Header().Set("Content-Type", "text/event-stream")
			parseW.Header().Set("Cache-Control", "no-cache")
			if parseRequestCount == 1 {
				_, _ = parseW.Write([]byte(buildParseSSEStream(
					"", `{"id":"chatcmpl_reasoning","object":"chat.completion.chunk","created":1,"model":"gpt-oss-120b","choices":[{"index":0,"delta":{"reasoning_content":"trace "},"finish_reason":null}],"usage":null}`,
					"", `{"id":"chatcmpl_text","object":"chat.completion.chunk","created":1,"model":"gpt-oss-120b","choices":[{"index":0,"delta":{"content":"answer"},"finish_reason":null}],"usage":null}`,
					"", `{"id":"chatcmpl_usage","object":"chat.completion.chunk","created":1,"model":"gpt-oss-120b","choices":[],"usage":{"prompt_tokens":23,"completion_tokens":10,"total_tokens":33}}`,
				)))
				return
			}

			_, _ = parseW.Write([]byte(buildParseSSEStream(
				"", `{"id":"chatcmpl_fast","object":"chat.completion.chunk","created":1,"model":"llama3.1-8b","choices":[{"index":0,"delta":{"content":"fast"},"finish_reason":null}],"usage":null}`,
				"", `{"id":"chatcmpl_fast_usage","object":"chat.completion.chunk","created":1,"model":"llama3.1-8b","choices":[],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			)))
		}))
		defer parseServer.Close()

		parseClient := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer.URL),
			option.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &CerebrasProvider{client: &parseClient, catalog: parseTestCerebrasCatalog()}

		parseEvents := make([]ChatEvent, 0, 4)
		parseResult, parseErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
			Model:           "gpt-oss-120b",
			SystemPrompt:    "system",
			UserMessage:     "hello",
			ThinkingEnabled: true,
			ThinkingEffort:  "high",
		}, func(parseEvent ChatEvent) error {
			parseEvents = append(parseEvents, parseEvent)
			return nil
		})
		if parseErr != nil {
			parseT2.Fatalf("ParseStreamChat(reasoning model): %v", parseErr)
		}
		parseExpectedEvents := []ChatEvent{
			{ThoughtDelta: "trace "},
			{ThoughtDone: true},
			{TextDelta: "answer"},
		}
		if !reflect.DeepEqual(parseEvents, parseExpectedEvents) {
			parseT2.Fatalf("reasoning-model events = %#v, want %#v", parseEvents, parseExpectedEvents)
		}
		if parseResult.Model != "gpt-oss-120b" || parseResult.PromptTokens != 23 || parseResult.CompletionTokens != 10 || parseResult.UsageSource != UsageSourceExact || parseResult.ProviderRequestID != "chatcmpl_usage" {
			parseT2.Fatalf("unexpected reasoning-model result: %+v", parseResult)
		}

		parseFastEvents := make([]ChatEvent, 0, 2)
		parseFastResult, parseFastErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
			Model:           "llama3.1-8b",
			SystemPrompt:    "system",
			UserMessage:     "fast",
			ThinkingEnabled: true,
			ThinkingEffort:  "high",
		}, func(parseEvent ChatEvent) error {
			parseFastEvents = append(parseFastEvents, parseEvent)
			return nil
		})
		if parseFastErr != nil {
			parseT2.Fatalf("ParseStreamChat(fast model): %v", parseFastErr)
		}
		parseExpectedFastEvents := []ChatEvent{{TextDelta: "fast"}}
		if !reflect.DeepEqual(parseFastEvents, parseExpectedFastEvents) {
			parseT2.Fatalf("fast-model events = %#v, want %#v", parseFastEvents, parseExpectedFastEvents)
		}
		if parseFastResult.Model != "llama3.1-8b" || parseFastResult.PromptTokens != 5 || parseFastResult.CompletionTokens != 2 || parseFastResult.UsageSource != UsageSourceExact || parseFastResult.ProviderRequestID != "chatcmpl_fast_usage" {
			parseT2.Fatalf("unexpected fast-model result: %+v", parseFastResult)
		}
		if parseRequestCount != 2 {
			parseT2.Fatalf("request count = %d, want 2", parseRequestCount)
		}
		if parseReasoningEffort, parseHasReasoningEffort := parseRequestBodies[0]["reasoning_effort"]; !parseHasReasoningEffort || parseReasoningEffort != "high" {
			parseT2.Fatalf("expected first request to include reasoning effort, got %#v", parseRequestBodies[0])
		}
		if _, parseHasReasoningEffort := parseRequestBodies[1]["reasoning_effort"]; parseHasReasoningEffort {
			parseT2.Fatalf("expected fast-model request to omit reasoning effort, got %#v", parseRequestBodies[1])
		}
	})

	parseT.Run("available provider errors and emit failure", func(parseT3 *testing.T) {
		parseProvider := ParseNewCerebrasProvider("test-key", parseTestCerebrasCatalog())

		if parseMemories, parseErr := parseProvider.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{
			UserMessage: "   ",
		}); parseErr != nil || parseMemories != nil {
			parseT3.Fatalf("ParseExtractUserMemories(blank) = %+v, %v; want nil,nil", parseMemories, parseErr)
		}

		parseSpeechErr := &UnsupportedCapabilityError{}
		if _, parseErr2 := parseProvider.ParseSynthesizeSpeech(context.Background(), SpeechRequest{
			Model: "gpt-oss-120b",
			Text:  "hello",
		}, func(SpeechChunk) error {
			return nil
		}); !errors.As(parseErr2, &parseSpeechErr) {
			parseT3.Fatalf("ParseSynthesizeSpeech() error = %v, want UnsupportedCapabilityError", parseErr2)
		}

		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			parseW.Header().Set("Content-Type", "text/event-stream")
			_, _ = parseW.Write([]byte(buildParseSSEStream(
				"", `{"id":"chatcmpl_emit","object":"chat.completion.chunk","created":1,"model":"gpt-oss-120b","choices":[{"index":0,"delta":{"content":"boom"},"finish_reason":null}],"usage":null}`,
			)))
		}))
		defer parseServer.Close()

		parseClient := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer.URL),
			option.WithHTTPClient(parseServer.Client()),
		)
		parseProviderWithStream := &CerebrasProvider{client: &parseClient, catalog: parseTestCerebrasCatalog()}
		parseExpectedErr := errors.New("emit failed")
		if _, parseErr3 := parseProviderWithStream.ParseStreamChat(context.Background(), ChatRequest{
			Model:       "gpt-oss-120b",
			UserMessage: "hello",
		}, func(ChatEvent) error {
			return parseExpectedErr
		}); !errors.Is(parseErr3, parseExpectedErr) {
			parseT3.Fatalf("ParseStreamChat(emit failure) error = %v, want %v", parseErr3, parseExpectedErr)
		}
	})
}
