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
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/db/sqlite"
	h "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// rowCount and windowSize size the table the probe interacts with. 5k rows is
// far past what fits on screen, which is the point: without windowing the
// commit cost scales with the dataset, and P4.1 is a Phase 3 prerequisite
// precisely because this scenario exposes that.
const (
	rowCount   = 5000
	windowSize = 40
)

// ---------------------------------------------------------------- workloads

// workload is one background job. It never renders; it exists to compete with
// the probe for the main thread.
type workload struct {
	name    string
	mu      sync.Mutex
	running bool
	stop    chan struct{}
	// completed counts units of work finished, so a "pass" achieved by doing
	// no background work at all is visible rather than silent.
	completed int
	errText   string
}

func (parseW *workload) begin() (chan struct{}, bool) {
	parseW.mu.Lock()
	defer parseW.mu.Unlock()
	if parseW.running {
		return nil, false
	}
	parseW.running = true
	parseW.stop = make(chan struct{})
	return parseW.stop, true
}

func (parseW *workload) end() {
	parseW.mu.Lock()
	if parseW.running {
		parseW.running = false
		close(parseW.stop)
	}
	parseW.mu.Unlock()
}

func (parseW *workload) tick(parseN int) {
	parseW.mu.Lock()
	parseW.completed += parseN
	parseW.mu.Unlock()
}

func (parseW *workload) fail(parseErr error) {
	parseW.mu.Lock()
	parseW.errText = parseErr.Error()
	parseW.mu.Unlock()
}

func (parseW *workload) snapshot() (int, string) {
	parseW.mu.Lock()
	defer parseW.mu.Unlock()
	return parseW.completed, parseW.errText
}

var (
	importWorkload  = &workload{name: "import"}
	reindexWorkload = &workload{name: "reindex"}
	decodeWorkload  = &workload{name: "decode"}
)

// database is opened once. Memory persistence keeps the harness measuring
// runtime behavior rather than IndexedDB flush cost; durability is P3.6's
// concern, not this scenario's.
var (
	databaseOnce sync.Once
	database     *sqlite.DB
	databaseErr  error
)

func openDatabase() (*sqlite.DB, error) {
	databaseOnce.Do(func() {
		database, databaseErr = sqlite.Open(context.Background(), sqlite.Options{
			Name:        "v5harness",
			Persistence: sqlite.Memory,
		})
		if databaseErr != nil {
			return
		}
		_, databaseErr = database.Exec(context.Background(),
			`CREATE TABLE IF NOT EXISTS rows_t (id INTEGER PRIMARY KEY, label TEXT, body TEXT)`)
	})
	return database, databaseErr
}

// runImport inserts 50k rows in batches, yielding between them so the work is
// genuinely concurrent with the probe rather than one uninterruptible block.
func runImport(parseStop chan struct{}) {
	parseDB, parseErr := openDatabase()
	if parseErr != nil {
		importWorkload.fail(parseErr)
		return
	}
	const batch = 500
	for parseOffset := 0; parseOffset < 50000; parseOffset += batch {
		select {
		case <-parseStop:
			return
		default:
		}
		parseTx := strings.Builder{}
		parseTx.WriteString("INSERT INTO rows_t (label, body) VALUES ")
		for parseI := range batch {
			if parseI > 0 {
				parseTx.WriteString(",")
			}
			fmt.Fprintf(&parseTx, "('row-%d','%s')", parseOffset+parseI, strings.Repeat("x", 48))
		}
		if _, parseErr := parseDB.Exec(context.Background(), parseTx.String()); parseErr != nil {
			importWorkload.fail(parseErr)
			return
		}
		importWorkload.tick(batch)
		yieldToLoop()
	}
}

// runReindex scans every row repeatedly, the read-side counterpart to the
// import's write pressure.
func runReindex(parseStop chan struct{}) {
	parseDB, parseErr := openDatabase()
	if parseErr != nil {
		reindexWorkload.fail(parseErr)
		return
	}
	for {
		select {
		case <-parseStop:
			return
		default:
		}
		parseRows, parseErr := parseDB.Query(context.Background(),
			`SELECT id, label FROM rows_t ORDER BY label LIMIT 2000`)
		if parseErr != nil {
			reindexWorkload.fail(parseErr)
			return
		}
		parseSeen := 0
		for parseRows.Next() {
			var parseID int
			var parseLabel string
			if parseErr := parseRows.Scan(&parseID, &parseLabel); parseErr != nil {
				break
			}
			parseSeen++
		}
		parseRows.Close()
		reindexWorkload.tick(parseSeen)
		yieldToLoop()
	}
}

// runDecodeLoop builds and parses a ~2MB JSON payload every 3s.
//
// It builds the payload locally rather than fetching one: fetch() itself runs
// off the main thread, so the main-thread cost this scenario needs to model is
// the DECODE, not the transfer. Modelling it locally also keeps the harness
// deterministic and offline.
func runDecodeLoop(parseStop chan struct{}) {
	parsePayload := buildLargeJSON()
	for {
		select {
		case <-parseStop:
			return
		case <-time.After(3 * time.Second):
		}
		parseParsed := js.Global().Get("JSON").Call("parse", parsePayload)
		decodeWorkload.tick(parseParsed.Get("items").Length())
		yieldToLoop()
	}
}

func buildLargeJSON() string {
	parseBuilder := strings.Builder{}
	parseBuilder.WriteString(`{"items":[`)
	for parseI := range 12000 {
		if parseI > 0 {
			parseBuilder.WriteString(",")
		}
		fmt.Fprintf(&parseBuilder, `{"id":%d,"name":"item-%d","tags":["a","b","c"],"body":"%s"}`,
			parseI, parseI, strings.Repeat("y", 96))
	}
	parseBuilder.WriteString(`]}`)
	return parseBuilder.String()
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
}

var allRows []row

func init() {
	parseRand := rand.New(rand.NewSource(1))
	allRows = make([]row, rowCount)
	for parseI := range allRows {
		allRows[parseI] = row{ID: parseI, Label: fmt.Sprintf("row-%04d-%c", parseI, 'a'+parseRand.Intn(26))}
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

	getVisible := ui.UseMemo(func() []row {
		getQuery := strings.ToLower(getFilter.Get())
		getOut := make([]row, 0, windowSize)
		getSkipped := 0
		for _, getRow := range allRows {
			if getQuery != "" && !strings.Contains(strings.ToLower(getRow.Label), getQuery) {
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
	}, getFilter.Get(), getOffset.Get())

	getCells := make([]any, 0, len(getVisible))
	for _, getRow := range getVisible {
		// WithKey so the reconciler takes the keyed path; an unkeyed 5k-row
		// window would diff positionally and hide the cost P4.1 targets.
		getCells = append(getCells, h.WithKey(h.Li(getRow.Label), strconv.Itoa(getRow.ID)))
	}

	return h.Main(
		h.Section(
			h.H1("v5 load harness"),
			h.Input(h.ID("filter"), h.Value(getFilter.Get()), h.OnInput(handleFilter)),
			h.Button(h.ID("scroll"), h.Type("button"), h.OnClick(handleScroll), "advance window"),
		),
		h.Ul(append([]any{h.ID("rows")}, getCells...)...),
	)
}

// -------------------------------------------------------------- JS bindings

func workloadObject(parseW *workload, parseRun func(chan struct{})) js.Value {
	parseObj := js.Global().Get("Object").New()
	parseObj.Set("name", parseW.name)
	parseObj.Set("start", js.FuncOf(func(js.Value, []js.Value) any {
		if parseStop, parseOk := parseW.begin(); parseOk {
			go parseRun(parseStop)
		}
		return nil
	}))
	parseObj.Set("stop", js.FuncOf(func(js.Value, []js.Value) any {
		parseW.end()
		return nil
	}))
	parseObj.Set("stats", js.FuncOf(func(js.Value, []js.Value) any {
		parseCompleted, parseErrText := parseW.snapshot()
		parseStats := js.Global().Get("Object").New()
		parseStats.Set("completed", parseCompleted)
		parseStats.Set("err", parseErrText)
		return parseStats
	}))
	return parseObj
}

// registerWorkloads exposes the background jobs.
func registerWorkloads() {
	parseObj := js.Global().Get("Object").New()
	parseObj.Set("import", workloadObject(importWorkload, runImport))
	parseObj.Set("reindex", workloadObject(reindexWorkload, runReindex))
	parseObj.Set("decode", workloadObject(decodeWorkload, runDecodeLoop))
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

func main() {
	parseChoice := applyURLScheduling()

	parseConfigObj := js.Global().Get("Object").New()
	parseConfigObj.Set("passiveEffectsAfterPaint", parseChoice.PassiveEffectsAfterPaint)
	parseConfigObj.Set("frameBudgetMs", parseChoice.FrameBudgetMs)
	parseConfigObj.Set("laneQueues", parseChoice.LaneQueues)
	js.Global().Set("__gwcV5Config", parseConfigObj)

	registerV5LoadHarnessProbe()
	registerWorkloads()
	registerProbes()

	ui.Render(ui.Component(renderApp), "#app")
	js.Global().Set("__gwcV5Ready", true)
	interop.KeepAlive()
}
