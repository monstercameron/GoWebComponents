//go:build js && wasm
// +build js,wasm

package ui

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/platform/jsdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

const propsKey = "__ui_props"

var runtimeInitialized bool

type componentMeta struct {
	hasArg  bool
	argType reflect.Type
	zeroArg reflect.Value
}

var componentMetaCache sync.Map

// Node is the public UI tree node type.
type Element = runtime.Element
type Node = *runtime.Element

type Event = runtime.GoEvent
type MouseEvent = runtime.GoEvent
type InputEvent = runtime.GoEvent
type ChangeEvent = runtime.GoEvent
type KeyboardEvent = runtime.GoEvent
type FocusEvent = runtime.GoEvent
type FormEvent = runtime.GoEvent

// Handler stores an event handler value in a form the runtime can consume.
type Handler struct {
	value interface{}
}

// ReactiveSource identifies shared values that a ReactiveRegion can subscribe to explicitly.
type ReactiveSource interface {
	ReactiveRegionSourceIDs() []string
}

// PortalTarget describes where a portal subtree should render.
type PortalTarget struct {
	Selector string
	Node     interface{}
}

// PortalProps configures a portal target and its children.
type PortalProps struct {
	Target   PortalTarget
	Child    Node
	Children []Node
}

// State provides access to hook-managed local state.
type State[T any] struct {
	get func() T
	set func(interface{})
}

// Reducer provides access to reducer-style local state transitions.
type Reducer[S any, A any] struct {
	get      func() S
	dispatch func(A)
}

// Ref stores a stable mutable reference across renders.
type Ref[T any] struct {
	raw *runtime.RefValue
}

// Transition exposes transition-pending state and a transition starter.
type Transition struct {
	pending func() bool
	start   func(func())
}

// Previous exposes the previous committed value for a hook call.
// The zero value is safe to use and reports no available previous value.
type Previous[T any] struct {
	value func() T
	ok    func() bool
}

type previousState[T any] struct {
	value T
	ok    bool
}

// Channel exposes the latest value observed from a channel.
// The zero value is safe to use and reports no available value.
type Channel[T any] struct {
	getValue func() T
	hasValue func() bool
	isClosed func() bool
}

type channelSnapshot[T any] struct {
	source <-chan T
	value  T
	ok     bool
	closed bool
}

// TaskState describes the lifecycle of a cancellable background task.
type TaskState[T any] struct {
	Value     T
	Running   bool
	Ready     bool
	Cancelled bool
	Started   bool
	Error     error
}

// Task exposes control over a cancellable background job.
// The zero value is safe to use.
type Task[T any] struct {
	get    func() TaskState[T]
	start  func()
	cancel func()
}

type AsyncBoundaryProps struct {
	Pending         bool
	Error           error
	Fallback        Node
	TimeoutFallback Node
	ErrorFallback   func(error) Node
	Content         Node
	Delay           time.Duration
	Timeout         time.Duration
}

type ErrorBoundaryProps struct {
	Fallback      Node
	ErrorFallback func(error, func()) Node
	OnError       func(error)
	Child         Node
	Children      []Node
	ResetKeys     []interface{}
}

type LazyNodeState struct {
	Node    Node
	Loading bool
	Error   error
	Ready   bool
}

type LazyNode struct {
	get    func() LazyNodeState
	reload func()
	cancel func()
}

type LazyProps struct {
	Loader          func(context.Context) (Node, error)
	Dependencies    []interface{}
	Fallback        Node
	TimeoutFallback Node
	ErrorFallback   func(error) Node
	Delay           time.Duration
	Timeout         time.Duration
}

type Debounced[T any] struct {
	get     func() T
	pending func() bool
}

type Throttled[T any] struct {
	get     func() T
	pending func() bool
}

type delayedValueState[T any] struct {
	value   T
	pending bool
}

type asyncBoundaryState struct {
	fallbackVisible bool
	timedOut        bool
}

type errorBoundaryComponent struct {
	boundaryType *runtime.ErrorBoundaryType
}

type runtimeErrorBoundaryComponent interface {
	runtimeErrorBoundary() *runtime.ErrorBoundaryType
}

// ErrorBoundary creates a subtree boundary with fallback rendering and reset behavior.
var ErrorBoundary = &errorBoundaryComponent{boundaryType: runtime.NewErrorBoundaryType()}

// CreateElement creates a UI node from a component function, provider, boundary, or existing node.
func CreateElement(parseComponent interface{}, parseProps ...interface{}) Node {
	if parseNode, parseOk := parseComponent.(*runtime.Element); parseOk && len(parseProps) == 0 {
		return parseNode
	}
	if parseBoundary, parseOk2 := parseComponent.(runtimeErrorBoundaryComponent); parseOk2 {
		var parseRawProps interface{}
		if len(parseProps) > 0 {
			parseRawProps = parseProps[0]
		}
		return createErrorBoundaryElement(parseBoundary, parseRawProps)
	}
	if parseProvider, parseOk3 := parseComponent.(contextProviderComponent); parseOk3 {
		var parseRawProps2 interface{}
		if len(parseProps) > 0 {
			parseRawProps2 = parseProps[0]
		}
		return createContextProviderElement(parseProvider, parseRawProps2)
	}
	var parseRawProps3 map[string]interface{}
	if len(parseProps) > 0 {
		parseRawProps3 = map[string]interface{}{}
		parseRawProps3[propsKey] = parseProps[0]
	}

	return runtime.CreateElement(getComponentHandle(parseComponent), parseRawProps3)
}

// runtimeErrorBoundary is a core package helper.
func (parseBoundary *errorBoundaryComponent) runtimeErrorBoundary() *runtime.ErrorBoundaryType {
	if parseBoundary == nil {
		return nil
	}
	return parseBoundary.boundaryType
}

// Fragment groups children without introducing an extra host element.
func Fragment(parseChildren ...Node) Node {
	return runtime.CreateElement("FRAGMENT", nil, toInterfaces(parseChildren)...)
}

// ReactiveRegion creates an explicit fine-grained subscribed region whose subtree can rerender
// without rerendering the owning component.
func ReactiveRegion(render func() Node, parseSources ...ReactiveSource) Node {
	parseIds := make([]string, 0, len(parseSources))
	parseSeen := make(map[string]struct{}, len(parseSources))
	for _, parseSource := range parseSources {
		if parseSource == nil {
			continue
		}
		for _, parseId := range parseSource.ReactiveRegionSourceIDs() {
			if parseId == "" {
				continue
			}
			if _, parseExists := parseSeen[parseId]; parseExists {
				continue
			}
			parseSeen[parseId] = struct{}{}
			parseIds = append(parseIds, parseId)
		}
	}
	return runtime.CreateElement(runtime.ReactiveRegionNodeType, map[string]interface{}{
		"__gwc_reactive_region_source_ids": parseIds,
		"__gwc_reactive_region_render": func() *runtime.Element {
			if render == nil {
				return nil
			}
			return render()
		},
	})
}

// Portal renders children into a separate target container while keeping logical ownership in the current tree.
func Portal(parseProps PortalProps) Node {
	parseChildren := make([]interface{}, 0, len(parseProps.Children)+1)
	if parseProps.Child != nil {
		parseChildren = append(parseChildren, parseProps.Child)
	}
	parseChildren = append(parseChildren, toInterfaces(parseProps.Children)...)

	parseRawProps := map[string]interface{}{}
	if parseProps.Target.Selector != "" {
		parseRawProps["portalTargetSelector"] = parseProps.Target.Selector
	}
	if parseProps.Target.Node != nil {
		parseRawProps["portalTargetNode"] = parseProps.Target.Node
	}

	return runtime.CreateElement(runtime.PortalNodeType, parseRawProps, parseChildren...)
}

// Render mounts the UI tree into the DOM element matched by selector.
func Render(parseRoot Node, parseSelector string) {
	ensureInitialized()
	parseRt := runtime.GetGlobalRuntime()
	parseRt.BeginStartupProfiling("render")
	if parseRt.HasPendingHotReloadSnapshot() {
		parseRt.HydrateTo(parseSelector, parseRoot)
		return
	}
	parseRt.RenderTo(parseSelector, parseRoot)
}

// RenderInto mounts the UI tree into an explicit DOM node.
func RenderInto(parseRoot Node, parseTarget interface{}) error {
	ensureInitialized()
	parseRt := runtime.GetGlobalRuntime()
	parseRt.BeginStartupProfiling("render")
	return parseRt.RenderInto(parseTarget, parseRoot)
}

// Hydrate is the public client-resume entrypoint for SSR hydration.
//
// It restores the optional bootstrap payload, reuses matching server-rendered
// DOM where possible, and falls back per subtree when hydration cannot
// continue safely.
func Hydrate(parseRoot Node, parseSelector string, parseOptions ...HydrationOptions) (SSRBootstrap, error) {
	runtime.BeginStartupProfiling("hydrate")
	parseBootstrapStarted := time.Now()
	parseBootstrapSource := "options"
	parseResolved := resolveHydrationOptions(parseOptions)
	parsePayload := parseResolved.Bootstrap
	switch {
	case parseResolved.ScriptID != "":
		parseParsed, parseErr := ReadBootstrapScript(parseResolved.ScriptID)
		if parseErr != nil {
			return SSRBootstrap{}, parseErr
		}
		parsePayload = parseParsed
		parseBootstrapSource = "script"
	case parseResolved.ReferenceScriptID != "":
		parseRef, parseErr2 := ReadBootstrapReferenceScript(parseResolved.ReferenceScriptID)
		if parseErr2 != nil {
			return SSRBootstrap{}, parseErr2
		}
		parseParsed2, parseErr2 := ReadBootstrapReference(parseRef)
		if parseErr2 != nil {
			return SSRBootstrap{}, parseErr2
		}
		parsePayload = parseParsed2
		parseBootstrapSource = "reference-script"
	case parseResolved.BootstrapRef.URL != "":
		parseParsed3, parseErr3 := ReadBootstrapReference(parseResolved.BootstrapRef)
		if parseErr3 != nil {
			return SSRBootstrap{}, parseErr3
		}
		parsePayload = parseParsed3
		parseBootstrapSource = "reference-url"
	}
	runtime.RecordStartupBootstrapRead(time.Since(parseBootstrapStarted).Nanoseconds(), parseBootstrapSource)
	storeHydrationStartupCost(parsePayload)
	if parseResolved.Observability.CorrelationID == "" && parsePayload.CorrelationID != "" {
		parseResolved.Observability.CorrelationID = parsePayload.CorrelationID
	}
	ensureInitialized()
	parseRt := runtime.GetGlobalRuntime()
	setParallelRegionHydrationObserver(parseRt, parseResolved.Observability.CorrelationID, func(parseMetrics runtime.HydrationMetrics) {
		dispatchSSRObservation(parseResolved.Observability, newSSRHydrationObservation(parseMetrics))
	})
	if parsePayload.IDSeed > 0 {
		parseRt.SetIDSeed(parsePayload.IDSeed)
	}
	if len(parsePayload.Atoms) > 0 {
		if parseErr4 := parseRt.RestoreAtomSnapshot(parsePayload.Atoms); parseErr4 != nil {
			return SSRBootstrap{}, parseErr4
		}
	}
	parseRt.SetNextHydrationStrict(parseResolved.Strict)
	parseRt.HydrateTo(parseSelector, parseRoot)
	if parseHydrationBridgeErr := handleParallelRegionHydrationSelector(parseSelector); parseHydrationBridgeErr != nil {
		return SSRBootstrap{}, parseHydrationBridgeErr
	}
	return parsePayload, nil
}

// HydrateInto resumes a UI tree into an explicit DOM node.
func HydrateInto(parseRoot Node, parseTarget interface{}, parseOptions ...HydrationOptions) (SSRBootstrap, error) {
	runtime.BeginStartupProfiling("hydrate")
	parseBootstrapStarted := time.Now()
	parseBootstrapSource := "options"
	parseResolved := resolveHydrationOptions(parseOptions)
	parsePayload := parseResolved.Bootstrap
	switch {
	case parseResolved.ScriptID != "":
		parseParsed, parseErr := ReadBootstrapScript(parseResolved.ScriptID)
		if parseErr != nil {
			return SSRBootstrap{}, parseErr
		}
		parsePayload = parseParsed
		parseBootstrapSource = "script"
	case parseResolved.ReferenceScriptID != "":
		parseRef, parseErr2 := ReadBootstrapReferenceScript(parseResolved.ReferenceScriptID)
		if parseErr2 != nil {
			return SSRBootstrap{}, parseErr2
		}
		parseParsed2, parseErr2 := ReadBootstrapReference(parseRef)
		if parseErr2 != nil {
			return SSRBootstrap{}, parseErr2
		}
		parsePayload = parseParsed2
		parseBootstrapSource = "reference-script"
	case parseResolved.BootstrapRef.URL != "":
		parseParsed3, parseErr3 := ReadBootstrapReference(parseResolved.BootstrapRef)
		if parseErr3 != nil {
			return SSRBootstrap{}, parseErr3
		}
		parsePayload = parseParsed3
		parseBootstrapSource = "reference-url"
	}
	runtime.RecordStartupBootstrapRead(time.Since(parseBootstrapStarted).Nanoseconds(), parseBootstrapSource)
	storeHydrationStartupCost(parsePayload)
	if parseResolved.Observability.CorrelationID == "" && parsePayload.CorrelationID != "" {
		parseResolved.Observability.CorrelationID = parsePayload.CorrelationID
	}
	ensureInitialized()
	parseRt := runtime.GetGlobalRuntime()
	setParallelRegionHydrationObserver(parseRt, parseResolved.Observability.CorrelationID, func(parseMetrics runtime.HydrationMetrics) {
		dispatchSSRObservation(parseResolved.Observability, newSSRHydrationObservation(parseMetrics))
	})
	if parsePayload.IDSeed > 0 {
		parseRt.SetIDSeed(parsePayload.IDSeed)
	}
	if len(parsePayload.Atoms) > 0 {
		if parseErr4 := parseRt.RestoreAtomSnapshot(parsePayload.Atoms); parseErr4 != nil {
			return SSRBootstrap{}, parseErr4
		}
	}
	parseRt.SetNextHydrationStrict(parseResolved.Strict)
	if parseErr5 := parseRt.HydrateInto(parseTarget, parseRoot); parseErr5 != nil {
		return SSRBootstrap{}, parseErr5
	}
	if parseHydrationBridgeErr := handleParallelRegionHydrationTarget(parseTarget); parseHydrationBridgeErr != nil {
		return SSRBootstrap{}, parseHydrationBridgeErr
	}
	return parsePayload, nil
}

// RenderToString renders a ui.Node tree to an HTML string for server-side rendering.
func RenderToString(parseRoot Node) (string, error) {
	return renderToStringObserved(parseRoot, SSRObservabilityOptions{})
}

// RenderToStringObserved renders a ui.Node tree to HTML and emits SSR metrics.
func RenderToStringObserved(parseRoot Node, parseOptions SSRObservabilityOptions) (string, error) {
	return renderToStringObserved(parseRoot, parseOptions)
}

// Text creates a text node from a string value.
func Text(parseContent string) Node {
	return runtime.Text(parseContent)
}

// UseState creates local component state.
func UseState[T any](parseInitialValue T) State[T] {
	get, set := runtime.GoUseStateGlobal(parseInitialValue)
	return State[T]{get: get, set: set}
}

// Get returns the current state value.
func (parseS State[T]) Get() T {
	return parseS.get()
}

// Set replaces the current state value.
func (parseS State[T]) Set(parseValue T) {
	parseS.set(parseValue)
}

// Update replaces the state value using the previous value.
func (parseS State[T]) Update(parseFn func(T) T) {
	parseS.set(parseFn)
}

// UseReducer creates reducer-driven local state.
func UseReducer[S any, A any](parseReducer func(S, A) S, parseInitialState S) Reducer[S, A] {
	parseState := UseState(parseInitialState)
	return Reducer[S, A]{
		get: func() S { return parseState.Get() },
		dispatch: func(parseAction A) {
			parseState.Update(func(parsePrev S) S {
				return parseReducer(parsePrev, parseAction)
			})
		},
	}
}

// Get is a core package helper.
func (parseR Reducer[S, A]) Get() S {
	if parseR.get == nil {
		var parseZero S
		return parseZero
	}
	return parseR.get()
}

// Dispatch is a core package helper.
func (parseR Reducer[S, A]) Dispatch(parseAction A) {
	if parseR.dispatch != nil {
		parseR.dispatch(parseAction)
	}
}

// UseEffect registers a side effect to run after commit when dependencies change.
func UseEffect(parseEffect func() func(), parseDeps ...interface{}) {
	runtime.GoUseEffectGlobal(parseEffect, parseDeps...)
}

// UseMemo memoizes a computed value until dependencies change.
func UseMemo[T any](parseCompute func() T, parseDeps ...interface{}) T {
	parseValue := runtime.GoUseMemoGlobalTyped(func() interface{} {
		return parseCompute()
	}, reflect.TypeOf((*T)(nil)).Elem(), parseDeps...)

	parseCast, parseOk := parseValue.(T)
	if parseOk {
		return parseCast
	}

	var parseZero T
	return parseZero
}

// UseCallback memoizes a callback until dependencies change.
func UseCallback[T any](parseFn T, parseDeps ...interface{}) T {
	parseValue := runtime.GoUseCallbackGlobal(parseFn, parseDeps...)
	parseCast, parseOk := parseValue.(T)
	if parseOk {
		return parseCast
	}

	var parseZero T
	return parseZero
}

// UseRef creates a stable mutable reference across renders.
func UseRef[T any](parseInitialValue T) Ref[T] {
	return Ref[T]{raw: runtime.GoUseRefGlobal(parseInitialValue)}
}

// UseContext reads the current value for a typed context.
func UseContext[T any](parseContext *Context[T]) T {
	if parseContext == nil || parseContext.descriptor == nil {
		panic(runtime.ActionableFrameworkPanic(runtime.ActionablePanicOptions{
			Source:  "ui",
			Subject: "ui.UseContext",
			Message: "ui.UseContext called with nil context descriptor",
			Path:    "ui.UseContext",
		}))
	}
	return castContextValue[T](runtime.GoUseContextValue(parseContext.descriptor))
}

// StartTransition schedules non-urgent updates in the transition lane.
func StartTransition(parseFn func()) {
	runtime.StartTransitionGlobal(parseFn)
}

// UseTransition returns transition pending state and a start helper.
func UseTransition() Transition {
	parsePending, _ := runtime.GoUseTransitionPendingGlobal()
	return Transition{
		pending: parsePending,
		start:   StartTransition,
	}
}

// Pending reports whether a transition is currently pending.
func (parseT Transition) Pending() bool {
	if parseT.pending == nil {
		return false
	}
	return parseT.pending()
}

// Start runs fn inside a transition.
func (parseT Transition) Start(parseFn func()) {
	if parseT.start != nil {
		parseT.start(parseFn)
	}
}

// Get returns the current ref value or the zero value for T.
func (parseR Ref[T]) Get() T {
	if parseR.raw == nil || parseR.raw.Current == nil {
		var parseZero T
		return parseZero
	}

	parseValue, parseOk := parseR.raw.Current.(T)
	if parseOk {
		return parseValue
	}

	var parseZero2 T
	return parseZero2
}

// Set updates the current ref value.
func (parseR Ref[T]) Set(parseValue T) {
	if parseR.raw != nil {
		parseR.raw.Current = parseValue
	}
}

// UsePrevious returns a handle for accessing the previous committed value.
// On the first render, Ok returns false and Get returns the zero value for T.
func UsePrevious[T any](parseValue T) Previous[T] {
	parseStateRef := UseRef(previousState[T]{})
	parseState := parseStateRef.Get()

	UseEffect(func() func() {
		parseStateRef.Set(previousState[T]{value: parseValue, ok: true})
		return nil
	}, parseValue)

	return Previous[T]{
		value: func() T { return parseState.value },
		ok:    func() bool { return parseState.ok },
	}
}

// Get returns the previous committed value or the zero value for T when unavailable.
func (parseP Previous[T]) Get() T {
	if parseP.value == nil {
		var parseZero T
		return parseZero
	}

	return parseP.value()
}

// Ok reports whether a previous committed value is available.
func (parseP Previous[T]) Ok() bool {
	if parseP.ok == nil {
		return false
	}

	return parseP.ok()
}

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
		if parseProps.ErrorFallback != nil {
			return parseProps.ErrorFallback(parseProps.Error)
		}
		if parseProps.Fallback != nil {
			return parseProps.Fallback
		}
		return nil
	}

	if !parseProps.Pending {
		return parseProps.Content
	}

	if parseState2.timedOut && parseProps.TimeoutFallback != nil {
		return parseProps.TimeoutFallback
	}

	if parseState2.fallbackVisible {
		return parseProps.Fallback
	}

	return parseProps.Content
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

// UseId returns a stable generated identifier for the current component instance.
func UseId() string {
	return runtime.GoUseIdGlobal()
}

// UseEvent wraps a Go function so it can be used as a stable event handler.
func UseEvent(parseFn interface{}) Handler {
	return Handler{value: runtime.GoUseFunc(parseFn)}
}

// WrapHandler wraps an already-prepared handler value.
func WrapHandler(parseValue interface{}) Handler {
	return Handler{value: parseValue}
}

// Value returns the wrapped handler payload.
func (parseH Handler) Value() interface{} {
	return parseH.value
}

// ensureInitialized is a core package helper.
func ensureInitialized() {
	if runtimeInitialized {
		return
	}

	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter:         jsdom.NewWASMDOMAdapter(),
		EventAdapter:       jsdom.NewWASMEventAdapter(),
		Scheduler:          jsdom.NewWASMScheduler(),
		BrowserState:       jsdom.NewWASMBrowserState(),
		HideRawPanicOutput: true,
	})
	runtimeInitialized = true
}

// renderComponent is a core package helper.
func renderComponent(parseComponent interface{}, parseRawProps map[string]interface{}) *runtime.Element {
	if parseComponent == nil {
		return nil
	}

	parseComponentValue := reflect.ValueOf(parseComponent)
	if !parseComponentValue.IsValid() || parseComponentValue.Kind() != reflect.Func {
		panic(actionableCreateElementPanic("ui.CreateElement requires a component function or ui.Node"))
	}

	parseMeta := getComponentMeta(parseComponentValue.Type())

	var parseResults []reflect.Value
	if parseMeta.hasArg {
		parseArg := parseMeta.zeroArg
		if parseProvided, parseOk := parseRawProps[propsKey]; parseOk {
			parseProvidedValue := reflect.ValueOf(parseProvided)
			if parseProvidedValue.IsValid() {
				switch {
				case parseProvidedValue.Type() == parseMeta.argType:
					parseArg = parseProvidedValue
				case parseProvidedValue.Type().AssignableTo(parseMeta.argType):
					parseArg = parseProvidedValue
				case parseProvidedValue.Type().ConvertibleTo(parseMeta.argType):
					parseArg = parseProvidedValue.Convert(parseMeta.argType)
				}
			}
		}

		var parseArgBuf [1]reflect.Value
		parseArgBuf[0] = parseArg
		parseResults = parseComponentValue.Call(parseArgBuf[:])
	} else {
		parseResults = parseComponentValue.Call(nil)
	}

	if len(parseResults) == 0 || !parseResults[0].IsValid() || parseResults[0].IsNil() {
		return nil
	}

	parseElement, _ := parseResults[0].Interface().(*runtime.Element)
	return parseElement
}

// getComponentMeta is a core package helper.
func getComponentMeta(parseComponentType reflect.Type) componentMeta {
	if parseCached, parseOk := componentMetaCache.Load(parseComponentType); parseOk {
		return parseCached.(componentMeta)
	}

	if parseComponentType.NumIn() > 1 {
		panic(actionableCreateElementPanic("ui.CreateElement components may accept at most one props argument"))
	}
	if parseComponentType.NumOut() != 1 {
		panic(actionableCreateElementPanic("ui.CreateElement components must return ui.Node"))
	}

	parseMeta := componentMeta{}
	if parseComponentType.NumIn() == 1 {
		parseMeta.hasArg = true
		parseMeta.argType = parseComponentType.In(0)
		parseMeta.zeroArg = reflect.Zero(parseMeta.argType)
	}

	parseStored, _ := componentMetaCache.LoadOrStore(parseComponentType, parseMeta)
	return parseStored.(componentMeta)
}

// toInterfaces is a core package helper.
func toInterfaces(parseChildren []Node) []interface{} {
	if len(parseChildren) == 0 {
		return nil
	}

	parseValues := make([]interface{}, 0, len(parseChildren))
	for _, parseChild := range parseChildren {
		parseValues = append(parseValues, parseChild)
	}

	return parseValues
}
