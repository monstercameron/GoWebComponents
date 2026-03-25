//go:build js && wasm && production
// +build js,wasm,production

package hotreload

// Configure is disabled in production builds.
func Configure(parseConfig Config) {
	_ = parseConfig
}

// Disable is a no-op in production builds.
func Disable() {}

// IsEnabled always reports false in production builds.
func IsEnabled() bool { return false }

// GetSnapshot returns an empty payload in production builds.
func GetSnapshot() (string, error) { return "", nil }

// ApplySnapshot is a no-op in production builds.
func ApplySnapshot(parsePayload string) error {
	_ = parsePayload
	return nil
}

// Prepare is a no-op in production builds.
func Prepare() {}
