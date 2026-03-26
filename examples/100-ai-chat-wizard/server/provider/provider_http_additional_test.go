package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func TestOpenAIProviderHTTPBackedBranches(parseT *testing.T) {
	parseT.Run("generate title and memories", func(parseT2 *testing.T) {
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			if !strings.HasSuffix(parseR.URL.Path, "/responses") {
				parseT2.Fatalf("unexpected path: %s", parseR.URL.Path)
			}
			parseBody, parseErr := io.ReadAll(parseR.Body)
			if parseErr != nil {
				parseT2.Fatalf("ReadAll: %v", parseErr)
			}
			var parsePayload map[string]any
			if parseErr2 := json.Unmarshal(parseBody, &parsePayload); parseErr2 != nil {
				parseT2.Fatalf("json.Unmarshal request: %v", parseErr2)
			}
			parseW.Header().Set("Content-Type", "application/json")
			if parseInput, _ := parsePayload["input"].(string); strings.Contains(parseInput, "User message:") {
				if parseGotModel, _ := parsePayload["model"].(string); parseGotModel != "gpt-5.4-mini" {
					parseT2.Fatalf("memory extraction model = %q, want gpt-5.4-mini", parseGotModel)
				}
				parseTextConfig, parseOk := parsePayload["text"].(map[string]any)
				if !parseOk {
					parseT2.Fatalf("missing text config in extraction request: %#v", parsePayload["text"])
				}
				formatConfig, parseOk := parseTextConfig["format"].(map[string]any)
				if !parseOk {
					parseT2.Fatalf("missing text.format in extraction request: %#v", parseTextConfig["format"])
				}
				if parseGotType, _ := formatConfig["type"].(string); parseGotType != "json_schema" {
					parseT2.Fatalf("text.format.type = %q, want json_schema", parseGotType)
				}
				if parseGotName, _ := formatConfig["name"].(string); parseGotName != "user_memories" {
					parseT2.Fatalf("text.format.name = %q, want user_memories", parseGotName)
				}
				if parseGotStrict, _ := formatConfig["strict"].(bool); !parseGotStrict {
					parseT2.Fatalf("text.format.strict = %v, want true", formatConfig["strict"])
				}
				parseSchema, parseOk := formatConfig["schema"].(map[string]any)
				if !parseOk {
					parseT2.Fatalf("text.format.schema missing from extraction request")
				}
				parseProperties, parseOk := parseSchema["properties"].(map[string]any)
				if !parseOk {
					parseT2.Fatalf("text.format.schema.properties missing from extraction request")
				}
				if _, parseOk2 := parseProperties["memories"]; !parseOk2 {
					parseT2.Fatalf("text.format.schema.properties.memories missing from extraction request")
				}
				_, _ = parseW.Write([]byte(`{"id":"resp_mem","object":"response","model":"gpt-5.4-mini","output":[{"id":"msg_mem","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"{\"memories\":[{\"key\":\"pref-editor\",\"category\":\"preference\",\"summary\":\"Prefers Neovim\",\"detail\":\"Uses it daily\",\"usefulness_score\":91,\"confidence_score\":0.8,\"rubric_reason\":\"stable preference\"}]}"}]}]}`))
				return
			}
			_, _ = parseW.Write([]byte(`{"id":"resp_title","object":"response","model":"gpt-5.4-nano","output":[{"id":"msg_title","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"  Daily Standup  "}]}]}`))
		}))
		defer parseServer.Close()

		parseClient := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer.URL),
			option.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &OpenAIProvider{client: &parseClient, catalog: parseTestOpenAICatalog()}

		parseTitle, parseErr3 := parseProvider.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "Summarize the chat", SystemPrompt: "Be concise"})
		if parseErr3 != nil {
			parseT2.Fatalf("GenerateTitle: %v", parseErr3)
		}
		if parseTitle != "Daily Standup" {
			parseT2.Fatalf("GenerateTitle() = %q, want Daily Standup", parseTitle)
		}

		parseMemories, parseErr3 := parseProvider.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{
			Model:       " gpt-5.4-mini ",
			UserMessage: "I prefer Neovim over VS Code.",
		})
		if parseErr3 != nil {
			parseT2.Fatalf("ExtractUserMemories: %v", parseErr3)
		}
		if len(parseMemories) != 1 || parseMemories[0].Key != "pref-editor" || parseMemories[0].Category != "preference" {
			parseT2.Fatalf("unexpected memories payload: %+v", parseMemories)
		}
	})

	parseT.Run("empty title and memory parse failure", func(parseT3 *testing.T) {
		parseCallCount := 0
		parseServer2 := httptest.NewServer(http.HandlerFunc(func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
			parseCallCount++
			parseW2.Header().Set("Content-Type", "application/json")
			if parseCallCount == 1 {
				_, _ = parseW2.Write([]byte(`{"id":"resp_title","object":"response","model":"gpt-5.4-nano","output":[{"id":"msg_title","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"   "}]}]}`))
				return
			}
			_, _ = parseW2.Write([]byte(`{"id":"resp_mem","object":"response","model":"gpt-5.4-mini","output":[{"id":"msg_mem","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"not-json"}]}]}`))
		}))
		defer parseServer2.Close()

		parseClient2 := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer2.URL),
			option.WithHTTPClient(parseServer2.Client()),
		)
		parseProvider2 := &OpenAIProvider{client: &parseClient2, catalog: parseTestOpenAICatalog()}

		if _, parseErr4 := parseProvider2.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "empty"}); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "empty title") {
			parseT3.Fatalf("GenerateTitle() error = %v, want empty title", parseErr4)
		}
		if _, parseErr5 := parseProvider2.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{UserMessage: "remember this"}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "parse") {
			parseT3.Fatalf("ExtractUserMemories() error = %v, want parse failure", parseErr5)
		}
	})

	parseT.Run("speech success and failure paths", func(parseT4 *testing.T) {
		parseT4.Run("success", func(parseT5 *testing.T) {
			parseServer3 := httptest.NewServer(http.HandlerFunc(func(parseW3 http.ResponseWriter, parseR3 *http.Request) {
				if !strings.Contains(parseR3.URL.Path, "/audio/speech") {
					parseT5.Fatalf("unexpected speech path: %s", parseR3.URL.Path)
				}
				parseW3.Header().Set("Content-Type", "audio/mpeg")
				_, _ = parseW3.Write([]byte("fake-mp3-audio"))
			}))
			defer parseServer3.Close()

			parseClient3 := openai.NewClient(
				option.WithAPIKey("test-key"),
				option.WithBaseURL(parseServer3.URL),
				option.WithHTTPClient(parseServer3.Client()),
			)
			parseProvider3 := &OpenAIProvider{client: &parseClient3, catalog: parseTestOpenAICatalog()}

			parseChunks := make([]SpeechChunk, 0, 2)
			parseResult, parseErr6 := parseProvider3.ParseSynthesizeSpeech(context.Background(), SpeechRequest{
				Model: "gpt-5.4-mini",
				Text:  "  hello speech  ",
			}, func(parseChunk SpeechChunk) error {
				parseChunks = append(parseChunks, parseChunk)
				return nil
			})
			if parseErr6 != nil {
				parseT5.Fatalf("SynthesizeSpeech(success): %v", parseErr6)
			}
			if parseResult.MimeType != openAITTSDefaultMimeType || parseResult.Model != string(openAITTSDaultModel) || parseResult.Voice != string(openAITTSDefaultVoice) || parseResult.Script != "hello speech" {
				parseT5.Fatalf("unexpected speech result: %+v", parseResult)
			}
			if len(parseChunks) != 2 || len(parseChunks[0].AudioChunk) == 0 || !parseChunks[1].Done {
				parseT5.Fatalf("unexpected speech chunks: %+v", parseChunks)
			}
			if parseChunks[0].MimeType != openAITTSDefaultMimeType || parseChunks[0].Model != string(openAITTSDaultModel) || parseChunks[0].Voice != string(openAITTSDefaultVoice) {
				parseT5.Fatalf("first speech chunk missing metadata: %+v", parseChunks[0])
			}
		})

		parseT4.Run("empty audio", func(parseT6 *testing.T) {
			parseServer4 := httptest.NewServer(http.HandlerFunc(func(parseW4 http.ResponseWriter, parseR4 *http.Request) {
				parseW4.Header().Set("Content-Type", "audio/mpeg")
			}))
			defer parseServer4.Close()

			parseClient4 := openai.NewClient(
				option.WithAPIKey("test-key"),
				option.WithBaseURL(parseServer4.URL),
				option.WithHTTPClient(parseServer4.Client()),
			)
			parseProvider4 := &OpenAIProvider{client: &parseClient4, catalog: parseTestOpenAICatalog()}

			if _, parseErr7 := parseProvider4.ParseSynthesizeSpeech(context.Background(), SpeechRequest{
				Model: "gpt-5.4-mini",
				Text:  "hello speech",
			}, func(SpeechChunk) error { return nil }); parseErr7 == nil || !strings.Contains(parseErr7.Error(), "synthesized audio was empty") {
				parseT6.Fatalf("SynthesizeSpeech(empty audio) error = %v, want empty audio failure", parseErr7)
			}
		})

		parseT4.Run("emit failure", func(parseT7 *testing.T) {
			parseServer5 := httptest.NewServer(http.HandlerFunc(func(parseW5 http.ResponseWriter, parseR5 *http.Request) {
				parseW5.Header().Set("Content-Type", "audio/mpeg")
				_, _ = parseW5.Write([]byte("fake-mp3-audio"))
			}))
			defer parseServer5.Close()

			parseClient5 := openai.NewClient(
				option.WithAPIKey("test-key"),
				option.WithBaseURL(parseServer5.URL),
				option.WithHTTPClient(parseServer5.Client()),
			)
			parseProvider5 := &OpenAIProvider{client: &parseClient5, catalog: parseTestOpenAICatalog()}

			parseExpectedErr := errors.New("emit failed")
			if _, parseErr8 := parseProvider5.ParseSynthesizeSpeech(context.Background(), SpeechRequest{
				Model: "gpt-5.4-mini",
				Text:  "hello speech",
			}, func(SpeechChunk) error { return parseExpectedErr }); !errors.Is(parseErr8, parseExpectedErr) {
				parseT7.Fatalf("SynthesizeSpeech(emit failure) error = %v, want %v", parseErr8, parseExpectedErr)
			}
		})
	})
}

func TestAnthropicProviderHTTPBackedBranches(parseT *testing.T) {
	parseT.Run("generate title", func(parseT2 *testing.T) {
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			if !strings.HasSuffix(parseR.URL.Path, "/messages") {
				parseT2.Fatalf("unexpected path: %s", parseR.URL.Path)
			}
			parseW.Header().Set("Content-Type", "application/json")
			_, _ = parseW.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":"  Anthropic summary  "}],"usage":{"input_tokens":3,"output_tokens":5},"stop_reason":"end_turn","stop_sequence":""}`))
		}))
		defer parseServer.Close()

		parseClient := anthropic.NewClient(
			anthropicoption.WithAPIKey("test-key"),
			anthropicoption.WithBaseURL(parseServer.URL),
			anthropicoption.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &AnthropicProvider{client: &parseClient, catalog: parseTestAnthropicCatalog()}

		parseTitle, parseErr := parseProvider.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "hello", SystemPrompt: "title"})
		if parseErr != nil {
			parseT2.Fatalf("GenerateTitle: %v", parseErr)
		}
		if parseTitle != "Anthropic summary" {
			parseT2.Fatalf("GenerateTitle() = %q, want Anthropic summary", parseTitle)
		}
	})

	parseT.Run("empty title", func(parseT3 *testing.T) {
		parseServer2 := httptest.NewServer(http.HandlerFunc(func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
			parseW2.Header().Set("Content-Type", "application/json")
			_, _ = parseW2.Write([]byte(`{"id":"msg_2","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":"   "}],"usage":{"input_tokens":3,"output_tokens":5},"stop_reason":"end_turn","stop_sequence":""}`))
		}))
		defer parseServer2.Close()

		parseClient2 := anthropic.NewClient(
			anthropicoption.WithAPIKey("test-key"),
			anthropicoption.WithBaseURL(parseServer2.URL),
			anthropicoption.WithHTTPClient(parseServer2.Client()),
		)
		parseProvider2 := &AnthropicProvider{client: &parseClient2, catalog: parseTestAnthropicCatalog()}

		if _, parseErr2 := parseProvider2.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "empty"}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "empty title") {
			parseT3.Fatalf("GenerateTitle() error = %v, want empty title", parseErr2)
		}
	})
}

func TestCerebrasProviderHTTPBackedBranches(parseT *testing.T) {
	parseT.Run("generate title", func(parseT2 *testing.T) {
		parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			if !strings.HasSuffix(parseR.URL.Path, "/chat/completions") {
				parseT2.Fatalf("unexpected path: %s", parseR.URL.Path)
			}
			parseW.Header().Set("Content-Type", "application/json")
			_, _ = parseW.Write([]byte(`{"id":"chatcmpl_1","object":"chat.completion","created":1,"model":"llama3.1-8b","choices":[{"index":0,"message":{"role":"assistant","content":"  Cerebras title  "},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`))
		}))
		defer parseServer.Close()

		parseClient := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer.URL),
			option.WithHTTPClient(parseServer.Client()),
		)
		parseProvider := &CerebrasProvider{client: &parseClient, catalog: parseTestCerebrasCatalog()}

		parseTitle, parseErr := parseProvider.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "hello", SystemPrompt: "title"})
		if parseErr != nil {
			parseT2.Fatalf("GenerateTitle: %v", parseErr)
		}
		if parseTitle != "Cerebras title" {
			parseT2.Fatalf("GenerateTitle() = %q, want Cerebras title", parseTitle)
		}
	})

	parseT.Run("empty response and empty title", func(parseT3 *testing.T) {
		parseCallCount := 0
		parseServer2 := httptest.NewServer(http.HandlerFunc(func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
			parseCallCount++
			parseW2.Header().Set("Content-Type", "application/json")
			if parseCallCount == 1 {
				_, _ = parseW2.Write([]byte(`{"id":"chatcmpl_empty","object":"chat.completion","created":1,"model":"llama3.1-8b","choices":[],"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}}`))
				return
			}
			_, _ = parseW2.Write([]byte(`{"id":"chatcmpl_blank","object":"chat.completion","created":1,"model":"llama3.1-8b","choices":[{"index":0,"message":{"role":"assistant","content":"   "},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
		}))
		defer parseServer2.Close()

		parseClient2 := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(parseServer2.URL),
			option.WithHTTPClient(parseServer2.Client()),
		)
		parseProvider2 := &CerebrasProvider{client: &parseClient2, catalog: parseTestCerebrasCatalog()}

		if _, parseErr2 := parseProvider2.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "empty"}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "empty response") {
			parseT3.Fatalf("GenerateTitle() error = %v, want empty response", parseErr2)
		}
		if _, parseErr3 := parseProvider2.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "blank"}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "empty title") {
			parseT3.Fatalf("GenerateTitle() error = %v, want empty title", parseErr3)
		}
	})
}

func TestProviderHTTPStubPayloadsRemainValidJSON(parseT *testing.T) {
	parsePayload := map[string]any{
		"id":     "resp_check",
		"object": "response",
		"output": []map[string]any{
			{
				"id":     "msg_check",
				"type":   "message",
				"role":   "assistant",
				"status": "completed",
				"content": []map[string]any{
					{"type": "output_text", "text": `{"memories":[]}`},
				},
			},
		},
	}
	parseEncoded, parseErr := json.Marshal(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("json.Marshal payload: %v", parseErr)
	}
	if !json.Valid(parseEncoded) {
		parseT.Fatalf("expected marshaled payload to be valid JSON: %s", string(parseEncoded))
	}
	if !errors.Is(nil, nil) {
		parseT.Fatal("expected nil to still compare equal under errors.Is")
	}
}
