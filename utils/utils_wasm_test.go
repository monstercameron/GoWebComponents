//go:build js && wasm && !production
// +build js,wasm,!production

package utils

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/interop"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(callback func(), delay int) {}

func TestInstallHotReloadBridgeExportsAndImportsSnapshot(t *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	if err := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "dark"); err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

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
		hotReloadExportStateSub.Cancel()
		hotReloadPrepareStateSub.Cancel()
		hotReloadImportStateSub.Cancel()
		hotReloadBridgeInstalled = false
	})

	InstallHotReloadBridge()

	exportFn := global.Get("exportAppState")
	if !exportFn.Present() {
		t.Fatal("expected exportAppState to be installed")
	}
	prepareFn := global.Get("prepareAppHotReload")
	if !prepareFn.Present() {
		t.Fatal("expected prepareAppHotReload to be installed")
	}
	importFn := global.Get("importAppState")
	if !importFn.Present() {
		t.Fatal("expected importAppState to be installed")
	}

	payload, err := exportFn.Invoke()
	if err != nil {
		t.Fatalf("expected export invocation to succeed, got %v", err)
	}
	if !payload.Present() {
		t.Fatal("expected exported hot reload payload")
	}
	exported := payload.String()
	if exported == "" {
		t.Fatal("expected exported hot reload payload")
	}

	if err := runtime.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "light"); err != nil {
		t.Fatalf("unexpected mutation error: %v", err)
	}

	if _, err := importFn.Invoke(exported); err != nil {
		t.Fatalf("expected import invocation to succeed, got %v", err)
	}

	value, ok := runtime.GetGlobalRuntime().GetAtomValue("hot-reload-theme")
	if !ok {
		t.Fatal("expected restored atom to be present")
	}
	if value != "dark" {
		t.Fatalf("expected restored atom value dark, got %#v", value)
	}

	if _, err := prepareFn.Invoke(); err != nil {
		t.Fatalf("expected prepare invocation to succeed, got %v", err)
	}
}
