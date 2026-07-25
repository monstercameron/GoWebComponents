package ui

import "github.com/monstercameron/GoWebComponents/v5/internal/runtime"

// DeferStatus is the state of a deferred, possibly-async view. It models Angular @defer's
// blocks: a placeholder before work begins, a loading block while an async resource resolves,
// the content once ready, and an error block on failure.
type DeferStatus int

const (
	// DeferPlaceholder is the pre-trigger state: the deferred work has not started.
	DeferPlaceholder DeferStatus = iota
	// DeferLoading is shown while the deferred async resource is resolving.
	DeferLoading
	// DeferReady is shown once the resource resolved successfully.
	DeferReady
	// DeferError is shown when the resource failed to resolve.
	DeferError
)

// DeferState[T] is the current state of a deferred view: its status plus the resolved data or
// error. It is a small value; copy it freely.
type DeferState[T any] struct {
	Status DeferStatus
	Data   T
	Err    error
}

// DeferBlocks[T] supplies the renderers for each state of a deferred view — the @defer sub-block
// set. Placeholder and Loading are optional (a nil renderer contributes nothing); Content
// renders the resolved data and Error renders the failure.
type DeferBlocks[T any] struct {
	Placeholder func() Node
	Loading     func() Node
	Content     func(T) Node
	Error       func(error) Node
}

// RenderDefer selects the block to render for a deferred view's current state — the pure heart of
// the @defer state machine. It is independent of the hook/render lifecycle, so the block-routing
// logic is unit-testable without a browser:
//
//	state := ui.UseAsyncDefer(visible, loadReport)
//	return ui.RenderDefer(state, ui.DeferBlocks[Report]{
//	    Placeholder: func() ui.Node { return h.Div(h.Text("scroll to load")) },
//	    Loading:     func() ui.Node { return h.Spinner() },
//	    Content:     func(r Report) ui.Node { return reportView(r) },
//	    Error:       func(err error) ui.Node { return h.Div(h.Text(err.Error())) },
//	})
func RenderDefer[T any](parseState DeferState[T], parseBlocks DeferBlocks[T]) Node {
	switch parseState.Status {
	case DeferLoading:
		if parseBlocks.Loading != nil {
			return parseBlocks.Loading()
		}
		return nil
	case DeferReady:
		if parseBlocks.Content != nil {
			return parseBlocks.Content(parseState.Data)
		}
		return nil
	case DeferError:
		if parseBlocks.Error != nil {
			return parseBlocks.Error(parseState.Err)
		}
		return nil
	default:
		if parseBlocks.Placeholder != nil {
			return parseBlocks.Placeholder()
		}
		return nil
	}
}

// UseAsyncDefer drives a deferred async resource through the placeholder → loading → ready/error
// states. It does nothing until triggered first becomes true (pair it with UseIntersection /
// UseIdle / UseTimerTrigger); then it runs load exactly once, showing Loading while it resolves
// and Ready or Error when it settles, re-rendering on each transition. Once started it never
// reverts to the placeholder. Pair it with RenderDefer to render the matching block.
//
// Natively (SSR/tests) effects are no-ops, so the resource is resolved in the browser; the
// returned state is always a valid snapshot to render.
func UseAsyncDefer[T any](parseTriggered bool, parseLoad func() (T, error)) DeferState[T] {
	parseState := UseState(DeferState[T]{Status: DeferPlaceholder})
	parseStarted := UseRef(false)
	parseShouldStart := parseTriggered && !parseStarted.Get()

	UseEffect(func() func() {
		if !parseShouldStart || parseLoad == nil {
			return nil
		}
		parseStarted.Set(true)
		parseState.Set(DeferState[T]{Status: DeferLoading})
		go func() {
			// Contain loader panics like every sibling async hook: an
			// unrecovered goroutine panic kills the whole wasm program.
			defer runtime.RecoverContainedPanic("ui", "UseAsyncDefer loader")
			parseData, parseErr := parseLoad()
			if parseErr != nil {
				parseState.Set(DeferState[T]{Status: DeferError, Err: parseErr})
				return
			}
			parseState.Set(DeferState[T]{Status: DeferReady, Data: parseData})
		}()
		return nil
	}, parseShouldStart)

	return parseState.Get()
}
