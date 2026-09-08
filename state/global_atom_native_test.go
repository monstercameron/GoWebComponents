//go:build !(js && wasm)

package state_test

import (
	"sync"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/state"
)

// TestGlobalAtomSeedsAndReads proves NewGlobalAtom seeds its default and Get
// reads it back through the shared runtime registry.
func TestGlobalAtomSeedsAndReads(parseT *testing.T) {
	parseAtom := state.NewGlobalAtom("test:ga:seed", 7)
	if parseGot := parseAtom.Get(); parseGot != 7 {
		parseT.Fatalf("expected seeded default 7, got %d", parseGot)
	}
}

// TestGlobalAtomSetGetRoundTrip proves an external write is observable.
func TestGlobalAtomSetGetRoundTrip(parseT *testing.T) {
	parseAtom := state.NewGlobalAtom("test:ga:roundtrip", "idle")
	parseAtom.Set("busy")
	if parseGot := parseAtom.Get(); parseGot != "busy" {
		parseT.Fatalf("expected 'busy' after Set, got %q", parseGot)
	}
}

// TestGlobalAtomDoesNotClobberExistingValue is the core G39 guarantee: a value
// written BEFORE a later handle is constructed survives — constructing a handle
// (with a different default) never overwrites an existing value. This is what
// makes a pre-render write durable instead of silently dropped.
func TestGlobalAtomDoesNotClobberExistingValue(parseT *testing.T) {
	parseWriter := state.NewGlobalAtom("test:ga:noclobber", "default-a")
	parseWriter.Set("written-before-second-handle")

	// A second handle for the same id, constructed later with a different default,
	// must observe the already-written value — not reseed it.
	parseReader := state.NewGlobalAtom("test:ga:noclobber", "default-b")
	if parseGot := parseReader.Get(); parseGot != "written-before-second-handle" {
		parseT.Fatalf("late handle clobbered existing value: got %q", parseGot)
	}
}

// TestGlobalAtomSharesRegistryWithRuntime proves a GlobalAtom write lands in the
// same registry UseAtom reads from (GetGlobalRuntime().GetAtomValue).
func TestGlobalAtomSharesRegistryWithRuntime(parseT *testing.T) {
	parseAtom := state.NewGlobalAtom("test:ga:shared", 0)
	parseAtom.Set(99)

	parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("test:ga:shared")
	if !parseOk {
		parseT.Fatal("expected atom present in shared registry")
	}
	if parseValue.(int) != 99 {
		parseT.Fatalf("expected shared registry value 99, got %v", parseValue)
	}
}

// TestGlobalAtomUpdate proves Update reads-then-writes.
func TestGlobalAtomUpdate(parseT *testing.T) {
	parseAtom := state.NewGlobalAtom("test:ga:update", 10)
	parseAtom.Update(func(parsePrev int) int { return parsePrev + 5 })
	if parseGot := parseAtom.Get(); parseGot != 15 {
		parseT.Fatalf("expected 15 after Update, got %d", parseGot)
	}
}

// TestGlobalAtomUpdateIsAtomicUnderConcurrency pins #76: many goroutines each
// incrementing the same atom via Update must not lose writes. The old
// Set(fn(Get())) implementation read-computed-wrote without a lock, so two
// updaters reading the same old value would clobber each other and the final
// count would fall short of the number of increments. The atomic registry
// update holds the lock across read-compute-write, so every increment lands.
func TestGlobalAtomUpdateIsAtomicUnderConcurrency(parseT *testing.T) {
	parseAtom := state.NewGlobalAtom("test:ga:atomic-update", 0)

	const parseGoroutines = 50
	const parsePerGoroutine = 200
	var parseWG sync.WaitGroup
	parseWG.Add(parseGoroutines)
	for parseG := 0; parseG < parseGoroutines; parseG++ {
		go func() {
			defer parseWG.Done()
			for parseI := 0; parseI < parsePerGoroutine; parseI++ {
				parseAtom.Update(func(parsePrev int) int { return parsePrev + 1 })
			}
		}()
	}
	parseWG.Wait()

	parseWant := parseGoroutines * parsePerGoroutine
	if parseGot := parseAtom.Get(); parseGot != parseWant {
		parseT.Fatalf("lost updates under concurrency: got %d, want %d", parseGot, parseWant)
	}
}

// TestGlobalAtomTypeMismatchFallsBackToDefault proves reading an id holding a
// different type returns the handle default rather than panicking.
func TestGlobalAtomTypeMismatchFallsBackToDefault(parseT *testing.T) {
	parseStringAtom := state.NewGlobalAtom("test:ga:mismatch", "hello")
	parseStringAtom.Set("world")

	parseIntAtom := state.NewGlobalAtom("test:ga:mismatch", -1)
	if parseGot := parseIntAtom.Get(); parseGot != -1 {
		parseT.Fatalf("expected default -1 on type mismatch, got %d", parseGot)
	}
}

// TestGlobalAtomID proves ID returns the key shared with UseAtom.
func TestGlobalAtomID(parseT *testing.T) {
	parseAtom := state.NewGlobalAtom("test:ga:id", false)
	if parseAtom.ID() != "test:ga:id" {
		parseT.Fatalf("expected id, got %q", parseAtom.ID())
	}
}
