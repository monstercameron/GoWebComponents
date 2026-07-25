//go:build js && wasm

package ui

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// captureCrashReports installs a panic-report hook for one test.
func captureCrashReports(parseT *testing.T) func() []runtime.PanicReport {
	parseT.Helper()
	parsePrevious := runtime.CurrentUnhandledPanicLoggingOptions()
	var parseMu sync.Mutex
	var parseReports []runtime.PanicReport
	runtime.ConfigureUnhandledPanicLogging(runtime.PanicLoggingOptions{
		HideRawPanicOutput: true,
		OnReport: func(parseReport runtime.PanicReport) {
			parseMu.Lock()
			parseReports = append(parseReports, parseReport)
			parseMu.Unlock()
		},
	})
	parseT.Cleanup(func() { runtime.ConfigureUnhandledPanicLogging(parsePrevious) })
	return func() []runtime.PanicReport {
		parseMu.Lock()
		defer parseMu.Unlock()
		return append([]runtime.PanicReport(nil), parseReports...)
	}
}

// TestSafeGoPanicKeepsRuntimeAliveWASM proves the core crash-containment
// promise on wasm: a panicking background goroutine produces a crash report
// while the mounted app keeps rendering state updates afterwards.
func TestSafeGoPanicKeepsRuntimeAliveWASM(parseT *testing.T) {
	var parseCount State[int]
	mountAsyncTestComponent(parseT, func() Node {
		parseCount = UseState(0)
		return Text("count")
	})
	parseGetReports := captureCrashReports(parseT)

	SafeGo("wasm crash probe", func() {
		panic("safego-wasm-boom")
	})
	waitAsyncCondition(parseT, "contained goroutine report", func() bool {
		return len(parseGetReports()) == 1
	})
	if !strings.Contains(parseGetReports()[0].Summary, "safego-wasm-boom") {
		parseT.Fatalf("report summary = %q", parseGetReports()[0].Summary)
	}

	// The page must remain interactive: state updates still commit.
	parseCount.Set(41)
	waitAsyncCondition(parseT, "post-crash state update", func() bool {
		return parseCount.Get() == 41
	})
}

// TestRenderPanicWithoutBoundaryRecoversWASM proves a component render panic
// with no error boundary no longer wedges the runtime: the panic is reported,
// the in-flight render is abandoned, and the next clean update renders.
func TestRenderPanicWithoutBoundaryRecoversWASM(parseT *testing.T) {
	var parseMode State[int]
	var parseRendered int
	mountAsyncTestComponent(parseT, func() Node {
		parseMode = UseState(0)
		parseRendered = parseMode.Get()
		if parseMode.Get() == 1 {
			panic("render-wasm-boom")
		}
		return Text("ok")
	})
	parseGetReports := captureCrashReports(parseT)
	if parseRendered != 0 {
		parseT.Fatalf("initial render saw mode %d", parseRendered)
	}

	// Trigger the panicking render. The work loop must contain it.
	parseMode.Set(1)
	waitAsyncCondition(parseT, "render panic report", func() bool {
		return len(parseGetReports()) >= 1
	})
	if !strings.Contains(parseGetReports()[0].Summary, "render-wasm-boom") {
		parseT.Fatalf("report summary = %q", parseGetReports()[0].Summary)
	}

	// Recovery: a clean update must render again after the contained crash.
	parseMode.Set(2)
	waitAsyncCondition(parseT, "post-panic clean render", func() bool {
		return parseRendered == 2
	})
}

// TestUseTaskPanicContainedWASM proves a panic inside a framework-owned hook
// goroutine (UseTask runner) is contained instead of killing the page.
func TestUseTaskPanicContainedWASM(parseT *testing.T) {
	var parseProbe State[int]
	mountAsyncTestComponent(parseT, func() Node {
		parseProbe = UseState(0)
		parseTask := UseTask(func(parseCtx context.Context) (int, error) {
			panic("usetask-wasm-boom")
		})
		UseEffect(func() func() {
			parseTask.Start()
			return nil
		}, nil)
		return Text("task host")
	})
	parseGetReports := captureCrashReports(parseT)

	waitAsyncCondition(parseT, "contained UseTask report", func() bool {
		for _, parseReport := range parseGetReports() {
			if strings.Contains(parseReport.Summary, "usetask-wasm-boom") {
				return true
			}
		}
		return false
	})

	parseProbe.Set(7)
	waitAsyncCondition(parseT, "post-task-crash update", func() bool {
		return parseProbe.Get() == 7
	})
	_ = time.Now()
}
