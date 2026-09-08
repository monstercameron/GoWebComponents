//go:build gwc_desktop

package desktop

import "testing"

// TestDesktopBuildMode verifies the opt-in source-file build gate.
func TestDesktopBuildMode(parseTest *testing.T) {
	if !IsDesktopBuild() {
		parseTest.Fatal("desktop build tag did not select desktop mode")
	}
}
