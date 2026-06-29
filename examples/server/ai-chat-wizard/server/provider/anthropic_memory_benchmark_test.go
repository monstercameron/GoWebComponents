package provider

import (
	"encoding/json"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

// BenchmarkResolveAnthropicMemoryCandidates measures tool-first extraction parsing overhead.
func BenchmarkResolveAnthropicMemoryCandidates(parseB *testing.B) {
	parseMessage := &anthropic.Message{}
	if parseErr := json.Unmarshal([]byte(`{"content":[{"type":"tool_use","id":"toolu_bench","name":"extract_user_memories","input":{"memories":[{"key":"pref-editor","category":"preference","summary":"Prefers Vim","detail":"Uses Vim daily","usefulness_score":91,"confidence_score":0.8,"rubric_reason":"stable preference"}]}}]}`), parseMessage); parseErr != nil {
		parseB.Fatalf("json.Unmarshal anthropic message: %v", parseErr)
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseCandidates, isParseResolved := parseResolveAnthropicMemoryCandidates(parseMessage)
		if !isParseResolved {
			parseB.Fatal("expected tool_use payload to resolve")
		}
		if len(parseCandidates) != 1 || parseCandidates[0].Key != "pref-editor" {
			parseB.Fatalf("unexpected candidates: %+v", parseCandidates)
		}
	}
}
