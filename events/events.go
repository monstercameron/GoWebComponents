// Package events provides a typed in-app publish/subscribe fan-out bus with
// component-lifecycle-tied subscriptions for the GoWebComponents framework.
//
// Every topic is a plain string key. Subscribers are typed via generics; a
// subscriber registered for type T on topic "foo" only receives publications
// of type T on that same topic — mismatched types are silently skipped.
//
// The package works without a running render loop (unit-testable on native Go)
// and compiles for both native and GOOS=js GOARCH=wasm targets.
package events

import (
	"sync"
	"sync/atomic"

	"github.com/monstercameron/GoWebComponents/ui"
)

// ---------------------------------------------------------------------------
// Internal registry
// ---------------------------------------------------------------------------

// entry holds one subscriber in the per-topic list.
type entry struct {
	id      uint64
	deliver func(parseValue any)
}

// topicState is the mutable state kept per topic string.
type topicState struct {
	mu        sync.Mutex
	entries   []entry
	hasLast   bool
	lastValue any
}

// globalRegistry is the process-wide topic registry. Keys are topic strings,
// values are *topicState.
var globalRegistry sync.Map

// globalIDCounter generates monotonically increasing subscriber IDs.
var globalIDCounter atomic.Uint64

// nextID returns the next unique subscriber ID.
func nextID() uint64 {
	return globalIDCounter.Add(1)
}

// stateFor returns the existing *topicState for a topic or creates one.
func stateFor(parseTopic string) *topicState {
	parseLoaded, _ := globalRegistry.LoadOrStore(parseTopic, &topicState{})
	return parseLoaded.(*topicState)
}

// ---------------------------------------------------------------------------
// TopicOption
// ---------------------------------------------------------------------------

// TopicOption is a functional option accepted by UseTopic.
type TopicOption struct {
	parseReplayLast bool
}

// WithReplayLast returns a TopicOption that causes a new subscriber to
// immediately receive the most recently published value on its topic, if any
// value has been published since the program started.
func WithReplayLast() TopicOption {
	return TopicOption{parseReplayLast: true}
}

// ---------------------------------------------------------------------------
// Core API (works without a render loop; unit-test directly)
// ---------------------------------------------------------------------------

// Subscribe registers parseHandler to receive every future value published on
// parseTopic whose type matches T. It returns an unsubscribe function; calling
// it removes the handler and, when the last subscriber leaves, clears the
// internal registry entry for that topic.
//
// Subscribe is goroutine-safe. The handler is invoked synchronously from
// Publish; a panicking handler is contained so it cannot stop delivery to
// sibling subscribers.
func Subscribe[T any](parseTopic string, parseHandler func(T)) (parseUnsubscribe func()) {
	return subscribeInternal(parseTopic, parseHandler, false)
}

// subscribeInternal is the shared implementation used by Subscribe and
// UseTopic. parseReplay controls whether the last published value is
// delivered immediately.
func subscribeInternal[T any](parseTopic string, parseHandler func(T), parseReplay bool) (parseUnsubscribe func()) {
	parseState := stateFor(parseTopic)
	parseID := nextID()

	parseWrapper := func(parseValue any) {
		parseTyped, parseOk := parseValue.(T)
		if !parseOk {
			return
		}
		// Contain panics so one bad subscriber cannot stop the fan-out.
		defer func() { recover() }() //nolint:errcheck
		parseHandler(parseTyped)
	}

	parseState.mu.Lock()
	parseState.entries = append(parseState.entries, entry{id: parseID, deliver: parseWrapper})
	// Capture replay snapshot while holding the lock so we see a consistent
	// last value.
	parseDoReplay := parseReplay && parseState.hasLast
	parseReplayValue := parseState.lastValue
	parseState.mu.Unlock()

	if parseDoReplay {
		parseWrapper(parseReplayValue)
	}

	return func() {
		parseState.mu.Lock()
		parsePrev := parseState.entries
		parseNext := make([]entry, 0, len(parsePrev))
		for _, parseE := range parsePrev {
			if parseE.id != parseID {
				parseNext = append(parseNext, parseE)
			}
		}
		parseState.entries = parseNext
		parseState.mu.Unlock()
	}
}

// Publish delivers parseValue to every subscriber currently registered on
// parseTopic whose handler type matches T. Delivery order matches registration
// order. A panicking subscriber is contained; other subscribers still receive
// the value. Publish is goroutine-safe.
func Publish[T any](parseTopic string, parseValue T) {
	parseState := stateFor(parseTopic)

	parseState.mu.Lock()
	// Record the last value for WithReplayLast subscribers.
	parseState.hasLast = true
	parseState.lastValue = parseValue
	// Snapshot the subscriber list so we release the lock before calling
	// handlers (handlers may themselves call Subscribe/Publish).
	parseSnapshot := make([]entry, len(parseState.entries))
	copy(parseSnapshot, parseState.entries)
	parseState.mu.Unlock()

	for _, parseE := range parseSnapshot {
		parseE.deliver(parseValue)
	}
}

// ---------------------------------------------------------------------------
// TopicHandle
// ---------------------------------------------------------------------------

// TopicHandle is returned by UseTopic and lets the caller publish values on
// the topic it was created for.
type TopicHandle[T any] struct {
	parseTopic string
}

// Topic returns the topic name this handle publishes to, for logging and comparison.
func (parseH TopicHandle[T]) Topic() string { return parseH.parseTopic }

// Publish delivers parseValue to all subscribers currently registered on the
// topic this handle was created for.
func (parseH TopicHandle[T]) Publish(parseValue T) {
	Publish[T](parseH.parseTopic, parseValue)
}

// ---------------------------------------------------------------------------
// Hook API (component-lifecycle-tied)
// ---------------------------------------------------------------------------

// UseTopic wires a typed pub/sub subscription to a component's lifetime via
// UseEffect. When parseHandler is non-nil the subscription is established
// during the effect phase and automatically removed when the component
// unmounts or when parseTopic changes. Pass nil for a publish-only component.
// Zero or more TopicOption values fine-tune subscription behaviour.
//
// UseTopic returns a TopicHandle whose Publish method sends values on the
// topic. Because UseEffect is a no-op on native (non-wasm) builds,
// UseTopic's subscription side is also a no-op on native; use the core
// Subscribe/Publish functions directly in unit tests.
func UseTopic[T any](parseTopic string, parseHandler func(T), parseOptions ...TopicOption) TopicHandle[T] {
	parseReplay := false
	for _, parseOpt := range parseOptions {
		if parseOpt.parseReplayLast {
			parseReplay = true
		}
	}

	ui.UseEffect(func() func() {
		if parseHandler == nil {
			return nil
		}
		parseUnsub := subscribeInternal(parseTopic, parseHandler, parseReplay)
		return parseUnsub
	}, parseTopic)

	return TopicHandle[T]{parseTopic: parseTopic}
}

// ---------------------------------------------------------------------------
// Test helper (unexported, same package)
// ---------------------------------------------------------------------------

// subscriberCount returns the number of handlers currently registered for
// parseTopic. It is intended for use in package tests to assert that
// unsubscribe leaves no leaked entries.
func subscriberCount(parseTopic string) int {
	parseLoaded, parseOk := globalRegistry.Load(parseTopic)
	if !parseOk {
		return 0
	}
	parseState := parseLoaded.(*topicState)
	parseState.mu.Lock()
	parseCount := len(parseState.entries)
	parseState.mu.Unlock()
	return parseCount
}
