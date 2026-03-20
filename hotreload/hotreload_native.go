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

// ExportSnapshot returns an empty payload on non-browser builds.
func ExportSnapshot() (string, error) { return "", nil }

// ImportSnapshot is a no-op on non-browser builds.
func ImportSnapshot(payload string) error {
	_ = payload
	return nil
}

// Prepare is a no-op on non-browser builds.
func Prepare() {}
