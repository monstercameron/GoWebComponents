//go:build js && wasm && production
// +build js,wasm,production

package hotreload

// Configure is disabled in production builds.
func Configure(config Config) {
	_ = config
}

// Disable is a no-op in production builds.
func Disable() {}

// Enabled always reports false in production builds.
func Enabled() bool { return false }

// IsEnabled is a compatibility wrapper around Enabled.
func IsEnabled() bool { return Enabled() }

// GetSnapshot returns an empty payload in production builds.
func GetSnapshot() (string, error) { return "", nil }

// ApplySnapshot is a no-op in production builds.
func ApplySnapshot(payload string) error {
	_ = payload
	return nil
}

// Prepare is a no-op in production builds.
func Prepare() {}
