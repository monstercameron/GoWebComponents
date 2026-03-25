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
func GetWindowEnv() WindowEnv {
	return WindowEnv{}
}

func OpenPersistentStore(ctx context.Context, options PersistentStoreOptions) (PersistentStore, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return PersistentStore{}, unavailable("OpenPersistentStore", options.Name)
}

func GetWindowLocation() (Location, error) {
	return Location{}, unavailable("Location", "window.location")
}

func GetWindowHistory() (History, error) {
	return History{}, unavailable("History", "window.history")
}

func GetClipboard() (Clipboard, error) {
	return Clipboard{}, unavailable("Clipboard", "navigator.clipboard")
}

func ScheduleTimeout(delay time.Duration, fn func()) (Timer, error) {
	_ = delay
	_ = fn
	return Timer{}, unavailable("ScheduleTimeout", "")
}

func ScheduleInterval(interval time.Duration, fn func()) (Timer, error) {
	_ = interval
	_ = fn
	return Timer{}, unavailable("ScheduleInterval", "")
}

func GetWindowEvents() (EventTarget, error) {
	return EventTarget{}, unavailable("EventTarget", "window")
}

func GetDocumentEvents() (EventTarget, error) {
	return EventTarget{}, unavailable("EventTarget", "document")
}

func GetDocument() (Document, error) {
	return Document{}, unavailable("Document", "document")
}

func GetMediaQuery(query string) (MediaQueryList, error) {
	return MediaQueryList{}, unavailable("GetMediaQuery", query)
}

func ImportModule(ctx context.Context, specifier string) (Module, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return Module{}, unavailable("ImportModule", specifier)
}

func OpenWorker(ctx context.Context, options WorkerOptions) (Worker, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return Worker{}, unavailable("OpenWorker", options.URL)
}

func OpenGoWASMWorker(ctx context.Context, options GoWASMWorkerOptions) (Worker, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return Worker{}, unavailable("OpenGoWASMWorker", options.WASMURL)
}

func GetWorkerScope() (WorkerScope, error) {
	return WorkerScope{}, unavailable("GetWorkerScope", "worker")
}

func OpenCrossTabChannel(options CrossTabChannelOptions) (CrossTabChannel, error) {
	return CrossTabChannel{}, unavailable("OpenCrossTabChannel", options.Name)
}

func OpenSecondaryWindowChannel(options WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenSecondaryWindowChannel", options.Name)
}

func OpenWindowOpenerChannel(options WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenWindowOpenerChannel", options.Name)
}
