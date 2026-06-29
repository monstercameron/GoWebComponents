package kvstate

import (
	"context"
	"sync"
	"time"
)

// WriteStrategy decides when a dirty value becomes durable. After each Set the
// binding calls OnWrite with a flush closure; the strategy chooses when (and how
// often) to invoke it. Close performs a final flush on teardown.
type WriteStrategy interface {
	OnWrite(parseCtx context.Context, parseKey string, parseFlush func(context.Context) error)
	Close(parseCtx context.Context) error
}

// Immediate flushes on every write. Simplest and safest; most IndexedDB traffic.
type Immediate struct{}

func (Immediate) OnWrite(parseCtx context.Context, parseKey string, parseFlush func(context.Context) error) {
	_ = parseFlush(parseCtx)
}

func (Immediate) Close(parseCtx context.Context) error { return nil }

// Debounced coalesces bursts of writes, flushing once after d of quiet. Good
// default for interactive state (typing, sliders).
func Debounced(parseD time.Duration) WriteStrategy {
	return &debouncedStrategy{d: parseD}
}

type debouncedStrategy struct {
	d         time.Duration
	mu        sync.Mutex
	timer     *time.Timer
	lastFlush func(context.Context) error
}

func (parseS *debouncedStrategy) OnWrite(parseCtx context.Context, parseKey string, parseFlush func(context.Context) error) {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	parseS.lastFlush = parseFlush
	if parseS.timer != nil {
		parseS.timer.Stop()
	}
	parseS.timer = time.AfterFunc(parseS.d, func() {
		parseS.mu.Lock()
		parseFn := parseS.lastFlush
		parseS.mu.Unlock()
		if parseFn != nil {
			_ = parseFn(context.Background())
		}
	})
}

func (parseS *debouncedStrategy) Close(parseCtx context.Context) error {
	parseS.mu.Lock()
	if parseS.timer != nil {
		parseS.timer.Stop()
	}
	parseFn := parseS.lastFlush
	parseS.mu.Unlock()
	if parseFn != nil {
		return parseFn(parseCtx)
	}
	return nil
}

// OnUnload defers flushing until teardown (Close) and, in the browser, also
// flushes on page hide. It minimizes writes for state that only needs to survive
// a reload, not every keystroke.
func OnUnload() WriteStrategy {
	return &onUnloadStrategy{}
}

type onUnloadStrategy struct {
	mu        sync.Mutex
	lastFlush func(context.Context) error
	hooked    bool
}

func (parseS *onUnloadStrategy) OnWrite(parseCtx context.Context, parseKey string, parseFlush func(context.Context) error) {
	parseS.mu.Lock()
	parseS.lastFlush = parseFlush
	parseNeedHook := !parseS.hooked
	parseS.hooked = true
	parseS.mu.Unlock()
	if parseNeedHook {
		registerUnloadFlush(func() {
			parseS.mu.Lock()
			parseFn := parseS.lastFlush
			parseS.mu.Unlock()
			if parseFn != nil {
				_ = parseFn(context.Background())
			}
		})
	}
}

func (parseS *onUnloadStrategy) Close(parseCtx context.Context) error {
	parseS.mu.Lock()
	parseFn := parseS.lastFlush
	parseS.mu.Unlock()
	if parseFn != nil {
		return parseFn(parseCtx)
	}
	return nil
}
