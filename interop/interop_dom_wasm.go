//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"errors"
	"strings"
	"sync"
	"syscall/js"
)

type moduleState struct {
	mu        sync.RWMutex
	specifier string
	value     js.Value
	disposed  bool
}

func (parseM *moduleState) export(parseName string) (js.Value, error) {
	parseM.mu.RLock()
	defer parseM.mu.RUnlock()
	if parseM.disposed {
		return js.Undefined(), wrapError("Module", parseM.specifier, CodeDisposed, errors.New("module handle is disposed"))
	}
	parseRaw := parseM.value.Get(parseName)
	if parseRaw.IsUndefined() || parseRaw.IsNull() {
		return js.Undefined(), wrapError("Module.Value", parseName, CodeMissingExport, errors.New("export not found"))
	}
	return parseRaw, nil
}

func (parseM *moduleState) callDefault(parseCtx context.Context, parseArgs ...any) (any, error) {
	parseRaw, parseErr := parseM.export("default")
	if parseErr != nil {
		return nil, parseErr
	}
	if parseRaw.Type() != js.TypeFunction {
		return nil, wrapError("Module.CallDefault", "default", CodeNotFunction, errors.New("default export is not a function"))
	}
	parseJsArgs := make([]interface{}, 0, len(parseArgs))
	for _, parseArg := range parseArgs {
		parseValue, parseErr2 := goValueToJS("Module.CallDefault", "default", parseArg)
		if parseErr2 != nil {
			return nil, parseErr2
		}
		parseJsArgs = append(parseJsArgs, parseValue)
	}
	parseResult, parseErr := awaitValue(parseCtx, "Module.CallDefault", "default", parseRaw.Invoke(parseJsArgs...))
	if parseErr != nil {
		return nil, parseErr
	}
	return jsValueToGo("Module.CallDefault", "default", parseResult)
}

func (parseM *moduleState) dispose() error {
	parseM.mu.Lock()
	defer parseM.mu.Unlock()
	parseM.disposed = true
	parseM.value = js.Undefined()
	return nil
}

func newEventTarget(parseName string, parseRaw js.Value) EventTarget {
	return EventTarget{
		dispatch: func(parseEventName string, parseDetail2 any) error {
			parseCustomEventCtor := js.Global().Get("CustomEvent")
			if parseCustomEventCtor.IsUndefined() || parseCustomEventCtor.IsNull() {
				return unavailable("EventTarget.Dispatch", parseName)
			}
			parseDetailValue, parseErr := goValueToJS("EventTarget.Dispatch", parseEventName, parseDetail2)
			if parseErr != nil {
				return parseErr
			}
			parseInit := js.Global().Get("Object").New()
			parseInit.Set("detail", parseDetailValue)
			parseRaw.Call("dispatchEvent", parseCustomEventCtor.New(parseEventName, parseInit))
			return nil
		},
		listen: func(parseEventName2 string, handler func(BrowserEvent)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("EventTarget.Listen", parseEventName2, CodeInvalid, errors.New("handler is nil"))
			}
			if parseAdd := parseRaw.Get("addEventListener"); parseAdd.Type() != js.TypeFunction {
				return Subscription{}, unavailable("EventTarget.Listen", parseName)
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				if len(parseArgs) == 0 {
					handler(BrowserEvent{Type: parseEventName2})
					return nil
				}
				parseEvent := parseArgs[0]
				parseDetail, _ := jsValueToGo("EventTarget.Listen", parseEventName2, parseEvent.Get("detail"))
				handler(BrowserEvent{
					Type:          parseEvent.Get("type").String(),
					Detail:        parseDetail,
					Target:        elementFromJSValue(parseEvent.Get("target")),
					CurrentTarget: elementFromJSValue(parseEvent.Get("currentTarget")),
				})
				return nil
			})
			parseRaw.Call("addEventListener", parseEventName2, parseListener)
			return Subscription{cancel: func() {
				parseRaw.Call("removeEventListener", parseEventName2, parseListener)
				parseListener.Release()
			}}, nil
		},
	}
}

func newElement(parseName string, parseRaw js.Value) Element {
	return Element{
		raw:       parseRaw,
		tagName:   func() string { return parseRaw.Get("tagName").String() },
		id:        func() string { return parseRaw.Get("id").String() },
		className: func() string { return parseRaw.Get("className").String() },
		focus: func() error {
			if parseFn := parseRaw.Get("focus"); parseFn.Type() != js.TypeFunction {
				return unavailable("Element.Focus", parseName)
			}
			parseRaw.Call("focus")
			return nil
		},
		blur: func() error {
			if parseFn2 := parseRaw.Get("blur"); parseFn2.Type() != js.TypeFunction {
				return unavailable("Element.Blur", parseName)
			}
			parseRaw.Call("blur")
			return nil
		},
		click: func() error {
			if parseFn3 := parseRaw.Get("click"); parseFn3.Type() != js.TypeFunction {
				return unavailable("Element.Click", parseName)
			}
			parseRaw.Call("click")
			return nil
		},
		setScrollTop: func(parseScrollTop float64) error {
			parseRaw.Set("scrollTop", parseScrollTop)
			return nil
		},
		scrollIntoView: func(parseOptions ScrollIntoViewOptions) error {
			if parseFn4 := parseRaw.Get("scrollIntoView"); parseFn4.Type() != js.TypeFunction {
				return unavailable("Element.ScrollIntoView", parseName)
			}
			parseInit := js.Global().Get("Object").New()
			hasOptions := false
			if strings.TrimSpace(parseOptions.Behavior) != "" {
				parseInit.Set("behavior", parseOptions.Behavior)
				hasOptions = true
			}
			if strings.TrimSpace(parseOptions.Block) != "" {
				parseInit.Set("block", parseOptions.Block)
				hasOptions = true
			}
			if strings.TrimSpace(parseOptions.Inline) != "" {
				parseInit.Set("inline", parseOptions.Inline)
				hasOptions = true
			}
			if !hasOptions {
				parseRaw.Call("scrollIntoView")
				return nil
			}
			parseRaw.Call("scrollIntoView", parseInit)
			return nil
		},
		boundingClientRect: func() (Rect, error) {
			if parseFn5 := parseRaw.Get("getBoundingClientRect"); parseFn5.Type() != js.TypeFunction {
				return Rect{}, unavailable("Element.BoundingClientRect", parseName)
			}
			return rectFromJSValue(parseRaw.Call("getBoundingClientRect")), nil
		},
		events: func() (EventTarget, error) {
			return newEventTarget(parseName, parseRaw), nil
		},
		observeResize: func(handler func(ResizeEntry)) (Subscription, error) {
			return observeResize(parseName, parseRaw, handler)
		},
		observeIntersection: func(parseOptions2 IntersectionObserverOptions, handler func(IntersectionEntry)) (Subscription, error) {
			return observeIntersection(parseName, parseRaw, parseOptions2, handler)
		},
		scrollMetrics: func() (float64, float64, float64, error) {
			return parseRaw.Get("scrollTop").Float(), parseRaw.Get("scrollHeight").Float(), parseRaw.Get("clientHeight").Float(), nil
		},
	}
}

func elementFromJSValue(parseValue js.Value) Element {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return Element{}
	}
	return newElement("element", parseValue)
}

func rectFromJSValue(parseValue js.Value) Rect {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return Rect{}
	}
	return Rect{
		X:      parseValue.Get("x").Float(),
		Y:      parseValue.Get("y").Float(),
		Width:  parseValue.Get("width").Float(),
		Height: parseValue.Get("height").Float(),
		Top:    parseValue.Get("top").Float(),
		Right:  parseValue.Get("right").Float(),
		Bottom: parseValue.Get("bottom").Float(),
		Left:   parseValue.Get("left").Float(),
	}
}

func observeResize(parseName string, parseRaw js.Value, parseHandler func(ResizeEntry)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("Element.ObserveResize", parseName, CodeInvalid, errors.New("handler is nil"))
	}
	parseCtor := js.Global().Get("ResizeObserver")
	if parseCtor.IsUndefined() || parseCtor.IsNull() {
		return Subscription{}, unavailable("Element.ObserveResize", parseName)
	}
	parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseEntries := parseArgs[0]
		parseLength := parseEntries.Get("length").Int()
		for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
			parseEntry := parseEntries.Index(parseIndex)
			parseHandler(ResizeEntry{
				Target:      elementFromJSValue(parseEntry.Get("target")),
				ContentRect: rectFromJSValue(parseEntry.Get("contentRect")),
			})
		}
		return nil
	})
	parseObserver := parseCtor.New(parseCallback)
	parseObserver.Call("observe", parseRaw)
	return Subscription{cancel: func() {
		parseObserver.Call("disconnect")
		parseCallback.Release()
	}}, nil
}

func observeIntersection(parseName string, parseRaw js.Value, parseOptions IntersectionObserverOptions, parseHandler func(IntersectionEntry)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("Element.ObserveIntersection", parseName, CodeInvalid, errors.New("handler is nil"))
	}
	parseCtor := js.Global().Get("IntersectionObserver")
	if parseCtor.IsUndefined() || parseCtor.IsNull() {
		return Subscription{}, unavailable("Element.ObserveIntersection", parseName)
	}
	parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseEntries := parseArgs[0]
		parseLength := parseEntries.Get("length").Int()
		for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
			parseEntry := parseEntries.Index(parseIndex)
			var parseRootBounds *Rect
			if parseBounds := parseEntry.Get("rootBounds"); !parseBounds.IsUndefined() && !parseBounds.IsNull() {
				parseRect := rectFromJSValue(parseBounds)
				parseRootBounds = &parseRect
			}
			parseHandler(IntersectionEntry{
				Target:             elementFromJSValue(parseEntry.Get("target")),
				IsIntersecting:     parseEntry.Get("isIntersecting").Bool(),
				IntersectionRatio:  parseEntry.Get("intersectionRatio").Float(),
				BoundingClientRect: rectFromJSValue(parseEntry.Get("boundingClientRect")),
				IntersectionRect:   rectFromJSValue(parseEntry.Get("intersectionRect")),
				RootBounds:         parseRootBounds,
			})
		}
		return nil
	})
	if len(parseOptions.Thresholds) == 0 && parseOptions.RootMargin == "" && parseOptions.Root.raw == nil {
		parseObserver := parseCtor.New(parseCallback)
		parseObserver.Call("observe", parseRaw)
		return Subscription{cancel: func() {
			parseObserver.Call("disconnect")
			parseCallback.Release()
		}}, nil
	}
	parseInit := js.Global().Get("Object").New()
	if parseOptions.RootMargin != "" {
		parseInit.Set("rootMargin", parseOptions.RootMargin)
	}
	if len(parseOptions.Thresholds) > 0 {
		parseThresholds := js.Global().Get("Array").New()
		for _, parseThreshold := range parseOptions.Thresholds {
			parseThresholds.Call("push", parseThreshold)
		}
		parseInit.Set("threshold", parseThresholds)
	}
	if parseRoot, parseOk := parseOptions.Root.raw.(js.Value); parseOk && !parseRoot.IsUndefined() && !parseRoot.IsNull() {
		parseInit.Set("root", parseRoot)
	}
	parseObserver2 := parseCtor.New(parseCallback, parseInit)
	parseObserver2.Call("observe", parseRaw)
	return Subscription{cancel: func() {
		parseObserver2.Call("disconnect")
		parseCallback.Release()
	}}, nil
}
