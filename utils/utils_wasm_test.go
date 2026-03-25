//go:build js && wasm && !production
// +build js,wasm,!production

package utils

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/hotreload"
)

func TestEnableHotReloadDelegatesToHotreloadPackage(parseT *testing.T) {
	hotreload.Disable()
	parseT.Cleanup(hotreload.Disable)

	EnableHotReload(true)
	if !hotreload.IsEnabled() {
		parseT.Fatal("expected compatibility wrapper to enable hotreload package")
	}

	EnableHotReload(false)
	if hotreload.IsEnabled() {
		parseT.Fatal("expected compatibility wrapper to disable hotreload package")
	}
}
