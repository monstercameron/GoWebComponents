package kvstate

import (
	"context"
	"sync"
)

// bindingLifetime owns an asynchronous binding's context and late-installed watcher.
type bindingLifetime struct {
	parseContext context.Context
	parseCancel  context.CancelFunc
	parseMutex   sync.Mutex
	parseStop    func()
}

// newBindingLifetime creates a cancellable scope for hydration and invalidation.
func newBindingLifetime(parseParent context.Context) *bindingLifetime {
	parseContext, parseCancel := context.WithCancel(parseParent)
	return &bindingLifetime{parseContext: parseContext, parseCancel: parseCancel}
}

// setStop immediately releases a watcher installed after the binding was closed.
func (parseLifetime *bindingLifetime) setStop(parseStop func()) {
	parseLifetime.parseMutex.Lock()
	if parseLifetime.parseContext.Err() != nil {
		parseLifetime.parseMutex.Unlock()
		parseStop()
		return
	}
	parseLifetime.parseStop = parseStop
	parseLifetime.parseMutex.Unlock()
}

// close cancels blocked loads before releasing the watcher, without holding locks.
func (parseLifetime *bindingLifetime) close() {
	parseLifetime.parseCancel()
	parseLifetime.parseMutex.Lock()
	parseStop := parseLifetime.parseStop
	parseLifetime.parseStop = nil
	parseLifetime.parseMutex.Unlock()
	if parseStop != nil {
		parseStop()
	}
}
