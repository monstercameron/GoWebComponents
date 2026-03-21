//go:build !js || !wasm
// +build !js !wasm

package interop

import (
	"context"
	"time"
)

// LocalStorage returns an unavailable stub on non-browser builds.
func LocalStorage() (Storage, error) {
	return Storage{}, unavailable("Storage", "localStorage")
}

// SessionStorage returns an unavailable stub on non-browser builds.
func SessionStorage() (Storage, error) {
	return Storage{}, unavailable("Storage", "sessionStorage")
}

func OpenPersistentStore(ctx context.Context, options PersistentStoreOptions) (PersistentStore, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return PersistentStore{}, unavailable("OpenPersistentStore", options.Name)
}

func WindowLocation() (Location, error) {
	return Location{}, unavailable("Location", "window.location")
}

func WindowHistory() (History, error) {
	return History{}, unavailable("History", "window.history")
}

func NavigatorClipboard() (Clipboard, error) {
	return Clipboard{}, unavailable("Clipboard", "navigator.clipboard")
}

func SetTimeout(delay time.Duration, fn func()) (Timer, error) {
	_ = delay
	_ = fn
	return Timer{}, unavailable("SetTimeout", "")
}

func SetInterval(interval time.Duration, fn func()) (Timer, error) {
	_ = interval
	_ = fn
	return Timer{}, unavailable("SetInterval", "")
}

func WindowEvents() (EventTarget, error) {
	return EventTarget{}, unavailable("EventTarget", "window")
}

func DocumentEvents() (EventTarget, error) {
	return EventTarget{}, unavailable("EventTarget", "document")
}

func CurrentDocument() (Document, error) {
	return Document{}, unavailable("Document", "document")
}

func MatchMedia(query string) (MediaQueryList, error) {
	return MediaQueryList{}, unavailable("MatchMedia", query)
}

func ImportModule(ctx context.Context, specifier string) (Module, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return Module{}, unavailable("ImportModule", specifier)
}

func NewWorker(ctx context.Context, options WorkerOptions) (Worker, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return Worker{}, unavailable("NewWorker", options.URL)
}

func OpenCrossTabChannel(options CrossTabChannelOptions) (CrossTabChannel, error) {
	return CrossTabChannel{}, unavailable("OpenCrossTabChannel", options.Name)
}

func OpenSecondaryWindowChannel(options WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenSecondaryWindowChannel", options.Name)
}

func WindowOpenerChannel(options WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("WindowOpenerChannel", options.Name)
}
