//go:build js && wasm
// +build js,wasm

package fetch

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkResolveMutationQueueOptions(b *testing.B) {
	b.ReportAllocs()
	options := []MutationQueueOptions{
		{
			StorageKey:  "bench-queue",
			MaxAttempts: 7,
			BaseDelay:   3 * time.Second,
			MaxDelay:    45 * time.Second,
		},
	}
	for b.Loop() {
		cfg := resolveMutationQueueOptions(options)
		if cfg.StorageKey == "" || cfg.MaxAttempts == 0 {
			b.Fatal("resolveMutationQueueOptions returned invalid config")
		}
	}
}

func BenchmarkNormalizeMutationID(b *testing.B) {
	b.ReportAllocs()
	now := time.Unix(1711324800, 12345)
	for b.Loop() {
		id := normalizeMutationID("", now, 9)
		if id == "" {
			b.Fatal("normalizeMutationID returned empty id")
		}
	}
}

func BenchmarkMergeResolvedMutation(b *testing.B) {
	b.ReportAllocs()
	existing := QueuedMutation{
		ID:       "mutation-1",
		Method:   "POST",
		URL:      "/api/messages",
		Headers:  map[string]string{"content-type": "application/json"},
		Metadata: map[string]string{"lane": "chat"},
	}
	draft := MutationDraft{
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
	now := time.Unix(1711324800, 0)
	for b.Loop() {
		merged := mergeResolvedMutation(existing, draft, now, fmt.Sprintf("attempt-%d", b.N))
		if merged.ID == "" || merged.Method == "" || merged.State != MutationQueued {
			b.Fatal("mergeResolvedMutation returned invalid record")
		}
	}
}
