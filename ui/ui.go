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

const (
	componentKey = "__ui_component"
	propsKey     = "__ui_props"
)

var initialized bool

type componentMeta struct {
	hasArg  bool
	argType reflect.Type
	zeroArg reflect.Value
}

var componentMetaCache sync.Map

type Node = *runtime.Element

type Event = runtime.GoEvent
type MouseEvent = runtime.GoEvent
type InputEvent = runtime.GoEvent
type ChangeEvent = runtime.GoEvent
type KeyboardEvent = runtime.GoEvent
type FocusEvent = runtime.GoEvent
type FormEvent = runtime.GoEvent

type Handler struct {
	value interface{}
}

type PortalTarget struct {
	Selector string
	Node     interface{}
}

type PortalProps struct {
	Target   PortalTarget
	Child    Node
	Children []Node
}

type State[T any] struct {
	get func() T
	set func(interface{})
}

type Reducer[S any, A any] struct {
	get      func() S
	dispatch func(A)
}

type Ref[T any] struct {
	raw *runtime.RefValue
}

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

var ErrorBoundary = &errorBoundaryComponent{boundaryType: runtime.NewErrorBoundaryType()}

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
	if fn, ok := component.(func() *runtime.Element); ok {
		return runtime.CreateElement(fn, nil)
	}

	rawProps := map[string]interface{}{
		componentKey: component,
	}

	if len(props) > 0 {
		rawProps[propsKey] = props[0]
	}

	return runtime.CreateElement(renderComponent, rawProps)
}

func (boundary *errorBoundaryComponent) runtimeErrorBoundary() *runtime.ErrorBoundaryType {
	if boundary == nil {
		return nil
	}
	return boundary.boundaryType
}

func Fragment(children ...Node) Node {
	return runtime.CreateElement("FRAGMENT", nil, toInterfaces(children)...)
}

func Portal(props PortalProps) Node {
	children := make([]interface{}, 0, len(props.Children)+1)
	if props.Child != nil {
		children = append(children, props.Child)
	}
	children = append(children, toInterfaces(props.Children)...)

	rawProps := map[string]interface{}{}
	if props.Target.Selector != "" {
		rawProps["portalTargetSelector"] = props.Target.Selector
	}
	if props.Target.Node != nil {
		rawProps["portalTargetNode"] = props.Target.Node
	}

	return runtime.CreateElement(runtime.PortalNodeType, rawProps, children...)
}

func Render(root Node, selector string) {
	ensureInitialized()
	runtime.GetGlobalRuntime().RenderTo(selector, root)
}

// Hydrate is the public client-resume entrypoint for SSR hydration.
//
// It restores the optional bootstrap payload, reuses matching server-rendered
// DOM where possible, and falls back per subtree when hydration cannot
// continue safely.
func Hydrate(root Node, selector string, options ...HydrationOptions) (SSRBootstrap, error) {
	resolved := resolveHydrationOptions(options)
	payload := resolved.Bootstrap
	switch {
	case resolved.ScriptID != "":
		parsed, err := ReadBootstrapScript(resolved.ScriptID)
		if err != nil {
			return SSRBootstrap{}, err
		}
		payload = parsed
	case resolved.ReferenceScriptID != "":
		ref, err := ReadBootstrapReferenceScript(resolved.ReferenceScriptID)
		if err != nil {
			return SSRBootstrap{}, err
		}
		parsed, err := ReadBootstrapReference(ref)
		if err != nil {
			return SSRBootstrap{}, err
		}
		payload = parsed
	case resolved.BootstrapRef.URL != "":
		parsed, err := ReadBootstrapReference(resolved.BootstrapRef)
		if err != nil {
			return SSRBootstrap{}, err
		}
		payload = parsed
	}
	ensureInitialized()
	rt := runtime.GetGlobalRuntime()
	if payload.IDSeed > 0 {
		rt.SetIDSeed(payload.IDSeed)
	}
	if len(payload.Atoms) > 0 {
		if err := rt.RestoreAtomSnapshot(payload.Atoms); err != nil {
			return SSRBootstrap{}, err
		}
	}
	rt.HydrateTo(selector, root)
	return payload, nil
}

func RenderToString(root Node) (string, error) {
	return runtime.RenderToString(root)
}

func Text(content string) Node {
	return runtime.Text(content)
}

func UseState[T any](initialValue T) State[T] {
	get, set := runtime.GoUseStateGlobal(initialValue)
	return State[T]{get: get, set: set}
}

func (s State[T]) Get() T {
	return s.get()
}

func (s State[T]) Set(value T) {
	s.set(value)
}

func (s State[T]) Update(fn func(T) T) {
	s.set(fn)
}

func UseReducer[S any, A any](reducer func(S, A) S, initialState S) Reducer[S, A] {
	state := UseState(initialState)
	return Reducer[S, A]{
		get: func() S { return state.Get() },
		dispatch: func(action A) {
			state.Update(func(prev S) S {
				return reducer(prev, action)
			})
		},
	}
}

func (r Reducer[S, A]) Get() S {
	if r.get == nil {
		var zero S
		return zero
	}
	return r.get()
}

func (r Reducer[S, A]) Dispatch(action A) {
	if r.dispatch != nil {
		r.dispatch(action)
	}
}

func UseEffect(effect func() func(), deps ...interface{}) {
	runtime.GoUseEffectGlobal(effect, deps...)
}

func UseMemo[T any](compute func() T, deps ...interface{}) T {
	value := runtime.GoUseMemoGlobal(func() interface{} {
		return compute()
	}, deps...)

	cast, ok := value.(T)
	if ok {
		return cast
	}

	var zero T
	return zero
}

func UseCallback[T any](fn T, deps ...interface{}) T {
	value := runtime.GoUseCallbackGlobal(fn, deps...)
	cast, ok := value.(T)
	if ok {
		return cast
	}

	var zero T
	return zero
}

func UseRef[T any](initialValue T) Ref[T] {
	return Ref[T]{raw: runtime.GoUseRefGlobal(initialValue)}
}

func UseContext[T any](context *Context[T]) T {
	if context == nil || context.descriptor == nil {
		panic("ui.UseContext called with nil context")
	}
	return castContextValue[T](runtime.GoUseContextValue(context.descriptor))
}

func StartTransition(fn func()) {
	runtime.StartTransitionGlobal(fn)
}

func UseTransition() Transition {
	pending, _ := runtime.GoUseTransitionPendingGlobal()
	return Transition{
		pending: pending,
		start:   StartTransition,
	}
}

func (t Transition) Pending() bool {
	if t.pending == nil {
		return false
	}
	return t.pending()
}

func (t Transition) Start(fn func()) {
	if t.start != nil {
		t.start(fn)
	}
}

func (r Ref[T]) Get() T {
	if r.raw == nil || r.raw.Current == nil {
		var zero T
		return zero
	}

	value, ok := r.raw.Current.(T)
	if ok {
		return value
	}

	var zero T
	return zero
}

func (r Ref[T]) Set(value T) {
	if r.raw != nil {
		r.raw.Current = value
	}
}

// UsePrevious returns a handle for accessing the previous committed value.
// On the first render, Ok returns false and Get returns the zero value for T.
func UsePrevious[T any](value T) Previous[T] {
	stateRef := UseRef(previousState[T]{})
	state := stateRef.Get()

	UseEffect(func() func() {
		stateRef.Set(previousState[T]{value: value, ok: true})
		return nil
	}, value)

	return Previous[T]{
		value: func() T { return state.value },
		ok:    func() bool { return state.ok },
	}
}

// Get returns the previous committed value or the zero value for T when unavailable.
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

// UseDeferredValue keeps returning the last committed value until a transition updates it.
func UseDeferredValue[T any](value T) T {
	deferred := UseState(value)
	current := deferred.Get()
	previous := UsePrevious(value)

	UseEffect(func() func() {
		if !previous.Ok() || reflect.DeepEqual(current, value) {
			return nil
		}
		StartTransition(func() {
			deferred.Set(value)
		})
		return nil
	}, value)

	return current
}

// UseChannel subscribes the component to values received from ch.
//
// The returned handle exposes the latest observed value, whether a value has
// been received yet, and whether the source channel has closed. When the
// component unmounts or the channel changes, the internal goroutine stops
// reading from the old channel.
func UseChannel[T any](ch <-chan T) Channel[T] {
	state := UseState(channelSnapshot[T]{})
	current := state.Get()

	visible := current
	if current.source != ch {
		visible = channelSnapshot[T]{}
	}

	UseEffect(func() func() {
		state.Set(channelSnapshot[T]{source: ch})
		if ch == nil {
			return nil
		}

		stop := make(chan struct{})
		go func() {
			for {
				select {
				case <-stop:
					return
				case value, ok := <-ch:
					select {
					case <-stop:
						return
					default:
					}

					if !ok {
						previous := state.Get()
						state.Set(channelSnapshot[T]{
							source: ch,
							value:  previous.value,
							ok:     previous.ok,
							closed: true,
						})
						return
					}

					state.Set(channelSnapshot[T]{
						source: ch,
						value:  value,
						ok:     true,
						closed: false,
					})
				}
			}
		}()

		return func() {
			close(stop)
		}
	}, ch)

	return Channel[T]{
		getValue: func() T { return visible.value },
		hasValue: func() bool { return visible.ok },
		isClosed: func() bool { return visible.closed },
	}
}

// Get returns the latest observed channel value or the zero value for T.
func (c Channel[T]) Get() T {
	if c.getValue == nil {
		var zero T
		return zero
	}

	return c.getValue()
}

// Ok reports whether the channel has produced at least one value.
func (c Channel[T]) Ok() bool {
	if c.hasValue == nil {
		return false
	}

	return c.hasValue()
}

// Closed reports whether the source channel has been observed closing.
func (c Channel[T]) Closed() bool {
	if c.isClosed == nil {
		return false
	}

	return c.isClosed()
}

// UseTask creates a cancellable background task driven by a Go function.
//
// The task only runs when Start is called. In-flight work is cancelled when the
// component unmounts or when Cancel is called explicitly.
func UseTask[T any](run func(context.Context) (T, error)) Task[T] {
	state := UseState(TaskState[T]{})
	cancelRef := UseRef((context.CancelFunc)(nil))
	requestSeq := UseRef(0)

	start := func() {
		if cancel := cancelRef.Get(); cancel != nil {
			cancel()
		}

		requestSeq.Set(requestSeq.Get() + 1)
		seq := requestSeq.Get()
		ctx, cancel := context.WithCancel(context.Background())
		cancelRef.Set(cancel)

		state.Update(func(prev TaskState[T]) TaskState[T] {
			prev.Running = true
			prev.Ready = false
			prev.Cancelled = false
			prev.Started = true
			prev.Error = nil
			return prev
		})

		go func() {
			value, err := run(ctx)
			if ctx.Err() != nil || requestSeq.Get() != seq {
				return
			}

			state.Set(TaskState[T]{
				Value:     value,
				Running:   false,
				Ready:     err == nil,
				Cancelled: false,
				Started:   true,
				Error:     err,
			})
		}()
	}

	cancel := func() {
		if activeCancel := cancelRef.Get(); activeCancel != nil {
			activeCancel()
			cancelRef.Set(nil)
		}

		state.Update(func(prev TaskState[T]) TaskState[T] {
			prev.Running = false
			prev.Cancelled = true
			prev.Started = true
			return prev
		})
	}

	UseEffect(func() func() {
		return func() {
			if activeCancel := cancelRef.Get(); activeCancel != nil {
				activeCancel()
				cancelRef.Set(nil)
			}
		}
	}, true)

	return Task[T]{
		get:    func() TaskState[T] { return state.Get() },
		start:  start,
		cancel: cancel,
	}
}

// Get returns the current task state.
func (t Task[T]) Get() TaskState[T] {
	if t.get == nil {
		var zero TaskState[T]
		return zero
	}

	return t.get()
}

// Start launches the task, cancelling any previous in-flight run.
func (t Task[T]) Start() {
	if t.start != nil {
		t.start()
	}
}

// Cancel cancels the in-flight task run, if any.
func (t Task[T]) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

// AsyncBoundary renders content, fallback, timeout fallback, or an error
// fallback depending on the current async state.
func AsyncBoundary(props AsyncBoundaryProps) Node {
	phase := UseState(asyncBoundaryState{fallbackVisible: props.Delay <= 0})

	UseEffect(func() func() {
		if !props.Pending || props.Error != nil {
			phase.Set(asyncBoundaryState{fallbackVisible: props.Delay <= 0})
			return nil
		}

		state := asyncBoundaryState{fallbackVisible: props.Delay <= 0}
		phase.Set(state)

		stop := make(chan struct{})
		if props.Delay > 0 {
			go func(delay time.Duration) {
				timer := time.NewTimer(delay)
				defer timer.Stop()

				select {
				case <-stop:
					return
				case <-timer.C:
				}

				current := phase.Get()
				current.fallbackVisible = true
				phase.Set(current)
			}(props.Delay)
		}

		if props.Timeout > 0 {
			go func(timeout time.Duration) {
				timer := time.NewTimer(timeout)
				defer timer.Stop()

				select {
				case <-stop:
					return
				case <-timer.C:
				}

				current := phase.Get()
				current.timedOut = true
				phase.Set(current)
			}(props.Timeout)
		}

		return func() {
			close(stop)
		}
	}, props.Pending, props.Error, props.Delay, props.Timeout)

	state := phase.Get()
	if props.Error != nil {
		if props.ErrorFallback != nil {
			return props.ErrorFallback(props.Error)
		}
		if props.Fallback != nil {
			return props.Fallback
		}
		return nil
	}

	if !props.Pending {
		return props.Content
	}

	if state.timedOut && props.TimeoutFallback != nil {
		return props.TimeoutFallback
	}

	if state.fallbackVisible {
		return props.Fallback
	}

	return props.Content
}

// UseLazyNode asynchronously resolves a ui.Node and tracks loading/error state.
func UseLazyNode(loader func(context.Context) (Node, error), deps ...interface{}) LazyNode {
	state := UseState(LazyNodeState{Loading: true})
	reloadTick := UseState(0)
	cancelRef := UseRef((context.CancelFunc)(nil))
	requestSeq := UseRef(0)

	startLoad := func() {
		if cancel := cancelRef.Get(); cancel != nil {
			cancel()
		}

		if loader == nil {
			state.Set(LazyNodeState{Error: context.Canceled})
			return
		}

		requestSeq.Set(requestSeq.Get() + 1)
		seq := requestSeq.Get()
		ctx, cancel := context.WithCancel(context.Background())
		cancelRef.Set(cancel)

		state.Update(func(prev LazyNodeState) LazyNodeState {
			prev.Loading = true
			prev.Error = nil
			return prev
		})

		go func() {
			node, err := loader(ctx)
			if ctx.Err() != nil || requestSeq.Get() != seq {
				return
			}

			state.Set(LazyNodeState{
				Node:    node,
				Loading: false,
				Error:   err,
				Ready:   err == nil,
			})
		}()
	}

	effectDeps := make([]interface{}, 0, len(deps)+1)
	effectDeps = append(effectDeps, reloadTick.Get())
	effectDeps = append(effectDeps, deps...)

	UseEffect(func() func() {
		startLoad()
		return func() {
			if cancel := cancelRef.Get(); cancel != nil {
				cancel()
				cancelRef.Set(nil)
			}
		}
	}, effectDeps...)

	return LazyNode{
		get: func() LazyNodeState { return state.Get() },
		reload: func() {
			reloadTick.Update(func(prev int) int { return prev + 1 })
		},
		cancel: func() {
			if cancel := cancelRef.Get(); cancel != nil {
				cancel()
				cancelRef.Set(nil)
			}
			state.Update(func(prev LazyNodeState) LazyNodeState {
				prev.Loading = false
				return prev
			})
		},
	}
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

// Lazy asynchronously resolves a subtree and renders it through AsyncBoundary.
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

// UseDebounced returns a delayed view of value that only updates after delay has
// elapsed without a newer value replacing it.
func UseDebounced[T any](value T, delay time.Duration) Debounced[T] {
	state := UseState(delayedValueState[T]{value: value})
	firstRun := UseRef(true)
	version := UseRef(0)

	UseEffect(func() func() {
		if firstRun.Get() {
			firstRun.Set(false)
			return nil
		}

		if delay <= 0 {
			state.Set(delayedValueState[T]{value: value})
			return nil
		}

		sequence := version.Get() + 1
		version.Set(sequence)
		state.Update(func(prev delayedValueState[T]) delayedValueState[T] {
			prev.pending = true
			return prev
		})

		stop := make(chan struct{})
		go func(next T, expected int) {
			timer := time.NewTimer(delay)
			defer timer.Stop()

			select {
			case <-stop:
				return
			case <-timer.C:
			}

			if version.Get() != expected {
				return
			}
			state.Set(delayedValueState[T]{value: next})
		}(value, sequence)

		return func() {
			close(stop)
		}
	}, value, delay)

	return Debounced[T]{
		get:     func() T { return state.Get().value },
		pending: func() bool { return state.Get().pending },
	}
}

func (d Debounced[T]) Get() T {
	if d.get == nil {
		var zero T
		return zero
	}
	return d.get()
}

func (d Debounced[T]) Pending() bool {
	if d.pending == nil {
		return false
	}
	return d.pending()
}

// UseThrottled returns a trailing-throttled view of value that updates at most
// once per interval while still eventually applying the latest value.
func UseThrottled[T any](value T, interval time.Duration) Throttled[T] {
	state := UseState(delayedValueState[T]{value: value})
	firstRun := UseRef(true)
	version := UseRef(0)
	lastEmit := UseRef(time.Time{})

	UseEffect(func() func() {
		if firstRun.Get() {
			firstRun.Set(false)
			lastEmit.Set(time.Now())
			return nil
		}

		now := time.Now()
		if interval <= 0 {
			lastEmit.Set(now)
			state.Set(delayedValueState[T]{value: value})
			return nil
		}

		if emittedAt := lastEmit.Get(); emittedAt.IsZero() || now.Sub(emittedAt) >= interval {
			lastEmit.Set(now)
			state.Set(delayedValueState[T]{value: value})
			return nil
		}

		sequence := version.Get() + 1
		version.Set(sequence)
		state.Update(func(prev delayedValueState[T]) delayedValueState[T] {
			prev.pending = true
			return prev
		})

		remaining := interval - now.Sub(lastEmit.Get())
		stop := make(chan struct{})
		go func(next T, expected int, wait time.Duration) {
			timer := time.NewTimer(wait)
			defer timer.Stop()

			select {
			case <-stop:
				return
			case <-timer.C:
			}

			if version.Get() != expected {
				return
			}
			lastEmit.Set(time.Now())
			state.Set(delayedValueState[T]{value: next})
		}(value, sequence, remaining)

		return func() {
			close(stop)
		}
	}, value, interval)

	return Throttled[T]{
		get:     func() T { return state.Get().value },
		pending: func() bool { return state.Get().pending },
	}
}

func (t Throttled[T]) Get() T {
	if t.get == nil {
		var zero T
		return zero
	}
	return t.get()
}

func (t Throttled[T]) Pending() bool {
	if t.pending == nil {
		return false
	}
	return t.pending()
}

func UseId() string {
	return runtime.GoUseIdGlobal()
}

func UseEvent(fn interface{}) Handler {
	return Handler{value: runtime.GoUseFunc(fn)}
}

func RawHandler(value interface{}) Handler {
	return Handler{value: value}
}

func (h Handler) Value() interface{} {
	return h.value
}

func ensureInitialized() {
	if initialized {
		return
	}

	runtime.InitGlobalRuntime(runtime.Config{
		DOMAdapter:   jsdom.NewWASMDOMAdapter(),
		EventAdapter: jsdom.NewWASMEventAdapter(),
		Scheduler:    jsdom.NewWASMScheduler(),
		BrowserState: jsdom.NewWASMBrowserState(),
	})
	initialized = true
}

func renderComponent(rawProps map[string]interface{}) *runtime.Element {
	component := rawProps[componentKey]
	if component == nil {
		return nil
	}

	componentValue := reflect.ValueOf(component)
	if !componentValue.IsValid() || componentValue.Kind() != reflect.Func {
		panic("ui.CreateElement requires a component function or ui.Node")
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
		panic("ui.CreateElement components may accept at most one props argument")
	}
	if componentType.NumOut() != 1 {
		panic("ui.CreateElement components must return ui.Node")
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
