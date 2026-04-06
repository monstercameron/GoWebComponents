//go:build js && wasm
// +build js,wasm

package interop

import (
	"syscall/js"
	"testing"
	"time"
)

// TestScheduleIntervalWasmInvokesAndCancels verifies repeated interval callbacks and idempotent cancellation under js/wasm.
func TestScheduleIntervalWasmInvokesAndCancels(parseT *testing.T) {
	var (
		parseIntervalCallback js.Value
		parseIntervalDelay    int
		parseClearedID        string
		parseCallbackCount    int
	)
	parseSetInterval := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseIntervalCallback = parseArgs[0]
		parseIntervalDelay = parseArgs[1].Int()
		return "interval-token"
	})
	defer parseSetInterval.Release()
	parseClearInterval := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 {
			parseClearedID = parseArgs[0].String()
		}
		return nil
	})
	defer parseClearInterval.Release()

	parseRestoreSetInterval := setGlobalValue("setInterval", parseSetInterval)
	defer parseRestoreSetInterval()
	parseRestoreClearInterval := setGlobalValue("clearInterval", parseClearInterval)
	defer parseRestoreClearInterval()

	parseTimer, parseErr := ScheduleInterval(25*time.Millisecond, func() {
		parseCallbackCount++
	})
	if parseErr != nil {
		parseT.Fatalf("expected interval timer, got %v", parseErr)
	}
	if parseIntervalDelay != 25 {
		parseT.Fatalf("expected millisecond delay of 25, got %d", parseIntervalDelay)
	}
	if !parseIntervalCallback.Truthy() {
		parseT.Fatal("expected interval callback to be registered")
	}

	parseIntervalCallback.Invoke()
	parseIntervalCallback.Invoke()
	if parseCallbackCount != 2 {
		parseT.Fatalf("expected interval callback to fire twice, got %d", parseCallbackCount)
	}

	if parseErr2 := parseTimer.Cancel(); parseErr2 != nil {
		parseT.Fatalf("expected interval cancel to succeed, got %v", parseErr2)
	}
	if parseErr3 := parseTimer.Cancel(); parseErr3 != nil {
		parseT.Fatalf("expected repeated interval cancel to stay safe, got %v", parseErr3)
	}
	if parseClearedID != "interval-token" {
		parseT.Fatalf("expected clearInterval to receive the interval token, got %q", parseClearedID)
	}

	if _, parseErr4 := ScheduleInterval(time.Second, nil); !IsCode(parseErr4, CodeInvalid) {
		parseT.Fatalf("expected nil interval callback error, got %v", parseErr4)
	}
}

// TestGetBrowserEventTargetsWasmResolvesWindowAndDocument verifies the simple browser event-target accessors on both present and missing globals.
func TestGetBrowserEventTargetsWasmResolvesWindowAndDocument(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseDocument := js.Global().Get("Object").New()
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()
	parseRestoreDocument := setGlobalValue("document", parseDocument)
	defer parseRestoreDocument()

	if _, parseErr := GetWindowEvents(); parseErr != nil {
		parseT.Fatalf("expected window event target, got %v", parseErr)
	}
	if _, parseErr2 := GetDocumentEvents(); parseErr2 != nil {
		parseT.Fatalf("expected document event target, got %v", parseErr2)
	}

	parseRestoreMissingWindow := setGlobalValue("window", js.Undefined())
	if _, parseErr3 := GetWindowEvents(); !IsCode(parseErr3, CodeUnavailable) {
		parseT.Fatalf("expected unavailable window events error, got %v", parseErr3)
	}
	parseRestoreMissingWindow()

	parseRestoreMissingDocument := setGlobalValue("document", js.Undefined())
	if _, parseErr4 := GetDocumentEvents(); !IsCode(parseErr4, CodeUnavailable) {
		parseT.Fatalf("expected unavailable document events error, got %v", parseErr4)
	}
	parseRestoreMissingDocument()
}
