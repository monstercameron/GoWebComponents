//go:build !js || !wasm

package hotreload

import "github.com/monstercameron/GoWebComponents/deprecation"

// Configure is unavailable on non-browser builds.
func Configure(parseConfig Config) {
	_ = parseConfig
}

// Disable is unavailable on non-browser builds.
func Disable() {}

// Enabled always reports false on non-browser builds.
func Enabled() bool { return false }

// IsEnabled reports whether hot reload is enabled.
//
// Deprecated: use Enabled.
func IsEnabled() bool {
	deprecation.Warn("hotreload.IsEnabled", "hotreload.Enabled")
	return Enabled()
}

// GetSnapshot returns an empty payload on non-browser builds.
func GetSnapshot() (string, error) { return "", nil }

// ApplySnapshot is a no-op on non-browser builds.
func ApplySnapshot(parsePayload string) error {
	_ = parsePayload
	return nil
}

// Prepare is a no-op on non-browser builds.
func Prepare() {}
