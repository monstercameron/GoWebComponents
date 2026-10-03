//go:build !js || !wasm

package kvstate

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// gatedLoadBackend blocks Load until released so a Set can land mid-hydration.
type gatedLoadBackend struct {
	externalAtomBackend
	parseGate chan struct{}
}

// Load waits for the gate, then reports the stored row.
func (parseBackend *gatedLoadBackend) Load(parseCtx context.Context, parseKey string) (Record, bool, error) {
	select {
	case <-parseBackend.parseGate:
	case <-parseCtx.Done():
		return Record{}, false, parseCtx.Err()
	}
	return parseBackend.externalAtomBackend.Load(parseCtx, parseKey)
}

// TestBindAtomSetBeforeHydrationWinsAndPersists pins that a Set made while the
// initial load is still pending is neither overwritten by the stored value nor
// dropped: it is persisted after the stored version, so it outranks the row.
func TestBindAtomSetBeforeHydrationWinsAndPersists(parseT *testing.T) {
	parseContext, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseName := parseT.Name()
	parseBackend := &gatedLoadBackend{
		externalAtomBackend: externalAtomBackend{
			parseRecord: Record{Key: "k", Value: []byte("5"), Version: 3},
			isFound:     true,
			parseSaved:  make(chan Record, 4),
		},
		parseGate: make(chan struct{}),
	}
	var parseAtom state.Atom[int]
	if _, parseErr := ui.RenderToString(ui.CreateElement(func() ui.Node {
		parseAtom = state.UseAtom(parseName, 1)
		return nil
	})); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseBound := BindAtom(parseContext, parseAtom, "k", Options{Name: parseName, Backend: parseBackend, ExternalInvalidation: true, Strategy: Immediate{}})

	parseBound.Set(9) // lands before the gate opens: engine not yet published
	close(parseBackend.parseGate)

	select {
	case parseRecord := <-parseBackend.parseSaved:
		if string(parseRecord.Value) != "9" || parseRecord.Version != 4 {
			parseT.Fatalf("persisted %q at version %d, want \"9\" at version 4", parseRecord.Value, parseRecord.Version)
		}
	case <-time.After(2 * time.Second):
		parseT.Fatal("a Set made during hydration was never persisted")
	}
	parseDeadline := time.Now().Add(time.Second)
	for parseBound.Loading() {
		if time.Now().After(parseDeadline) {
			parseT.Fatal("binding never finished loading")
		}
		time.Sleep(time.Millisecond)
	}
	if parseGot := parseBound.Get(); parseGot != 9 {
		parseT.Fatalf("atom = %d after hydration, want the local write 9 (stored 5 must not overwrite it)", parseGot)
	}
}

// TestBindAtomCancelReleasesGoroutines pins that cancelling a binding's context
// tears down its hydrate, write and cross-tab watcher goroutines.
func TestBindAtomCancelReleasesGoroutines(parseT *testing.T) {
	runtime.GC()
	parseBefore := runtime.NumGoroutine()
	for parseI := 0; parseI < 10; parseI++ {
		parseContext, parseCancel := context.WithCancel(context.Background())
		parseName := parseT.Name() + string(rune('a'+parseI))
		parseBackend := &externalAtomBackend{parseRecord: Record{Key: "k", Version: 1}, parseSaved: make(chan Record, 4)}
		var parseAtom state.Atom[int]
		if _, parseErr := ui.RenderToString(ui.CreateElement(func() ui.Node {
			parseAtom = state.UseAtom(parseName, 0)
			return nil
		})); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		parseBound := BindAtom(parseContext, parseAtom, "k", Options{Name: parseName, Backend: parseBackend, ExternalInvalidation: true, Strategy: Immediate{}})
		parseDeadline := time.Now().Add(time.Second)
		for parseBound.Loading() && time.Now().Before(parseDeadline) {
			time.Sleep(time.Millisecond)
		}
		parseBound.Set(parseI + 1)
		<-parseBackend.parseSaved
		parseCancel()
	}
	parseDeadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > parseBefore && time.Now().Before(parseDeadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if parseAfter := runtime.NumGoroutine(); parseAfter > parseBefore {
		parseT.Fatalf("goroutines grew from %d to %d after cancelling 10 bindings", parseBefore, parseAfter)
	}
}
