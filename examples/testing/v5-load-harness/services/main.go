//go:build js && wasm

// Command services is the harness's domain worker (plan items P3.5, P3.10).
//
// It runs the three background workloads §1.2 defines — a 50k-row SQLite
// import, a full-text re-index over them, and a 2MB JSON decode — none of which
// belong on the render thread. That relocation IS v5's thesis: heavier work
// takes longer to complete, never longer to paint.
//
// The v4 harness ran all three on the render thread and measured a loaded p95
// frame of 1633 ms against a 16.8 ms idle frame. This binary exists so the same
// harness can measure the same workloads with them somewhere else.
//
// It speaks a deliberately small protocol, because the point of the measurement
// is where the WORK runs, not how elaborate the messaging is:
//
//	in   {op:"start"|"stop", workload:"import"|"reindex"|"decode"}
//	out  {workload, completed, err}   progress, posted as it happens
package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/db/sqlite"
)

// workload tracks one background job's progress.
//
// Mirrors the app-side type it replaces. The counters live here now because
// this is where the work happens; the render thread only ever sees the numbers.
type workload struct {
	name    string
	mu      sync.Mutex
	running bool
	stop    chan struct{}

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
	parseW.completed = 0
	parseW.errText = ""
	return parseW.stop, true
}

func (parseW *workload) end() {
	parseW.mu.Lock()
	defer parseW.mu.Unlock()
	if !parseW.running {
		return
	}
	parseW.running = false
	close(parseW.stop)
}

func (parseW *workload) tick(parseCount int) {
	parseW.mu.Lock()
	parseW.completed += parseCount
	parseCompleted := parseW.completed
	parseW.mu.Unlock()
	postProgress(parseW.name, parseCompleted, "")
}

func (parseW *workload) fail(parseErr error) {
	parseW.mu.Lock()
	parseW.errText = parseErr.Error()
	parseW.running = false
	parseCompleted := parseW.completed
	parseErrText := parseW.errText
	parseW.mu.Unlock()
	postProgress(parseW.name, parseCompleted, parseErrText)
}

var (
	importWorkload  = &workload{name: "import"}
	reindexWorkload = &workload{name: "reindex"}
	decodeWorkload  = &workload{name: "decode"}
)

// postProgress sends one progress update to the render thread.
//
// Posted on every tick rather than polled, so the render thread never blocks to
// ask. A poll would be a round trip per frame, which is the cost this whole
// architecture exists to remove.
func postProgress(parseName string, parseCompleted int, parseErrText string) {
	parseMessage := js.Global().Get("Object").New()
	parseMessage.Set("workload", parseName)
	parseMessage.Set("completed", parseCompleted)
	parseMessage.Set("err", parseErrText)
	js.Global().Call("postMessage", parseMessage)
}

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

// yieldToLoop hands the worker's event loop a turn.
//
// Still needed here even though nothing paints on this thread: the worker has
// its own message loop, and a workload that never yielded would starve the stop
// message and make the harness unable to end a run.
func yieldToLoop() {
	parseDone := make(chan struct{})
	var parseCallback js.Func
	parseCallback = js.FuncOf(func(js.Value, []js.Value) any {
		parseCallback.Release()
		close(parseDone)
		return nil
	})
	js.Global().Call("setTimeout", parseCallback, 0)
	<-parseDone
}

// runImport inserts 50k rows in batches.
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
// Built locally rather than fetched: fetch() already runs off the render thread,
// so the cost this scenario models is the DECODE. Building it locally also keeps
// the harness deterministic and offline.
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

// workloadByName resolves a workload and its runner.
func workloadByName(parseName string) (*workload, func(chan struct{}), bool) {
	switch parseName {
	case "import":
		return importWorkload, runImport, true
	case "reindex":
		return reindexWorkload, runReindex, true
	case "decode":
		return decodeWorkload, runDecodeLoop, true
	default:
		return nil, nil, false
	}
}

func main() {
	js.Global().Set("onmessage", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		if len(parseArgs) == 0 {
			return nil
		}
		parseData := parseArgs[0].Get("data")
		if parseData.IsUndefined() || parseData.IsNull() {
			return nil
		}

		parseOp := parseData.Get("op").String()
		parseName := parseData.Get("workload").String()
		parseWorkload, parseRun, hasWorkload := workloadByName(parseName)
		if !hasWorkload {
			// Reported rather than ignored: a typo'd workload name would
			// otherwise produce a run whose numbers look valid and measured
			// nothing.
			postProgress(parseName, 0, "unknown workload "+parseName)
			return nil
		}

		switch parseOp {
		case "start":
			if parseStop, parseOk := parseWorkload.begin(); parseOk {
				go parseRun(parseStop)
			}
		case "stop":
			parseWorkload.end()
		default:
			postProgress(parseName, 0, "unknown op "+parseOp)
		}
		return nil
	}))

	// Tell the render thread the worker is ready. Without it the app would
	// start posting work into a scope whose onmessage handler does not exist
	// yet, and those messages are dropped silently.
	parseReady := js.Global().Get("Object").New()
	parseReady.Set("workload", "")
	parseReady.Set("ready", true)
	js.Global().Call("postMessage", parseReady)

	select {}
}
