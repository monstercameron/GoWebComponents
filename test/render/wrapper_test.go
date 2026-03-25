package render

import (
	"context"
	"testing"
	"time"
)

func expectPanic(parseT *testing.T, parseFn func()) {
	parseT.Helper()
	defer func() {
		if recover() == nil {
			parseT.Fatalf("expected panic")
		}
	}()
	parseFn()
}

func TestWithQueuedSchedulerReturnsOption(parseT *testing.T) {
	if parseOption := WithQueuedScheduler(); parseOption == nil {
		parseT.Fatalf("expected queued scheduler option")
	}
}

func TestNewResourceControllerWrapperResolves(parseT *testing.T) {
	parseController := NewResourceController[string]()
	parseResult := make(chan string, 1)

	go func() {
		parseValue, parseErr := parseController.Await(context.Background())
		if parseErr != nil {
			parseResult <- "error"
			return
		}
		parseResult <- parseValue
	}()

	select {
	case <-parseController.Started():
	case <-time.After(2 * time.Second):
		parseT.Fatalf("timed out waiting for resource attempt start")
	}

	parseController.Resolve("ready")
	select {
	case parseGot := <-parseResult:
		if parseGot != "ready" {
			parseT.Fatalf("expected resolved value ready, got %q", parseGot)
		}
	case <-time.After(2 * time.Second):
		parseT.Fatalf("timed out waiting for resolved value")
	}
}

func TestNewPanicsWithNilTestingTB(parseT *testing.T) {
	expectPanic(parseT, func() {
		_ = New(nil)
	})
}
