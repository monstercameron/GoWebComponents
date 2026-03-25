//go:build !js || !wasm
// +build !js !wasm

package hotreload

// Configure is unavailable on non-browser builds.
func Configure(config Config) {
	_ = config
}

// Disable is unavailable on non-browser builds.
func Disable() {}

// Enabled always reports false on non-browser builds.
func Enabled() bool { return false }

// GetSnapshot returns an empty payload on non-browser builds.
func GetSnapshot() (string, error) { return "", nil }

// ApplySnapshot is a no-op on non-browser builds.
func ApplySnapshot(payload string) error {
	_ = payload
	return nil
}

// Prepare is a no-op on non-browser builds.
func Prepare() {}
