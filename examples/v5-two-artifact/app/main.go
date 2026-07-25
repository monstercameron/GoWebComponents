//go:build js && wasm

// Command app is the render-thread half of v5's two-artifact packaging
// (plan item P3.10).
//
// The whole point of this binary is what it does NOT contain. It renders, holds
// resident projections, and issues commands — and it imports no database engine,
// so wazero and SQLite never enter app.wasm. M5 measures the result.
//
// The domain half is ../services, which does import the engine and is compiled
// separately into services.wasm for a worker.
//
// It is also the reference for the two rules that make a worker-backed app work
// at all, both of which are easy to get wrong in ways that look like a hung page
// rather than a bug:
//
//	1. Never call a command from a DOM callback. See runCommand.
//	2. Apply the result through ui.PostAsync. See applyResult.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/projection"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// row is what the domain publishes and this binary renders.
type row struct {
	Name  string `json:"name"`
	Total int    `json:"total"`
}

// The commands this app can issue, declared once each.
//
// Declaring them at package scope is what makes a misspelling at a call site an
// undefined identifier rather than a runtime failure in the worker (P3.7 (d)).
var (
	addRow    = projection.Define[addRowArgs, addRowResult]("addRow")
	removeRow = projection.Define[removeRowArgs, struct{}]("removeRow")
)

type addRowArgs struct {
	Name string `json:"name"`
}

type addRowResult struct {
	ID string `json:"id"`
}

type removeRowArgs struct {
	ID string `json:"id"`
}

// workerPoster posts one command to services.wasm.
//
// Poster is deliberately the only thing the transport has to implement:
// correlating a reply back to the request that is waiting for it is
// WorkerClient's job, not the application's, because getting it wrong produces
// a reply delivered to the wrong caller rather than an error.
type workerPoster struct {
	worker js.Value
}

func (parsePoster *workerPoster) Post(parseRequestID uint64, parseName string, parseRequest []byte) error {
	if parsePoster.worker.IsUndefined() || parsePoster.worker.IsNull() {
		return projection.WorkerDeath("the domain worker was never started")
	}
	parseMessage := js.Global().Get("Object").New()
	// float64 because that is what a JS number is. Request ids are small
	// counters, so the 2^53 exact range is not a constraint worth engineering
	// around, but sending a uint64 unconverted would silently truncate.
	parseMessage.Set("id", float64(parseRequestID))
	parseMessage.Set("name", parseName)
	parseMessage.Set("request", string(parseRequest))
	parsePoster.worker.Call("postMessage", parseMessage)
	return nil
}

// startWorker boots services.wasm and wires replies back into the client.
func startWorker(parseClient *projection.WorkerClient, parsePoster *workerPoster) {
	parseWorker := js.Global().Get("Worker").New("worker.js")
	parsePoster.worker = parseWorker

	parseWorker.Set("onmessage", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		if len(parseArgs) == 0 {
			return nil
		}
		parseData := parseArgs[0].Get("data")
		if parseData.IsUndefined() || parseData.IsNull() {
			return nil
		}
		// The worker's boot signal carries no request id; nothing is waiting on
		// it, and delivering it under id 0 would be reported as a reply for an
		// unknown request on every start.
		if parseData.Get("ready").Truthy() {
			return nil
		}

		parseRequestID := uint64(parseData.Get("id").Float())
		if parseErrText := parseData.Get("err").String(); parseErrText != "" {
			// Rejection, not WorkerDeath: the worker answered and refused. That
			// distinction is what tells a retry policy whether spending another
			// attempt could possibly help.
			parseClient.DeliverRejection(parseRequestID, parseErrText)
			return nil
		}
		parseClient.Deliver(parseRequestID, []byte(parseData.Get("payload").String()), nil)
		return nil
	}))

	// A worker that dies takes every in-flight request with it. Without this the
	// callers of those requests wait for a reply that can never arrive, and the
	// page looks hung rather than broken.
	parseWorker.Set("onerror", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		parseReason := "the domain worker failed"
		if len(parseArgs) > 0 {
			if parseMessage := parseArgs[0].Get("message"); parseMessage.Truthy() {
				parseReason = parseMessage.String()
			}
		}
		parseClient.WorkerDied(parseReason)
		return nil
	}))
}

type jsonCodec struct{}

func (jsonCodec) Encode(parseValue any) ([]byte, error) { return json.Marshal(parseValue) }
func (jsonCodec) Decode(parseData []byte, parseTarget any) error {
	return json.Unmarshal(parseData, parseTarget)
}

// runCommand issues a command without blocking the JS event loop.
//
// THE RULE: never call Invoke directly from a DOM event handler.
//
// Invoke waits for the worker's reply. A DOM callback runs ON the JS event
// loop, and the reply arrives AS a message event on that same loop — so waiting
// inside the callback prevents the event that would end the wait from ever
// being dispatched. The page deadlocks, with no error and no failing call to
// point at; it simply stops. That is a property of the environment, not a bug
// in Invoke, and no amount of care inside Invoke can fix it.
//
// Running the wait on a goroutine returns the callback to the loop immediately,
// so the message event is dispatched, Deliver hands the reply over, and this
// goroutine wakes.
func runCommand(parseWork func()) {
	go parseWork()
}

// applyResult moves a worker result back onto the render thread.
//
// THE OTHER RULE: a command result arrives on a goroutine, which is outside the
// frame loop. Writing render state from there mutates it at an arbitrary moment
// relative to whatever the reconciler is doing. PostAsync queues the write to
// one defined point in the frame instead, so a reply cannot land mid-render.
//
// Several writes in one call is not incidental: everything inside a single post
// is applied in one block, so a result that updates three pieces of state
// produces one render rather than three.
func applyResult(parseApply func()) {
	ui.PostAsync(parseApply)
}

func main() {
	parseProjection, parseErr := projection.New(func(parsePayload []byte) (row, error) {
		var parseRow row
		return parseRow, json.Unmarshal(parsePayload, &parseRow)
	}, projection.Options{})
	if parseErr != nil {
		panic(parseErr)
	}

	// The registry check that types cannot do: verify at startup that the worker
	// actually handles every command this binary declares, rather than finding
	// out on the first click of a rarely-used feature.
	parseRegistry := projection.NewRegistry([]string{"addRow", "removeRow"})
	if parseVerifyErr := parseRegistry.Verify(addRow, removeRow); parseVerifyErr != nil {
		panic(parseVerifyErr)
	}

	parsePoster := &workerPoster{}
	parseClient, parseClientErr := projection.NewWorkerClient(parsePoster)
	if parseClientErr != nil {
		panic(parseClientErr)
	}
	startWorker(parseClient, parsePoster)

	parseResilient, parseResilientErr := projection.NewResilient(parseClient, projection.Policy{MaxAttempts: 2})
	if parseResilientErr != nil {
		panic(parseResilientErr)
	}

	// One command, issued the way every command should be: off the event loop,
	// with its result applied through the inbox.
	runCommand(func() {
		parseResult, parseInvokeErr := addRow.Invoke(
			context.Background(), parseResilient, jsonCodec{}, addRowArgs{Name: "first"})
		if parseInvokeErr != nil {
			// Reported rather than swallowed. A command that silently does
			// nothing is the failure mode that costs the most to diagnose.
			js.Global().Get("console").Call("error", "addRow failed: "+parseInvokeErr.Error())
			return
		}
		applyResult(func() {
			js.Global().Get("console").Call("log", "addRow committed "+parseResult.ID)
		})
	})

	parseRows := parseProjection.Rows()
	ui.Render(html.VirtualList(html.VirtualListProps{
		ItemCount:      len(parseRows),
		ItemHeight:     32,
		ViewportHeight: 640,
		Key:            func(parseIndex int) any { return string(parseRows[parseIndex].Key) },
		Render: func(parseIndex int) ui.Node {
			return html.Div(html.Props{Class: "row"},
				html.Text(fmt.Sprintf("%s — %d", parseRows[parseIndex].Value.Name, parseRows[parseIndex].Value.Total)))
		},
	}), "root")

	_ = removeRow
	select {}
}
