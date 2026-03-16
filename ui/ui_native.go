//go:build !js || !wasm
// +build !js !wasm

package ui

import (
	"fmt"
	"reflect"
	"sync"

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

func CreateElement(component interface{}, props ...interface{}) Node {
	if node, ok := component.(*runtime.Element); ok && len(props) == 0 {
		return node
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
