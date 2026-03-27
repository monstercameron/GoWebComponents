//go:build js && wasm
// +build js,wasm

package interop

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"syscall/js"
	"testing"
	"time"
)

func setGlobalValue(parseName string, parseValue interface{}) func() {
	parseGlobal := js.Global()
	parsePrev := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parsePrev)
	}
}

func makePromise(parseValue js.Value) js.Value {
	return js.Global().Get("Promise").Call("resolve", parseValue)
}

func installMockMessageChannelConstructor(parseT *testing.T) func() {
	parseT.Helper()
	var parseFuncs []js.Func
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseChannel := js.Global().Get("Object").New()
		parsePort1 := js.Global().Get("Object").New()
		parsePort2 := js.Global().Get("Object").New()
		parseMessageListeners1 := js.Global().Get("Array").New()
		parseMessageListeners2 := js.Global().Get("Array").New()
		parseMessageErrorListeners1 := js.Global().Get("Array").New()
		parseMessageErrorListeners2 := js.Global().Get("Array").New()

		parseRemoveListener := func(parseListeners js.Value, parseCallback js.Value) {
			for parseIndex := 0; parseIndex < parseListeners.Length(); parseIndex++ {
				parseCurrent := parseListeners.Index(parseIndex)
				if !parseCurrent.IsUndefined() && !parseCurrent.IsNull() && parseCurrent.Equal(parseCallback) {
					parseListeners.SetIndex(parseIndex, js.Null())
				}
			}
		}
		parseEmitEvent := func(parseListeners js.Value, parsePayload js.Value, parsePorts js.Value) {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parsePayload)
			if parsePorts.IsUndefined() || parsePorts.IsNull() {
				parseEvent.Set("ports", js.Global().Get("Array").New())
			} else {
				parseEvent.Set("ports", parsePorts)
			}
			for parseIndex := 0; parseIndex < parseListeners.Length(); parseIndex++ {
				parseCallback := parseListeners.Index(parseIndex)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
		}

		parseAddEventListener1 := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			switch parseArgs2[0].String() {
			case "message":
				parseMessageListeners1.Call("push", parseArgs2[1])
			case "messageerror":
				parseMessageErrorListeners1.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener1 := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			switch parseArgs3[0].String() {
			case "message":
				parseRemoveListener(parseMessageListeners1, parseArgs3[1])
			case "messageerror":
				parseRemoveListener(parseMessageErrorListeners1, parseArgs3[1])
			}
			return nil
		})
		parseStart1 := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parsePort1.Set("__started", true)
			return nil
		})
		parseClose1 := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parsePort1.Set("__closed", true)
			return nil
		})
		parsePostMessage1 := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			if parsePort1.Get("__closed").Truthy() {
				return nil
			}
			parsePorts := js.Undefined()
			if len(parseArgs6) > 1 {
				parsePorts = parseArgs6[1]
			}
			parseEmitEvent(parseMessageListeners2, parseArgs6[0], parsePorts)
			return nil
		})

		parseAddEventListener2 := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
			switch parseArgs7[0].String() {
			case "message":
				parseMessageListeners2.Call("push", parseArgs7[1])
			case "messageerror":
				parseMessageErrorListeners2.Call("push", parseArgs7[1])
			}
			return nil
		})
		parseRemoveEventListener2 := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
			switch parseArgs8[0].String() {
			case "message":
				parseRemoveListener(parseMessageListeners2, parseArgs8[1])
			case "messageerror":
				parseRemoveListener(parseMessageErrorListeners2, parseArgs8[1])
			}
			return nil
		})
		parseStart2 := js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} {
			parsePort2.Set("__started", true)
			return nil
		})
		parseClose2 := js.FuncOf(func(parseThis10 js.Value, parseArgs10 []js.Value) interface{} {
			parsePort2.Set("__closed", true)
			return nil
		})
		parsePostMessage2 := js.FuncOf(func(parseThis11 js.Value, parseArgs11 []js.Value) interface{} {
			if parsePort2.Get("__closed").Truthy() {
				return nil
			}
			parsePorts := js.Undefined()
			if len(parseArgs11) > 1 {
				parsePorts = parseArgs11[1]
			}
			parseEmitEvent(parseMessageListeners1, parseArgs11[0], parsePorts)
			return nil
		})

		parseFuncs = append(parseFuncs,
			parseAddEventListener1,
			parseRemoveEventListener1,
			parseStart1,
			parseClose1,
			parsePostMessage1,
			parseAddEventListener2,
			parseRemoveEventListener2,
			parseStart2,
			parseClose2,
			parsePostMessage2,
		)

		parsePort1.Set("addEventListener", parseAddEventListener1)
		parsePort1.Set("removeEventListener", parseRemoveEventListener1)
		parsePort1.Set("start", parseStart1)
		parsePort1.Set("close", parseClose1)
		parsePort1.Set("postMessage", parsePostMessage1)
		parsePort2.Set("addEventListener", parseAddEventListener2)
		parsePort2.Set("removeEventListener", parseRemoveEventListener2)
		parsePort2.Set("start", parseStart2)
		parsePort2.Set("close", parseClose2)
		parsePort2.Set("postMessage", parsePostMessage2)
		parseChannel.Set("port1", parsePort1)
		parseChannel.Set("port2", parsePort2)
		return parseChannel
	})
	parseRestore := setGlobalValue("MessageChannel", parseCtor)
	return func() {
		parseRestore()
		for _, parseFn := range parseFuncs {
			parseFn.Release()
		}
		parseCtor.Release()
	}
}

func TestGlobalThisValueSurfaceSupportsPropertiesAndFunctions(parseT *testing.T) {
	parseGlobal, parseErr := GetGlobalThis()
	if parseErr != nil {
		parseT.Fatalf("expected globalThis wrapper, got %v", parseErr)
	}

	parsePrevValue := parseGlobal.Get("__interopValueProbe")
	parsePrevFn := parseGlobal.Get("__interopFnProbe")
	parseT.Cleanup(func() {
		if parsePrevValue.Present() {
			_ = parseGlobal.Set("__interopValueProbe", parsePrevValue)
		} else {
			_ = parseGlobal.Delete("__interopValueProbe")
		}
		if parsePrevFn.Present() {
			_ = parseGlobal.Set("__interopFnProbe", parsePrevFn)
		} else {
			_ = parseGlobal.Delete("__interopFnProbe")
		}
	})

	if parseErr2 := parseGlobal.Set("__interopValueProbe", map[string]any{"count": 7, "label": "ok"}); parseErr2 != nil {
		parseT.Fatalf("expected global property write to succeed, got %v", parseErr2)
	}

	parseStored := parseGlobal.Get("__interopValueProbe")
	if !parseStored.Present() {
		parseT.Fatal("expected stored probe value to be present")
	}
	parseDecoded, parseErr := parseStored.ToGo()
	if parseErr != nil {
		parseT.Fatalf("expected probe value to decode, got %v", parseErr)
	}
	parsePayload, parseOk := parseDecoded.(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected decoded probe value to be a map, got %#v", parseDecoded)
	}
	if parsePayload["count"] != float64(7) || parsePayload["label"] != "ok" {
		parseT.Fatalf("unexpected decoded payload: %#v", parsePayload)
	}

	var parseSeen string
	parseSub, parseErr := parseGlobal.SetFunction("__interopFnProbe", func(parseArgs ...Value) any {
		if len(parseArgs) != 2 {
			parseSeen = fmt.Sprintf("unexpected:%d", len(parseArgs))
			return parseSeen
		}
		parseSeen = fmt.Sprintf("%s:%d", parseArgs[0].String(), parseArgs[1].Int())
		return parseSeen
	})
	if parseErr != nil {
		parseT.Fatalf("expected function binding to succeed, got %v", parseErr)
	}
	defer parseSub.Cancel()

	parseResult, parseErr := parseGlobal.Get("__interopFnProbe").Invoke("alpha", 4)
	if parseErr != nil {
		parseT.Fatalf("expected function invocation to succeed, got %v", parseErr)
	}
	if !parseResult.Present() {
		parseT.Fatal("expected function result to be present")
	}
	if parseResult.String() != "alpha:4" {
		parseT.Fatalf("unexpected function result: %q", parseResult.String())
	}
	if parseSeen != "alpha:4" {
		parseT.Fatalf("expected callback to observe arguments, got %q", parseSeen)
	}
}

func TestInvokeReturnsStructuredErrorWhenFunctionThrows(parseT *testing.T) {
	parseThrowing := js.Global().Get("Function").New("throw new Error('invoke boom')")
	_, parseErr := Value{raw: parseThrowing}.Invoke()
	if parseErr == nil {
		parseT.Fatal("expected thrown JavaScript exception to surface as an interop error")
	}
	if !IsCode(parseErr, CodeRemote) {
		parseT.Fatalf("expected remote interop error code, got %v", parseErr)
	}
	if !strings.Contains(parseErr.Error(), "invoke boom") {
		parseT.Fatalf("expected wrapped JavaScript error message, got %v", parseErr)
	}

	parseObject := js.Global().Get("Object").New()
	parseObject.Set("explode", parseThrowing)
	_, parseErr = Value{raw: parseObject}.Call("explode")
	if parseErr == nil {
		parseT.Fatal("expected thrown JavaScript method exception to surface as an interop error")
	}
	if !IsCode(parseErr, CodeRemote) {
		parseT.Fatalf("expected remote interop error code from Call, got %v", parseErr)
	}
	if !strings.Contains(parseErr.Error(), "invoke boom") {
		parseT.Fatalf("expected wrapped JavaScript method error message, got %v", parseErr)
	}

}

func TestCallPreservesMethodThisBinding(parseT *testing.T) {
	parseObject := js.Global().Get("Object").New()
	parseObject.Set("count", 3)
	parseIncrement := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		parseNext := parseThis.Get("count").Int() + 1
		parseThis.Set("count", parseNext)
		return parseNext
	})
	defer parseIncrement.Release()
	parseObject.Set("increment", parseIncrement)

	parseResult, parseErr := Value{raw: parseObject}.Call("increment")
	if parseErr != nil {
		parseT.Fatalf("expected bound method call to succeed, got %v", parseErr)
	}
	if !parseResult.Present() {
		parseT.Fatal("expected bound method result to be present")
	}
	if parseResult.Int() != 4 {
		parseT.Fatalf("expected bound method to increment count, got %d", parseResult.Int())
	}
	if parseObject.Get("count").Int() != 4 {
		parseT.Fatalf("expected method to mutate receiver count, got %d", parseObject.Get("count").Int())
	}
}

func TestLocalStorageWrapperTracksKeysAndValues(parseT *testing.T) {
	var parseKeys []string
	parseValues := map[string]string{}
	parseStorage := js.Global().Get("Object").New()
	parseReindex := func() {
		parseStorage.Set("length", len(parseKeys))
	}

	getItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return js.Null()
		}
		if parseValue, parseOk := parseValues[parseArgs[0].String()]; parseOk {
			return parseValue
		}
		return js.Null()
	})
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseKey := parseArgs2[0].String()
		if _, parseOk2 := parseValues[parseKey]; !parseOk2 {
			parseKeys = append(parseKeys, parseKey)
		}
		parseValues[parseKey] = parseArgs2[1].String()
		parseReindex()
		return nil
	})
	defer setItemFn.Release()
	parseRemoveItemFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseKey2 := parseArgs3[0].String()
		delete(parseValues, parseKey2)
		parseNext := parseKeys[:0]
		for _, parseItem := range parseKeys {
			if parseItem != parseKey2 {
				parseNext = append(parseNext, parseItem)
			}
		}
		parseKeys = parseNext
		parseReindex()
		return nil
	})
	defer parseRemoveItemFn.Release()
	clearFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseValues = map[string]string{}
		parseKeys = nil
		parseReindex()
		return nil
	})
	defer clearFn.Release()
	parseKeyFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseIndex := parseArgs5[0].Int()
		if parseIndex < 0 || parseIndex >= len(parseKeys) {
			return js.Null()
		}
		return parseKeys[parseIndex]
	})
	defer parseKeyFn.Release()

	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseReindex()

	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseLocal, parseErr := GetLocalStorage()
	if parseErr != nil {
		parseT.Fatalf("expected localStorage wrapper, got %v", parseErr)
	}
	if parseErr2 := parseLocal.SetItem("theme", "dark"); parseErr2 != nil {
		parseT.Fatalf("expected set item to succeed, got %v", parseErr2)
	}
	parseValue2, parseOk3, parseErr := parseLocal.GetItem("theme")
	if parseErr != nil || !parseOk3 || parseValue2 != "dark" {
		parseT.Fatalf("unexpected storage read: value=%q ok=%t err=%v", parseValue2, parseOk3, parseErr)
	}
	parseLength, parseErr := parseLocal.Len()
	if parseErr != nil || parseLength != 1 {
		parseT.Fatalf("expected storage len 1, got %d err=%v", parseLength, parseErr)
	}
	parseKey3, parseOk3, parseErr := parseLocal.Key(0)
	if parseErr != nil || !parseOk3 || parseKey3 != "theme" {
		parseT.Fatalf("unexpected storage key: key=%q ok=%t err=%v", parseKey3, parseOk3, parseErr)
	}
	if parseErr3 := parseLocal.RemoveItem("theme"); parseErr3 != nil {
		parseT.Fatalf("expected remove item to succeed, got %v", parseErr3)
	}
	if _, parseOk4, parseErr4 := parseLocal.GetItem("theme"); parseErr4 != nil || parseOk4 {
		parseT.Fatalf("expected removed item to disappear, ok=%t err=%v", parseOk4, parseErr4)
	}
}

func TestLocalStorageGetManyReturnsPresentValues(parseT *testing.T) {
	parseStorage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		switch parseArgs[0].String() {
		case "theme":
			return "dark"
		case "locale":
			return "en-US"
		default:
			return js.Null()
		}
	})
	defer getItemFn.Release()
	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
	parseStorage.Set("removeItem", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
	parseStorage.Set("clear", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))
	parseStorage.Set("key", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return js.Null() }))
	parseStorage.Set("length", 2)
	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseLocal, parseErr := GetLocalStorage()
	if parseErr != nil {
		parseT.Fatalf("expected localStorage wrapper, got %v", parseErr)
	}
	parseValues, parseErr := parseLocal.GetMany("theme", "locale", "missing")
	if parseErr != nil {
		parseT.Fatalf("expected batched storage read, got %v", parseErr)
	}
	if parseValues["theme"] != "dark" || parseValues["locale"] != "en-US" {
		parseT.Fatalf("unexpected batched storage values: %#v", parseValues)
	}
	if _, parseOk := parseValues["missing"]; parseOk {
		parseT.Fatalf("expected missing storage value to be omitted, got %#v", parseValues)
	}
}

func TestOpenPersistentStoreUsesIndexedDB(parseT *testing.T) {
	parseRestoreIndexedDB := installMockIndexedDB(parseT)
	defer parseRestoreIndexedDB()

	store, parseErr := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:         "cache",
		DatabaseName: "gwc-tests",
		Version:      1,
	})
	if parseErr != nil {
		parseT.Fatalf("expected persistent store to open, got %v", parseErr)
	}
	defer func() {
		if parseCloseErr := store.Close(); parseCloseErr != nil {
			parseT.Fatalf("expected persistent store close to succeed, got %v", parseCloseErr)
		}
	}()

	if store.Backend() != "indexedDB" {
		parseT.Fatalf("expected indexedDB backend, got %q", store.Backend())
	}

	parseT.Run("close binds database receiver", func(parseT2 *testing.T) {
		parseConstructor := js.Global().Get("Object")
		parseMockDB := parseConstructor.New()
		isParseClosedWithBoundReceiver := false
		parseCloseFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			if parseThis.Equal(parseMockDB) {
				isParseClosedWithBoundReceiver = true
			}
			return nil
		})
		defer parseCloseFn.Release()
		parseMockDB.Set("close", parseCloseFn)

		parseMockStore := newIndexedDBPersistentStore(parseMockDB, persistentStoreConfig{databaseName: "gwc-tests", name: "cache"})
		if parseErr2 := parseMockStore.Close(); parseErr2 != nil {
			parseT2.Fatalf("expected mock persistent store close to succeed, got %v", parseErr2)
		}
		if !isParseClosedWithBoundReceiver {
			parseT2.Fatalf("expected persistent store close to call IndexedDB close with the database as receiver")
		}
	})
	if parseErr3 := store.SetItem(context.Background(), "theme", "dark"); parseErr3 != nil {
		parseT.Fatalf("expected persistent write to succeed, got %v", parseErr3)
	}
	if parseErr4 := store.SetJSON(context.Background(), "profile", map[string]any{"locale": "en-US", "count": 3}); parseErr4 != nil {
		parseT.Fatalf("expected persistent JSON write to succeed, got %v", parseErr4)
	}

	parseValue, parseOk, parseErr := store.GetItem(context.Background(), "theme")
	if parseErr != nil || !parseOk || parseValue != "dark" {
		parseT.Fatalf("unexpected persistent read: value=%q ok=%t err=%v", parseValue, parseOk, parseErr)
	}
	parseDecoded, parseOk, parseErr := LoadPersistentJSON[struct {
		Locale string `json:"locale"`
		Count  int    `json:"count"`
	}](context.Background(), store, "profile")
	if parseErr != nil || !parseOk {
		parseT.Fatalf("expected typed persistent JSON decode, ok=%t err=%v", parseOk, parseErr)
	}
	if parseDecoded.Locale != "en-US" || parseDecoded.Count != 3 {
		parseT.Fatalf("unexpected decoded JSON payload: %+v", parseDecoded)
	}

	parseKeys, parseErr := store.Keys(context.Background())
	if parseErr != nil {
		parseT.Fatalf("expected persistent keys, got %v", parseErr)
	}
	if len(parseKeys) != 2 || parseKeys[0] != "profile" || parseKeys[1] != "theme" {
		parseT.Fatalf("unexpected persistent keys: %#v", parseKeys)
	}
	parseLength, parseErr := store.Len(context.Background())
	if parseErr != nil || parseLength != 2 {
		parseT.Fatalf("expected persistent len 2, got %d err=%v", parseLength, parseErr)
	}

	if parseErr5 := store.RemoveItem(context.Background(), "theme"); parseErr5 != nil {
		parseT.Fatalf("expected persistent remove to succeed, got %v", parseErr5)
	}
	if _, parseOk2, parseErr6 := store.GetItem(context.Background(), "theme"); parseErr6 != nil || parseOk2 {
		parseT.Fatalf("expected removed persistent key to disappear, ok=%t err=%v", parseOk2, parseErr6)
	}
	if parseErr7 := store.Clear(context.Background()); parseErr7 != nil {
		parseT.Fatalf("expected persistent clear to succeed, got %v", parseErr7)
	}
	parseLength, parseErr = store.Len(context.Background())
	if parseErr != nil || parseLength != 0 {
		parseT.Fatalf("expected persistent len 0 after clear, got %d err=%v", parseLength, parseErr)
	}
}

func TestLocalStorageRemoveItemReturnsStructuredErrorWhenNotCallable(parseT *testing.T) {
	parseStorage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return js.Null()
	})
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return nil
	})
	defer setItemFn.Release()
	clearFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		return nil
	})
	defer clearFn.Release()
	parseKeyFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		return js.Null()
	})
	defer parseKeyFn.Release()

	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", js.Undefined())
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)

	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseLocal, parseErr := GetLocalStorage()
	if parseErr != nil {
		parseT.Fatalf("expected localStorage wrapper, got %v", parseErr)
	}
	parseErr = parseLocal.RemoveItem("theme")
	if !IsCode(parseErr, CodeNotFunction) {
		parseT.Fatalf("expected removeItem to return CodeNotFunction, got %v", parseErr)
	}
}

func TestLocalStorageRemoveItemPreservesMethodThisBinding(parseT *testing.T) {
	parseStorage := js.Global().Get("Object").New()
	parseStorage.Set("__removed", "")
	getItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return js.Null()
	})
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return nil
	})
	defer setItemFn.Release()
	parseRemoveItemFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		if parseThis3.IsUndefined() || parseThis3.IsNull() || parseThis3.Get("__removed").IsUndefined() {
			panic("illegal invocation")
		}
		if len(parseArgs3) > 0 {
			parseThis3.Set("__removed", parseArgs3[0].String())
		}
		return nil
	})
	defer parseRemoveItemFn.Release()
	clearFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		return nil
	})
	defer clearFn.Release()
	parseKeyFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		return js.Null()
	})
	defer parseKeyFn.Release()

	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)

	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseLocal, parseErr := GetLocalStorage()
	if parseErr != nil {
		parseT.Fatalf("expected localStorage wrapper, got %v", parseErr)
	}
	if parseErr2 := parseLocal.RemoveItem("auth-token"); parseErr2 != nil {
		parseT.Fatalf("expected removeItem to preserve receiver binding, got %v", parseErr2)
	}
	if parseGot := parseStorage.Get("__removed").String(); parseGot != "auth-token" {
		parseT.Fatalf("expected removeItem to run against the storage object, got %q", parseGot)
	}
}

func TestOpenPersistentStoreFallsBackWhenIndexedDBUnavailable(parseT *testing.T) {
	parseRestoreIndexedDB := setGlobalValue("indexedDB", js.Undefined())
	defer parseRestoreIndexedDB()

	parseStorage := js.Global().Get("Object").New()
	parseData := map[string]string{}
	getItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if parseValue, parseOk := parseData[parseArgs[0].String()]; parseOk {
			return parseValue
		}
		return js.Null()
	})
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseData[parseArgs2[0].String()] = parseArgs2[1].String()
		parseStorage.Set("length", len(parseData))
		return nil
	})
	defer setItemFn.Release()
	parseRemoveItemFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		delete(parseData, parseArgs3[0].String())
		parseStorage.Set("length", len(parseData))
		return nil
	})
	defer parseRemoveItemFn.Release()
	clearFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		for parseKey := range parseData {
			delete(parseData, parseKey)
		}
		parseStorage.Set("length", 0)
		return nil
	})
	defer clearFn.Release()
	parseKeyFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseKeys := make([]string, 0, len(parseData))
		for parseKey2 := range parseData {
			parseKeys = append(parseKeys, parseKey2)
		}
		sort.Strings(parseKeys)
		parseIndex := parseArgs5[0].Int()
		if parseIndex < 0 || parseIndex >= len(parseKeys) {
			return js.Null()
		}
		return parseKeys[parseIndex]
	})
	defer parseKeyFn.Release()
	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)

	store, parseErr := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:            "cache",
		FallbackBackend: "localStorage",
		FallbackResolver: func() (Storage, error) {
			return Storage{
				getItem: func(parseKey3 string) (string, bool, error) {
					parseValue2 := parseStorage.Call("getItem", parseKey3)
					if parseValue2.IsUndefined() || parseValue2.IsNull() {
						return "", false, nil
					}
					return parseValue2.String(), true, nil
				},
				setItem: func(parseKey4 string, parseValue5 string) error {
					parseStorage.Call("setItem", parseKey4, parseValue5)
					return nil
				},
				removeItem: func(parseKey5 string) error {
					parseStorage.Call("removeItem", parseKey5)
					return nil
				},
				clear: func() error {
					parseStorage.Call("clear")
					return nil
				},
				length: func() (int, error) {
					return parseStorage.Get("length").Int(), nil
				},
				key: func(parseIndex2 int) (string, bool, error) {
					parseValue3 := parseStorage.Call("key", parseIndex2)
					if parseValue3.IsNull() || parseValue3.IsUndefined() {
						return "", false, nil
					}
					return parseValue3.String(), true, nil
				},
			}, nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected fallback persistent store, got %v", parseErr)
	}
	if store.Backend() != "localStorage" {
		parseT.Fatalf("expected fallback backend label, got %q", store.Backend())
	}
	if parseErr2 := store.SetItem(context.Background(), "draft", "ready"); parseErr2 != nil {
		parseT.Fatalf("expected fallback persistent write, got %v", parseErr2)
	}
	parseValue4, parseOk2, parseErr := store.GetItem(context.Background(), "draft")
	if parseErr != nil || !parseOk2 || parseValue4 != "ready" {
		parseT.Fatalf("unexpected fallback read: value=%q ok=%t err=%v", parseValue4, parseOk2, parseErr)
	}
}

func TestOpenPersistentStoreReportsBlockedUpgrade(parseT *testing.T) {
	parseBlocked := 0
	parseRestoreIndexedDB := installMockIndexedDBWithOptions(parseT, mockIndexedDBOptions{
		blockedOpenCounts: map[string]int{"gwc-blocked": 1},
	})
	defer parseRestoreIndexedDB()

	_, parseErr := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:         "cache",
		DatabaseName: "gwc-blocked",
		Version:      2,
		OnBlocked: func(parseEvent PersistentStoreBlockedEvent) {
			parseBlocked++
			if parseEvent.DatabaseName != "gwc-blocked" || parseEvent.StoreName != "cache" || parseEvent.RequestedVersion != 2 {
				parseT.Fatalf("unexpected blocked event: %+v", parseEvent)
			}
		},
	})
	if !IsCode(parseErr, CodeBlocked) {
		parseT.Fatalf("expected blocked error, got %v", parseErr)
	}
	if parseBlocked != 1 {
		parseT.Fatalf("expected blocked callback once, got %d", parseBlocked)
	}
}

func TestOpenPersistentStoreDeletesCorruptDatabaseAndRecovers(parseT *testing.T) {
	parseRestoreIndexedDB := installMockIndexedDBWithOptions(parseT, mockIndexedDBOptions{
		openFailures: map[string][]mockIndexedDBError{
			"gwc-recover": {{Name: "InvalidStateError", Message: "backing store is corrupted"}},
		},
	})
	defer parseRestoreIndexedDB()

	store, parseErr := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:               "cache",
		DatabaseName:       "gwc-recover",
		DeleteOnCorruption: true,
	})
	if parseErr != nil {
		parseT.Fatalf("expected corruption recovery to succeed, got %v", parseErr)
	}
	defer func() {
		if parseCloseErr := store.Close(); parseCloseErr != nil {
			parseT.Fatalf("expected close after recovery to succeed, got %v", parseCloseErr)
		}
	}()
	if parseErr2 := store.SetItem(context.Background(), "theme", "dark"); parseErr2 != nil {
		parseT.Fatalf("expected recovered store to accept writes, got %v", parseErr2)
	}
}

func TestPersistentStoreSetItemReportsQuotaExceeded(parseT *testing.T) {
	parseRestoreIndexedDB := installMockIndexedDBWithOptions(parseT, mockIndexedDBOptions{
		putFailures: map[string][]mockIndexedDBError{
			"gwc-quota/cache": {{Name: "QuotaExceededError", Message: "storage quota exceeded"}},
		},
	})
	defer parseRestoreIndexedDB()

	store, parseErr := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:         "cache",
		DatabaseName: "gwc-quota",
	})
	if parseErr != nil {
		parseT.Fatalf("expected persistent store to open, got %v", parseErr)
	}
	defer func() {
		if parseCloseErr := store.Close(); parseCloseErr != nil {
			parseT.Fatalf("expected store close to succeed, got %v", parseCloseErr)
		}
	}()

	parseErr = store.SetItem(context.Background(), "theme", "dark")
	if !IsCode(parseErr, CodeQuotaExceeded) {
		parseT.Fatalf("expected quota exceeded error, got %v", parseErr)
	}
}

type mockIndexedDBError struct {
	Name    string
	Message string
}

type mockIndexedDBOptions struct {
	openFailures      map[string][]mockIndexedDBError
	putFailures       map[string][]mockIndexedDBError
	deleteFailures    map[string][]mockIndexedDBError
	blockedOpenCounts map[string]int
}

func installMockIndexedDB(parseT *testing.T) func() {
	return installMockIndexedDBWithOptions(parseT, mockIndexedDBOptions{})
}

func installMockIndexedDBWithOptions(parseT *testing.T, parseOptions mockIndexedDBOptions) func() {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseDatabaseStores := map[string]map[string]map[string]string{}
	parseDatabaseVersions := map[string]int{}
	var parseFuncs []js.Func
	parseReleaseLater := func(parseFn2 js.Func) js.Func {
		parseFuncs = append(parseFuncs, parseFn2)
		return parseFn2
	}
	parseConsumeFailure := func(parseFailures map[string][]mockIndexedDBError, parseKey5 string) (mockIndexedDBError, bool) {
		parseEntries := parseFailures[parseKey5]
		if len(parseEntries) == 0 {
			return mockIndexedDBError{}, false
		}
		parseFailure := parseEntries[0]
		parseFailures[parseKey5] = parseEntries[1:]
		return parseFailure, true
	}
	parseSchedule := func(parseRun func()) {
		var parseCallback js.Func
		parseCallback = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseCallback.Release()
			parseRun()
			return nil
		})
		parseGlobal.Call("setTimeout", parseCallback, 0)
	}
	parseNewRequest := func() js.Value {
		parseRequest := parseObjectCtor.New()
		parseRequest.Set("result", js.Null())
		parseRequest.Set("error", js.Null())
		return parseRequest
	}
	parseEmitFailure := func(parseRequest10 js.Value, parseFailure5 mockIndexedDBError) {
		parseSchedule(func() {
			parseErrValue := parseObjectCtor.New()
			parseErrValue.Set("name", parseFailure5.Name)
			parseErrValue.Set("message", parseFailure5.Message)
			parseRequest10.Set("error", parseErrValue)
			parseHandler := parseRequest10.Get("onerror")
			if parseHandler.Type() == js.TypeFunction {
				parseRequestEvent := parseObjectCtor.New()
				parseRequestEvent.Set("target", parseRequest10)
				parseHandler.Invoke(parseRequestEvent)
			}
		})
	}
	parseEmitBlocked := func(parseRequest11 js.Value) {
		parseSchedule(func() {
			parseHandler2 := parseRequest11.Get("onblocked")
			if parseHandler2.Type() == js.TypeFunction {
				parseRequestEvent2 := parseObjectCtor.New()
				parseRequestEvent2.Set("target", parseRequest11)
				parseHandler2.Invoke(parseRequestEvent2)
			}
		})
	}
	parseEmitSuccess := func(parseRequest12 js.Value, parseResult any) {
		parseSchedule(func() {
			parseRequest12.Set("result", parseResult)
			parseHandler3 := parseRequest12.Get("onsuccess")
			if parseHandler3.Type() == js.TypeFunction {
				parseRequestEvent3 := parseObjectCtor.New()
				parseRequestEvent3.Set("target", parseRequest12)
				parseHandler3.Invoke(parseRequestEvent3)
			}
		})
	}
	buildDatabase := func(parseDatabaseName3 string) js.Value {
		parseDb := parseObjectCtor.New()
		parseObjectStoreNames := parseObjectCtor.New()
		parseObjectStoreNames.Set("contains", parseReleaseLater(js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			_, parseOk := parseDatabaseStores[parseDatabaseName3][parseArgs2[0].String()]
			return parseOk
		})))
		parseDb.Set("objectStoreNames", parseObjectStoreNames)
		parseDb.Set("createObjectStore", parseReleaseLater(js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			storeName := parseArgs3[0].String()
			if parseDatabaseStores[parseDatabaseName3] == nil {
				parseDatabaseStores[parseDatabaseName3] = map[string]map[string]string{}
			}
			if parseDatabaseStores[parseDatabaseName3][storeName] == nil {
				parseDatabaseStores[parseDatabaseName3][storeName] = map[string]string{}
			}
			return parseObjectCtor.New()
		})))
		parseDb.Set("transaction", parseReleaseLater(js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			storeName := parseArgs4[0].String()
			parseTransaction := parseObjectCtor.New()
			parseTransaction.Set("objectStore", parseReleaseLater(js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
				storeName := parseArgs5[0].String()
				storeData := parseDatabaseStores[parseDatabaseName3][storeName]
				store := parseObjectCtor.New()
				store.Set("get", parseReleaseLater(js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
					parseRequest2 := parseNewRequest()
					parseKey := parseArgs6[0].String()
					if parseValue, parseOk2 := storeData[parseKey]; parseOk2 {
						parseEntry := parseObjectCtor.New()
						parseEntry.Set("key", parseKey)
						parseEntry.Set("value", parseValue)
						parseEmitSuccess(parseRequest2, parseEntry)
					} else {
						parseEmitSuccess(parseRequest2, js.Null())
					}
					return parseRequest2
				})))
				store.Set("put", parseReleaseLater(js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
					parseRequest3 := parseNewRequest()
					parseEntry2 := parseArgs7[0]
					if parseFailure2, parseOk3 := parseConsumeFailure(parseOptions.putFailures, parseDatabaseName3+"/"+storeName); parseOk3 {
						parseEmitFailure(parseRequest3, parseFailure2)
						return parseRequest3
					}
					storeData[parseEntry2.Get("key").String()] = parseEntry2.Get("value").String()
					parseEmitSuccess(parseRequest3, parseEntry2.Get("key"))
					return parseRequest3
				})))
				store.Set("delete", parseReleaseLater(js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
					parseRequest4 := parseNewRequest()
					delete(storeData, parseArgs8[0].String())
					parseEmitSuccess(parseRequest4, js.Undefined())
					return parseRequest4
				})))
				store.Set("clear", parseReleaseLater(js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} {
					parseRequest5 := parseNewRequest()
					for parseKey2 := range storeData {
						delete(storeData, parseKey2)
					}
					parseEmitSuccess(parseRequest5, js.Undefined())
					return parseRequest5
				})))
				store.Set("count", parseReleaseLater(js.FuncOf(func(parseThis10 js.Value, parseArgs10 []js.Value) interface{} {
					parseRequest6 := parseNewRequest()
					parseEmitSuccess(parseRequest6, len(storeData))
					return parseRequest6
				})))
				store.Set("getAllKeys", parseReleaseLater(js.FuncOf(func(parseThis11 js.Value, parseArgs11 []js.Value) interface{} {
					parseRequest7 := parseNewRequest()
					parseKeys := make([]string, 0, len(storeData))
					for parseKey3 := range storeData {
						parseKeys = append(parseKeys, parseKey3)
					}
					sort.Strings(parseKeys)
					parseKeysValue := js.Global().Get("Array").New(len(parseKeys))
					for parseIndex, parseKey4 := range parseKeys {
						parseKeysValue.SetIndex(parseIndex, parseKey4)
					}
					parseEmitSuccess(parseRequest7, parseKeysValue)
					return parseRequest7
				})))
				return store
			})))
			_ = storeName
			return parseTransaction
		})))
		parseDb.Set("close", parseReleaseLater(js.FuncOf(func(parseThis12 js.Value, parseArgs12 []js.Value) interface{} { return nil })))
		return parseDb
	}
	parseIndexedDB := parseObjectCtor.New()
	parseIndexedDB.Set("open", parseReleaseLater(js.FuncOf(func(parseThis13 js.Value, parseArgs13 []js.Value) interface{} {
		parseRequest8 := parseNewRequest()
		parseDatabaseName := parseArgs13[0].String()
		parseVersion := 1
		if len(parseArgs13) > 1 && parseArgs13[1].Type() != js.TypeUndefined {
			parseVersion = parseArgs13[1].Int()
		}
		if parseOptions.blockedOpenCounts[parseDatabaseName] > 0 {
			parseOptions.blockedOpenCounts[parseDatabaseName]--
			parseEmitBlocked(parseRequest8)
			return parseRequest8
		}
		if parseFailure3, parseOk4 := parseConsumeFailure(parseOptions.openFailures, parseDatabaseName); parseOk4 {
			parseEmitFailure(parseRequest8, parseFailure3)
			return parseRequest8
		}
		if parseDatabaseStores[parseDatabaseName] == nil {
			parseDatabaseStores[parseDatabaseName] = map[string]map[string]string{}
		}
		parsePreviousVersion := parseDatabaseVersions[parseDatabaseName]
		parseDatabaseVersions[parseDatabaseName] = parseVersion
		parseDb2 := buildDatabase(parseDatabaseName)
		parseSchedule(func() {
			if parsePreviousVersion == 0 || parseVersion > parsePreviousVersion {
				parseRequest8.Set("result", parseDb2)
				parseHandler4 := parseRequest8.Get("onupgradeneeded")
				if parseHandler4.Type() == js.TypeFunction {
					parseRequestEvent4 := parseObjectCtor.New()
					parseRequestEvent4.Set("target", parseRequest8)
					parseHandler4.Invoke(parseRequestEvent4)
				}
			}
			parseRequest8.Set("result", parseDb2)
			parseHandler5 := parseRequest8.Get("onsuccess")
			if parseHandler5.Type() == js.TypeFunction {
				parseRequestEvent5 := parseObjectCtor.New()
				parseRequestEvent5.Set("target", parseRequest8)
				parseHandler5.Invoke(parseRequestEvent5)
			}
		})
		return parseRequest8
	})))
	parseIndexedDB.Set("deleteDatabase", parseReleaseLater(js.FuncOf(func(parseThis14 js.Value, parseArgs14 []js.Value) interface{} {
		parseRequest9 := parseNewRequest()
		parseDatabaseName2 := parseArgs14[0].String()
		if parseFailure4, parseOk5 := parseConsumeFailure(parseOptions.deleteFailures, parseDatabaseName2); parseOk5 {
			parseEmitFailure(parseRequest9, parseFailure4)
			return parseRequest9
		}
		delete(parseDatabaseStores, parseDatabaseName2)
		delete(parseDatabaseVersions, parseDatabaseName2)
		parseEmitSuccess(parseRequest9, js.Undefined())
		return parseRequest9
	})))
	parseRestore := setGlobalValue("indexedDB", parseIndexedDB)
	parseT.Cleanup(func() {
		parseRestore()
		for _, parseFn := range parseFuncs {
			parseFn.Release()
		}
	})
	return parseRestore
}

func TestWindowHistoryPushStateRoundTripsDecodedState(parseT *testing.T) {
	parseHistory := js.Global().Get("Object").New()
	parseHistory.Set("length", 2)
	parsePushStateFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseHistory.Set("state", parseArgs[0])
		return nil
	})
	defer parsePushStateFn.Release()
	parseHistory.Set("pushState", parsePushStateFn)
	parseHistory.Set("replaceState", parsePushStateFn)
	parseHistory.Set("back", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
	parseHistory.Set("forward", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
	parseHistory.Set("go", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))

	parseWindow := js.Global().Get("Object").New()
	parseWindow.Set("history", parseHistory)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseWrapped, parseErr := GetWindowHistory()
	if parseErr != nil {
		parseT.Fatalf("expected history wrapper, got %v", parseErr)
	}
	if parseErr2 := parseWrapped.PushState(map[string]any{"step": "upload", "count": 2}, "Upload", "/upload"); parseErr2 != nil {
		parseT.Fatalf("expected pushState to succeed, got %v", parseErr2)
	}
	parseState, parseErr := parseWrapped.State()
	if parseErr != nil {
		parseT.Fatalf("expected history state, got %v", parseErr)
	}
	parsePayload := map[string]any{}
	if parseErr3 := Decode(parseState, &parsePayload); parseErr3 != nil {
		parseT.Fatalf("expected decoded history state, got %v", parseErr3)
	}
	if parsePayload["step"] != "upload" {
		parseT.Fatalf("unexpected history payload: %#v", parsePayload)
	}
}

func TestSharedWindowEnvLookupStringReadsMountedSelector(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseWindow.Set("__gwcExampleMountSelector", "#demo-root")
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseEnv, parseErr := GetWindowEnv()
	if parseErr != nil {
		parseT.Fatalf("expected window env, got %v", parseErr)
	}
	parseSelector, parseOk := parseEnv.LookupString("__gwcExampleMountSelector")
	if !parseOk {
		parseT.Fatal("expected mount selector to be present")
	}
	if parseSelector != "#demo-root" {
		parseT.Fatalf("unexpected mount selector: %q", parseSelector)
	}
	if parseResolved := parseEnv.String("__gwcExampleMountSelector", "#app"); parseResolved != "#demo-root" {
		parseT.Fatalf("expected shared env string to return mounted selector, got %q", parseResolved)
	}
}

func TestSharedWindowEnvStringFallsBackForMissingOrNullishValues(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseWindow.Set("__gwcEmpty", "")
	parseWindow.Set("__gwcNullish", "<null>")
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseEnv, parseErr := GetWindowEnv()
	if parseErr != nil {
		parseT.Fatalf("expected window env, got %v", parseErr)
	}
	if _, parseOk := parseEnv.LookupString("__gwcMissing"); parseOk {
		parseT.Fatal("expected missing shared env value to be absent")
	}
	if _, parseOk2 := parseEnv.LookupString("__gwcEmpty"); parseOk2 {
		parseT.Fatal("expected empty shared env value to be absent")
	}
	if _, parseOk3 := parseEnv.LookupString("__gwcNullish"); parseOk3 {
		parseT.Fatal("expected nullish shared env value to be absent")
	}
	if parseResolved := parseEnv.String("__gwcMissing", "#app"); parseResolved != "#app" {
		parseT.Fatalf("expected fallback for missing env, got %q", parseResolved)
	}
	if parseResolved2 := parseEnv.String("__gwcNullish", "#app"); parseResolved2 != "#app" {
		parseT.Fatalf("expected fallback for nullish env, got %q", parseResolved2)
	}
}

func TestNavigatorClipboardAwaitingPromise(parseT *testing.T) {
	var parseWritten string
	parseWriteTextFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseWritten = parseArgs[0].String()
		return makePromise(js.Undefined())
	})
	defer parseWriteTextFn.Release()
	parseReadTextFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return makePromise(js.ValueOf("copied"))
	})
	defer parseReadTextFn.Release()

	parseClipboard := js.Global().Get("Object").New()
	parseClipboard.Set("writeText", parseWriteTextFn)
	parseClipboard.Set("readText", parseReadTextFn)
	parseNavigator := js.Global().Get("Object").New()
	parseNavigator.Set("clipboard", parseClipboard)
	parseWindow := js.Global().Get("Object").New()
	parseWindow.Set("navigator", parseNavigator)
	parseRestoreNavigator := setGlobalValue("navigator", parseNavigator)
	defer parseRestoreNavigator()
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseWrapped, parseErr := GetClipboard()
	if parseErr != nil {
		parseT.Fatalf("expected clipboard wrapper, got %v", parseErr)
	}
	if parseErr2 := parseWrapped.WriteText(context.Background(), "hello"); parseErr2 != nil {
		parseT.Fatalf("expected writeText to succeed, got %v", parseErr2)
	}
	if parseWritten != "hello" {
		parseT.Fatalf("expected clipboard write payload, got %q", parseWritten)
	}
	parseText, parseErr := parseWrapped.ReadText(context.Background())
	if parseErr != nil || parseText != "copied" {
		parseT.Fatalf("expected clipboard read payload, got %q err=%v", parseText, parseErr)
	}
}

func TestTimersUseBrowserCallbacks(parseT *testing.T) {
	var parseTimeoutDelay int
	var parseClearedTimeout int
	setTimeoutFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseTimeoutDelay = parseArgs[1].Int()
		parseArgs[0].Invoke()
		return 7
	})
	defer setTimeoutFn.Release()
	clearTimeoutFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseClearedTimeout++
		return nil
	})
	defer clearTimeoutFn.Release()

	parseRestoreSetTimeout := setGlobalValue("setTimeout", setTimeoutFn)
	defer parseRestoreSetTimeout()
	parseRestoreClearTimeout := setGlobalValue("clearTimeout", clearTimeoutFn)
	defer parseRestoreClearTimeout()

	parseFired := 0
	parseTimer, parseErr := ScheduleTimeout(25*time.Millisecond, func() { parseFired++ })
	if parseErr != nil {
		parseT.Fatalf("expected timeout to succeed, got %v", parseErr)
	}
	if parseFired != 1 || parseTimeoutDelay != 25 {
		parseT.Fatalf("unexpected timeout behavior: fired=%d delay=%d", parseFired, parseTimeoutDelay)
	}
	if parseErr2 := parseTimer.Cancel(); parseErr2 != nil {
		parseT.Fatalf("expected cancel to be safe after fire, got %v", parseErr2)
	}
	if parseClearedTimeout != 0 {
		parseT.Fatalf("expected fired timeout cancel to be a no-op, got %d clear calls", parseClearedTimeout)
	}
}

func TestWindowEventsDispatchCustomEvents(parseT *testing.T) {
	var parseListener js.Value
	parseWindow := js.Global().Get("Object").New()
	parseAddEventListenerFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 2 {
			parseListener = parseArgs[1]
		}
		return nil
	})
	defer parseAddEventListenerFn.Release()
	parseRemoveEventListenerFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseListener = js.Undefined()
		return nil
	})
	defer parseRemoveEventListenerFn.Release()
	parseDispatchEventFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		if len(parseArgs3) > 0 && parseListener.Truthy() {
			parseListener.Invoke(parseArgs3[0])
		}
		return true
	})
	defer parseDispatchEventFn.Release()
	parseWindow.Set("addEventListener", parseAddEventListenerFn)
	parseWindow.Set("removeEventListener", parseRemoveEventListenerFn)
	parseWindow.Set("dispatchEvent", parseDispatchEventFn)

	parseCustomEventCtor := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseEvent := js.Global().Get("Object").New()
		parseEvent.Set("type", parseArgs4[0].String())
		parseEvent.Set("detail", parseArgs4[1].Get("detail"))
		return parseEvent
	})
	defer parseCustomEventCtor.Release()

	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()
	parseRestoreCustomEvent := setGlobalValue("CustomEvent", parseCustomEventCtor)
	defer parseRestoreCustomEvent()

	parseTarget, parseErr := GetWindowEvents()
	if parseErr != nil {
		parseT.Fatalf("expected window event target, got %v", parseErr)
	}
	var parseReceived CustomEvent
	parseSub, parseErr := parseTarget.Subscribe("asset-ready", func(parseEvent2 CustomEvent) {
		parseReceived = parseEvent2
	})
	if parseErr != nil {
		parseT.Fatalf("expected subscribe to succeed, got %v", parseErr)
	}
	defer parseSub.Cancel()

	if parseErr2 := parseTarget.Dispatch("asset-ready", map[string]any{"id": "asset-42"}); parseErr2 != nil {
		parseT.Fatalf("expected dispatch to succeed, got %v", parseErr2)
	}
	parsePayload := map[string]string{}
	if parseErr3 := Decode(parseReceived.Detail, &parsePayload); parseErr3 != nil {
		parseT.Fatalf("expected event detail to decode, got %v", parseErr3)
	}
	if parseReceived.Type != "asset-ready" || parsePayload["id"] != "asset-42" {
		parseT.Fatalf("unexpected event payload: %+v decoded=%#v", parseReceived, parsePayload)
	}
}

func TestSubscribeDecodedProjectsTypedCustomEventDetail(parseT *testing.T) {
	var parseListener js.Value
	parseWindow := js.Global().Get("Object").New()
	parseAddEventListenerFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 2 {
			parseListener = parseArgs[1]
		}
		return nil
	})
	defer parseAddEventListenerFn.Release()
	parseRemoveEventListenerFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseListener = js.Undefined()
		return nil
	})
	defer parseRemoveEventListenerFn.Release()
	parseDispatchEventFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		if len(parseArgs3) > 0 && parseListener.Truthy() {
			parseListener.Invoke(parseArgs3[0])
		}
		return true
	})
	defer parseDispatchEventFn.Release()
	parseWindow.Set("addEventListener", parseAddEventListenerFn)
	parseWindow.Set("removeEventListener", parseRemoveEventListenerFn)
	parseWindow.Set("dispatchEvent", parseDispatchEventFn)

	parseCustomEventCtor := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseEvent := js.Global().Get("Object").New()
		parseEvent.Set("type", parseArgs4[0].String())
		parseEvent.Set("detail", parseArgs4[1].Get("detail"))
		return parseEvent
	})
	defer parseCustomEventCtor.Release()

	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()
	parseRestoreCustomEvent := setGlobalValue("CustomEvent", parseCustomEventCtor)
	defer parseRestoreCustomEvent()

	parseTarget, parseErr := GetWindowEvents()
	if parseErr != nil {
		parseT.Fatalf("expected window event target, got %v", parseErr)
	}
	type ratingChange struct {
		Score  int    `json:"score"`
		Source string `json:"source"`
	}
	var (
		parseReceived DecodedCustomEvent[ratingChange]
		parseGotErr   error
	)
	parseSub, parseErr := SubscribeDecoded(parseTarget, "rating-change", func(parseEvent2 DecodedCustomEvent[ratingChange], parseErr3 error) {
		parseReceived = parseEvent2
		parseGotErr = parseErr3
	})
	if parseErr != nil {
		parseT.Fatalf("expected decoded subscription to succeed, got %v", parseErr)
	}
	defer parseSub.Cancel()

	if parseErr2 := parseTarget.Dispatch("rating-change", map[string]any{"score": 5, "source": "widget"}); parseErr2 != nil {
		parseT.Fatalf("expected dispatch to succeed, got %v", parseErr2)
	}
	if parseGotErr != nil {
		parseT.Fatalf("expected decoded event payload, got %v", parseGotErr)
	}
	if parseReceived.Type != "rating-change" || parseReceived.Detail.Score != 5 || parseReceived.Detail.Source != "widget" {
		parseT.Fatalf("unexpected decoded event payload: %+v", parseReceived)
	}
}

func TestWindowEventsListenReturnsBrowserEventTargets(parseT *testing.T) {
	var parseListener js.Value
	parseWindow := js.Global().Get("Object").New()
	parseAddEventListenerFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 2 {
			parseListener = parseArgs[1]
		}
		return nil
	})
	defer parseAddEventListenerFn.Release()
	parseRemoveEventListenerFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseListener = js.Undefined()
		return nil
	})
	defer parseRemoveEventListenerFn.Release()
	parseWindow.Set("addEventListener", parseAddEventListenerFn)
	parseWindow.Set("removeEventListener", parseRemoveEventListenerFn)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseTarget := js.Global().Get("Object").New()
	parseTarget.Set("tagName", "SECTION")
	parseTarget.Set("id", "metrics")
	parseTarget.Set("className", "panel")

	parseWrapped, parseErr := GetWindowEvents()
	if parseErr != nil {
		parseT.Fatalf("expected window event target, got %v", parseErr)
	}
	var parseReceived BrowserEvent
	parseSub, parseErr := parseWrapped.Listen("resize", func(parseEvent2 BrowserEvent) {
		parseReceived = parseEvent2
	})
	if parseErr != nil {
		parseT.Fatalf("expected generic listener to succeed, got %v", parseErr)
	}
	defer parseSub.Cancel()

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("type", "resize")
	parseEvent.Set("target", parseTarget)
	parseEvent.Set("currentTarget", parseWindow)
	parseListener.Invoke(parseEvent)

	if parseReceived.Type != "resize" {
		parseT.Fatalf("expected resize event type, got %+v", parseReceived)
	}
	if parseReceived.Target.TagName() != "SECTION" || parseReceived.Target.ID() != "metrics" {
		parseT.Fatalf("expected wrapped target element, got %+v", parseReceived.Target)
	}
}

func TestCurrentDocumentElementHelpers(parseT *testing.T) {
	var (
		parseFocusCalls int
		parseBlurCalls  int
		parseClickCalls int
		parseScrollArg  js.Value
		parseListener   js.Value
	)

	parseElement := js.Global().Get("Object").New()
	parseElement.Set("tagName", "DIV")
	parseElement.Set("id", "hero")
	parseElement.Set("className", "surface primary")
	parseFocusFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseFocusCalls++
		return nil
	})
	defer parseFocusFn.Release()
	parseBlurFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseBlurCalls++
		return nil
	})
	defer parseBlurFn.Release()
	parseClickFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseClickCalls++
		return nil
	})
	defer parseClickFn.Release()
	parseScrollFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		if len(parseArgs4) > 0 {
			parseScrollArg = parseArgs4[0]
		}
		return nil
	})
	defer parseScrollFn.Release()
	parseRectFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseRect := js.Global().Get("Object").New()
		parseRect.Set("x", 10)
		parseRect.Set("y", 12)
		parseRect.Set("width", 240)
		parseRect.Set("height", 80)
		parseRect.Set("top", 12)
		parseRect.Set("right", 250)
		parseRect.Set("bottom", 92)
		parseRect.Set("left", 10)
		return parseRect
	})
	defer parseRectFn.Release()
	parseAddEventListenerFn := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		if len(parseArgs6) >= 2 {
			parseListener = parseArgs6[1]
		}
		return nil
	})
	defer parseAddEventListenerFn.Release()
	parseRemoveEventListenerFn := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
		parseListener = js.Undefined()
		return nil
	})
	defer parseRemoveEventListenerFn.Release()
	parseElement.Set("focus", parseFocusFn)
	parseElement.Set("blur", parseBlurFn)
	parseElement.Set("click", parseClickFn)
	parseElement.Set("scrollIntoView", parseScrollFn)
	parseElement.Set("getBoundingClientRect", parseRectFn)
	parseElement.Set("addEventListener", parseAddEventListenerFn)
	parseElement.Set("removeEventListener", parseRemoveEventListenerFn)

	parseDocument := js.Global().Get("Object").New()
	getElementByIDFn := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
		return parseElement
	})
	defer getElementByIDFn.Release()
	parseQuerySelectorFn := js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} {
		return parseElement
	})
	defer parseQuerySelectorFn.Release()
	parseDocument.Set("getElementById", getElementByIDFn)
	parseDocument.Set("querySelector", parseQuerySelectorFn)
	parseRestoreDocument := setGlobalValue("document", parseDocument)
	defer parseRestoreDocument()

	parseWrapped, parseErr := GetDocument()
	if parseErr != nil {
		parseT.Fatalf("expected current document wrapper, got %v", parseErr)
	}
	parseByID, parseOk, parseErr := parseWrapped.ElementByID("hero")
	if parseErr != nil || !parseOk {
		parseT.Fatalf("expected element by id, ok=%t err=%v", parseOk, parseErr)
	}
	if parseByID.TagName() != "DIV" || parseByID.ClassName() != "surface primary" {
		parseT.Fatalf("unexpected element metadata: tag=%q class=%q", parseByID.TagName(), parseByID.ClassName())
	}
	if parseErr2 := parseByID.Focus(); parseErr2 != nil {
		parseT.Fatalf("expected focus to succeed, got %v", parseErr2)
	}
	if parseErr3 := parseByID.Blur(); parseErr3 != nil {
		parseT.Fatalf("expected blur to succeed, got %v", parseErr3)
	}
	if parseErr4 := parseByID.Click(); parseErr4 != nil {
		parseT.Fatalf("expected click to succeed, got %v", parseErr4)
	}
	if parseErr5 := parseByID.ScrollIntoView(ScrollIntoViewOptions{Behavior: "smooth", Block: "center"}); parseErr5 != nil {
		parseT.Fatalf("expected scrollIntoView to succeed, got %v", parseErr5)
	}
	parseRect2, parseErr := parseByID.BoundingClientRect()
	if parseErr != nil {
		parseT.Fatalf("expected bounding rect, got %v", parseErr)
	}
	if parseRect2.Width != 240 || parseRect2.Top != 12 {
		parseT.Fatalf("unexpected bounding rect: %+v", parseRect2)
	}
	if parseFocusCalls != 1 || parseBlurCalls != 1 || parseClickCalls != 1 {
		parseT.Fatalf("unexpected element method calls: focus=%d blur=%d click=%d", parseFocusCalls, parseBlurCalls, parseClickCalls)
	}
	if parseScrollArg.IsUndefined() || parseScrollArg.IsNull() || parseScrollArg.Get("behavior").String() != "smooth" || parseScrollArg.Get("block").String() != "center" {
		parseT.Fatalf("expected scroll options to be forwarded, got %v", parseScrollArg)
	}

	var parseReceived BrowserEvent
	parseSub, parseErr := parseByID.Listen("asset-ready", func(parseEvent2 BrowserEvent) {
		parseReceived = parseEvent2
	})
	if parseErr != nil {
		parseT.Fatalf("expected element listener to succeed, got %v", parseErr)
	}
	defer parseSub.Cancel()
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("type", "asset-ready")
	parseEvent.Set("detail", map[string]any{"asset": "hero"})
	parseEvent.Set("target", parseElement)
	parseEvent.Set("currentTarget", parseElement)
	parseListener.Invoke(parseEvent)
	if parseReceived.Type != "asset-ready" || parseReceived.Target.ID() != "hero" {
		parseT.Fatalf("unexpected element event payload: %+v", parseReceived)
	}

	parseQueried, parseOk, parseErr := parseWrapped.QuerySelector("#hero")
	if parseErr != nil || !parseOk || parseQueried.ID() != "hero" {
		parseT.Fatalf("expected querySelector result, ok=%t err=%v id=%q", parseOk, parseErr, parseQueried.ID())
	}
}

func TestCurrentDocumentElementsByIDBatchesLookups(parseT *testing.T) {
	parseFirst := js.Global().Get("Object").New()
	parseFirst.Set("tagName", "DIV")
	parseFirst.Set("id", "hero")
	parseSecond := js.Global().Get("Object").New()
	parseSecond.Set("tagName", "ASIDE")
	parseSecond.Set("id", "sidebar")

	parseDocument := js.Global().Get("Object").New()
	getElementByIDFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		switch parseArgs[0].String() {
		case "hero":
			return parseFirst
		case "sidebar":
			return parseSecond
		default:
			return js.Null()
		}
	})
	defer getElementByIDFn.Release()
	parseDocument.Set("getElementById", getElementByIDFn)
	parseQuerySelectorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return js.Null()
	})
	defer parseQuerySelectorFn.Release()
	parseDocument.Set("querySelector", parseQuerySelectorFn)
	parseRestoreDocument := setGlobalValue("document", parseDocument)
	defer parseRestoreDocument()

	parseWrapped, parseErr := GetDocument()
	if parseErr != nil {
		parseT.Fatalf("expected current document wrapper, got %v", parseErr)
	}
	parseElements, parseErr := parseWrapped.ElementsByID("hero", "sidebar", "missing")
	if parseErr != nil {
		parseT.Fatalf("expected batched element lookup, got %v", parseErr)
	}
	if parseElements["hero"].ID() != "hero" || parseElements["sidebar"].TagName() != "ASIDE" {
		parseT.Fatalf("unexpected batched element lookup results: %#v", parseElements)
	}
	if _, parseOk := parseElements["missing"]; parseOk {
		parseT.Fatalf("expected missing id to be omitted, got %#v", parseElements)
	}
}

func TestElementObserverHelpers(parseT *testing.T) {
	parseElement := js.Global().Get("Object").New()
	parseElement.Set("tagName", "ARTICLE")
	parseElement.Set("id", "observer-target")

	parseDocument := js.Global().Get("Object").New()
	getElementByIDFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parseElement
	})
	defer getElementByIDFn.Release()
	parseDocument.Set("getElementById", getElementByIDFn)
	parseQuerySelectorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return parseElement
	})
	defer parseQuerySelectorFn.Release()
	parseDocument.Set("querySelector", parseQuerySelectorFn)
	parseRestoreDocument := setGlobalValue("document", parseDocument)
	defer parseRestoreDocument()

	var (
		parseResizeCallback       js.Value
		parseIntersectionCallback js.Value
		parseResizeDisconnects    int
		parseIntersectDisconnects int
		parseIntersectionInit     js.Value
	)

	parseResizeObserverCtor := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseResizeCallback = parseArgs3[0]
		parseObserver := js.Global().Get("Object").New()
		parseObserveFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil })
		parseDisconnectFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parseResizeDisconnects++
			return nil
		})
		parseObserver.Set("observe", parseObserveFn)
		parseObserver.Set("disconnect", parseDisconnectFn)
		return parseObserver
	})
	defer parseResizeObserverCtor.Release()
	parseIntersectionObserverCtor := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		parseIntersectionCallback = parseArgs6[0]
		if len(parseArgs6) > 1 {
			parseIntersectionInit = parseArgs6[1]
		}
		parseObserver2 := js.Global().Get("Object").New()
		parseObserveFn2 := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} { return nil })
		parseDisconnectFn2 := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
			parseIntersectDisconnects++
			return nil
		})
		parseObserver2.Set("observe", parseObserveFn2)
		parseObserver2.Set("disconnect", parseDisconnectFn2)
		return parseObserver2
	})
	defer parseIntersectionObserverCtor.Release()
	parseRestoreResizeObserver := setGlobalValue("ResizeObserver", parseResizeObserverCtor)
	defer parseRestoreResizeObserver()
	parseRestoreIntersectionObserver := setGlobalValue("IntersectionObserver", parseIntersectionObserverCtor)
	defer parseRestoreIntersectionObserver()

	parseWrapped, parseErr := GetDocument()
	if parseErr != nil {
		parseT.Fatalf("expected current document wrapper, got %v", parseErr)
	}
	parseTarget, parseOk, parseErr := parseWrapped.ElementByID("observer-target")
	if parseErr != nil || !parseOk {
		parseT.Fatalf("expected target element, ok=%t err=%v", parseOk, parseErr)
	}

	var parseResizeEntry ResizeEntry
	parseResizeSub, parseErr := parseTarget.ObserveResize(func(parseEntry ResizeEntry) {
		parseResizeEntry = parseEntry
	})
	if parseErr != nil {
		parseT.Fatalf("expected resize observer to succeed, got %v", parseErr)
	}
	defer parseResizeSub.Cancel()

	parseResizeRect := js.Global().Get("Object").New()
	parseResizeRect.Set("width", 320)
	parseResizeRect.Set("height", 180)
	parseResizeRect.Set("x", 0)
	parseResizeRect.Set("y", 0)
	parseResizeRect.Set("top", 0)
	parseResizeRect.Set("right", 320)
	parseResizeRect.Set("bottom", 180)
	parseResizeRect.Set("left", 0)
	parseResizePayload := js.Global().Get("Object").New()
	parseResizePayload.Set("target", parseElement)
	parseResizePayload.Set("contentRect", parseResizeRect)
	parseResizeCallback.Invoke(js.Global().Get("Array").Call("of", parseResizePayload))
	if parseResizeEntry.Target.ID() != "observer-target" || parseResizeEntry.ContentRect.Width != 320 {
		parseT.Fatalf("unexpected resize payload: %+v", parseResizeEntry)
	}

	var parseIntersectionEntry IntersectionEntry
	parseIntersectionSub, parseErr := parseTarget.ObserveIntersection(func(parseEntry2 IntersectionEntry) {
		parseIntersectionEntry = parseEntry2
	}, IntersectionObserverOptions{RootMargin: "12px", Thresholds: []float64{0.25, 0.75}})
	if parseErr != nil {
		parseT.Fatalf("expected intersection observer to succeed, got %v", parseErr)
	}
	defer parseIntersectionSub.Cancel()

	parseIntersectionRect := js.Global().Get("Object").New()
	parseIntersectionRect.Set("width", 120)
	parseIntersectionRect.Set("height", 60)
	parseIntersectionRect.Set("x", 10)
	parseIntersectionRect.Set("y", 20)
	parseIntersectionRect.Set("top", 20)
	parseIntersectionRect.Set("right", 130)
	parseIntersectionRect.Set("bottom", 80)
	parseIntersectionRect.Set("left", 10)
	parseRootBounds := js.Global().Get("Object").New()
	parseRootBounds.Set("width", 500)
	parseRootBounds.Set("height", 400)
	parseRootBounds.Set("x", 0)
	parseRootBounds.Set("y", 0)
	parseRootBounds.Set("top", 0)
	parseRootBounds.Set("right", 500)
	parseRootBounds.Set("bottom", 400)
	parseRootBounds.Set("left", 0)
	parseIntersectionPayload := js.Global().Get("Object").New()
	parseIntersectionPayload.Set("target", parseElement)
	parseIntersectionPayload.Set("isIntersecting", true)
	parseIntersectionPayload.Set("intersectionRatio", 0.75)
	parseIntersectionPayload.Set("boundingClientRect", parseIntersectionRect)
	parseIntersectionPayload.Set("intersectionRect", parseIntersectionRect)
	parseIntersectionPayload.Set("rootBounds", parseRootBounds)
	parseIntersectionCallback.Invoke(js.Global().Get("Array").Call("of", parseIntersectionPayload))
	if !parseIntersectionEntry.IsIntersecting || parseIntersectionEntry.Target.ID() != "observer-target" || parseIntersectionEntry.RootBounds == nil || parseIntersectionEntry.RootBounds.Width != 500 {
		parseT.Fatalf("unexpected intersection payload: %+v", parseIntersectionEntry)
	}
	if parseIntersectionInit.IsUndefined() || parseIntersectionInit.IsNull() || parseIntersectionInit.Get("rootMargin").String() != "12px" || parseIntersectionInit.Get("threshold").Get("length").Int() != 2 {
		parseT.Fatalf("expected intersection observer init to carry options, got %v", parseIntersectionInit)
	}

	parseResizeSub.Cancel()
	parseIntersectionSub.Cancel()
	if parseResizeDisconnects == 0 || parseIntersectDisconnects == 0 {
		parseT.Fatalf("expected observers to disconnect on cancel, resize=%d intersection=%d", parseResizeDisconnects, parseIntersectDisconnects)
	}
}

func TestMatchMediaSubscriptionReceivesChanges(parseT *testing.T) {
	var parseChangeListener js.Value
	parseMediaQuery := js.Global().Get("Object").New()
	parseMediaQuery.Set("matches", false)
	parseMediaQuery.Set("media", "(prefers-color-scheme: dark)")
	parseAddEventListenerFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 2 {
			parseChangeListener = parseArgs[1]
		}
		return nil
	})
	defer parseAddEventListenerFn.Release()
	parseRemoveEventListenerFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseChangeListener = js.Undefined()
		return nil
	})
	defer parseRemoveEventListenerFn.Release()
	parseMediaQuery.Set("addEventListener", parseAddEventListenerFn)
	parseMediaQuery.Set("removeEventListener", parseRemoveEventListenerFn)

	parseMatchMediaFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		return parseMediaQuery
	})
	defer parseMatchMediaFn.Release()
	parseWindow := js.Global().Get("Object").New()
	parseWindow.Set("matchMedia", parseMatchMediaFn)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseList, parseErr := GetMediaQuery("(prefers-color-scheme: dark)")
	if parseErr != nil {
		parseT.Fatalf("expected media query list, got %v", parseErr)
	}
	var parseEvent MediaQueryEvent
	parseSub, parseErr := parseList.Subscribe(func(parseNext MediaQueryEvent) {
		parseEvent = parseNext
	})
	if parseErr != nil {
		parseT.Fatalf("expected media query subscribe to succeed, got %v", parseErr)
	}
	defer parseSub.Cancel()

	parseChange := js.Global().Get("Object").New()
	parseChange.Set("matches", true)
	parseChange.Set("media", "(prefers-color-scheme: dark)")
	parseChangeListener.Invoke(parseChange)
	if !parseEvent.Matches || parseEvent.Media == "" {
		parseT.Fatalf("unexpected media query event: %+v", parseEvent)
	}
}

func TestImportModuleCallDefaultAndDispose(parseT *testing.T) {
	parseSumFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return js.ValueOf(parseArgs[0].Float() + parseArgs[1].Float())
	})
	defer parseSumFn.Release()
	parseDefaultFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parsePayload := js.Global().Get("Object").New()
		parsePayload.Set("ok", true)
		parsePayload.Set("label", parseArgs2[0].String())
		return parsePayload
	})
	defer parseDefaultFn.Release()

	parseModuleNS := js.Global().Get("Object").New()
	parseModuleNS.Set("sum", parseSumFn)
	parseModuleNS.Set("default", parseDefaultFn)
	parseModuleNS.Set("version", "1.0.0")

	parseImportFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		return makePromise(parseModuleNS)
	})
	defer parseImportFn.Release()
	parseRestoreImport := setGlobalValue("__gwcImportModule", parseImportFn)
	defer parseRestoreImport()

	parseModule, parseErr := ImportModule(context.Background(), "/demo/math.js")
	if parseErr != nil {
		parseT.Fatalf("expected import to succeed, got %v", parseErr)
	}
	parseSum, parseErr := parseModule.Call(context.Background(), "sum", 2, 3)
	if parseErr != nil {
		parseT.Fatalf("expected named export call to succeed, got %v", parseErr)
	}
	if parseSum.(float64) != 5 {
		parseT.Fatalf("expected module sum result 5, got %#v", parseSum)
	}
	parseDefaultValue, parseErr := parseModule.CallDefault(context.Background(), "asset")
	if parseErr != nil {
		parseT.Fatalf("expected default export to succeed, got %v", parseErr)
	}
	parsePayload2 := map[string]any{}
	if parseErr2 := Decode(parseDefaultValue, &parsePayload2); parseErr2 != nil {
		parseT.Fatalf("expected default export payload to decode, got %v", parseErr2)
	}
	if parsePayload2["label"] != "asset" {
		parseT.Fatalf("unexpected default export payload: %#v", parsePayload2)
	}
	parseVersion, parseErr := parseModule.Value(context.Background(), "version")
	if parseErr != nil || parseVersion.(string) != "1.0.0" {
		parseT.Fatalf("expected module value export, got %#v err=%v", parseVersion, parseErr)
	}
	if parseErr3 := parseModule.Dispose(); parseErr3 != nil {
		parseT.Fatalf("expected dispose to succeed, got %v", parseErr3)
	}
	if _, parseErr4 := parseModule.Value(context.Background(), "version"); !IsCode(parseErr4, CodeDisposed) {
		parseT.Fatalf("expected disposed module error, got %v", parseErr4)
	}
}

func TestNewWorkerRequestDecodedSupportsReadyProgressAndResult(parseT *testing.T) {
	var parseCreated int
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseErrorListeners := js.Global().Get("Array").New()
		parseRaw.Set("__readySent", false)
		parseEmitMessage := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			if len(parseArgs2) > 0 {
				parseEvent.Set("data", parseArgs2[0])
			}
			for parseI := 0; parseI < parseMessageListeners.Length(); parseI++ {
				parseCallback := parseMessageListeners.Index(parseI)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parseEmitError := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			parseEvent2 := js.Global().Get("Object").New()
			if len(parseArgs3) > 0 {
				parseEvent2.Set("message", parseArgs3[0])
			}
			for parseI2 := 0; parseI2 < parseErrorListeners.Length(); parseI2++ {
				parseCallback2 := parseErrorListeners.Index(parseI2)
				if parseCallback2.IsUndefined() || parseCallback2.IsNull() {
					continue
				}
				parseCallback2.Invoke(parseEvent2)
			}
			return nil
		})
		parseAddEventListener := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEventType := parseArgs4[0].String()
			parseCallback3 := parseArgs4[1]
			switch parseEventType {
			case "message":
				parseMessageListeners.Call("push", parseCallback3)
				if !parseRaw.Get("__readySent").Bool() {
					parseRaw.Set("__readySent", true)
					parseReadyValue, _ := goValueToJS("test", "worker-ready", map[string]any{"phase": "ready", "name": "bootstrap"})
					parseRaw.Call("__emitMessage", parseReadyValue)
				}
			case "error":
				parseErrorListeners.Call("push", parseCallback3)
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parseEventType2 := parseArgs5[0].String()
			parseCallback4 := parseArgs5[1]
			var parseListeners js.Value
			switch parseEventType2 {
			case "message":
				parseListeners = parseMessageListeners
			case "error":
				parseListeners = parseErrorListeners
			default:
				return nil
			}
			for parseI3 := 0; parseI3 < parseListeners.Length(); parseI3++ {
				parseCurrent := parseListeners.Index(parseI3)
				if parseCurrent.IsUndefined() || parseCurrent.IsNull() {
					continue
				}
				if parseCurrent.Equal(parseCallback4) {
					parseListeners.SetIndex(parseI3, js.Null())
				}
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			parseValue, parseErr := jsValueToGo("test", "worker-post", parseArgs6[0])
			if parseErr != nil {
				parseRaw.Call("__emitError", parseErr.Error())
				return nil
			}
			parseMessage := workerMessageFromGo(parseValue)
			parseProgressValue, _ := goValueToJS("test", "worker-progress", map[string]any{
				"id":      parseMessage.ID,
				"phase":   "progress",
				"name":    parseMessage.Name,
				"payload": map[string]any{"percent": 40},
			})
			parseRaw.Call("__emitMessage", parseProgressValue)
			parseResultValue, _ := goValueToJS("test", "worker-result", map[string]any{
				"id":      parseMessage.ID,
				"phase":   "result",
				"name":    parseMessage.Name,
				"payload": map[string]any{"summary": "indexed 12 documents"},
			})
			parseRaw.Call("__emitMessage", parseResultValue)
			return nil
		})
		parseTerminate := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
			parseRaw.Set("__terminated", true)
			return nil
		})
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("__emitError", parseEmitError)
		parseCreated++
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr2 := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/search.mjs", Ready: true})
	if parseErr2 != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr2)
	}
	if parseCreated != 1 {
		parseT.Fatalf("expected one worker instance, got %d", parseCreated)
	}

	type progressPayload struct {
		Percent int `json:"percent"`
	}
	type resultPayload struct {
		Summary string `json:"summary"`
	}
	var parseProgress []int
	parseResult, parseErr2 := RequestWorkerDecoded[map[string]any, progressPayload, resultPayload](context.Background(), parseWorker, "build-index", map[string]any{"query": "atlas"}, func(parseMessage2 DecodedWorkerMessage[progressPayload], parseErr3 error) {
		if parseErr3 != nil {
			parseT.Fatalf("expected decoded progress payload, got %v", parseErr3)
		}
		parseProgress = append(parseProgress, parseMessage2.Payload.Percent)
	})
	if parseErr2 != nil {
		parseT.Fatalf("expected worker request to succeed, got %v", parseErr2)
	}
	if len(parseProgress) != 1 || parseProgress[0] != 40 {
		parseT.Fatalf("unexpected worker progress updates: %#v", parseProgress)
	}
	if parseResult.Summary != "indexed 12 documents" {
		parseT.Fatalf("unexpected worker result payload: %+v", parseResult)
	}
}

func TestWorkerRequestFailsOnMessageError(parseT *testing.T) {
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseErrorListeners := js.Global().Get("Array").New()
		parseMessageErrorListeners := js.Global().Get("Array").New()
		parseEmitMessageError := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			if len(parseArgs2) > 0 {
				parseEvent.Set("message", parseArgs2[0])
			}
			for parseI := 0; parseI < parseMessageErrorListeners.Length(); parseI++ {
				parseCallback := parseMessageErrorListeners.Index(parseI)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parseAddEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			switch parseArgs3[0].String() {
			case "message":
				parseMessageListeners.Call("push", parseArgs3[1])
			case "error":
				parseErrorListeners.Call("push", parseArgs3[1])
			case "messageerror":
				parseMessageErrorListeners.Call("push", parseArgs3[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil })
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parseRaw.Call("__emitMessageError", "structured clone failed")
			return nil
		})
		parseTerminate := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		parseRaw.Set("__emitMessageError", parseEmitMessageError)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/messageerror.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}
	parseCtx, parseCancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer parseCancel()
	if _, parseErr2 := parseWorker.Request(parseCtx, "decode-job", map[string]any{"input": "demo"}, nil); !IsCode(parseErr2, CodeDecode) {
		parseT.Fatalf("expected decode error from messageerror, got %v", parseErr2)
	}
}

func TestWorkerRequestFailsOnMalformedMessage(parseT *testing.T) {
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parsePostMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			for parseI := 0; parseI < parseMessageListeners.Length(); parseI++ {
				parseCallback := parseMessageListeners.Index(parseI)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parseTerminate := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil })
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/malformed.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}
	parseCtx, parseCancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer parseCancel()
	if _, parseErr2 := parseWorker.Request(parseCtx, "decode-job", map[string]any{"input": "demo"}, nil); !IsCode(parseErr2, CodeDecode) {
		parseT.Fatalf("expected decode error from malformed message event, got %v", parseErr2)
	}
}

func TestWorkerSubscribeReportsMessageError(parseT *testing.T) {
	var (
		parseReceivedErr error
		parseWorkerRaw   js.Value
	)
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseWorkerRaw = parseRaw
		parseMessageErrorListeners := js.Global().Get("Array").New()
		parseEmitMessageError := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			if len(parseArgs2) > 0 {
				parseEvent.Set("message", parseArgs2[0])
			}
			for parseI := 0; parseI < parseMessageErrorListeners.Length(); parseI++ {
				parseCallback := parseMessageErrorListeners.Index(parseI)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parseAddEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			if parseArgs3[0].String() == "messageerror" {
				parseMessageErrorListeners.Call("push", parseArgs3[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil })
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil })
		parseTerminate := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		parseRaw.Set("__emitMessageError", parseEmitMessageError)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/subscribe.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}
	parseSubscription, parseErr2 := parseWorker.Subscribe(func(parseMessage WorkerMessage, parseErr3 error) {
		_ = parseMessage
		parseReceivedErr = parseErr3
	})
	if parseErr2 != nil {
		parseT.Fatalf("expected worker subscribe to succeed, got %v", parseErr2)
	}
	defer parseSubscription.Cancel()

	parseWorkerRaw.Call("__emitMessageError", "channel decode failed")
	if !IsCode(parseReceivedErr, CodeDecode) {
		parseT.Fatalf("expected subscribe messageerror to surface as decode error, got %v", parseReceivedErr)
	}
}

func TestWorkerTerminateAndRestartSwapActiveInstance(parseT *testing.T) {
	var parseCreated int
	var parsePosts int
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseRaw.Set("addEventListener", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
		parseRaw.Set("removeEventListener", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
		parseRaw.Set("postMessage", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parsePosts++
			return nil
		}))
		parseRaw.Set("terminate", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parseRaw.Set("__terminated", true)
			return nil
		}))
		parseCreated++
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/report.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}
	if parseErr2 := parseWorker.Post(map[string]any{"phase": "message"}); parseErr2 != nil {
		parseT.Fatalf("expected initial worker post to succeed, got %v", parseErr2)
	}
	if parseErr3 := parseWorker.Terminate(); parseErr3 != nil {
		parseT.Fatalf("expected terminate to succeed, got %v", parseErr3)
	}
	if parseErr4 := parseWorker.Post(map[string]any{"phase": "message"}); !IsCode(parseErr4, CodeDisposed) {
		parseT.Fatalf("expected disposed error after terminate, got %v", parseErr4)
	}
	if parseErr5 := parseWorker.Restart(context.Background()); parseErr5 != nil {
		parseT.Fatalf("expected restart to succeed, got %v", parseErr5)
	}
	if parseErr6 := parseWorker.Post(map[string]any{"phase": "message"}); parseErr6 != nil {
		parseT.Fatalf("expected worker post to succeed after restart, got %v", parseErr6)
	}
	if parseCreated != 2 {
		parseT.Fatalf("expected two worker instances after restart, got %d", parseCreated)
	}
	if parsePosts != 2 {
		parseT.Fatalf("expected posts to reach the active workers only, got %d", parsePosts)
	}
}

func TestWorkerRequestHonorsContextTimeout(parseT *testing.T) {
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseRaw.Set("addEventListener", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
		parseRaw.Set("removeEventListener", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
		parseRaw.Set("postMessage", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))
		parseRaw.Set("terminate", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil }))
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/slow.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}
	parseCtx, parseCancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer parseCancel()
	if _, parseErr2 := parseWorker.Request(parseCtx, "slow-job", map[string]any{"input": "demo"}, nil); !IsCode(parseErr2, CodeTimeout) {
		parseT.Fatalf("expected timeout error, got %v", parseErr2)
	}
}

// TestOpenWorkerPoolUsesBrowserWorkers verifies the pool routes typed requests
// through real browser-worker wrappers under js/wasm.
func TestOpenWorkerPoolUsesBrowserWorkers(parseT *testing.T) {
	var parseWorkerCount int
	var parseTerminateCount int
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseWorkerCount++
		parseWorkerID := parseWorkerCount
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			parseEvent.Set("ports", js.Global().Get("Array").New())
			for parseIndex := 0; parseIndex < parseMessageListeners.Length(); parseIndex++ {
				parseCallback := parseMessageListeners.Index(parseIndex)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parseRequestID := parseArgs5[0].Get("id").String()
			parseEnvelope := js.Global().Get("Object").New()
			parseEnvelope.Set("id", parseRequestID)
			parseEnvelope.Set("phase", "result")
			parseEnvelope.Set("name", parseArgs5[0].Get("name"))
			parsePayload := js.Global().Get("Object").New()
			parsePayload.Set("worker", parseWorkerID)
			parseEnvelope.Set("payload", parsePayload)
			parseRaw.Call("__emitMessage", parseEnvelope)
			return nil
		})
		parseTerminate := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			parseTerminateCount++
			return nil
		})
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       2,
		QueueLimit: 0,
		OpenWorker: func(parseCtx context.Context) (Worker, error) {
			return OpenWorker(parseCtx, WorkerOptions{URL: "/workers/pool.js"})
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected browser-backed worker pool, got %v", parseErr)
	}

	parseResultACh := make(chan int, 1)
	go func() {
		parseResult, parseErr := RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](context.Background(), parsePool, "task-a", map[string]int{"job": 1}, nil)
		if parseErr != nil {
			parseT.Errorf("expected browser-backed pooled request A, got %v", parseErr)
			return
		}
		parseResultACh <- parseResult["worker"]
	}()
	parseResultBCh := make(chan int, 1)
	go func() {
		parseResult, parseErr := RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](context.Background(), parsePool, "task-b", map[string]int{"job": 2}, nil)
		if parseErr != nil {
			parseT.Errorf("expected browser-backed pooled request B, got %v", parseErr)
			return
		}
		parseResultBCh <- parseResult["worker"]
	}()

	parseWorkerA := readWorkerPoolInt(parseT, parseResultACh, "browser-backed pooled request A result")
	parseWorkerB := readWorkerPoolInt(parseT, parseResultBCh, "browser-backed pooled request B result")
	if parseWorkerA <= 0 || parseWorkerB <= 0 {
		parseT.Fatalf("expected browser-backed pooled results from constructed workers, got %d and %d", parseWorkerA, parseWorkerB)
	}

	if parseErr := parsePool.Close(); parseErr != nil {
		parseT.Fatalf("expected browser-backed pool close, got %v", parseErr)
	}
	if parseWorkerCount != 2 || parseTerminateCount != 2 {
		parseT.Fatalf("expected two browser workers opened and terminated, got opened=%d terminated=%d", parseWorkerCount, parseTerminateCount)
	}
}

func TestOpenMessageChannelTransfersPorts(parseT *testing.T) {
	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()

	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}
	parseBranch, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected branch message channel, got %v", parseErr)
	}

	parsePrimaryCh := make(chan string, 1)
	parsePrimarySub, parseErr := SubscribeDecodedMessagePort[struct {
		Kind string `json:"kind"`
	}](parseChannel.Port2(), func(parseMessage DecodedMessagePortMessage[struct {
		Kind string `json:"kind"`
	}], parseErr2 error) {
		if parseErr2 != nil {
			parsePrimaryCh <- "error:" + parseErr2.Error()
			return
		}
		if parseMessage.Payload.Kind == "handoff" {
			if len(parseMessage.Ports) != 1 {
				parsePrimaryCh <- fmt.Sprintf("ports:%d", len(parseMessage.Ports))
				return
			}
			if parseErr3 := parseMessage.Ports[0].Post(map[string]any{"kind": "branch-ack"}); parseErr3 != nil {
				parsePrimaryCh <- "post:" + parseErr3.Error()
				return
			}
		}
		parsePrimaryCh <- parseMessage.Payload.Kind
	})
	if parseErr != nil {
		parseT.Fatalf("expected primary message-port subscription, got %v", parseErr)
	}
	defer parsePrimarySub.Cancel()

	parseBranchCh := make(chan string, 1)
	parseBranchSub, parseErr := SubscribeDecodedMessagePort[struct {
		Kind string `json:"kind"`
	}](parseBranch.Port1(), func(parseMessage DecodedMessagePortMessage[struct {
		Kind string `json:"kind"`
	}], parseErr2 error) {
		if parseErr2 != nil {
			parseBranchCh <- "error:" + parseErr2.Error()
			return
		}
		parseBranchCh <- parseMessage.Payload.Kind
	})
	if parseErr != nil {
		parseT.Fatalf("expected branch message-port subscription, got %v", parseErr)
	}
	defer parseBranchSub.Cancel()

	if parseErr2 := parseChannel.Port1().Post(map[string]any{"kind": "hello"}); parseErr2 != nil {
		parseT.Fatalf("expected direct message-port publish to succeed, got %v", parseErr2)
	}
	select {
	case parseGot := <-parsePrimaryCh:
		if parseGot != "hello" {
			parseT.Fatalf("expected primary hello payload, got %q", parseGot)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for direct message-port payload")
	}

	if parseErr3 := parseChannel.Port1().PostPorts(map[string]any{"kind": "handoff"}, parseBranch.Port2()); parseErr3 != nil {
		parseT.Fatalf("expected message-port handoff to succeed, got %v", parseErr3)
	}
	select {
	case parseGot2 := <-parsePrimaryCh:
		if parseGot2 != "handoff" {
			parseT.Fatalf("expected handoff payload, got %q", parseGot2)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for handoff payload")
	}
	select {
	case parseGot3 := <-parseBranchCh:
		if parseGot3 != "branch-ack" {
			parseT.Fatalf("expected branch ack payload, got %q", parseGot3)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for transferred message-port payload")
	}
}

func TestMessagePortCloseDisposesPort(parseT *testing.T) {
	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()

	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}
	if parseErr2 := parseChannel.Port1().Close(); parseErr2 != nil {
		parseT.Fatalf("expected message port close to succeed, got %v", parseErr2)
	}
	if parseErr3 := parseChannel.Port1().Post(map[string]any{"kind": "after-close"}); !IsCode(parseErr3, CodeDisposed) {
		parseT.Fatalf("expected disposed error after message port close, got %v", parseErr3)
	}
	if parseErr4 := parseChannel.Port1().Close(); !IsCode(parseErr4, CodeDisposed) {
		parseT.Fatalf("expected disposed error on duplicate message port close, got %v", parseErr4)
	}
}

func TestWorkerPostPortsTransfersMessagePort(parseT *testing.T) {
	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()

	var parseTransferCount int
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseRaw.Set("addEventListener", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
		parseRaw.Set("removeEventListener", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
		parseRaw.Set("terminate", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))
		parseRaw.Set("postMessage", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			if len(parseArgs5) > 1 {
				parseTransferCount = parseArgs5[1].Length()
				if parseTransferCount > 0 {
					parseAck := js.Global().Get("Object").New()
					parseAck.Set("kind", "worker-ack")
					parseArgs5[1].Index(0).Call("postMessage", parseAck)
				}
			}
			return nil
		}))
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/ports.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}
	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}

	parseAckCh := make(chan string, 1)
	parseAckSub, parseErr := SubscribeDecodedMessagePort[struct {
		Kind string `json:"kind"`
	}](parseChannel.Port1(), func(parseMessage DecodedMessagePortMessage[struct {
		Kind string `json:"kind"`
	}], parseErr2 error) {
		if parseErr2 != nil {
			parseAckCh <- "error:" + parseErr2.Error()
			return
		}
		parseAckCh <- parseMessage.Payload.Kind
	})
	if parseErr != nil {
		parseT.Fatalf("expected port subscription, got %v", parseErr)
	}
	defer parseAckSub.Cancel()

	if parseErr2 := parseWorker.PostPorts(map[string]any{"kind": "handoff"}, parseChannel.Port2()); parseErr2 != nil {
		parseT.Fatalf("expected worker post-with-ports to succeed, got %v", parseErr2)
	}
	select {
	case parseGot := <-parseAckCh:
		if parseGot != "worker-ack" {
			parseT.Fatalf("expected worker ack over transferred port, got %q", parseGot)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for worker port ack")
	}
	if parseTransferCount != 1 {
		parseT.Fatalf("expected one transferred message port, got %d", parseTransferCount)
	}
}

func TestTwoWorkersCommunicateOverTransferredPorts(parseT *testing.T) {
	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()

	var parseWorkerCount int
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseWorkerCount++
		parseWorkerID := parseWorkerCount
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			if len(parseArgs4) > 1 {
				parseEvent.Set("ports", parseArgs4[1])
			} else {
				parseEvent.Set("ports", js.Global().Get("Array").New())
			}
			for parseIndex := 0; parseIndex < parseMessageListeners.Length(); parseIndex++ {
				parseCallback := parseMessageListeners.Index(parseIndex)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			if len(parseArgs5) < 2 || parseArgs5[1].Length() == 0 {
				return nil
			}
			parsePort := parseArgs5[1].Index(0)
			parsePortListener := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
				parseKind := parseArgs6[0].Get("data").Get("kind").String()
				switch parseWorkerID {
				case 1:
					if parseKind != "ping" {
						return nil
					}
					parsePayload := js.Global().Get("Object").New()
					parsePayload.Set("worker", "worker-a")
					parsePayload.Set("kind", "received-ping")
					parseEnvelope := js.Global().Get("Object").New()
					parseEnvelope.Set("phase", "message")
					parseEnvelope.Set("name", "relay")
					parseEnvelope.Set("payload", parsePayload)
					parseRaw.Call("__emitMessage", parseEnvelope)
					parseAck := js.Global().Get("Object").New()
					parseAck.Set("kind", "pong")
					parsePort.Call("postMessage", parseAck)
				case 2:
					if parseKind != "pong" {
						return nil
					}
					parsePayload := js.Global().Get("Object").New()
					parsePayload.Set("worker", "worker-b")
					parsePayload.Set("kind", "received-pong")
					parseEnvelope := js.Global().Get("Object").New()
					parseEnvelope.Set("phase", "message")
					parseEnvelope.Set("name", "relay")
					parseEnvelope.Set("payload", parsePayload)
					parseRaw.Call("__emitMessage", parseEnvelope)
				}
				return nil
			})
			parsePort.Call("addEventListener", "message", parsePortListener)
			parsePort.Call("start")
			if parseWorkerID == 2 {
				parsePing := js.Global().Get("Object").New()
				parsePing.Set("kind", "ping")
				parsePort.Call("postMessage", parsePing)
			}
			return nil
		})
		parseTerminate := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} { return nil })
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorkerA, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/a.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker A wrapper, got %v", parseErr)
	}
	parseWorkerB, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/b.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker B wrapper, got %v", parseErr)
	}
	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}

	type relayPayload struct {
		Worker string `json:"worker"`
		Kind   string `json:"kind"`
	}
	parseWorkerACh := make(chan relayPayload, 1)
	parseWorkerASub, parseErr := SubscribeDecodedWorker[relayPayload](parseWorkerA, func(parseMessage DecodedWorkerMessage[relayPayload], parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected worker A decoded message, got %v", parseErr2)
		}
		parseWorkerACh <- parseMessage.Payload
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker A subscription, got %v", parseErr)
	}
	defer parseWorkerASub.Cancel()

	parseWorkerBCh := make(chan relayPayload, 1)
	parseWorkerBSub, parseErr := SubscribeDecodedWorker[relayPayload](parseWorkerB, func(parseMessage DecodedWorkerMessage[relayPayload], parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected worker B decoded message, got %v", parseErr2)
		}
		parseWorkerBCh <- parseMessage.Payload
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker B subscription, got %v", parseErr)
	}
	defer parseWorkerBSub.Cancel()

	if parseErr2 := parseWorkerA.PostPorts(map[string]any{"kind": "connect-a"}, parseChannel.Port1()); parseErr2 != nil {
		parseT.Fatalf("expected worker A port handoff to succeed, got %v", parseErr2)
	}
	if parseErr3 := parseWorkerB.PostPorts(map[string]any{"kind": "connect-b"}, parseChannel.Port2()); parseErr3 != nil {
		parseT.Fatalf("expected worker B port handoff to succeed, got %v", parseErr3)
	}

	select {
	case parseGot := <-parseWorkerACh:
		if parseGot.Worker != "worker-a" || parseGot.Kind != "received-ping" {
			parseT.Fatalf("expected worker A to receive ping over transferred port, got %+v", parseGot)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for worker A to receive ping from worker B")
	}

	select {
	case parseGot2 := <-parseWorkerBCh:
		if parseGot2.Worker != "worker-b" || parseGot2.Kind != "received-pong" {
			parseT.Fatalf("expected worker B to receive pong over transferred port, got %+v", parseGot2)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for worker B to receive pong from worker A")
	}
}

func TestSubscribeDecodedWorkerRetainsTransferredPorts(parseT *testing.T) {
	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()

	var parseWorkerRaw js.Value
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			if len(parseArgs4) > 1 {
				parseEvent.Set("ports", parseArgs4[1])
			} else {
				parseEvent.Set("ports", js.Global().Get("Array").New())
			}
			for parseIndex := 0; parseIndex < parseMessageListeners.Length(); parseIndex++ {
				parseCallback := parseMessageListeners.Index(parseIndex)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil })
		parseTerminate := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		parseWorkerRaw = parseRaw
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/subscribe-ports.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}
	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}

	parseAckCh := make(chan string, 1)
	parseAckSub, parseErr := SubscribeDecodedMessagePort[struct {
		Kind string `json:"kind"`
	}](parseChannel.Port1(), func(parseMessage DecodedMessagePortMessage[struct {
		Kind string `json:"kind"`
	}], parseErr2 error) {
		if parseErr2 != nil {
			parseAckCh <- "error:" + parseErr2.Error()
			return
		}
		parseAckCh <- parseMessage.Payload.Kind
	})
	if parseErr != nil {
		parseT.Fatalf("expected port ack subscription, got %v", parseErr)
	}
	defer parseAckSub.Cancel()

	parseWorkerSub, parseErr := SubscribeDecodedWorker[struct {
		Kind string `json:"kind"`
	}](parseWorker, func(parseMessage DecodedWorkerMessage[struct {
		Kind string `json:"kind"`
	}], parseErr2 error) {
		if parseErr2 != nil {
			parseAckCh <- "worker:" + parseErr2.Error()
			return
		}
		if parseMessage.Payload.Kind != "handoff" || len(parseMessage.Ports) != 1 {
			parseAckCh <- fmt.Sprintf("worker:%s:%d", parseMessage.Payload.Kind, len(parseMessage.Ports))
			return
		}
		if parseErr3 := parseMessage.Ports[0].Post(map[string]any{"kind": "subscribe-ack"}); parseErr3 != nil {
			parseAckCh <- "post:" + parseErr3.Error()
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected decoded worker subscription, got %v", parseErr)
	}
	defer parseWorkerSub.Cancel()

	parsePorts := js.Global().Get("Array").New(1)
	parsePorts.SetIndex(0, parseChannel.Port2().raw.(js.Value))
	parseEnvelope := js.Global().Get("Object").New()
	parseEnvelope.Set("phase", "message")
	parseEnvelope.Set("name", "handoff")
	parsePayload := js.Global().Get("Object").New()
	parsePayload.Set("kind", "handoff")
	parseEnvelope.Set("payload", parsePayload)
	parseWorkerRaw.Call("__emitMessage", parseEnvelope, parsePorts)

	select {
	case parseGot := <-parseAckCh:
		if parseGot != "subscribe-ack" {
			parseT.Fatalf("expected subscribe ack over transferred worker port, got %q", parseGot)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for transferred worker subscription port ack")
	}
}

func TestWorkerScopeSubscribeReceivesTransferredPorts(parseT *testing.T) {
	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()
	parseRestoreDocument := setGlobalValue("document", js.Undefined())
	defer parseRestoreDocument()

	var parseMessageListener js.Value
	parseAddEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if parseArgs[0].String() == "message" {
			parseMessageListener = parseArgs[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if parseArgs2[0].String() == "message" && parseMessageListener.Equal(parseArgs2[1]) {
			parseMessageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parsePostMessage := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
	defer parsePostMessage.Release()
	parseRestoreAdd := setGlobalValue("addEventListener", parseAddEventListener)
	defer parseRestoreAdd()
	parseRestoreRemove := setGlobalValue("removeEventListener", parseRemoveEventListener)
	defer parseRestoreRemove()
	parseRestorePostMessage := setGlobalValue("postMessage", parsePostMessage)
	defer parseRestorePostMessage()

	parseScope, parseErr := GetWorkerScope()
	if parseErr != nil {
		parseT.Fatalf("expected worker scope, got %v", parseErr)
	}
	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}

	parseAckCh := make(chan string, 1)
	parseAckSub, parseErr := SubscribeDecodedMessagePort[struct {
		Kind string `json:"kind"`
	}](parseChannel.Port1(), func(parseMessage DecodedMessagePortMessage[struct {
		Kind string `json:"kind"`
	}], parseErr2 error) {
		if parseErr2 != nil {
			parseAckCh <- "error:" + parseErr2.Error()
			return
		}
		parseAckCh <- parseMessage.Payload.Kind
	})
	if parseErr != nil {
		parseT.Fatalf("expected ack subscription, got %v", parseErr)
	}
	defer parseAckSub.Cancel()

	parseScopeSub, parseErr := parseScope.Subscribe(func(parseMessage WorkerMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseAckCh <- "scope:" + parseErr2.Error()
			return
		}
		if parseMessage.Name != "handoff" || len(parseMessage.Ports) != 1 {
			parseAckCh <- fmt.Sprintf("scope:%s:%d", parseMessage.Name, len(parseMessage.Ports))
			return
		}
		if parseErr3 := parseMessage.Ports[0].Post(map[string]any{"kind": "scope-ack"}); parseErr3 != nil {
			parseAckCh <- "post:" + parseErr3.Error()
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker-scope subscription, got %v", parseErr)
	}
	defer parseScopeSub.Cancel()

	parseEnvelope := js.Global().Get("Object").New()
	parseEnvelope.Set("phase", "message")
	parseEnvelope.Set("name", "handoff")
	parsePorts := js.Global().Get("Array").New(1)
	parsePorts.SetIndex(0, parseChannel.Port2().raw.(js.Value))
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("data", parseEnvelope)
	parseEvent.Set("ports", parsePorts)
	parseMessageListener.Invoke(parseEvent)

	select {
	case parseGot := <-parseAckCh:
		if parseGot != "scope-ack" {
			parseT.Fatalf("expected scope ack over transferred port, got %q", parseGot)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for worker-scope port ack")
	}
}

func TestWorkerScopePostPortsTransfersMessagePort(parseT *testing.T) {
	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()
	parseRestoreDocument := setGlobalValue("document", js.Undefined())
	defer parseRestoreDocument()

	parsePostedCount := 0
	parsePostMessage := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parsePostedCount++
		if len(parseArgs) < 2 || parseArgs[1].Length() != 1 {
			return nil
		}
		parseAck := js.Global().Get("Object").New()
		parseAck.Set("kind", "scope-post-ack")
		parseArgs[1].Index(0).Call("postMessage", parseAck)
		return nil
	})
	defer parsePostMessage.Release()
	parseRestorePostMessage := setGlobalValue("postMessage", parsePostMessage)
	defer parseRestorePostMessage()

	parseScope, parseErr := GetWorkerScope()
	if parseErr != nil {
		parseT.Fatalf("expected worker scope, got %v", parseErr)
	}
	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}

	parseAckCh := make(chan string, 1)
	parseAckSub, parseErr := SubscribeDecodedMessagePort[struct {
		Kind string `json:"kind"`
	}](parseChannel.Port1(), func(parseMessage DecodedMessagePortMessage[struct {
		Kind string `json:"kind"`
	}], parseErr2 error) {
		if parseErr2 != nil {
			parseAckCh <- "error:" + parseErr2.Error()
			return
		}
		parseAckCh <- parseMessage.Payload.Kind
	})
	if parseErr != nil {
		parseT.Fatalf("expected scope post ack subscription, got %v", parseErr)
	}
	defer parseAckSub.Cancel()

	if parseErr2 := parseScope.PostPorts(WorkerMessage{Phase: "message", Name: "handoff", Payload: map[string]any{"kind": "handoff"}}, parseChannel.Port2()); parseErr2 != nil {
		parseT.Fatalf("expected worker scope post-with-ports to succeed, got %v", parseErr2)
	}
	select {
	case parseGot := <-parseAckCh:
		if parseGot != "scope-post-ack" {
			parseT.Fatalf("expected scope post ack over transferred port, got %q", parseGot)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for worker-scope post-with-ports ack")
	}
	if parsePostedCount != 1 {
		parseT.Fatalf("expected one worker-scope postMessage call, got %d", parsePostedCount)
	}
}

// TestGetSharedMemorySupportReportsCapabilities verifies shared-memory support
// follows the runtime capability set and cross-origin isolation state.
func TestGetSharedMemorySupportReportsCapabilities(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	parseSupport, parseErr := GetSharedMemorySupport()
	parseRestore()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.IsCrossOriginIsolated {
		parseT.Fatalf("expected cross-origin-isolated support state, got %+v", parseSupport)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}
	if !parseSupport.CanUseSharedMemory {
		parseT.Fatalf("expected shared memory to be usable when isolation is enabled, got %+v", parseSupport)
	}

	parseRestore = setGlobalValue("crossOriginIsolated", false)
	defer parseRestore()
	parseDisabledSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection with disabled isolation, got %v", parseErr)
	}
	if parseDisabledSupport.IsCrossOriginIsolated {
		parseT.Fatalf("expected disabled cross-origin isolation, got %+v", parseDisabledSupport)
	}
	if parseDisabledSupport.CanUseSharedMemory {
		parseT.Fatalf("expected shared memory to be unavailable without isolation, got %+v", parseDisabledSupport)
	}
	if !parseDisabledSupport.HasSharedArrayBuffer || !parseDisabledSupport.HasAtomics {
		parseT.Fatalf("expected runtime capabilities to remain visible, got %+v", parseDisabledSupport)
	}
}

// TestOpenSharedBufferRequiresCrossOriginIsolation verifies shared buffers stay
// blocked until cross-origin isolation is enabled.
func TestOpenSharedBufferRequiresCrossOriginIsolation(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", false)
	defer parseRestore()

	if _, parseErr := OpenSharedBuffer(16); !IsCode(parseErr, CodeUnavailable) {
		parseT.Fatalf("expected unavailable shared buffer without isolation, got %v", parseErr)
	}
}

// TestOpenSharedBufferSupportsByteAndAtomicAccess verifies byte copies and the
// core int32 atomic operations over a shared buffer.
func TestOpenSharedBufferSupportsByteAndAtomicAccess(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	defer parseRestore()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}

	parseBuffer, parseErr := OpenSharedBuffer(16)
	if parseErr != nil {
		parseT.Fatalf("expected shared buffer, got %v", parseErr)
	}
	if parseBuffer.GetByteLength() != 16 {
		parseT.Fatalf("expected 16-byte shared buffer, got %d", parseBuffer.GetByteLength())
	}
	if parseBuffer.GetInt32Length() != 4 {
		parseT.Fatalf("expected four int32 slots, got %d", parseBuffer.GetInt32Length())
	}

	if parseWritten, parseErr := parseBuffer.WriteBytes(2, []byte{1, 2, 3, 4}); parseErr != nil || parseWritten != 4 {
		parseT.Fatalf("expected 4 shared-buffer bytes written, got count=%d err=%v", parseWritten, parseErr)
	}
	parseRead := make([]byte, 4)
	if parseCount, parseErr := parseBuffer.ReadBytes(2, parseRead); parseErr != nil || parseCount != 4 {
		parseT.Fatalf("expected 4 shared-buffer bytes read, got count=%d err=%v", parseCount, parseErr)
	}
	if fmt.Sprintf("%v", parseRead) != "[1 2 3 4]" {
		parseT.Fatalf("expected byte roundtrip, got %v", parseRead)
	}

	if parseWritten, parseErr := parseBuffer.WriteBytes(14, []byte{9, 8, 7}); parseErr != nil || parseWritten != 2 {
		parseT.Fatalf("expected tail write truncation to 2 bytes, got count=%d err=%v", parseWritten, parseErr)
	}
	parseTail := make([]byte, 4)
	if parseCount, parseErr := parseBuffer.ReadBytes(12, parseTail); parseErr != nil || parseCount != 4 {
		parseT.Fatalf("expected tail bytes read, got count=%d err=%v", parseCount, parseErr)
	}
	if fmt.Sprintf("%v", parseTail) != "[0 0 9 8]" {
		parseT.Fatalf("expected shared-buffer tail read, got %v", parseTail)
	}
	if parseWritten, parseErr := parseBuffer.WriteBytes(16, []byte{7, 6}); parseErr != nil || parseWritten != 0 {
		parseT.Fatalf("expected zero-byte write at end of shared buffer, got count=%d err=%v", parseWritten, parseErr)
	}
	if parseCount, parseErr := parseBuffer.ReadBytes(16, make([]byte, 2)); parseErr != nil || parseCount != 0 {
		parseT.Fatalf("expected zero-byte read at end of shared buffer, got count=%d err=%v", parseCount, parseErr)
	}
	if _, parseErr := parseBuffer.ReadBytes(-1, make([]byte, 1)); !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected invalid negative shared-buffer read, got %v", parseErr)
	}
	if _, parseErr := parseBuffer.WriteBytes(17, []byte{1}); !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected invalid shared-buffer write past end, got %v", parseErr)
	}

	if parseErr := parseBuffer.StoreInt32(0, 7); parseErr != nil {
		parseT.Fatalf("expected shared-buffer int32 store, got %v", parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 7 {
		parseT.Fatalf("expected int32 load=7, got value=%d err=%v", parseValue, parseErr)
	}
	if parsePrevious, parseErr := parseBuffer.AddInt32(0, 3); parseErr != nil || parsePrevious != 7 {
		parseT.Fatalf("expected add to return previous value 7, got value=%d err=%v", parsePrevious, parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 10 {
		parseT.Fatalf("expected int32 load=10 after add, got value=%d err=%v", parseValue, parseErr)
	}
	if parsePrevious, parseErr := parseBuffer.SubInt32(0, 2); parseErr != nil || parsePrevious != 10 {
		parseT.Fatalf("expected sub to return previous value 10, got value=%d err=%v", parsePrevious, parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 8 {
		parseT.Fatalf("expected int32 load=8 after sub, got value=%d err=%v", parseValue, parseErr)
	}

	if parseErr := parseBuffer.StoreInt32(1, 15); parseErr != nil {
		parseT.Fatalf("expected second int32 store, got %v", parseErr)
	}
	if parsePrevious, parseErr := parseBuffer.AndInt32(1, 10); parseErr != nil || parsePrevious != 15 {
		parseT.Fatalf("expected and to return previous value 15, got value=%d err=%v", parsePrevious, parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(1); parseErr != nil || parseValue != 10 {
		parseT.Fatalf("expected int32 load=10 after and, got value=%d err=%v", parseValue, parseErr)
	}
	if parsePrevious, parseErr := parseBuffer.OrInt32(1, 5); parseErr != nil || parsePrevious != 10 {
		parseT.Fatalf("expected or to return previous value 10, got value=%d err=%v", parsePrevious, parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(1); parseErr != nil || parseValue != 15 {
		parseT.Fatalf("expected int32 load=15 after or, got value=%d err=%v", parseValue, parseErr)
	}
	if parsePrevious, parseErr := parseBuffer.XorInt32(1, 3); parseErr != nil || parsePrevious != 15 {
		parseT.Fatalf("expected xor to return previous value 15, got value=%d err=%v", parsePrevious, parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(1); parseErr != nil || parseValue != 12 {
		parseT.Fatalf("expected int32 load=12 after xor, got value=%d err=%v", parseValue, parseErr)
	}
	if parsePrevious, parseErr := parseBuffer.ExchangeInt32(1, 21); parseErr != nil || parsePrevious != 12 {
		parseT.Fatalf("expected exchange to return previous value 12, got value=%d err=%v", parsePrevious, parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(1); parseErr != nil || parseValue != 21 {
		parseT.Fatalf("expected int32 load=21 after exchange, got value=%d err=%v", parseValue, parseErr)
	}
	if parsePrevious, parseErr := parseBuffer.CompareExchangeInt32(1, 20, 99); parseErr != nil || parsePrevious != 21 {
		parseT.Fatalf("expected failed compare-exchange to return current value 21, got value=%d err=%v", parsePrevious, parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(1); parseErr != nil || parseValue != 21 {
		parseT.Fatalf("expected int32 load=21 after failed compare-exchange, got value=%d err=%v", parseValue, parseErr)
	}
	if parsePrevious, parseErr := parseBuffer.CompareExchangeInt32(1, 21, 34); parseErr != nil || parsePrevious != 21 {
		parseT.Fatalf("expected successful compare-exchange to return previous value 21, got value=%d err=%v", parsePrevious, parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(1); parseErr != nil || parseValue != 34 {
		parseT.Fatalf("expected int32 load=34 after compare-exchange, got value=%d err=%v", parseValue, parseErr)
	}
	if _, parseErr := parseBuffer.LoadInt32(4); !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected invalid int32 load past end, got %v", parseErr)
	}
	if parseErr := parseBuffer.StoreInt32(-1, 1); !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected invalid int32 store before start, got %v", parseErr)
	}
}

// TestOpenSharedBufferWaitInt32RequiresWorkerScope verifies blocking waits stay
// unavailable on the main-thread path while notify remains callable.
func TestOpenSharedBufferWaitInt32RequiresWorkerScope(parseT *testing.T) {
	parseRestoreIsolation := setGlobalValue("crossOriginIsolated", true)
	defer parseRestoreIsolation()
	parseRestoreDocument := setGlobalValue("document", js.Global().Get("Object").New())
	defer parseRestoreDocument()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	parseAtomics := js.Global().Get("Atomics")
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics || parseAtomics.Get("wait").Type() != js.TypeFunction || parseAtomics.Get("notify").Type() != js.TypeFunction {
		parseT.Skipf("shared-memory wait/notify runtime support is unavailable: %+v", parseSupport)
	}

	parseBuffer, parseErr := OpenSharedBuffer(4)
	if parseErr != nil {
		parseT.Fatalf("expected shared buffer, got %v", parseErr)
	}
	if parseErr := parseBuffer.StoreInt32(0, 1); parseErr != nil {
		parseT.Fatalf("expected shared-buffer store before wait test, got %v", parseErr)
	}
	if _, parseErr := parseBuffer.WaitInt32(0, 1, time.Millisecond); !IsCode(parseErr, CodeUnavailable) {
		parseT.Fatalf("expected main-thread shared-buffer wait to be unavailable, got %v", parseErr)
	}
	if parseCount, parseErr := parseBuffer.NotifyInt32(0, 1); parseErr != nil || parseCount != 0 {
		parseT.Fatalf("expected main-thread shared-buffer notify wake count 0, got count=%d err=%v", parseCount, parseErr)
	}
}

// TestOpenSharedBufferWaitAndNotifyInt32 verifies worker-safe wait states and
// notify wake counts over one shared int32 slot.
func TestOpenSharedBufferWaitAndNotifyInt32(parseT *testing.T) {
	parseRestoreIsolation := setGlobalValue("crossOriginIsolated", true)
	defer parseRestoreIsolation()
	parseRestoreDocument := setGlobalValue("document", js.Undefined())
	defer parseRestoreDocument()
	parsePostMessage := js.FuncOf(func(js.Value, []js.Value) interface{} { return nil })
	defer parsePostMessage.Release()
	parseRestorePostMessage := setGlobalValue("postMessage", parsePostMessage)
	defer parseRestorePostMessage()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	parseAtomics := js.Global().Get("Atomics")
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics || parseAtomics.Get("wait").Type() != js.TypeFunction || parseAtomics.Get("notify").Type() != js.TypeFunction {
		parseT.Skipf("shared-memory wait/notify runtime support is unavailable: %+v", parseSupport)
	}

	parseBuffer, parseErr := OpenSharedBuffer(4)
	if parseErr != nil {
		parseT.Fatalf("expected shared buffer, got %v", parseErr)
	}
	if parseErr := parseBuffer.StoreInt32(0, 2); parseErr != nil {
		parseT.Fatalf("expected shared-buffer store before wait test, got %v", parseErr)
	}

	if parseStatus, parseErr := parseBuffer.WaitInt32(0, 1, time.Millisecond); parseErr != nil || parseStatus != "not-equal" {
		parseT.Fatalf("expected shared-buffer wait result not-equal, got status=%q err=%v", parseStatus, parseErr)
	}
	if parseStatus, parseErr := parseBuffer.WaitInt32(0, 2, 5*time.Millisecond); parseErr != nil || parseStatus != "timed-out" {
		parseT.Fatalf("expected shared-buffer wait timeout, got status=%q err=%v", parseStatus, parseErr)
	}
	if parseCount, parseErr := parseBuffer.NotifyInt32(0, 1); parseErr != nil || parseCount != 0 {
		parseT.Fatalf("expected shared-buffer notify wake count 0 without waiters, got count=%d err=%v", parseCount, parseErr)
	}
}

// TestWorkerPostTransmitsSharedBufferPayload verifies workers can receive
// shared buffers both directly and inside structured worker envelopes.
func TestWorkerPostTransmitsSharedBufferPayload(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	defer parseRestore()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}

	parseBuffer, parseErr := OpenSharedBuffer(4)
	if parseErr != nil {
		parseT.Fatalf("expected shared buffer, got %v", parseErr)
	}

	var parsePostCount int
	var parseSawDirect bool
	var parseSawEnvelope bool
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseRaw.Set("addEventListener", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
		parseRaw.Set("removeEventListener", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
		parseRaw.Set("terminate", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))
		parseRaw.Set("postMessage", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parsePostCount++
			parseAtomics := js.Global().Get("Atomics")
			parseSharedCtor := js.Global().Get("SharedArrayBuffer")
			switch parsePostCount {
			case 1:
				if parseSharedCtor.Type() != js.TypeFunction || !parseArgs5[0].InstanceOf(parseSharedCtor) {
					return nil
				}
				parseSawDirect = true
				parseAtomics.Call("store", js.Global().Get("Int32Array").New(parseArgs5[0]), 0, 11)
			case 2:
				parsePayload := parseArgs5[0].Get("payload")
				if parseSharedCtor.Type() != js.TypeFunction || !parsePayload.InstanceOf(parseSharedCtor) {
					return nil
				}
				parseSawEnvelope = true
				parseAtomics.Call("store", js.Global().Get("Int32Array").New(parsePayload), 0, 22)
			}
			return nil
		}))
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/shared-buffer-post.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}

	if parseErr := parseWorker.Post(parseBuffer); parseErr != nil {
		parseT.Fatalf("expected direct shared-buffer post to succeed, got %v", parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 11 {
		parseT.Fatalf("expected direct shared-buffer worker mutation, got value=%d err=%v", parseValue, parseErr)
	}

	if parseErr := parseWorker.Post(WorkerMessage{Phase: "message", Name: "shared", Payload: parseBuffer}); parseErr != nil {
		parseT.Fatalf("expected structured shared-buffer post to succeed, got %v", parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 22 {
		parseT.Fatalf("expected structured shared-buffer worker mutation, got value=%d err=%v", parseValue, parseErr)
	}

	if parsePostCount != 2 || !parseSawDirect || !parseSawEnvelope {
		parseT.Fatalf("expected both shared-buffer worker post paths, got count=%d direct=%v envelope=%v", parsePostCount, parseSawDirect, parseSawEnvelope)
	}
}

// TestWorkerPostTransmitsNestedSharedBufferPayload verifies nested map and
// struct payload graphs preserve shared buffers on the worker send path.
func TestWorkerPostTransmitsNestedSharedBufferPayload(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	defer parseRestore()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}

	type nestedState struct {
		Buffer SharedBuffer `json:"buffer"`
		Label  string       `json:"label"`
	}
	type nestedPayload struct {
		State nestedState `json:"state"`
	}

	parseBuffer, parseErr := OpenSharedBuffer(4)
	if parseErr != nil {
		parseT.Fatalf("expected shared buffer, got %v", parseErr)
	}

	var parsePostCount int
	var parseSawNestedMap bool
	var parseSawNestedStruct bool
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseRaw.Set("addEventListener", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
		parseRaw.Set("removeEventListener", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
		parseRaw.Set("terminate", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))
		parseRaw.Set("postMessage", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parsePostCount++
			parseAtomics := js.Global().Get("Atomics")
			parseSharedCtor := js.Global().Get("SharedArrayBuffer")
			switch parsePostCount {
			case 1:
				parseNestedBuffer := parseArgs5[0].Get("state").Get("buffer")
				if parseSharedCtor.Type() != js.TypeFunction || !parseNestedBuffer.InstanceOf(parseSharedCtor) {
					return nil
				}
				parseSawNestedMap = true
				parseAtomics.Call("store", js.Global().Get("Int32Array").New(parseNestedBuffer), 0, 31)
			case 2:
				parseNestedBuffer := parseArgs5[0].Get("payload").Get("state").Get("buffer")
				if parseSharedCtor.Type() != js.TypeFunction || !parseNestedBuffer.InstanceOf(parseSharedCtor) {
					return nil
				}
				parseSawNestedStruct = true
				parseAtomics.Call("store", js.Global().Get("Int32Array").New(parseNestedBuffer), 0, 47)
			}
			return nil
		}))
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/shared-buffer-nested-post.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}

	if parseErr := parseWorker.Post(map[string]any{
		"state": map[string]any{
			"buffer": parseBuffer,
			"label":  "nested-map",
		},
	}); parseErr != nil {
		parseT.Fatalf("expected nested shared-buffer map post to succeed, got %v", parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 31 {
		parseT.Fatalf("expected nested map shared-buffer worker mutation, got value=%d err=%v", parseValue, parseErr)
	}

	if parseErr := parseWorker.Post(WorkerMessage{
		Phase: "message",
		Name:  "nested",
		Payload: nestedPayload{
			State: nestedState{
				Buffer: parseBuffer,
				Label:  "nested-struct",
			},
		},
	}); parseErr != nil {
		parseT.Fatalf("expected nested shared-buffer struct post to succeed, got %v", parseErr)
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 47 {
		parseT.Fatalf("expected nested struct shared-buffer worker mutation, got value=%d err=%v", parseValue, parseErr)
	}

	if parsePostCount != 2 || !parseSawNestedMap || !parseSawNestedStruct {
		parseT.Fatalf("expected nested shared-buffer worker post paths, got count=%d map=%v struct=%v", parsePostCount, parseSawNestedMap, parseSawNestedStruct)
	}
}

// TestWorkerPostTransmitsNestedBinaryPayload verifies nested map and struct
// payload graphs preserve binary leaves as typed-array worker payloads.
func TestWorkerPostTransmitsNestedBinaryPayload(parseT *testing.T) {
	type nestedBytes struct {
		Blob []byte `json:"blob"`
	}
	type nestedPayload struct {
		Data nestedBytes `json:"data"`
	}

	var parsePostCount int
	var parseSawNestedMap bool
	var parseSawNestedStruct bool
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseRaw.Set("addEventListener", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
		parseRaw.Set("removeEventListener", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
		parseRaw.Set("terminate", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))
		parseRaw.Set("postMessage", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parsePostCount++
			switch parsePostCount {
			case 1:
				parseValue, parseErr := jsValueToGo("test", "nested-bytes-map", parseArgs5[0].Get("data").Get("blob"))
				if parseErr != nil {
					parseT.Fatalf("expected nested binary map payload decode, got %v", parseErr)
				}
				parseBytes, parseOk := parseValue.([]byte)
				if !parseOk || !bytes.Equal(parseBytes, []byte{1, 2, 3}) {
					parseT.Fatalf("expected nested binary map payload [1 2 3], got %T %v", parseValue, parseValue)
				}
				parseSawNestedMap = true
			case 2:
				parseValue, parseErr := jsValueToGo("test", "nested-bytes-struct", parseArgs5[0].Get("payload").Get("data").Get("blob"))
				if parseErr != nil {
					parseT.Fatalf("expected nested binary struct payload decode, got %v", parseErr)
				}
				parseBytes, parseOk := parseValue.([]byte)
				if !parseOk || !bytes.Equal(parseBytes, []byte{4, 5, 6}) {
					parseT.Fatalf("expected nested binary struct payload [4 5 6], got %T %v", parseValue, parseValue)
				}
				parseSawNestedStruct = true
			}
			return nil
		}))
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/binary-nested-post.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}

	if parseErr := parseWorker.Post(map[string]any{
		"data": map[string]any{
			"blob": []byte{1, 2, 3},
		},
	}); parseErr != nil {
		parseT.Fatalf("expected nested binary map post to succeed, got %v", parseErr)
	}

	if parseErr := parseWorker.Post(WorkerMessage{
		Phase: "message",
		Name:  "nested-bytes",
		Payload: nestedPayload{
			Data: nestedBytes{Blob: []byte{4, 5, 6}},
		},
	}); parseErr != nil {
		parseT.Fatalf("expected nested binary struct post to succeed, got %v", parseErr)
	}

	if parsePostCount != 2 || !parseSawNestedMap || !parseSawNestedStruct {
		parseT.Fatalf("expected nested binary worker post paths, got count=%d map=%v struct=%v", parsePostCount, parseSawNestedMap, parseSawNestedStruct)
	}
}

// TestWorkerSubscribeReceivesSharedBufferPayload verifies incoming worker
// messages decode shared buffers without dropping the underlying memory.
func TestWorkerSubscribeReceivesSharedBufferPayload(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	defer parseRestore()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}

	var parseWorkerRaw js.Value
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			parseEvent.Set("ports", js.Global().Get("Array").New())
			for parseIndex := 0; parseIndex < parseMessageListeners.Length(); parseIndex++ {
				parseCallback := parseMessageListeners.Index(parseIndex)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil })
		parseTerminate := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		parseWorkerRaw = parseRaw
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/shared-buffer-subscribe.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}

	parseSharedCtor := js.Global().Get("SharedArrayBuffer")
	parseRawBuffer := parseSharedCtor.New(4)
	js.Global().Get("Atomics").Call("store", js.Global().Get("Int32Array").New(parseRawBuffer), 0, 42)

	parseResultCh := make(chan int32, 1)
	parseSubscription, parseErr := parseWorker.Subscribe(func(parseMessage WorkerMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected worker shared-buffer message, got %v", parseErr2)
		}
		parseBuffer, parseOk := parseMessage.Payload.(SharedBuffer)
		if !parseOk {
			parseT.Fatalf("expected worker payload shared buffer, got %T", parseMessage.Payload)
		}
		parseValue, parseErr3 := parseBuffer.LoadInt32(0)
		if parseErr3 != nil {
			parseT.Fatalf("expected shared-buffer load from worker payload, got %v", parseErr3)
		}
		if parseErr4 := parseBuffer.StoreInt32(0, 77); parseErr4 != nil {
			parseT.Fatalf("expected shared-buffer store from worker payload, got %v", parseErr4)
		}
		parseResultCh <- parseValue
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker subscription, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEnvelope := js.Global().Get("Object").New()
	parseEnvelope.Set("phase", "message")
	parseEnvelope.Set("name", "shared")
	parseEnvelope.Set("payload", parseRawBuffer)
	parseWorkerRaw.Call("__emitMessage", parseEnvelope)

	select {
	case parseValue := <-parseResultCh:
		if parseValue != 42 {
			parseT.Fatalf("expected worker shared-buffer payload value 42, got %d", parseValue)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for worker shared-buffer payload")
	}
	if parseCurrent := js.Global().Get("Atomics").Call("load", js.Global().Get("Int32Array").New(parseRawBuffer), 0).Int(); parseCurrent != 77 {
		parseT.Fatalf("expected shared-buffer worker payload mutation to persist, got %d", parseCurrent)
	}
}

// TestWorkerSubscribeReceivesNestedSharedBufferPayload verifies nested worker
// payload objects preserve shared buffers during JS-to-Go decoding.
func TestWorkerSubscribeReceivesNestedSharedBufferPayload(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	defer parseRestore()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}

	var parseWorkerRaw js.Value
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			parseEvent.Set("ports", js.Global().Get("Array").New())
			for parseIndex := 0; parseIndex < parseMessageListeners.Length(); parseIndex++ {
				parseCallback := parseMessageListeners.Index(parseIndex)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("postMessage", js.FuncOf(func(js.Value, []js.Value) interface{} { return nil }))
		parseRaw.Set("terminate", js.FuncOf(func(js.Value, []js.Value) interface{} { return nil }))
		parseWorkerRaw = parseRaw
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/shared-buffer-nested-subscribe.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}

	parseSharedCtor := js.Global().Get("SharedArrayBuffer")
	parseRawBuffer := parseSharedCtor.New(4)
	js.Global().Get("Atomics").Call("store", js.Global().Get("Int32Array").New(parseRawBuffer), 0, 52)

	parseResultCh := make(chan int32, 1)
	parseSubscription, parseErr := parseWorker.Subscribe(func(parseMessage WorkerMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected nested worker shared-buffer message, got %v", parseErr2)
		}
		parsePayload, parseOk := parseMessage.Payload.(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected nested worker payload map, got %T", parseMessage.Payload)
		}
		parseState, parseOk := parsePayload["state"].(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected nested worker payload state map, got %T", parsePayload["state"])
		}
		parseBuffer, parseOk := parseState["buffer"].(SharedBuffer)
		if !parseOk {
			parseT.Fatalf("expected nested worker payload buffer, got %T", parseState["buffer"])
		}
		parseValue, parseErr3 := parseBuffer.LoadInt32(0)
		if parseErr3 != nil {
			parseT.Fatalf("expected nested shared-buffer load from worker payload, got %v", parseErr3)
		}
		if parseErr4 := parseBuffer.StoreInt32(0, 83); parseErr4 != nil {
			parseT.Fatalf("expected nested shared-buffer store from worker payload, got %v", parseErr4)
		}
		parseResultCh <- parseValue
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker subscription, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEnvelope := js.Global().Get("Object").New()
	parseEnvelope.Set("phase", "message")
	parseEnvelope.Set("name", "nested")
	parsePayload := js.Global().Get("Object").New()
	parseState := js.Global().Get("Object").New()
	parseState.Set("buffer", parseRawBuffer)
	parseState.Set("label", "nested")
	parsePayload.Set("state", parseState)
	parseEnvelope.Set("payload", parsePayload)
	parseWorkerRaw.Call("__emitMessage", parseEnvelope)

	select {
	case parseValue := <-parseResultCh:
		if parseValue != 52 {
			parseT.Fatalf("expected nested worker shared-buffer payload value 52, got %d", parseValue)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for nested worker shared-buffer payload")
	}
	if parseCurrent := js.Global().Get("Atomics").Call("load", js.Global().Get("Int32Array").New(parseRawBuffer), 0).Int(); parseCurrent != 83 {
		parseT.Fatalf("expected nested worker shared-buffer payload mutation to persist, got %d", parseCurrent)
	}
}

// TestWorkerSubscribeReceivesNestedBinaryPayload verifies nested worker payload
// objects decode typed-array leaves into Go byte slices.
func TestWorkerSubscribeReceivesNestedBinaryPayload(parseT *testing.T) {
	var parseWorkerRaw js.Value
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			parseEvent.Set("ports", js.Global().Get("Array").New())
			for parseIndex := 0; parseIndex < parseMessageListeners.Length(); parseIndex++ {
				parseCallback := parseMessageListeners.Index(parseIndex)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("postMessage", js.FuncOf(func(js.Value, []js.Value) interface{} { return nil }))
		parseRaw.Set("terminate", js.FuncOf(func(js.Value, []js.Value) interface{} { return nil }))
		parseWorkerRaw = parseRaw
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseWorker, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/binary-nested-subscribe.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker wrapper, got %v", parseErr)
	}

	parseResultCh := make(chan []byte, 1)
	parseSubscription, parseErr := parseWorker.Subscribe(func(parseMessage WorkerMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected nested worker binary message, got %v", parseErr2)
		}
		parsePayload, parseOk := parseMessage.Payload.(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected nested worker binary payload map, got %T", parseMessage.Payload)
		}
		parseData, parseOk := parsePayload["data"].(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected nested worker binary data map, got %T", parsePayload["data"])
		}
		parseBlob, parseOk := parseData["blob"].([]byte)
		if !parseOk {
			parseT.Fatalf("expected nested worker binary blob, got %T", parseData["blob"])
		}
		parseResultCh <- append([]byte(nil), parseBlob...)
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker subscription, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEnvelope := js.Global().Get("Object").New()
	parseEnvelope.Set("phase", "message")
	parseEnvelope.Set("name", "nested-bytes")
	parsePayload := js.Global().Get("Object").New()
	parseData := js.Global().Get("Object").New()
	parseBytes := js.Global().Get("Uint8Array").New(3)
	js.CopyBytesToJS(parseBytes, []byte{7, 8, 9})
	parseData.Set("blob", parseBytes)
	parsePayload.Set("data", parseData)
	parseEnvelope.Set("payload", parsePayload)
	parseWorkerRaw.Call("__emitMessage", parseEnvelope)

	select {
	case parseValue := <-parseResultCh:
		if !bytes.Equal(parseValue, []byte{7, 8, 9}) {
			parseT.Fatalf("expected nested worker binary payload [7 8 9], got %v", parseValue)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for nested worker binary payload")
	}
}

// TestMessagePortCarriesSharedBufferPayload verifies message ports can transport
// shared buffers without copying the underlying shared memory.
func TestMessagePortCarriesSharedBufferPayload(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	defer parseRestore()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}

	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()

	parseBuffer, parseErr := OpenSharedBuffer(4)
	if parseErr != nil {
		parseT.Fatalf("expected shared buffer, got %v", parseErr)
	}
	if parseErr := parseBuffer.StoreInt32(0, 5); parseErr != nil {
		parseT.Fatalf("expected shared-buffer store before port transfer, got %v", parseErr)
	}

	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}

	parseResultCh := make(chan int32, 1)
	parseSubscription, parseErr := parseChannel.Port2().Subscribe(func(parseMessage MessagePortMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected shared-buffer message-port payload, got %v", parseErr2)
		}
		parseBufferPayload, parseOk := parseMessage.Payload.(SharedBuffer)
		if !parseOk {
			parseT.Fatalf("expected message-port shared buffer payload, got %T", parseMessage.Payload)
		}
		parsePrevious, parseErr3 := parseBufferPayload.AddInt32(0, 2)
		if parseErr3 != nil {
			parseT.Fatalf("expected shared-buffer add from port payload, got %v", parseErr3)
		}
		parseResultCh <- parsePrevious
	})
	if parseErr != nil {
		parseT.Fatalf("expected message-port subscription, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	if parseErr := parseChannel.Port1().Post(parseBuffer); parseErr != nil {
		parseT.Fatalf("expected shared-buffer post over message port, got %v", parseErr)
	}

	select {
	case parsePrevious := <-parseResultCh:
		if parsePrevious != 5 {
			parseT.Fatalf("expected message-port shared-buffer previous value 5, got %d", parsePrevious)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for message-port shared-buffer payload")
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 7 {
		parseT.Fatalf("expected shared-buffer mutation to persist after port transfer, got value=%d err=%v", parseValue, parseErr)
	}
}

// TestMessagePortCarriesNestedSharedBufferPayload verifies nested payload maps
// preserve shared buffers across message-port transport.
func TestMessagePortCarriesNestedSharedBufferPayload(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	defer parseRestore()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}

	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()

	parseBuffer, parseErr := OpenSharedBuffer(4)
	if parseErr != nil {
		parseT.Fatalf("expected shared buffer, got %v", parseErr)
	}
	if parseErr := parseBuffer.StoreInt32(0, 12); parseErr != nil {
		parseT.Fatalf("expected nested shared-buffer store before port transfer, got %v", parseErr)
	}

	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}

	parseResultCh := make(chan int32, 1)
	parseSubscription, parseErr := parseChannel.Port2().Subscribe(func(parseMessage MessagePortMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected nested shared-buffer message-port payload, got %v", parseErr2)
		}
		parsePayload, parseOk := parseMessage.Payload.(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected nested message-port payload map, got %T", parseMessage.Payload)
		}
		parseState, parseOk := parsePayload["state"].(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected nested message-port state map, got %T", parsePayload["state"])
		}
		parseBufferPayload, parseOk := parseState["buffer"].(SharedBuffer)
		if !parseOk {
			parseT.Fatalf("expected nested message-port shared buffer payload, got %T", parseState["buffer"])
		}
		parsePrevious, parseErr3 := parseBufferPayload.ExchangeInt32(0, 19)
		if parseErr3 != nil {
			parseT.Fatalf("expected nested shared-buffer exchange from port payload, got %v", parseErr3)
		}
		parseResultCh <- parsePrevious
	})
	if parseErr != nil {
		parseT.Fatalf("expected nested message-port subscription, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	if parseErr := parseChannel.Port1().Post(map[string]any{
		"state": map[string]any{
			"buffer": parseBuffer,
			"label":  "nested-port",
		},
	}); parseErr != nil {
		parseT.Fatalf("expected nested shared-buffer post over message port, got %v", parseErr)
	}

	select {
	case parsePrevious := <-parseResultCh:
		if parsePrevious != 12 {
			parseT.Fatalf("expected nested message-port shared-buffer previous value 12, got %d", parsePrevious)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for nested message-port shared-buffer payload")
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 19 {
		parseT.Fatalf("expected nested shared-buffer mutation to persist after port transfer, got value=%d err=%v", parseValue, parseErr)
	}
}

// TestMessagePortCarriesNestedBinaryPayload verifies nested payload maps
// preserve binary leaves across message-port transport.
func TestMessagePortCarriesNestedBinaryPayload(parseT *testing.T) {
	parseRestoreChannel := installMockMessageChannelConstructor(parseT)
	defer parseRestoreChannel()

	parseChannel, parseErr := OpenMessageChannel()
	if parseErr != nil {
		parseT.Fatalf("expected message channel, got %v", parseErr)
	}

	parseResultCh := make(chan []byte, 1)
	parseSubscription, parseErr := parseChannel.Port2().Subscribe(func(parseMessage MessagePortMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected nested binary message-port payload, got %v", parseErr2)
		}
		parsePayload, parseOk := parseMessage.Payload.(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected nested message-port binary payload map, got %T", parseMessage.Payload)
		}
		parseData, parseOk := parsePayload["data"].(map[string]any)
		if !parseOk {
			parseT.Fatalf("expected nested message-port binary data map, got %T", parsePayload["data"])
		}
		parseBlob, parseOk := parseData["blob"].([]byte)
		if !parseOk {
			parseT.Fatalf("expected nested message-port binary blob, got %T", parseData["blob"])
		}
		parseResultCh <- append([]byte(nil), parseBlob...)
	})
	if parseErr != nil {
		parseT.Fatalf("expected nested message-port subscription, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	if parseErr := parseChannel.Port1().Post(map[string]any{
		"data": map[string]any{
			"blob": []byte{9, 10, 11},
		},
	}); parseErr != nil {
		parseT.Fatalf("expected nested binary post over message port, got %v", parseErr)
	}

	select {
	case parseValue := <-parseResultCh:
		if !bytes.Equal(parseValue, []byte{9, 10, 11}) {
			parseT.Fatalf("expected nested message-port binary payload [9 10 11], got %v", parseValue)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for nested message-port binary payload")
	}
}

// TestTwoWorkersObserveSharedBufferUpdates verifies separate workers can observe
// updates through the same shared buffer without relaying the payload through
// the main thread.
func TestTwoWorkersObserveSharedBufferUpdates(parseT *testing.T) {
	parseRestore := setGlobalValue("crossOriginIsolated", true)
	defer parseRestore()

	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		parseT.Fatalf("expected shared-memory support inspection, got %v", parseErr)
	}
	if !parseSupport.HasSharedArrayBuffer || !parseSupport.HasAtomics {
		parseT.Skipf("shared-memory runtime support is unavailable: %+v", parseSupport)
	}

	var parseWorkerCount int
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseWorkerCount++
		parseWorkerID := parseWorkerCount
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			parseEvent.Set("ports", js.Global().Get("Array").New())
			for parseIndex := 0; parseIndex < parseMessageListeners.Length(); parseIndex++ {
				parseCallback := parseMessageListeners.Index(parseIndex)
				if parseCallback.IsUndefined() || parseCallback.IsNull() {
					continue
				}
				parseCallback.Invoke(parseEvent)
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parseSharedCtor := js.Global().Get("SharedArrayBuffer")
			if parseSharedCtor.Type() != js.TypeFunction || !parseArgs5[0].InstanceOf(parseSharedCtor) {
				return nil
			}
			parseView := js.Global().Get("Int32Array").New(parseArgs5[0])
			parseAtomics := js.Global().Get("Atomics")
			parseEnvelope := js.Global().Get("Object").New()
			parseEnvelope.Set("phase", "message")
			parseEnvelope.Set("name", "shared")
			parsePayload := js.Global().Get("Object").New()
			if parseWorkerID == 1 {
				parseAtomics.Call("store", parseView, 0, 64)
				parsePayload.Set("worker", "worker-a")
				parsePayload.Set("value", 64)
			} else {
				parsePayload.Set("worker", "worker-b")
				parsePayload.Set("value", parseAtomics.Call("load", parseView, 0).Int())
			}
			parseEnvelope.Set("payload", parsePayload)
			parseRaw.Call("__emitMessage", parseEnvelope)
			return nil
		})
		parseTerminate := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("terminate", parseTerminate)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreWorker := setGlobalValue("Worker", parseCtor)
	defer parseRestoreWorker()

	parseBuffer, parseErr := OpenSharedBuffer(4)
	if parseErr != nil {
		parseT.Fatalf("expected shared buffer, got %v", parseErr)
	}

	parseWorkerA, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/shared-a.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker A wrapper, got %v", parseErr)
	}
	parseWorkerB, parseErr := OpenWorker(context.Background(), WorkerOptions{URL: "/workers/shared-b.js"})
	if parseErr != nil {
		parseT.Fatalf("expected worker B wrapper, got %v", parseErr)
	}

	type sharedPayload struct {
		Worker string `json:"worker"`
		Value  int    `json:"value"`
	}
	parseWorkerACh := make(chan sharedPayload, 1)
	parseWorkerASub, parseErr := SubscribeDecodedWorker[sharedPayload](parseWorkerA, func(parseMessage DecodedWorkerMessage[sharedPayload], parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected worker A shared-buffer payload, got %v", parseErr2)
		}
		parseWorkerACh <- parseMessage.Payload
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker A subscription, got %v", parseErr)
	}
	defer parseWorkerASub.Cancel()

	parseWorkerBCh := make(chan sharedPayload, 1)
	parseWorkerBSub, parseErr := SubscribeDecodedWorker[sharedPayload](parseWorkerB, func(parseMessage DecodedWorkerMessage[sharedPayload], parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected worker B shared-buffer payload, got %v", parseErr2)
		}
		parseWorkerBCh <- parseMessage.Payload
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker B subscription, got %v", parseErr)
	}
	defer parseWorkerBSub.Cancel()

	if parseErr := parseWorkerA.Post(parseBuffer); parseErr != nil {
		parseT.Fatalf("expected worker A shared-buffer post, got %v", parseErr)
	}
	select {
	case parsePayload := <-parseWorkerACh:
		if parsePayload.Worker != "worker-a" || parsePayload.Value != 64 {
			parseT.Fatalf("expected worker A to store shared-buffer value 64, got %+v", parsePayload)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for worker A shared-buffer update")
	}

	if parseErr := parseWorkerB.Post(parseBuffer); parseErr != nil {
		parseT.Fatalf("expected worker B shared-buffer post, got %v", parseErr)
	}
	select {
	case parsePayload := <-parseWorkerBCh:
		if parsePayload.Worker != "worker-b" || parsePayload.Value != 64 {
			parseT.Fatalf("expected worker B to observe shared-buffer value 64, got %+v", parsePayload)
		}
	case <-time.After(100 * time.Millisecond):
		parseT.Fatal("timed out waiting for worker B shared-buffer observation")
	}
	if parseValue, parseErr := parseBuffer.LoadInt32(0); parseErr != nil || parseValue != 64 {
		parseT.Fatalf("expected shared-buffer value 64 after worker coordination, got value=%d err=%v", parseValue, parseErr)
	}
}

func TestOpenCrossTabChannelUsesBroadcastChannel(parseT *testing.T) {
	var parsePosted any
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			if parseArgs3[0].String() != "message" {
				return nil
			}
			parseCallback := parseArgs3[1]
			for parseI := 0; parseI < parseMessageListeners.Length(); parseI++ {
				parseCurrent := parseMessageListeners.Index(parseI)
				if !parseCurrent.IsUndefined() && !parseCurrent.IsNull() && parseCurrent.Equal(parseCallback) {
					parseMessageListeners.SetIndex(parseI, js.Null())
				}
			}
			return nil
		})
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			for parseI2 := 0; parseI2 < parseMessageListeners.Length(); parseI2++ {
				parseCallback2 := parseMessageListeners.Index(parseI2)
				if parseCallback2.IsUndefined() || parseCallback2.IsNull() {
					continue
				}
				parseCallback2.Invoke(parseEvent)
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parseValue, parseErr := jsValueToGo("test", "broadcast-channel", parseArgs5[0])
			if parseErr == nil {
				parsePosted = parseValue
			}
			parseRaw.Call("__emitMessage", parseArgs5[0])
			return nil
		})
		parseCloseFn := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			parseRaw.Set("__closed", true)
			return nil
		})
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("close", parseCloseFn)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreBroadcast := setGlobalValue("BroadcastChannel", parseCtor)
	defer parseRestoreBroadcast()

	parseChannel, parseErr2 := OpenCrossTabChannel(CrossTabChannelOptions{Name: "theme"})
	if parseErr2 != nil {
		parseT.Fatalf("expected cross-tab channel, got %v", parseErr2)
	}
	if parseChannel.Transport() != "broadcast-channel" {
		parseT.Fatalf("expected broadcast transport, got %q", parseChannel.Transport())
	}

	var parseReceived DecodedCrossTabEnvelope[struct {
		Theme string `json:"theme"`
	}]
	parseSubscription, parseErr2 := SubscribeDecodedCrossTab(parseChannel, func(parseMessage DecodedCrossTabEnvelope[struct {
		Theme string `json:"theme"`
	}], parseErr6 error) {
		if parseErr6 != nil {
			parseT.Fatalf("expected decoded broadcast payload, got %v", parseErr6)
		}
		parseReceived = parseMessage
	})
	if parseErr2 != nil {
		parseT.Fatalf("expected subscription to succeed, got %v", parseErr2)
	}
	defer parseSubscription.Cancel()

	if parseErr3 := parseChannel.Publish(map[string]any{"theme": "dark"}); parseErr3 != nil {
		parseT.Fatalf("expected publish to succeed, got %v", parseErr3)
	}
	if parseReceived.Payload.Theme != "dark" || parseReceived.Name != "theme" || parseReceived.Source == "" || parseReceived.Sequence != 1 {
		parseT.Fatalf("unexpected decoded broadcast envelope: %+v", parseReceived)
	}

	parseRawPosted, parseOk := parsePosted.(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected posted envelope map, got %#v", parsePosted)
	}
	if parseRawPosted["name"] != "theme" {
		parseT.Fatalf("expected published envelope name, got %#v", parseRawPosted)
	}
	parsePayload, parseOk := parseRawPosted["payload"].(map[string]any)
	if !parseOk || parsePayload["theme"] != "dark" {
		parseT.Fatalf("unexpected published payload: %#v", parseRawPosted["payload"])
	}

	if parseErr4 := parseChannel.Close(); parseErr4 != nil {
		parseT.Fatalf("expected close to succeed, got %v", parseErr4)
	}
	if parseErr5 := parseChannel.Publish(map[string]any{"theme": "light"}); !IsCode(parseErr5, CodeDisposed) {
		parseT.Fatalf("expected disposed error after close, got %v", parseErr5)
	}
}

func TestOpenCrossTabChannelFallsBackToStorageEvents(parseT *testing.T) {
	parseRestoreBroadcast := setGlobalValue("BroadcastChannel", js.Undefined())
	defer parseRestoreBroadcast()

	var (
		parseStorageListener js.Value
		parseLastSetKey      string
		parseLastSetValue    string
		parseLastRemovedKey  string
	)
	parseWindow := js.Global().Get("Object").New()
	parseAddEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if parseArgs[0].String() == "storage" {
			parseStorageListener = parseArgs[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if parseArgs2[0].String() == "storage" && parseStorageListener.Equal(parseArgs2[1]) {
			parseStorageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseStorage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return js.Null() })
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseLastSetKey = parseArgs4[0].String()
		parseLastSetValue = parseArgs4[1].String()
		return nil
	})
	defer setItemFn.Release()
	parseRemoveItemFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseLastRemovedKey = parseArgs5[0].String()
		return nil
	})
	defer parseRemoveItemFn.Release()
	clearFn := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
	defer clearFn.Release()
	parseKeyFn := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} { return js.Null() })
	defer parseKeyFn.Release()
	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)
	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseChannel, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "prefs"})
	if parseErr != nil {
		parseT.Fatalf("expected storage-fallback channel, got %v", parseErr)
	}
	if parseChannel.Transport() != "storage-event" {
		parseT.Fatalf("expected storage-event fallback, got %q", parseChannel.Transport())
	}

	var parseReceived DecodedCrossTabEnvelope[struct {
		Mode string `json:"mode"`
	}]
	parseSubscription, parseErr := SubscribeDecodedCrossTab(parseChannel, func(parseMessage DecodedCrossTabEnvelope[struct {
		Mode string `json:"mode"`
	}], parseErr5 error) {
		if parseErr5 != nil {
			parseT.Fatalf("expected decoded storage payload, got %v", parseErr5)
		}
		parseReceived = parseMessage
	})
	if parseErr != nil {
		parseT.Fatalf("expected storage subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	if parseErr2 := parseChannel.Publish(map[string]any{"mode": "dark"}); parseErr2 != nil {
		parseT.Fatalf("expected storage publish to succeed, got %v", parseErr2)
	}
	if parseLastSetKey != "__gwc_cross_tab__:prefs" || parseLastRemovedKey != "__gwc_cross_tab__:prefs" {
		parseT.Fatalf("unexpected storage keys: set=%q removed=%q", parseLastSetKey, parseLastRemovedKey)
	}
	if parseLastSetValue == "" {
		parseT.Fatal("expected storage fallback to serialize the published envelope")
	}

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("key", "__gwc_cross_tab__:prefs")
	parseEvent.Set("newValue", `{"name":"prefs","payload":{"mode":"light"},"source":"tab-2","sequence":7,"sentAt":"2026-03-18T16:00:00Z"}`)
	parseStorageListener.Invoke(parseEvent)
	if parseReceived.Payload.Mode != "light" || parseReceived.Source != "tab-2" || parseReceived.Sequence != 7 || parseReceived.Name != "prefs" {
		parseT.Fatalf("unexpected decoded storage envelope: %+v", parseReceived)
	}

	if parseErr3 := parseChannel.Close(); parseErr3 != nil {
		parseT.Fatalf("expected storage-fallback close to succeed, got %v", parseErr3)
	}
	if parseErr4 := parseChannel.Publish(map[string]any{"mode": "reset"}); !IsCode(parseErr4, CodeDisposed) {
		parseT.Fatalf("expected disposed error after close, got %v", parseErr4)
	}
}

func TestMultiClientStorageFallbackLifecycleFlow(parseT *testing.T) {
	parseRestoreBroadcast := setGlobalValue("BroadcastChannel", js.Undefined())
	defer parseRestoreBroadcast()

	var parseStorageListener js.Value
	parseWindow := js.Global().Get("Object").New()
	parseAddEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if parseArgs[0].String() == "storage" {
			parseStorageListener = parseArgs[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if parseArgs2[0].String() == "storage" && parseStorageListener.Equal(parseArgs2[1]) {
			parseStorageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseStorage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return js.Null() })
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil })
	defer setItemFn.Release()
	parseRemoveItemFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil })
	defer parseRemoveItemFn.Release()
	clearFn := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
	defer clearFn.Release()
	parseKeyFn := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} { return js.Null() })
	defer parseKeyFn.Release()
	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)
	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseChannel, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients-fallback"})
	if parseErr != nil {
		parseT.Fatalf("expected storage-fallback channel, got %v", parseErr)
	}
	if parseChannel.Transport() != "storage-event" {
		parseT.Fatalf("expected storage-event fallback transport, got %q", parseChannel.Transport())
	}

	type peerState struct {
		hellos       int
		disconnected bool
		expired      bool
		lastSentAt   time.Time
	}
	parsePeers := map[string]peerState{}
	parseExpirePeers := func(parseNow time.Time, parseLease time.Duration) {
		for parseId, parseState := range parsePeers {
			if parseState.disconnected || parseState.lastSentAt.IsZero() {
				continue
			}
			if parseNow.Sub(parseState.lastSentAt) > parseLease {
				parseState.expired = true
				parsePeers[parseId] = parseState
			}
		}
	}

	parseSubscription, parseErr := SubscribeClientMessages(parseChannel, func(parseMessage ClientMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected decoded client storage message, got %v", parseErr2)
		}
		if parseMessage.Topic != ClientPresenceTopic {
			return
		}
		parseState2 := parsePeers[parseMessage.Source.ID]
		switch parseMessage.Kind {
		case ClientHello:
			parseState2.hellos++
			parseState2.disconnected = false
			parseState2.expired = false
			parseState2.lastSentAt = parseMessage.SentAt
		case ClientGoodbye:
			parseState2.disconnected = true
			parseState2.lastSentAt = parseMessage.SentAt
		}
		parsePeers[parseMessage.Source.ID] = parseState2
	})
	if parseErr != nil {
		parseT.Fatalf("expected storage multi-client subscription, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEmitStorage := func(parseData string) {
		parseEvent := js.Global().Get("Object").New()
		parseEvent.Set("key", "__gwc_cross_tab__:clients-fallback")
		parseEvent.Set("newValue", parseData)
		parseStorageListener.Invoke(parseEvent)
	}

	parseEmitStorage(`{"name":"clients-fallback","payload":{"kind":"hello","topic":"clients","source":{"id":"storefront-2","app":"atlas","surface":"tab"},"sentAt":"2026-03-19T10:00:00Z"},"source":"tab-2","sequence":1,"sentAt":"2026-03-19T10:00:00Z"}`)
	parseEmitStorage(`{"name":"clients-fallback","payload":{"kind":"hello","topic":"clients","source":{"id":"storefront-2","app":"atlas","surface":"tab"},"sentAt":"2026-03-19T10:00:02Z"},"source":"tab-2","sequence":2,"sentAt":"2026-03-19T10:00:02Z"}`)
	parseState3 := parsePeers["storefront-2"]
	if parseState3.hellos != 2 || parseState3.disconnected || parseState3.expired {
		parseT.Fatalf("expected duplicate fallback hello traffic to be tolerated, got %+v", parseState3)
	}

	parseExpirePeers(time.Date(2026, 3, 19, 10, 0, 6, 0, time.UTC), 3*time.Second)
	parseState3 = parsePeers["storefront-2"]
	if !parseState3.expired {
		parseT.Fatalf("expected fallback peer lease to expire after inactivity, got %+v", parseState3)
	}

	parseEmitStorage(`{"name":"clients-fallback","payload":{"kind":"hello","topic":"clients","source":{"id":"storefront-2","app":"atlas","surface":"tab"},"sentAt":"2026-03-19T10:00:07Z"},"source":"tab-2","sequence":3,"sentAt":"2026-03-19T10:00:07Z"}`)
	parseState3 = parsePeers["storefront-2"]
	if parseState3.hellos != 3 || parseState3.expired || parseState3.disconnected {
		parseT.Fatalf("expected fallback peer to reconnect on fresh hello, got %+v", parseState3)
	}

	parseEmitStorage(`{"name":"clients-fallback","payload":{"kind":"goodbye","topic":"clients","source":{"id":"storefront-2","app":"atlas","surface":"tab"},"sentAt":"2026-03-19T10:00:08Z"},"source":"tab-2","sequence":4,"sentAt":"2026-03-19T10:00:08Z"}`)
	parseState3 = parsePeers["storefront-2"]
	if !parseState3.disconnected {
		parseT.Fatalf("expected fallback peer goodbye to mark the peer disconnected, got %+v", parseState3)
	}
}

func TestPublishClientBinaryCrossTabUsesBroadcastChannel(parseT *testing.T) {
	var parsePosted js.Value
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseMessageListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseMessageListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			if parseArgs3[0].String() != "message" {
				return nil
			}
			parseCallback := parseArgs3[1]
			for parseI := 0; parseI < parseMessageListeners.Length(); parseI++ {
				parseCurrent := parseMessageListeners.Index(parseI)
				if !parseCurrent.IsUndefined() && !parseCurrent.IsNull() && parseCurrent.Equal(parseCallback) {
					parseMessageListeners.SetIndex(parseI, js.Null())
				}
			}
			return nil
		})
		parseEmitMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			for parseI2 := 0; parseI2 < parseMessageListeners.Length(); parseI2++ {
				parseCallback2 := parseMessageListeners.Index(parseI2)
				if parseCallback2.IsUndefined() || parseCallback2.IsNull() {
					continue
				}
				parseCallback2.Invoke(parseEvent)
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parsePosted = parseArgs5[0]
			parseRaw.Call("__emitMessage", parseArgs5[0])
			return nil
		})
		parseCloseFn := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			parseRaw.Set("__closed", true)
			return nil
		})
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("close", parseCloseFn)
		parseRaw.Set("__emitMessage", parseEmitMessage)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreBroadcast := setGlobalValue("BroadcastChannel", parseCtor)
	defer parseRestoreBroadcast()

	parseChannel, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "assets"})
	if parseErr != nil {
		parseT.Fatalf("expected broadcast cross-tab channel, got %v", parseErr)
	}

	var parseReceived ClientMessage
	parseSubscription, parseErr := SubscribeClientMessages(parseChannel, func(parseMessage ClientMessage, parseErr3 error) {
		if parseErr3 != nil {
			parseT.Fatalf("expected decoded client message, got %v", parseErr3)
		}
		parseReceived = parseMessage
	})
	if parseErr != nil {
		parseT.Fatalf("expected client-message subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseSelf := ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}
	parseBinaryPayload := []byte{1, 2, 3, 4}
	if parseErr2 := PublishClientBinaryCrossTab(parseChannel, "asset:preview", parseSelf, ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: parseBinaryPayload}); parseErr2 != nil {
		parseT.Fatalf("expected binary cross-tab publish to succeed, got %v", parseErr2)
	}

	if parsePosted.IsUndefined() || parsePosted.IsNull() {
		parseT.Fatal("expected broadcast channel to capture a posted payload")
	}
	parsePostedMessage := parsePosted.Get("payload")
	if parsePostedMessage.Get("encoding").String() != "binary" || parsePostedMessage.Get("contentType").String() != "application/octet-stream" {
		parseT.Fatalf("unexpected posted binary metadata: %s %s", parsePostedMessage.Get("encoding").String(), parsePostedMessage.Get("contentType").String())
	}
	parsePostedBytes := make([]byte, parsePostedMessage.Get("payload").Length())
	js.CopyBytesToGo(parsePostedBytes, parsePostedMessage.Get("payload"))
	if string(parsePostedBytes) != string(parseBinaryPayload) {
		parseT.Fatalf("unexpected posted binary bytes: %v", parsePostedBytes)
	}

	parseDecodedBytes, parseOk := parseReceived.Payload.([]byte)
	if !parseOk {
		parseT.Fatalf("expected received payload bytes, got %T", parseReceived.Payload)
	}
	if parseReceived.Encoding != ClientPayloadBinary || parseReceived.ContentType != "application/octet-stream" || string(parseDecodedBytes) != string(parseBinaryPayload) {
		parseT.Fatalf("unexpected received binary message: %+v payload=%v", parseReceived, parseDecodedBytes)
	}
}

func TestPublishClientHelloCarriesDefaultCapabilitiesByTransport(parseT *testing.T) {
	var parsePosted js.Value
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRaw := js.Global().Get("Object").New()
		parseListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil })
		parsePostMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parsePosted = parseArgs4[0]
			return nil
		})
		parseCloseFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil })
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("close", parseCloseFn)
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreBroadcast := setGlobalValue("BroadcastChannel", parseCtor)
	defer parseRestoreBroadcast()

	parseChannel, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients"})
	if parseErr != nil {
		parseT.Fatalf("expected cross-tab channel, got %v", parseErr)
	}
	if parseErr2 := PublishClientHello(parseChannel, ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}); parseErr2 != nil {
		parseT.Fatalf("expected hello publish to succeed, got %v", parseErr2)
	}
	parseCapabilities := parsePosted.Get("payload").Get("capabilities")
	if parseCapabilities.Get("protocolVersion").String() != "v1" {
		parseT.Fatalf("expected default protocol version, got %q", parseCapabilities.Get("protocolVersion").String())
	}
	parseEncodings := parseCapabilities.Get("encodings")
	if parseEncodings.Length() != 2 || parseEncodings.Index(0).String() != "json" || parseEncodings.Index(1).String() != "binary" {
		parseT.Fatalf("expected broadcast hello to advertise json and binary, got %#v", parseEncodings)
	}

	parseRestoreBroadcastFallback := setGlobalValue("BroadcastChannel", js.Undefined())
	defer parseRestoreBroadcastFallback()
	parseWindow := js.Global().Get("Object").New()
	parseAddEventListener2 := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
	defer parseAddEventListener2.Release()
	parseRemoveEventListener2 := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} { return nil })
	defer parseRemoveEventListener2.Release()
	parseWindow.Set("addEventListener", parseAddEventListener2)
	parseWindow.Set("removeEventListener", parseRemoveEventListener2)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()
	parseStorage := js.Global().Get("Object").New()
	var parseStored string
	getItemFn := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} { return js.Null() })
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} {
		parseStored = parseArgs9[1].String()
		return nil
	})
	defer setItemFn.Release()
	parseRemoveItemFn := js.FuncOf(func(parseThis10 js.Value, parseArgs10 []js.Value) interface{} { return nil })
	defer parseRemoveItemFn.Release()
	clearFn := js.FuncOf(func(parseThis11 js.Value, parseArgs11 []js.Value) interface{} { return nil })
	defer clearFn.Release()
	parseKeyFn := js.FuncOf(func(parseThis12 js.Value, parseArgs12 []js.Value) interface{} { return js.Null() })
	defer parseKeyFn.Release()
	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)
	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseFallbackChannel, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients-fallback"})
	if parseErr != nil {
		parseT.Fatalf("expected storage fallback channel, got %v", parseErr)
	}
	if parseErr3 := PublishClientHello(parseFallbackChannel, ClientIdentity{ID: "storefront-2", App: "atlas", Surface: "tab"}); parseErr3 != nil {
		parseT.Fatalf("expected fallback hello publish to succeed, got %v", parseErr3)
	}
	if !strings.Contains(parseStored, `"encodings":["json"]`) {
		parseT.Fatalf("expected storage fallback hello to advertise json-only encoding, got %s", parseStored)
	}
}

func TestPublishClientBinaryCrossTabRejectsStorageFallback(parseT *testing.T) {
	parseRestoreBroadcast := setGlobalValue("BroadcastChannel", js.Undefined())
	defer parseRestoreBroadcast()

	parseWindow := js.Global().Get("Object").New()
	parseAddEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} { return nil })
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil })
	defer parseRemoveEventListener.Release()
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseStorage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return js.Null() })
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil })
	defer setItemFn.Release()
	parseRemoveItemFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return nil })
	defer parseRemoveItemFn.Release()
	clearFn := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} { return nil })
	defer clearFn.Release()
	parseKeyFn := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} { return js.Null() })
	defer parseKeyFn.Release()
	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)
	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseChannel, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "prefs"})
	if parseErr != nil {
		parseT.Fatalf("expected storage-fallback channel, got %v", parseErr)
	}

	parseErr = PublishClientBinaryCrossTab(parseChannel, "asset:preview", ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}, ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{9, 8, 7}})
	if !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected invalid binary transport error, got %v", parseErr)
	}
}

func TestMultiClientBroadcastLateJoinReconnectAndGoodbye(parseT *testing.T) {
	type mockBroadcastChannel struct {
		raw       js.Value
		listeners js.Value
	}

	parseChannelsByName := map[string][]mockBroadcastChannel{}
	parseCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseName := parseArgs[0].String()
		parseRaw := js.Global().Get("Object").New()
		parseListeners := js.Global().Get("Array").New()
		parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			if parseArgs2[0].String() == "message" {
				parseListeners.Call("push", parseArgs2[1])
			}
			return nil
		})
		parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			if parseArgs3[0].String() != "message" {
				return nil
			}
			parseCallback := parseArgs3[1]
			for parseI := 0; parseI < parseListeners.Length(); parseI++ {
				parseCurrent := parseListeners.Index(parseI)
				if !parseCurrent.IsUndefined() && !parseCurrent.IsNull() && parseCurrent.Equal(parseCallback) {
					parseListeners.SetIndex(parseI, js.Null())
				}
			}
			return nil
		})
		parsePostMessage := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseEvent := js.Global().Get("Object").New()
			parseEvent.Set("data", parseArgs4[0])
			for _, parseChannel := range parseChannelsByName[parseName] {
				if parseClosed := parseChannel.raw.Get("__closed"); !parseClosed.IsUndefined() && !parseClosed.IsNull() && parseClosed.Bool() {
					continue
				}
				for parseI2 := 0; parseI2 < parseChannel.listeners.Length(); parseI2++ {
					parseCallback2 := parseChannel.listeners.Index(parseI2)
					if parseCallback2.IsUndefined() || parseCallback2.IsNull() {
						continue
					}
					parseCallback2.Invoke(parseEvent)
				}
			}
			return nil
		})
		parseCloseFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			parseRaw.Set("__closed", true)
			return nil
		})
		parseRaw.Set("addEventListener", parseAddEventListener)
		parseRaw.Set("removeEventListener", parseRemoveEventListener)
		parseRaw.Set("postMessage", parsePostMessage)
		parseRaw.Set("close", parseCloseFn)
		parseChannelsByName[parseName] = append(parseChannelsByName[parseName], mockBroadcastChannel{raw: parseRaw, listeners: parseListeners})
		return parseRaw
	})
	defer parseCtor.Release()
	parseRestoreBroadcast := setGlobalValue("BroadcastChannel", parseCtor)
	defer parseRestoreBroadcast()

	parseAlpha, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients"})
	if parseErr != nil {
		parseT.Fatalf("expected alpha cross-tab channel, got %v", parseErr)
	}
	defer parseAlpha.Close()

	parseAlphaSelf := ClientIdentity{ID: "alpha-1", App: "atlas", Surface: "tab-a"}
	parseBetaSelf := ClientIdentity{ID: "beta-1", App: "atlas", Surface: "tab-b"}

	var parseBetaHellos int
	var isBetaSawGoodbye bool
	var isBetaSawReconnect bool
	var isBetaSawResult bool

	parseAlphaSubscription, parseErr := SubscribeClientMessages(parseAlpha, func(parseMessage ClientMessage, parseErr8 error) {
		if parseErr8 != nil {
			parseT.Fatalf("expected alpha subscription to decode messages, got %v", parseErr8)
		}
		if parseMessage.Kind == ClientQuery && parseMessage.Topic == ClientPresenceTopic && parseMessage.Source.ID == parseBetaSelf.ID {
			if parsePublishErr := PublishClientResult(parseAlpha, ClientPresenceTopic, parseAlphaSelf, parseBetaSelf.ID, map[string]any{"peer": parseAlphaSelf.Surface}); parsePublishErr != nil {
				parseT.Fatalf("expected alpha to answer beta query, got %v", parsePublishErr)
			}
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected alpha multi-client subscription, got %v", parseErr)
	}
	defer parseAlphaSubscription.Cancel()

	if parseErr2 := PublishClientHello(parseAlpha, parseAlphaSelf); parseErr2 != nil {
		parseT.Fatalf("expected alpha hello to succeed, got %v", parseErr2)
	}

	parseBeta, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients"})
	if parseErr != nil {
		parseT.Fatalf("expected beta cross-tab channel, got %v", parseErr)
	}
	defer parseBeta.Close()

	parseBetaSubscription, parseErr := SubscribeClientMessages(parseBeta, func(parseMessage2 ClientMessage, parseErr9 error) {
		if parseErr9 != nil {
			parseT.Fatalf("expected beta subscription to decode messages, got %v", parseErr9)
		}
		if parseMessage2.Source.ID != parseAlphaSelf.ID {
			return
		}
		switch parseMessage2.Kind {
		case ClientHello:
			parseBetaHellos++
			if isBetaSawGoodbye {
				isBetaSawReconnect = true
			}
		case ClientGoodbye:
			isBetaSawGoodbye = true
		case ClientResult:
			if parseMessage2.Target == parseBetaSelf.ID {
				isBetaSawResult = true
			}
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected beta multi-client subscription, got %v", parseErr)
	}
	defer parseBetaSubscription.Cancel()

	if parseErr3 := PublishClientHello(parseAlpha, parseAlphaSelf); parseErr3 != nil {
		parseT.Fatalf("expected duplicate alpha hello to succeed, got %v", parseErr3)
	}
	if parseErr4 := PublishClientQuery(parseBeta, ClientPresenceTopic, parseBetaSelf); parseErr4 != nil {
		parseT.Fatalf("expected beta late-join discovery query to succeed, got %v", parseErr4)
	}
	if parseErr5 := PublishClientGoodbye(parseAlpha, parseAlphaSelf); parseErr5 != nil {
		parseT.Fatalf("expected alpha goodbye to succeed, got %v", parseErr5)
	}
	if parseErr6 := parseAlpha.Close(); parseErr6 != nil {
		parseT.Fatalf("expected alpha close to succeed, got %v", parseErr6)
	}

	parseAlphaReconnect, parseErr := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients"})
	if parseErr != nil {
		parseT.Fatalf("expected alpha reconnect channel, got %v", parseErr)
	}
	defer parseAlphaReconnect.Close()
	if parseErr7 := PublishClientHello(parseAlphaReconnect, parseAlphaSelf); parseErr7 != nil {
		parseT.Fatalf("expected alpha reconnect hello to succeed, got %v", parseErr7)
	}

	if parseBetaHellos < 2 {
		parseT.Fatalf("expected beta to observe duplicate or reconnect hello traffic, saw %d hellos", parseBetaHellos)
	}
	if !isBetaSawResult {
		parseT.Fatal("expected beta to receive a targeted discovery result from alpha")
	}
	if !isBetaSawGoodbye {
		parseT.Fatal("expected beta to observe alpha goodbye before reconnect")
	}
	if !isBetaSawReconnect {
		parseT.Fatal("expected beta to observe alpha reconnect hello after goodbye")
	}
}

func TestOpenSecondaryWindowChannelPublishesAndReceivesMessages(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("origin", "https://app.example.test")
	parseWindow.Set("location", parseLocation)
	var parseOpened js.Value
	parseOpenFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOpened = js.Global().Get("Object").New()
		parseOpened.Set("closed", false)
		parseOpened.Set("focus", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseOpened.Set("__focused", true)
			return nil
		}))
		parseOpened.Set("close", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			parseOpened.Set("closed", true)
			return nil
		}))
		parseOpened.Set("postMessage", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseOpened.Set("__posted", parseArgs4[0])
			parseOpened.Set("__targetOrigin", parseArgs4[1].String())
			return nil
		}))
		return parseOpened
	})
	defer parseOpenFn.Release()
	var parseMessageListener js.Value
	parseAddEventListener := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		if parseArgs5[0].String() == "message" {
			parseMessageListener = parseArgs5[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		if parseArgs6[0].String() == "message" && parseMessageListener.Equal(parseArgs6[1]) {
			parseMessageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("open", parseOpenFn)
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseChannel, parseErr := OpenSecondaryWindowChannel(WindowChannelOptions{
		URL:  "/popup.html",
		Name: "inspector",
	})
	if parseErr != nil {
		parseT.Fatalf("expected popup channel, got %v", parseErr)
	}
	if parseChannel.TargetOrigin() != "https://app.example.test" {
		parseT.Fatalf("expected same-origin default target, got %q", parseChannel.TargetOrigin())
	}

	var parseReceived DecodedWindowEnvelope[struct {
		View string `json:"view"`
	}]
	parseSubscription, parseErr := SubscribeDecodedWindow(parseChannel, func(parseMessage DecodedWindowEnvelope[struct {
		View string `json:"view"`
	}], parseErr6 error) {
		if parseErr6 != nil {
			parseT.Fatalf("expected decoded popup message, got %v", parseErr6)
		}
		parseReceived = parseMessage
	})
	if parseErr != nil {
		parseT.Fatalf("expected popup subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	if parseErr2 := parseChannel.Publish(map[string]any{"view": "orders"}); parseErr2 != nil {
		parseT.Fatalf("expected popup publish to succeed, got %v", parseErr2)
	}
	if parseOpened.Get("__targetOrigin").String() != "https://app.example.test" {
		parseT.Fatalf("expected popup postMessage target origin, got %q", parseOpened.Get("__targetOrigin").String())
	}

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("source", parseOpened)
	parseEvent.Set("origin", "https://app.example.test")
	parseEvent.Set("data", js.Global().Get("Object").New())
	parseEvent.Get("data").Set("name", "inspector")
	parseEvent.Get("data").Set("source", "popup-1")
	parseEvent.Get("data").Set("sentAt", "2026-03-18T16:15:00Z")
	parseEvent.Get("data").Set("payload", js.Global().Get("Object").New())
	parseEvent.Get("data").Get("payload").Set("view", "catalog")
	parseMessageListener.Invoke(parseEvent)
	if parseReceived.Payload.View != "catalog" || parseReceived.Name != "inspector" || parseReceived.Source != "popup-1" {
		parseT.Fatalf("unexpected popup message envelope: %+v", parseReceived)
	}

	if parseErr3 := parseChannel.Focus(); parseErr3 != nil {
		parseT.Fatalf("expected popup focus to succeed, got %v", parseErr3)
	}
	if !parseOpened.Get("__focused").Bool() {
		parseT.Fatal("expected popup focus helper to call the window focus method")
	}
	if parseChannel.Closed() {
		parseT.Fatal("expected popup to start open")
	}
	if parseErr4 := parseChannel.Close(); parseErr4 != nil {
		parseT.Fatalf("expected popup close to succeed, got %v", parseErr4)
	}
	if !parseChannel.Closed() {
		parseT.Fatal("expected popup close to mark the window handle closed")
	}
	if parseErr5 := parseChannel.Publish(map[string]any{"view": "retry"}); !IsCode(parseErr5, CodeDisposed) {
		parseT.Fatalf("expected disposed error after popup close, got %v", parseErr5)
	}
}

func TestPublishClientBinaryWindowRoundTripsPayload(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("origin", "https://app.example.test")
	parseWindow.Set("location", parseLocation)
	var parseOpened js.Value
	parseOpenFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOpened = js.Global().Get("Object").New()
		parseOpened.Set("closed", false)
		parseOpened.Set("focus", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
		parseOpened.Set("close", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			parseOpened.Set("closed", true)
			return nil
		}))
		parseOpened.Set("postMessage", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseOpened.Set("__posted", parseArgs4[0])
			parseOpened.Set("__targetOrigin", parseArgs4[1].String())
			return nil
		}))
		return parseOpened
	})
	defer parseOpenFn.Release()
	var parseMessageListener js.Value
	parseAddEventListener := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		if parseArgs5[0].String() == "message" {
			parseMessageListener = parseArgs5[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		if parseArgs6[0].String() == "message" && parseMessageListener.Equal(parseArgs6[1]) {
			parseMessageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("open", parseOpenFn)
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseChannel, parseErr := OpenSecondaryWindowChannel(WindowChannelOptions{URL: "/popup.html", Name: "inspector"})
	if parseErr != nil {
		parseT.Fatalf("expected popup channel, got %v", parseErr)
	}

	var parseReceived ClientMessage
	parseSubscription, parseErr := SubscribeClientWindowMessages(parseChannel, func(parseMessage ClientMessage, parseErr3 error) {
		if parseErr3 != nil {
			parseT.Fatalf("expected decoded client window message, got %v", parseErr3)
		}
		parseReceived = parseMessage
	})
	if parseErr != nil {
		parseT.Fatalf("expected client-window subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseSelf := ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}
	parseBinaryPayload := []byte{5, 6, 7, 8}
	if parseErr2 := PublishClientBinaryWindow(parseChannel, "asset:preview", parseSelf, "popup-1", ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: parseBinaryPayload}); parseErr2 != nil {
		parseT.Fatalf("expected binary window publish to succeed, got %v", parseErr2)
	}

	parsePosted := parseOpened.Get("__posted")
	if parsePosted.IsUndefined() || parsePosted.IsNull() {
		parseT.Fatal("expected posted popup message")
	}
	parsePostedMessage := parsePosted.Get("payload")
	if parsePostedMessage.Get("encoding").String() != "binary" || parsePostedMessage.Get("contentType").String() != "application/octet-stream" || parsePostedMessage.Get("target").String() != "popup-1" {
		parseT.Fatalf("unexpected posted binary window metadata: %#v", parsePostedMessage)
	}
	parsePostedBytes := make([]byte, parsePostedMessage.Get("payload").Length())
	js.CopyBytesToGo(parsePostedBytes, parsePostedMessage.Get("payload"))
	if string(parsePostedBytes) != string(parseBinaryPayload) {
		parseT.Fatalf("unexpected posted binary window bytes: %v", parsePostedBytes)
	}

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("source", parseOpened)
	parseEvent.Set("origin", "https://app.example.test")
	parseEvent.Set("data", parsePosted)
	parseMessageListener.Invoke(parseEvent)

	parseDecodedBytes, parseOk := parseReceived.Payload.([]byte)
	if !parseOk {
		parseT.Fatalf("expected received popup payload bytes, got %T", parseReceived.Payload)
	}
	if parseReceived.Encoding != ClientPayloadBinary || parseReceived.ContentType != "application/octet-stream" || parseReceived.Target != "popup-1" || string(parseDecodedBytes) != string(parseBinaryPayload) {
		parseT.Fatalf("unexpected received popup binary message: %+v payload=%v", parseReceived, parseDecodedBytes)
	}
}

func TestMultiClientWindowOrphanedPopupReportsDisposed(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("origin", "https://app.example.test")
	parseWindow.Set("location", parseLocation)
	var parseOpened js.Value
	parseOpenFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOpened = js.Global().Get("Object").New()
		parseOpened.Set("closed", false)
		parseOpened.Set("focus", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
		parseOpened.Set("close", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			parseOpened.Set("closed", true)
			return nil
		}))
		parseOpened.Set("postMessage", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseOpened.Set("__posted", parseArgs4[0])
			return nil
		}))
		return parseOpened
	})
	defer parseOpenFn.Release()
	var parseMessageListener js.Value
	parseAddEventListener := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		if parseArgs5[0].String() == "message" {
			parseMessageListener = parseArgs5[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		if parseArgs6[0].String() == "message" && parseMessageListener.Equal(parseArgs6[1]) {
			parseMessageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("open", parseOpenFn)
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseChannel, parseErr := OpenSecondaryWindowChannel(WindowChannelOptions{URL: "/popup.html", Name: "inspector"})
	if parseErr != nil {
		parseT.Fatalf("expected popup channel, got %v", parseErr)
	}

	var isSawHello bool
	parseSubscription, parseErr := SubscribeClientWindowMessages(parseChannel, func(parseMessage ClientMessage, parseErr2 error) {
		if parseErr2 != nil {
			parseT.Fatalf("expected decoded popup lifecycle message, got %v", parseErr2)
		}
		if parseMessage.Kind == ClientHello && parseMessage.Source.ID == "popup-1" {
			isSawHello = true
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected popup multi-client subscription, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("source", parseOpened)
	parseEvent.Set("origin", "https://app.example.test")
	parseEvent.Set("data", js.Global().Get("Object").New())
	parseEvent.Get("data").Set("name", "inspector")
	parseEvent.Get("data").Set("source", "popup-window")
	parseEvent.Get("data").Set("payload", js.Global().Get("Object").New())
	parseEvent.Get("data").Get("payload").Set("kind", "hello")
	parseEvent.Get("data").Get("payload").Set("topic", "clients")
	parseEvent.Get("data").Get("payload").Set("source", js.Global().Get("Object").New())
	parseEvent.Get("data").Get("payload").Get("source").Set("id", "popup-1")
	parseEvent.Get("data").Get("payload").Get("source").Set("app", "atlas")
	parseEvent.Get("data").Get("payload").Get("source").Set("surface", "popup")
	parseMessageListener.Invoke(parseEvent)
	if !isSawHello {
		parseT.Fatal("expected popup hello to be observed before orphaning the handle")
	}

	parseOpened.Set("closed", true)
	if !parseChannel.Closed() {
		parseT.Fatal("expected popup channel to report closed after the peer handle closes")
	}
	parseErr = PublishClientWindowMessage(parseChannel, ClientMessage{Kind: ClientGoodbye, Topic: ClientPresenceTopic, Source: ClientIdentity{ID: "opener-1", App: "atlas", Surface: "tab"}, Target: "popup-1"})
	if !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected disposed error after popup orphaning, got %v", parseErr)
	}
	parseErr = PublishClientBinaryWindow(parseChannel, "asset:preview", ClientIdentity{ID: "opener-1", App: "atlas", Surface: "tab"}, "popup-1", ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{1, 2}})
	if !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected disposed error for binary publish after popup orphaning, got %v", parseErr)
	}
}

func TestWindowOpenerChannelUsesOpenerHandle(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("origin", "https://app.example.test")
	parseWindow.Set("location", parseLocation)
	parseOpener := js.Global().Get("Object").New()
	parseOpener.Set("closed", false)
	parseOpener.Set("postMessage", js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOpener.Set("__posted", parseArgs[0])
		parseOpener.Set("__targetOrigin", parseArgs[1].String())
		return nil
	}))
	parseWindow.Set("opener", parseOpener)
	var parseMessageListener js.Value
	parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if parseArgs2[0].String() == "message" {
			parseMessageListener = parseArgs2[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		if parseArgs3[0].String() == "message" && parseMessageListener.Equal(parseArgs3[1]) {
			parseMessageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseChannel, parseErr := OpenWindowOpenerChannel(WindowChannelOptions{Name: "inspector"})
	if parseErr != nil {
		parseT.Fatalf("expected opener channel, got %v", parseErr)
	}
	if parseErr2 := parseChannel.Publish(map[string]any{"route": "/orders"}); parseErr2 != nil {
		parseT.Fatalf("expected opener publish to succeed, got %v", parseErr2)
	}
	if parseOpener.Get("__targetOrigin").String() != "https://app.example.test" {
		parseT.Fatalf("expected opener target origin, got %q", parseOpener.Get("__targetOrigin").String())
	}
	if parseErr3 := parseChannel.Close(); !IsCode(parseErr3, CodeUnavailable) {
		parseT.Fatalf("expected opener channel close to stay unavailable, got %v", parseErr3)
	}

	var parseReceivedName string
	parseSubscription, parseErr := parseChannel.Subscribe(func(parseMessage WindowEnvelope, parseErr4 error) {
		if parseErr4 != nil {
			parseT.Fatalf("expected opener message to decode, got %v", parseErr4)
		}
		parseReceivedName = parseMessage.Name
	})
	if parseErr != nil {
		parseT.Fatalf("expected opener subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("source", parseOpener)
	parseEvent.Set("origin", "https://app.example.test")
	parseEvent.Set("data", js.Global().Get("Object").New())
	parseEvent.Get("data").Set("name", "inspector")
	parseMessageListener.Invoke(parseEvent)
	if parseReceivedName != "inspector" {
		parseT.Fatalf("expected opener message name, got %q", parseReceivedName)
	}
}

func TestMultiClientWindowOpenerLifecycleFlow(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("origin", "https://app.example.test")
	parseWindow.Set("location", parseLocation)
	parseOpener := js.Global().Get("Object").New()
	parseOpener.Set("closed", false)
	parseOpener.Set("postMessage", js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOpener.Set("__posted", parseArgs[0])
		parseOpener.Set("__targetOrigin", parseArgs[1].String())
		return nil
	}))
	parseWindow.Set("opener", parseOpener)
	var parseMessageListener js.Value
	parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if parseArgs2[0].String() == "message" {
			parseMessageListener = parseArgs2[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		if parseArgs3[0].String() == "message" && parseMessageListener.Equal(parseArgs3[1]) {
			parseMessageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseChannel, parseErr := OpenWindowOpenerChannel(WindowChannelOptions{Name: "inspector"})
	if parseErr != nil {
		parseT.Fatalf("expected opener channel, got %v", parseErr)
	}

	parsePopupSelf := ClientIdentity{ID: "popup-1", App: "atlas", Surface: "popup"}
	if parseErr2 := PublishClientHelloWindow(parseChannel, parsePopupSelf); parseErr2 != nil {
		parseT.Fatalf("expected opener hello publish to succeed, got %v", parseErr2)
	}
	parsePosted := parseOpener.Get("__posted")
	if parsePosted.Get("payload").Get("kind").String() != "hello" || parsePosted.Get("payload").Get("topic").String() != "clients" {
		parseT.Fatalf("expected opener hello payload, got %#v", parsePosted)
	}
	if parsePosted.Get("payload").Get("capabilities").Get("protocolVersion").String() != "v1" {
		parseT.Fatalf("expected opener hello to advertise capabilities, got %#v", parsePosted.Get("payload").Get("capabilities"))
	}

	if parseErr3 := PublishClientGoodbyeWindow(parseChannel, parsePopupSelf); parseErr3 != nil {
		parseT.Fatalf("expected opener goodbye publish to succeed, got %v", parseErr3)
	}
	parsePosted = parseOpener.Get("__posted")
	if parsePosted.Get("payload").Get("kind").String() != "goodbye" {
		parseT.Fatalf("expected opener goodbye payload, got %#v", parsePosted)
	}

	var isSawHello bool
	var isSawGoodbye bool
	parseSubscription, parseErr := SubscribeClientWindowMessages(parseChannel, func(parseMessage ClientMessage, parseErr4 error) {
		if parseErr4 != nil {
			parseT.Fatalf("expected decoded opener client message, got %v", parseErr4)
		}
		switch parseMessage.Kind {
		case ClientHello:
			if parseMessage.Source.ID == "opener-1" {
				isSawHello = true
			}
		case ClientGoodbye:
			if parseMessage.Source.ID == "opener-1" {
				isSawGoodbye = true
			}
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected opener multi-client subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEmitWindowMessage := func(parseKind string) {
		parseEvent := js.Global().Get("Object").New()
		parseEvent.Set("source", parseOpener)
		parseEvent.Set("origin", "https://app.example.test")
		parseEvent.Set("data", js.Global().Get("Object").New())
		parseEvent.Get("data").Set("name", "inspector")
		parseEvent.Get("data").Set("source", "opener-window")
		parseEvent.Get("data").Set("payload", js.Global().Get("Object").New())
		parseEvent.Get("data").Get("payload").Set("kind", parseKind)
		parseEvent.Get("data").Get("payload").Set("topic", "clients")
		parseEvent.Get("data").Get("payload").Set("source", js.Global().Get("Object").New())
		parseEvent.Get("data").Get("payload").Get("source").Set("id", "opener-1")
		parseEvent.Get("data").Get("payload").Get("source").Set("app", "atlas")
		parseEvent.Get("data").Get("payload").Get("source").Set("surface", "tab")
		parseMessageListener.Invoke(parseEvent)
	}

	parseEmitWindowMessage("hello")
	parseEmitWindowMessage("goodbye")
	if !isSawHello || !isSawGoodbye {
		parseT.Fatalf("expected opener lifecycle traffic to decode, sawHello=%v sawGoodbye=%v", isSawHello, isSawGoodbye)
	}
}

func TestMultiClientWindowSubscriptionRejectsOriginMismatchAndStaleOpener(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("origin", "https://app.example.test")
	parseWindow.Set("location", parseLocation)
	parseOpener := js.Global().Get("Object").New()
	parseOpener.Set("closed", false)
	parseOpener.Set("postMessage", js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return nil
	}))
	parseWindow.Set("opener", parseOpener)
	var parseMessageListener js.Value
	parseAddEventListener := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if parseArgs2[0].String() == "message" {
			parseMessageListener = parseArgs2[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		if parseArgs3[0].String() == "message" && parseMessageListener.Equal(parseArgs3[1]) {
			parseMessageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseChannel, parseErr := OpenWindowOpenerChannel(WindowChannelOptions{Name: "inspector"})
	if parseErr != nil {
		parseT.Fatalf("expected opener channel, got %v", parseErr)
	}

	var parseOriginMismatchErr error
	parseSubscription, parseErr := SubscribeClientWindowMessages(parseChannel, func(parseMessage ClientMessage, parseErr4 error) {
		if parseErr4 != nil {
			parseOriginMismatchErr = parseErr4
			return
		}
	})
	if parseErr != nil {
		parseT.Fatalf("expected opener security subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("source", parseOpener)
	parseEvent.Set("origin", "https://evil.example.test")
	parseEvent.Set("data", js.Global().Get("Object").New())
	parseEvent.Get("data").Set("name", "inspector")
	parseEvent.Get("data").Set("payload", js.Global().Get("Object").New())
	parseEvent.Get("data").Get("payload").Set("kind", "hello")
	parseEvent.Get("data").Get("payload").Set("topic", "clients")
	parseEvent.Get("data").Get("payload").Set("source", js.Global().Get("Object").New())
	parseEvent.Get("data").Get("payload").Get("source").Set("id", "opener-1")
	parseEvent.Get("data").Get("payload").Get("source").Set("app", "atlas")
	parseEvent.Get("data").Get("payload").Get("source").Set("surface", "tab")
	parseMessageListener.Invoke(parseEvent)
	if !IsCode(parseOriginMismatchErr, CodeUnauthorized) {
		parseT.Fatalf("expected target-origin mismatch to produce unauthorized error, got %v", parseOriginMismatchErr)
	}

	parseOpener.Set("closed", true)
	if parseErr2 := PublishClientHelloWindow(parseChannel, ClientIdentity{ID: "popup-1", App: "atlas", Surface: "popup", Role: "operator"}); !IsCode(parseErr2, CodeDisposed) {
		parseT.Fatalf("expected stale opener handle to reject hello publish, got %v", parseErr2)
	}
	if parseErr3 := PublishClientIntent(parseChannel, "operator:inventory", ClientIdentity{ID: "popup-1", App: "atlas", Surface: "popup", Role: "operator"}, "opener-1", map[string]any{"sku": "SKU-44"}); !IsCode(parseErr3, CodeDisposed) {
		parseT.Fatalf("expected stale opener handle to reject privileged intent publish, got %v", parseErr3)
	}
}

func TestSurfaceSignalWindowHelpersPublishAndDecode(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("origin", "https://app.example.test")
	parseWindow.Set("location", parseLocation)
	var parseOpened js.Value
	parseOpenFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOpened = js.Global().Get("Object").New()
		parseOpened.Set("closed", false)
		parseOpened.Set("postMessage", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			parseOpened.Set("__posted", parseArgs2[0])
			return nil
		}))
		parseOpened.Set("focus", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
		parseOpened.Set("close", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseOpened.Set("closed", true)
			return nil
		}))
		return parseOpened
	})
	defer parseOpenFn.Release()
	var parseMessageListener js.Value
	parseAddEventListener := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		if parseArgs5[0].String() == "message" {
			parseMessageListener = parseArgs5[1]
		}
		return nil
	})
	defer parseAddEventListener.Release()
	parseRemoveEventListener := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		if parseArgs6[0].String() == "message" && parseMessageListener.Equal(parseArgs6[1]) {
			parseMessageListener = js.Null()
		}
		return nil
	})
	defer parseRemoveEventListener.Release()
	parseWindow.Set("open", parseOpenFn)
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseRestoreWindow := setGlobalValue("window", parseWindow)
	defer parseRestoreWindow()

	parseChannel, parseErr := OpenSecondaryWindowChannel(WindowChannelOptions{
		URL:  "/popup.html",
		Name: "ops",
	})
	if parseErr != nil {
		parseT.Fatalf("expected popup channel, got %v", parseErr)
	}

	if parseErr2 := PublishRouteFocus(parseChannel, "/orders/42", "tab=activity", "order-heading"); parseErr2 != nil {
		parseT.Fatalf("expected route focus publish, got %v", parseErr2)
	}
	parsePosted := parseOpened.Get("__posted")
	if parsePosted.IsUndefined() || parsePosted.IsNull() {
		parseT.Fatal("expected helper publish to post a window payload")
	}
	if parsePosted.Get("payload").Get("kind").String() != "route" || parsePosted.Get("payload").Get("route").Get("path").String() != "/orders/42" {
		parseT.Fatalf("unexpected posted route signal: %#v", parsePosted)
	}

	var parseReceived DecodedWindowEnvelope[SurfaceSignal]
	parseSubscription, parseErr := SubscribeSurfaceSignals(parseChannel, func(parseMessage DecodedWindowEnvelope[SurfaceSignal], parseErr3 error) {
		if parseErr3 != nil {
			parseT.Fatalf("expected decoded surface signal, got %v", parseErr3)
		}
		parseReceived = parseMessage
	})
	if parseErr != nil {
		parseT.Fatalf("expected surface subscription to succeed, got %v", parseErr)
	}
	defer parseSubscription.Cancel()

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("source", parseOpened)
	parseEvent.Set("origin", "https://app.example.test")
	parseEvent.Set("data", js.Global().Get("Object").New())
	parseEvent.Get("data").Set("name", "ops")
	parseEvent.Get("data").Set("source", "popup-1")
	parseEvent.Get("data").Set("payload", js.Global().Get("Object").New())
	parseEvent.Get("data").Get("payload").Set("kind", "intent")
	parseEvent.Get("data").Get("payload").Set("intent", js.Global().Get("Object").New())
	parseEvent.Get("data").Get("payload").Get("intent").Set("action", "focus-panel")
	parseEvent.Get("data").Get("payload").Get("intent").Set("target", "audit-log")
	parseEvent.Get("data").Get("payload").Get("intent").Set("params", js.Global().Get("Object").New())
	parseEvent.Get("data").Get("payload").Get("intent").Get("params").Set("tab", "alerts")
	parseMessageListener.Invoke(parseEvent)

	if parseReceived.Name != "ops" || parseReceived.Source != "popup-1" {
		parseT.Fatalf("unexpected decoded surface envelope metadata: %+v", parseReceived)
	}
	if parseReceived.Payload.Kind != SurfaceSignalIntent || parseReceived.Payload.Intent == nil || parseReceived.Payload.Intent.Action != SurfaceIntentFocusPanel || parseReceived.Payload.Intent.Target != "audit-log" || parseReceived.Payload.Intent.Params["tab"] != "alerts" {
		parseT.Fatalf("unexpected decoded surface payload: %+v", parseReceived.Payload)
	}
}

func BenchmarkLocalStorageGetItem(parseB *testing.B) {
	parseStorage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return "dark"
	})
	defer getItemFn.Release()
	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
	parseStorage.Set("removeItem", js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} { return nil }))
	parseStorage.Set("clear", js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} { return nil }))
	parseStorage.Set("key", js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} { return js.Null() }))
	parseStorage.Set("length", 1)
	parseRestoreStorage := setGlobalValue("localStorage", parseStorage)
	defer parseRestoreStorage()

	parseLocal, parseErr := GetLocalStorage()
	if parseErr != nil {
		parseB.Fatal(parseErr)
	}
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseValue, parseOk, parseErr2 := parseLocal.GetItem("theme")
		if parseErr2 != nil || !parseOk || parseValue != "dark" {
			parseB.Fatalf("unexpected storage read: %q ok=%t err=%v", parseValue, parseOk, parseErr2)
		}
	}
}

func BenchmarkModuleCall(parseB *testing.B) {
	parseExportFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return js.ValueOf(parseArgs[0].Float() + 1)
	})
	defer parseExportFn.Release()
	parseModuleNS := js.Global().Get("Object").New()
	parseModuleNS.Set("next", parseExportFn)
	parseImportFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return makePromise(parseModuleNS)
	})
	defer parseImportFn.Release()
	parseRestoreImport := setGlobalValue("__gwcImportModule", parseImportFn)
	defer parseRestoreImport()

	parseModule, parseErr := ImportModule(context.Background(), "/bench/module.js")
	if parseErr != nil {
		parseB.Fatal(parseErr)
	}
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseValue, parseErr2 := parseModule.Call(context.Background(), "next", 41)
		if parseErr2 != nil || parseValue.(float64) != 42 {
			parseB.Fatalf("unexpected module call result %#v err=%v", parseValue, parseErr2)
		}
	}
}
