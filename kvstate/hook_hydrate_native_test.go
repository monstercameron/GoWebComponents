//go:build !js || !wasm

package kvstate

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestPersistedHydrationKeepsSetMadeBeforeHydration pins the hook fix: a Set
// that lands before the engine is published is parked, wins over the stored
// row, and is persisted at a version above the stored one.
func TestPersistedHydrationKeepsSetMadeBeforeHydration(parseT *testing.T) {
	parseBackend := &gatedLoadBackend{
		externalAtomBackend: externalAtomBackend{
			parseRecord: Record{Key: "k", Value: []byte("5"), Version: 3},
			isFound:     true,
			parseSaved:  make(chan Record, 4),
		},
		parseGate: make(chan struct{}),
	}
	parseOpts := Options{Name: parseT.Name(), Backend: parseBackend, ExternalInvalidation: true, Strategy: Immediate{}}.withDefaults()
	parseLifetime := newBindingLifetime(context.Background())
	defer parseLifetime.close()
	parseShared := &boundAtomState{}

	var parseMu sync.Mutex
	parseValue := 1
	parseLoading := true
	parseSinks := persistedSinks[int]{
		setValue:   func(parseV int) { parseMu.Lock(); parseValue = parseV; parseMu.Unlock() },
		setLoading: func(parseV bool) { parseMu.Lock(); parseLoading = parseV; parseMu.Unlock() },
		setErr:     func(parseErr error) { parseT.Errorf("unexpected error: %v", parseErr) },
	}

	parseDone := make(chan struct{})
	go func() {
		runPersistedHydration(parseLifetime, parseShared, parseOpts, "k", 1, parseSinks)
		close(parseDone)
	}()

	// Lands while Load is still gated: the engine is not published yet.
	setPersistedValue(parseLifetime.parseContext, parseShared, parseOpts, "k", 9, parseSinks)
	close(parseBackend.parseGate)
	<-parseDone

	select {
	case parseRecord := <-parseBackend.parseSaved:
		if string(parseRecord.Value) != "9" || parseRecord.Version != 4 {
			parseT.Fatalf("persisted %q at version %d, want \"9\" at version 4", parseRecord.Value, parseRecord.Version)
		}
	case <-time.After(2 * time.Second):
		parseT.Fatal("a Set made during hydration was never persisted")
	}
	parseMu.Lock()
	defer parseMu.Unlock()
	if parseValue != 9 || parseLoading {
		parseT.Fatalf("value=%d loading=%v, want the local write 9 and loading=false (stored 5 must not overwrite it)", parseValue, parseLoading)
	}
}

// TestPersistedHydrationAppliesStoredValueWithoutEarlySet pins the unchanged
// path: with no early Set the stored row is applied.
func TestPersistedHydrationAppliesStoredValueWithoutEarlySet(parseT *testing.T) {
	parseBackend := &externalAtomBackend{
		parseRecord: Record{Key: "k", Value: []byte("5"), Version: 3},
		isFound:     true,
		parseSaved:  make(chan Record, 4),
	}
	parseOpts := Options{Name: parseT.Name(), Backend: parseBackend, ExternalInvalidation: true, Strategy: Immediate{}}.withDefaults()
	parseLifetime := newBindingLifetime(context.Background())
	defer parseLifetime.close()
	parseValue := 1
	parseSinks := persistedSinks[int]{
		setValue:   func(parseV int) { parseValue = parseV },
		setLoading: func(bool) {},
		setErr:     func(parseErr error) { parseT.Errorf("unexpected error: %v", parseErr) },
	}
	runPersistedHydration(parseLifetime, &boundAtomState{}, parseOpts, "k", 1, parseSinks)
	if parseValue != 5 {
		parseT.Fatalf("value=%d, want stored 5", parseValue)
	}
}
