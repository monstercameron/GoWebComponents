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

// ExportSnapshot returns an empty payload in production builds.
func ExportSnapshot() (string, error) { return "", nil }

// ImportSnapshot is a no-op in production builds.
func ImportSnapshot(payload string) error {
	_ = payload
	return nil
}

// Prepare is a no-op in production builds.
func Prepare() {}
