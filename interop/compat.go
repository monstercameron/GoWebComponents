package interop

import "context"
import "time"

// GlobalThis preserves the legacy interop global accessor name.
func GlobalThis() (Value, error) {
	return GetGlobalThis()
}

// LocalStorage preserves the legacy storage accessor name.
func LocalStorage() (Storage, error) {
	return GetLocalStorage()
}

// SessionStorage preserves the legacy storage accessor name.
func SessionStorage() (Storage, error) {
	return GetSessionStorage()
}

// NavigatorClipboard preserves the legacy clipboard accessor name.
func NavigatorClipboard() (Clipboard, error) {
	return GetClipboard()
}

// CurrentDocument preserves the legacy document accessor name.
func CurrentDocument() (Document, error) {
	return GetDocument()
}

// SharedWindowEnv preserves the legacy shared window env accessor name.
func SharedWindowEnv() (WindowEnv, error) {
	return GetWindowEnv()
}

// NewGoWASMWorker preserves the legacy Go WASM worker constructor name.
func NewGoWASMWorker(parseWorkerCtx context.Context, parseWorkerOptions GoWASMWorkerOptions) (Worker, error) {
	return OpenGoWASMWorker(parseWorkerCtx, parseWorkerOptions)
}

// SetTimeout preserves the legacy timer helper name.
func SetTimeout(parseTimerDelay time.Duration, parseTimerFn func()) (Timer, error) {
	return ScheduleTimeout(parseTimerDelay, parseTimerFn)
}

// SetInterval preserves the legacy interval helper name.
func SetInterval(parseTimerInterval time.Duration, parseTimerFn func()) (Timer, error) {
	return ScheduleInterval(parseTimerInterval, parseTimerFn)
}
