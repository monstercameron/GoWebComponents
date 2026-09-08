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
//  1. Never call a command from a DOM callback. See runCommand.
//  2. Apply the result through ui.PostAsync. See applyResult.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/projection"
	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/ui"
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
	heavyWork = projection.Define[heavyWorkArgs, heavyWorkResult]("heavyWork")

	// parseVersion is bumped whenever a command commits, so the render tree has
	// something to depend on.
	//
	// A GLOBAL atom rather than a hook, because the write happens on a goroutine
	// handling a worker reply — there is no fiber there, and state.UseAtom would
	// panic. The read side is inside the component, where a fiber does exist.
	//
	// It exists at all because projection.Projection has no change notification:
	// Rows() is its entire surface. If the projection ever grows a Subscribe, this
	// counter and the Set below should be deleted in the same commit — it is a
	// stand-in for a missing signal, not a pattern worth copying.
	parseVersion = state.NewGlobalAtom("v5-two-artifact:commit-version", 0)
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

type heavyWorkArgs struct {
	Rows int `json:"rows"`
}

type heavyWorkResult struct {
	Inserted int `json:"inserted"`
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

// workerReady closes when services.wasm has installed its message handler.
//
// A command posted before that lands in a scope with no onmessage and is dropped
// silently — no error anywhere, and Invoke waits for a reply that was never
// going to come. The services binary announces itself for exactly this reason,
// and an app that ignores the announcement gets a page that loads fine and then
// does nothing.
// parseProjectionRef is the projection the worker's published deltas are applied
// to.
//
// Package-level because the message handler installed by startWorker needs it and
// main owns its construction. Threading it through startWorker's parameters would
// be tidier, but it would also mean the transport knows about the projection —
// and the point of WorkerClient is that the transport carries bytes and correlates
// ids, nothing more.
var parseProjectionRef *projection.Projection[row]

var workerReady = make(chan struct{})

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
		// UNSOLICITED STATE PUBLICATION, checked before the id-bearing paths.
		//
		// This message has no request id because nothing is waiting for it: the
		// domain publishes when its data changes, not when it is asked. Routing on
		// shape rather than on id keeps it out of the reply-correlation table, where
		// an unmatched id is treated as worker death.
		//
		// This is the half of the split that makes the whole thing worth doing.
		// Without it an app can issue commands and read their return values but can
		// never learn what the domain now holds — every list rendered from a
		// projection stays empty while every command succeeds.
		if parseOpsValue := parseData.Get("ops"); parseOpsValue.Truthy() {
			var parseOps []projection.Op
			if parseDecodeErr := json.Unmarshal([]byte(parseOpsValue.String()), &parseOps); parseDecodeErr != nil {
				js.Global().Get("console").Call("error", "delta decode failed: "+parseDecodeErr.Error())
				return nil
			}
			// Applied through PostAsync for the same reason every other reply is: this
			// runs on the JS event loop, and mutating projection state that a render
			// reads from, mid-frame, is the tear the inbox exists to prevent.
			applyResult(func() {
				if parseApplyErr := parseProjectionRef.Apply(parseOps); parseApplyErr != nil {
					// A rejected delta means the two projections have diverged. Silence
					// here leaves a list quietly wrong forever.
					js.Global().Get("console").Call("error", "projection apply failed: "+parseApplyErr.Error())
					return
				}
				parseVersion.Set(parseVersion.Get() + 1)
			})
			return nil
		}

		// The worker's boot signal carries no request id; nothing is waiting on
		// it, and delivering it under id 0 would be reported as a reply for an
		// unknown request on every start.
		if parseData.Get("ready").Truthy() {
			// Closed once. A worker that announced itself twice would otherwise
			// panic the app on the second close, turning a harmless duplicate
			// into a crash.
			select {
			case <-workerReady:
			default:
				close(workerReady)
			}
			return nil
		}

		parseRequestID := uint64(parseData.Get("id").Float())
		// Checked for PRESENCE, not for emptiness. js.Value.String() on an
		// absent field returns the literal "<undefined>", which is a non-empty
		// string — so reading it directly turned every SUCCESSFUL reply into a
		// rejection carrying the text "<undefined>". The reply plumbing looked
		// like a domain that refused everything.
		parseErrValue := parseData.Get("err")
		if parseErrText := parseErrValue.String(); !parseErrValue.IsUndefined() &&
			!parseErrValue.IsNull() && parseErrText != "" {
			// An error with NO request id is not a refusal, it is the worker
			// itself failing — worker.js reports a failed instantiate that way.
			// Delivering it against id 0 finds no waiter and drops it silently,
			// so a services.wasm that never loads presents as a command that
			// never returns, with nothing in the console to point at. Failing
			// every in-flight request is the honest translation.
			if parseRequestID == 0 {
				parseClient.WorkerDied(parseErrText)
				return nil
			}
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
	// Published before startWorker installs the handler that writes to it, so a
	// delta arriving on the worker's very first message cannot find a nil target.
	parseProjectionRef = parseProjection

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
	//
	// It also publishes M10's timing half. The metric is time from navigation
	// to the first command COMPLETING — not to the worker existing — because a
	// worker that has booted but cannot yet answer has not moved the app any
	// closer to being usable. performance.now() is measured from navigation
	// start, so the value needs no arithmetic on the reading side.
	runCommand(func() {
		// Wait for the worker to be able to receive before sending. This is the
		// handshake the whole transport depends on and the easiest thing to
		// leave out, because leaving it out fails only by hanging.
		<-workerReady

		parseResult, parseInvokeErr := addRow.Invoke(
			context.Background(), parseResilient, jsonCodec{}, addRowArgs{Name: "first"})
		if parseInvokeErr != nil {
			// Reported rather than swallowed. A command that silently does
			// nothing is the failure mode that costs the most to diagnose, and
			// a probe waiting on the timing below would otherwise just hang.
			js.Global().Set("__v5FirstCommandError", parseInvokeErr.Error())
			js.Global().Get("console").Call("error", "addRow failed: "+parseInvokeErr.Error())
			return
		}
		applyResult(func() {
			js.Global().Set("__v5FirstCommandMs",
				js.Global().Get("performance").Call("now").Float())
			js.Global().Get("console").Call("log", "addRow committed "+parseResult.ID)
			// THE LAST MILE. Everything above this line is the part the two-artifact
			// split is usually described by — worker boots, command goes out, reply
			// comes back through the inbox. None of it is worth anything until the
			// reply changes what is on the screen, and that step is a state write, not
			// a log line.
			//
			// This example previously ended at the console.log. The transport worked,
			// the command committed, the timing hook fired, there were no errors — and
			// the page rendered an empty <div>, because nothing ever told the runtime
			// that there was new data. It is the same failure shape the M10 write-up
			// describes: it fails by producing nothing.
			//
			// projection.Projection is a passive container: its only accessor is
			// Rows(), with no Subscribe and no change signal. So the app has to hold
			// the rows itself and re-publish them here. That is the honest shape of
			// the API as it stands today.
			parseVersion.Set(parseVersion.Get() + 1)
		})
	})

	// Rendered as a COMPONENT, not as a one-shot snapshot.
	//
	// The old code evaluated parseProjection.Rows() once, at boot, and handed the
	// resulting slice to ui.Render. That slice was empty — the first command was
	// still waiting on the worker handshake — and because ui.Render is not reactive
	// to a plain Go slice, it stayed empty forever.
	//
	// Reading the projection INSIDE the component body, and depending on a state
	// value the async reply bumps, is what makes a worker result appear. The
	// version counter is deliberately crude: it exists because the projection has
	// no change notification to subscribe to, and pretending otherwise would hide
	// the gap rather than document it.
	ui.Render(ui.CreateElement(func() ui.Node {
		// state.UseAtom, NOT parseVersion.Get(). Both read the same atom id, and
		// only this one SUBSCRIBES the fiber.
		//
		// GlobalAtom.Get() reads the registry directly and returns; it does not
		// register the rendering component as a listener, and nothing warns you. Set
		// then "schedules a re-render of every component subscribed via UseAtom" —
		// which, with a bare Get(), is none of them. The result is a component that
		// reads fresh data on every render it happens to do, and never does one.
		//
		// The pairing is the point: GlobalAtom.Set is callable from any goroutine
		// (the worker reply has no fiber), while UseAtom is the reactive read inside
		// a fiber. Same id, two halves.
		_ = state.UseAtom(parseVersion.ID(), 0).Get()
		parseRows := parseProjection.Rows()
		if len(parseRows) == 0 {
			return html.Div(html.Props{Class: "empty"},
				html.Text("Waiting for the domain worker…"))
		}
		return html.VirtualList(html.VirtualListProps{
			ItemCount:      len(parseRows),
			ItemHeight:     32,
			ViewportHeight: 640,
			Key:            func(parseIndex int) any { return string(parseRows[parseIndex].Key) },
			Render: func(parseIndex int) ui.Node {
				return html.Div(html.Props{Class: "row"},
					html.Text(fmt.Sprintf("%s — %d", parseRows[parseIndex].Value.Name, parseRows[parseIndex].Value.Total)))
			},
		})
	}), "#app")

	// THE EXPERIMENT.
	//
	// Two globals, one doing the work where v5 says it belongs and one doing the
	// SAME work on the render thread. A probe can call either and watch frame
	// times, which is the only way to answer "is the thesis achieved" with a number
	// instead of an argument.
	js.Global().Set("__v5RunInWorker", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		parseRows := 400
		if len(parseArgs) > 0 && parseArgs[0].Type() == js.TypeNumber {
			parseRows = parseArgs[0].Int()
		}
		runCommand(func() {
			<-workerReady
			if _, parseErr := heavyWork.Invoke(context.Background(), parseResilient, jsonCodec{},
				heavyWorkArgs{Rows: parseRows}); parseErr != nil {
				js.Global().Get("console").Call("error", "heavyWork failed: "+parseErr.Error())
				return
			}
			applyResult(func() {
				js.Global().Set("__v5WorkerWorkDone", true)
			})
		})
		return nil
	}))

	// The control. Identical arithmetic, run as an ordinary Go loop on the render
	// thread — which in Go/wasm means inside ONE JS event-loop turn, because the Go
	// scheduler cannot preempt across the JS boundary. This is the "before" the
	// two-artifact split exists to remove.
	js.Global().Set("__v5RunOnRenderThread", js.FuncOf(func(_ js.Value, parseArgs []js.Value) any {
		parseRows := 400
		if len(parseArgs) > 0 && parseArgs[0].Type() == js.TypeNumber {
			parseRows = parseArgs[0].Int()
		}
		go func() {
			parseChurn := 0
			for parseIndex := 0; parseIndex < parseRows; parseIndex++ {
				for parseInner := 0; parseInner < 20000; parseInner++ {
					parseChurn = (parseChurn*31 + parseInner) % 1000003
				}
			}
			applyResult(func() {
				js.Global().Set("__v5RenderThreadWorkDone", parseChurn)
			})
		}()
		return nil
	}))

	_ = removeRow
	select {}
}
