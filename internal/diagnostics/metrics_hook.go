package diagnostics

import "sync"

var (
	panicObserverMu sync.RWMutex
	panicObserver   func(parseCode string)
)

// SetPanicObserver installs the function NotifyPanic calls with the stable gwc
// error code of every reported panic. A nil value removes it, which is the
// default. The observer must be cheap and must not panic; it feeds metrics
// bridges that live outside the core library.
func SetPanicObserver(parseObserver func(parseCode string)) {
	panicObserverMu.Lock()
	panicObserver = parseObserver
	panicObserverMu.Unlock()
}

// NotifyPanic reports one panic with parseCode to the installed observer and does
// nothing when none is installed. It recovers from a misbehaving observer so a
// metrics failure can never mask the panic being reported.
func NotifyPanic(parseCode string) {
	panicObserverMu.RLock()
	parseObserver := panicObserver
	panicObserverMu.RUnlock()
	if parseObserver == nil {
		return
	}
	defer func() { _ = recover() }()
	parseObserver(parseCode)
}
