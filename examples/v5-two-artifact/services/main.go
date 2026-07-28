//go:build js && wasm

// Command services is the domain-worker half of v5's two-artifact packaging
// (plan item P3.10).
//
// This is where the engine lives. It imports db/offthread/server, which imports
// db/sqlite, which embeds wazero — roughly a megabyte of wasm interpreter that
// must not be in app.wasm. It also runs the command runtime and the delta
// publication engine, because both belong beside the data they operate on.
//
// M10 measures this binary's size and its time to first command.
//
// The protocol is the smallest thing that can carry a typed command and its
// reply. Both directions are keyed by a request id, which is what lets several
// commands be in flight at once without a reply reaching the wrong caller:
//
//	in   {id, name, request}
//	out  {id, payload}        success
//	out  {id, err}            the domain refused
//	out  {ready:true}         posted once, when this handler exists
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/db/offthread"
	"github.com/monstercameron/GoWebComponents/v5/db/offthread/server"
	"github.com/monstercameron/GoWebComponents/v5/db/sqlite"
	"github.com/monstercameron/GoWebComponents/v5/delta"
	"github.com/monstercameron/GoWebComponents/v5/domain"
)

// commandHandler runs one named command and returns its encoded result.
type commandHandler func(parseCtx context.Context, parseRequest []byte) ([]byte, error)

type addRowArgs struct {
	Name string `json:"name"`
}

type addRowResult struct {
	ID string `json:"id"`
}

type removeRowArgs struct {
	ID string `json:"id"`
}

// worker holds everything the domain half owns. None of it exists in app.wasm.
type worker struct {
	db      *sqlite.DB
	server  *server.Server
	runtime *domain.Runtime
	engine  *delta.Engine

	mutex  sync.Mutex
	nextID int
}

// handlers maps wire names to implementations.
//
// The names here are the other half of projection.Registry.Verify on the app
// side: the app declares what it intends to call, this declares what exists, and
// the two are checked against each other at startup rather than on the first
// click of a rarely-used feature.
func (parseWorker *worker) handlers() map[string]commandHandler {
	return map[string]commandHandler{
		"addRow":    parseWorker.handleAddRow,
		"removeRow": parseWorker.handleRemoveRow,
		"heavyWork": parseWorker.handleHeavyWork,
	}
}

func (parseWorker *worker) handleAddRow(parseCtx context.Context, parseRequest []byte) ([]byte, error) {
	var parseArgs addRowArgs
	if parseErr := json.Unmarshal(parseRequest, &parseArgs); parseErr != nil {
		return nil, fmt.Errorf("addRow: %w", parseErr)
	}
	if parseArgs.Name == "" {
		return nil, fmt.Errorf("addRow: a name is required")
	}

	parseWorker.mutex.Lock()
	parseWorker.nextID++
	parseRowID := strconv.Itoa(parseWorker.nextID)
	parseWorker.mutex.Unlock()

	// Through the command runtime rather than straight to the database, so the
	// insert inherits P3.4's exactly-once handling: a command id that has already
	// been applied does not apply again.
	if _, parseErr := parseWorker.runtime.Execute(domain.CommandID("addRow:"+parseRowID), func() error {
		_, parseExecErr := parseWorker.db.Exec(parseCtx,
			`INSERT INTO rows_t (id, name, total) VALUES (?, ?, 0)`, parseRowID, parseArgs.Name)
		return parseExecErr
	}); parseErr != nil {
		return nil, fmt.Errorf("addRow: %w", parseErr)
	}

	// Publish AFTER the write commits, so the app's projection can never show a
	// row the database does not have.
	parseWorker.publishRows(parseCtx)

	return json.Marshal(addRowResult{ID: parseRowID})
}

func (parseWorker *worker) handleRemoveRow(parseCtx context.Context, parseRequest []byte) ([]byte, error) {
	var parseArgs removeRowArgs
	if parseErr := json.Unmarshal(parseRequest, &parseArgs); parseErr != nil {
		return nil, fmt.Errorf("removeRow: %w", parseErr)
	}
	if parseArgs.ID == "" {
		return nil, fmt.Errorf("removeRow: an id is required")
	}
	if _, parseErr := parseWorker.runtime.Execute(domain.CommandID("removeRow:"+parseArgs.ID), func() error {
		_, parseExecErr := parseWorker.db.Exec(parseCtx, `DELETE FROM rows_t WHERE id = ?`, parseArgs.ID)
		return parseExecErr
	}); parseErr != nil {
		return nil, fmt.Errorf("removeRow: %w", parseErr)
	}
	parseWorker.publishRows(parseCtx)
	return json.Marshal(struct{}{})
}

type heavyWorkArgs struct {
	Rows int `json:"rows"`
}

// handleHeavyWork is the experiment: a deliberately expensive domain operation,
// run where v5 says domain work belongs.
//
// It does real database writes AND real CPU, because those block differently and
// the thesis has to survive both. If the two-artifact split works, the render
// thread should not be able to tell this is happening.
func (parseWorker *worker) handleHeavyWork(parseCtx context.Context, parseRequest []byte) ([]byte, error) {
	var parseArgs heavyWorkArgs
	if parseErr := json.Unmarshal(parseRequest, &parseArgs); parseErr != nil {
		return nil, fmt.Errorf("heavyWork: %w", parseErr)
	}
	if parseArgs.Rows <= 0 {
		parseArgs.Rows = 2000
	}

	for parseIndex := 0; parseIndex < parseArgs.Rows; parseIndex++ {
		parseWorker.mutex.Lock()
		parseWorker.nextID++
		parseRowID := strconv.Itoa(parseWorker.nextID)
		parseWorker.mutex.Unlock()

		// CPU alongside the I/O. A pure INSERT loop would mostly measure SQLite,
		// and the question is whether ANY sustained Go work in the worker reaches
		// the render thread.
		parseChurn := 0
		for parseInner := 0; parseInner < 20000; parseInner++ {
			parseChurn = (parseChurn*31 + parseInner) % 1000003
		}

		if _, parseErr := parseWorker.db.Exec(parseCtx,
			`INSERT INTO rows_t (id, name, total) VALUES (?, ?, ?)`,
			parseRowID, fmt.Sprintf("bulk-%s", parseRowID), parseChurn); parseErr != nil {
			return nil, fmt.Errorf("heavyWork insert: %w", parseErr)
		}
	}

	parseWorker.publishRows(parseCtx)
	return json.Marshal(struct {
		Inserted int `json:"inserted"`
	}{Inserted: parseArgs.Rows})
}

// hashPayload derives a delta.Row version from the payload bytes.
//
// FNV-1a rather than a counter because the engine treats an unchanged version as
// "this row did not change". Deriving it from the content makes that true by
// construction: a row republishes exactly when its bytes differ, so a full
// re-publish of an unchanged table produces zero ops and zero messages.
//
// A collision would suppress a real update. For a 64-bit hash over small JSON
// payloads that is not a practical concern here, but it IS the failure mode to
// remember before reusing this on data where a missed update is unacceptable —
// there, carry a real version from the row itself.
func hashPayload(parsePayload []byte) uint64 {
	const parseOffset uint64 = 14695981039346656037
	const parsePrime uint64 = 1099511628211
	parseHash := parseOffset
	for _, parseByte := range parsePayload {
		parseHash = (parseHash ^ uint64(parseByte)) * parsePrime
	}
	return parseHash
}

// publishRows recomputes the published projection and posts the DELTA to the app.
//
// This is the return half of the two-artifact split, and it is the half that was
// missing: without it the app can tell the domain to do something but can never
// learn what the domain now knows. The command reply carries an id, not state, so
// a page rendering from a projection stayed empty forever while every command
// succeeded — a failure that produces no error anywhere.
//
// Deltas rather than a snapshot, which is the entire reason delta.Engine exists:
// re-sending 20,000 rows because one changed is what makes a worker-backed list
// feel worse than no worker at all. The engine holds one integer per key and
// resends payloads only for inserts and updates — a reorder or a delete carries no
// data.
//
// Posted WITHOUT a request id. Ids correlate replies to waiting callers; this is
// unsolicited, so it is tagged by shape ("ops") and the app routes on that. Giving
// it id 0 would look like a reply to an unknown request, which is exactly how the
// worker-death path is signalled.
func (parseWorker *worker) publishRows(parseCtx context.Context) {
	parseCursor, parseErr := parseWorker.db.Query(parseCtx, `SELECT id, name, total FROM rows_t ORDER BY id`)
	if parseErr != nil {
		// Reported, not swallowed: a publication that silently stops leaves the UI
		// frozen on stale data with nothing to indicate it.
		js.Global().Get("console").Call("error", "publishRows query failed: "+parseErr.Error())
		return
	}
	defer parseCursor.Close()

	type publishedRow struct {
		Name  string `json:"name"`
		Total int    `json:"total"`
	}
	parseRows := []delta.Row{}
	for parseCursor.Next() {
		var parseID, parseName string
		var parseTotal int
		if parseScanErr := parseCursor.Scan(&parseID, &parseName, &parseTotal); parseScanErr != nil {
			js.Global().Get("console").Call("error", "publishRows scan failed: "+parseScanErr.Error())
			return
		}
		parseEncoded, parseEncodeErr := json.Marshal(publishedRow{Name: parseName, Total: parseTotal})
		if parseEncodeErr != nil {
			js.Global().Get("console").Call("error", "publishRows encode failed: "+parseEncodeErr.Error())
			return
		}
		parseRows = append(parseRows, delta.Row{
			Key: delta.Key(parseID),
			// Version is the producer's responsibility and the engine's ONLY way to
			// notice an update. Hashing the payload means it changes exactly when the
			// data changes: a counter would republish unchanged rows, and a constant
			// would never republish changed ones.
			Version: hashPayload(parseEncoded),
			Payload: parseEncoded,
		})
	}
	if parseRowsErr := parseCursor.Err(); parseRowsErr != nil {
		js.Global().Get("console").Call("error", "publishRows iterate failed: "+parseRowsErr.Error())
		return
	}

	parseOps, parseOpsBuildErr := parseWorker.engine.Publish(parseRows)
	if parseOpsBuildErr != nil {
		js.Global().Get("console").Call("error", "publishRows delta failed: "+parseOpsBuildErr.Error())
		return
	}
	if len(parseOps) == 0 {
		// Nothing changed. Saying so costs a message; saying nothing costs nothing
		// and is correct — the app's projection already matches.
		return
	}

	parseEncodedOps, parseOpsErr := json.Marshal(parseOps)
	if parseOpsErr != nil {
		js.Global().Get("console").Call("error", "publishRows marshal ops failed: "+parseOpsErr.Error())
		return
	}
	parseMessage := js.Global().Get("Object").New()
	parseMessage.Set("ops", string(parseEncodedOps))
	js.Global().Call("postMessage", parseMessage)
}

// postReply answers one request. Every path through the handler must reach this
// exactly once: a request with no reply leaves its caller waiting for something
// that will never arrive, which presents as a hung page rather than an error.
func postReply(parseRequestID float64, parsePayload []byte, parseErr error) {
	parseMessage := js.Global().Get("Object").New()
	parseMessage.Set("id", parseRequestID)
	if parseErr != nil {
		parseMessage.Set("err", parseErr.Error())
	} else {
		parseMessage.Set("payload", string(parsePayload))
	}
	js.Global().Call("postMessage", parseMessage)
}

func main() {
	parseCtx := context.Background()

	parseDB, parseErr := sqlite.Open(parseCtx, sqlite.Options{
		Name:        "v5-two-artifact",
		Persistence: sqlite.IndexedDB,
	})
	if parseErr != nil {
		panic(parseErr)
	}
	defer parseDB.Close()

	if _, parseErr := parseDB.Exec(parseCtx,
		`CREATE TABLE IF NOT EXISTS rows_t (id TEXT PRIMARY KEY, name TEXT, total INTEGER)`); parseErr != nil {
		panic(parseErr)
	}

	parseServer, parseServerErr := server.New(parseDB)
	if parseServerErr != nil {
		panic(parseServerErr)
	}

	parseWorker := &worker{
		db:      parseDB,
		server:  parseServer,
		runtime: domain.NewRuntime(domain.NewMemoryCheckpointStore()),
		engine:  delta.New(),
	}

	// Long bulk commands must return to this message loop, or a cancel posted
	// while one is running waits behind the very run it is meant to stop. This is
	// the mechanism BulkCommand.YieldEvery needs; the domain layer deliberately
	// does not know how a turn is given back.
	parseWorker.runtime.SetYield(func() {
		parseDone := make(chan struct{})
		var parseCallback js.Func
		parseCallback = js.FuncOf(func(js.Value, []js.Value) any {
			parseCallback.Release()
			close(parseDone)
			return nil
		})
		js.Global().Call("setTimeout", parseCallback, 0)
		<-parseDone
	})

	parseHandlers := parseWorker.handlers()

	js.Global().Set("onmessage", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		if len(parseArgs) == 0 {
			return nil
		}
		parseData := parseArgs[0].Get("data")
		if parseData.IsUndefined() || parseData.IsNull() {
			return nil
		}
		parseRequestID := parseData.Get("id").Float()
		parseName := parseData.Get("name").String()
		parseRequest := []byte(parseData.Get("request").String())

		parseHandler, hasHandler := parseHandlers[parseName]
		if !hasHandler {
			// Answered rather than ignored. Dropping an unknown command would
			// leave its caller waiting forever for a reply that a typo
			// guaranteed could never come.
			postReply(parseRequestID, nil, fmt.Errorf("no handler for command %q", parseName))
			return nil
		}

		// On its own goroutine: a handler that touches the database can block,
		// and blocking here would stop this scope from reading any further
		// message — including a cancel for the very command that is blocking.
		go func() {
			defer func() {
				// A panicking handler must still answer. Without this the caller
				// waits forever and the only evidence is a console trace in a
				// worker nobody is looking at.
				if parseRecovered := recover(); parseRecovered != nil {
					postReply(parseRequestID, nil, fmt.Errorf("command %q panicked: %v", parseName, parseRecovered))
				}
			}()
			parsePayload, parseHandlerErr := parseHandler(parseCtx, parseRequest)
			postReply(parseRequestID, parsePayload, parseHandlerErr)
		}()
		return nil
	}))

	// Posted only after onmessage exists. A command sent before this lands in a
	// scope with no handler and is dropped silently, which looks exactly like a
	// worker that is merely slow.
	parseReady := js.Global().Get("Object").New()
	parseReady.Set("ready", true)
	js.Global().Call("postMessage", parseReady)

	// The offthread server is what lets the render thread run queries without an
	// engine of its own; it is reachable through the same protocol.
	_ = offthread.OpFlush

	select {}
}
