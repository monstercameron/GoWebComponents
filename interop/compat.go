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
func SharedWindowEnv() WindowEnv {
	return GetWindowEnv()
}

// NewGoWASMWorker preserves the legacy Go WASM worker constructor name.
func NewGoWASMWorker(ctx context.Context, options GoWASMWorkerOptions) (Worker, error) {
	return OpenGoWASMWorker(ctx, options)
}

// SetTimeout preserves the legacy timer helper name.
func SetTimeout(delay time.Duration, fn func()) (Timer, error) {
	return ScheduleTimeout(delay, fn)
}

// SetInterval preserves the legacy interval helper name.
func SetInterval(interval time.Duration, fn func()) (Timer, error) {
	return ScheduleInterval(interval, fn)
}
