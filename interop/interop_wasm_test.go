//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"syscall/js"
	"testing"
	"time"
)

func setGlobalValue(name string, value interface{}) func() {
	global := js.Global()
	prev := global.Get(name)
	global.Set(name, value)
	return func() {
		global.Set(name, prev)
	}
}

func makePromise(value js.Value) js.Value {
	return js.Global().Get("Promise").Call("resolve", value)
}

func TestGlobalThisValueSurfaceSupportsPropertiesAndFunctions(t *testing.T) {
	global, err := GlobalThis()
	if err != nil {
		t.Fatalf("expected globalThis wrapper, got %v", err)
	}

	prevValue := global.Get("__interopValueProbe")
	prevFn := global.Get("__interopFnProbe")
	t.Cleanup(func() {
		if prevValue.Present() {
			_ = global.Set("__interopValueProbe", prevValue)
		} else {
			_ = global.Delete("__interopValueProbe")
		}
		if prevFn.Present() {
			_ = global.Set("__interopFnProbe", prevFn)
		} else {
			_ = global.Delete("__interopFnProbe")
		}
	})

	if err := global.Set("__interopValueProbe", map[string]any{"count": 7, "label": "ok"}); err != nil {
		t.Fatalf("expected global property write to succeed, got %v", err)
	}

	stored := global.Get("__interopValueProbe")
	if !stored.Present() {
		t.Fatal("expected stored probe value to be present")
	}
	decoded, err := stored.ToGo()
	if err != nil {
		t.Fatalf("expected probe value to decode, got %v", err)
	}
	payload, ok := decoded.(map[string]any)
	if !ok {
		t.Fatalf("expected decoded probe value to be a map, got %#v", decoded)
	}
	if payload["count"] != float64(7) || payload["label"] != "ok" {
		t.Fatalf("unexpected decoded payload: %#v", payload)
	}

	var seen string
	sub, err := global.SetFunction("__interopFnProbe", func(args ...Value) any {
		if len(args) != 2 {
			seen = fmt.Sprintf("unexpected:%d", len(args))
			return seen
		}
		seen = fmt.Sprintf("%s:%d", args[0].String(), args[1].Int())
		return seen
	})
	if err != nil {
		t.Fatalf("expected function binding to succeed, got %v", err)
	}
	defer sub.Cancel()

	result, err := global.Get("__interopFnProbe").Invoke("alpha", 4)
	if err != nil {
		t.Fatalf("expected function invocation to succeed, got %v", err)
	}
	if !result.Present() {
		t.Fatal("expected function result to be present")
	}
	if result.String() != "alpha:4" {
		t.Fatalf("unexpected function result: %q", result.String())
	}
	if seen != "alpha:4" {
		t.Fatalf("expected callback to observe arguments, got %q", seen)
	}
}

func TestLocalStorageWrapperTracksKeysAndValues(t *testing.T) {
	var keys []string
	values := map[string]string{}
	storage := js.Global().Get("Object").New()
	reindex := func() {
		storage.Set("length", len(keys))
	}

	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return js.Null()
		}
		if value, ok := values[args[0].String()]; ok {
			return value
		}
		return js.Null()
	})
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		key := args[0].String()
		if _, ok := values[key]; !ok {
			keys = append(keys, key)
		}
		values[key] = args[1].String()
		reindex()
		return nil
	})
	defer setItemFn.Release()
	removeItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		key := args[0].String()
		delete(values, key)
		next := keys[:0]
		for _, item := range keys {
			if item != key {
				next = append(next, item)
			}
		}
		keys = next
		reindex()
		return nil
	})
	defer removeItemFn.Release()
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		values = map[string]string{}
		keys = nil
		reindex()
		return nil
	})
	defer clearFn.Release()
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		index := args[0].Int()
		if index < 0 || index >= len(keys) {
			return js.Null()
		}
		return keys[index]
	})
	defer keyFn.Release()

	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", removeItemFn)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	reindex()

	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	local, err := LocalStorage()
	if err != nil {
		t.Fatalf("expected localStorage wrapper, got %v", err)
	}
	if err := local.SetItem("theme", "dark"); err != nil {
		t.Fatalf("expected set item to succeed, got %v", err)
	}
	value, ok, err := local.GetItem("theme")
	if err != nil || !ok || value != "dark" {
		t.Fatalf("unexpected storage read: value=%q ok=%t err=%v", value, ok, err)
	}
	length, err := local.Len()
	if err != nil || length != 1 {
		t.Fatalf("expected storage len 1, got %d err=%v", length, err)
	}
	key, ok, err := local.Key(0)
	if err != nil || !ok || key != "theme" {
		t.Fatalf("unexpected storage key: key=%q ok=%t err=%v", key, ok, err)
	}
	if err := local.RemoveItem("theme"); err != nil {
		t.Fatalf("expected remove item to succeed, got %v", err)
	}
	if _, ok, err := local.GetItem("theme"); err != nil || ok {
		t.Fatalf("expected removed item to disappear, ok=%t err=%v", ok, err)
	}
}

func TestLocalStorageGetManyReturnsPresentValues(t *testing.T) {
	storage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		switch args[0].String() {
		case "theme":
			return "dark"
		case "locale":
			return "en-US"
		default:
			return js.Null()
		}
	})
	defer getItemFn.Release()
	storage.Set("getItem", getItemFn)
	storage.Set("setItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("removeItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("clear", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("key", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() }))
	storage.Set("length", 2)
	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	local, err := LocalStorage()
	if err != nil {
		t.Fatalf("expected localStorage wrapper, got %v", err)
	}
	values, err := local.GetMany("theme", "locale", "missing")
	if err != nil {
		t.Fatalf("expected batched storage read, got %v", err)
	}
	if values["theme"] != "dark" || values["locale"] != "en-US" {
		t.Fatalf("unexpected batched storage values: %#v", values)
	}
	if _, ok := values["missing"]; ok {
		t.Fatalf("expected missing storage value to be omitted, got %#v", values)
	}
}

func TestOpenPersistentStoreUsesIndexedDB(t *testing.T) {
	restoreIndexedDB := installMockIndexedDB(t)
	defer restoreIndexedDB()

	store, err := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:         "cache",
		DatabaseName: "gwc-tests",
		Version:      1,
	})
	if err != nil {
		t.Fatalf("expected persistent store to open, got %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("expected persistent store close to succeed, got %v", closeErr)
		}
	}()

	if store.Backend() != "indexedDB" {
		t.Fatalf("expected indexedDB backend, got %q", store.Backend())
	}

	t.Run("close binds database receiver", func(t *testing.T) {
		constructor := js.Global().Get("Object")
		mockDB := constructor.New()
		closedWithBoundReceiver := false
		closeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if this.Equal(mockDB) {
				closedWithBoundReceiver = true
			}
			return nil
		})
		defer closeFn.Release()
		mockDB.Set("close", closeFn)

		mockStore := newIndexedDBPersistentStore(mockDB, persistentStoreConfig{databaseName: "gwc-tests", name: "cache"})
		if err := mockStore.Close(); err != nil {
			t.Fatalf("expected mock persistent store close to succeed, got %v", err)
		}
		if !closedWithBoundReceiver {
			t.Fatalf("expected persistent store close to call IndexedDB close with the database as receiver")
		}
	})
	if err := store.SetItem(context.Background(), "theme", "dark"); err != nil {
		t.Fatalf("expected persistent write to succeed, got %v", err)
	}
	if err := store.SetJSON(context.Background(), "profile", map[string]any{"locale": "en-US", "count": 3}); err != nil {
		t.Fatalf("expected persistent JSON write to succeed, got %v", err)
	}

	value, ok, err := store.GetItem(context.Background(), "theme")
	if err != nil || !ok || value != "dark" {
		t.Fatalf("unexpected persistent read: value=%q ok=%t err=%v", value, ok, err)
	}
	decoded, ok, err := LoadPersistentJSON[struct {
		Locale string `json:"locale"`
		Count  int    `json:"count"`
	}](context.Background(), store, "profile")
	if err != nil || !ok {
		t.Fatalf("expected typed persistent JSON decode, ok=%t err=%v", ok, err)
	}
	if decoded.Locale != "en-US" || decoded.Count != 3 {
		t.Fatalf("unexpected decoded JSON payload: %+v", decoded)
	}

	keys, err := store.Keys(context.Background())
	if err != nil {
		t.Fatalf("expected persistent keys, got %v", err)
	}
	if len(keys) != 2 || keys[0] != "profile" || keys[1] != "theme" {
		t.Fatalf("unexpected persistent keys: %#v", keys)
	}
	length, err := store.Len(context.Background())
	if err != nil || length != 2 {
		t.Fatalf("expected persistent len 2, got %d err=%v", length, err)
	}

	if err := store.RemoveItem(context.Background(), "theme"); err != nil {
		t.Fatalf("expected persistent remove to succeed, got %v", err)
	}
	if _, ok, err := store.GetItem(context.Background(), "theme"); err != nil || ok {
		t.Fatalf("expected removed persistent key to disappear, ok=%t err=%v", ok, err)
	}
	if err := store.Clear(context.Background()); err != nil {
		t.Fatalf("expected persistent clear to succeed, got %v", err)
	}
	length, err = store.Len(context.Background())
	if err != nil || length != 0 {
		t.Fatalf("expected persistent len 0 after clear, got %d err=%v", length, err)
	}
}

func TestLocalStorageRemoveItemReturnsStructuredErrorWhenNotCallable(t *testing.T) {
	storage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.Null()
	})
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	})
	defer setItemFn.Release()
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	})
	defer clearFn.Release()
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.Null()
	})
	defer keyFn.Release()

	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", js.Undefined())
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	storage.Set("length", 0)

	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	local, err := LocalStorage()
	if err != nil {
		t.Fatalf("expected localStorage wrapper, got %v", err)
	}
	err = local.RemoveItem("theme")
	if !IsCode(err, CodeNotFunction) {
		t.Fatalf("expected removeItem to return CodeNotFunction, got %v", err)
	}
}

func TestOpenPersistentStoreFallsBackWhenIndexedDBUnavailable(t *testing.T) {
	restoreIndexedDB := setGlobalValue("indexedDB", js.Undefined())
	defer restoreIndexedDB()

	storage := js.Global().Get("Object").New()
	data := map[string]string{}
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if value, ok := data[args[0].String()]; ok {
			return value
		}
		return js.Null()
	})
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		data[args[0].String()] = args[1].String()
		storage.Set("length", len(data))
		return nil
	})
	defer setItemFn.Release()
	removeItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		delete(data, args[0].String())
		storage.Set("length", len(data))
		return nil
	})
	defer removeItemFn.Release()
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		for key := range data {
			delete(data, key)
		}
		storage.Set("length", 0)
		return nil
	})
	defer clearFn.Release()
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		index := args[0].Int()
		if index < 0 || index >= len(keys) {
			return js.Null()
		}
		return keys[index]
	})
	defer keyFn.Release()
	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", removeItemFn)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	storage.Set("length", 0)

	store, err := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:            "cache",
		FallbackBackend: "localStorage",
		FallbackResolver: func() (Storage, error) {
			return Storage{
				getItem: func(key string) (string, bool, error) {
					value := storage.Call("getItem", key)
					if value.IsUndefined() || value.IsNull() {
						return "", false, nil
					}
					return value.String(), true, nil
				},
				setItem: func(key string, value string) error {
					storage.Call("setItem", key, value)
					return nil
				},
				removeItem: func(key string) error {
					storage.Call("removeItem", key)
					return nil
				},
				clear: func() error {
					storage.Call("clear")
					return nil
				},
				length: func() (int, error) {
					return storage.Get("length").Int(), nil
				},
				key: func(index int) (string, bool, error) {
					value := storage.Call("key", index)
					if value.IsNull() || value.IsUndefined() {
						return "", false, nil
					}
					return value.String(), true, nil
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("expected fallback persistent store, got %v", err)
	}
	if store.Backend() != "localStorage" {
		t.Fatalf("expected fallback backend label, got %q", store.Backend())
	}
	if err := store.SetItem(context.Background(), "draft", "ready"); err != nil {
		t.Fatalf("expected fallback persistent write, got %v", err)
	}
	value, ok, err := store.GetItem(context.Background(), "draft")
	if err != nil || !ok || value != "ready" {
		t.Fatalf("unexpected fallback read: value=%q ok=%t err=%v", value, ok, err)
	}
}

func TestOpenPersistentStoreReportsBlockedUpgrade(t *testing.T) {
	blocked := 0
	restoreIndexedDB := installMockIndexedDBWithOptions(t, mockIndexedDBOptions{
		blockedOpenCounts: map[string]int{"gwc-blocked": 1},
	})
	defer restoreIndexedDB()

	_, err := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:         "cache",
		DatabaseName: "gwc-blocked",
		Version:      2,
		OnBlocked: func(event PersistentStoreBlockedEvent) {
			blocked++
			if event.DatabaseName != "gwc-blocked" || event.StoreName != "cache" || event.RequestedVersion != 2 {
				t.Fatalf("unexpected blocked event: %+v", event)
			}
		},
	})
	if !IsCode(err, CodeBlocked) {
		t.Fatalf("expected blocked error, got %v", err)
	}
	if blocked != 1 {
		t.Fatalf("expected blocked callback once, got %d", blocked)
	}
}

func TestOpenPersistentStoreDeletesCorruptDatabaseAndRecovers(t *testing.T) {
	restoreIndexedDB := installMockIndexedDBWithOptions(t, mockIndexedDBOptions{
		openFailures: map[string][]mockIndexedDBError{
			"gwc-recover": {{Name: "InvalidStateError", Message: "backing store is corrupted"}},
		},
	})
	defer restoreIndexedDB()

	store, err := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:               "cache",
		DatabaseName:       "gwc-recover",
		DeleteOnCorruption: true,
	})
	if err != nil {
		t.Fatalf("expected corruption recovery to succeed, got %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("expected close after recovery to succeed, got %v", closeErr)
		}
	}()
	if err := store.SetItem(context.Background(), "theme", "dark"); err != nil {
		t.Fatalf("expected recovered store to accept writes, got %v", err)
	}
}

func TestPersistentStoreSetItemReportsQuotaExceeded(t *testing.T) {
	restoreIndexedDB := installMockIndexedDBWithOptions(t, mockIndexedDBOptions{
		putFailures: map[string][]mockIndexedDBError{
			"gwc-quota/cache": {{Name: "QuotaExceededError", Message: "storage quota exceeded"}},
		},
	})
	defer restoreIndexedDB()

	store, err := OpenPersistentStore(context.Background(), PersistentStoreOptions{
		Name:         "cache",
		DatabaseName: "gwc-quota",
	})
	if err != nil {
		t.Fatalf("expected persistent store to open, got %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("expected store close to succeed, got %v", closeErr)
		}
	}()

	err = store.SetItem(context.Background(), "theme", "dark")
	if !IsCode(err, CodeQuotaExceeded) {
		t.Fatalf("expected quota exceeded error, got %v", err)
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

func installMockIndexedDB(t *testing.T) func() {
	return installMockIndexedDBWithOptions(t, mockIndexedDBOptions{})
}

func installMockIndexedDBWithOptions(t *testing.T, options mockIndexedDBOptions) func() {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	databaseStores := map[string]map[string]map[string]string{}
	databaseVersions := map[string]int{}
	var funcs []js.Func
	releaseLater := func(fn js.Func) js.Func {
		funcs = append(funcs, fn)
		return fn
	}
	consumeFailure := func(failures map[string][]mockIndexedDBError, key string) (mockIndexedDBError, bool) {
		entries := failures[key]
		if len(entries) == 0 {
			return mockIndexedDBError{}, false
		}
		failure := entries[0]
		failures[key] = entries[1:]
		return failure, true
	}
	schedule := func(run func()) {
		var callback js.Func
		callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			callback.Release()
			run()
			return nil
		})
		global.Call("setTimeout", callback, 0)
	}
	newRequest := func() js.Value {
		request := objectCtor.New()
		request.Set("result", js.Null())
		request.Set("error", js.Null())
		return request
	}
	emitFailure := func(request js.Value, failure mockIndexedDBError) {
		schedule(func() {
			errValue := objectCtor.New()
			errValue.Set("name", failure.Name)
			errValue.Set("message", failure.Message)
			request.Set("error", errValue)
			handler := request.Get("onerror")
			if handler.Type() == js.TypeFunction {
				requestEvent := objectCtor.New()
				requestEvent.Set("target", request)
				handler.Invoke(requestEvent)
			}
		})
	}
	emitBlocked := func(request js.Value) {
		schedule(func() {
			handler := request.Get("onblocked")
			if handler.Type() == js.TypeFunction {
				requestEvent := objectCtor.New()
				requestEvent.Set("target", request)
				handler.Invoke(requestEvent)
			}
		})
	}
	emitSuccess := func(request js.Value, result any) {
		schedule(func() {
			request.Set("result", result)
			handler := request.Get("onsuccess")
			if handler.Type() == js.TypeFunction {
				requestEvent := objectCtor.New()
				requestEvent.Set("target", request)
				handler.Invoke(requestEvent)
			}
		})
	}
	buildDatabase := func(databaseName string) js.Value {
		db := objectCtor.New()
		objectStoreNames := objectCtor.New()
		objectStoreNames.Set("contains", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			_, ok := databaseStores[databaseName][args[0].String()]
			return ok
		})))
		db.Set("objectStoreNames", objectStoreNames)
		db.Set("createObjectStore", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			storeName := args[0].String()
			if databaseStores[databaseName] == nil {
				databaseStores[databaseName] = map[string]map[string]string{}
			}
			if databaseStores[databaseName][storeName] == nil {
				databaseStores[databaseName][storeName] = map[string]string{}
			}
			return objectCtor.New()
		})))
		db.Set("transaction", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			storeName := args[0].String()
			transaction := objectCtor.New()
			transaction.Set("objectStore", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				storeName := args[0].String()
				storeData := databaseStores[databaseName][storeName]
				store := objectCtor.New()
				store.Set("get", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					request := newRequest()
					key := args[0].String()
					if value, ok := storeData[key]; ok {
						entry := objectCtor.New()
						entry.Set("key", key)
						entry.Set("value", value)
						emitSuccess(request, entry)
					} else {
						emitSuccess(request, js.Null())
					}
					return request
				})))
				store.Set("put", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					request := newRequest()
					entry := args[0]
					if failure, ok := consumeFailure(options.putFailures, databaseName+"/"+storeName); ok {
						emitFailure(request, failure)
						return request
					}
					storeData[entry.Get("key").String()] = entry.Get("value").String()
					emitSuccess(request, entry.Get("key"))
					return request
				})))
				store.Set("delete", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					request := newRequest()
					delete(storeData, args[0].String())
					emitSuccess(request, js.Undefined())
					return request
				})))
				store.Set("clear", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					request := newRequest()
					for key := range storeData {
						delete(storeData, key)
					}
					emitSuccess(request, js.Undefined())
					return request
				})))
				store.Set("count", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					request := newRequest()
					emitSuccess(request, len(storeData))
					return request
				})))
				store.Set("getAllKeys", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					request := newRequest()
					keys := make([]string, 0, len(storeData))
					for key := range storeData {
						keys = append(keys, key)
					}
					sort.Strings(keys)
					emitSuccess(request, js.ValueOf(keys))
					return request
				})))
				return store
			})))
			_ = storeName
			return transaction
		})))
		db.Set("close", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })))
		return db
	}
	indexedDB := objectCtor.New()
	indexedDB.Set("open", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		request := newRequest()
		databaseName := args[0].String()
		version := 1
		if len(args) > 1 && args[1].Type() != js.TypeUndefined {
			version = args[1].Int()
		}
		if options.blockedOpenCounts[databaseName] > 0 {
			options.blockedOpenCounts[databaseName]--
			emitBlocked(request)
			return request
		}
		if failure, ok := consumeFailure(options.openFailures, databaseName); ok {
			emitFailure(request, failure)
			return request
		}
		if databaseStores[databaseName] == nil {
			databaseStores[databaseName] = map[string]map[string]string{}
		}
		previousVersion := databaseVersions[databaseName]
		databaseVersions[databaseName] = version
		db := buildDatabase(databaseName)
		schedule(func() {
			if previousVersion == 0 || version > previousVersion {
				request.Set("result", db)
				handler := request.Get("onupgradeneeded")
				if handler.Type() == js.TypeFunction {
					requestEvent := objectCtor.New()
					requestEvent.Set("target", request)
					handler.Invoke(requestEvent)
				}
			}
			request.Set("result", db)
			handler := request.Get("onsuccess")
			if handler.Type() == js.TypeFunction {
				requestEvent := objectCtor.New()
				requestEvent.Set("target", request)
				handler.Invoke(requestEvent)
			}
		})
		return request
	})))
	indexedDB.Set("deleteDatabase", releaseLater(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		request := newRequest()
		databaseName := args[0].String()
		if failure, ok := consumeFailure(options.deleteFailures, databaseName); ok {
			emitFailure(request, failure)
			return request
		}
		delete(databaseStores, databaseName)
		delete(databaseVersions, databaseName)
		emitSuccess(request, js.Undefined())
		return request
	})))
	restore := setGlobalValue("indexedDB", indexedDB)
	t.Cleanup(func() {
		restore()
		for _, fn := range funcs {
			fn.Release()
		}
	})
	return restore
}

func TestWindowHistoryPushStateRoundTripsDecodedState(t *testing.T) {
	history := js.Global().Get("Object").New()
	history.Set("length", 2)
	pushStateFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		history.Set("state", args[0])
		return nil
	})
	defer pushStateFn.Release()
	history.Set("pushState", pushStateFn)
	history.Set("replaceState", pushStateFn)
	history.Set("back", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	history.Set("forward", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	history.Set("go", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))

	window := js.Global().Get("Object").New()
	window.Set("history", history)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	wrapped, err := WindowHistory()
	if err != nil {
		t.Fatalf("expected history wrapper, got %v", err)
	}
	if err := wrapped.PushState(map[string]any{"step": "upload", "count": 2}, "Upload", "/upload"); err != nil {
		t.Fatalf("expected pushState to succeed, got %v", err)
	}
	state, err := wrapped.State()
	if err != nil {
		t.Fatalf("expected history state, got %v", err)
	}
	payload := map[string]any{}
	if err := Decode(state, &payload); err != nil {
		t.Fatalf("expected decoded history state, got %v", err)
	}
	if payload["step"] != "upload" {
		t.Fatalf("unexpected history payload: %#v", payload)
	}
}

func TestNavigatorClipboardAwaitingPromise(t *testing.T) {
	var written string
	writeTextFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		written = args[0].String()
		return makePromise(js.Undefined())
	})
	defer writeTextFn.Release()
	readTextFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makePromise(js.ValueOf("copied"))
	})
	defer readTextFn.Release()

	clipboard := js.Global().Get("Object").New()
	clipboard.Set("writeText", writeTextFn)
	clipboard.Set("readText", readTextFn)
	navigator := js.Global().Get("Object").New()
	navigator.Set("clipboard", clipboard)
	window := js.Global().Get("Object").New()
	window.Set("navigator", navigator)
	restoreNavigator := setGlobalValue("navigator", navigator)
	defer restoreNavigator()
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	wrapped, err := NavigatorClipboard()
	if err != nil {
		t.Fatalf("expected clipboard wrapper, got %v", err)
	}
	if err := wrapped.WriteText(context.Background(), "hello"); err != nil {
		t.Fatalf("expected writeText to succeed, got %v", err)
	}
	if written != "hello" {
		t.Fatalf("expected clipboard write payload, got %q", written)
	}
	text, err := wrapped.ReadText(context.Background())
	if err != nil || text != "copied" {
		t.Fatalf("expected clipboard read payload, got %q err=%v", text, err)
	}
}

func TestTimersUseBrowserCallbacks(t *testing.T) {
	var timeoutDelay int
	var clearedTimeout int
	setTimeoutFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		timeoutDelay = args[1].Int()
		args[0].Invoke()
		return 7
	})
	defer setTimeoutFn.Release()
	clearTimeoutFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		clearedTimeout++
		return nil
	})
	defer clearTimeoutFn.Release()

	restoreSetTimeout := setGlobalValue("setTimeout", setTimeoutFn)
	defer restoreSetTimeout()
	restoreClearTimeout := setGlobalValue("clearTimeout", clearTimeoutFn)
	defer restoreClearTimeout()

	fired := 0
	timer, err := SetTimeout(25*time.Millisecond, func() { fired++ })
	if err != nil {
		t.Fatalf("expected timeout to succeed, got %v", err)
	}
	if fired != 1 || timeoutDelay != 25 {
		t.Fatalf("unexpected timeout behavior: fired=%d delay=%d", fired, timeoutDelay)
	}
	if err := timer.Cancel(); err != nil {
		t.Fatalf("expected cancel to be safe after fire, got %v", err)
	}
	if clearedTimeout != 0 {
		t.Fatalf("expected fired timeout cancel to be a no-op, got %d clear calls", clearedTimeout)
	}
}

func TestWindowEventsDispatchCustomEvents(t *testing.T) {
	var listener js.Value
	window := js.Global().Get("Object").New()
	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			listener = args[1]
		}
		return nil
	})
	defer addEventListenerFn.Release()
	removeEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		listener = js.Undefined()
		return nil
	})
	defer removeEventListenerFn.Release()
	dispatchEventFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 && listener.Truthy() {
			listener.Invoke(args[0])
		}
		return true
	})
	defer dispatchEventFn.Release()
	window.Set("addEventListener", addEventListenerFn)
	window.Set("removeEventListener", removeEventListenerFn)
	window.Set("dispatchEvent", dispatchEventFn)

	customEventCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := js.Global().Get("Object").New()
		event.Set("type", args[0].String())
		event.Set("detail", args[1].Get("detail"))
		return event
	})
	defer customEventCtor.Release()

	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()
	restoreCustomEvent := setGlobalValue("CustomEvent", customEventCtor)
	defer restoreCustomEvent()

	target, err := WindowEvents()
	if err != nil {
		t.Fatalf("expected window event target, got %v", err)
	}
	var received CustomEvent
	sub, err := target.Subscribe("asset-ready", func(event CustomEvent) {
		received = event
	})
	if err != nil {
		t.Fatalf("expected subscribe to succeed, got %v", err)
	}
	defer sub.Cancel()

	if err := target.Dispatch("asset-ready", map[string]any{"id": "asset-42"}); err != nil {
		t.Fatalf("expected dispatch to succeed, got %v", err)
	}
	payload := map[string]string{}
	if err := Decode(received.Detail, &payload); err != nil {
		t.Fatalf("expected event detail to decode, got %v", err)
	}
	if received.Type != "asset-ready" || payload["id"] != "asset-42" {
		t.Fatalf("unexpected event payload: %+v decoded=%#v", received, payload)
	}
}

func TestSubscribeDecodedProjectsTypedCustomEventDetail(t *testing.T) {
	var listener js.Value
	window := js.Global().Get("Object").New()
	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			listener = args[1]
		}
		return nil
	})
	defer addEventListenerFn.Release()
	removeEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		listener = js.Undefined()
		return nil
	})
	defer removeEventListenerFn.Release()
	dispatchEventFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 && listener.Truthy() {
			listener.Invoke(args[0])
		}
		return true
	})
	defer dispatchEventFn.Release()
	window.Set("addEventListener", addEventListenerFn)
	window.Set("removeEventListener", removeEventListenerFn)
	window.Set("dispatchEvent", dispatchEventFn)

	customEventCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := js.Global().Get("Object").New()
		event.Set("type", args[0].String())
		event.Set("detail", args[1].Get("detail"))
		return event
	})
	defer customEventCtor.Release()

	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()
	restoreCustomEvent := setGlobalValue("CustomEvent", customEventCtor)
	defer restoreCustomEvent()

	target, err := WindowEvents()
	if err != nil {
		t.Fatalf("expected window event target, got %v", err)
	}
	type ratingChange struct {
		Score  int    `json:"score"`
		Source string `json:"source"`
	}
	var (
		received DecodedCustomEvent[ratingChange]
		gotErr   error
	)
	sub, err := SubscribeDecoded(target, "rating-change", func(event DecodedCustomEvent[ratingChange], err error) {
		received = event
		gotErr = err
	})
	if err != nil {
		t.Fatalf("expected decoded subscription to succeed, got %v", err)
	}
	defer sub.Cancel()

	if err := target.Dispatch("rating-change", map[string]any{"score": 5, "source": "widget"}); err != nil {
		t.Fatalf("expected dispatch to succeed, got %v", err)
	}
	if gotErr != nil {
		t.Fatalf("expected decoded event payload, got %v", gotErr)
	}
	if received.Type != "rating-change" || received.Detail.Score != 5 || received.Detail.Source != "widget" {
		t.Fatalf("unexpected decoded event payload: %+v", received)
	}
}

func TestWindowEventsListenReturnsBrowserEventTargets(t *testing.T) {
	var listener js.Value
	window := js.Global().Get("Object").New()
	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			listener = args[1]
		}
		return nil
	})
	defer addEventListenerFn.Release()
	removeEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		listener = js.Undefined()
		return nil
	})
	defer removeEventListenerFn.Release()
	window.Set("addEventListener", addEventListenerFn)
	window.Set("removeEventListener", removeEventListenerFn)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	target := js.Global().Get("Object").New()
	target.Set("tagName", "SECTION")
	target.Set("id", "metrics")
	target.Set("className", "panel")

	wrapped, err := WindowEvents()
	if err != nil {
		t.Fatalf("expected window event target, got %v", err)
	}
	var received BrowserEvent
	sub, err := wrapped.Listen("resize", func(event BrowserEvent) {
		received = event
	})
	if err != nil {
		t.Fatalf("expected generic listener to succeed, got %v", err)
	}
	defer sub.Cancel()

	event := js.Global().Get("Object").New()
	event.Set("type", "resize")
	event.Set("target", target)
	event.Set("currentTarget", window)
	listener.Invoke(event)

	if received.Type != "resize" {
		t.Fatalf("expected resize event type, got %+v", received)
	}
	if received.Target.TagName() != "SECTION" || received.Target.ID() != "metrics" {
		t.Fatalf("expected wrapped target element, got %+v", received.Target)
	}
}

func TestCurrentDocumentElementHelpers(t *testing.T) {
	var (
		focusCalls int
		blurCalls  int
		clickCalls int
		scrollArg  js.Value
		listener   js.Value
	)

	element := js.Global().Get("Object").New()
	element.Set("tagName", "DIV")
	element.Set("id", "hero")
	element.Set("className", "surface primary")
	focusFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		focusCalls++
		return nil
	})
	defer focusFn.Release()
	blurFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		blurCalls++
		return nil
	})
	defer blurFn.Release()
	clickFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		clickCalls++
		return nil
	})
	defer clickFn.Release()
	scrollFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			scrollArg = args[0]
		}
		return nil
	})
	defer scrollFn.Release()
	rectFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		rect := js.Global().Get("Object").New()
		rect.Set("x", 10)
		rect.Set("y", 12)
		rect.Set("width", 240)
		rect.Set("height", 80)
		rect.Set("top", 12)
		rect.Set("right", 250)
		rect.Set("bottom", 92)
		rect.Set("left", 10)
		return rect
	})
	defer rectFn.Release()
	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			listener = args[1]
		}
		return nil
	})
	defer addEventListenerFn.Release()
	removeEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		listener = js.Undefined()
		return nil
	})
	defer removeEventListenerFn.Release()
	element.Set("focus", focusFn)
	element.Set("blur", blurFn)
	element.Set("click", clickFn)
	element.Set("scrollIntoView", scrollFn)
	element.Set("getBoundingClientRect", rectFn)
	element.Set("addEventListener", addEventListenerFn)
	element.Set("removeEventListener", removeEventListenerFn)

	document := js.Global().Get("Object").New()
	getElementByIDFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return element
	})
	defer getElementByIDFn.Release()
	querySelectorFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return element
	})
	defer querySelectorFn.Release()
	document.Set("getElementById", getElementByIDFn)
	document.Set("querySelector", querySelectorFn)
	restoreDocument := setGlobalValue("document", document)
	defer restoreDocument()

	wrapped, err := CurrentDocument()
	if err != nil {
		t.Fatalf("expected current document wrapper, got %v", err)
	}
	byID, ok, err := wrapped.ElementByID("hero")
	if err != nil || !ok {
		t.Fatalf("expected element by id, ok=%t err=%v", ok, err)
	}
	if byID.TagName() != "DIV" || byID.ClassName() != "surface primary" {
		t.Fatalf("unexpected element metadata: tag=%q class=%q", byID.TagName(), byID.ClassName())
	}
	if err := byID.Focus(); err != nil {
		t.Fatalf("expected focus to succeed, got %v", err)
	}
	if err := byID.Blur(); err != nil {
		t.Fatalf("expected blur to succeed, got %v", err)
	}
	if err := byID.Click(); err != nil {
		t.Fatalf("expected click to succeed, got %v", err)
	}
	if err := byID.ScrollIntoView(ScrollIntoViewOptions{Behavior: "smooth", Block: "center"}); err != nil {
		t.Fatalf("expected scrollIntoView to succeed, got %v", err)
	}
	rect, err := byID.BoundingClientRect()
	if err != nil {
		t.Fatalf("expected bounding rect, got %v", err)
	}
	if rect.Width != 240 || rect.Top != 12 {
		t.Fatalf("unexpected bounding rect: %+v", rect)
	}
	if focusCalls != 1 || blurCalls != 1 || clickCalls != 1 {
		t.Fatalf("unexpected element method calls: focus=%d blur=%d click=%d", focusCalls, blurCalls, clickCalls)
	}
	if scrollArg.IsUndefined() || scrollArg.IsNull() || scrollArg.Get("behavior").String() != "smooth" || scrollArg.Get("block").String() != "center" {
		t.Fatalf("expected scroll options to be forwarded, got %v", scrollArg)
	}

	var received BrowserEvent
	sub, err := byID.Listen("asset-ready", func(event BrowserEvent) {
		received = event
	})
	if err != nil {
		t.Fatalf("expected element listener to succeed, got %v", err)
	}
	defer sub.Cancel()
	event := js.Global().Get("Object").New()
	event.Set("type", "asset-ready")
	event.Set("detail", map[string]any{"asset": "hero"})
	event.Set("target", element)
	event.Set("currentTarget", element)
	listener.Invoke(event)
	if received.Type != "asset-ready" || received.Target.ID() != "hero" {
		t.Fatalf("unexpected element event payload: %+v", received)
	}

	queried, ok, err := wrapped.QuerySelector("#hero")
	if err != nil || !ok || queried.ID() != "hero" {
		t.Fatalf("expected querySelector result, ok=%t err=%v id=%q", ok, err, queried.ID())
	}
}

func TestCurrentDocumentElementsByIDBatchesLookups(t *testing.T) {
	first := js.Global().Get("Object").New()
	first.Set("tagName", "DIV")
	first.Set("id", "hero")
	second := js.Global().Get("Object").New()
	second.Set("tagName", "ASIDE")
	second.Set("id", "sidebar")

	document := js.Global().Get("Object").New()
	getElementByIDFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		switch args[0].String() {
		case "hero":
			return first
		case "sidebar":
			return second
		default:
			return js.Null()
		}
	})
	defer getElementByIDFn.Release()
	document.Set("getElementById", getElementByIDFn)
	querySelectorFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.Null()
	})
	defer querySelectorFn.Release()
	document.Set("querySelector", querySelectorFn)
	restoreDocument := setGlobalValue("document", document)
	defer restoreDocument()

	wrapped, err := CurrentDocument()
	if err != nil {
		t.Fatalf("expected current document wrapper, got %v", err)
	}
	elements, err := wrapped.ElementsByID("hero", "sidebar", "missing")
	if err != nil {
		t.Fatalf("expected batched element lookup, got %v", err)
	}
	if elements["hero"].ID() != "hero" || elements["sidebar"].TagName() != "ASIDE" {
		t.Fatalf("unexpected batched element lookup results: %#v", elements)
	}
	if _, ok := elements["missing"]; ok {
		t.Fatalf("expected missing id to be omitted, got %#v", elements)
	}
}

func TestElementObserverHelpers(t *testing.T) {
	element := js.Global().Get("Object").New()
	element.Set("tagName", "ARTICLE")
	element.Set("id", "observer-target")

	document := js.Global().Get("Object").New()
	getElementByIDFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return element
	})
	defer getElementByIDFn.Release()
	document.Set("getElementById", getElementByIDFn)
	querySelectorFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return element
	})
	defer querySelectorFn.Release()
	document.Set("querySelector", querySelectorFn)
	restoreDocument := setGlobalValue("document", document)
	defer restoreDocument()

	var (
		resizeCallback       js.Value
		intersectionCallback js.Value
		resizeDisconnects    int
		intersectDisconnects int
		intersectionInit     js.Value
	)

	resizeObserverCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resizeCallback = args[0]
		observer := js.Global().Get("Object").New()
		observeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		disconnectFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resizeDisconnects++
			return nil
		})
		observer.Set("observe", observeFn)
		observer.Set("disconnect", disconnectFn)
		return observer
	})
	defer resizeObserverCtor.Release()
	intersectionObserverCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		intersectionCallback = args[0]
		if len(args) > 1 {
			intersectionInit = args[1]
		}
		observer := js.Global().Get("Object").New()
		observeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		disconnectFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			intersectDisconnects++
			return nil
		})
		observer.Set("observe", observeFn)
		observer.Set("disconnect", disconnectFn)
		return observer
	})
	defer intersectionObserverCtor.Release()
	restoreResizeObserver := setGlobalValue("ResizeObserver", resizeObserverCtor)
	defer restoreResizeObserver()
	restoreIntersectionObserver := setGlobalValue("IntersectionObserver", intersectionObserverCtor)
	defer restoreIntersectionObserver()

	wrapped, err := CurrentDocument()
	if err != nil {
		t.Fatalf("expected current document wrapper, got %v", err)
	}
	target, ok, err := wrapped.ElementByID("observer-target")
	if err != nil || !ok {
		t.Fatalf("expected target element, ok=%t err=%v", ok, err)
	}

	var resizeEntry ResizeEntry
	resizeSub, err := target.ObserveResize(func(entry ResizeEntry) {
		resizeEntry = entry
	})
	if err != nil {
		t.Fatalf("expected resize observer to succeed, got %v", err)
	}
	defer resizeSub.Cancel()

	resizeRect := js.Global().Get("Object").New()
	resizeRect.Set("width", 320)
	resizeRect.Set("height", 180)
	resizeRect.Set("x", 0)
	resizeRect.Set("y", 0)
	resizeRect.Set("top", 0)
	resizeRect.Set("right", 320)
	resizeRect.Set("bottom", 180)
	resizeRect.Set("left", 0)
	resizePayload := js.Global().Get("Object").New()
	resizePayload.Set("target", element)
	resizePayload.Set("contentRect", resizeRect)
	resizeCallback.Invoke(js.Global().Get("Array").Call("of", resizePayload))
	if resizeEntry.Target.ID() != "observer-target" || resizeEntry.ContentRect.Width != 320 {
		t.Fatalf("unexpected resize payload: %+v", resizeEntry)
	}

	var intersectionEntry IntersectionEntry
	intersectionSub, err := target.ObserveIntersection(func(entry IntersectionEntry) {
		intersectionEntry = entry
	}, IntersectionObserverOptions{RootMargin: "12px", Thresholds: []float64{0.25, 0.75}})
	if err != nil {
		t.Fatalf("expected intersection observer to succeed, got %v", err)
	}
	defer intersectionSub.Cancel()

	intersectionRect := js.Global().Get("Object").New()
	intersectionRect.Set("width", 120)
	intersectionRect.Set("height", 60)
	intersectionRect.Set("x", 10)
	intersectionRect.Set("y", 20)
	intersectionRect.Set("top", 20)
	intersectionRect.Set("right", 130)
	intersectionRect.Set("bottom", 80)
	intersectionRect.Set("left", 10)
	rootBounds := js.Global().Get("Object").New()
	rootBounds.Set("width", 500)
	rootBounds.Set("height", 400)
	rootBounds.Set("x", 0)
	rootBounds.Set("y", 0)
	rootBounds.Set("top", 0)
	rootBounds.Set("right", 500)
	rootBounds.Set("bottom", 400)
	rootBounds.Set("left", 0)
	intersectionPayload := js.Global().Get("Object").New()
	intersectionPayload.Set("target", element)
	intersectionPayload.Set("isIntersecting", true)
	intersectionPayload.Set("intersectionRatio", 0.75)
	intersectionPayload.Set("boundingClientRect", intersectionRect)
	intersectionPayload.Set("intersectionRect", intersectionRect)
	intersectionPayload.Set("rootBounds", rootBounds)
	intersectionCallback.Invoke(js.Global().Get("Array").Call("of", intersectionPayload))
	if !intersectionEntry.IsIntersecting || intersectionEntry.Target.ID() != "observer-target" || intersectionEntry.RootBounds == nil || intersectionEntry.RootBounds.Width != 500 {
		t.Fatalf("unexpected intersection payload: %+v", intersectionEntry)
	}
	if intersectionInit.IsUndefined() || intersectionInit.IsNull() || intersectionInit.Get("rootMargin").String() != "12px" || intersectionInit.Get("threshold").Get("length").Int() != 2 {
		t.Fatalf("expected intersection observer init to carry options, got %v", intersectionInit)
	}

	resizeSub.Cancel()
	intersectionSub.Cancel()
	if resizeDisconnects == 0 || intersectDisconnects == 0 {
		t.Fatalf("expected observers to disconnect on cancel, resize=%d intersection=%d", resizeDisconnects, intersectDisconnects)
	}
}

func TestMatchMediaSubscriptionReceivesChanges(t *testing.T) {
	var changeListener js.Value
	mediaQuery := js.Global().Get("Object").New()
	mediaQuery.Set("matches", false)
	mediaQuery.Set("media", "(prefers-color-scheme: dark)")
	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			changeListener = args[1]
		}
		return nil
	})
	defer addEventListenerFn.Release()
	removeEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		changeListener = js.Undefined()
		return nil
	})
	defer removeEventListenerFn.Release()
	mediaQuery.Set("addEventListener", addEventListenerFn)
	mediaQuery.Set("removeEventListener", removeEventListenerFn)

	matchMediaFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return mediaQuery
	})
	defer matchMediaFn.Release()
	window := js.Global().Get("Object").New()
	window.Set("matchMedia", matchMediaFn)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	list, err := MatchMedia("(prefers-color-scheme: dark)")
	if err != nil {
		t.Fatalf("expected media query list, got %v", err)
	}
	var event MediaQueryEvent
	sub, err := list.Subscribe(func(next MediaQueryEvent) {
		event = next
	})
	if err != nil {
		t.Fatalf("expected media query subscribe to succeed, got %v", err)
	}
	defer sub.Cancel()

	change := js.Global().Get("Object").New()
	change.Set("matches", true)
	change.Set("media", "(prefers-color-scheme: dark)")
	changeListener.Invoke(change)
	if !event.Matches || event.Media == "" {
		t.Fatalf("unexpected media query event: %+v", event)
	}
}

func TestImportModuleCallDefaultAndDispose(t *testing.T) {
	sumFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.ValueOf(args[0].Float() + args[1].Float())
	})
	defer sumFn.Release()
	defaultFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		payload := js.Global().Get("Object").New()
		payload.Set("ok", true)
		payload.Set("label", args[0].String())
		return payload
	})
	defer defaultFn.Release()

	moduleNS := js.Global().Get("Object").New()
	moduleNS.Set("sum", sumFn)
	moduleNS.Set("default", defaultFn)
	moduleNS.Set("version", "1.0.0")

	importFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makePromise(moduleNS)
	})
	defer importFn.Release()
	restoreImport := setGlobalValue("__gwcImportModule", importFn)
	defer restoreImport()

	module, err := ImportModule(context.Background(), "/demo/math.js")
	if err != nil {
		t.Fatalf("expected import to succeed, got %v", err)
	}
	sum, err := module.Call(context.Background(), "sum", 2, 3)
	if err != nil {
		t.Fatalf("expected named export call to succeed, got %v", err)
	}
	if sum.(float64) != 5 {
		t.Fatalf("expected module sum result 5, got %#v", sum)
	}
	defaultValue, err := module.CallDefault(context.Background(), "asset")
	if err != nil {
		t.Fatalf("expected default export to succeed, got %v", err)
	}
	payload := map[string]any{}
	if err := Decode(defaultValue, &payload); err != nil {
		t.Fatalf("expected default export payload to decode, got %v", err)
	}
	if payload["label"] != "asset" {
		t.Fatalf("unexpected default export payload: %#v", payload)
	}
	version, err := module.Value(context.Background(), "version")
	if err != nil || version.(string) != "1.0.0" {
		t.Fatalf("expected module value export, got %#v err=%v", version, err)
	}
	if err := module.Dispose(); err != nil {
		t.Fatalf("expected dispose to succeed, got %v", err)
	}
	if _, err := module.Value(context.Background(), "version"); !IsCode(err, CodeDisposed) {
		t.Fatalf("expected disposed module error, got %v", err)
	}
}

func TestNewWorkerRequestDecodedSupportsReadyProgressAndResult(t *testing.T) {
	var created int
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		raw := js.Global().Get("Object").New()
		messageListeners := js.Global().Get("Array").New()
		errorListeners := js.Global().Get("Array").New()
		raw.Set("__readySent", false)
		emitMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			event := js.Global().Get("Object").New()
			if len(args) > 0 {
				event.Set("data", args[0])
			}
			for i := 0; i < messageListeners.Length(); i++ {
				callback := messageListeners.Index(i)
				if callback.IsUndefined() || callback.IsNull() {
					continue
				}
				callback.Invoke(event)
			}
			return nil
		})
		emitError := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			event := js.Global().Get("Object").New()
			if len(args) > 0 {
				event.Set("message", args[0])
			}
			for i := 0; i < errorListeners.Length(); i++ {
				callback := errorListeners.Index(i)
				if callback.IsUndefined() || callback.IsNull() {
					continue
				}
				callback.Invoke(event)
			}
			return nil
		})
		addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			eventType := args[0].String()
			callback := args[1]
			switch eventType {
			case "message":
				messageListeners.Call("push", callback)
				if !raw.Get("__readySent").Bool() {
					raw.Set("__readySent", true)
					readyValue, _ := goValueToJS("test", "worker-ready", map[string]any{"phase": "ready", "name": "bootstrap"})
					raw.Call("__emitMessage", readyValue)
				}
			case "error":
				errorListeners.Call("push", callback)
			}
			return nil
		})
		removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			eventType := args[0].String()
			callback := args[1]
			var listeners js.Value
			switch eventType {
			case "message":
				listeners = messageListeners
			case "error":
				listeners = errorListeners
			default:
				return nil
			}
			for i := 0; i < listeners.Length(); i++ {
				current := listeners.Index(i)
				if current.IsUndefined() || current.IsNull() {
					continue
				}
				if current.Equal(callback) {
					listeners.SetIndex(i, js.Null())
				}
			}
			return nil
		})
		postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			value, err := jsValueToGo("test", "worker-post", args[0])
			if err != nil {
				raw.Call("__emitError", err.Error())
				return nil
			}
			message := workerMessageFromGo(value)
			progressValue, _ := goValueToJS("test", "worker-progress", map[string]any{
				"id":      message.ID,
				"phase":   "progress",
				"name":    message.Name,
				"payload": map[string]any{"percent": 40},
			})
			raw.Call("__emitMessage", progressValue)
			resultValue, _ := goValueToJS("test", "worker-result", map[string]any{
				"id":      message.ID,
				"phase":   "result",
				"name":    message.Name,
				"payload": map[string]any{"summary": "indexed 12 documents"},
			})
			raw.Call("__emitMessage", resultValue)
			return nil
		})
		terminate := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			raw.Set("__terminated", true)
			return nil
		})
		raw.Set("addEventListener", addEventListener)
		raw.Set("removeEventListener", removeEventListener)
		raw.Set("postMessage", postMessage)
		raw.Set("terminate", terminate)
		raw.Set("__emitMessage", emitMessage)
		raw.Set("__emitError", emitError)
		created++
		return raw
	})
	defer ctor.Release()
	restoreWorker := setGlobalValue("Worker", ctor)
	defer restoreWorker()

	worker, err := NewWorker(context.Background(), WorkerOptions{URL: "/workers/search.mjs", Ready: true})
	if err != nil {
		t.Fatalf("expected worker wrapper, got %v", err)
	}
	if created != 1 {
		t.Fatalf("expected one worker instance, got %d", created)
	}

	type progressPayload struct {
		Percent int `json:"percent"`
	}
	type resultPayload struct {
		Summary string `json:"summary"`
	}
	var progress []int
	result, err := RequestWorkerDecoded[map[string]any, progressPayload, resultPayload](context.Background(), worker, "build-index", map[string]any{"query": "atlas"}, func(message DecodedWorkerMessage[progressPayload], err error) {
		if err != nil {
			t.Fatalf("expected decoded progress payload, got %v", err)
		}
		progress = append(progress, message.Payload.Percent)
	})
	if err != nil {
		t.Fatalf("expected worker request to succeed, got %v", err)
	}
	if len(progress) != 1 || progress[0] != 40 {
		t.Fatalf("unexpected worker progress updates: %#v", progress)
	}
	if result.Summary != "indexed 12 documents" {
		t.Fatalf("unexpected worker result payload: %+v", result)
	}
}

func TestWorkerTerminateAndRestartSwapActiveInstance(t *testing.T) {
	var created int
	var posts int
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		raw := js.Global().Get("Object").New()
		raw.Set("addEventListener", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		raw.Set("removeEventListener", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		raw.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			posts++
			return nil
		}))
		raw.Set("terminate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			raw.Set("__terminated", true)
			return nil
		}))
		created++
		return raw
	})
	defer ctor.Release()
	restoreWorker := setGlobalValue("Worker", ctor)
	defer restoreWorker()

	worker, err := NewWorker(context.Background(), WorkerOptions{URL: "/workers/report.js"})
	if err != nil {
		t.Fatalf("expected worker wrapper, got %v", err)
	}
	if err := worker.Post(map[string]any{"phase": "message"}); err != nil {
		t.Fatalf("expected initial worker post to succeed, got %v", err)
	}
	if err := worker.Terminate(); err != nil {
		t.Fatalf("expected terminate to succeed, got %v", err)
	}
	if err := worker.Post(map[string]any{"phase": "message"}); !IsCode(err, CodeDisposed) {
		t.Fatalf("expected disposed error after terminate, got %v", err)
	}
	if err := worker.Restart(context.Background()); err != nil {
		t.Fatalf("expected restart to succeed, got %v", err)
	}
	if err := worker.Post(map[string]any{"phase": "message"}); err != nil {
		t.Fatalf("expected worker post to succeed after restart, got %v", err)
	}
	if created != 2 {
		t.Fatalf("expected two worker instances after restart, got %d", created)
	}
	if posts != 2 {
		t.Fatalf("expected posts to reach the active workers only, got %d", posts)
	}
}

func TestWorkerRequestHonorsContextTimeout(t *testing.T) {
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		raw := js.Global().Get("Object").New()
		raw.Set("addEventListener", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		raw.Set("removeEventListener", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		raw.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		raw.Set("terminate", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		return raw
	})
	defer ctor.Release()
	restoreWorker := setGlobalValue("Worker", ctor)
	defer restoreWorker()

	worker, err := NewWorker(context.Background(), WorkerOptions{URL: "/workers/slow.js"})
	if err != nil {
		t.Fatalf("expected worker wrapper, got %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := worker.Request(ctx, "slow-job", map[string]any{"input": "demo"}, nil); !IsCode(err, CodeTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestOpenCrossTabChannelUsesBroadcastChannel(t *testing.T) {
	var posted any
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		raw := js.Global().Get("Object").New()
		messageListeners := js.Global().Get("Array").New()
		addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].String() == "message" {
				messageListeners.Call("push", args[1])
			}
			return nil
		})
		removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].String() != "message" {
				return nil
			}
			callback := args[1]
			for i := 0; i < messageListeners.Length(); i++ {
				current := messageListeners.Index(i)
				if !current.IsUndefined() && !current.IsNull() && current.Equal(callback) {
					messageListeners.SetIndex(i, js.Null())
				}
			}
			return nil
		})
		emitMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			event := js.Global().Get("Object").New()
			event.Set("data", args[0])
			for i := 0; i < messageListeners.Length(); i++ {
				callback := messageListeners.Index(i)
				if callback.IsUndefined() || callback.IsNull() {
					continue
				}
				callback.Invoke(event)
			}
			return nil
		})
		postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			value, err := jsValueToGo("test", "broadcast-channel", args[0])
			if err == nil {
				posted = value
			}
			raw.Call("__emitMessage", args[0])
			return nil
		})
		closeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			raw.Set("__closed", true)
			return nil
		})
		raw.Set("addEventListener", addEventListener)
		raw.Set("removeEventListener", removeEventListener)
		raw.Set("postMessage", postMessage)
		raw.Set("close", closeFn)
		raw.Set("__emitMessage", emitMessage)
		return raw
	})
	defer ctor.Release()
	restoreBroadcast := setGlobalValue("BroadcastChannel", ctor)
	defer restoreBroadcast()

	channel, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "theme"})
	if err != nil {
		t.Fatalf("expected cross-tab channel, got %v", err)
	}
	if channel.Transport() != "broadcast-channel" {
		t.Fatalf("expected broadcast transport, got %q", channel.Transport())
	}

	var received DecodedCrossTabEnvelope[struct {
		Theme string `json:"theme"`
	}]
	subscription, err := SubscribeDecodedCrossTab(channel, func(message DecodedCrossTabEnvelope[struct {
		Theme string `json:"theme"`
	}], err error) {
		if err != nil {
			t.Fatalf("expected decoded broadcast payload, got %v", err)
		}
		received = message
	})
	if err != nil {
		t.Fatalf("expected subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	if err := channel.Publish(map[string]any{"theme": "dark"}); err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}
	if received.Payload.Theme != "dark" || received.Name != "theme" || received.Source == "" || received.Sequence != 1 {
		t.Fatalf("unexpected decoded broadcast envelope: %+v", received)
	}

	rawPosted, ok := posted.(map[string]any)
	if !ok {
		t.Fatalf("expected posted envelope map, got %#v", posted)
	}
	if rawPosted["name"] != "theme" {
		t.Fatalf("expected published envelope name, got %#v", rawPosted)
	}
	payload, ok := rawPosted["payload"].(map[string]any)
	if !ok || payload["theme"] != "dark" {
		t.Fatalf("unexpected published payload: %#v", rawPosted["payload"])
	}

	if err := channel.Close(); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}
	if err := channel.Publish(map[string]any{"theme": "light"}); !IsCode(err, CodeDisposed) {
		t.Fatalf("expected disposed error after close, got %v", err)
	}
}

func TestOpenCrossTabChannelFallsBackToStorageEvents(t *testing.T) {
	restoreBroadcast := setGlobalValue("BroadcastChannel", js.Undefined())
	defer restoreBroadcast()

	var (
		storageListener js.Value
		lastSetKey      string
		lastSetValue    string
		lastRemovedKey  string
	)
	window := js.Global().Get("Object").New()
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "storage" {
			storageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "storage" && storageListener.Equal(args[1]) {
			storageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	storage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() })
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		lastSetKey = args[0].String()
		lastSetValue = args[1].String()
		return nil
	})
	defer setItemFn.Release()
	removeItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		lastRemovedKey = args[0].String()
		return nil
	})
	defer removeItemFn.Release()
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer clearFn.Release()
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() })
	defer keyFn.Release()
	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", removeItemFn)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	storage.Set("length", 0)
	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	channel, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "prefs"})
	if err != nil {
		t.Fatalf("expected storage-fallback channel, got %v", err)
	}
	if channel.Transport() != "storage-event" {
		t.Fatalf("expected storage-event fallback, got %q", channel.Transport())
	}

	var received DecodedCrossTabEnvelope[struct {
		Mode string `json:"mode"`
	}]
	subscription, err := SubscribeDecodedCrossTab(channel, func(message DecodedCrossTabEnvelope[struct {
		Mode string `json:"mode"`
	}], err error) {
		if err != nil {
			t.Fatalf("expected decoded storage payload, got %v", err)
		}
		received = message
	})
	if err != nil {
		t.Fatalf("expected storage subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	if err := channel.Publish(map[string]any{"mode": "dark"}); err != nil {
		t.Fatalf("expected storage publish to succeed, got %v", err)
	}
	if lastSetKey != "__gwc_cross_tab__:prefs" || lastRemovedKey != "__gwc_cross_tab__:prefs" {
		t.Fatalf("unexpected storage keys: set=%q removed=%q", lastSetKey, lastRemovedKey)
	}
	if lastSetValue == "" {
		t.Fatal("expected storage fallback to serialize the published envelope")
	}

	event := js.Global().Get("Object").New()
	event.Set("key", "__gwc_cross_tab__:prefs")
	event.Set("newValue", `{"name":"prefs","payload":{"mode":"light"},"source":"tab-2","sequence":7,"sentAt":"2026-03-18T16:00:00Z"}`)
	storageListener.Invoke(event)
	if received.Payload.Mode != "light" || received.Source != "tab-2" || received.Sequence != 7 || received.Name != "prefs" {
		t.Fatalf("unexpected decoded storage envelope: %+v", received)
	}

	if err := channel.Close(); err != nil {
		t.Fatalf("expected storage-fallback close to succeed, got %v", err)
	}
	if err := channel.Publish(map[string]any{"mode": "reset"}); !IsCode(err, CodeDisposed) {
		t.Fatalf("expected disposed error after close, got %v", err)
	}
}

func TestMultiClientStorageFallbackLifecycleFlow(t *testing.T) {
	restoreBroadcast := setGlobalValue("BroadcastChannel", js.Undefined())
	defer restoreBroadcast()

	var storageListener js.Value
	window := js.Global().Get("Object").New()
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "storage" {
			storageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "storage" && storageListener.Equal(args[1]) {
			storageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	storage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() })
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer setItemFn.Release()
	removeItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer removeItemFn.Release()
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer clearFn.Release()
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() })
	defer keyFn.Release()
	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", removeItemFn)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	storage.Set("length", 0)
	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	channel, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients-fallback"})
	if err != nil {
		t.Fatalf("expected storage-fallback channel, got %v", err)
	}
	if channel.Transport() != "storage-event" {
		t.Fatalf("expected storage-event fallback transport, got %q", channel.Transport())
	}

	type peerState struct {
		hellos       int
		disconnected bool
		expired      bool
		lastSentAt   time.Time
	}
	peers := map[string]peerState{}
	expirePeers := func(now time.Time, lease time.Duration) {
		for id, state := range peers {
			if state.disconnected || state.lastSentAt.IsZero() {
				continue
			}
			if now.Sub(state.lastSentAt) > lease {
				state.expired = true
				peers[id] = state
			}
		}
	}

	subscription, err := SubscribeClientMessages(channel, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected decoded client storage message, got %v", err)
		}
		if message.Topic != ClientPresenceTopic {
			return
		}
		state := peers[message.Source.ID]
		switch message.Kind {
		case ClientHello:
			state.hellos++
			state.disconnected = false
			state.expired = false
			state.lastSentAt = message.SentAt
		case ClientGoodbye:
			state.disconnected = true
			state.lastSentAt = message.SentAt
		}
		peers[message.Source.ID] = state
	})
	if err != nil {
		t.Fatalf("expected storage multi-client subscription, got %v", err)
	}
	defer subscription.Cancel()

	emitStorage := func(data string) {
		event := js.Global().Get("Object").New()
		event.Set("key", "__gwc_cross_tab__:clients-fallback")
		event.Set("newValue", data)
		storageListener.Invoke(event)
	}

	emitStorage(`{"name":"clients-fallback","payload":{"kind":"hello","topic":"clients","source":{"id":"storefront-2","app":"atlas","surface":"tab"},"sentAt":"2026-03-19T10:00:00Z"},"source":"tab-2","sequence":1,"sentAt":"2026-03-19T10:00:00Z"}`)
	emitStorage(`{"name":"clients-fallback","payload":{"kind":"hello","topic":"clients","source":{"id":"storefront-2","app":"atlas","surface":"tab"},"sentAt":"2026-03-19T10:00:02Z"},"source":"tab-2","sequence":2,"sentAt":"2026-03-19T10:00:02Z"}`)
	state := peers["storefront-2"]
	if state.hellos != 2 || state.disconnected || state.expired {
		t.Fatalf("expected duplicate fallback hello traffic to be tolerated, got %+v", state)
	}

	expirePeers(time.Date(2026, 3, 19, 10, 0, 6, 0, time.UTC), 3*time.Second)
	state = peers["storefront-2"]
	if !state.expired {
		t.Fatalf("expected fallback peer lease to expire after inactivity, got %+v", state)
	}

	emitStorage(`{"name":"clients-fallback","payload":{"kind":"hello","topic":"clients","source":{"id":"storefront-2","app":"atlas","surface":"tab"},"sentAt":"2026-03-19T10:00:07Z"},"source":"tab-2","sequence":3,"sentAt":"2026-03-19T10:00:07Z"}`)
	state = peers["storefront-2"]
	if state.hellos != 3 || state.expired || state.disconnected {
		t.Fatalf("expected fallback peer to reconnect on fresh hello, got %+v", state)
	}

	emitStorage(`{"name":"clients-fallback","payload":{"kind":"goodbye","topic":"clients","source":{"id":"storefront-2","app":"atlas","surface":"tab"},"sentAt":"2026-03-19T10:00:08Z"},"source":"tab-2","sequence":4,"sentAt":"2026-03-19T10:00:08Z"}`)
	state = peers["storefront-2"]
	if !state.disconnected {
		t.Fatalf("expected fallback peer goodbye to mark the peer disconnected, got %+v", state)
	}
}

func TestPublishClientBinaryCrossTabUsesBroadcastChannel(t *testing.T) {
	var posted js.Value
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		raw := js.Global().Get("Object").New()
		messageListeners := js.Global().Get("Array").New()
		addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].String() == "message" {
				messageListeners.Call("push", args[1])
			}
			return nil
		})
		removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].String() != "message" {
				return nil
			}
			callback := args[1]
			for i := 0; i < messageListeners.Length(); i++ {
				current := messageListeners.Index(i)
				if !current.IsUndefined() && !current.IsNull() && current.Equal(callback) {
					messageListeners.SetIndex(i, js.Null())
				}
			}
			return nil
		})
		emitMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			event := js.Global().Get("Object").New()
			event.Set("data", args[0])
			for i := 0; i < messageListeners.Length(); i++ {
				callback := messageListeners.Index(i)
				if callback.IsUndefined() || callback.IsNull() {
					continue
				}
				callback.Invoke(event)
			}
			return nil
		})
		postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			posted = args[0]
			raw.Call("__emitMessage", args[0])
			return nil
		})
		closeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			raw.Set("__closed", true)
			return nil
		})
		raw.Set("addEventListener", addEventListener)
		raw.Set("removeEventListener", removeEventListener)
		raw.Set("postMessage", postMessage)
		raw.Set("close", closeFn)
		raw.Set("__emitMessage", emitMessage)
		return raw
	})
	defer ctor.Release()
	restoreBroadcast := setGlobalValue("BroadcastChannel", ctor)
	defer restoreBroadcast()

	channel, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "assets"})
	if err != nil {
		t.Fatalf("expected broadcast cross-tab channel, got %v", err)
	}

	var received ClientMessage
	subscription, err := SubscribeClientMessages(channel, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected decoded client message, got %v", err)
		}
		received = message
	})
	if err != nil {
		t.Fatalf("expected client-message subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	self := ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}
	binaryPayload := []byte{1, 2, 3, 4}
	if err := PublishClientBinaryCrossTab(channel, "asset:preview", self, ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: binaryPayload}); err != nil {
		t.Fatalf("expected binary cross-tab publish to succeed, got %v", err)
	}

	if posted.IsUndefined() || posted.IsNull() {
		t.Fatal("expected broadcast channel to capture a posted payload")
	}
	postedMessage := posted.Get("payload")
	if postedMessage.Get("encoding").String() != "binary" || postedMessage.Get("contentType").String() != "application/octet-stream" {
		t.Fatalf("unexpected posted binary metadata: %s %s", postedMessage.Get("encoding").String(), postedMessage.Get("contentType").String())
	}
	postedBytes := make([]byte, postedMessage.Get("payload").Length())
	js.CopyBytesToGo(postedBytes, postedMessage.Get("payload"))
	if string(postedBytes) != string(binaryPayload) {
		t.Fatalf("unexpected posted binary bytes: %v", postedBytes)
	}

	decodedBytes, ok := received.Payload.([]byte)
	if !ok {
		t.Fatalf("expected received payload bytes, got %T", received.Payload)
	}
	if received.Encoding != ClientPayloadBinary || received.ContentType != "application/octet-stream" || string(decodedBytes) != string(binaryPayload) {
		t.Fatalf("unexpected received binary message: %+v payload=%v", received, decodedBytes)
	}
}

func TestPublishClientHelloCarriesDefaultCapabilitiesByTransport(t *testing.T) {
	var posted js.Value
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		raw := js.Global().Get("Object").New()
		listeners := js.Global().Get("Array").New()
		addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].String() == "message" {
				listeners.Call("push", args[1])
			}
			return nil
		})
		removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			posted = args[0]
			return nil
		})
		closeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
		raw.Set("addEventListener", addEventListener)
		raw.Set("removeEventListener", removeEventListener)
		raw.Set("postMessage", postMessage)
		raw.Set("close", closeFn)
		return raw
	})
	defer ctor.Release()
	restoreBroadcast := setGlobalValue("BroadcastChannel", ctor)
	defer restoreBroadcast()

	channel, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients"})
	if err != nil {
		t.Fatalf("expected cross-tab channel, got %v", err)
	}
	if err := PublishClientHello(channel, ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}); err != nil {
		t.Fatalf("expected hello publish to succeed, got %v", err)
	}
	capabilities := posted.Get("payload").Get("capabilities")
	if capabilities.Get("protocolVersion").String() != "v1" {
		t.Fatalf("expected default protocol version, got %q", capabilities.Get("protocolVersion").String())
	}
	encodings := capabilities.Get("encodings")
	if encodings.Length() != 2 || encodings.Index(0).String() != "json" || encodings.Index(1).String() != "binary" {
		t.Fatalf("expected broadcast hello to advertise json and binary, got %#v", encodings)
	}

	restoreBroadcastFallback := setGlobalValue("BroadcastChannel", js.Undefined())
	defer restoreBroadcastFallback()
	window := js.Global().Get("Object").New()
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer removeEventListener.Release()
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()
	storage := js.Global().Get("Object").New()
	var stored string
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() })
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		stored = args[1].String()
		return nil
	})
	defer setItemFn.Release()
	removeItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer removeItemFn.Release()
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer clearFn.Release()
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() })
	defer keyFn.Release()
	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", removeItemFn)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	storage.Set("length", 0)
	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	fallbackChannel, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients-fallback"})
	if err != nil {
		t.Fatalf("expected storage fallback channel, got %v", err)
	}
	if err := PublishClientHello(fallbackChannel, ClientIdentity{ID: "storefront-2", App: "atlas", Surface: "tab"}); err != nil {
		t.Fatalf("expected fallback hello publish to succeed, got %v", err)
	}
	if !strings.Contains(stored, `"encodings":["json"]`) {
		t.Fatalf("expected storage fallback hello to advertise json-only encoding, got %s", stored)
	}
}

func TestPublishClientBinaryCrossTabRejectsStorageFallback(t *testing.T) {
	restoreBroadcast := setGlobalValue("BroadcastChannel", js.Undefined())
	defer restoreBroadcast()

	window := js.Global().Get("Object").New()
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer removeEventListener.Release()
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	storage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() })
	defer getItemFn.Release()
	setItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer setItemFn.Release()
	removeItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer removeItemFn.Release()
	clearFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer clearFn.Release()
	keyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() })
	defer keyFn.Release()
	storage.Set("getItem", getItemFn)
	storage.Set("setItem", setItemFn)
	storage.Set("removeItem", removeItemFn)
	storage.Set("clear", clearFn)
	storage.Set("key", keyFn)
	storage.Set("length", 0)
	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	channel, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "prefs"})
	if err != nil {
		t.Fatalf("expected storage-fallback channel, got %v", err)
	}

	err = PublishClientBinaryCrossTab(channel, "asset:preview", ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}, ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{9, 8, 7}})
	if !IsCode(err, CodeInvalid) {
		t.Fatalf("expected invalid binary transport error, got %v", err)
	}
}

func TestMultiClientBroadcastLateJoinReconnectAndGoodbye(t *testing.T) {
	type mockBroadcastChannel struct {
		raw       js.Value
		listeners js.Value
	}

	channelsByName := map[string][]mockBroadcastChannel{}
	ctor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		name := args[0].String()
		raw := js.Global().Get("Object").New()
		listeners := js.Global().Get("Array").New()
		addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].String() == "message" {
				listeners.Call("push", args[1])
			}
			return nil
		})
		removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if args[0].String() != "message" {
				return nil
			}
			callback := args[1]
			for i := 0; i < listeners.Length(); i++ {
				current := listeners.Index(i)
				if !current.IsUndefined() && !current.IsNull() && current.Equal(callback) {
					listeners.SetIndex(i, js.Null())
				}
			}
			return nil
		})
		postMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			event := js.Global().Get("Object").New()
			event.Set("data", args[0])
			for _, channel := range channelsByName[name] {
				if closed := channel.raw.Get("__closed"); !closed.IsUndefined() && !closed.IsNull() && closed.Bool() {
					continue
				}
				for i := 0; i < channel.listeners.Length(); i++ {
					callback := channel.listeners.Index(i)
					if callback.IsUndefined() || callback.IsNull() {
						continue
					}
					callback.Invoke(event)
				}
			}
			return nil
		})
		closeFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			raw.Set("__closed", true)
			return nil
		})
		raw.Set("addEventListener", addEventListener)
		raw.Set("removeEventListener", removeEventListener)
		raw.Set("postMessage", postMessage)
		raw.Set("close", closeFn)
		channelsByName[name] = append(channelsByName[name], mockBroadcastChannel{raw: raw, listeners: listeners})
		return raw
	})
	defer ctor.Release()
	restoreBroadcast := setGlobalValue("BroadcastChannel", ctor)
	defer restoreBroadcast()

	alpha, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients"})
	if err != nil {
		t.Fatalf("expected alpha cross-tab channel, got %v", err)
	}
	defer alpha.Close()

	alphaSelf := ClientIdentity{ID: "alpha-1", App: "atlas", Surface: "tab-a"}
	betaSelf := ClientIdentity{ID: "beta-1", App: "atlas", Surface: "tab-b"}

	var betaHellos int
	var betaSawGoodbye bool
	var betaSawReconnect bool
	var betaSawResult bool

	alphaSubscription, err := SubscribeClientMessages(alpha, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected alpha subscription to decode messages, got %v", err)
		}
		if message.Kind == ClientQuery && message.Topic == ClientPresenceTopic && message.Source.ID == betaSelf.ID {
			if publishErr := PublishClientResult(alpha, ClientPresenceTopic, alphaSelf, betaSelf.ID, map[string]any{"peer": alphaSelf.Surface}); publishErr != nil {
				t.Fatalf("expected alpha to answer beta query, got %v", publishErr)
			}
		}
	})
	if err != nil {
		t.Fatalf("expected alpha multi-client subscription, got %v", err)
	}
	defer alphaSubscription.Cancel()

	if err := PublishClientHello(alpha, alphaSelf); err != nil {
		t.Fatalf("expected alpha hello to succeed, got %v", err)
	}

	beta, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients"})
	if err != nil {
		t.Fatalf("expected beta cross-tab channel, got %v", err)
	}
	defer beta.Close()

	betaSubscription, err := SubscribeClientMessages(beta, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected beta subscription to decode messages, got %v", err)
		}
		if message.Source.ID != alphaSelf.ID {
			return
		}
		switch message.Kind {
		case ClientHello:
			betaHellos++
			if betaSawGoodbye {
				betaSawReconnect = true
			}
		case ClientGoodbye:
			betaSawGoodbye = true
		case ClientResult:
			if message.Target == betaSelf.ID {
				betaSawResult = true
			}
		}
	})
	if err != nil {
		t.Fatalf("expected beta multi-client subscription, got %v", err)
	}
	defer betaSubscription.Cancel()

	if err := PublishClientHello(alpha, alphaSelf); err != nil {
		t.Fatalf("expected duplicate alpha hello to succeed, got %v", err)
	}
	if err := PublishClientQuery(beta, ClientPresenceTopic, betaSelf); err != nil {
		t.Fatalf("expected beta late-join discovery query to succeed, got %v", err)
	}
	if err := PublishClientGoodbye(alpha, alphaSelf); err != nil {
		t.Fatalf("expected alpha goodbye to succeed, got %v", err)
	}
	if err := alpha.Close(); err != nil {
		t.Fatalf("expected alpha close to succeed, got %v", err)
	}

	alphaReconnect, err := OpenCrossTabChannel(CrossTabChannelOptions{Name: "clients"})
	if err != nil {
		t.Fatalf("expected alpha reconnect channel, got %v", err)
	}
	defer alphaReconnect.Close()
	if err := PublishClientHello(alphaReconnect, alphaSelf); err != nil {
		t.Fatalf("expected alpha reconnect hello to succeed, got %v", err)
	}

	if betaHellos < 2 {
		t.Fatalf("expected beta to observe duplicate or reconnect hello traffic, saw %d hellos", betaHellos)
	}
	if !betaSawResult {
		t.Fatal("expected beta to receive a targeted discovery result from alpha")
	}
	if !betaSawGoodbye {
		t.Fatal("expected beta to observe alpha goodbye before reconnect")
	}
	if !betaSawReconnect {
		t.Fatal("expected beta to observe alpha reconnect hello after goodbye")
	}
}

func TestOpenSecondaryWindowChannelPublishesAndReceivesMessages(t *testing.T) {
	window := js.Global().Get("Object").New()
	location := js.Global().Get("Object").New()
	location.Set("origin", "https://app.example.test")
	window.Set("location", location)
	var opened js.Value
	openFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		opened = js.Global().Get("Object").New()
		opened.Set("closed", false)
		opened.Set("focus", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("__focused", true)
			return nil
		}))
		opened.Set("close", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("closed", true)
			return nil
		}))
		opened.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("__posted", args[0])
			opened.Set("__targetOrigin", args[1].String())
			return nil
		}))
		return opened
	})
	defer openFn.Release()
	var messageListener js.Value
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" {
			messageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" && messageListener.Equal(args[1]) {
			messageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("open", openFn)
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	channel, err := OpenSecondaryWindowChannel(WindowChannelOptions{
		URL:  "/popup.html",
		Name: "inspector",
	})
	if err != nil {
		t.Fatalf("expected popup channel, got %v", err)
	}
	if channel.TargetOrigin() != "https://app.example.test" {
		t.Fatalf("expected same-origin default target, got %q", channel.TargetOrigin())
	}

	var received DecodedWindowEnvelope[struct {
		View string `json:"view"`
	}]
	subscription, err := SubscribeDecodedWindow(channel, func(message DecodedWindowEnvelope[struct {
		View string `json:"view"`
	}], err error) {
		if err != nil {
			t.Fatalf("expected decoded popup message, got %v", err)
		}
		received = message
	})
	if err != nil {
		t.Fatalf("expected popup subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	if err := channel.Publish(map[string]any{"view": "orders"}); err != nil {
		t.Fatalf("expected popup publish to succeed, got %v", err)
	}
	if opened.Get("__targetOrigin").String() != "https://app.example.test" {
		t.Fatalf("expected popup postMessage target origin, got %q", opened.Get("__targetOrigin").String())
	}

	event := js.Global().Get("Object").New()
	event.Set("source", opened)
	event.Set("origin", "https://app.example.test")
	event.Set("data", js.Global().Get("Object").New())
	event.Get("data").Set("name", "inspector")
	event.Get("data").Set("source", "popup-1")
	event.Get("data").Set("sentAt", "2026-03-18T16:15:00Z")
	event.Get("data").Set("payload", js.Global().Get("Object").New())
	event.Get("data").Get("payload").Set("view", "catalog")
	messageListener.Invoke(event)
	if received.Payload.View != "catalog" || received.Name != "inspector" || received.Source != "popup-1" {
		t.Fatalf("unexpected popup message envelope: %+v", received)
	}

	if err := channel.Focus(); err != nil {
		t.Fatalf("expected popup focus to succeed, got %v", err)
	}
	if !opened.Get("__focused").Bool() {
		t.Fatal("expected popup focus helper to call the window focus method")
	}
	if channel.Closed() {
		t.Fatal("expected popup to start open")
	}
	if err := channel.Close(); err != nil {
		t.Fatalf("expected popup close to succeed, got %v", err)
	}
	if !channel.Closed() {
		t.Fatal("expected popup close to mark the window handle closed")
	}
	if err := channel.Publish(map[string]any{"view": "retry"}); !IsCode(err, CodeDisposed) {
		t.Fatalf("expected disposed error after popup close, got %v", err)
	}
}

func TestPublishClientBinaryWindowRoundTripsPayload(t *testing.T) {
	window := js.Global().Get("Object").New()
	location := js.Global().Get("Object").New()
	location.Set("origin", "https://app.example.test")
	window.Set("location", location)
	var opened js.Value
	openFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		opened = js.Global().Get("Object").New()
		opened.Set("closed", false)
		opened.Set("focus", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		opened.Set("close", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("closed", true)
			return nil
		}))
		opened.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("__posted", args[0])
			opened.Set("__targetOrigin", args[1].String())
			return nil
		}))
		return opened
	})
	defer openFn.Release()
	var messageListener js.Value
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" {
			messageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" && messageListener.Equal(args[1]) {
			messageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("open", openFn)
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	channel, err := OpenSecondaryWindowChannel(WindowChannelOptions{URL: "/popup.html", Name: "inspector"})
	if err != nil {
		t.Fatalf("expected popup channel, got %v", err)
	}

	var received ClientMessage
	subscription, err := SubscribeClientWindowMessages(channel, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected decoded client window message, got %v", err)
		}
		received = message
	})
	if err != nil {
		t.Fatalf("expected client-window subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	self := ClientIdentity{ID: "storefront-1", App: "atlas", Surface: "tab"}
	binaryPayload := []byte{5, 6, 7, 8}
	if err := PublishClientBinaryWindow(channel, "asset:preview", self, "popup-1", ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: binaryPayload}); err != nil {
		t.Fatalf("expected binary window publish to succeed, got %v", err)
	}

	posted := opened.Get("__posted")
	if posted.IsUndefined() || posted.IsNull() {
		t.Fatal("expected posted popup message")
	}
	postedMessage := posted.Get("payload")
	if postedMessage.Get("encoding").String() != "binary" || postedMessage.Get("contentType").String() != "application/octet-stream" || postedMessage.Get("target").String() != "popup-1" {
		t.Fatalf("unexpected posted binary window metadata: %#v", postedMessage)
	}
	postedBytes := make([]byte, postedMessage.Get("payload").Length())
	js.CopyBytesToGo(postedBytes, postedMessage.Get("payload"))
	if string(postedBytes) != string(binaryPayload) {
		t.Fatalf("unexpected posted binary window bytes: %v", postedBytes)
	}

	event := js.Global().Get("Object").New()
	event.Set("source", opened)
	event.Set("origin", "https://app.example.test")
	event.Set("data", posted)
	messageListener.Invoke(event)

	decodedBytes, ok := received.Payload.([]byte)
	if !ok {
		t.Fatalf("expected received popup payload bytes, got %T", received.Payload)
	}
	if received.Encoding != ClientPayloadBinary || received.ContentType != "application/octet-stream" || received.Target != "popup-1" || string(decodedBytes) != string(binaryPayload) {
		t.Fatalf("unexpected received popup binary message: %+v payload=%v", received, decodedBytes)
	}
}

func TestMultiClientWindowOrphanedPopupReportsDisposed(t *testing.T) {
	window := js.Global().Get("Object").New()
	location := js.Global().Get("Object").New()
	location.Set("origin", "https://app.example.test")
	window.Set("location", location)
	var opened js.Value
	openFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		opened = js.Global().Get("Object").New()
		opened.Set("closed", false)
		opened.Set("focus", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		opened.Set("close", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("closed", true)
			return nil
		}))
		opened.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("__posted", args[0])
			return nil
		}))
		return opened
	})
	defer openFn.Release()
	var messageListener js.Value
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" {
			messageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" && messageListener.Equal(args[1]) {
			messageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("open", openFn)
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	channel, err := OpenSecondaryWindowChannel(WindowChannelOptions{URL: "/popup.html", Name: "inspector"})
	if err != nil {
		t.Fatalf("expected popup channel, got %v", err)
	}

	var sawHello bool
	subscription, err := SubscribeClientWindowMessages(channel, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected decoded popup lifecycle message, got %v", err)
		}
		if message.Kind == ClientHello && message.Source.ID == "popup-1" {
			sawHello = true
		}
	})
	if err != nil {
		t.Fatalf("expected popup multi-client subscription, got %v", err)
	}
	defer subscription.Cancel()

	event := js.Global().Get("Object").New()
	event.Set("source", opened)
	event.Set("origin", "https://app.example.test")
	event.Set("data", js.Global().Get("Object").New())
	event.Get("data").Set("name", "inspector")
	event.Get("data").Set("source", "popup-window")
	event.Get("data").Set("payload", js.Global().Get("Object").New())
	event.Get("data").Get("payload").Set("kind", "hello")
	event.Get("data").Get("payload").Set("topic", "clients")
	event.Get("data").Get("payload").Set("source", js.Global().Get("Object").New())
	event.Get("data").Get("payload").Get("source").Set("id", "popup-1")
	event.Get("data").Get("payload").Get("source").Set("app", "atlas")
	event.Get("data").Get("payload").Get("source").Set("surface", "popup")
	messageListener.Invoke(event)
	if !sawHello {
		t.Fatal("expected popup hello to be observed before orphaning the handle")
	}

	opened.Set("closed", true)
	if !channel.Closed() {
		t.Fatal("expected popup channel to report closed after the peer handle closes")
	}
	err = PublishClientWindowMessage(channel, ClientMessage{Kind: ClientGoodbye, Topic: ClientPresenceTopic, Source: ClientIdentity{ID: "opener-1", App: "atlas", Surface: "tab"}, Target: "popup-1"})
	if !IsCode(err, CodeDisposed) {
		t.Fatalf("expected disposed error after popup orphaning, got %v", err)
	}
	err = PublishClientBinaryWindow(channel, "asset:preview", ClientIdentity{ID: "opener-1", App: "atlas", Surface: "tab"}, "popup-1", ClientBinaryPayload{ContentType: "application/octet-stream", Bytes: []byte{1, 2}})
	if !IsCode(err, CodeDisposed) {
		t.Fatalf("expected disposed error for binary publish after popup orphaning, got %v", err)
	}
}

func TestWindowOpenerChannelUsesOpenerHandle(t *testing.T) {
	window := js.Global().Get("Object").New()
	location := js.Global().Get("Object").New()
	location.Set("origin", "https://app.example.test")
	window.Set("location", location)
	opener := js.Global().Get("Object").New()
	opener.Set("closed", false)
	opener.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		opener.Set("__posted", args[0])
		opener.Set("__targetOrigin", args[1].String())
		return nil
	}))
	window.Set("opener", opener)
	var messageListener js.Value
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" {
			messageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" && messageListener.Equal(args[1]) {
			messageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	channel, err := WindowOpenerChannel(WindowChannelOptions{Name: "inspector"})
	if err != nil {
		t.Fatalf("expected opener channel, got %v", err)
	}
	if err := channel.Publish(map[string]any{"route": "/orders"}); err != nil {
		t.Fatalf("expected opener publish to succeed, got %v", err)
	}
	if opener.Get("__targetOrigin").String() != "https://app.example.test" {
		t.Fatalf("expected opener target origin, got %q", opener.Get("__targetOrigin").String())
	}
	if err := channel.Close(); !IsCode(err, CodeUnavailable) {
		t.Fatalf("expected opener channel close to stay unavailable, got %v", err)
	}

	var receivedName string
	subscription, err := channel.Subscribe(func(message WindowEnvelope, err error) {
		if err != nil {
			t.Fatalf("expected opener message to decode, got %v", err)
		}
		receivedName = message.Name
	})
	if err != nil {
		t.Fatalf("expected opener subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	event := js.Global().Get("Object").New()
	event.Set("source", opener)
	event.Set("origin", "https://app.example.test")
	event.Set("data", js.Global().Get("Object").New())
	event.Get("data").Set("name", "inspector")
	messageListener.Invoke(event)
	if receivedName != "inspector" {
		t.Fatalf("expected opener message name, got %q", receivedName)
	}
}

func TestMultiClientWindowOpenerLifecycleFlow(t *testing.T) {
	window := js.Global().Get("Object").New()
	location := js.Global().Get("Object").New()
	location.Set("origin", "https://app.example.test")
	window.Set("location", location)
	opener := js.Global().Get("Object").New()
	opener.Set("closed", false)
	opener.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		opener.Set("__posted", args[0])
		opener.Set("__targetOrigin", args[1].String())
		return nil
	}))
	window.Set("opener", opener)
	var messageListener js.Value
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" {
			messageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" && messageListener.Equal(args[1]) {
			messageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	channel, err := WindowOpenerChannel(WindowChannelOptions{Name: "inspector"})
	if err != nil {
		t.Fatalf("expected opener channel, got %v", err)
	}

	popupSelf := ClientIdentity{ID: "popup-1", App: "atlas", Surface: "popup"}
	if err := PublishClientHelloWindow(channel, popupSelf); err != nil {
		t.Fatalf("expected opener hello publish to succeed, got %v", err)
	}
	posted := opener.Get("__posted")
	if posted.Get("payload").Get("kind").String() != "hello" || posted.Get("payload").Get("topic").String() != "clients" {
		t.Fatalf("expected opener hello payload, got %#v", posted)
	}
	if posted.Get("payload").Get("capabilities").Get("protocolVersion").String() != "v1" {
		t.Fatalf("expected opener hello to advertise capabilities, got %#v", posted.Get("payload").Get("capabilities"))
	}

	if err := PublishClientGoodbyeWindow(channel, popupSelf); err != nil {
		t.Fatalf("expected opener goodbye publish to succeed, got %v", err)
	}
	posted = opener.Get("__posted")
	if posted.Get("payload").Get("kind").String() != "goodbye" {
		t.Fatalf("expected opener goodbye payload, got %#v", posted)
	}

	var sawHello bool
	var sawGoodbye bool
	subscription, err := SubscribeClientWindowMessages(channel, func(message ClientMessage, err error) {
		if err != nil {
			t.Fatalf("expected decoded opener client message, got %v", err)
		}
		switch message.Kind {
		case ClientHello:
			if message.Source.ID == "opener-1" {
				sawHello = true
			}
		case ClientGoodbye:
			if message.Source.ID == "opener-1" {
				sawGoodbye = true
			}
		}
	})
	if err != nil {
		t.Fatalf("expected opener multi-client subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	emitWindowMessage := func(kind string) {
		event := js.Global().Get("Object").New()
		event.Set("source", opener)
		event.Set("origin", "https://app.example.test")
		event.Set("data", js.Global().Get("Object").New())
		event.Get("data").Set("name", "inspector")
		event.Get("data").Set("source", "opener-window")
		event.Get("data").Set("payload", js.Global().Get("Object").New())
		event.Get("data").Get("payload").Set("kind", kind)
		event.Get("data").Get("payload").Set("topic", "clients")
		event.Get("data").Get("payload").Set("source", js.Global().Get("Object").New())
		event.Get("data").Get("payload").Get("source").Set("id", "opener-1")
		event.Get("data").Get("payload").Get("source").Set("app", "atlas")
		event.Get("data").Get("payload").Get("source").Set("surface", "tab")
		messageListener.Invoke(event)
	}

	emitWindowMessage("hello")
	emitWindowMessage("goodbye")
	if !sawHello || !sawGoodbye {
		t.Fatalf("expected opener lifecycle traffic to decode, sawHello=%v sawGoodbye=%v", sawHello, sawGoodbye)
	}
}

func TestMultiClientWindowSubscriptionRejectsOriginMismatchAndStaleOpener(t *testing.T) {
	window := js.Global().Get("Object").New()
	location := js.Global().Get("Object").New()
	location.Set("origin", "https://app.example.test")
	window.Set("location", location)
	opener := js.Global().Get("Object").New()
	opener.Set("closed", false)
	opener.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return nil
	}))
	window.Set("opener", opener)
	var messageListener js.Value
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" {
			messageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" && messageListener.Equal(args[1]) {
			messageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	channel, err := WindowOpenerChannel(WindowChannelOptions{Name: "inspector"})
	if err != nil {
		t.Fatalf("expected opener channel, got %v", err)
	}

	var originMismatchErr error
	subscription, err := SubscribeClientWindowMessages(channel, func(message ClientMessage, err error) {
		if err != nil {
			originMismatchErr = err
			return
		}
	})
	if err != nil {
		t.Fatalf("expected opener security subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	event := js.Global().Get("Object").New()
	event.Set("source", opener)
	event.Set("origin", "https://evil.example.test")
	event.Set("data", js.Global().Get("Object").New())
	event.Get("data").Set("name", "inspector")
	event.Get("data").Set("payload", js.Global().Get("Object").New())
	event.Get("data").Get("payload").Set("kind", "hello")
	event.Get("data").Get("payload").Set("topic", "clients")
	event.Get("data").Get("payload").Set("source", js.Global().Get("Object").New())
	event.Get("data").Get("payload").Get("source").Set("id", "opener-1")
	event.Get("data").Get("payload").Get("source").Set("app", "atlas")
	event.Get("data").Get("payload").Get("source").Set("surface", "tab")
	messageListener.Invoke(event)
	if !IsCode(originMismatchErr, CodeUnauthorized) {
		t.Fatalf("expected target-origin mismatch to produce unauthorized error, got %v", originMismatchErr)
	}

	opener.Set("closed", true)
	if err := PublishClientHelloWindow(channel, ClientIdentity{ID: "popup-1", App: "atlas", Surface: "popup", Role: "operator"}); !IsCode(err, CodeDisposed) {
		t.Fatalf("expected stale opener handle to reject hello publish, got %v", err)
	}
	if err := PublishClientIntent(channel, "operator:inventory", ClientIdentity{ID: "popup-1", App: "atlas", Surface: "popup", Role: "operator"}, "opener-1", map[string]any{"sku": "SKU-44"}); !IsCode(err, CodeDisposed) {
		t.Fatalf("expected stale opener handle to reject privileged intent publish, got %v", err)
	}
}

func TestSurfaceSignalWindowHelpersPublishAndDecode(t *testing.T) {
	window := js.Global().Get("Object").New()
	location := js.Global().Get("Object").New()
	location.Set("origin", "https://app.example.test")
	window.Set("location", location)
	var opened js.Value
	openFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		opened = js.Global().Get("Object").New()
		opened.Set("closed", false)
		opened.Set("postMessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("__posted", args[0])
			return nil
		}))
		opened.Set("focus", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		opened.Set("close", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			opened.Set("closed", true)
			return nil
		}))
		return opened
	})
	defer openFn.Release()
	var messageListener js.Value
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" {
			messageListener = args[1]
		}
		return nil
	})
	defer addEventListener.Release()
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if args[0].String() == "message" && messageListener.Equal(args[1]) {
			messageListener = js.Null()
		}
		return nil
	})
	defer removeEventListener.Release()
	window.Set("open", openFn)
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	restoreWindow := setGlobalValue("window", window)
	defer restoreWindow()

	channel, err := OpenSecondaryWindowChannel(WindowChannelOptions{
		URL:  "/popup.html",
		Name: "ops",
	})
	if err != nil {
		t.Fatalf("expected popup channel, got %v", err)
	}

	if err := PublishRouteFocus(channel, "/orders/42", "tab=activity", "order-heading"); err != nil {
		t.Fatalf("expected route focus publish, got %v", err)
	}
	posted := opened.Get("__posted")
	if posted.IsUndefined() || posted.IsNull() {
		t.Fatal("expected helper publish to post a window payload")
	}
	if posted.Get("payload").Get("kind").String() != "route" || posted.Get("payload").Get("route").Get("path").String() != "/orders/42" {
		t.Fatalf("unexpected posted route signal: %#v", posted)
	}

	var received DecodedWindowEnvelope[SurfaceSignal]
	subscription, err := SubscribeSurfaceSignals(channel, func(message DecodedWindowEnvelope[SurfaceSignal], err error) {
		if err != nil {
			t.Fatalf("expected decoded surface signal, got %v", err)
		}
		received = message
	})
	if err != nil {
		t.Fatalf("expected surface subscription to succeed, got %v", err)
	}
	defer subscription.Cancel()

	event := js.Global().Get("Object").New()
	event.Set("source", opened)
	event.Set("origin", "https://app.example.test")
	event.Set("data", js.Global().Get("Object").New())
	event.Get("data").Set("name", "ops")
	event.Get("data").Set("source", "popup-1")
	event.Get("data").Set("payload", js.Global().Get("Object").New())
	event.Get("data").Get("payload").Set("kind", "intent")
	event.Get("data").Get("payload").Set("intent", js.Global().Get("Object").New())
	event.Get("data").Get("payload").Get("intent").Set("action", "focus-panel")
	event.Get("data").Get("payload").Get("intent").Set("target", "audit-log")
	event.Get("data").Get("payload").Get("intent").Set("params", js.Global().Get("Object").New())
	event.Get("data").Get("payload").Get("intent").Get("params").Set("tab", "alerts")
	messageListener.Invoke(event)

	if received.Name != "ops" || received.Source != "popup-1" {
		t.Fatalf("unexpected decoded surface envelope metadata: %+v", received)
	}
	if received.Payload.Kind != SurfaceSignalIntent || received.Payload.Intent == nil || received.Payload.Intent.Action != SurfaceIntentFocusPanel || received.Payload.Intent.Target != "audit-log" || received.Payload.Intent.Params["tab"] != "alerts" {
		t.Fatalf("unexpected decoded surface payload: %+v", received.Payload)
	}
}

func BenchmarkLocalStorageGetItem(b *testing.B) {
	storage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return "dark"
	})
	defer getItemFn.Release()
	storage.Set("getItem", getItemFn)
	storage.Set("setItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("removeItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("clear", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	storage.Set("key", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return js.Null() }))
	storage.Set("length", 1)
	restoreStorage := setGlobalValue("localStorage", storage)
	defer restoreStorage()

	local, err := LocalStorage()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value, ok, err := local.GetItem("theme")
		if err != nil || !ok || value != "dark" {
			b.Fatalf("unexpected storage read: %q ok=%t err=%v", value, ok, err)
		}
	}
}

func BenchmarkModuleCall(b *testing.B) {
	exportFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.ValueOf(args[0].Float() + 1)
	})
	defer exportFn.Release()
	moduleNS := js.Global().Get("Object").New()
	moduleNS.Set("next", exportFn)
	importFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makePromise(moduleNS)
	})
	defer importFn.Release()
	restoreImport := setGlobalValue("__gwcImportModule", importFn)
	defer restoreImport()

	module, err := ImportModule(context.Background(), "/bench/module.js")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value, err := module.Call(context.Background(), "next", 41)
		if err != nil || value.(float64) != 42 {
			b.Fatalf("unexpected module call result %#v err=%v", value, err)
		}
	}
}
