//go:build !js

package fetch

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// TestCachedResourceConcurrentInvariants hammers one cached resource from
// many goroutines mixing LoadCached, Set, Update, Invalidate, and Get, then
// checks the invariants the cancel-path and snapshot-serialization fixes are
// supposed to guarantee:
//  1. LoadCached never returns a zero value with a nil error (the waiter
//     woken-before-publish bug class);
//  2. every returned value is either the loader value or an optimistic write;
//  3. the registry entry is never left permanently pending.
func TestCachedResourceConcurrentInvariants(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	const parseKey = "invariant-fuzz"
	var parseLoads int32
	parseLoader := func(parseCtx context.Context) (string, error) {
		atomic.AddInt32(&parseLoads, 1)
		return "loaded", nil
	}

	parseValid := func(parseVal string) bool {
		if parseVal == "loaded" || parseVal == "" {
			return true
		}
		if len(parseVal) >= 4 && parseVal[:4] == "opt-" {
			return true
		}
		return false
	}

	// UseCachedResource is a render-thread hook; the returned handle is the
	// goroutine-safe surface. Create it once here, share it across workers.
	parseHandle := UseCachedResource(parseKey, parseLoader)

	var parseWg sync.WaitGroup
	const parseWorkers = 8
	const parseOpsPerWorker = 200
	for parseW := 0; parseW < parseWorkers; parseW++ {
		parseW2 := parseW
		parseWg.Add(1)
		go func() {
			defer parseWg.Done()
			for parseI := 0; parseI < parseOpsPerWorker; parseI++ {
				switch parseI % 5 {
				case 0:
					parseVal, parseErr := LoadCached(context.Background(), parseKey, parseLoader)
					if parseErr != nil {
						parseT.Errorf("worker %d op %d: LoadCached error: %v", parseW2, parseI, parseErr)
						return
					}
					// Invariant 1+2: a nil error must come with a real value.
					if !parseValid(parseVal) || parseVal == "" {
						parseT.Errorf("worker %d op %d: LoadCached returned %q with nil error", parseW2, parseI, parseVal)
						return
					}
				case 1:
					parseHandle.Set(fmt.Sprintf("opt-%d-%d", parseW2, parseI))
				case 2:
					parseHandle.Update(func(parsePrev string) string {
						if parsePrev == "" {
							return "opt-upd"
						}
						return parsePrev
					})
				case 3:
					InvalidateResource(parseKey)
				case 4:
					parseState := parseHandle.Get()
					if parseState.Ready && !parseValid(parseState.Value) {
						parseT.Errorf("worker %d op %d: Get observed invalid value %q", parseW2, parseI, parseState.Value)
						return
					}
				}
			}
		}()
	}
	parseWg.Wait()

	// Invariant 3: a final LoadCached must terminate with a valid value —
	// the entry must not be wedged pending.
	parseFinal, parseErr := LoadCached(context.Background(), parseKey, parseLoader)
	if parseErr != nil || !parseValid(parseFinal) || parseFinal == "" {
		parseT.Fatalf("final LoadCached wedged or invalid: value=%q err=%v", parseFinal, parseErr)
	}
	if atomic.LoadInt32(&parseLoads) == 0 {
		parseT.Fatal("loader never ran — test exercised nothing")
	}
}
