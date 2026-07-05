//go:build !js || !wasm

package ui

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

const propsKey = "__ui_props"

type componentMeta struct {
	hasArg      bool
	argType     reflect.Type
	zeroArg     reflect.Value
	getArgValue func(map[string]any) reflect.Value
}

var componentMetaCache sync.Map
var nativeIDMu sync.Mutex
var nativeIDCounter int

// Node is the public UI tree node type.
type Element = runtime.Element
type Node = *runtime.Element

type Event struct{}
type MouseEvent = Event
type InputEvent = Event
type ChangeEvent = Event
type KeyboardEvent = Event
type FocusEvent = Event
type FormEvent = Event

// GetValue is a core package helper.
func (Event) GetValue() string {
	return ""
}

// IsChecked is a core package helper.
func (Event) IsChecked() bool {
	return false
}

// GetKeyCode is a core package helper.
func (Event) GetKeyCode() int {
	return 0
}

// GetKey is a core package helper.
func (Event) GetKey() string {
	return ""
}

// PreventDefault is a core package helper.
func (Event) PreventDefault() {}

// StopPropagation is a core package helper.
func (Event) StopPropagation() {}

// Transition exposes transition-pending state and a transition starter.
type Transition struct {
	pending func() bool
	start   func(func())
}

// State provides access to hook-managed local state.
type State[T any] struct {
	get func() T
	set func(any)
}

// Ref stores a stable mutable reference across renders.
type Ref[T any] struct {
	current *T
}

// Reducer provides access to reducer-style local state transitions.
type Reducer[S any, A any] struct {
	get      func() S
	dispatch func(A)
}

// Previous exposes the previous committed value for a hook call.
type Previous[T any] struct {
	value func() T
	ok    func() bool
}

// Debounced returns a delayed view of a value on browser builds; on native builds it is immediate.
type Debounced[T any] struct {
	get     func() T
	pending func() bool
}

// Throttled returns a trailing-throttled view of a value on browser builds; on native builds it is immediate.
type Throttled[T any] struct {
	get     func() T
	pending func() bool
}

// Handler stores an event handler value in a form the runtime can consume.
type Handler struct {
	value any
}

// ReactiveSource identifies shared values that a ReactiveRegion can subscribe to explicitly.
type ReactiveSource interface {
	ReactiveRegionSourceIDs() []string
}

// PortalTarget describes where a portal subtree should render.
type PortalTarget struct {
	Selector string
	Node     any
}

// PortalProps configures a portal target and its children.
type PortalProps struct {
	Target   PortalTarget
	Child    Node
	Children []Node
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
	ResetKeys     []any
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
	Dependencies    []any
	Fallback        Node
	TimeoutFallback Node
	ErrorFallback   func(error) Node
	Delay           time.Duration
	Timeout         time.Duration
}

// CreateElement creates a UI node from a component function, provider, boundary, or existing node.
func CreateElement(parseComponent any, parseProps ...any) Node {
	if parseNode, parseOk := parseComponent.(*runtime.Element); parseOk && len(parseProps) == 0 {
		return parseNode
	}
	if parseBoundary, parseOk2 := parseComponent.(runtimeErrorBoundaryComponent); parseOk2 {
		var parseRawProps any
		if len(parseProps) > 0 {
			parseRawProps = parseProps[0]
		}
		return createErrorBoundaryElement(parseBoundary, parseRawProps)
	}
	if parseProvider, parseOk3 := parseComponent.(contextProviderComponent); parseOk3 {
		var parseRawProps2 any
		if len(parseProps) > 0 {
			parseRawProps2 = parseProps[0]
		}
		return createContextProviderElement(parseProvider, parseRawProps2)
	}
	var parseRawProps3 map[string]any
	if len(parseProps) > 0 {
		parseRawProps3 = map[string]any{}
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

// ReactiveRegion creates an explicit fine-grained subscribed region.
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
	return runtime.CreateElement(runtime.ReactiveRegionNodeType, map[string]any{
		"__gwc_reactive_region_source_ids": parseIds,
		"__gwc_reactive_region_render": func() *runtime.Element {
			if render == nil {
				return nil
			}
			return render()
		},
	})
}

// Portal renders children inline on non-browser targets.
func Portal(parseProps PortalProps) Node {
	parseChildren := make([]Node, 0, len(parseProps.Children)+1)
	if parseProps.Child != nil {
		parseChildren = append(parseChildren, parseProps.Child)
	}
	parseChildren = append(parseChildren, parseProps.Children...)
	return Fragment(parseChildren...)
}

// Text creates a text node.
func Text(parseContent string) Node {
	return &runtime.Element{
		Type:        "TEXT_ELEMENT",
		TextContent: parseContent,
		Children:    []any{},
	}
}

// RenderToString renders a ui.Node tree to HTML on non-browser targets.
func RenderToString(parseRoot Node) (string, error) {
	return renderToStringObserved(parseRoot, SSRObservabilityOptions{})
}

// RenderToStringObserved renders a ui.Node tree to HTML and emits SSR metrics.
func RenderToStringObserved(parseRoot Node, parseOptions SSRObservabilityOptions) (string, error) {
	return renderToStringObserved(parseRoot, parseOptions)
}

// Render is browser-only; the native SSR slice exposes RenderToString instead.
func Render(parseRoot Node, parseSelector string) {
	panic(actionableUnsupportedOnServerPanic("Render"))
}

// Run is browser-only; on the native/SSR slice it panics like Render. Server-side
// code should call RenderToString (or the SSR streaming APIs) instead of mounting
// and blocking.
func Run(parseSelector string, parseComponent any, parseProps ...any) {
	_ = parseSelector
	_ = parseComponent
	_ = parseProps
	panic(actionableUnsupportedOnServerPanic("Run"))
}

// RenderInto is a non-browser stub that returns an UnsupportedOnServer error.
func RenderInto(parseRoot Node, parseTarget any) error {
	_ = parseRoot
	_ = parseTarget
	return UnsupportedOnServer("RenderInto")
}

// Hydrate is a non-browser stub that returns an UnsupportedOnServer error.
func Hydrate(parseRoot Node, parseSelector string, parseOptions ...HydrationOptions) (SSRBootstrap, error) {
	return SSRBootstrap{}, UnsupportedOnServer("Hydrate")
}

// HydrateInto is a non-browser stub that returns an UnsupportedOnServer error.
func HydrateInto(parseRoot Node, parseTarget any, parseOptions ...HydrationOptions) (SSRBootstrap, error) {
	_ = parseRoot
	_ = parseTarget
	_ = parseOptions
	return SSRBootstrap{}, UnsupportedOnServer("HydrateInto")
}

// StartTransition runs fn immediately on non-browser targets.
func StartTransition(parseFn func()) {
	if parseFn != nil {
		parseFn()
	}
}

// UseTransition returns a no-op transition helper on non-browser targets.
func UseTransition() Transition {
	return Transition{
		pending: func() bool { return false },
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

// UseState creates local component state on non-browser targets.
func UseState[T any](parseInitialValue T) State[T] {
	parseCurrent := parseInitialValue
	return State[T]{
		get: func() T { return parseCurrent },
		set: func(parseNext any) {
			if parseValue, parseOk := parseNext.(T); parseOk {
				parseCurrent = parseValue
				return
			}
			if parseUpdater, parseOk2 := parseNext.(func(T) T); parseOk2 {
				parseCurrent = parseUpdater(parseCurrent)
			}
		},
	}
}

// Get returns the current state value.
func (parseS State[T]) Get() T {
	if parseS.get == nil {
		var parseZero T
		return parseZero
	}
	return parseS.get()
}

// Set replaces the current state value.
func (parseS State[T]) Set(parseValue T) {
	if parseS.set != nil {
		parseS.set(parseValue)
	}
}

// Update replaces the state value using the previous value.
func (parseS State[T]) Update(parseFn func(T) T) {
	if parseS.set != nil && parseFn != nil {
		parseS.set(parseFn)
	}
}

// UseRef creates a stable mutable reference across renders.
func UseRef[T any](parseInitialValue T) Ref[T] {
	return Ref[T]{current: &parseInitialValue}
}

// Get returns the current ref value.
func (parseR Ref[T]) Get() T {
	if parseR.current == nil {
		var parseZero T
		return parseZero
	}
	return *parseR.current
}

// Set updates the current ref value.
func (parseR Ref[T]) Set(parseValue T) {
	if parseR.current != nil {
		*parseR.current = parseValue
	}
}

// UseEffect is a no-op on non-browser targets.
func UseEffect(parseEffect func() func(), parseDeps ...any) {}

// UseMemoOf computes directly on non-browser targets (native UseMemo parity).
func UseMemoOf[T any, D comparable](parseCompute func(D) T, parseDep D) T {
	if parseCompute == nil {
		var parseZero T
		return parseZero
	}
	return parseCompute(parseDep)
}

// UseEffectOf is a no-op on non-browser targets (native UseEffect parity).
func UseEffectOf[D comparable](parseEffect func() func(), parseDep D) {}

// UseLayoutEffect is a no-op on non-browser targets (G36).
func UseLayoutEffect(parseEffect func() func(), parseDeps ...any) {}

// UseId returns a stable generated identifier for the current component instance.
func UseId() string {
	nativeIDMu.Lock()
	defer nativeIDMu.Unlock()
	nativeIDCounter++
	return fmt.Sprintf("gwc-native:%d", nativeIDCounter)
}

// UseMemo computes value immediately on non-browser targets.
func UseMemo[T any](parseCompute func() T, parseDeps ...any) T {
	if parseCompute == nil {
		var parseZero T
		return parseZero
	}
	return parseCompute()
}

// UseCallback returns fn unchanged on non-browser targets.
func UseCallback[T any](parseFn T, parseDeps ...any) T {
	return parseFn
}

// UseReducer returns a lightweight reducer-backed handle on non-browser targets.
func UseReducer[S any, A any](parseReducer func(S, A) S, parseInitialState S) Reducer[S, A] {
	parseCurrent := parseInitialState
	return Reducer[S, A]{
		get: func() S { return parseCurrent },
		dispatch: func(parseAction A) {
			if parseReducer != nil {
				parseCurrent = parseReducer(parseCurrent, parseAction)
			}
		},
	}
}

// Get returns the current reducer state.
func (parseR Reducer[S, A]) Get() S {
	if parseR.get == nil {
		var parseZero S
		return parseZero
	}
	return parseR.get()
}

// Dispatch applies an action to the reducer state.
func (parseR Reducer[S, A]) Dispatch(parseAction A) {
	if parseR.dispatch != nil {
		parseR.dispatch(parseAction)
	}
}

// UsePrevious reports no previous value on non-browser targets.
func UsePrevious[T any](parseValue T) Previous[T] {
	return Previous[T]{
		value: func() T {
			var parseZero T
			return parseZero
		},
		ok: func() bool { return false },
	}
}

// Get returns the previous committed value or the zero value when unavailable.
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

// UseDebounced returns the current value unchanged on non-browser targets.
func UseDebounced[T any](parseValue T, parseDelay time.Duration) Debounced[T] {
	return Debounced[T]{
		get:     func() T { return parseValue },
		pending: func() bool { return false },
	}
}

// Get returns the current debounced value.
func (parseD Debounced[T]) Get() T {
	if parseD.get == nil {
		var parseZero T
		return parseZero
	}
	return parseD.get()
}

// Pending reports whether a debounced update is pending.
func (parseD Debounced[T]) Pending() bool {
	if parseD.pending == nil {
		return false
	}
	return parseD.pending()
}

// UseThrottled returns the current value unchanged on non-browser targets.
func UseThrottled[T any](parseValue T, parseInterval time.Duration) Throttled[T] {
	return Throttled[T]{
		get:     func() T { return parseValue },
		pending: func() bool { return false },
	}
}

// Get returns the current throttled value.
func (parseT Throttled[T]) Get() T {
	if parseT.get == nil {
		var parseZero T
		return parseZero
	}
	return parseT.get()
}

// Pending reports whether a throttled update is pending.
func (parseT Throttled[T]) Pending() bool {
	if parseT.pending == nil {
		return false
	}
	return parseT.pending()
}

// UseDeferredValue returns value unchanged on non-browser targets.
func UseDeferredValue[T any](parseValue T) T {
	return parseValue
}

// UseContext is a core package helper.
func UseContext[T any](parseContext *Context[T]) T {
	panic(actionableUnsupportedOnServerPanic("UseContext"))
}

// AsyncBoundary renders children resolving async loading, error, and pending states on the server.
func AsyncBoundary(parseProps AsyncBoundaryProps) Node {
	parseFallback := parseProps.Fallback
	if parseProps.Pending && parseProps.Timeout > 0 && parseProps.TimeoutFallback != nil {
		parseFallback = parseProps.TimeoutFallback
	}
	return createAsyncBoundaryElement(parseProps, parseProps.Pending, parseFallback)
}

// UseLazyNode executes the loader synchronously on the server and returns the result as a LazyNode.
func UseLazyNode(parseLoader func(context.Context) (Node, error), parseDeps ...any) LazyNode {
	parseState := LazyNodeState{}
	if parseLoader == nil {
		parseState.Error = UnsupportedOnServer("UseLazyNode")
	} else {
		parseNode, parseErr := parseLoader(context.Background())
		parseState.Node = parseNode
		parseState.Error = parseErr
		parseState.Ready = parseErr == nil && parseNode != nil
	}

	return LazyNode{get: func() LazyNodeState { return parseState }}
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

// Lazy renders a lazy-loaded node, delegating to AsyncBoundary for fallback and error states.
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

// UseEvent wraps a Go function so it can be used as a stable event handler.
func UseEvent(parseFn any) Handler {
	return Handler{value: parseFn}
}

// WrapHandler wraps an already-prepared handler value.
func WrapHandler(parseValue any) Handler {
	return Handler{value: parseValue}
}

// Value returns the wrapped handler payload.
func (parseH Handler) Value() any {
	return parseH.value
}

// renderComponent is a core package helper.
func renderComponent(parseComponent any, parseRawProps map[string]any) *runtime.Element {
	if parseComponent == nil {
		return nil
	}

	switch parseTypedComponent := parseComponent.(type) {
	case func() Node:
		return parseTypedComponent()
	case func(map[string]any) Node:
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
func buildComponentArgValueLoader(parseArgType reflect.Type, parseZeroArg reflect.Value) func(map[string]any) reflect.Value {
	return func(parseRawProps map[string]any) reflect.Value {
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
func getComponentMapProps(parseRawProps map[string]any) map[string]any {
	if parseRawProps == nil {
		return nil
	}
	parseProvidedProps, parseOk := parseRawProps[propsKey]
	if !parseOk || parseProvidedProps == nil {
		return nil
	}
	switch parseTypedProps := parseProvidedProps.(type) {
	case map[string]any:
		return parseTypedProps
	case runtime.Attrs:
		return map[string]any(parseTypedProps)
	default:
		return nil
	}
}

// getComponentAttrsProps resolves one component props payload as runtime.Attrs for direct-call fast paths.
func getComponentAttrsProps(parseRawProps map[string]any) runtime.Attrs {
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
	case map[string]any:
		return runtime.Attrs(parseTypedProps)
	default:
		return nil
	}
}

// toInterfaces is a core package helper.
func toInterfaces(parseChildren []Node) []any {
	if len(parseChildren) == 0 {
		return nil
	}

	parseValues := make([]any, 0, len(parseChildren))
	for _, parseChild := range parseChildren {
		parseValues = append(parseValues, parseChild)
	}

	return parseValues
}

// UnsupportedOnServer explains the current SSR limitation for hook-based components.
func UnsupportedOnServer(parseName string) error {
	return fmt.Errorf("%s", unsupportedOnServerMessage(parseName))
}

// ReadBootstrapScript is a non-browser stub that returns an UnsupportedOnServer error.
func ReadBootstrapScript(parseScriptID string) (SSRBootstrap, error) {
	return SSRBootstrap{}, UnsupportedOnServer("ReadBootstrapScript")
}

// ReadBootstrapReferenceScript is a non-browser stub that returns an UnsupportedOnServer error.
func ReadBootstrapReferenceScript(parseScriptID string) (SSRBootstrapReference, error) {
	return SSRBootstrapReference{}, UnsupportedOnServer("ReadBootstrapReferenceScript")
}

// ReadBootstrapReference is a non-browser stub that returns an UnsupportedOnServer error.
func ReadBootstrapReference(parseRef SSRBootstrapReference) (SSRBootstrap, error) {
	return SSRBootstrap{}, UnsupportedOnServer("ReadBootstrapReference")
}
