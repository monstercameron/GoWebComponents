package kvstate

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

// TestBindingLifetimeLateWatch verifies unmount before hydration cannot leak watchers.
func TestBindingLifetimeLateWatch(parseT *testing.T) {
	for parseIndex := 0; parseIndex < 100; parseIndex++ {
		parseLifetime := newBindingLifetime(context.Background())
		var parseStops atomic.Int32
		var parseWait sync.WaitGroup
		parseWait.Add(2)
		go func() { defer parseWait.Done(); parseLifetime.setStop(func() { parseStops.Add(1) }) }()
		go func() { defer parseWait.Done(); parseLifetime.close() }()
		parseWait.Wait()
		parseLifetime.close()
		if parseStops.Load() != 1 || parseLifetime.parseContext.Err() == nil {
			parseT.Fatal("watcher leaked or released twice")
		}
	}
}
