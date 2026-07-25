package renderworker

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// PoolOptions configures one generic Go WASM render worker pool.
type PoolOptions struct {
	GetSize         int
	GetQueueLimit   int
	GetRuntimeURL   string
	GetWASMURL      string
	GetNamePrefix   string
	IsReady         bool
	GetReadyTimeout time.Duration
}

// BuildRenderWorkerPool opens one fixed-size pool of Go WASM workers for generic render tasks.
func BuildRenderWorkerPool(parseCtx context.Context, parseOptions PoolOptions) (interop.WorkerPool, error) {
	if parseOptions.GetSize <= 0 {
		return interop.WorkerPool{}, fmt.Errorf("renderworker: worker count must be positive")
	}
	if parseOptions.GetQueueLimit < 0 {
		return interop.WorkerPool{}, fmt.Errorf("renderworker: worker queue limit must be non-negative")
	}
	parseRuntimeURL := strings.TrimSpace(parseOptions.GetRuntimeURL)
	parseWASMURL := strings.TrimSpace(parseOptions.GetWASMURL)
	if parseRuntimeURL == "" {
		return interop.WorkerPool{}, fmt.Errorf("renderworker: runtime URL is required")
	}
	if parseWASMURL == "" {
		return interop.WorkerPool{}, fmt.Errorf("renderworker: wasm URL is required")
	}
	parseNamePrefix := strings.TrimSpace(parseOptions.GetNamePrefix)
	if parseNamePrefix == "" {
		parseNamePrefix = "render-worker"
	}
	parseReadyTimeout := parseOptions.GetReadyTimeout
	if parseReadyTimeout <= 0 {
		parseReadyTimeout = 5 * time.Second
	}
	var parseWorkerIndexCounter uint64
	return interop.OpenWorkerPool(parseCtx, interop.WorkerPoolOptions{
		Size:       parseOptions.GetSize,
		QueueLimit: parseOptions.GetQueueLimit,
		OpenWorker: func(parseOpenCtx context.Context) (interop.Worker, error) {
			parseWorkerIndex := atomic.AddUint64(&parseWorkerIndexCounter, 1)
			parseWorkerName := fmt.Sprintf("%s-%d", parseNamePrefix, parseWorkerIndex)
			return interop.OpenGoWASMWorker(parseOpenCtx, interop.GoWASMWorkerOptions{
				RuntimeURL:   parseRuntimeURL,
				WASMURL:      parseWASMURL,
				Name:         parseWorkerName,
				Ready:        parseOptions.IsReady,
				ReadyTimeout: parseReadyTimeout,
			})
		},
	})
}
