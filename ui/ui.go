//go:build js && wasm

package ui

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/jsdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/pluginruntime"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/interop"
)

const propsKey = "__ui_props"

var runtimeInitialized bool

// SchedulingOptions selects the v5 scheduling behaviors.
//
// FrameBudgetMs and LaneQueues are ON by default. Both pass their acceptance
// test AND leave the Example 201 browser benchmark green, which is the gate a
// scheduling default has to clear — the native suite cannot see the scenarios
// that a slicing change breaks.
//
// PassiveEffectsAfterPaint and AsyncIngress stay off: each changes observable
// semantics (effect ordering, and when an async write lands), so they are
// migrations rather than defaults. See docs/plans/v5-plan.md.
type SchedulingOptions struct {
	// PassiveEffectsAfterPaint keeps layout effects synchronous inside the
	// commit task and defers passive effects past the paint boundary, so a slow
	// UseEffect can no longer hold the frame.
	//
	// Changes effect ORDERING, not just timing: every layout effect in the tree
	// then precedes every passive effect.
	PassiveEffectsAfterPaint bool
	// FrameBudgetMs gives the work loop a real wall-clock slice budget instead
	// of a fixed fiber count.
	//
	// ON by default: zero takes the 5ms default, a positive value sets an
	// explicit budget, and NEGATIVE opts out to count-only slicing.
	FrameBudgetMs float64
	// LaneQueues defers work marked at a lower priority than the running pass to
	// a follow-up pass, with a per-lane deadline so deferral cannot starve.
	//
	// ON by default; set DisableLaneQueues to opt out.
	LaneQueues bool
	// DisableLaneQueues turns per-lane deferral back off.
	DisableLaneQueues bool
	// AsyncIngress routes state writes made OUTSIDE the frame loop — from a
	// worker reply, a gRPC callback, or any goroutine — through the async inbox,
	// so they are applied at one defined point per frame instead of at whatever
	// moment they happen to arrive.
	//
	// This is the flag that makes existing async code safe without rewriting it.
	// Without it, isolation depends on every such call site remembering to use
	// ui.PostAsync, and a call site that forgets gets no warning.
	//
	// It changes WHEN an async write lands: one task later, at the drain. Writes
	// posted together are applied in one block, so several results produce one
	// render rather than several.
	AsyncIngress bool
}

var schedulingOptions SchedulingOptions

// ConfigureScheduling selects v5 scheduling behaviors. Call it BEFORE the first
// Render or Hydrate; afterwards the runtime is already built and this is a
// no-op, which it reports rather than failing silently.
//
// Without this the flags on runtime.Config are unreachable from an application,
// so R2's "flip the flag once its acceptance test passes" could not be carried
// out by the people the flags are for.
func ConfigureScheduling(parseOptions SchedulingOptions) {
	if runtimeInitialized {
		runtime.ReportDiagnostic("ui", runtime.DiagnosticWarning,
			"ConfigureScheduling was called after the runtime was initialized; call it before the first Render or Hydrate")
		return
	}
	schedulingOptions = parseOptions
}

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

// NewErrorBoundary is the function entry point for an error boundary, mirroring AsyncBoundary(props)
// and Lazy(props): NewErrorBoundary(ErrorBoundaryProps{...}) instead of CreateElement(ErrorBoundary,
// props). (The ErrorBoundary identifier is a component var, so the func needs a distinct name.)
func NewErrorBoundary(parseProps ErrorBoundaryProps) Node {
	return CreateElement(ErrorBoundary, parseProps)
}

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
	return runtime.CreateElementOwned(runtime.ReactiveRegionNodeType, map[string]interface{}{
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

	return runtime.CreateElementOwned(runtime.PortalNodeType, parseRawProps, parseChildren...)
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

// runKeepAlive is the post-mount blocking call used by Run. It defaults to
// interop.KeepAlive — the same keep-alive primitive behind utils.WaitForever — and
// is a package variable only so tests can substitute a non-blocking stub.
// Production builds always block here.
var runKeepAlive = interop.KeepAlive

// Run is the one-line entrypoint for a browser app. It builds the component with
// CreateElement, mounts it into the element matched by selector via Render, then
// blocks forever (the same keep-alive as utils.WaitForever) so the js/wasm program
// stays alive to handle events. Run never returns.
//
// Pass the component plus any props CreateElement accepts; props are optional for
// no-prop components:
//
//	func main() { ui.Run("#app", renderHelloApp) }
//
// Run is a convenience over the primitives, not a replacement for them. For SSR,
// native, or tests — or when main must do work after mount — call Render (or
// RenderInto) and block with utils.WaitForever yourself.
func Run(parseSelector string, parseComponent interface{}, parseProps ...interface{}) {
	Render(CreateElement(parseComponent, parseProps...), parseSelector)
	runKeepAlive()
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
// hydrateWithBootstrap is the shared body of Hydrate and HydrateInto: it resolves the bootstrap
// payload (options/script/reference), records startup cost, wires the hydration observer, restores
// the ID seed + atom snapshot, sets strict mode, then mounts. The ONLY per-entrypoint difference is
// the mount call, supplied as parseMount, so the bootstrap/observability logic lives in exactly one
// place.
func hydrateWithBootstrap(parseOptions []HydrationOptions, parseMount func(parseRt *runtime.Runtime) error) (SSRBootstrap, error) {
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
	parseRt.SetNextHydrationObserver(parseResolved.Observability.CorrelationID, func(parseMetrics runtime.HydrationMetrics) {
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
	if parseMountErr := parseMount(parseRt); parseMountErr != nil {
		return SSRBootstrap{}, parseMountErr
	}
	return parsePayload, nil
}

func Hydrate(parseRoot Node, parseSelector string, parseOptions ...HydrationOptions) (SSRBootstrap, error) {
	return hydrateWithBootstrap(parseOptions,
		func(parseRt *runtime.Runtime) error { parseRt.HydrateTo(parseSelector, parseRoot); return nil },
	)
}

// HydrateInto resumes a UI tree into an explicit DOM node.
func HydrateInto(parseRoot Node, parseTarget interface{}, parseOptions ...HydrationOptions) (SSRBootstrap, error) {
	return hydrateWithBootstrap(parseOptions,
		func(parseRt *runtime.Runtime) error { return parseRt.HydrateInto(parseTarget, parseRoot) },
	)
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
// Dispatched actions are applied via UseState's functional updater, which
// serializes reducer invocations through the runtime scheduler — each dispatch
// receives the latest committed state as its previous-state argument.
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

// UseMemoOf memoizes one computation keyed by a single comparable dependency
// with zero steady-state allocations: pass a static (non-capturing) compute
// function that derives the value from the dependency. Uses the same memo
// slot as UseMemo; do not alternate between the two across renders.
func UseMemoOf[T any, D comparable](parseCompute func(D) T, parseDep D) T {
	return runtime.GoUseMemoOf(parseCompute, parseDep)
}

// UseEffectOf registers an effect keyed by a single comparable dependency
// without the variadic deps allocation. Same slot as UseEffect; do not
// alternate between the two across renders.
func UseEffectOf[D comparable](parseEffect func() func(), parseDep D) {
	runtime.GoUseEffectOf(parseEffect, parseDep)
}

// UseLayoutEffect registers a layout effect (G36): it runs synchronously after
// the commit mutates the DOM, before the browser paints, and before this
// component's passive UseEffect callbacks. Use it for post-render DOM work that
// must complete before paint — focusing a just-mounted input, measuring an
// element (offsetWidth/getBoundingClientRect), or scrolling — instead of guessing
// a setTimeout/requestAnimationFrame delay. Same (effect, deps...) contract as
// UseEffect; return a cleanup func or nil.
func UseLayoutEffect(parseEffect func() func(), parseDeps ...interface{}) {
	runtime.GoUseLayoutEffectGlobal(parseEffect, parseDeps...)
}

// UseMemo memoizes a computed value until dependencies change.
func UseMemo[T any](parseCompute func() T, parseDeps ...interface{}) T {
	// Typed pass-through: no func()->any adapter closure per call and no
	// per-call reflect (the hot-reload coercion target comes from T inside
	// the runtime, on the restore path only).
	return runtime.GoUseMemoFor(parseCompute, parseDeps...)
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

// PostAsync hands work to the render thread from outside it.
//
// This is the supported way for anything that is not already on the frame loop
// — a gRPC callback, a Web Worker reply, a goroutine, a channel receive — to
// change state that the UI renders from. Call it with a function that performs
// the mutation; that function runs later, at one defined point in the frame,
// with the in-flight tree guaranteed not to be mid-render.
//
// Two properties follow, and neither is achievable from the call site alone:
//
//   - Isolation. Nothing outside the frame loop touches hook state, so a render
//     in flight cannot be torn by a reply that happens to arrive during it. The
//     call site cannot arrange this itself, because it has no way to know
//     whether a render is running.
//   - Batching. Everything posted between two drains is applied in one
//     synchronous block, so twenty worker messages in a frame produce one render
//     pass rather than twenty.
//
// Safe to call from any goroutine. Work is queued, never dropped: if the queue
// outgrows its bound the runtime drains early and reports that batching
// degraded, rather than discarding anything.
//
//	go func() {
//	    parseRows := fetchFromWorker()
//	    ui.PostAsync(func() { setRows(parseRows) })
//	}()
//
// With Config.AsyncIngress on, state setters called off the frame loop are
// routed through this automatically, so existing async code becomes safe without
// being rewritten. Calling PostAsync explicitly is still worthwhile when several
// writes belong together — one post means one render for the whole group.
func PostAsync(parseFn func()) {
	runtime.PostAsyncGlobal(parseFn)
}

// AsyncIngressEnabled reports whether off-loop state writes are routed through
// the async inbox automatically.
//
// Worth checking in library code that must work under both settings: with it
// off, an off-loop setter still applies where it is called, so such code should
// post explicitly rather than assume the runtime will.
func AsyncIngressEnabled() bool {
	return runtime.AsyncIngressEnabledGlobal()
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
	if parseValue != nil {
		parseValueType := reflect.TypeOf(parseValue)
		if parseValueType != nil && parseValueType.Kind() == reflect.Func {
			if parseWrappedValue, isWrapped := runtime.BuildDOMWrappedFunctionIfReadyGlobal(parseValue); isWrapped {
				return Handler{value: parseWrappedValue}
			}
		}
	}
	return Handler{value: parseValue}
}

// Value returns the wrapped handler payload.
func (parseH Handler) Value() interface{} {
	return parseH.value
}

// ensureInitialized is a core package helper.
func ensureInitialized() {
	if runtimeInitialized {
		ensureUIEventListeners()
		return
	}

	applyInteractiveGCPacing()
	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter:               jsdom.NewWASMDOMAdapter(),
		EventAdapter:             jsdom.NewWASMEventAdapter(),
		Scheduler:                jsdom.NewWASMScheduler(),
		BrowserState:             jsdom.NewWASMBrowserState(),
		HideRawPanicOutput:       true,
		PassiveEffectsAfterPaint: schedulingOptions.PassiveEffectsAfterPaint,
		FrameBudgetMs:            schedulingOptions.FrameBudgetMs,
		LaneQueues:               schedulingOptions.LaneQueues,
		DisableLaneQueues:        schedulingOptions.DisableLaneQueues,
		AsyncIngress:             schedulingOptions.AsyncIngress,
	})
	if _, parseErr := pluginruntime.BootGlobalKernel(pluginruntime.BootstrapOptions{}); parseErr != nil {
		runtime.ReportDiagnostic("pluginruntime", runtime.DiagnosticWarning, "plugin kernel bootstrap failed: "+parseErr.Error())
	}
	ensureUIEventListeners()
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
