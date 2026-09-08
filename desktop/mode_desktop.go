//go:build gwc_desktop

package desktop

// IsDesktopBuild reports the compile-time target, not permission to use a native API.
// Use Client.Supports or Client.Require for the connected host's effective policy.
func IsDesktopBuild() bool { return true }
