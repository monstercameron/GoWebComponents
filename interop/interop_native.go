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
func OpenPersistentStore(ctx context.Context, options PersistentStoreOptions) (PersistentStore, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return PersistentStore{}, unavailable("OpenPersistentStore", options.Name)
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
func ScheduleTimeout(delay time.Duration, fn func()) (Timer, error) {
	_ = delay
	_ = fn
	return Timer{}, unavailable("ScheduleTimeout", "")
}

// ScheduleInterval is a non-browser stub that always returns an unavailable error.
func ScheduleInterval(interval time.Duration, fn func()) (Timer, error) {
	_ = interval
	_ = fn
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
func GetMediaQuery(query string) (MediaQueryList, error) {
	return MediaQueryList{}, unavailable("GetMediaQuery", query)
}

// ImportModule is a non-browser stub that always returns an unavailable error.
func ImportModule(ctx context.Context, specifier string) (Module, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return Module{}, unavailable("ImportModule", specifier)
}

// OpenWorker is a non-browser stub that always returns an unavailable error.
func OpenWorker(ctx context.Context, options WorkerOptions) (Worker, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return Worker{}, unavailable("OpenWorker", options.URL)
}

// OpenGoWASMWorker is a non-browser stub that always returns an unavailable error.
func OpenGoWASMWorker(ctx context.Context, options GoWASMWorkerOptions) (Worker, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return Worker{}, unavailable("OpenGoWASMWorker", options.WASMURL)
}

// GetWorkerScope is a non-browser stub that always returns an unavailable error.
func GetWorkerScope() (WorkerScope, error) {
	return WorkerScope{}, unavailable("GetWorkerScope", "worker")
}

// OpenCrossTabChannel is a non-browser stub that always returns an unavailable error.
func OpenCrossTabChannel(options CrossTabChannelOptions) (CrossTabChannel, error) {
	return CrossTabChannel{}, unavailable("OpenCrossTabChannel", options.Name)
}

// OpenSecondaryWindowChannel is a non-browser stub that always returns an unavailable error.
func OpenSecondaryWindowChannel(options WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenSecondaryWindowChannel", options.Name)
}

// OpenWindowOpenerChannel is a non-browser stub that always returns an unavailable error.
func OpenWindowOpenerChannel(options WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenWindowOpenerChannel", options.Name)
}
