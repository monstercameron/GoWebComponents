//go:build !js || !wasm
// +build !js !wasm

package interop

import (
	"context"
	"time"
)

// GetLocalStorage returns an unavailable stub on non-browser builds.
func GetLocalStorage() (Storage, error) {
	return Storage{}, unavailable("Storage", "localStorage")
}

// GetSessionStorage returns an unavailable stub on non-browser builds.
func GetSessionStorage() (Storage, error) {
	return Storage{}, unavailable("Storage", "sessionStorage")
}

// GetWindowEnv returns an empty reader on non-browser builds.
func GetWindowEnv() (WindowEnv, error) {
	return WindowEnv{}, nil
}

// OpenPersistentStore is a non-browser stub that always returns an unavailable error.
func OpenPersistentStore(parseCtx context.Context, parseOptions PersistentStoreOptions) (PersistentStore, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return PersistentStore{}, unavailable("OpenPersistentStore", parseOptions.Name)
}

// GetWindowLocation is a non-browser stub that always returns an unavailable error.
func GetWindowLocation() (Location, error) {
	return Location{}, unavailable("Location", "window.location")
}

// GetWindowHistory is a non-browser stub that always returns an unavailable error.
func GetWindowHistory() (History, error) {
	return History{}, unavailable("History", "window.history")
}

// GetClipboard is a non-browser stub that always returns an unavailable error.
func GetClipboard() (Clipboard, error) {
	return Clipboard{}, unavailable("Clipboard", "navigator.clipboard")
}

// ScheduleTimeout is a non-browser stub that always returns an unavailable error.
func ScheduleTimeout(parseDelay time.Duration, parseFn func()) (Timer, error) {
	_ = parseDelay
	_ = parseFn
	return Timer{}, unavailable("ScheduleTimeout", "")
}

// ScheduleInterval is a non-browser stub that always returns an unavailable error.
func ScheduleInterval(parseInterval time.Duration, parseFn func()) (Timer, error) {
	_ = parseInterval
	_ = parseFn
	return Timer{}, unavailable("ScheduleInterval", "")
}

// GetWindowEvents is a non-browser stub that always returns an unavailable error.
func GetWindowEvents() (EventTarget, error) {
	return EventTarget{}, unavailable("EventTarget", "window")
}

// GetDocumentEvents is a non-browser stub that always returns an unavailable error.
func GetDocumentEvents() (EventTarget, error) {
	return EventTarget{}, unavailable("EventTarget", "document")
}

// GetDocument is a non-browser stub that always returns an unavailable error.
func GetDocument() (Document, error) {
	return Document{}, unavailable("Document", "document")
}

// GetMediaQuery is a non-browser stub that always returns an unavailable error.
func GetMediaQuery(parseQuery string) (MediaQueryList, error) {
	return MediaQueryList{}, unavailable("GetMediaQuery", parseQuery)
}

// ImportModule is a non-browser stub that always returns an unavailable error.
func ImportModule(parseCtx context.Context, parseSpecifier string) (Module, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return Module{}, unavailable("ImportModule", parseSpecifier)
}

// OpenWorker is a non-browser stub that always returns an unavailable error.
func OpenWorker(parseCtx context.Context, parseOptions WorkerOptions) (Worker, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return Worker{}, unavailable("OpenWorker", parseOptions.URL)
}

// OpenGoWASMWorker is a non-browser stub that always returns an unavailable error.
func OpenGoWASMWorker(parseCtx context.Context, parseOptions GoWASMWorkerOptions) (Worker, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return Worker{}, unavailable("OpenGoWASMWorker", parseOptions.WASMURL)
}

// GetWorkerScope is a non-browser stub that always returns an unavailable error.
func GetWorkerScope() (WorkerScope, error) {
	return WorkerScope{}, unavailable("GetWorkerScope", "worker")
}

// OpenCrossTabChannel is a non-browser stub that always returns an unavailable error.
func OpenCrossTabChannel(parseOptions CrossTabChannelOptions) (CrossTabChannel, error) {
	return CrossTabChannel{}, unavailable("OpenCrossTabChannel", parseOptions.Name)
}

// OpenSecondaryWindowChannel is a non-browser stub that always returns an unavailable error.
func OpenSecondaryWindowChannel(parseOptions WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenSecondaryWindowChannel", parseOptions.Name)
}

// OpenWindowOpenerChannel is a non-browser stub that always returns an unavailable error.
func OpenWindowOpenerChannel(parseOptions WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenWindowOpenerChannel", parseOptions.Name)
}
