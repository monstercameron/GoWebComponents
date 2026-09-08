//go:build !js || !wasm

package kvstate

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type externalAtomBackend struct {
	parseMutex  sync.Mutex
	parseRecord Record
	isFound     bool
	parseSaved  chan Record
}

// Load returns controllable live records and tombstones.
func (parseBackend *externalAtomBackend) Load(context.Context, string) (Record, bool, error) {
	parseBackend.parseMutex.Lock()
	defer parseBackend.parseMutex.Unlock()
	return parseBackend.parseRecord, parseBackend.isFound, nil
}

// Save records asynchronous writes for deterministic assertions.
func (parseBackend *externalAtomBackend) Save(_ context.Context, parseRecord Record) error {
	parseBackend.parseSaved <- parseRecord
	return nil
}

// Delete is unused by these binding tests.
func (parseBackend *externalAtomBackend) Delete(context.Context, string) error { return nil }

// Keys is unused by these binding tests.
func (parseBackend *externalAtomBackend) Keys(context.Context) ([]string, error) { return nil, nil }

// setRecord controls the next invalidation result.
func (parseBackend *externalAtomBackend) setRecord(parseRecord Record, isFound bool) {
	parseBackend.parseMutex.Lock()
	defer parseBackend.parseMutex.Unlock()
	parseBackend.parseRecord, parseBackend.isFound = parseRecord, isFound
}

// TestBindAtomExternalTombstoneAndCancellation exercises real binding lifecycle and versions.
func TestBindAtomExternalTombstoneAndCancellation(parseTest *testing.T) {
	parseContext, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseName := parseTest.Name()
	parseBackend := &externalAtomBackend{parseRecord: Record{Key: "a", Version: 8}, parseSaved: make(chan Record, 4)}
	var parseAtom state.Atom[int]
	if _, parseErr := ui.RenderToString(ui.CreateElement(func() ui.Node {
		parseAtom = state.UseAtom(parseName, 4)
		return nil
	})); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseBound := BindAtom(parseContext, parseAtom, "a", Options{Name: parseName, Backend: parseBackend, ExternalInvalidation: true, Conflict: Versioned{}, Strategy: Immediate{}})
	parseDeadline := time.Now().Add(time.Second)
	for {
		hubsMu.Lock()
		parseHub := externalHubs[parseName]
		hubsMu.Unlock()
		if parseHub != nil && !parseBound.Loading() {
			break
		}
		if time.Now().After(parseDeadline) {
			parseTest.Fatal("binding never subscribed")
		}
		time.Sleep(time.Millisecond)
	}
	parseBound.Set(9)
	select {
	case parseRecord := <-parseBackend.parseSaved:
		if parseRecord.Version != 9 {
			parseTest.Fatalf("recreation version = %d", parseRecord.Version)
		}
	case <-time.After(time.Second):
		parseTest.Fatal("no persisted write")
	}
	parseBackend.setRecord(Record{Key: "a", Version: 7}, false)
	Invalidate(parseName)
	if parseBound.Get() != 9 {
		parseTest.Fatal("stale tombstone reset local value")
	}
	parseBackend.setRecord(Record{Key: "a", Version: 10}, false)
	Invalidate(parseName)
	if parseBound.Get() != 4 {
		parseTest.Fatal("new tombstone did not reset initial value")
	}
	parseCancel()
	parseBound.Set(77)
	if parseBound.Get() != 4 {
		parseTest.Fatal("cancelled binding changed atom")
	}
	parseDeadline = time.Now().Add(time.Second)
	for {
		hubsMu.Lock()
		parseHub := externalHubs[parseName]
		hubsMu.Unlock()
		if parseHub == nil {
			break
		}
		if time.Now().After(parseDeadline) {
			parseTest.Fatal("cancelled binding retained watcher")
		}
		time.Sleep(time.Millisecond)
	}
}
