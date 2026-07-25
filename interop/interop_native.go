//go:build !js || !wasm

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
func OpenPersistentStore(parseStoreCtx context.Context, parseStoreOptions PersistentStoreOptions) (PersistentStore, error) {
	_ = parseStoreCtx
	return PersistentStore{}, unavailable("OpenPersistentStore", parseStoreOptions.Name)
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
func ScheduleTimeout(parseTimerDelay time.Duration, parseTimerFn func()) (Timer, error) {
	_ = parseTimerDelay
	_ = parseTimerFn
	return Timer{}, unavailable("ScheduleTimeout", "")
}

// ScheduleInterval is a non-browser stub that always returns an unavailable error.
func ScheduleInterval(parseTimerInterval time.Duration, parseTimerFn func()) (Timer, error) {
	_ = parseTimerInterval
	_ = parseTimerFn
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
func GetMediaQuery(parseMediaQuery string) (MediaQueryList, error) {
	return MediaQueryList{}, unavailable("GetMediaQuery", parseMediaQuery)
}

// RequestPersistentStorage is a non-browser stub that always returns an
// unavailable error.
func RequestPersistentStorage(parseCtx context.Context) (bool, error) {
	_ = parseCtx
	return false, unavailable("RequestPersistentStorage", "navigator.storage.persist")
}

// IsStoragePersisted is a non-browser stub that always returns an unavailable
// error.
func IsStoragePersisted(parseCtx context.Context) (bool, error) {
	_ = parseCtx
	return false, unavailable("IsStoragePersisted", "navigator.storage.persisted")
}

// ImportModule is a non-browser stub that always returns an unavailable error.
func ImportModule(parseImportCtx context.Context, parseImportSpecifier string) (Module, error) {
	_ = parseImportCtx
	return Module{}, unavailable("ImportModule", parseImportSpecifier)
}

// OpenWorker is a non-browser stub that always returns an unavailable error.
func OpenWorker(parseWorkerCtx context.Context, parseWorkerOptions WorkerOptions) (Worker, error) {
	_ = parseWorkerCtx
	return Worker{}, unavailable("OpenWorker", parseWorkerOptions.URL)
}

// OpenGoWASMWorker is a non-browser stub that always returns an unavailable error.
func OpenGoWASMWorker(parseWorkerCtx context.Context, parseWorkerOptions GoWASMWorkerOptions) (Worker, error) {
	_ = parseWorkerCtx
	return Worker{}, unavailable("OpenGoWASMWorker", parseWorkerOptions.WASMURL)
}

// Transferable is the non-browser form of a movable binary payload (v5 P3.1).
//
// Off-browser there is no postMessage and nothing to transfer, so this holds
// the bytes and every send path reports unavailable. It exists so code that
// builds transferables compiles unchanged on the native and SSR slices.
type Transferable struct {
	data []byte
}

// NewTransferable is a non-browser stub that retains the bytes.
func NewTransferable(parseData []byte) (Transferable, error) {
	return Transferable{data: parseData}, nil
}

// Bytes returns the retained payload.
func (parseT Transferable) Bytes() []byte { return parseT.data }

// Len reports the payload length.
func (parseT Transferable) Len() int { return len(parseT.data) }

// IsDetached is always false off-browser: nothing can take ownership away.
func (parseT Transferable) IsDetached() bool { return false }

// PostTransferable is a non-browser stub that always returns an unavailable error.
func (parseW Worker) PostTransferable(parsePayload any, parseBuffers ...Transferable) error {
	return unavailable("PostTransferable", "worker")
}

// PostTransferable is a non-browser stub that always returns an unavailable error.
func (parseP MessagePort) PostTransferable(parsePayload any, parseBuffers ...Transferable) error {
	return unavailable("PostTransferable", "message port")
}

// GetWorkerScope is a non-browser stub that always returns an unavailable error.
func GetWorkerScope() (WorkerScope, error) {
	return WorkerScope{}, unavailable("GetWorkerScope", "worker")
}

// OpenMessageChannel is a non-browser stub that always returns an unavailable error.
func OpenMessageChannel() (MessageChannel, error) {
	return MessageChannel{}, unavailable("OpenMessageChannel", "MessageChannel")
}

// GetSharedMemorySupport is a non-browser stub that always returns an unavailable error.
func GetSharedMemorySupport() (SharedMemorySupport, error) {
	return SharedMemorySupport{}, unavailable("GetSharedMemorySupport", "SharedArrayBuffer")
}

// OpenSharedBuffer is a non-browser stub that always returns an unavailable error.
func OpenSharedBuffer(parseByteLength int) (SharedBuffer, error) {
	_ = parseByteLength
	return SharedBuffer{}, unavailable("OpenSharedBuffer", "SharedArrayBuffer")
}

// OpenCrossTabChannel is a non-browser stub that always returns an unavailable error.
func OpenCrossTabChannel(parseChannelOptions CrossTabChannelOptions) (CrossTabChannel, error) {
	return CrossTabChannel{}, unavailable("OpenCrossTabChannel", parseChannelOptions.Name)
}

// OpenSecondaryWindowChannel is a non-browser stub that always returns an unavailable error.
func OpenSecondaryWindowChannel(parseWindowOptions WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenSecondaryWindowChannel", parseWindowOptions.Name)
}

// OpenWindowOpenerChannel is a non-browser stub that always returns an unavailable error.
func OpenWindowOpenerChannel(parseWindowOptions WindowChannelOptions) (WindowChannel, error) {
	return WindowChannel{}, unavailable("OpenWindowOpenerChannel", parseWindowOptions.Name)
}
