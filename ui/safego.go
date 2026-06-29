package ui

import "github.com/monstercameron/GoWebComponents/v4/internal/runtime"

// SafeGo starts fn on a new goroutine with framework crash containment. In
// wasm a panic that escapes any goroutine exits the whole Go program and
// leaves the page dead; SafeGo recovers the panic, prints the structured
// agent-readable crash report to the console, and lets the rest of the app
// keep running. Use it instead of the bare go statement for application
// background work. The subject names the task in the crash report.
func SafeGo(parseSubject string, parseFn func()) {
	runtime.SafeGo("app", parseSubject, parseFn)
}
