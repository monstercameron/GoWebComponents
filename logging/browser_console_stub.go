//go:build !js || !wasm
// +build !js !wasm

package logging

// BrowserConsoleOptions configures browser event logging on js/wasm builds.
type BrowserConsoleOptions struct {
	Scope                  string
	LogClicks              bool
	LogChanges             bool
	LogSubmits             bool
	LogWindowErrors        bool
	LogNavigation          bool
	LogMount               bool
	LogDocumentReady       bool
	LogUnhandledRejections bool
	LogVisibility          bool
	LogResize              bool
	LogBeforeUnload        bool
}

// AttachBrowserConsole is a no-op on non-browser targets.
func AttachBrowserConsole(parseConsoleOptions BrowserConsoleOptions) func() {
	_ = parseConsoleOptions
	return func() {}
}
