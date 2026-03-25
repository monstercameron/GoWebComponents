package render

import (
	"context"
	"sync"
)

type ResourceAttempt struct {
	Index     int
	Cancelled bool
}

type resourceOutcome[T any] struct {
	value T
	err   error
}

type resourcePending[T any] struct {
	index int
	done  chan resourceOutcome[T]
}

// ResourceController provides deterministic control over one async resource loader in tests.
type ResourceController[T any] struct {
	mu       sync.Mutex
	attempts []ResourceAttempt
	pending  *resourcePending[T]
	started  chan int
}

// NewResourceController creates one async controller for fetch/resource tests.
func NewResourceController[T any]() *ResourceController[T] {
	return &ResourceController[T]{
		started: make(chan int, 16),
	}
}

// Loader returns a loader function compatible with fetch.UseResource or fetch.UseCachedResource.
func (c *ResourceController[T]) Loader() func(context.Context) (T, error) {
	return c.Await
}

// Await blocks until the current attempt is resolved, rejected, or cancelled.
func (c *ResourceController[T]) Await(ctx context.Context) (T, error) {
	var zero T
	if c == nil {
		return zero, nil
	}

	pending := &resourcePending[T]{done: make(chan resourceOutcome[T], 1)}
	c.mu.Lock()
	pending.index = len(c.attempts) + 1
	c.attempts = append(c.attempts, ResourceAttempt{Index: pending.index})
	c.pending = pending
	c.mu.Unlock()

	select {
	case c.started <- pending.index:
	default:
	}

	select {
	case outcome := <-pending.done:
		return outcome.value, outcome.err
	case <-ctx.Done():
		c.mu.Lock()
		if c.pending == pending {
			c.pending = nil
			c.attempts[pending.index-1].Cancelled = true
		}
		c.mu.Unlock()
		return zero, ctx.Err()
	}
}

// Resolve completes the current pending attempt successfully.
func (c *ResourceController[T]) Resolve(value T) {
	if c == nil {
		return
	}
	c.finish(resourceOutcome[T]{value: value})
}

// Reject completes the current pending attempt with an error.
func (c *ResourceController[T]) Reject(err error) {
	if c == nil {
		return
	}
	c.finish(resourceOutcome[T]{err: err})
}

// Cancel completes the current pending attempt with context cancellation.
func (c *ResourceController[T]) Cancel() {
	if c == nil {
		return
	}
	c.finish(resourceOutcome[T]{err: context.Canceled}, true)
}

// Pending reports whether one attempt is currently stalled.
func (c *ResourceController[T]) Pending() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pending != nil
}

// AttemptCount reports how many attempts have started.
func (c *ResourceController[T]) AttemptCount() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.attempts)
}

// Attempts returns a snapshot of all started attempts.
func (c *ResourceController[T]) Attempts() []ResourceAttempt {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ResourceAttempt(nil), c.attempts...)
}

// Started returns a channel that receives each started attempt index.
func (c *ResourceController[T]) Started() <-chan int {
	if c == nil {
		return nil
	}
	return c.started
}

func (c *ResourceController[T]) finish(outcome resourceOutcome[T], markCancelled ...bool) {
	c.mu.Lock()
	pending := c.pending
	if pending != nil {
		c.pending = nil
		if len(markCancelled) > 0 && markCancelled[0] {
			c.attempts[pending.index-1].Cancelled = true
		}
	}
	c.mu.Unlock()
	if pending != nil {
		pending.done <- outcome
	}
}
