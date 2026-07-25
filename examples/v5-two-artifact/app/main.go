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

// workerTransport carries commands to services.wasm over postMessage.
//
// Only the shape matters for packaging: it moves bytes and reports failures in
// the vocabulary projection understands. It never imports a database.
type workerTransport struct {
	worker js.Value
}

func (parseTransport *workerTransport) Send(parseCtx context.Context, parseName string, parseRequest []byte) ([]byte, error) {
	if parseTransport.worker.IsUndefined() {
		return nil, projection.WorkerDeath("the domain worker was never started")
	}
	// A real implementation awaits a reply keyed by a request id. The packaging
	// question this example answers is which SYMBOLS end up in which binary, so
	// the reply plumbing is deliberately out of scope here.
	return nil, projection.WorkerDeath("reply plumbing is out of scope for the packaging example")
}

type jsonCodec struct{}

func (jsonCodec) Encode(parseValue any) ([]byte, error) { return json.Marshal(parseValue) }
func (jsonCodec) Decode(parseData []byte, parseTarget any) error {
	return json.Unmarshal(parseData, parseTarget)
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

	parseTransport := &workerTransport{}
	parseClient, parseClientErr := projection.NewResilient(parseTransport, projection.Policy{MaxAttempts: 2})
	if parseClientErr != nil {
		panic(parseClientErr)
	}

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

	_ = parseClient
	select {}
}
