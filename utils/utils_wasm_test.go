//go:build js && wasm && !production
// +build js,wasm,!production

package utils

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/hotreload"
)

func TestEnableHotReloadDelegatesToHotreloadPackage(t *testing.T) {
	hotreload.Disable()
	t.Cleanup(hotreload.Disable)

	EnableHotReload(true)
	if !hotreload.Enabled() {
		t.Fatal("expected compatibility wrapper to enable hotreload package")
	}

	EnableHotReload(false)
	if hotreload.Enabled() {
		t.Fatal("expected compatibility wrapper to disable hotreload package")
	}
}
