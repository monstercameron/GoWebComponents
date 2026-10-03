package render

import (
	"context"
	"testing"
	"time"
)

// TestResourceControllerRecordsCancelOfSupersededAttempt verifies an attempt
// that was superseded by a newer one and then canceled by its context is
// recorded as canceled, the way a refetch cancels the previous load.
func TestResourceControllerRecordsCancelOfSupersededAttempt(parseT *testing.T) {
	parseC := NewResourceController[int]()
	parseCtx1, parseCancel1 := context.WithCancel(context.Background())
	parseCtx2, parseCancel2 := context.WithCancel(context.Background())
	defer parseCancel2()

	parseDone1 := make(chan error, 1)
	go func() {
		_, parseErr := parseC.Await(parseCtx1)
		parseDone1 <- parseErr
	}()
	<-parseC.Started()
	go func() { _, _ = parseC.Await(parseCtx2) }()
	<-parseC.Started()

	parseCancel1()
	select {
	case <-parseDone1:
	case <-time.After(2 * time.Second):
		parseT.Fatal("first attempt did not return after its context was canceled")
	}
	if parseAttempts := parseC.Attempts(); len(parseAttempts) != 2 || !parseAttempts[0].Cancelled {
		parseT.Fatalf("attempt 1 must be recorded as canceled, got %+v", parseAttempts)
	}
}

// TestResourceControllerAwaitNilContext verifies a nil context is tolerated.
func TestResourceControllerAwaitNilContext(parseT *testing.T) {
	parseC := NewResourceController[int]()
	parseDone := make(chan int, 1)
	go func() {
		parseValue, _ := parseC.Await(nil) //nolint:staticcheck // nil context is the case under test
		parseDone <- parseValue
	}()
	<-parseC.Started()
	parseC.Resolve(7)
	select {
	case parseValue := <-parseDone:
		if parseValue != 7 {
			parseT.Fatalf("got %d, want 7", parseValue)
		}
	case <-time.After(2 * time.Second):
		parseT.Fatal("Await hung")
	}
}
