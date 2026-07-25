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
package main

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v4/db/offthread"
	"github.com/monstercameron/GoWebComponents/v4/db/offthread/server"
	"github.com/monstercameron/GoWebComponents/v4/db/sqlite"
	"github.com/monstercameron/GoWebComponents/v4/internal/delta"
	"github.com/monstercameron/GoWebComponents/v4/internal/domain"
)

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

	parseServer, parseServerErr := server.New(parseDB)
	if parseServerErr != nil {
		panic(parseServerErr)
	}

	// The command runtime provides P3.4's replay and resume guarantees; the
	// delta engine publishes O(change) updates to the render thread.
	parseRuntime := domain.NewRuntime(domain.NewMemoryCheckpointStore())
	parseEngine := delta.New()

	// A real worker reads postMessage and dispatches here. What matters for
	// packaging is that these symbols live in THIS binary.
	_ = parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpFlush})
	_ = parseRuntime
	_ = parseEngine

	select {}
}
