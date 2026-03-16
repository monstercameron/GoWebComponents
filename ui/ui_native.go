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

const (
	componentKey = "__ui_component"
	propsKey     = "__ui_props"
)

type componentMeta struct {
	hasArg  bool
	argType reflect.Type
	zeroArg reflect.Value
}

var componentMetaCache sync.Map

type Node = *runtime.Element

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

func CreateElement(component interface{}, props ...interface{}) Node {
	if node, ok := component.(*runtime.Element); ok && len(props) == 0 {
		return node
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

func Fragment(children ...Node) Node {
	return runtime.CreateElement("FRAGMENT", nil, toInterfaces(children)...)
}

func Portal(props PortalProps) Node {
	children := make([]Node, 0, len(props.Children)+1)
	if props.Child != nil {
		children = append(children, props.Child)
	}
	children = append(children, props.Children...)
	return Fragment(children...)
}

func Text(content string) Node {
	return &runtime.Element{
		Type:        "TEXT_ELEMENT",
		TextContent: content,
		Children:    []interface{}{},
	}
}

// RenderToString renders a ui.Node tree to HTML on non-browser targets.
func RenderToString(root Node) (string, error) {
	return runtime.RenderToString(root)
}

// Render is browser-only; the native SSR slice exposes RenderToString instead.
func Render(root Node, selector string) {
	panic("ui.Render is only available in js/wasm builds; use ui.RenderToString on the server")
}

func Hydrate(root Node, selector string, options ...HydrationOptions) (SSRBootstrap, error) {
	return SSRBootstrap{}, UnsupportedOnServer("Hydrate")
}

func UseContext[T any](context *Context[T]) T {
	panic(UnsupportedOnServer("UseContext").Error())
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

func UseEvent(fn interface{}) Handler {
	return Handler{value: fn}
}

func RawHandler(value interface{}) Handler {
	return Handler{value: value}
}

func (h Handler) Value() interface{} {
	return h.value
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

// UnsupportedOnServer explains the current SSR limitation for hook-based components.
func UnsupportedOnServer(name string) error {
	return fmt.Errorf("ui.%s is not available on non-js/wasm builds in the current SSR slice", name)
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
