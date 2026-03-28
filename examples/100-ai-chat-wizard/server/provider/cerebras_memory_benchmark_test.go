package provider

import (
	"testing"

	"github.com/openai/openai-go"
)

// BenchmarkResolveCerebrasMemoryCandidates measures deterministic JSON payload parsing overhead.
func BenchmarkResolveCerebrasMemoryCandidates(parseB *testing.B) {
	parseResponse := &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: `{"memories":[{"key":"pref-editor","category":"preference","summary":"Prefers Vim","detail":"Uses Vim daily","usefulness_score":91,"confidence_score":0.8,"rubric_reason":"stable preference"}]}`,
				},
			},
		},
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseCandidates, parseErr := parseResolveCerebrasMemoryCandidates(parseResponse)
		if parseErr != nil {
			parseB.Fatalf("parseResolveCerebrasMemoryCandidates: %v", parseErr)
		}
		if len(parseCandidates) != 1 || parseCandidates[0].Key != "pref-editor" {
			parseB.Fatalf("unexpected candidates: %+v", parseCandidates)
		}
	}
}
