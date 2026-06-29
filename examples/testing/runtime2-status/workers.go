//go:build js && wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
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

// buildRuntime2StatusWorkerTraceID builds a unique per-probe trace ID so worker logs remain unambiguous across worker lanes.
func buildRuntime2StatusWorkerTraceID(parseRegionID string, parseCount int, parseProbe int) string {
	parseSequence := atomic.AddUint64(&getRuntime2StatusWorkerTraceCounter, 1)
	return fmt.Sprintf("trace-%d:%s:%d:%d", parseSequence, parseRegionID, parseCount, parseProbe+1)
}

// buildRuntime2StatusWorkerMetricsText formats the debug metrics state for the page card.
func buildRuntime2StatusWorkerMetricsText(parseMetrics runtime2StatusWorkerMetricsState) string {
	parseFleetState := "idle"
	if parseMetrics.IsBooting {
		parseFleetState = "booting"
	}
	if parseMetrics.IsReady {
		parseFleetState = "ready"
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
			"Fleet state: %s\n"+
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
		parseFleetState,
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

// buildRuntime2StatusWorkerBatchStats folds one batch into total worker duration, total iterations, and digest xor.
func buildRuntime2StatusWorkerBatchStats(parseResults []runtime2StatusWorkerResult) (time.Duration, int, uint64) {
	var parseBatchDuration time.Duration
	parseBatchIterations := 0
	var parseDigestXOR uint64
	for _, parseResult := range parseResults {
		parseBatchDuration += time.Duration(parseResult.GetWorkDurationMS) * time.Millisecond
		parseBatchIterations += parseResult.GetWorkIterations
		parseDigestXOR ^= parseResult.GetWorkDigest
	}
	return parseBatchDuration, parseBatchIterations, parseDigestXOR
}

// renderRuntime2StatusWorkerMetrics renders the runtime2 worker telemetry card.
func renderRuntime2StatusWorkerMetrics(parseMetrics runtime2StatusWorkerMetricsState) ui.Node {
	return Div(
		ClassStr("rounded-[28px] border border-sky-300/20 bg-sky-400/10 p-5 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.24em] text-sky-100"),
			Text("Runtime2 Worker Metrics"),
		),
		H3(
			ClassStr("mt-4 text-2xl font-black tracking-tight text-white"),
			Text("Worker telemetry"),
		),
		Pre(
			ClassStr("mt-5 overflow-x-auto rounded-2xl border border-white/10 bg-slate-950/60 p-4 text-xs leading-6 text-slate-200"),
			Text(buildRuntime2StatusWorkerMetricsText(parseMetrics)),
		),
	)
}

// openRuntime2StatusWorkerFleet opens one dedicated Go WASM worker per probe lane used by example 200.
func openRuntime2StatusWorkerFleet(parseCtx context.Context) ([]interop.Worker, error) {
	parseWorkers := make([]interop.Worker, 0, getRuntime2StatusWorkerCount)
	for parseWorkerIndex := 0; parseWorkerIndex < getRuntime2StatusWorkerCount; parseWorkerIndex++ {
		parseWorkerName := fmt.Sprintf("runtime2-status-%d", parseWorkerIndex+1)
		fmt.Printf("[runtime2-status/runtime2] opening worker %s\n", parseWorkerName)
		parseWorker, parseErr := interop.OpenGoWASMWorker(parseCtx, interop.GoWASMWorkerOptions{
			RuntimeURL:   getRuntime2StatusWorkerRuntimeURL,
			WASMURL:      getRuntime2StatusWorkerWASMURL,
			Name:         parseWorkerName,
			Ready:        true,
			ReadyTimeout: getRuntime2StatusWorkerReadyTimeout,
		})
		if parseErr != nil {
			_ = closeRuntime2StatusWorkerFleet(parseWorkers)
			return nil, parseErr
		}
		parseWorkers = append(parseWorkers, parseWorker)
	}
	return parseWorkers, nil
}

// closeRuntime2StatusWorkerFleet terminates each worker in the provided fleet and joins non-disposed errors.
func closeRuntime2StatusWorkerFleet(parseWorkers []interop.Worker) error {
	var parseCloseErrors []error
	for _, parseWorker := range parseWorkers {
		if parseErr := parseWorker.Terminate(); parseErr != nil && !interop.IsCode(parseErr, interop.CodeDisposed) {
			parseCloseErrors = append(parseCloseErrors, parseErr)
		}
	}
	return errors.Join(parseCloseErrors...)
}

// requestRuntime2StatusWorkerFleet runs one probe request on each dedicated worker and returns typed results in probe order.
func requestRuntime2StatusWorkerFleet(parseCtx context.Context, parseWorkers []interop.Worker, parseRegionID string, parseCount int, parseWorkScale int) (runtime2StatusWorkerFleetReport, error) {
	parseRegionID = strings.TrimSpace(parseRegionID)
	parseReport := runtime2StatusWorkerFleetReport{
		Results:        make([]runtime2StatusWorkerResult, getRuntime2StatusWorkerCount),
		RequestedCount: getRuntime2StatusWorkerCount,
	}
	if len(parseWorkers) != getRuntime2StatusWorkerCount {
		parseErr := fmt.Errorf("runtime2 worker fleet size mismatch: have=%d want=%d", len(parseWorkers), getRuntime2StatusWorkerCount)
		fmt.Printf("[runtime2-status/runtime2][error] %v\n", parseErr)
		parseReport.FailureCount = parseReport.RequestedCount
		return parseReport, parseErr
	}
	var parseResultErr error
	var parseResultMu sync.Mutex
	var parseWait sync.WaitGroup
	parseStartedAt := time.Now()
	parseTone := formatRuntime2StatusTone(parseCount)

	for parseProbeIndex := 0; parseProbeIndex < getRuntime2StatusWorkerCount; parseProbeIndex++ {
		parseWait.Add(1)
		go func(parseProbe int, parseWorker interop.Worker) {
			defer parseWait.Done()
			parseProbeCtx, parseProbeCancel := context.WithTimeout(parseCtx, getRuntime2StatusWorkerRequestTimeout)
			defer parseProbeCancel()
			parseTraceID := buildRuntime2StatusWorkerTraceID(parseRegionID, parseCount, parseProbe)

			parseResult, parseErr := interop.RequestWorkerDecoded[runtime2StatusWorkerRequest, struct{}, runtime2StatusWorkerResult](
				parseProbeCtx,
				parseWorker,
				getRuntime2StatusWorkerRequestName,
				runtime2StatusWorkerRequest{
					RegionID:   parseRegionID,
					Count:      parseCount,
					Probe:      parseProbe + 1,
					Tone:       parseTone,
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
				fmt.Printf("[runtime2-status/runtime2][warn] probe %d failed trace=%s: %v\n", parseProbe+1, parseTraceID, parseErr)
				return
			}

			parseExpectedProbe := parseProbe + 1
			parseResultRegionID := strings.TrimSpace(parseResult.RegionID)
			if parseResult.Probe != parseExpectedProbe || parseResult.Count != parseCount || parseResultRegionID != parseRegionID {
				fmt.Printf(
					"[runtime2-status/runtime2][warn] probe mismatch expected{probe=%d,region=%s,count=%d} got{probe=%d,region=%s,count=%d} worker=%s trace=%s\n",
					parseExpectedProbe,
					parseRegionID,
					parseCount,
					parseResult.Probe,
					parseResultRegionID,
					parseResult.Count,
					strings.TrimSpace(parseResult.Worker),
					strings.TrimSpace(parseResult.GetTraceID),
				)
			}
			parseResultMu.Lock()
			parseReport.SuccessCount++
			parseReport.Results[parseProbe] = parseResult
			parseResultMu.Unlock()
		}(parseProbeIndex, parseWorkers[parseProbeIndex])
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
func startRuntime2StatusWorkerBatch(parseParentCtx context.Context, parseRegionID string, parseCount int, parseWorkers []interop.Worker, parseFleetSnapshotRef ui.Ref[runtime2StatusWorkerFleetState], parseMetricsRef ui.Ref[runtime2StatusWorkerMetricsState]) context.CancelFunc {
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

		parseResults, parseErr := requestRuntime2StatusWorkerFleet(parseCtx, parseWorkers, parseRegionID, parseCount, getRuntime2StatusWorkerWorkScale)
		parseBatchDuration, parseBatchIterations, parseBatchDigestXOR := buildRuntime2StatusWorkerBatchStats(parseResults.Results)
		if parseCtx.Err() != nil {
			fmt.Printf("[runtime2-status/runtime2][warn] worker batch cancelled region=%s count=%d reason=%v\n", parseRegionID, parseCount, parseCtx.Err())
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

// handleRuntime2StatusWorkerFleetEffect boots and tears down the worker fleet for the example panel.
func handleRuntime2StatusWorkerFleetEffect(parseRegionID string, parseCount int, parseShouldBoot bool, parseWorkersRef ui.Ref[[]interop.Worker], parseFleetSnapshotRef ui.Ref[runtime2StatusWorkerFleetState], parseMetricsRef ui.Ref[runtime2StatusWorkerMetricsState]) {
	ui.UseEffect(func() func() {
		if !parseShouldBoot {
			return nil
		}
		if len(parseWorkersRef.Get()) != 0 {
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
			parsePrevious.LastEventText = "booting runtime2 worker fleet"
		})
		fmt.Printf("[runtime2-status/runtime2] worker fleet boot starting region=%s workers=%d\n", parseRegionID, getRuntime2StatusWorkerCount)
		go func() {
			parseWorkers, parseErr := openRuntime2StatusWorkerFleet(parseCtx)
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
					parsePrevious.LastEventText = "runtime2 worker fleet boot failed"
				})
				parseFleetSnapshotRef.Set(runtime2StatusWorkerFleetState{
					IsLoading: false,
					ErrorText: parseErr.Error(),
					Results:   nil,
				})
				fmt.Printf("[runtime2-status/runtime2][error] worker fleet boot failed region=%s workers=%d duration=%s error=%v\n", parseRegionID, getRuntime2StatusWorkerCount, time.Since(parseOpenStartedAt), parseErr)
				return
			}
			if parseCtx.Err() != nil {
				_ = closeRuntime2StatusWorkerFleet(parseWorkers)
				return
			}
			parseWorkersRef.Set(parseWorkers)
			applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
				parsePrevious.IsBooting = false
				parsePrevious.IsReady = true
				parsePrevious.BootSuccessCount++
				parsePrevious.LastBootDuration = time.Since(parseOpenStartedAt)
				parsePrevious.LastRegionID = parseRegionID
				parsePrevious.LastWorkerCount = getRuntime2StatusWorkerCount
				parsePrevious.LastErrorText = ""
				parsePrevious.LastEventText = "runtime2 worker fleet ready"
			})
			startRuntime2StatusWorkerBatch(parseCtx, parseRegionID, parseCount, append([]interop.Worker(nil), parseWorkers...), parseFleetSnapshotRef, parseMetricsRef)
			fmt.Printf("[runtime2-status/runtime2] worker fleet ready region=%s workers=%d duration=%s\n", parseRegionID, getRuntime2StatusWorkerCount, time.Since(parseOpenStartedAt))
		}()

		return func() {
			applyRuntime2StatusWorkerMetrics(parseMetricsRef, func(parsePrevious *runtime2StatusWorkerMetricsState) {
				parsePrevious.IsBooting = false
				parsePrevious.IsReady = false
				parsePrevious.LastRegionID = parseRegionID
				parsePrevious.LastWorkerCount = 0
				parsePrevious.LastEventText = "runtime2 worker fleet closed"
			})
			fmt.Printf("[runtime2-status/runtime2] worker fleet closing region=%s\n", parseRegionID)
			parseCancel()
			if parseWorkers := parseWorkersRef.Get(); len(parseWorkers) != 0 {
				parseWorkersRef.Set(nil)
				if parseErr := closeRuntime2StatusWorkerFleet(parseWorkers); parseErr != nil {
					fmt.Printf("[runtime2-status/runtime2][warn] worker fleet close returned error region=%s: %v\n", parseRegionID, parseErr)
				}
			}
		}
	}, parseRegionID, parseShouldBoot)
}

// handleRuntime2StatusWorkerRefreshEffect refreshes the worker panel whenever the owner count or fleet readiness changes.
func handleRuntime2StatusWorkerRefreshEffect(parseRegionID string, parseCount int, parseShouldBoot bool, parseWorkersRef ui.Ref[[]interop.Worker], parseFleetSnapshotRef ui.Ref[runtime2StatusWorkerFleetState], parseMetricsRef ui.Ref[runtime2StatusWorkerMetricsState]) {
	ui.UseEffect(func() func() {
		if !parseShouldBoot {
			return nil
		}
		parseWorkers := parseWorkersRef.Get()
		if len(parseWorkers) == 0 {
			return nil
		}
		if len(parseWorkers) != getRuntime2StatusWorkerCount {
			fmt.Printf(
				"[runtime2-status/runtime2][warn] skipping worker batch because fleet size mismatch region=%s have=%d want=%d\n",
				parseRegionID,
				len(parseWorkers),
				getRuntime2StatusWorkerCount,
			)
			return nil
		}

		return startRuntime2StatusWorkerBatch(context.Background(), parseRegionID, parseCount, append([]interop.Worker(nil), parseWorkers...), parseFleetSnapshotRef, parseMetricsRef)
	}, parseCount, parseShouldBoot)
}

// renderRuntime2StatusWorkerFleet renders the Go WASM worker fleet panel and the latest probe results.
func renderRuntime2StatusWorkerFleet(parseProps runtime2StatusWorkerFleetProps) ui.Node {
	parseShouldBoot := parseProps.Count != 0
	parseWorkersRef := ui.UseRef[[]interop.Worker](nil)
	parseFleetAtom := state.UseAtom(buildRuntime2StatusWorkerFleetAtomID(parseProps.RegionID), runtime2StatusWorkerFleetState{IsLoading: false})
	parseFleetSnapshotRef := ui.UseRef(runtime2StatusWorkerFleetState{IsLoading: false})
	parseMetricsRef := ui.UseRef(runtime2StatusWorkerMetricsState{})
	parseRenderCount := trackRuntime2StatusRenderCount("runtime2-worker-fleet")
	handleRuntime2StatusWorkerFleetEffect(parseProps.RegionID, parseProps.Count, parseShouldBoot, parseWorkersRef, parseFleetSnapshotRef, parseMetricsRef)
	handleRuntime2StatusWorkerRefreshEffect(parseProps.RegionID, parseProps.Count, parseShouldBoot, parseWorkersRef, parseFleetSnapshotRef, parseMetricsRef)

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
		ClassStr("rounded-[28px] border border-emerald-300/20 bg-emerald-400/10 p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.24em] text-emerald-100"),
			Text("Runtime2 Worker Fleet"),
		),
		H2(
			ClassStr("mt-4 text-3xl font-black tracking-tight text-white"),
			Text("8 runtime2 Go WASM workers"),
		),
		P(
			ClassStr("mt-4 text-sm leading-7 text-slate-300"),
			Text("The main runtime2 WASM opens eight dedicated Go WASM workers, then fans out one CPU-bound probe per lane whenever the counter changes. The fleet stays in standby at count 0 to avoid cold-load churn, and telemetry view updates are manual to avoid app-shell rerender spam."),
		),
		P(
			ClassStr("mt-3 text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
			Textf("Render pass #%d", parseRenderCount),
		),
		Button(
			OnClick(parseRefreshTelemetryView),
			ClassStr("mt-4 rounded-xl border border-white/10 bg-white/[0.05] px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-slate-200 transition-colors hover:bg-white/[0.08]"),
			Text("Refresh telemetry view"),
		),
		renderRuntime2StatusWorkerMetrics(parseMetrics),
	}
	if parseFleet.IsLoading {
		parseFleetNodes = append(parseFleetNodes,
			P(
				ClassStr("mt-4 text-sm leading-7 text-emerald-50/90"),
				Text("Booting the worker fleet and waiting for the first probe results."),
			),
		)
	}
	if !parseShouldBoot && len(parseFleet.Results) == 0 && parseFleet.ErrorText == "" {
		parseFleetNodes = append(parseFleetNodes,
			P(
				ClassStr("mt-4 text-sm leading-7 text-slate-300"),
				Text("Standby: increment or decrement Owner State to activate the worker fleet."),
			),
		)
	}
	if parseFleet.ErrorText != "" {
		parseFleetNodes = append(parseFleetNodes,
			Div(
				ClassStr("mt-5 rounded-2xl border border-rose-300/20 bg-rose-400/10 p-4"),
				P(
					ClassStr("text-xs font-semibold uppercase tracking-[0.22em] text-rose-100"),
					Text("Worker error"),
				),
				P(
					ClassStr("mt-3 text-sm leading-7 text-rose-50/90"),
					Text(parseFleet.ErrorText),
				),
			),
		)
	}
	if len(parseFleet.Results) == 0 && parseFleet.ErrorText == "" && !parseFleet.IsLoading {
		parseFleetNodes = append(parseFleetNodes,
			P(
				ClassStr("mt-4 text-sm leading-7 text-slate-300"),
				Text("No worker probe results are available yet."),
			),
		)
	}
	if len(parseFleet.Results) > 0 {
		parseWorkerCards := make([]interface{}, 0, len(parseFleet.Results))
		for _, parseResult := range parseFleet.Results {
			parseWorkerCards = append(parseWorkerCards,
				Div(
					ClassStr("rounded-2xl border border-white/10 bg-slate-950/55 p-4"),
					P(
						ClassStr("text-xs font-semibold uppercase tracking-[0.22em] text-emerald-100"),
						Textf("Probe %d", parseResult.Probe),
					),
					P(
						ClassStr("mt-2 text-sm font-semibold text-white"),
						Text(parseResult.Worker),
					),
					P(
						ClassStr("mt-2 text-sm leading-7 text-slate-300"),
						Text(parseResult.Summary),
					),
					P(
						ClassStr("mt-2 text-xs leading-6 text-slate-400"),
						Textf("trace %s | iterations %d | worker CPU %dms | digest %d", parseResult.GetTraceID, parseResult.GetWorkIterations, parseResult.GetWorkDurationMS, parseResult.GetWorkDigest),
					),
					P(
						ClassStr("mt-3 text-xs leading-6 text-slate-400"),
						Textf("Region %s | count %d | tone %s", parseResult.RegionID, parseResult.Count, parseResult.Tone),
					),
				),
			)
		}
		parseFleetNodes = append(parseFleetNodes, append([]interface{}{
			ClassStr("mt-5 grid gap-3"),
		}, parseWorkerCards...)...)
	}
	return Div(parseFleetNodes...)
}
