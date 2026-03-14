//go:build js && wasm
// +build js,wasm

package ui

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/internal/platform/jsdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

const (
	componentKey = "__ui_component"
	propsKey     = "__ui_props"
)

var initialized bool

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

type State[T any] struct {
	get func() T
	set func(interface{})
}

type Ref[T any] struct {
	raw *runtime.RefValue
}

func CreateElement(component interface{}, props ...interface{}) Node {
	if node, ok := component.(*runtime.Element); ok && len(props) == 0 {
		return node
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

func Render(root Node, selector string) {
	ensureInitialized()
	runtime.GetGlobalRuntime().RenderTo(selector, root)
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

	componentType := componentValue.Type()
	if componentType.NumIn() > 1 {
		panic("ui.CreateElement components may accept at most one props argument")
	}
	if componentType.NumOut() != 1 {
		panic("ui.CreateElement components must return ui.Node")
	}

	args := []reflect.Value{}
	if componentType.NumIn() == 1 {
		arg := reflect.Zero(componentType.In(0))
		if provided, ok := rawProps[propsKey]; ok {
			providedValue := reflect.ValueOf(provided)
			if providedValue.IsValid() {
				switch {
				case providedValue.Type().AssignableTo(componentType.In(0)):
					arg = providedValue
				case providedValue.Type().ConvertibleTo(componentType.In(0)):
					arg = providedValue.Convert(componentType.In(0))
				}
			}
		}
		args = append(args, arg)
	}

	results := componentValue.Call(args)
	if len(results) == 0 || !results[0].IsValid() || results[0].IsNil() {
		return nil
	}

	element, _ := results[0].Interface().(*runtime.Element)
	return element
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
