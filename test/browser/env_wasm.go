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

func Install(parseTb testing.TB, parseOptions ...Options) *Environment {
	parseTb.Helper()
	parseResolved := Options{}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}

	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseEnv := &Environment{
		tb:                   parseTb,
		prevWindow:           parseGlobal.Get("window"),
		prevLocation:         parseGlobal.Get("location"),
		prevHistory:          parseGlobal.Get("history"),
		prevLocalStorage:     parseGlobal.Get("localStorage"),
		prevSessionStorage:   parseGlobal.Get("sessionStorage"),
		prevBroadcastChannel: parseGlobal.Get("BroadcastChannel"),
		prevWorker:           parseGlobal.Get("Worker"),
		window:               parseObjectCtor.New(),
		location:             parseObjectCtor.New(),
		history:              parseObjectCtor.New(),
		localStorage:         parseObjectCtor.New(),
		sessionStorage:       parseObjectCtor.New(),
		localValues:          cloneStringMap(parseResolved.LocalStorage),
		sessionValues:        cloneStringMap(parseResolved.SessionStorage),
		mediaMatches:         cloneBoolMap(parseResolved.MediaMatches),
		channels:             map[string][]*mockBroadcastChannel{},
	}

	parseEnv.installLocation(parseObjectCtor)
	parseEnv.installHistory()
	parseEnv.installStorage(parseEnv.localStorage, parseEnv.localValues)
	parseEnv.installStorage(parseEnv.sessionStorage, parseEnv.sessionValues)
	parseEnv.installWindow(parseObjectCtor)
	parseEnv.installBroadcastChannel()
	parseEnv.installWorker()
	parseEnv.SetPath(parseResolved.Path, parseResolved.HashRouting)

	parseGlobal.Set("window", parseEnv.window)
	parseGlobal.Set("location", parseEnv.location)
	parseGlobal.Set("history", parseEnv.history)
	parseGlobal.Set("localStorage", parseEnv.localStorage)
	parseGlobal.Set("sessionStorage", parseEnv.sessionStorage)
	parseGlobal.Set("BroadcastChannel", parseEnv.window.Get("BroadcastChannel"))
	parseGlobal.Set("Worker", parseEnv.window.Get("Worker"))

	parseTb.Cleanup(func() {
		parseEnv.Restore()
	})
	return parseEnv
}

func (parseE *Environment) Window() js.Value {
	if parseE == nil {
		return js.Undefined()
	}
	return parseE.window
}

func (parseE *Environment) SetPath(parsePath string, isHashRouting bool) {
	if parseE == nil {
		return
	}
	parseTrimmed := strings.TrimSpace(parsePath)
	if parseTrimmed == "" {
		parseTrimmed = "/"
	}
	if isHashRouting {
		parseE.location.Set("hash", "#"+strings.TrimPrefix(parseTrimmed, "#"))
		return
	}
	if !strings.HasPrefix(parseTrimmed, "/") {
		parseTrimmed = "/" + strings.TrimPrefix(parseTrimmed, "#")
	}
	if parseIndex := strings.Index(parseTrimmed, "?"); parseIndex >= 0 {
		parseE.location.Set("pathname", parseTrimmed[:parseIndex])
		parseE.location.Set("search", parseTrimmed[parseIndex:])
		return
	}
	parseE.location.Set("pathname", parseTrimmed)
	parseE.location.Set("search", "")
}

func (parseE *Environment) SetLocalStorage(parseKey, parseValue string) {
	if parseE == nil {
		return
	}
	parseE.localValues[parseKey] = parseValue
}

func (parseE *Environment) SetSessionStorage(parseKey, parseValue string) {
	if parseE == nil {
		return
	}
	parseE.sessionValues[parseKey] = parseValue
}

func (parseE *Environment) LocalStorageSnapshot() map[string]string {
	if parseE == nil {
		return nil
	}
	return cloneStringMap(parseE.localValues)
}

func (parseE *Environment) SessionStorageSnapshot() map[string]string {
	if parseE == nil {
		return nil
	}
	return cloneStringMap(parseE.sessionValues)
}

func (parseE *Environment) SetMediaMatch(parseQuery string, isMatches bool) {
	if parseE == nil {
		return
	}
	parseE.mediaMatches[strings.TrimSpace(parseQuery)] = isMatches
}

func (parseE *Environment) Workers() []*MockWorker {
	if parseE == nil {
		return nil
	}
	parseE.mu.RLock()
	defer parseE.mu.RUnlock()
	return append([]*MockWorker(nil), parseE.workers...)
}

func (parseE *Environment) OpenCalls() []OpenCall {
	if parseE == nil {
		return nil
	}
	parseE.mu.RLock()
	defer parseE.mu.RUnlock()
	return append([]OpenCall(nil), parseE.openCalls...)
}

func (parseE *Environment) OpenedWindows() []*MockWindow {
	if parseE == nil {
		return nil
	}
	parseE.mu.RLock()
	defer parseE.mu.RUnlock()
	return append([]*MockWindow(nil), parseE.openedWindows...)
}

func (parseE *Environment) BroadcastMessages() []BroadcastMessage {
	if parseE == nil {
		return nil
	}
	parseE.mu.RLock()
	defer parseE.mu.RUnlock()
	return append([]BroadcastMessage(nil), parseE.broadcastLog...)
}

func (parseE *Environment) SetOpener(parseWindow *MockWindow) {
	if parseE == nil {
		return
	}
	parseE.defaultOpener = parseWindow
	if parseWindow == nil {
		parseE.window.Set("opener", js.Null())
		return
	}
	parseE.window.Set("opener", parseWindow.raw)
}

func (parseE *Environment) Restore() {
	if parseE == nil {
		return
	}
	parseGlobal := js.Global()
	setOrDelete(parseGlobal, "window", parseE.prevWindow)
	setOrDelete(parseGlobal, "location", parseE.prevLocation)
	setOrDelete(parseGlobal, "history", parseE.prevHistory)
	setOrDelete(parseGlobal, "localStorage", parseE.prevLocalStorage)
	setOrDelete(parseGlobal, "sessionStorage", parseE.prevSessionStorage)
	setOrDelete(parseGlobal, "BroadcastChannel", parseE.prevBroadcastChannel)
	setOrDelete(parseGlobal, "Worker", parseE.prevWorker)
	for _, parseFn := range parseE.release {
		parseFn.Release()
	}
	parseE.release = nil
}

func (parseE *Environment) installLocation(parseObjectCtor js.Value) {
	parseE.location.Set("hash", "#/")
	parseE.location.Set("pathname", "/")
	parseE.location.Set("search", "")
	parseReplaceFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 {
			parseE.SetPath(parseArgs[0].String(), strings.HasPrefix(parseArgs[0].String(), "#"))
		}
		return nil
	})
	parseE.release = append(parseE.release, parseReplaceFn)
	parseE.location.Set("replace", parseReplaceFn)
}

func (parseE *Environment) installHistory() {
	parsePushStateFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 3 {
			parseE.SetPath(parseArgs[2].String(), false)
		}
		return nil
	})
	parseReplaceStateFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if len(parseArgs2) >= 3 {
			parseE.SetPath(parseArgs2[2].String(), false)
		}
		return nil
	})
	parseE.release = append(parseE.release, parsePushStateFn, parseReplaceStateFn)
	parseE.history.Set("pushState", parsePushStateFn)
	parseE.history.Set("replaceState", parseReplaceStateFn)
}

func (parseE *Environment) installStorage(parseTarget js.Value, parseValues map[string]string) {
	getItem := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return js.Null()
		}
		if parseValue, parseOk := parseValues[parseArgs[0].String()]; parseOk {
			return parseValue
		}
		return js.Null()
	})
	setItem := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if len(parseArgs2) >= 2 {
			parseValues[parseArgs2[0].String()] = parseArgs2[1].String()
		}
		return nil
	})
	parseRemoveItem := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		if len(parseArgs3) > 0 {
			delete(parseValues, parseArgs3[0].String())
		}
		return nil
	})
	clearFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		for parseKey := range parseValues {
			delete(parseValues, parseKey)
		}
		return nil
	})
	parseKeyFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		if len(parseArgs5) == 0 {
			return js.Null()
		}
		parseIndex := parseArgs5[0].Int()
		if parseIndex < 0 || parseIndex >= len(parseValues) {
			return js.Null()
		}
		parseI := 0
		for parseKey2 := range parseValues {
			if parseI == parseIndex {
				return parseKey2
			}
			parseI++
		}
		return js.Null()
	})
	parseE.release = append(parseE.release, getItem, setItem, parseRemoveItem, clearFn, parseKeyFn)
	parseTarget.Set("getItem", getItem)
	parseTarget.Set("setItem", setItem)
	parseTarget.Set("removeItem", parseRemoveItem)
	parseTarget.Set("clear", clearFn)
	parseTarget.Set("key", parseKeyFn)
}

func (parseE *Environment) installWindow(parseObjectCtor js.Value) {
	parseAddEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} { return nil })
	parseRemoveEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil })
	parseMatchMedia := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseQuery := ""
		if len(parseArgs3) > 0 {
			parseQuery = strings.TrimSpace(parseArgs3[0].String())
		}
		parseResult := parseObjectCtor.New()
		parseResult.Set("media", parseQuery)
		parseResult.Set("matches", parseE.mediaMatches[parseQuery])
		parseResult.Set("addListener", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))
		parseResult.Set("removeListener", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil }))
		parseResult.Set("addEventListener", js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil }))
		parseResult.Set("removeEventListener", js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} { return nil }))
		return parseResult
	})
	parseOpenFn := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
		parseUrl := ""
		parseName := ""
		parseFeatures := ""
		if len(parseArgs8) > 0 {
			parseUrl = parseArgs8[0].String()
		}
		if len(parseArgs8) > 1 {
			parseName = parseArgs8[1].String()
		}
		if len(parseArgs8) > 2 {
			parseFeatures = parseArgs8[2].String()
		}
		parseWindow := parseE.newMockWindow(parseName)
		parseE.mu.Lock()
		parseE.openCalls = append(parseE.openCalls, OpenCall{URL: parseUrl, Name: parseName, Features: parseFeatures})
		parseE.openedWindows = append(parseE.openedWindows, parseWindow)
		parseE.mu.Unlock()
		return parseWindow.raw
	})
	parseE.release = append(parseE.release, parseAddEventListener, parseRemoveEventListener, parseMatchMedia, parseOpenFn)
	parseE.window.Set("addEventListener", parseAddEventListener)
	parseE.window.Set("removeEventListener", parseRemoveEventListener)
	parseE.window.Set("location", parseE.location)
	parseE.window.Set("history", parseE.history)
	parseE.window.Set("localStorage", parseE.localStorage)
	parseE.window.Set("sessionStorage", parseE.sessionStorage)
	parseE.window.Set("matchMedia", parseMatchMedia)
	parseE.window.Set("open", parseOpenFn)
	parseE.window.Set("opener", js.Null())
}

func (parseE *Environment) installBroadcastChannel() {
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseName := ""
		if len(parseArgs) > 0 {
			parseName = parseArgs[0].String()
		}
		parseChannel := parseE.newBroadcastChannel(parseName)
		parseE.mu.Lock()
		parseE.channels[parseName] = append(parseE.channels[parseName], parseChannel)
		parseE.mu.Unlock()
		return parseChannel.raw
	})
	parseE.release = append(parseE.release, parseCtor)
	parseE.window.Set("BroadcastChannel", parseCtor)
}

func (parseE *Environment) installWorker() {
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseUrl := ""
		parseName := ""
		parseKind := ""
		if len(parseArgs) > 0 {
			parseUrl = parseArgs[0].String()
		}
		if len(parseArgs) > 1 {
			parseInit := parseArgs[1]
			if !parseInit.IsUndefined() && !parseInit.IsNull() {
				parseName = parseInit.Get("name").String()
				parseKind = parseInit.Get("type").String()
			}
		}
		parseWorker := parseE.newMockWorker(parseUrl, parseName, parseKind)
		parseE.mu.Lock()
		parseE.workers = append(parseE.workers, parseWorker)
		parseE.mu.Unlock()
		return parseWorker.raw
	})
	parseE.release = append(parseE.release, parseCtor)
	parseE.window.Set("Worker", parseCtor)
}

func (parseE *Environment) newMockWorker(parseUrl, parseName, parseKind string) *MockWorker {
	parseRaw := js.Global().Get("Object").New()
	parseWorker := &MockWorker{
		env:       parseE,
		raw:       parseRaw,
		url:       parseUrl,
		name:      parseName,
		kind:      parseKind,
		listeners: map[string][]js.Value{},
	}
	parsePostMessage := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 {
			parseWorker.messages = append(parseWorker.messages, jsValueToAny(parseArgs[0]))
		}
		return nil
	})
	parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if len(parseArgs2) >= 2 {
			parseWorker.listeners[parseArgs2[0].String()] = append(parseWorker.listeners[parseArgs2[0].String()], parseArgs2[1])
		}
		return nil
	})
	parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
	parseTerminate := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil })
	parseE.release = append(parseE.release, parsePostMessage, parseAddEventListener, parseRemoveEventListener, parseTerminate)
	parseRaw.Set("postMessage", parsePostMessage)
	parseRaw.Set("addEventListener", parseAddEventListener)
	parseRaw.Set("removeEventListener", parseRemoveEventListener)
	parseRaw.Set("terminate", parseTerminate)
	return parseWorker
}

func (parseW *MockWorker) URL() string {
	if parseW == nil {
		return ""
	}
	return parseW.url
}

func (parseW *MockWorker) PostedMessages() []any {
	if parseW == nil {
		return nil
	}
	return append([]any(nil), parseW.messages...)
}

func (parseW *MockWorker) EmitMessage(parseValue any) {
	if parseW == nil {
		return
	}
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("data", parseValue)
	for _, parseListener := range parseW.listeners["message"] {
		parseListener.Invoke(parseEvent)
	}
}

func (parseW *MockWorker) EmitError(parseMessage string) {
	if parseW == nil {
		return
	}
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("message", parseMessage)
	for _, parseListener := range parseW.listeners["error"] {
		parseListener.Invoke(parseEvent)
	}
}

func (parseE *Environment) newBroadcastChannel(parseName string) *mockBroadcastChannel {
	parseRaw := js.Global().Get("Object").New()
	parseChannel := &mockBroadcastChannel{
		env:       parseE,
		raw:       parseRaw,
		name:      parseName,
		listeners: map[string][]js.Value{},
	}
	parsePostMessage := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseValue := jsValueToAny(parseArgs[0])
		parseChannel.messages = append(parseChannel.messages, parseValue)
		parseE.mu.Lock()
		parseE.broadcastLog = append(parseE.broadcastLog, BroadcastMessage{Channel: parseName, Data: parseValue})
		parseTargets := append([]*mockBroadcastChannel(nil), parseE.channels[parseName]...)
		parseE.mu.Unlock()
		parseEvent := js.Global().Get("Object").New()
		parseEvent.Set("data", parseArgs[0])
		for _, parseTarget := range parseTargets {
			if parseTarget.closed {
				continue
			}
			for _, parseListener := range parseTarget.listeners["message"] {
				parseListener.Invoke(parseEvent)
			}
		}
		return nil
	})
	parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if len(parseArgs2) >= 2 {
			parseChannel.listeners[parseArgs2[0].String()] = append(parseChannel.listeners[parseArgs2[0].String()], parseArgs2[1])
		}
		return nil
	})
	parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
	parseCloseFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseChannel.closed = true
		return nil
	})
	parseE.release = append(parseE.release, parsePostMessage, parseAddEventListener, parseRemoveEventListener, parseCloseFn)
	parseRaw.Set("name", parseName)
	parseRaw.Set("postMessage", parsePostMessage)
	parseRaw.Set("addEventListener", parseAddEventListener)
	parseRaw.Set("removeEventListener", parseRemoveEventListener)
	parseRaw.Set("close", parseCloseFn)
	return parseChannel
}

func (parseE *Environment) newMockWindow(parseName string) *MockWindow {
	parseRaw := js.Global().Get("Object").New()
	parseWindow := &MockWindow{
		env:       parseE,
		raw:       parseRaw,
		name:      parseName,
		listeners: map[string][]js.Value{},
	}
	parsePostMessage := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 {
			parseWindow.messages = append(parseWindow.messages, jsValueToAny(parseArgs[0]))
		}
		return nil
	})
	parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if len(parseArgs2) >= 2 {
			parseWindow.listeners[parseArgs2[0].String()] = append(parseWindow.listeners[parseArgs2[0].String()], parseArgs2[1])
		}
		return nil
	})
	parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
	parseCloseFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseWindow.closed = true
		parseRaw.Set("closed", true)
		return nil
	})
	parseE.release = append(parseE.release, parsePostMessage, parseAddEventListener, parseRemoveEventListener, parseCloseFn)
	parseRaw.Set("postMessage", parsePostMessage)
	parseRaw.Set("addEventListener", parseAddEventListener)
	parseRaw.Set("removeEventListener", parseRemoveEventListener)
	parseRaw.Set("close", parseCloseFn)
	parseRaw.Set("closed", false)
	return parseWindow
}

func (parseW *MockWindow) PostedMessages() []any {
	if parseW == nil {
		return nil
	}
	return append([]any(nil), parseW.messages...)
}

func (parseW *MockWindow) EmitMessage(parseData any, parseOrigin string) {
	if parseW == nil {
		return
	}
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("data", parseData)
	parseEvent.Set("origin", parseOrigin)
	for _, parseListener := range parseW.listeners["message"] {
		parseListener.Invoke(parseEvent)
	}
}

func cloneStringMap(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return map[string]string{}
	}
	parseCloned := make(map[string]string, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}

func cloneBoolMap(parseValues map[string]bool) map[string]bool {
	if len(parseValues) == 0 {
		return map[string]bool{}
	}
	parseCloned := make(map[string]bool, len(parseValues))
	for parseKey, parseValue := range parseValues {
		parseCloned[parseKey] = parseValue
	}
	return parseCloned
}

func jsValueToAny(parseValue js.Value) any {
	switch parseValue.Type() {
	case js.TypeString:
		return parseValue.String()
	case js.TypeBoolean:
		return parseValue.Bool()
	case js.TypeNumber:
		return parseValue.Float()
	default:
		return parseValue
	}
}

func setOrDelete(parseTarget js.Value, parseProperty string, parseValue js.Value) {
	if parseValue.IsUndefined() {
		parseTarget.Delete(parseProperty)
		return
	}
	parseTarget.Set(parseProperty, parseValue)
}
