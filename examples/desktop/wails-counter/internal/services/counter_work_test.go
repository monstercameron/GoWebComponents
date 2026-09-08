package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRunWorkCancellationTerminatesBackend verifies actual work state, not a cancelled caller wait.
func TestRunWorkCancellationTerminatesBackend(parseT *testing.T) {
	parseService := &CounterService{}
	parseContext, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseFinished := make(chan error, 1)
	go func() { parseFinished <- parseService.RunWork(parseContext) }()
	parseDeadline := time.Now().Add(time.Second)
	for parseService.GetWorkState().Active != 1 {
		if time.Now().After(parseDeadline) {
			parseT.Fatal("native work never became active")
		}
		time.Sleep(time.Millisecond)
	}
	parseCancel()
	select {
	case parseErr := <-parseFinished:
		if !errors.Is(parseErr, context.Canceled) {
			parseT.Fatalf("RunWork returned %v", parseErr)
		}
	case <-time.After(time.Second):
		parseT.Fatal("native work did not terminate on cancellation")
	}
	parseState := parseService.GetWorkState()
	if parseState.Active != 0 || parseState.Cancelled != 1 {
		parseT.Fatalf("backend state = %+v", parseState)
	}
}

// TestRunWorkRejectsInvalidContexts ensures no backend work starts after cancellation or expiry.
func TestRunWorkRejectsInvalidContexts(parseT *testing.T) {
	parseService := &CounterService{}
	if parseErr := parseService.RunWork(nil); parseErr == nil { //nolint:staticcheck // Deliberately tests the service's invalid-context guard.
		parseT.Fatal("nil context accepted")
	}
	parseContext, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if parseErr := parseService.RunWork(parseContext); !errors.Is(parseErr, context.Canceled) {
		parseT.Fatal(parseErr)
	}
	parseContext, parseCancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer parseCancel()
	if parseErr := parseService.RunWork(parseContext); !errors.Is(parseErr, context.DeadlineExceeded) {
		parseT.Fatal(parseErr)
	}
	if parseState := parseService.GetWorkState(); parseState.Active != 0 || parseState.Cancelled != 0 {
		parseT.Fatalf("invalid context changed state: %+v", parseState)
	}
}
