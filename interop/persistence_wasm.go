//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"errors"
	"sort"
	"strings"
	"syscall/js"
)

const defaultPersistentDatabaseName = "GoWebComponents"

type persistentStoreConfig struct {
	name               string
	databaseName       string
	version            int
	deleteOnCorruption bool
	onBlocked          func(PersistentStoreBlockedEvent)
	fallbackResolver   func() (Storage, error)
	fallbackBackend    string
}

type persistentStoreFailure struct {
	code        ErrorCode
	message     string
	recoverable bool
}

func OpenPersistentStore(ctx context.Context, options PersistentStoreOptions) (PersistentStore, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	config, err := resolvePersistentStoreConfig(options)
	if err != nil {
		return PersistentStore{}, err
	}
	store, err := openIndexedDBPersistentStore(ctx, config)
	if err == nil {
		return store, nil
	}
	if !IsCode(err, CodeUnavailable) || config.fallbackResolver == nil {
		return PersistentStore{}, err
	}
	storage, fallbackErr := config.fallbackResolver()
	if fallbackErr != nil {
		return PersistentStore{}, fallbackErr
	}
	return wrapStoragePersistentStore(storage, config.fallbackBackend), nil
}

func resolvePersistentStoreConfig(options PersistentStoreOptions) (persistentStoreConfig, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return persistentStoreConfig{}, wrapError("OpenPersistentStore", options.Name, CodeInvalid, errors.New("store name is empty"))
	}
	databaseName := strings.TrimSpace(options.DatabaseName)
	if databaseName == "" {
		databaseName = defaultPersistentDatabaseName
	}
	version := options.Version
	if version <= 0 {
		version = 1
	}
	fallbackBackend := strings.TrimSpace(options.FallbackBackend)
	if fallbackBackend == "" && options.FallbackResolver != nil {
		fallbackBackend = "storage"
	}
	return persistentStoreConfig{
		name:               name,
		databaseName:       databaseName,
		version:            version,
		deleteOnCorruption: options.DeleteOnCorruption,
		onBlocked:          options.OnBlocked,
		fallbackResolver:   options.FallbackResolver,
		fallbackBackend:    fallbackBackend,
	}, nil
}

func openIndexedDBPersistentStore(ctx context.Context, config persistentStoreConfig) (PersistentStore, error) {
	return openIndexedDBPersistentStoreWithRecovery(ctx, config, config.deleteOnCorruption)
}

func openIndexedDBPersistentStoreWithRecovery(ctx context.Context, config persistentStoreConfig, allowRecovery bool) (PersistentStore, error) {
	rawIndexedDB := js.Global().Get("indexedDB")
	if rawIndexedDB.IsUndefined() || rawIndexedDB.IsNull() {
		return PersistentStore{}, unavailable("OpenPersistentStore", config.name)
	}

	request := rawIndexedDB.Call("open", config.databaseName, config.version)
	resultCh := make(chan js.Value, 1)
	failureCh := make(chan persistentStoreFailure, 1)
	var upgradeErr error

	var onUpgrade js.Func
	var onSuccess js.Func
	var onError js.Func
	var onBlocked js.Func
	cleanup := func() {
		request.Set("onupgradeneeded", js.Undefined())
		request.Set("onsuccess", js.Undefined())
		request.Set("onerror", js.Undefined())
		request.Set("onblocked", js.Undefined())
		onUpgrade.Release()
		onSuccess.Release()
		onError.Release()
		onBlocked.Release()
	}

	onUpgrade = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		db := request.Get("result")
		if db.IsUndefined() || db.IsNull() {
			upgradeErr = wrapError("OpenPersistentStore", config.name, CodeUnavailable, errors.New("indexedDB open request returned no database handle"))
			return nil
		}
		stores := db.Get("objectStoreNames")
		contains := false
		if !stores.IsUndefined() && !stores.IsNull() {
			contains = stores.Call("contains", config.name).Bool()
		}
		if contains {
			return nil
		}
		options := js.Global().Get("Object").New()
		options.Set("keyPath", "key")
		db.Call("createObjectStore", config.name, options)
		return nil
	})
	onSuccess = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if upgradeErr != nil {
			reportPersistentStoreFailure(failureCh, persistentStoreFailure{code: CodeInvalid, message: upgradeErr.Error()})
			return nil
		}
		db := request.Get("result")
		if db.IsUndefined() || db.IsNull() {
			reportPersistentStoreFailure(failureCh, persistentStoreFailure{code: CodeUnavailable, message: "indexedDB open request returned no database handle"})
			return nil
		}
		stores := db.Get("objectStoreNames")
		if stores.IsUndefined() || stores.IsNull() || !stores.Call("contains", config.name).Bool() {
			reportPersistentStoreFailure(failureCh, persistentStoreFailure{code: CodeInvalid, message: "object store is missing; bump the database version before adding a new store"})
			return nil
		}
		reportPersistentStoreResult(resultCh, db)
		return nil
	})
	onError = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reportPersistentStoreFailure(failureCh, classifyPersistentStoreFailure(request.Get("error"), "indexedDB open request failed"))
		return nil
	})
	onBlocked = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if config.onBlocked != nil {
			config.onBlocked(PersistentStoreBlockedEvent{
				DatabaseName:     config.databaseName,
				StoreName:        config.name,
				RequestedVersion: config.version,
			})
		}
		reportPersistentStoreFailure(failureCh, persistentStoreFailure{code: CodeBlocked, message: "indexedDB upgrade is blocked by another open tab, worker, or window"})
		return nil
	})

	request.Set("onupgradeneeded", onUpgrade)
	request.Set("onsuccess", onSuccess)
	request.Set("onerror", onError)
	request.Set("onblocked", onBlocked)

	defer cleanup()

	select {
	case db := <-resultCh:
		return newIndexedDBPersistentStore(db, config), nil
	case failure := <-failureCh:
		if allowRecovery && config.deleteOnCorruption && failure.recoverable {
			if err := deleteIndexedDBDatabase(ctx, config.databaseName); err == nil {
				return openIndexedDBPersistentStoreWithRecovery(ctx, config, false)
			} else {
				return PersistentStore{}, wrapError("OpenPersistentStore", config.name, failure.code, errors.New(failure.message+"; automatic database reset failed: "+err.Error()))
			}
		}
		return PersistentStore{}, wrapError("OpenPersistentStore", config.name, failure.code, errors.New(failure.message))
	case <-ctx.Done():
		return PersistentStore{}, persistentContextError("OpenPersistentStore", config.name, ctx.Err())
	}
}

func newIndexedDBPersistentStore(db js.Value, config persistentStoreConfig) PersistentStore {
	backend := func() string { return "indexedDB" }
	target := config.databaseName + "/" + config.name
	var onVersionChange js.Func
	onVersionChange = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if closeFn := db.Get("close"); closeFn.Type() == js.TypeFunction {
			closeFn.Invoke()
		}
		return nil
	})
	db.Set("onversionchange", onVersionChange)
	withStore := func(ctx context.Context, mode string, op string, run func(js.Value) (js.Value, error)) (js.Value, error) {
		if ctx == nil {
			ctx = context.Background()
		}
		transaction := db.Call("transaction", config.name, mode)
		store := transaction.Call("objectStore", config.name)
		request, err := run(store)
		if err != nil {
			return js.Undefined(), err
		}
		return awaitIDBRequest(ctx, op, target, request)
	}
	return PersistentStore{
		backend: backend,
		getItem: func(ctx context.Context, key string) (string, bool, error) {
			result, err := withStore(ctx, "readonly", "PersistentStore.GetItem", func(store js.Value) (js.Value, error) {
				return store.Call("get", key), nil
			})
			if err != nil {
				return "", false, err
			}
			if result.IsUndefined() || result.IsNull() {
				return "", false, nil
			}
			value := result.Get("value")
			if value.IsUndefined() || value.IsNull() {
				return "", false, nil
			}
			return value.String(), true, nil
		},
		setItem: func(ctx context.Context, key string, value string) error {
			entry := js.Global().Get("Object").New()
			entry.Set("key", key)
			entry.Set("value", value)
			_, err := withStore(ctx, "readwrite", "PersistentStore.SetItem", func(store js.Value) (js.Value, error) {
				return store.Call("put", entry), nil
			})
			return err
		},
		removeItem: func(ctx context.Context, key string) error {
			_, err := withStore(ctx, "readwrite", "PersistentStore.RemoveItem", func(store js.Value) (js.Value, error) {
				return store.Call("delete", key), nil
			})
			return err
		},
		clear: func(ctx context.Context) error {
			_, err := withStore(ctx, "readwrite", "PersistentStore.Clear", func(store js.Value) (js.Value, error) {
				return store.Call("clear"), nil
			})
			return err
		},
		keys: func(ctx context.Context) ([]string, error) {
			result, err := withStore(ctx, "readonly", "PersistentStore.Keys", func(store js.Value) (js.Value, error) {
				return store.Call("getAllKeys"), nil
			})
			if err != nil {
				return nil, err
			}
			length := result.Length()
			keys := make([]string, 0, length)
			for i := 0; i < length; i++ {
				keys = append(keys, result.Index(i).String())
			}
			sort.Strings(keys)
			return keys, nil
		},
		length: func(ctx context.Context) (int, error) {
			result, err := withStore(ctx, "readonly", "PersistentStore.Len", func(store js.Value) (js.Value, error) {
				return store.Call("count"), nil
			})
			if err != nil {
				return 0, err
			}
			return result.Int(), nil
		},
		close: func() error {
			db.Set("onversionchange", js.Undefined())
			onVersionChange.Release()
			if closeFn := db.Get("close"); closeFn.Type() == js.TypeFunction {
				closeFn.Invoke()
			}
			return nil
		},
	}
}

func wrapStoragePersistentStore(storage Storage, backend string) PersistentStore {
	return PersistentStore{
		backend: func() string { return backend },
		getItem: func(ctx context.Context, key string) (string, bool, error) {
			return storage.GetItem(key)
		},
		setItem: func(ctx context.Context, key string, value string) error {
			return storage.SetItem(key, value)
		},
		removeItem: func(ctx context.Context, key string) error {
			return storage.RemoveItem(key)
		},
		clear: func(ctx context.Context) error {
			return storage.Clear()
		},
		keys: func(ctx context.Context) ([]string, error) {
			length, err := storage.Len()
			if err != nil {
				return nil, err
			}
			keys := make([]string, 0, length)
			for index := 0; index < length; index++ {
				key, ok, keyErr := storage.Key(index)
				if keyErr != nil {
					return nil, keyErr
				}
				if ok {
					keys = append(keys, key)
				}
			}
			sort.Strings(keys)
			return keys, nil
		},
		length: func(ctx context.Context) (int, error) {
			return storage.Len()
		},
		close: func() error { return nil },
	}
}

func awaitIDBRequest(ctx context.Context, op string, target string, request js.Value) (js.Value, error) {
	resultCh := make(chan js.Value, 1)
	failureCh := make(chan persistentStoreFailure, 1)
	var onSuccess js.Func
	var onError js.Func
	cleanup := func() {
		request.Set("onsuccess", js.Undefined())
		request.Set("onerror", js.Undefined())
		onSuccess.Release()
		onError.Release()
	}
	onSuccess = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reportPersistentStoreResult(resultCh, request.Get("result"))
		return nil
	})
	onError = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reportPersistentStoreFailure(failureCh, classifyPersistentStoreFailure(request.Get("error"), "indexedDB request failed"))
		return nil
	})
	request.Set("onsuccess", onSuccess)
	request.Set("onerror", onError)
	defer cleanup()

	select {
	case result := <-resultCh:
		return result, nil
	case failure := <-failureCh:
		return js.Undefined(), wrapError(op, target, failure.code, errors.New(failure.message))
	case <-ctx.Done():
		return js.Undefined(), persistentContextError(op, target, ctx.Err())
	}
}

func reportPersistentStoreResult(ch chan js.Value, value js.Value) {
	select {
	case ch <- value:
	default:
	}
}

func reportPersistentStoreFailure(ch chan persistentStoreFailure, failure persistentStoreFailure) {
	select {
	case ch <- failure:
	default:
	}
}

func classifyPersistentStoreFailure(rawError js.Value, fallbackMessage string) persistentStoreFailure {
	message := strings.TrimSpace(jsValueSummary(rawError))
	if message == "" {
		message = fallbackMessage
	}
	name := strings.TrimSpace(rawError.Get("name").String())
	messageLower := strings.ToLower(message)
	switch {
	case name == "QuotaExceededError" || strings.Contains(messageLower, "quota"):
		return persistentStoreFailure{code: CodeQuotaExceeded, message: message}
	case name == "VersionError" || name == "ConstraintError":
		return persistentStoreFailure{code: CodeInvalid, message: message}
	case name == "InvalidStateError" || name == "UnknownError" || strings.Contains(messageLower, "corrupt") || strings.Contains(messageLower, "corruption") || strings.Contains(messageLower, "malformed"):
		return persistentStoreFailure{code: CodeUnavailable, message: message, recoverable: true}
	default:
		return persistentStoreFailure{code: CodeUnavailable, message: message}
	}
}

func deleteIndexedDBDatabase(ctx context.Context, databaseName string) error {
	rawIndexedDB := js.Global().Get("indexedDB")
	if rawIndexedDB.IsUndefined() || rawIndexedDB.IsNull() {
		return unavailable("OpenPersistentStore", databaseName)
	}
	deleteFn := rawIndexedDB.Get("deleteDatabase")
	if deleteFn.Type() != js.TypeFunction {
		return wrapError("OpenPersistentStore", databaseName, CodeUnavailable, errors.New("indexedDB deleteDatabase is unavailable"))
	}
	request := deleteFn.Invoke(databaseName)
	resultCh := make(chan struct{}, 1)
	failureCh := make(chan persistentStoreFailure, 1)
	var onSuccess js.Func
	var onError js.Func
	var onBlocked js.Func
	cleanup := func() {
		request.Set("onsuccess", js.Undefined())
		request.Set("onerror", js.Undefined())
		request.Set("onblocked", js.Undefined())
		onSuccess.Release()
		onError.Release()
		onBlocked.Release()
	}
	onSuccess = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		select {
		case resultCh <- struct{}{}:
		default:
		}
		return nil
	})
	onError = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reportPersistentStoreFailure(failureCh, classifyPersistentStoreFailure(request.Get("error"), "indexedDB delete request failed"))
		return nil
	})
	onBlocked = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reportPersistentStoreFailure(failureCh, persistentStoreFailure{code: CodeBlocked, message: "indexedDB delete is blocked by another open tab, worker, or window"})
		return nil
	})
	request.Set("onsuccess", onSuccess)
	request.Set("onerror", onError)
	request.Set("onblocked", onBlocked)
	defer cleanup()

	select {
	case <-resultCh:
		return nil
	case failure := <-failureCh:
		return wrapError("OpenPersistentStore", databaseName, failure.code, errors.New(failure.message))
	case <-ctx.Done():
		return persistentContextError("OpenPersistentStore", databaseName, ctx.Err())
	}
}

func persistentContextError(op string, target string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return wrapError(op, target, CodeTimeout, err)
	}
	return wrapError(op, target, CodeCancelled, err)
}
