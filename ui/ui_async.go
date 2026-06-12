//go:build js && wasm

package ui

import (
	"context"
	"reflect"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// UseDeferredValue keeps returning the last committed value until a transition updates it.
func UseDeferredValue[T any](parseValue T) T {
	parseDeferred := UseState(parseValue)
	parseCurrent := parseDeferred.Get()
	parsePrevious := UsePrevious(parseValue)

	UseEffect(func() func() {
		if !parsePrevious.Ok() || reflect.DeepEqual(parseCurrent, parseValue) {
			return nil
		}
		StartTransition(func() {
			parseDeferred.Set(parseValue)
		})
		return nil
	}, parseValue)

	return parseCurrent
}

// UseChannel subscribes the component to values received from ch.
//
// The returned handle exposes the latest observed value, whether a value has
// been received yet, and whether the source channel has closed. When the
// component unmounts or the channel changes, the internal goroutine stops
// reading from the old channel.
func UseChannel[T any](parseCh <-chan T) Channel[T] {
	parseState := UseState(channelSnapshot[T]{})
	parseCurrent := parseState.Get()

	parseVisible := parseCurrent
	if parseCurrent.source != parseCh {
		parseVisible = channelSnapshot[T]{}
	}

	UseEffect(func() func() {
		parseState.Set(channelSnapshot[T]{source: parseCh})
		if parseCh == nil {
			return nil
		}

		parseStop := make(chan struct{})
		go func() {
			defer runtime.RecoverContainedPanic("ui", "UseChannel subscription")
			for {
				select {
				case <-parseStop:
					return
				case parseValue, parseOk := <-parseCh:
					select {
					case <-parseStop:
						return
					default:
					}

					if !parseOk {
						parsePrevious := parseState.Get()
						parseState.Set(channelSnapshot[T]{
							source: parseCh,
							value:  parsePrevious.value,
							ok:     parsePrevious.ok,
							closed: true,
						})
						return
					}

					parseState.Set(channelSnapshot[T]{
						source: parseCh,
						value:  parseValue,
						ok:     true,
						closed: false,
					})
				}
			}
		}()

		return func() {
			close(parseStop)
		}
	}, parseCh)

	return Channel[T]{
		getValue: func() T { return parseVisible.value },
		hasValue: func() bool { return parseVisible.ok },
		isClosed: func() bool { return parseVisible.closed },
	}
}

// Get returns the latest observed channel value or the zero value for T.
func (parseC Channel[T]) Get() T {
	if parseC.getValue == nil {
		var parseZero T
		return parseZero
	}

	return parseC.getValue()
}

// Ok reports whether the channel has produced at least one value.
func (parseC Channel[T]) Ok() bool {
	if parseC.hasValue == nil {
		return false
	}

	return parseC.hasValue()
}

// Closed reports whether the source channel has been observed closing.
func (parseC Channel[T]) Closed() bool {
	if parseC.isClosed == nil {
		return false
	}

	return parseC.isClosed()
}

// UseTask creates a cancellable background task driven by a Go function.
//
// The task only runs when Start is called. In-flight work is cancelled when the
// component unmounts or when Cancel is called explicitly.
//
// The context passed to parseRun is derived from context.Background(). Component
// or request-scoped deadline/value propagation is not currently supported; callers
// that need deadline or value propagation should wrap the provided context inside
// parseRun using context.WithDeadline or context.WithValue before passing it to
// downstream calls.
func UseTask[T any](parseRun func(context.Context) (T, error)) Task[T] {
	parseState := UseState(TaskState[T]{})
	parseCancelRef := UseRef((context.CancelFunc)(nil))
	parseRequestSeq := UseRef(0)

	parseStart := func() {
		if parseCancel := parseCancelRef.Get(); parseCancel != nil {
			parseCancel()
		}

		parseRequestSeq.Set(parseRequestSeq.Get() + 1)
		parseSeq := parseRequestSeq.Get()
		parseCtx, parseCancel2 := context.WithCancel(context.Background())
		parseCancelRef.Set(parseCancel2)

		parseState.Update(func(parsePrev TaskState[T]) TaskState[T] {
			parsePrev.Running = true
			parsePrev.Ready = false
			parsePrev.Cancelled = false
			parsePrev.Started = true
			parsePrev.Error = nil
			return parsePrev
		})

		go func() {
			defer runtime.RecoverContainedPanic("ui", "UseTask runner")
			parseValue, parseErr := parseRun(parseCtx)
			if parseCtx.Err() != nil || parseRequestSeq.Get() != parseSeq {
				return
			}

			parseState.Set(TaskState[T]{
				Value:     parseValue,
				Running:   false,
				Ready:     parseErr == nil,
				Cancelled: false,
				Started:   true,
				Error:     parseErr,
			})
		}()
	}

	parseCancel3 := func() {
		if parseActiveCancel := parseCancelRef.Get(); parseActiveCancel != nil {
			parseActiveCancel()
			parseCancelRef.Set(nil)
		}

		parseState.Update(func(parsePrev2 TaskState[T]) TaskState[T] {
			parsePrev2.Running = false
			parsePrev2.Cancelled = true
			parsePrev2.Started = true
			return parsePrev2
		})
	}

	UseEffect(func() func() {
		return func() {
			if parseActiveCancel2 := parseCancelRef.Get(); parseActiveCancel2 != nil {
				parseActiveCancel2()
				parseCancelRef.Set(nil)
			}
		}
	}, true)

	return Task[T]{
		get:    func() TaskState[T] { return parseState.Get() },
		start:  parseStart,
		cancel: parseCancel3,
	}
}

// Get returns the current task state.
func (parseT Task[T]) Get() TaskState[T] {
	if parseT.get == nil {
		var parseZero TaskState[T]
		return parseZero
	}

	return parseT.get()
}

// Start launches the task, cancelling any previous in-flight run.
func (parseT Task[T]) Start() {
	if parseT.start != nil {
		parseT.start()
	}
}

// Cancel cancels the in-flight task run, if any.
func (parseT Task[T]) Cancel() {
	if parseT.cancel != nil {
		parseT.cancel()
	}
}

// AsyncBoundary renders content, fallback, timeout fallback, or an error
// fallback depending on the current async state.
func AsyncBoundary(parseProps AsyncBoundaryProps) Node {
	parsePhase := UseState(asyncBoundaryState{fallbackVisible: parseProps.Delay <= 0})

	UseEffect(func() func() {
		if !parseProps.Pending || parseProps.Error != nil {
			parsePhase.Set(asyncBoundaryState{fallbackVisible: parseProps.Delay <= 0})
			return nil
		}

		parseState := asyncBoundaryState{fallbackVisible: parseProps.Delay <= 0}
		parsePhase.Set(parseState)

		parseStop := make(chan struct{})
		if parseProps.Delay > 0 {
			go func(parseDelay time.Duration) {
				defer runtime.RecoverContainedPanic("ui", "AsyncBoundary delay timer")
				parseTimer := time.NewTimer(parseDelay)
				defer parseTimer.Stop()

				select {
				case <-parseStop:
					return
				case <-parseTimer.C:
				}

				parseCurrent := parsePhase.Get()
				parseCurrent.fallbackVisible = true
				parsePhase.Set(parseCurrent)
			}(parseProps.Delay)
		}

		if parseProps.Timeout > 0 {
			go func(parseTimeout time.Duration) {
				defer runtime.RecoverContainedPanic("ui", "AsyncBoundary timeout timer")
				parseTimer2 := time.NewTimer(parseTimeout)
				defer parseTimer2.Stop()

				select {
				case <-parseStop:
					return
				case <-parseTimer2.C:
				}

				parseCurrent2 := parsePhase.Get()
				parseCurrent2.timedOut = true
				parsePhase.Set(parseCurrent2)
			}(parseProps.Timeout)
		}

		return func() {
			close(parseStop)
		}
	}, parseProps.Pending, parseProps.Error, parseProps.Delay, parseProps.Timeout)

	parseState2 := parsePhase.Get()
	if parseProps.Error != nil {
		return createAsyncBoundaryElement(parseProps, false, nil)
	}

	if !parseProps.Pending {
		return createAsyncBoundaryElement(parseProps, false, nil)
	}

	if parseState2.timedOut && parseProps.TimeoutFallback != nil {
		return createAsyncBoundaryElement(parseProps, true, parseProps.TimeoutFallback)
	}

	if parseState2.fallbackVisible {
		return createAsyncBoundaryElement(parseProps, true, nil)
	}

	return createAsyncBoundaryElement(parseProps, false, nil)
}

// UseLazyNode asynchronously resolves a ui.Node and tracks loading/error state.
func UseLazyNode(parseLoader func(context.Context) (Node, error), parseDeps ...interface{}) LazyNode {
	parseState := UseState(LazyNodeState{Loading: true})
	parseReloadTick := UseState(0)
	parseCancelRef := UseRef((context.CancelFunc)(nil))
	parseRequestSeq := UseRef(0)

	parseStartLoad := func() {
		if parseCancel := parseCancelRef.Get(); parseCancel != nil {
			parseCancel()
		}

		if parseLoader == nil {
			parseState.Set(LazyNodeState{Error: context.Canceled})
			return
		}

		parseRequestSeq.Set(parseRequestSeq.Get() + 1)
		parseSeq := parseRequestSeq.Get()
		parseCtx, parseCancel2 := context.WithCancel(context.Background())
		parseCancelRef.Set(parseCancel2)

		parseState.Update(func(parsePrev LazyNodeState) LazyNodeState {
			parsePrev.Loading = true
			parsePrev.Error = nil
			return parsePrev
		})

		go func() {
			defer runtime.RecoverContainedPanic("ui", "UseLazyNode loader")
			parseNode, parseErr := parseLoader(parseCtx)
			if parseCtx.Err() != nil || parseRequestSeq.Get() != parseSeq {
				return
			}

			parseState.Set(LazyNodeState{
				Node:    parseNode,
				Loading: false,
				Error:   parseErr,
				Ready:   parseErr == nil,
			})
		}()
	}

	parseEffectDeps := make([]interface{}, 0, len(parseDeps)+1)
	parseEffectDeps = append(parseEffectDeps, parseReloadTick.Get())
	parseEffectDeps = append(parseEffectDeps, parseDeps...)

	UseEffect(func() func() {
		parseStartLoad()
		return func() {
			if parseCancel3 := parseCancelRef.Get(); parseCancel3 != nil {
				parseCancel3()
				parseCancelRef.Set(nil)
			}
		}
	}, parseEffectDeps...)

	return LazyNode{
		get: func() LazyNodeState { return parseState.Get() },
		reload: func() {
			parseReloadTick.Update(func(parsePrev2 int) int { return parsePrev2 + 1 })
		},
		cancel: func() {
			if parseCancel4 := parseCancelRef.Get(); parseCancel4 != nil {
				parseCancel4()
				parseCancelRef.Set(nil)
			}
			parseState.Update(func(parsePrev3 LazyNodeState) LazyNodeState {
				parsePrev3.Loading = false
				return parsePrev3
			})
		},
	}
}

// Get is a core package helper.
func (parseL LazyNode) Get() LazyNodeState {
	if parseL.get == nil {
		return LazyNodeState{}
	}
	return parseL.get()
}

// Reload is a core package helper.
func (parseL LazyNode) Reload() {
	if parseL.reload != nil {
		parseL.reload()
	}
}

// Cancel is a core package helper.
func (parseL LazyNode) Cancel() {
	if parseL.cancel != nil {
		parseL.cancel()
	}
}

// Lazy asynchronously resolves a subtree and renders it through AsyncBoundary.
func Lazy(parseProps LazyProps) Node {
	handle := UseLazyNode(parseProps.Loader, parseProps.Dependencies...)
	parseState := handle.Get()

	return AsyncBoundary(AsyncBoundaryProps{
		Pending:         parseState.Loading,
		Error:           parseState.Error,
		Fallback:        parseProps.Fallback,
		TimeoutFallback: parseProps.TimeoutFallback,
		ErrorFallback:   parseProps.ErrorFallback,
		Content:         parseState.Node,
		Delay:           parseProps.Delay,
		Timeout:         parseProps.Timeout,
	})
}

// UseDebounced returns a delayed view of value that only updates after delay has
// elapsed without a newer value replacing it.
func UseDebounced[T any](parseValue T, parseDelay time.Duration) Debounced[T] {
	parseState := UseState(delayedValueState[T]{value: parseValue})
	parseFirstRun := UseRef(true)
	parseVersion := UseRef(0)

	UseEffect(func() func() {
		if parseFirstRun.Get() {
			parseFirstRun.Set(false)
			return nil
		}

		if parseDelay <= 0 {
			parseState.Set(delayedValueState[T]{value: parseValue})
			return nil
		}

		parseSequence := parseVersion.Get() + 1
		parseVersion.Set(parseSequence)
		parseState.Update(func(parsePrev delayedValueState[T]) delayedValueState[T] {
			parsePrev.pending = true
			return parsePrev
		})

		parseStop := make(chan struct{})
		go func(parseNext T, parseExpected int) {
			defer runtime.RecoverContainedPanic("ui", "UseDebounced timer")
			parseTimer := time.NewTimer(parseDelay)
			defer parseTimer.Stop()

			select {
			case <-parseStop:
				return
			case <-parseTimer.C:
			}

			if parseVersion.Get() != parseExpected {
				return
			}
			parseState.Set(delayedValueState[T]{value: parseNext})
		}(parseValue, parseSequence)

		return func() {
			close(parseStop)
		}
	}, parseValue, parseDelay)

	return Debounced[T]{
		get:     func() T { return parseState.Get().value },
		pending: func() bool { return parseState.Get().pending },
	}
}

// Get is a core package helper.
func (parseD Debounced[T]) Get() T {
	if parseD.get == nil {
		var parseZero T
		return parseZero
	}
	return parseD.get()
}

// Pending is a core package helper.
func (parseD Debounced[T]) Pending() bool {
	if parseD.pending == nil {
		return false
	}
	return parseD.pending()
}

// UseThrottled returns a trailing-throttled view of value that updates at most
// once per interval while still eventually applying the latest value.
func UseThrottled[T any](parseValue T, parseInterval time.Duration) Throttled[T] {
	parseState := UseState(delayedValueState[T]{value: parseValue})
	parseFirstRun := UseRef(true)
	parseVersion := UseRef(0)
	parseLastEmit := UseRef(time.Time{})

	UseEffect(func() func() {
		if parseFirstRun.Get() {
			parseFirstRun.Set(false)
			parseLastEmit.Set(time.Now())
			return nil
		}

		parseNow := time.Now()
		if parseInterval <= 0 {
			parseLastEmit.Set(parseNow)
			parseState.Set(delayedValueState[T]{value: parseValue})
			return nil
		}

		if parseEmittedAt := parseLastEmit.Get(); parseEmittedAt.IsZero() || parseNow.Sub(parseEmittedAt) >= parseInterval {
			parseLastEmit.Set(parseNow)
			parseState.Set(delayedValueState[T]{value: parseValue})
			return nil
		}

		parseSequence := parseVersion.Get() + 1
		parseVersion.Set(parseSequence)
		parseState.Update(func(parsePrev delayedValueState[T]) delayedValueState[T] {
			parsePrev.pending = true
			return parsePrev
		})

		parseRemaining := parseInterval - parseNow.Sub(parseLastEmit.Get())
		parseStop := make(chan struct{})
		go func(parseNext T, parseExpected int, parseWait time.Duration) {
			defer runtime.RecoverContainedPanic("ui", "UseThrottled timer")
			parseTimer := time.NewTimer(parseWait)
			defer parseTimer.Stop()

			select {
			case <-parseStop:
				return
			case <-parseTimer.C:
			}

			if parseVersion.Get() != parseExpected {
				return
			}
			parseLastEmit.Set(time.Now())
			parseState.Set(delayedValueState[T]{value: parseNext})
		}(parseValue, parseSequence, parseRemaining)

		return func() {
			close(parseStop)
		}
	}, parseValue, parseInterval)

	return Throttled[T]{
		get:     func() T { return parseState.Get().value },
		pending: func() bool { return parseState.Get().pending },
	}
}

// Get is a core package helper.
func (parseT Throttled[T]) Get() T {
	if parseT.get == nil {
		var parseZero T
		return parseZero
	}
	return parseT.get()
}

// Pending is a core package helper.
func (parseT Throttled[T]) Pending() bool {
	if parseT.pending == nil {
		return false
	}
	return parseT.pending()
}
