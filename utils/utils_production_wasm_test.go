//go:build js && wasm && production
// +build js,wasm,production

package utils

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/hotreload"
)

func TestEnableHotReloadIsDisabledInProduction(t *testing.T) {
	EnableHotReload(true)

	if IsHotReloadEnabled() {
		t.Fatal("expected hot reload to remain disabled in production")
	}
	if hotreload.Enabled() {
		t.Fatal("expected hotreload package to remain disabled in production")
	}
}
