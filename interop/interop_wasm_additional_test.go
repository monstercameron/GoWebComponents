//go:build js && wasm
// +build js,wasm

package interop

import (
	"sync"
	"syscall/js"
	"testing"
	"time"
)

// TestScheduleIntervalWasmRunsAndCancels verifies the live interval helper fires repeatedly and stops once canceled.
func TestScheduleIntervalWasmRunsAndCancels(parseT *testing.T) {
	parseT.Run("nil-callback", func(parseT2 *testing.T) {
		if _, parseErr := ScheduleInterval(time.Millisecond, nil); !IsCode(parseErr, CodeInvalid) {
			parseT2.Fatalf("expected invalid nil callback error, got %v", parseErr)
		}
	})

	parseTicks := make(chan int, 4)
	var (
		parseMu    sync.Mutex
		parseCount int
	)
	parseTimer, parseErr := ScheduleInterval(time.Millisecond, func() {
		parseMu.Lock()
		parseCount++
		parseCurrent := parseCount
		parseMu.Unlock()
		select {
		case parseTicks <- parseCurrent:
		default:
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected interval scheduling to succeed, got %v", parseErr)
	}

	parseDeadline := time.After(200 * time.Millisecond)
	parseObserved := 0
	for parseObserved < 2 {
		select {
		case parseObserved = <-parseTicks:
		case <-parseDeadline:
			parseT.Fatalf("expected interval callback to fire repeatedly, observed count %d", parseObserved)
		}
	}

	if parseErr2 := parseTimer.Cancel(); parseErr2 != nil {
		parseT.Fatalf("expected interval cancel to succeed, got %v", parseErr2)
	}

	parseMu.Lock()
	parseBeforeCancel := parseCount
	parseMu.Unlock()
	time.Sleep(25 * time.Millisecond)
	parseMu.Lock()
	parseAfterCancel := parseCount
	parseMu.Unlock()
	if parseAfterCancel != parseBeforeCancel {
		parseT.Fatalf("expected interval to stop after cancel, before=%d after=%d", parseBeforeCancel, parseAfterCancel)
	}
}

// TestDocumentElementsByIDWasmReturnsPresentMatches verifies the batched document lookup helper only returns matched elements.
func TestDocumentElementsByIDWasmReturnsPresentMatches(parseT *testing.T) {
	parseDocument := js.Global().Get("Object").New()
	parseGetElementByID := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return js.Null()
		}
		switch parseArgs[0].String() {
		case "save":
			parseElement := js.Global().Get("Object").New()
			parseElement.Set("id", "save")
			parseElement.Set("tagName", "BUTTON")
			return parseElement
		case "name":
			parseElement := js.Global().Get("Object").New()
			parseElement.Set("id", "name")
			parseElement.Set("tagName", "INPUT")
			return parseElement
		default:
			return js.Null()
		}
	})
	defer parseGetElementByID.Release()
	parseDocument.Set("getElementById", parseGetElementByID)

	parseElements := documentElementsByID(parseDocument, []string{"save", "missing", "name"})
	if len(parseElements) != 2 {
		parseT.Fatalf("expected only matched elements, got %#v", parseElements)
	}
	if parseElements["save"].Get("tagName").String() != "BUTTON" || parseElements["name"].Get("tagName").String() != "INPUT" {
		parseT.Fatalf("unexpected documentElementsByID result: %#v", parseElements)
	}
	if _, parseOk := parseElements["missing"]; parseOk {
		parseT.Fatalf("expected missing element lookup to be omitted, got %#v", parseElements)
	}
}
