//go:build js && wasm && production
// +build js,wasm,production

package utils

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

func TestEnableHotReloadIsDisabledInProduction(t *testing.T) {
	global, err := interop.GlobalThis()
	if err != nil {
		t.Fatalf("expected browser global, got %v", err)
	}
	prevExport := global.Get("exportAppState")
	prevPrepare := global.Get("prepareAppHotReload")
	prevImport := global.Get("importAppState")

	t.Cleanup(func() {
		if prevExport.Present() {
			_ = global.Set("exportAppState", prevExport)
		} else {
			_ = global.Delete("exportAppState")
		}
		if prevPrepare.Present() {
			_ = global.Set("prepareAppHotReload", prevPrepare)
		} else {
			_ = global.Delete("prepareAppHotReload")
		}
		if prevImport.Present() {
			_ = global.Set("importAppState", prevImport)
		} else {
			_ = global.Delete("importAppState")
		}
	})

	EnableHotReload(true)

	if IsHotReloadEnabled() {
		t.Fatal("expected hot reload to remain disabled in production")
	}
	if global.Get("exportAppState").Present() {
		t.Fatal("expected exportAppState to stay unset in production")
	}
	if global.Get("prepareAppHotReload").Present() {
		t.Fatal("expected prepareAppHotReload to stay unset in production")
	}
	if global.Get("importAppState").Present() {
		t.Fatal("expected importAppState to stay unset in production")
	}
}
