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
	hasArg      bool
	argType     reflect.Type
	zeroArg     reflect.Value
	getArgValue func(map[string]interface{}) reflect.Value
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

	return runtime.CreateElementOwned(getComponentHandle(parseComponent), parseRawProps3)
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
	setParallelRegionHydrationObserver(parseRt, parseResolved.Observability.CorrelationID, func() error {
		return handleParallelRegionHydrationSelector(parseSelector)
	}, func(parseMetrics runtime.HydrationMetrics) {
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
	setParallelRegionHydrationObserver(parseRt, parseResolved.Observability.CorrelationID, func() error {
		return handleParallelRegionHydrationTarget(parseTarget)
	}, func(parseMetrics runtime.HydrationMetrics) {
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

	switch parseTypedComponent := parseComponent.(type) {
	case func() Node:
		return parseTypedComponent()
	case func(map[string]interface{}) Node:
		return parseTypedComponent(getComponentMapProps(parseRawProps))
	case func(runtime.Attrs) Node:
		return parseTypedComponent(getComponentAttrsProps(parseRawProps))
	}

	parseComponentValue := reflect.ValueOf(parseComponent)
	if !parseComponentValue.IsValid() || parseComponentValue.Kind() != reflect.Func {
		panic(actionableCreateElementPanic("ui.CreateElement requires a component function or ui.Node"))
	}

	parseMeta := getComponentMeta(parseComponentValue.Type())

	var parseResults []reflect.Value
	if parseMeta.hasArg {
		var parseArgBuf [1]reflect.Value
		parseArgBuf[0] = parseMeta.getArgValue(parseRawProps)
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
		parseMeta.getArgValue = buildComponentArgValueLoader(parseMeta.argType, parseMeta.zeroArg)
	}

	parseStored, _ := componentMetaCache.LoadOrStore(parseComponentType, parseMeta)
	return parseStored.(componentMeta)
}

// buildComponentArgValueLoader builds one cached props-to-argument resolver for one typed component signature.
func buildComponentArgValueLoader(parseArgType reflect.Type, parseZeroArg reflect.Value) func(map[string]interface{}) reflect.Value {
	return func(parseRawProps map[string]interface{}) reflect.Value {
		if parseRawProps == nil {
			return parseZeroArg
		}
		parseProvided, parseOk := parseRawProps[propsKey]
		if !parseOk {
			return parseZeroArg
		}
		parseProvidedValue := reflect.ValueOf(parseProvided)
		if !parseProvidedValue.IsValid() {
			return parseZeroArg
		}
		switch {
		case parseProvidedValue.Type() == parseArgType:
			return parseProvidedValue
		case parseProvidedValue.Type().AssignableTo(parseArgType):
			return parseProvidedValue
		case parseProvidedValue.Type().ConvertibleTo(parseArgType):
			return parseProvidedValue.Convert(parseArgType)
		default:
			return parseZeroArg
		}
	}
}

// getComponentMapProps resolves one component props payload as a plain map for direct-call fast paths.
func getComponentMapProps(parseRawProps map[string]interface{}) map[string]interface{} {
	if parseRawProps == nil {
		return nil
	}
	parseProvidedProps, parseOk := parseRawProps[propsKey]
	if !parseOk || parseProvidedProps == nil {
		return nil
	}
	switch parseTypedProps := parseProvidedProps.(type) {
	case map[string]interface{}:
		return parseTypedProps
	case runtime.Attrs:
		return map[string]interface{}(parseTypedProps)
	default:
		return nil
	}
}

// getComponentAttrsProps resolves one component props payload as runtime.Attrs for direct-call fast paths.
func getComponentAttrsProps(parseRawProps map[string]interface{}) runtime.Attrs {
	if parseRawProps == nil {
		return nil
	}
	parseProvidedProps, parseOk := parseRawProps[propsKey]
	if !parseOk || parseProvidedProps == nil {
		return nil
	}
	switch parseTypedProps := parseProvidedProps.(type) {
	case runtime.Attrs:
		return parseTypedProps
	case map[string]interface{}:
		return runtime.Attrs(parseTypedProps)
	default:
		return nil
	}
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
