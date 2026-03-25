//go:build js && wasm
// +build js,wasm

package browser

import (
	"strings"
	"sync"
	"syscall/js"
	"testing"
)

type Options struct {
	Path           string
	HashRouting    bool
	LocalStorage   map[string]string
	SessionStorage map[string]string
	MediaMatches   map[string]bool
}

type Environment struct {
	tb testing.TB

	prevWindow           js.Value
	prevLocation         js.Value
	prevHistory          js.Value
	prevLocalStorage     js.Value
	prevSessionStorage   js.Value
	prevBroadcastChannel js.Value
	prevWorker           js.Value

	window         js.Value
	location       js.Value
	history        js.Value
	localStorage   js.Value
	sessionStorage js.Value

	localValues   map[string]string
	sessionValues map[string]string
	mediaMatches  map[string]bool

	release []js.Func

	mu            sync.RWMutex
	workers       []*MockWorker
	channels      map[string][]*mockBroadcastChannel
	openCalls     []OpenCall
	openedWindows []*MockWindow
	defaultOpener *MockWindow
	broadcastLog  []BroadcastMessage
}

type OpenCall struct {
	URL      string
	Name     string
	Features string
}

type BroadcastMessage struct {
	Channel string
	Data    any
}

type MockWorker struct {
	env       *Environment
	raw       js.Value
	url       string
	name      string
	kind      string
	messages  []any
	listeners map[string][]js.Value
}

type MockWindow struct {
	env       *Environment
	raw       js.Value
	name      string
	closed    bool
	messages  []any
	listeners map[string][]js.Value
}

type mockBroadcastChannel struct {
	env       *Environment
	raw       js.Value
	name      string
	closed    bool
	messages  []any
	listeners map[string][]js.Value
}

func Install(tb testing.TB, options ...Options) *Environment {
	tb.Helper()
	resolved := Options{}
	if len(options) > 0 {
		resolved = options[0]
	}

	global := js.Global()
	objectCtor := global.Get("Object")
	env := &Environment{
		tb:                   tb,
		prevWindow:           global.Get("window"),
		prevLocation:         global.Get("location"),
		prevHistory:          global.Get("history"),
		prevLocalStorage:     global.Get("localStorage"),
		prevSessionStorage:   global.Get("sessionStorage"),
		prevBroadcastChannel: global.Get("BroadcastChannel"),
		prevWorker:           global.Get("Worker"),
		window:               objectCtor.New(),
		location:             objectCtor.New(),
		history:              objectCtor.New(),
		localStorage:         objectCtor.New(),
		sessionStorage:       objectCtor.New(),
		localValues:          cloneStringMap(resolved.LocalStorage),
		sessionValues:        cloneStringMap(resolved.SessionStorage),
		mediaMatches:         cloneBoolMap(resolved.MediaMatches),
		channels:             map[string][]*mockBroadcastChannel{},
	}

	env.installLocation(objectCtor)
	env.installHistory()
	env.installStorage(env.localStorage, env.localValues)
	env.installStorage(env.sessionStorage, env.sessionValues)
	env.installWindow(objectCtor)
	env.installBroadcastChannel()
	env.installWorker()
	env.SetPath(resolved.Path, resolved.HashRouting)

	global.Set("window", env.window)
	global.Set("location", env.location)
	global.Set("history", env.history)
	global.Set("localStorage", env.localStorage)
	global.Set("sessionStorage", env.sessionStorage)
	global.Set("BroadcastChannel", env.window.Get("BroadcastChannel"))
	global.Set("Worker", env.window.Get("Worker"))

	tb.Cleanup(func() {
		env.Restore()
	})
	return env
}

func (e *Environment) Window() js.Value {
	if e == nil {
		return js.Undefined()
	}
	return e.window
}

func (e *Environment) SetPath(path string, hashRouting bool) {
	if e == nil {
		return
	}
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = "/"
	}
	if hashRouting {
		e.location.Set("hash", "#"+strings.TrimPrefix(trimmed, "#"))
		return
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + strings.TrimPrefix(trimmed, "#")
	}
	if index := strings.Index(trimmed, "?"); index >= 0 {
		e.location.Set("pathname", trimmed[:index])
		e.location.Set("search", trimmed[index:])
		return
	}
	e.location.Set("pathname", trimmed)
	e.location.Set("search", "")
}

func (e *Environment) SetLocalStorage(key, value string) {
	if e == nil {
		return
	}
	e.localValues[key] = value
}

func (e *Environment) SetSessionStorage(key, value string) {
	if e == nil {
		return
	}
	e.sessionValues[key] = value
}

func (e *Environment) LocalStorageSnapshot() map[string]string {
	if e == nil {
		return nil
	}
	return cloneStringMap(e.localValues)
}

func (e *Environment) SessionStorageSnapshot() map[string]string {
	if e == nil {
		return nil
	}
	return cloneStringMap(e.sessionValues)
}

func (e *Environment) SetMediaMatch(query string, matches bool) {
	if e == nil {
		return
	}
	e.mediaMatches[strings.TrimSpace(query)] = matches
}

func (e *Environment) Workers() []*MockWorker {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]*MockWorker(nil), e.workers...)
}

func (e *Environment) OpenCalls() []OpenCall {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]OpenCall(nil), e.openCalls...)
}

func (e *Environment) OpenedWindows() []*MockWindow {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]*MockWindow(nil), e.openedWindows...)
}

func (e *Environment) BroadcastMessages() []BroadcastMessage {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]BroadcastMessage(nil), e.broadcastLog...)
}

func (e *Environment) SetOpener(window *MockWindow) {
	if e == nil {
		return
	}
	e.defaultOpener = window
	if window == nil {
		e.window.Set("opener", js.Null())
		return
	}
	e.window.Set("opener", window.raw)
}

func (e *Environment) Restore() {
	if e == nil {
		return
	}
	global := js.Global()
	setOrDelete(global, "window", e.prevWindow)
	setOrDelete(global, "location", e.prevLocation)
	setOrDelete(global, "history", e.prevHistory)
	setOrDelete(global, "localStorage", e.prevLocalStorage)
	setOrDelete(global, "sessionStorage", e.prevSessionStorage)
	setOrDelete(global, "BroadcastChannel", e.prevBroadcastChannel)
	setOrDelete(global, "Worker", e.prevWorker)
	for _, fn := range e.release {
		fn.Release()
	}
	e.release = nil
}

func (e *Environment) installLocation(objectCtor js.Value) {
	e.location.Set("hash", "#/")
	e.location.Set("pathname", "/")
	e.location.Set("search", "")
	replaceFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			e.SetPath(args[0].String(), strings.HasPrefix(args[0].String(), "#"))
		}
		return nil
	})
	e.release = append(e.release, replaceFn)
	e.location.Set("replace", replaceFn)
}

func (e *Environment) installHistory() {
	pushStateFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 3 {
			e.SetPath(args[2].String(), false)
		}
		return nil
	})
	replaceStateFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 3 {
			e.SetPath(args[2].String(), false)
		}
		return nil
	})
	e.release = append(e.release, pushStateFn, replaceStateFn)
	e.history.Set("pushState", pushStateFn)
	e.history.Set("replaceState", replaceStateFn)
}

func (e *Environment) installStorage(target js.Value, values map[string]string) {
	getItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return js.Null()
		}
		if value, ok := values[args[0].String()]; ok {
			return value
		}
		return js.Null()
	})
	setItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			values[args[0].String()] = args[1].String()
		}
		return nil
	})
	removeItem := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			delete(values, args[0].String())
		}
		return nil
	})
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		for key := range values {
			delete(values, key)
		}
		return nil
	})
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return js.Null()
		}
		index := args[0].Int()
		if index < 0 || index >= len(values) {
			return js.Null()
		}
		i := 0
		for key := range values {
			if i == index {
				return key
			}
			i++
		}
		return js.Null()
	})
	e.release = append(e.release, getItem, setItem, removeItem, clearFn, keyFn)
	target.Set("getItem", getItem)
	target.Set("setItem", setItem)
	target.Set("removeItem", removeItem)
	target.Set("clear", clearFn)
	target.Set("key", keyFn)
}

func (e *Environment) installWindow(objectCtor js.Value) {
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	matchMedia := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		query := ""
		if len(args) > 0 {
			query = strings.TrimSpace(args[0].String())
		}
		result := objectCtor.New()
		result.Set("media", query)
		result.Set("matches", e.mediaMatches[query])
		result.Set("addListener", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		result.Set("removeListener", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		result.Set("addEventListener", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		result.Set("removeEventListener", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		return result
	})
	openFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		url := ""
		name := ""
		features := ""
		if len(args) > 0 {
			url = args[0].String()
		}
		if len(args) > 1 {
			name = args[1].String()
		}
		if len(args) > 2 {
			features = args[2].String()
		}
		window := e.newMockWindow(name)
		e.mu.Lock()
		e.openCalls = append(e.openCalls, OpenCall{URL: url, Name: name, Features: features})
		e.openedWindows = append(e.openedWindows, window)
		e.mu.Unlock()
		return window.raw
	})
	e.release = append(e.release, addEventListener, removeEventListener, matchMedia, openFn)
	e.window.Set("addEventListener", addEventListener)
	e.window.Set("removeEventListener", removeEventListener)
	e.window.Set("location", e.location)
	e.window.Set("history", e.history)
	e.window.Set("localStorage", e.localStorage)
	e.window.Set("sessionStorage", e.sessionStorage)
	e.window.Set("matchMedia", matchMedia)
	e.window.Set("open", openFn)
	e.window.Set("opener", js.Null())
}

func (e *Environment) installBroadcastChannel() {
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		name := ""
		if len(args) > 0 {
			name = args[0].String()
		}
		channel := e.newBroadcastChannel(name)
		e.mu.Lock()
		e.channels[name] = append(e.channels[name], channel)
		e.mu.Unlock()
		return channel.raw
	})
	e.release = append(e.release, ctor)
	e.window.Set("BroadcastChannel", ctor)
}

func (e *Environment) installWorker() {
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		url := ""
		name := ""
		kind := ""
		if len(args) > 0 {
			url = args[0].String()
		}
		if len(args) > 1 {
			init := args[1]
			if !init.IsUndefined() && !init.IsNull() {
				name = init.Get("name").String()
				kind = init.Get("type").String()
			}
		}
		worker := e.newMockWorker(url, name, kind)
		e.mu.Lock()
		e.workers = append(e.workers, worker)
		e.mu.Unlock()
		return worker.raw
	})
	e.release = append(e.release, ctor)
	e.window.Set("Worker", ctor)
}

func (e *Environment) newMockWorker(url, name, kind string) *MockWorker {
	raw := js.Global().Get("Object").New()
	worker := &MockWorker{
		env:       e,
		raw:       raw,
		url:       url,
		name:      name,
		kind:      kind,
		listeners: map[string][]js.Value{},
	}
	postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			worker.messages = append(worker.messages, jsValueToAny(args[0]))
		}
		return nil
	})
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			worker.listeners[args[0].String()] = append(worker.listeners[args[0].String()], args[1])
		}
		return nil
	})
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	terminate := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	e.release = append(e.release, postMessage, addEventListener, removeEventListener, terminate)
	raw.Set("postMessage", postMessage)
	raw.Set("addEventListener", addEventListener)
	raw.Set("removeEventListener", removeEventListener)
	raw.Set("terminate", terminate)
	return worker
}

func (w *MockWorker) URL() string {
	if w == nil {
		return ""
	}
	return w.url
}

func (w *MockWorker) PostedMessages() []any {
	if w == nil {
		return nil
	}
	return append([]any(nil), w.messages...)
}

func (w *MockWorker) EmitMessage(value any) {
	if w == nil {
		return
	}
	event := js.Global().Get("Object").New()
	event.Set("data", value)
	for _, listener := range w.listeners["message"] {
		listener.Invoke(event)
	}
}

func (w *MockWorker) EmitError(message string) {
	if w == nil {
		return
	}
	event := js.Global().Get("Object").New()
	event.Set("message", message)
	for _, listener := range w.listeners["error"] {
		listener.Invoke(event)
	}
}

func (e *Environment) newBroadcastChannel(name string) *mockBroadcastChannel {
	raw := js.Global().Get("Object").New()
	channel := &mockBroadcastChannel{
		env:       e,
		raw:       raw,
		name:      name,
		listeners: map[string][]js.Value{},
	}
	postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		value := jsValueToAny(args[0])
		channel.messages = append(channel.messages, value)
		e.mu.Lock()
		e.broadcastLog = append(e.broadcastLog, BroadcastMessage{Channel: name, Data: value})
		targets := append([]*mockBroadcastChannel(nil), e.channels[name]...)
		e.mu.Unlock()
		event := js.Global().Get("Object").New()
		event.Set("data", args[0])
		for _, target := range targets {
			if target.closed {
				continue
			}
			for _, listener := range target.listeners["message"] {
				listener.Invoke(event)
			}
		}
		return nil
	})
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			channel.listeners[args[0].String()] = append(channel.listeners[args[0].String()], args[1])
		}
		return nil
	})
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	closeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		channel.closed = true
		return nil
	})
	e.release = append(e.release, postMessage, addEventListener, removeEventListener, closeFn)
	raw.Set("name", name)
	raw.Set("postMessage", postMessage)
	raw.Set("addEventListener", addEventListener)
	raw.Set("removeEventListener", removeEventListener)
	raw.Set("close", closeFn)
	return channel
}

func (e *Environment) newMockWindow(name string) *MockWindow {
	raw := js.Global().Get("Object").New()
	window := &MockWindow{
		env:       e,
		raw:       raw,
		name:      name,
		listeners: map[string][]js.Value{},
	}
	postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			window.messages = append(window.messages, jsValueToAny(args[0]))
		}
		return nil
	})
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			window.listeners[args[0].String()] = append(window.listeners[args[0].String()], args[1])
		}
		return nil
	})
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	closeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		window.closed = true
		raw.Set("closed", true)
		return nil
	})
	e.release = append(e.release, postMessage, addEventListener, removeEventListener, closeFn)
	raw.Set("postMessage", postMessage)
	raw.Set("addEventListener", addEventListener)
	raw.Set("removeEventListener", removeEventListener)
	raw.Set("close", closeFn)
	raw.Set("closed", false)
	return window
}

func (w *MockWindow) PostedMessages() []any {
	if w == nil {
		return nil
	}
	return append([]any(nil), w.messages...)
}

func (w *MockWindow) EmitMessage(data any, origin string) {
	if w == nil {
		return
	}
	event := js.Global().Get("Object").New()
	event.Set("data", data)
	event.Set("origin", origin)
	for _, listener := range w.listeners["message"] {
		listener.Invoke(event)
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneBoolMap(values map[string]bool) map[string]bool {
	if len(values) == 0 {
		return map[string]bool{}
	}
	cloned := make(map[string]bool, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func jsValueToAny(value js.Value) any {
	switch value.Type() {
	case js.TypeString:
		return value.String()
	case js.TypeBoolean:
		return value.Bool()
	case js.TypeNumber:
		return value.Float()
	default:
		return value
	}
}

func setOrDelete(target js.Value, property string, value js.Value) {
	if value.IsUndefined() {
		target.Delete(property)
		return
	}
	target.Set(property, value)
}
