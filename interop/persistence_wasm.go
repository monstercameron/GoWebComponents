//go:build js && wasm

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

// OpenPersistentStore opens an IndexedDB-backed PersistentStore with the given options.
func OpenPersistentStore(parseCtx context.Context, parseOptions PersistentStoreOptions) (PersistentStore, error) {
	parseConfig, parseErr := resolvePersistentStoreConfig(parseOptions)
	if parseErr != nil {
		return PersistentStore{}, parseErr
	}
	store, parseErr := openIndexedDBPersistentStore(parseCtx, parseConfig)
	if parseErr == nil {
		return store, nil
	}
	if !IsCode(parseErr, CodeUnavailable) || parseConfig.fallbackResolver == nil {
		return PersistentStore{}, parseErr
	}
	parseStorage, parseFallbackErr := parseConfig.fallbackResolver()
	if parseFallbackErr != nil {
		return PersistentStore{}, parseFallbackErr
	}
	return wrapStoragePersistentStore(parseStorage, parseConfig.fallbackBackend), nil
}

func resolvePersistentStoreConfig(parseOptions PersistentStoreOptions) (persistentStoreConfig, error) {
	parseName := strings.TrimSpace(parseOptions.Name)
	if parseName == "" {
		return persistentStoreConfig{}, wrapError("OpenPersistentStore", parseOptions.Name, CodeInvalid, errors.New("store name is empty"))
	}
	parseDatabaseName := strings.TrimSpace(parseOptions.DatabaseName)
	if parseDatabaseName == "" {
		parseDatabaseName = defaultPersistentDatabaseName
	}
	parseVersion := parseOptions.Version
	if parseVersion <= 0 {
		parseVersion = 1
	}
	parseFallbackBackend := strings.TrimSpace(parseOptions.FallbackBackend)
	if parseFallbackBackend == "" && parseOptions.FallbackResolver != nil {
		parseFallbackBackend = "storage"
	}
	return persistentStoreConfig{
		name:               parseName,
		databaseName:       parseDatabaseName,
		version:            parseVersion,
		deleteOnCorruption: parseOptions.DeleteOnCorruption,
		onBlocked:          parseOptions.OnBlocked,
		fallbackResolver:   parseOptions.FallbackResolver,
		fallbackBackend:    parseFallbackBackend,
	}, nil
}

func openIndexedDBPersistentStore(parseCtx context.Context, parseConfig persistentStoreConfig) (PersistentStore, error) {
	return openIndexedDBPersistentStoreWithRecovery(parseCtx, parseConfig, parseConfig.deleteOnCorruption)
}

func openIndexedDBPersistentStoreWithRecovery(parseCtx context.Context, parseConfig persistentStoreConfig, isAllowRecovery bool) (PersistentStore, error) {
	parseRawIndexedDB := js.Global().Get("indexedDB")
	if parseRawIndexedDB.IsUndefined() || parseRawIndexedDB.IsNull() {
		return PersistentStore{}, unavailable("OpenPersistentStore", parseConfig.name)
	}

	parseRequest := parseRawIndexedDB.Call("open", parseConfig.databaseName, parseConfig.version)
	parseResultCh := make(chan js.Value, 1)
	parseFailureCh := make(chan persistentStoreFailure, 1)
	var parseUpgradeErr error

	var parseOnUpgrade js.Func
	var parseOnSuccess js.Func
	var parseOnError js.Func
	var parseOnBlocked js.Func
	parseCleanup := func() {
		parseRequest.Set("onupgradeneeded", js.Undefined())
		parseRequest.Set("onsuccess", js.Undefined())
		parseRequest.Set("onerror", js.Undefined())
		parseRequest.Set("onblocked", js.Undefined())
		parseOnUpgrade.Release()
		parseOnSuccess.Release()
		parseOnError.Release()
		parseOnBlocked.Release()
	}

	parseOnUpgrade = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("openIndexedDBPersistentStoreWithRecovery callback")
		parseDb := parseRequest.Get("result")
		if parseDb.IsUndefined() || parseDb.IsNull() {
			parseUpgradeErr = wrapError("OpenPersistentStore", parseConfig.name, CodeUnavailable, errors.New("indexedDB open request returned no database handle"))
			return nil
		}
		parseStores := parseDb.Get("objectStoreNames")
		isParseContains := false
		if !parseStores.IsUndefined() && !parseStores.IsNull() {
			isParseContains = parseStores.Call("contains", parseConfig.name).Bool()
		}
		if isParseContains {
			return nil
		}
		parseOptions := js.Global().Get("Object").New()
		parseOptions.Set("keyPath", "key")
		parseDb.Call("createObjectStore", parseConfig.name, parseOptions)
		return nil
	})
	parseOnSuccess = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer RecoverContainedPanic("openIndexedDBPersistentStoreWithRecovery callback")
		if parseUpgradeErr != nil {
			reportPersistentStoreFailure(parseFailureCh, persistentStoreFailure{code: CodeInvalid, message: parseUpgradeErr.Error()})
			return nil
		}
		parseDb2 := parseRequest.Get("result")
		if parseDb2.IsUndefined() || parseDb2.IsNull() {
			reportPersistentStoreFailure(parseFailureCh, persistentStoreFailure{code: CodeUnavailable, message: "indexedDB open request returned no database handle"})
			return nil
		}
		parseStores2 := parseDb2.Get("objectStoreNames")
		if parseStores2.IsUndefined() || parseStores2.IsNull() || !parseStores2.Call("contains", parseConfig.name).Bool() {
			reportPersistentStoreFailure(parseFailureCh, persistentStoreFailure{code: CodeInvalid, message: "object store is missing; bump the database version before adding a new store"})
			return nil
		}
		reportPersistentStoreResult(parseResultCh, parseDb2)
		return nil
	})
	parseOnError = js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		defer RecoverContainedPanic("openIndexedDBPersistentStoreWithRecovery callback")
		reportPersistentStoreFailure(parseFailureCh, classifyPersistentStoreFailure(parseRequest.Get("error"), "indexedDB open request failed"))
		return nil
	})
	parseOnBlocked = js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		defer RecoverContainedPanic("openIndexedDBPersistentStoreWithRecovery callback")
		if parseConfig.onBlocked != nil {
			parseConfig.onBlocked(PersistentStoreBlockedEvent{
				DatabaseName:     parseConfig.databaseName,
				StoreName:        parseConfig.name,
				RequestedVersion: parseConfig.version,
			})
		}
		reportPersistentStoreFailure(parseFailureCh, persistentStoreFailure{code: CodeBlocked, message: "indexedDB upgrade is blocked by another open tab, worker, or window"})
		return nil
	})

	parseRequest.Set("onupgradeneeded", parseOnUpgrade)
	parseRequest.Set("onsuccess", parseOnSuccess)
	parseRequest.Set("onerror", parseOnError)
	parseRequest.Set("onblocked", parseOnBlocked)

	defer parseCleanup()

	select {
	case parseDb3 := <-parseResultCh:
		return newIndexedDBPersistentStore(parseDb3, parseConfig), nil
	case parseFailure := <-parseFailureCh:
		if isAllowRecovery && parseConfig.deleteOnCorruption && parseFailure.recoverable {
			if parseErr := deleteIndexedDBDatabase(parseCtx, parseConfig.databaseName); parseErr == nil {
				return openIndexedDBPersistentStoreWithRecovery(parseCtx, parseConfig, false)
			} else {
				return PersistentStore{}, wrapError("OpenPersistentStore", parseConfig.name, parseFailure.code, errors.New(parseFailure.message+"; automatic database reset failed: "+parseErr.Error()))
			}
		}
		return PersistentStore{}, wrapError("OpenPersistentStore", parseConfig.name, parseFailure.code, errors.New(parseFailure.message))
	case <-parseCtx.Done():
		return PersistentStore{}, persistentContextError("OpenPersistentStore", parseConfig.name, parseCtx.Err())
	}
}

func newIndexedDBPersistentStore(parseDb js.Value, parseConfig persistentStoreConfig) PersistentStore {
	parseBackend := func() string { return "indexedDB" }
	parseTarget := parseConfig.databaseName + "/" + parseConfig.name
	var parseOnVersionChange js.Func
	parseOnVersionChange = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("newIndexedDBPersistentStore callback")
		if parseCloseFn := parseDb.Get("close"); parseCloseFn.Type() == js.TypeFunction {
			parseDb.Call("close")
		}
		return nil
	})
	parseDb.Set("onversionchange", parseOnVersionChange)
	parseWithStore := func(parseCtx context.Context, parseMode string, parseOp string, parseRun func(js.Value) (js.Value, error)) (js.Value, error) {
		if parseCtx == nil {
			parseCtx = context.Background()
		}
		parseTransaction := parseDb.Call("transaction", parseConfig.name, parseMode)
		store := parseTransaction.Call("objectStore", parseConfig.name)
		parseRequest, parseErr := parseRun(store)
		if parseErr != nil {
			return js.Undefined(), parseErr
		}
		return awaitIDBRequest(parseCtx, parseOp, parseTarget, parseRequest)
	}
	return PersistentStore{
		backend: parseBackend,
		getItem: func(parseCtx2 context.Context, parseKey string) (string, bool, error) {
			parseResult, parseErr2 := parseWithStore(parseCtx2, "readonly", "PersistentStore.GetItem", func(store js.Value) (js.Value, error) {
				return store.Call("get", parseKey), nil
			})
			if parseErr2 != nil {
				return "", false, parseErr2
			}
			if parseResult.IsUndefined() || parseResult.IsNull() {
				return "", false, nil
			}
			parseValue := parseResult.Get("value")
			if parseValue.IsUndefined() || parseValue.IsNull() {
				return "", false, nil
			}
			return parseValue.String(), true, nil
		},
		setItem: func(parseCtx3 context.Context, parseKey2 string, parseValue2 string) error {
			parseEntry := js.Global().Get("Object").New()
			parseEntry.Set("key", parseKey2)
			parseEntry.Set("value", parseValue2)
			_, parseErr3 := parseWithStore(parseCtx3, "readwrite", "PersistentStore.SetItem", func(store js.Value) (js.Value, error) {
				return store.Call("put", parseEntry), nil
			})
			return parseErr3
		},
		removeItem: func(parseCtx4 context.Context, parseKey3 string) error {
			_, parseErr4 := parseWithStore(parseCtx4, "readwrite", "PersistentStore.RemoveItem", func(store js.Value) (js.Value, error) {
				return store.Call("delete", parseKey3), nil
			})
			return parseErr4
		},
		clear: func(parseCtx5 context.Context) error {
			_, parseErr5 := parseWithStore(parseCtx5, "readwrite", "PersistentStore.Clear", func(store js.Value) (js.Value, error) {
				return store.Call("clear"), nil
			})
			return parseErr5
		},
		keys: func(parseCtx6 context.Context) ([]string, error) {
			parseResult2, parseErr6 := parseWithStore(parseCtx6, "readonly", "PersistentStore.Keys", func(store js.Value) (js.Value, error) {
				return store.Call("getAllKeys"), nil
			})
			if parseErr6 != nil {
				return nil, parseErr6
			}
			parseLength := parseResult2.Length()
			parseKeys := make([]string, 0, parseLength)
			for parseI := 0; parseI < parseLength; parseI++ {
				parseKeys = append(parseKeys, parseResult2.Index(parseI).String())
			}
			sort.Strings(parseKeys)
			return parseKeys, nil
		},
		length: func(parseCtx7 context.Context) (int, error) {
			parseResult3, parseErr7 := parseWithStore(parseCtx7, "readonly", "PersistentStore.Len", func(store js.Value) (js.Value, error) {
				return store.Call("count"), nil
			})
			if parseErr7 != nil {
				return 0, parseErr7
			}
			return parseResult3.Int(), nil
		},
		close: func() error {
			parseDb.Set("onversionchange", js.Undefined())
			parseOnVersionChange.Release()
			if parseCloseFn2 := parseDb.Get("close"); parseCloseFn2.Type() == js.TypeFunction {
				parseDb.Call("close")
			}
			return nil
		},
	}
}

func wrapStoragePersistentStore(parseStorage Storage, parseBackend string) PersistentStore {
	return PersistentStore{
		backend: func() string { return parseBackend },
		getItem: func(parseCtx context.Context, parseKey2 string) (string, bool, error) {
			return parseStorage.GetItem(parseKey2)
		},
		setItem: func(parseCtx2 context.Context, parseKey3 string, parseValue string) error {
			return parseStorage.SetItem(parseKey3, parseValue)
		},
		removeItem: func(parseCtx3 context.Context, parseKey4 string) error {
			return parseStorage.RemoveItem(parseKey4)
		},
		clear: func(parseCtx4 context.Context) error {
			return parseStorage.Clear()
		},
		keys: func(parseCtx5 context.Context) ([]string, error) {
			parseLength, parseErr := parseStorage.Len()
			if parseErr != nil {
				return nil, parseErr
			}
			parseKeys := make([]string, 0, parseLength)
			for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
				parseKey, parseOk, parseKeyErr := parseStorage.Key(parseIndex)
				if parseKeyErr != nil {
					return nil, parseKeyErr
				}
				if parseOk {
					parseKeys = append(parseKeys, parseKey)
				}
			}
			sort.Strings(parseKeys)
			return parseKeys, nil
		},
		length: func(parseCtx6 context.Context) (int, error) {
			return parseStorage.Len()
		},
		close: func() error { return nil },
	}
}

func awaitIDBRequest(parseCtx context.Context, parseOp string, parseTarget string, parseRequest js.Value) (js.Value, error) {
	parseResultCh := make(chan js.Value, 1)
	parseFailureCh := make(chan persistentStoreFailure, 1)
	var parseOnSuccess js.Func
	var parseOnError js.Func
	parseCleanup := func() {
		parseRequest.Set("onsuccess", js.Undefined())
		parseRequest.Set("onerror", js.Undefined())
		parseOnSuccess.Release()
		parseOnError.Release()
	}
	parseOnSuccess = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("awaitIDBRequest callback")
		reportPersistentStoreResult(parseResultCh, parseRequest.Get("result"))
		return nil
	})
	parseOnError = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer RecoverContainedPanic("awaitIDBRequest callback")
		reportPersistentStoreFailure(parseFailureCh, classifyPersistentStoreFailure(parseRequest.Get("error"), "indexedDB request failed"))
		return nil
	})
	parseRequest.Set("onsuccess", parseOnSuccess)
	parseRequest.Set("onerror", parseOnError)
	defer parseCleanup()

	select {
	case parseResult := <-parseResultCh:
		return parseResult, nil
	case parseFailure := <-parseFailureCh:
		return js.Undefined(), wrapError(parseOp, parseTarget, parseFailure.code, errors.New(parseFailure.message))
	case <-parseCtx.Done():
		return js.Undefined(), persistentContextError(parseOp, parseTarget, parseCtx.Err())
	}
}

func reportPersistentStoreResult(parseCh chan js.Value, parseValue js.Value) {
	select {
	case parseCh <- parseValue:
	default:
	}
}

func reportPersistentStoreFailure(parseCh chan persistentStoreFailure, parseFailure persistentStoreFailure) {
	select {
	case parseCh <- parseFailure:
	default:
	}
}

func classifyPersistentStoreFailure(parseRawError js.Value, parseFallbackMessage string) persistentStoreFailure {
	parseMessage := strings.TrimSpace(jsValueSummary(parseRawError))
	if parseMessage == "" {
		parseMessage = parseFallbackMessage
	}
	parseName := strings.TrimSpace(parseRawError.Get("name").String())
	parseMessageLower := strings.ToLower(parseMessage)
	switch {
	case parseName == "QuotaExceededError" || strings.Contains(parseMessageLower, "quota"):
		return persistentStoreFailure{code: CodeQuotaExceeded, message: parseMessage}
	case parseName == "VersionError" || parseName == "ConstraintError":
		return persistentStoreFailure{code: CodeInvalid, message: parseMessage}
	case parseName == "InvalidStateError" || parseName == "UnknownError" || strings.Contains(parseMessageLower, "corrupt") || strings.Contains(parseMessageLower, "corruption") || strings.Contains(parseMessageLower, "malformed"):
		return persistentStoreFailure{code: CodeUnavailable, message: parseMessage, recoverable: true}
	default:
		return persistentStoreFailure{code: CodeUnavailable, message: parseMessage}
	}
}

func deleteIndexedDBDatabase(parseCtx context.Context, parseDatabaseName string) error {
	parseRawIndexedDB := js.Global().Get("indexedDB")
	if parseRawIndexedDB.IsUndefined() || parseRawIndexedDB.IsNull() {
		return unavailable("OpenPersistentStore", parseDatabaseName)
	}
	parseDeleteFn := parseRawIndexedDB.Get("deleteDatabase")
	if parseDeleteFn.Type() != js.TypeFunction {
		return wrapError("OpenPersistentStore", parseDatabaseName, CodeUnavailable, errors.New("indexedDB deleteDatabase is unavailable"))
	}
	parseRequest := parseDeleteFn.Invoke(parseDatabaseName)
	parseResultCh := make(chan struct{}, 1)
	parseFailureCh := make(chan persistentStoreFailure, 1)
	var parseOnSuccess js.Func
	var parseOnError js.Func
	var parseOnBlocked js.Func
	parseCleanup := func() {
		parseRequest.Set("onsuccess", js.Undefined())
		parseRequest.Set("onerror", js.Undefined())
		parseRequest.Set("onblocked", js.Undefined())
		parseOnSuccess.Release()
		parseOnError.Release()
		parseOnBlocked.Release()
	}
	parseOnSuccess = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer RecoverContainedPanic("deleteIndexedDBDatabase callback")
		select {
		case parseResultCh <- struct{}{}:
		default:
		}
		return nil
	})
	parseOnError = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		defer RecoverContainedPanic("deleteIndexedDBDatabase callback")
		reportPersistentStoreFailure(parseFailureCh, classifyPersistentStoreFailure(parseRequest.Get("error"), "indexedDB delete request failed"))
		return nil
	})
	parseOnBlocked = js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		defer RecoverContainedPanic("deleteIndexedDBDatabase callback")
		reportPersistentStoreFailure(parseFailureCh, persistentStoreFailure{code: CodeBlocked, message: "indexedDB delete is blocked by another open tab, worker, or window"})
		return nil
	})
	parseRequest.Set("onsuccess", parseOnSuccess)
	parseRequest.Set("onerror", parseOnError)
	parseRequest.Set("onblocked", parseOnBlocked)
	defer parseCleanup()

	select {
	case <-parseResultCh:
		return nil
	case parseFailure := <-parseFailureCh:
		return wrapError("OpenPersistentStore", parseDatabaseName, parseFailure.code, errors.New(parseFailure.message))
	case <-parseCtx.Done():
		return persistentContextError("OpenPersistentStore", parseDatabaseName, parseCtx.Err())
	}
}

func persistentContextError(parseOp string, parseTarget string, parseErr error) error {
	if errors.Is(parseErr, context.DeadlineExceeded) {
		return wrapError(parseOp, parseTarget, CodeTimeout, parseErr)
	}
	return wrapError(parseOp, parseTarget, CodeCancelled, parseErr)
}
