//go:build !gwc_desktop

package desktop

// IsDesktopBuild reports the compile-time target, not permission to use a native API.
// An ordinary build is web/native-test mode unless gwc_desktop was explicitly set.
func IsDesktopBuild() bool { return false }
