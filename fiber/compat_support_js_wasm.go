//go:build js && wasm
// +build js,wasm

package fiber

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall/js"

	iruntime "github.com/monstercameron/GoWebComponents/internal/runtime"
)

type Attrs map[string]interface{}

type poolUtilizationStats struct {
	totalAllocations int64
	totalPoolHits    int64
	fiberHits        int64
	fiberMisses      int64
}

type poolSizeStats struct {
	fiber int32
}

type poolConfigStats struct {
	fiberPoolSize int32
}

var (
	fiberPool        sync.Pool
	poolUtilization  poolUtilizationStats
	poolSizes        poolSizeStats
	poolConfig       = poolConfigStats{fiberPoolSize: 1024}
)

func debugf(namespace, format string, a ...interface{}) {}

func optimizePoolSizes() {}

func finalizeHookOrder(_ *Hooks) error { return nil }

func getFunctionName(fn interface{}) string {
	return fmt.Sprintf("%T", fn)
}

func fastEqual(left, right interface{}) bool {
	return reflect.DeepEqual(left, right)
}

func checkMemoryPressure() {}

func startSchedulerSection() {
	atomic.StoreInt32(&schedulerActive, 1)
}

func endSchedulerSection() {
	atomic.StoreInt32(&schedulerActive, 0)
}

func processUIQueue() {
	for {
		select {
		case fn := <-uiQueue:
			if fn != nil {
				fn()
			}
		default:
			return
		}
	}
}

func createElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	if props == nil {
		props = make(map[string]interface{})
	}
	props["children"] = children
	return &Element{
		Type:  typ,
		Props: props,
		Children: children,
	}
}

func createDom(fiber *Fiber) js.Value {
	document := js.Global().Get("document")
	if document.IsUndefined() || document.IsNull() {
		return js.Null()
	}

	if tagName, ok := fiber.typeOf.(string); ok && tagName == "TEXT_ELEMENT" {
		text := ""
		if fiber.props != nil {
			if value, ok := fiber.props["nodeValue"].(string); ok {
				text = value
			}
		}
		return document.Call("createTextNode", text)
	}

	tagName, _ := fiber.typeOf.(string)
	if tagName == "" {
		tagName = "div"
	}
	return document.Call("createElement", tagName)
}

func updateDom(dom js.Value, _ map[string]interface{}, nextProps map[string]interface{}) {
	if dom.IsUndefined() || dom.IsNull() || nextProps == nil {
		return
	}
	for key, value := range nextProps {
		switch key {
		case "children", "key":
			continue
		case "nodeValue":
			if dom.Get("nodeType").Int() == 3 {
				dom.Set("nodeValue", fmt.Sprint(value))
			}
		default:
			dom.Call("setAttribute", key, fmt.Sprint(value))
		}
	}
}

func getEffectFibersSlice() *[]*Fiber {
	effectFibers := make([]*Fiber, 0, 16)
	return &effectFibers
}

func returnEffectFibersSlice(effectFibers *[]*Fiber) {}

func releaseHooks(hooks *Hooks) {}

func resetFiber(fiber *Fiber) {
	if fiber == nil {
		return
	}
	fiber.parent = nil
	fiber.alternate = nil
	fiber.child = nil
	fiber.sibling = nil
	fiber.hooks = nil
	fiber.typeOf = nil
	fiber.props = nil
	fiber.dom = js.Value{}
	fiber.effectTag = ""
	fiber.effects = nil
	fiber.eventCallbacks = nil
}

func GoUseState[T any](initialValue T) (func() T, func(T)) {
	get, set := iruntime.GoUseStateGlobal(initialValue)
	return get, func(value T) {
		set(value)
	}
}

func GoUseEffect(effect func(), deps []interface{}) {
	iruntime.GoUseEffectGlobal(func() func() {
		effect()
		return nil
	}, deps...)
}

func GoUseFetch(url string, options ...interface{}) (func() FetchState, func()) {
	get, refetch := iruntime.GoUseFetchGlobal(url, options...)
	return func() FetchState {
		state := get()
		return FetchState{
			Data:    state.Data,
			Error:   state.Error,
			Loading: state.Loading,
		}
	}, refetch
}

func Text(content string) *Element {
	return &Element{
		Type: "TEXT_ELEMENT",
		Props: map[string]interface{}{
			"nodeValue": content,
			"children":  []interface{}{},
		},
		Children: []interface{}{},
	}
}

func tag(name string, props Attrs, children ...interface{}) *Element {
	return createElement(name, map[string]interface{}(props), children...)
}

func Div(props Attrs, children ...interface{}) *Element { return tag("div", props, children...) }
func H1(props Attrs, children ...interface{}) *Element { return tag("h1", props, children...) }
func H3(props Attrs, children ...interface{}) *Element { return tag("h3", props, children...) }
func H4(props Attrs, children ...interface{}) *Element { return tag("h4", props, children...) }
func P(props Attrs, children ...interface{}) *Element { return tag("p", props, children...) }
func Span(props Attrs, children ...interface{}) *Element { return tag("span", props, children...) }
func Pre(props Attrs, children ...interface{}) *Element { return tag("pre", props, children...) }
func Ul(props Attrs, children ...interface{}) *Element { return tag("ul", props, children...) }
func Li(props Attrs, children ...interface{}) *Element { return tag("li", props, children...) }
func Button(props Attrs, children ...interface{}) *Element { return tag("button", props, children...) }
func Input(props Attrs, children ...interface{}) *Element { return tag("input", props, children...) }