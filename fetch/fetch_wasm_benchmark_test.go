//go:build js && wasm

package fetch

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkResolveMutationQueueOptions(parseB *testing.B) {
	parseB.ReportAllocs()
	parseOptions := []MutationQueueOptions{
		{
			StorageKey:  "bench-queue",
			MaxAttempts: 7,
			BaseDelay:   3 * time.Second,
			MaxDelay:    45 * time.Second,
		},
	}
	for parseB.Loop() {
		parseCfg := resolveMutationQueueOptions(parseOptions)
		if parseCfg.StorageKey == "" || parseCfg.MaxAttempts == 0 {
			parseB.Fatal("resolveMutationQueueOptions returned invalid config")
		}
	}
}

func BenchmarkNormalizeMutationID(parseB *testing.B) {
	parseB.ReportAllocs()
	parseNow := time.Unix(1711324800, 12345)
	for parseB.Loop() {
		parseId := normalizeMutationID("", parseNow, 9)
		if parseId == "" {
			parseB.Fatal("normalizeMutationID returned empty id")
		}
	}
}

func BenchmarkMergeResolvedMutation(parseB *testing.B) {
	parseB.ReportAllocs()
	parseExisting := QueuedMutation{
		ID:       "mutation-1",
		Method:   "POST",
		URL:      "/api/messages",
		Headers:  map[string]string{"content-type": "application/json"},
		Metadata: map[string]string{"lane": "chat"},
	}
	parseDraft := MutationDraft{
		ID:       "mutation-2",
		Kind:     "chat.send",
		DedupKey: "chat:send:bench",
		Method:   "post",
		URL:      "/api/messages",
		Headers: map[string]string{
			"content-type": "application/json",
			"x-bench":      "true",
		},
		Body: map[string]interface{}{
			"message": "hello",
		},
		Metadata: map[string]string{
			"source": "benchmark",
		},
	}
	parseNow := time.Unix(1711324800, 0)
	for parseB.Loop() {
		parseMerged := mergeResolvedMutation(parseExisting, parseDraft, parseNow, fmt.Sprintf("attempt-%d", parseB.N))
		if parseMerged.ID == "" || parseMerged.Method == "" || parseMerged.State != MutationQueued {
			parseB.Fatal("mergeResolvedMutation returned invalid record")
		}
	}
}
