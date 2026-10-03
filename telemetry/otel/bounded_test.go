//go:build !js || !wasm

package otel

import (
	"fmt"
	"math"
	"net/http"
	"runtime/metrics"
	"sync"
	"testing"
)

// metricsHistogram aliases the runtime histogram type for table setup.
type metricsHistogram = metrics.Float64Histogram

// infinity returns +Inf for the open top bucket of a runtime histogram.
func infinity() float64 { return math.Inf(1) }

// newPatternRequest builds a request whose ServeMux pattern is parsePattern.
func newPatternRequest(parsePattern string) *http.Request {
	parseRequest, _ := http.NewRequest(http.MethodGet, "http://example.test"+parsePattern, nil)
	parseRequest.Pattern = parsePattern
	return parseRequest
}

func TestBoundedSetAdmitsUpToCapThenOverflows(parseT *testing.T) {
	parseSet := NewBoundedSet(3)
	for _, parseValue := range []string{"a", "b", "c"} {
		if parseGot := parseSet.Bound(parseValue); parseGot != parseValue {
			parseT.Fatalf("Bound(%q) = %q", parseValue, parseGot)
		}
	}
	if parseGot := parseSet.Bound("d"); parseGot != OverflowValue {
		parseT.Fatalf("Bound(d) = %q, want %q", parseGot, OverflowValue)
	}
	if parseGot := parseSet.Bound("a"); parseGot != "a" {
		parseT.Fatalf("admitted value lost: %q", parseGot)
	}
	if parseSet.Len() != 3 {
		parseT.Fatalf("Len = %d, want 3", parseSet.Len())
	}
}

func TestBoundedSetDefaultCapIs200(parseT *testing.T) {
	parseSet := NewBoundedSet(0)
	var parseOverflow int
	for parseIndex := 0; parseIndex < 300; parseIndex++ {
		if parseSet.Bound(fmt.Sprintf("v%d", parseIndex)) == OverflowValue {
			parseOverflow++
		}
	}
	if parseSet.Len() != 200 || parseOverflow != 100 {
		parseT.Fatalf("len=%d overflow=%d, want 200/100", parseSet.Len(), parseOverflow)
	}
}

func TestBoundedSetIsSafeForConcurrentUse(parseT *testing.T) {
	parseSet := NewBoundedSet(10)
	var parseGroup sync.WaitGroup
	for parseWorker := 0; parseWorker < 8; parseWorker++ {
		parseGroup.Add(1)
		go func() {
			defer parseGroup.Done()
			for parseIndex := 0; parseIndex < 100; parseIndex++ {
				parseSet.Bound(fmt.Sprintf("k%d", parseIndex))
			}
		}()
	}
	parseGroup.Wait()
	if parseSet.Len() != 10 {
		parseT.Fatalf("Len = %d, want 10", parseSet.Len())
	}
}
