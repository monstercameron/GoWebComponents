package ui

import (
	"encoding/json"

	"github.com/monstercameron/GoWebComponents/v5/interop"
)

// PersistStorageArea identifies which browser storage area to use.
type PersistStorageArea string

const (
	// PersistLocal uses localStorage (persists across sessions).
	PersistLocal PersistStorageArea = "local"
	// PersistSession uses sessionStorage (cleared when tab closes).
	PersistSession PersistStorageArea = "session"
)

// PersistedState is a state handle backed by browser storage. On native/SSR
// builds, storage writes are silently degraded to in-memory state.
type PersistedState[T any] struct {
	parseState State[T]
	parseErr   State[error]
}

type persistedStorage interface {
	GetItem(string) (string, bool, error)
	SetItem(string, string) error
}

type persistedStateHandle[T any] interface {
	Get() T
	Set(T)
}

type persistedErrorHandle interface {
	Set(error)
}

type persistedEventTarget interface {
	Listen(string, func(interop.BrowserEvent)) (interop.Subscription, error)
}

var getPersistStorage = func(parseArea PersistStorageArea) (persistedStorage, error) {
	if parseArea == PersistSession {
		return interop.SessionStorage()
	}
	return interop.LocalStorage()
}

var getPersistWindowEvents = func() (persistedEventTarget, error) {
	return interop.GetWindowEvents()
}

// Get returns the current value.
func (parsePs PersistedState[T]) Get() T {
	return parsePs.parseState.Get()
}

// Set writes the value to in-memory state and to the backing storage area.
// If the storage write fails (e.g. quota exceeded), the error is captured and
// retrievable via Err(); the in-memory state is still updated.
func (parsePs PersistedState[T]) Set(parseVal T) {
	parsePs.parseState.Set(parseVal)
}

// Err returns the last storage error, or nil when no error has occurred.
func (parsePs PersistedState[T]) Err() error {
	return parsePs.parseErr.Get()
}

// resolveStorage returns the interop.Storage for the given PersistStorageArea.
func resolveStorage(parseArea PersistStorageArea) (persistedStorage, error) {
	return getPersistStorage(parseArea)
}

// UsePersistedState returns a PersistedState backed by the chosen storage area.
// On first use it reads the stored JSON for parseKey from the chosen storage
// area; if found and decodable, that value is used as the initial state,
// otherwise parseInitial is used. Corrupted stored values do not panic — they
// are silently discarded. On native/SSR builds, storage is unavailable and the
// hook degrades to plain in-memory state.
func UsePersistedState[T any](parseKey string, parseInitial T, parseArea PersistStorageArea) PersistedState[T] {
	parseStoredInitial := loadStoredInitial[T](parseKey, parseInitial, parseArea)

	parseValState := UseState(parseStoredInitial)
	parseErrState := UseState[error](nil)

	// UseEffect registers the storage write-through and cross-tab sync listener.
	// On native builds UseEffect is a no-op, so the closure never runs.
	UseEffect(func() func() {
		return subscribePersistedStorageSync(parseKey, parseArea, parseValState)
	}, parseKey, string(parseArea))

	// Wrap the state so that Set also writes through to storage.
	parseResult := PersistedState[T]{
		parseErr: parseErrState,
	}

	parseWriteThrough := func(parseVal T) {
		writePersistedStateValue(parseKey, parseVal, parseArea, parseValState, parseErrState)
	}

	parseResult.parseState = State[T]{
		get:  parseValState.get,
		slot: parseValState.slot,
		set: func(parseRaw any) {
			if parseTyped, parseOk := parseRaw.(T); parseOk {
				parseWriteThrough(parseTyped)
				return
			}
			if parseUpdater, parseOk2 := parseRaw.(func(T) T); parseOk2 {
				parseWriteThrough(parseUpdater(parseValState.Get()))
			}
		},
	}

	return parseResult
}

func subscribePersistedStorageSync[T any](parseKey string, parseArea PersistStorageArea, parseValState persistedStateHandle[T]) func() {
	parseStore, parseStoreErr := resolveStorage(parseArea)
	if parseStoreErr != nil {
		// Storage unavailable (native/SSR); nothing to set up.
		return func() {}
	}

	// Subscribe to window "storage" events for cross-tab synchronisation.
	parseWin, parseWinErr := getPersistWindowEvents()
	if parseWinErr != nil {
		return func() {}
	}

	parseSub, parseSubErr := parseWin.Listen("storage", func(parseEvent interop.BrowserEvent) {
		// Re-read from storage when another tab writes the same key.
		parseStored, parseFound, parseReadErr := parseStore.GetItem(parseKey)
		if parseReadErr != nil || !parseFound {
			return
		}
		var parseDecoded T
		if parseUnmarshalErr := json.Unmarshal([]byte(parseStored), &parseDecoded); parseUnmarshalErr != nil {
			return
		}
		parseValState.Set(parseDecoded)
	})
	if parseSubErr != nil {
		return func() {}
	}

	return func() { parseSub.Cancel() }
}

func writePersistedStateValue[T any](parseKey string, parseVal T, parseArea PersistStorageArea, parseValState persistedStateHandle[T], parseErrState persistedErrorHandle) {
	parseValState.Set(parseVal)
	parseStore, parseStoreErr := resolveStorage(parseArea)
	if parseStoreErr != nil {
		// Unavailable on native; swallow silently.
		return
	}
	parseData, parseMarshalErr := json.Marshal(parseVal)
	if parseMarshalErr != nil {
		parseErrState.Set(parseMarshalErr)
		return
	}
	if parseSetErr := parseStore.SetItem(parseKey, string(parseData)); parseSetErr != nil {
		parseErrState.Set(parseSetErr)
		// In-memory state is already updated; keep going.
	} else {
		parseErrState.Set(nil)
	}
}

// loadStoredInitial reads the stored JSON for parseKey from the chosen storage
// area. If anything goes wrong (unavailable, missing, corrupt) it returns
// parseInitial without panicking.
func loadStoredInitial[T any](parseKey string, parseInitial T, parseArea PersistStorageArea) T {
	parseStore, parseStoreErr := resolveStorage(parseArea)
	if parseStoreErr != nil {
		return parseInitial
	}
	parseRaw, parseFound, parseGetErr := parseStore.GetItem(parseKey)
	if parseGetErr != nil || !parseFound {
		return parseInitial
	}
	var parseLoaded T
	parseDecodeErr := func() (parseErr error) {
		defer func() {
			if parseR := recover(); parseR != nil {
				// Map the panic to a non-nil error so we fall back to the initial value.
				parseErr = &json.SyntaxError{}
			}
		}()
		return json.Unmarshal([]byte(parseRaw), &parseLoaded)
	}()
	if parseDecodeErr != nil {
		return parseInitial
	}
	return parseLoaded
}
