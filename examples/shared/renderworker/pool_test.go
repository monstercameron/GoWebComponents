package renderworker

import (
	"context"
	"strings"
	"testing"
)

// TestBuildRenderWorkerPoolRejectsInvalidWorkerCount validates that non-positive worker counts fail fast.
func TestBuildRenderWorkerPoolRejectsInvalidWorkerCount(parseT *testing.T) {
	_, parseErr := BuildRenderWorkerPool(context.Background(), PoolOptions{
		GetSize:       0,
		GetQueueLimit: 0,
		GetRuntimeURL: "runtime.js",
		GetWASMURL:    "worker.wasm",
	})
	if parseErr == nil {
		parseT.Fatalf("expected worker-count validation error")
	}
	if !strings.Contains(parseErr.Error(), "worker count") {
		parseT.Fatalf("expected worker-count validation message, got %v", parseErr)
	}
}

// TestBuildRenderWorkerPoolRejectsNegativeQueueLimit validates that negative queue limits fail fast.
func TestBuildRenderWorkerPoolRejectsNegativeQueueLimit(parseT *testing.T) {
	_, parseErr := BuildRenderWorkerPool(context.Background(), PoolOptions{
		GetSize:       1,
		GetQueueLimit: -1,
		GetRuntimeURL: "runtime.js",
		GetWASMURL:    "worker.wasm",
	})
	if parseErr == nil {
		parseT.Fatalf("expected queue-limit validation error")
	}
	if !strings.Contains(parseErr.Error(), "queue limit") {
		parseT.Fatalf("expected queue-limit validation message, got %v", parseErr)
	}
}

// TestBuildRenderWorkerPoolRequiresRuntimeURL validates runtime URL guard behavior.
func TestBuildRenderWorkerPoolRequiresRuntimeURL(parseT *testing.T) {
	_, parseErr := BuildRenderWorkerPool(context.Background(), PoolOptions{
		GetSize:       1,
		GetQueueLimit: 0,
		GetRuntimeURL: " ",
		GetWASMURL:    "worker.wasm",
	})
	if parseErr == nil {
		parseT.Fatalf("expected runtime URL validation error")
	}
	if !strings.Contains(parseErr.Error(), "runtime URL") {
		parseT.Fatalf("expected runtime URL validation message, got %v", parseErr)
	}
}

// TestBuildRenderWorkerPoolRequiresWASMURL validates wasm URL guard behavior.
func TestBuildRenderWorkerPoolRequiresWASMURL(parseT *testing.T) {
	_, parseErr := BuildRenderWorkerPool(context.Background(), PoolOptions{
		GetSize:       1,
		GetQueueLimit: 0,
		GetRuntimeURL: "runtime.js",
		GetWASMURL:    " ",
	})
	if parseErr == nil {
		parseT.Fatalf("expected wasm URL validation error")
	}
	if !strings.Contains(parseErr.Error(), "wasm URL") {
		parseT.Fatalf("expected wasm URL validation message, got %v", parseErr)
	}
}
