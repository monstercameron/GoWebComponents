//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"errors"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// GetLocalStorage returns the browser localStorage wrapper.
func GetLocalStorage() (Storage, error) {
	return resolveStorage("localStorage")
}

// GetSessionStorage returns the browser sessionStorage wrapper.
func GetSessionStorage() (Storage, error) {
	return resolveStorage("sessionStorage")
}

// GetWindowEnv returns a lightweight reader for shared values attached to window.
func GetWindowEnv() (WindowEnv, error) {
	parseRawWindow, parseErr := globalProperty("WindowEnv", "window")
	if parseErr != nil {
		return WindowEnv{}, parseErr
	}
	return WindowEnv{lookup: func(parseName string) (Value, bool) {
		parseKey := strings.TrimSpace(parseName)
		if parseKey == "" {
			return Value{}, false
		}
		parseValue := parseRawWindow.Get(parseKey)
		if parseValue.IsUndefined() || parseValue.IsNull() {
			return Value{}, false
		}
		return Value{raw: parseValue}, true
	}}, nil
}

func resolveStorage(parseName string) (Storage, error) {
	parseRaw, parseErr := globalProperty("Storage", parseName)
	if parseErr != nil {
		return Storage{}, parseErr
	}
	parseGetItem := func(parseKey string) (parseValue string, isParseFound bool, parseErr2 error) {
		defer recoverInteropException("Storage.GetItem", parseName+".getItem", &parseErr2)
		if _, parseErr2 = getStorageMethod(parseRaw, parseName, "getItem", "Storage.GetItem"); parseErr2 != nil {
			return "", false, parseErr2
		}
		parseValue2 := parseRaw.Call("getItem", parseKey)
		if parseValue2.IsUndefined() || parseValue2.IsNull() {
			return "", false, nil
		}
		return parseValue2.String(), true, nil
	}
	parseSetItem := func(parseKey string, parseValue string) (parseErr2 error) {
		defer recoverInteropException("Storage.SetItem", parseName+".setItem", &parseErr2)
		if _, parseErr2 = getStorageMethod(parseRaw, parseName, "setItem", "Storage.SetItem"); parseErr2 != nil {
			return parseErr2
		}
		parseRaw.Call("setItem", parseKey, parseValue)
		return nil
	}
	parseClear := func() (parseErr2 error) {
		defer recoverInteropException("Storage.Clear", parseName+".clear", &parseErr2)
		if _, parseErr2 = getStorageMethod(parseRaw, parseName, "clear", "Storage.Clear"); parseErr2 != nil {
			return parseErr2
		}
		parseRaw.Call("clear")
		return nil
	}
	return Storage{
		getItem: parseGetItem,
		getMany: func(parseKeys []string) (map[string]string, error) {
			parseValues := make(map[string]string, len(parseKeys))
			for _, parseKey := range parseKeys {
				parseValue, isParseFound, parseErr2 := parseGetItem(parseKey)
				if parseErr2 != nil {
					return nil, parseErr2
				}
				if isParseFound {
					parseValues[parseKey] = parseValue
				}
			}
			return parseValues, nil
		},
		setItem: parseSetItem,
		removeItem: func(parseKey3 string) (parseErr2 error) {
			defer recoverInteropException("Storage.RemoveItem", parseName+".removeItem", &parseErr2)
			if _, parseErr2 = getStorageMethod(parseRaw, parseName, "removeItem", "Storage.RemoveItem"); parseErr2 != nil {
				return parseErr2
			}
			parseRaw.Call("removeItem", parseKey3)
			return nil
		},
		clear: parseClear,
		length: func() (int, error) {
			return parseRaw.Get("length").Int(), nil
		},
		key: func(parseIndex int) (parseKey string, isParseFound bool, parseErr2 error) {
			defer recoverInteropException("Storage.Key", parseName+".key", &parseErr2)
			if _, parseErr2 = getStorageMethod(parseRaw, parseName, "key", "Storage.Key"); parseErr2 != nil {
				return "", false, parseErr2
			}
			parseValue2 := parseRaw.Call("key", parseIndex)
			if parseValue2.IsUndefined() || parseValue2.IsNull() {
				return "", false, nil
			}
			return parseValue2.String(), true, nil
		},
	}, nil
}

// getStorageMethod returns one callable storage method or a structured interoperability error.
func getStorageMethod(parseRaw js.Value, parseName string, parseMethodName string, parseOp string) (js.Value, error) {
	parseMethod := parseRaw.Get(parseMethodName)
	if parseMethod.Type() != js.TypeFunction {
		return js.Undefined(), &Error{Op: parseOp, Target: parseName + "." + parseMethodName, Code: CodeNotFunction, Err: errors.New("storage " + parseMethodName + " is not callable")}
	}
	return parseMethod, nil
}

// GetWindowLocation returns a Location backed by the browser window.location object.
func GetWindowLocation() (Location, error) {
	parseRaw, parseErr := globalPath("Location", "window", "location")
	if parseErr != nil {
		return Location{}, parseErr
	}
	return Location{
		href:     func() string { return parseRaw.Get("href").String() },
		pathname: func() string { return parseRaw.Get("pathname").String() },
		search:   func() string { return parseRaw.Get("search").String() },
		hash:     func() string { return parseRaw.Get("hash").String() },
		origin:   func() string { return parseRaw.Get("origin").String() },
		assign: func(parseRawURL string) error {
			parseRaw.Call("assign", parseRawURL)
			return nil
		},
		replace: func(parseRawURL2 string) error {
			parseRaw.Call("replace", parseRawURL2)
			return nil
		},
		reload: func() error {
			parseRaw.Call("reload")
			return nil
		},
	}, nil
}

// GetWindowHistory returns a History backed by the browser window.history object.
func GetWindowHistory() (History, error) {
	parseRaw, parseErr := globalPath("History", "window", "history")
	if parseErr != nil {
		return History{}, parseErr
	}
	return History{
		length: func() (int, error) {
			return parseRaw.Get("length").Int(), nil
		},
		state: func() (any, error) {
			return jsValueToGo("History.State", "history.state", parseRaw.Get("state"))
		},
		back: func() error {
			parseRaw.Call("back")
			return nil
		},
		forward: func() error {
			parseRaw.Call("forward")
			return nil
		},
		goDelta: func(parseDelta int) error {
			parseRaw.Call("go", parseDelta)
			return nil
		},
		pushState: func(parseState any, parseTitle string, parseRawURL string) error {
			parseValue, parseErr2 := goValueToJS("History.PushState", "state", parseState)
			if parseErr2 != nil {
				return parseErr2
			}
			parseRaw.Call("pushState", parseValue, parseTitle, parseRawURL)
			return nil
		},
		replaceState: func(parseState2 any, parseTitle2 string, parseRawURL2 string) error {
			parseValue2, parseErr3 := goValueToJS("History.ReplaceState", "state", parseState2)
			if parseErr3 != nil {
				return parseErr3
			}
			parseRaw.Call("replaceState", parseValue2, parseTitle2, parseRawURL2)
			return nil
		},
	}, nil
}

// GetClipboard returns a Clipboard backed by navigator.clipboard.
func GetClipboard() (Clipboard, error) {
	parseRaw, parseErr := globalPath("Clipboard", "navigator", "clipboard")
	if parseErr != nil {
		return Clipboard{}, parseErr
	}
	parseWrapped := Value{raw: parseRaw}
	return Clipboard{
		writeText: func(parseCtx context.Context, parseText string) error {
			parseValue, parseErr2 := parseWrapped.Call("writeText", parseText)
			if parseErr2 != nil {
				return parseErr2
			}
			parseRawValue, parseOk := parseValue.rawValue()
			if !parseOk {
				return unavailable("Clipboard.WriteText", "navigator.clipboard.writeText")
			}
			_, parseErr2 = awaitValue(parseCtx, "Clipboard.WriteText", "navigator.clipboard.writeText", parseRawValue)
			return parseErr2
		},
		readText: func(parseCtx2 context.Context) (string, error) {
			parseValue, parseErr3 := parseWrapped.Call("readText")
			if parseErr3 != nil {
				return "", parseErr3
			}
			parseRawValue, parseOk := parseValue.rawValue()
			if !parseOk {
				return "", unavailable("Clipboard.ReadText", "navigator.clipboard.readText")
			}
			parseResolved, parseErr3 := awaitValue(parseCtx2, "Clipboard.ReadText", "navigator.clipboard.readText", parseRawValue)
			if parseErr3 != nil {
				return "", parseErr3
			}
			if parseResolved.IsUndefined() || parseResolved.IsNull() {
				return "", nil
			}
			return parseResolved.String(), nil
		},
	}, nil
}

// ScheduleTimeout schedules fn to run once after delay using window.setTimeout.
func ScheduleTimeout(parseDelay time.Duration, parseFn func()) (Timer, error) {
	if parseFn == nil {
		return Timer{}, wrapError("ScheduleTimeout", "", CodeInvalid, errors.New("callback is nil"))
	}
	var (
		parseCallback js.Func
		parseOnce     sync.Once
		parseId       js.Value
	)
	parseCallback = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOnce.Do(func() {
			parseCallback.Release()
		})
		parseFn()
		return nil
	})
	parseId = js.Global().Call("setTimeout", parseCallback, durationMS(parseDelay))
	return Timer{
		cancel: func() error {
			parseOnce.Do(func() {
				js.Global().Call("clearTimeout", parseId)
				parseCallback.Release()
			})
			return nil
		},
	}, nil
}

// ScheduleInterval schedules fn to run repeatedly at interval using window.setInterval.
func ScheduleInterval(parseInterval time.Duration, parseFn func()) (Timer, error) {
	if parseFn == nil {
		return Timer{}, wrapError("ScheduleInterval", "", CodeInvalid, errors.New("callback is nil"))
	}
	var (
		parseCallback js.Func
		parseOnce     sync.Once
		parseId       js.Value
	)
	parseCallback = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseFn()
		return nil
	})
	parseId = js.Global().Call("setInterval", parseCallback, durationMS(parseInterval))
	return Timer{
		cancel: func() error {
			parseOnce.Do(func() {
				js.Global().Call("clearInterval", parseId)
				parseCallback.Release()
			})
			return nil
		},
	}, nil
}

// GetWindowEvents returns an EventTarget for the browser window object.
func GetWindowEvents() (EventTarget, error) {
	parseRaw, parseErr := globalProperty("EventTarget", "window")
	if parseErr != nil {
		return EventTarget{}, parseErr
	}
	return newEventTarget("window", parseRaw), nil
}

// GetDocumentEvents returns an EventTarget for the browser document object.
func GetDocumentEvents() (EventTarget, error) {
	parseRaw, parseErr := globalProperty("EventTarget", "document")
	if parseErr != nil {
		return EventTarget{}, parseErr
	}
	return newEventTarget("document", parseRaw), nil
}

// GetDocument returns a Document backed by the browser document object.
func GetDocument() (Document, error) {
	parseRaw, parseErr := globalProperty("Document", "document")
	if parseErr != nil {
		return Document{}, parseErr
	}
	return Document{
		elementByID: func(parseId2 string) (Element, bool, error) {
			parseValue := parseRaw.Call("getElementById", parseId2)
			if parseValue.IsUndefined() || parseValue.IsNull() {
				return Element{}, false, nil
			}
			return newElement("document.getElementById", parseValue), true, nil
		},
		elementsByID: func(parseIds []string) (map[string]Element, error) {
			parseResolved := make(map[string]Element, len(parseIds))
			for parseId, parseValue2 := range documentElementsByID(parseRaw, parseIds) {
				parseResolved[parseId] = newElement("document.getElementById", parseValue2)
			}
			return parseResolved, nil
		},
		querySelector: func(parseSelector string) (Element, bool, error) {
			parseValue3 := parseRaw.Call("querySelector", parseSelector)
			if parseValue3.IsUndefined() || parseValue3.IsNull() {
				return Element{}, false, nil
			}
			return newElement("document.querySelector", parseValue3), true, nil
		},
	}, nil
}

var (
	storageGetManyHelperOnce sync.Once
	storageGetManyHelper     js.Value
	documentByIDHelperOnce   sync.Once
	documentByIDHelper       js.Value
)

func storageGetMany(parseRaw js.Value, parseKeys []string) map[string]string {
	storageGetManyHelperOnce.Do(func() {
		storageGetManyHelper = js.Global().Get("Function").New("storage", "keys", `
			const result = {};
			for (const key of keys) {
				const value = storage.getItem(key);
				if (value !== null && value !== undefined) {
					result[key] = value;
				}
			}
			return result;
		`)
	})
	parseValues := storageGetManyHelper.Invoke(parseRaw, stringArrayValue(parseKeys))
	if parseValues.IsUndefined() || parseValues.IsNull() {
		return map[string]string{}
	}
	parseResult := make(map[string]string, len(parseKeys))
	parseObjectKeys := js.Global().Get("Object").Call("keys", parseValues)
	for parseIndex := 0; parseIndex < parseObjectKeys.Get("length").Int(); parseIndex++ {
		parseKey := parseObjectKeys.Index(parseIndex).String()
		parseResult[parseKey] = parseValues.Get(parseKey).String()
	}
	return parseResult
}

func documentElementsByID(parseRaw js.Value, parseIds []string) map[string]js.Value {
	documentByIDHelperOnce.Do(func() {
		documentByIDHelper = js.Global().Get("Function").New("documentRef", "ids", `
			const result = {};
			for (const id of ids) {
				const value = documentRef.getElementById(id);
				if (value !== null && value !== undefined) {
					result[id] = value;
				}
			}
			return result;
		`)
	})
	parseValues := documentByIDHelper.Invoke(parseRaw, stringArrayValue(parseIds))
	if parseValues.IsUndefined() || parseValues.IsNull() {
		return map[string]js.Value{}
	}
	parseResult := make(map[string]js.Value, len(parseIds))
	parseObjectKeys := js.Global().Get("Object").Call("keys", parseValues)
	for parseIndex := 0; parseIndex < parseObjectKeys.Get("length").Int(); parseIndex++ {
		parseKey := parseObjectKeys.Index(parseIndex).String()
		parseResult[parseKey] = parseValues.Get(parseKey)
	}
	return parseResult
}

func stringArrayValue(parseValues []string) js.Value {
	parseArray := js.Global().Get("Array").New()
	for _, parseValue := range parseValues {
		parseArray.Call("push", parseValue)
	}
	return parseArray
}

// GetMediaQuery returns a MediaQueryList for the given CSS media query string.
func GetMediaQuery(parseQuery string) (MediaQueryList, error) {
	parseWindow, parseErr := globalProperty("GetMediaQuery", "window")
	if parseErr != nil {
		return MediaQueryList{}, parseErr
	}
	parseRaw := parseWindow.Call("matchMedia", parseQuery)
	if parseRaw.IsUndefined() || parseRaw.IsNull() {
		return MediaQueryList{}, unavailable("GetMediaQuery", parseQuery)
	}
	return MediaQueryList{
		matches: func() bool { return parseRaw.Get("matches").Bool() },
		media:   func() string { return parseRaw.Get("media").String() },
		subscribe: func(handler func(MediaQueryEvent)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("MatchMedia.Subscribe", parseQuery, CodeInvalid, errors.New("handler is nil"))
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				parseEvent := parseRaw
				if len(parseArgs) > 0 && !parseArgs[0].IsUndefined() && !parseArgs[0].IsNull() {
					parseEvent = parseArgs[0]
				}
				handler(MediaQueryEvent{
					Matches: parseEvent.Get("matches").Bool(),
					Media:   parseEvent.Get("media").String(),
				})
				return nil
			})
			if parseAdd := parseRaw.Get("addEventListener"); parseAdd.Type() == js.TypeFunction {
				parseRaw.Call("addEventListener", "change", parseListener)
				return Subscription{cancel: func() {
					parseRaw.Call("removeEventListener", "change", parseListener)
					parseListener.Release()
				}}, nil
			}
			parseRaw.Call("addListener", parseListener)
			return Subscription{cancel: func() {
				parseRaw.Call("removeListener", parseListener)
				parseListener.Release()
			}}, nil
		},
	}, nil
}

// ImportModule dynamically imports a JavaScript ES module by specifier.
func ImportModule(parseCtx context.Context, parseSpecifier string) (Module, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if strings.TrimSpace(parseSpecifier) == "" {
		return Module{}, wrapError("ImportModule", parseSpecifier, CodeInvalid, errors.New("specifier is empty"))
	}
	parseImportFn := js.Global().Get("__gwcImportModule")
	if parseImportFn.IsUndefined() || parseImportFn.IsNull() {
		parseConstructor := js.Global().Get("Function")
		if parseConstructor.IsUndefined() || parseConstructor.IsNull() {
			return Module{}, unavailable("ImportModule", parseSpecifier)
		}
		parseImportFn = parseConstructor.New("specifier", "return import(specifier);")
	}
	parseRawModule, parseErr := awaitValue(parseCtx, "ImportModule", parseSpecifier, parseImportFn.Invoke(parseSpecifier))
	if parseErr != nil {
		return Module{}, parseErr
	}
	parseState := &moduleState{specifier: parseSpecifier, value: parseRawModule}
	return Module{
		call: func(parseCtx2 context.Context, parseExport string, parseArgs ...any) (any, error) {
			parseRaw, parseErr2 := parseState.export(parseExport)
			if parseErr2 != nil {
				return nil, parseErr2
			}
			if parseRaw.Type() != js.TypeFunction {
				return nil, wrapError("Module.Call", parseExport, CodeNotFunction, errors.New("export is not a function"))
			}
			parseJsArgs := make([]interface{}, 0, len(parseArgs))
			for _, parseArg := range parseArgs {
				parseValue, parseErr3 := goValueToJS("Module.Call", parseExport, parseArg)
				if parseErr3 != nil {
					return nil, parseErr3
				}
				parseJsArgs = append(parseJsArgs, parseValue)
			}
			parseResult, parseErr2 := awaitValue(parseCtx2, "Module.Call", parseExport, parseRaw.Invoke(parseJsArgs...))
			if parseErr2 != nil {
				return nil, parseErr2
			}
			return jsValueToGo("Module.Call", parseExport, parseResult)
		},
		callDefault: func(parseCtx3 context.Context, parseArgs2 ...any) (any, error) {
			return parseState.callDefault(parseCtx3, parseArgs2...)
		},
		value: func(parseCtx4 context.Context, parseExport2 string) (any, error) {
			parseRaw2, parseErr4 := parseState.export(parseExport2)
			if parseErr4 != nil {
				return nil, parseErr4
			}
			parseResolved, parseErr4 := awaitValue(parseCtx4, "Module.Value", parseExport2, parseRaw2)
			if parseErr4 != nil {
				return nil, parseErr4
			}
			return jsValueToGo("Module.Value", parseExport2, parseResolved)
		},
		dispose: parseState.dispose,
	}, nil
}

// OpenCrossTabChannel opens a BroadcastChannel or storage-fallback channel with the given options.
