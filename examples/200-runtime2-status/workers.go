//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	getRuntime2StatusWorkerCount          = 8
	getRuntime2StatusWorkerRequestName    = "runtime2-status-probe"
	getRuntime2StatusWorkerRuntimeURL     = "../../third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js"
	getRuntime2StatusWorkerWASMURL        = "../../bin/runtime2-status-worker.wasm"
	getRuntime2StatusWorkerReadyTimeout   = 5 * time.Second
	getRuntime2StatusWorkerRequestTimeout = 2 * time.Second
	getRuntime2StatusWorkerWorkScale      = 2
)

type runtime2StatusWorkerRequest struct {
	RegionID   string `json:"regionId"`
	Count      int    `json:"count"`
	Probe      int    `json:"probe"`
	Tone       string `json:"tone"`
	WorkScale  int    `json:"workScale"`
	GetTraceID string `json:"traceId"`
}

type runtime2StatusWorkerResult struct {
	Worker            string `json:"worker"`
	RegionID          string `json:"regionId"`
	Count             int    `json:"count"`
	Probe             int    `json:"probe"`
	Tone              string `json:"tone"`
	Summary           string `json:"summary"`
	GetWorkIterations int    `json:"workIterations"`
	GetWorkDigest     uint64 `json:"workDigest"`
	GetWorkDurationMS int64  `json:"workDurationMs"`
	GetTraceID        string `json:"traceId"`
}

type runtime2StatusWorkerFleetState struct {
	IsLoading bool
	ErrorText string
	Results   []runtime2StatusWorkerResult
}

type runtime2StatusWorkerFleetReport struct {
	Results        []runtime2StatusWorkerResult
	RequestedCount int
	SuccessCount   int
	FailureCount   int
	Duration       time.Duration
}

type runtime2StatusWorkerMetricsState struct {
	IsBooting           bool
	IsReady             bool
	BootCount           int
	BootSuccessCount    int
	BootFailureCount    int
	LastBootDuration    time.Duration
	RequestCount        int
	RequestSuccessCount int
	RequestFailureCount int
	ProbeCount          int
	ProbeSuccessCount   int
	ProbeFailureCount   int
	LastBatchDuration   time.Duration
	LastRegionID        string
	LastCount           int
	LastWorkerCount     int
	LastResultCount     int
	LastWorkScale       int
	LastBatchWorkerTime time.Duration
	LastBatchIterations int
	LastBatchDigestXOR  uint64
	LastErrorText       string
	LastEventText       string
}

type runtime2StatusWorkerFleetProps struct {
	RegionID string
	Count    int
}

var getRuntime2StatusWorkerTraceCounter uint64

// buildRuntime2StatusWorkerFleetAtomID builds a stable atom ID for one worker fleet region.
func buildRuntime2StatusWorkerFleetAtomID(parseRegionID string) string {
	return fmt.Sprintf("examples.runtime2-status.worker-fleet.%s", strings.TrimSpace(parseRegionID))
}

// resetRuntime2StatusWorkerTraceCounter resets per-probe trace sequencing for a fresh app boot.
func resetRuntime2StatusWorkerTraceCounter() {
	atomic.StoreUint64(&getRuntime2StatusWorkerTraceCounter, 0)
}

// applyRuntime2StatusWorkerMetrics applies one mutation to the worker metrics ref without forcing a rerender.
func applyRuntime2StatusWorkerMetrics(parseMetricsRef ui.Ref[runtime2StatusWorkerMetricsState], parseApply func(*runtime2StatusWorkerMetricsState)) {
	parseMetrics := parseMetricsRef.Get()
	if parseApply != nil {
		parseApply(&parseMetrics)
	}
	parseMetricsRef.Set(parseMetrics)
}

// buildRuntime2StatusWorkerTraceID builds a unique per-probe trace ID so worker logs remain unambiguous across pooled workers.
func buildRuntime2StatusWorkerTraceID(parseRegionID string, parseCount int, parseProbe int) string {
	parseSequence := atomic.AddUint64(&getRuntime2StatusWorkerTraceCounter, 1)
	return fmt.Sprintf("trace-%d:%s:%d:%d", parseSequence, parseRegionID, parseCount, parseProbe+1)
}

// buildRuntime2StatusWorkerMetricsText formats the debug metrics state for the page card.
func buildRuntime2StatusWorkerMetricsText(parseMetrics runtime2StatusWorkerMetricsState) string {
	parsePoolState := "idle"
	if parseMetrics.IsBooting {
		parsePoolState = "booting"
	}
	if parseMetrics.IsReady {
		parsePoolState = "ready"
	}
	parseLastError := strings.TrimSpace(parseMetrics.LastErrorText)
	if parseLastError == "" {
		parseLastError = "none"
	}
	parseLastEvent := strings.TrimSpace(parseMetrics.LastEventText)
	if parseLastEvent == "" {
		parseLastEvent = "none"
	}
	parseWorkScale := parseMetrics.LastWorkScale
	if parseWorkScale < 1 {
		parseWorkScale = getRuntime2StatusWorkerWorkScale
	}
	return fmt.Sprintf(
		"Runtime2 worker debug metrics\n"+
			"Configured worker count: %d\n"+
			"Worker work scale: %dx\n"+
			"Pool state: %s\n"+
			"Boot count: %d\n"+
			"Boot successes: %d\n"+
			"Boot failures: %d\n"+
			"Last boot duration: %s\n"+
			"Request batches: %d\n"+
			"Batch successes: %d\n"+
			"Batch failures: %d\n"+
			"Probe requests: %d\n"+
			"Probe successes: %d\n"+
			"Probe failures: %d\n"+
			"Last batch duration: %s\n"+
			"Last region: %s\n"+
			"Last count: %d\n"+
			"Last worker count: %d\n"+
			"Last result count: %d\n"+
			"Last batch worker CPU time: %s\n"+
			"Last batch iterations: %d\n"+
			"Last batch digest xor: %d\n"+
			"Last error: %s\n"+
			"Last event: %s",
		getRuntime2StatusWorkerCount,
		parseWorkScale,
		parsePoolState,
		parseMetrics.BootCount,
		parseMetrics.BootSuccessCount,
		parseMetrics.BootFailureCount,
		parseMetrics.LastBootDuration,
		parseMetrics.RequestCount,
		parseMetrics.RequestSuccessCount,
		parseMetrics.RequestFailureCount,
		parseMetrics.ProbeCount,
		parseMetrics.ProbeSuccessCount,
		parseMetrics.ProbeFailureCount,
		parseMetrics.LastBatchDuration,
		parseMetrics.LastRegionID,
		parseMetrics.LastCount,
		parseMetrics.LastWorkerCount,
		parseMetrics.LastResultCount,
		parseMetrics.LastBatchWorkerTime,
		parseMetrics.LastBatchIterations,
		parseMetrics.LastBatchDigestXOR,
		parseLastError,
		parseLastEvent,
	)
}

// buildRuntime2StatusWorkerBatchDuration returns the summed worker compute duration reported by the latest batch.
func buildRuntime2StatusWorkerBatchDuration(parseResults []runtime2StatusWorkerResult) time.Duration {
	var parseBatchDuration time.Duration
	for _, parseResult := range parseResults {
		parseBatchDuration += time.Duration(parseResult.GetWorkDurationMS) * time.Millisecond
	}
	return parseBatchDuration
}

// buildRuntime2StatusWorkerBatchIterations returns the summed worker iterations reported by the latest batch.
func buildRuntime2StatusWorkerBatchIterations(parseResults []runtime2StatusWorkerResult) int {
	parseBatchIterations := 0
	for _, parseResult := range parseResults {
		parseBatchIterations += parseResult.GetWorkIterations
	}
	return parseBatchIterations
}

// buildRuntime2StatusWorkerBatchDigestXOR folds worker digests into one deterministic batch fingerprint.
func buildRuntime2StatusWorkerBatchDigestXOR(parseResults []runtime2StatusWorkerResult) uint64 {
	var parseDigestXOR uint64
	for _, parseResult := range parseResults {
		parseDigestXOR ^= parseResult.GetWorkDigest
	}
	return parseDigestXOR
}

// renderRuntime2StatusWorkerMetrics renders the runtime2 worker telemetry card.
func renderRuntime2StatusWorkerMetrics(parseMetrics runtime2StatusWorkerMetricsState) ui.Node {
	return Div(
		Class("rounded-[28px] border border-sky-300/20 bg-sky-400/10 p-5 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			Class("text-xs font-semibold uppercase tracking-[0.24em] text-sky-100"),
			Text("Runtime2 Worker Metrics"),
		),
		H3(
			Class("mt-4 text-2xl font-black tracking-tight text-white"),
			Text("Worker telemetry"),
		),
		Pre(
			Class("mt-5 overflow-x-auto rounded-2xl border border-white/10 bg-slate-950/60 p-4 text-xs leading-6 text-slate-200"),
			Text(buildRuntime2StatusWorkerMetricsText(parseMetrics)),
		),
	)
}

// openRuntime2StatusWorkerPool opens the 8-worker Go WASM pool used by example 200.
func openRuntime2StatusWorkerPool(parseCtx context.Context) (interop.WorkerPool, error) {
	parseWorkerIndex := 0
	return interop.OpenWorkerPool(parseCtx, interop.WorkerPoolOptions{
		Size:       getRuntime2StatusWorkerCount,
		QueueLimit: 0,
		OpenWorker: func(openCtx context.Context) (interop.Worker, error) {
			parseWorkerIndex++
			parseWorkerName := fmt.Sprintf("runtime2-status-%d", parseWorkerIndex)
			fmt.Printf("[runtime2-status/runtime2] opening worker %s\n", parseWorkerName)
			return interop.OpenGoWASMWorker(openCtx, interop.GoWASMWorkerOptions{
				RuntimeURL:   getRuntime2StatusWorkerRuntimeURL,
				WASMURL:      getRuntime2StatusWorkerWASMURL,
				Name:         parseWorkerName,
				Ready:        true,
				ReadyTimeout: getRuntime2StatusWorkerReadyTimeout,
			})
		},
	})
}

// requestRuntime2StatusWorkerFleet runs one probe request on each pooled worker and returns the typed results in probe order.
func requestRuntime2StatusWorkerFleet(parseCtx context.Context, parsePool interop.WorkerPool, parseRegionID string, parseCount int, parseWorkScale int) (runtime2StatusWorkerFleetReport, error) {
	parseReport := runtime2StatusWorkerFleetReport{
		Results:        make([]runtime2StatusWorkerResult, getRuntime2StatusWorkerCount),
		RequestedCount: getRuntime2StatusWorkerCount,
	}
	var parseResultErr error
	var parseResultMu sync.Mutex
	var parseWait sync.WaitGroup
	parseStartedAt := time.Now()

	for parseProbeIndex := 0; parseProbeIndex < getRuntime2StatusWorkerCount; parseProbeIndex++ {
		parseWait.Add(1)
		go func(parseProbe int) {
			defer parseWait.Done()
			parseProbeCtx, parseProbeCancel := context.WithTimeout(parseCtx, getRuntime2StatusWorkerRequestTimeout)
			defer parseProbeCancel()
			parseProbeStartedAt := time.Now()
			parseTraceID := buildRuntime2StatusWorkerTraceID(parseRegionID, parseCount, parseProbe)
			fmt.Printf("[runtime2-status/runtime2] dispatching probe %d region=%s count=%d trace=%s\n", parseProbe+1, parseRegionID, parseCount, parseTraceID)

			parseResult, parseErr := interop.RequestWorkerDecoded[runtime2StatusWorkerRequest, struct{}, runtime2StatusWorkerResult](
				parseProbeCtx,
				parsePool,
				getRuntime2StatusWorkerRequestName,
				runtime2StatusWorkerRequest{
					RegionID:   parseRegionID,
					Count:      parseCount,
					Probe:      parseProbe + 1,
					Tone:       formatRuntime2StatusTone(parseCount),
					WorkScale:  parseWorkScale,
					GetTraceID: parseTraceID,
				},
				nil,
			)
			if parseErr != nil {
				parseResultMu.Lock()
				parseReport.FailureCount++
				if parseResultErr == nil {
					parseResultErr = parseErr
				}
				parseResultMu.Unlock()
				fmt.Printf("[runtime2-status/runtime2] probe %d failed after %s trace=%s: %v\n", parseProbe+1, time.Since(parseProbeStartedAt), parseTraceID, parseErr)
				return
			}

			parseResultMu.Lock()
			parseReport.SuccessCount++
			parseReport.Results[parseProbe] = parseResult
			parseResultMu.Unlock()
			fmt.Printf("[runtime2-status/runtime2] probe %d succeeded after %s worker=%s trace=%s\n", parseResult.Probe, time.Since(parseProbeStartedAt), parseResult.Worker, parseResult.GetTraceID)
		}(parseProbeIndex)
	}

	parseWait.Wait()
	parseReport.Duration = time.Since(parseStartedAt)
	fmt.Printf(
		"[runtime2-status/runtime2] worker batch complete region=%s count=%d requested=%d success=%d failure=%d duration=%s\n",
		parseRegionID,
		parseCount,
		parseReport.RequestedCount,
		parseReport.SuccessCount,
		parseReport.FailureCount,
		parseReport.Duration,
	)
	if parseResultErr != nil {
		return parseReport, parseResultErr
	}
	return parseReport, nil
}

// startRuntime2StatusWorkerBatch starts one asynchronous worker batch and returns its cancel function.
func startRuntime2StatusWorkerBatch(parseParentCtx context.Context, parseRegionID string, parseCount int, parsePool interop.WorkerPool, parseFleetSnapshotRef ui.Ref[runtime2StatusWorkerFleetState], parseMetricsRef ui.Ref[runtime2StatusWorkerMetricsState]) context.CancelFunc {
	parseCtx, parseCancel := context.WithTimeout(parseParentCtx, 3*time.Second)
	parseBatchStartedAt := time.Now()
	applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
		parsePrevious.IsBooting = false
		parsePrevious.IsReady = true
		parsePrevious.RequestCount++
		parsePrevious.ProbeCount += getRuntime2StatusWorkerCount
		parsePrevious.LastRegionID = parseRegionID
		parsePrevious.LastCount = parseCount
		parsePrevious.LastWorkerCount = getRuntime2StatusWorkerCount
		parsePrevious.LastWorkScale = getRuntime2StatusWorkerWorkScale
		parsePrevious.LastErrorText = ""
		parsePrevious.LastEventText = "starting runtime2 probe batch"
	})
	fmt.Printf("[runtime2-status/runtime2] worker batch starting region=%s count=%d workers=%d\n", parseRegionID, parseCount, getRuntime2StatusWorkerCount)

	go func() {
		defer parseCancel()

		parseResults, parseErr := requestRuntime2StatusWorkerFleet(parseCtx, parsePool, parseRegionID, parseCount, getRuntime2StatusWorkerWorkScale)
		parseBatchDuration := buildRuntime2StatusWorkerBatchDuration(parseResults.Results)
		parseBatchIterations := buildRuntime2StatusWorkerBatchIterations(parseResults.Results)
		parseBatchDigestXOR := buildRuntime2StatusWorkerBatchDigestXOR(parseResults.Results)
		if parseCtx.Err() != nil {
			return
		}
		if parseErr != nil {
			applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
				parsePrevious.RequestFailureCount++
				parsePrevious.ProbeSuccessCount += parseResults.SuccessCount
				parsePrevious.ProbeFailureCount += parseResults.FailureCount
				parsePrevious.LastBatchDuration = time.Since(parseBatchStartedAt)
				parsePrevious.LastRegionID = parseRegionID
				parsePrevious.LastCount = parseCount
				parsePrevious.LastWorkerCount = parseResults.RequestedCount
				parsePrevious.LastResultCount = parseResults.SuccessCount
				parsePrevious.LastWorkScale = getRuntime2StatusWorkerWorkScale
				parsePrevious.LastBatchWorkerTime = parseBatchDuration
				parsePrevious.LastBatchIterations = parseBatchIterations
				parsePrevious.LastBatchDigestXOR = parseBatchDigestXOR
				parsePrevious.LastErrorText = parseErr.Error()
				parsePrevious.LastEventText = "runtime2 probe batch failed"
			})
			parseFleetSnapshotRef.Set(runtime2StatusWorkerFleetState{
				IsLoading: false,
				ErrorText: parseErr.Error(),
				Results:   append([]runtime2StatusWorkerResult(nil), parseResults.Results...),
			})
			fmt.Printf("[runtime2-status/runtime2] worker batch failed region=%s count=%d success=%d failure=%d duration=%s error=%v\n", parseRegionID, parseCount, parseResults.SuccessCount, parseResults.FailureCount, time.Since(parseBatchStartedAt), parseErr)
			return
		}

		applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
			parsePrevious.RequestSuccessCount++
			parsePrevious.ProbeSuccessCount += parseResults.SuccessCount
			parsePrevious.ProbeFailureCount += parseResults.FailureCount
			parsePrevious.LastBatchDuration = time.Since(parseBatchStartedAt)
			parsePrevious.LastRegionID = parseRegionID
			parsePrevious.LastCount = parseCount
			parsePrevious.LastWorkerCount = parseResults.RequestedCount
			parsePrevious.LastResultCount = parseResults.SuccessCount
			parsePrevious.LastWorkScale = getRuntime2StatusWorkerWorkScale
			parsePrevious.LastBatchWorkerTime = parseBatchDuration
			parsePrevious.LastBatchIterations = parseBatchIterations
			parsePrevious.LastBatchDigestXOR = parseBatchDigestXOR
			parsePrevious.LastErrorText = ""
			parsePrevious.LastEventText = "runtime2 probe batch complete"
		})
		parseFleetSnapshotRef.Set(runtime2StatusWorkerFleetState{
			IsLoading: false,
			Results:   append([]runtime2StatusWorkerResult(nil), parseResults.Results...),
		})
		fmt.Printf(
			"[runtime2-status/runtime2] worker batch success region=%s count=%d success=%d failure=%d duration=%s worker-cpu=%s iterations=%d digest-xor=%d\n",
			parseRegionID,
			parseCount,
			parseResults.SuccessCount,
			parseResults.FailureCount,
			time.Since(parseBatchStartedAt),
			parseBatchDuration,
			parseBatchIterations,
			parseBatchDigestXOR,
		)
	}()

	return parseCancel
}

// handleRuntime2StatusWorkerPoolEffect boots and tears down the worker pool for the example panel.
func handleRuntime2StatusWorkerPoolEffect(parseRegionID string, parseCount int, parseShouldBoot bool, parsePoolRef ui.Ref[*interop.WorkerPool], parseFleetSnapshotRef ui.Ref[runtime2StatusWorkerFleetState], parseMetricsRef ui.Ref[runtime2StatusWorkerMetricsState]) {
	ui.UseEffect(func() func() {
		if !parseShouldBoot {
			return nil
		}
		if parsePoolRef.Get() != nil {
			return nil
		}

		parseCtx, parseCancel := context.WithCancel(context.Background())
		parseOpenStartedAt := time.Now()
		applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
			parsePrevious.IsBooting = true
			parsePrevious.IsReady = false
			parsePrevious.BootCount++
			parsePrevious.LastRegionID = parseRegionID
			parsePrevious.LastWorkerCount = getRuntime2StatusWorkerCount
			parsePrevious.LastErrorText = ""
			parsePrevious.LastEventText = "booting runtime2 worker pool"
		})
		fmt.Printf("[runtime2-status/runtime2] worker pool boot starting region=%s workers=%d\n", parseRegionID, getRuntime2StatusWorkerCount)
		go func() {
			parsePool, parseErr := openRuntime2StatusWorkerPool(parseCtx)
			if parseErr != nil {
				if parseCtx.Err() != nil {
					return
				}
				applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
					parsePrevious.IsBooting = false
					parsePrevious.IsReady = false
					parsePrevious.BootFailureCount++
					parsePrevious.LastBootDuration = time.Since(parseOpenStartedAt)
					parsePrevious.LastRegionID = parseRegionID
					parsePrevious.LastWorkerCount = getRuntime2StatusWorkerCount
					parsePrevious.LastErrorText = parseErr.Error()
					parsePrevious.LastEventText = "runtime2 worker pool boot failed"
				})
				parseFleetSnapshotRef.Set(runtime2StatusWorkerFleetState{
					IsLoading: false,
					ErrorText: parseErr.Error(),
					Results:   nil,
				})
				fmt.Printf("[runtime2-status/runtime2] worker pool boot failed region=%s workers=%d duration=%s error=%v\n", parseRegionID, getRuntime2StatusWorkerCount, time.Since(parseOpenStartedAt), parseErr)
				return
			}
			if parseCtx.Err() != nil {
				_ = parsePool.Close()
				return
			}
			parsePoolRef.Set(&parsePool)
			applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
				parsePrevious.IsBooting = false
				parsePrevious.IsReady = true
				parsePrevious.BootSuccessCount++
				parsePrevious.LastBootDuration = time.Since(parseOpenStartedAt)
				parsePrevious.LastRegionID = parseRegionID
				parsePrevious.LastWorkerCount = getRuntime2StatusWorkerCount
				parsePrevious.LastErrorText = ""
				parsePrevious.LastEventText = "runtime2 worker pool ready"
			})
			startRuntime2StatusWorkerBatch(parseCtx, parseRegionID, parseCount, parsePool, parseFleetSnapshotRef, parseMetricsRef)
			fmt.Printf("[runtime2-status/runtime2] worker pool ready region=%s workers=%d duration=%s\n", parseRegionID, getRuntime2StatusWorkerCount, time.Since(parseOpenStartedAt))
		}()

		return func() {
			applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
				parsePrevious.IsBooting = false
				parsePrevious.IsReady = false
				parsePrevious.LastRegionID = parseRegionID
				parsePrevious.LastWorkerCount = 0
				parsePrevious.LastEventText = "runtime2 worker pool closed"
			})
			fmt.Printf("[runtime2-status/runtime2] worker pool closing region=%s\n", parseRegionID)
			parseCancel()
			if parsePool := parsePoolRef.Get(); parsePool != nil {
				parsePoolRef.Set(nil)
				_ = parsePool.Close()
			}
		}
	}, parseRegionID, parseShouldBoot)
}

// handleRuntime2StatusWorkerRefreshEffect refreshes the worker panel whenever the owner count or pool readiness changes.
func handleRuntime2StatusWorkerRefreshEffect(parseRegionID string, parseCount int, parseShouldBoot bool, parsePoolRef ui.Ref[*interop.WorkerPool], parseFleetSnapshotRef ui.Ref[runtime2StatusWorkerFleetState], parseMetricsRef ui.Ref[runtime2StatusWorkerMetricsState]) {
	ui.UseEffect(func() func() {
		if !parseShouldBoot {
			return nil
		}
		parsePool := parsePoolRef.Get()
		if parsePool == nil {
			return nil
		}

		return startRuntime2StatusWorkerBatch(context.Background(), parseRegionID, parseCount, *parsePool, parseFleetSnapshotRef, parseMetricsRef)
	}, parseCount, parseShouldBoot)
}

// renderRuntime2StatusWorkerFleet renders the Go WASM worker pool panel and the latest probe results.
func renderRuntime2StatusWorkerFleet(parseProps runtime2StatusWorkerFleetProps) ui.Node {
	parseShouldBoot := parseProps.Count != 0
	parsePoolRef := ui.UseRef[*interop.WorkerPool](nil)
	parseFleetAtom := state.UseAtom(buildRuntime2StatusWorkerFleetAtomID(parseProps.RegionID), runtime2StatusWorkerFleetState{IsLoading: false})
	parseFleetSnapshotRef := ui.UseRef(runtime2StatusWorkerFleetState{IsLoading: false})
	parseMetricsRef := ui.UseRef(runtime2StatusWorkerMetricsState{})
	parseRenderCount := trackRuntime2StatusRenderCount("runtime2-worker-fleet")
	handleRuntime2StatusWorkerPoolEffect(parseProps.RegionID, parseProps.Count, parseShouldBoot, parsePoolRef, parseFleetSnapshotRef, parseMetricsRef)
	handleRuntime2StatusWorkerRefreshEffect(parseProps.RegionID, parseProps.Count, parseShouldBoot, parsePoolRef, parseFleetSnapshotRef, parseMetricsRef)

	parseFleet := parseFleetAtom.Get()
	parseMetrics := parseMetricsRef.Get()
	parseRefreshTelemetryView := ui.UseEvent(func() {
		parseSnapshot := parseFleetSnapshotRef.Get()
		if reflect.DeepEqual(parseFleetAtom.Get(), parseSnapshot) {
			return
		}
		parseFleetAtom.Set(parseSnapshot)
	})
	parseFleetNodes := []interface{}{
		Class("rounded-[28px] border border-emerald-300/20 bg-emerald-400/10 p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			Class("text-xs font-semibold uppercase tracking-[0.24em] text-emerald-100"),
			Text("Runtime2 Worker Fleet"),
		),
		H2(
			Class("mt-4 text-3xl font-black tracking-tight text-white"),
			Text("8 runtime2 Go WASM workers"),
		),
		P(
			Class("mt-4 text-sm leading-7 text-slate-300"),
			Text("The main runtime2 WASM opens eight Go WASM workers, then fans out a CPU-bound probe workload to each worker whenever the counter changes. The fleet stays in standby at count 0 to avoid cold-load churn, and telemetry view updates are manual to avoid app-shell rerender spam."),
		),
		P(
			Class("mt-3 text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
			Textf("Render pass #%d", parseRenderCount),
		),
		Button(
			OnClick(parseRefreshTelemetryView),
			Class("mt-4 rounded-xl border border-white/10 bg-white/[0.05] px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-slate-200 transition-colors hover:bg-white/[0.08]"),
			Text("Refresh telemetry view"),
		),
		renderRuntime2StatusWorkerMetrics(parseMetrics),
	}
	if parseFleet.IsLoading {
		parseFleetNodes = append(parseFleetNodes,
			P(
				Class("mt-4 text-sm leading-7 text-emerald-50/90"),
				Text("Booting the worker pool and waiting for the first probe results."),
			),
		)
	}
	if !parseShouldBoot && len(parseFleet.Results) == 0 && parseFleet.ErrorText == "" {
		parseFleetNodes = append(parseFleetNodes,
			P(
				Class("mt-4 text-sm leading-7 text-slate-300"),
				Text("Standby: increment or decrement Owner State to activate the worker fleet."),
			),
		)
	}
	if parseFleet.ErrorText != "" {
		parseFleetNodes = append(parseFleetNodes,
			Div(
				Class("mt-5 rounded-2xl border border-rose-300/20 bg-rose-400/10 p-4"),
				P(
					Class("text-xs font-semibold uppercase tracking-[0.22em] text-rose-100"),
					Text("Worker error"),
				),
				P(
					Class("mt-3 text-sm leading-7 text-rose-50/90"),
					Text(parseFleet.ErrorText),
				),
			),
		)
	}
	if len(parseFleet.Results) == 0 && parseFleet.ErrorText == "" && !parseFleet.IsLoading {
		parseFleetNodes = append(parseFleetNodes,
			P(
				Class("mt-4 text-sm leading-7 text-slate-300"),
				Text("No worker probe results are available yet."),
			),
		)
	}
	if len(parseFleet.Results) > 0 {
		parseWorkerCards := make([]interface{}, 0, len(parseFleet.Results))
		for _, parseResult := range parseFleet.Results {
			parseWorkerCards = append(parseWorkerCards,
				Div(
					Class("rounded-2xl border border-white/10 bg-slate-950/55 p-4"),
					P(
						Class("text-xs font-semibold uppercase tracking-[0.22em] text-emerald-100"),
						Textf("Probe %d", parseResult.Probe),
					),
					P(
						Class("mt-2 text-sm font-semibold text-white"),
						Text(parseResult.Worker),
					),
					P(
						Class("mt-2 text-sm leading-7 text-slate-300"),
						Text(parseResult.Summary),
					),
					P(
						Class("mt-2 text-xs leading-6 text-slate-400"),
						Textf("trace %s | iterations %d | worker CPU %dms | digest %d", parseResult.GetTraceID, parseResult.GetWorkIterations, parseResult.GetWorkDurationMS, parseResult.GetWorkDigest),
					),
					P(
						Class("mt-3 text-xs leading-6 text-slate-400"),
						Textf("Region %s | count %d | tone %s", parseResult.RegionID, parseResult.Count, parseResult.Tone),
					),
				),
			)
		}
		parseFleetNodes = append(parseFleetNodes, append([]interface{}{
			Class("mt-5 grid gap-3"),
		}, parseWorkerCards...)...)
	}
	return Div(parseFleetNodes...)
}
