//go:build js && wasm

package ui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// mountAsyncTestComponent mounts a component with a synchronous mock scheduler
// so scheduled work (effects, state-driven re-renders) executes immediately.
func mountAsyncTestComponent(parseT *testing.T, parseComponent func() Node) {
	parseT.Helper()
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseScheduler := mockdom.NewMockScheduler(true)

	parsePreviousInitialized := runtimeInitialized
	runtimeInitialized = true
	parseT.Cleanup(func() {
		runtimeInitialized = parsePreviousInitialized
	})
	resetUIRuntime(runtime.Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseContainer := parseAdapter.CreateElement("div")
	if parseErr := RenderInto(CreateElement(func(parseProps map[string]interface{}) Node {
		return parseComponent()
	}, nil), parseContainer); parseErr != nil {
		parseT.Fatalf("mount: %v", parseErr)
	}
}

// waitAsyncCondition polls until the condition holds or the deadline passes.
func waitAsyncCondition(parseT *testing.T, parseWhat string, parseCheck func() bool) {
	parseT.Helper()
	parseDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(parseDeadline) {
		if parseCheck() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	parseT.Fatalf("timed out waiting for %s", parseWhat)
}

// TestAsyncBoundaryBranchesWASM pins the wasm AsyncBoundary contract: error
// renders the error fallback, pending renders the fallback, and completion
// renders the content.
func TestAsyncBoundaryBranchesWASM(parseT *testing.T) {
	parseContent := Text("content")
	parseFallback := Text("loading")
	parseErrorNode := Text("failed")

	var parseGotError, parseGotPending, parseGotDone Node
	mountAsyncTestComponent(parseT, func() Node {
		parseGotError = AsyncBoundary(AsyncBoundaryProps{
			Error:         errors.New("boom"),
			ErrorFallback: func(error) Node { return parseErrorNode },
			Content:       parseContent,
		})
		parseGotPending = AsyncBoundary(AsyncBoundaryProps{
			Pending:  true,
			Fallback: parseFallback,
			Content:  parseContent,
		})
		parseGotDone = AsyncBoundary(AsyncBoundaryProps{
			Content: parseContent,
		})
		return parseGotDone
	})

	if parseGotError == nil || parseGotError.Type != runtime.AsyncBoundaryNodeType {
		parseT.Fatalf("error branch rendered %#v", parseGotError)
	}
	parseErrorFallbackFn, parseOk := parseGotError.Props["errorFallback"].(func(error) Node)
	if !parseOk || parseErrorFallbackFn(errors.New("boom")) != parseErrorNode {
		parseT.Fatalf("error branch did not preserve error fallback, props=%#v", parseGotError.Props)
	}
	if parseGotPending == nil {
		parseT.Fatal("pending branch rendered nil")
	}
	if parseGotPending.Type != runtime.AsyncBoundaryNodeType || parseGotPending.Props["pending"] != true || parseGotPending.Props["fallback"] != parseFallback {
		parseT.Fatalf("pending branch rendered %#v props=%#v", parseGotPending, parseGotPending.Props)
	}
	if parseGotDone == nil {
		parseT.Fatal("done branch rendered nil")
	}
	if parseGotDone.Type != runtime.AsyncBoundaryNodeType || parseGotDone.Props["content"] != parseContent {
		parseT.Fatalf("done branch rendered %#v props=%#v", parseGotDone, parseGotDone.Props)
	}
}

// TestUseLazyNodeLoadsAndReloadsWASM verifies the lazy-node lifecycle on wasm:
// the loader runs after mount, the loaded node becomes available, Reload runs
// the loader again, and Cancel is safe to call.
func TestUseLazyNodeLoadsAndReloadsWASM(parseT *testing.T) {
	parseLoaded := Text("lazy-content")
	parseLoadCount := 0
	var parseHandle LazyNode

	mountAsyncTestComponent(parseT, func() Node {
		parseHandle = UseLazyNode(func(parseCtx context.Context) (Node, error) {
			parseLoadCount++
			return parseLoaded, nil
		})
		return Text("host")
	})

	waitAsyncCondition(parseT, "lazy node load", func() bool {
		parseState := parseHandle.Get()
		return !parseState.Loading && parseState.Node == parseLoaded
	})
	if parseLoadCount != 1 {
		parseT.Fatalf("loader ran %d times after mount", parseLoadCount)
	}

	parseHandle.Reload()
	waitAsyncCondition(parseT, "lazy node reload", func() bool {
		return parseLoadCount >= 2 && !parseHandle.Get().Loading
	})

	parseHandle.Cancel()
	if parseHandle.Get().Node != parseLoaded {
		parseT.Fatal("cancel after completion must not drop the loaded node")
	}
}

// TestUseLazyNodeErrorWASM verifies loader errors surface in the state.
func TestUseLazyNodeErrorWASM(parseT *testing.T) {
	parseBoom := errors.New("lazy-boom")
	var parseHandle LazyNode

	mountAsyncTestComponent(parseT, func() Node {
		parseHandle = UseLazyNode(func(parseCtx context.Context) (Node, error) {
			return nil, parseBoom
		})
		return Text("host")
	})

	waitAsyncCondition(parseT, "lazy node error", func() bool {
		parseState := parseHandle.Get()
		return !parseState.Loading && parseState.Error != nil
	})
	if !errors.Is(parseHandle.Get().Error, parseBoom) {
		parseT.Fatalf("expected loader error, got %v", parseHandle.Get().Error)
	}
}

// TestUseChannelReceivesAndClosesWASM verifies the channel subscription hook:
// values arriving on the channel become visible with Ok=true, and closing the
// channel flips Closed while preserving the last value.
func TestUseChannelReceivesAndClosesWASM(parseT *testing.T) {
	parseCh := make(chan int, 4)
	var parseHandle Channel[int]

	mountAsyncTestComponent(parseT, func() Node {
		parseHandle = UseChannel(parseCh)
		return Text("host")
	})

	if parseHandle.Ok() {
		parseT.Fatal("Ok must be false before any value arrives")
	}

	parseCh <- 41
	waitAsyncCondition(parseT, "first channel value", func() bool {
		return parseHandle.Ok() && parseHandle.Get() == 41
	})

	parseCh <- 42
	waitAsyncCondition(parseT, "second channel value", func() bool {
		return parseHandle.Get() == 42
	})

	close(parseCh)
	waitAsyncCondition(parseT, "channel close", func() bool {
		return parseHandle.Closed()
	})
	if parseHandle.Get() != 42 {
		parseT.Fatalf("close dropped the last value: %v", parseHandle.Get())
	}
}
