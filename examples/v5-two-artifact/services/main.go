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

	"github.com/monstercameron/GoWebComponents/v4/db/offthread"
	"github.com/monstercameron/GoWebComponents/v4/db/offthread/server"
	"github.com/monstercameron/GoWebComponents/v4/db/sqlite"
	"github.com/monstercameron/GoWebComponents/v4/delta"
	"github.com/monstercameron/GoWebComponents/v4/domain"
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
	return json.Marshal(struct{}{})
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
