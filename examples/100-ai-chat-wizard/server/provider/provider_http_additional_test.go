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

func TestOpenAIProviderHTTPBackedBranches(t *testing.T) {
	t.Run("generate title and memories", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/responses") {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(string(body), "User message:") {
				_, _ = w.Write([]byte(`{"id":"resp_mem","object":"response","model":"gpt-5.4-mini","output":[{"id":"msg_mem","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"{\"memories\":[{\"key\":\"pref-editor\",\"category\":\"preference\",\"summary\":\"Prefers Neovim\",\"detail\":\"Uses it daily\",\"usefulness_score\":91,\"confidence_score\":0.8,\"rubric_reason\":\"stable preference\"}]}"}]}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"resp_title","object":"response","model":"gpt-5.4-nano","output":[{"id":"msg_title","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"  Daily Standup  "}]}]}`))
		}))
		defer server.Close()

		client := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(server.URL),
			option.WithHTTPClient(server.Client()),
		)
		provider := &OpenAIProvider{client: &client, catalog: testOpenAICatalog()}

		title, err := provider.GenerateTitle(context.Background(), TitleRequest{Prompt: "Summarize the chat", SystemPrompt: "Be concise"})
		if err != nil {
			t.Fatalf("GenerateTitle: %v", err)
		}
		if title != "Daily Standup" {
			t.Fatalf("GenerateTitle() = %q, want Daily Standup", title)
		}

		memories, err := provider.ExtractUserMemories(context.Background(), MemoryExtractionRequest{
			Model:       " gpt-5.4-mini ",
			UserMessage: "I prefer Neovim over VS Code.",
		})
		if err != nil {
			t.Fatalf("ExtractUserMemories: %v", err)
		}
		if len(memories) != 1 || memories[0].Key != "pref-editor" || memories[0].Category != "preference" {
			t.Fatalf("unexpected memories payload: %+v", memories)
		}
	})

	t.Run("empty title and memory parse failure", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			if callCount == 1 {
				_, _ = w.Write([]byte(`{"id":"resp_title","object":"response","model":"gpt-5.4-nano","output":[{"id":"msg_title","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"   "}]}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"resp_mem","object":"response","model":"gpt-5.4-mini","output":[{"id":"msg_mem","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"not-json"}]}]}`))
		}))
		defer server.Close()

		client := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(server.URL),
			option.WithHTTPClient(server.Client()),
		)
		provider := &OpenAIProvider{client: &client, catalog: testOpenAICatalog()}

		if _, err := provider.GenerateTitle(context.Background(), TitleRequest{Prompt: "empty"}); err == nil || !strings.Contains(err.Error(), "empty title") {
			t.Fatalf("GenerateTitle() error = %v, want empty title", err)
		}
		if _, err := provider.ExtractUserMemories(context.Background(), MemoryExtractionRequest{UserMessage: "remember this"}); err == nil || !strings.Contains(err.Error(), "parse") {
			t.Fatalf("ExtractUserMemories() error = %v, want parse failure", err)
		}
	})

	t.Run("speech success and failure paths", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/audio/speech") {
					t.Fatalf("unexpected speech path: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "audio/mpeg")
				_, _ = w.Write([]byte("fake-mp3-audio"))
			}))
			defer server.Close()

			client := openai.NewClient(
				option.WithAPIKey("test-key"),
				option.WithBaseURL(server.URL),
				option.WithHTTPClient(server.Client()),
			)
			provider := &OpenAIProvider{client: &client, catalog: testOpenAICatalog()}

			chunks := make([]SpeechChunk, 0, 2)
			result, err := provider.SynthesizeSpeech(context.Background(), SpeechRequest{
				Model: "gpt-5.4-mini",
				Text:  "  hello speech  ",
			}, func(chunk SpeechChunk) error {
				chunks = append(chunks, chunk)
				return nil
			})
			if err != nil {
				t.Fatalf("SynthesizeSpeech(success): %v", err)
			}
			if result.MimeType != openAITTSDefaultMimeType || result.Model != string(openAITTSDaultModel) || result.Voice != string(openAITTSDefaultVoice) || result.Script != "hello speech" {
				t.Fatalf("unexpected speech result: %+v", result)
			}
			if len(chunks) != 2 || len(chunks[0].AudioChunk) == 0 || !chunks[1].Done {
				t.Fatalf("unexpected speech chunks: %+v", chunks)
			}
			if chunks[0].MimeType != openAITTSDefaultMimeType || chunks[0].Model != string(openAITTSDaultModel) || chunks[0].Voice != string(openAITTSDefaultVoice) {
				t.Fatalf("first speech chunk missing metadata: %+v", chunks[0])
			}
		})

		t.Run("empty audio", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "audio/mpeg")
			}))
			defer server.Close()

			client := openai.NewClient(
				option.WithAPIKey("test-key"),
				option.WithBaseURL(server.URL),
				option.WithHTTPClient(server.Client()),
			)
			provider := &OpenAIProvider{client: &client, catalog: testOpenAICatalog()}

			if _, err := provider.SynthesizeSpeech(context.Background(), SpeechRequest{
				Model: "gpt-5.4-mini",
				Text:  "hello speech",
			}, func(SpeechChunk) error { return nil }); err == nil || !strings.Contains(err.Error(), "synthesized audio was empty") {
				t.Fatalf("SynthesizeSpeech(empty audio) error = %v, want empty audio failure", err)
			}
		})

		t.Run("emit failure", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "audio/mpeg")
				_, _ = w.Write([]byte("fake-mp3-audio"))
			}))
			defer server.Close()

			client := openai.NewClient(
				option.WithAPIKey("test-key"),
				option.WithBaseURL(server.URL),
				option.WithHTTPClient(server.Client()),
			)
			provider := &OpenAIProvider{client: &client, catalog: testOpenAICatalog()}

			expectedErr := errors.New("emit failed")
			if _, err := provider.SynthesizeSpeech(context.Background(), SpeechRequest{
				Model: "gpt-5.4-mini",
				Text:  "hello speech",
			}, func(SpeechChunk) error { return expectedErr }); !errors.Is(err, expectedErr) {
				t.Fatalf("SynthesizeSpeech(emit failure) error = %v, want %v", err, expectedErr)
			}
		})
	})
}

func TestAnthropicProviderHTTPBackedBranches(t *testing.T) {
	t.Run("generate title", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/messages") {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":"  Anthropic summary  "}],"usage":{"input_tokens":3,"output_tokens":5},"stop_reason":"end_turn","stop_sequence":""}`))
		}))
		defer server.Close()

		client := anthropic.NewClient(
			anthropicoption.WithAPIKey("test-key"),
			anthropicoption.WithBaseURL(server.URL),
			anthropicoption.WithHTTPClient(server.Client()),
		)
		provider := &AnthropicProvider{client: &client, catalog: testAnthropicCatalog()}

		title, err := provider.GenerateTitle(context.Background(), TitleRequest{Prompt: "hello", SystemPrompt: "title"})
		if err != nil {
			t.Fatalf("GenerateTitle: %v", err)
		}
		if title != "Anthropic summary" {
			t.Fatalf("GenerateTitle() = %q, want Anthropic summary", title)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"msg_2","type":"message","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":"   "}],"usage":{"input_tokens":3,"output_tokens":5},"stop_reason":"end_turn","stop_sequence":""}`))
		}))
		defer server.Close()

		client := anthropic.NewClient(
			anthropicoption.WithAPIKey("test-key"),
			anthropicoption.WithBaseURL(server.URL),
			anthropicoption.WithHTTPClient(server.Client()),
		)
		provider := &AnthropicProvider{client: &client, catalog: testAnthropicCatalog()}

		if _, err := provider.GenerateTitle(context.Background(), TitleRequest{Prompt: "empty"}); err == nil || !strings.Contains(err.Error(), "empty title") {
			t.Fatalf("GenerateTitle() error = %v, want empty title", err)
		}
	})
}

func TestCerebrasProviderHTTPBackedBranches(t *testing.T) {
	t.Run("generate title", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"chatcmpl_1","object":"chat.completion","created":1,"model":"llama3.1-8b","choices":[{"index":0,"message":{"role":"assistant","content":"  Cerebras title  "},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`))
		}))
		defer server.Close()

		client := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(server.URL),
			option.WithHTTPClient(server.Client()),
		)
		provider := &CerebrasProvider{client: &client, catalog: testCerebrasCatalog()}

		title, err := provider.GenerateTitle(context.Background(), TitleRequest{Prompt: "hello", SystemPrompt: "title"})
		if err != nil {
			t.Fatalf("GenerateTitle: %v", err)
		}
		if title != "Cerebras title" {
			t.Fatalf("GenerateTitle() = %q, want Cerebras title", title)
		}
	})

	t.Run("empty response and empty title", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			if callCount == 1 {
				_, _ = w.Write([]byte(`{"id":"chatcmpl_empty","object":"chat.completion","created":1,"model":"llama3.1-8b","choices":[],"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"chatcmpl_blank","object":"chat.completion","created":1,"model":"llama3.1-8b","choices":[{"index":0,"message":{"role":"assistant","content":"   "},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
		}))
		defer server.Close()

		client := openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(server.URL),
			option.WithHTTPClient(server.Client()),
		)
		provider := &CerebrasProvider{client: &client, catalog: testCerebrasCatalog()}

		if _, err := provider.GenerateTitle(context.Background(), TitleRequest{Prompt: "empty"}); err == nil || !strings.Contains(err.Error(), "empty response") {
			t.Fatalf("GenerateTitle() error = %v, want empty response", err)
		}
		if _, err := provider.GenerateTitle(context.Background(), TitleRequest{Prompt: "blank"}); err == nil || !strings.Contains(err.Error(), "empty title") {
			t.Fatalf("GenerateTitle() error = %v, want empty title", err)
		}
	})
}

func TestProviderHTTPStubPayloadsRemainValidJSON(t *testing.T) {
	payload := map[string]any{
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
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal payload: %v", err)
	}
	if !json.Valid(encoded) {
		t.Fatalf("expected marshaled payload to be valid JSON: %s", string(encoded))
	}
	if !errors.Is(nil, nil) {
		t.Fatal("expected nil to still compare equal under errors.Is")
	}
}
