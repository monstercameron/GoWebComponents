//go:build !gwc_desktop

package desktop

import "testing"

// TestWebBuildMode verifies ordinary builds never imply desktop mode.
func TestWebBuildMode(parseTest *testing.T) {
	if IsDesktopBuild() {
		parseTest.Fatal("untagged build selected desktop mode")
	}
}
