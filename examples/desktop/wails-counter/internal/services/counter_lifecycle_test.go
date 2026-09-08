package services

import (
	"context"
	"sort"
	"sync"
	"testing"
)

// TestCounterServiceIncrementIsSafeForConcurrentCalls verifies that every
// concurrent native request receives one distinct counter value.
func TestCounterServiceIncrementIsSafeForConcurrentCalls(parseT *testing.T) {
	const parseCalls = 128
	parseService := &CounterService{}
	parseValues := make(chan int, parseCalls)
	var parseWait sync.WaitGroup
	parseWait.Add(parseCalls)
	for parseIndex := 0; parseIndex < parseCalls; parseIndex++ {
		go func() {
			defer parseWait.Done()
			parseValues <- parseService.Increment().Value
		}()
	}
	parseWait.Wait()
	close(parseValues)

	parseSorted := make([]int, 0, parseCalls)
	for parseValue := range parseValues {
		parseSorted = append(parseSorted, parseValue)
	}
	sort.Ints(parseSorted)
	for parseIndex, parseValue := range parseSorted {
		parseExpected := parseIndex + 1
		if parseValue != parseExpected {
			parseT.Fatalf("concurrent increment value[%d] = %d, want %d; protect CounterService.value", parseIndex, parseValue, parseExpected)
		}
	}
}

// TestCounterServiceRunProgressHonorsCanceledContext verifies cancellation
// before the first progress event and protects this contract from regressions.
func TestCounterServiceRunProgressHonorsCanceledContext(parseT *testing.T) {
	parseContext, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	parseErr := (&CounterService{}).RunProgress(parseContext)
	if parseErr != context.Canceled {
		parseT.Fatalf("RunProgress canceled error = %v, want %v", parseErr, context.Canceled)
	}
}
