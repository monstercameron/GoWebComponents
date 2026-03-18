//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// LocalStorage returns the browser localStorage wrapper.
func LocalStorage() (Storage, error) {
	return resolveStorage("localStorage")
}

// SessionStorage returns the browser sessionStorage wrapper.
func SessionStorage() (Storage, error) {
	return resolveStorage("sessionStorage")
}

func resolveStorage(name string) (Storage, error) {
	raw, err := globalProperty("Storage", name)
	if err != nil {
		return Storage{}, err
	}
	return Storage{
		getItem: func(key string) (string, bool, error) {
			value := raw.Call("getItem", key)
			if value.IsUndefined() || value.IsNull() {
				return "", false, nil
			}
			return value.String(), true, nil
		},
		setItem: func(key string, value string) error {
			raw.Call("setItem", key, value)
			return nil
		},
		removeItem: func(key string) error {
			raw.Call("removeItem", key)
			return nil
		},
		clear: func() error {
			raw.Call("clear")
			return nil
		},
		length: func() (int, error) {
			return raw.Get("length").Int(), nil
		},
		key: func(index int) (string, bool, error) {
			value := raw.Call("key", index)
			if value.IsUndefined() || value.IsNull() {
				return "", false, nil
			}
			return value.String(), true, nil
		},
	}, nil
}

func WindowLocation() (Location, error) {
	raw, err := globalPath("Location", "window", "location")
	if err != nil {
		return Location{}, err
	}
	return Location{
		href:     func() string { return raw.Get("href").String() },
		pathname: func() string { return raw.Get("pathname").String() },
		search:   func() string { return raw.Get("search").String() },
		hash:     func() string { return raw.Get("hash").String() },
		origin:   func() string { return raw.Get("origin").String() },
		assign: func(rawURL string) error {
			raw.Call("assign", rawURL)
			return nil
		},
		replace: func(rawURL string) error {
			raw.Call("replace", rawURL)
			return nil
		},
		reload: func() error {
			raw.Call("reload")
			return nil
		},
	}, nil
}

func WindowHistory() (History, error) {
	raw, err := globalPath("History", "window", "history")
	if err != nil {
		return History{}, err
	}
	return History{
		length: func() (int, error) {
			return raw.Get("length").Int(), nil
		},
		state: func() (any, error) {
			return jsValueToGo("History.State", "history.state", raw.Get("state"))
		},
		back: func() error {
			raw.Call("back")
			return nil
		},
		forward: func() error {
			raw.Call("forward")
			return nil
		},
		goDelta: func(delta int) error {
			raw.Call("go", delta)
			return nil
		},
		pushState: func(state any, title string, rawURL string) error {
			value, err := goValueToJS("History.PushState", "state", state)
			if err != nil {
				return err
			}
			raw.Call("pushState", value, title, rawURL)
			return nil
		},
		replaceState: func(state any, title string, rawURL string) error {
			value, err := goValueToJS("History.ReplaceState", "state", state)
			if err != nil {
				return err
			}
			raw.Call("replaceState", value, title, rawURL)
			return nil
		},
	}, nil
}

func NavigatorClipboard() (Clipboard, error) {
	raw, err := globalPath("Clipboard", "navigator", "clipboard")
	if err != nil {
		return Clipboard{}, err
	}
	return Clipboard{
		writeText: func(ctx context.Context, text string) error {
			_, err := awaitValue(ctx, "Clipboard.WriteText", "navigator.clipboard.writeText", raw.Call("writeText", text))
			return err
		},
		readText: func(ctx context.Context) (string, error) {
			value, err := awaitValue(ctx, "Clipboard.ReadText", "navigator.clipboard.readText", raw.Call("readText"))
			if err != nil {
				return "", err
			}
			if value.IsUndefined() || value.IsNull() {
				return "", nil
			}
			return value.String(), nil
		},
	}, nil
}

func SetTimeout(delay time.Duration, fn func()) (Timer, error) {
	if fn == nil {
		return Timer{}, wrapError("SetTimeout", "", CodeInvalid, errors.New("callback is nil"))
	}
	var (
		callback js.Func
		once     sync.Once
		id       js.Value
	)
	callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		once.Do(func() {
			callback.Release()
		})
		fn()
		return nil
	})
	id = js.Global().Call("setTimeout", callback, durationMS(delay))
	return Timer{
		cancel: func() error {
			once.Do(func() {
				js.Global().Call("clearTimeout", id)
				callback.Release()
			})
			return nil
		},
	}, nil
}

func SetInterval(interval time.Duration, fn func()) (Timer, error) {
	if fn == nil {
		return Timer{}, wrapError("SetInterval", "", CodeInvalid, errors.New("callback is nil"))
	}
	var (
		callback js.Func
		once     sync.Once
		id       js.Value
	)
	callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fn()
		return nil
	})
	id = js.Global().Call("setInterval", callback, durationMS(interval))
	return Timer{
		cancel: func() error {
			once.Do(func() {
				js.Global().Call("clearInterval", id)
				callback.Release()
			})
			return nil
		},
	}, nil
}

func WindowEvents() (EventTarget, error) {
	raw, err := globalProperty("EventTarget", "window")
	if err != nil {
		return EventTarget{}, err
	}
	return newEventTarget("window", raw), nil
}

func DocumentEvents() (EventTarget, error) {
	raw, err := globalProperty("EventTarget", "document")
	if err != nil {
		return EventTarget{}, err
	}
	return newEventTarget("document", raw), nil
}

func MatchMedia(query string) (MediaQueryList, error) {
	window, err := globalProperty("MatchMedia", "window")
	if err != nil {
		return MediaQueryList{}, err
	}
	raw := window.Call("matchMedia", query)
	if raw.IsUndefined() || raw.IsNull() {
		return MediaQueryList{}, unavailable("MatchMedia", query)
	}
	return MediaQueryList{
		matches: func() bool { return raw.Get("matches").Bool() },
		media:   func() string { return raw.Get("media").String() },
		subscribe: func(handler func(MediaQueryEvent)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("MatchMedia.Subscribe", query, CodeInvalid, errors.New("handler is nil"))
			}
			listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				event := raw
				if len(args) > 0 && !args[0].IsUndefined() && !args[0].IsNull() {
					event = args[0]
				}
				handler(MediaQueryEvent{
					Matches: event.Get("matches").Bool(),
					Media:   event.Get("media").String(),
				})
				return nil
			})
			if add := raw.Get("addEventListener"); add.Type() == js.TypeFunction {
				raw.Call("addEventListener", "change", listener)
				return Subscription{cancel: func() {
					raw.Call("removeEventListener", "change", listener)
					listener.Release()
				}}, nil
			}
			raw.Call("addListener", listener)
			return Subscription{cancel: func() {
				raw.Call("removeListener", listener)
				listener.Release()
			}}, nil
		},
	}, nil
}

func ImportModule(ctx context.Context, specifier string) (Module, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(specifier) == "" {
		return Module{}, wrapError("ImportModule", specifier, CodeInvalid, errors.New("specifier is empty"))
	}
	importFn := js.Global().Get("__gwcImportModule")
	if importFn.IsUndefined() || importFn.IsNull() {
		constructor := js.Global().Get("Function")
		if constructor.IsUndefined() || constructor.IsNull() {
			return Module{}, unavailable("ImportModule", specifier)
		}
		importFn = constructor.New("specifier", "return import(specifier);")
	}
	rawModule, err := awaitValue(ctx, "ImportModule", specifier, importFn.Invoke(specifier))
	if err != nil {
		return Module{}, err
	}
	state := &moduleState{specifier: specifier, value: rawModule}
	return Module{
		call: func(ctx context.Context, export string, args ...any) (any, error) {
			raw, err := state.export(export)
			if err != nil {
				return nil, err
			}
			if raw.Type() != js.TypeFunction {
				return nil, wrapError("Module.Call", export, CodeNotFunction, errors.New("export is not a function"))
			}
			jsArgs := make([]interface{}, 0, len(args))
			for _, arg := range args {
				value, err := goValueToJS("Module.Call", export, arg)
				if err != nil {
					return nil, err
				}
				jsArgs = append(jsArgs, value)
			}
			result, err := awaitValue(ctx, "Module.Call", export, raw.Invoke(jsArgs...))
			if err != nil {
				return nil, err
			}
			return jsValueToGo("Module.Call", export, result)
		},
		callDefault: func(ctx context.Context, args ...any) (any, error) {
			return state.callDefault(ctx, args...)
		},
		value: func(ctx context.Context, export string) (any, error) {
			raw, err := state.export(export)
			if err != nil {
				return nil, err
			}
			resolved, err := awaitValue(ctx, "Module.Value", export, raw)
			if err != nil {
				return nil, err
			}
			return jsValueToGo("Module.Value", export, resolved)
		},
		dispose: state.dispose,
	}, nil
}

type moduleState struct {
	mu        sync.RWMutex
	specifier string
	value     js.Value
	disposed  bool
}

func (m *moduleState) export(name string) (js.Value, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.disposed {
		return js.Undefined(), wrapError("Module", m.specifier, CodeDisposed, errors.New("module handle is disposed"))
	}
	raw := m.value.Get(name)
	if raw.IsUndefined() || raw.IsNull() {
		return js.Undefined(), wrapError("Module.Value", name, CodeMissingExport, errors.New("export not found"))
	}
	return raw, nil
}

func (m *moduleState) callDefault(ctx context.Context, args ...any) (any, error) {
	raw, err := m.export("default")
	if err != nil {
		return nil, err
	}
	if raw.Type() != js.TypeFunction {
		return nil, wrapError("Module.CallDefault", "default", CodeNotFunction, errors.New("default export is not a function"))
	}
	jsArgs := make([]interface{}, 0, len(args))
	for _, arg := range args {
		value, err := goValueToJS("Module.CallDefault", "default", arg)
		if err != nil {
			return nil, err
		}
		jsArgs = append(jsArgs, value)
	}
	result, err := awaitValue(ctx, "Module.CallDefault", "default", raw.Invoke(jsArgs...))
	if err != nil {
		return nil, err
	}
	return jsValueToGo("Module.CallDefault", "default", result)
}

func (m *moduleState) dispose() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disposed = true
	m.value = js.Undefined()
	return nil
}

func newEventTarget(name string, raw js.Value) EventTarget {
	return EventTarget{
		dispatch: func(eventName string, detail any) error {
			customEventCtor := js.Global().Get("CustomEvent")
			if customEventCtor.IsUndefined() || customEventCtor.IsNull() {
				return unavailable("EventTarget.Dispatch", name)
			}
			detailValue, err := goValueToJS("EventTarget.Dispatch", eventName, detail)
			if err != nil {
				return err
			}
			init := js.Global().Get("Object").New()
			init.Set("detail", detailValue)
			raw.Call("dispatchEvent", customEventCtor.New(eventName, init))
			return nil
		},
		subscribe: func(eventName string, handler func(CustomEvent)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("EventTarget.Subscribe", eventName, CodeInvalid, errors.New("handler is nil"))
			}
			listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				if len(args) == 0 {
					handler(CustomEvent{Type: eventName})
					return nil
				}
				detail, _ := jsValueToGo("EventTarget.Subscribe", eventName, args[0].Get("detail"))
				handler(CustomEvent{
					Type:   args[0].Get("type").String(),
					Detail: detail,
				})
				return nil
			})
			raw.Call("addEventListener", eventName, listener)
			return Subscription{cancel: func() {
				raw.Call("removeEventListener", eventName, listener)
				listener.Release()
			}}, nil
		},
	}
}

func globalProperty(op string, name string) (js.Value, error) {
	value := js.Global().Get(name)
	if (value.IsUndefined() || value.IsNull()) && name != "window" {
		if window := js.Global().Get("window"); !window.IsUndefined() && !window.IsNull() {
			value = window.Get(name)
		}
	}
	if value.IsUndefined() || value.IsNull() {
		return js.Undefined(), unavailable(op, name)
	}
	return value, nil
}

func globalPath(op string, head string, tail string) (js.Value, error) {
	root, err := globalProperty(op, head)
	if err != nil {
		return js.Undefined(), err
	}
	value := root.Get(tail)
	if (value.IsUndefined() || value.IsNull()) && head != "window" {
		if window := js.Global().Get("window"); !window.IsUndefined() && !window.IsNull() {
			nextRoot := window.Get(head)
			if !nextRoot.IsUndefined() && !nextRoot.IsNull() {
				value = nextRoot.Get(tail)
			}
		}
	}
	if value.IsUndefined() || value.IsNull() {
		return js.Undefined(), unavailable(op, head+"."+tail)
	}
	return value, nil
}

func durationMS(value time.Duration) int {
	if value <= 0 {
		return 0
	}
	return int(value / time.Millisecond)
}

func awaitValue(ctx context.Context, op string, target string, value js.Value) (js.Value, error) {
	if !isPromise(value) {
		return value, nil
	}
	resolvedCh := make(chan js.Value, 1)
	rejectedCh := make(chan js.Value, 1)
	var (
		resolve js.Func
		reject  js.Func
		once    sync.Once
	)
	cleanup := func() {
		resolve.Release()
		reject.Release()
	}
	resolve = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		once.Do(func() {
			if len(args) > 0 {
				resolvedCh <- args[0]
			} else {
				resolvedCh <- js.Undefined()
			}
			cleanup()
		})
		return nil
	})
	reject = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		once.Do(func() {
			if len(args) > 0 {
				rejectedCh <- args[0]
			} else {
				rejectedCh <- js.ValueOf("promise rejected")
			}
			cleanup()
		})
		return nil
	})
	value.Call("then", resolve)
	value.Call("catch", reject)

	select {
	case resolved := <-resolvedCh:
		return resolved, nil
	case rejected := <-rejectedCh:
		return js.Undefined(), wrapError(op, target, CodePromiseRejected, errors.New(jsValueSummary(rejected)))
	case <-ctx.Done():
		return js.Undefined(), wrapError(op, target, CodePromiseRejected, ctx.Err())
	}
}

func isPromise(value js.Value) bool {
	if value.IsUndefined() || value.IsNull() {
		return false
	}
	if value.Type() != js.TypeObject && value.Type() != js.TypeFunction {
		return false
	}
	then := value.Get("then")
	return then.Type() == js.TypeFunction
}

func goValueToJS(op string, target string, value any) (interface{}, error) {
	switch typed := value.(type) {
	case nil:
		return js.Null(), nil
	case js.Value:
		return typed, nil
	case bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return js.ValueOf(typed), nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, wrapError(op, target, CodeEncode, err)
		}
		return js.Global().Get("JSON").Call("parse", string(data)), nil
	}
}

func jsValueToGo(op string, target string, value js.Value) (any, error) {
	if value.IsUndefined() || value.IsNull() {
		return nil, nil
	}
	switch value.Type() {
	case js.TypeBoolean:
		return value.Bool(), nil
	case js.TypeString:
		return value.String(), nil
	case js.TypeNumber:
		return value.Float(), nil
	case js.TypeObject:
		stringified := js.Global().Get("JSON").Call("stringify", value)
		if stringified.IsUndefined() || stringified.IsNull() {
			return nil, wrapError(op, target, CodeDecode, errors.New("value is not serializable"))
		}
		var decoded any
		if err := json.Unmarshal([]byte(stringified.String()), &decoded); err != nil {
			return nil, wrapError(op, target, CodeDecode, err)
		}
		return decoded, nil
	case js.TypeFunction:
		return nil, wrapError(op, target, CodeDecode, errors.New("function values are not serializable"))
	default:
		return value.String(), nil
	}
}

func jsValueSummary(value js.Value) string {
	if value.IsUndefined() {
		return "undefined"
	}
	if value.IsNull() {
		return "null"
	}
	switch value.Type() {
	case js.TypeString:
		return value.String()
	case js.TypeBoolean:
		if value.Bool() {
			return "true"
		}
		return "false"
	case js.TypeNumber:
		return fmt.Sprint(value.Float())
	default:
		stringified := js.Global().Get("JSON").Call("stringify", value)
		if stringified.IsUndefined() || stringified.IsNull() {
			return value.String()
		}
		return stringified.String()
	}
}
