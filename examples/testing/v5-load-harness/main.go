//go:build js && wasm

// Command v5-load-harness is the subject app for the v5 responsiveness harness
// (plan item P0.2).
//
// It exists to be perturbed. Background workloads run concurrently while a
// foreground probe drives a real interaction, and the harness measures ONLY the
// probe. The whole v5 thesis is that the workloads must fail to affect it:
//
//	heavier work takes longer to complete, never longer to paint
//
// §1.2 of the plan defines the scenario this implements:
//   - bulk import of 50k rows into SQLite
//   - a full-text re-index over those rows
//   - a 2MB JSON fetch + decode every 3s
//   - a 5k-row virtualized table under scroll and filter
//   - continuous typing in a filter input (the latency probe)
//
// The JS surface the harness expects:
//
//	window.__gwcV5Workloads = { import, reindex, fetch, ... }  start/stop/stats
//	window.__gwcV5Probes    = { typing, filter }               run(durationMs)
//	window.__gwcV5Probe     = () => string                     (probe.go)
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	goruntime "runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/gcpacing"
	h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/projection"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// rowCount and windowSize size the table the probe interacts with. 5k rows is
// far past what fits on screen, which is the point: without windowing the
// commit cost scales with the dataset, and P4.1 is a Phase 3 prerequisite
// precisely because this scenario exposes that.
const (
	rowCount   = 5000
	windowSize = 40
)

// The background workloads now run in services.wasm, not here.
//
// That relocation IS what this harness measures. In v4 all three ran on the
// render thread and produced a loaded p95 frame of 1633 ms against a 16.8 ms
// idle frame — a 97x gap. This binary keeps only the render-thread half: the
// table, the probe, and a cache of the worker's progress.
//
// Progress is PUSHED by the worker rather than polled, so reading stats never
// costs a round trip. A poll per frame would reintroduce exactly the
// main-thread wait the architecture exists to remove.

// workloadStats is the render thread's cached view of one worker workload.
type workloadStats struct {
	mu        sync.Mutex
	completed int
	errText   string
}

func (parseStats *workloadStats) record(parseCompleted int, parseErrText string) {
	parseStats.mu.Lock()
	defer parseStats.mu.Unlock()
	parseStats.completed = parseCompleted
	if parseErrText != "" {
		parseStats.errText = parseErrText
	}
}

func (parseStats *workloadStats) snapshot() (int, string) {
	parseStats.mu.Lock()
	defer parseStats.mu.Unlock()
	return parseStats.completed, parseStats.errText
}

var (
	importStats  = &workloadStats{}
	reindexStats = &workloadStats{}
	decodeStats  = &workloadStats{}
)

func statsByName(parseName string) (*workloadStats, bool) {
	switch parseName {
	case "import":
		return importStats, true
	case "reindex":
		return reindexStats, true
	case "decode":
		return decodeStats, true
	default:
		return nil, false
	}
}

// domainWorker is the services.wasm worker this binary drives.
var (
	domainWorker     js.Value
	domainWorkerOnce sync.Once
	domainWorkerErr  string
)

// workerMessageCount counts progress messages delivered to the RENDER thread.
//
// Relocating work to a worker moves the compute, not the notification. Every
// message is a wasm callback plus one boundary crossing per field read, on the
// thread the whole architecture exists to protect, so a chatty worker can cost
// more than the work it took away. Counting it is the difference between
// knowing that and assuming it.
var workerMessageCount atomic.Int64

// startDomainWorker creates the worker and wires its progress messages.
//
// Called once, at startup, so the worker's wasm is instantiating while the app
// renders its first frame rather than after the harness asks for work.
func startDomainWorker() {
	domainWorkerOnce.Do(func() {
		parseWorkerCtor := js.Global().Get("Worker")
		if parseWorkerCtor.IsUndefined() {
			domainWorkerErr = "this context has no Worker constructor"
			return
		}
		domainWorker = parseWorkerCtor.New("./worker.js")

		domainWorker.Set("onmessage", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
			workerMessageCount.Add(1)
			if len(parseArgs) == 0 {
				return nil
			}
			parseData := parseArgs[0].Get("data")
			if parseData.IsUndefined() || parseData.IsNull() {
				return nil
			}
			parseName := parseData.Get("workload").String()
			parseErrText := ""
			if parseErr := parseData.Get("err"); !parseErr.IsUndefined() && !parseErr.IsNull() {
				parseErrText = parseErr.String()
			}
			if parseStats, hasStats := statsByName(parseName); hasStats {
				parseStats.record(parseData.Get("completed").Int(), parseErrText)
			} else if parseErrText != "" {
				// A worker-level failure carries no workload name. Recording it
				// against all three keeps a broken worker from reading as three
				// workloads that simply did nothing.
				importStats.record(0, parseErrText)
				reindexStats.record(0, parseErrText)
				decodeStats.record(0, parseErrText)
			}
			return nil
		}))

		domainWorker.Set("onerror", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
			domainWorkerErr = "worker error"
			importStats.record(0, domainWorkerErr)
			reindexStats.record(0, domainWorkerErr)
			decodeStats.record(0, domainWorkerErr)
			return nil
		}))
	})
}

// postToWorker sends one start/stop instruction.
func postToWorker(parseOp string, parseName string) {
	startDomainWorker()
	if domainWorker.IsUndefined() {
		if parseStats, hasStats := statsByName(parseName); hasStats {
			parseStats.record(0, domainWorkerErr)
		}
		return
	}
	parseMessage := js.Global().Get("Object").New()
	parseMessage.Set("op", parseOp)
	parseMessage.Set("workload", parseName)
	domainWorker.Call("postMessage", parseMessage)
}

// yieldToLoop hands control back to the JS event loop so background work
// interleaves with rendering instead of monopolizing the thread.
func yieldToLoop() {
	parseDone := make(chan struct{})
	var parseFn js.Func
	parseFn = js.FuncOf(func(js.Value, []js.Value) any {
		parseFn.Release()
		close(parseDone)
		return nil
	})
	js.Global().Call("setTimeout", parseFn, 0)
	<-parseDone
}

// ------------------------------------------------------------------ the app

type row struct {
	ID    int
	Label string
	// LabelLower is precomputed because the filter is case-insensitive and runs
	// over every row on every keystroke.
	//
	// Calling strings.ToLower inside the filter allocated a new string per row
	// per keypress — about 5,000 allocations per keystroke and 200,000 per
	// measured window. That dominated both metrics the harness was failing:
	// the allocation burst is what provokes M7's rare collection, and the scan
	// is a large part of the oninput script time M2 counts.
	//
	// It is a defect in the SUBJECT app, not in the framework, and it mattered
	// because the subject's own inefficiency was being attributed to the runtime
	// under measurement.
	LabelLower string
}

var allRows []row

func init() {
	parseRand := rand.New(rand.NewSource(1))
	allRows = make([]row, rowCount)
	for parseI := range allRows {
		parseLabel := fmt.Sprintf("row-%04d-%c", parseI, 'a'+parseRand.Intn(26))
		allRows[parseI] = row{ID: parseI, Label: parseLabel, LabelLower: strings.ToLower(parseLabel)}
	}
}

// renderApp is the tree the probe interacts with: a filter input over a
// windowed 5k-row table.
//
// The window is what keeps commit cost proportional to the viewport rather than
// the dataset (M4). Without it this scenario cannot pass M1, which is why P4.1
// is a Phase 3 prerequisite.
func renderApp() ui.Node {
	getFilter := ui.UseState("")
	getOffset := ui.UseState(0)

	handleFilter := ui.UseEvent(func(getE ui.Event) {
		getFilter.Set(getE.GetValue())
		getOffset.Set(0)
	})
	handleScroll := ui.UseEvent(func() {
		getOffset.Update(func(getPrev int) int {
			if getPrev+windowSize >= rowCount {
				return 0
			}
			return getPrev + windowSize
		})
	})

	// The input value stays urgent; the LIST it filters does not.
	//
	// Typing must feel immediate, and the 5,000-row scan behind it must not hold
	// the frame that shows the character. UseDeferredValue is the tool v5 built
	// for exactly that split: the input renders from getFilter on the urgent
	// lane, and the expensive derived work follows on a lower one.
	//
	// Without it every keystroke ran the whole filter synchronously inside the
	// oninput handler, which is what the long-frame attribution named.
	getDeferredFilter := ui.UseDeferredValue(getFilter.Get())

	getVisible := ui.UseMemo(func() []row {
		getQuery := strings.ToLower(getDeferredFilter)
		getOut := make([]row, 0, windowSize)
		getSkipped := 0
		for _, getRow := range allRows {
			if getQuery != "" && !strings.Contains(getRow.LabelLower, getQuery) {
				continue
			}
			if getSkipped < getOffset.Get() {
				getSkipped++
				continue
			}
			getOut = append(getOut, getRow)
			if len(getOut) == windowSize {
				break
			}
		}
		return getOut
	}, getDeferredFilter, getOffset.Get())

	getCells := make([]any, 0, len(getVisible))
	for _, getRow := range getVisible {
		// WithKey so the reconciler takes the keyed path; an unkeyed 5k-row
		// window would diff positionally and hide the cost P4.1 targets.
		getCells = append(getCells, h.WithKey(h.Li(getRow.Label), strconv.Itoa(getRow.ID)))
	}

	return h.Main(
		h.Attr("data-filter", getFilter.Get()),
		h.Attr("data-offset", strconv.Itoa(getOffset.Get())),
		h.Attr("data-deferred-filter", getDeferredFilter),
		h.Section(
			h.H2("Workload subject"),
			h.Label(h.For("filter"), "Filter rows"),
			h.Input(h.ID("filter"), h.Value(getFilter.Get()), h.OnInput(handleFilter)),
			h.Button(h.ID("scroll"), h.Type("button"), h.OnClick(handleScroll), "advance window"),
		),
		h.Ul(append([]any{h.ID("rows")}, getCells...)...),
	)
}

// -------------------------------------------------------------- JS bindings

// workloadObject exposes one worker-backed workload with the SAME JS shape the
// harness already drives.
//
// The contract is deliberately unchanged: start, stop, stats. Only the
// implementation moved. Changing the surface at the same time as the
// architecture would make any measured difference impossible to attribute.
func workloadObject(parseName string, parseStats *workloadStats) js.Value {
	parseObj := js.Global().Get("Object").New()
	parseObj.Set("name", parseName)
	parseObj.Set("start", js.FuncOf(func(js.Value, []js.Value) any {
		postToWorker("start", parseName)
		return nil
	}))
	parseObj.Set("stop", js.FuncOf(func(js.Value, []js.Value) any {
		postToWorker("stop", parseName)
		return nil
	}))
	parseObj.Set("stats", js.FuncOf(func(js.Value, []js.Value) any {
		// Answered from the cache the worker pushes into. No round trip, which
		// is the point — the harness reads this between frames.
		parseCompleted, parseErrText := parseStats.snapshot()
		parseSnapshot := js.Global().Get("Object").New()
		parseSnapshot.Set("completed", parseCompleted)
		parseSnapshot.Set("err", parseErrText)
		return parseSnapshot
	}))
	return parseObj
}

// registerWorkloads exposes the background jobs.
func registerWorkloads() {
	parseObj := js.Global().Get("Object").New()
	parseObj.Set("import", workloadObject("import", importStats))
	parseObj.Set("reindex", workloadObject("reindex", reindexStats))
	parseObj.Set("decode", workloadObject("decode", decodeStats))
	js.Global().Set("__gwcV5Workloads", parseObj)
}

// registerProbes exposes the measured interactions.
//
// Probes drive REAL DOM events rather than calling state setters directly, so
// the Event Timing entries the harness reads have genuine interactionIds. A
// probe that mutated state behind the DOM's back would produce no interaction
// records at all, and M3 would silently measure nothing.
func registerProbes() {
	parseObj := js.Global().Get("Object").New()

	parseObj.Set("typing", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		parseDuration := 4000
		if len(parseArgs) > 0 {
			parseDuration = parseArgs[0].Int()
		}
		return driveTyping(parseDuration)
	}))

	js.Global().Set("__gwcV5Probes", parseObj)

	// Forces N collections and reports each pause, so M7's budget can be checked
	// against what Go's stop-the-world actually costs in wasm rather than against
	// an assumption. If every forced pause on this heap lands near the observed
	// worst, the budget is below the platform's floor and no pacing reaches it.
	js.Global().Set("__gwcV5ForceGC", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		parseRounds := 10
		if len(parseArgs) > 0 && parseArgs[0].Type() == js.TypeNumber {
			parseRounds = parseArgs[0].Int()
		}
		var parseStats goruntime.MemStats
		parsePauses := make([]string, 0, parseRounds)
		for parseI := 0; parseI < parseRounds; parseI++ {
			goruntime.GC()
			goruntime.ReadMemStats(&parseStats)
			parsePauses = append(parsePauses,
				strconv.FormatUint(parseStats.PauseNs[(parseStats.NumGC+255)%256], 10))
		}
		goruntime.ReadMemStats(&parseStats)
		return "{\"pauseNs\":[" + strings.Join(parsePauses, ",") + "],\"heapAllocBytes\":" +
			strconv.FormatUint(parseStats.HeapAlloc, 10) + "}"
	}))

	// M12's pause half: what a RESIDENT projection costs the render thread in
	// collection pauses.
	//
	// The memory half is met and sets Resident(). The pause half is the one that
	// could quietly undo M1 — holding rows on the render thread to avoid worker
	// round trips is only a win if the garbage it creates does not cost more
	// than the round trips saved. It has to be measured in a browser: js/wasm
	// marks single-threaded, without native Go's parallel assist, so a native
	// number would be a green check that means nothing.
	js.Global().Set("__gwcV5FillProjection", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		parseRows := 20000
		if len(parseArgs) > 0 && parseArgs[0].Type() == js.TypeNumber {
			parseRows = parseArgs[0].Int()
		}
		return fillResidentProjection(parseRows)
	}))

	// Exposed so a probe can read how much the worker talked back, which is the
	// half of "move the work off-thread" that moving the work does not fix.
	js.Global().Set("__gwcV5WorkerMessages", js.FuncOf(func(js.Value, []js.Value) any {
		return int(workerMessageCount.Load())
	}))
}

// driveTyping holds the measurement window open for the requested duration.
//
// It does NOT synthesize input events any more. Event Timing only records
// entries for TRUSTED events — ones the browser itself originated — so
// dispatchEvent(new Event("input")) produces no interactionId and therefore no
// interaction records at all. The first real harness run proved it: M3 came
// back with n=0 while frames were visibly janking, i.e. the metric silently
// measured an empty set.
//
// Real keystrokes now come from the Playwright driver, which types into
// #filter for the whole run. This probe just keeps the window open so the
// driver and the harness stay aligned. Opening the page by hand and typing
// works the same way.
// driveTyping waits out a measured window while EXTERNAL input drives the app.
//
// IT DOES NOT TYPE, AND IT CANNOT.
//
// M3 reads the Event Timing API, which only records TRUSTED events. An event
// synthesised here with dispatchEvent produces no entry at all, so a probe that
// "typed" from page script would generate zero samples while looking like it
// worked. Real keystrokes must come from the automation driver (Playwright's
// keyboard goes through CDP), typing into the #filter input this app renders.
//
// This function used to be exactly the setTimeout below with no explanation, and
// the consequence was severe: every recorded M1/M3 number was taken with NOTHING
// INTERACTING. M3 at least said so — "no interactions recorded" — but M1 compared
// an idle arm against a loaded arm in which the probe also did nothing, and
// reported equivalence. Measured 2026-07-26, once a driver supplied real
// keystrokes, M1 failed 2 runs in 3, M2 rose from 0 to 7-27 long frames, and M7
// went from an unmeasured 0 to a consistent ~10ms.
//
// So: the window still just waits, because waiting is all this side can honestly
// do. What changed is that the harness now REFUSES to report M1 or M3 from a run
// with no interactions, instead of quietly scoring an idle page.
func driveTyping(parseDurationMs int) js.Value {
	parseExecutor := js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		parseResolve := parseArgs[0]
		var parseDone js.Func
		parseDone = js.FuncOf(func(js.Value, []js.Value) any {
			parseDone.Release()
			parseResolve.Invoke()
			return nil
		})
		js.Global().Call("setTimeout", parseDone, parseDurationMs)
		return nil
	})
	return js.Global().Get("Promise").New(parseExecutor)
}

// applyURLScheduling reads v5 flags off the query string so the same page can
// be measured as v4 and as v5 without rebuilding.
//
//	?v5=1                     all three, defaults
//	?paintSplit=1&frameMs=5   individually
func applyURLScheduling() SchedulingChoice {
	parseSearch := js.Global().Get("location").Get("search").String()
	parseParams := js.Global().Get("URLSearchParams").New(parseSearch)

	parseHas := func(parseKey string) bool {
		parseValue := parseParams.Call("get", parseKey)
		return !parseValue.IsNull() && parseValue.String() != "" && parseValue.String() != "0"
	}

	parseAll := parseHas("v5")
	parseChoice := SchedulingChoice{
		PassiveEffectsAfterPaint: parseAll || parseHas("paintSplit"),
		LaneQueues:               parseAll || parseHas("lanes"),
	}
	if parseAll {
		parseChoice.FrameBudgetMs = -1 // the 5ms default
	}
	if parseRaw := parseParams.Call("get", "frameMs"); !parseRaw.IsNull() {
		if parseParsed, parseErr := strconv.ParseFloat(parseRaw.String(), 64); parseErr == nil {
			parseChoice.FrameBudgetMs = parseParsed
		}
	}

	ui.ConfigureScheduling(ui.SchedulingOptions{
		PassiveEffectsAfterPaint: parseChoice.PassiveEffectsAfterPaint,
		FrameBudgetMs:            parseChoice.FrameBudgetMs,
		LaneQueues:               parseChoice.LaneQueues,
	})
	return parseChoice
}

// SchedulingChoice is echoed to JS so a report records which runtime produced
// it. A baseline compared against an unknown configuration is worthless.
type SchedulingChoice struct {
	PassiveEffectsAfterPaint bool
	FrameBudgetMs            float64
	LaneQueues               bool
}

// gcPercentOverride reads a GOGC override from localStorage, or 0 for none.
// residentProjection is held for the lifetime of the page on purpose.
//
// A projection that is filled and dropped measures allocation, not RESIDENCY.
// M12 asks what it costs to KEEP rows on the render thread, so the rows have to
// survive the collections being measured.
var residentProjection *projection.Projection[projectionRow]

// projectionRow is a row shaped like something an application would hold: a few
// strings rather than one integer, because a projection of integers would
// understate the pointer graph the collector has to walk.
type projectionRow struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Note  string `json:"note"`
}

// fillResidentProjection builds a resident projection of the requested size and
// reports what it holds.
func fillResidentProjection(parseRows int) string {
	if parseRows <= 0 {
		residentProjection = nil
		goruntime.GC()
		return `{"resident":0}`
	}

	parseBuilt, parseErr := projection.New(func(parsePayload []byte) (projectionRow, error) {
		var parseRow projectionRow
		return parseRow, json.Unmarshal(parsePayload, &parseRow)
	}, projection.Options{Resident: parseRows})
	if parseErr != nil {
		return `{"resident":0,"err":"` + parseErr.Error() + `"}`
	}

	parseOps := make([]projection.Op, 0, parseRows)
	var parseAnchor projection.Key
	for parseIndex := range parseRows {
		parseKey := projection.Key("row-" + strconv.Itoa(parseIndex))
		parsePayload, _ := json.Marshal(projectionRow{
			ID:    string(parseKey),
			Label: "label-" + strconv.Itoa(parseIndex),
			Note:  "note-" + strconv.Itoa(parseIndex) + "-" + strings.Repeat("x", 24),
		})
		parseOps = append(parseOps, projection.Op{
			Kind: projection.OpInsert, Key: parseKey, AfterKey: parseAnchor, Payload: parsePayload,
		})
		parseAnchor = parseKey
	}
	if parseApplyErr := parseBuilt.Apply(parseOps); parseApplyErr != nil {
		return `{"resident":0,"err":"` + parseApplyErr.Error() + `"}`
	}

	residentProjection = parseBuilt
	return `{"resident":` + strconv.Itoa(parseBuilt.Len()) + `}`
}

// gcMemoryLimitOverride reads a soft memory limit from localStorage, or 0.
func gcMemoryLimitOverride() int64 {
	parseStorage := js.Global().Get("localStorage")
	if !parseStorage.Truthy() {
		return 0
	}
	parseValue := parseStorage.Call("getItem", "gwc:gcmem")
	if !parseValue.Truthy() {
		return 0
	}
	parseLimit, parseErr := strconv.ParseInt(parseValue.String(), 10, 64)
	if parseErr != nil || parseLimit <= 0 {
		return 0
	}
	return parseLimit
}

// gcPercentOverrideOr returns the localStorage GOGC override or a fallback.
func gcPercentOverrideOr(parseFallback int) int {
	if parseOverride := gcPercentOverride(); parseOverride > 0 {
		return parseOverride
	}
	return parseFallback
}

func gcPercentOverride() int {
	parseStorage := js.Global().Get("localStorage")
	if !parseStorage.Truthy() {
		return 0
	}
	parseValue := parseStorage.Call("getItem", "gwc:gogc")
	if !parseValue.Truthy() {
		return 0
	}
	parsePercent, parseErr := strconv.Atoi(parseValue.String())
	if parseErr != nil || parsePercent <= 0 {
		return 0
	}
	return parsePercent
}

func main() {
	// P4.4: the render thread pays for pauses, not for total collection CPU, so
	// it takes the responsive profile. Applied here rather than inside the
	// framework because pacing is process-global — a library that set it would
	// be deciding for an application that may have its own view.
	//
	// localStorage["gwc:gogc"] overrides the percentage, so M7 can be swept
	// against GOGC empirically. Whether a pause budget is missed because the
	// pacing is wrong or because Go/wasm has a floor under it is not answerable
	// from one setting, and guessing which it is picks the wrong fix.
	//
	// localStorage["gwc:gcmem"] sets a soft memory limit in bytes. GOGC alone
	// cannot make collections frequent here: the heap is under a megabyte, so a
	// percentage-of-live target is reached rarely at any setting, and every
	// collection therefore follows a long gap. A gap is what makes one
	// expensive — forced collections cost 0.2-0.5ms warm and 6.3ms cold. A limit
	// triggers on absolute size instead, which is the only knob that can keep
	// the collector warm on a heap this small.
	if parseLimit := gcMemoryLimitOverride(); parseLimit > 0 {
		debug.SetGCPercent(gcPercentOverrideOr(40))
		debug.SetMemoryLimit(parseLimit)
	} else if parseOverride := gcPercentOverride(); parseOverride > 0 {
		debug.SetGCPercent(parseOverride)
	} else if _, _, parseErr := gcpacing.Apply(gcpacing.ProfileResponsive, 0); parseErr != nil {
		js.Global().Get("console").Call("warn", "gc pacing not applied: "+parseErr.Error())
	}

	// Pay the first collection at boot, where a pause is invisible among startup
	// work, instead of leaving it to land mid-session.
	//
	// Measured rather than assumed: forcing 15 collections gives a 6.30ms worst
	// pause on a cold heap and 0.50ms once the collector has run, with means of
	// 0.97ms and 0.21ms. Go's stop-the-world in wasm is not expensive — its
	// FIRST one is. Leaving that to happen during interaction is what put M7 at
	// ~6.4ms against a 3ms budget, and a 6ms stall the user actually sees is
	// worth more than a 6ms stall during a load screen nobody is interacting
	// with.
	//
	// CORRECTION 2026-07-26: ONE collection was not enough, and the reason is
	// specific. A standalone probe recording EVERY cycle's pause rather than the
	// max shows the cost sits at a fixed ordinal — the SECOND collection:
	//
	//	perCycle=[0.0  8.1  0.8 1.0 0.4 0.5 0.6 0.4 0.4 0.9 0.5 ...]
	//	               ^^^ always cycle 2, then sub-millisecond forever
	//
	// Measured identically at 0.07MB and 1MB heaps, with 0 and 8000 live js.Func
	// callbacks, and against both pointer-free and fiber-shaped heaps: ~8ms at
	// cycle 2 and 0.2-1.4ms everywhere after. It is a one-time warmup in Go's
	// wasm collector, not a function of heap size, pointer density, or handler
	// count — which is why every mitigation aimed at those (GOGC 40/20/10, memory
	// limit, arena pre-grow) left M7 unchanged.
	//
	// So warm past it. Three collections, not one: cycle 1 is free, cycle 2 is
	// the expensive one, and the third confirms the collector has settled before
	// the app becomes interactive.
	//
	// HONEST CAVEAT: this did NOT move M7 (measured 9.2-10.8ms over four runs
	// with it, against 9.5-12.5ms without). The cycle-2 warmup is real in
	// isolation but is not what the harness is hitting. Kept because paying a
	// known one-time cost at boot rather than mid-session is correct regardless,
	// and it costs two extra collections on a quiescent heap. Do not read its
	// presence as evidence that M7 was addressed.
	for range 3 {
		goruntime.GC()
	}

	parseChoice := applyURLScheduling()

	parseConfigObj := js.Global().Get("Object").New()
	parseConfigObj.Set("passiveEffectsAfterPaint", parseChoice.PassiveEffectsAfterPaint)
	parseConfigObj.Set("frameBudgetMs", parseChoice.FrameBudgetMs)
	parseConfigObj.Set("laneQueues", parseChoice.LaneQueues)
	js.Global().Set("__gwcV5Config", parseConfigObj)

	registerV5LoadHarnessProbe()

	// Start the domain worker before the first render, so its wasm instantiates
	// while this thread paints rather than after the harness asks for work. A
	// lazily-created worker would put its whole instantiation cost inside the
	// first measured workload and attribute it to the workload.
	startDomainWorker()

	registerWorkloads()
	registerProbes()

	ui.Render(ui.Component(renderDashboard), "#root")
	js.Global().Set("__gwcV5Ready", true)
	interop.KeepAlive()
}
