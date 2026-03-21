//go:build !js || !wasm
// +build !js !wasm

package ui

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

const propsKey = "__ui_props"

type componentMeta struct {
	hasArg  bool
	argType reflect.Type
	zeroArg reflect.Value
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

func (Event) GetValue() string {
	return ""
}

func (Event) IsChecked() bool {
	return false
}

func (Event) GetKeyCode() int {
	return 0
}

func (Event) GetKey() string {
	return ""
}

func (Event) PreventDefault() {}

func (Event) StopPropagation() {}

// Transition exposes transition-pending state and a transition starter.
type Transition struct {
	pending func() bool
	start   func(func())
}

// State provides access to hook-managed local state.
type State[T any] struct {
	get func() T
	set func(interface{})
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

type errorBoundaryComponent struct {
	boundaryType *runtime.ErrorBoundaryType
}

type runtimeErrorBoundaryComponent interface {
	runtimeErrorBoundary() *runtime.ErrorBoundaryType
}

// ErrorBoundary creates a subtree boundary with fallback rendering and reset behavior.
var ErrorBoundary = &errorBoundaryComponent{boundaryType: runtime.NewErrorBoundaryType()}

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

// CreateElement creates a UI node from a component function, provider, boundary, or existing node.
func CreateElement(component interface{}, props ...interface{}) Node {
	if node, ok := component.(*runtime.Element); ok && len(props) == 0 {
		return node
	}
	if boundary, ok := component.(runtimeErrorBoundaryComponent); ok {
		var rawProps interface{}
		if len(props) > 0 {
			rawProps = props[0]
		}
		return createErrorBoundaryElement(boundary, rawProps)
	}
	if provider, ok := component.(contextProviderComponent); ok {
		var rawProps interface{}
		if len(props) > 0 {
			rawProps = props[0]
		}
		return createContextProviderElement(provider, rawProps)
	}
	var rawProps map[string]interface{}
	if len(props) > 0 {
		rawProps = map[string]interface{}{}
		rawProps[propsKey] = props[0]
	}

	return runtime.CreateElement(getComponentHandle(component), rawProps)
}

func (boundary *errorBoundaryComponent) runtimeErrorBoundary() *runtime.ErrorBoundaryType {
	if boundary == nil {
		return nil
	}
	return boundary.boundaryType
}

// Fragment groups children without introducing an extra host element.
func Fragment(children ...Node) Node {
	return runtime.CreateElement("FRAGMENT", nil, toInterfaces(children)...)
}

// ReactiveRegion creates an explicit fine-grained subscribed region.
func ReactiveRegion(render func() Node, sources ...ReactiveSource) Node {
	ids := make([]string, 0, len(sources))
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}
		for _, id := range source.ReactiveRegionSourceIDs() {
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return runtime.CreateElement(runtime.ReactiveRegionNodeType, map[string]interface{}{
		"__gwc_reactive_region_source_ids": ids,
		"__gwc_reactive_region_render": func() *runtime.Element {
			if render == nil {
				return nil
			}
			return render()
		},
	})
}

// Portal renders children inline on non-browser targets.
func Portal(props PortalProps) Node {
	children := make([]Node, 0, len(props.Children)+1)
	if props.Child != nil {
		children = append(children, props.Child)
	}
	children = append(children, props.Children...)
	return Fragment(children...)
}

// Text creates a text node.
func Text(content string) Node {
	return &runtime.Element{
		Type:        "TEXT_ELEMENT",
		TextContent: content,
		Children:    []interface{}{},
	}
}

// RenderToString renders a ui.Node tree to HTML on non-browser targets.
func RenderToString(root Node) (string, error) {
	return renderToStringObserved(root, SSRObservabilityOptions{})
}

// RenderToStringObserved renders a ui.Node tree to HTML and emits SSR metrics.
func RenderToStringObserved(root Node, options SSRObservabilityOptions) (string, error) {
	return renderToStringObserved(root, options)
}

// Render is browser-only; the native SSR slice exposes RenderToString instead.
func Render(root Node, selector string) {
	panic(actionableUnsupportedOnServerPanic("Render"))
}

func RenderInto(root Node, target interface{}) error {
	_ = root
	_ = target
	return UnsupportedOnServer("RenderInto")
}

func Hydrate(root Node, selector string, options ...HydrationOptions) (SSRBootstrap, error) {
	return SSRBootstrap{}, UnsupportedOnServer("Hydrate")
}

func HydrateInto(root Node, target interface{}, options ...HydrationOptions) (SSRBootstrap, error) {
	_ = root
	_ = target
	_ = options
	return SSRBootstrap{}, UnsupportedOnServer("HydrateInto")
}

// StartTransition runs fn immediately on non-browser targets.
func StartTransition(fn func()) {
	if fn != nil {
		fn()
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
func (t Transition) Pending() bool {
	if t.pending == nil {
		return false
	}
	return t.pending()
}

// Start runs fn inside a transition.
func (t Transition) Start(fn func()) {
	if t.start != nil {
		t.start(fn)
	}
}

// UseState creates local component state on non-browser targets.
func UseState[T any](initialValue T) State[T] {
	current := initialValue
	return State[T]{
		get: func() T { return current },
		set: func(next interface{}) {
			if value, ok := next.(T); ok {
				current = value
				return
			}
			if updater, ok := next.(func(T) T); ok {
				current = updater(current)
			}
		},
	}
}

// Get returns the current state value.
func (s State[T]) Get() T {
	if s.get == nil {
		var zero T
		return zero
	}
	return s.get()
}

// Set replaces the current state value.
func (s State[T]) Set(value T) {
	if s.set != nil {
		s.set(value)
	}
}

// Update replaces the state value using the previous value.
func (s State[T]) Update(fn func(T) T) {
	if s.set != nil && fn != nil {
		s.set(fn)
	}
}

// UseRef creates a stable mutable reference across renders.
func UseRef[T any](initialValue T) Ref[T] {
	return Ref[T]{current: &initialValue}
}

// Get returns the current ref value.
func (r Ref[T]) Get() T {
	if r.current == nil {
		var zero T
		return zero
	}
	return *r.current
}

// Set updates the current ref value.
func (r Ref[T]) Set(value T) {
	if r.current != nil {
		*r.current = value
	}
}

// UseEffect is a no-op on non-browser targets.
func UseEffect(effect func() func(), deps ...interface{}) {}

// UseId returns a stable generated identifier for the current component instance.
func UseId() string {
	nativeIDMu.Lock()
	defer nativeIDMu.Unlock()
	nativeIDCounter++
	return fmt.Sprintf("gwc-native:%d", nativeIDCounter)
}

// UseMemo computes value immediately on non-browser targets.
func UseMemo[T any](compute func() T, deps ...interface{}) T {
	if compute == nil {
		var zero T
		return zero
	}
	return compute()
}

// UseCallback returns fn unchanged on non-browser targets.
func UseCallback[T any](fn T, deps ...interface{}) T {
	return fn
}

// UseReducer returns a lightweight reducer-backed handle on non-browser targets.
func UseReducer[S any, A any](reducer func(S, A) S, initialState S) Reducer[S, A] {
	current := initialState
	return Reducer[S, A]{
		get: func() S { return current },
		dispatch: func(action A) {
			if reducer != nil {
				current = reducer(current, action)
			}
		},
	}
}

// Get returns the current reducer state.
func (r Reducer[S, A]) Get() S {
	if r.get == nil {
		var zero S
		return zero
	}
	return r.get()
}

// Dispatch applies an action to the reducer state.
func (r Reducer[S, A]) Dispatch(action A) {
	if r.dispatch != nil {
		r.dispatch(action)
	}
}

// UsePrevious reports no previous value on non-browser targets.
func UsePrevious[T any](value T) Previous[T] {
	return Previous[T]{
		value: func() T {
			var zero T
			return zero
		},
		ok: func() bool { return false },
	}
}

// Get returns the previous committed value or the zero value when unavailable.
func (p Previous[T]) Get() T {
	if p.value == nil {
		var zero T
		return zero
	}
	return p.value()
}

// Ok reports whether a previous committed value is available.
func (p Previous[T]) Ok() bool {
	if p.ok == nil {
		return false
	}
	return p.ok()
}

// UseDebounced returns the current value unchanged on non-browser targets.
func UseDebounced[T any](value T, delay time.Duration) Debounced[T] {
	return Debounced[T]{
		get:     func() T { return value },
		pending: func() bool { return false },
	}
}

// Get returns the current debounced value.
func (d Debounced[T]) Get() T {
	if d.get == nil {
		var zero T
		return zero
	}
	return d.get()
}

// Pending reports whether a debounced update is pending.
func (d Debounced[T]) Pending() bool {
	if d.pending == nil {
		return false
	}
	return d.pending()
}

// UseThrottled returns the current value unchanged on non-browser targets.
func UseThrottled[T any](value T, interval time.Duration) Throttled[T] {
	return Throttled[T]{
		get:     func() T { return value },
		pending: func() bool { return false },
	}
}

// Get returns the current throttled value.
func (t Throttled[T]) Get() T {
	if t.get == nil {
		var zero T
		return zero
	}
	return t.get()
}

// Pending reports whether a throttled update is pending.
func (t Throttled[T]) Pending() bool {
	if t.pending == nil {
		return false
	}
	return t.pending()
}

// UseDeferredValue returns value unchanged on non-browser targets.
func UseDeferredValue[T any](value T) T {
	return value
}

func UseContext[T any](context *Context[T]) T {
	panic(actionableUnsupportedOnServerPanic("UseContext"))
}

func AsyncBoundary(props AsyncBoundaryProps) Node {
	if props.Error != nil {
		if props.ErrorFallback != nil {
			return props.ErrorFallback(props.Error)
		}
		if props.Fallback != nil {
			return props.Fallback
		}
		return nil
	}
	if props.Pending {
		if props.Timeout > 0 && props.TimeoutFallback != nil {
			return props.TimeoutFallback
		}
		return props.Fallback
	}
	return props.Content
}

func UseLazyNode(loader func(context.Context) (Node, error), deps ...interface{}) LazyNode {
	state := LazyNodeState{}
	if loader == nil {
		state.Error = UnsupportedOnServer("UseLazyNode")
	} else {
		node, err := loader(context.Background())
		state.Node = node
		state.Error = err
		state.Ready = err == nil && node != nil
	}

	return LazyNode{get: func() LazyNodeState { return state }}
}

func (l LazyNode) Get() LazyNodeState {
	if l.get == nil {
		return LazyNodeState{}
	}
	return l.get()
}

func (l LazyNode) Reload() {
	if l.reload != nil {
		l.reload()
	}
}

func (l LazyNode) Cancel() {
	if l.cancel != nil {
		l.cancel()
	}
}

func Lazy(props LazyProps) Node {
	handle := UseLazyNode(props.Loader, props.Dependencies...)
	state := handle.Get()
	return AsyncBoundary(AsyncBoundaryProps{
		Pending:         state.Loading,
		Error:           state.Error,
		Fallback:        props.Fallback,
		TimeoutFallback: props.TimeoutFallback,
		ErrorFallback:   props.ErrorFallback,
		Content:         state.Node,
		Delay:           props.Delay,
		Timeout:         props.Timeout,
	})
}

// UseEvent wraps a Go function so it can be used as a stable event handler.
func UseEvent(fn interface{}) Handler {
	return Handler{value: fn}
}

// RawHandler wraps an already-prepared handler value.
func RawHandler(value interface{}) Handler {
	return Handler{value: value}
}

// Value returns the wrapped handler payload.
func (h Handler) Value() interface{} {
	return h.value
}

func renderComponent(component interface{}, rawProps map[string]interface{}) *runtime.Element {
	if component == nil {
		return nil
	}

	componentValue := reflect.ValueOf(component)
	if !componentValue.IsValid() || componentValue.Kind() != reflect.Func {
		panic(actionableCreateElementPanic("ui.CreateElement requires a component function or ui.Node"))
	}

	meta := getComponentMeta(componentValue.Type())

	var results []reflect.Value
	if meta.hasArg {
		arg := meta.zeroArg
		if provided, ok := rawProps[propsKey]; ok {
			providedValue := reflect.ValueOf(provided)
			if providedValue.IsValid() {
				switch {
				case providedValue.Type() == meta.argType:
					arg = providedValue
				case providedValue.Type().AssignableTo(meta.argType):
					arg = providedValue
				case providedValue.Type().ConvertibleTo(meta.argType):
					arg = providedValue.Convert(meta.argType)
				}
			}
		}

		var argBuf [1]reflect.Value
		argBuf[0] = arg
		results = componentValue.Call(argBuf[:])
	} else {
		results = componentValue.Call(nil)
	}

	if len(results) == 0 || !results[0].IsValid() || results[0].IsNil() {
		return nil
	}

	element, _ := results[0].Interface().(*runtime.Element)
	return element
}

func getComponentMeta(componentType reflect.Type) componentMeta {
	if cached, ok := componentMetaCache.Load(componentType); ok {
		return cached.(componentMeta)
	}

	if componentType.NumIn() > 1 {
		panic(actionableCreateElementPanic("ui.CreateElement components may accept at most one props argument"))
	}
	if componentType.NumOut() != 1 {
		panic(actionableCreateElementPanic("ui.CreateElement components must return ui.Node"))
	}

	meta := componentMeta{}
	if componentType.NumIn() == 1 {
		meta.hasArg = true
		meta.argType = componentType.In(0)
		meta.zeroArg = reflect.Zero(meta.argType)
	}

	stored, _ := componentMetaCache.LoadOrStore(componentType, meta)
	return stored.(componentMeta)
}

func toInterfaces(children []Node) []interface{} {
	if len(children) == 0 {
		return nil
	}

	values := make([]interface{}, 0, len(children))
	for _, child := range children {
		values = append(values, child)
	}

	return values
}

// UnsupportedOnServer explains the current SSR limitation for hook-based components.
func UnsupportedOnServer(name string) error {
	return fmt.Errorf("%s", unsupportedOnServerMessage(name))
}

func ReadBootstrapScript(scriptID string) (SSRBootstrap, error) {
	return SSRBootstrap{}, UnsupportedOnServer("ReadBootstrapScript")
}

func ReadBootstrapReferenceScript(scriptID string) (SSRBootstrapReference, error) {
	return SSRBootstrapReference{}, UnsupportedOnServer("ReadBootstrapReferenceScript")
}

func ReadBootstrapReference(ref SSRBootstrapReference) (SSRBootstrap, error) {
	return SSRBootstrap{}, UnsupportedOnServer("ReadBootstrapReference")
}
